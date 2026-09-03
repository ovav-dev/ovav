# Fish Config — OVAV Canonical Source

## Metadata

| Field | Value |
|-------|-------|
| version | 1.0.0 |
| tool | fish |
| platform | wsl2-linux |

## User Customization Section

Edit these variables at the top of `config.fish`:

```fish
set -gx OVAV_USER "braka"
set -gx OVAV_HOME "$HOME"
set -gx OVAV_SYS "$HOME/.config"
set -gx OVAV_ROOT "$HOME/Systems/OVAV"
```

## Deploy Target

- `config/fish/config.fish` → deploys to `~/.config/fish/config.fish` (WSL)
- `config/fish/05-ovav-tmux-session.fish` → deploys to `~/.config/fish/conf.d/`

Each new Alacritty window starts an isolated tmux session named
`alacritty-<fish-pid>`. It never attaches to the shared `main` session.
