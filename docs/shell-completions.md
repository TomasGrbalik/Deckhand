# Shell Completions

Deckhand provides tab completion for commands, subcommands, and flags.

## Bash

Add to `~/.bashrc`:

```bash
eval "$(deckhand completion bash)"
```

Or install system-wide:

```bash
deckhand completion bash > /etc/bash_completion.d/deckhand
```

## Zsh

Add to `~/.zshrc`:

```bash
eval "$(deckhand completion zsh)"
```

Or install to your completions directory:

```bash
deckhand completion zsh > "${fpath[1]}/_deckhand"
```

If completions don't work, make sure `compinit` is called in your `~/.zshrc`:

```bash
autoload -Uz compinit && compinit
```

## Fish

```bash
deckhand completion fish | source
```

Or install permanently:

```bash
deckhand completion fish > ~/.config/fish/completions/deckhand.fish
```
