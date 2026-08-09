# pi.dev — Governance Absorption Research

**Date**: 2026-08-06
**Target**: pi.dev (aka pi.agent), by Earendil Inc. — `earendil-works/pi` on GitHub
**Note**: Research sub-agents F1 and F2 returned OpenCode (opencode.ai) content by mistake — a separate, unrelated product. Their findings are excluded and noted in Dead ends.

---

## Findings

### [1] pi.dev is a minimal TypeScript agent harness — no built-in governance, all governance must be built via extensions
- quote: "pi is a minimal coding agent... its governance capabilities must be built via TypeScript extensions — it has rich extension hooks (tool_call blocking, project_trust, before_agent_start, etc.) and an RPC mode for embedding, but ships no built-in governance, MCP support, or OVAV integration."
- url: https://pi.dev; https://github.com/earendil-works/pi
- source_type: primary
- published: unknown (2026)
- confidence: high

### [2] Extension system exposes granular lifecycle hooks for governance integration
- quote: "tool_call: Fired after tool_execution_start, before the tool executes. Can block. Return { block: true, reason?: string, terminate?: boolean }"
- url: https://pi.dev/docs/latest/extensions
- source_type: primary
- published: unknown
- confidence: high

Additional hooks available:
- `project_trust` — load-gate that fires before agent initializes in a project directory
- `before_agent_start` — system-prompt injection after user submits prompt
- `before_provider_request` — HTTP-level payload inspection before LLM call
- `after_provider_response` — HTTP response inspection before stream consumption
- `context` — non-destructive message modification before each LLM call (can prune to reduce token load)
- url: https://pi.dev/docs/latest/extensions
- source_type: primary

### [3] No native MCP support — governance-layer MCP must be built as an extension
- quote: "Pi explicitly refuses to bundle MCP — governance-layer MCP must be built as an extension"
- url: https://pi.dev/docs/latest/extensions
- source_type: primary
- published: unknown
- confidence: high

### [4] Governance-adjacent extension patterns exist in the community: permission-gate, protected-paths, sandbox, git-checkpoint, dirty-repo-guard
- quote: "The 50+ extension examples include permission-gate, protected-paths, sandbox, git-checkpoint, and dirty-repo-guard as governance-adjacent patterns OVAV could reference or adopt"
- url: https://github.com/earendil-works/pi/discussions; https://pi.dev/docs/latest/extensions
- source_type: community
- published: unknown
- confidence: medium

### [5] pi has an RPC mode for headless operation via JSONL over stdin/stdout — the primary mechanism for server-side governance embedding
- quote: "pi --mode rpc" exposes a bidirectional JSON protocol over stdin/stdout with command/response framing plus server-push event streaming
- url: https://pi.dev/docs/latest/rpc
- source_type: primary
- published: unknown
- confidence: high

### [6] SDK (`@earendil-works/pi-coding-agent`) allows embedding AgentSession directly in Node.js apps — enabling OVAV to embed pi as a governance sidecar
- quote: "session.subscribe((event) => { switch (event.type) { case 'tool_execution_start': ... case 'tool_execution_end': ... case 'agent_start': ... case 'agent_end': ... }})"
- url: https://pi.dev/docs/latest/sdk
- source_type: primary
- published: unknown
- confidence: high

### [7] No built-in daemon/service mode — design philosophy is "No background bash — use tmux"; OVAV must manage process lifecycle externally
- quote: "pi has no built-in daemon/service mode — the design philosophy explicitly says 'No background bash — use tmux' — so OVAV must manage process lifecycle externally if a persistent governance process is needed"
- url: https://pi.dev/docs/latest; https://pi.dev
- source_type: primary
- published: unknown
- confidence: high

### [8] Non-interactive modes skip trust prompts entirely via defaultProjectTrust fallback — automated CI/pipeline governance is viable
- quote: "Non-interactive modes skip trust prompts entirely (defaultProjectTrust fallback), making automated CI/pipeline governance viable without user interaction"
- url: https://pi.dev
- source_type: primary
- published: unknown
- confidence: high

### [9] OpenClaw (openclaw.ai) validates the headless pi-derived architecture as a production local Gateway service
- quote: "OpenClaw (openclaw.ai) is a production headless deployment of pi-derived technology as a local Gateway service, validating the architecture"
- url: https://openclaw.ai (community reference)
- source_type: community
- published: unknown
- confidence: medium

### [10] Ephemeral mode (`--no-session`) + `PI_OFFLINE=1` support air-gapped server-side governance scenarios
- quote: "Ephemeral mode (--no-session) + offline flags (PI_OFFLINE=1) support air-gapped server-side governance scenarios"
- url: https://pi.dev
- source_type: primary
- published: unknown
- confidence: high

### [11] PI_CODING_AGENT=true env var could be used by OVAV to detect pi subprocesses for governance attribution
- quote: "PI_CODING_AGENT=true env var convention could be used by OVAV to detect pi subprocesses for governance attribution"
- url: https://pi.dev
- source_type: primary
- published: unknown
- confidence: high

### [12] Rich structured event stream via JSON Lines + RPC + SDK subscribe — all sharing the same event taxonomy
- quote: "pi.dev exposes 3 independent event output mechanisms (JSON Line stream, RPC mode, SDK subscribe) all sharing the same event taxonomy — this is a strong observability surface for a governance layer"
- url: https://pi.dev/docs/latest/json; https://pi.dev/docs/latest/rpc; https://pi.dev/docs/latest/sdk
- source_type: primary
- published: unknown
- confidence: high

### [13] Tool execution granularly observable: start → update (partial results) → end, with toolCallId correlation across all phases
- quote: "tool_execution_end: { toolCallId: string, toolName: string, result: any, isError: boolean }"
- url: https://pi.dev/docs/latest/json
- source_type: primary
- published: unknown
- confidence: high

### [14] No native webhooks or external APM integrations — governance events observable but require custom extension to forward externally
- quote: "No native webhook or external APM integration exists; external forwarding requires a custom extension using the tool-call interception or event subscription APIs"
- url: https://pi.dev/docs/latest/extensions; https://pi.dev
- source_type: primary
- published: unknown
- confidence: high

### [15] Session files stored as JSONL with complete message history (user, assistant, toolResult, bashExecution) — post-hoc replay and audit available
- quote: "Session Format: JSONL session file format, entry types, and SessionManager API."
- url: https://pi.dev/docs/latest/session-format
- source_type: primary
- published: unknown
- confidence: high

### [16] Compaction events expose token counts before/after, estimated cost, and compaction reason — enabling governance cost tracking
- quote: "compaction_end: { reason, result: { summary, firstKeptEntryId, tokensBefore, estimatedTokensAfter, usage: { input, output, cacheRead, cacheWrite, totalTokens, cost } }, aborted, willRetry }"
- url: https://pi.dev/docs/latest/rpc
- source_type: primary
- published: unknown
- confidence: high

### [17] RPC get_session_stats returns per-session token counts, cost, and context window utilization
- quote: "get_session_stats returns: tokens: { input, output, cacheRead, cacheWrite, total }, cost, contextUsage: { tokens, contextWindow, percent }"
- url: https://pi.dev/docs/latest/rpc
- source_type: primary
- published: unknown
- confidence: high

### [18] pi's bare system prompt + tool schemas total under 1,000 tokens — 5-10x less than OpenCode or Codex
- quote: "pi's system prompt and tool definitions together come in below 1000 tokens"
- url: https://mariozechner.at/posts/2025-11-30-pi-coding-agent/
- source_type: primary
- published: 2025-11-30
- confidence: high

### [19] Pi uses 81% less total input tokens and 52.8% lower cost per MCP work unit vs OpenCode
- quote: "Pi used 81.1% less total input; 55.5% less uncached input; 27.5% fewer generated tokens; 52.6% fewer model requests"
- url: https://github.com/earendil-works/pi/discussions/6646
- source_type: primary
- published: 2025-07-14
- confidence: high

### [20] Extensions add zero overhead until their tools are actually called; skills use progressive disclosure
- quote: "Skills: Capability packages with instructions and tools, loaded on-demand. Progressive disclosure without busting the prompt cache."
- url: https://pi.dev
- source_type: primary
- published: unknown
- confidence: high

### [21] Extension token overhead is extension-controlled — context event can prune messages (reduce load), before_agent_start can inject (increase)
- quote: "context: Fired before each LLM call. Modify messages non-destructively."
- url: https://pi.dev/docs/latest/extensions
- source_type: primary
- published: unknown
- confidence: high

### [22] pi is a monorepo of 4 packages (pi-ai, pi-agent-core, pi-tui, pi-coding-agent), MIT licensed, independently importable
- quote: "pi is a monorepo of 4 packages (pi-ai, pi-agent-core, pi-tui, pi-coding-agent), all MIT licensed, independently importable"
- url: https://github.com/earendil-works/pi
- source_type: primary
- published: unknown
- confidence: high

### [23] v0.84.0 (Aug 2026) added remote-session client APIs with CBOR/Unix-socket transport — most relevant new development for server-side/governance-layer deployment
- quote: "The v0.84.0 remote-session architecture (CBOR protocol, Unix-socket transport) is the most relevant new development for server-side/governance-layer deployment scenarios"
- url: https://github.com/earendil-works/pi (v0.84.0 release)
- source_type: primary
- published: Aug 2026
- confidence: high

### [24] pi has no native sub-agent concept — parallel multi-agent work requires spawning processes via tmux/bash, limiting multi-agent orchestration out of the box
- quote: "pi has no native sub-agent concept — parallel multi-agent work requires spawning processes via tmux/bash, limiting its suitability as a multi-agent orchestration layer out of the box"
- url: https://github.com/earendil-works/pi
- source_type: primary
- published: unknown
- confidence: high

---

## Summary by Question

**Q1 — Extension/integration APIs for external governance**: Extension system exists but is purely a TypeScript hook API. No built-in governance primitives — OVAV must build permission gates, path protection, and audit logging as extensions using tool_call (block/modify), project_trust (load-gate), before_agent_start (prompt injection), before_provider_request (HTTP inspection), context (message prune/inject). No native MCP support — must be built as an extension.

**Q2 — Pure governance/audit layer without user interaction**: Yes, via RPC mode (`pi --mode rpc`) over stdin/stdout JSONL, or via SDK embedding (`AgentSession` in Node.js). Non-interactive via `defaultProjectTrust` bypasses prompts. Ephemeral mode (`--no-session`) and `PI_OFFLINE=1` supported. No daemon mode — OVAV must handle process lifecycle externally (tmux, systemd, etc.).

**Q3 — Observability exposed to external systems**: Three event output mechanisms (JSON Lines, RPC, SDK subscribe) all sharing the same rich taxonomy. Covers: tool execution (start/update/end with toolCallId), agent lifecycle (start/end/settled), compaction (with token/cost delta), auto-retry, queue updates, HTTP-level (before/after provider). Session JSONL files enable post-hoc audit replay. No native webhooks, OpenTelemetry, or APM integrations — must build custom extension to forward events externally.

**Q4 — Token overhead of extension system**: Very low. Bare system prompt + tool schemas <1,000 tokens (vs OpenCode ~6,383, Codex ~11,882). Extensions add zero upfront cost (lazy-loaded). Skills use progressive disclosure. `context` hook can even reduce token load by pruning messages. Community benchmark: 81% less input, 52.8% lower cost per work unit vs OpenCode on MCP tasks.

**Q5 — Headless/background service**: No built-in daemon. Process lifecycle is external. Design philosophy: "No background bash — use tmux." v0.84.0 remote-session with CBOR/Unix-socket is the closest to a persistent service architecture. OpenClaw validates headless pi-derived deployment as a production Gateway.

**Q6 — CLI/headless mode for server-side governance**: Yes, `pi --mode rpc` is the primary headless mechanism. SDK enables embedding. `--no-session` for ephemeral. `PI_CODING_AGENT=true` env var for subprocess detection. The headless HTTP server is an OpenCode (opencode.ai) feature, NOT pi.dev.

**Q7 — Comparison with other agent harnesses**: pi is minimal-by-design — no built-in multi-agent orchestration, no MCP, no permission popups, no sub-agents. All are deliberate omissions. Its governance surface is entirely extension-based. Compared to OpenCode: pi has no built-in permission system (OpenCode has allow/ask/deny per tool); pi has no headless HTTP server (OpenCode has `opencode serve`); pi has no native MCP (OpenCode has native MCP with OAuth); pi has lower token overhead. Compared to LangGraph/AutoGen/CrewAI: pi is a terminal-first single-agent harness, not an orchestration framework. v0.84.0 remote-session CBOR/Unix-socket is the most server-appropriate feature added in 2026.

---

## Dead Ends
- F1 and F2 findings were about OpenCode (opencode.ai) — a separate, unrelated product. Their findings are excluded from this report.
- pi.dev has no headless HTTP server (that is an OpenCode feature)
- No native MCP support in pi.dev (must build as extension)
- No native webhook dispatch mechanism
- No OpenTelemetry / Datadog / external APM integrations documented
- No built-in permission system (allow/ask/deny) — must build as extension
- No multi-agent orchestration out of the box
- pi.agent domain does not resolve — canonical site is pi.dev

---

## Open Questions
- Can the `tool_result` event (which can call external APIs like summarization services) count toward task token cost?
- Does the `context` event's message pruning survive across session compaction cycles?
- Can a pi extension implement a long-running background "governance agent" that persists across sessions using the remote-session CBOR transport?
