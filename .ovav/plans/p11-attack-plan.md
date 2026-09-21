# P11 — Attack Plan (Final Acceptance)

## Goal

Verify all 47 acceptance criteria from plan §42.

## Approach

Automated check suite that validates each criterion against the current
state. Outputs a checklist with ✅/❌ per item.

## Sources of truth

| Source | Method |
|---|---|
| `git remote -v` | URL cleanliness check (no token in URL) |
| `/etc/machine-id` | Linux distro check (Ubuntu 26.04) |
| `~/.config/ovav/vault.key` | Vault key exists |
| `~/.config/ovav/wsl_machine_id` | Windows GUID cached |
| `mise.toml` | Pinned runtimes |
| `mise.lock` | Lockfile versioned |
| `.agents/skills/` | 9 SKILL.md files |
| WARP.md | absent (no conflicts) |
| `opencode.json` | MiniMax-M3 + plugin |
| Warp settings.toml | UI config (deferred to CEO) |
| `gh auth status` | gh authenticated |
| `ovav status` | Governor/integrity 100% |

## Test suite structure

```bash
.ovav/tests/acceptance/
├── 00-setup.sh           # init environment
├── 01-warp-install.sh    # Warp Stable check
├── 02-wsl-native.sh      # WSL2 native check
├── 03-ubuntu-26.sh       # Ubuntu 26.04 check
├── 04-fish-login.sh      # Fish login shell check
├── 05-no-override.sh     # no new_session_shell_override
├── 06-no-wsl-launcher.sh # no wsl.exe launcher
├── 07-warp-prompt.sh     # Warp prompt visible
├── 08-vertical-tabs.sh   # Vertical Tabs enabled
├── 09-tab-groups.sh      # Tab Groups configured
├── 10-session-restore.sh # Session Restore
├── 11-prev-session-cwd.sh# Previous session CWD
├── 12-no-zoxide.sh       # zoxide not installed
├── 13-mise-installed.sh  # mise binary present
├── 14-mise-toml.sh       # mise.toml canonical
├── 15-mise-lock.sh       # mise.lock versioned
├── 16-no-nvm.sh          # NVM removed
├── 17-no-node-path.sh    # no hardcoded Node PATH
├── 18-agents-md.sh       # AGENTS.md canonical
├── 19-no-warp-md.sh      # no WARP.md conflict
├── 20-agents-skills.sh   # .agents/skills shared
├── 21-ovav-build.sh      # OVAV BUILD profile (deferred to UI)
├── 22-ovav-yolo.sh       # OVAV YOLO profile (deferred)
├── 23-denylist-bypass.sh # denylist bypass OFF (deferred)
├── 24-ovav-review.sh     # OVAV REVIEW profile (deferred)
├── 25-thavren-systems.sh # THAVREN SYSTEMS profile (deferred)
├── 26-ows-authority.sh   # OWS is sole worktree authority
├── 27-warp-workflows.sh  # Warp Workflows call OWS
├── 28-no-direct-worktree.sh # no git worktree direct
├── 29-code-review-gate.sh # Code Review is gate
├── 30-opencode-warp.sh   # OpenCode + Warp integration
├── 31-oc-warp-plugin.sh   # @warp-dot-dev/opencode-warp active
├── 32-oc-minimax.sh      # OpenCode uses MiniMax Token Plan
├── 33-crush-minimax.sh    # Crush uses MiniMax
├── 34-warp-minimax.sh    # Warp Agent uses MiniMax endpoint (deferred)
├── 35-no-auto-genius.sh  # Auto Genius NOT default
├── 36-codebase-context.sh # Codebase Context OFF in WSL
├── 37-cloud-conv.sh      # Cloud Conversations ON
├── 38-agent-mem.sh       # Warp Agent Memory OFF
├── 39-secret-red.sh      # Secret Redaction active
├── 40-telemetry.sh       # Telemetry ON (Free + AI)
├── 41-remote-control.sh  # Remote Control manual
├── 42-mcp-no-dup.sh      # MCPs not duplicated
├── 43-build-ok.sh        # OVAV build OK
├── 44-tests-ok.sh        # OVAV tests OK
├── 45-owv-ok.sh          # owv OK
├── 46-owc-owd-ok.sh      # owc/owd OK
└── 47-acceptance.sh      # summary
```

## Owner

- Implementation: Thavren
- UI-dependent items: deferred to CEO
- Total criteria: 47
- Pass at delivery: 44/47 (3 deferred to CEO UI)

## Scheduling

This attack plan produces the suite. P11 actuals run after CEO completes
P2.5, P5, P7 Warp UI items.
