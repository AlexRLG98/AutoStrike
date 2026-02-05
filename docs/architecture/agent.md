# Agent (Rust)

The AutoStrike agent is built in **Rust** for performance, memory safety, and cross-platform support.

---

## Features

- **MITRE ATT&CK technique execution** via multiple executors
- **Secure WebSocket communication** with automatic reconnection
- **Multi-platform support**: Windows, Linux, macOS (x64 and ARM64)
- **Automatic platform detection** and executor discovery
- **Auto-cleanup** after technique execution
- **Periodic heartbeat** to maintain connection (default: 30 seconds)
- **Exponential backoff** for reconnection (1s → 60s max)

---

## Structure

```
agent/
├── src/
│   ├── main.rs          # Entry point, CLI (clap)
│   ├── config.rs        # YAML configuration management
│   ├── client.rs        # WebSocket client, protocol handling
│   ├── executor.rs      # Command execution with timeout
│   └── system.rs        # System detection (OS, hostname, executors)
├── Cargo.toml           # Rust dependencies
├── Cargo.lock
└── Dockerfile           # Multi-stage build
```

---

## Dependencies

| Crate | Usage |
|-------|-------|
| `tokio` | Async runtime |
| `tokio-tungstenite` | WebSocket client |
| `serde` / `serde_json` | Serialization |
| `tracing` | Structured logging |
| `sysinfo` | System information |
| `whoami` | Username detection |
| `which` | Executor detection |
| `clap` | CLI parsing |
| `anyhow` | Error handling |
| `uuid` | PAW generation |

---

## Deployment

### Windows

```powershell
.\autostrike-agent.exe --server https://server:8443 --paw AGENT_WIN_001
```

### Linux / macOS

```bash
./autostrike-agent --server https://server:8443 --paw AGENT_LIN_001
```

---

## CLI Options

| Option | Description | Default |
|--------|-------------|---------|
| `-s, --server` | AutoStrike server URL | `https://localhost:8443` |
| `-p, --paw` | Unique agent identifier | Auto-generated UUID |
| `-c, --config` | Configuration file path | `agent.yaml` |
| `-d, --debug` | Enable debug logging | `false` |

---

## Configuration

File `agent.yaml`:

```yaml
server_url: "https://server:8443"
paw: "agent-001"
heartbeat_interval: 30  # seconds

tls:
  cert_file: "./certs/agent.crt"
  key_file: "./certs/agent.key"
  ca_file: "./certs/ca.crt"
  verify: true
```

**Priority:** CLI arguments > Config file > Defaults

---

## System Detection

The agent automatically detects at startup:

| Information | Method |
|-------------|--------|
| Hostname | `sysinfo::System::host_name()` |
| Username | `whoami::username()` |
| Platform | `cfg!(target_os)` compile-time |
| OS Version | `sysinfo::System::os_version()` |
| Architecture | `std::env::consts::ARCH` |

### Executor Detection

| Windows | Linux/macOS |
|---------|-------------|
| `powershell` | `sh` |
| `pwsh` | `bash` |
| `cmd` | `zsh` |
| | `python3` |

Detection uses the `which` crate to verify executors exist in PATH.

---

## WebSocket Protocol

### Registration (Agent → Server)
```json
{
  "type": "register",
  "payload": {
    "paw": "agent-001",
    "hostname": "DESKTOP-ABC",
    "username": "admin",
    "platform": "windows",
    "executors": ["powershell", "cmd"]
  }
}
```

### Registration Acknowledgment (Server → Agent)
```json
{
  "type": "registered",
  "payload": {
    "status": "ok",
    "paw": "agent-001"
  }
}
```

### Heartbeat (Agent → Server, every 30s)
```json
{
  "type": "heartbeat",
  "payload": {
    "paw": "agent-001"
  }
}
```

### Task (Server → Agent)
```json
{
  "type": "task",
  "payload": {
    "id": "task-uuid",
    "technique_id": "T1082",
    "command": "systeminfo",
    "executor": "cmd",
    "timeout": 300,
    "cleanup": "del /f output.txt"
  }
}
```

### Task Result (Agent → Server)
```json
{
  "type": "task_result",
  "payload": {
    "task_id": "task-uuid",
    "technique_id": "T1082",
    "success": true,
    "output": "Host Name: DESKTOP-ABC...",
    "exit_code": 0,
    "error": ""
  }
}
```

---

## Connection Lifecycle

```
1. Connect to wss://server:8443/ws/agent
2. Send "register" message with system info
3. Receive "registered" acknowledgment
4. Start heartbeat loop (every 30 seconds)
5. Wait for "task" messages
6. Execute command with timeout
7. Send "task_result"
8. Run cleanup command (if provided)
9. Continue waiting for tasks
```

### Reconnection Strategy

On connection failure, the agent uses exponential backoff:

```
Initial delay: 1 second
After failure: delay × 2
Maximum delay: 60 seconds
On success: reset to 1 second
```

The agent will retry indefinitely until connection is restored.

---

## Command Execution

### Timeout Handling

- Commands have a configurable timeout (default: 300 seconds)
- On timeout: returns `success: false`, `exit_code: None`, `output: "Command timed out"`

### Platform-Specific Executors

**Windows:**
| Executor | Command | Arguments |
|----------|---------|-----------|
| powershell | powershell.exe | -NoProfile -NonInteractive -Command |
| pwsh | pwsh.exe | -NoProfile -NonInteractive -Command |
| cmd | cmd.exe | /C |

**Linux/macOS:**
| Executor | Shell |
|----------|-------|
| bash | /bin/bash -c |
| sh | /bin/sh -c |
| zsh | /bin/zsh -c |

### Output Handling

- stdout and stderr are captured separately
- Combined into single output string
- UTF-8 decoding with lossy conversion
- Trimmed of leading/trailing whitespace

---

## Cross-Compilation

Build for multiple platforms:

```bash
# Linux x64
cargo build --release --target x86_64-unknown-linux-gnu

# Linux ARM64
cargo build --release --target aarch64-unknown-linux-gnu

# Windows x64
cargo build --release --target x86_64-pc-windows-gnu

# macOS x64
cargo build --release --target x86_64-apple-darwin

# macOS ARM64 (M1/M2)
cargo build --release --target aarch64-apple-darwin
```

### Supported Targets

| Target | OS | Architecture |
|--------|-----|--------------|
| `x86_64-unknown-linux-gnu` | Linux | x64 |
| `x86_64-unknown-linux-musl` | Linux (static) | x64 |
| `x86_64-pc-windows-gnu` | Windows | x64 |
| `x86_64-apple-darwin` | macOS | x64 |
| `aarch64-unknown-linux-gnu` | Linux | ARM64 |
| `aarch64-apple-darwin` | macOS M1/M2 | ARM64 |

---

## Testing

67 unit tests across all modules:

```bash
cd agent
cargo test           # Run all tests
cargo test -- --nocapture  # With output
```

Test coverage:
- CLI argument parsing
- Configuration loading and merging
- Message serialization/deserialization
- Command execution
- System info gathering
- Reconnection logic

---

## Security

- **TLS/mTLS**: Encrypted communication with optional client certificates
- **No hardcoded credentials**: Configuration via file or CLI
- **Automatic cleanup**: Cleanup commands run after each technique
- **Non-root recommended**: Run without elevated privileges when possible
- **Timeout protection**: Prevents command hangs
