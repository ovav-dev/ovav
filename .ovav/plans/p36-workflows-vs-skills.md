# P36 — Warp Drive Workflows vs Skills

## Distinction (per plan §36)

```
Warp Workflow      → acceso humano rápido a comandos
Skill              → procedimiento para agentes
OWS                → implementación real authority
```

## Examples

### Human flow

```text
CEO presses Ctrl+P
  ↓
Warp workflow picker: "OVAV · Verify"
  ↓
Executes: ovav worktree owv
  ↓
Output shown in current shell
```

### Agent flow

```text
OpenCode /ovav-verify
  ↓
Loads .opencode/skills/ovav-verify/SKILL.md
  ↓
SKILL.md guides: owv + interpret + report
  ↓
Agent executes via shell, parses output
```

### Both terminate at OWS

Both use the same authority: `ovav worktree owv`.

## Why split?

- Humans think in commands (verify, status, done)
- Agents think in procedures (interpret, report, debug)
- Same backend, different UX surfaces

## Status

✅ P36 100% — distinction documented, both paths terminate at OWS.
