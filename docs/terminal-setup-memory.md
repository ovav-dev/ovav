# OVAV Terminal Configuration Memory

> **Canonical Reference** — This document is the source of truth for OVAV terminal setup.
> All agents must consult this before modifying terminal configurations.
> Last updated: 2026-08-09

## Environment Detection & Config Paths

### WezTerm (Windows + WSL2)

| Environment | Config Path |
|-------------|-------------|
| Windows WSL2 | `C:\Users\<username>\.wezterm.lua` → WSL: `/mnt/c/Users/<username>/.wezterm.lua` |
| Windows Native | `%USERPROFILE%\.wezterm.lua` |
| WSL2 native | `~/.wezterm.lua` (WSL home) |

**OVAV Config Source**: `.ovav/source/configs/wezterm/config.lua`
**Deploy**: `config/wezterm/wezterm.lua` (template version)

**User Variables** (top of config.lua):
```lua
local USER = {
  wsl_username = 'braka',
  wsl_distro = 'Ubuntu-24.04',
  wsl_domain_label = 'WSL:Ubuntu-24.04',
  paths = {
    home  = '/home/braka',
    system = '/home/braka/.config',
    ovav  = '/home/braka/Systems/OVAV',
  },
}
```

### Windows Terminal

| Environment | Config Path |
|-------------|-------------|
| Windows | `%LOCALAPPDATA%\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json` |
| WSL2 access | `/mnt/c/Users/<username>/AppData/Local/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json` |

**OVAV Config Source**: `.ovav/source/configs/windows-terminal/settings.json`
**Deploy**: `config/windows-terminal/settings.json`

### Fish Shell (WSL2/Linux)

| Config | Path |
|--------|------|
| Main config | `~/.config/fish/config.fish` |
| Aliases | `~/.config/fish/aliases.fish` |
| Prompt | `~/.config/fish/fish_prompt.fish` |

**OVAV Config Source**: `.ovav/source/configs/fish/`
**Deploy**: `config/fish/`

## OVAV Convert Commands

```bash
# Project configs to deploy directory
ovav convert --configs

# Inject configs to user home (auto-detects environment)
ovav convert --inject

# Force overwrite existing configs
ovav convert --inject --force

# Check sync status
ovav convert --status

# Verbose output
ovav convert -v
```

## Environment Detection Logic

```
detectEnvironment():
  if /proc/version exists and contains "microsoft" → "windows-wsl"
  else if GOOS == "windows" → "windows"
  else → "linux"
```

## Config Injection Targets by Environment

### windows-wsl
- `wezterm/wezterm.lua` → `/mnt/c/Users/Alexa/.wezterm.lua` (Windows WezTerm)
- `wezterm/wezterm.lua` → `~/.wezterm.lua` (WSL WezTerm)
- `fish/config.fish` → `~/.config/fish/config.fish`
- `commands/aliases.fish` → `~/.config/fish/aliases.fish`
- `git/gitconfig` → `~/.gitconfig`
- `windows-terminal/settings.json` → `/mnt/c/Users/Alexa/AppData/Local/Packages/.../settings.json`

### windows
- `wezterm/wezterm.lua` → `%USERPROFILE%\.wezterm.lua`
- `windows-terminal/settings.json` → `%LOCALAPPDATA%\Packages\...\settings.json`

### linux
- `fish/config.fish` → `~/.config/fish/config.fish`
- `git/gitconfig` → `~/.gitconfig`

## Theme System

**Canonical Source**: `.ovav/visual/theme/theme.yaml`

**Auto-detection modules**:
- `.ovav/source/configs/theme/auto.wezterm.lua` — WezTerm theme switcher
- `.ovav/source/configs/theme/auto.windows-terminal.json` — Windows Terminal schemes

**Color Variables** (dark mode default):
```lua
bg = "#242424"
fg = "#d4d4d4"
accent_green = "#7eb77f"
accent_yellow = "#d4a85c"
accent_pink = "#c47d8a"
```

## Workspace Isolation

**WezTerm**: `ALT+1/2/3` switches workspaces:
- `ALT+1` → HOME (`/home/braka`)
- `ALT+2` → SYS (`/home/braka/.config`)
- `ALT+3` → OVAV (`/home/braka/Systems/OVAV`)

**Key bindings**:
- `ALT+h/j/k/l` — Navigate panes (vim-style)
- `ALT+SHIFT+h/j/k/l` — Resize panes
- `ALT+t` — Split horizontal
- `ALT+SHIFT+D` — Split vertical

## Known Issues & Solutions

| Issue | Solution |
|-------|----------|
| Config changes not taking effect | Use `ovav convert --inject --force` to overwrite |
| First WezTerm tab = cmd.exe | Known Windows behavior; ALT+1/2/3 spawns WSL tabs correctly |
| `[I]` icon in prompt | Nerd Font glyph in shell's oh-my-posh, not WezTerm |
| Themes not switching | Set `OVAV_COLOR_SCHEME=dark` or `light` env var |

## Git Config

**OVAV Signing**: GPG key `3DAC13769287AC80`
**AI+Human Model**: Author=agent, Committer=human (gets "Verified" badge)

```bash
# OVAV commit wrapper
.ovav/ovav-commit-wrapper -- <files>
```
