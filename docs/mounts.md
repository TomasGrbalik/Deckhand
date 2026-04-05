# Mounts

Mounts let you share data between the host and your devcontainer. There are three types: volumes, secrets, and sockets.

Mounts can be defined in the global config (`~/.config/deckhand/config.yaml`) to apply to all projects, or in the project config (`.deckhand.yaml`) for project-specific needs. Templates also declare default mounts (e.g. a `workspace` volume).

## Volumes

Named Docker volumes for persistent data.

```yaml
mounts:
  volumes:
    - name: workspace
      target: /workspace
    - name: go-mod-cache
      target: /home/dev/go/pkg/mod
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Volume identifier. Prefixed with the project name in Docker (e.g. `my-api-workspace`). |
| `target` | string | Mount path inside the container. |
| `enabled` | bool | Set to `false` to disable an inherited volume. |

## Secrets

Pass credentials into the container as environment variables or file mounts.

```yaml
mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}
      env: GH_TOKEN
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Secret identifier. |
| `source` | string | Host value: `${VAR}` for an environment variable, or a file path (`~` is expanded). |
| `target` | string | Mount path inside the container (for file-based secrets). |
| `env` | string | Environment variable name to set inside the container. |
| `readonly` | bool | Mount the file as read-only. |
| `enabled` | bool | Set to `false` to disable an inherited secret. |

A secret must have at least one of `env` or `target`.

### Source resolution

- `${VAR}` — resolved from the host environment at render time
- `~/path` — expanded to the home directory
- If the source can't be resolved (missing env var or file), the mount is skipped with a warning

## Sockets

Forward Unix sockets from the host into the container.

```yaml
mounts:
  sockets:
    - name: ssh-agent
      source: ${SSH_AUTH_SOCK}
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Socket identifier. |
| `source` | string | Host socket path: `${VAR}` or a file path. |
| `target` | string | Socket path inside the container. |
| `env` | string | Environment variable pointing to the socket inside the container. |
| `enabled` | bool | Set to `false` to disable an inherited socket. |

## Merge order

Mounts from multiple sources are merged by `name` within each category:

1. Template defaults (lowest precedence)
2. Global config
3. Project config (highest precedence)

A mount with the same name in a higher-precedence source replaces the lower one entirely. Setting `enabled: false` removes an inherited mount.

## Credential recipes

Common configurations for forwarding credentials into the devcontainer.

### SSH agent forwarding

```yaml
mounts:
  sockets:
    - name: ssh-agent
      source: ${SSH_AUTH_SOCK}
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK
```

### GitHub token (HTTPS)

```yaml
mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}
      env: GH_TOKEN
```

### SSH key (file)

```yaml
mounts:
  secrets:
    - name: ssh-key
      source: ~/.ssh/id_ed25519
      target: /home/dev/.ssh/id_ed25519
      readonly: true
```

### Git configuration

```yaml
mounts:
  secrets:
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
```

### GPG agent and keys

```yaml
mounts:
  sockets:
    - name: gpg-agent
      source: ${HOME}/.gnupg/S.gpg-agent.extra
      target: /run/gpg-agent.sock
  secrets:
    - name: gpg-pubring
      source: ~/.gnupg/pubring.kbx
      target: /home/dev/.gnupg/pubring.kbx
      readonly: true
    - name: gpg-trustdb
      source: ~/.gnupg/trustdb.gpg
      target: /home/dev/.gnupg/trustdb.gpg
      readonly: true
```

### Docker socket

```yaml
mounts:
  sockets:
    - name: docker
      source: /var/run/docker.sock
      target: /var/run/docker.sock
```

### Google Cloud credentials

```yaml
mounts:
  secrets:
    - name: gcloud-creds
      source: ~/.config/gcloud/credentials.json
      target: /home/dev/.config/gcloud/credentials.json
      env: GOOGLE_APPLICATION_CREDENTIALS
      readonly: true
```
