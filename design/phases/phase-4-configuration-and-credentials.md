# Phase 4: Configuration & Mounts

**Status:** Draft

**Goal:** Three mount primitives (volumes, secrets, sockets) replace hardcoded mounts in templates. Global config provides defaults across all projects. Config precedence is implemented.

---

## Design Decisions

### Three mount primitives

Instead of hardcoding support for SSH, GPG, GitHub, etc., the app supports three generic mount primitives. Templates and config declare what they need using these. The app itself doesn't know what SSH or GPG is — it just mounts what's configured.

| Primitive | What it does | Compose mapping |
|---|---|---|
| **Volume** | Persistent named storage | Named volume in service `volumes:` + top-level `volumes:` declaration |
| **Secret** | Credentials — env vars or files | `environment:` entries and/or bind-mounted files in `volumes:` |
| **Socket** | Unix socket forwarding | Bind mount in `volumes:` + env var in `environment:` |

### Workspace becomes a named volume

The current bind mount (`..:/workspace`) is replaced with a named Docker volume. This is the right model for a remote dev environment:

- No UID/permission conflicts between host and container
- Better I/O performance (native filesystem, not bind-overlaid)
- Clean separation — the container owns its workspace
- Code enters via `git clone` inside the container (using GH_TOKEN or SSH key), not from the host

The workspace volume is declared in each template's `metadata.yaml` as a default mount. Users can override it (e.g., switch back to a bind mount if they want).

**Breaking change from Phase 1:** The build context moves from `..` (project root) to `.deckhand/`. Dockerfiles can no longer `COPY` project files during build. This is intentional — dev container Dockerfiles install tools and configure the environment, they don't copy application code. Code lives in the workspace volume.

### Global config provides mount defaults

Users configure credentials once in `~/.config/deckhand/config.yaml`. These apply to every project unless overridden. SSH keys, GH_TOKEN, git config — set once, use everywhere.

### Config precedence

Two kinds of config, two precedence chains.

**Scalar settings** (project name, default template, shell):

```
flags → env vars → project config → global config
```

Example: `deckhand up --project other-name` overrides the name in `.deckhand.yaml`.

**Mounts** (structured, not expressible as flags):

```
project config → global config → template defaults
```

Mounts merge by `name`. A mount with the same name in a higher-precedence source replaces the lower one entirely.

---

## Mount Schemas

### Volume mounts

Named Docker volumes for persistent storage.

```yaml
mounts:
  volumes:
    - name: workspace
      target: /workspace
    - name: go-cache
      target: /home/dev/.cache/go
```

Renders in compose as:

```yaml
services:
  devcontainer:
    volumes:
      - myproject-workspace:/workspace
      - myproject-go-cache:/home/dev/.cache/go
volumes:
  myproject-workspace:
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "myproject"
      dev.deckhand.volume: "workspace"
  myproject-go-cache:
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "myproject"
      dev.deckhand.volume: "go-cache"
```

Volume names are prefixed with the project name to avoid collisions between projects. Labels are used for discovery — `destroy` finds volumes by `dev.deckhand.project` label, not by name prefix.

**Note:** Renaming a project orphans its existing volumes. The `destroy` command uses labels to locate volumes, so orphaned volumes from the old name remain until manually removed.

### Secret mounts

Credentials injected as environment variables, files, or both.

The `source` field determines what kind of secret it is:
- Starts with `${...}` → env var reference, resolved from host environment at render time
- Otherwise → file path on the host (tilde-expanded), bind-mounted into the container

**Env var passthrough** — reads from host environment, sets in container:

```yaml
mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}
      env: GH_TOKEN
```

Renders as:

```yaml
    environment:
      GH_TOKEN: "ghp_abc123..."
```

The `${GH_TOKEN}` is resolved at render time from the host environment. If the env var isn't set, the secret is skipped with a warning.

**File mount** — mounts a host file into the container:

```yaml
mounts:
  secrets:
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
```

Renders as:

```yaml
    volumes:
      - /home/user/.gitconfig:/home/dev/.gitconfig:ro
```

**File + env var** — mount a file AND set an env var pointing to it:

```yaml
mounts:
  secrets:
    - name: gcloud-creds
      source: ~/.config/gcloud/credentials.json
      target: /home/dev/.config/gcloud/credentials.json
      env: GOOGLE_APPLICATION_CREDENTIALS
      readonly: true
```

Renders as:

```yaml
    volumes:
      - /home/user/.config/gcloud/credentials.json:/home/dev/.config/gcloud/credentials.json:ro
    environment:
      GOOGLE_APPLICATION_CREDENTIALS: /home/dev/.config/gcloud/credentials.json
```

### Socket mounts

Unix sockets forwarded from host into container.

```yaml
mounts:
  sockets:
    - name: ssh-agent
      source: ${SSH_AUTH_SOCK}
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK

    - name: gpg-agent
      source: ${HOME}/.gnupg/S.gpg-agent.extra
      target: /run/gpg-agent.sock
```

Renders as:

```yaml
    volumes:
      - /run/user/1000/ssh-agent.sock:/run/ssh-agent.sock
      - /home/user/.gnupg/S.gpg-agent.extra:/run/gpg-agent.sock
    environment:
      SSH_AUTH_SOCK: /run/ssh-agent.sock
```

If the source path (after env var resolution) doesn't exist or the env var isn't set, the socket is skipped with a warning.

---

## Config Files

### Global config

```yaml
# ~/.config/deckhand/config.yaml
defaults:
  template: base
  shell: zsh

# Used by `deckhand connect` (Phase 2) — defined here, wired later.
ssh:
  user: dev
  host: 100.64.1.3

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

### Project config

```yaml
# .deckhand.yaml
project: my-api
template: go

ports:
  - port: 3000
    name: api
    protocol: http

env:
  GO_ENV: development

mounts:
  volumes:
    - name: go-mod-cache
      target: /home/dev/go/pkg/mod

  secrets:
    # Disable a global mount
    - name: gh-token
      enabled: false

    # Add a project-specific secret
    - name: api-key
      source: ${MY_API_KEY}
      env: API_KEY
```

### Template metadata

```yaml
# templates/go/metadata.yaml
name: go
description: Go development environment
variables:
  go_version:
    default: "1.22"
    description: Go version
mounts:
  volumes:
    - name: workspace
      target: /workspace
    - name: go-mod-cache
      target: /home/dev/go/pkg/mod
    - name: go-build-cache
      target: /home/dev/.cache/go-build
```

Every template declares a `workspace` volume. This is the lowest-precedence default.

---

## Mount Merging

Merge order (later wins):

1. Template defaults (`metadata.yaml`)
2. Global config (`~/.config/deckhand/config.yaml`)
3. Project config (`.deckhand.yaml`)

Rules:

- Mounts merge by `name` within each category (volumes, secrets, sockets)
- Same-name mount in higher-precedence config **replaces** the lower one entirely
- `enabled: false` removes a mount inherited from a lower level
- Env var references (`${VAR}`) are resolved at render time, not at config load time
- If an env var or source path is unresolvable, skip the mount and log a warning
- `~` in source paths is expanded to the user's home directory

After merging, the service layer produces a flat, resolved set of mounts that gets passed to the template for rendering.

---

## Compose Template Changes

Templates no longer hardcode any mounts. The compose template renders whatever mounts are configured:

```yaml
# compose.yaml.tmpl
# Generated by deckhand — DO NOT EDIT
# Source: .deckhand.yaml
# Regenerate with: deckhand up
services:
  devcontainer:
    build:
      context: .
      dockerfile: Dockerfile
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "{{ .Name }}"
      dev.deckhand.service: "devcontainer"
{{- if .Volumes }}
    volumes:
{{- range .Volumes }}
      - {{ .ComposeEntry }}
{{- end }}
{{- end }}
{{- if .ExposedPorts }}
    ports:
{{- range .ExposedPorts }}
      - "127.0.0.1:{{ .Port }}:{{ .Port }}"
{{- end }}
{{- end }}
{{- if .Environment }}
    environment:
{{- range .Environment }}
      {{ .Key }}: "{{ .Value }}"
{{- end }}
{{- end }}
    command: sleep infinity
{{- if .NamedVolumes }}

volumes:
{{- range .NamedVolumes }}
  {{ .ComposeName }}:
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "{{ $.Name }}"
      dev.deckhand.volume: "{{ .MountName }}"
{{- end }}
{{- end }}
```

The template data struct provides pre-computed fields:

- **`.Volumes`** — all volume entries (named volumes + bind mounts from secrets/sockets), each with a `.ComposeEntry` method that returns the compose-format string
- **`.Environment`** — sorted slice of key-value pairs (not a map — Go maps iterate in random order, which would make compose output non-deterministic between runs). Merged from static `env` values + mount-injected env vars. On collision, secrets override static `env` values and a warning is logged.
- **`.NamedVolumes`** — list of named volume identifiers that need top-level declaration
- **`.ExposedPorts`** — unchanged from current behavior

Note: `build.context` changes from `..` to `.` because the build context moves into `.deckhand/` (the workspace is a volume now, not a parent directory bind mount).

---

## Env var vs. Secret: When to Use Which

The existing `env` field in `.deckhand.yaml` is for **static values** that are part of the project config:

```yaml
env:
  GO_ENV: development
  DATABASE_URL: postgresql://dev:secret@postgres:5432/appdb
```

`mounts.secrets` is for **values from the host** that shouldn't be stored in the config file:

```yaml
mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}    # Resolved from host at runtime
      env: GH_TOKEN
```

Both end up in the compose `environment:` section. The difference is intent: `env` is checked into version control, secrets reference external state.

---

## Destroy and Named Volumes

With workspace data living in named volumes, `deckhand destroy` becomes a more consequential operation — it's deleting code, not just stopping containers.

Current behavior (Phase 1): stops containers, removes volumes and networks, deletes `.deckhand/`.

Updated behavior:
- `destroy` lists the named volumes it will remove and their sizes
- Requires `--yes` to skip confirmation (same as now, but the prompt should name the volumes explicitly)
- Volumes are discovered by Docker label (`dev.deckhand.project=<name>`), not by name-prefix matching — prefix matching is fragile (project `my` would match `my-api-workspace`)
- Named volumes are created with labels: `dev.deckhand.managed=true`, `dev.deckhand.project=<name>`, `dev.deckhand.volume=<mount-name>`

This is handled in Task 5 alongside the workspace volume change.

---

## Tasks

### Task 1: Mount domain types

Add the three mount primitive types and a grouping struct to the domain layer.

- `VolumeMount` — name, target, enabled
- `SecretMount` — name, source, target, env, readonly, enabled
- `SocketMount` — name, source, target, env, enabled
- `Mounts` — groups all three slices
- All three types support `enabled` (defaults to true, `false` removes an inherited mount)
- Add `Mounts` field to `TemplateMeta` and `Project`
- Validation: a secret must have at least one of `env` or `target` (otherwise it has nowhere to go)
- Unit tests for validation and any helper methods

### Task 2: Global config

- `GlobalConfig` domain type (defaults, ssh, mounts)
- Loader in `internal/config/` for `~/.config/deckhand/config.yaml`
- Handle missing file gracefully (return empty defaults)
- Path helper for global config location
- Tests: load valid file, missing file, partial file

### Task 3: Mount merging and env var resolution

- `MountMerger` in the service layer
- Merge order: template defaults → global → project
- Name-based dedup within each mount category
- `enabled: false` removes inherited mounts
- Env var resolution: expand `${VAR}` from `os.Getenv`
- Tilde expansion: `~` → home directory
- Skip unresolvable mounts (return warnings)
- Tests: merge scenarios, env resolution, tilde expansion, skip behavior

### Task 4: Render mounts in compose templates

- Update `templateData` to include resolved mount data (`.Volumes`, `.Environment`, `.NamedVolumes`)
- Update `TemplateService.Render()` to accept merged mounts
- Update all `compose.yaml.tmpl` files to use mount-based rendering
- Remove hardcoded `- ..:/workspace` from templates
- Tests: rendered compose has correct volumes, environment, top-level volumes section

### Task 5: Workspace as named volume

- Add default `workspace` volume to each template's `metadata.yaml`
- Update Dockerfile templates: change build context from `..` to `.deckhand/`
- Update `destroy` to list and remove project-prefixed named volumes
- Destroy confirmation prompt must name the volumes being deleted
- Verify `deckhand up` creates and uses named volume
- Tests: compose output uses named volume, not bind mount
- Tests: destroy removes named volumes

### Task 6: Wire global config into the up/init flow

- `deckhand up` loads global config and merges mounts before rendering
- `deckhand init` displays global config mount summary (so the user knows what's active) but does NOT prompt to change them — global mounts are managed in the global config file, not per-project
- `deckhand init` does NOT write mounts to `.deckhand.yaml` unless the user adds project-specific ones later
- `deckhand down` never touches volumes — only stops containers (confirm existing behavior holds)
- CLI passes global config path (respect `--config` flag for project, separate for global)
- Integration tests: full flow with global + project + template mounts

---

## Credential Recipes

These are not app features — they're example configs users can set in their global config.

**GitHub via HTTPS (GH_TOKEN):**
```yaml
mounts:
  secrets:
    - name: gh-token
      source: ${GH_TOKEN}
      env: GH_TOKEN
```

**GitHub via SSH key:**
```yaml
mounts:
  secrets:
    - name: ssh-key
      source: ~/.ssh/id_ed25519
      target: /home/dev/.ssh/id_ed25519
      readonly: true
```

**SSH agent forwarding:**
```yaml
mounts:
  sockets:
    - name: ssh-agent
      source: ${SSH_AUTH_SOCK}
      target: /run/ssh-agent.sock
      env: SSH_AUTH_SOCK
```

**Git config:**
```yaml
mounts:
  secrets:
    - name: gitconfig
      source: ~/.gitconfig
      target: /home/dev/.gitconfig
      readonly: true
```

**GPG agent:**
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
