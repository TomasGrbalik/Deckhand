# The complete guide to containerized remote development over SSH

**Every development dependency lives inside Docker containers; the host runs nothing but Docker, SSH, and optionally Tailscale.** You connect from an M2 iPad Pro (or any thin client) via Mosh/SSH into tmux, then work inside purpose-built dev containers using Neovim for terminal editing or code-server for a browser-based VS Code experience. This architecture gives you reproducible, isolated, easily reconstructable environments across any project — while your "local" device needs nothing more than a good terminal app and a browser.

This guide covers the full stack: container architecture, Neovim 0.11 configuration, code-server setup, iPad workflows, SSH/networking optimization, and server provisioning. All information reflects the state of tooling as of early 2026.

---

## 1. Containerized dev environment architecture

### The dual-editor container: Neovim and code-server in one image

The core idea is a single container image that bundles **both Neovim** (accessed via terminal/SSH) **and code-server** (accessed via browser), sharing the same filesystem, language servers, linters, and formatters. You switch editors without switching environments.

The container runs with `sleep infinity` or a lightweight init, and you interact with it two ways: `docker compose exec devcontainer nvim` for terminal editing, or navigate to `http://localhost:8080` for the browser IDE. Both editors see identical project files, installed tools, and environment variables.

**Base image selection matters.** For dev containers with mixed language stacks (Node, Python, Go, Rust), **Ubuntu 24.04 is the recommended base** — its glibc compatibility is critical because many VS Code extensions, LSP servers, and npm native modules depend on it. Alpine's musl libc causes Python builds to run up to 50x slower and breaks numerous VS Code extensions. Microsoft's `mcr.microsoft.com/devcontainers/base:ubuntu` provides a preconfigured non-root `vscode` user and common tools, though a custom Ubuntu base gives maximum control. The community image `ghcr.io/containercraft/devcontainer` ships with both Neovim (LazyVim config) and code-server preinstalled — useful as a reference or direct base.

Here is a production-ready Dockerfile for a dual-editor polyglot container:

```dockerfile
FROM ubuntu:24.04 AS base
ARG USERNAME=dev
ARG USER_UID=1000
ARG USER_GID=1000
ARG CODE_SERVER_VERSION=4.96.4

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd -s /bin/zsh --uid $USER_UID --gid $USER_GID -m $USERNAME

RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget unzip ripgrep fd-find tree jq \
    build-essential cmake python3 python3-pip python3-venv \
    zsh tmux openssh-client gnupg2 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Neovim (latest stable from GitHub releases)
RUN curl -fsSL "https://github.com/neovim/neovim/releases/download/v0.11.0/nvim-linux-x86_64.tar.gz" \
    | tar -xz -C /opt \
    && ln -s /opt/nvim-linux-x86_64/bin/nvim /usr/local/bin/nvim

# code-server (VS Code in browser)
RUN curl -fsSL https://code-server.dev/install.sh | sh

# Node.js 22
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs

# Go
RUN curl -fsSL https://go.dev/dl/go1.23.4.linux-amd64.tar.gz | tar -xz -C /usr/local
ENV PATH="/usr/local/go/bin:/home/${USERNAME}/go/bin:${PATH}"

# Rust (as non-root)
USER $USERNAME
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/home/${USERNAME}/.cargo/bin:${PATH}"

WORKDIR /workspace
EXPOSE 8080 3000
```

### Docker Compose patterns for multi-service stacks

The dev container serves as your primary workspace alongside databases and other services. Use `docker compose exec` to enter it, and run code-server as a background process inside:

```yaml
services:
  devcontainer:
    build:
      context: .
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - .:/workspace
      - node-modules:/workspace/node_modules
      - go-cache:/home/dev/go
      - cargo-registry:/home/dev/.cargo/registry
      - ${HOME}/.config/nvim:/home/dev/.config/nvim
      - ${HOME}/.gitconfig:/home/dev/.gitconfig:ro
    ports:
      - "127.0.0.1:3000:3000"
      - "127.0.0.1:8080:8080"
    environment:
      - DATABASE_URL=postgresql://dev:secret@postgres:5432/appdb
      - REDIS_URL=redis://redis:6379
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }
    command: sleep infinity
    networks: [dev-network]

  postgres:
    image: postgres:16-alpine
    volumes:
      - postgres-data:/var/lib/postgresql/data
    environment:
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: appdb
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dev"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks: [dev-network]

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes: [redis-data:/data]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
    networks: [dev-network]

volumes:
  postgres-data:
  redis-data:
  node-modules:
  go-cache:
  cargo-registry:

networks:
  dev-network:
    driver: bridge
```

**Named volumes for dependency directories are essential.** Place `node_modules`, Go module caches, pip caches, and Cargo registries on named volumes rather than bind mounts. On Linux hosts, the performance difference is negligible (no VM boundary), but named volumes provide clean isolation — `docker compose down -v` wipes dependencies without touching source code, and `docker compose up` reinstalls cleanly. On macOS/Windows, named volumes avoid the 3.5x performance penalty of cross-VM bind mounts for `npm install`.

Use **Docker Compose profiles** for optional services like database admin tools or monitoring:

```yaml
  adminer:
    image: adminer:latest
    ports: ["127.0.0.1:8081:8080"]
    profiles: ["debug"]
```

Start with `docker compose --profile debug up -d` when needed. Docker Compose's **watch mode** (`docker compose watch`) can automatically sync file changes or trigger rebuilds — define this in a `develop.watch` block for hot-reload workflows.

### Multi-project isolation and Makefile patterns

Each project gets its own `.devcontainer/` directory with Dockerfile, docker-compose.yml, and a `.env` file containing `COMPOSE_PROJECT_NAME=project-alpha`. This env var namespaces all containers, networks, and volumes, ensuring complete isolation between projects.

A project Makefile eliminates repetitive commands:

```makefile
COMPOSE = docker compose
SERVICE = devcontainer

up:        ## Start all services
	$(COMPOSE) up -d
down:      ## Stop all services
	$(COMPOSE) down
destroy:   ## Stop + remove all volumes (fresh start)
	$(COMPOSE) down -v --remove-orphans
shell:     ## Open shell in dev container
	$(COMPOSE) exec $(SERVICE) zsh
nvim:      ## Open Neovim in dev container
	$(COMPOSE) exec $(SERVICE) nvim
code:      ## Start code-server
	$(COMPOSE) exec -d $(SERVICE) code-server --bind-addr 0.0.0.0:8080 /workspace
	@echo "code-server at http://localhost:8080"
rebuild:   ## Rebuild without cache
	$(COMPOSE) build --no-cache
logs:      ## Follow logs
	$(COMPOSE) logs -f
```

Run `make destroy && make rebuild && make up` for a completely fresh environment in under a minute.

### Dotfiles injection strategies

Three approaches, each with tradeoffs. **Bind mounts** give real-time sync — changes to your Neovim config on the host appear immediately in the container. Add mounts in your docker-compose.yml for `~/.config/nvim`, `~/.gitconfig`, `~/.tmux.conf`, and `~/.zshrc`. For portability across machines, use the **devcontainer.json dotfiles repository** feature, which clones a git repo and runs an install script at container creation:

```jsonc
{
  "dotfiles": {
    "repository": "https://github.com/your-username/dotfiles",
    "installCommand": "install.sh"
  }
}
```

For git operations inside containers, forward your SSH agent by mounting the socket: `-v ${SSH_AUTH_SOCK}:/ssh-agent` with `SSH_AUTH_SOCK=/ssh-agent` as an environment variable.

---

## 2. Neovim 0.11 and the 2025 plugin ecosystem

### Native LSP changes everything

Neovim 0.11, released in **March 2025**, introduced `vim.lsp.config()` and `vim.lsp.enable()` — two APIs that make the `nvim-lspconfig` plugin optional for basic setups. You can now configure LSP servers with zero plugins by creating files under `~/.config/nvim/lsp/`:

```lua
-- ~/.config/nvim/lsp/lua_ls.lua
return {
  cmd = { 'lua-language-server' },
  filetypes = { 'lua' },
  root_markers = { '.luarc.json', '.git' },
  settings = {
    Lua = {
      runtime = { version = 'LuaJIT' },
      diagnostics = { globals = { 'vim' } },
    },
  },
}
```

Then in your init.lua: `vim.lsp.enable({ 'lua_ls', 'ts_ls', 'pyright', 'rust_analyzer', 'gopls' })`. Neovim 0.11 also ships **default LSP keybindings** that activate automatically when a server attaches: `grn` for rename, `gra` for code action, `grr` for references, `gri` for implementations, and `K` for hover documentation (now with tree-sitter-highlighted markdown).

**mason.nvim remains the standard for installing LSP servers**, but its role shifted. In containers, you have two strategies: install mason and let it download servers on first launch (requires internet), or skip mason entirely by installing servers via the Dockerfile's package manager (`npm install -g typescript-language-server`, `pip install pyright`, etc.) and pointing `vim.lsp.config` at them directly. The second approach produces fully offline-capable containers.

Other 0.11 highlights include **built-in commenting** (`gc`/`gb` operators, making Comment.nvim unnecessary), a native `vim.snippet` API, async tree-sitter highlighting for large files, and an auto dark-mode feature that responds to terminal theme changes.

### The essential plugin stack for 2025

**lazy.nvim** remains the universal plugin manager, with its lockfile support, automatic bytecode compilation, and lazy-loading capabilities. The core plugin stack for a containerized remote setup:

**blink.cmp** has overtaken nvim-cmp as the recommended completion engine. It uses a Rust-based fuzzy matcher, ships LSP/buffer/path/snippet sources built-in, and integrates seamlessly with Neovim 0.11's native LSP:

```lua
return {
  'saghen/blink.cmp',
  version = '1.*',
  event = { 'InsertEnter', 'CmdlineEnter' },
  opts = {
    keymap = { preset = 'default' },
    completion = { documentation = { auto_show = true } },
    sources = { default = { 'lsp', 'path', 'snippets', 'buffer' } },
    fuzzy = { implementation = 'prefer_rust_with_warning' },
  },
}
```

For fuzzy finding, the ecosystem split three ways. **fzf-lua** became LazyVim's default in v14 (late 2024), prized for speed via the native fzf binary. **telescope.nvim** retains ~18.5k GitHub stars and the richest extension ecosystem. **snacks.picker** is the newest entrant from Folke (LazyVim author). All three are viable — fzf-lua for raw speed, Telescope for maximum extensibility.

The rest of the essential stack: **oil.nvim** for filesystem editing as a buffer, **conform.nvim** for formatting (configure `format_on_save` with per-filetype formatters), **nvim-lint** for async linting triggered on `BufWritePost` and `InsertLeave`, **treesitter** for syntax highlighting and indentation, **gitsigns.nvim** for git gutter integration, **which-key.nvim** for keybinding discovery, and **lualine.nvim** for a statusline.

### tmux configuration tuned for remote Neovim

The single most important tmux setting for Neovim users is **`set -sg escape-time 10`**. The default 500ms delay causes a visible lag when pressing Escape to exit insert mode, because tmux buffers the escape sequence. Setting it to 10ms (or 0) eliminates this entirely — Neovim's `:checkhealth` will warn if this value exceeds 300ms.

A complete remote-optimized tmux.conf:

```bash
set -g prefix C-a
unbind C-b
bind C-a send-prefix

# Critical for Neovim
set -sg escape-time 10
set -g focus-events on

# True color support
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",xterm-256color:RGB"

# OSC 52 clipboard (critical for remote)
set -s set-clipboard on

# Mouse support
set -g mouse on

# Vi copy mode
setw -g mode-keys vi
bind -T copy-mode-vi v send-keys -X begin-selection
bind -T copy-mode-vi y send-keys -X copy-pipe

# Smart splits with current path
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"
bind c new-window -c "#{pane_current_path}"

# 1-indexed (easier to reach)
set -g base-index 1
setw -g pane-base-index 1
```

**tmuxinator** provides YAML-based session templates for repeatable project layouts. A typical config launches an editor window, a server window, and a logs window with predefined commands, plus hooks like `on_project_start: docker compose up -d`. Install with `gem install tmuxinator` and store configs at `~/.config/tmuxinator/`. Use **tmux-resurrect** plus **tmux-continuum** plugins for automatic session persistence across server reboots.

### OSC 52 clipboard: the full chain from Neovim to iPad

**OSC 52 is the mechanism that makes remote clipboard work without X11 forwarding.** When you yank text in Neovim, it emits an OSC 52 escape sequence. tmux passes this through the SSH connection to your local terminal emulator, which writes to the system clipboard. The chain: Neovim → tmux → SSH → local terminal → system clipboard.

Neovim 0.10+ includes a built-in OSC 52 provider:

```lua
vim.g.clipboard = {
  name = 'OSC 52',
  copy = {
    ['+'] = require('vim.ui.clipboard.osc52').copy('+'),
    ['*'] = require('vim.ui.clipboard.osc52').copy('*'),
  },
  paste = {
    ['+'] = require('vim.ui.clipboard.osc52').paste('+'),
    ['*'] = require('vim.ui.clipboard.osc52').paste('*'),
  },
}
vim.opt.clipboard = 'unnamedplus'
```

tmux needs `set -s set-clipboard on` in tmux.conf (do **not** use `set-clipboard external`, which causes issues). On the client side, **Blink Shell, iTerm2, Alacritty, Kitty, WezTerm, and Ghostty** all support OSC 52 — macOS Terminal.app and GNOME Terminal do not.

---

## 3. code-server and openvscode-server inside containers

### Two approaches to VS Code in a browser

**code-server** (by Coder, v4.109.5 as of March 2026) wraps VS Code via git submodule plus patch files, adding its own CLI, password authentication, built-in proxying, and PWA support. It's designed for individual self-hosting and has **~70k GitHub stars**. Default port is 8080.

**openvscode-server** (by Gitpod, v1.106.3 as of December 2025) is a direct fork with minimal changes — it adds only the server code needed to run VS Code in a browser. It tracks upstream VS Code versions directly (v1.106.3 *is* VS Code 1.106.3) and is the same architecture used by Gitpod and GitHub Codespaces at scale. Default port is 3000.

Both use the **Open VSX Registry** for extensions, not the Microsoft Marketplace. Open VSX has ~5,000 extensions compared to Microsoft's ~48,000, though the most popular development extensions are available. For extensions only on the Microsoft Marketplace, download `.vsix` files from GitHub releases and install manually with `code-server --install-extension /path/to/file.vsix`.

For a self-hosted single-developer setup, **code-server is the pragmatic choice** — its built-in password auth, config file, and proxy features reduce setup friction. Pre-install extensions in the Dockerfile:

```dockerfile
FROM codercom/code-server:latest
USER root
RUN apt-get update && apt-get install -y nodejs npm python3 python3-pip git \
    && rm -rf /var/lib/apt/lists/*
RUN code-server --install-extension ms-python.python \
    && code-server --install-extension dbaeumer.vscode-eslint
USER coder
WORKDIR /home/coder/project
EXPOSE 8080
```

### Persistence and secure access

Mount these paths as volumes for persistence: `~/.local/share/code-server` (extensions, settings, keybindings) and `~/.config/code-server` (config.yaml including auth settings). Project files come in via bind mount.

**The recommended access method is SSH tunneling** — it provides encryption and authentication with zero additional configuration:

```bash
ssh -N -L 8080:127.0.0.1:8080 user@remote-server
# Then open http://localhost:8080 — set code-server to --auth none
```

For iPad access where SSH tunneling is less convenient, **Tailscale provides the best balance of security and usability**. Install Tailscale on both the server and iPad, then access code-server via the Tailscale IP or use `tailscale serve --bg 8080` for automatic HTTPS. Caddy can also reverse-proxy code-server with automatic Let's Encrypt certificates for a custom domain. Critically, the nginx/Caddy config must include WebSocket upgrade headers (`proxy_set_header Upgrade $http_upgrade`).

---

## 4. The M2 iPad Pro as a development thin client

### Blink Shell is the clear winner for terminal-based development

**Blink Shell** ($19.99/year) is the consensus best SSH/Mosh client for iPad development work. Its native Mosh implementation survives network changes (Wi-Fi to cellular), device sleep, and even iPad reboots without dropping the session. This single feature makes mobile development viable — you open your iPad lid and your tmux session is exactly where you left it, with zero reconnection delay.

Key capabilities include full SSH key management (Ed25519, Secure Enclave keys, SSH certificates), a built-in `code` command that opens code-server in an integrated browser tab (called **Blink Code**), SFTP integration with the Files app, SSH agent forwarding, port forwarding (local/remote/dynamic), and what the community calls "legendary" external keyboard support with full shortcut remapping. Blink also supports OSC 52, completing the clipboard chain from remote Neovim to the iPad's system clipboard.

**Termius** (free basic, ~$15/month premium) offers cross-platform sync across every OS, an encrypted credential vault, FIDO2 security keys, and AI-assisted command suggestions. It's stronger for team use and multi-platform workflows but lacks Blink's code-server integration. **Prompt 3** by Panic ($24.99 one-time) supports SSH, Mosh, and Eternal Terminal with a polished Apple-ecosystem design and Panic Sync across devices. **a-Shell** is free and provides a local terminal with Python, JavaScript, and C compilation via WebAssembly, but it's not a serious SSH client.

### Browser-based editing via Safari

When using code-server through Safari, **install it as a PWA** (Share → Add to Home Screen) for a near-native experience. PWA mode gives fullscreen real estate and prevents Safari from intercepting keyboard shortcuts like Cmd+W (which closes files in VS Code but closes the Safari tab otherwise). However, iPadOS Safari has real limitations: Ctrl+C may not reliably stop terminal processes (workaround: add a custom keybinding that sends `\u0003`), background tabs get aggressively suspended under memory pressure, and all iPad browsers use WebKit under the hood — Chrome and Edge offer zero advantage over Safari for web apps.

Blink Code is generally preferred over Safari for code-server because it handles keyboard shortcuts natively, provides edge-to-edge display, and integrates SSH tunnel management.

### Keyboard, trackpad, and Stage Manager workflows

The Magic Keyboard for iPad Pro works well for coding. The critical configuration for Vim users: **Settings → General → Keyboard → Hardware Keyboard → Modifier Keys → set Caps Lock to Escape**. This works system-wide across all apps. Additional modifier keys (Control, Option, Command, Globe) are individually remappable. Blink Shell adds its own remapping layer on top.

**Stage Manager on M2 iPad Pro** (iPadOS 18+) enables practical multi-window development. The recommended layout: Blink Shell and Safari (with code-server or documentation) side by side as your primary stage, with additional stages for Slack, browser reference, or other tools. With an external display, you get up to 8 simultaneous app windows across two screens. iPadOS 26 (expected fall 2025) brings macOS-inspired windowing with traffic-light controls and saved workspace layouts.

### Working around iPadOS constraints

The fundamental limitation is **no local Docker** — all computation happens remotely. This is actually the architecture's strength: the iPad stays cool, battery lasts 10+ hours for terminal work, and your expensive compute runs on a server with fast internet. **Mosh + tmux is the critical technology stack** that makes this work. iPadOS aggressively suspends background apps, killing SSH connections within seconds to minutes. Mosh's UDP-based protocol survives this suspension transparently, and tmux preserves your session server-side even if Mosh somehow disconnects. Together, they completely neutralize iPadOS's background-killing behavior.

---

## 5. SSH optimization and networking

### An SSH config built for remote development

```ssh-config
Host *
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
    ControlPersist 600
    ServerAliveInterval 60
    ServerAliveCountMax 3
    AddKeysToAgent yes
    IdentitiesOnly yes
    Compression no

Host dev-server
    HostName your-server.example.com
    User developer
    IdentityFile ~/.ssh/id_ed25519
    LocalForward 8080 localhost:8080
```

**ControlMaster** multiplexes multiple SSH sessions over one TCP connection — the second `ssh dev-server` connects instantly because it reuses the existing connection. `ControlPersist 600` keeps the master alive for 10 minutes after the last session closes. **ServerAliveInterval 60** sends keepalive packets every 60 seconds to prevent NAT timeouts from killing idle connections. Leave `Compression no` for fast networks; enable only on high-latency links.

Use **Ed25519 keys exclusively** — they provide equivalent security to RSA-3072+ with dramatically shorter keys (~68 characters vs ~3072), faster operations, and no dependence on PRNG quality. Generate with `ssh-keygen -t ed25519 -C "user@purpose-2026-03"`.

### Tailscale replaces most networking complexity

Tailscale creates a **peer-to-peer WireGuard mesh VPN** where every device gets a stable `100.x.y.z` IP address. Install it on your server and iPad, and they can reach each other directly without port forwarding, dynamic DNS, or firewall rules. MagicDNS gives every device a human-readable hostname (`dev-server.tailnet-name.ts.net`).

**Tailscale SSH** goes further — it replaces traditional SSH key management entirely. Enable it with `sudo tailscale set --ssh` on the server. Authentication uses WireGuard node keys tied to your Tailscale identity, eliminating the need to generate, distribute, or rotate SSH keys. Access is controlled via JSON ACL policies in the admin console.

**Tailscale Funnel** temporarily exposes a local service to the public internet over HTTPS — ideal for webhook testing during development. Run `tailscale funnel 3000` and your dev server becomes reachable at `https://your-machine.tailnet-name.ts.net` with automatic Let's Encrypt certificates, no firewall changes required. The free Personal plan supports **3 users and 100 devices** with nearly all features.

On iPad, Tailscale has a full-featured iOS app. Combined with code-server, it provides secure browser-based access to your dev environment from anywhere without SSH tunnels.

### Mosh alongside SSH tunnels

Mosh provides roaming, local echo, and UDP-based resilience, but it **cannot do port forwarding**. The practical pattern: run a persistent SSH tunnel for port forwards (code-server on 8080, database on 5432) via autossh, and use Mosh for your interactive terminal session:

```bash
# Persistent tunnel (systemd service or background process)
autossh -M 0 -N -o "ServerAliveInterval 30" -o "ServerAliveCountMax 3" \
    -L 8080:localhost:8080 user@remote-server &

# Interactive session via Mosh
mosh user@remote-server
```

Mosh 1.4.0 (the latest release, from October 2022) added true color support and OSC 52 clipboard integration. Development is slow but the protocol is stable and has had zero security vulnerabilities in over a decade.

---

## 6. Minimal server provisioning

### The host runs exactly six components

A properly minimal Docker dev server has: **Ubuntu Server 24.04 LTS**, **Docker Engine + Compose plugin** (from Docker's official apt repo, not the distro package), **openssh-server**, **UFW**, **fail2ban**, and **Tailscale**. Everything else — language runtimes, editors, databases, tools — runs in containers.

Install Docker from the official repository only. The distro `docker.io` package lags significantly behind. Docker Compose V1 (Python-based, invoked as `docker-compose` with a hyphen) reached end of life in July 2023 — use V2 exclusively, invoked as `docker compose` with a space. The `version:` key in compose.yaml is deprecated and ignored.

### The Docker-versus-UFW firewall trap

**Docker bypasses UFW entirely.** It manipulates iptables directly, inserting ACCEPT rules in the NAT and FORWARD chains before UFW's chains are evaluated. Any port published with `-p` is exposed to the internet regardless of UFW deny rules. This is a well-documented, serious security issue.

The fix: append rules to `/etc/ufw/after.rules` that route Docker traffic through UFW's user chain:

```
# Add to end of /etc/ufw/after.rules
*filter
:ufw-user-forward - [0:0]
:DOCKER-USER - [0:0]
-A DOCKER-USER -j ufw-user-forward
-A DOCKER-USER -m conntrack --ctstate RELATED,ESTABLISHED -j RETURN
-A DOCKER-USER -i docker0 -o docker0 -j ACCEPT
-A DOCKER-USER -s 172.16.0.0/12 -j RETURN
-A DOCKER-USER -m conntrack --ctstate NEW -j DROP
COMMIT
```

After this patch, all Docker-published ports are blocked by default. Allow specific ports with `ufw route allow proto tcp from any to any port 8080`. The simplest alternative: **bind all container ports to `127.0.0.1`** (`"127.0.0.1:8080:8080"` in compose) and access everything via Tailscale or SSH tunnels.

### Cloud-init for one-command server bootstrap

This cloud-init config provisions a complete dev server from a fresh Ubuntu instance:

```yaml
#cloud-config
disable_root: true
ssh_pwauth: false
package_update: true
package_upgrade: true

users:
  - name: devuser
    lock_passwd: true
    ssh_authorized_keys:
      - ssh-ed25519 AAAA... your-key-here
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    shell: /bin/bash

write_files:
  - path: /etc/docker/daemon.json
    content: |
      {
        "log-driver": "json-file",
        "log-opts": { "max-size": "10m", "max-file": "3" },
        "storage-driver": "overlay2"
      }

runcmd:
  - sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
  - sed -i 's/^#\?PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
  - systemctl restart sshd
  - ufw default deny incoming && ufw default allow outgoing
  - ufw allow 22/tcp && ufw allow 41641/udp && ufw --force enable
  - install -m 0755 -d /etc/apt/keyrings
  - curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  - chmod a+r /etc/apt/keyrings/docker.asc
  - echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/docker.list
  - apt-get update
  - apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  - usermod -aG docker devuser
  - curl -fsSL https://tailscale.com/install.sh | sh
  - systemctl enable tailscaled
```

**Docker does not rotate logs by default.** Without the `daemon.json` log configuration above, a single chatty container can fill your disk. The `max-size: 10m` and `max-file: 3` settings cap each container's logs at 30MB. Existing containers must be recreated to pick up new settings.

### Backups: what to protect and how

Source code lives in git and doesn't need backup. Dockerfiles, compose files, and dotfiles should also be in git. **The critical backup target is Docker volumes** — database data, container-specific state, and any persistent caches.

For a simple approach, a cron script that runs `docker run --rm -v VOLUME:/source:ro alpine tar czf /backup/VOLUME-DATE.tar.gz -C /source .` for each volume works. For production-grade backups, **Restic** provides encrypted, deduplicated, incremental backups to S3, Backblaze B2, or SFTP. Tools like **Backrest** (web UI for Restic) or **resticker** (cron-based Restic in Docker) automate this entirely within containers. For databases specifically, always dump before backing up the volume — a `pg_dump` piped through gzip is more reliable than filesystem-level volume snapshots.

Schedule backups with a systemd timer running daily at 3 AM, with retention policies like `--keep-daily 7 --keep-weekly 4 --keep-monthly 6`.

---

## Conclusion: the complete workflow in practice

The full workflow comes together as follows. You provision a server once with cloud-init, getting Docker, SSH, UFW, and Tailscale. You clone a project repo containing a `.devcontainer/` directory with a Dockerfile, docker-compose.yml, and Makefile. You run `make up` to build and start the dev container alongside its databases. From your M2 iPad Pro, you open Blink Shell, Mosh into the server, attach to tmux, and run `make nvim` — you're editing in a fully-featured Neovim 0.11 setup with LSP, completions via blink.cmp, fuzzy finding, and formatting. Yanking text copies to your iPad clipboard via OSC 52. When you want a GUI editor, you open Blink Code or a Safari PWA pointed at `https://dev-server.ts.net:8080` and get full VS Code in the browser, with the same files, same tools, same environment. When you're done with a project, `make destroy` wipes everything. When you start again, `make up` rebuilds it identically.

The key insight is that **Mosh + tmux + Tailscale eliminates the three historic pain points of remote development**: connection fragility, networking complexity, and clipboard isolation. Combined with containerized tooling that makes environments disposable and reproducible, this architecture turns a thin tablet into a genuinely capable development workstation — with the heavy computation, fast internet, and persistent state living on a server that's always on.