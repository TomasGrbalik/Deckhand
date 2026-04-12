# Networking

Deckhand supports an opt-in shared Docker network that assigns static IPs to devcontainers. This enables direct SSH access from your client machine via Tailscale subnet routing, without port juggling.

## Overview

When configured, each devcontainer gets a predictable IP address on a shared bridge network. You can then SSH directly to any container by IP on port 22, instead of mapping unique host ports per project.

**Architecture:**

```
MacBook → Tailscale → Remote Server → Docker bridge (172.30.0.0/24) → Containers
                                        ├── project-a  172.30.0.10
                                        ├── project-b  172.30.0.11
                                        └── project-c  172.30.0.12
```

## Setup

### 1. Create the Docker network

Deckhand does not create the network automatically — you create it once:

```bash
docker network create --driver=bridge --subnet=172.30.0.0/24 --gateway=172.30.0.1 ssh-net
```

### 2. Configure global config

Add a `network` block to `~/.config/deckhand/config.yaml`:

```yaml
network:
  name: ssh-net
  subnet: 172.30.0.0/24
  gateway: 172.30.0.1
```

All three fields are required. If any field is missing, networking is disabled and Deckhand behaves as before.

### 3. Run `deckhand up`

```bash
cd ~/my-project
deckhand up
```

Deckhand will:
- Verify the Docker network exists (fail with a helpful error if not)
- Assign the next free IP starting at `.10` (e.g., `172.30.0.10`)
- Record the assignment in `~/.config/deckhand/network-state.yaml`
- Generate a compose file with the devcontainer dual-homed on both `default` and the shared network

### 4. Verify

```bash
deckhand list
```

When networking is configured, the output includes an IP column:

```
PROJECT    STATUS    IP             SERVICES        UPTIME
my-api     running   172.30.0.10   devcontainer    2h 15m
frontend   running   172.30.0.11   devcontainer    1h 30m
```

## Tailscale Subnet Routing

To reach the Docker network from your client machine, advertise the subnet via Tailscale on the remote server:

```bash
# On the remote server
sudo tailscale up --advertise-routes=172.30.0.0/24
```

Then approve the route in the Tailscale admin console (or via `tailscale set --accept-routes` on the client).

Once the route is active, you can SSH directly:

```bash
ssh dev@172.30.0.10
```

## VS Code Remote SSH

Add entries to your local `~/.ssh/config`:

```
Host devcontainer-my-api
    HostName 172.30.0.10
    User dev
    Port 22

Host devcontainer-frontend
    HostName 172.30.0.11
    User dev
    Port 22
```

Then in VS Code: **Remote-SSH: Connect to Host** → select `devcontainer-my-api`.

## How It Works

### IP allocation

- IPs are allocated sequentially starting at `.10` in the configured subnet
- The first 9 addresses (`.1`–`.9`) are reserved for the gateway and manual assignments
- The broadcast address is excluded from allocation
- Each project gets one IP, keyed by project name
- If a project already has an IP assigned, it is reused on subsequent `deckhand up` runs

### State file

Assignments are stored in `~/.config/deckhand/network-state.yaml`:

```yaml
assignments:
  my-api: 172.30.0.10
  frontend: 172.30.0.11
```

### Compose output

The generated `docker-compose.yml` includes network configuration:

```yaml
services:
  devcontainer:
    # ... build, labels, volumes, etc.
    networks:
      default: {}
      ssh-net:
        ipv4_address: 172.30.0.10
    command: sleep infinity

  postgres:
    # ... image, ports, etc.
    networks:
      default: {}

networks:
  ssh-net:
    external: true
```

- The **devcontainer** is dual-homed: it's on both `default` (for companion communication) and the shared network (for SSH access)
- **Companion services** (postgres, redis, etc.) stay on `default` only — they don't need external access

### Lifecycle

| Command | Network behavior |
|---------|-----------------|
| `up` | Assigns IP (or reuses existing), verifies network exists |
| `down` | No change — IP remains assigned |
| `destroy` | Frees the IP so it can be reused by another project |
| `list` | Shows IP column when networking is configured |
| `doctor` | Checks if the Docker network exists |

## Troubleshooting

### `network "ssh-net" does not exist`

The Docker network hasn't been created yet. Run the `docker network create` command shown in the error message.

### `deckhand doctor` shows network check failing

Same as above — create the network. `doctor` shows the exact command:

```
[FAIL] Docker network: network "ssh-net" not found — create it with:
  docker network create --driver=bridge --subnet=172.30.0.0/24 --gateway=172.30.0.1 ssh-net
```

### IP not showing in `deckhand list`

- Check that `~/.config/deckhand/config.yaml` has all three network fields (`name`, `subnet`, `gateway`)
- Run `deckhand up` to trigger IP allocation

### Can't SSH to container IP

1. Verify the Tailscale subnet route is advertised and approved
2. Verify the container has an SSH server running (template-dependent)
3. Check `docker inspect <container>` to confirm the IP assignment
4. Test connectivity: `ping 172.30.0.10` from the client

## Opt-in behavior

Networking is entirely opt-in. If the global config has no `network` block (or any field is missing), Deckhand behaves exactly as before — no shared network, no IP column in `list`, no network check in `doctor`.
