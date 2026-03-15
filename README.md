# Deckhand

A CLI tool for orchestrating Docker-based development environments on remote servers.

Deckhand manages the full lifecycle of containerized dev environments — from spinning up containers with the right tools and services, to opening shells and streaming logs. Installed on the server, not the client.

## Quick Start

### Prerequisites

- Go 1.22+
- Docker with Compose V2

### Install

```bash
go install github.com/TomasGrbalik/deckhand/cmd/deckhand@latest
```

Or build from source:

```bash
git clone https://github.com/TomasGrbalik/deckhand.git
cd deckhand
go build -o deckhand ./cmd/deckhand
```

### Usage

```bash
# Initialize a new project
cd ~/my-project
deckhand init

# Start the dev environment
deckhand up

# Open an interactive shell in the container
deckhand shell

# Run a one-off command
deckhand exec go test ./...

# View container logs
deckhand logs

# Stop the environment (preserves volumes)
deckhand down

# Tear down everything
deckhand destroy --yes
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create a `.deckhand.yaml` config file |
| `up` | Render templates and start containers |
| `down` | Stop containers (preserves volumes) |
| `destroy` | Stop containers, remove volumes and generated files |
| `shell` | Open an interactive shell in a container |
| `exec` | Run a one-off command in the devcontainer |
| `logs` | Stream container logs |

### init

```bash
deckhand init [--template <name>] [--project <name>]
```

Creates a `.deckhand.yaml` config file in the current directory. Defaults to the `base` template and uses the directory name as the project name.

### up

```bash
deckhand up [--build]
```

Renders the Dockerfile and docker-compose.yml into `.deckhand/`, then starts all containers. Use `--build` to force an image rebuild.

### down

```bash
deckhand down
```

Stops all containers. Volumes and generated files are preserved so `deckhand up` can restart quickly.

### destroy

```bash
deckhand destroy [--yes]
```

Stops containers, removes volumes and networks, and deletes the `.deckhand/` directory. Prompts for confirmation unless `--yes` is passed.

### shell

```bash
deckhand shell [--service <name>] [--cmd <command>]
```

Opens an interactive TTY shell in a running container. Defaults to the `devcontainer` service with `zsh`.

### exec

```bash
deckhand exec <command> [args...]
```

Runs a command in the devcontainer without a TTY. Arguments are passed through as-is.

### logs

```bash
deckhand logs [service] [--follow] [--tail <n>]
```

Streams container logs. Defaults to the `devcontainer` service, last 100 lines. Use `--follow` to stream continuously.

## Project Config

Deckhand reads `.deckhand.yaml` from the project root:

```yaml
project: my-api
template: base

ports:
  - port: 3000
    name: api
    protocol: http
  - port: 5432
    name: postgres
    protocol: tcp
    internal: true

env:
  DATABASE_URL: postgresql://dev:secret@postgres:5432/appdb
```

## How It Works

1. `deckhand init` creates `.deckhand.yaml` with your project config
2. `deckhand up` renders a Dockerfile and docker-compose.yml from bundled templates into `.deckhand/`, then runs `docker compose up`
3. Containers are labeled with `dev.deckhand.*` for discovery
4. `shell`, `exec`, and `logs` find containers by label and interact via the Docker SDK
5. All ports bind to `127.0.0.1` only — use SSH tunnels for remote access

## Status

Phase 1 (Skeleton & Core Lifecycle) is complete. See `design/` for architecture and planning docs.
