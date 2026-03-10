# Building a Go CLI for Docker dev containers

**Cobra + Bubbletea v2 + the Docker Go SDK form the backbone of a modern devbox tool**, with a hexagonal architecture that separates CLI commands and TUI views from shared business logic. The Go ecosystem in 2025–2026 provides mature, production-tested libraries for every layer of this tool — from interactive prompts to remote Docker control to self-updating binaries. What follows is a concrete architectural blueprint drawn from the codebases of lazydocker, DevPod, nerdctl, and the latest library releases.

---

## The right framework stack for CLI-plus-TUI

**Cobra** remains the dominant Go CLI framework with **184,000+ importers**, active maintenance via the GitHub Secure Open Source Fund, and battle-testing in the Docker CLI, GitHub CLI, and Kubernetes. Its subcommand trees, POSIX-compliant flags, and automatic shell completions (bash/zsh/fish/PowerShell) make it the clear choice over urfave/cli v3 (released, but smaller ecosystem) and Kong (~2,700 stars, minimal adoption). Critically, Cobra has the most documented integration patterns with Bubbletea.

**Bubbletea v2** shipped in March 2025 with a new ncurses-based "Cursed Renderer" that's orders of magnitude faster than v1. The import path moved to `charm.land/bubbletea/v2`. Key changes include declarative `View` fields replacing imperative commands, split key events (`KeyPressMsg`/`KeyReleaseMsg`), and Lip Gloss v2 integration with automatic color downsampling. Start with v2 for any new tool.

For **interactive prompts**, use **charmbracelet/huh v2** exclusively. It's built on Bubbletea and runs in two modes: standalone (`form.Run()` in CLI commands) or embedded as a `tea.Model` inside a full TUI. It provides Input, Select, MultiSelect, Confirm, and FilePicker components with validation, theming, and accessibility. The competing library AlecAivazis/survey was **archived in April 2024** — do not use it. For the devbox init flow (prompting for GitHub token, GPG key, git config), huh handles everything:

```go
var token, gitName, gitEmail string
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Title("GitHub Token").EchoMode(huh.EchoModePassword).Value(&token),
        huh.NewInput().Title("Git Name").Value(&gitName),
        huh.NewInput().Title("Git Email").Value(&gitEmail),
    ),
)
err := form.Run()
```

For **config management**, **koanf v2** is technically superior to Viper. Viper forcibly lowercases all keys (breaking YAML/TOML specs), pulls in a heavy dependency tree, and produces **313% larger binaries** than equivalent koanf programs. Koanf's modular provider/parser architecture means you only import what you need. It has a `posflag` provider that reads from pflag (Cobra's flag library), enabling the standard flag→config→env precedence chain. Use **YAML** as the config format — it aligns with the Docker ecosystem (docker-compose.yml, Kubernetes manifests) and your users will already be YAML-literate.

---

## Hexagonal architecture separating CLI from business logic

The critical architectural decision is placing a **service layer** between presentation (CLI/TUI) and infrastructure (Docker client, filesystem). Both Cobra commands in `cmd/` and Bubbletea models in `internal/tui/` import `internal/service/` — neither contains business logic. This pattern comes directly from lazydocker's codebase (38K+ stars), where the `Gui` struct holds a reference to `DockerCommand` and delegates all Docker operations to it.

```
devbox/
├── cmd/devbox/main.go                # Entry point, version injection
├── internal/
│   ├── cli/                          # Cobra commands
│   │   ├── root.go                   # Root command, global flags
│   │   ├── up.go                     # devbox up
│   │   ├── down.go                   # devbox down
│   │   ├── shell.go                  # devbox shell (exec into container)
│   │   ├── list.go                   # devbox list
│   │   ├── init.go                   # devbox init (uses huh prompts)
│   │   └── tui.go                    # devbox tui (launches full Bubbletea TUI)
│   ├── domain/                       # Pure Go models, no external deps
│   │   ├── container.go              # Container entity
│   │   ├── template.go              # Template definition
│   │   ├── workspace.go             # Workspace state
│   │   └── ports.go                 # Interface definitions (ContainerManager, TemplateStore)
│   ├── service/                      # Business logic (shared by CLI + TUI)
│   │   ├── container_service.go      # List, start, stop, exec orchestration
│   │   ├── template_service.go       # Template resolution, inheritance, rendering
│   │   ├── setup_service.go          # Init flow: credentials, git config
│   │   └── workspace_service.go      # Workspace lifecycle
│   ├── infra/                        # Infrastructure implementations
│   │   ├── docker/
│   │   │   ├── client.go            # Docker SDK wrapper, remote connection
│   │   │   ├── container.go         # Container CRUD, exec, stats
│   │   │   └── compose.go           # Compose file generation + orchestration
│   │   ├── template/
│   │   │   ├── local.go             # ~/.devbox/templates/ file operations
│   │   │   └── github.go            # Pull templates from GitHub repos
│   │   └── credentials/
│   │       ├── ssh.go               # SSH agent socket detection
│   │       ├── gpg.go               # GPG socket forwarding
│   │       └── github.go            # GitHub token management
│   ├── config/                       # koanf-based config loading
│   │   ├── config.go                # Config struct, defaults, loading
│   │   └── paths.go                 # XDG paths (~/.devbox/)
│   └── tui/                          # Bubbletea TUI (future)
│       ├── app.go                   # Root tea.Model
│       ├── views/                   # Dashboard, logs, setup views
│       └── styles/                  # Lip Gloss styles
├── .goreleaser.yaml
└── go.mod
```

The **Cobra + Bubbletea integration** follows three patterns used simultaneously. Plain commands (`devbox up`, `devbox down`, `devbox list`) run as standard CLI output. Interactive commands (`devbox init`) launch standalone huh forms. A dedicated `devbox tui` subcommand launches the full-screen Bubbletea dashboard. Since Bubbletea takes control of stdin/stdout when running, logging must go to a file during TUI mode, and you cannot mix `fmt.Println` with Bubbletea rendering.

---

## Docker SDK patterns for container management and exec

Use the stable **`github.com/docker/docker/client`** package (12,175+ importers). A new high-level SDK (`github.com/docker/go-sdk`) was published in October 2025 but remains at v0.1.0-alpha — too unstable for production. The stable client covers everything devbox needs.

**Connecting to a remote Docker daemon** is cleanest via the `connhelper` package from docker/cli, which uses SSH natively:

```go
helper, _ := connhelper.GetConnectionHelper("ssh://user@remote-server")
httpClient := &http.Client{
    Transport: &http.Transport{DialContext: helper.Dialer},
}
cli, _ := client.NewClientWithOpts(
    client.WithHTTPClient(httpClient),
    client.WithHost(helper.Host),
    client.WithDialContext(helper.Dialer),
    client.WithAPIVersionNegotiation(),
)
```

This requires SSH key-based auth on the remote server. Alternatively, setting `DOCKER_HOST=ssh://user@server` and using `client.FromEnv` works for simple cases.

**Implementing interactive exec** (the `devbox shell` command) is the most complex piece. The flow mirrors docker CLI's own `exec.go`: create an exec instance with TTY enabled, attach to it (hijacking the HTTP connection), put the local terminal into raw mode, handle bidirectional I/O copy, and forward terminal resize signals:

```go
execConfig := container.ExecOptions{
    AttachStdin: true, AttachStdout: true, AttachStderr: true,
    Tty: true, Cmd: []string{"bash"},
    Env: []string{"TERM=xterm-256color"}, WorkingDir: "/workspace",
}
execResp, _ := cli.ContainerExecCreate(ctx, containerID, execConfig)
attachResp, _ := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: true})

oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
defer term.Restore(int(os.Stdin.Fd()), oldState)

// Handle SIGWINCH for terminal resize (critical for nvim)
go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGWINCH)
    for range sigCh {
        w, h, _ := term.GetSize(int(os.Stdout.Fd()))
        cli.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{Width: uint(w), Height: uint(h)})
    }
}()

// Bidirectional copy — with Tty:true, output is raw (no multiplexing)
go io.Copy(attachResp.Conn, os.Stdin)
io.Copy(os.Stdout, attachResp.Reader)
```

When `Tty: true`, output is raw — use `io.Copy`. When `Tty: false`, stdout/stderr are multiplexed and require `stdcopy.StdCopy` to demux. This distinction trips up many implementations.

**For container stats**, `ContainerStatsOneShot` returns CPU, memory, and network metrics. CPU percentage requires calculating the delta between `CPUStats` and `PreCPUStats`. For a live dashboard, `ContainerStats(ctx, id, true)` streams JSON objects continuously.

---

## Docker Compose: the hybrid approach

Three options exist for Compose control from Go, and the **best strategy combines two of them**:

**compose-go** (`github.com/compose-spec/compose-go/v2`) is the reference library for parsing and generating docker-compose.yml files. It provides full `types.Project` structs that can be built programmatically and marshaled to YAML. Use this for template rendering — build a `types.Project` from your template system, inject devbox-specific labels, SSH/GPG mounts, and environment variables, then call `project.MarshalYAML()`.

**The Compose SDK** (`github.com/docker/compose/v5/pkg/compose`) enables programmatic `Up`, `Down`, `Restart`, and `Logs` operations. Initialize it with a Docker CLI instance and call `compose.NewComposeService(dockerCli)`. One caveat: Docker CLI v29.0.0+ depends on `github.com/moby/moby` while Compose v5 depends on `github.com/docker/docker`, causing potential import conflicts — pin `docker/cli v28.5.2+incompatible` if needed.

**Shelling out** to `docker compose` remains the simplest fallback for up/down operations. Always pass `-p projectName` for deterministic naming and use `--format json` for structured output.

The recommended hybrid for devbox: use **compose-go to generate** YAML files from templates, the **Docker SDK directly for exec, stats, and logs** (maximum TTY control), and either the **Compose SDK or shell out for up/down** operations.

---

## What to learn from DevPod, lazydocker, and nerdctl

**DevPod** (Go, by Loft Labs) provides the most relevant credential injection patterns. It uses a provider/agent architecture where the same binary (`devpod agent`) is injected into remote environments. Its credential system is the gold standard: Git credentials flow through a helper binary injected into the container, SSH keys forward via agent socket mounting, Docker registry creds use a credential helper, and GPG keys flow through a reverse SSH tunnel forwarding the gpg-agent socket. All controlled by context-level boolean options (`SSH_INJECT_GIT_CREDENTIALS`, `GPG_AGENT_FORWARDING`, etc.). However, DevPod's development has **slowed significantly** since late 2025 as Loft Labs shifted focus to vcluster — a community fork effort has emerged.

**lazydocker** (38K+ stars) demonstrates the cleanest TUI/logic separation in a Go Docker tool. Its `pkg/commands/DockerCommand` struct wraps the Docker SDK client and provides all business operations; `pkg/gui/` only handles rendering. It uses a **hybrid SDK + shell approach** — the Docker SDK for container inspection and event streaming, but shells out to `docker compose` for Compose-specific operations. Its event-driven refresh pattern (Docker daemon event stream + throttled UI updates at 30ms intervals) is worth emulating for a live dashboard.

**nerdctl** shows excellent Cobra command organization for a Docker-compatible CLI. Its `pkg/api/types/container_types.go` defines a comprehensive options struct (~470 lines) that cleanly separates user-facing configuration from internal state. The OCI hook system and label-based metadata pattern are directly applicable.

Newer tools like **gomanagedocker** and **docker-tui** (both 2024–2025) use Bubbletea instead of the older gocui framework, confirming that **Bubbletea is now the standard** for new Go TUI projects. docker-tui even includes an MCP server for AI assistant integration — a forward-looking pattern.

---

## Credential injection done right

**SSH agent forwarding** requires bind-mounting the host SSH socket into the container and setting `SSH_AUTH_SOCK`. On Linux, mount `$SSH_AUTH_SOCK` directly. On macOS with Docker Desktop, use the magic path `/run/host-services/ssh-auth.sock`. At build time, BuildKit's `--ssh` flag (`RUN --mount=type=ssh`) provides ephemeral access that never persists in image layers.

**GPG agent forwarding** uses the **extra socket** (`gpgconf --list-dir agent-extra-socket`), which is specifically hardened for remote use. Mount it to the container's expected gpg-agent socket path, import the public keyring, and set `.gnupg` permissions to 700. DevPod handles this via a reverse SSH tunnel; for a simpler devbox, direct socket mounting works when host and container share a filesystem (same Linux server).

**GitHub tokens** should follow a security hierarchy: BuildKit `--mount=type=secret` for builds (never cached in layers), **mounted files at `/run/secrets/`** for runtime (preferred), and environment variables only as a last resort (visible in `docker inspect`). Retrieve the token via `gh auth token`, write it to a temp file with 0600 permissions, and bind-mount read-only. Setting `GH_TOKEN` as an env var inside the container makes the `gh` CLI work transparently.

---

## Template system with inheritance

Store templates at `~/.devbox/templates/` with each template as a directory containing a `devbox-template.yaml` manifest, a Dockerfile template, and a docker-compose template. Use Go's `text/template` with `{{ block }}` and `{{ define }}` directives for inheritance — a base template defines blocks that language-specific templates override:

```yaml
# devbox-template.yaml
id: go
name: "Go Development"
extends: base
options:
  goVersion: {type: string, default: "1.23", proposals: ["1.23", "1.22"]}
  includePostgres: {type: boolean, default: false}
```

For pulling templates from GitHub, clone with `--depth=1` (works with private repos via SSH) or download tarballs via the GitHub API. Parse `.devcontainer/devcontainer.json` if present to stay compatible with the devcontainer spec.

Use **compose-go types** to programmatically build the final docker-compose.yml rather than rendering YAML strings directly — this gives you type safety, automatic validation, and clean merging of base + override compose configurations. After template rendering, parse with compose-go, inject devbox-specific labels and mounts, then marshal back to YAML.

---

## Container labeling for lifecycle tracking

Use **reverse-DNS labels** to track devbox-managed containers:

```go
const LabelPrefix = "dev.devbox."
labels := map[string]string{
    "dev.devbox.managed":     "true",           // Primary selector
    "dev.devbox.version":     version,          // devbox version
    "dev.devbox.project":     projectName,      // Project identifier
    "dev.devbox.template":    templateID,       // Template used
    "dev.devbox.created":     time.Now().Format(time.RFC3339),
    "dev.devbox.config-hash": sha256(configYAML), // Detect config drift
}
```

Filter devbox containers with `filters.Arg("label", "dev.devbox.managed=true")`. The config hash label enables detecting when a rebuild is needed — compare the stored hash against the current template/config hash. This mirrors how Docker Compose uses `com.docker.compose.project` and `com.docker.compose.service` labels for container discovery, and how DevPod stores `dev.containers.id` for workspace tracking.

---

## Distribution via GoReleaser and self-update

**GoReleaser v2** (current as of 2025) handles cross-compilation, GitHub Releases, Homebrew Casks, Docker images, and Linux packages (`.deb`/`.rpm` via nFPM) from a single `.goreleaser.yaml`. The OSS version covers everything a CLI tool needs — Pro adds advanced changelogs, NPM publishing, and macOS notarization. Build fully static binaries with `CGO_ENABLED=0`, strip symbols with `-s -w` ldflags (25–30% size reduction), and use GoReleaser's built-in UPX integration for an additional ~70% compression.

For **self-updating**, use **`creativeprojects/go-selfupdate`** — the most actively maintained library (December 2025 release). It supports GitHub/GitLab/Gitea sources, detects ARM architectures correctly, validates SHA256 checksums against GoReleaser's `checksums.txt`, and supports ECDSA signature verification. Implement it as an optional `devbox update` command. Most major Go tools (gh CLI, Terraform) actually delegate updates to package managers rather than self-updating — self-update is best as a convenience, not the primary mechanism.

Key `go.mod` dependencies for the full stack:

- `github.com/spf13/cobra` — CLI framework
- `charm.land/bubbletea/v2` — TUI framework
- `github.com/charmbracelet/huh/v2` — Interactive prompts
- `github.com/knadh/koanf/v2` — Config management
- `github.com/docker/docker/client` — Docker SDK
- `github.com/docker/cli/cli/connhelper` — SSH remote Docker
- `github.com/compose-spec/compose-go/v2` — Compose file parsing/generation
- `golang.org/x/term` — Terminal raw mode for exec
- `github.com/creativeprojects/go-selfupdate` — Binary self-update

## Conclusion

The architecture converges on a clear pattern: **Cobra for CLI, Bubbletea v2 for TUI, and a shared service layer in `internal/service/`** that both presentation layers call. The Docker SDK handles exec and stats directly (giving full TTY control), while compose-go generates YAML files and either the Compose SDK or shell execution handles up/down orchestration. DevPod's credential injection patterns — agent socket forwarding for SSH/GPG, mounted files for tokens, context-level boolean flags — provide the security model. The most counterintuitive finding is that **koanf v2 beats Viper** on every technical metric despite Viper's far larger mindshare, and that charmbracelet/huh has completely replaced the now-archived survey library as the prompt standard. For distribution, GoReleaser v2 OSS plus `go-selfupdate` covers every platform with minimal configuration.