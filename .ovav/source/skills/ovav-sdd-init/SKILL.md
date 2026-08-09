---
name: ovav-sdd-init
description: Triggers automatic project discovery (h_sdd_init) when a project is unknown or stale. Use for stack detection, artifact map generation, and project initialization.
owner_profile: ovav_systems_architect
risk_level: low
rule_pack_id: rp_sdd_init_v1
---

# OVAV SDD Init Skill

Triggers automatic project discovery (h_sdd_init) when a project
is unknown or stale.

## owner_profile
ovav_systems_architect

## owner_lane
runtime_governance

## provenance
S49 repo-local OpenCode pack materialization

## allowed_tools
source-local file read, JSON processing, stack detection

## denied_tools
global config write, plugin install, Engram write, install/apply

## permission_level
source_local_read_only

## memory_write
false

## risk_level
low

## evals_required
eval_sdd_init_detects_stack

## rollback_strategy
delete stale cache

## output_contract
PROJECT_DISCOVERY.json + PROJECT_DISCOVERY.md

## last_validated_at
deterministic

## References

- .ovav/registry/artifacts.yaml

## Implementation Note

`tools/harnesses/h_sdd_init.py` does NOT exist yet. Skill is tracked but harness requires implementation. Track in SK1/SK4 work.