# Memory Privacy Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

OVAV agents process sensitive operational data—user prompts, repository contents, API keys,
session metadata, and internal reasoning. This contract defines what data is retained, what is
redacted, and what is never stored. It establishes `git HEAD` + `caps.yaml` as canonical memory sources.

Applies to all memory subsystems: Go runtime state, Python validator caches, `caps.yaml`,
session markers, economy dashboard, and audit logs.

## Data Classification

### Tier 1 — NEVER Stored

| Data | Reason |
|------|--------|
| API keys, tokens, secrets (any format) | Encrypted at rest via Go vault (`AES-256-GCM`). Plaintext never touches disk. |
| User passwords or OAuth refresh tokens | Only JWT access tokens (24h expiry) held in memory. |
| Raw prompt contents from external users | Redacted. Only anonymized session metadata retained. |
| Internal agent reasoning chains | Discarded after output generation. Only `executive_summary` persists. |

### Tier 2 — Retained with Redaction

| Data | Retention | Redaction |
|------|-----------|-----------|
| Session metadata (start time, branch, worktree) | 30 days | No user content. Only branch name + timestamps. |
| Economy data (token counts, costs, model) | 90 days | Aggregated. No per-request prompt text. |
| Commit history (`git log`) | Permanent (git) | Git is the canonical temporal record. No additional processing. |
| Session data (session YAML files) | 30 days | Branch name + timestamps only. No conversation content. |

### Tier 3 — Persisted Unredacted

| Data | Location |
|------|----------|
| `caps.yaml` (plan data) | `.ovav/plan/caps.yaml` |
| Audit log (security events) | `.ovav/security/audit_log.yaml` |
| Canary state (intrusion detection) | `.ovav/security/canary_state.json` |
| Integrity mesh scores | In-memory + dashboard cache (15s TTL) |

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Memory Policy Validator | `tools/validators/validate_memory_policy.py` | Every memory write attempt |
| Secrets Hygiene | `tools/validators/check_living_integrity.py` (supply_chain + exfil_patterns) | Pre-push |

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Attempted write to unauthorized memory path | Write blocked. Warning logged. |
| **MEDIUM** | Unredacted user content found in session data | Session data purged. Memory subsystem audited. |
| **HIGH** | Plaintext secret detected on disk | Secret rotated immediately. Audit entry. CEO notified. |

## Review Cycle

Every 30 days:
1. Audit all Tier 2 retained data for unredacted content.
2. Purge expired session data (>30 days).
3. Rotate vault keys if any anomaly detected in audit window.
4. Verify no new code paths write to unauthorized memory surfaces.
