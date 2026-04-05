# Environment Variables

Deckhand reads two environment variables that override values in `.deckhand.yaml`.

| Variable | Overrides | Description |
|----------|-----------|-------------|
| `DECKHAND_PROJECT` | `project` | Override the project name. |
| `DECKHAND_TEMPLATE` | `template` | Override the template name. |

## Usage

```bash
DECKHAND_PROJECT=ci-build DECKHAND_TEMPLATE=base deckhand up
```

This starts the environment using `ci-build` as the project name and `base` as the template, regardless of what `.deckhand.yaml` says.

## Precedence

Environment variables sit between the config file and CLI flags:

1. Template defaults
2. Global config (`~/.config/deckhand/config.yaml`)
3. Project config (`.deckhand.yaml`)
4. **Environment variables** (`DECKHAND_PROJECT`, `DECKHAND_TEMPLATE`)
5. CLI flags (`--template`, `--project`)

An empty environment variable does not override the config file value — only non-empty values take effect.

## When to use

- **CI pipelines** — use `DECKHAND_PROJECT` to give each build a unique project name so environments don't collide.
- **Testing templates** — use `DECKHAND_TEMPLATE` to try a different template without editing the config file.
