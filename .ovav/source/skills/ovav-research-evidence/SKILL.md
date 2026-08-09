---
name: ovav-research-evidence
description: Guides research, benchmark and source-verification tasks for Eidren through OVAV Research Intelligence. Use for source quality validation, evidence scoring, comparative benchmarking, and decision brief generation.
owner_profile: ovav_research_analyst
permission_level: governed
risk_level: low
rule_pack_id: rp_research_evidence_v1
---

# OVAV Research Evidence Skill

Guides research, benchmark and source-verification tasks for Eidren through the OVAV Research Intelligence service category.

## Ownership
owner_profile: ovav_research_analyst; owner_lane: source_verification; provenance: S49 repo-local OpenCode pack materialization

## allowed_tools
source-local file read, evidence comparison, matrix generation

## denied_tools
external service authority (without explicit gate), global config write, plugin install, Engram write

## Permission
permission_level: source_local_read_write; memory_write: false; risk_level: controlled

## evals_required
eval_opencode_repo_local_pack

## rollback_strategy
revert decision brief to draft

## output_contract
RESEARCH_SCOPE.md + SOURCE_MAP.md + EVIDENCE_REVIEW.md + DECISION_BRIEF.md

## last_validated_at
deterministic

## References
- .ovav/registry/service_profiles.yaml
- docs/03_P0_SERVICE_PROFILES_THAVREN_EIDREN.md
- .ovav/source/skills/ovav-research-session/SKILL.md
- references/evidence-methods.md — research evidence contract and output contract details
