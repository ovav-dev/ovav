# @ovav/pi-memory

> OVAV MEMORY v3 — Proactive memory injection + governance layer for [pi coding agent](https://pi.dev)

**@ovav/pi-memory** turns pi into a governed, self-aware coding agent with:

- **Proactive memory injection** — Cells are injected `before_agent_start` when error patterns are detected
- **LiveProfiler signal detection** — RETRY_LOOP (3-4 errors), ERROR_LOOP (5+), EDIT_BURST (4+ writes)
- **F0-F5 governance validators** — architecture, infrastructure, roles, security, integration gates
- **PermissionGate** — blocks dangerous commands (`rm -rf`, `chmod 777`, credential exfil)
- **AuditLogger** — full event stream to `.ovav/audit/`
- **Privacy tags** — SECRET / SENSITIVE / PROJECT / PUBLIC cell classification

## Install

```bash
# From git
pi install git:github.com/ovav-systems/ovav

# Or copy extension manually
cp src/extensions/ovav-memory.ts ~/.pi/agent/extensions/ovav-memory.ts
```

Requires: pi ≥ 0.84, Node.js ≥ 20

## Quick start

```bash
pi
# Extension auto-loads. OVAV MEMORY v3 shows notification on session start.
```

## Tools

| Tool | Description |
|------|-------------|
| `ovav_validate` | Run F0-F5 governance validators |
| `ovav_daily` | Daily governance brief |
| `ovav_next_work` | Next recommended work item |
| `ovav_check_integrity` | System integrity check |

## Architecture

```
~/.pi/agent/extensions/
└── ovav-memory.ts        ← Extension entry point

.ovav/runtime/livemem/cells/  ← Cell storage (one JSON per cell)
.ovav/audit/                   ← Audit event stream
```

### Memory flow

```
tool_call / tool_result
       ↓
   LiveProfiler (5-min sliding window)
       ↓
   Detector (classifies signal, looks up Cell)
       ↓
   before_agent_start injects Cell summary
       ↓
   LLM receives: <ovav_memory_inject>...</ovav_memory_inject>
```

### Signal thresholds

| Signal | Trigger |
|--------|---------|
| RETRY_LOOP | 3-4 errors in window |
| ERROR_LOOP | 5+ errors in window |
| EDIT_BURST | 4+ writes in window |
| CONTEXT_PRESS | 3+ reads in window |

### Cell weight

- Initial weight: 0.5
- Injection helps → +0.1 (max 1.0)
- Injection doesn't help → -0.05 (min 0.0)
- Cells below 0.6 weight are not injected (configurable)

## Governance validators

| Gate | Checks |
|------|--------|
| F0 | Plan exists, VERSION consistent |
| F1 | Architecture: .gitignore, go.mod, VERSION, internal/ boundaries |
| F2 | Roles: service areas in caps.yaml |
| F3 | Security: no plaintext secrets, no exfil patterns |
| F4 | Integration: runs all gates, computes HEALTHY/DEGRADED/CRITICAL |

## Privacy

Cells are tagged:

- `PUBLIC` — safe to inject in any scope
- `PROJECT` — only in project scope
- `SENSITIVE` — only in project scope with confirmation
- `SECRET` — never injected automatically

## Configuration

```typescript
// In extension: OVAVMemory accepts config
const ovav = new OVAVMemory({
  projectPath: process.cwd(),
  harnessScope: "pi",
  minWeight: 0.6,  // minimum cell weight to inject
});
```

## License

MIT — OVAV Systems
