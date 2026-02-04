# AutoStrike

## Breach and Attack Simulation (BAS) Platform

**EIP Project - EPITECH Promotion 2028**

---

## What is AutoStrike?

AutoStrike is an open-source platform for **continuous security defense validation** through attack simulations based on the **MITRE ATT&CK** framework.

### Key Features

| Feature | Description |
|---------|-------------|
| **MITRE ATT&CK Matrix** | Interactive visualization of detection coverage |
| **Attack Scenarios** | Automated technique execution with phases |
| **Multi-platform Agents** | Windows and Linux support |
| **Real-time Dashboard** | Live execution monitoring via WebSocket |
| **Security Scoring** | Quantified defense effectiveness |
| **Safe Mode** | Execute only non-destructive techniques |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│              Control Server (Go) - Port 8443                │
│                                                             │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│   │Dashboard │  │ REST API │  │WebSocket │  │Orchestrat.│  │
│   │ (React)  │  │/api/v1/* │  │  /ws/*   │  │           │  │
│   └──────────┘  └──────────┘  └──────────┘  └───────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
        ┌──────────┐   ┌──────────┐   ┌──────────┐
        │  Agent   │   │  Agent   │   │  Agent   │
        │ (Rust)   │   │ (Rust)   │   │ (Rust)   │
        │ Windows  │   │  Linux   │   │  macOS   │
        └──────────┘   └──────────┘   └──────────┘
```

A single server on **port 8443** serves the Dashboard, REST API, and WebSocket connections.

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Frontend** | React 18, TypeScript, TailwindCSS, Chart.js |
| **Backend** | Go 1.21+, Gin Framework, SQLite |
| **Agent** | Rust 1.75+, Tokio, tokio-tungstenite |
| **Communication** | REST API, WebSocket (real-time), TLS |

---

## Quick Start

```bash
# Clone and install
git clone https://github.com/AutoStrike-EIP/AutoStrike_POC.git
cd AutoStrike_POC
make install

# Start server and dashboard
make run

# Connect an agent (in another terminal)
make agent
```

Access the dashboard at **https://localhost:8443**

See the [Quick Start Guide](guide/quickstart.md) for detailed instructions.

---

## Dashboard Pages

| Page | Description |
|------|-------------|
| **Dashboard** | Overview with stats and recent activity |
| **Agents** | Connected agents and their status |
| **Techniques** | Browse MITRE ATT&CK techniques |
| **ATT&CK Matrix** | Interactive MITRE matrix visualization |
| **Scenarios** | Attack scenarios with Run capability |
| **Executions** | Execution history and results |
| **Settings** | Configuration options |

---

## Current Techniques (15 total)

### Discovery (9)
T1082, T1083, T1057, T1016, T1049, T1087, T1069, T1018, T1007

### Execution (3)
T1059.001, T1059.003, T1059.004

### Persistence (2)
T1053.005, T1547.001

### Defense Evasion (1)
T1070.004

All techniques support **Safe Mode** for production-safe testing.

---

## API Overview

### Authentication
| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/auth/login` | Login (username/password) |
| `POST /api/v1/auth/refresh` | Refresh access token |
| `GET /api/v1/auth/me` | Get current user info |

### Core API
| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/agents` | List connected agents |
| `GET /api/v1/techniques` | List MITRE techniques |
| `GET /api/v1/scenarios` | List attack scenarios |
| `POST /api/v1/executions` | Start an execution |
| `GET /api/v1/executions/:id/results` | Get execution results |
| `POST /api/v1/executions/:id/stop` | Stop running execution |

See the complete [API Reference](api/reference.md).

---

## WebSocket Events

Real-time updates via `wss://localhost:8443/ws/dashboard`:

- `execution_started` - Execution began
- `execution_completed` - Execution finished
- `execution_cancelled` - Execution stopped

---

## Security Score

Measures how well your defenses perform:

| Result | Points | Meaning |
|--------|--------|---------|
| Blocked | 100 | Attack prevented |
| Detected | 50 | Attack seen but not stopped |
| Success | 0 | Attack succeeded undetected |

**Formula:** `(blocked×100 + detected×50) / (total×100) × 100%`

---

## Links

- [GitHub Repository](https://github.com/AutoStrike-EIP/AutoStrike_POC)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [API Reference](api/reference.md)
- [Quick Start Guide](guide/quickstart.md)
