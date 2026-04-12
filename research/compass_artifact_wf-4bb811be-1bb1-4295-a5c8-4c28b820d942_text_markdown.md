# Docker containers on Tailscale for VS Code Remote SSH

**The subnet router approach is the most practical path for your setup.** It uses a single Tailscale device (your existing host server), avoids per-container Tailscale overhead, and works reliably with VS Code Remote SSH — which has a known, still-open incompatibility with Tailscale's built-in SSH server. With 5–8 long-lived containers on the free plan, the subnet router keeps you at just 2 Tailscale nodes (MacBook + server) while giving your MacBook direct SSH access to every container's internal IP.

The key insight shaping this recommendation: VS Code Remote SSH requires OpenSSH's `direct-tcpip` channel type for port forwarding, which **Tailscale SSH does not support** (GitHub issue #5295, open since August 2022). This means every approach — subnet router, sidecar, or per-container Tailscale — requires running a traditional `sshd` inside each container anyway. That eliminates the sidecar pattern's main advantage (keyless Tailscale SSH) and makes the subnet router the clear winner for simplicity.

---

## Tailscale's free plan comfortably fits any approach

Tailscale released **Pricing v4 on April 8, 2026**, significantly restructuring limits. The Personal (free) plan now offers **unlimited user devices**, up to **50 tagged resources** (servers, containers with tags), and **6 users**. The previous v3 plan capped devices at 100. Under either model, 5–8 containers plus a MacBook and server fall well within limits.

The distinction matters for approach selection. Under v4, devices break into three categories: user devices (unlimited, free), tagged resources (up to 50 on Personal), and ephemeral resources (1,000 minutes/month). Each Docker container running its own Tailscale instance counts as a separate node — either a user device or tagged resource depending on configuration. A subnet router, by contrast, adds **zero additional devices** since containers behind it never join the tailnet directly. For a personal free plan, this difference is academic at 5–8 containers, but the subnet router's minimal footprint is the cleanest option.

---

## The subnet router approach, step by step

The subnet router transforms your existing host server into a gateway. Your MacBook sends traffic destined for Docker internal IPs through the Tailscale tunnel to the server, which forwards it into the Docker bridge network. The traffic flow is straightforward:

```
MacBook (100.x.y.z) ──WireGuard tunnel──▶ Dev Server (100.a.b.c / 172.20.0.1) ──bridge──▶ Container (172.20.0.10)
```

### Create a custom Docker network with static IPs

Do not use Docker's default bridge (`172.17.0.0/16`). It doesn't support static IP assignment, and container IPs shift on restart. Create a dedicated network with a small, conflict-free subnet:

```bash
docker network create --driver bridge --subnet 172.20.0.0/24 --gateway 172.20.0.1 devnet
```

In your Docker Compose (or Deckhand configuration), assign each container a fixed IP:

```yaml
services:
  dev-python:
    build: ./containers/python
    container_name: dev-python
    hostname: dev-python
    networks:
      devnet:
        ipv4_address: 172.20.0.10
    volumes:
      - ./projects/python:/workspace
    restart: unless-stopped

  dev-node:
    container_name: dev-node
    networks:
      devnet:
        ipv4_address: 172.20.0.11
    # ... similar config

networks:
  devnet:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/24
          gateway: 172.20.0.1
```

Static IPs persist across `docker compose down && docker compose up` as long as the network definition remains unchanged. Choose **172.20.0.0/24** or another range unlikely to conflict with your home LAN, VPNs, or cloud networks. The `172.16.0.0/12` block is heavily used by Docker and corporate networks — pick a less common slice of it.

### Configure the host as a subnet router

Three steps on the server. First, enable IP forwarding (required for the kernel to route packets between interfaces):

```bash
echo 'net.ipv4.ip_forward = 1' | sudo tee /etc/sysctl.d/99-tailscale.conf
echo 'net.ipv6.conf.all.forwarding = 1' | sudo tee -a /etc/sysctl.d/99-tailscale.conf
sudo sysctl -p /etc/sysctl.d/99-tailscale.conf
```

Second, advertise the Docker network route to your tailnet:

```bash
sudo tailscale set --advertise-routes=172.20.0.0/24
```

Use `tailscale set` rather than `tailscale up` — it modifies only the specified setting without resetting your existing configuration. Third, approve the route in the Tailscale admin console at `login.tailscale.com/admin/machines`: find your server, click its menu, select **Edit route settings**, enable the `172.20.0.0/24` route, and save. Optionally, disable key expiry on the server to avoid periodic reauthentication.

Tailscale subnet routers use **SNAT by default**, meaning traffic arriving at containers appears to come from the Docker gateway (172.20.0.1). This is ideal — containers already know how to route back through the gateway without any additional configuration.

### Install SSH servers in each container

Every container needs a running `sshd`. A minimal Dockerfile pattern:

```dockerfile
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    openssh-server sudo curl git vim \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir /var/run/sshd

# Create dev user with passwordless sudo
RUN useradd -rm -d /home/dev -s /bin/bash -G sudo -u 1000 dev \
    && echo 'dev ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

# SSH key auth only
RUN mkdir -p /home/dev/.ssh && chmod 700 /home/dev/.ssh
COPY authorized_keys /home/dev/.ssh/authorized_keys
RUN chmod 600 /home/dev/.ssh/authorized_keys \
    && chown -R dev:dev /home/dev/.ssh

# Harden SSH config
RUN sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

RUN ssh-keygen -A
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
```

Place your MacBook's public key (e.g., `~/.ssh/id_ed25519.pub`) in an `authorized_keys` file in each container's build context. Containers **do not need to publish port 22 to the host** — the subnet router provides direct network-layer access to each container's internal IP. For persistent host keys (avoiding "host key changed" warnings on rebuild), mount a volume for `/etc/ssh/ssh_host_*_key`.

### Configure the MacBook

On macOS, Tailscale **auto-accepts subnet routes by default** — no extra configuration required. Verify by checking that "Use Tailscale subnets" is enabled in the Tailscale menu bar app's settings. Then configure `~/.ssh/config`:

```ssh-config
Host dev-python
    HostName 172.20.0.10
    User dev
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ServerAliveInterval 15
    ServerAliveCountMax 3

Host dev-node
    HostName 172.20.0.11
    User dev
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ServerAliveInterval 15
    ServerAliveCountMax 3

# Repeat for each container...
```

`StrictHostKeyChecking no` and `UserKnownHostsFile /dev/null` prevent warnings when containers are rebuilt (generating new host keys). **ServerAliveInterval 15** is essential — it sends keepalives every 15 seconds, preventing silent disconnections when your MacBook sleeps or switches networks. In VS Code, press `F1` → "Remote-SSH: Connect to Host" → select `dev-python` from the list. VS Code connects directly through the Tailscale tunnel to the container.

### DNS and hostname resolution

**MagicDNS does not resolve container names behind a subnet router** — it only resolves devices directly on the tailnet. For 5–8 containers, the SSH config aliases (`Host dev-python`, etc.) handle naming for SSH and VS Code. If you want broader hostname resolution (for browsers, curl, etc.), add entries to `/etc/hosts` on your MacBook:

```
172.20.0.10  dev-python
172.20.0.11  dev-node
172.20.0.12  dev-rust
```

This is simple and sufficient. A more elaborate option — running dnsmasq on the server and configuring Tailscale split DNS — is overkill for this container count.

### ACL configuration

If you haven't modified your Tailscale ACL policy from the default (which allows all traffic between all devices), **no ACL changes are needed**. If you have customized ACLs, add a rule allowing your user to reach the Docker subnet:

```json
{
  "acls": [
    {"action": "accept", "src": ["your-email@example.com"], "dst": ["172.20.0.0/24:*"]}
  ]
}
```

You can optionally set up auto-approvers to skip manual route approval when the server re-advertises routes:

```json
{
  "autoApprovers": {
    "routes": {
      "172.20.0.0/24": ["your-email@example.com"]
    }
  }
}
```

---

## Why the sidecar pattern loses its edge here

The Tailscale sidecar pattern — running a `tailscale/tailscale` container alongside each app container with shared network namespaces — is Tailscale's officially recommended Docker approach. Each container gets its own **100.x.y.z IP** and **MagicDNS hostname**, enabling connections like `ssh dev@dev-python.tailnet.ts.net`. The overhead is modest at roughly **20 MB RAM per sidecar**.

The pattern's killer feature is Tailscale SSH (`--ssh`), which eliminates SSH key management entirely — authentication happens through your Tailscale identity. However, **VS Code Remote SSH cannot work with Tailscale SSH**. The extension requires `direct-tcpip` and `direct-streamlocal@openssh.com` SSH channel types for its port-forwarding mechanism. Tailscale's SSH implementation (built on `gliderlabs/ssh`) doesn't support these. This is tracked in GitHub issue #5295, open since August 2022 with no resolution as of April 2026.

This means even with sidecars, you must install and run `sshd` inside each container and manage SSH keys — exactly as with the subnet router approach. The sidecar then adds complexity (Docker Compose boilerplate per container, auth key/OAuth management, extra running processes) without meaningful benefit for VS Code SSH workflows. Where the sidecar approach shines is when you need per-container ACLs, automatic HTTPS certificates via Tailscale Serve, or MagicDNS names without manual DNS configuration — none of which are critical for a personal dev setup.

A Docker Compose sidecar configuration, for reference:

```yaml
services:
  ts-devenv:
    image: tailscale/tailscale:latest
    hostname: dev-python
    environment:
      - TS_AUTHKEY=tskey-client-XXXXX?ephemeral=false
      - TS_STATE_DIR=/var/lib/tailscale
      - TS_EXTRA_ARGS=--advertise-tags=tag:container
      - TS_USERSPACE=false
    volumes:
      - ts-state:/var/lib/tailscale
    devices:
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - net_admin
    restart: unless-stopped

  dev-python:
    build: ./containers/python
    network_mode: service:ts-devenv
    depends_on:
      - ts-devenv
```

If you go this route, use **OAuth client secrets** (format: `tskey-client-XXXXX?ephemeral=false`) rather than auth keys — they never expire, while auth keys cap at 90 days. Tags are required with OAuth (e.g., `tag:container`), and tagged nodes automatically have key expiry disabled.

---

## Per-container Tailscale is the heaviest option

Installing Tailscale directly inside each application container (no sidecar) gives each container its own tailnet identity but requires the most work: modifying every Dockerfile, managing `tailscaled` in each container alongside your application process, granting `NET_ADMIN` capability and `/dev/net/tun` access, and persisting Tailscale state via volumes. Each container becomes a separate tagged resource on your plan (**5–8 out of 50 allowed** on the free plan). The approach makes sense for containers that need full Tailscale features (exit node, subnet routing from inside the container), but for SSH-only access, it's unnecessarily heavy.

---

## Troubleshooting the subnet router setup

The most common issues and their fixes:

- **Ping works but SSH fails**: Check that `sshd` is actually running inside the container (`docker exec dev-python ps aux | grep sshd`) and that the authorized_keys file has correct permissions (700 on `.ssh`, 600 on `authorized_keys`, owned by the target user).

- **No connectivity to container IPs from MacBook**: Verify IP forwarding is active on the server (`sysctl net.ipv4.ip_forward` should return `1`), confirm the route is approved in the admin console, and check whether the host firewall blocks forwarded traffic (`sudo iptables -L FORWARD -n -v`). If using `ufw`, add `sudo ufw route allow from any to 172.20.0.0/24`.

- **IP address conflicts**: If your home network or another VPN uses addresses in the 172.20.x.x range, containers will be unreachable. Check with `netstat -rn | grep 172.20` on the MacBook before configuring. Switch to an unused range like `10.99.0.0/24` if conflicts exist.

- **Container IPs change after restart**: This only happens with Docker's default bridge or if you don't assign static `ipv4_address` values. Always use a custom network with explicit IPAM configuration.

- **"Host key changed" warnings**: Rebuilding a container generates new SSH host keys. Either use `StrictHostKeyChecking no` in your SSH config (acceptable for dev containers) or mount persistent host keys as a volume.

- **VS Code connection timeouts**: Add `"remote.SSH.connectTimeout": 30` to VS Code settings. Verify the Tailscale tunnel is direct (not relayed) with `tailscale ping <server-ip>` — DERP-relayed connections add latency.

---

## Conclusion

**The subnet router is the right call for this setup.** It adds zero extra Tailscale devices, requires no per-container Tailscale configuration, and since VS Code forces you to run `sshd` in containers regardless of approach, the sidecar's main advantage evaporates. The total configuration is: one `tailscale set --advertise-routes` command on the server, one route approval in the admin console, static IPs in your Docker Compose, `sshd` in each container's Dockerfile, and SSH config entries on your MacBook. From there, VS Code Remote SSH connects to each container as if it were a regular remote machine.

The one scenario where you should reconsider is if you later need **per-container ACL policies** or **automatic HTTPS certificates** for web services inside containers — those require each container to have its own Tailscale identity, making the sidecar pattern worthwhile despite the added complexity. For pure SSH-based development, the subnet router is simpler, leaner, and fully sufficient.