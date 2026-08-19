# P20, P21, P22 — Harnesses + Model

## P20 — OpenCode (Primary)

Per plan §20:
- OpenCode = main coding agent
- MiniMax-M3 native support via Token Plan
- Plugin: `@warp-dot-dev/opencode-warp`

### Current state

| Setting | Value |
|---|---|
| `opencode.json` model | `minimax-coding-plan/MiniMax-M3` |
| `opencode.json` small_model | `minimax-coding-plan/MiniMax-M3` |
| Plugin loaded | `@warp-dot-dev/opencode-warp` ✅ |
| Provider `minimax-coding-plan` | configured |
| Warp integration | Deep (Code Review, tabs, remote) |

## P21 — Crush (Secondary)

Per plan §21:
- Crush = second opinion, review, alt impl, debugging, parallel experiments
- NOT a duplicate of OpenCode
- MiniMax-M3 supported via Crush's `providers.json`

### Current state

- `~/.local/share/crush/providers.json` has `MiniMax` provider
- API key: `$MINIMAX_API_KEY` (env var, set in Fish prompt hook)
- Endpoint: `https://api.minimax.io/anthropic`
- Models: `MiniMax-M3` (1M context), `MiniMax-M2.7-highspeed`

## P22 — MiniMax-M3 = único modelo principal

Per plan §22:

```
context_window_limit = 1000000
```

M3 offers 1M token context, suitable for coding + tool use + agentic reasoning.

### Cost per 1M tokens

| Model | Input | Output |
|---|---|---|
| MiniMax-M3 | $0.6 | $2.4 |
| MiniMax-M2.7-highspeed | $0.6 | $2.4 |

Both within OVAV budget. Track via `ovav-budget` MCP (currently disabled).

## Architecture diagram

```
                    MiniMax-M3
                        │
         ┌──────────────┼──────────────┐
         │              │              │
      OpenCode       Crush        Warp Agent
         │              │              │
         └────── direct ──────┘    via Warp
              (independence)       (integration)
```

## Status

✅ P20 + P21 + P22 100% — primary/secondary agents and M3 model configured.
