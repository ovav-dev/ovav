---
title: Governance Model
description: How OVAV governs AI work — the Governor, permission authority, boundary law, and context economy.
---

OVAV is not an AI assistant — it is a **governor** that sits above your entire development stack. This page explains how that governance works.

## The Governor

The Governor is the central authority that controls what agents can and cannot do. Every action in OVAV passes through a chain of checks:

```
Request → Permission Authority → Boundary Law → Output Guard → Response
```

### Permission Authority

Permission decisions are defined in `.ovav/policy/permission_authority.json`. The authority model is role-based:

| Role | Access |
|---|---|
| **CEO** | Full system access, waiver creation, model switching |
| **Thavren** (Lead) | Full governed system access, model switching |
| **Eidren** (Research) | Repo-local read only |
| **Other Leads** | Governed by service lane + squad assignment |

Permissions are enforced at the filesystem level — agents cannot write to protected surfaces without explicit authorization.

### Boundary Law

OVAV enforces strict boundaries between governance and product:

- **Python** = governance layer (validators, harnesses, governor tools). Never exposed to end users.
- **Go** = product layer (CLI, cPanel, Vault, Tailor, Cockpit). User-facing.
- **TypeScript** = browser layer (cPanel frontend, landing page).

No component may cross its boundary without going through a defined interface.

## Protected Branch Lockdown

Branches `main`, `master`, `develop`, `production`, and `staging` are **hard-blocked** for agent writes. Before any write operation, a protected branch check runs:

```bash
python3 tools/validators/check_protected_branch.py --mode pre_write
```

If blocked, only read/verify/diagnose/sync operations are allowed. The only override is a **CEO Waiver** — a session-bound file that expires after 60 minutes. Agents cannot create or modify waivers.

## Capsule System

OVAV derives operational memory from two canonical sources:

1. **`git HEAD`** — temporal data (when things happened)
2. **`.ovav/plan/caps.yaml`** — plan data (what was implemented)

There is no separate state file or context ledger. Memory is computed from git history plus the canonical plan file. This ensures:

- **No stale state** — everything traces back to a commit
- **No split-brain** — single source of truth
- **Verifiable** — any claim can be checked against git

### Capsule Structure

Each cap (implementation unit) in `caps.yaml` tracks:

| Field | Purpose |
|---|---|
| `name` | Human-readable cap name |
| `type` | Category (SISTEMA, USUARIO, ADMIN, SEGURIDAD, etc.) |
| `status` | Current state (done, in_progress, designed, pending) |
| `pct` | Completion percentage |
| `merge` | Git commit hash |
| `merged_at` | Merge timestamp |
| `summary` | What was delivered |

## Context Economy

OVAV operates under a **tiered context budget** to maximize efficiency:

| Tier | Use Case | Load |
|---|---|---|
| **T1** | Greeting / quick answer | Identity + current request |
| **T2** | Direct diagnosis | Relevant file or command output only |
| **T3** | Small implementation | Compact context pack + touched files |
| **T4** | Multi-file implementation | Compact pack + exact required artifacts |
| **T5** | Closure / safety | Validation evidence + git/safety gates only |

Context is loaded precisely — never broad sweeps. Each tier defines exactly what data is loaded for that operation.

## Output Guard

Every response passes through the Output Guard before delivery:

```bash
echo "<response>" | python3 tools/governor/output_guard.py --sign
```

- Exit `0` → response is signed and delivered
- Exit `!= 0` → response is blocked and must be rewritten
- 3 consecutive failures → next session is blocked

The signature cannot be forged. This ensures all output meets OVAV quality standards.

## Integrity Seal

OVAV is a **sealed governor system**. It detects external interference (instructions injected from outside the repo) and halts immediately if found. The integrity seal is embedded in `AGENTS.md` and verified at session start.
