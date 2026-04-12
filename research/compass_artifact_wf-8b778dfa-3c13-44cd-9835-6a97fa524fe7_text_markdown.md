# SSH into Docker containers over Tailscale: a complete networking guide

**The simplest and most robust approach for a personal dev setup with 5-8 containers across multiple Docker Compose projects is a shared external bridge network with static IPs, advertised as a single Tailscale subnet route.** This gives each container a predictable IP on port 22, avoids port-number juggling, preserves project-level network isolation, and requires only one `/24` route in Tailscale. For those who want zero networking configuration, host-port mapping is a viable alternative that trades IP elegance for setup simplicity. Below is a detailed analysis of every approach, with exact configurations you can use with Deckhand.

---

## 1. Advertising multiple Docker subnets via Tailscale

Tailscale's subnet router fully supports comma-separated route advertisement. The syntax accepts multiple CIDRs with no documented limit:

```bash
sudo tailscale set --advertise-routes=172.20.0.0/24,172.21.0.0/24,172.22.0.0/24
```

Each route requires approval in the Tailscale admin console (Machines → three-dot menu → Edit route settings), or you can automate this with ACL `autoApprovers`:

```json
{
  "autoApprovers": {
    "routes": {
      "172.20.0.0/16": ["tag:docker-router"]
    }
  }
}
```

**You can advertise a broad CIDR like `172.20.0.0/16`** to cover all Docker networks at once. Tailscale won't reject it. However, this carries a real risk: any tailnet member with `--accept-routes` enabled will route all `172.20.x.x` traffic through your server, which breaks if another tailnet device sits on an overlapping subnet. Tailscale uses longest-prefix matching, so a more specific `/24` route from another subnet router would take precedence, but **failover does not cross prefix boundaries** — if the `/24` router dies, Tailscale drops that traffic rather than falling back to your `/16`.

The dynamic-creation problem is the real pain point. Every `docker compose up` creates a new bridge network with an unpredictable subnet. `tailscale set --advertise-routes` is a **replace-not-append** operation — you must pass the complete route list every time. A script watching `docker events --filter type=network` can automate this, but it adds fragile moving parts. This is why approaches 2 and 3 below are better: they make subnets predictable so you can advertise a single stable route.

**Host prerequisites for any subnet router setup:**

```bash
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.d/99-tailscale.conf
echo 'net.ipv6.conf.all.forwarding = 1' | sudo tee -a /etc/sysctl.d/99-tailscale.conf
sudo sysctl -p /etc/sysctl.d/99-tailscale.conf
```

Tailscale can run natively on the host or in a container. For subnet routing to Docker networks, native installation is simplest. If containerized, you **must** use `network_mode: host` plus `NET_ADMIN` and `NET_RAW` capabilities and `/dev/net/tun` access — otherwise the Tailscale container can't see the Docker bridge interfaces.

---

## 2. The shared external network is the cleanest solution

A container can absolutely connect to **two Docker networks simultaneously** — its project-isolated network and a shared SSH-access network. This is a first-class Docker feature, explicitly documented and widely used.

**Step 1: Create the shared network once (before any Compose project starts):**

```bash
docker network create --driver=bridge --subnet=172.30.0.0/24 --gateway=172.30.0.1 ssh-net
```

**Step 2: Reference it in each Compose file with a static IP:**

```yaml
# project-alpha/docker-compose.yml
services:
  devbox:
    image: my-dev-image
    networks:
      default: {}          # project-alpha's isolated bridge
      ssh-net:
        ipv4_address: 172.30.0.10

networks:
  ssh-net:
    external: true
```

```yaml
# project-beta/docker-compose.yml
services:
  devbox:
    image: my-dev-image
    networks:
      default: {}
      ssh-net:
        ipv4_address: 172.30.0.11

networks:
  ssh-net:
    external: true
```

**Critical detail**: when you explicitly list networks under a service, you *must* include `default` if you still want the container on its project network. Omitting it means the container only joins the networks you list.

**Step 3: Advertise only this one subnet via Tailscale:**

```bash
sudo tailscale set --advertise-routes=172.30.0.0/24
```

**Step 4: Configure `~/.ssh/config` on your MacBook:**

```
Host project-alpha
    HostName 172.30.0.10
    User devuser
    Port 22

Host project-beta
    HostName 172.30.0.11
    User devuser
    Port 22
```

VS Code Remote SSH picks these up in the Remote Explorer. Each opens a separate window connected directly to the container.

### Does the second network break isolation?

**No.** Containers on `project-alpha_default` that are *not* on `ssh-net` remain fully isolated from containers on `project-beta_default`. The dual-homed container (on both networks) can reach containers on both, but **it does not act as a router** between them — Linux containers have IP forwarding disabled by default, and Docker's iptables rules prevent cross-bridge traffic. DNS resolution is scoped per network, so service names from one project aren't resolvable on the other. The isolation model is preserved; the dual-homed container simply has two doors, and nothing passes through it.

### Gotchas to watch

**IP uniqueness is your responsibility.** Docker doesn't check for static IP conflicts across Compose files. If two containers claim `172.30.0.10`, the second fails to start. For Deckhand, this means the orchestrator should assign IPs from a registry or use a naming convention (e.g., project index × 1 + base offset). The external network survives `docker compose down` by design — it requires explicit `docker network rm ssh-net` to remove. And `ssh-net` **must exist before** `docker compose up` runs, or Compose errors with "network ssh-net declared as external, but could not be found."

---

## 3. Controlling Docker's IPAM makes subnets predictable

Even if you don't use a shared external network, you can force Docker to assign all auto-created bridge networks within a known range by configuring `/etc/docker/daemon.json`:

```json
{
  "default-address-pools": [
    {
      "base": "172.20.0.0/16",
      "size": 24
    }
  ]
}
```

This tells Docker to carve `/24` subnets out of `172.20.0.0/16` for every new bridge network (including Compose-created `<project>_default` networks). You get up to **256 networks**, each with **254 usable IPs**. After editing, restart Docker with `sudo systemctl restart docker` — this only affects newly created networks, not existing ones.

With this in place, you can advertise `172.20.0.0/16` via Tailscale and know that all future Docker networks fall within it. This pairs well with approach 1 (broad CIDR advertisement) because you've eliminated the unpredictability.

**Key gotchas**: the key is `"default-address-pools"` (plural with 's') — the singular form silently fails. Ensure your chosen base doesn't overlap with your LAN. Docker's built-in defaults span `172.17.0.0/16` through `172.31.0.0/16` plus `192.168.0.0/16`, so picking `172.20.0.0/16` overlaps with Docker's default pools. Either accept this (your config replaces the defaults entirely) or pick something distinctive like `10.200.0.0/16`. Explicitly created networks (via `docker network create --subnet=...`) bypass `default-address-pools` entirely.

The limitation compared to approach 2 is that **container IPs within each network are still dynamic** unless you pin them in the Compose file. You know the network will be `172.20.3.0/24`, but which container gets `.2` vs `.3` depends on startup order. For SSH access, you still need static IPs or a discovery mechanism.

---

## 4. Host-port mapping sidesteps networking entirely

The simplest approach ignores container IPs altogether: map each container's SSH port to a unique host port and connect to the server's Tailscale IP.

```yaml
# project-alpha/docker-compose.yml
services:
  devbox:
    image: my-dev-image
    ports:
      - "2210:22"

# project-beta/docker-compose.yml
services:
  devbox:
    image: my-dev-image
    ports:
      - "2211:22"
```

```
# ~/.ssh/config on MacBook
Host project-alpha
    HostName my-server.tailnet-name.ts.net  # MagicDNS name
    User devuser
    Port 2210

Host project-beta
    HostName my-server.tailnet-name.ts.net
    User devuser
    Port 2211
```

**This requires zero network configuration** — no custom subnets, no daemon.json changes, no external networks, no route advertisement. It works on Docker Desktop (Mac/Windows) too, and VS Code Remote SSH handles custom ports natively via the `Port` directive.

The downsides are real but manageable at small scale. **Port management becomes a coordination problem**: Deckhand needs a port allocation scheme to avoid conflicts. Every tool that connects needs to know the non-standard port. And there's no direct container-to-container SSH — though for a dev workflow focused on VS Code → container connections, that rarely matters.

| Aspect | Port mapping | Shared network + subnet route |
|--------|-------------|-------------------------------|
| Setup complexity | Very low | Medium |
| Network config needed | None | External network + Tailscale route |
| Container addressing | `host:port` | `container-ip:22` |
| Port 22 everywhere | No (unique ports) | Yes |
| Works on Docker Desktop | Yes | No (subnet routing is Linux-only) |
| Scales to 20+ containers | Awkward | Clean |

---

## 5. Macvlan and ipvlan are overkill here

Macvlan gives each container its own MAC address and an IP on your physical LAN — the container appears as a separate device on the network. Ipvlan is similar but shares the host's MAC.

**Both have a critical limitation**: the host cannot communicate with macvlan/ipvlan containers without a manual shim interface workaround (`ip link add macvlan-shim link eth0 type macvlan mode bridge`). This shim doesn't persist across reboots without additional configuration.

Additional dealbreakers for a dev SSH setup: **macvlan/ipvlan only works on Linux** (not Docker Desktop), isn't supported in rootless Docker, doesn't work on most cloud VMs (AWS/GCP/Azure block multiple MACs per interface), and requires IP range coordination with your LAN's DHCP server. The complexity-to-benefit ratio is poor when bridge networks with Tailscale subnet routing accomplish the same goal with less friction. Macvlan is appropriate when containers need to be directly addressable from your physical LAN like standalone machines — not the case here.

---

## 6. Recommended setup for your Deckhand workflow

For **5-8 dev containers across multiple isolated Docker Compose projects**, accessed via VS Code Remote SSH from a MacBook over Tailscale, here is the recommended architecture ranked by robustness and simplicity:

### Primary recommendation: shared external network + single Tailscale route

This is the best balance of simplicity, predictability, and cleanliness for your use case.

**On the Docker host (one-time setup):**

```bash
# 1. Install Tailscale on the host (or run it in a container with network_mode: host)
# 2. Enable IP forwarding
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.d/99-tailscale.conf
sudo sysctl -p /etc/sysctl.d/99-tailscale.conf

# 3. Create the shared SSH network
docker network create --driver=bridge --subnet=172.30.0.0/24 --gateway=172.30.0.1 ssh-net

# 4. Advertise the route
sudo tailscale set --advertise-routes=172.30.0.0/24

# 5. Approve the route in Tailscale admin console (or use autoApprovers in ACLs)
```

**In each Deckhand-managed Compose file**, add the shared network with a static IP:

```yaml
services:
  devbox:
    image: my-dev-image
    networks:
      default: {}
      ssh-net:
        ipv4_address: 172.30.0.${PROJECT_INDEX}  # e.g., .10, .11, .12

networks:
  ssh-net:
    external: true
```

**On your MacBook** (ensure `tailscale up --accept-routes` is active):

```
# ~/.ssh/config
Host alpha-dev
    HostName 172.30.0.10
    User devuser

Host beta-dev
    HostName 172.30.0.11
    User devuser
```

Open VS Code → Remote Explorer → select the host → you're in the container.

**Why this wins**: one stable subnet, one Tailscale route, standard port 22, static IPs that survive container restarts, project networks remain isolated, and Deckhand can manage IP assignment by incrementing a counter. The `ssh-net` network persists through `docker compose down` cycles, and the Tailscale route never needs updating.

### Fallback: host-port mapping (if you want zero Docker networking config)

If the external network approach feels like too much orchestration for Deckhand to manage, port mapping works fine at your scale. Have Deckhand assign ports from a range (2210-2220), map them in each Compose file, and point VS Code at `my-server.tailnet-name.ts.net:221X`. No subnet routing needed — just Tailscale on the host for basic connectivity.

### What to avoid

Skip the **Tailscale sidecar pattern** (one Tailscale container per service) for SSH use cases — it's designed for giving services their own tailnet identity, and SSH into a sidecar lands you in the Tailscale container's filesystem, not your dev container's. Skip **macvlan/ipvlan** entirely. Skip **broad CIDR advertisement** (`172.16.0.0/12`) unless you're the only person on your tailnet and comfortable with the routing implications. And avoid **dynamic route re-advertisement scripts** — they add fragility that the shared-network approach eliminates completely.

### Docker 28+ compatibility note

Recent Docker versions (28+) introduced iptables/nftables changes that can break subnet routing to containers. If you hit connectivity issues after upgrading Docker, add `"ip-forward-no-drop": true` to your `/etc/docker/daemon.json` and restart Docker. This is tracked in moby/moby#51015 and affects anyone using Tailscale subnet routing to reach Docker bridge networks.