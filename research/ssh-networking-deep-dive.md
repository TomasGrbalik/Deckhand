# SSH, networking & port exposure: the full connection architecture

**Tailscale is a connectivity layer — it gets your iPad to the server. SSH does everything else.** All port forwarding, all service access, all security goes through SSH tunnels. This keeps the architecture simple and uniform: one transport, one auth mechanism, one mental model.

---

## The network topology

```
iPad (Blink Shell / Safari)
  │
  ├─ Tailscale VPN ─── stable 100.x.y.z IP to server ─── that's it
  │
  ├─ Mosh (over Tailscale IP) ──── interactive terminal ──── tmux ──── nvim
  │
  └─ SSH (over Tailscale IP) ──── port tunnels ──── localhost:PORT ──── Docker containers
       │
       ├─ -L 8080:localhost:8080 ──► code-server in container
       ├─ -L 5173:localhost:5173 ──► Vite dev server in container
       ├─ -L 3000:localhost:3000 ──► Express API in container
       └─ -L 5432:localhost:5432 ──► Postgres in container
```

Tailscale replaces "knowing your server's public IP, configuring firewall rules, dealing with dynamic DNS." Your server is always `dev-server.your-tailnet.ts.net` or `100.x.y.z`. You SSH to that address. That's Tailscale's entire job.

---

## The two-connection model on iPad

iPad's background app behavior makes a single connection impractical. You need two:

**Connection 1 — Mosh: your interactive terminal**

```bash
# In Blink Shell
mosh user@100.x.y.z -- tmux new -A -s main
```

Mosh gives you: instant reconnection after iPad sleep, local echo for fast typing feel, survival across WiFi/cellular transitions. tmux keeps your session alive server-side even if Mosh somehow disconnects. This is where you run Neovim, watch logs, run tests.

Mosh **cannot do port forwarding**. That's not a limitation of your setup — it's a fundamental protocol constraint (UDP state sync has no concept of TCP tunnels).

**Connection 2 — SSH: your tunnel carrier**

```bash
# In a second Blink Shell tab
ssh -N \
  -L 8080:localhost:8080 \
  -L 5173:localhost:5173 \
  -L 3000:localhost:3000 \
  user@100.x.y.z
```

`-N` means "no shell, just hold tunnels." This session sits quietly forwarding TCP. When you open Safari on your iPad and go to `http://localhost:8080`, traffic flows through this SSH tunnel to the server's localhost:8080, which Docker has mapped to the container's port 8080.

If this SSH session drops (iPad sleep, network change), the tunnels die and you need to reconnect. Blink Shell can auto-reconnect SSH sessions, but it's not as resilient as Mosh. This is an acceptable tradeoff — losing a tunnel means refreshing a browser tab, not losing your editor state.

### Opening Safari to reach code-server

With the SSH tunnel active:

1. Open Safari on iPad
2. Navigate to `http://localhost:8080`
3. code-server loads — you're editing in VS Code in the browser
4. Share → Add to Home Screen for PWA mode (fullscreen, no Safari toolbar)

Note: **HTTP, not HTTPS.** The SSH tunnel provides encryption end-to-end. code-server's built-in TLS is unnecessary and causes certificate warnings in Safari. Configure code-server with `auth: none` and `cert: false` — this is safe because the only path to port 8080 is through your SSH tunnel.

---

## Docker port binding: the one rule

**Every port in docker-compose.yml binds to `127.0.0.1`. No exceptions.**

```yaml
ports:
  - "127.0.0.1:8080:8080"   # code-server
  - "127.0.0.1:5173:5173"   # Vite
  - "127.0.0.1:3000:3000"   # Express/API
  - "127.0.0.1:5432:5432"   # Postgres
```

Why this matters: Docker bypasses UFW/iptables entirely. If you write `"8080:8080"` (which defaults to `0.0.0.0`), that port is exposed to the entire internet regardless of your firewall rules. Docker inserts its own DNAT rules in the iptables DOCKER chain that are evaluated before UFW's chains. Binding to `127.0.0.1` means the port is only reachable from the server's own loopback interface — which is exactly where SSH tunnels terminate.

Container-to-container traffic doesn't need port mappings at all. The `postgres` container is reachable from the `devcontainer` at `postgres:5432` via Docker's internal DNS. Only ports that your **iPad needs to reach** get `-p` mappings.

---

## Adding a new port at runtime

This is the most common friction point: "I just started a WebSocket server on port 4000 in my container. How do I reach it?"

### Step 1: Does the port mapping exist in Docker?

Docker doesn't allow adding port mappings to a running container. If port 4000 wasn't in your docker-compose.yml, the container needs to be recreated. Two approaches:

**Approach A — Pre-map a port range in the template:**

```yaml
# docker-compose.yml
ports:
  - "127.0.0.1:8080:8080"
  - "127.0.0.1:3000:3000"
  - "127.0.0.1:4000-4010:4000-4010"  # 10 spare ports
```

This gives you headroom. Any process inside the container that binds to ports 4000–4010 is immediately accessible from the host's localhost. No container restart needed.

**Approach B — devbox adds the port and recreates:**

```bash
devbox port add 4000 --name websocket
# → updates docker-compose.yml: adds "127.0.0.1:4000:4000"
# → runs: docker compose up -d
# → Compose detects the port change and recreates the devcontainer service
# → other services (postgres, redis) are untouched
```

The recreate takes ~2 seconds. Your tmux session inside the container is lost (it's a new container), but your source code is on a bind mount and untouched. This is why pre-mapping a range is preferred for frequently-changed setups.

### Step 2: Add the port to your SSH tunnel

You have three options, from easiest to most manual:

**Option 1 — SSH escape sequence (no reconnect needed):**

Inside your active SSH tunnel session in Blink Shell, type:

```
[Enter]~C
```

The tilde (`~`) must be the first character after a newline. You'll see an `ssh>` prompt. Type:

```
-L 4000:localhost:4000
```

Press Enter. The tunnel is live. No disconnection, no reconnection. This is the fastest path.

**Option 2 — Reconnect with the new port:**

Close the SSH tunnel tab in Blink and open a new one:

```bash
ssh -N \
  -L 8080:localhost:8080 \
  -L 5173:localhost:5173 \
  -L 3000:localhost:3000 \
  -L 4000:localhost:4000 \
  user@100.x.y.z
```

**Option 3 — devbox generates the command:**

```bash
devbox connect
# Reads all port labels from running containers
# Outputs the full SSH command with every mapped port:
#
#   ssh -N \
#     -L 8080:localhost:8080 \
#     -L 5173:localhost:5173 \
#     -L 3000:localhost:3000 \
#     -L 4000:localhost:4000 \
#     user@100.x.y.z
#
# Copy this into Blink Shell on iPad
```

### Step 3: Open in browser

`http://localhost:4000` in Safari. Done.

---

## What devbox implements for all of this

### Port management commands

```bash
devbox port list                       # Show all mapped ports for current project
devbox port add 4000 --name ws         # Add port mapping (recreates container)
devbox port remove 4000               # Remove port mapping (recreates container)
devbox port range 4000-4010           # Pre-map a range of spare ports
devbox connect                         # Generate SSH tunnel command with all ports
devbox connect --clipboard             # Copy the SSH command to clipboard
```

### Example output of `devbox port list`

```
PROJECT: my-api (running)
┌──────┬─────────────┬──────────┬──────────────────────────────────────────┐
│ PORT │ NAME        │ PROTOCOL │ ACCESS                                   │
├──────┼─────────────┼──────────┼──────────────────────────────────────────┤
│ 8080 │ code-server │ http     │ ssh -L 8080:localhost:8080 → browser     │
│ 3000 │ api         │ http     │ ssh -L 3000:localhost:3000 → browser     │
│ 5173 │ vite        │ http     │ ssh -L 5173:localhost:5173 → browser     │
│ 5432 │ postgres    │ tcp      │ ssh -L 5432:localhost:5432 → psql/GUI    │
│ 6379 │ redis       │ tcp      │ (internal only, container-to-container)  │
└──────┴─────────────┴──────────┴──────────────────────────────────────────┘

Connect command (copy to Blink Shell):
  ssh -N -L 8080:localhost:8080 -L 3000:localhost:3000 -L 5173:localhost:5173 -L 5432:localhost:5432 user@100.64.1.3
```

### How the docker-compose.yml gets generated

When `devbox up` runs, it reads the template's port definitions and generates:

```yaml
services:
  devcontainer:
    build: .devcontainer/
    volumes:
      - .:/workspace
      - node-modules:/workspace/node_modules
    ports:
      - "127.0.0.1:8080:8080"     # code-server
      - "127.0.0.1:3000:3000"     # api
      - "127.0.0.1:5173:5173"     # vite
      - "127.0.0.1:4000-4010:4000-4010"  # spare range
    labels:
      dev.devbox.managed: "true"
      dev.devbox.project: "my-api"
      dev.devbox.port.8080: "code-server"
      dev.devbox.port.8080.protocol: "http"
      dev.devbox.port.3000: "api"
      dev.devbox.port.3000.protocol: "http"
      dev.devbox.port.5173: "vite"
      dev.devbox.port.5173.protocol: "http"
      dev.devbox.port.5432: "postgres"
      dev.devbox.port.5432.protocol: "tcp"
      dev.devbox.port.5432.internal: "true"
    command: sleep infinity
    networks: [dev-network]

  postgres:
    image: postgres:16-alpine
    # NO ports section — internal only
    networks: [dev-network]

  redis:
    image: redis:7-alpine
    # NO ports section — internal only
    networks: [dev-network]

networks:
  dev-network:
    driver: bridge
```

Note that postgres and redis have **no port mappings** unless you explicitly want to reach them from your iPad (e.g., to use a local database GUI). They're accessible from the devcontainer via Docker DNS. devbox only adds host port mappings for services you actually need to reach from outside.

### Label schema for port discovery

```go
// Reading ports from a running container
labels := container.Labels
// Filter keys starting with "dev.devbox.port."
// Parse: dev.devbox.port.{PORT} = name
//        dev.devbox.port.{PORT}.protocol = http|tcp
//        dev.devbox.port.{PORT}.internal = true|false
```

`devbox port list` and `devbox connect` query these labels from the Docker API — no need to parse compose files at runtime.

---

## The template's port definition

In `devbox-template.yaml`:

```yaml
ports:
  - port: 8080
    name: code-server
    protocol: http
    description: "VS Code in browser"

  - port: 3000
    name: api
    protocol: http
    description: "Application server"

  - port: 5432
    name: postgres
    protocol: tcp
    internal: true          # container-to-container only by default
    description: "PostgreSQL"

spare_port_range: "4000-4010"   # pre-mapped for ad-hoc services
```

When a user runs `devbox init` or `devbox up`, this gets merged with any project-level overrides and rendered into the compose file.

---

## SSH configuration on the server

The server's `/etc/ssh/sshd_config` needs no special changes for this setup. The defaults work. But these settings are worth verifying:

```
AllowTcpForwarding yes          # Required for -L tunnels (default: yes)
GatewayPorts no                 # Keep tunnels bound to localhost (default: no)
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
```

`GatewayPorts no` is important — it ensures SSH tunnels bind to `127.0.0.1` on the server, not `0.0.0.0`. Combined with Docker's `127.0.0.1` port binding, this means no service is ever reachable except through an authenticated SSH session.

On the **client side** (iPad's Blink Shell or laptop's `~/.ssh/config`):

```
Host devbox
    HostName 100.x.y.z                 # Tailscale IP
    User dev
    IdentityFile ~/.ssh/id_ed25519
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
    ControlPersist 600
    ServerAliveInterval 30
    ServerAliveCountMax 5
```

`ControlMaster auto` means the first SSH connection becomes the master, and subsequent connections (including tunnel-only sessions) multiplex over it. This makes `ssh -N -L ...` instant to establish if you already have an active SSH session.

---

## The complete daily workflow

1. **Pick up iPad, open Blink Shell**
2. **Tab 1**: `mosh devbox -- tmux attach` — your terminal session is exactly where you left it. Neovim, running servers, logs — all intact.
3. **Tab 2**: `ssh -N -L 8080:localhost:8080 -L 5173:localhost:5173 devbox` — or paste the output of `devbox connect` that you ran earlier in tmux.
4. **Switch to Safari**: `http://localhost:8080` — code-server loads. `http://localhost:5173` — your Vite app loads with HMR.
5. **Need a new port?** In your Mosh/tmux session: start the service on port 4000 (it's already pre-mapped via the spare range). In the SSH tunnel tab: `[Enter]~C` then `-L 4000:localhost:4000`. Safari: `http://localhost:4000`.
6. **iPad sleeps for 2 hours. You open it again.** Mosh reconnects in <1 second. SSH tunnel may need a tab restart — `ssh -N -L ... devbox` again (3 seconds with ControlMaster). Done.

---

## What devbox implements in Go for this

### Core types

```go
// internal/domain/ports.go
type PortMapping struct {
    Port        int
    Name        string
    Protocol    string  // "http" or "tcp"
    Internal    bool    // container-to-container only
}

type PortRange struct {
    Start int
    End   int
}
```

### Networking service

```go
// internal/service/network_service.go
type NetworkService struct {
    docker DockerClient
    config *Config
}

// Returns the SSH tunnel command with all mapped ports
func (s *NetworkService) GenerateConnectCommand(project string) (string, error) {
    containers := s.docker.ListByLabel("dev.devbox.project", project)
    ports := extractPortLabels(containers)

    // Filter out internal-only ports
    tunnelPorts := filterExternalPorts(ports)

    // Build: ssh -N -L 8080:localhost:8080 -L 3000:localhost:3000 ... user@host
    return buildSSHCommand(s.config.SSHUser, s.config.SSHHost, tunnelPorts), nil
}

// Adds a port mapping — triggers compose recreate
func (s *NetworkService) AddPort(project string, port int, name string) error {
    // 1. Load project's docker-compose.yml via compose-go
    // 2. Add "127.0.0.1:PORT:PORT" to devcontainer service
    // 3. Add label "dev.devbox.port.PORT=name"
    // 4. Write updated compose file
    // 5. docker compose up -d (triggers selective recreate)
}
```

### CLI commands

```go
// internal/cli/port.go
var portCmd = &cobra.Command{Use: "port", Short: "Manage port mappings"}

var portListCmd = &cobra.Command{
    Use: "list",
    Run: func(cmd *cobra.Command, args []string) {
        svc := service.NewNetworkService(...)
        ports, _ := svc.ListPorts(currentProject)
        renderPortTable(ports)
        fmt.Println("\nConnect command:")
        cmd, _ := svc.GenerateConnectCommand(currentProject)
        fmt.Println(" ", cmd)
    },
}

var connectCmd = &cobra.Command{
    Use:   "connect",
    Short: "Print SSH tunnel command for all mapped ports",
    Run: func(cmd *cobra.Command, args []string) {
        svc := service.NewNetworkService(...)
        sshCmd, _ := svc.GenerateConnectCommand(currentProject)
        fmt.Println(sshCmd)
    },
}
```

---

## Summary

Tailscale = reach the server. SSH = tunnel everything. Mosh = interactive terminal. Docker binds to `127.0.0.1` only. Pre-map a spare port range to avoid container recreation. `devbox connect` generates the full SSH tunnel command. `devbox port add` handles the rest.