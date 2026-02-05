//! WebSocket client for agent-server communication.

use anyhow::{Context, Result};
use futures_util::{SinkExt, StreamExt};
use serde::{Deserialize, Serialize};
use tokio::time::{interval, Duration};
use tokio_tungstenite::{
    connect_async_with_config,
    tungstenite::{
        client::IntoClientRequest,
        http::header::{HeaderName, HeaderValue},
        Message as WsMessage,
    },
};
use tracing::{debug, error, info, warn};

use crate::config::AgentConfig;
use crate::executor::CommandExecutor;
use crate::system::SystemInfo;

/// Message structure for agent-server WebSocket communication.
#[derive(Debug, Serialize, Deserialize)]
pub struct AgentMessage {
    /// Message type (register, heartbeat, task, task_result, etc.)
    #[serde(rename = "type")]
    pub msg_type: String,
    /// Message payload as JSON value.
    pub payload: serde_json::Value,
}

/// Payload for agent registration with the server.
#[derive(Debug, Serialize, Deserialize)]
pub struct RegisterPayload {
    /// Unique agent identifier.
    pub paw: String,
    /// Agent hostname.
    pub hostname: String,
    /// Current username.
    pub username: String,
    /// Operating system platform (linux, windows, darwin).
    pub platform: String,
    /// Available command executors (sh, bash, powershell, etc.).
    pub executors: Vec<String>,
}

/// Payload for task execution requests from the server.
#[derive(Debug, Deserialize)]
pub struct TaskPayload {
    /// Unique task identifier.
    pub id: String,
    /// MITRE ATT&CK technique ID.
    pub technique_id: String,
    /// Command to execute.
    pub command: String,
    /// Executor type (sh, bash, powershell, etc.).
    pub executor: String,
    /// Execution timeout in seconds.
    pub timeout: Option<u64>,
    /// Optional cleanup command to run after execution.
    pub cleanup: Option<String>,
}

/// WebSocket client for communicating with the AutoStrike server.
pub struct AgentClient {
    /// Agent configuration.
    pub config: AgentConfig,
    /// System information.
    pub sys_info: SystemInfo,
    /// Command executor instance.
    pub executor: CommandExecutor,
}

impl AgentClient {
    /// Creates a new agent client with the given configuration and system info.
    pub fn new(config: AgentConfig, sys_info: SystemInfo) -> Result<Self> {
        let executor = CommandExecutor::new();

        Ok(Self {
            config,
            sys_info,
            executor,
        })
    }

    /// Runs the agent client with automatic reconnection on failure.
    pub async fn run(&mut self) -> Result<()> {
        let mut retry_delay = Duration::from_secs(1);
        let max_delay = Duration::from_secs(60);

        loop {
            match self.connect_and_run().await {
                Ok(_) => {
                    retry_delay = Duration::from_secs(1);
                    info!("Connection closed, reconnecting...");
                }
                Err(e) => {
                    error!(
                        "Connection error: {}, reconnecting in {:?}...",
                        e, retry_delay
                    );
                    tokio::time::sleep(retry_delay).await;

                    retry_delay = std::cmp::min(retry_delay * 2, max_delay);
                }
            }
        }
    }

    async fn connect_and_run(&mut self) -> Result<()> {
        let ws_url = self
            .config
            .server_url
            .replace("https://", "wss://")
            .replace("http://", "ws://");
        let ws_url = format!("{}/ws/agent", ws_url);

        info!("Connecting to {}", ws_url);

        // Build request with optional X-Agent-Key header
        let mut request = ws_url.into_client_request()?;
        if let Some(ref secret) = self.config.agent_secret {
            request.headers_mut().insert(
                HeaderName::from_static("x-agent-key"),
                HeaderValue::from_str(secret).context("Invalid agent secret value")?,
            );
            debug!("Added X-Agent-Key header for authentication");
        }

        let (ws_stream, _) = connect_async_with_config(request, None)
            .await
            .context("Failed to connect to server")?;

        let (mut write, mut read) = ws_stream.split();

        let register_msg = AgentMessage {
            msg_type: "register".to_string(),
            payload: serde_json::to_value(RegisterPayload {
                paw: self.config.paw.clone(),
                hostname: self.sys_info.hostname.clone(),
                username: self.sys_info.username.clone(),
                platform: self.sys_info.platform.clone(),
                executors: self.sys_info.executors.clone(),
            })?,
        };

        write
            .send(WsMessage::Text(serde_json::to_string(&register_msg)?))
            .await?;
        info!("Registered with server");

        let heartbeat_interval = self.config.heartbeat_interval;
        let paw = self.config.paw.clone();

        let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(32);

        let tx_heartbeat = tx.clone();
        tokio::spawn(async move {
            let mut interval = interval(Duration::from_secs(heartbeat_interval));
            loop {
                interval.tick().await;
                let msg = AgentMessage {
                    msg_type: "heartbeat".to_string(),
                    payload: serde_json::json!({ "paw": paw }),
                };
                match serde_json::to_string(&msg) {
                    Ok(json_str) => {
                        if tx_heartbeat.send(json_str).await.is_err() {
                            break;
                        }
                    }
                    Err(e) => {
                        error!("Failed to serialize heartbeat: {}", e);
                        break;
                    }
                }
            }
        });

        loop {
            tokio::select! {
                Some(msg) = rx.recv() => {
                    write.send(WsMessage::Text(msg)).await?;
                }

                msg = read.next() => {
                    match msg {
                        Some(Ok(WsMessage::Text(text))) => {
                            match serde_json::from_str::<AgentMessage>(&text) {
                                Ok(agent_msg) => {
                                    self.handle_message(agent_msg, &tx).await?;
                                }
                                Err(e) => {
                                    warn!("Failed to parse message: {} - content: {}", e, text);
                                }
                            }
                        }
                        Some(Ok(WsMessage::Ping(data))) => {
                            write.send(WsMessage::Pong(data)).await?;
                        }
                        Some(Ok(WsMessage::Close(_))) => {
                            info!("Server closed connection");
                            break;
                        }
                        Some(Err(e)) => {
                            error!("WebSocket error: {}", e);
                            break;
                        }
                        None => break,
                        _ => {}
                    }
                }
            }
        }

        Ok(())
    }

    /// Handles incoming messages from the server.
    pub async fn handle_message(
        &self,
        msg: AgentMessage,
        tx: &tokio::sync::mpsc::Sender<String>,
    ) -> Result<()> {
        debug!("Received message: {:?}", msg.msg_type);

        match msg.msg_type.as_str() {
            "task" => {
                let task: TaskPayload = serde_json::from_value(msg.payload)?;
                self.execute_task(task, tx).await?;
            }
            "ping" => {
                let pong = AgentMessage {
                    msg_type: "pong".to_string(),
                    payload: serde_json::json!({}),
                };
                tx.send(serde_json::to_string(&pong)?).await?;
            }
            _ => {
                warn!("Unknown message type: {}", msg.msg_type);
            }
        }

        Ok(())
    }

    /// Executes a task and sends the result back to the server.
    pub async fn execute_task(
        &self,
        task: TaskPayload,
        tx: &tokio::sync::mpsc::Sender<String>,
    ) -> Result<()> {
        info!(
            "Executing task {} (technique: {})",
            task.id, task.technique_id
        );

        let timeout = task.timeout.unwrap_or(300);
        let result = self
            .executor
            .execute(&task.executor, &task.command, Duration::from_secs(timeout))
            .await;

        let response = AgentMessage {
            msg_type: "task_result".to_string(),
            payload: serde_json::json!({
                "task_id": task.id,
                "technique_id": task.technique_id,
                "success": result.success,
                "output": result.output,
                "exit_code": result.exit_code,
            }),
        };

        tx.send(serde_json::to_string(&response)?).await?;

        if let Some(cleanup) = task.cleanup {
            debug!("Executing cleanup command");
            let _ = self
                .executor
                .execute(&task.executor, &cleanup, Duration::from_secs(30))
                .await;
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::TlsConfig;

    fn create_test_config() -> AgentConfig {
        AgentConfig {
            server_url: "https://test.server:8443".to_string(),
            paw: "test-paw-123".to_string(),
            heartbeat_interval: 30,
            tls: TlsConfig::default(),
            agent_secret: None,
        }
    }

    fn create_test_config_with_secret() -> AgentConfig {
        AgentConfig {
            server_url: "https://test.server:8443".to_string(),
            paw: "test-paw-123".to_string(),
            heartbeat_interval: 30,
            tls: TlsConfig::default(),
            agent_secret: Some("test-secret".to_string()),
        }
    }

    fn create_test_sys_info() -> SystemInfo {
        SystemInfo {
            hostname: "test-host".to_string(),
            username: "test-user".to_string(),
            platform: "linux".to_string(),
            executors: vec!["sh".to_string(), "bash".to_string()],
            os_version: "5.0".to_string(),
            architecture: "x86_64".to_string(),
        }
    }

    #[test]
    fn test_agent_message_serialization() {
        let msg = AgentMessage {
            msg_type: "register".to_string(),
            payload: serde_json::json!({"key": "value"}),
        };

        let json = serde_json::to_string(&msg).unwrap();
        assert!(json.contains("\"type\":\"register\""));
        assert!(json.contains("\"payload\""));
    }

    #[test]
    fn test_agent_message_deserialization() {
        let json = r#"{"type": "task", "payload": {"id": "123"}}"#;
        let msg: AgentMessage = serde_json::from_str(json).unwrap();

        assert_eq!(msg.msg_type, "task");
        assert_eq!(msg.payload["id"], "123");
    }

    #[test]
    fn test_agent_message_debug() {
        let msg = AgentMessage {
            msg_type: "test".to_string(),
            payload: serde_json::json!({}),
        };

        let debug_str = format!("{:?}", msg);
        assert!(debug_str.contains("msg_type"));
    }

    #[test]
    fn test_agent_client_new() {
        let config = create_test_config();
        let sys_info = create_test_sys_info();

        let client = AgentClient::new(config, sys_info);
        assert!(client.is_ok());
    }

    #[test]
    fn test_agent_client_with_secret() {
        let config = create_test_config_with_secret();
        let sys_info = create_test_sys_info();

        let client = AgentClient::new(config, sys_info).unwrap();
        assert_eq!(client.config.agent_secret, Some("test-secret".to_string()));
    }

    #[tokio::test]
    async fn test_handle_message_ping() {
        let config = create_test_config();
        let sys_info = create_test_sys_info();
        let client = AgentClient::new(config, sys_info).unwrap();

        let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(32);

        let msg = AgentMessage {
            msg_type: "ping".to_string(),
            payload: serde_json::json!({}),
        };

        let result = client.handle_message(msg, &tx).await;
        assert!(result.is_ok());

        let response = rx.recv().await.unwrap();
        assert!(response.contains("pong"));
    }

    #[tokio::test]
    async fn test_handle_message_unknown_type() {
        let config = create_test_config();
        let sys_info = create_test_sys_info();
        let client = AgentClient::new(config, sys_info).unwrap();

        let (tx, _rx) = tokio::sync::mpsc::channel::<String>(32);

        let msg = AgentMessage {
            msg_type: "unknown_type".to_string(),
            payload: serde_json::json!({}),
        };

        let result = client.handle_message(msg, &tx).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_message_task() {
        let config = create_test_config();
        let sys_info = create_test_sys_info();
        let client = AgentClient::new(config, sys_info).unwrap();

        let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(32);

        let msg = AgentMessage {
            msg_type: "task".to_string(),
            payload: serde_json::json!({
                "id": "task-test",
                "technique_id": "T1082",
                "command": "echo hello",
                "executor": "sh"
            }),
        };

        let result = client.handle_message(msg, &tx).await;
        assert!(result.is_ok());

        let response = rx.recv().await.unwrap();
        assert!(response.contains("task_result"));
        assert!(response.contains("task-test"));
    }

    #[test]
    fn test_url_conversion_https_to_wss() {
        let url = "https://server:8443".replace("https://", "wss://");
        assert_eq!(url, "wss://server:8443");
    }

    #[test]
    fn test_url_conversion_http_to_ws() {
        let url = "http://server:8080"
            .replace("https://", "wss://")
            .replace("http://", "ws://");
        assert_eq!(url, "ws://server:8080");
    }

    #[test]
    fn test_task_payload_deserialization() {
        let json = r#"{
            "id": "task-1",
            "technique_id": "T1059",
            "command": "echo test",
            "executor": "sh",
            "timeout": 60,
            "cleanup": "rm -f /tmp/test"
        }"#;

        let task: TaskPayload = serde_json::from_str(json).unwrap();
        assert_eq!(task.id, "task-1");
        assert_eq!(task.technique_id, "T1059");
        assert_eq!(task.command, "echo test");
        assert_eq!(task.executor, "sh");
        assert_eq!(task.timeout, Some(60));
        assert_eq!(task.cleanup, Some("rm -f /tmp/test".to_string()));
    }

    #[test]
    fn test_task_payload_optional_fields() {
        let json = r#"{
            "id": "task-2",
            "technique_id": "T1082",
            "command": "uname -a",
            "executor": "bash"
        }"#;

        let task: TaskPayload = serde_json::from_str(json).unwrap();
        assert!(task.timeout.is_none());
        assert!(task.cleanup.is_none());
    }

    #[tokio::test]
    async fn test_execute_task_with_cleanup() {
        let config = create_test_config();
        let sys_info = create_test_sys_info();
        let client = AgentClient::new(config, sys_info).unwrap();

        let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(32);

        let task = TaskPayload {
            id: "cleanup-task".to_string(),
            technique_id: "T1059".to_string(),
            command: "echo main".to_string(),
            executor: "sh".to_string(),
            timeout: Some(5),
            cleanup: Some("echo cleanup".to_string()),
        };

        let result = client.execute_task(task, &tx).await;
        assert!(result.is_ok());

        let response = rx.recv().await.unwrap();
        assert!(response.contains("cleanup-task"));
    }

    #[tokio::test]
    async fn test_execute_task_default_timeout() {
        let config = create_test_config();
        let sys_info = create_test_sys_info();
        let client = AgentClient::new(config, sys_info).unwrap();

        let (tx, mut rx) = tokio::sync::mpsc::channel::<String>(32);

        let task = TaskPayload {
            id: "timeout-task".to_string(),
            technique_id: "T1082".to_string(),
            command: "echo quick".to_string(),
            executor: "sh".to_string(),
            timeout: None,
            cleanup: None,
        };

        let result = client.execute_task(task, &tx).await;
        assert!(result.is_ok());

        let response = rx.recv().await.unwrap();
        assert!(response.contains("timeout-task"));
    }
}
