# Sync/Converter Gap Analysis: OVAV Proposal vs CLI Runtime Reality

**Generated:** 2026-07-28
**Scope:** `go-runtime/internal/convert/` + `go-runtime/cmd/convert_agents/` + `go-runtime/internal/sync/`

---

## 1. The Pipeline — What Actually Happens

```
┌──────────────────────────────────────────────────────────────┐
│  CANONICAL LAYER (What OVAV defines)                         │
│  ovav/agents/{areas,leads,teams}/*.yaml        (80+ files)   │
│  .ovav/service_areas/**/*.yaml                  (100+ files)  │
│  .ovav/plan/caps.yaml                           (1000+ lines) │
└──────────────────────────┬───────────────────────────────────┘
                           │ convert_agents reads from here
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  CONVERTER ENGINE (go-runtime/internal/convert/)             │
│  LoadAgents() → YAML parse → ConvertArea/Lead/Team() → .md   │
│  Only reads: ovav/agents/{areas,leads,teams}/*.yaml          │
│  Ignores: .ovav/service_areas/ (entire subtree)              │
│  Ignored: .ovav/plan/caps.yaml                               │
└──────────────────────────┬───────────────────────────────────┘
                           │ writes to output dirs
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  CLI RUNTIME (What the CLI sees)                             │
│  For opencode/mimocode: only 10 area-*.md files              │
│  Lead and team files: stripped (AreasOnly() == true)          │
│  ~100+ service_area artifacts: absent                         │
│  caps.yaml content: absent (hardcoded boilerplate only)       │
└──────────────────────────────────────────────────────────────┘
```

### 1.1 sync.go Does NOT Trigger Conversion

The sync engine (`internal/sync/`) detects changes via SHA-256 checksums across `syncDirectories` (which includes `ovav/agents/areas`, `ovav/agents/leads`, `ovav/agents/teams`, and `runtimes`). However, it **never calls `convert.GenerateAll()`**. The sync pipeline is pure change detection and manifest management — it tracks what changed but does not regenerate runtime output. The converter must be run independently via `go run -C go-runtime ./cmd/convert_agents`.

**Finding:** sync and convert are two independent pipelines with no coupling. There is no `materialize.py` file in the repo at all — that reference in the `tools/permissions/` path does not exist.

---

## 2. What `convert_agents` Actually Converts

### 2.1 Source → Target Mapping

| Canonical Source | Parsed Into | OpenCode Output | Mimocode Output | Claude Output | Cursor Output |
|---|---|---|---|---|---|
| `ovav/agents/areas/*.yaml` | `Area` struct | `area-{id}.md` | `area-{id}.md` | `area-{id}.md` | `area-{id}.mdc` |
| `ovav/agents/leads/*.yaml` | `Lead` struct | `lead-{id}.md` (hidden) | `lead-{id}.md` (hidden) | `lead-{id}.md` | `lead-{id}.mdc` |
| `ovav/agents/teams/*.yaml` | `TeamMember` struct | `team-{id}.md` (hidden) | `team-{id}.md` (hidden) | `team-{id}.md` | `team-{id}.mdc` |
| `.ovav/source/agents/ovav.md` | raw copy | `ovav.md` | `ovav.md` | `ovav.md` | `ovav.mdc` |
| `.ovav/service_areas/**/*.yaml` | **NOT LOADED** | -- | -- | -- | -- |
| `.ovav/plan/caps.yaml` | **NOT LOADED** | -- | -- | -- | -- |

### 2.2 Canonical Agent YAML → Runtime .md: Field Preservation Matrix

#### Area (area-platform-engineering.yaml as reference)

| Canonical Field | Preserved in Runtime? | Notes |
|---|---|---|
| `id` | Partial | Used for filename only, not in body |
| `name` | ✅ Full | Frontmatter `name:` + Identity Guard + footer |
| `description` | ✅ Full | Frontmatter `description:` |
| `color` | ✅ Full | Frontmatter `color:` + body |
| `lead` | ✅ Full | Body `**Lead:**` |
| `surface` | ✅ Full | Body `**Superficie:**` |
| `functions` (10 items) | ✅ Full | Enumerated numbered list |
| `limitations` (10 items) | ✅ Full | Enumerated with ❌ |
| `hard_stop` | ✅ Full | Verbatim in code block |
| `squad_preview` (12 members) | ✅ Full | Table: Miembro/País/Especialidad |
| `delegation` | ✅ Full | Verbatim |
| `references` | ✅ Full | Enumerated |
| `governance_wiring` (6 gates) | ✅ Full | Enumerated |
| `permission` (edit/bash/external_directory) | ✅ Full | Frontmatter YAML |
| `ovav_connection.instructions` | ✅ Full | Frontmatter `instructions:` list |
| `ovav_connection.skills` (11) | ✅ Full | Enumerated list in body |
| `ovav_connection.cli_commands` (7) | ✅ Full | Code block in body |
| `ovav_connection.contracts` (3) | ✅ Full | Enumerated list |
| `ovav_connection.laws` (4) | ✅ Full | Enumerated list |
| **OVAV_IDENTITY_GUARD block** | ✅ **Generated** | Injected by WriteIdentityGuard(), not from YAML |

**Area verdict:** ~95% of canonical Area YAML content reaches the runtime. The core operational profile is faithfully transmitted.

#### Lead (lead-thavren.yaml as reference)

| Canonical Field | OpenCode | Mimocode | Notes |
|---|---|---|---|
| `id` | Filename only | Filename only | |
| `name` | ✅ | ✅ | |
| `display_name` | ✅ | ✅ | |
| `description` | ✅ | ✅ | Frontmatter |
| `area` | ❌ | ❌ | Only `display_name` shown, machine ID lost |
| `origin` | ✅ | ✅ | |
| `color` | ✅ | ✅ | |
| `authority` | ✅ | ✅ | As literal string, not loaded |
| `functions` (10) | ✅ | ✅ | |
| `limitations` (10) | ✅ | ✅ | |
| `hard_stop` | ✅ | ✅ | |
| `squad` (12 members) | ✅ | ✅ | |
| `delegation` | ✅ | ✅ | |
| `references` | ✅ | ✅ | |
| `permission` | ✅ | ✅ | |
| `steps` | ❌ | ✅ | **INCONSISTENCY** |
| `response_style` | ❌ | ✅ | **INCONSISTENCY** |
| `knowledge_rules` | ❌ | ✅ | **INCONSISTENCY** |

**Lead verdict:** OpenCode converter silently drops `response_style` and `knowledge_rules` (max_words, format, domain rules, behavior rules). Mimocode preserves them. Claude and Cursor converters drop ALL of these.

#### Team Member (team-orin.yaml as reference)

| Canonical Field | Preserved? | Notes |
|---|---|---|
| `name`, `function` | ✅ | |
| `area`, `lead`, `country` | ✅ | |
| `actions` | ✅ | |
| `hard_stop`, `response` | ✅ | |
| `model` | ✅ | Frontmatter |
| `permission` | ✅ | |
| `steps` | ✅ | OpenCode + Mimocode |
| `response_style` | ✅ | OpenCode + Mimocode |
| `knowledge_rules` | ✅ | OpenCode + Mimocode |

**Team verdict:** Team conversion is the most complete. All fields reach the runtime for OpenCode/Mimocode. Claude and Cursor drop response_style and knowledge_rules.

---

## 3. Major Gap Analysis

### GAP-1 [CRITICAL]: Service Lane Architecture — Zero Propagation

The `.ovav/service_areas/platform_engineering/lanes.yaml` defines **12 service lanes** (architecture_contracts, repo_local_implementation, opencode_experience, runtime_governance, harness_validation, workstation_configuration, install_backup_recovery, adapters_protocols, memory_snapshot_continuity, release_closure, launch_readiness, security_scope). Each lane has:
- A designated squad
- Specific artifacts
- Gate requirements
- Activation rules (lead_resolves_first, delegation_by_size_risk)

**What the runtime agent sees:** Nothing. Zero. The converter has no code path that reads from `.ovav/service_areas/`. There is no import, no file read, no YAML parsing for any file under that directory.

### GAP-2 [CRITICAL]: Capabilities Registry — Zero Propagation

`.ovav/service_areas/platform_engineering/capabilities.yaml` defines:
- `default_tools`: 6 allowed operations (read_repo_by_task, edit_repo_by_task, run_source_local_validators, inspect_git_status, build_snapshot, create_sanitized_handoff)
- `denied_without_grant`: 4 permanently blocked operations (unscoped_home_write, secret_exfiltration, uncontrolled_global_install, broad_git_staging)
- `context_classes`: allowed + limited context categories
- `execution` capabilities: source_local, git, system, external permissions matrix

**What the runtime agent sees:** Nothing. The only "capability" information in the runtime is the `permission` block (bash + external_directory) which only covers a fraction of what capabilities.yaml defines.

### GAP-3 [CRITICAL]: Lead Self-Model Artifacts — Zero Propagation

Each lead has 6+ YAML files under `.ovav/service_areas/{area}/{lead}/`:

| File | Content | Size |
|---|---|---|
| `THAVREN_SELF_MODEL.yaml` | Real contributions, ovav systems not mine, blind spots, metrics | 101 lines |
| `CRITERIA.yaml` | 11 versioned decision criteria with origin, evidence, confidence | 219 lines |
| `EVOLUTION.yaml` | 6 sessions of evolution log, decisions, growth scores | 206 lines |
| `OPERATING_LEVEL.yaml` | Foundational law: "AVANZADO+ baseline, knowledge compression mandate" | 120 lines |
| `OVAV_RELATIONSHIP.yaml` | Relationship contract: Thavren↔OVAV principles, protocols, boundaries | 117 lines |
| `SELF_SNAPSHOT.yaml` | Identity snapshot | varies |

**Total:** ~800+ lines per lead of deep identity, criteria, and operational knowledge.

**What the runtime agent sees:** Nothing. The runtime agent file for Thavren is 135 lines of the YAML frontmatter converted to markdown. 100% of the self-model depth is invisible to the CLI.

### GAP-4 [HIGH]: caps.yaml Content — Hardcoded Boilerplate Only

The converter hardcodes three governance contract references in every area output:
```
- visual_delivery_contract.yaml
- safe_stop_contract.yaml
- context_economy_contract.yaml
```

These are **NOT read from caps.yaml**. If the plan adds or changes contracts, the converter code must be manually updated. The caps.yaml file (3300+ lines of active sprint data, layered health, migration tracking, priority zero activation) provides rich context that could help agents understand system state — but none of it is exposed.

### GAP-5 [HIGH]: AreasOnly — Lead and Team Hierarchy Flattened

For OpenCode and Mimocode, `AreasOnly()` returns `true`. This means:
- **10 area agents** are generated and visible to the CLI user
- **10 lead agents** (Thavren, Eidren, Elena, etc.) are stripped — never generated
- **60+ team members** (Orin, Soren, Marco, etc.) — never generated
- `cleanNonAreaAgents()` additionally removes any leftover lead/team files

**Rationale from code comments:** "prevents internal governance roles from leaking into the user-visible list." However, this means the CLI user has no access to:
- Individual lead profiles, criteria, or squad structure
- Team member capabilities, models, or response styles
- The rich delegation and cross-area routing defined in each lead's YAML

The `--levels all` override exists but requires conscious invocation.

### GAP-6 [MEDIUM]: OpenCode Lead Converter Drops response_style and knowledge_rules

OpenCode's `ConvertLead()` (lines 186-266 of `opencode.go`) omits `ResponseStyle` and `KnowledgeRules` entirely, while Mimocode's `ConvertLead()` (lines 181-273 of `mimocode.go`) includes them. This is a silent information loss affecting all 10 lead profiles — Thavren's response rules (max 150 words, result_first, icon + table structure) and knowledge rules (Go TDD, security least-privilege, etc.) are stripped from the OpenCode runtime but preserved in Mimocode.

### GAP-7 [MEDIUM]: Squad Members — Shallow Table Only

When an area or lead agent is generated, squad members appear as a 3-column table (Name, Country, Specialty). But each squad member has their own canonical team YAML file with:
- Full function description
- Authorized actions list
- Model assignment
- Permission block
- Response style rules
- Knowledge rules
- Hard stop and out-of-scope responses

The converter does NOT cross-reference squad member names against the team YAML directory. The team files exist in `ovav/agents/teams/` but are only used when `ConvertTeam()` is called (which doesn't happen for AreasOnly runtimes). Even when team files ARE generated, the area/lead output doesn't link to them.

### GAP-8 [LOW]: Claude and Cursor Converters — Severely Stripped

Claude Code and Cursor converters are minimal (Claude: 113 lines, Cursor: 103 lines). They preserve only:
- Name, description, type, color (frontmatter)
- Identity guard
- Functions and limitations (bare bullet list)
- Hard stop

They strip: squad members, delegation, references, governance wiring, ovav_connection (skills/CLI/laws/contracts), permission block details, response_style, knowledge_rules, steps.

---

## 4. What OVAV Defines vs What CLI Receives — Summary

| OVAV Canonical Artifact | CLI Runtime Visibility | % Propagated |
|---|---|---|
| Agent YAML core fields (name, functions, limits, hard_stop) | ✅ Full | 95% |
| Agent YAML squad preview | ✅ Table | 90% |
| Agent YAML ovav_connection (skills, CLI, contracts, laws) | ✅ Full | 100% |
| Agent YAML permission block | ✅ Full | 100% |
| Lead response_style / knowledge_rules | ⚠️ Partial (Mimocode only) | 50% |
| Agent YAML squad member team files | ❌ Not cross-referenced | 0% |
| `.ovav/service_areas/{area}/lanes.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/capabilities.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/area_boundaries.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/model_body_ladder.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/human_topology.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/lead_contract.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/{lead}/SELF_MODEL.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/{lead}/CRITERIA.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/{lead}/EVOLUTION.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/{lead}/OPERATING_LEVEL.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/{area}/{lead}/OVAV_RELATIONSHIP.yaml` | ❌ Absent | 0% |
| `.ovav/service_areas/shared/*.yaml` (8+ contracts) | ⚠️ Referenced via hardcoded names | File path only |
| `.ovav/plan/caps.yaml` (3300+ lines) | ❌ Absent | 0% |
| Lead agents (for AreasOnly runtimes) | ❌ Not generated | 0% |
| Team agents (for AreasOnly runtimes) | ❌ Not generated | 0% |

---

## 5. Root Cause

The converter's `LoadAgents()` function only reads from three directories:

```go
areas, err = loadAreas(filepath.Join(canonicalRoot, "areas"))     // ovav/agents/areas/
leads, err = loadLeads(filepath.Join(canonicalRoot, "leads"))     // ovav/agents/leads/
teams, err = loadTeams(filepath.Join(canonicalRoot, "teams"))     // ovav/agents/teams/
```

There is **no code anywhere in the convert package** that reads from `.ovav/service_areas/`, `.ovav/plan/caps.yaml`, or any other governance artifact. The converter treats agent YAML files as the sole source of truth for agent identity — the entire service area architecture (lanes, capabilities, self-models, criteria, evolution logs, operating levels) exists in a parallel universe that the CLI runtime never sees.

---

## 6. Recommendations

1. **Wire service_area artifacts into the converter** — Import `lanes.yaml` and `capabilities.yaml` per area, render them in the agent body under a "Service Architecture" section.

2. **Cross-reference squad members with team YAML files** — When generating area/lead output, load each squad member's team YAML and include their function and model in the squad table.

3. **Fix OpenCode lead converter parity** — Add `response_style` and `knowledge_rules` to `OpenCodeConverter.ConvertLead()` to match Mimocode.

4. **Enrich Claude/Cursor converters** — Emit ovav_connection, squad, and delegation sections currently stripped.

5. **Add caps.yaml context injection** — Read `realtime_state` and `current_state.summary` from caps.yaml and inject a contextual header in the generated agent body.

6. **Provide a `--levels all` flag by default** or document it explicitly — currently leads and teams are invisible to the primary CLI user.

7. **Consider sync → convert coupling** — When the sync engine detects changes in `ovav/agents/` or `.ovav/service_areas/`, automatically trigger `convert_agents` regeneration.
