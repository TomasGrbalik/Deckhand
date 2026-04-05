# Command Reference

All commands support the global `--verbose` (`-v`) flag for detailed output.

## init

Create a new `.deckhand.yaml` config file.

```bash
deckhand init [--template <name>] [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--template` | *(interactive)* | Template to use (skips interactive picker) |
| `--project` | directory name | Project name |

Walks you through template selection, template variables, companion services, and project naming. Creates `.deckhand.yaml` in the current directory.

Templates are loaded in precedence order: project-local (`.deckhand/templates/`) > user (`~/.config/deckhand/templates/`) > embedded.

## up

Start the dev environment.

```bash
deckhand up [--build]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--build` | `false` | Force image rebuild |

Renders the Dockerfile and docker-compose.yml into `.deckhand/`, then starts all containers with `docker compose up`.

## down

Stop the dev environment.

```bash
deckhand down
```

Stops all containers. Volumes and generated files are preserved so `deckhand up` can restart quickly.

## destroy

Destroy the dev environment and remove all generated files.

```bash
deckhand destroy [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | `false` | Skip confirmation prompt |

Stops containers, removes Docker volumes, and deletes the `.deckhand/` directory. Prompts for confirmation unless `--yes` is passed.

## shell

Open an interactive shell in a container.

```bash
deckhand shell [--service <name>] [--cmd <command>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--service` | `devcontainer` | Target service name |
| `--cmd` | `zsh` | Shell command to run |

## exec

Run a command in the devcontainer.

```bash
deckhand exec <command> [args...]
```

Runs the given command in the `devcontainer` service. Arguments are passed through as-is.

```bash
deckhand exec go test ./...
deckhand exec npm install
```

## logs

Stream container logs.

```bash
deckhand logs [service] [--follow] [--tail <n>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--follow`, `-f` | `false` | Follow log output |
| `--tail` | `100` | Number of lines to show from the end |

Defaults to the `devcontainer` service. Pass a service name to view logs for a companion service.

```bash
deckhand logs              # last 100 lines from devcontainer
deckhand logs -f           # stream devcontainer logs
deckhand logs postgres     # last 100 lines from postgres
```

## status

Show status of the current project's containers.

```bash
deckhand status
```

Displays a table with columns: SERVICE, IMAGE, STATUS, HEALTH, PORTS.

## list

List all deckhand environments on this host.

```bash
deckhand list
```

Displays a table with columns: PROJECT, STATUS, SERVICES, UPTIME.

## port

Manage port mappings for the current project.

### port list

```bash
deckhand port list
```

Shows all port mappings with columns: PORT, NAME, PROTOCOL, ACCESS. External ports include the SSH tunnel command; internal ports show "internal only".

### port add

```bash
deckhand port add <port> [--name <label>] [--protocol <proto>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | *(empty)* | Human-readable label for the port |
| `--protocol` | `http` | Protocol: `http` or `tcp` |

Adds a port mapping to the project config and recreates the environment.

```bash
deckhand port add 8080 --name api --protocol http
deckhand port add 5432 --protocol tcp
```

### port remove

```bash
deckhand port remove <port>
```

Removes a port mapping from the project config and recreates the environment.

## connect

Print the SSH tunnel command for this project's ports.

```bash
deckhand connect --host <target>
```

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | *(required)* | SSH target (e.g. `user@myserver`, `hostname:port`) |

Outputs a single SSH command with `-L` tunnel mappings for all external ports.

```bash
$ deckhand connect --host dev@myserver
ssh -N -L 3000:localhost:3000 -L 8080:localhost:8080 dev@myserver
```

## template

Manage templates.

### template list

```bash
deckhand template list
```

Displays a table with columns: NAME, DESCRIPTION, SOURCE.

Templates are loaded from three sources (later sources override earlier ones with the same name):
1. Embedded (built-in)
2. User (`~/.config/deckhand/templates/`)
3. Project-local (`.deckhand/templates/`)

## doctor

Validate prerequisites and diagnose setup problems.

```bash
deckhand doctor
```

Runs these checks:
- **Docker Daemon** — verifies the Docker daemon is reachable
- **Compose V2** — checks Docker Compose is available and reports its version
- **Global Config** — validates `~/.config/deckhand/config.yaml`
- **Project Config** — validates `.deckhand.yaml` (skipped if not present)
- **Template** — verifies the configured template exists and is loadable

Each check reports PASS, FAIL, or SKIP. Exits with a non-zero code if any check fails.

## completion

Generate shell completion scripts.

```bash
deckhand completion bash
deckhand completion zsh
deckhand completion fish
```

See [Shell Completions](shell-completions.md) for setup instructions.
