# Deckhand: Architecture

## The Big Picture

Deckhand has three layers. Each layer only talks to the one below it:

```
┌─────────────────────────────────────────────┐
│  Presentation (what the user sees)          │
│  CLI commands (Cobra) + TUI dashboard       │
│  Handles flags, prompts, output formatting  │
└──────────────────────┬──────────────────────┘
                       │ calls
┌──────────────────────▼──────────────────────┐
│  Services (the brain)                       │
│  Business logic, orchestration              │
│  "start environment" = render compose file  │
│  + build image + docker up + health check   │
└──────────────────────┬──────────────────────┘
                       │ calls
┌──────────────────────▼──────────────────────┐
│  Infrastructure (talks to the outside)      │
│  Docker SDK, filesystem, template files     │
└─────────────────────────────────────────────┘
```

Why this matters: the CLI layer never talks to Docker directly. If we add a TUI dashboard later, it calls the same services. If we swap out how templates are stored (embedded → git), only the infrastructure layer changes.

---

## Project Structure

```
deckhand/
├── cmd/deckhand/
│   └── main.go              # Entry point. Wires everything together, runs the CLI.
│
├── internal/                # Private to this project. Go enforces this —
│   │                        # nothing outside deckhand can import these packages.
│   │
│   ├── cli/                 # One file per command group
│   │   ├── root.go          # Root command, global flags
│   │   ├── init.go          # deckhand init
│   │   ├── up.go            # deckhand up
│   │   ├── down.go          # deckhand down
│   │   ├── destroy.go       # deckhand destroy
│   │   ├── list.go          # deckhand list
│   │   ├── status.go        # deckhand status
│   │   ├── shell.go         # deckhand shell
│   │   ├── exec.go          # deckhand exec
│   │   ├── logs.go          # deckhand logs
│   │   ├── port.go          # deckhand port (list/add/remove)
│   │   ├── connect.go       # deckhand connect
│   │   └── template.go      # deckhand template list
│   │
│   ├── service/             # Business logic. No Docker imports, no CLI imports.
│   │   ├── environment.go   # Up, down, destroy, status orchestration
│   │   ├── container.go     # Shell, exec, logs
│   │   ├── port.go          # Port add/remove/list, connect command generation
│   │   └── template.go      # Template listing, rendering
│   │
│   ├── domain/              # Pure data types. Zero dependencies on anything.
│   │   ├── project.go       # Project config (what .deckhand.yaml maps to)
│   │   ├── container.go     # Container info (name, status, ports, health)
│   │   ├── template.go      # Template definition
│   │   └── port.go          # Port mapping
│   │
│   ├── infra/               # Talks to external systems
│   │   ├── docker/
│   │   │   ├── client.go    # Docker SDK wrapper (connect, ping, version)
│   │   │   ├── container.go # Container operations (list, exec, logs, stats)
│   │   │   └── compose.go   # Compose file rendering + docker compose up/down
│   │   └── template/
│   │       └── embedded.go  # Reads bundled templates from the binary
│   │
│   └── config/
│       ├── config.go        # Load/parse config (project + global)
│       └── paths.go         # Where things live (~/.config/deckhand/, .deckhand/)
│
├── templates/               # Bundled template files (embedded into binary at build)
│   ├── base/
│   │   ├── Dockerfile.tmpl
│   │   └── compose.yaml.tmpl
│   ├── go/
│   │   ├── Dockerfile.tmpl
│   │   └── compose.yaml.tmpl
│   └── node/
│       ├── Dockerfile.tmpl
│       └── compose.yaml.tmpl
│
├── go.mod                   # Go's package manifest (like package.json)
├── go.sum                   # Lock file (like package-lock.json)
└── .goreleaser.yaml         # Build/release config (later)
```

### Go concepts worth knowing

- **`cmd/deckhand/main.go`** — Go convention. The directory name under `cmd/` becomes the binary name. `main.go` is just the entry point that wires things together and calls `cli.Execute()`.

- **`internal/`** — Go has a built-in privacy mechanism: anything under `internal/` can only be imported by code in the parent directory. This means our packages are truly private — no one can import `deckhand/internal/service` from another project. It's Go's way of saying "this is implementation detail."

- **One package = one directory** — In Go, every directory is a package. Files in the same directory share the same namespace (no need to import between files in the same package). So `service/environment.go` and `service/port.go` can call each other's functions directly.

- **`go.mod`** — Like `package.json`. Declares the module name and dependencies. `go.sum` is the lock file.

---

## How the Layers Connect

Here's a concrete example: what happens when the user runs `deckhand up`.

```
User runs: deckhand up --build

1. cli/up.go
   - Cobra parses the --build flag
   - Loads project config via config.Load()
   - Creates an EnvironmentService (injecting dependencies)
   - Calls environmentService.Up(config, options)

2. service/environment.go — Up()
   - Calls templateService.Render(config) → produces Dockerfile + compose YAML
   - Writes rendered files to .deckhand/
   - Calls docker.ComposeUp(projectDir, build=true)
   - Calls docker.WaitForHealthy(services)
   - Returns status info (what's running, ports, errors)

3. infra/docker/compose.go — ComposeUp()
   - Shells out to `docker compose -f .deckhand/docker-compose.yml up -d`
   - (We use the docker compose CLI here, not the SDK, because
   -  compose orchestration is complex and the CLI handles it well)

4. cli/up.go
   - Receives status info from the service
   - Formats and prints the status table + connect command
```

The service layer is where the interesting logic lives. The CLI layer is thin (parse flags → call service → print result). The infra layer is thin (translate service requests into Docker API calls).

---

## Key Dependencies

| What | Library | Why |
|------|---------|-----|
| CLI framework | `spf13/cobra` | Industry standard. Docker CLI, GitHub CLI, kubectl all use it. |
| Interactive prompts | `charmbracelet/huh` | Modern, good-looking prompts built on Bubbletea. |
| Config loading | `knadh/koanf` | Lighter and more correct than the popular Viper library. |
| Docker operations | `docker/docker/client` | Official Docker SDK for container inspection, exec, logs. |
| Compose file parsing | `compose-spec/compose-go` | Official library for building docker-compose.yml programmatically. |
| Terminal raw mode | `golang.org/x/term` | Needed for `deckhand shell` to handle TTY properly. |
| TUI (later) | `charmbracelet/bubbletea` | For the dashboard view. Same ecosystem as huh. |

---

## What Gets Generated

When `deckhand up` runs, the `.deckhand/` directory ends up looking like:

```
.deckhand/
├── docker-compose.yml    # Generated from template + .deckhand.yaml
└── Dockerfile            # Generated from template
```

Both files have a header:
```
# Generated by deckhand — DO NOT EDIT
# Source: .deckhand.yaml
# Regenerate with: deckhand up
```

The user's `.deckhand.yaml` stays in the project root and is the only file they edit.

---

## Open Questions

- **SSH host for `connect` command** — How does deckhand know the server's Tailscale IP? Global config? Auto-detect via `tailscale status`? Flag? Deferred for now.
- **Template inheritance** — Base templates that language templates extend. The mechanism (Go template blocks vs file overlay) needs design when we build the template system.
- **Credential injection** — The exact mounts and env vars for SSH agent, GPG, GitHub token. Needs its own design doc when we reach that phase.
