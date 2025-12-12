# Musing CLI

Professional command-line tooling for the musing-tu development stack, built with Go and the Charm Bracelet ecosystem (Bubble Tea, Huh, Lip Gloss).

## Features

- 🎨 **Professional TUI** - Native Bubble Tea interface with Lip Gloss styling
- ⚡ **Blazing Fast** - Native Go binary with 1-3ms startup time
- 📊 **Live Monitoring** - Real-time service health dashboard with auto-refresh
- 🐳 **Docker Integration** - Seamless Docker Desktop and Compose management
- 🔒 **Production Safety** - Interactive confirmation prompts for production deployments
- 🌐 **SSH Tunnel Monitoring** - Track DigitalOcean production tunnel status
- 🎯 **Type Safe** - Go's type system prevents runtime errors

## Installation

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
musing dev
```

## Commands

### `musing monitor`

Live monitoring dashboard with real-time health checks (updates every 3 seconds).

```bash
musing monitor
```

**Features:**
- 📊 Real-time service health monitoring
- 🎨 Organized sections: Docker → Database → API Services → Frontend → SSH Tunnel(s)
- 🔴🟢 Visual status indicators (● green = running, ● red = down)
- ⌨️ Keyboard controls: `q`, `Ctrl+C`, or `Esc` to exit
- 🔄 Auto-refresh every 3 seconds

**Output:**
```
╭──────────────────────────────────────────────╮
│  Development Stack - Live Monitor           │
╰──────────────────────────────────────────────╯

Updated: 14:32:15 PST

━━━ Docker ━━━
  ● Docker Desktop

━━━ Database ━━━
  ● MongoDB               :27018

━━━ API Services (12) ━━━
  ● networks-api          :8085
  ● random-facts-api      :8082
  ● alcohol-free-api      :8081
  ...

━━━ Frontend ━━━
  ● Angular               :3000

━━━ SSH Tunnel(s) ━━━
  ● DigitalOcean          :27019

Press q or Ctrl+C to exit • Updates every 3 seconds
```

### `musing dev`

Manage the development stack.

```bash
# Start all services
musing dev

# Force rebuild images
musing dev --rebuild

# Start and follow logs
musing dev --logs

# Stop all services
musing dev --stop
```

**Features:**
- 🔍 Auto-detects and starts Docker Desktop if needed
- 📂 Validates required API repositories exist
- ⏳ Progress indicators for long operations
- 🏥 Health checks for MongoDB and Angular
- 📦 Displays service URLs in styled format
- 💡 Helpful next-step suggestions

### `musing deploy`

Deploy MongoDB data collections to development or production.

```bash
# Deploy all collections to development (default)
musing deploy

# Deploy specific collection to development
musing deploy news

# Deploy to production (with confirmation prompt)
musing deploy --env prod
musing deploy -e prod

# Deploy specific collection to production
musing deploy news --env prod

# Flags can go before or after arguments
musing deploy --env prod news
musing deploy news --env prod
```

**Features:**
- 🔒 **Production Safety**: Interactive confirmation before prod deployments
- 🌐 **Tunnel Verification**: Checks SSH tunnel is open before prod deployment
- 📁 **Flexible Targets**: Deploy all collections or specific ones (news, projects, etc.)
- ⚡ **Smart Defaults**: Deploys to dev environment by default
- 🎯 **Flexible Syntax**: Flags work before or after arguments (like docker, kubectl, git)

## UX Highlights

### Bubble Tea TUI

- **Live Monitor**: Full-screen interactive dashboard with auto-refresh
- **Keyboard Controls**: Intuitive navigation (q/Ctrl+C/Esc to exit)
- **Alt Screen Buffer**: No terminal clutter, clean return to prompt

### Lip Gloss Styling

- **Headers**: Rounded borders with magenta/purple theme
- **Sections**: Organized with styled dividers (━━━)
- **Status Indicators**: Color-coded dots (green/red)
- **Timestamps**: Subtle gray italic formatting

### Huh Prompts

- **Native Confirmations**: Built-in Bubble Tea prompts (no external dependencies)
- **Production Safety**: Clear yes/no prompts for destructive operations
- **Keyboard Friendly**: Tab/Enter navigation

### Professional Polish

- No emoji characters (clean, professional output)
- Consistent color scheme (magenta/purple accent)
- Clear visual hierarchy
- Fast, responsive interface

## Architecture

```
musing-cli/
├── main.go                          # Entry point with arg reordering
├── cmd/
│   ├── dev.go                       # Dev command with Docker management
│   ├── deploy.go                    # MongoDB deployment with safety checks
│   └── monitor.go                   # Live TUI monitoring dashboard
├── internal/
│   ├── config/
│   │   └── config.go                # Service configurations & ports
│   ├── docker/
│   │   └── docker.go                # Docker Desktop & Compose operations
│   ├── health/
│   │   └── health.go                # Port health checks with latency
│   ├── mongo/
│   │   └── mongo.go                 # MongoDB deployment operations
│   └── ui/
│       ├── confirm.go               # Huh confirmation prompts
│       └── ui.go                    # Styled output helpers
└── README.md
```

## Comparison with Bash Scripts

| Feature | Bash Scripts | Musing CLI |
|---------|-------------|-----------|
| Startup Time | ~20-50ms | 1-3ms |
| Type Safety | ❌ | ✅ |
| Error Handling | Basic | Comprehensive |
| UX | Basic colors | Bubble Tea TUI |
| Testability | Difficult | Easy |
| Cross-platform | Unix only | Any OS |
| Live Monitoring | ❌ | ✅ (3s refresh) |
| Production Safety | ❌ | ✅ (confirmations) |
| SSH Tunnel Monitoring | ❌ | ✅ |

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
ui.Success("Operation succeeded")
ui.Error("Operation failed")
ui.Info("Some information")

// Confirmations (using Huh)
if ui.Confirm("Continue?", false) {
    // User said yes
}

// Advanced confirmations
confirmed := ui.ConfirmWithBubbles(ui.ConfirmOptions{
    Title:       "Deploy to production?",
    Description: "This will overwrite production data",
    Affirmative: "Yes, deploy",
    Negative:    "Cancel",
    Inline:      true,
})
```

## Future Enhancements

- [x] **Deploy Command** - MongoDB data deployment with prod safety ✅
- [x] **Live Monitoring** - Real-time TUI dashboard ✅
- [x] **SSH Tunnel Monitoring** - Track production tunnel status ✅
- [ ] **Tunnel Command** - Auto-start/stop SSH tunnels
- [ ] **Build Command** - Angular build with size analysis
- [ ] **Logs Command** - Selective log streaming by service
- [ ] **Restart Command** - Restart individual services
- [ ] **Config Command** - Manage CLI configuration (default ports, colors, etc.)
- [ ] **Split-pane TUI** - Logs + status in one view

## Why Go + Charm Bracelet?

1. **Go** - Service layer already in Go, consistent ecosystem
2. **Bubble Tea** - Professional TUI framework with full control
3. **Huh** - Native form/confirmation prompts
4. **Lip Gloss** - Beautiful styling without manual ANSI codes
5. **Speed** - 50x faster startup than Bun, 100x faster than Node
6. **Distribution** - Single binary, zero runtime dependencies
7. **Native Feel** - Professional CLI/TUI experience matching kubectl, gh, docker

## Contributing

This is a personal development tool, but contributions welcome! The codebase is intentionally simple and well-commented.

## License

MIT
