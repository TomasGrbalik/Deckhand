# Phase 3: Templates & Interactive Init

**Status:** Draft

**Goal:** Templates are a first-class concept with metadata, variables, and multiple sources. `init` becomes interactive.

Optional services (postgres, redis) are deferred to a later phase.

## Design Decisions

- **Self-contained templates.** Each template is a directory with everything it needs — no inheritance or shared base layers. Simpler to understand, easier to customize.
- **Template metadata.** Each template directory contains a `metadata.yaml` describing the template and its variables with defaults.
- **Template variables.** Templates use `{{ .Vars.key }}` in their `.tmpl` files. Every variable has a default value in metadata so templates always render without user input.
- **Two template sources.** Embedded (bundled in binary, `templates/` dir) and user filesystem (`~/.config/deckhand/templates/`). User templates override bundled ones by name.
- **Init service layer.** Template discovery and config generation logic lives in the service layer. Only huh form interaction stays in the CLI layer.

## Template Directory Structure

```
templates/
  go/
    metadata.yaml
    Dockerfile.tmpl
    compose.yaml.tmpl
  node/
    metadata.yaml
    Dockerfile.tmpl
    compose.yaml.tmpl
```

### metadata.yaml

```yaml
name: go
description: Go development (Go toolchain, gopls, delve)
variables:
  go_version:
    default: "1.23"
    description: Go version
```

### Variable usage in templates

```dockerfile
FROM golang:{{ .Vars.go_version }}-bookworm
```

### Variables in .deckhand.yaml

```yaml
project: my-api
template: go
variables:
  go_version: "1.23"
```

Stored as `Variables map[string]string` in `domain.Project`.

## Template Sources & Override Order

1. User filesystem: `~/.config/deckhand/templates/<name>/`
2. Embedded: compiled into binary via `embed.FS`

If a user template has the same name as a bundled one, the user template wins. `template list` shows all available templates with their source.

## Interactive Init Flow

1. Pick template (huh select — lists all available templates)
2. Set template variables (show each with its default, user can accept or change)
3. Confirm/edit project name (default: directory name)
4. Write `.deckhand.yaml`

Skippable with flags:
- `--template go` — skip template picker, use defaults for variables
- `--project name` — skip project name prompt
- If all values provided via flags, no interactive prompts shown

## Tasks

### Template metadata & variables

- [ ] Add `metadata.yaml` to existing `base` template
- [ ] Add `Variables map[string]string` field to `domain.Project`
- [ ] Define `domain.TemplateMeta` type (name, description, variables with defaults)
- [ ] Parse `metadata.yaml` in template loading (both embedded and filesystem)
- [ ] Update `TemplateService.Render()` to merge variable defaults with project overrides and pass `.Vars` to templates
- [ ] Update config `Load`/`Save` to round-trip the `variables` field

### Language templates

- [ ] Create `go` template — Go toolchain, gopls, delve (self-contained Dockerfile + compose)
- [ ] Create `node` template — Node.js, npm, typescript-language-server
- [ ] Create `python` template — Python, pip, pyright

### Template discovery

- [ ] Implement template listing from embedded FS (read directories + metadata)
- [ ] Implement template listing from user filesystem (`~/.config/deckhand/templates/`)
- [ ] Merge both sources (user overrides embedded) in a `TemplateRegistry` or similar
- [ ] Implement `deckhand template list` command — show name, description, source

### Interactive init

- [ ] Extract init logic into service layer (list templates, build project config from selections)
- [ ] Implement huh form: template picker (select from available templates)
- [ ] Implement huh form: variable editor (show each variable with default, allow override)
- [ ] Implement huh form: project name (default: directory name)
- [ ] Wire flags (`--template`, `--project`) to skip corresponding prompts
- [ ] When all values provided via flags, skip all prompts entirely
