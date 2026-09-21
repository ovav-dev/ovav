# OVAV Validator Registry

> **Auto-generated** from `internal/validators/*.go`. DO NOT EDIT MANUALLY.
> Run `ovav docs generate` to refresh.

**Total validators**: 76

| ID | Name | Weight | Description |
|----|------|--------|-------------|
| `advanced_hardening` | F5 Advanced Hardening | 16 | Validates new states governance, gate liberation, and advanced surfaces |
| `adversarial_verification` | Adversarial Verification | 5 | Runs 3 independent checks per claim. 2-of-3 reject = claim dismissed. Absorbed from MiMoCode adversarial jury pattern. |
| `agent_governance` | Agent Governance | 15 | Validates agent permissions, boundary laws, and file consistency (harness-aware) |
| `agent_permission_invariants` | Agent Permission Invariants | 7 | Validates lead-agent permission invariants against area surface |
| `agent_runtime_enforcement` | Agent Runtime Enforcement | 8 | Validates agent runtime modules: service_area_router, context_gateway, tool_gateway, handoff_protocol, delegation_router, observability_engine |
| `agent_surface_hierarchy` | Agent Surface Hierarchy | 10 | Validates agent surface hierarchy: TAB areas, @ leads, and hidden squads |
| `agent_ux_visual_delivery` | Agent UX Visual Delivery Validator | 8 | Validates OVAV contract references in core agent files and response skill |
| `architecture_governance` | F3 — Architecture Governance | 5 | Validates stack purity, migration progress, and architectural governance compliance |
| `bash_readline_bindings` | Bash Readline Bindings Validator | 6 | Validates OVAV inputrc is present and has recommended bash conveniences |
| `behavioral_directives` | Behavioral Directives | 7 | Validates canonical identity guard generation and source-level area hard-stop contracts |
| `bootstrap_chain` | Bootstrap Chain Validator | 9 | Validates the integrity seal chain from AGENTS.md through OVAV_INTEGRITY_SEAL |
| `canonical_integrity` | Canonical Integrity | 8 | Detects SHA256-duplicate files and broken Python imports |
| `caps_schema` | Caps Canonical Schema | 15 | Ensures caps.yaml has the required canonical structure |
| `caps_single_next` | Caps Single Next Pointer | 25 | Ensures caps.yaml has exactly one active next_phase — no conflicting next pointers |
| `config_integrity` | Config Integrity | 15 | Validates YAML/JSON config syntax, canonical sources, and bootstrap chain |
| `config_syntax` | Config Syntax | 16 | Validates YAML and JSON syntax across all config surfaces |
| `context_economy` | Context Economy Validator | 8 | Validates context economy contract references in active agent, skill, and command surfaces |
| `context_firewall` | Context Firewall | 16 | Validates injection detection patterns, L5 firewall, and context economy |
| `contract_enforcement` | F2 — Contract Enforcement | 4 | Validates service area contracts — completeness, bidirectionality, required fields |
| `contract_freshness` | Contract Freshness | 10 | Checks governance contracts for staleness and integrity |
| `credential_governance` | Credential Governance | 18 | Validates credential vault, provider config, scope isolation, and budget tracking |
| `exfil_patterns` | Exfiltration Patterns | 10 | Scans output logs for data exfiltration patterns |
| `f1_architecture` | F1 Architecture Integrity | 7 | Validates F1 architecture behavior: permission authority, simulation, Rego, and bootstrap |
| `f2_infrastructure` | F2 Infrastructure Governance | 8 | Validates F2 infrastructure: system paths, plugin governance, live behavior, config authority, claims |
| `f3_roles` | F3 Role Governance | 8 | Validates F3 roles: lead/team agent frontmatter, research profile, sandbox rules, temporal limits |
| `feedback_loop` | L7 Feedback Loop Validator | 10 | Validates L7 feedback loop: sanitization, beliefs, compaction, and gate safe-stop |
| `gate_self_protection` | Gate Self-Protection | 18 | Validates host defense gate integrity via hash verification |
| `git_push` | Git Push Gate | 10 | Enforces HTTPS-only push, no split remotes, no force push |
| `harness_contract_alignment` | Harness Contract Alignment | 10 | Validates single authority source (.ovav/plan/caps.yaml) exists |
| `harness_integrity` | Harness Integrity Validator | 8 | Validates harness contract alignment, grouping, and file integrity |
| `host_config_drift` | Host Config Drift | 25 | Validates host configuration integrity, quarantine state, and intrusion detection |
| `install_verification` | Install Pipeline Verification | 20 | Validates backup/rollback integrity, boundary enforcement, and pipeline completeness |
| `integrity_baseline_fresh` | Integrity Baseline Freshness | 10 | Verifies runtime integrity baseline is recent and (optionally) pinned |
| `invalid_fixtures` | Invalid Fixtures Validator | 5 | Meta-validator: verifies that deliberately broken fixture registries are correctly rejected |
| `it_keybindings` | IT Keybindings Validator | 8 | Validates OVAV IT v0.2 keybindings: null/empty IDs, unresolved actions, duplicates |
| `it_live_keybindings` | IT Live Keybindings Validator | 8 | Validates the LIVE Intelligent Terminal settings.json (drift detection vs fragment) |
| `lead_scope` | Lead Scope Validator | 5 | Validates that each lead agent file defines its authorized scope section |
| `ledger_deprecation` | Ledger Deprecation Enforcer | 5 | Fails if active_context_ledger.yaml exists — it is permanently deprecated |
| `ledger_write_path` | Ledger Write Path (deprecated) | 0 | Permanently deprecated. active_context_ledger.yaml must not exist. |
| `model_policy` | Model Policy Validator | 15 | Validates authorized model list and detects forbidden model references |
| `multi_platform` | C7 Multi-Platform Validator | 8 | Validates Windows/macOS readiness and cross-platform configs |
| `network_hardening` | F0.4 Network Hardening | 14 | Validates network allowlist, critical domains, and guard configuration |
| `permission_drift` | Permission Policy Drift | 10 | Detects drift between canonical permissions and runtime state |
| `pinned_baseline_drift` | Pinned Baseline Drift | 15 | Compares current runtime baseline against last CEO-approved pinned baseline |
| `plugin_security` | Plugin & Network Security | 15 | Validates opencode plugins, network hardening, and SSH config |
| `protected_branch` | Protected Branch Gate | 15 | Blocks write operations on protected branches without CEO waiver |
| `red_team_audit` | R5 Red Team Boundary Audit | 9 | Red Team automated boundary audit — verifies LAW-001 compliance across all 9 areas, agent scope enforcement, and cross-area violation detection |
| `registry_drift` | Registry Drift | 14 | Validates registry YAML declarations reconcile with real filesystem |
| `registry_validator` | Registry File Validator | 5 | Validates required registry YAML files exist and parse correctly |
| `rego_policies` | Rego Policy Integrity | 5 | Validates Rego policy engine and deny/allow rule presence |
| `runtime_integrity` | Runtime Integrity | 20 | Verifies protected runtime files against an explicit hash baseline and git state |
| `runtime_wiring` | Runtime Wiring | 14 | Validates harness surface files, governance terms, and stale pattern detection (harness-aware) |
| `secrets_hygiene` | Secrets Hygiene | 20 | Scans codebase for plaintext secrets, tokens, and credentials |
| `security_hardening` | F4 Security Hardening | 20 | Validates bash command governance, unsafe selectors, and deny-by-default enforcement |
| `security_policy` | Security Policy Enforcement | 25 | Enforces maximum-security posture rules (NIST/CIS/OWASP aligned) |
| `service_area_governance` | Service Area Governance | 5 | Validates service area registry files exist and contain required governance terms |
| `service_area_router` | Service Area Router Validator | 8 | Validates all 10 area agent profiles have hard stop contracts covering all other areas |
| `session_context_guard` | Session Context Guard | 20 | Validates session integrity seal, governance files, and injection detection |
| `single_authority` | Single Authority Source | 18 | Validates single canonical authority: caps.yaml + git HEAD |
| `squad_normalization` | Squad Normalization Validator | 7 | Validates squad registry, operators, delegation rules, and runtime governance files |
| `ssh_profile` | SSH Profile Validator | 6 | Validates OVAV Thavren SSH profile artifacts — docs, templates, policy, and install plan |
| `stale_artifact_references` | Stale Artifact References Validator | 7 | Detects stale S* segment references and pre-B18 BUILD references in active files |
| `supply_chain` | Supply Chain Integrity | 20 | Verifies the canonical SBOM against git HEAD and reports worktree drift separately |
| `surface_drift` | Surface Drift Detector | 12 | Detects drift between plan-unlocked surfaces and runtime-blocked surfaces |
| `thought_firewall` | Thought Firewall | 18 | Validates protected branch blocks thought-modification intents (SIS Layer 4) |
| `tool_config_profiles` | Tool Config Profiles Validator | 6 | Validates tool_configs.yaml registry, CLI tool, and bin/ovav consistency |
| `tool_readiness` | Tool Readiness Matrix Validator | 12 | Validates tool readiness matrix, capability boundary, and surface constraints |
| `validate_memory_policy` | Memory Policy Validator | 6 | Validates memory_policy.yaml privacy tags and write pipeline |
| `validate_phase_dag` | Phase DAG Validator | 7 | Validates phase_dag.yaml phase order, transitions, and blocking rules |
| `validate_service_profiles` | Service Profiles Validator | 7 | Validates service_profiles.yaml against expected profiles, lanes, and squads |
| `validate_skills` | Skills Validator | 6 | Validates skills registry YAML structure, rule packs, scores, and cross-references |
| `validator_coverage` | Validator Coverage Auditor | 5 | Measures validator and harness coverage across automation surfaces |
| `wezterm_path_integrity` | WezTerm Path Integrity Validator | 7 | Enforces single canonical WezTerm path — no duplicates, proxy markers present, blocked paths clean |
| `wezterm_workspace_isolation` | WezTerm Workspace Isolation Validator | 5 | Enforces WezTerm workspace isolation — single workspace per context, no cross-contamination |
| `workspace_isolation` | Workspace Isolation Validator | 10 | Validates WezTerm workspace isolation, source template, and governed config |
| `zero_trust` | L6 Zero Trust Security | 22 | Validates F0 hardening baseline, quarantine, provenance, and risk thresholds |

## Categories

### Agent Governance (6)

- `agent_governance`
- `agent_permission_invariants`
- `agent_runtime_enforcement`
- `agent_surface_hierarchy`
- `agent_ux_visual_delivery`
- `permission_drift`

### Capability Registry (2)

- `caps_schema`
- `caps_single_next`

### Context Economy (4)

- `context_economy`
- `context_firewall`
- `session_context_guard`
- `thought_firewall`

### Deployment (1)

- `install_verification`

### General (46)

- `adversarial_verification`
- `architecture_governance`
- `behavioral_directives`
- `bootstrap_chain`
- `config_syntax`
- `contract_enforcement`
- `contract_freshness`
- `credential_governance`
- `exfil_patterns`
- `f1_architecture`
- `f2_infrastructure`
- `f3_roles`
- `gate_self_protection`
- `git_push`
- `harness_contract_alignment`
- `host_config_drift`
- `invalid_fixtures`
- `lead_scope`
- `ledger_deprecation`
- `ledger_write_path`
- `model_policy`
- `multi_platform`
- `pinned_baseline_drift`
- `protected_branch`
- `red_team_audit`
- `registry_drift`
- `registry_validator`
- `rego_policies`
- `runtime_wiring`
- `secrets_hygiene`
- `service_area_governance`
- `service_area_router`
- `single_authority`
- `squad_normalization`
- `ssh_profile`
- `stale_artifact_references`
- `surface_drift`
- `tool_config_profiles`
- `tool_readiness`
- `validate_memory_policy`
- `validate_phase_dag`
- `validate_service_profiles`
- `validate_skills`
- `validator_coverage`
- `wezterm_workspace_isolation`
- `workspace_isolation`

### Orchestration (1)

- `feedback_loop`

### Security (6)

- `advanced_hardening`
- `network_hardening`
- `plugin_security`
- `security_hardening`
- `security_policy`
- `zero_trust`

### Supply Chain (7)

- `canonical_integrity`
- `config_integrity`
- `harness_integrity`
- `integrity_baseline_fresh`
- `runtime_integrity`
- `supply_chain`
- `wezterm_path_integrity`

### Workstation (3)

- `bash_readline_bindings`
- `it_keybindings`
- `it_live_keybindings`

