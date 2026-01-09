# Claude AI Context - Musing CLI

This document provides comprehensive context for Claude AI assistants working on the Musing CLI project.

## Project Overview

**Musing CLI** is a project-agnostic command-line tool for managing multi-service development stacks with Docker, MongoDB, and microservices. It's built in Go using the Charm Bracelet ecosystem (Bubble Tea, Huh, Lip Gloss) to provide a modern, fast, and beautiful terminal user interface.

### Design Philosophy

This CLI is designed to be **completely agnostic** - it can manage any development stack with similar architecture patterns:
- **Frontend-agnostic**: Works with Angular, React, Vue, Svelte, or any framework
- **Backend-agnostic**: Supports Go, Node, Python, or any microservice architecture
- Docker Compose orchestration
- Port-based health checking (framework-independent)
- MongoDB database deployments (adaptable to other databases)
- SSH tunnels for remote database access
- Local development and production environments

The tool replaces slow, error-prone bash scripts with a fast, type-safe Go binary featuring a professional terminal UI.

### Current Use Case: Musing-TU

The CLI currently manages the **musing-tu portfolio ecosystem** (stevengregory.io):

**Stack Overview**:
- **Frontend**: Angular 21 application showcasing digital art/NFTs, blog posts, and dynamic content
- **Backend**: 12+ Go-based microservices (APIs for art, news, quotes, Bitcoin prices, etc.)
- **Database**: MongoDB for content storage and configuration
- **Infrastructure**: Docker Compose (local dev), DigitalOcean (production)
- **CI/CD**: GitHub Actions automated deployments

**Portfolio Features**:
- Digital art and NFT showcase (SuperRare, OpenSea collections)
- Blog/news content management
- Bitcoin price tracking
- Random quotes and facts
- Personal streak tracking (e.g., alcohol-free days)
- Social network links integration

This is just one example - the CLI's architecture supports any similar multi-service development environment.

### Key Facts
- **Language**: Go (speed, type safety, consistency with microservice ecosystems)
- **Primary Dependencies**: Charm Bracelet ecosystem (bubbletea, lipgloss, huh), urfave/cli
- **Binary Name**: `musing` (configurable for other projects)
- **Startup Time**: 1-3ms (50x faster than bash scripts)
- **Distribution**: Single binary with zero runtime dependencies
- **Portability**: Project-agnostic design adaptable to any stack

## Architecture

### Directory Structure
```
musing-cli/
├── cmd/
│   ├── musing/
│   │   └── main.go          # Entry point with arg reordering and ASCII banner
│   ├── dev.go               # Development stack management
│   ├── deploy.go            # MongoDB deployment with safety checks
│   └── monitor.go           # Live TUI monitoring dashboard
├── internal/                 # Internal packages
│   ├── config/
│   │   └── config.go        # Service configurations & port definitions
│   ├── docker/
│   │   └── docker.go        # Docker Desktop & Compose operations
│   ├── health/
│   │   └── health.go        # Port health checks with latency measurement
│   ├── mongo/
│   │   └── deploy.go        # MongoDB deployment operations
│   └── ui/
│       ├── confirm.go       # Huh confirmation prompts
│       ├── gum.go           # Legacy (unused)
│       └── spinner.go       # Spinner utilities
├── go.mod
├── go.sum
├── .gitignore
├── README.md
└── CLAUDE.md                # This file
```

### Key Patterns

#### 1. Argument Reordering (cmd/musing/main.go:52-80)
The `reorderArgs()` function allows flags to appear after positional arguments (like `docker`, `kubectl`, `git`):
```bash
musing deploy news --env prod  # Works
musing deploy --env prod news  # Also works
```

#### 2. Banner Display (cmd/musing/main.go:16-22, 30-33)
ASCII art banner is shown for all commands EXCEPT `monitor` (which has its own full-screen TUI).

#### 3. UI Styling
- **Color scheme**: Magenta/purple (`lipgloss.Color("99")`)
- **No emojis**: Professional, clean output
- **Consistent formatting**: Rounded borders, styled sections, color-coded status indicators

## Commands

### 1. `musing monitor`
**Location**: `cmd/monitor.go`

Live monitoring dashboard with real-time health checks (3-second refresh).

**Features**:
- Full-screen Bubble Tea TUI using alt screen buffer
- Real-time service health monitoring
- Organized sections: Docker → Database → API Services → Frontend → SSH Tunnel(s)
- Visual status indicators (● green = running, ● red = down)
- Keyboard controls: `q`, `Ctrl+C`, `Esc` to exit
- Auto-refresh every 3 seconds

**Implementation Notes**:
- Uses `tea.Program` with `tea.WithAltScreen()` for full-screen experience
- Custom `model` struct implements `tea.Model` interface
- Health checks run on ticker via `tea.Tick` messages
- No banner shown (has own full-screen interface)

### 2. `musing dev`
**Location**: `cmd/dev.go`

Manages the development stack (Docker Compose services).

**Flags**:
- `--rebuild`: Force rebuild Docker images
- `--logs`: Start and follow logs
- `--stop`: Stop all services

**Features**:
- Auto-detects and starts Docker Desktop if needed
- Validates required API repositories exist
- Progress indicators for long operations
- Health checks for MongoDB and Angular
- Displays service URLs in styled format
- Helpful next-step suggestions

### 3. `musing deploy`
**Location**: `cmd/deploy.go`

Deploys MongoDB data collections to development or production.

**Usage**:
```bash
musing deploy              # All collections to dev
musing deploy news         # Specific collection to dev
musing deploy --env prod   # All collections to prod (with confirmation)
musing deploy news -e prod # Specific collection to prod
```

**Features**:
- Auto-discovery: Scans data directory for .json files - no hardcoded collections
- Production safety: Interactive confirmation before prod deployments
- Tunnel verification: Checks SSH tunnel is open before prod deployment
- Smart SSH hints: Generates tunnel command from config (or uses placeholder)
- Flexible targets: Deploy all collections or specific ones
- Smart defaults: Deploys to dev by default
- Flexible syntax: Flags work before or after arguments

**Safety Checks**:
1. Confirms production deployment with user
2. Verifies SSH tunnel is open (port 27019) before proceeding
3. Generates helpful SSH tunnel command using production config if available
4. Clear messaging about data overwrite

**Configuration**:
Optional production settings in `.musing.yaml`:
```yaml
production:
  server: root@your-server.com
  remoteDBPort: 27017
```
If not configured, defaults to placeholder `<your-server>` in error messages.

## Internal Packages

### config/config.go
Centralized service configuration and port definitions loaded from `.musing.yaml`.

**Key structs**:
- `ProjectConfig`: Root configuration with services, database, and optional production settings
- `ServiceConfig`: Individual service definition (name, port, type)
- `DatabaseConfig`: Database configuration (type, name, ports, dataDir)
- `ProductionConfig`: Optional production deployment settings (server, remoteDBPort)

**Key functions**:
- `FindProjectRoot()`: Searches upward for .musing.yaml and loads config
- `GetConfig()`: Returns singleton config instance
- `GetAPIRepos()`: Dynamically discovers API repository paths from config

### docker/docker.go
Docker Desktop and Docker Compose operations.

**Key functions**:
- `IsDockerRunning()`: Checks if Docker Desktop is running
- `StartDocker()`: Attempts to start Docker Desktop
- Docker Compose operations (start, stop, rebuild)

### health/health.go
Port-based health checking with latency measurement.

**Key functions**:
- `CheckPort(port int)`: Returns true if service responds on port
- `CheckPortWithLatency(port int)`: Returns (running bool, latency time.Duration)
- Used by monitor command for real-time status

### mongo/deploy.go
MongoDB deployment operations with auto-discovery.

**Key functions**:
- `DiscoverCollections(dataDir string)`: Auto-discovers all .json files in data directory
- `isJSONArray(filePath string)`: Auto-detects if JSON file is array or object
- `DeployCollection(uri, db, key, dataDir)`: Deploys single collection to MongoDB
- `DeployAll(uri, db, dataDir)`: Deploys all discovered collections
- `getCollectionKeys(collections)`: Returns list of available collection keys for errors

**How it works**:
- Scans data directory for .json files at runtime
- Collection names derived from filenames (e.g., `news.json` → `news` collection)
- Hyphens converted to underscores (e.g., `social-networks.json` → `social_networks`)
- Automatically detects JSON array vs object format for mongoimport --jsonArray flag
- No hardcoded collection definitions - completely project-agnostic

### ui/
UI helper functions for styled output and user interaction.

**confirm.go**:
- `Confirm(prompt string, defaultValue bool)`: Simple yes/no prompts using Huh
- `ConfirmWithBubbles(opts ConfirmOptions)`: Advanced confirmation with custom styling

**spinner.go**:
- Spinner utilities for long-running operations

## Code Style Guidelines

### When Working on This Project

1. **No emojis in output**: Keep output professional and clean
2. **Consistent color scheme**: Use magenta/purple (`lipgloss.Color("99")`) for primary styling
3. **Type safety**: Leverage Go's type system, avoid `interface{}` where possible
4. **Error handling**: Always handle errors explicitly, provide helpful messages
5. **UI consistency**: Match existing Lip Gloss styling patterns
6. **Comments**: Clear comments for complex logic, especially in TUI code

### Common Patterns

**Styled Headers**:
```go
headerStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("99")).
    Bold(true).
    Border(lipgloss.RoundedBorder()).
    Padding(0, 1)
```

**Status Indicators**:
```go
if running {
    status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("●")
} else {
    status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("●")
}
```

**Confirmations**:
```go
if !ui.Confirm("Deploy to production?", false) {
    return nil
}
```

## Development Workflow

### Building
```bash
go build -o musing ./cmd/musing
```

### Running Without Building
```bash
go run ./cmd/musing monitor
go run ./cmd/musing dev --stop
```

### Installing Globally
```bash
sudo cp musing /usr/local/bin/
```

### Dependencies
```bash
go mod tidy  # Clean up dependencies
go get <package>  # Add new dependency
```

## Testing Strategy

### Manual Testing Checklist
- [ ] `musing monitor` - Check live dashboard updates
- [ ] `musing dev` - Verify Docker start/stop
- [ ] `musing dev --rebuild` - Check rebuild works
- [ ] `musing deploy` - Test dev deployment
- [ ] `musing deploy --env prod` - Test confirmation prompt
- [ ] Flag ordering: `musing deploy news --env prod` vs `musing deploy --env prod news`
- [ ] Keyboard controls in monitor: `q`, `Ctrl+C`, `Esc`

## Future Enhancements

Roadmap items documented in README.md:
- [ ] Tunnel Command - Auto-start/stop SSH tunnels
- [ ] Build Command - Angular build with size analysis
- [ ] Logs Command - Selective log streaming by service
- [ ] Restart Command - Restart individual services
- [ ] Config Command - Manage CLI configuration
- [ ] Split-pane TUI - Logs + status in one view

## Common Issues & Solutions

### Issue: Monitor not updating
**Solution**: Check health check timing in `cmd/monitor.go`, verify ticker is sending messages

### Issue: Docker commands fail
**Solution**: Verify Docker Desktop is running, check `docker/docker.go` implementation

### Issue: Flags not recognized
**Solution**: Verify `reorderArgs()` in `cmd/musing/main.go` is handling flag correctly

### Issue: Banner showing in monitor
**Solution**: Check `cmd/musing/main.go:31` condition to skip banner for monitor command

## Dependencies Reference

### Core
- `github.com/urfave/cli/v2` - CLI framework
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/charmbracelet/huh` - Form/prompt components

### Utilities
- `github.com/common-nighthawk/go-figure` - ASCII art banners

### Standard Library
Heavy use of: `os/exec`, `net`, `time`, `strings`, `fmt`

## File Modification Notes

- **cmd/musing/main.go**: Banner logic, arg reordering, command registration
- **cmd/*.go**: Command implementations - each is self-contained
- **internal/config/config.go**: Add new services/ports here
- **internal/health/health.go**: Modify health check logic here
- **.gitignore**: Excludes binaries, .DS_Store, IDE files, env files

## Context for Code Changes

When making changes:
1. **Maintain consistency**: Match existing patterns and styling
2. **Test thoroughly**: Go's compilation catches many bugs, but test runtime behavior
3. **Update docs**: Keep README.md in sync with code changes
4. **Consider UX**: This tool should feel professional and fast
5. **Binary size**: Avoid heavy dependencies that bloat the binary

## Related Projects

This CLI manages the **musing-tu** development stack, which includes:
- Multiple Go-based API services (12+)
- Angular frontend
- MongoDB database
- Docker Compose orchestration
- DigitalOcean production environment with SSH tunnels

The CLI is a development tool to make working with this complex stack easier and more enjoyable.
