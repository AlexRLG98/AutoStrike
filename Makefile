# AutoStrike - Breach and Attack Simulation Platform
# Main Makefile

.PHONY: all build clean test dev docker help
.PHONY: server-build server-dev server-test
.PHONY: agent-build agent-test
.PHONY: dashboard-build dashboard-test
.PHONY: certs docker-build docker-up docker-down
.PHONY: run stop logs agent deps deps-install install setup

# Variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Colors
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

help: ## Show this help
	@echo "$(CYAN)AutoStrike$(RESET) - Breach and Attack Simulation Platform"
	@echo ""
	@echo "$(GREEN)First time? Run:$(RESET)"
	@echo "  $(CYAN)make setup$(RESET)  - Install all dependencies + certificates"
	@echo ""
	@echo "$(GREEN)Quick Start:$(RESET)"
	@echo "  $(CYAN)make run$(RESET)    - Build and start everything"
	@echo "  $(CYAN)make agent$(RESET)  - Connect an agent"
	@echo "  $(CYAN)make stop$(RESET)   - Stop all services"
	@echo ""
	@echo "$(GREEN)Available targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

all: build ## Build all components

# =============================================================================
# Quick Start - Single command to run everything
# =============================================================================

run: stop server-build-quick dashboard-build-quick ## Build and run the server (serves API + Dashboard)
	@echo ""
	@echo "$(GREEN)════════════════════════════════════════$(RESET)"
	@echo "$(GREEN)  Starting AutoStrike...$(RESET)"
	@echo "$(GREEN)════════════════════════════════════════$(RESET)"
	@echo ""
	@# Setup directories
	@mkdir -p server/data server/certs
	@# Generate certs if missing
	@if [ ! -f server/certs/server.crt ]; then \
		echo "$(YELLOW)Generating TLS certificates...$(RESET)"; \
		openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
			-keyout server/certs/server.key \
			-out server/certs/server.crt \
			-subj "/CN=localhost" 2>/dev/null; \
		cp server/certs/server.crt server/certs/ca.crt; \
	fi
	@# Start server (serves both API and Dashboard)
	@echo "$(YELLOW)Starting server...$(RESET)"
	@cd server && ./autostrike-server > /tmp/autostrike-server.log 2>&1 &
	@sleep 2
	@curl -s http://localhost:8443/health > /dev/null && echo "$(GREEN)✓ Server running$(RESET)" || echo "$(YELLOW)Server starting...$(RESET)"
	@echo ""
	@echo "$(GREEN)════════════════════════════════════════$(RESET)"
	@echo "$(GREEN)  AutoStrike is running!$(RESET)"
	@echo "$(GREEN)════════════════════════════════════════$(RESET)"
	@echo ""
	@echo "  $(CYAN)http://localhost:8443$(RESET)"
	@echo ""
	@echo "  Routes:"
	@echo "    /           Dashboard"
	@echo "    /api/v1/*   REST API"
	@echo "    /ws/*       WebSocket"
	@echo "    /health     Health check"
	@echo ""
	@echo "  Commands:"
	@echo "    $(CYAN)make agent$(RESET)  - Connect an agent"
	@echo "    $(CYAN)make stop$(RESET)   - Stop all services"
	@echo "    $(CYAN)make logs$(RESET)   - View logs"
	@echo ""

dev: run ## Alias for 'make run'

# =============================================================================
# Build
# =============================================================================

build: server-build agent-build dashboard-build ## Build all components
	@echo "$(GREEN)All components built successfully!$(RESET)"

server-build: ## Build the Go server
	@echo "$(YELLOW)Building server...$(RESET)"
	cd server && CGO_ENABLED=1 go build -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o autostrike-server ./cmd/autostrike

server-build-quick: ## Build server (quick, for run target)
	@cd server && go build -o autostrike-server ./cmd/autostrike 2>/dev/null || \
		(echo "$(YELLOW)Building server...$(RESET)" && go build -o autostrike-server ./cmd/autostrike)

agent-build: ## Build the Rust agent
	@echo "$(YELLOW)Building agent...$(RESET)"
	cd agent && PATH="$$HOME/.cargo/bin:$$HOME/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/bin:$$PATH" cargo build --release

agent-build-quick: ## Build agent (quick)
	@cd agent && PATH="$$HOME/.cargo/bin:$$HOME/.rustup/toolchains/stable-x86_64-unknown-linux-gnu/bin:$$PATH" cargo build --release 2>/dev/null

dashboard-build: ## Build the React dashboard
	@echo "$(YELLOW)Building dashboard...$(RESET)"
	cd dashboard && npm run build

dashboard-build-quick: ## Build dashboard if not exists
	@if [ ! -f dashboard/dist/index.html ]; then \
		echo "$(YELLOW)Building dashboard...$(RESET)"; \
		cd dashboard && npm install --silent && npm run build; \
	fi

# =============================================================================
# Testing
# =============================================================================

test: server-test agent-test dashboard-test ## Run all tests

server-test: ## Run server tests
	@echo "$(YELLOW)Testing server...$(RESET)"
	cd server && go test -v ./...

agent-test: ## Run agent tests
	@echo "$(YELLOW)Testing agent...$(RESET)"
	cd agent && PATH="$$HOME/.cargo/bin:$$PATH" cargo test

dashboard-test: ## Run dashboard tests
	@echo "$(YELLOW)Testing dashboard...$(RESET)"
	cd dashboard && npm run test -- --run

lint: ## Run linters
	@echo "$(YELLOW)Linting...$(RESET)"
	cd server && go vet ./...
	cd agent && PATH="$$HOME/.cargo/bin:$$PATH" cargo clippy
	cd dashboard && npm run lint

# =============================================================================
# Agent
# =============================================================================

agent: agent-build-quick ## Build and run the agent
	@echo "$(YELLOW)Starting agent...$(RESET)"
	@cd agent && ./target/release/autostrike-agent \
		--server http://localhost:8443 \
		--paw agent-$$(hostname)-$$$$ \
		--agent-secret agent-dev-secret

# =============================================================================
# Docker
# =============================================================================

docker-build: ## Build Docker images
	@echo "$(YELLOW)Building Docker images...$(RESET)"
	docker-compose build

docker-up: ## Start Docker containers
	@echo "$(YELLOW)Starting containers...$(RESET)"
	docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "$(YELLOW)Stopping containers...$(RESET)"
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

# =============================================================================
# Certificates
# =============================================================================

certs: ## Generate TLS certificates
	@echo "$(YELLOW)Generating certificates...$(RESET)"
	chmod +x scripts/generate-certs.sh
	./scripts/generate-certs.sh ./certs

# =============================================================================
# Utilities
# =============================================================================

deps: ## Check and install system dependencies (Go, Rust, Node)
	@echo "$(YELLOW)Checking system dependencies...$(RESET)"
	@echo ""
	@echo "$(CYAN)Build tools:$(RESET)"
	@which git > /dev/null 2>&1 && echo "$(GREEN)  ✓ git$(RESET)" || { echo "$(YELLOW)  ✗ git - sudo apt install git$(RESET)"; exit 1; }
	@which make > /dev/null 2>&1 && echo "$(GREEN)  ✓ make$(RESET)" || { echo "$(YELLOW)  ✗ make - sudo apt install make$(RESET)"; exit 1; }
	@which curl > /dev/null 2>&1 && echo "$(GREEN)  ✓ curl$(RESET)" || { echo "$(YELLOW)  ✗ curl - sudo apt install curl$(RESET)"; exit 1; }
	@which openssl > /dev/null 2>&1 && echo "$(GREEN)  ✓ openssl$(RESET)" || { echo "$(YELLOW)  ✗ openssl - sudo apt install openssl$(RESET)"; exit 1; }
	@(which gcc > /dev/null 2>&1 || which cc > /dev/null 2>&1) && echo "$(GREEN)  ✓ gcc$(RESET)" || { echo "$(YELLOW)  ✗ gcc - sudo apt install build-essential$(RESET)"; exit 1; }
	@which pkg-config > /dev/null 2>&1 && echo "$(GREEN)  ✓ pkg-config$(RESET)" || echo "$(YELLOW)  ⚠ pkg-config (optional) - sudo apt install pkg-config$(RESET)"
	@echo ""
	@echo "$(CYAN)Languages:$(RESET)"
	@# Check Go
	@which go > /dev/null 2>&1 && echo "$(GREEN)  ✓ Go $$(go version | cut -d' ' -f3) (need 1.21+)$(RESET)" || { \
		echo "$(YELLOW)  ✗ Go not found$(RESET)"; \
		echo "    Ubuntu/Debian: sudo snap install go --classic"; \
		echo "    macOS: brew install go"; \
		echo "    Other: https://go.dev/dl/"; \
		exit 1; \
	}
	@# Check Rust
	@(which cargo > /dev/null 2>&1 || test -f "$$HOME/.cargo/bin/cargo") && echo "$(GREEN)  ✓ Rust $$(rustc --version 2>/dev/null | cut -d' ' -f2 || $$HOME/.cargo/bin/rustc --version 2>/dev/null | cut -d' ' -f2) (need 1.75+)$(RESET)" || { \
		echo "$(YELLOW)  ✗ Rust not found$(RESET)"; \
		echo "    All platforms: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"; \
		echo "    Then run: source ~/.cargo/env"; \
		exit 1; \
	}
	@# Check Node
	@which node > /dev/null 2>&1 && echo "$(GREEN)  ✓ Node $$(node -v) (need v18+)$(RESET)" || { \
		echo "$(YELLOW)  ✗ Node.js not found$(RESET)"; \
		echo "    Ubuntu/Debian: curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && sudo apt install -y nodejs"; \
		echo "    macOS: brew install node"; \
		exit 1; \
	}
	@# Check npm
	@which npm > /dev/null 2>&1 && echo "$(GREEN)  ✓ npm $$(npm -v)$(RESET)" || { echo "$(YELLOW)  ✗ npm (comes with Node)$(RESET)"; exit 1; }
	@echo ""
	@echo "$(CYAN)Optional (for Docker/cross-compile):$(RESET)"
	@which docker > /dev/null 2>&1 && echo "$(GREEN)  ✓ docker $$(docker --version | cut -d' ' -f3 | tr -d ',')$(RESET)" || echo "$(YELLOW)  ⚠ docker (optional) - https://docs.docker.com/get-docker/$(RESET)"
	@which docker-compose > /dev/null 2>&1 && echo "$(GREEN)  ✓ docker-compose$(RESET)" || (which docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1 && echo "$(GREEN)  ✓ docker compose$(RESET)" || echo "$(YELLOW)  ⚠ docker-compose (optional)$(RESET)")
	@echo ""
	@echo "$(GREEN)All required dependencies OK!$(RESET)"

deps-install: ## Auto-install missing dependencies (requires sudo)
	@echo "$(YELLOW)Installing system dependencies...$(RESET)"
	@echo ""
	@# Detect package manager
	@if which apt-get > /dev/null 2>&1; then \
		echo "$(CYAN)Detected: Debian/Ubuntu$(RESET)"; \
		sudo apt-get update; \
		sudo apt-get install -y git make curl openssl build-essential pkg-config; \
		echo ""; \
		echo "$(YELLOW)Installing Go...$(RESET)"; \
		which go > /dev/null 2>&1 || sudo snap install go --classic; \
		echo ""; \
		echo "$(YELLOW)Installing Node.js...$(RESET)"; \
		which node > /dev/null 2>&1 || (curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - && sudo apt-get install -y nodejs); \
		echo ""; \
		echo "$(YELLOW)Installing Rust...$(RESET)"; \
		which cargo > /dev/null 2>&1 || (curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && echo "Run: source ~/.cargo/env"); \
	elif which brew > /dev/null 2>&1; then \
		echo "$(CYAN)Detected: macOS (Homebrew)$(RESET)"; \
		brew install git curl openssl pkg-config go node; \
		which cargo > /dev/null 2>&1 || (curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y); \
	elif which dnf > /dev/null 2>&1; then \
		echo "$(CYAN)Detected: Fedora/RHEL$(RESET)"; \
		sudo dnf install -y git make curl openssl gcc pkg-config golang nodejs; \
		which cargo > /dev/null 2>&1 || (curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y); \
	elif which pacman > /dev/null 2>&1; then \
		echo "$(CYAN)Detected: Arch Linux$(RESET)"; \
		sudo pacman -S --noconfirm git make curl openssl gcc pkgconf go nodejs npm rust; \
	else \
		echo "$(YELLOW)Unknown package manager. Please install manually:$(RESET)"; \
		echo "  - git, make, curl, openssl, gcc, pkg-config"; \
		echo "  - Go 1.21+, Node.js 18+, Rust 1.75+"; \
		exit 1; \
	fi
	@echo ""
	@echo "$(GREEN)Installation complete!$(RESET)"
	@echo "$(YELLOW)Note: Run 'source ~/.cargo/env' if Rust was just installed$(RESET)"

install: deps ## Install all project dependencies
	@echo "$(YELLOW)Installing project dependencies...$(RESET)"
	cd server && go mod download
	cd agent && PATH="$$HOME/.cargo/bin:$$PATH" cargo fetch
	cd dashboard && npm install
	@echo "$(GREEN)✓ All dependencies installed$(RESET)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(RESET)"
	rm -rf dist/
	rm -rf server/autostrike-server
	rm -rf agent/target/
	rm -rf dashboard/dist/
	rm -rf dashboard/node_modules/

setup: deps install certs ## Initial project setup (installs everything)
	@echo ""
	@echo "$(GREEN)════════════════════════════════════════$(RESET)"
	@echo "$(GREEN)  Setup complete!$(RESET)"
	@echo "$(GREEN)════════════════════════════════════════$(RESET)"
	@echo ""
	@echo "  Next steps:"
	@echo "    $(CYAN)make run$(RESET)    - Start the server"
	@echo "    $(CYAN)make agent$(RESET)  - Connect an agent"
	@echo ""
	@echo "  Dashboard: $(CYAN)https://localhost:8443$(RESET)"
	@echo ""

stop: ## Stop all running services
	@echo "$(YELLOW)Stopping services...$(RESET)"
	-@pkill -f "autostrike-server" 2>/dev/null
	-@pkill -f "autostrike-agent" 2>/dev/null
	@echo "$(GREEN)✓ Services stopped$(RESET)"

logs: ## Show server logs
	@echo "$(CYAN)=== Server Logs ===$(RESET)"
	@tail -30 /tmp/autostrike-server.log 2>/dev/null || echo "No server logs"

dist: ## Create distribution package
	mkdir -p dist
	$(MAKE) build
	cp server/autostrike-server dist/
	cp agent/target/release/autostrike-agent dist/
	cp -r dashboard/dist dist/dashboard
	tar -czvf autostrike-$(VERSION).tar.gz dist/
