# OVAV Workstation Architecture 2026

## High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ USER                                                            │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ Intelligent Terminal 0.2.2192 (Windows host)                    │
│  - Mica background                                              │
│  - Tab row + status bar                                         │
│  - Agent Pane (Ctrl+Shift+.)                                    │
│  - Command Palette (Ctrl+Shift+P)                               │
└────────────────┬────────────────────────────────────────────────┘
                 │
       ┌─────────┴──────────┐
       ▼                    ▼
┌─────────────────┐  ┌──────────────────────┐
│ Bash Tab (OVAV) │  │ OpenCode Tab (ACP)   │
│ Ubuntu 26.04    │  │ Ubuntu 26.04         │
│ cwd: ~/ovav     │  │ cwd: ~/ovav          │
└────────┬────────┘  └──────────┬───────────┘
         │                      │
         │       ┌──────────────┘
         ▼       ▼
┌─────────────────────────────────────────────────────────────────┐
│ OVAV Runtime (Go 2.0.0 — CAPA 9, v3.4.0)                        │
│  - CLI governance                                               │
│  - F0-F5 validators                                             │
│  - Memory bridge                                                │
│  - Worktree system                                              │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ OpenCode 1.18.18 (single-binary)                                │
│  - ACP backend (opencode acp --stdio)                           │
│  - Model: openai/gpt-5 (FAST), openai/gpt-5 (DEEP)              │
│  - MCP support                                                  │
│  - theme=system (follows terminal)                              │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ Models / Tools                                                  │
│  - OpenAI                                                       │
│  - MiniMax                                                      │
│  - Bash tool, file tool, web tool                               │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│ Result + history → Atuin DB → terminal display                  │
└─────────────────────────────────────────────────────────────────┘
```

## Bash Tab Layout (Rule #6)

```
TAB 1 — OVAV DEV
┌───────────────────────────────┬──────────────────────┐
│ MAIN (Bash / OVAV)            │ AGENT PANE (OpenCode)│
│                               │                      │
├───────────────────────────────┼──────────────────────┤
│ SERVICES / LOGS               │ TEST / SCRATCH       │
│                               │                      │
└───────────────────────────────┴──────────────────────┘

TAB 2 — OpenCode (TUI native, cwd=~/Systems/ovav)
TAB 3 — OPS (git, logs, ovav doctor)
TAB 4 — Scratch (general Bash)
```

## Command Layering (Rule #41)

USER → INTELLIGENT TERMINAL → BASH/WSL → OVAV → OPENCODE → MODELS/TOOLS

Atuin, fzf, zoxide, Starship = auxiliary infrastructure (NOT governors of OVAV).

## Ownership Map (Rule #18)

| Key | Owner | Why |
|---|---|---|
| `Ctrl-R` | **Atuin** | history search with context (cwd, exit, duration) |
| `Ctrl-T` | **fzf** | fuzzy file picker |
| `Alt-C` | **fzf** | fuzzy directory picker |
| Up Arrow | **Atuin** (--disable-up-arrow via shell) | history search |
| Tab | **Bash-completion + fzf** | completion menu |
| `cd` | **bash builtin** | zoxide augments, never replaces |

## Theme Sync (Rule #22)

```
Windows app theme
   ↓
Intelligent Terminal theme = "system"
   ↓
   ├─ Dark  → OVAV Night  colorScheme  →  ANSI Night  →  OpenCode dark theme
   └─ Light → OVAV Day    colorScheme  →  ANSI Day    →  OpenCode light theme
```

The `theme=system` setting in `~/.config/opencode/tui.json` makes OpenCode TUI
follow the terminal background automatically. Manual switching available via
`:theme` command in OpenCode.

## AI Workflow (Rule #10)

### A. Inline Shell UX
- Completion: bash-completion + Atuin + fzf
- History: Atuin
- Autosuggestion: Atuin `--disable-up-arrow` enabled shell history
- Syntax awareness: bash 5.x

### B. Quick AI
- Intent → command natural
- Uses Intelligent Terminal `Command Prompt AI` action
- Routes via OpenCode ACP

### C. Deep Agent
```
Command fails (exit != 0)
  ↓
Intelligent Terminal OSC133 detects exit code
  ↓
Captures context (cwd, exit, output tail)
  ↓
Routes to OpenCode ACP
  ↓
OVAV reasoning
  ↓
Diagnostic + fix suggestion
  ↓
User confirms
  ↓
Execution
  ↓
Atuin logs new command
```

## File Backups (Rule #39)

```
~/.ovav-backups/<timestamp>/
├── bashrc.bak
├── starship.toml.bak
├── atuin-config.toml.bak
├── intel-terminal-settings.json.bak
├── intel-terminal-state.json.bak
└── powershell-profile.ps1.bak
```

Per-layer rollback available via `workstation/scripts/rollback.sh`.