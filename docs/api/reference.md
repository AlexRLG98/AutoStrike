# API Reference

Base URL: `https://localhost:8443/api/v1`

---

## Authentication

### JWT Tokens (Optional)

Authentication is **disabled by default** in development mode. To enable it, set `JWT_SECRET` in your environment.

When enabled, all API requests require a JWT token in the Authorization header:

```http
Authorization: Bearer <token>
```

The JWT token is signed with the `JWT_SECRET` and contains:
- `sub`: User ID
- `role`: User role (admin, operator, viewer)
- `exp`: Token expiration date

### Auth Endpoints (Public)

#### Login
```http
POST /api/v1/auth/login
```

**Body:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

#### Refresh Token
```http
POST /api/v1/auth/refresh
```

**Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Logout
```http
POST /api/v1/auth/logout
```

#### Get Current User
```http
GET /api/v1/auth/me
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "user-uuid",
  "username": "admin",
  "email": "admin@autostrike.local",
  "role": "admin"
}
```

### Agent Authentication

Agents use a specific header:

```http
X-Agent-Key: <agent_secret>
```

The agent secret is defined in the `AGENT_SECRET` environment variable.

### Token Generation (Development)

For testing, generate a JWT token with:

```bash
SECRET="your-jwt-secret"
HEADER=$(echo -n '{"alg":"HS256","typ":"JWT"}' | base64 -w0 | tr '/+' '_-' | tr -d '=')
PAYLOAD=$(echo -n '{"sub":"admin","role":"admin","exp":'$(($(date +%s) + 86400))'}' | base64 -w0 | tr '/+' '_-' | tr -d '=')
SIGNATURE=$(echo -n "${HEADER}.${PAYLOAD}" | openssl dgst -sha256 -hmac "${SECRET}" -binary | base64 -w0 | tr '/+' '_-' | tr -d '=')
echo "${HEADER}.${PAYLOAD}.${SIGNATURE}"
```

---

## Health Check

### Server Health

```http
GET /health
```

**Response:**

```json
{
  "status": "ok",
  "auth_enabled": true
}
```

The `auth_enabled` field indicates whether JWT authentication is enabled on the server.

---

## Agents

### List Agents

```http
GET /api/v1/agents
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `all` | boolean | If `true`, returns all agents. Default: only online agents |

**Response:**

```json
[
  {
    "paw": "agent-001",
    "hostname": "WORKSTATION-01",
    "username": "admin",
    "platform": "windows",
    "executors": ["powershell", "cmd"],
    "status": "online",
    "last_seen": "2024-01-01T12:00:00Z",
    "created_at": "2024-01-01T10:00:00Z"
  }
]
```

### Get Agent

```http
GET /api/v1/agents/:paw
```

### Register Agent

```http
POST /api/v1/agents
```

**Body:**

```json
{
  "paw": "agent-001",
  "hostname": "WORKSTATION-01",
  "username": "admin",
  "platform": "windows",
  "executors": ["powershell", "cmd"]
}
```

### Delete Agent

```http
DELETE /api/v1/agents/:paw
```

### Heartbeat

```http
POST /api/v1/agents/:paw/heartbeat
```

Updates the agent's `last_seen` timestamp.

---

## Techniques

### List Techniques

```http
GET /api/v1/techniques
```

**Response:**

```json
[
  {
    "id": "T1082",
    "name": "System Information Discovery",
    "description": "Adversaries may attempt to get detailed information...",
    "tactic": "discovery",
    "platforms": ["windows", "linux"],
    "executors": [
      {
        "type": "cmd",
        "command": "systeminfo",
        "cleanup": "",
        "timeout": 60
      }
    ],
    "detection": [
      {
        "source": "Process Creation",
        "indicator": "systeminfo.exe execution"
      }
    ],
    "is_safe": true
  }
]
```

### Get Technique

```http
GET /api/v1/techniques/:id
```

### Techniques by Tactic

```http
GET /api/v1/techniques/tactic/:tactic
```

Available MITRE tactics:

| Tactic | Description |
|--------|-------------|
| `reconnaissance` | Gathering information |
| `resource-development` | Establishing resources |
| `initial-access` | Getting into the network |
| `execution` | Running malicious code |
| `persistence` | Maintaining presence |
| `privilege-escalation` | Gaining higher permissions |
| `defense-evasion` | Avoiding detection |
| `credential-access` | Stealing credentials |
| `discovery` | Exploring the environment |
| `lateral-movement` | Moving through the network |
| `collection` | Gathering target data |
| `command-and-control` | Communicating with compromised systems |
| `exfiltration` | Stealing data |
| `impact` | Manipulating or destroying systems |

### Techniques by Platform

```http
GET /api/v1/techniques/platform/:platform
```

Platforms: `windows`, `linux`, `darwin`

### MITRE Coverage

```http
GET /api/v1/techniques/coverage
```

**Response:**

```json
{
  "discovery": 9,
  "execution": 3,
  "persistence": 2,
  "defense-evasion": 1
}
```

### Import Techniques

```http
POST /api/v1/techniques/import
```

**Body:**

```json
{
  "path": "/path/to/techniques.yaml"
}
```

---

## Scenarios

### List Scenarios

```http
GET /api/v1/scenarios
```

**Response:**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "APT29 Discovery",
    "description": "APT29 discovery techniques simulation",
    "phases": [
      {
        "name": "Initial Discovery",
        "techniques": ["T1082", "T1083"],
        "order": 1
      }
    ],
    "tags": ["apt29", "discovery"],
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z"
  }
]
```

### Get Scenario

```http
GET /api/v1/scenarios/:id
```

### Scenarios by Tag

```http
GET /api/v1/scenarios/tag/:tag
```

### Create Scenario

```http
POST /api/v1/scenarios
```

**Body:**

```json
{
  "name": "APT29 Discovery",
  "description": "APT29 discovery techniques simulation",
  "phases": [
    {
      "name": "Initial Discovery",
      "techniques": ["T1082", "T1083"],
      "order": 1
    }
  ],
  "tags": ["apt29", "discovery"]
}
```

**Response (201):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "APT29 Discovery",
  "description": "APT29 discovery techniques simulation",
  "phases": [...],
  "tags": ["apt29", "discovery"],
  "created_at": "2024-01-01T10:00:00Z"
}
```

**Errors:**

| Code | Description |
|------|-------------|
| 400 | Missing required fields (name, phases) or invalid technique |
| 500 | Server error |

### Update Scenario

```http
PUT /api/v1/scenarios/:id
```

**Body:**

```json
{
  "name": "APT29 Discovery Updated",
  "description": "Updated description",
  "phases": [
    {
      "name": "Phase 1",
      "techniques": ["T1082"],
      "order": 1
    }
  ],
  "tags": ["apt29"]
}
```

**Errors:**

| Code | Description |
|------|-------------|
| 400 | Missing required fields or invalid technique |
| 404 | Scenario not found |
| 500 | Server error |

### Delete Scenario

```http
DELETE /api/v1/scenarios/:id
```

**Response:** 204 No Content

---

## Executions

### List Recent Executions

```http
GET /api/v1/executions
```

Returns the 50 most recent executions.

**Response:**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "scenario_id": "scenario-001",
    "status": "completed",
    "started_at": "2024-01-01T12:00:00Z",
    "completed_at": "2024-01-01T12:05:00Z",
    "safe_mode": true,
    "score": {
      "overall": 75.0,
      "blocked": 3,
      "detected": 2,
      "successful": 1,
      "total": 6
    }
  }
]
```

**Execution Status Values:**

| Status | Description |
|--------|-------------|
| `pending` | Execution created, not yet started |
| `running` | Execution in progress |
| `completed` | Execution finished successfully |
| `failed` | Execution encountered an error |
| `cancelled` | Execution was stopped by user |

### Get Execution

```http
GET /api/v1/executions/:id
```

### Execution Results

```http
GET /api/v1/executions/:id/results
```

**Response:**

```json
[
  {
    "id": "result-uuid",
    "execution_id": "550e8400-e29b-41d4-a716-446655440000",
    "technique_id": "T1082",
    "agent_paw": "agent-001",
    "status": "detected",
    "output": "Host Name: WORKSTATION-01...",
    "detected": true,
    "start_time": "2024-01-01T12:00:05Z",
    "end_time": "2024-01-01T12:00:10Z"
  }
]
```

**Result Status Values:**

| Status | Description |
|--------|-------------|
| `pending` | Task not yet executed |
| `running` | Task currently executing |
| `success` | Task executed, not detected (bad for defense) |
| `blocked` | Task blocked by security controls (good for defense) |
| `detected` | Task executed but detected (partial defense) |
| `failed` | Task execution failed |
| `skipped` | Task skipped (e.g., incompatible platform) |
| `timeout` | Task timed out |

### Start Execution

```http
POST /api/v1/executions
```

**Body:**

```json
{
  "scenario_id": "scenario-001",
  "agent_paws": ["agent-001", "agent-002"],
  "safe_mode": true
}
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scenario_id": "scenario-001",
  "status": "running",
  "started_at": "2024-01-01T12:00:00Z",
  "safe_mode": true
}
```

### Complete Execution

```http
POST /api/v1/executions/:id/complete
```

Manually marks an execution as completed and calculates the security score.

### Stop Execution

```http
POST /api/v1/executions/:id/stop
```

Stops a running or pending execution.

**Success Response (200):**

```json
{
  "status": "cancelled"
}
```

**Errors:**

| Code | Description |
|------|-------------|
| 404 | Execution not found |
| 409 | Execution already completed or cancelled |
| 500 | Server error |

---

## Security Score Calculation

The security score is calculated using the formula:

```
score = (blocked * 100 + detected * 50) / (total * 100) * 100
```

| Result | Points | Description |
|--------|--------|-------------|
| Blocked | 100 | Full protection - attack was prevented |
| Detected | 50 | Partial protection - attack was seen but not stopped |
| Success | 0 | No protection - attack succeeded undetected |

**Example:** 5 techniques tested, 2 blocked, 2 detected, 1 successful
```
score = (2*100 + 2*50) / (5*100) * 100 = 300/500 * 100 = 60%
```

---

## WebSocket Protocol

### Connection Endpoints

| Endpoint | Purpose |
|----------|---------|
| `wss://localhost:8443/ws/agent` | Agent connections |
| `wss://localhost:8443/ws/dashboard` | Dashboard real-time updates |

### Message Format

All WebSocket messages follow this JSON structure:

```json
{
  "type": "message_type",
  "payload": { ... }
}
```

### Connection Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| Ping interval | 54 seconds | Server sends ping frames |
| Pong timeout | 60 seconds | Connection closed if no pong |
| Max message size | 512 KB | Maximum frame size |
| Write timeout | 10 seconds | Maximum write duration |

---

## WebSocket (Dashboard)

### Connection

```
wss://localhost:8443/ws/dashboard
```

The dashboard connects to receive real-time notifications.

### Server -> Dashboard Messages

**Execution Started:**
```json
{
  "type": "execution_started",
  "payload": {
    "execution_id": "550e8400-e29b-41d4-a716-446655440000",
    "data": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "scenario_id": "scenario-001",
      "status": "running"
    }
  }
}
```

**Execution Completed:**
```json
{
  "type": "execution_completed",
  "payload": {
    "execution_id": "550e8400-e29b-41d4-a716-446655440000",
    "data": {
      "status": "completed"
    }
  }
}
```

**Execution Cancelled:**
```json
{
  "type": "execution_cancelled",
  "payload": {
    "execution_id": "550e8400-e29b-41d4-a716-446655440000",
    "data": {
      "status": "cancelled"
    }
  }
}
```

**Pong (response to ping):**
```json
{
  "type": "pong",
  "payload": {}
}
```

### Dashboard -> Server Messages

**Ping:**
```json
{
  "type": "ping",
  "payload": {}
}
```

---

## WebSocket (Agents)

### Connection

```
wss://localhost:8443/ws/agent
```

### Agent -> Server Messages

**Registration (sent immediately after connection):**
```json
{
  "type": "register",
  "payload": {
    "paw": "agent-001",
    "hostname": "WORKSTATION-01",
    "username": "admin",
    "platform": "windows",
    "executors": ["powershell", "cmd"]
  }
}
```

**Heartbeat (sent every 30 seconds by default):**
```json
{
  "type": "heartbeat",
  "payload": {
    "paw": "agent-001"
  }
}
```

**Task Result:**
```json
{
  "type": "task_result",
  "payload": {
    "task_id": "task-uuid",
    "technique_id": "T1082",
    "success": true,
    "output": "Host Name: WORKSTATION-01...",
    "exit_code": 0,
    "error": ""
  }
}
```

### Server -> Agent Messages

**Registration Acknowledgment:**
```json
{
  "type": "registered",
  "payload": {
    "status": "ok",
    "paw": "agent-001"
  }
}
```

**Task:**
```json
{
  "type": "task",
  "payload": {
    "id": "task-uuid",
    "technique_id": "T1082",
    "command": "systeminfo",
    "executor": "cmd",
    "timeout": 300,
    "cleanup": ""
  }
}
```

**Task Acknowledgment:**
```json
{
  "type": "task_ack",
  "payload": {
    "task_id": "task-uuid",
    "status": "received"
  }
}
```

**Ping:**
```json
{
  "type": "ping",
  "payload": {}
}
```

---

## Agent Connection Lifecycle

```
1. Agent connects to wss://server:8443/ws/agent
2. Agent sends "register" message with system info
3. Server responds with "registered" acknowledgment
4. Agent starts sending "heartbeat" every 30 seconds
5. Server sends "task" messages when execution starts
6. Agent executes command and sends "task_result"
7. Server sends "task_ack" acknowledgment
8. On disconnect, server marks agent as "offline"
```

### Reconnection Strategy

The agent uses exponential backoff for reconnection:

- Initial delay: 1 second
- Multiplier: 2x after each failure
- Maximum delay: 60 seconds
- Reset to 1 second after successful connection

---

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Invalid request |
| 401 | Not authenticated |
| 403 | Access denied |
| 404 | Resource not found |
| 409 | Conflict (e.g., execution already completed) |
| 500 | Server error |

**Error Response Format:**

```json
{
  "error": "error description"
}
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SECRET` | JWT signing secret (enables auth when set) | - |
| `AGENT_SECRET` | Agent authentication secret | - |
| `DATABASE_PATH` | SQLite database path | `./data/autostrike.db` |
| `ALLOWED_ORIGINS` | CORS allowed origins | `localhost:3000,localhost:8443` |
| `DASHBOARD_PATH` | Path to dashboard dist folder | - |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
