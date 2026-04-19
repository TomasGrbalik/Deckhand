# Template metadata: `command` and `exec_user` overrides

**Status:** Design approved
**Date:** 2026-04-19
**Scope:** Deckhand feature + motivating `general` template in a new `deckhand-templates` repo

## Motivation

Deckhand's shared `compose.yaml.tmpl` hardcodes `command: sleep infinity` on the devcontainer service. Templates that need a long-running daemon (sshd, a supervisor, a language server) currently must ship a full `compose.yaml.tmpl` override — duplicating ~90 lines of boilerplate to change one line.

Related gap: `internal/infra/docker/container.go` `Exec` never sets a user, so `deckhand shell` / `exec` always use the image's `USER` directive. A template that needs `USER root` (to start sshd on :22) has no way to say "but drop shell/exec sessions into `dev`."

Together these two gaps block the first-class use case of this spec's consumer: a **general-purpose Ubuntu dev container with an SSH server running so you can `ssh dev@<container-ip>` directly via Tailscale** (per `docs/networking.md`).

## Goals

1. Let a template declare the devcontainer's long-running command declaratively, without replacing the compose template.
2. Let a template declare which user `deckhand shell` and `deckhand exec` should drop into, independently of the Dockerfile's `USER`.
3. Ship a `general` template (in a new `deckhand-templates` private repo) that exercises both.
4. Preserve byte-for-byte backwards compatibility for `base` and `python`.

## Non-goals

- No support for multiple devcontainer services.
- No runtime-reconfigurable `command` (rebuild required to change).
- No `user:` for companions — only the devcontainer.
- No `.NET` template in this round — deferred.

## Design

### Metadata schema additions

Two new optional top-level fields in `metadata.yaml`:

```yaml
command: <string>      # Optional. Devcontainer command. Defaults to "sleep infinity" when absent.
exec_user: <string>    # Optional. User for `deckhand shell`/`exec`. Defaults to "" (image default user).
```

Both fields are strings; absent is equivalent to empty. The service layer applies the `sleep infinity` default so `templateData.Command` is guaranteed non-empty at template-render time — the compose template never sees the default.

### Code changes — deckhand repo

Branch: `feat/metadata-command-and-exec-user`. One commit, conventional-commits format:
`feat: support command and exec_user overrides in template metadata`.

1. **`internal/domain/`** — metadata struct gains `Command string` and `ExecUser string`. `templateData` (compose render input) gains `Command string`.
2. **`internal/infra/template/`** — YAML parser reads both new fields. Absent = `""`.
3. **`internal/service/`** —
   - When building `templateData`, set `Command = metadata.Command; if Command == "" { Command = "sleep infinity" }`.
   - Expose `ExecUser` on whatever service-layer surface the CLI uses for `shell`/`exec`.
4. **`internal/infra/docker/container.go`** — extend `Exec` to accept a `user string` parameter. When non-empty, set `User:` on `ExecCreateOptions`. Update all callers.
5. **`internal/cli/`** — `shell` and `exec` subcommands look up the project's template metadata and pass `ExecUser` down to the infra call.
6. **`templates/compose.yaml.tmpl:37`** — replace `command: sleep infinity` with `command: {{ .Command }}`.
7. **Tests:**
   - Template parser: `command`/`exec_user` parsed when present, empty string when absent.
   - Service render: default `"sleep infinity"` substituted when metadata `command` is empty; passthrough when set.
   - Compose render: both cases produce valid YAML with the expected `command:` value.
   - Exec infra: user flag honored when non-empty; omitted from `ExecCreateOptions` when empty.
   - Snapshot tests for `base` and `python` rendered compose output must be unchanged.
8. **Docs:** update `docs/custom-templates.md` metadata schema table with both fields; add a short "long-running daemon" example pointing at `general`.

### Template repo — `deckhand-templates`

Separate private GitHub repo (user: `TomasGrbalik`). Layout:

```
deckhand-templates/
├── README.md          # Purpose + install (`git clone <repo> ~/.config/deckhand/templates`)
├── .gitignore
└── general/
    ├── metadata.yaml
    └── Dockerfile.tmpl
```

Install on server: `git clone git@github.com:TomasGrbalik/deckhand-templates ~/.config/deckhand/templates`. Each top-level directory becomes a discoverable template name. Initial commit on `main` is README + `.gitignore`; `general/` lands via branch `feat/general-template` → PR.

### `general/metadata.yaml`

```yaml
name: general
description: Ubuntu dev container with SSH access, claude-code, and common tools
command: "/usr/sbin/sshd -D -e"
exec_user: dev
variables:
  github_user:
    default: "TomasGrbalik"
    description: GitHub username whose public keys authorize SSH (ignored if authorized_keys is set)
  authorized_keys:
    default: ""
    description: Explicit authorized_keys content (newline-separated). Overrides github_user when non-empty.
mounts:
  volumes:
    - name: workspace
      target: /workspace
```

### `general/Dockerfile.tmpl`

- Base: `ubuntu:24.04`.
- Single apt install: `git curl wget ripgrep build-essential zsh ca-certificates openssh-server iputils-ping jq unzip less vim htop tmux rsync sudo`, then `rm -rf /var/lib/apt/lists/*`.
- `yq`: download architecture-correct binary from `github.com/mikefarah/yq/releases/latest/download/yq_linux_$(dpkg --print-architecture)` to `/usr/local/bin/yq`.
- User: `groupadd -f -g 1000 dev && useradd -m -u 1000 -g dev -o -s /bin/zsh dev`; grant passwordless sudo via `/etc/sudoers.d/dev`.
- claude-code: install as `dev` via `su - dev -c "curl -fsSL https://claude.ai/install.sh | bash"`.
- SSH:
  - `mkdir -p /home/dev/.ssh && chmod 700 /home/dev/.ssh && chown dev:dev /home/dev/.ssh`
  - Go-template conditional on `.Vars.authorized_keys`:
    - If non-empty: `echo '{{ .Vars.authorized_keys }}' > /home/dev/.ssh/authorized_keys`
    - Else: `curl -fsSL https://github.com/{{ .Vars.github_user }}.keys > /home/dev/.ssh/authorized_keys`
  - `chmod 600 /home/dev/.ssh/authorized_keys && chown dev:dev /home/dev/.ssh/authorized_keys`
  - Write `/etc/ssh/sshd_config.d/deckhand.conf` with: `PasswordAuthentication no`, `PermitRootLogin no`, `AllowUsers dev`.
  - `ssh-keygen -A` at build time — host keys baked into the image; stable across container recreations. Rotate by rebuilding (expected, infrequent).
  - `mkdir -p /run/sshd` (sshd refuses to start without it).
- `WORKDIR /workspace`.
- **No final `USER` directive** — container runs as root so sshd can bind :22 and read host keys; `exec_user: dev` in metadata makes `deckhand shell`/`exec` land as dev.

### Deliberate deviation from `docs/custom-templates.md`

The existing guide mandates "end with `USER dev`." That convention is incompatible with a template whose primary purpose is running a daemon that requires root (sshd). Once this spec lands, the docs should note that templates which declare `exec_user` may omit the `USER` directive. This doc change is part of the deckhand PR.

## Testing plan

Local end-to-end on this dev machine (not a real Tailscale server):

1. `go build ./... && go test ./...` on the feature branch — all pass.
2. `go install ./cmd/deckhand` to put the feature-branch binary on `$PATH`.
3. Install the `general` template to `~/.config/deckhand/templates/general/`.
4. Scratch project with `.deckhand.yaml` → `template: general`, `variables: { github_user: TomasGrbalik }`.
5. `deckhand up` — build succeeds, container starts.
6. `deckhand exec -- ss -tlnp` — confirms sshd listening on :22.
7. `deckhand shell` — lands as `dev`.
8. Inside shell: `claude --version`, `jq --version`, `ping -c1 8.8.8.8`, `sudo -n true` all succeed.
9. Verify host keys are baked: `deckhand exec -- ls /etc/ssh/ssh_host_*_key` prints files.
10. Verify authorized_keys populated: `deckhand exec -- cat /home/dev/.ssh/authorized_keys` shows GitHub-fetched keys.

Real SSH (`ssh dev@<container-ip>`) is validated post-merge on the actual Tailscale-routed server.

## Rollout

1. Deckhand PR merges to `main`.
2. Tag + release (or user rebuilds from source on the target server).
3. `deckhand-templates` repo bootstrapped; `general/` PR merges.
4. Devbox: update deckhand binary, `git clone deckhand-templates ~/.config/deckhand/templates`.

## Open questions

None. All decisions settled:
- Sshd runs by default via `command:` override (not a companion).
- Authorized keys: GitHub fetch default, explicit-keys variable override.
- claude-code: standalone installer (no Node in the image).
- `exec_user` ships with `command` in one PR.
