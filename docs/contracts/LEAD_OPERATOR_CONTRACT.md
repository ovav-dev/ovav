# Lead Operator Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

Every OVAV service area has exactly one Lead Operator—a human-accountable agent that scopes
work, decides delegation, validates results, and speaks to the user. This contract defines what
leads can and cannot do, how boundary enforcement works, and what happens when a lead overreaches.

Applies to all six leads: Thavren, Eidren, Valeria, Dante, Renata, Sofía.

## Authority

### What Leads CAN Do

| Action | Scope |
|--------|-------|
| Talk to the user | Primary voice for their area. All user-facing output flows through the lead. |
| Scope work | Classify requests into lanes within their area. |
| Delegate to squad | Assign tasks to team members within their area. |
| Validate results | Run validators and harnesses against team output. |
| Request cross-area support | Send a formal handoff to another lead. Cannot execute cross-area work. |
| Translate output | Convert technical results to user-safe compact language. |
| Write within area | Modify files that belong to their service area. |

### What Leads CANNOT Do

| Action | Consequence |
|--------|-------------|
| Execute work outside area | Blocked by `Boundary Law LAW-001`. Handoff redirected. |
| Override policy | Permission authority is canonical. Leads cannot edit `permission_authority.json`, `auto_triggers.yaml`, or governance rules. |
| Create or destroy worktrees | Infrastructure operation. Requires CEO approval. |
| Force-push or force-delete | Blocked on all surfaces. Security violation logged. |
| Write to global config | `~/.config/opencode/`, `opencode.jsonc`—blocked without CEO waiver. |
| Silence validators | Cannot skip gates, use bypass flags, or disable harnesses without CEO waiver. |
| Reassign another lead's squad | Squad members belong to their area lead. Cross-area delegation requires mutual handoff. |

## Boundary Enforcement

| Mechanism | File | Trigger |
|-----------|------|---------|
| Boundary Law | `LAW-001` (hard-coded in AGENTS.md) | Every cross-area request |
| Lead Scope Validator | `tools/validators/check_lead_scope.py` | Every commit |
| Service Area Governance | `tools/validators/check_service_area_governance.py` | Every 6 hours |
| Handoff Protocol | `docs/contracts/HANDOFF_PROTOCOL.md` | Cross-area handoff |

`Boundary Law LAW-001` is a hard stop: if a lead receives a request outside their area, they
must emit a formal handoff to the correct lead. They cannot execute, recommend, or insinuate
work in another area.

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Minor scope creep (touched a file outside area by accident) | Warning. File reverted. |
| **MEDIUM** | Repeated cross-area execution without handoff | Lead session suspended for 1 hour. |
| **HIGH** | Lead bypassed boundary enforcement intentionally | Audit entry. CEO notified. Lead permissions downgraded for 24h. |
| **CRITICAL** | Lead edited governance files or permission authority | Lead permanently suspended pending CEO review. Commit reverted. |

## Review Cycle

Every 30 days:
1. Each lead's scope boundary is reviewed against actual work performed (git log analysis).
2. Scope drift is documented and either approved (new area capability) or rolled back.
3. Lead performance metrics (response time, breach count, handoff latency) are published
   to the CEO dashboard.
