# P18, P24, P25 — Architecture Details

## P18 — Tab Configs (post-stabilization)

Per plan §18, create 4 Tab Configs AFTER everything else stabilizes:

```
OVAV CORE       → ows + ovav status
OVAV AGENT      → OpenCode + monitor MCPs
OVAV REVIEW     → read-only audit layout
OVAV SYSTEMS    → thavren.systems profile layout
```

### Rules

- Tab Configs **MUST NOT** create worktrees
- Tab Configs **MUST NOT** introduce `wsl.exe -d` overrides
- Tab Configs **MUST NOT** introduce `git worktree add`
- Open layouts over worktrees that OWS already created

### Capture method

```
Save as new config
  ↓
Audit generated TOML
  ↓
Commit to .ovav/warp/tab-configs/
```

## P24 — Warp Agent Architecture (clarification)

Per plan §24, don't confuse BYOM with local execution:

```
Warp Desktop
  ↓
Warp backend/harness (server-side)
  ↓
MiniMax endpoint (api.minimax.io)
  ↓
Warp backend
  ↓
Warp Desktop
```

API key is stored locally but **passes through Warp** for each request
because Warp Agent harness runs server-side.

**Path of greater independence:** OpenCode → MiniMax (direct)
**Path of greater integration:** Warp Agent → MiniMax (via Warp)

Both have purpose. Use each as documented.

## P25 — Cloud Agents (NOT default)

Cloud Agents use Warp infrastructure/credits, NOT local custom endpoint.

```text
local Warp Agent + MiniMax   ✅ available
OpenCode + MiniMax           ✅ operational
Crush + MiniMax              ✅ operational

Warp Cloud Agent             manual/opt-in ONLY
```

Per plan §25: **never launch Cloud Agents automatically from OVAV**. No cost policy.

## Status

✅ P18 + P24 + P25 100% — architecture documented.

## Files

- `.ovav/warp/tab-configs/` (future, post-stabilization)
- `.ovav/plans/p25-cloud-agents-policy.md` (this commit)
