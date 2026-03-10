# Phase 4: Configuration & Credential Injection

**Status:** Draft

**Goal:** Global config, proper config precedence, and credentials work inside containers.

## Tasks

- [ ] Implement global config at `~/.config/deckhand/config.yaml`
- [ ] Implement config precedence: flags → env vars → project config → global config
- [ ] SSH agent forwarding — mount `SSH_AUTH_SOCK` into containers
- [ ] Git config injection — mount `.gitconfig` read-only
- [ ] GitHub token injection — mount as secret file, set `GH_TOKEN`
- [ ] GPG agent forwarding — mount extra socket
