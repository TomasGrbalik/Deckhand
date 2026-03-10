# Phase 3: Templates & Interactive Init

**Status:** Draft

**Goal:** Multiple bundled templates, interactive project setup with prompts.

## Tasks

- [ ] Create `go` template (Go toolchain, gopls, delve)
- [ ] Create `node` template (Node.js, npm, typescript-language-server)
- [ ] Create `python` template (Python, pip, pyright)
- [ ] Implement template rendering with Go's `text/template` (variable substitution)
- [ ] Implement `deckhand template list` — show available bundled templates
- [ ] Add optional services to templates (postgres, redis) with health checks
- [ ] Make `deckhand init` interactive using huh forms (template picker, services, project name)
- [ ] Support `--template` flag to skip interactive prompt
