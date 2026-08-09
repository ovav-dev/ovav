# OVAV × mimocode/opencode Integration

**Status:** ✅ Active (v1.1.0 — 2026-07-02)
**Owner:** Thavren (Lead, Platform Engineering)
**Scope:** OVAV SYSTEMS only. (OVAV PRODUCT was uninstalled 2026-07-02.)

---

## Purpose

This document specifies how OVAV governs **two CLI surfaces** simultaneously:

1. **[opencode](https://opencode.ai)** — the upstream CLI
2. **[mimocode](https://mimocode.ai)** — an XIOAMI-maintained fork of opencode with a free model tier

**The contract:** when the user types `mimocode` *or* `opencode` from **any
directory**, that CLI must boot with OVAV's authoritative governance, all 80+
OVAV agents, the OVAV skills catalog, and OVAV defaults (default agent,
model, permissions). Both CLIs coexist without config collision.

---

## Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │ shell: user types `mimocode` or `opencode`   │
                    └─────────────────────┬───────────────────────┘
                                          │
                                          ▼
                    ┌─────────────────────────────────────────────┐
                    │  ~/.local/bin/mimocode    (OVAV wrapper)      │
                    │  ~/.local/bin/opencode    (OVAV wrapper)      │
                    │     ↓ source shared lib ↓                    │
                    │  ~/.local/bin/ovav-cli-shim.sh  (lib)        │
                    └─────────────────────┬───────────────────────┘
                                          │ exec
                  ┌───────────────────────┼───────────────────────┐
                  ▼                       ▼                       ▼
        /home/braka/.mimocode/    /home/braka/.opencode/
        bin/mimo (v0.1.4)         bin/opencode
                  │                       │
                  ▼                       ▼
        CWD/.<runtime>/{agents, skills, commands, plugins, themes}
                  │                       │
                  └──── symlinks ─────────┴──→ OVAV canonical assets:
                                              runtimes/<runtime>/agents/
                                              .opencode/skills/
                                              .opencode/commands/
                                              .opencode/plugins/
                                              .opencode/themes/
```

### Why two wrappers + a shared lib

Both CLIs share the same OVAV root and asset paths but have **independent
config layouts**:

| Surface | Bin | Config home | CWD dotdir | OVAV assets dir |
|---|---|---|---|---|
| mimocode | `~/.mimocode/bin/mimo` | `~/.mimocode/` | `.mimocode/` | `runtimes/mimocode/` |
| opencode | `~/.opencode/bin/opencode` | `~/.config/opencode/` | `.opencode/` | `runtimes/opencode/` |

The shared lib `ovav-cli-shim.sh` does the work once:

- resolves OVAV root (anchored at the lib's location)
- detects OVAV project mode vs global mode
- creates the CWD shim for whichever runtime is being invoked
- injects OVAV defaults into argv

Each wrapper is a thin shim that sources the lib and execs the upstream
binary. There is **no config collision** because:

- `mimocode.json` and `opencode.json` are separate files
- `~/.mimocode/` and `~/.opencode/` are separate home dirs
- CWD gets two separate dotdirs, each owned by one CLI
- `mimo` and `opencode` are independent processes

---

## The auto-shim (closes the global-mode gap)

mimocode v0.1.4 and opencode both discover agents from
**`CWD/.<runtime>/agents/`** — the home-dir equivalents are not consulted
for agent enumeration. This means a "global config" alone is not enough;
the CWD must have a valid `.<runtime>/agents/` directory.

**Solution:** when the wrapper is invoked from a CWD that is not an OVAV
worktree, it creates the CWD shim on first run:

```
CWD/.mimocode/agents      → /home/braka/Systems/OVAV/runtimes/mimocode/agents
CWD/.mimocode/skills      → /home/braka/Systems/OVAV/.opencode/skills
CWD/.mimocode/commands    → /home/braka/Systems/OVAV/.opencode/commands
CWD/.mimocode/plugins     → /home/braka/Systems/OVAV/.opencode/plugins
CWD/.mimocode/themes      → /home/braka/Systems/OVAV/.opencode/themes

CWD/.opencode/agents      → /home/braka/Systems/OVAV/runtimes/opencode/agents
CWD/.opencode/skills      → /home/braka/Systems/OVAV/.opencode/skills
CWD/.opencode/commands    → /home/braka/Systems/OVAV/.opencode/commands
CWD/.opencode/plugins     → /home/braka/Systems/OVAV/.opencode/plugins
CWD/.opencode/themes      → /home/braka/Systems/OVAV/.opencode/themes
```

The shim is:

- **Idempotent** — only creates links that don't already exist. User-
  authored content (e.g. `.mimocode/tools/memory.ts`) is preserved.
- **Non-destructive** — never overwrites an existing entry. A project
  that ships its own `.opencode/config.json` keeps it.
- **Persistent** — the symlinks stay in the CWD until the user removes
  them with `rm -rf CWD/.mimocode CWD/.opencode`. Subsequent invocations
  detect the existing shim and skip creation.
- **Safe to skip** — `OVAV_NO_SHIM=1` disables auto-shim entirely.

### Blacklisted paths

The shim is **not** created in:

- System paths: `/`, `/tmp`, `/var`, `/etc`, `/sys`, `/proc`, `/dev`,
  `/usr`, `/lib*`, `/bin`, `/sbin`, `/root`, `/snap`, `/boot`
- The CLIs' own install dirs: `~/.mimocode`, `~/.opencode`,
  `~/.config/*`
- Non-writable CWDs

This is enforced by `ovav_ensure_shim` in the lib.

---

## Behavior

### OVAV Project mode (CWD inside an OVAV worktree)

Detection: ancestor contains both `AGENTS.md` AND `.ovav/`.

- The wrapper **does not** create a shim (the project already has its own
  assets in `.mimocode/` and `.opencode/`).
- The wrapper passes the project root as the positional `[project]` arg
  to the CLI, so both `mimo <root>` and `opencode <root>` resolve
  project-local dotdirs first.
- The CLI's own `mimocode.json` / `opencode.json` (symlink to
  `.mimocode/global_config/config.json`) is loaded for OVAV defaults.

### OVAV Global mode (CWD outside any OVAV worktree)

- The wrapper creates the CWD shim on first run (idempotent thereafter).
- Injects `--agent Platform Engineering` and
  `--model opencode/deepseek-v4-pro` unless the user already passed them.
- Does not pass a positional `[project]` arg — the CLI's CWD-resolution
  finds the shim we just created and loads from there.
- The CLIs' home-dir configs (`~/.mimocode/config.json` and
  `~/.config/opencode/opencode.jsonc`) are loaded for OVAV defaults and
  provider config.

---

## Defaults (both CLIs)

| Field | Value |
|---|---|
| `default_agent` | `Platform Engineering` |
| `model` | `opencode/deepseek-v4-pro` |
| `small_model` | `opencode/deepseek-v4-flash` |
| `username` | `Alexander Salvador` |
| `default_provider` | `opencode` (deepseek) + `mimo` (XIOAMI free) |
| `instructions` | `<OVAV>/AGENTS.md` (canonical) |
| `permission.bash` | inherited from `opencode.json` (deny `apt install *`, `git push*`, `sudo *`, etc.) |
| `permission.external_directory` | `/home/braka/*`, `/tmp/opencode/*` allowed |

Override per-session with `OVAV_AGENT=…`, `OVAV_MODEL=…`,
`OVAV_SMALL_MODEL=…` env vars.

---

## Agents

Generated by the OVAV converter (`go-runtime/internal/convert/mimocode.go`):

- **Areas** (10): mode=`primary`, hidden=`false`, visible in TAB selector
- **Leads** (10): mode=`primary`, hidden=`true`, invocable by name
- **Teams** (60): mode=`subagent`, hidden=`true`, used via Task or @-mention

All agents carry the `OVAV_IDENTITY_GUARD v1.1` block at the head of
their prompts, suppressing native model meta-identity.

**Validation** (Go test, runs in CI):

```
go test ./go-runtime/internal/convert/ -run TestMimocodeBrain -v
```

Result: `PASS` (10 areas, 10 leads, 60 teams, 81 entries).

**Runtime validation** (this integration):

| CWD | CLI | Agents loaded | OVAV areas visible |
|---|---|---|---|
| OVAV worktree | `mimocode debug config` | 79 | 10/10 ✅ |
| CWD with shim (any non-system dir) | `mimocode debug config` | 79 | 10/10 ✅ |
| CWD with shim (any non-system dir) | `opencode debug config` | 80 | 10/10 ✅ |

---

## Memory tools (mimocode native, no BM25)

mimocode v0.1.4 stores session memory as **plain Markdown files** under
`/home/braka/Labs/mimocode/data/memory/`:

```
data/memory/
├── sessions/
│   └── ses_<id>/
│       ├── checkpoint.md   ← active context (60–250 lines, structured)
│       ├── notes.md        ← free-form notes
│       └── tasks/T*/progress.md
└── projects/
    └── <project-uuid>/
```

There is **no BM25 index** (the `.idx` files in `data/snapshot/` are git
pack indices, unrelated to search). The upstream `memory` tool's `search`
operation returns 0 matches for queries that ARE in `checkpoint.md` — it
is broken.

**OVAV wrapper provides three working memory commands** (pure grep/awk,
work without upstream's broken tool):

```bash
mimocode memory-search <query>        # grep -ri over data/memory/
mimocode memory-search <q> --project <pid>   # scope to one project
mimocode memory-list                  # sessions with line counts + Topics
mimocode memory-show <session-id>     # dump checkpoint + notes + tasks
```

Override the memory root with `MIMOCODE_MEMORY_DIR=…`.

---

## File map (canonical)

| File | Role |
|---|---|
| `~/.local/bin/mimocode` | symlink → OVAV wrapper for mimocode CLI |
| `~/.local/bin/opencode` | symlink → OVAV wrapper for opencode CLI |
| `OVAV/.mimocode/global_bin/mimocode` | canonical mimocode wrapper (thin shim) |
| `OVAV/.mimocode/global_bin/opencode` | canonical opencode wrapper (thin shim) |
| `OVAV/.mimocode/global_bin/ovav-cli-shim.sh` | shared lib (root detect, mode, shim, defaults) |
| `OVAV/.mimocode/global_config/config.json` | canonical OVAV config (projected to `mimocode.json`) |
| `OVAV/mimocode.json` | symlink → `global_config/config.json` (project-level config) |
| `OVAV/opencode.json` | opencode project-level config (in-tree, canonical) |
| `OVAV/runtimes/mimocode/agents/*.md` | 79 mimocode agents (10 areas + 9 leads + 60 teams) |
| `OVAV/runtimes/opencode/agents/*.md` | 80 opencode agents (10 areas + 10 leads + 60 teams) |
| `OVAV/.opencode/skills/`, `commands/`, `plugins/`, `themes/` | shared OVAV assets |
| `OVAV/go-runtime/internal/convert/mimocode.go` | agent converter (single source of truth) |

---

## Operational notes

- **Adding a new OVAV agent**: regenerate via the converter. Agents
  appear automatically in `runtimes/{mimocode,opencode}/agents/`.
- **Changing the OVAV default_agent / model**: edit
  `OVAV/.mimocode/global_config/config.json`. The change propagates
  immediately to both CLIs on next invocation.
- **Toggling the XIOAMI free-tier model**: keep the `mimo` provider in
  the config and use `--model mimo/mimo-auto` per-invocation.
- **Cleaning up the CWD shim**: `rm -rf CWD/.mimocode CWD/.opencode`.
  Idempotent: the shim is recreated on next invocation.
- **Disabling the shim globally**: `OVAV_NO_SHIM=1 mimocode …` or
  `OVAV_NO_SHIM=1 opencode …`.
- **Disabling auto-`--pure`**: `MIMOCODE_KEEP_LOCAL_PLUGINS=1 mimocode …`
  (useful when a project intentionally ships local plugins that depend
  on `@mimo-ai/plugin` despite its missing `exports` field).

---

## Governance

This integration is governed by the canonical OVAV policy:

- `.ovav/policy/permission_authority.json` — Thavren (this lead) is the
  governor for this surface.
- `go-runtime/internal/convert/mimocode.go` — source of truth for the
  mimocode agent format. Do not hand-edit `runtimes/mimocode/agents/*.md`;
  regenerate via the converter.
- `check_living_integrity`, `check_secrets_hygiene`,
  `check_permission_policy_drift` — all apply to this surface like any
  other OVAV-owned surface.
- Both wrappers are signed outputs; any change must be reviewed under
  the OVAV git workflow.
