# AGENTS.md

## Purpose

This file gives Codex durable project guidance for the `musing-cli` repository.
Follow it for work in this repo unless a more specific nested `AGENTS.md` or an
explicit user instruction overrides it.

## Project Snapshot

- This is a Go CLI named `musing` for managing multi-service development stacks.
- The command layer lives in `cmd/` and uses Cobra.
- The entry point is `cmd/musing/main.go`; command registration and global
  banner/version behavior live in `cmd/root.go`.
- Shared implementation packages live under `internal/`:
  - `internal/config`: `.musing.yaml` discovery and typed project config.
  - `internal/docker`: Docker Desktop and Docker Compose helpers.
  - `internal/health`: port and HTTP health checks.
  - `internal/mongo`: JSON collection discovery and `mongoimport` deployment.
  - `internal/ui`: terminal styling, prompts, and spinner helpers.
- The CLI is project-agnostic. Avoid hard-coding the current `stevengregory.io`
  stack unless a task explicitly asks for that.

## Current Tooling

- Go version is declared in `go.mod`.
- Build locally with `make build`; it writes the ignored `./musing` binary.
- Run the full test suite with `go test ./...`.
- Run `go mod tidy` after dependency changes.
- Use `gofmt` on changed Go files.
- Use `git diff --check` before committing code changes.
- Release packaging is configured with `.goreleaser.yml`.
- The GitHub release workflow ignores Markdown-only changes.

## Safety Rules

- Do not run production-affecting commands such as `musing deploy ... prod`,
  `musing tunnel start`, `musing tunnel stop`, `ssh`, `mongoimport`, or `kill`
  unless the user explicitly asks for that runtime action.
- Do not start or stop Docker services with `musing dev`, `docker compose up`,
  or `docker compose down` unless the task requires it or the user asks.
- Prefer unit tests and pure helper tests over live Docker, SSH, or MongoDB
  checks when validating routine code changes.
- Do not commit local binaries, `dist/`, `build/`, `bin/`, environment files, or
  real production secrets.
- If adding examples for `.musing.yaml`, keep hostnames, usernames, ports, and
  key paths generic unless the user provides public-safe values.

## Code Conventions

- Keep command implementations in `cmd/` and reusable logic in `internal/`.
- Use the existing helpers `loadProjectConfig` and `changeToProjectRoot` for
  project discovery and cwd-sensitive commands.
- Preserve the repo's explicit error handling style. Return useful errors and
  let `rootCmd` handle process exit behavior.
- Keep the CLI fast and dependency-light. Do not add new production
  dependencies without a clear reason.
- Match the existing terminal UI style: magenta/purple accents, Lip Gloss/Huh
  components, professional output, and no emoji.
- Existing Unicode status symbols are part of the current UI; do not churn them
  unless the task is specifically about output design.
- Be careful with Lip Gloss imports: most code uses
  `github.com/charmbracelet/lipgloss`, while `bubble-table` styling currently
  requires `charm.land/lipgloss/v2`.
- Keep comments focused on non-obvious behavior, especially around TUI update
  loops, shell command seams, or production safety.

## Testing Guidance

- For most changes, run:

  ```bash
  go test ./...
  make build
  ```

- For dependency or formatting changes, also run:

  ```bash
  go mod tidy
  git diff --check
  ```

- For command parsing, config discovery, alias mapping, SSH args, and Mongo
  collection discovery, add or update unit tests rather than relying on manual
  terminal runs.
- Manual checks for `monitor`, `dev`, `deploy`, `tunnel`, and `ssh` can touch
  the user's Docker stack, database, or SSH config. Treat them as opt-in.

## Git Guidance

- Use Conventional Commits for commits, for example `feat:`, `fix:`,
  `refactor:`, `test:`, `docs:`, `ci:`, and `chore(deps):`.
- Before pushing, fetch or otherwise make sure the branch is up to date with
  its remote; rebase local commits over upstream changes when appropriate.
- Keep dependency updates focused in `go.mod` and `go.sum`, with only the
  compatibility source changes required to keep the build passing.

## Documentation Notes

- Keep `README.md` aligned with user-facing commands, flags, config fields, and
  installation instructions.
- `CLAUDE.md` contains useful background, but verify its details against the
  live code before copying from it because some implementation notes may drift.
- When documenting config, include `services`, `database`, optional
  `production`, optional `apiReposDir`, and service-level `alias`/`collection`
  only when relevant to the change.
