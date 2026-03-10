# Deckhand: Implementation Phases

Each phase produces a usable tool. We don't build everything and ship at the end — each phase adds real functionality you can use.

---

## Phase 1: Skeleton & Core Lifecycle

**Goal:** You can `deckhand up` a hardcoded dev container, `deckhand shell` into it, and `deckhand down` to stop it. The tool exists, builds, and does something real.

- [ ] Initialize Go module, project structure (`cmd/`, `internal/`, `templates/`)
- [ ] Set up root Cobra command with `--version`, `--help`, `--verbose` flags
- [ ] Define domain types: Project, Container, PortMapping
- [ ] Implement config loading — read `.deckhand.yaml` from current directory
- [ ] Create a single `base` template (Dockerfile.tmpl + compose.yml.tmpl)
- [ ] Implement `deckhand init` — create `.deckhand.yaml` with defaults (no prompts yet, just flags)
- [ ] Implement `deckhand up` — render templates to `.deckhand/`, run `docker compose up -d`
- [ ] Implement `deckhand down` — run `docker compose down`
- [ ] Implement `deckhand destroy` — `docker compose down -v`, remove `.deckhand/`
- [ ] Implement `deckhand shell` — exec into devcontainer with full TTY support
- [ ] Implement `deckhand exec` — run a one-off command in devcontainer
- [ ] Implement `deckhand logs` — stream container logs with `--follow` and `--tail`

---

## Phase 2: Status, Listing & Port Management

**Goal:** You can see what's running, manage ports, and generate the SSH connect command.

- [ ] Add container labels (`dev.deckhand.managed`, `dev.deckhand.project`, etc.)
- [ ] Implement `deckhand status` — show services, health, ports for current project
- [ ] Implement `deckhand list` — show all deckhand-managed environments on the host
- [ ] Implement `deckhand port list` — show all port mappings from labels
- [ ] Implement `deckhand port add <port>` — update compose file, recreate container
- [ ] Implement `deckhand port remove <port>` — update compose file, recreate container
- [ ] Implement `deckhand connect` — generate SSH tunnel command from mapped ports

---

## Phase 3: Templates & Interactive Init

**Goal:** Multiple bundled templates, interactive project setup with prompts.

- [ ] Create `go` template (Go toolchain, gopls, delve)
- [ ] Create `node` template (Node.js, npm, typescript-language-server)
- [ ] Create `python` template (Python, pip, pyright)
- [ ] Implement template rendering with Go's `text/template` (variable substitution)
- [ ] Implement `deckhand template list` — show available bundled templates
- [ ] Add optional services to templates (postgres, redis) with health checks
- [ ] Make `deckhand init` interactive using huh forms (template picker, services, project name)
- [ ] Support `--template` flag to skip interactive prompt

---

## Phase 4: Configuration & Credential Injection

**Goal:** Global config, proper config precedence, and credentials work inside containers.

- [ ] Implement global config at `~/.config/deckhand/config.yaml`
- [ ] Implement config precedence: flags → env vars → project config → global config
- [ ] SSH agent forwarding — mount `SSH_AUTH_SOCK` into containers
- [ ] Git config injection — mount `.gitconfig` read-only
- [ ] GitHub token injection — mount as secret file, set `GH_TOKEN`
- [ ] GPG agent forwarding — mount extra socket

---

## Phase 5: Polish & Distribution

**Goal:** The tool is installable, robust, and pleasant to use.

- [ ] Add shell completions (bash, zsh, fish) via Cobra
- [ ] Error handling polish — helpful messages, not stack traces
- [ ] Set up GoReleaser for cross-platform builds
- [ ] GitHub Actions CI — build, test, lint on push
- [ ] Release to GitHub Releases
- [ ] Write README with install instructions and quick start

---

## Future (not planned yet)

- TUI dashboard (`deckhand tui`) via Bubbletea
- External templates from git repositories
- Template inheritance (language templates extend base)
- Self-update command
- `deckhand connect` SSH host auto-detection via Tailscale
