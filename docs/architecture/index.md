# Architecture

AutoStrike uses a 3-tier architecture with three main components.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                     PRESENTATION LAYER                               │
│                     Dashboard (React + TypeScript)                   │
│                     Served on port 8443 by Go server                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                            HTTPS / WebSocket
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      SERVICE LAYER (API)                            │
│                     Control Server (Go + Gin)                       │
│                          Port 8443                                  │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│   │  REST API   │  │  WebSocket  │  │ Orchestrator│                │
│   │  /api/v1/*  │  │  /ws/*      │  │             │                │
│   └─────────────┘  └─────────────┘  └─────────────┘                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                              WebSocket (TLS)
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          AGENTS (Rust)                              │
│                                                                     │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │
│   │   Windows   │  │    Linux    │  │    macOS    │                │
│   │   Agent     │  │    Agent    │  │    Agent    │                │
│   └─────────────┘  └─────────────┘  └─────────────┘                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Components

| Component | Language | Role |
|-----------|----------|------|
| [Dashboard](dashboard.md) | React/TypeScript | User interface (11 pages) |
| [Backend](backend.md) | Go | API, orchestration, storage, WebSocket hub |
| [Agent](agent.md) | Rust | Technique execution on endpoints |

---

## Communication

| Path | Protocol | Purpose |
|------|----------|---------|
| Dashboard ↔ Backend | HTTPS + WebSocket | REST API + real-time updates |
| Backend ↔ Agents | WebSocket (TLS) | Commands and results |

### Single Port Architecture

The Go server on **port 8443** handles everything:
- Static files (Dashboard)
- REST API (`/api/v1/*`)
- WebSocket for agents (`/ws/agent`)
- WebSocket for dashboard (`/ws/dashboard`)
- Health check (`/health`)

---

## Data Flow

```
1. User clicks "Run" on a scenario in Dashboard
2. Dashboard sends POST /api/v1/executions
3. Server creates execution, plans tasks
4. Server broadcasts "execution_started" to dashboards
5. Server sends "task" to each agent via WebSocket
6. Agent executes command, sends "task_result"
7. Server updates result, checks completion
8. Server broadcasts "execution_completed" when done
9. Dashboard auto-refreshes via WebSocket event
```

---

## Key Technologies

| Layer | Technologies |
|-------|--------------|
| Frontend | React 18, TypeScript, TailwindCSS, TanStack Query, Chart.js |
| Backend | Go 1.21+, Gin, gorilla/websocket, SQLite, zap logger |
| Agent | Rust 1.75+, tokio, tokio-tungstenite, serde, tracing |
