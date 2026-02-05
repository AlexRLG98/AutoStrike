# Quick Start Guide

This guide will help you run your first attack simulation in 5 minutes.

---

## Prerequisites

| Component | Version | Purpose |
|-----------|---------|---------|
| Go | 1.21+ | Server compilation |
| Node.js | 18+ | Dashboard build |
| Rust | 1.75+ | Agent compilation |

---

## 1. Installation

```bash
# Clone the repository
git clone https://github.com/AutoStrike-EIP/AutoStrike_POC.git
cd AutoStrike_POC

# Install dependencies and build
make install
```

---

## 2. Start AutoStrike

```bash
make run
```

The server starts on **https://localhost:8443** and serves:

| Path | Description |
|------|-------------|
| `/` | Dashboard (React SPA) |
| `/api/v1/*` | REST API |
| `/ws/agent` | WebSocket for agents |
| `/ws/dashboard` | WebSocket for real-time updates |
| `/health` | Health check endpoint |

!!! note "Authentication"
    Authentication is **disabled** by default in development.
    To enable it, set `JWT_SECRET` in `server/.env`.

---

## 3. Connect an Agent

Open a new terminal and run:

```bash
make agent
```

This compiles and runs the Rust agent, which will:

1. Connect to `wss://localhost:8443/ws/agent`
2. Register with hostname, platform, and available executors
3. Start sending heartbeats every 30 seconds
4. Wait for tasks from the server

Verify the agent is connected in the **Agents** page (status should be "Online").

---

## 4. Run a Scenario

1. Navigate to **Scenarios** in the dashboard
2. Click **Run** on "Discovery - Basic" (or any available scenario)
3. In the modal:
   - Select target agents (checkboxes)
   - Toggle **Safe Mode** (recommended for first run)
4. Click **Run Execution**

The execution will start and you'll be redirected to the Executions page.

---

## 5. Monitor Execution

The **Executions** page shows real-time progress:

| Status | Description |
|--------|-------------|
| **Pending** | Execution created, waiting to start |
| **Running** | Techniques being executed on agents |
| **Completed** | All techniques finished |
| **Cancelled** | Execution stopped by user |

Click on an execution row to see detailed results.

---

## 6. Analyze Results

In **Execution Details**, each technique shows:

| Status | Meaning | Score Impact |
|--------|---------|--------------|
| **Blocked** | Attack was prevented by security controls | +100 points |
| **Detected** | Attack succeeded but was detected | +50 points |
| **Success** | Attack succeeded undetected | +0 points |
| **Failed** | Technique execution failed | Not counted |

**Security Score Formula:**
```
Score = (blocked * 100 + detected * 50) / (total * 100) * 100%
```

Higher scores indicate better security posture.

---

## 7. Explore the MITRE Matrix

The **ATT&CK Matrix** page provides a visual overview:

| Color | Meaning |
|-------|---------|
| Green cell | Techniques available and safe |
| Red dot | Technique is unsafe (use with caution) |
| Click | View technique details and detection info |

Filter by platform (Windows/Linux) using the dropdown.

---

## Useful Commands

| Command | Description |
|---------|-------------|
| `make run` | Start the server |
| `make agent` | Connect a local agent |
| `make stop` | Stop all services |
| `make logs` | View server logs |
| `make test` | Run all tests |
| `make build` | Build all components |

---

## Available Techniques

AutoStrike includes 15 MITRE ATT&CK techniques:

### Discovery (9 techniques)
- T1082 - System Information Discovery
- T1083 - File and Directory Discovery
- T1057 - Process Discovery
- T1016 - System Network Configuration Discovery
- T1049 - System Network Connections Discovery
- T1087 - Account Discovery
- T1069 - Permission Groups Discovery
- T1018 - Remote System Discovery
- T1007 - System Service Discovery

### Execution (3 techniques)
- T1059.001 - PowerShell
- T1059.003 - Windows Command Shell
- T1059.004 - Unix Shell

### Persistence (2 techniques)
- T1053.005 - Scheduled Task
- T1547.001 - Registry Run Keys

### Defense Evasion (1 technique)
- T1070.004 - File Deletion

---

## Troubleshooting

### Agent won't connect

1. Check server is running: `curl -k https://localhost:8443/health`
2. Verify WebSocket endpoint: `wss://localhost:8443/ws/agent`
3. Check agent logs for connection errors
4. Ensure no firewall blocking port 8443

### Execution stuck in "Running"

1. Check if agent is still online in Agents page
2. Agent may have disconnected - reconnect with `make agent`
3. Click **Stop** to cancel the execution

### Dashboard not loading

1. Verify server is running on port 8443
2. Clear browser cache
3. Check browser console for errors
4. Try incognito/private window

---

## Next Steps

- [Create a custom scenario](../mitre/techniques.md)
- [API Reference](../api/reference.md)
- [Architecture Overview](../architecture/index.md)
- [Deployment Guide](installation.md)
