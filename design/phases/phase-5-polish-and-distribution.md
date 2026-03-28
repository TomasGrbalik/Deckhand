# Phase 5: Polish, Feature Completion & Distribution

**Status:** Draft

**Goal:** Fill remaining feature gaps (companion services, env var overrides), add developer ergonomics (doctor command, shell completions, config versioning), polish error handling, and set up release automation. After this phase, deckhand is shippable.

---

## Design Decisions

### Companion services as Go structs, not template files

Service definitions (postgres, redis) are just image + env + healthcheck — no Dockerfile or compose template needed per service. A `CompanionRegistry` holds hardcoded definitions. The existing compose template renders them generically alongside the devcontainer. This avoids template proliferation.

### Config version as int

An integer version (1, 2, 3) is simpler to compare than semver. The config schema is internal to deckhand and doesn't need semver granularity. Adding `version: 1` now means future schema changes can be detected and migrated.

### Env var overrides via direct os.Getenv

The Phase 4 design specifies `flags → env vars → project config → global config` precedence for scalars, but the env var layer was never wired. Only two scalars need overrides (`DECKHAND_PROJECT`, `DECKHAND_TEMPLATE`), so direct `os.Getenv` after unmarshal is clearer than using koanf's env provider.

### Doctor in service layer, not CLI-only

Putting health check logic in the service layer means a future TUI dashboard can reuse the same checks. The CLI is a thin presenter.

### Linux-only releases

Deckhand is a server-side tool. GoReleaser builds target linux/amd64 and linux/arm64 only.

---

## Companion Services

### Service catalog

Two built-in services initially:

**PostgreSQL:**
```yaml
name: postgres
description: PostgreSQL 16
image: postgres:16-alpine
ports:
  - port: 5432
    name: postgres
    protocol: tcp
    internal: true
environment:
  POSTGRES_USER: dev
  POSTGRES_PASSWORD: dev
  POSTGRES_DB: devdb
healthcheck:
  test: "pg_isready -U dev"
  interval: 5s
  timeout: 3s
  retries: 5
```

**Redis:**
```yaml
name: redis
description: Redis 7
image: redis:7-alpine
ports:
  - port: 6379
    name: redis
    protocol: tcp
    internal: true
healthcheck:
  test: "redis-cli ping"
  interval: 5s
  timeout: 3s
  retries: 5
```

### How they render in compose

Selected services appear as additional service blocks in `.deckhand/docker-compose.yml`:

```yaml
services:
  devcontainer:
    # ... existing devcontainer block ...
  postgres:
    image: postgres:16-alpine
    labels:
      dev.deckhand.managed: "true"
      dev.deckhand.project: "my-api"
      dev.deckhand.service: "postgres"
    ports:
      - "127.0.0.1:5432:5432"
    environment:
      POSTGRES_USER: "dev"
      POSTGRES_PASSWORD: "dev"
      POSTGRES_DB: "devdb"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dev"]
      interval: 5s
      timeout: 3s
      retries: 5
```

### How they appear in .deckhand.yaml

```yaml
version: 1
project: my-api
template: base
services:
  - name: postgres
  - name: redis
```

The `services` field is a list of `ServiceConfig` entries (name, optional version, optional `enabled: false` to disable).

### Init flow

After variable editing and before project name prompt, `init` shows a `huh.NewMultiSelect` for companion services. Skipped when `--template` flag is set (non-interactive mode).

---

## Doctor Command

`deckhand doctor` validates prerequisites and prints pass/fail for each check:

```
[PASS] Docker daemon: Docker 24.0.7
[PASS] Compose V2: 2.24.5
[PASS] Global config: valid
[SKIP] Project config: no .deckhand.yaml in current directory
```

Checks in order:
1. Docker daemon reachable (ping)
2. Docker Compose V2 available (`docker compose version --short`)
3. Global config valid (attempt load, report pass/fail/skip)
4. Project config valid (attempt load if `.deckhand.yaml` exists, skip if not)
5. Template exists (if project config loaded, check template is available)

Exit code 0 if all pass, 1 if any fail.

---

## Tasks

### Task 1: Config version field

- Add `Version int` to `domain.Project` as the first field (`yaml:"version"`)
- `config.Load()`: if `Version == 0` (absent), treat as v1; if `> 1`, error with upgrade message
- `config.Save()`: set `Version = 1` if zero before marshalling
- `init.BuildProject()` sets `Version = 1`
- Tests: load without version, load with v1, load with v2 (error), BuildProject sets version

### Task 2: Env var overrides for scalar config

- `config.Load()`: after unmarshal, check `DECKHAND_PROJECT` and `DECKHAND_TEMPLATE` env vars; override if set
- Tests: `t.Setenv()` to verify overrides work, unset env vars don't affect config

### Task 3: Shell completions

- Remove `cmd.CompletionOptions.DisableDefaultCmd = true` from `root.go`
- Exposes `deckhand completion bash|zsh|fish`
- Tests: smoke test that completion produces output and exits 0

### Task 4: Companion services — domain and registry

- New `domain.CompanionService` struct (Name, Description, Image, Ports, Environment, HealthCheck, Volumes)
- New `domain.HealthCheck` struct (Test, Interval, Timeout, Retries)
- New `domain.ServiceConfig` struct (Name, Version, Enabled) for project config
- Add `Services []ServiceConfig` to `domain.Project`
- New `service.CompanionRegistry` with hardcoded postgres/redis; methods: `ListAvailable()`, `Resolve(name, version)`
- Tests: registry returns correct services, unknown name errors, validation

### Task 5: Companion services — template rendering

- Extend `templateData` with `Companions []CompanionTemplateData`
- Add `buildCompanionData()` resolving `ServiceConfig` via registry
- Update `Render()` to include companion data
- Update `compose.yaml.tmpl` (base + python) to render companion service blocks
- Tests: render with companions produces valid YAML, render without is unchanged

### Task 6: Companion services — init flow

- `InitService.ListCompanions()` delegating to registry
- Update `BuildProject()` to accept selected service names → `[]ServiceConfig`
- CLI: `huh.NewMultiSelect` after variable editing, before project name; skip in non-interactive mode
- Tests: BuildProject with/without services

### Task 7: Doctor command

- New `service.DoctorService` with `CheckResult` struct and `DockerChecker` interface
- Add `ComposeVersion()` to infra docker layer
- New `cli/doctor.go` — thin command printing `[PASS]`/`[FAIL]`/`[SKIP]`
- Wire into root command
- Tests: fake DockerChecker for all-pass, docker-fail, compose-fail, bad-config

### Task 8: Error handling polish

- Add actionable hints to common errors (Docker not running, config not found, no environment)
- New `cli/errors.go` with `humanizeError()` matching known patterns
- Audit all layer boundaries for proper error wrapping
- Tests: specific error types get correct suggestions

### Task 9: GoReleaser and CI

- New `.goreleaser.yaml` — linux amd64+arm64, CGO_ENABLED=0, version via ldflags
- Add build verification job to CI (matrix: linux/amd64, linux/arm64)
- New `.github/workflows/release.yml` — tag-triggered, goreleaser-action, GitHub Releases

### Task 10: README update

- Update install section with binary download from GitHub Releases
- Document all commands including status, list, port, connect, template list, doctor, completion
- Document global config, mounts, credential recipes, companion services
- Document env var overrides
- Update "Status" section
