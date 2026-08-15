# OVAV Workstation Market Decisions 2026

> All decisions backed by official documentation and repositories (URLs at bottom).

## Decision Matrix

| Capability | Candidates | Winner | Evidence |
|---|---|---|---|
| **History DB** | Atuin, custom file, bash builtin | **Atuin v18.19.0** | Only SQLite + E2E + context (cwd, exit, duration, hostname). 600M+ commands synced. |
| **Fuzzy finder** | fzf, skim, brother | **fzf 0.74.2** | 82.6k★, 3,720 commits. Standard. Compatible with Atuin (Ctrl-R disable). |
| **Smart cd** | zoxide, z.lua, autojump, fasd | **zoxide** | 38.6k★, Rust, 626 commits, sponsors active (Warp, Recall.ai). |
| **Prompt** | Starship, oh-my-posh, p10k | **Starship** | 59.4k★, 4,362 commits, truecolor, Debian signed. |
| **Inline completion** | inshellisense, bash-completion, Carapace | **Atuin + fzf + bash-completion** | inshellisense = "must be last command in .bashrc" (fragile). |
| **Readline replacement** | ble.sh, zsh, fish | **NONE — keep Bash** | ble.sh stable = 7 years old (2019). Conflicts with everything. |
| **Completion library** | Carapace, bash-completion | **bash-completion** | Already installed. Carapace only valuable for Cobra devs. |
| **PowerShell** | PSReadLine | **PSReadLine** | Built-in PowerShell 7+. Predictive IntelliSense mature. |
| **AI agent** | OpenCode, Copilot CLI, Claude | **OpenCode 1.18.18** | 197k★, 16M devs/month, ACP production-ready. |
| **Terminal host** | Intelligent Terminal, Windows Terminal | **Intelligent Terminal** | v0.2 has Agent Pane + ACP first-class (OpenCode as backend). |
| **AI inline completion** | Atuin AI, inshellisense, GitHub Copilot | **OpenCode via Intelligent Terminal** | Atuin AI = "free during testing" = unstable business model. |

## Explicitly Rejected (with reason)

### ❌ ble.sh
- Stable v0.3.4 = 2019 (7 years without release)
- v0.4-devel3 = 2023-04 (incomplete)
- Single maintainer
- Overrides `trap`, `readonly`, `bind`, `history`, `read`, `exit`
- Documented conflict with fzf (Sec 2.8 of blesh docs)
- Likely conflict with Intelligent Terminal OSC133 + Atuin bash-preexec
- **Verdict**: Cost > benefit. Bash 5.x + Atuin + fzf gives better UX with zero risk.

### ❌ inshellisense
- Microsoft maintained, 318 commits
- Duplicates Atuin + fzf fundamentally
- "Must be last command in .bashrc" = fragile
- **Verdict**: Pure overengineering for OVAV.

### ❌ Atuin pty-proxy
- Launched in v18.19.0 (2026-08-03), <30 days old
- Atuin doc: "bash-preexec has limitations"
- Sits between terminal and shell = potential break point
- **Verdict**: Wait until v18.21+ stable. Enable only after OSC133 + Autofix + ACP all PASS.

### ❌ Atuin AI
- "Free during testing" = uncertain business model
- Cloud service = new vendor dependency
- Risk of switching to paid without notice
- **Verdict**: Use OpenCode as AI layer. Do not add another AI vendor.

### ❌ Atuin auto-sync
- Requires Atuin Cloud account
- New vendor relationship
- CEO has not approved subscription
- **Verdict**: Disabled by default. Enable when explicit CEO approval + secret in vault.

### ❌ Intelligent Terminal as production host
- v0.2 = experimental, 1.7k★
- Docs state: "Intelligent Terminal is in an experimental stage"
- **Verdict**: Use for OVAV development (acceptable risk). Not for production-critical workloads.

### ❌ Fish / Zsh / Warp / WezTerm / Ghostty / Zellij / tmux
- **CEO directive**: explicit exclusion in original mission
- Would replace Bash stack
- **Verdict**: Out of scope.

## Conflict Resolution (Rule #18)

### Ctrl-R Ownership

| Tool | Default behavior | OVAV decision |
|------|------------------|---------------|
| Atuin | Ctrl-R = history search | **OWN** |
| fzf | Ctrl-R = file history | **DISABLE** (`FZF_CTRL_R_COMMAND=""` implied via Atuin `init bash`) |
| Bash | Ctrl-R = reverse-search-history (Readline) | **DISABLE** (overridden by Atuin) |
| PSReadLine | Ctrl-R = history search | **OWN** (PowerShell only) |

Result: **Ctrl-R has exactly ONE owner per shell**: Atuin in Bash, PSReadLine in pwsh.

### Up Arrow

| Tool | Default | OVAV |
|------|---------|------|
| Atuin | Up arrow = history (with `--disable-up-arrow` it's shell default) | **`--disable-up-arrow`** — keep shell default, Atuin owns Ctrl-R |
| Readline | Up arrow = previous-history | OWN (default) |

### TAB Completion

| Tool | Default | OVAV |
|------|---------|------|
| Bash-completion | TAB = menu-complete | OWN (primary) |
| fzf | TAB in picker = select | OWN (in fzf UI only) |
| Atuin | TAB in picker = edit without run | OWN (in Atuin UI only) |

## Theme Decision

### Why custom OVAV palette (not Catppuccin / Tokyo Night / Rosé Pine)

- **Brand identity** — OVAV needs visual signature, not generic palette
- **Mica compatibility** — OVAV Night/Day designed to breathe over Mica
- **Semantic token coherence** — paired `fg/bg` per state (Elena's premium layer)
- **Adaptive CLI** — Starship, fzf, Atuin, OpenCode all consume same palette

## Tools Confirmed Available (audit 2026-08-13)

```
✓ Atuin 18.19.0   (/home/braka/.atuin/bin/atuin)
✓ fzf 0.74.2      (/home/braka/.local/bin/fzf)
✓ zoxide          (/home/braka/.local/bin/zoxide)
✓ Starship        (/home/braka/.local/bin/starship)
✓ mise            (/home/braka/.local/bin/mise)
✓ uv              (/home/braka/.local/bin/uv)
✓ crush           (/home/braka/.local/bin/crush)
✓ OpenCode 1.18.18 (/home/braka/.opencode/bin/opencode)
✓ OVAV 3.4.0      (/home/braka/.local/bin/ovav) — Go 2.0.0-go runtime
✓ Intelligent Terminal 0.2.2192 (Microsoft.IntelligentTerminal_8wekyb3d8bbwe)
✓ Ubuntu 26.04 LTS (WSL2)
✓ PowerShell 7.6.4 (Windows host)
✓ Bash 5.x
```

## OpenCode Status

- ✅ Canonical Linux install: `/usr/local/bin/opencode` → `/home/braka/.opencode/bin/opencode`
- ✅ Single binary (183 MB), version 1.18.18
- ✅ ACP backend support via `opencode acp --stdio`
- ✅ theme=system in tui.json
- ✅ MCP support available

## Intelligent Terminal Settings Status

- ✅ settings.json located at:
  `/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json`
- ⚠️ Backup created at `~/.ovav-backups/<timestamp>/intel-terminal-settings.json.bak`
- ✅ Merge via `install.sh` (surgical, jq-validated)
- ⚠️ Requires Intelligent Terminal restart to apply

## Sources Consulted

- https://github.com/atuinsh/atuin (v18.19.0, 2026-08-03)
- https://github.com/junegunn/fzf (v0.74.2, 3,720 commits)
- https://github.com/ajeetdsouza/zoxide (38.6k★, 626 commits)
- https://github.com/starship/starship (59.4k★, 4,362 commits)
- https://github.com/microsoft/inshellisense (10.6k★, 318 commits)
- https://github.com/akinomyoga/ble.sh (4.6k★ — last stable 2019)
- https://github.com/carapace-sh/carapace (1.4k★, 2,155 commits)
- https://github.com/PowerShell/PSReadLine (4.3k★, mature)
- https://github.com/anomalyco/opencode (197k★, 16M devs/month)
- https://github.com/microsoft/terminal (v1.24.11911.0)
- https://github.com/microsoft/intelligent-terminal (v0.2.2192)
- https://opencode.ai/docs/acp
- https://opencode.ai/docs/themes
- https://opencode.ai/docs/mcp-servers
- https://learn.microsoft.com/en-us/windows/terminal/customize-settings/startup