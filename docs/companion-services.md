# Companion Services

Companion services run alongside your devcontainer — databases, caches, and other dependencies your project needs.

## Selecting services

During `deckhand init`, you're prompted to select companion services. They're saved to `.deckhand.yaml`:

```yaml
services:
  - name: postgres
    enabled: true
  - name: redis
    enabled: true
```

To disable a service later, set `enabled: false` and run `deckhand up` to recreate the environment.

## Available services

### PostgreSQL

| | |
|---|---|
| Image | `postgres:16-alpine` |
| Port | 5432 |
| Health check | `pg_isready -U dev` |
| Volume | `postgres-data:/var/lib/postgresql/data` |

Default environment variables:

| Variable | Value |
|----------|-------|
| `POSTGRES_USER` | `dev` |
| `POSTGRES_PASSWORD` | `dev` |
| `POSTGRES_DB` | `devdb` |

Connect from your devcontainer:

```bash
psql -h postgres -U dev devdb
```

### Redis

| | |
|---|---|
| Image | `redis:7-alpine` |
| Port | 6379 |
| Health check | `redis-cli ping` |
| Volume | `redis-data:/data` |

Connect from your devcontainer:

```bash
redis-cli -h redis
```

## How they work

When `deckhand up` renders the docker-compose.yml, enabled companion services are added as separate services in the same Compose project. They:

- Share a Docker network with the devcontainer, so you can reach them by service name (e.g. `postgres`, `redis`)
- Have health checks configured so the devcontainer can wait for them to be ready
- Use named Docker volumes for data persistence across restarts
- Bind ports to `127.0.0.1` only (same as the devcontainer)

## Accessing companion service ports

Companion service ports are automatically added as internal ports. To expose them for SSH tunneling, add them to your `ports` config:

```yaml
ports:
  - port: 5432
    name: postgres
    protocol: tcp
```

Then use `deckhand connect` to get the tunnel command.
