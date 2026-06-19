# Claude AI Context - Musing CLI

This document provides comprehensive context for Claude AI assistants working on the Musing CLI project.

> Keep this file in sync with the code. When commands, flags, config fields, function
> signatures, or dependencies change, update the relevant section here. See also
> [AGENTS.md](AGENTS.md) (Codex guidance) and [README.md](README.md) (user-facing docs).

## Project Overview

**Musing CLI** is a project-agnostic command-line tool for managing multi-service development stacks with Docker, MongoDB, and microservices. It's built in Go using [Cobra](https://github.com/spf13/cobra) for command parsing and the Charm Bracelet ecosystem (Bubble Tea, Bubbles, bubble-table, Lip Gloss, Huh) for a modern, fast, and beautiful terminal user interface.

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
- **Frontend**: Angular application showcasing digital art/NFTs, blog posts, and dynamic content
- **Backend**: Multiple Go-based microservices (APIs for art, news, quotes, Bitcoin prices, etc.)
- **Database**: MongoDB for content storage and configuration
- **Infrastructure**: Docker Compose (local dev), DigitalOcean (production)
- **CI/CD**: GitHub Actions automated tag-and-release

This is just one example - the CLI's architecture supports any similar multi-service development environment. Nothing about the current stack is hard-coded; everything is driven by a per-project `.musing.yaml`.

### Key Facts
- **Language**: Go (see `go.mod` for the exact toolchain version)
- **CLI framework**: `github.com/spf13/cobra`
- **TUI/styling**: Charm Bracelet — `bubbletea`, `bubbles`, `evertras/bubble-table`, `lipgloss` (+ `charm.land/lipgloss/v2`), `huh`
- **Config format**: `.musing.yaml` parsed with `gopkg.in/yaml.v3`
- **Binary Name**: `musing`
- **Distribution**: Single binary; Homebrew tap + GitHub releases via GoReleaser
- **Runtime tool dependencies** (invoked via `os/exec`, not Go deps): `docker`, `mongoimport`, `ssh`, `lsof`, `kill`, and `gum` (for styled non-TUI output)
- **Portability**: Project-agnostic design adaptable to any stack

## Architecture

### Directory Structure
```
musing-cli/
├── cmd/
│   ├── musing/
│   │   └── main.go          # Entry point — only calls cmd.Execute()
│   ├── root.go              # Cobra root command: banner, version vars, command groups
│   ├── project.go           # Shared helpers: loadProjectConfig, changeToProjectRoot
│   ├── dev.go               # `dev` + start/stop/rebuild/logs subcommands
│   ├── deploy.go            # `deploy` with positional env arg + alias resolution
│   ├── monitor.go           # Live TUI monitoring dashboard
│   ├── ssh.go               # `ssh` interactive session + buildSSHArgs/expandHomeDir helpers
│   ├── tunnel.go            # `tunnel` + start/stop/status subcommands
│   ├── monitor_test.go
│   ├── ssh_test.go
│   └── tunnel_test.go
├── internal/
│   ├── config/
│   │   ├── config.go        # .musing.yaml discovery & typed project config
│   │   └── config_test.go
│   ├── docker/
│   │   └── docker.go        # Docker Desktop & Docker Compose operations
│   ├── health/
│   │   └── health.go        # Port + HTTP health checks with latency measurement
│   ├── mongo/
│   │   ├── deploy.go        # Collection auto-discovery + mongoimport deployment
│   │   └── deploy_test.go
│   └── ui/
│       ├── confirm.go       # Huh confirmation prompts
│       ├── gum.go           # Styled output helpers (Info/Success/Error/Warning, Spin) via `gum`
│       └── spinner.go       # Bubble Tea spinner (SpinWithBubbles), falls back to gum Spin
├── .github/
│   └── workflows/
│       └── release.yml      # Auto-tag + GoReleaser on push to main
├── dist/                    # GoReleaser build output (gitignored)
├── images/                  # README screenshots
├── .goreleaser.yml          # Build/archive/Homebrew/release config
├── Makefile                 # build / install / clean targets
├── AGENTS.md                # Codex agent guidance
├── CLAUDE.md                # This file
├── README.md
├── go.mod
└── go.sum
```

### Key Patterns

#### 1. Cobra command tree (cmd/root.go)
The root command and all subcommands are wired with Cobra. `cmd/musing/main.go` is a trivial entry point that calls `cmd.Execute()`. There is **no manual argument reordering** — Cobra (via pflag) natively accepts flags before or after positional arguments. Commands are organized into two help groups:
- **Core Commands**: `dev`, `deploy`, `monitor`, `ssh`, `tunnel`
- **Additional Commands**: `version`, `help`, `completion`

The root command sets `SilenceUsage` and `SilenceErrors`; commands return errors and `Execute()` exits non-zero. Shell completion is enabled (`CompletionOptions.DisableDefaultCmd = false`), and `deploy` provides dynamic completion for collection/alias names and environments.

#### 2. Banner display (cmd/root.go:29-47, 118-149)
The ASCII art banner (`go-figure`, color `99`) is printed in the root command's `PersistentPreRunE` for every command **except** those filtered by `shouldShowBanner()` — `monitor`, `version`, `completion` (and its shell subcommands), and `help`. The check walks up the command's parents and also suppresses the banner during shell-completion requests. The `monitor` command draws its own banner inside its full-screen TUI `View()`.

#### 3. Project discovery (cmd/project.go, internal/config)
Cwd-sensitive commands use two shared helpers:
- `loadProjectConfig()` → finds the project root and returns `(root, *ProjectConfig, error)`.
- `changeToProjectRoot()` → does the above and `os.Chdir`es into the root (needed so `docker compose` runs against the project's `compose.yaml`).

`config.FindProjectRoot()` searches **upward** from the cwd for a directory containing `.musing.yaml`, and additionally requires a sibling `compose.yaml` to confirm the root. `config.MustFindProjectRoot()` is a variant that prints a friendly error and `os.Exit(1)`s — used by commands that only need config loaded (e.g. `ssh`, `tunnel`, deploy completion).

#### 4. UI styling
- **Primary accent**: magenta/purple. In the `cmd/` TUI styles this is the hex `lipgloss.Color("#FF00FF")`; the banner uses `lipgloss.Color("99")`. The `internal/ui/gum.go` helpers use ANSI-256 codes (212 magenta, 196 red, 214 amber, 205/246 for spinners).
- **Status colors**: green `#00FF00` (running), red `#FF0000` (down).
- **No emoji**: output uses Unicode status glyphs only (`✓ ✗ ● ━ • ℹ ⚠`), never emoji.
- **Consistent formatting**: rounded borders, styled section headers, color-coded status indicators.

## Commands

### `musing dev`
**Location**: `cmd/dev.go`

Manages the Docker Compose development stack. The bare `dev` command starts the stack; subcommands provide explicit verbs.

**Subcommands**:
- `dev` / `dev start` — start all services
- `dev stop` — `docker compose down`
- `dev rebuild` — `docker compose down`, then `build --no-cache`, then start
- `dev logs` — follow `docker compose logs -f`

**Behavior**:
- `changeToProjectRoot()` first so Compose runs in the right directory.
- `docker.EnsureRunning(false)` auto-starts Docker Desktop if needed.
- `checkAPIRepos()` warns about missing API repositories (discovered via `config.GetAPIRepos()`) and prompts to continue.
- After `up -d`, waits ~5s, then prints a sectioned health summary (Docker → Database → API Services → Frontend) via `printServiceStatus()`.

### `musing deploy`
**Location**: `cmd/deploy.go`

Deploys MongoDB JSON data collections to development or production.

**Usage** (positional `env`, defaults `collection=all`, `env=dev`):
```bash
musing deploy              # All collections to dev
musing deploy news         # Collection or alias to dev
musing deploy news prod    # Collection or alias to prod
```

**Aliases**: `serviceAliases()` builds a short-name → collection-key map from API services in `.musing.yaml` that declare an `alias`. The target is the service's `collection` field, or the alias itself when `collection` is omitted. Aliases also feed shell completion.

**Safety / flow**:
1. `prod` requires interactive confirmation (`ui.Confirm`, default No).
2. For `prod`, if the prod DB port is closed, the SSH tunnel is **auto-opened** (`tunnelStart()`) before deploy and **auto-closed** afterward (only if this command opened it).
3. For `dev`, the dev DB port must already be open (tells the user to run `musing dev` otherwise).
4. Invalid env values (anything other than `dev`/`prod`) are rejected.

### `musing monitor`
**Location**: `cmd/monitor.go`

Full-screen live monitoring dashboard (Bubble Tea, alt screen, 3-second refresh).

**Features**:
- Sections: Docker → Database → API Services → Frontend → SSH Tunnel(s).
- Visual status (`●` green running / red down); built on `bubble-table` plus a custom section renderer.
- Keyboard controls: `q`, `Ctrl+C`, `Esc` to exit; mouse cell motion enabled.
- Draws its own banner; refuses to start (with a hint) if Docker is not running.

**Implementation notes**:
- `monitorModel` implements `tea.Model`; health checks run off `tickMsg` via `tea.Tick`.
- Imports both `github.com/charmbracelet/lipgloss` and `charm.land/lipgloss/v2` (the latter is required by `bubble-table`).

### `musing ssh`
**Location**: `cmd/ssh.go`

Opens an **interactive** SSH session to the configured production server. Requires a `production` block in `.musing.yaml`. Builds args with `buildSSHArgs(cfg, false)` (no tunnel) and runs `ssh` with inherited stdio. `expandHomeDir()` expands a leading `~/` in `sshKeyPath`.

### `musing tunnel`
**Location**: `cmd/tunnel.go`

Manages the SSH tunnel to the production database. Bare `tunnel` starts it (or reports status if already up).

**Subcommands**: `start`, `stop`, `status`.
- `tunnelStart()` — no-op (shows status) if the prod port is already open; otherwise runs `ssh -f -N -L <prodPort>:localhost:<remotePort> <server>` via `buildSSHArgs(cfg, true)`. Detects auth failures and suggests `ssh-copy-id`.
- `tunnelStop()` — finds the PID on the prod port with `lsof -ti` and `kill`s it.
- `tunnelStatus()` — reports whether the prod port is open.
- Port defaults when unset in config: local/prod port `27019`, remote port `27017`.
- For testability, `cmd/tunnel.go` indirects `exec.Command` and `health.CheckPort` through package vars (`execCommand`, `checkPort`).

### `musing version`
**Location**: `cmd/root.go` (subcommand) — prints the version string. `musing --version` also works (Cobra's built-in version flag, with a custom template that prints just the number).

### `musing completion`
Cobra's built-in shell completion generator (`bash`, `zsh`, `fish`, `powershell`).

## Internal Packages

### config/config.go
`.musing.yaml` discovery and typed configuration.

**Types**:
- `ProjectConfig` — `Services []ServiceConfig`, `Database DatabaseConfig`, `Production *ProductionConfig` (optional), `APIReposDir string` (optional).
- `ServiceConfig` — `Name`, `Port`, `Type` (`frontend`/`api`/`database`), `Alias` (optional short deploy key), `Collection` (optional data-file key; defaults to `Alias`).
- `DatabaseConfig` — `Type`, `Name`, `DevPort`, `ProdPort`, `DataDir`.
- `ProductionConfig` — `Server`, `RemoteDBPort`, `SSHKeyPath`.

**Functions**:
- `FindProjectRoot()` — searches upward for `.musing.yaml`, loads it, and requires a sibling `compose.yaml`.
- `MustFindProjectRoot()` — same, but prints a friendly error and exits on failure.
- `GetConfig()` — returns the loaded singleton (`*ProjectConfig`, or `nil` if not loaded).
- `GetAPIRepos()` — returns filesystem paths for every `type: api` service, resolved under `resolveAPIReposParent()`.
- `resolveAPIReposParent(root, apiReposDir)` — empty → project root's parent (sibling layout); `~/` → home-expanded; absolute → as-is; otherwise relative to the project root's parent.

### docker/docker.go
Docker Desktop and Docker Compose operations.

- `CheckRunning()` — `docker info`; returns error if the daemon is down.
- `IsDockerDesktopInstalled()` — checks for `/Applications/Docker.app` (macOS).
- `EnsureRunning(promptUser bool)` — starts Docker Desktop if needed (tries `docker desktop start`, falls back to `open -a Docker -g`) and polls up to 90s for readiness.
- `ComposeUp()`, `ComposeDown()`, `ComposeBuild(noCache bool)`, `ComposeLogs(follow bool)` — thin wrappers over `docker compose …`.

### health/health.go
Port and HTTP health checking with latency measurement.

- `PortStatus{ Port int; Open bool; Latency time.Duration }`.
- `CheckPort(port int) PortStatus` — 2s TCP dial to `localhost:port`.
- `CheckHTTP(url string) HTTPStatus` — 3s HTTP GET; `Available` true for 2xx.
- `FormatLatency(d time.Duration) string` — millisecond formatting (`"timeout"` for zero).

### mongo/deploy.go
MongoDB deployment with auto-discovery.

- `Collection{ Name, File string; IsArray bool }`.
- `DiscoverCollections(dataDir) (map[string]Collection, error)` — scans `dataDir` for `.json` files. Map key = filename without `.json`; collection `Name` = key with `-` → `_`. Auto-detects array vs object by inspecting the first non-whitespace byte.
- `DeployCollection(uri, db, collectionKey, dataDir)` — runs `mongoimport --uri … --db … --collection … --file … --drop` (adds `--jsonArray` for arrays).
- `DeployAll(uri, db, dataDir)` — deploys every discovered collection.
- `getCollectionKeys(...)` — keys list for error messages.

### ui/
Terminal output, prompts, and spinners.

- **confirm.go** — `Confirm(prompt, defaultYes)` and `ConfirmWithBubbles(opts)` using Huh. On error/cancel, `Confirm` returns the default.
- **spinner.go** — `SpinWithBubbles(message, command, args...)` runs a command behind a Bubble Tea spinner; when there is no TTY it falls back to `Spin` (gum).
- **gum.go** — styled output via the external `gum` binary. `Info`, `Success`, `Warning`, `Error` (used throughout `cmd/`) route through `gum style`; `Spin` is the non-TTY spinner fallback. This file is **actively used**, not legacy.

## Configuration

Place `.musing.yaml` next to `compose.yaml` in the project root. Commands search upward from the cwd until they find that pair.

```yaml
# Optional: directory containing API repos.
# Empty → the project root's parent (sibling layout).
# Relative → resolved from the project root's parent; absolute and ~/ paths honored as-is.
apiReposDir: services

services:
  - name: Angular            # frontend
    port: 3000
    type: frontend

  - name: my-news-api        # API service
    port: 8080
    type: api
    alias: news              # optional: short key → `musing deploy news`
    collection: news-items   # optional: data-file key; defaults to alias when omitted

database:
  type: MongoDB
  name: mydb
  devPort: 27018
  prodPort: 27019
  dataDir: data

# Optional: production deployment / tunnel / ssh settings
production:
  server: root@your-server.com   # SSH target
  remoteDBPort: 27017            # remote DB port (typically 27017 for MongoDB)
  sshKeyPath: ~/.ssh/your-key    # optional; supports ~ expansion
```

## Code Style Guidelines

1. **No emoji** in output — use the existing Unicode status glyphs.
2. **Consistent accent color**: magenta/purple (`#FF00FF` in `cmd/` TUI styles; `99` for the banner; matching ANSI codes in `ui/gum.go`).
3. **Type safety**: lean on Go's type system; avoid `interface{}` where avoidable.
4. **Explicit error handling**: return useful, wrapped errors; let `rootCmd` own process exit. Don't `os.Exit` from deep in command logic (except `MustFindProjectRoot`).
5. **UI consistency**: match existing Lip Gloss / Huh patterns.
6. **Keep it fast & dependency-light**: don't add production dependencies without a clear reason.
7. **Reuse the helpers**: use `loadProjectConfig` / `changeToProjectRoot` for project discovery; use `serviceAliases` for deploy key mapping.
8. **Lip Gloss imports**: most code uses `github.com/charmbracelet/lipgloss`; `bubble-table` styling in `monitor.go` requires `charm.land/lipgloss/v2`. Keep both straight.
9. **Comments**: focus on non-obvious behavior (TUI update loops, shell-command seams, production safety).

## Build, Version & Release

### Building
```bash
make build                 # builds ./musing (gitignored)
go build -o musing ./cmd/musing
```

### Running without building
```bash
go run ./cmd/musing monitor
go run ./cmd/musing dev stop
```

### Installing locally
```bash
make install               # build + sudo cp musing /usr/local/bin/
```

### Version injection
Version metadata lives in **`cmd/root.go`** as package-level vars `Version`, `Commit`, `Date` (defaults `"0.1.0"`, `"none"`, `"unknown"`). Releases inject real values via GoReleaser ldflags targeting `…/cmd.Version` / `.Commit` / `.Date`.

> ⚠️ Note: the local `Makefile` ldflags currently target `main.version`/`main.commit`/`main.date`, which no longer exist (the vars moved to package `cmd`). The Go linker silently ignores unknown `-X` targets, so `make build` produces a binary that reports the default `0.1.0`. GoReleaser builds are correct.

### Release pipeline
- `.github/workflows/release.yml`: on push to `main` (ignoring Markdown/LICENSE/images), auto-bumps a tag (`mathieudutour/github-tag-action`, default patch) and runs GoReleaser.
- `.goreleaser.yml`: cross-compiles linux/darwin/windows × amd64/arm64 (excluding windows/arm64), publishes archives + checksums, a GitHub release, and a Homebrew formula to the `stevengregory/homebrew-musing` tap.

## Testing

Unit tests exist and should pass before changes land:
```bash
go test ./...
```

Current coverage (pure/unit, no live Docker/SSH/Mongo):
- `cmd/ssh_test.go` — `buildSSHArgs` (key + tunnel args, default ports), `expandHomeDir`.
- `cmd/tunnel_test.go` — tunnel start/stop behavior using the `execCommand`/`checkPort` seams.
- `cmd/monitor_test.go` — `getStatus`.
- `internal/config/config_test.go` — project-root discovery, `compose.yaml` requirement, `GetAPIRepos`, `apiReposDir` resolution.
- `internal/mongo/deploy_test.go` — collection discovery (incl. empty-file error), `getCollectionKeys`.

Prefer unit tests for command parsing, config discovery, alias mapping, SSH-arg building, and Mongo discovery. Manual checks of `dev`/`deploy`/`monitor`/`tunnel`/`ssh` touch the user's Docker/DB/SSH and are opt-in.

### Manual smoke checklist
- [ ] `musing monitor` — live dashboard updates
- [ ] `musing dev` / `dev stop` / `dev rebuild` / `dev logs`
- [ ] `musing deploy` (dev) and `musing deploy news prod` (confirmation + tunnel auto-open/close)
- [ ] `musing tunnel start|status|stop`
- [ ] `musing ssh`
- [ ] `musing version` and `musing --version`

## Future Enhancements

Roadmap candidates (the SSH tunnel command is already implemented):
- [ ] Build command — frontend build with size analysis
- [ ] Selective log streaming by service
- [ ] Restart individual services
- [ ] Config command — manage/validate `.musing.yaml`
- [ ] Split-pane TUI — logs + status in one view

## Common Issues & Solutions

### Monitor not updating
Check the ticker timing in `cmd/monitor.go` (`tickCmd` / `tickMsg`) and that `checkHealthCmd` is being scheduled.

### Docker commands fail
Verify Docker Desktop is running; review `internal/docker/docker.go` (`CheckRunning`, `EnsureRunning`).

### "Could not find project root"
Run inside a directory tree that contains both `.musing.yaml` **and** `compose.yaml`. `FindProjectRoot` requires both.

### Styled output missing/odd
`Info`/`Success`/`Warning`/`Error` shell out to `gum`. Ensure `gum` is installed for non-TUI styled output.

### Banner showing where it shouldn't
Check `shouldShowBanner()` in `cmd/root.go` — it suppresses the banner for `monitor`, `version`, `completion`, and `help`.

## Dependencies Reference

### Core
- `github.com/spf13/cobra` — CLI framework (command tree, flags, completion)
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — spinner and shared components
- `github.com/evertras/bubble-table` — table widget for `monitor`
- `github.com/charmbracelet/lipgloss` + `charm.land/lipgloss/v2` — terminal styling
- `github.com/charmbracelet/huh` — confirmation prompts
- `gopkg.in/yaml.v3` — `.musing.yaml` parsing

### Utilities
- `github.com/common-nighthawk/go-figure` — ASCII art banner

### Standard library
Heavy use of `os/exec`, `net`, `net/http`, `time`, `strings`, `fmt`, `path/filepath`, `context`.

### External CLI tools (runtime, not Go deps)
`docker`, `mongoimport`, `ssh`, `lsof`, `kill`, `gum`.

## File Modification Notes

- **cmd/musing/main.go**: entry point only (`cmd.Execute()`).
- **cmd/root.go**: command registration, groups, banner logic, version vars.
- **cmd/project.go**: shared project-discovery helpers — reuse these.
- **cmd/*.go**: each command is self-contained; tunnel/ssh share `buildSSHArgs`.
- **internal/config/config.go**: add services/fields and discovery logic here.
- **internal/health/health.go**: health-check logic.
- **internal/mongo/deploy.go**: collection discovery and `mongoimport` flags.
- **internal/ui/**: styled output, prompts, spinners.
- **.musing.yaml**: per-project config (lives in the managed project, not this repo).

## Context for Code Changes

When making changes:
1. **Maintain consistency**: match existing patterns and styling.
2. **Test**: run `go test ./...` and `make build`; add unit tests for parsing/discovery/helpers.
3. **Update docs**: keep `README.md`, `AGENTS.md`, and this file in sync.
4. **Mind production safety**: don't weaken the deploy confirmation or tunnel auto-open/close behavior.
5. **Keep it agnostic**: avoid hard-coding the current `stevengregory.io` stack.
