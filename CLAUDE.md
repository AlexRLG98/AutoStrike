# AutoStrike - Developer Context

## Project Overview

AutoStrike is a **Breach and Attack Simulation (BAS)** platform for security testing based on the **MITRE ATT&CK** framework. It allows SOC teams and security professionals to validate their detection capabilities through automated attack simulations.

## Architecture

### Components

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Dashboard    │────▶│     Server      │◀────│     Agent       │
│  (React + TS)   │     │      (Go)       │     │     (Rust)      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
     Port 3000              Port 8443              WebSocket
```

### Directory Structure

```
autostrike/
├── server/          # Go backend (Hexagonal Architecture)
│   ├── cmd/         # Entry point
│   ├── internal/
│   │   ├── domain/      # Business logic (entities, services, interfaces)
│   │   ├── application/ # Use cases (service orchestration)
│   │   └── infrastructure/ # Adapters (HTTP, SQLite, WebSocket)
│   └── pkg/         # Shared utilities
├── agent/           # Rust agent
│   └── src/         # Client, executor, config, system info
├── dashboard/       # React frontend
│   └── src/
│       ├── components/
│       ├── hooks/       # Custom React hooks (useWebSocket)
│       ├── pages/
│       └── lib/
├── docs/            # MkDocs documentation
└── scripts/         # Build and deployment scripts
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Server** | Go 1.21, Gin, SQLite, WebSocket |
| **Agent** | Rust 1.83+, tokio, tokio-tungstenite |
| **Dashboard** | React 18, TypeScript, TailwindCSS, Vite |
| **Documentation** | MkDocs Material |

## Key Patterns

### Server (Hexagonal Architecture)
- **Domain Layer**: Pure business logic, no external dependencies
- **Application Layer**: Use case orchestration
- **Infrastructure Layer**: External adapters (HTTP, persistence, WebSocket)
- Dependencies flow INWARD toward domain

### Agent (Async Rust)
- Exponential backoff for reconnection
- Platform-specific command execution (PowerShell/bash)
- JSON-based WebSocket protocol

## Development Commands

```bash
# Server
cd server && go build ./...
cd server && go test ./...

# Agent
cd agent && cargo build --release
cd agent && cargo test

# Dashboard
cd dashboard && npm install
cd dashboard && npm run build
cd dashboard && npm run dev

# Full stack (Docker)
docker compose up --build

# Generate TLS certificates
make certs
```

## API Endpoints

Base URL: `https://localhost:8443/api/v1`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/agents` | GET | List all agents |
| `/agents/:paw` | GET | Get agent details |
| `/techniques` | GET | List MITRE techniques |
| `/techniques/tactic/:tactic` | GET | Techniques by tactic |
| `/executions` | GET | List all executions |
| `/executions/:id` | GET | Get execution details |
| `/executions` | POST | Start execution |
| `/executions/:id/results` | GET | Get execution results |
| `/executions/:id/stop` | POST | Stop running execution |
| `/executions/:id/complete` | POST | Mark execution complete |

## WebSocket Protocol

### Agent ↔ Server
Connection: `wss://server:8443/ws/agent`

```json
// Register
{"type": "register", "payload": {"paw": "...", "hostname": "...", ...}}

// Task
{"type": "task", "payload": {"id": "...", "command": "...", "executor": "..."}}

// Result
{"type": "task_result", "payload": {"task_id": "...", "success": true, ...}}
```

### Dashboard ↔ Server
Connection: `wss://server:8443/ws/dashboard`

```json
// Server notifications (execution_cancelled, execution_completed, execution_started)
{"type": "execution_cancelled", "payload": {"execution_id": "...", "data": {}}}
```

## Environment Variables

### Server (.env)
- `JWT_SECRET` - JWT signing key
- `AGENT_SECRET` - Agent authentication
- `DATABASE_PATH` - SQLite database path

### Dashboard (.env)
- `VITE_SERVER_URL` - Backend server URL
- `VITE_API_BASE_URL` - API base path

## Important Notes

1. **Security**: This is a security testing tool. Use only on authorized systems.
2. **Safe Mode**: Always test with `safe_mode: true` first.
3. **mTLS**: Production deployments should use mutual TLS.
4. **Logging**: Server uses structured logging (zap), agent uses tracing.

## Testing

Test coverage:
- Server: Unit tests for services and handlers (`go test ./...`)
- Agent: Unit tests in `executor.rs` (`cargo test`)
- Dashboard: 115 tests (`npm run test`)

## Contributing

1. Follow hexagonal architecture for server changes
2. Use `cargo fmt` and `cargo clippy` for Rust
3. Use `npm run lint` for TypeScript
4. Update documentation in `docs/` for user-facing changes
