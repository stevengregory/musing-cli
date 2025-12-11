# Musing CLI

Beautiful command-line tooling for the musing-tu development stack, built with Go + Charm (Gum & Bubbles).

## Features

- ✨ **Beautiful UX** - Styled output with Gum, animated spinners with Bubbles
- ⚡ **Blazing Fast** - Native Go binary with 1-3ms startup time
- 📊 **Live Monitoring** - Real-time service health dashboard
- 🐳 **Docker Integration** - Seamless Docker Compose management
- 🎯 **Type Safe** - Go's type system prevents runtime errors

## Installation

### Prerequisites

```bash
# Install Gum (required for beautiful styling)
brew install gum
```

### Build from Source

```bash
cd musing-cli
go build -o musing
```

### Optional: Install Globally

```bash
# Move binary to PATH
sudo cp musing /usr/local/bin/

# Now use from anywhere
musing status
```

## Commands

### `musing status`

Show current development stack status.

```bash
# One-time status check
musing status

# Live monitoring (updates every 2s)
musing status --watch
```

**Output:**
```
╔════════════════════════════════════════════════════════════╗
║                                                            ║
║                🚀 Development Stack Status                 ║
║                                                            ║
╚════════════════════════════════════════════════════════════╝

Core Services

✓ MongoDB              :27018 [5ms]
✓ Frontend             :3000  [124ms]

API Services

✓ networks-api         :8085  [45ms]
✓ random-facts-api     :8082  [38ms]
✗ alcohol-free-api     :8081  [timeout]
✓ random-quotes-api    :8083  [52ms]
...
```

### `musing dev`

Manage the development stack.

```bash
# Start all services
musing dev

# Force rebuild images
musing dev --rebuild

# Start and deploy MongoDB data
musing dev --data

# Start and follow logs
musing dev --logs

# Stop all services
musing dev --stop
```

**Features:**
- 🔍 Validates Docker daemon status
- 📂 Checks for required API repositories
- ✅ Interactive confirmation for missing repos
- ⏳ Animated spinners for long operations (Bubbles)
- 🏥 Health checks for MongoDB and Frontend
- 📦 Displays service URLs in styled box

## UX Highlights

### Gum Styling

- **Headers** - Double-bordered, centered, colored
- **Success/Error/Info** - Color-coded with icons (✓ ✗ ℹ ⚠)
- **Boxes** - Rounded borders for grouped information
- **Interactive Prompts** - Beautiful yes/no confirmations

### Bubbles Spinners

Animated spinners for long-running operations:
- Building Docker images
- Starting/stopping services
- Deploying data

**Graceful degradation**: Falls back to Gum spinners when no TTY available (CI/CD environments).

## Architecture

```
musing-cli/
├── main.go                          # Entry point
├── cmd/
│   ├── status.go                    # Status command with live watch mode
│   └── dev.go                       # Dev command with Docker management
├── internal/
│   ├── config/
│   │   └── config.go                # Service configurations & ports
│   ├── docker/
│   │   └── docker.go                # Docker Compose operations
│   ├── health/
│   │   └── health.go                # Port/HTTP health checks
│   └── ui/
│       ├── gum.go                   # Gum wrapper functions
│       └── spinner.go               # Bubbles spinner component
└── README.md
```

## Comparison with Bash Scripts

| Feature | Bash Scripts | Musing CLI |
|---------|-------------|-----------|
| Startup Time | ~20-50ms | 1-3ms |
| Type Safety | ❌ | ✅ |
| Error Handling | Basic | Comprehensive |
| UX | Basic colors | Gum + Bubbles |
| Testability | Difficult | Easy |
| Cross-platform | Unix only | Any OS |
| Live Monitoring | ❌ | ✅ |

## Development

### Run without Building

```bash
go run . status
go run . dev --stop
```

### Add New Command

1. Create `cmd/yourcommand.go`
2. Implement command using urfave/cli patterns
3. Add to `main.go` commands slice
4. Rebuild: `go build -o musing`

### Using UI Helpers

```go
import "github.com/stevengregory/musing-cli/internal/ui"

// Styled output
ui.Header("My Header")
ui.Success("Operation succeeded")
ui.Error("Operation failed")
ui.Info("Some information")
ui.Warning("Be careful!")

// Spinners
ui.SpinWithBubbles("Building...", "docker", "compose", "build")
ui.Spin("Quick task...", "echo", "hello")

// Confirmations
if ui.Confirm("Continue?", false) {
    // User said yes
}

// Boxes
ui.Box("Title", "Content goes here")
```

## Future Enhancements

- [ ] **Deploy Command** - MongoDB data deployment (currently uses `./scripts/deploy.sh`)
- [ ] **Tunnel Command** - SSH tunnel management for production MongoDB
- [ ] **Build Command** - Angular build with size analysis
- [ ] **Logs Command** - Selective log streaming by service
- [ ] **Restart Command** - Restart individual services
- [ ] **Config Command** - Manage CLI configuration (default ports, colors, etc.)
- [ ] **Full TUI Dashboard** - Split-pane view with logs + status (using Bubble Tea)

## Why Go + Gum + Bubbles?

1. **Go** - Your service layer is already in Go, so no new language
2. **Gum** - Instant beautiful styling without TUI complexity
3. **Bubbles** - Animated spinners for professional feel
4. **Speed** - 50x faster startup than Bun, 100x faster than Node
5. **Distribution** - Single binary, no runtime dependencies

## Contributing

This is a personal development tool, but contributions welcome! The codebase is intentionally simple and well-commented.

## License

MIT
