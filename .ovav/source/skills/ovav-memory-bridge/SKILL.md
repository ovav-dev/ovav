---
name: ovav-memory-bridge
description: Use when OVAV agents need to read, write, or reconcile memory artifacts in the host mimo memory directory. Provides correct paths, scope resolution, and operation patterns.
license: Apache-2.0
metadata:
  author: dante (OVAV)
  version: "1.0"
status: active
owner_profile: ovav_systems_architect
owner_lane: runtime_governance
permission_level: project_read
memory_write: scoped
memory_write_scope: runtime_governance_only
risk_level: low
last_updated: 2026-07-04
---

# OVAV Memory Host Bridge

## Purpose

OVAV agents delegate the `memory` tool to the host runtime (mimo). This skill
documents the correct paths, scope resolution rules, and operation patterns
so agents don't waste turns on "memory empty" false-negatives.

## Canonical paths

```
HOST MEMORY ROOT: .ovav/memory/
├── working.json              # Active memory (current session)
├── short_term.json           # Short-term cache (recent context)
├── archival.json             # Archived memory (long-term storage)
├── decay_state.json          # Memory decay tracking
├── governor_meta.json        # Governor metadata
├── cards/                    # Knowledge cards (YAML format)
│   ├── crush-integration-*.yaml
│   ├── omars-*.yaml
│   └── ovav-system-*.yaml
└── vectors/                  # Vector index for semantic search
    └── index.json
```

**OVAV project (this repo):** `projects/1ddba6f2-e966-456f-a4ed-56798a001aef/`
**bt-sys-react project (Dante):** `projects/944fca42-9485-479e-b52c-55c9bac228cf/`

## Memory tool — correct usage

### Search patterns that WORK

```javascript
// ✅ Broad rare term (BM25-friendly)
memory({ operation: "search", query: "BLOQUEO" })      // → 10 matches
memory({ operation: "search", query: "F77" })          // → 5 matches
memory({ operation: "search", query: "drag" })        // → matches

// ✅ Feature ID or component name
memory({ operation: "search", query: "F45 component" })
memory({ operation: "search", query: "css-dni" })

// ✅ Verbatim quote fragment
memory({ operation: "search", query: "BLOQUEO SUSTRACCION" })
```

### Patterns that FAIL — avoid

```javascript
// ❌ Too generic — BM25 returns 0
memory({ operation: "search", query: "memory tool bug" })

// ❌ Full sentence — tokenizer splits
memory({ operation: "search", query: "the user said that the footer should be on the right side" })

// ❌ Global scope without content
memory({ operation: "search", query: "anything", scope: "global" })
// → 0 matches because /home/braka/Labs/mimocode/data/memory/global/ doesn't exist

// ❌ Project scope without scope_id
memory({ operation: "search", query: "foo", scope: "project" })
// → may return empty if project_id resolves wrong
```

### Fallback when memory returns 0

1. **Confirm content exists** with grep:
   ```bash
   grep -rn "TERM" /home/braka/Labs/mimocode/data/memory/projects/1ddba6f2-e966-456f-a4ed-56798a001aef/
   ```
2. **Read the file directly** with the Read tool.
3. **Try variant queries** with 1-2 rare terms only.

## External directory write — autonomy rules

**Effective 2026-07-04:** OVAV agents have `thavren: allow` on external_directory
in the canonical permission policy (`.ovav/policy/permission_authority.json`).

**Runtime reality:** The session-scoped permission snapshot may NOT include
`/home/braka/Labs/*` even though the file config does. If blocked:

1. Check `/home/braka/Systems/OVAV/.mimocode/global_config/config.json`
   has `"/home/braka/*": "allow"` under `permission.external_directory`
2. If missing, **add it** (this skill authorizes the edit under `runtime_governance_only` scope).
3. **Restart session** to reload the permission snapshot.

### Authorized writes (no CEO waiver required)

Under this skill's `memory_write_scope: runtime_governance_only`:

- ✅ Read any file in `/home/braka/Labs/mimocode/data/memory/`
- ✅ Write to `projects/1ddba6f2-.../MEMORY*.md` (OVAV project)
- ✅ Write to `sessions/<current_session>/notes.md`
- ✅ Write to `sessions/<current_session>/checkpoint.md`

### Requires CEO waiver

- ❌ Write to other project memory dirs (cross-project pollution)
- ❌ Delete memory files (always backup first)
- ❌ Modify the host `mimo` binary
- ❌ Recursive delete without backup

## Orphan detection & reconciliation

When you find an orphan dir like `944fca42-...228rf/` (typo suffix):

1. **Compare** canonical vs orphan: `diff -q` and `wc -c`
2. **If sizes differ**, the orphan is NOT residue — it has unique content
3. **Never delete without backup**:
   ```bash
   cp orphan_file .ovav/staging/<date>-cleanup/orphan-bkp.md
   ```
4. **Merge strategy** depends on content shape:
   - Sections differ → concatenate with `[merged from <orphan_id> <date>]` markers
   - Whole file different → copy orphan over canonical with timestamp prefix
5. **Deprecate orphan** by renaming to `<original>.deprecated-<date>`

## Known issues (2026-07-04)

- **Session permission snapshot mismatch:** Config files allow `/home/braka/*` but
  runtime snapshots may exclude specific subpaths. Fix: restart session after
  config edit.
- **`memory_write: false` legacy:** All skills declared `memory_write: false`
  until v3 policy (2026-07-04). v3 enables per-lane scoped writes.
- **BM25 index in-memory:** Host mimo builds BM25 index at boot; if file changes
  don't reflect in searches, suggest user restart mimo session.

## History

- **2026-07-04:** Created after T-MEM-001 (Memory Tool failure report).
  Replaces ad-hoc grep fallback patterns. Documents 3 successful search patterns
  (BLOQUEO, F77, drag) and 4 failure modes (generic, sentence, global-no-content,
  project-no-id).