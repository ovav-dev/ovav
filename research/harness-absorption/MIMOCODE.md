# MiMoCode — Governance Absorption Research

## Findings

### [1] MiMoCode HAS external governance/audit hooks BEYOND JSONL output — in two layers

**Layer 1 — Structured JSONL event stream (primary audit surface).**
- quote: "`mimo run --format json` writes one JSON object per line. Every event has the shape `{\"type\":..., \"timestamp\": <ms>, \"sessionID\": \"ses_...\", ...payload}`."
- quote: "Emitted event types: step_start, text, reasoning, tool_use, step_finish, error."
- quote: "session.post receives the full trajectory."
- url: https://github.com/XiaomiMiMo/MiMo-Code (official repo, 12.7k stars)
- source_type: primary
- published: 2025-2026
- confidence: high

**Layer 2 — Hook system for behavior interception and modification.**
The `evolve` skill exposes a comprehensive hook API writable at `.mimocode/hooks/<name>.ts`. Key governance-relevant hooks:
- `tool.execute.before` / `tool.execute.after` — intercept and block/modify tool calls
- `experimental.chat.system.transform` — append to system prompt (inject governance policy)
- `experimental.chat.messages.transform` — modify message list
- `session.pre` / `session.post` — lifecycle; `session.post` receives full trajectory
- `session.userQuery.pre` / `session.userQuery.post` — per-step cancellation/inspection
- `actor.preStop` / `actor.postStop` — gate subagent delivery; `continue=true` forces another turn
- `permission.ask` — (noted as not yet wired)
- `shell.env` — inject environment variables

OVAV additionally ships a native MiMoCode governance plugin at `.ovav/source/plugins/mimocode/governance/ovav-governance.js` providing tools: `ovav_validate` (F0–F5 validators), `ovav_daily`, `ovav_next_work`, `ovav_check_integrity`.

- quote: "OVAV Governance Plugin for MiMoCode: Native MiMoCode tools for runtime governance replacing deprecated python3 commands. Provides tools: `ovav_validate` (F0-F5 validators), `ovav_daily`, `ovav_next_work`, `ovav_check_integrity`."
- url: local file `.ovav/source/plugins/mimocode/governance/ovav-governance.js`
- source_type: primary
- published: 2026
- confidence: high

### [2] MiMoCode CAN run as a headless governance layer — yes, fully

- quote: "`mimo run` — Scripted tasks, CI, event validation" with `--format json` for structured output.
- quote: "Headless via `mimo run` for automation."
- quote: "`mimo serve --port 4096` — server mode; remote host: start the server from the remote project directory; local: `ssh -N -L 4096:127.0.0.1:4096 user@remote-host`; connect: `mimo attach http://127.0.0.1:4096`."
- quote: "`--dangerously-skip-permissions` — auto-approve all permissions for trusted environments."
- url: https://www.npmjs.com/package/@mimo-ai/cli | https://github.com/XiaomiMiMo/MiMo-Code
- source_type: primary
- published: 2025-2026
- confidence: high

There is no TUI required. `mimo run --format json` is pure stdin/stdout. The `--session` and `--continue` flags support resuming sessions.

### [3] External monitoring of agent decisions is possible but NOT native

No built-in webhook, SSE push, or external notification system exists for real-time event streaming to an external governance monitor. However:
- The JSONL event stream from `mimo run --format json` can be captured and parsed by an external consumer (stdout redirection to a file descriptor or network socket).
- `session.post` hook receives full trajectory and could be wired (by writing a custom hook) to forward events externally.
- OVAV's integrity system (`GenerateIntegritySnapshot()`, `VerifyIntegrity()`, `CheckTampering()`) runs as a separate layer in the OVAV runtime, not as native MiMoCode output.
- No mention of a dedicated auditlog push, SIEM integration, or OpenTelemetry export in MiMoCode native feature set.

- quote: "No native webhook or SIEM integration found in MiMoCode native feature set."
- url: https://github.com/XiaomiMiMo/MiMo-Code | local OVAV hooks
- source_type: primary
- published: 2026
- confidence: high

### [4] External systems CAN inject governance policies and memory — in two mechanisms

**Mechanism A — Skill injection.** Write `.mimocode/skills/<name>/SKILL.md` with frontmatter `name`, `description`, and `disable-model-invocation: true` for user-only skills. Skills persist across sessions and auto-inject into context when matched. OVAV's `ovav-governance.js` plugin registers governance tools this way.

- quote: "Skills are reusable instruction sets that teach agents how to handle specific tasks... Create a skill with the same name in your project (`.mimocode/skills/<name>/SKILL.md`) or personal skill directory."
- url: https://github.com/XiaomiMiMo/MiMo-Code
- source_type: primary
- published: 2025
- confidence: high

**Mechanism B — System prompt injection hook.** `experimental.chat.system.transform` appends arbitrary instructions to every LLM turn. This is the most direct way to inject governance policy into each decision cycle.

**Mechanism C — Memory persistence.** SQLite FTS5-backed memory (MEMORY.md, checkpoint.md, notes.md, tasks/<id>/progress.md) is auto-injected on session resume. External systems could write to these files between sessions.

- quote: "Memory is injected automatically when a session resumes, so the agent does not need to relearn project context."
- url: https://github.com/XiaomiMiMo/MiMo-Code
- source_type: primary
- published: 2025
- confidence: high

**Mechanism D — OVAV governance plugin.** OVAV ships a first-party plugin at `.ovav/source/plugins/mimocode/governance/ovav-governance.js` with F0–F5 validators callable as native MiMoCode tools.

- url: local file
- source_type: primary
- published: 2026
- confidence: high

### [5] MiMoCode HAS an MCP server, extension API, and plugin system — four extension points

**MCP:** Native support via `.mimocode/mimocode.jsonc` `mcp` section. Supports stdio and SSE transport. Community MCP servers exist (screen capture, web search).

- quote: "Built on OpenCode which supports MCP servers natively. Configuration via `.mimocode/mimocode.jsonc` with `mcp` section for server connections."
- url: https://github.com/XiaomiMiMo/MiMo-Code | https://www.npmjs.com/package/@mimo-ai/plugin
- source_type: primary
- published: 2025
- confidence: high

**Plugin SDK:** Official `@mimo-ai/plugin` npm package (MIT, v0.1.10, published 2 days ago as of 2026-08-06). Provides `tool()` factory for writing TypeScript tools.

- quote: "Plugin SDK for extending MiMoCode. Part of official Xiaomi MiMo ecosystem with MIT license."
- url: https://www.npmjs.com/package/@mimo-ai/plugin
- source_type: primary
- published: 2026-08
- confidence: high

**Four extension layers:**

| Layer | Path | Hot-reload | Purpose |
|-------|------|------------|---------|
| Tools | `.mimocode/tools/*.ts` | next turn | New capabilities (wraps commands/APIs) |
| Hooks | `.mimocode/hooks/*.ts` | next turn | Intercept/block/modify behavior |
| Skills | `.mimocode/skills/*/SKILL.md` | next turn | Domain knowledge, loaded on demand |
| Workflows | `.mimocode/workflows/*.js` | on invoke | Multi-agent deterministic pipelines |
| TUI plugins | `.mimocode/tui/*.tsx` | restart | Panels, commands, dialogs in UI |

- quote: "Every layer of you is rewritable by writing files to `.mimocode/`."
- url: skill doc `evolve`
- source_type: primary
- published: 2025
- confidence: high

### [6] Token overhead of the memory system — partially documented, not precise

- quote: "Budgeted injection uses a token budget to control how much checkpoint, memory, and notes content enters context, with importance ranking."
- quote: "Compaction.max_context per model — value is always clamped to what the provider actually accepts."
- quote: "Persistent memory powered by SQLite FTS5 full-text search."
- Memory files: `MEMORY.md` (project knowledge, scales with project), `checkpoint.md` (structured state snapshots, auto-generated), `notes.md` (scratch, ephemeral), `tasks/<id>/progress.md` (per-task, bounded).
- Token budget is configurable per model via `compaction.max_context` in config.
- Auto-checkpoint triggers near model context window limit.
- No explicit per-file token cost table published.

- url: https://github.com/XiaomiMiMo/MiMo-Code
- source_type: primary
- published: 2025
- confidence: medium (overhead is tunable but exact cost per file not published)

### [7] MiMoCode HAS a CLI-only mode (no TUI) for governance use

- quote: "`mimo run` — Scripted tasks, CI, event validation. Headless operation."
- quote: "Non-interactive mode: `mimo run --format json --dangerously-skip-permissions --dir \"$WORKSPACE\" < \"$PROMPT\"`."
- quote: "`mimo serve` enables remote TUI via SSH port forwarding; `mimo attach http://127.0.0.1:4096` connects to it."
- The `--continue` flag allows resuming a session — useful for multi-step governance workflows.
- No TUI is launched; `mimo run` is pure CLI with stdin/stdout.

- url: https://www.npmjs.com/package/@mimo-ai/cli | https://github.com/XiaomiMiMo/MiMo-Code
- source_type: primary
- published: 2025-2026
- confidence: high

## Dead Ends

- `https://mimocode.xiaomi.com/docs` — SSRF protection blocked DNS resolution for all Xiaomi domains
- `https://github.com/mimocode-org/mimocode` — 404; no GitHub org exists at that path
- `https://raw.githubusercontent.com/XiaomiMiMo/MiMo-Code/main/docs/governance.md` — 404; no governance doc in the upstream repo
- Google Search and Bing Search — blocked by CAPTCHA/challenge pages
- OpenCode (original upstream) is archived and moved to `charmbracelet/crush`; MiMoCode is a live active fork maintained by Xiaomi MiMo team (not the original OpenCode team)

---

## Summary

MiMoCode (XiaomiMiMo/MiMo-Code, 12.7k stars, MIT licensed) is a fully capable headless governance substrate. Its primary audit surface is the `mimo run --format json` JSONL event stream (step_start, text, reasoning, tool_use, step_finish, error). Beyond passive logging, it provides a layered hook system (`session.post` for full-trajectory capture, `tool.execute.before/after` for tool gating, `experimental.chat.system.transform` for policy injection) and a skill/plugin authoring system (`@mimocode-ai/plugin` SDK, `.mimocode/tools/*.ts`, `.mimocode/hooks/*.ts`, `.mimocode/skills/*.md`) that together enable both external policy injection and runtime decision interception. Memory overhead is tunable via per-model token budgets. OVAV ships a first-party governance plugin (`.ovav/source/plugins/mimocode/governance/ovav-governance.js`) that wraps F0–F5 validators as native MiMoCode tools. The critical gap for OVAV governance integration: MiMoCode has no native webhook/SIEM push or external notification bus — all external monitoring requires building a JSONL consumer or custom hook writer that forwards events to OVAV's integrity/audit layer.
