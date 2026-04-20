# Deckhand

A CLI tool for orchestrating Docker-based development environments on remote servers.

Deckhand manages the full lifecycle of containerized dev environments — from spinning up containers with the right tools and services, to managing ports, opening shells, and streaming logs. Installed on the server, not the client.

## Install

### One-liner install (Linux amd64/arm64)

```bash
curl -fsSL https://raw.githubusercontent.com/TomasGrbalik/Deckhand/main/install.sh | sh
```

This detects your architecture, downloads the latest release tarball and its `checksums.txt`, verifies the SHA256, and installs the binary to `/usr/local/bin/deckhand`.

Pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/TomasGrbalik/Deckhand/main/install.sh | VERSION=v0.2.0 sh
```

### Binary download (manual)

If you prefer to download manually:

```bash
# Linux amd64
VERSION=$(curl -sL https://api.github.com/repos/TomasGrbalik/Deckhand/releases/latest | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')
curl -Lo deckhand.tar.gz "https://github.com/TomasGrbalik/Deckhand/releases/latest/download/deckhand_${VERSION}_linux_amd64.tar.gz"
tar xzf deckhand.tar.gz deckhand
sudo mv deckhand /usr/local/bin/
rm deckhand.tar.gz
```

Each release includes a `checksums.txt` file for verification. For arm64, replace `amd64` with `arm64`.

### go install

```bash
go install github.com/TomasGrbalik/Deckhand/cmd/deckhand@latest
```

### Build from source

```bash
git clone https://github.com/TomasGrbalik/Deckhand.git
cd Deckhand
go build -o deckhand ./cmd/deckhand
```

### Prerequisites

- Docker with Compose V2

## Quick Start

```bash
cd ~/my-project
deckhand init          # Create .deckhand.yaml config
deckhand up            # Start containers
deckhand connect --host user@myserver  # Get SSH tunnel command
deckhand shell         # Open a shell in the container
deckhand down          # Stop the environment
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Create a `.deckhand.yaml` config file |
| `up` | Start the dev environment |
| `down` | Stop the dev environment |
| `destroy` | Destroy the environment and remove all generated files |
| `shell` | Open an interactive shell in a container |
| `exec` | Run a command in the devcontainer |
| `logs` | Stream container logs |
| `status` | Show status of the current project's containers |
| `list` | List all deckhand environments on this host |
| `port list` | Show all port mappings for the current project |
| `port add` | Add a port mapping |
| `port remove` | Remove a port mapping |
| `connect` | Print the SSH tunnel command for this project's ports |
| `template list` | List available templates |
| `doctor` | Validate prerequisites and diagnose setup problems |
| `completion` | Generate shell completions for bash, zsh, or fish |

See [docs/commands.md](docs/commands.md) for detailed usage, flags, and examples.

## How It Works

1. `deckhand init` creates `.deckhand.yaml` with your project config
2. `deckhand up` renders a Dockerfile and docker-compose.yml from templates into `.deckhand/`, then runs `docker compose up`
3. Containers are labeled with `dev.deckhand.*` for discovery
4. `shell`, `exec`, and `logs` find containers by label and interact via the Docker SDK
5. All ports bind to `127.0.0.1` only — use `deckhand connect` to get the SSH tunnel command

## Documentation

- [Command Reference](docs/commands.md) — usage, flags, and examples for every command
- [Configuration](docs/configuration.md) — project and global config files
- [Custom Templates](docs/custom-templates.md) — create templates for any language or toolchain
- [Networking](docs/networking.md) — shared Docker network with static IPs for direct SSH access via Tailscale
- [Companion Services](docs/companion-services.md) — PostgreSQL, Redis, and more alongside your devcontainer
- [Mounts](docs/mounts.md) — volumes, secrets, sockets, and credential recipes
- [Environment Variables](docs/environment-variables.md) — override project settings via env vars
- [Shell Completions](docs/shell-completions.md) — tab completion for bash, zsh, and fish

## Status

All 5 implementation phases are complete. Deckhand supports the full dev environment lifecycle, port management, SSH tunneling, companion services, credential forwarding, and shell completions.
