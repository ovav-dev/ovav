# P38 — MCP Audit (no duplication policy)

## Policy

Per plan §38:
- OpenCode: dev MCPs, Playwright, repo-specific
- Warp Agent: only when Warp needs it directly
- Crush: minimum

No auto-duplicate across harnesses.

## Current MCP inventory

### OpenCode (`opencode.json` MCPs)

| MCP | Status | Owner |
|---|---|---|
| `atuin` | enabled | Dev (shell history) |
| `ovav-playwright` | enabled | Dev (browser automation) |
| `ovav-budget` | disabled | Dev (finance — off until needed) |
| `ovav-git` | disabled | Dev (lazy load) |
| `ovav-design-system` | disabled | Dev (lazy load) |
| `ovav-figma` | disabled | Dev (lazy load) |
| `ovav-sqlite` | disabled | Dev (lazy load) |

### Warp Agent

No MCPs configured. Warp uses its own internal MCPs (filesystem, code).

### Crush

No MCPs configured (use defaults).

## Audit checklist

- [x] No duplicate Playwright across harnesses
- [x] No duplicate Git ops across harnesses
- [x] OpenCode MCPs enabled: minimum (2)
- [x] OpenCode MCPs disabled: optional (lazy load)
- [x] Warp Agent MCPs: 0 (correct)
- [x] Crush MCPs: 0 (correct)

## Decision

Audit clean. No duplication. Plan §38 satisfied.

## Future state

If any MCP needs to be duplicated (e.g., file system access for Warp agent):
- Document the reason in `.ovav/plans/mcp-justifications.md`
- Each duplication adds: connection, OAuth, process, context, failure point
- Two duplications max per MCP across 3 harnesses

## Status

✅ P38 100% — no action needed.
