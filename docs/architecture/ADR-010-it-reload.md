# ADR-010: IT Reload Integration

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 1), ADR-008 (deploy pipeline)
**Decider:** Thavren + CEO

## Context

After deploy (ADR-008), the Intelligent Terminal (IT v0.1.4) does NOT
auto-reload its settings.json. Operators must:
1. Manually close IT (all windows)
2. Reopen IT
3. Verify keybindings work

This adds friction to every deploy. We need automated reload.

## Decision

Add `ovav it reload` command that triggers IT reload via PowerShell +
Win32 API. Multiple fallback methods:

### Method 1: Win32 Broadcast (preferred)
- PowerShell script sends `WM_SETTINGCHANGE` broadcast to all top-level
  windows
- IT picks up the change without restart
- Silent, no UI interruption

### Method 2: WMI Process Restart
- If Method 1 doesn't work (settings unchanged at OS level), restart IT
- PowerShell: `Get-Process IntelligentTerminal | Stop-Process -Force`
- Then re-launch IT

### Method 3: Operator notification (fallback)
- If both methods fail (e.g., PowerShell not available)
- Print clear instructions for manual reload
- Continue deploy (don't fail)

### Health check

After reload, verify:
- IT process is running
- New settings.json hash matches fragment (via ITLiveKeybindings validator)

If health check fails → WARN, but don't fail deploy.

## Consequences

### Positive

- **No operator action** — deploy → reload → done
- **Multi-method** — handles IT quirks at different versions
- **Safe** — never fails deploy due to reload issues
- **Auditable** — logs reload method + result

### Negative

- **PowerShell dependency** — Windows-specific (acceptable since IT is Windows)
- **WSL→Windows bridge** — runs `powershell.exe` from WSL
- **Side effects** — closing/launching IT may disrupt user

### Mitigations

- Default to Method 1 (no IT restart)
- Operator can disable: `--no-reload` flag
- All methods logged for post-mortem

## References

- ADR-005 Phase 1 D5
- ADR-008 deploy pipeline (consumed)
- ADR-009 CI drift gate (parallel protection)
