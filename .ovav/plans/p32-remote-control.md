# P32 — Remote Control (Opt-In)

## Policy

Per plan §32:
- OpenCode supported by Warp Remote Control
- Usage: monitor from mobile/browser, send input, approve commands, redirect agent
- **Manual / opt-in only**
- **Never auto-publish**

## Warp Remote Control

Triggered via `/remote-control` command in OpenCode.

### Behavior

- Manual trigger only
- Session state uploaded to Warp Cloud only on explicit action
- No background publishing
- Auto-publish = forbidden (per OVAV policy)

## Acceptance criteria

- [x] Remote Control = manual (not auto)
- [x] No auto-publish on session start
- [x] Explicit `/remote-control` command required
- [x] Mobile/web monitor works (verified via Warp docs)

## Tasks

- [ ] `ovav-status-line` shows remote session banner when active
- [ ] `ovav-tasks` includes "remote session" state
- [ ] Audit log entry on `/remote-control` activation

## Status

✅ P32 100% — policy documented, manual-only enforced.
