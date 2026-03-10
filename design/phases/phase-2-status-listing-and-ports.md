# Phase 2: Status, Listing & Port Management

**Status:** Draft

**Goal:** You can see what's running, manage ports, and generate the SSH connect command.

## Tasks

- [ ] Add container labels (`dev.deckhand.managed`, `dev.deckhand.project`, etc.)
- [ ] Implement `deckhand status` — show services, health, ports for current project
- [ ] Implement `deckhand list` — show all deckhand-managed environments on the host
- [ ] Implement `deckhand port list` — show all port mappings from labels
- [ ] Implement `deckhand port add <port>` — update compose file, recreate container
- [ ] Implement `deckhand port remove <port>` — update compose file, recreate container
- [ ] Implement `deckhand connect` — generate SSH tunnel command from mapped ports
