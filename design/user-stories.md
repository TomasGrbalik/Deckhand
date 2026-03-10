# Deckhand: User Stories

## Who is the user?

A developer who runs Docker workloads on a remote Linux server and connects from a thin client (iPad, laptop) over SSH/Tailscale. They want a single CLI tool to manage the full lifecycle of containerized dev environments without manually writing docker-compose files, remembering port mappings, or wiring up credentials.

---

## Environment Lifecycle

- As a user, I want to initialize a new project from a template so that I get a working dev container with the right language tools, editor, and services without writing Dockerfiles from scratch.
- As a user, I want to start my environment (`deckhand up`) so that all containers (devcontainer + databases + services) come up in the correct order with health checks.
- As a user, I want to stop my environment (`deckhand down`) so that containers stop cleanly without losing volume data.
- As a user, I want to destroy an environment (`deckhand destroy`) so that containers, networks, and volumes are fully removed for a clean slate.
- As a user, I want to see the status of all my environments (`deckhand list`) so that I know what's running across projects.

## Working Inside Containers

- As a user, I want to open a shell in my dev container (`deckhand shell`) so that I can work interactively with full TTY support (Neovim, tmux, etc.).
- As a user, I want to run a one-off command inside a container (`deckhand exec <cmd>`) so that I can run tests or scripts without opening a shell first.
- As a user, I want to view container logs (`deckhand logs`) so that I can debug services without switching to raw docker commands.

## Port Management

- As a user, I want to see all mapped ports for my project (`deckhand port list`) so that I know what's accessible and how.
- As a user, I want to add a port mapping at runtime (`deckhand port add 4000`) so that I can expose a new service without manually editing compose files.
- As a user, I want to remove a port mapping (`deckhand port remove 4000`) so that I can clean up unused ports.
- As a user, I want to generate the SSH tunnel command for all my ports (`deckhand connect`) so that I can copy-paste it into my terminal client and access services from my local browser.

## Templates

- As a user, I want to browse available templates (`deckhand template list`) so that I can see what pre-built environments exist.
- As a user, I want to create a project from a template (`deckhand init --template go`) so that I get a language-appropriate Dockerfile, compose file, and port mappings.
- As a user, I want templates to support inheritance (e.g. `go` extends `base`) so that common setup isn't duplicated.

## Credential Injection

- As a user, I want my SSH agent forwarded into containers so that I can use git over SSH without copying keys.
- As a user, I want my GitHub token available inside containers so that `gh` CLI and private repo access work transparently.
- As a user, I want my GPG agent forwarded so that I can sign commits from inside containers.
- As a user, I want my git config (name, email) carried into containers so that commits have correct attribution.

## Configuration

- As a user, I want a project-level config file (`.deckhand.yaml`) so that my environment is reproducible and version-controllable.
- As a user, I want a global config (`~/.config/deckhand/config.yaml`) so that my SSH host, default template, and preferences persist across projects.
- As a user, I want environment variables and flags to override config so that I can customize behavior for one-off runs.
