---
name: ovav-governance
description: OVAV governance validation — F0 architecture, F1 infrastructure, F2 roles, F3 security, F4 integration, F5 permissions. Use when checking project health, validating governance gates, or running system-wide integrity checks.
---

# OVAV Governance

OVAV MEMORY v3 governance layer for pi coding agent.

## Tools

### `/ovav_validate`

Run F0-F5 governance validators on the current project.

```bash
# Full validation (all gates)
pi ovav_validate --scope full

# Fast (F4+F5 only)
pi ovav_validate --scope fast

# Critical (F4 security only)
pi ovav_validate --scope critical
```

Returns: architecture score, infrastructure score, roles score, security score, integration score, overall status (HEALTHY / DEGRADED / CRITICAL).

### `/ovav_daily`

Daily governance brief — validator health, open decisions, recent memory injections, system status.

### `/ovav_next_work`

Next recommended work item from OVAV decision ledger, ranked by priority.

### `/ovav_check_integrity`

Full integrity check: workspace safety, secrets hygiene, permission policy drift, contract freshness.

## How it works

OVAV MEMORY v3 runs 5 governance validators:

| Gate | What it checks |
|------|----------------|
| F0 | Plan exists, caps.yaml valid, go.mod present |
| F1 | Architecture boundaries (.gitignore, VERSION, go.mod) |
| F2 | Service areas defined in caps.yaml |
| F3 | Security: no plaintext secrets, no exfil patterns, no dangerous extensions |
| F4 | Integration: runs F0-F3 in parallel, computes weighted score |

Plus: PermissionGate (command blocking), AuditLogger (event stream), CellStore (proactive memory).

## Privacy

Cells are tagged: PUBLIC / PROJECT / SENSITIVE / SECRET.
SECRET cells are never injected. SENSITIVE cells restricted to project scope.

## Files

Extension: `~/.pi/agent/extensions/ovav-memory.ts`
Cells: `.ovav/runtime/livemem/cells/`
Audit: `.ovav/audit/`
