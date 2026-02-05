//! AutoStrike Agent - Breach and Attack Simulation agent.
//!
//! This agent connects to the AutoStrike server via WebSocket and executes
//! MITRE ATT&CK techniques for security testing purposes.

mod client;
mod config;
mod executor;
mod system;

use anyhow::Result;
use clap::Parser;
use tracing::{error, info};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use client::AgentClient;
use config::AgentConfig;
use system::SystemInfo;

/// Command-line arguments for the AutoStrike agent.
#[derive(Parser, Debug)]
#[command(name = "autostrike-agent")]
#[command(about = "AutoStrike BAS Agent for security testing")]
struct Args {
    /// Server URL
    #[arg(short, long, default_value = "https://localhost:8443")]
    server: String,

    /// Agent PAW (unique identifier)
    #[arg(short, long)]
    paw: Option<String>,

    /// Configuration file path
    #[arg(short, long, default_value = "agent.yaml")]
    config: String,

    /// Enable debug logging
    #[arg(short, long)]
    debug: bool,

    /// Agent authentication secret (X-Agent-Key header)
    #[arg(short = 'k', long)]
    agent_secret: Option<String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();

    // Initialize logging
    let log_level = if args.debug { "debug" } else { "info" };
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(log_level))
        .with(tracing_subscriber::fmt::layer())
        .init();

    info!("AutoStrike Agent starting...");

    // Load configuration
    let config = AgentConfig::load(&args.config, &args.server, args.paw, args.agent_secret)?;
    info!("Configuration loaded");

    // Gather system information
    let sys_info = SystemInfo::gather();
    info!(
        hostname = %sys_info.hostname,
        platform = %sys_info.platform,
        "System information gathered"
    );

    // Create and run agent client
    let mut client = AgentClient::new(config, sys_info)?;

    if let Err(e) = client.run().await {
        error!("Agent error: {}", e);
        return Err(e);
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_args_default_values() {
        let args = Args::try_parse_from(["autostrike-agent"]).unwrap();

        assert_eq!(args.server, "https://localhost:8443");
        assert!(args.paw.is_none());
        assert_eq!(args.config, "agent.yaml");
        assert!(!args.debug);
        assert!(args.agent_secret.is_none());
    }

    #[test]
    fn test_args_with_server() {
        let args =
            Args::try_parse_from(["autostrike-agent", "--server", "https://custom:9999"]).unwrap();

        assert_eq!(args.server, "https://custom:9999");
    }

    #[test]
    fn test_args_with_short_server() {
        let args = Args::try_parse_from(["autostrike-agent", "-s", "https://short:8080"]).unwrap();

        assert_eq!(args.server, "https://short:8080");
    }

    #[test]
    fn test_args_with_paw() {
        let args = Args::try_parse_from(["autostrike-agent", "--paw", "my-custom-paw"]).unwrap();

        assert_eq!(args.paw, Some("my-custom-paw".to_string()));
    }

    #[test]
    fn test_args_with_short_paw() {
        let args = Args::try_parse_from(["autostrike-agent", "-p", "short-paw"]).unwrap();

        assert_eq!(args.paw, Some("short-paw".to_string()));
    }

    #[test]
    fn test_args_with_config() {
        let args =
            Args::try_parse_from(["autostrike-agent", "--config", "/etc/agent.yaml"]).unwrap();

        assert_eq!(args.config, "/etc/agent.yaml");
    }

    #[test]
    fn test_args_with_short_config() {
        let args = Args::try_parse_from(["autostrike-agent", "-c", "custom.yaml"]).unwrap();

        assert_eq!(args.config, "custom.yaml");
    }

    #[test]
    fn test_args_with_debug() {
        let args = Args::try_parse_from(["autostrike-agent", "--debug"]).unwrap();

        assert!(args.debug);
    }

    #[test]
    fn test_args_with_short_debug() {
        let args = Args::try_parse_from(["autostrike-agent", "-d"]).unwrap();

        assert!(args.debug);
    }

    #[test]
    fn test_args_with_agent_secret() {
        let args =
            Args::try_parse_from(["autostrike-agent", "--agent-secret", "my-secret"]).unwrap();

        assert_eq!(args.agent_secret, Some("my-secret".to_string()));
    }

    #[test]
    fn test_args_with_short_agent_secret() {
        let args = Args::try_parse_from(["autostrike-agent", "-k", "short-secret"]).unwrap();

        assert_eq!(args.agent_secret, Some("short-secret".to_string()));
    }

    #[test]
    fn test_args_all_options() {
        let args = Args::try_parse_from([
            "autostrike-agent",
            "-s",
            "https://server:443",
            "-p",
            "agent-paw",
            "-c",
            "config.yaml",
            "-d",
            "-k",
            "secret-key",
        ])
        .unwrap();

        assert_eq!(args.server, "https://server:443");
        assert_eq!(args.paw, Some("agent-paw".to_string()));
        assert_eq!(args.config, "config.yaml");
        assert!(args.debug);
        assert_eq!(args.agent_secret, Some("secret-key".to_string()));
    }

    #[test]
    fn test_args_debug_trait() {
        let args = Args::try_parse_from(["autostrike-agent"]).unwrap();
        let debug_str = format!("{:?}", args);

        assert!(debug_str.contains("server"));
        assert!(debug_str.contains("paw"));
        assert!(debug_str.contains("config"));
        assert!(debug_str.contains("debug"));
        assert!(debug_str.contains("agent_secret"));
    }

    #[test]
    fn test_log_level_selection() {
        let debug_mode = true;
        let log_level = if debug_mode { "debug" } else { "info" };
        assert_eq!(log_level, "debug");

        let info_mode = false;
        let log_level = if info_mode { "debug" } else { "info" };
        assert_eq!(log_level, "info");
    }
}
