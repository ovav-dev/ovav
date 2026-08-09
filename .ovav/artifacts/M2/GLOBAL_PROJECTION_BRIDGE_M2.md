# M2 — Global Projection Bridge

**Status:** prepared source-local payload.

M2 prepares the minimal OpenCode global projection needed for OVAV routing from
`/home/braka`, but does not write `~/.config/opencode` in this runtime.

## Payload

- `opencode.jsonc`
- `AGENTS.md`
- `agents/ovav-platform-engineering.md`
- `agents/ovav-research-intelligence.md`

## Safety

- No HOME writes performed.
- No global OpenCode config writes performed.
- No plugin installation performed.
- Payload is source-local under `.ovav/artifacts/M2/global_projection_payload`.

## Validation

- `python3 tools/harnesses/global_projection_bridge_m2.py`
- `python3 tools/harnesses/check_global_projection_bridge_m2.py`
