# OpenCode — Governance Absorption Research

> Generated 2026-08-06 · depth: standard · workspace: research/harness-absorption/

## Executive summary

OpenCode (opencode.ai) is a substantially governance-ready substrate in 2026:

- **Governance**: Layered permission system (tool-level allow/ask/deny, per-agent overrides, `permission.task` glob controls, experimental `policies` for provider-level access, Enterprise SSO + internal AI gateway enforcement) [1][5].
- **Observability**: SSE event stream at `GET /event` + SDK `client.event.subscribe()`, 25+ plugin hook event types covering permission, tool execution, session lifecycle, and TUI; structured logging API at `POST /log`; full session audit via REST export/diff/list/delete [6][7].
- **Headless operation**: `opencode serve` exposes a full OpenAPI 3.1 HTTP server; `opencode run --auto` enables batch non-interactive execution; `opencode acp` provides stdio/JSON-RPC for agent orchestration [5][9].
- **External control**: `POST /session/:id/permissions/:permissionID` enables external approval workflows; `POST /tui/*` drives TUI programmatically; SSE stream allows external consumers to monitor all agent events [1][6].
- **Extensions**: Native MCP server support (local + remote with OAuth), 30+ community plugins, Agent Skills (on-demand SKILL.md loading), Custom Tools (TypeScript with Zod schemas, polyglot execution), full ACP protocol in Zed/JetBrains/Neovim [5][9].
- **Governance gap**: No dedicated audit log API with filtering/query; no external policy engine hooks (OPA); token overhead of memory/hooks/eventStore not publicly documented for OpenCode specifically; F3/F4 of this research covered "pi.dev" not OpenCode — treat those separately.

---

## Background & scope

Research question: What governance, observability, and external-integration capabilities does OpenCode (opencode.ai) have in 2026 for running a system like OVAV SYSTEM?

Six angles were investigated: governance architecture, event/observability system, headless operation, CLI/deployment interfaces, token overhead, and extensions. All primary sources are official OpenCode docs (opencode.ai/docs/) as of 2026-08-06. Token overhead data for OpenCode specifically was not found; comparative data from a competing system (pi.dev) is presented separately.

---

## 1. Governance Architecture

### [1] Tool-level permission system with allow/ask/deny per tool category
OpenCode enforces a `permission` config in `opencode.json` where each tool (read, edit, bash, task, skill, etc.) can be set to `"allow"`, `"ask"`, or `"deny"`. Permission keys use wildcard glob patterns — the last matching rule wins. Example: `mymcp_*` denies every tool from an MCP server; `mymcp_search: "ask"` targets a single tool. [1]
- quote: "Permission patterns use simple wildcard matching: `*` matches zero or more of any character, `?` matches exactly one character."
- url: https://opencode.ai/docs/permissions
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [2] Built-in primary agents with hardcoded permission tiers
OpenCode ships two built-in primary agents: `build` (full-access, all tools enabled) and `plan` (denies file edits and bash by default, prompts for approval). Switchable via Tab key. [1]
- quote: "Build is the default primary agent with all tools enabled. Plan is a restricted agent designed for planning and analysis... all of the following are set to ask: file edits: All writes, patches, and edits; bash: All bash commands."
- url: https://opencode.ai/docs/agents
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [3] Per-agent permission overrides merged with global config
Agent-specific permission rules in `opencode.json` or `.opencode/agents/<name>.md` are merged with global config; agent rules take precedence. Enables role-based agent definitions (e.g., a `review` agent denying edits and bash but allowing webfetch). [1]
- quote: "Agent permissions are merged with the global config, and agent rules take precedence."
- url: https://opencode.ai/docs/agents#permissions
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [4] Task subagent invocation controlled via `permission.task` glob patterns
The `permission.task` key controls which subagents an agent can spawn via the Task tool, using glob patterns. Setting `"*": "deny"` for task plus selective allows enables strict whitelisting of subagent types. [1]
- quote: "Control which subagents an agent can invoke via the Task tool with permission.task. Uses glob patterns for flexible matching."
- url: https://opencode.ai/docs/agents#task-permissions
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [5] Experimental policy system for provider-level access control
A separate `experimental.policies` array in `opencode.json` controls whether OpenCode may use a resource via `provider.use` actions with allow/deny. Operates independently of permissions; supports wildcard resource patterns. Global config policies take priority over project policies. [1]
- quote: "Policies control whether OpenCode may perform an action on a named resource. This feature is experimental... Policies are separate from permissions. Permissions control what tools can do during a session, while policies control whether OpenCode may use a resource such as an LLM provider."
- url: https://opencode.ai/docs/policies
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [6] Skill loading governed by `permission.skill` patterns with allow/ask/deny
The `skill` permission key gates which skills an agent can load, with wildcard support. Skills with `deny` are hidden entirely. Skills are loaded on-demand via the native `skill` tool. [1]
- quote: "Control which skills agents can access using pattern-based permissions in opencode.json... Patterns support wildcards: internal-* matches internal-docs, internal-tools, etc."
- url: https://opencode.ai/docs/skills#configure-permissions
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [7] Doom loop guard against repeated identical tool calls
The `doom_loop` permission key (defaults to `"ask"`) triggers when the same tool call repeats 3 times with identical input, protecting against stuck agent loops. [1]
- quote: "doom_loop — triggered when the same tool call repeats 3 times with identical input."
- url: https://opencode.ai/docs/permissions
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [8] External directory access gated separately with expand-path rules
Access to paths outside the working directory is gated by `external_directory` permission (defaults to `"ask"`). Paths use `~` or `$HOME` expansion. A common safe pattern layers `external_directory: allow` with `edit: deny`. [1]
- quote: "Use external_directory to allow tool calls that touch paths outside the working directory where OpenCode was started."
- url: https://opencode.ai/docs/permissions#external-directories
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [9] Auto mode auto-approves non-explicitly-denied permission requests
Running `opencode --auto` or `opencode run --auto` auto-approves any permission request not explicitly set to `"deny"`, while `"ask"` rules still prompt. Explicit deny rules are always enforced. [1]
- quote: "Start OpenCode with --auto to automatically approve permission requests that are not explicitly denied."
- url: https://opencode.ai/docs/permissions#auto-mode
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [10] Enterprise: Central Config + SSO + internal AI gateway enforcement
Enterprise tier supports centralized config integrating with SSO providers and routing all LLM traffic exclusively through the organization's internal AI gateway. Providers can be fully disabled via policy. [1]
- quote: "Through the central config, OpenCode can integrate with your organization's SSO provider for authentication. With the central config, OpenCode can also be configured to use only your internal AI gateway. You can also disable all other AI providers."
- url: https://opencode.ai/docs/enterprise
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [11] AGENTS.md project rules layered with global rules and precedence ordering
Project-level `AGENTS.md`/`CLAUDE.md` and global `~/.config/opencode/AGENTS.md` provide behavioral instructions with local winning over global. The `instructions` array in `opencode.json` can reference remote URLs. [1]
- quote: "When opencode starts, it looks for rule files in this order: Local files (AGENTS.md, CLAUDE.md), Global file (~/.config/opencode/AGENTS.md), Claude Code file (~/.claude/CLAUDE.md). The first matching file wins in each category."
- url: https://opencode.ai/docs/rules
- source_type: primary
- published: Aug 6 2026
- confidence: high

---

## 2. Event System & Observability

### [12] SSE event stream at `GET /event` for real-time external monitoring
`GET /event` exposes a Server-Sent Events stream. First event is `server.connected`, then bus events follow. [6]
- quote: "GET /event — Server-sent events stream. First event is server.connected, then bus events"
- url: https://opencode.ai/docs/server
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [13] JS/TS SDK provides `client.event.subscribe()` returning async iterable SSE stream
`event.subscribe()` returns an async iterable SSE stream of all bus events for programmatic consumption. [6]
- quote: "event.subscribe() — Server-sent events stream. Response: Server-sent events stream. Example: for await (const event of events.stream) { console.log('Event:', event.type, event.properties) }"
- url: https://opencode.ai/docs/sdk
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [14] Plugin system exposes 25+ named event types across 8 categories
Plugin hooks cover: Command (command.executed), File (file.edited, file.watcher.updated), LSP (lsp.client.diagnostics, lsp.updated), Message (message.removed, message.updated), Permission (permission.asked, permission.replied), Server (server.connected), Session (session.created/compacted/deleted/diff/error/idle/status/updated), Shell (shell.env), Tool (tool.execute.after, tool.execute.before), TUI (tui.prompt.append, tui.command.execute, tui.toast.show). [6]
- quote: "Plugins can subscribe to events as seen below..."
- url: https://opencode.ai/docs/plugins
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [15] Tool execution intercept hooks enable external monitoring and blocking
`tool.execute.before` can throw to block a tool call; `tool.execute.after` can inspect output. Example blocking .env file reads: `if (input.tool === 'read' && output.args.filePath.includes('.env')) { throw new Error('Do not read .env files') }`. [6]
- quote: "If input.tool === 'read' and output.args.filePath.includes('.env') { throw new Error('Do not read .env files') }"
- url: https://opencode.ai/docs/plugins
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [16] Structured logging API at `POST /log` for observability integration
`POST /log` accepts `{ service, level, message, extra? }` with levels debug/info/warn/error. [6]
- quote: "POST /log — Write log entry. Body: { service, level, message, extra? }. Response: boolean"
- url: https://opencode.ai/docs/server
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [17] Full session audit via REST API and CLI: list, export, diff, summarize, delete
REST: `GET /session` (list), `GET /session/:id/diff` (diff), `POST /session/:id/summarize`, `DELETE /session/:id`. CLI: `opencode export [sessionID]`, `opencode session list`. [6]
- quote: "GET /session — List all sessions; DELETE /session/:id — Delete a session and all its data; GET /session/:id/diff — Get the diff for this session; POST /session/:id/summarize — Summarize the session; opencode export [sessionID] — Export session data as JSON; opencode session list — List all OpenCode sessions"
- url: https://opencode.ai/docs/server; https://opencode.ai/docs/cli
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [18] Experimental event system gated behind `OPENCODE_EXPERIMENTAL_EVENT_SYSTEM` env var
Advanced event capabilities require the `OPENCODE_EXPERIMENTAL_EVENT_SYSTEM` environment variable to be set. [6]
- quote: "OPENCODE_EXPERIMENTAL_EVENT_SYSTEM — Enable experimental event system"
- url: https://opencode.ai/docs/cli
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [19] `experimental.session.compacting` hook for context injection during compaction
Fires before the LLM generates a continuation summary; allows plugins to inject domain-specific context into session compaction. [6]
- quote: "Customize the context included when a session is compacted — The experimental.session.compacting hook fires before the LLM generates a continuation summary. Use it to inject domain-specific context."
- url: https://opencode.ai/docs/plugins
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [20] Permission events fire on every access-control decision for audit trail
`permission.asked` and `permission.replied` events track all authorization decisions. [6]
- quote: "Permission Events: permission.asked, permission.replied"
- url: https://opencode.ai/docs/plugins
- source_type: primary
- published: Aug 6 2026
- confidence: high

---

## 3. Headless / Governance Layer Operation

### [21] `opencode serve` runs a production HTTP server with OpenAPI 3.1 spec
`opencode serve` exposes a full-featured HTTP API server (default `127.0.0.1:4096`) with mDNS discovery, HTTP basic auth (`OPENCODE_SERVER_PASSWORD`), and CORS configuration. OpenAPI spec at `/doc`. [5][9]
- quote: "The server exposes an OpenAPI 3.1 spec endpoint... Use the opencode server to interact with opencode programmatically."
- url: https://opencode.ai/docs/server/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [22] `opencode run` enables non-interactive batch execution with `--auto` mode
`opencode run` executes scripts/non-interactive workloads; `--auto` auto-approves non-deny permission requests. [1]
- quote: "Start OpenCode with --auto to automatically approve permission requests that are not explicitly denied."
- url: https://opencode.ai/docs/permissions#auto-mode
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [23] `opencode acp` provides stdio/JSON-RPC for editor and external orchestration
`opencode acp` starts the agent as a JSON-RPC subprocess over stdio, fully compatible with the Agent Client Protocol (ACP). Enables integration with Zed, JetBrains, Avante.nvim, and CodeCompanion.nvim. [5]
- quote: "OpenCode supports the Agent Client Protocol (ACP), allowing you to use it directly in compatible editors and IDEs."
- url: https://opencode.ai/docs/acp/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [24] External permission approval via REST API enables governance workflows
`POST /session/:id/permissions/:permissionID` with body `{ response, remember? }` allows external systems to approve/deny permission requests programmatically. [1]
- quote: "POST /session/:id/permissions/:permissionID — Respond to a permission request. Body: { response, remember? }."
- url: https://opencode.ai/docs/server
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [25] TUI can be driven programmatically via `POST /tui/*` endpoints
`POST /tui/appendPrompt`, `POST /tui/submitPrompt`, `POST /tui/showToast`, `POST /tui/executeCommand` allow external systems to drive the TUI. `GET /tui/control/next` and `POST /tui/control/response` support interactive TUI control. [5][9]
- quote: "POST /tui/* (drive TUI programmatically: append-prompt, submit-prompt, show-toast, execute-command)"
- url: https://opencode.ai/docs/server/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [26] No dedicated single-toggle "governance mode" — governance is compositionally achieved
There is no single governance mode switch. Governance is achieved by composing per-agent permissions, `--auto` mode, Enterprise central config, `experimental.policies`, and plugin hooks. [1][5]
- url: https://opencode.ai/docs/
- source_type: primary
- published: Aug 6 2026
- confidence: high

---

## 4. CLI Mode & Deployment

### [27] GitHub Actions integration via `anomalyco/opencode/github@latest`
OpenCode ships as a GitHub Action for event-driven CI/CD workflows. [5]
- url: https://opencode.ai/docs/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [28] `@opencode-ai/sdk` npm package for programmatic TypeScript control
`npm install @opencode-ai/sdk` provides `createOpencode()` to spawn server+client, `createOpencodeClient()` to attach to existing server. Full session management, file operations, TUI control, structured output with retry, and SSE event subscription. [5][9]
- quote: "The opencode JS/TS SDK provides a type-safe client for interacting with the server."
- url: https://opencode.ai/docs/sdk/
- source_type: primary
- published: Aug 6 2026
- confidence: high

---

## 5. Extensions & External Integrations

### [29] Native MCP server support (local stdio + remote HTTP with OAuth)
OpenCode MCP supports local child-process servers (`type: "local"`) and remote HTTP servers (`type: "remote"`). Remote servers get automatic OAuth 2.0 Dynamic Client Registration (RFC 7591), 401 detection, and token storage at `~/.local/share/opencode/mcp-auth.json`. MCP servers are available globally or per-agent via glob pattern overrides. Org-level defaults can be pushed via `.well-known/opencode` endpoint. [5]
- quote: "OpenCode supports both local and remote servers... MCP servers are automatically available to the LLM alongside built-in tools."
- url: https://opencode.ai/docs/mcp-servers/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [30] Plugin system with 30+ event hooks; custom tools via Zod schema
Plugins (JS/TS modules in `.opencode/plugins/`, `~/.config/opencode/plugins/`, or npm) hook into tool execution, session lifecycle, file watching, LSP, shell env, TUI, permission events, and more. Custom tools use `tool()` helper with Zod schema args; can invoke scripts in any language via Bun shell utilities. [5][9]
- quote: "Plugins allow you to extend OpenCode by hooking into various events and customizing behavior."
- url: https://opencode.ai/docs/plugins/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [31] 30+ community plugins covering observability, auth, workflow, and AI platform integration
Ecosystem plugins include: `opencode-sentry-monitor` (Sentry AI monitoring), `opencode-helicone-session` (request grouping), `opencode-wakatime` (usage tracking), `opencode-tavily` (web search), `opencode-jfrog-plugin` (JFrog), `opencode-firecrawl` (web scraping), `opencode-vibeguard` (PII redaction), `opencode-supermemory` (cross-session memory), `opencode-conductor` (Spec→Plan→Implement lifecycle), `opencode-background-agents` (async delegation), `opencode-worktree` (git worktree management). [5]
- quote: "A collection of community projects built on OpenCode."
- url: https://opencode.ai/docs/ecosystem/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [32] Agent Skills: on-demand SKILL.md loading with per-skill permission controls
Skills are `.opencode/skills/<name>/SKILL.md` (project), `~/.config/opencode/skills/<name>/SKILL.md` (global), or Claude-compatible paths. Loaded lazily via the `skill` tool; `permission.skill` patterns control access. Skills are OVAV-compatible format (SKILL.md with YAML frontmatter). [5]
- quote: "Agent skills let OpenCode discover reusable instructions from your repo or home directory. Skills are loaded on-demand via the native `skill` tool."
- url: https://opencode.ai/docs/skills/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [33] Full ACP protocol support in Zed, JetBrains, Avante.nvim, CodeCompanion.nvim
OpenCode's ACP enables editor-native agent integration. Zed uses `agent_servers` config; JetBrains uses `acp.json`; Avante.nvim uses `acp_providers`; CodeCompanion.nvim uses adapter pattern. Full feature parity except `/undo`/`/redo`. [5]
- quote: "OpenCode supports the Agent Client Protocol (ACP), allowing you to use it directly in compatible editors and IDEs."
- url: https://opencode.ai/docs/acp/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [34] Community projects include Discord bot, mobile UI, multi-agent orchestration harness
Beyond plugins: `kimaki` (Discord bot for session control), `portal` (mobile-first web UI over Tailscale/VPN), `opencode-workspace` (16-component orchestration harness), `ai-sdk-provider-opencode-sdk` (Vercel AI SDK provider), `micode`/`octto` (brainstorm/plan/implement workflows). [5][9]
- quote: "A collection of community projects built on OpenCode."
- url: https://opencode.ai/docs/ecosystem/
- source_type: primary
- published: Aug 6 2026
- confidence: high

### [35] Enterprise: MCP defaults via `.well-known`, SSO, internal AI gateway, private NPM registry
Enterprise central config pushes default MCP server configurations org-wide with opt-in semantics. SSO integration, internal AI gateway enforcement, and private NPM registry (JFrog Artifactory, Nexus) via Bun's native `.npmrc`. [5][9]
- quote: "With the central config, OpenCode can also be configured to use only your internal AI gateway. You can also disable all other AI providers."
- url: https://opencode.ai/docs/enterprise/
- source_type: primary
- published: Aug 6 2026
- confidence: high

---

## Open Questions

1. **Token overhead**: OpenCode's token overhead for memory/hooks/eventStore is not publicly documented. A competing system (pi.dev) benchmarks at ~1,148 tokens fresh-request overhead vs OpenCode's ~6,383 [single source]. OVAV should measure this directly against its own governance payload size.
2. **No external policy engine hooks**: No OPA/OpenPolicyAgent or equivalent external policy evaluation is documented. Governance is config-driven inside OpenCode; external systems can only approve/deny via REST API.
3. **No dedicated audit log API with filtering**: Session audit is available via full session export/diff, but no granular event query API exists. OVAV would need to ingest the SSE stream and maintain its own event store.
4. **`eventStore` concept**: OVAV references an `eventStore` concept that does not appear in OpenCode's public docs — the equivalent is the SSE stream + plugin hook system. OVAV's own `go-runtime/internal/ows/handlers.go` likely clarifies how OVAV binds to OpenCode's event bus.
5. **F3/F4 research covered "pi.dev" not OpenCode**: Two of five research angles (F3, F4) investigated pi.dev (pi.dev) which appears to be a separate coding agent project. Token overhead and headless operation data from those findings should not be assumed to apply to OpenCode.
6. **Experimental event system scope**: `OPENCODE_EXPERIMENTAL_EVENT_SYSTEM` enables capabilities beyond the documented 25+ hook types, but the delta is not described in public docs.

---

## Sources

[1] OpenCode Permissions — https://opencode.ai/docs/permissions (Aug 6 2026)
[2] OpenCode Agents — https://opencode.ai/docs/agents (Aug 6 2026)
[3] OpenCode Policies — https://opencode.ai/docs/policies (Aug 6 2026)
[4] OpenCode Skills — https://opencode.ai/docs/skills (Aug 6 2026)
[5] OpenCode Server — https://opencode.ai/docs/server (Aug 6 2026)
[6] OpenCode Plugins — https://opencode.ai/docs/plugins (Aug 6 2026)
[7] OpenCode SDK — https://opencode.ai/docs/sdk (Aug 6 2026)
[8] OpenCode MCP Servers — https://opencode.ai/docs/mcp-servers/ (Aug 6 2026)
[9] OpenCode Ecosystem — https://opencode.ai/docs/ecosystem/ (Aug 6 2026)
[10] OpenCode Enterprise — https://opencode.ai/docs/enterprise (Aug 6 2026)
[11] OpenCode ACP — https://opencode.ai/docs/acp/ (Aug 6 2026)
[12] OpenCode Custom Tools — https://opencode.ai/docs/custom-tools/ (Aug 6 2026)
[13] OpenCode CLI — https://opencode.ai/docs/cli (Aug 6 2026)
