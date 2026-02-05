# Installation

## Prerequisites

### Server

| Component | Minimum Version | Recommended |
|-----------|-----------------|-------------|
| Go | 1.21+ | 1.22+ |
| Node.js | 18+ | 20 LTS |
| SQLite | 3.35+ | 3.40+ |
| Docker | 24+ (optional) | 25+ |
| Docker Compose | 2.20+ (optional) | 2.24+ |

### Agent

| OS | Minimum Version | Architecture |
|----|-----------------|--------------|
| Windows | 10 (1809+) | x64, ARM64 |
| Linux | Ubuntu 20.04+ / Debian 11+ | x64, ARM64 |
| macOS | 12 (Monterey)+ | x64, ARM64 |

**Required privileges:**
- Windows: Administrator rights (for certain techniques)
- Linux: Root or sudo access (for certain techniques)
- macOS: Administrator rights (for certain techniques)

---

## Server Installation

### Option 1: Quick Start with Makefile

```bash
# Clone the repository
git clone https://github.com/AutoStrike-EIP/AutoStrike_POC.git
cd AutoStrike_POC

# Install dependencies and build everything
make install

# Start the server (serves dashboard + API on port 8443)
make run
```

The server starts on **https://localhost:8443** and serves:
- Dashboard (React SPA)
- REST API (`/api/v1/*`)
- WebSocket (`/ws/*`)
- Health check (`/health`)

### Option 2: Manual Installation

#### 1. Clone the repository

```bash
git clone https://github.com/AutoStrike-EIP/AutoStrike_POC.git
cd AutoStrike_POC
```

#### 2. Environment Configuration

Create a `.env` file in the server directory:

```bash
cd server
cp .env.example .env
nano .env
```

Variables:

```env
# Database Configuration
DATABASE_PATH=./data/autostrike.db

# Dashboard Configuration (path to built dashboard)
DASHBOARD_PATH=../dashboard/dist

# JWT Configuration (optional - auth disabled if not set)
JWT_SECRET=your-secure-jwt-secret-32-characters

# Agent Authentication (optional)
AGENT_SECRET=your-secure-agent-secret

# CORS Origins
ALLOWED_ORIGINS=localhost:3000,localhost:8443

# Log Level (debug, info, warn, error)
LOG_LEVEL=info
```

!!! note "Authentication"
    Authentication is **disabled** by default when `JWT_SECRET` is not set.
    This is suitable for development. Set `JWT_SECRET` in production.

#### 3. Build Backend (Go)

```bash
cd server
go mod download
go build -o autostrike ./cmd/autostrike
./autostrike
```

#### 4. Build Frontend (React)

```bash
cd dashboard
npm install
npm run build
```

The built dashboard is served by the Go server on port 8443.

### Option 3: Docker Installation

```bash
# Build and start containers
docker compose up -d

# View logs
docker compose logs -f

# Stop containers
docker compose down
```

### Verify Installation

Test that the server responds:

```bash
# Health check
curl -k https://localhost:8443/health

# Expected response
{"status": "ok", "auth_enabled": false}
```

---

## Agent Installation

### Compile from Source

#### Prerequisites

- Rust 1.75+ with Cargo
- OpenSSL development libraries (Linux)

```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Build the agent
cd agent
cargo build --release

# Executable is in target/release/autostrike-agent
```

### Quick Start (Makefile)

```bash
# From project root
make agent
```

This compiles and runs the agent, connecting to `https://localhost:8443`.

### Windows (PowerShell Admin)

```powershell
# Navigate to agent directory
cd agent

# Build
cargo build --release

# Run the agent
.\target\release\autostrike-agent.exe --server https://server:8443 --paw "agent-win-001"
```

#### Install as Windows Service

```powershell
# Create the service
New-Service -Name "AutoStrike-Agent" `
  -BinaryPathName "C:\Path\To\autostrike-agent.exe --server https://server:8443" `
  -DisplayName "AutoStrike Agent" `
  -StartupType Automatic

# Start the service
Start-Service -Name "AutoStrike-Agent"

# Check status
Get-Service -Name "AutoStrike-Agent"
```

### Linux

```bash
# Build
cd agent
cargo build --release

# Run the agent
sudo ./target/release/autostrike-agent --server https://server:8443 --paw "agent-lin-001"
```

#### Install as systemd Service

Create `/etc/systemd/system/autostrike-agent.service`:

```ini
[Unit]
Description=AutoStrike Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/autostrike
ExecStart=/opt/autostrike/autostrike-agent --server https://server:8443
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable autostrike-agent
sudo systemctl start autostrike-agent
sudo systemctl status autostrike-agent
```

### macOS

```bash
# Build
cd agent
cargo build --release

# Run the agent
sudo ./target/release/autostrike-agent --server https://server:8443 --paw "agent-mac-001"
```

### Agent CLI Options

| Option | Description | Default |
|--------|-------------|---------|
| `-s, --server` | AutoStrike server URL | `https://localhost:8443` |
| `-p, --paw` | Unique agent identifier | Auto-generated UUID |
| `-c, --config` | Configuration file path | `agent.yaml` |
| `-d, --debug` | Enable debug logging | `false` |

### Agent Configuration File

Create `agent.yaml`:

```yaml
server_url: "https://server:8443"
paw: "agent-001"
heartbeat_interval: 30  # seconds

tls:
  cert_file: "./certs/agent.crt"  # optional
  key_file: "./certs/agent.key"   # optional
  ca_file: "./certs/ca.crt"       # optional
  verify: true
```

---

## Update

### Server

```bash
# Stop the server
make stop

# Pull updates
git pull origin main

# Rebuild and restart
make install
make run
```

### Agent

```bash
# Pull updates
git pull origin main

# Rebuild
cd agent
cargo build --release

# Restart service
sudo systemctl restart autostrike-agent
```

---

## Troubleshooting

### Server won't start

1. Check environment variables:
   ```bash
   cat server/.env
   ```

2. Check database file permissions:
   ```bash
   ls -la server/data/
   ```

3. Check logs:
   ```bash
   make logs
   ```

### Agent won't connect

1. Verify network connectivity:
   ```bash
   curl -k https://server:8443/health
   ```

2. Check TLS certificate:
   ```bash
   openssl s_client -connect server:8443
   ```

3. Check agent logs:
   ```bash
   journalctl -u autostrike-agent -f
   ```

### "Connection refused" error

- Ensure the server is listening on the correct interface
- Check firewall rules
- Confirm port 8443 is open:
  ```bash
  sudo netstat -tlnp | grep 8443
  ```

---

## Advanced Configuration

See the following sections for more details:

- [Backend Architecture](../architecture/backend.md)
- [Deployment Guide](./deployment.md)
- [API Reference](../api/reference.md)
