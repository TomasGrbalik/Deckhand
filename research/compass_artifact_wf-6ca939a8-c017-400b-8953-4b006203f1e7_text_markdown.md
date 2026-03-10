# Remote development over SSH: the definitive 2025 playbook

**A thin laptop connected to a remote Linux server over SSH, with Docker containers running your workloads, is one of the most powerful development setups available today.** The architecture offloads all compute to the server while keeping the local machine cool, quiet, and battery-efficient. But the experience hinges on tooling choices—the wrong combination creates friction that compounds across thousands of daily interactions. This guide covers every layer of that stack: editor, terminal multiplexer, SSH transport, Docker workflows, and networking, with concrete configuration and hard-won trade-offs calibrated for advanced web developers building APIs and applications in 2025.

The optimal setup depends on how you weight four priorities: editing latency, local resource usage, environment reproducibility, and GUI access. No single tool wins on all four. The two dominant approaches—VS Code Remote-SSH and tmux+Neovim—represent fundamentally different trade-offs, and the supporting infrastructure (SSH tunnels, Docker patterns, overlay networks) matters just as much as the editor choice.

---

## VS Code Remote-SSH: the GUI path with hidden costs

VS Code Remote-SSH implements a **split client/server architecture**. The local Electron shell renders UI, themes, and keymaps. A headless `vscode-server` process on the remote machine runs all workspace extensions—language servers, linters, debuggers, Git operations—with full filesystem access. Communication flows through an encrypted SSH tunnel, with the server binding to a random localhost port that gets forwarded back to the client.

This design means IntelliSense, go-to-definition, and diagnostics execute at server speed, not network speed. The network hop only carries UI protocol messages, not raw file I/O. Extensions are classified as either UI extensions (themes, keymaps—run locally) or workspace extensions (language servers, formatters—run remotely), and VS Code auto-selects the correct location.

**Dev Containers integration** works elegantly in this architecture. Connect via Remote-SSH first, then "Reopen in Container" builds or starts a Docker container on the remote daemon, installs vscode-server inside it, and reconnects—no local Docker required. The `devcontainer.json` spec supports Features (modular OCI-distributed tool installers), Docker Compose for multi-service stacks, and granular port forwarding with `portsAttributes` controlling auto-open behavior per port. A well-configured devcontainer.json for a typical web app stack looks like:

```jsonc
{
  "image": "mcr.microsoft.com/devcontainers/javascript-node:1-22-bookworm",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  },
  "forwardPorts": [3000, 5173, 5432],
  "portsAttributes": {
    "3000": { "label": "API", "onAutoForward": "notify" },
    "5173": { "label": "Vite", "onAutoForward": "openBrowser" },
    "5432": { "label": "Postgres", "onAutoForward": "silent" }
  },
  "postCreateCommand": "npm ci",
  "remoteUser": "node",
  "customizations": {
    "vscode": {
      "settings": { "files.watcherExclude": { "**/node_modules/**": true } },
      "extensions": ["dbaeumer.vscode-eslint", "esbenp.prettier-vscode"]
    }
  }
}
```

The resource cost is real, though. The client consumes **300–600 MB RAM** (Electron baseline). The server-side vscode-server plus extension host typically uses **400–800 MB**, and the TypeScript language server alone can consume 500 MB–2 GB for large projects. File watching is the single biggest resource drain—an unexcluded `node_modules` directory generates enormous event traffic over the tunnel. Aggressive `files.watcherExclude` and `search.exclude` settings are mandatory, not optional.

Latency is generally imperceptible below **100 ms RTT**. At 200 ms+ it becomes noticeable; at 300 ms+ development becomes uncomfortable. The primary latency sources are extension host round-trips (every completion request), file system operations, and terminal I/O. VS Code's port forwarding adds marginal overhead versus raw `ssh -L` (~1 ms)—negligible for HTTP requests but worth noting for high-bandwidth operations.

Key 2025 changes include a **glibc 2.28 minimum requirement** (Debian 10+, Ubuntu 20.04+), Remote Tunnels as an SSH-free alternative via `code tunnel`, and continued Dev Container spec evolution with Feature dependencies and lifecycle hooks. Reconnection after sleep/wake remains a known pain point—`ControlMaster auto` in SSH config and `ServerAliveInterval 30` are essential mitigations. If the remote server is RAM-constrained, the vscode-server process getting OOM-killed causes frustrating reconnection loops; ensure **4 GB+ RAM** on the server for web development workloads alongside Docker containers.

---

## tmux + Neovim: the thin-client purist's advantage

The terminal-based approach inverts the resource equation. Neovim running on the remote server consumes **~89 MB RAM**. The local terminal emulator (Ghostty, Kitty, Alacritty) uses **30–100 MB**. The SSH connection transmits only terminal escape sequences—typically **1–10 KB/s** during active coding. On a constrained 8 GB server running Docker containers, this is the difference between comfortable headroom and OOM pressure.

The critical architectural insight: **LSP runs as a local process on the remote server**, communicating with Neovim via stdio pipes with zero network latency. The only network hop is keystroke-to-screen-update rendering. This makes completions, diagnostics, and go-to-definition feel instantaneous regardless of connection quality. VS Code Remote achieves similar server-side execution, but its heavier protocol adds measurable overhead that compounds on slower links.

A production tmux configuration requires a few non-obvious settings. **`set -sg escape-time 10`** is critical—the default 500 ms adds perceptible lag on every Escape press. `set -g focus-events on` enables Neovim's autoread. True color requires `set -g default-terminal "tmux-256color"` plus `terminal-overrides` for the outer terminal. For undercurl (used by LSP diagnostics), specific escape sequence overrides are needed:

```bash
set -as terminal-overrides ',*:Smulx=\E[4::%p1%dm'
set -as terminal-overrides ',*:Setulc=\E[58::2::%p1%{65536}%/%d::%p1%{256}%/%{255}%&%d::%p1%{255}%&%d%;m'
```

The session organization pattern for web development: one tmux session per project, with dedicated windows for editing (Neovim), servers (split panes for Docker logs, dev server, database CLI), Git (lazygit), and tests. Tmuxinator or tmuxp define these layouts as version-controlled YAML, enabling one-command workspace reconstruction.

The 2025 Neovim plugin ecosystem has matured substantially. **lazy.nvim** handles plugin management with lazy-loading. **mason.nvim** installs language servers (typescript-language-server, pyright, gopls, rust-analyzer). **blink.cmp** has emerged as the preferred completion engine, partially written in Rust for fuzzy matching performance, replacing nvim-cmp in many setups. **conform.nvim** and **nvim-lint** handle formatting and linting. The essential navigation stack is telescope.nvim (fuzzy finding with ripgrep), harpoon 2 (quick-switch between 3–5 core files), and oil.nvim (edit filesystem as a buffer). Neovim 0.11 (2025) introduced native `vim.lsp.config()` and `vim.lsp.enable()` APIs, simplifying LSP configuration significantly.

**Clipboard sharing** between remote Neovim and local system clipboard uses OSC 52 escape sequences. Since Neovim 0.10 (May 2024), this is built-in—Neovim auto-detects OSC 52 capability. tmux requires `set -s set-clipboard on` to forward OSC 52 to the outer terminal. All modern terminal emulators (Ghostty, Kitty, WezTerm, Alacritty, iTerm2) support OSC 52. The one caveat: paste direction (system clipboard → Neovim) works via tmux's buffer, not the system clipboard directly. For most workflows this is transparent.

The killer feature remains **session persistence**. When SSH drops—laptop sleeps, WiFi changes, VPN reconnects—tmux keeps every process alive. `tmux attach` restores everything instantly. tmux-resurrect and tmux-continuum extend this to survive server reboots, auto-saving layouts every 15 minutes and restoring on tmux server start. VS Code Remote reconnects but loses terminal state and takes several seconds to re-establish the extension host.

For Docker workflows, the recommended pattern is **edit on host, run in containers**: Neovim and LSP run on the remote server, source code is bind-mounted into containers via Docker Compose. The LSP has native filesystem access without container overhead. When container-specific toolchains are needed (unusual for web development), mount your Neovim config into the container and run `docker compose exec -it app nvim`.

---

## SSH configuration is the invisible foundation

The SSH layer underpins everything, and its configuration directly impacts every interaction. A well-optimized `~/.ssh/config` is non-negotiable:

```
Host devserver
    HostName server.example.com
    User dev
    IdentityFile ~/.ssh/id_ed25519
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
    ControlPersist 600
    ServerAliveInterval 30
    ServerAliveCountMax 5
    Compression yes

Host *
    AddKeysToAgent yes
    IdentitiesOnly yes
    ForwardAgent no
```

**ControlMaster** multiplexes all SSH sessions over a single TCP connection. The first connection takes ~1 second; subsequent ones complete in ~50 ms. This dramatically accelerates git operations, scp transfers, and VS Code's dual-connection requirement. `ControlPersist 600` keeps the master alive 10 minutes after the last session closes.

**ServerAliveInterval 30** with **ServerAliveCountMax 5** declares the connection dead after 150 seconds of unresponsiveness, preventing hung terminals. Without this, a network disruption leaves SSH sessions frozen indefinitely.

**Compression** helps on WAN links below ~10 Mbps but adds CPU overhead that hurts on fast LANs. Enable it per-host rather than globally. **ForwardAgent** should be `no` globally—root users on the remote server can hijack forwarded agents. Use `ProxyJump` for bastion hosts instead, which keeps keys entirely on your local machine.

For persistent background tunnels (database, cache, monitoring UIs), **autossh** with `ServerAliveInterval`-based monitoring is the standard:

```bash
autossh -M 0 -f -N \
  -o "ServerAliveInterval 30" -o "ServerAliveCountMax 3" \
  -o "ExitOnForwardFailure=yes" \
  -L 5432:localhost:5432 -L 6379:redis:6379 \
  devserver
```

When your Docker Compose stack exposes many services on dynamic ports, **sshuttle** provides transparent subnet-level access without individual port forwards. It routes traffic through SSH at the OS level, avoiding TCP-over-TCP performance problems: `sshuttle -r devserver 172.17.0.0/16` gives your local machine direct access to all Docker container IPs. The trade-off is requiring root/sudo locally and Python on the remote.

For webhook testing and sharing work with teammates, **reverse port forwarding** (`ssh -R 0.0.0.0:8080:localhost:3000`) exposes local services through the remote server. This requires `GatewayPorts clientspecified` in the server's sshd_config.

---

## Docker workflows that stay fast and reproducible

On a remote Linux server, **bind mounts perform at native filesystem speed**—no virtualization layer like Docker Desktop on macOS. This eliminates the single largest pain point of local Docker development. Bind-mount your source code freely; use named volumes only for dependency directories (node_modules), build caches, and database data.

The anonymous volume trick prevents platform-specific native module conflicts:

```yaml
volumes:
  - .:/app              # Bind mount source code
  - /app/node_modules   # Anonymous volume preserves container's modules
```

For multi-service stacks, **Docker Compose with health checks** replaces the old `wait-for-it` script pattern. Use `depends_on` with `condition: service_healthy` and service-specific health check commands (`pg_isready`, `redis-cli ping`, `curl -f http://localhost/health`). Development-specific overrides belong in `docker-compose.override.yml`, which Compose merges automatically.

**Docker-in-Docker (DinD) vs Docker-outside-of-Docker (DooD)** is a critical design choice for dev containers. DooD mounts the host's Docker socket, sharing the daemon—simpler, faster image pulls (shared cache), but no isolation and bind mount paths must reference the host filesystem. DinD runs a separate daemon inside the container, requiring `--privileged` (security concern) but providing full isolation. For most web development, **DooD is sufficient**. Use DinD when you need volume mounts that reference paths inside the dev container or want multi-project isolation.

Reproducibility hinges on three practices: **pin base image versions** (at minimum major version tags like `node:20-bookworm-slim`), **use lock files with deterministic installs** (`npm ci`, not `npm install`), and **pre-build dev container images** in CI. The `devcontainer build` CLI or GitHub Actions can push pre-built images to a registry, cutting first-connection time from minutes to seconds:

```bash
devcontainer build --workspace-folder . --push true \
  --image-name ghcr.io/myorg/devcontainer:latest
```

---

## Tailscale and other tools that change the game

**Tailscale** is arguably the single most impactful infrastructure addition to a remote development workflow. Built on WireGuard (kernel-level, ~4,000 lines of code), it creates a peer-to-peer overlay network where every device gets a stable `100.x.x.x` IP and MagicDNS hostname. Your remote server becomes `devserver.tailnet.ts.net`—accessible from anywhere, no port forwarding or firewall configuration required.

**Tailscale SSH** eliminates SSH key management entirely, handling authentication through your identity provider. **Tailscale Funnel** exposes local services to the public internet with automatic HTTPS—replacing ngrok for webhook testing with one command: `tailscale funnel 3000`. For Docker networks, a **subnet router** on the Docker host advertises container subnets to your tailnet, giving transparent access to all container IPs from your laptop. The free tier covers 100 devices. For full self-hosting, **Headscale** (open-source coordination server, v0.26) provides most Tailscale features without the SaaS dependency.

**JetBrains Gateway** uses a thick-backend architecture (full IDE on server, thin client locally) and has improved significantly in 2025—editor operations for multiple languages now execute on the frontend, reducing perceived latency. However, it still carries "active development" status, requires **2–4 GB+ server RAM** for the IDE backend, and needs a paid JetBrains license. **Fleet was cancelled in December 2025**; JetBrains redirected that team to "JetBrains Air," an AI-focused development environment.

**Coder** (open-source, self-hosted CDE platform) provisions workspaces via Terraform on any infrastructure and supports any IDE via SSH. It makes sense for teams needing standardized environments with governance, but adds significant operational overhead for solo developers. **DevPod** by Loft Labs is a compelling lighter alternative—an open-source, client-only tool that uses `devcontainer.json` with any backend (local Docker, cloud VMs, SSH servers), costing 5–10x less than managed platforms with no vendor lock-in.

**Mosh** replaces SSH's transport with UDP-based state synchronization, providing local echo prediction and surviving network changes. It's ideal for unreliable connections but **cannot do port forwarding** and development has stagnated (last release: October 2022). With Tailscale providing a stable overlay network, many of Mosh's benefits become redundant.

**Zellij** offers a modern tmux alternative with sane defaults, floating panes, WASM plugins, and built-in session resurrection. It's pre-1.0 but very usable (~20K GitHub stars). The trade-off: it's not universally installed on remote servers, its mode system can conflict with Neovim's modal editing, and the tmux ecosystem is significantly more mature.

---

## Decision framework: which approach for which developer

| Dimension | VS Code Remote-SSH | tmux + Neovim | JetBrains Gateway |
|---|---|---|---|
| **Editing latency** | Good below 100 ms RTT; degrades noticeably above 200 ms | Excellent; terminal rendering is minimal overhead | Improving; split architecture in 2025 helps, still heavier than both |
| **Server RAM** | 1–3 GB (vscode-server + TS server) | 100–200 MB (Neovim + LSPs) | 2–4 GB (full IDE backend) |
| **Client RAM** | 300–600 MB (Electron) | 30–100 MB (terminal) | 200–400 MB (JetBrains Client) |
| **Reproducibility** | Excellent (Dev Containers spec, devcontainer.json) | Good (dotfiles repo, but manual container setup) | Good (Dev Containers plugin) |
| **GUI access** | Excellent (built-in port forwarding, browser preview) | Requires SSH -L or Tailscale; no integrated browser | Good (built-in port forwarding) |
| **Session persistence** | Reconnects but loses terminal state | tmux survives everything; instant resume | Reconnects; state recovery inconsistent |
| **Learning curve** | Low (familiar VS Code UX) | High (1–3 months to surpass VS Code productivity) | Medium (familiar to JetBrains users) |
| **Plugin ecosystem** | 50,000+ marketplace extensions | Smaller but deeply composable Lua plugins | Large but some plugins broken in remote mode |
| **Connection stability** | Known reconnection pain points after sleep/wake | Rock-solid (tmux is server-side) | Improving via Toolbox App; still inconsistent |

**Choose VS Code Remote-SSH** when your priority is fast team onboarding, Dev Containers reproducibility, or when you need rich GUI debugging (breakpoints, variable inspection) and Jupyter notebooks. It's the right choice when connection quality is consistently good (<100 ms) and the server has ample RAM.

**Choose tmux + Neovim** when you prioritize minimal resource usage, session resilience, and raw editing speed. It's objectively superior for constrained servers (≤8 GB RAM shared with Docker), unstable/high-latency connections, and developers who work across multiple remote servers. The investment pays compound returns for anyone spending 6+ hours daily in an editor.

**Choose JetBrains Gateway** when you're already invested in the JetBrains ecosystem and need its specialized language tooling (IntelliJ for Java/Kotlin, PyCharm for Python). Accept the higher resource cost and occasional stability issues.

**The networking layer is independent of editor choice.** Tailscale (or Headscale) should be part of every remote setup—it eliminates SSH key management, provides stable addressing, and enables transparent Docker network access. Layer autossh tunnels or sshuttle on top for specific port forwarding needs. Configure SSH with ControlMaster, keepalives, and per-host settings regardless of which editor you use.

## Conclusion

The most effective remote development setup in 2025 is not a single tool but a layered architecture. **Tailscale provides the networking foundation**, eliminating the complexity of SSH tunneling and firewall configuration. **Docker Compose with bind mounts on Linux** delivers native-speed, reproducible environments. The editor choice—VS Code Remote or tmux+Neovim—is ultimately a trade-off between immediate accessibility and long-term efficiency, with both approaches being genuinely excellent when properly configured. The non-obvious insight is that the infrastructure layers (SSH config, Docker patterns, overlay networking) matter as much as the editor, and getting them right eliminates entire categories of friction that no editor can compensate for.