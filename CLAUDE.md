# AutoStrike - Developer Context

## Project Overview

AutoStrike is a **Breach and Attack Simulation (BAS)** platform for security testing based on the **MITRE ATT&CK** framework. It allows SOC teams and security professionals to validate their detection capabilities through automated attack simulations.

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Server (Go) - Port 8443                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Dashboard  │  │  REST API   │  │  WebSocket  │              │
│  │  (Static)   │  │  /api/v1/*  │  │  /ws/*      │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
                              ▲
                              │ WebSocket
                    ┌─────────┴─────────┐
                    │      Agent        │
                    │     (Rust)        │
                    └───────────────────┘
```

**Single server on port 8443** serves:
- Dashboard (static files from `dashboard/dist`)
- REST API (`/api/v1/*`)
- WebSocket (`/ws/agent`, `/ws/dashboard`)
- Health check (`/health`)

### Directory Structure

```
autostrike/
├── server/          # Go backend (Hexagonal Architecture)
│   ├── cmd/         # Entry point
│   ├── configs/     # YAML technique definitions
│   │   └── techniques/  # discovery.yaml, execution.yaml, etc.
│   ├── internal/
│   │   ├── domain/      # Business logic (entities, services, interfaces)
│   │   ├── application/ # Use cases (service orchestration)
│   │   └── infrastructure/ # Adapters (HTTP, SQLite, WebSocket)
│   └── pkg/         # Shared utilities
├── agent/           # Rust agent
│   └── src/         # Client, executor, config, system info
├── dashboard/       # React frontend
│   └── src/
│       ├── components/  # MitreMatrix, RunExecutionModal, Layout, ProtectedRoute
│       ├── contexts/    # AuthContext (authentication state)
│       ├── hooks/       # useWebSocket
│       ├── pages/       # Dashboard, Agents, Techniques, Matrix, Scenarios, Executions, ExecutionDetails, Settings, Login, Analytics, Scheduler, Admin/*
│       └── lib/         # API client
├── docs/            # MkDocs documentation
└── scripts/         # Build and deployment scripts
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Server** | Go 1.21+, Gin, SQLite, gorilla/websocket |
| **Agent** | Rust 1.75+, tokio, tokio-tungstenite |
| **Dashboard** | React 18, TypeScript, TailwindCSS, Vite, TanStack Query, Chart.js |
| **Documentation** | MkDocs Material |

## Key Patterns

### Server (Hexagonal Architecture)
- **Domain Layer**: Pure business logic, no external dependencies
- **Application Layer**: Use case orchestration
- **Infrastructure Layer**: External adapters (HTTP, persistence, WebSocket)
- Dependencies flow INWARD toward domain

### Agent (Async Rust)
- Exponential backoff for reconnection (1s → 60s max)
- Platform-specific command execution (PowerShell/cmd/bash/sh)
- JSON-based WebSocket protocol
- Heartbeat every 30 seconds

### Dashboard (React)
- TanStack Query for server state
- WebSocket for real-time updates
- Conditional polling as fallback

## Development Commands

```bash
# Quick Start (recommended)
make run        # Build and start everything on https://localhost:8443
make agent      # Connect an agent
make stop       # Stop all services
make logs       # View server logs

# Server
cd server && go build ./...
cd server && go test ./... -cover

# Agent
cd agent && cargo build --release
cd agent && cargo test

# Dashboard
cd dashboard && npm install
cd dashboard && npm run build
cd dashboard && npm test -- --run
cd dashboard && npm run lint
cd dashboard && npm run type-check

# Full stack (Docker)
docker compose up --build

# Generate TLS certificates
make certs
```

## API Endpoints

Base URL: `https://localhost:8443/api/v1`

### Authentication (public routes)
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/login` | POST | Login with username/password |
| `/auth/refresh` | POST | Refresh access token |
| `/auth/logout` | POST | Invalidate tokens |
| `/auth/me` | GET | Get current user info (requires token) |

### Core API (protected when auth enabled)
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (returns `{"status": "ok", "auth_enabled": bool}`) |
| `/agents` | GET | List agents (`?all=true` for offline too) |
| `/agents/:paw` | GET | Get agent details |
| `/agents` | POST | Register agent |
| `/agents/:paw` | DELETE | Delete agent |
| `/agents/:paw/heartbeat` | POST | Update last_seen |
| `/techniques` | GET | List all techniques |
| `/techniques/:id` | GET | Get technique by ID |
| `/techniques/tactic/:tactic` | GET | Techniques by tactic |
| `/techniques/platform/:platform` | GET | Techniques by platform |
| `/techniques/coverage` | GET | MITRE coverage stats |
| `/techniques/import` | POST | Import from YAML |
| `/scenarios` | GET | List scenarios |
| `/scenarios/:id` | GET | Get scenario |
| `/scenarios/tag/:tag` | GET | Scenarios by tag |
| `/scenarios` | POST | Create scenario |
| `/scenarios/:id` | PUT | Update scenario |
| `/scenarios/:id` | DELETE | Delete scenario |
| `/executions` | GET | List executions (limit 50) |
| `/executions/:id` | GET | Get execution |
| `/executions` | POST | Start execution |
| `/executions/:id/results` | GET | Get results |
| `/executions/:id/stop` | POST | Stop execution |
| `/executions/:id/complete` | POST | Complete execution |

### Admin API (requires admin role)
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/users` | GET | List all users |
| `/admin/users/:id` | GET | Get user by ID |
| `/admin/users` | POST | Create user |
| `/admin/users/:id` | PUT | Update user |
| `/admin/users/:id/role` | PUT | Update user role |
| `/admin/users/:id` | DELETE | Deactivate user |
| `/admin/users/:id/reactivate` | POST | Reactivate user |
| `/admin/users/:id/reset-password` | POST | Reset password |

### Schedules API
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/schedules` | GET | List all schedules |
| `/schedules/:id` | GET | Get schedule by ID |
| `/schedules` | POST | Create schedule |
| `/schedules/:id` | PUT | Update schedule |
| `/schedules/:id` | DELETE | Delete schedule |
| `/schedules/:id/pause` | POST | Pause schedule |
| `/schedules/:id/resume` | POST | Resume schedule |
| `/schedules/:id/run` | POST | Run schedule now |
| `/schedules/:id/runs` | GET | Get schedule run history |

### Notifications API
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/notifications` | GET | Get notifications |
| `/notifications/unread/count` | GET | Get unread count |
| `/notifications/:id/read` | POST | Mark as read |
| `/notifications/read-all` | POST | Mark all as read |
| `/notifications/settings` | GET/POST/PUT/DELETE | Manage notification settings |
| `/notifications/smtp` | GET | Get SMTP config (admin) |
| `/notifications/smtp/test` | POST | Test SMTP (admin) |

### Analytics API
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/analytics/period` | GET | Get period stats |
| `/analytics/comparison` | GET | Compare periods |
| `/analytics/trend` | GET | Get score trend |
| `/analytics/summary` | GET | Get execution summary |

### Permissions API
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/permissions/matrix` | GET | Get permission matrix |
| `/permissions/me` | GET | Get my permissions |
| `/permissions/check` | POST | Check permission |

## WebSocket Protocol

### Agent ↔ Server
Connection: `wss://server:8443/ws/agent`

```json
// Register (Agent → Server)
{"type": "register", "payload": {"paw": "...", "hostname": "...", "platform": "...", "executors": [...]}}

// Registered (Server → Agent)
{"type": "registered", "payload": {"status": "ok", "paw": "..."}}

// Heartbeat (Agent → Server, every 30s)
{"type": "heartbeat", "payload": {"paw": "..."}}

// Task (Server → Agent)
{"type": "task", "payload": {"id": "...", "technique_id": "...", "command": "...", "executor": "...", "timeout": 300}}

// Task Result (Agent → Server)
{"type": "task_result", "payload": {"task_id": "...", "technique_id": "...", "success": true, "output": "...", "exit_code": 0}}

// Task Ack (Server → Agent)
{"type": "task_ack", "payload": {"task_id": "...", "status": "received"}}
```

### Dashboard ↔ Server
Connection: `wss://server:8443/ws/dashboard`

```json
// Server broadcasts to all dashboards
{"type": "execution_started", "payload": {"execution_id": "...", "data": {...}}}
{"type": "execution_completed", "payload": {"execution_id": "...", "data": {...}}}
{"type": "execution_cancelled", "payload": {"execution_id": "...", "data": {...}}}

// Dashboard can send ping
{"type": "ping", "payload": {}}
// Server responds with pong
{"type": "pong", "payload": {}}
```

## Environment Variables

### Server (.env)
- `DATABASE_PATH` - SQLite database path (default: `./data/autostrike.db`)
- `DASHBOARD_PATH` - Path to dashboard dist folder (default: `../dashboard/dist`)
- `JWT_SECRET` - JWT signing key (optional - auth disabled if not set)
- `AGENT_SECRET` - Agent authentication secret
- `ENABLE_AUTH` - Explicit auth override (`true`/`false`)
- `ALLOWED_ORIGINS` - CORS origins (default: `localhost:3000,localhost:8443`)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

**Authentication behavior:**
- `JWT_SECRET` not set → Auth **disabled** (development mode)
- `JWT_SECRET` set → Auth **enabled** (production mode)

### Dashboard (.env) - Only for Vite dev server
- `VITE_SERVER_URL` - Backend server URL
- `VITE_API_BASE_URL` - API base path
- `VITE_WS_HOST` - WebSocket host override

## Available Techniques (15 total)

| Tactic | Count | IDs |
|--------|-------|-----|
| Discovery | 9 | T1082, T1083, T1057, T1016, T1049, T1087, T1069, T1018, T1007 |
| Execution | 3 | T1059.001, T1059.003, T1059.004 |
| Persistence | 2 | T1053.005, T1547.001 |
| Defense Evasion | 1 | T1070.004 |

All techniques are **safe mode compatible** (non-destructive).

## Important Notes

1. **Security**: This is a security testing tool. Use only on authorized systems.
2. **Safe Mode**: Always test with `safe_mode: true` first.
3. **mTLS**: Production deployments should use mutual TLS.
4. **Logging**: Server uses structured logging (zap), agent uses tracing.

## Testing

**Test coverage (Phase 3):**
- **Server**: 200+ tests
  - application: 83.0%
  - entity: 95.0%
  - service: 99.2%
  - handlers: 87.5%
  - websocket: 91.6%
  - middleware: 94.3%
  - rest/server: 87.9%
  - sqlite: 85.0%
- **Agent**: 67 unit tests (`cargo test`)
- **Dashboard**: 513 tests across 25 files (`npm test`)

```bash
# Run all tests
cd server && go test ./... -cover
cd agent && cargo test
cd dashboard && npm test -- --run
```

## Security Score Formula

```
score = (blocked * 100 + detected * 50) / (total * 100) * 100
```

| Result | Points | Meaning |
|--------|--------|---------|
| Blocked | 100 | Attack prevented |
| Detected | 50 | Attack seen but not stopped |
| Success | 0 | Attack succeeded undetected |

## Contributing

1. Follow hexagonal architecture for server changes
2. Use `cargo fmt` and `cargo clippy` for Rust
3. Use `npm run lint` and `npm run type-check` for TypeScript
4. Add tests for new functionality
5. Update documentation in `docs/` for user-facing changes
