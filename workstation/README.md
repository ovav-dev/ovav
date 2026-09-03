# OVAV Alacritty Workstation 2026

> Cohesive workstation terminal where Windows, panes, Fish, OVAV, OpenCode, ACP,
> history, and context form a single operational flow.

**Mission:** govern the active Alacritty → WSL2 → tmux → OpenCode workstation
for OVAV. Warp, Windows Terminal, and Intelligent Terminal are historical artifacts,
not active deployment targets.

---

## Quickstart

```bash
# From OVAV repo root:
bash workstation/scripts/install.sh        # Idempotent installer
bash workstation/tests/test-e2e.sh        # Verify installation
bash workstation/scripts/benchmark.sh     # Measure performance
bash workstation/scripts/rollback.sh      # Restore from latest backup
```

The installer:

1. Backs up existing configs to `~/.ovav-backups/<timestamp>/`
2. Appends OVAV block to `~/.bashrc` (idempotent)
3. Installs `~/.config/starship.toml` with OVAV Night/Day palettes
4. Installs `~/.config/atuin/config.toml`
5. Installs `~/.config/opencode/tui.json` + 2 OVAV themes
6. Installs the Alacritty Shift+Enter CSI-u bridge
7. Skips inactive Warp/Windows Terminal/Intelligent Terminal surfaces
8. Installs PowerShell profile with PSReadLine Predictive IntelliSense

---

## Architecture at a Glance

```
USER
 ↓
 Alacritty 0.17.0  (Windows host terminal)
 ↓
WSL2 Ubuntu 26.04  (Linux runtime)
 ↓
 tmux 3.6  (multiplexer — session main)
 ↓
  Fish 4.2.1  (shell — each new Alacritty window gets isolated tmux)
 ↓
OVAV runtime  (Go CLI — governor layer)
 ↓
OpenCode  (agent runtime — ACP backend)
 ↓
OpenAI / MiniMax  (models)
 ↓
Atuin + fzf + zoxide + Starship  (auxiliary infrastructure)
 ↓
history / context / observability  (feedback to terminal)
```

Full architecture details: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

### New-window isolation

Fish must not auto-attach a new Alacritty window to tmux session `main`.
Instead, each window receives its own `alacritty-<fish-pid>` tmux session.
The installer removes only the legacy, exact auto-attach block from the user
Fish config, creates a timestamped backup first, and validates Fish syntax.
The existing `main` session is never terminated or reconfigured by this step.

---

## What's Included

| Component | Purpose | Config file |
|-----------|---------|-------------|
| OVAV bashrc | PATH, Atuin, zoxide, fzf, Starship, OpenCode, aliases | `configs/bashrc/ovav.bashrc` |
| Starship | Premium minimal prompt (Night/Day palettes) | `configs/starship/starship.toml` |
| Atuin | History DB, search, context (NO pty-proxy) | `configs/atuin/config.toml` |
| OpenCode TUI | theme=system, ACP backend, FAST/DEEP models | `configs/opencode/tui.json` |
| OVAV Night theme | Dark ANSI palette for OpenCode | `configs/opencode/themes/ovav-night.json` |
| OVAV Day theme | Light ANSI palette for OpenCode | `configs/opencode/themes/ovav-day.json` |
| Alacritty | Shift+Enter bridge, font, clipboard | `configs/alacritty/keybindings.toml` |
| tmux | Extended keys, clipboard, mouse policy | `configs/tmux/tmux.conf` |
| PowerShell | PSReadLine Predictive IntelliSense | `configs/powershell/Microsoft.PowerShell_profile.ps1` |

---

## Market Decisions (TL;DR)

| Capability | Winner | Reason |
|---|---|---|
| History | **Atuin** | Unique SQLite + E2E + context (cwd, exit, duration). v18.19.0 current. |
| Fuzzy finder | **fzf** | Standard. Atuin+fzf+bash-completion > inshellisense. |
| Navigation | **zoxide** | Rust, mature, sponsors active. |
| Prompt | **Starship** | Reference, low overhead, truecolor. |
| Readline replacement | **NONE** | ble.sh = 7 años sin release, conflictos con todo. |
| Inline completion | **Atuin+fzf+bash-completion** | inshellisense duplica y es frágil. |
| PowerShell line editor | **PSReadLine** | Built-in pwsh 7+. Predictive IntelliSense maduro. |
| AI agent | **OpenCode** | 197k★, ACP production-ready, theme=system. |
| Terminal host | **Alacritty** | Confirmed live process: `C:\Program Files\Alacritty\alacritty.exe`. |

Full decisions with evidence: [`docs/DECISIONS.md`](docs/DECISIONS.md).

---

## What Is NOT Installed (and Why)

- ❌ **ble.sh** — Mantenedor único, 7 años sin release, conflictos documentados con fzf
  y Atuin. Overhead ~150-300ms sin justificación.
- ❌ **inshellisense** — Duplica Atuin+fzf, restricción "último en .bashrc" es frágil.
- ❌ **Atuin pty-proxy** — Lanzado en v18.19.0 (2026-08-03), <30 días. Riesgoso.
- ❌ **Atuin AI** — "Free during testing" = cambiar a paid sin aviso.
- ❌ **Warp / Windows Terminal / Intelligent Terminal** — no están instalados en el
  entorno actual; sus archivos permanecen como histórico no operativo.
- ❌ **WezTerm / Ghostty / Zellij** — no están instalados ni son targets activos.

---

## Keybindings

See [`docs/CHEATSHEET.md`](docs/CHEATSHEET.md) for the full cheatsheet.

### Active tmux workspace routes

| Shortcut | Workspace | Path |
|---|---|---|
| `Alt+1` | HOME | `/home/braka` |
| `Alt+2` | OVAV | `/home/braka/Systems/ovav` |
| `Alt+3` | AKRYNT | `/home/braka/Systems/projects/work/akrynt-agent` |

Routes target stable tmux window names, not mutable window numbers.

---

## Remaining Limitations

1. **OpenCode TUI requiere reinicio** tras cambiar `~/.config/opencode/tui.json`.
2. **Atuin pty-proxy NO activado** — desactivado por madurez <30d. Esperar v18.21+.
3. **Atuin sync NO activado** — espera credenciales Atuin Cloud en vault OVAV.
4. **OpenCode es single-binary** — si OpenCode cambia la
   API en el futuro, requerir reaplicar tui.json.
5. **Alacritty config path** — actualmente `%APPDATA%\alacritty\alacritty.toml`
   para el usuario Windows `Alexa`.

---

## File Layout

```
workstation/
├── configs/
│   ├── bashrc/ovav.bashrc               # Bash runtime additions
│   ├── starship/starship.toml           # Prompt + dual palette
│   ├── atuin/config.toml                # History DB settings
│   ├── opencode/
│   │   ├── tui.json                     # TUI config (theme=system)
│   │   └── themes/
│   │       ├── ovav-night.json          # Dark ANSI palette
│   │       └── ovav-day.json            # Light ANSI palette
│   ├── alacritty/
│   │   └── keybindings.toml             # Active host bridge
│   ├── tmux/
│   │   └── tmux.conf                    # Active multiplexer config
│   └── powershell/
│       └── Microsoft.PowerShell_profile.ps1
├── scripts/
│   ├── install.sh                       # Idempotent installer
│   ├── rollback.sh                      # Restore from latest backup
│   └── benchmark.sh                     # Performance before/after
├── tests/
│   └── test-e2e.sh                      # End-to-end smoke test
├── docs/
│   ├── ARCHITECTURE.md                  # Full architecture diagram
│   ├── CHEATSHEET.md                    # Keybinds map
│   ├── DECISIONS.md                     # Market decisions + evidence
│   └── README.md
└── README.md                            # This file
```
