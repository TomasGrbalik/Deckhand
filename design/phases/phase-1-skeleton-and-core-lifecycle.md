# Phase 1: Skeleton & Core Lifecycle

**Status:** Active

**Goal:** You can `deckhand up` a hardcoded dev container, `deckhand shell` into it, and `deckhand down` to stop it. The tool exists, builds, and does something real.

---

## Scope & Boundaries

What's IN this phase:
- Go project scaffold that compiles and runs
- A single `base` template (Ubuntu, basic dev tools, no language-specific stuff)
- Core lifecycle: init, up, down, destroy
- Container interaction: shell, exec, logs
- Project config loading from `.deckhand.yaml`
- Tests for every piece

What's NOT in this phase (comes later):
- Multiple templates or interactive init prompts (Phase 3)
- Port management commands or `connect` (Phase 2)
- Container labels for discovery (Phase 2)
- Global config or credential injection (Phase 4)
- `list` and `status` commands (Phase 2 — need labels first)

---

## Tasks

### 1. Project scaffold

Initialize Go module and directory structure. After this task, `go build ./cmd/deckhand` produces a binary that prints help text.

Files to create:
- `go.mod` — module `github.com/TomasGrbalik/deckhand`
- `cmd/deckhand/main.go` — entry point, calls `cli.Execute()`
- `internal/cli/root.go` — root Cobra command with `--version`, `--verbose` flags
- `.gitignore` — Go binaries, `.deckhand/` directory

Dependencies to add: `spf13/cobra`.

**Tests:** Run `go build` and `go vet` pass. The binary runs and prints help.

### 2. Domain types

Define the core data structures that every other package will use. These live in `internal/domain/` and have zero external dependencies.

- `project.go` — `Project` struct: name, template name, ports list, env map. This is what `.deckhand.yaml` deserializes into.
- `container.go` — `Container` struct: ID, name, service name, status, health.
- `port.go` — `PortMapping` struct: port number, name, protocol (http/tcp), internal flag.

**Tests:** Unit tests that construct these types and verify fields. Simple but ensures the package compiles and the structs serialize/deserialize correctly with YAML tags.

### 3. Config loading

Read `.deckhand.yaml` from the current directory and parse it into a `domain.Project`. Lives in `internal/config/`.

- `config.go` — `Load(path string) (*domain.Project, error)`. Uses `koanf` to read YAML.
- `paths.go` — helper functions: `ProjectConfigPath()` returns `.deckhand.yaml`, `GeneratedDir()` returns `.deckhand/`.

Dependencies to add: `knadh/koanf`.

**Tests:** Write a temp `.deckhand.yaml` file, call `Load()`, verify the returned `Project` has correct values. Test missing file returns a clear error. Test invalid YAML returns a clear error.

### 4. Base template

Create the simplest possible dev container template. Lives in `templates/base/`.

`Dockerfile.tmpl`:
- Based on `ubuntu:24.04`
- Installs basic tools: git, curl, wget, ripgrep, build-essential, zsh
- Creates a non-root user (`dev`, UID 1000)
- Sets working directory to `/workspace`

`compose.yaml.tmpl`:
- Single `devcontainer` service
- Bind-mounts current directory to `/workspace`
- Ports: `127.0.0.1:8080:8080` (hardcoded for now)
- `command: sleep infinity`
- Uses Go `text/template` syntax with variables from `domain.Project`

Embed templates into the binary using `internal/infra/template/embedded.go` with Go's `embed.FS`.

**Tests:** Render both templates with a sample `Project`, verify output is valid YAML (for compose) and valid Dockerfile syntax. Verify the compose output binds ports to `127.0.0.1`. Verify the `DO NOT EDIT` header is present.

### 5. Template rendering service

Lives in `internal/service/template.go`. Takes a `domain.Project`, finds the matching template, renders it with Go's `text/template`, and returns the rendered Dockerfile and compose YAML as strings.

Define an interface for the template source:
```go
type TemplateSource interface {
    Load(templateName string) (dockerfile string, compose string, error)
}
```

The embedded template loader (`internal/infra/template/embedded.go`) implements this interface. Currently `Load` is a package-level function — task 5 should wrap it in a struct type to satisfy the interface. Later, a git-based loader will too.

**Tests:** Render the base template with a test project config. Verify output contains expected service name, port bindings, image name, volume mounts.

### 6. Docker infrastructure layer

Lives in `internal/infra/docker/`. Wraps Docker operations that deckhand needs.

`compose.go` — Shells out to `docker compose` CLI:
- `ComposeUp(projectDir, composePath string, build bool) error`
- `ComposeDown(projectDir, composePath string) error`
- `ComposeDestroy(projectDir, composePath string) error` — down with `-v --remove-orphans`

`container.go` — Uses Docker SDK (`docker/docker/client`):
- `Exec(containerName string, cmd []string, tty bool) error` — interactive exec with TTY, raw mode, SIGWINCH
- `Logs(containerName string, follow bool, tail string) (io.ReadCloser, error)` — stream logs
- `FindContainer(projectName, serviceName string) (string, error)` — find container ID by compose project + service name

Why the split: compose operations are best done via the CLI (it handles all the orchestration complexity). Container-level operations (exec, logs) need the SDK for proper TTY handling.

Dependencies to add: `docker/docker/client`, `golang.org/x/term`.

**Tests:**
- `compose.go`: Integration tests (require Docker). Start a simple container, verify it's running, stop it. Tag with build constraint so `go test -short` skips them.
- `container.go`: Integration tests for exec and logs against a running container. Test that `FindContainer` returns the right ID.

### 7. Environment service

Lives in `internal/service/environment.go`. Orchestrates the full lifecycle. This is the "brain" that the CLI commands call.

```go
type EnvironmentService struct {
    templates TemplateSource
    compose   ComposeRunner
    config    *domain.Project
}
```

Methods:
- `Up(build bool) error` — render templates → write to `.deckhand/` → compose up
- `Down() error` — compose down
- `Destroy() error` — compose destroy → remove `.deckhand/` directory

Define interfaces for `ComposeRunner` so the service can be tested without Docker:
```go
type ComposeRunner interface {
    Up(projectDir, composePath string, build bool) error
    Down(projectDir, composePath string) error
    Destroy(projectDir, composePath string) error
}
```

**Tests:** Unit tests with a fake `ComposeRunner` and fake `TemplateSource`. Verify `Up()` calls render then compose up. Verify `Destroy()` calls compose destroy and cleans up the directory. No Docker needed.

### 8. Container service

Lives in `internal/service/container.go`. Handles shell, exec, and logs.

```go
type ContainerService struct {
    docker ContainerRunner
}
```

Define the interface:
```go
type ContainerRunner interface {
    Exec(containerName string, cmd []string, tty bool) error
    Logs(containerName string, follow bool, tail string) (io.ReadCloser, error)
    FindContainer(projectName, serviceName string) (string, error)
}
```

Methods:
- `Shell(project string, service string, cmd string) error` — find container, exec with TTY
- `Exec(project string, service string, cmd []string) error` — find container, exec without TTY
- `Logs(project string, service string, follow bool, tail int) error` — find container, stream logs to stdout

**Tests:** Unit tests with a fake `ContainerRunner`. Verify correct container is looked up, correct commands are passed through.

### 9. CLI commands

One file per command in `internal/cli/`. Each command is thin: parse flags → load config → create service → call method → print output.

- `init.go` — `deckhand init [--template base] [--project name]`. Writes a default `.deckhand.yaml`. No interactive prompts yet (Phase 3).
- `up.go` — `deckhand up [--build]`. Calls `EnvironmentService.Up()`, prints what happened.
- `down.go` — `deckhand down`. Calls `EnvironmentService.Down()`.
- `destroy.go` — `deckhand destroy [--yes]`. Confirms unless `--yes`, calls `EnvironmentService.Destroy()`.
- `shell.go` — `deckhand shell [--service name] [--cmd command]`. Calls `ContainerService.Shell()`.
- `exec.go` — `deckhand exec <cmd> [args...]`. Calls `ContainerService.Exec()`.
- `logs.go` — `deckhand logs [service] [--follow] [--tail n]`. Calls `ContainerService.Logs()`.

Wire all commands to root in `root.go`.

**Tests:** Smoke tests that verify commands are registered (no typos in command names), flags parse correctly, and `--help` produces expected output.

---

## Suggested Implementation Order

The tasks above are numbered in dependency order. Each builds on the previous:

1. **Scaffold** — the binary exists
2. **Domain types** — shared data structures
3. **Config loading** — can read project config
4. **Base template** — has something to render
5. **Template rendering service** — can produce compose/Dockerfile from config
6. **Docker infra layer** — can talk to Docker
7. **Environment service** — orchestrates lifecycle
8. **Container service** — handles shell/exec/logs
9. **CLI commands** — wires it all together

After task 9, the full Phase 1 flow works end-to-end.

---

## End-to-End Smoke Test

When Phase 1 is complete, this sequence should work:

```bash
mkdir ~/test-project && cd ~/test-project

# Create config
deckhand init --template base --project test-project
# → creates .deckhand.yaml

# Start environment
deckhand up
# → creates .deckhand/Dockerfile, .deckhand/docker-compose.yml
# → builds image, starts container
# → prints status

# Open a shell
deckhand shell
# → drops into zsh inside the container at /workspace
# → project files are visible
# → exit to return

# Run a command
deckhand exec ls /workspace
# → prints directory listing

# View logs
deckhand logs
# → shows container output

# Stop
deckhand down
# → containers stop, volumes preserved

# Start again
deckhand up
# → containers start, data intact

# Full cleanup
deckhand destroy --yes
# → containers removed, volumes removed, .deckhand/ removed
```
