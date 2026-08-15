# OVAV Intelligent Terminal Workstation 2026

> Cohesive workstation terminal where Tabs, Panes, Bash, OVAV, OpenCode, ACP,
> history, and context form a single operational flow.

**Mission:** convert Intelligent Terminal into the primary development workstation
for OVAV, leveraging 2026 capabilities without overengineering.

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
6. Surgically merges OVAV profiles/schemes/actions into
   Intelligent Terminal's `settings.json`
7. Installs PowerShell profile with PSReadLine Predictive IntelliSense

---

## Architecture at a Glance

```
USER
 ↓
Intelligent Terminal 0.2.2192  (UI host — Agent Pane + ACP)
 ↓
WSL2 Ubuntu 26.04  (Linux runtime)
 ↓
Bash 5.x  (shell — no replacement, no Fish/Zsh)
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
| Intelligent Terminal | Profiles, color schemes, actions, keybinds | `configs/intelligent-terminal/settings-fragment.json` |
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
| Terminal host | **Intelligent Terminal** | Agent Pane + ACP first-class (experimental pero usable). |

Full decisions with evidence: [`docs/DECISIONS.md`](docs/DECISIONS.md).

---

## What Is NOT Installed (and Why)

- ❌ **ble.sh** — Mantenedor único, 7 años sin release, conflictos documentados con fzf
  y Atuin. Overhead ~150-300ms sin justificación.
- ❌ **inshellisense** — Duplica Atuin+fzf, restricción "último en .bashrc" es frágil.
- ❌ **Atuin pty-proxy** — Lanzado en v18.19.0 (2026-08-03), <30 días. Riesgoso.
- ❌ **Atuin AI** — "Free during testing" = cambiar a paid sin aviso.
- ❌ **Fish / Zsh / Warp / WezTerm / Ghostty / Zellij / tmux** — reemplazan stack;
  el CEO explícitamente los excluyó.

---

## Keybindings

See [`docs/CHEATSHEET.md`](docs/CHEATSHEET.md) for the full cheatsheet.

---

## Remaining Limitations

1. **Intelligent Terminal v0.2** = experimental, 1.7k★. Windows Terminal v1.24 es
   más estable pero no tiene Agent Pane + ACP nativos. Workaround: Intelligent Terminal
   para flujos AI, Windows Terminal como fallback.
2. **Atuin pty-proxy NO activado** — desactivado por madurez <30d. Esperar v18.21+.
3. **Atuin sync NO activado** — espera credenciales Atuin Cloud en vault OVAV.
4. **OpenCode es single-binary** — ACP backend funciona pero si OpenCode cambia la
   API en el futuro, requerir reaplicar tui.json.
5. **Intelligent Terminal package path assume usuario "Alexa"** — si cambia el
   username Windows, ajustar `INTEL_TERM_SETTINGS` en `scripts/install.sh`.
6. **Fonts** — Cascadia Mono NF requiere descarga manual desde
   [microsoft/cascadia-code](https://github.com/microsoft/cascadia-code/releases).

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
│   ├── intelligent-terminal/
│   │   └── settings-fragment.json       # Surgical merge fragment
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