---
name: ovav-artifact-flow
description: Enforces artifact-first SDD flow with OpenSpec-style hard gates. Blocks apply if required upstream artifacts are missing. Use for artifact dependency validation, phase DAG enforcement, and spec-anchored review.
owner_profile: ovav_systems_architect
risk_level: medium
rule_pack_id: rp_artifact_flow_v2
---

# OVAV Artifact Flow Skill (v2 — OpenSpec-enhanced)

Enforces the artifact-first SDD flow with hard gates inspired by OpenSpec (61K★).
Every phase transition requires evidence. No phase skip without CEO waiver.

## Current Baseline

- Phase DAG: `init → explore → proposal → spec → design → tasks → apply → verify → archive`
- Blocking rules enforce artifact-first progression
- Go validator: `phase_dag.go` (175 LOC) validates transitions

## Hard Gates (OpenSpec-pattern)

### Gate 1: INIT → EXPLORE
- **Required artifact:** `PROJECT_DISCOVERY.md`
- **Evidence:** Stack detection output, file count, language breakdown
- **Block if:** No discovery file or stale >7 days

### Gate 2: EXPLORE → PROPOSAL
- **Required artifact:** `EXPLORATION_NOTES.md` + `FLOW_MAP.md`
- **Evidence:** At least 3 exploration angles documented
- **Block if:** Exploration notes empty or single-perspective

### Gate 3: PROPOSAL → SPEC
- **Required artifact:** `PROPOSAL.md`
- **Evidence:** Problem statement, proposed solution, alternatives considered
- **Block if:** No alternatives documented (groupthink detector)

### Gate 4: SPEC → DESIGN
- **Required artifact:** `SPEC.md`
- **Evidence:** Functional requirements, non-functional requirements, constraints
- **Block if:** Missing acceptance criteria or test strategy

### Gate 5: DESIGN → TASKS
- **Required artifact:** `DESIGN.md`
- **Evidence:** Architecture decisions, component diagram, data flow
- **Block if:** No architecture decision records (ADRs)

### Gate 6: TASKS → APPLY
- **Required artifact:** `TASKS.md`
- **Evidence:** Task breakdown with effort estimates, dependencies, acceptance criteria
- **Block if:** Tasks without acceptance criteria or dependencies

### Gate 7: APPLY → VERIFY
- **Required artifact:** `APPLY_LOG.md`
- **Evidence:** Files modified, tests passing, validators green
- **Block if:** Any F0 validator failing

### Gate 8: VERIFY → ARCHIVE
- **Required artifact:** `VERIFY_REPORT.md`
- **Evidence:** All gates passed, no regressions, coverage maintained
- **Block if:** Coverage dropped or new TODOs introduced

## Spec-Anchored Review Pattern

Every commit must map to a spec section:
```
Refs: SPEC.md#section-id
Covers: [S3, S7]
```

Reviewer checks:
1. Does the diff match the spec section?
2. Is there test evidence for each claim?
3. Are acceptance criteria met?

**Evidence required:** test name, command output, or file:line — prose is not evidence.

## Adversarial Verification (absorbed from MiMoCode)

Before marking any phase complete, run adversarial check:
- 3 independent checks per claim
- 2-of-3 reject = claim dismissed
- Abstentions count as invalid (quorum required)

## Cross-Area Integrations

| Area | Lead | Integration |
|---|---|---|
| Platform Engineering | Thavren | Phase DAG enforcement, validator pipeline |
| Research Intelligence | Eidren | Evidence verification, source quality |
| Digital Product | Dante | Design spec → implementation mapping |

## Output Standards

- **Phase transitions:** Each transition logged with timestamp, artifacts verified, gate status
- **Block reasons:** Specific artifact missing + remediation hint
- **Veredict:** `phase_approved | phase_blocked (reason) | phase_skipped (waiver)`

## Delivery

- Precise, gate-driven, evidence-first.
- Frameworks: artifact-first SDD, OpenSpec hard gates, adversarial verification, spec-anchored review.
- Veredict: gate passed / gate blocked — with specific artifact and remediation.
