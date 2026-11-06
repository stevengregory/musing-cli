# CLAUDE.md

Brief for Claude Code in this repo. Keep it to commands, safety rules, invariants, and gotchas that aren't obvious from the code. Don't mirror structure, signatures, or dependency lists here; read the source instead.

## Project

`musing` is a Go CLI (Cobra + Charm Bracelet TUI) for managing multi-service dev stacks: Docker Compose, MongoDB data deploys, SSH tunnels to production, and port-based health checks, all driven by a per-project `.musing.yaml`. Its main user is the stevengregory.io stack, but it must stay project-agnostic: never hard-code that stack.

`README.md` is the canonical reference for commands, flags, and `.musing.yaml` fields.

## Commands

```bash
go test ./...       # must pass
make build          # writes ./musing (gitignored)
gofmt -w <files>    # on changed Go files
go mod tidy         # after dependency changes
git diff --check    # before committing
```

## Safety

- Don't run anything that touches the user's Docker stack, databases, or servers unless asked: `musing dev|deploy|tunnel|ssh`, `docker compose up/down`, `mongoimport`, `mongosh`, `ssh`, `kill`. `musing monitor` is a full-screen TUI; don't launch it from a shell. `make install` runs `sudo`; leave it to the user.
- MongoDB integration tests are opt-in and need a **disposable** local instance plus `mongosh` and `mongoimport`. The URI must be an explicit loopback `host:port` with no credentials, path, or query:
  ```bash
  MUSING_MONGO_INTEGRATION=1 MUSING_MONGO_TEST_URI=mongodb://127.0.0.1:27099 \
    go test ./internal/mongo -run '^TestMongoSchema.*Integration$' -v
  ```

## Invariants (don't weaken)

- `deploy … prod` asks for confirmation (default No), opens the SSH tunnel only if it's down, and afterwards closes only a tunnel it opened.
- Schema-backed deploys preflight every selected schema on the server before mutating any collection. Replacement is still non-atomic after preflight. The deprecated `database.schemas` map must keep working alongside service-level `schema`.
- `tunnel stop` kills only the single `LISTEN` process on the prod port whose real argv (read via `internal/process`) is `ssh` with exactly `buildSSHArgs(cfg, true)`. Anything else is refused and left running.

## Gotchas

- The project root needs both `.musing.yaml` and `compose.yaml`. Commands that run `docker compose` must `changeToProjectRoot()` first.
- `ui.Info`/`Success`/`Warning`/`Error`, and the spinner when there's no TTY, shell out to the `gum` binary. Other runtime binaries: `docker`, `mongoimport`, `mongosh` (schema deploys only), `ssh`, `lsof`, `kill`.
- Lip Gloss is imported twice: `github.com/charmbracelet/lipgloss` everywhere, plus `charm.land/lipgloss/v2` for `bubble-table` styling in `cmd/monitor.go`.
- New commands are registered in `cmd/root.go` with a `GroupID` (`core` or `additional`). The ASCII banner prints before every command not excluded in `shouldShowBanner()`; add new commands there if their output must stay clean.
- `make build` doesn't inject the version: the Makefile's ldflags target `main.*`, but the vars are `cmd.Version`/`Commit`/`Date`, so local builds report `0.1.0`. GoReleaser injects correctly.
- Unit tests stub shell-outs through package-level vars: `execCommand`, `checkPort`, `readProcessCommand` in `cmd/`; `lookPath`, `runCommand` in `internal/mongo`. Don't invoke real binaries in unit tests.
- `config.MustFindProjectRoot()` calls `os.Exit`, so code paths using it can't be unit-tested. Prefer `loadProjectConfig()` in new code.

## Conventions

- Command wiring in `cmd/`, reusable logic in `internal/`. Use `loadProjectConfig()`/`changeToProjectRoot()` for project discovery.
- Return wrapped errors and let `rootCmd` handle exit.
- Output: magenta/purple accents, the existing Unicode status glyphs (`✓ ✗ ●`), no emoji. Match the surrounding Lip Gloss/Huh code.
- Stay dependency-light; don't add production dependencies without a clear reason.
- When commands, flags, or config fields change, update `README.md`. Keep the rules here consistent with `AGENTS.md`, Codex's equivalent file.

## Git & release

- Conventional Commits (`feat:`, `fix:`, `docs:`, `chore(deps):`, …). PRs are squash-merged, so the PR title becomes the commit on `main`; make it conventional.
- Any push to `main` that changes a file other than Markdown, `LICENSE`, or `images/` auto-tags a patch release and runs GoReleaser (GitHub release + Homebrew tap).
