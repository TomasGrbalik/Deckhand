# Configuration

Deckhand uses two config files: a per-project config and an optional global config.

## Project Config

**Location:** `.deckhand.yaml` in the project root

Created by `deckhand init`. This is the primary config file for your dev environment.

```yaml
version: 1
project: my-api
template: go

ports:
  - port: 3000
    name: api
    protocol: http
  - port: 5432
    name: postgres
    protocol: tcp
    internal: true

env:
  GO_ENV: development
  DATABASE_URL: postgresql://dev:secret@postgres:5432/appdb

variables:
  go_version: "1.23"

mounts:
  volumes:
    - name: go-mod-cache
      target: /home/dev/go/pkg/mod

services:
  - name: postgres
    enabled: true
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `version` | int | `1` | Config schema version. Only version 1 is supported. |
| `project` | string | | Project name, used for Docker labels and volume naming. |
| `template` | string | | Template to use (e.g. `base`, `python`). |
| `ports` | list | `[]` | Port mappings exposed by the dev environment. |
| `env` | map | `{}` | Static environment variables passed to the devcontainer. |
| `variables` | map | `{}` | Template-specific variables (override template defaults). |
| `mounts` | object | | Volumes, secrets, and sockets. See [Mounts](mounts.md). |
| `services` | list | `[]` | Companion services. See [Companion Services](companion-services.md). |

### Port mapping fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | | Port number (e.g. `3000`). |
| `name` | string | | Human-readable label. |
| `protocol` | string | | `http` or `tcp`. |
| `internal` | bool | `false` | If true, not exposed for SSH tunneling. |

### Service fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | | Service name (e.g. `postgres`, `redis`). |
| `version` | string | | Service version (reserved for future use). |
| `enabled` | bool | | Whether the service is active. |

## Global Config

**Location:** `~/.config/deckhand/config.yaml`

Optional. Applies defaults and shared settings across all projects.

```yaml
defaults:
  template: go
  shell: zsh

ssh:
  user: dev
  host: 100.64.1.3

network:
  name: ssh-net
  subnet: 172.30.0.0/24
  gateway: 172.30.0.1

mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}
      env: GH_TOKEN
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
  sockets:
    - name: ssh-agent
      source: ${SSH_AUTH_SOCK}
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `defaults.template` | string | Default template for all projects. |
| `defaults.shell` | string | Default shell command (e.g. `zsh`). |
| `ssh.user` | string | SSH user for `deckhand connect`. |
| `ssh.host` | string | SSH host for `deckhand connect`. |
| `network.name` | string | Name of the shared Docker network for SSH access. See [Networking](networking.md). |
| `network.subnet` | string | Subnet CIDR (e.g. `172.30.0.0/24`). |
| `network.gateway` | string | Gateway address (e.g. `172.30.0.1`). |
| `mounts` | object | Global mounts merged into every project. See [Mounts](mounts.md). |

## Precedence

Settings are resolved in this order (later wins):

1. Template defaults (from `metadata.yaml`)
2. Global config (`~/.config/deckhand/config.yaml`)
3. Project config (`.deckhand.yaml`)
4. Environment variables (`DECKHAND_PROJECT`, `DECKHAND_TEMPLATE`)
5. CLI flags (`--template`, `--project`)

## Generated Files

Running `deckhand up` renders templates into the `.deckhand/` directory:

```text
.deckhand/
  Dockerfile
  docker-compose.yml
```

These files are generated and include a "DO NOT EDIT" header. They are recreated on every `deckhand up`. Add `.deckhand/` to your `.gitignore`.
