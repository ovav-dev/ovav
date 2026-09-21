# OVAV Terminal Configuration Memory

> **Canonical Reference** — This document is the source of truth for OVAV terminal setup.
> All agents must consult this before modifying terminal configurations.
> Last updated: 2026-08-13

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

**OVAV Config Source**: `.ovav/source/configs/windows-terminal/ovav.fragment.json`
**Deploy**: `config/windows-terminal/ovav.fragment.json` (merge fragment, never a full `settings.json` replacement)

### Safe Windows Terminal Workflow

Windows Terminal configuration is merge-only. Never copy an OVAV file over the installed `settings.json`, and never use `ovav convert --inject` for this target.

1. Open Windows Terminal once so its installed `settings.json` exists.
2. Run a dry-run plan. This parses the installed settings and fragment, validates both against OVAV's Windows Terminal 1.24 structural subset, merges named schemes/themes/profiles/actions, validates the merged result, and reports a UTC timestamped backup path.
3. Review the merged projection and confirm unrelated settings, actions, profiles, schemes, and themes remain present.
4. Apply only through a separately approved installer that first copies the installed file to the exact timestamped backup path, writes the planned merge, runs PowerShell `Test-Json`, reruns OVAV structural validation, and restores the backup on failure.

```powershell
# Canonical no-write planner
ovav terminal windows plan `
  --settings "$env:LOCALAPPDATA\Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json" `
  --fragment ".ovav\source\configs\windows-terminal\ovav.fragment.json"

# Obsolete compatibility wrapper: also dry-run only; -Apply is rejected
.\.ovav\templates\windows-terminal\merge-wt-settings.ps1
```

The repository does not claim or bundle the complete Microsoft vendor schema. The planner enforces the WT 1.24 structural subset OVAV uses: object/array types, required names, paired light/dark scheme and UI-theme references, profile GUIDs and command types. The installed Windows Terminal remains the final vendor-schema authority.

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

# Inject supported configs to user home (Windows Terminal is excluded)
ovav convert --inject

# Force supported non-Windows-Terminal replacements
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
- `windows-terminal/ovav.fragment.json` → dry-run merge plan only; no home write

### windows
- `wezterm/wezterm.lua` → `%USERPROFILE%\.wezterm.lua`
- `windows-terminal/ovav.fragment.json` → dry-run merge plan only; no home write

### linux
- `fish/config.fish` → `~/.config/fish/config.fish`
- `git/gitconfig` → `~/.gitconfig`

## Theme System

**Canonical Source**: `.ovav/visual/theme/theme.yaml`

**Auto-detection modules**:
- `.ovav/source/configs/theme/auto.wezterm.lua` — WezTerm theme switcher
- `.ovav/source/configs/windows-terminal/ovav.fragment.json` — paired Windows Terminal schemes and UI themes

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
| Windows Terminal changes not taking effect | Re-run the dry-run merge plan; never overwrite installed `settings.json` |
| First WezTerm tab = cmd.exe | Known Windows behavior; ALT+1/2/3 spawns WSL tabs correctly |
| `[I]` icon in prompt | Nerd Font glyph in shell's oh-my-posh, not WezTerm |
| Themes not switching | Set `OVAV_COLOR_SCHEME=dark` or `light` env var |

## Git Config

**OVAV Signing**: GPG primary `1D70BE0236928C49921A781F5F384C5B35CDD0F8`, signing subkey `7DE5923582A84DBB` (rotated 2026-08-13 after laptop reformat). Original RSA-4096 `3DAC13769287AC80` is no longer in `~/.gnupg`.
**AI+Human Model**: Author=agent, Committer=human (gets "Verified" badge)

```bash
# OVAV commit wrapper
.ovav/ovav-commit-wrapper -- <files>
```
