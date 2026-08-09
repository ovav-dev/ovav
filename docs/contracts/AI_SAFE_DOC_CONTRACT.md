# AI-Safe Document Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

All OVAV documentation—specs, contracts, runbooks, agent definitions, and research—must be
AI-safe. An AI-safe document prevents hallucination, requires source-linked claims, and maintains
structural integrity so automated validators can parse and verify it without ambiguity.

This contract applies to every `.md`, `.yaml`, and `.json` document under `docs/`, `.ovav/`,
`.opencode/`, and `go-runtime/` that is consumed by an OVAV agent or validator at runtime.

## Input / Output Schema

| Direction | Schema |
|-----------|--------|
| **Input** | Document content on disk (Markdown, YAML, or JSON). |
| **Output** | Validated document with metadata: `version`, `owner`, `area`, `last_verified`, `sources`, `confidence`. |

Every claim that references external data (research, benchmarks, competitor analysis) **MUST**:
1. Link to a live source URL or a frozen artifact path.
2. Declare a `confidence` level: `high` (primary source), `medium` (secondary), `low` (hearsay/estimate).
3. Carry a `last_verified` date.

Hallucination markers—unsourced numeric claims, invented tool names, fabricated benchmark
results—are treated as hard violations.

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Living Integrity | `tools/validators/check_living_integrity.py` | Every commit (pre-push) |
| Registry Drift | `tools/validators/check_registry_drift.py` | Every 6 hours + post-merge |
| Stale References | `tools/validators/check_stale_artifact_references.py` | Every 12 hours |

These validators scan documents for:
- Missing `version`/`owner`/`area` metadata.
- Claims without `source` annotations.
- Absolute paths or dead internal links.
- Drift between registry definitions and document references.

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Missing metadata field | Warning in integrity mesh. Document still usable. |
| **MEDIUM** | Unsourced quantitative claim | Document flagged `integrity:degraded`. Blocks artifact promotion. |
| **HIGH** | Fabricated source / hallucination | Document quarantined (`integrity:blocked`). Agent that produced it suspended. Commit reverted. |

A quarantined document cannot be referenced by any agent, skill, or harness until the lead
operator for the area re-verifies and re-commits with corrected sources.

## Review Cycle

Every 30 calendar days, the Area Lead must:
1. Re-verify all source links are live (or mark them `frozen` with archive path).
2. Update `last_verified` timestamps.
3. Confirm `confidence` levels are still accurate.

Expired documents (review overdue > 60 days) are automatically downgraded to `confidence:low`
and flagged in the integrity mesh.
