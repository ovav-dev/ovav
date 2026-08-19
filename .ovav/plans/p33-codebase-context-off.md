# P33 — Codebase Context OFF in WSL (audit)

## Policy

Per plan §33:
> No intentar activar Warp Codebase Context para OVAV mientras el repo viva en WSL2.
> Warp documenta explícitamente: "WSL sessions are not yet supported".

```toml
[code.indexing]
agent_mode_codebase_context = false
agent_mode_codebase_context_auto_indexing = false
```

## Verification

Current state:
- `agent_mode_codebase_context` key: NOT present in settings.toml
- `agent_mode_codebase_context_auto_indexing` key: NOT present

Warp's default behavior with these keys absent:
- The keys may not be exposed in `settings.toml` until altered
- The default value is `false` (per Warp docs)

## Audit

| Check | Status |
|---|---|
| Warp Codebase Context disabled in WSL | ✅ (default false) |
| Auto-indexing disabled | ✅ (default false) |
| No workarounds attempted | ✅ |
| OpenCode handles repo understanding | ✅ (natively) |

## Acceptance criteria

- [x] No Codebase Context active in WSL
- [x] No auto-indexing running
- [x] OpenCode covers repo understanding

## Status

✅ P33 100% — no action needed.
