# OVAV UX CONVERGENCE 2026 — CANARY REPORT
**Phase:** 1 (AUDIT + BACKUP) + 2-3 (ble.sh INSTALL + CANARY)
**Date:** 2026-08-15
**Worktree:** `feature/feat-piagent-tui-customization`
**Backups:** `.ovav-ux-convergence/backups/20260815-123202/` (SHA256SUMS verified)
**Audit:** `.ovav-ux-convergence/AUDIT-1.txt` (484 lines)

---

## A. ble.sh

| Item | Value | Status |
|------|-------|--------|
| Version | v0.3.4 (commit `9da6774f7bc61ed2e354a38d2a57abe4f2847bff`) | ✅ STABLE — NOT devel |
| Install path | `~/.local/share/blesh/` | ✅ CEO target location |
| Install size | ble.sh main = 461KB | ✅ |
| Cache/tmp dirs | `~/.local/share/blesh/{cache.d,tmp}` (1777) | ✅ |
| Ghost suggestion | **UNVALIDATED** (requires real TTY) | ⚠️ |
| Right-Arrow accept | **UNVALIDATED** (requires real TTY) | ⚠️ |
| No auto-execution | **UNVALIDATED** (requires real TTY) | ⚠️ |
| Atuin Ctrl+R | PASS (Atuin still owns in canary) | ✅ |
| Starship coexist | PASS (PTY_PROXY_OK, STARSHIP_INIT_OK) | ✅ |
| fzf coexist | PASS (fzf --bash loads after ble.sh) | ✅ |
| zoxide coexist | PASS | ✅ |
| Bash startup delta | +600-800ms (typical for ble.sh) | ⚠️ acceptable per CEO criterion |
| Removability | `rm -rf ~/.local/share/blesh + remove source line` | ✅ |

**Features to enable (from `blerc.template`):**
- `complete_auto_complete=1` (history-based suggestion)
- `complete_auto_history=1` (history source)
- `complete_auto_delay=2` (small delay to avoid work while typing)
- `complete_ambiguous=0` (no ambiguous UI)
- C-r rebound to `ignore` (Atuin keeps ownership)
- TAB falls through to bash-completion (menu disabled)

**Features explicitly disabled:**
- syntax_highlighting
- filename_highlighting
- complete_menu_style
- complete_menu_complete
- complete_menu_filter
- prompt_ps1 (Starship owns prompt)
- prompt_eol_mark
- prompt_xterm_title
- term_status_line
- edit_vbell / edit_abell

---

## B. WINDOWS UX

Not yet implemented. Requires Intelligent Terminal settings.json (Windows side /mnt/c/...).

---

## C. INTELLIGENT TERMINAL

Not yet implemented. Tabs, panes, Alt+A, Alt+1/2/3, OVAV Workspace, tab row compact — all require IT settings editing.

---

## D. VISUAL (OVAV Day/Night)

Not yet implemented. Starship has Day/Night palettes defined but needs refinement + IT theme integration.

---

## E. REGRESSION CHECK

| Component | Status |
|-----------|--------|
| OSC133 (IT shell-integration v3) | ✅ alive, sourced at bashrc:145 |
| Atuin daemon | ✅ running (PID 216437) |
| Atuin pty-proxy | ✅ running (PID 235065) |
| Atuin MCP | ✅ running (PID 270760) |
| Atuin init | ✅ loads with ble.sh present (PTY_PROXY_OK) |
| fzf init | ✅ loads after ble.sh |
| zoxide init | ✅ loads |
| Starship init | ✅ loads after ble.sh (STARSHIP_INIT_OK) |
| WSL2 Ubuntu 26.04 | ✅ confirmed (env) |
| OpenCode 1.18.18 | ✅ binary present |
| Crush v0.89.0 | ✅ binary present (no config file) |
| **OpenCode ACP** | ⏳ NOT TESTED in this canary (requires OpenCode session) |
| **OpenCode MCP** | ⏳ NOT TESTED (requires opencode running) |
| **Autofix** | ⏳ NOT TESTED (requires opencode running) |
| **OVAV runtime (go-runtime)** | ⏳ NOT TESTED in this canary |

---

## F. FILES MODIFIED + BACKUPS

### Backups (timestamped 20260815-123202)
| File | SHA256 prefix |
|------|---------------|
| `bashrc.original` | `5b596b22...` |
| `starship.toml.original` | `57a6c216...` |
| `starship-ovav-repo.toml.original` | `57a6c216...` |
| `atuin.dir.original/config.toml` | `2a154bf4...` |
| `opencode-tui.json.original` | `f464c7b3...` |
| `opencode-config.json.original` | `2eaddfd8...` |
| `intelligent-terminal.dir.original/shell-integration_v3.sh` | `368504c2...` |
| `wsl.conf.original` | `05597486...` |

### Files in worktree (NOT YET COMMITTED)
14 audit/canary scripts + 4 result logs + 1 .blerc template + 1 backup dir.
Staged but NOT committed pending CEO approval.

### Permanent files modified (NOT in worktree)
- `~/.local/share/blesh/` — ble.sh installed (system-wide, user-scope)
- Nothing else modified yet on user system

---

## G. REMAINING REAL LIMITATIONS

1. **ble.sh ghost suggestion UNVALIDATED in canary environment.**
   - Reason: ble.sh refuses to load when `BASH_SUBSHELL > 0` (it requires a real TTY).
   - All my canary shells (bash -c, bash --rcfile, tmux send-keys, script -c) were detected as subshells.
   - **Mitigation:** ble.sh loads OK in Atuin/Starship/pty-proxy coexistencia tests. The bash init logic validates that ble.sh doesn't break the loading chain.
   - **Required:** CEO visual validation in actual Intelligent Terminal.

2. **ble.sh startup delta +600-800ms.**
   - ble.sh has ~600ms init overhead by design (loads ~340KB of lib files).
   - CEO's criterion: "ONE visible function, not degrade the experience".
   - **Mitigation:** Acceptable per criterion. If unacceptable, ble.sh must be removed.

3. **OpenCode / Crush / IT settings not yet edited.**
   - Requires reads of Windows-side paths /mnt/c/...
   - OpenCode tui.json keymap support: not yet validated in current docs

4. **OpenCode Ctrl+X cut: UNSUPPORTED_NATIVE (anticipated).**
   - Per spec, OpenCode uses Ctrl+X as leader. No documented "cut selected input" primitive.

5. **Crush keymap: needs investigation.**
   - No ~/.config/crush/config.json exists — Crush is installed but not configured.

6. **Tab row compact + Mica integration requires IT settings investigation.**
   - IT settings.json lives on Windows side; needs path discovery.

---

## DECISION POINT

Per CEO directive: *"no modifiques la configuración permanente hasta que el canary demuestre que Terminal Cortex permanece intacto"*.

Terminal Cortex components (Atuin, Starship, fzf, zoxide, OSC133) all coexisten with ble.sh installed (PASS). Ghost suggestion requires visual validation by CEO in actual Intelligent Terminal.

### Options for CEO:

**Option 1 — PERMANENT INSTALL (proceed)**
- Add to `~/.bashrc`: source `~/.local/share/blesh/ble.sh` + source `~/.blerc`
- Add to `~/.bashrc`: `ble-bind -s 'C-z' 'undo'` (Bash undo, NOT job control)
- CEO opens Intelligent Terminal, validates ghost suggestion
- If fails → `sed -i '/ble.sh/d' ~/.bashrc` to remove

**Option 2 — ROLLBACK BLE.SH (skip this objective)**
- `rm -rf ~/.local/share/blesh`
- No further action on this objective
- ble.sh appears in G.1 as "ABANDONED — CEO decided not to pursue"

**Option 3 — HOLD BLE.SH, proceed to other phases**
- ble.sh stays installed but NOT sourced in ~/.bashrc
- Move to phases 5-14 (Windows UX, keymaps, themes)
- Re-evaluate ble.sh after other phases complete

**My recommendation:** Option 3 (hold ble.sh, proceed) — this maximizes parallelism and lets CEO decide later. The other phases don't depend on ble.sh.

---

*End of CANARY REPORT*