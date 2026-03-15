# Deckhand

CLI tool for orchestrating Docker-based dev environments on remote servers. Written in Go. Installed on the server, not the client.

## Project Structure

```
cmd/deckhand/main.go     Entry point
internal/cli/            Cobra commands (thin — parse flags, call service, print output)
internal/service/        Business logic (shared by CLI and future TUI)
internal/domain/         Pure data types (zero dependencies)
internal/infra/docker/   Docker SDK wrapper, compose operations
internal/infra/template/ Template loading (embedded in binary)
internal/config/         Config loading (project + global)
templates/               Bundled template files (embedded via embed.FS)
design/                  Design docs (user stories, commands, architecture, phases)
research/                Research docs (reference only, not part of the tool)
```

## Architecture Rules

- Three layers: CLI → Service → Infrastructure. Each layer only calls the one below it.
- CLI layer is thin: parse flags, call service, format output. No business logic.
- Service layer contains all orchestration logic. No Docker SDK imports, no CLI imports.
- Domain types have zero external dependencies.
- Infra layer wraps external systems (Docker, filesystem). Thin adapters.

## Key Dependencies

- `spf13/cobra` — CLI framework
- `charmbracelet/huh` — Interactive prompts
- `knadh/koanf` — Config (not Viper)
- `docker/docker/client` — Docker SDK
- `golang.org/x/term` — TTY handling for shell/exec

## Testing

- Every task includes its tests — implementation and tests are one unit.
- Service layer: unit tests with interface-based fakes (no Docker needed).
- Infra layer: integration tests against real Docker (skip with `go test -short`).
- CLI layer: minimal smoke tests (flag parsing, exit codes).
- Template tests: render and validate output is correct YAML with expected structure.
- No mocking the Docker SDK — use interfaces in the service layer instead.

## Implementation Phases

See `design/phases/` for full details. **Phase 1 — Skeleton & Core Lifecycle** is complete. Next: **Phase 2**.

## Conventions

- Generated files go in `.deckhand/` with a "DO NOT EDIT" header.
- User config is `.deckhand.yaml` in project root.
- Global config is `~/.config/deckhand/config.yaml`.
- Container labels use `dev.deckhand.*` prefix for discovery.
- All Docker ports bind to `127.0.0.1` only.

## Quick Reference

```bash
go build ./...          # Build project
go test ./...           # Run tests
golangci-lint run ./... # Run linter
golangci-lint fmt ./... # Format code
```

## Quality Gate

- Tests must pass before completing any task
- `go test ./...` at the end of every task
- Run `go mod tidy` after adding/removing dependencies
- Quality hooks run automatically — don't disable them

## For Agents

- Before starting work, validate that this CLAUDE.md still reflects the actual codebase. If the structure, conventions, or architecture have drifted, update this file first.
- Check `design/phases/` to understand what phase we're in and what's been completed.
- Read the relevant design docs before implementing — don't guess at the intended behavior.
- The user is learning Go — explain Go-specific decisions when they're non-obvious.
