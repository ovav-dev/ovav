# OVAV Workstation Keybinding Cheatsheet

> Active stack: Alacritty 0.17.0 → WSL2 → tmux 3.6 → Fish 4.2.1 → OpenCode 1.18.26.

## Strategy

| Mod | Role |
|-----|------|
| `Alt+1/2/3` | Workspace-level routes |
| `Ctrl+Alt+...` | Pane operations (split, focus, resize) |
| `Alt+...` | Named workspace navigation |
| `Ctrl+...` | Shell-level (readline, Atuin, fzf) |
| **No conflict** | OpenCode, Readline defaults, fzf defaults, Atuin defaults |

## Alacritty + tmux (active host)

| Shortcut | Action |
|----------|--------|
| `Alt+1` | HOME — `/home/braka` |
| `Alt+2` | OVAV — `/home/braka/Systems/ovav` |
| `Alt+3` | AKRYNT — `/home/braka/Systems/projects/work/akrynt-agent` |
| `Alt+A`, `n/p` | Next/previous tmux window |
| `Ctrl+Shift+C` | Copy terminal selection |
| `Ctrl+V` | Paste clipboard (host terminal) |

## Bash / Readline

| Shortcut | Action |
|----------|--------|
| `Ctrl+R` | Atuin fuzzy history search |
| `Ctrl+T` | fzf file picker |
| `Alt+C` | fzf directory picker |
| `Tab` | Bash completion |
| `Ctrl+A/E` | Beginning/end of line (Readline default) |
| `Ctrl+U/K` | Kill to start/end (Readline default) |
| `Ctrl+L` | Clear screen (Readline default) |

## Atuin (history)

| Shortcut | Action |
|----------|--------|
| `Ctrl+R` | Open Atuin search UI |
| `↑ / ↓` | Navigate results |
| `Enter` | Run selected command |
| `Tab` | Edit selected command (without running) |
| `Esc` | Cancel |
| `Ctrl+Y` | Copy command to clipboard |
| `Ctrl+O` | Filter by session/directory |

## fzf (fuzzy)

| Shortcut | Action |
|----------|--------|
| `Ctrl+T` | Fuzzy file picker (paste path) |
| `Alt+C` | Fuzzy directory picker (cd) |
| `** + Tab` | File completion under cursor |
| Custom alias | `fzf-git-branch`, `fzf-git-log` (per-script) |

## zoxide

| Shortcut | Action |
|----------|--------|
| `z <partial>` | Smart jump (highest-ranked directory match) |
| `zi <partial>` | Interactive (uses fzf) |
| `z -` | Jump to previous directory |

## Starship (prompt)

No keybindings — visual only. Renders contextual modules on each line.

## OpenCode TUI (when in the `ovav` tmux window)

| Shortcut | Action |
|----------|--------|
| `Enter` | Submit prompt |
| `Shift+Enter` / `Ctrl+J` | Insert newline |
| `Ctrl+X`, `N` | New OpenCode session |
| `Ctrl+X`, `L` | Open session list |
| `Ctrl+X`, `G` | Session timeline |
| `Ctrl+X`, `B` | Toggle sidebar |
| `Ctrl+X`, `Y` | Copy message |
| `Shift+Tab` | Cycle agent mode |
| `Ctrl+P` | Command palette (OpenCode internal) |
| `Ctrl+C` | Cancel current operation |
| `Ctrl+D` | Exit |
| `↑ / ↓` | Navigate history |
| `Ctrl+R` | OpenCode internal search (NOT Atuin) |

> Note: when in OpenCode TUI, keybinds are owned by OpenCode, not the terminal.

## PSReadLine (PowerShell 7+)

| Shortcut | Action |
|----------|--------|
| `→ / ←` | Accept inline prediction (right) / dismiss (left) |
| `Ctrl+R` | Reverse search history |
| `Tab` | Menu complete |
| `Ctrl+Space` | Accept current suggestion |
| `F2` | Toggle inline/list view |

## Active workspace routes

| Route | tmux window | Default cwd | Use |
|-----|---------|-------------|-----|
| HOME | `home` | `/home/braka` | Shell / scratch |
| OVAV | `ovav` | `/home/braka/Systems/ovav` | OpenCode + runtime |
| AKRYNT | `akrynt` | `/home/braka/Systems/projects/work/akrynt-agent` | Project work |

## Historical workspace action (inactive)

Opens TAB 1 with this layout:

```
┌───────────────────────────────┬──────────────────────┐
│ MAIN (Bash / OVAV)            │ AGENT PANE           │
│                               │ (OpenCode TUI)       │
├───────────────────────────────┼──────────────────────┤
│ SERVICES / LOGS               │ TEST / SCRATCH       │
└───────────────────────────────┴──────────────────────┘
```
