# WezTerm Config — OVAV Canonical Source

## Metadata

| Field | Value |
|-------|-------|
| version | 2.0.0 |
| tool | wezterm |
| platform | windows-wsl2 |
| min_version | 20260805 |
| user_variables | wsl_username, wsl_distro, wsl_domain_label, paths.home, paths.system, paths.ovav |

## User Customization Section

Edit these variables in `config.lua` at the top:

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

## Projector

This config is synced to:
- `config/wezterm/wezterm.lua` (deploy target)

## Dependencies

- WezTerm nightly 20260805+
- Nerd Font (Cascadia Code installed on Windows)
- WSL2 with Ubuntu-24.04
- fish shell

## Key Features

- Workspace isolation (ALT+1/2/3)
- WSL2 domain configuration
- OVAV color palette (eye-friendly, bajo contraste)
- Tab bar con acentos OVAV
- Status bar minimal
