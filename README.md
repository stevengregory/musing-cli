# Musing CLI

A fast command-line tool for managing multi-service development stacks with Docker, MongoDB, and microservices.

## What Does It Do?

This CLI provides professional tooling for complex development environments:

- **Live monitoring dashboard** - Real-time health checks for all services (3-second refresh)
- **Docker stack management** - Start/stop/rebuild services with auto-detection
- **Safe deployments** - MongoDB data sync with confirmations and tunnel verification
- **Beautiful TUI** - Professional terminal UI powered by Charm Bracelet

## My Workflow

This is my DevOps command center for [stevengregory.io](https://stevengregory.io). A tool managing my full-stack application: Angular frontend, Go service layer, and MongoDB database. Built for speedy local development and deployment of decoupled, multi-service stacks.

## Prerequisites

- Go 1.21 or higher
- Docker Desktop (for `dev` command)
- MongoDB (local or remote access)
- Docker Compose (for service orchestration)

## Installation

```bash
# Build from source
go build -o musing

# Install globally
sudo cp musing /usr/local/bin/
```

## Commands

### monitor

Live dashboard with real-time health checks.

```bash
musing monitor
```

**Features:**
- Real-time service health monitoring (3-second refresh)
- Color-coded status indicators for each service
- Organized sections: Docker → Database → API Services → Frontend → SSH Tunnels
- Keyboard controls: `q`, `Ctrl+C`, or `Esc` to exit

### dev

Manage the development stack.

```bash
musing dev              # Start all services
musing dev --rebuild    # Force rebuild images
musing dev --logs       # Start and follow logs
musing dev --stop       # Stop all services
```

**Features:**
- Auto-detects and starts Docker Desktop if needed
- Validates required repositories exist
- Health checks for MongoDB and frontend
- Progress indicators for long operations

### deploy

Deploy MongoDB collections to dev or production.

```bash
musing deploy              # All collections to dev
musing deploy news         # Specific collection to dev
musing deploy --env prod   # All to prod (with confirmation)
musing deploy news -e prod # Specific collection to prod
```

**How it works:**
- Auto-discovers all `.json` files in your data directory
- Collection names derived from filenames (e.g., `news.json` → `news` collection)
- Automatically detects JSON arrays vs. objects
- No manual configuration needed

**Production safety:**
- Interactive confirmation required
- Verifies SSH tunnel connectivity
- Clear warnings about data overwrite

## Configuration

Create a `.musing.yaml` file in your project root to define your stack:

```yaml
services:
  # Frontend
  - name: Angular
    port: 3000
    type: frontend

  # API Services
  - name: my-api
    port: 8080
    type: api

# Database configuration
database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data

# Optional: Production deployment settings
production:
  server: root@your-server.com      # SSH server for production access
  remoteDBPort: 27017                # Remote database port (typically 27017 for MongoDB)
```

## Why This Approach?

**Project-agnostic design** means you can adapt it for any stack:
- Works with any frontend framework (Angular, React, Vue, etc.)
- Backend-agnostic (Go, Node, Python microservices)
- Service configurations in `internal/config/config.go`
- Docker Compose integration
- Port-based health checking (framework-independent)
- MongoDB deployment patterns
- SSH tunnel support for remote databases

**Key benefits**:
- Fast startup (1-3ms)
- Type-safe Go prevents runtime errors
- Professional terminal UI with Bubble Tea
- Single binary with zero dependencies

## Development

```bash
# Run without installing
go run . monitor
go run . dev

# Manage dependencies
go mod tidy
```

## Architecture

```
musing-cli/
├── main.go              # Entry point
├── cmd/                 # Commands (dev, deploy, monitor)
├── internal/
│   ├── config/         # Service configs & ports
│   ├── docker/         # Docker operations
│   ├── health/         # Health checks
│   ├── mongo/          # MongoDB deployment
│   └── ui/             # Styled output & prompts
```

**Tech Stack**:
- Go (fast, type-safe, single binary)
- Bubble Tea (interactive TUI)
- Lip Gloss (terminal styling)
- Huh (confirmation prompts)

See [CLAUDE.md](CLAUDE.md) for detailed architecture and development guidelines.

## License

MIT
