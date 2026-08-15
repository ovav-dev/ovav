# OVAV Workstation Keybinding Cheatsheet

> Designed for zero conflicts with Bash, Readline, fzf, Atuin, OpenCode, PSReadLine.

## Strategy

| Mod | Role |
|-----|------|
| `Ctrl+Shift+...` | Terminal-level (tabs, panes, workspace, palette) |
| `Ctrl+Alt+...` | Pane operations (split, focus, resize) |
| `Alt+...` | Pane navigation |
| `Ctrl+...` | Shell-level (readline, Atuin, fzf) |
| **No conflict** | OpenCode, Readline defaults, fzf defaults, Atuin defaults |

## Intelligent Terminal (Windows host)

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+~` | **OVAV Workspace** — opens full layout (main + agent + logs) |
| `Ctrl+Shift+T` | New OVAV Tab |
| `Ctrl+Shift+P` | Command Palette |
| `Ctrl+Shift+F` | Search (find match) |
| `Ctrl+Shift+Z` | Toggle pane zoom |
| `Ctrl+Shift+W` | Close pane |
| `Ctrl+Alt+D` | Split pane vertical |
| `Ctrl+Alt+S` | Split pane horizontal |
| `Alt+Left/Right/Up/Down` | Focus pane direction |
| `Ctrl+Shift+M` | New tab (terminal default, preserved) |

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

## OpenCode TUI (when in OpenCode tab)

| Shortcut | Action |
|----------|--------|
| `Enter` | Submit prompt |
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

## Profiles → Quick Reference

| Tab | Profile | Default cwd | Use |
|-----|---------|-------------|-----|
| Tab 1 | OVAV Ubuntu | ~/Systems/ovav | Main development |
| Tab 2 | OpenCode Ubuntu | ~/Systems/ovav | AI agent TUI |
| Tab 3 | OVAV Ubuntu | ~/Systems/ovav | OPS (git/logs/doctor) |
| Tab 4 | OVAV Scratch | ~$HOME | Free Bash |

## Workspace Action (Ctrl+Shift+~)

Opens TAB 1 with this layout:

```
┌───────────────────────────────┬──────────────────────┐
│ MAIN (Bash / OVAV)            │ AGENT PANE           │
│                               │ (OpenCode TUI)       │
├───────────────────────────────┼──────────────────────┤
│ SERVICES / LOGS               │ TEST / SCRATCH       │
└───────────────────────────────┴──────────────────────┘
```