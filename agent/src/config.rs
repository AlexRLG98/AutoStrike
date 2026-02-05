//! Agent configuration management.

use anyhow::Result;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Agent configuration loaded from file or CLI arguments.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentConfig {
    /// URL of the AutoStrike server.
    pub server_url: String,
    /// Unique agent identifier (PAW).
    pub paw: String,
    /// Heartbeat interval in seconds.
    pub heartbeat_interval: u64,
    /// TLS configuration for secure connections.
    pub tls: TlsConfig,
    /// Agent authentication secret (X-Agent-Key header).
    #[serde(default)]
    pub agent_secret: Option<String>,
}

/// TLS configuration for secure server connections.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TlsConfig {
    /// Path to client certificate file.
    pub cert_file: Option<String>,
    /// Path to client private key file.
    pub key_file: Option<String>,
    /// Path to CA certificate file for server verification.
    pub ca_file: Option<String>,
    /// Whether to verify server certificate.
    pub verify: bool,
}

impl Default for TlsConfig {
    fn default() -> Self {
        Self {
            cert_file: None,
            key_file: None,
            ca_file: None,
            verify: true,
        }
    }
}

impl AgentConfig {
    /// Loads configuration from file with CLI argument overrides.
    ///
    /// Priority: CLI argument > config file > generated default.
    pub fn load(
        path: &str,
        server: &str,
        paw: Option<String>,
        agent_secret: Option<String>,
    ) -> Result<Self> {
        // Try to load from file first
        let file_config = if std::path::Path::new(path).exists() {
            let mut settings = config::Config::default();
            settings.merge(config::File::with_name(path))?;
            Some(settings.try_into::<AgentConfig>()?)
        } else {
            None
        };

        // Priority: CLI arg > config file > generate new
        let resolved_paw = paw
            .or_else(|| file_config.as_ref().map(|c| c.paw.clone()))
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        // Priority: CLI arg > config file > None
        let resolved_secret =
            agent_secret.or_else(|| file_config.as_ref().and_then(|c| c.agent_secret.clone()));

        Ok(AgentConfig {
            server_url: server.to_string(),
            paw: resolved_paw,
            heartbeat_interval: file_config
                .as_ref()
                .map(|c| c.heartbeat_interval)
                .unwrap_or(30),
            tls: file_config
                .as_ref()
                .map(|c| c.tls.clone())
                .unwrap_or_default(),
            agent_secret: resolved_secret,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_tls_config_default() {
        let tls = TlsConfig::default();
        assert!(tls.cert_file.is_none());
        assert!(tls.key_file.is_none());
        assert!(tls.ca_file.is_none());
        assert!(tls.verify);
    }

    #[test]
    fn test_tls_config_with_values() {
        let tls = TlsConfig {
            cert_file: Some("/path/to/cert.pem".to_string()),
            key_file: Some("/path/to/key.pem".to_string()),
            ca_file: Some("/path/to/ca.pem".to_string()),
            verify: false,
        };
        assert_eq!(tls.cert_file.as_deref(), Some("/path/to/cert.pem"));
        assert_eq!(tls.key_file.as_deref(), Some("/path/to/key.pem"));
        assert_eq!(tls.ca_file.as_deref(), Some("/path/to/ca.pem"));
        assert!(!tls.verify);
    }

    #[test]
    fn test_load_with_cli_paw() {
        let config = AgentConfig::load(
            "nonexistent.yaml",
            "https://test.server:8443",
            Some("custom-paw".to_string()),
            None,
        )
        .unwrap();

        assert_eq!(config.server_url, "https://test.server:8443");
        assert_eq!(config.paw, "custom-paw");
        assert_eq!(config.heartbeat_interval, 30);
        assert!(config.tls.verify);
        assert!(config.agent_secret.is_none());
    }

    #[test]
    fn test_load_with_agent_secret() {
        let config = AgentConfig::load(
            "nonexistent.yaml",
            "https://test.server:8443",
            Some("paw".to_string()),
            Some("my-secret".to_string()),
        )
        .unwrap();

        assert_eq!(config.agent_secret, Some("my-secret".to_string()));
    }

    #[test]
    fn test_load_generates_uuid_paw() {
        let config =
            AgentConfig::load("nonexistent.yaml", "https://server:8443", None, None).unwrap();

        assert!(!config.paw.is_empty());
        assert!(Uuid::parse_str(&config.paw).is_ok());
    }

    #[test]
    fn test_load_uses_server_url() {
        let config =
            AgentConfig::load("nonexistent.yaml", "https://custom.server:9999", None, None)
                .unwrap();

        assert_eq!(config.server_url, "https://custom.server:9999");
    }

    #[test]
    fn test_load_default_heartbeat() {
        let config =
            AgentConfig::load("nonexistent.yaml", "https://server:8443", None, None).unwrap();

        assert_eq!(config.heartbeat_interval, 30);
    }

    #[test]
    fn test_config_clone() {
        let config = AgentConfig {
            server_url: "https://server:8443".to_string(),
            paw: "test-paw".to_string(),
            heartbeat_interval: 60,
            tls: TlsConfig::default(),
            agent_secret: Some("secret".to_string()),
        };

        let cloned = config.clone();
        assert_eq!(cloned.server_url, config.server_url);
        assert_eq!(cloned.paw, config.paw);
        assert_eq!(cloned.heartbeat_interval, config.heartbeat_interval);
        assert_eq!(cloned.agent_secret, config.agent_secret);
    }

    #[test]
    fn test_config_debug() {
        let config = AgentConfig {
            server_url: "https://server:8443".to_string(),
            paw: "test-paw".to_string(),
            heartbeat_interval: 30,
            tls: TlsConfig::default(),
            agent_secret: None,
        };

        let debug_str = format!("{:?}", config);
        assert!(debug_str.contains("server_url"));
        assert!(debug_str.contains("paw"));
    }

    #[test]
    fn test_tls_config_clone() {
        let tls = TlsConfig {
            cert_file: Some("cert.pem".to_string()),
            key_file: Some("key.pem".to_string()),
            ca_file: None,
            verify: true,
        };

        let cloned = tls.clone();
        assert_eq!(cloned.cert_file, tls.cert_file);
        assert_eq!(cloned.key_file, tls.key_file);
        assert_eq!(cloned.ca_file, tls.ca_file);
        assert_eq!(cloned.verify, tls.verify);
    }

    #[test]
    fn test_load_from_config_file() {
        use std::fs;
        use std::io::Write;

        let temp_dir = std::env::temp_dir();
        let config_path = temp_dir.join("test_agent_config_integration.yaml");

        let config_content = r#"
server_url: "https://file-server:8443"
paw: "file-paw-123"
heartbeat_interval: 45
tls:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  ca_file: ~
  verify: false
agent_secret: "file-secret"
"#;

        let mut file = fs::File::create(&config_path).unwrap();
        file.write_all(config_content.as_bytes()).unwrap();

        let config = AgentConfig::load(
            config_path.to_str().unwrap(),
            "https://cli-server:8443",
            None,
            None,
        )
        .unwrap();

        assert_eq!(config.server_url, "https://cli-server:8443");
        assert_eq!(config.paw, "file-paw-123");
        assert_eq!(config.heartbeat_interval, 45);
        assert_eq!(config.tls.cert_file.as_deref(), Some("/path/to/cert.pem"));
        assert!(!config.tls.verify);
        assert_eq!(config.agent_secret, Some("file-secret".to_string()));

        fs::remove_file(&config_path).ok();
    }

    #[test]
    fn test_load_cli_paw_overrides_file() {
        use std::fs;
        use std::io::Write;

        let temp_dir = std::env::temp_dir();
        let config_path = temp_dir.join("test_agent_config_override_integration.yaml");

        let config_content = r#"
server_url: "https://file-server:8443"
paw: "file-paw-456"
heartbeat_interval: 30
tls:
  verify: true
"#;

        let mut file = fs::File::create(&config_path).unwrap();
        file.write_all(config_content.as_bytes()).unwrap();

        let config = AgentConfig::load(
            config_path.to_str().unwrap(),
            "https://server:8443",
            Some("cli-paw-789".to_string()),
            None,
        )
        .unwrap();

        assert_eq!(config.paw, "cli-paw-789");

        fs::remove_file(&config_path).ok();
    }

    #[test]
    fn test_load_cli_secret_overrides_file() {
        use std::fs;
        use std::io::Write;

        let temp_dir = std::env::temp_dir();
        let config_path = temp_dir.join("test_agent_config_secret_override.yaml");

        let config_content = r#"
server_url: "https://file-server:8443"
paw: "file-paw"
heartbeat_interval: 30
tls:
  verify: true
agent_secret: "file-secret"
"#;

        let mut file = fs::File::create(&config_path).unwrap();
        file.write_all(config_content.as_bytes()).unwrap();

        let config = AgentConfig::load(
            config_path.to_str().unwrap(),
            "https://server:8443",
            None,
            Some("cli-secret".to_string()),
        )
        .unwrap();

        assert_eq!(config.agent_secret, Some("cli-secret".to_string()));

        fs::remove_file(&config_path).ok();
    }

    #[test]
    fn test_config_serialization() {
        let config = AgentConfig {
            server_url: "https://server:8443".to_string(),
            paw: "test-paw".to_string(),
            heartbeat_interval: 60,
            tls: TlsConfig::default(),
            agent_secret: Some("test-secret".to_string()),
        };

        let json = serde_json::to_string(&config).unwrap();
        assert!(json.contains("server_url"));
        assert!(json.contains("paw"));
        assert!(json.contains("heartbeat_interval"));
        assert!(json.contains("agent_secret"));
    }

    #[test]
    fn test_config_deserialization() {
        let json = r#"{
            "server_url": "https://server:8443",
            "paw": "deserialized-paw",
            "heartbeat_interval": 120,
            "tls": {
                "cert_file": null,
                "key_file": null,
                "ca_file": null,
                "verify": true
            }
        }"#;

        let config: AgentConfig = serde_json::from_str(json).unwrap();
        assert_eq!(config.server_url, "https://server:8443");
        assert_eq!(config.paw, "deserialized-paw");
        assert_eq!(config.heartbeat_interval, 120);
        assert!(config.agent_secret.is_none()); // Default is None
    }

    #[test]
    fn test_config_deserialization_with_secret() {
        let json = r#"{
            "server_url": "https://server:8443",
            "paw": "paw",
            "heartbeat_interval": 30,
            "tls": { "verify": true },
            "agent_secret": "my-secret"
        }"#;

        let config: AgentConfig = serde_json::from_str(json).unwrap();
        assert_eq!(config.agent_secret, Some("my-secret".to_string()));
    }
}
