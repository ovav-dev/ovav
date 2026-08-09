# AI Agent Systems 2026 — Best Practices Research Brief

**Date**: 2026-07-28
**Researcher**: Thavren / OVAV Platform Engineering
**Scope**: Multi-agent architectures, specialization, communication, governance, routing, memory, evaluation, security

---

## 1. Multi-agent Architectures

### 1.1 Framework Landscape (July 2026)

| Framework | Stars | Status | Key Pattern |
|-----------|-------|--------|-------------|
| **CrewAI** | 56.3k | Active | Role-playing autonomous agents, sequential/hierarchical delegation |
| **LangGraph** | 38.3k | Active | Low-level graph orchestration, durable execution, checkpointing, HITL |
| **OpenAI Agents SDK** | 28.2k | Production | Agents + handoffs + guardrails + tracing. Provider-agnostic (100+ LLMs) |
| **Google A2A** | 25.1k | Production (v1.0) | Open protocol for opaque agent↔agent communication. Linux Foundation |
| **OpenAI Swarm** | 21.9k | Replaced → Agents SDK | Educational handoff-based multi-agent. Now superseded |
| **Microsoft Agent Framework** | 12.5k | Production (v1.0) | Enterprise multi-agent. Python + .NET. Graph workflows. A2A+MCP |
| **AutoGen** | 60.1k | Maintenance | Pioneered multi-agent patterns. Superseded by MAF |
| **Anthropic Agent SDK** | N/A | Production | Claude Code as library. Subagents, hooks, skills, sessions, MCP |
| **MCP** | 8.7k | Production | Agent↔tool protocol. Not agent↔agent |

### 1.2 Consensus Patterns

**Anthropic's "Building Effective Agents"** (Dec 2024) established the industry taxonomy:

**Workflows** (predefined code paths):
1. **Prompt Chaining**: Sequential steps, each processing previous output. Gate checks.
2. **Routing**: Classify input → direct to specialized handler. Enables model tiering.
3. **Parallelization**: Sectioning (independent subtasks) or Voting (multiple perspectives).
4. **Orchestrator-Workers**: Central LLM dynamically breaks down tasks, delegates.
5. **Evaluator-Optimizer**: Generator + evaluator feedback loop (analogous to GAN).

**Agents** (LLM-driven autonomous loops):
6. **Autonomous Agent**: LLM + tools + environment feedback loop. Self-directed planning.

**Key insight**: "Success isn't about building the most sophisticated system. It's about building the right system." Start simple, add agentic complexity only when measurably better.

### 1.3 Architectural Convergence

All major frameworks now converge on:
- **Graph-based orchestration** (MAF, LangGraph) for complex workflows
- **Handoff-based delegation** (OpenAI Agents SDK, Swarm) for role switching
- **Agent-as-Tool** pattern (OpenAI, MAF, Anthropic subagents) for hierarchical composition
- **Dual runtime**: Python + TypeScript/JS across the board

### OVAV Implications
- OVAV's squad delegation model (lead → specialized squad members via skills) aligns with the industry's agent specialization pattern
- OVAV's protected branch lockdown is a governance pattern the industry is converging toward (see §4)
- OVAV should consider formalizing the routing logic as an explicit "Orchestrator" pattern rather than implicit routing

---

## 2. Agent Specialization

### 2.1 Emerging Agent Roles

Industry is converging on these specialized agent archetypes:

| Role | Example | Framework |
|------|---------|-----------|
| **Orchestrator / Triage** | Routes tasks to specialists | OpenAI triage_agent, CrewAI manager |
| **Coder / Implementer** | Writes code, runs tests | Claude Code, GitHub Copilot, SWE-bench agents |
| **Reviewer / Auditor** | Reviews code, security audit | Anthropic subagent pattern |
| **Researcher / Analyst** | Web search, evidence gathering | Deep research agents |
| **Planner / Architect** | Produces design specs, task plans | LangGraph Deep Agents |
| **Safety / Guard** | Content filtering, permission enforcement | OpenAI guardrail agents |
| **Memory / State Keeper** | Manages cross-session context | Anthropic sessions, LangGraph memory |

### 2.2 Specialization Patterns

- **Anthropic**: Subagents with `AgentDefinition` (description, prompt, tools whitelist). Main agent delegates to code-reviewer, researcher, etc.
- **OpenAI**: `Agent.as_tool()` wraps agent as callable tool. `Handoffs` for ownership transfer.
- **MAF**: Agent Skills with domain-specific knowledge bases from files/inline code.
- **CrewAI**: Role-based agents with goal + backstory. Sequential or hierarchical delegation.

### OVAV Implications
- OVAV already implements 6 specialized leads (Thavren, Sofía, Valeria, Renata, Elena, Eidren) with role-specific skills. This aligns with industry best practice.
- Need to formalize handoff contracts between leads more explicitly.
- Consider: each lead should have a published "capability card" (analogous to A2A AgentCard).

---

## 3. Agent-to-Agent Communication

### 3.1 A2A Protocol (Google / Linux Foundation)

**Status**: v1.0 released. 25.1k stars. Multi-language SDKs (Python, Go, JS, Java, .NET, Rust).

**Core concepts**:
- **Agent Card**: JSON metadata document declaring identity, capabilities, skills, endpoint, auth
- **Task-based model**: All work tracked as stateful Tasks (TASK_STATE_WORKING → COMPLETED/FAILED/CANCELED)
- **Three protocol bindings**: JSON-RPC, gRPC, HTTP/REST
- **Streaming + Push Notifications**: SSE for real-time, webhooks for async
- **Enterprise security**: OAuth2, API keys, Mutual TLS, in-task authorization

**A2A vs MCP**:
- A2A = agent ↔ agent (collaboration, task delegation)
- MCP = agent ↔ tools (tool exposure, resource access)
- Complementary: A2A agents can use MCP tools internally

### 3.2 MCP (Anthropic)

**Status**: 8.7k stars on spec repo. Widely adopted (OpenAI Agents SDK, MAF, Anthropic Agent SDK, Claude Code, MiMoCode all support MCP).

**Core**: JSON-RPC 2.0 protocol exposing Tools, Resources, Prompts. Client-server architecture.

### 3.3 Internal Agent Communication Patterns

- **OpenAI Handoffs**: Agent transfers control to another agent. Filter callbacks for conditional routing.
- **Anthropic Subagents**: Main agent invokes specialized agents via the Agent tool. Messages include `parent_tool_use_id` for traceability.
- **MAF Workflows**: Graph nodes as agents. Checkpointing for durability.

### OVAV Implications
- OVAV uses internal Go-based routing via skills (not A2A). For local-only operation this is fine.
- If OVAV ever needs to interoperate with external agents, A2A AgentCard is the standard to adopt.
- OVAV's squad delegation (workflow + skill) is functionally equivalent to Anthropic's subagent pattern.

---

## 4. Agent Governance

### 4.1 Permission Models

| Framework | Model | Mechanism |
|-----------|-------|-----------|
| **Anthropic Agent SDK** | Tool whitelist | `allowed_tools: ["Read", "Glob"]`, `disallowed_tools`, `permission_mode: "acceptEdits"` |
| **OpenAI Agents SDK** | Guardrails | Input guardrails (pre-LLM), output guardrails (post-LLM), tool guardrails. Tripwire for hard block |
| **A2A** | Enterprise auth | OAuth2, API keys, Mutual TLS. In-task authorization with scoped capabilities |

### 4.2 Guardrail Patterns (OpenAI Agents SDK)

1. **Input Guardrails**: Run before (blocking) or alongside (parallel) the main agent. Tripwire = exception, prevents execution.
2. **Output Guardrails**: Run on final agent output. Tripwire = block delivery.
3. **Tool Guardrails**: Per-function-tool input/output validation. Reject, replace output, or tripwire.

### 4.3 Hooks (Anthropic Agent SDK)

Lifecycle hooks: `PreToolUse`, `PostToolUse`, `Stop`, `SessionStart`, `SessionEnd`, `UserPromptSubmit`.

Key use case: **audit logs** — hook logs all file changes. **Security enforcement** — hook validates tool parameters before execution.

### 4.4 Industry Best Practices

- **Simplicity over complexity**: Anthropic explicitly warns against over-engineering. "Optimize single LLM calls with retrieval first."
- **Transparency**: Show agent's planning steps explicitly.
- **Poka-yoke tools**: Design tools so they're hard to misuse. Absolute paths over relative paths.
- **Sandbox testing**: Always test agents in sandboxed environments before deployment.

### OVAV Implications
- OVAV's existing governance is **ahead of the curve**: protected branches, output guard, identity guard, runtime gates, security gates.
- The `ovav-security-gates` and `ovav-runtime-gates` skills are equivalent to OpenAI's guardrail system.
- OVAV's permission_authority.json mirrors Anthropic's `allowed_tools` model.
- **Improvement area**: OVAV could add hook-style lifecycle checks (PreToolUse validation) — currently done via Go runtime calls but could be more systematic.

---

## 5. Model Routing

### 5.1 Industry Patterns

**Anthropic Recommends**:
- Routing workflow: classify complexity → easy to Haiku (cheaper), hard to Sonnet/Opus (capable)
- Explicit cost/latency vs capability tradeoff

**OpenAI Agents SDK**:
- Per-agent model selection: `Agent(model="gpt-5.4-mini")` vs `Agent(model="gpt-4o")`
- Model override at runtime: `RunConfig(model="gpt-4o")` for dynamic switching

**MAF**:
- Multi-provider support: OpenAI, Azure, Foundry, any OpenAI-compatible
- Per-agent client configuration

**MiMoCode**:
- Uses deepseek models (deepseek-v4-pro observed). OpenCode fork with model/service routing.
- Community projects: `OH-MY-MIMO-CODE` implements model routing for multi-agent workflows.
- `mimo-opencode-bridge`: OpenAI-compatible proxy to Xiaomi MiMo models (no API key needed).

### 5.2 Consensus
- **Tiered models per agent role**: Light/specialized tasks → cheap models. Complex/general tasks → premium models.
- **Provider abstraction**: All frameworks have a model client abstraction layer.

### OVAV Implications
- OVAV should consider per-lead model routing. Eidren (research) could use a different model than Thavren (platform).
- The model routing is currently implicit in MiMoCode's provider config. Could be formalized as an OVAV capability.
- MiMoCode being a Xiaomi OpenCode fork means the model backend is Xiaomi-controlled. The bridge project allows using other providers.

---

## 6. Memory Systems

### 6.1 Industry Approaches

| Framework | Short-term | Long-term | Mechanism |
|-----------|-----------|-----------|-----------|
| **LangGraph** | In-graph state | Persistent memory across sessions | Checkpointing, durable execution |
| **OpenAI Agents SDK** | Session state | SQLite/Redis/SQLAlchemy/MongoDB sessions | `ConversationHistory`, auto-managed |
| **Anthropic Agent SDK** | JSONL session files | Resumable sessions | `resume: sessionId`, fork support |
| **Anthropic API** | Prompt caching | 5min/1hr TTL cache | Automatic + explicit breakpoints |

### 6.2 Best Practices

- **Short-term**: Working memory within a task/agent run. State graph with checkpointing.
- **Long-term**: Cross-session persistence. Anthropic: JSONL session files. OpenAI: database-backed sessions.
- **Context management**: Prompt caching (Anthropic) — cache system prompt + tools, pay 90% less for cache hits.
- **Memory scoping**: Project memory vs session memory vs global memory.

### OVAV Implications
- OVAV has `ovav-memory-bridge` for memory operations. 
- OVAV's memory system (global/projects/sessions) with BM25 is already sophisticated.
- The MiMoCode community has `mimocode-mempalace` (semantic recall, knowledge graph) — could be evaluated as a plugin.
- **Gap**: OVAV doesn't have durable execution/checkpointing (LangGraph's model). If an agent run fails mid-task, there's no automatic resume.
- Anthropic's prompt caching model could inform OVAV's context economy system.

---

## 7. Evaluation & Benchmarking

### 7.1 Industry Approach

| Framework | Evaluation Method |
|-----------|------------------|
| **OpenAI Agents SDK** | Built-in tracing with visualization UI. Per-agent run traces with spans. |
| **LangGraph** | LangSmith for agent evals, observability, trajectory debugging. |
| **MAF** | OpenTelemetry integration. AF Labs for benchmarking. DevUI for debugging. |
| **Anthropic** | Explicit recommendation: measure performance, iterate. SWE-bench for coding agents. |

### 7.2 Key Patterns
- **Tracing is universal**: All frameworks provide tracing. OpenTelemetry is the standard (MAF).
- **Eval as first-class**: OpenAI's guardrails ARE evaluation functions. LangSmith's eval framework.
- **Benchmarking**: SWE-bench for coding, custom domain benchmarks for specialized agents.

### OVAV Implications
- OVAV has F0–F5 validators and `ovav_validate` for integrity scoring. This is a custom eval harness.
- **Gap**: OVAV lacks automated benchmark runs against standard agent benchmarks.
- OVAV's evaluation pipeline (Law Compliance → Stress Gate → Red Team → Semantic Drift → Decision) is a comprehensive custom eval that most frameworks don't have.
- Recommendation: publish OVAV's evaluation methodology. It's a differentiator.

---

## 8. Security

### 8.1 Prompt Injection & Tool Safety

**OpenAI Agents SDK**:
- Input guardrails for prompt injection detection (run another LLM to classify)
- Tool guardrails: validate arguments before execution, filter outputs after
- Tripwire mechanism: hard block on violation

**Anthropic Agent SDK**:
- Permission modes: `default` (ask user), `acceptEdits` (auto-approve safe ops)
- `disallowed_tools`: explicit block list
- Hooks for pre/post tool validation (e.g., block `rm -rf /`)

**MiMoCode Community**:
- `mimocode-plusplus-v2` (security hooks): Command guard + path guard + policy engine
- `mimocode-ponytail` (YAGNI enforcement): Block unnecessary dependencies
- `mimocode-rtk-hook` (token optimization): Remove noise from outputs

### 8.2 Enterprise Patterns (A2A)
- OAuth2 Authorization Code + Client Credentials + Device Code flows
- API Key security schemes
- Mutual TLS for server identity
- In-task authorization: scoped capabilities within a running task
- Agent Card signing for authenticity verification

### 8.3 Consensus
- **Defense in depth**: LLM-level guardrails + tool-level validation + hook-based enforcement
- **Principle of least privilege**: Per-agent tool whitelists, not blanket access
- **Audit trail**: Log all tool calls, file modifications, command executions

### OVAV Implications
- OVAV's security architecture is **industry-leading**: protected branches, output guard, runtime gates, security gates.
- OVAV's permission_authority.json (Thavren: full access, Eidren: read-only) implements least privilege.
- OVAV's `ovav-security-gates` with command blocking, actor audit is equivalent to Anthropic's hook system.
- The MiMoCode community security hooks (`mimocode-plusplus-v2`) could be evaluated as OVAV plugins.

---

## 9. MiMoCode / Xiaomi MiMo Ecosystem

### 9.1 What is MiMoCode?

MiMoCode is an AI coding agent CLI produced by Xiaomi, built as a fork/derivative of OpenCode (the open-source Codex CLI). It uses:
- **Models**: deepseek series (deepseek-v4-pro observed in system prompts)
- **Architecture**: OpenCode-compatible with `.mimocode/` directory structure (skills, agents, hooks, memory)
- **Distribution**: Available via npm (`@mimo-ai/cli`) and as a desktop app (MIMO Work)

### 9.2 Community Ecosystem (19 repos on GitHub)

Key community projects:
- `opencode-plusplus` (108★): Reliability harness adding context management, edit boundaries, verification gates, impact analysis, repair loops. Works with OpenCode AND MiMoCode.
- `OH-MY-MIMO-CODE` (2★): Multi-agent workflows, model routing, harness prompts for MiMoCode TUI.
- `mimo-opencode-bridge` (2★): OpenAI-compatible proxy for Xiaomi MiMo free models. Bridge to Claude Code, Cline, Continue without API keys.
- `mimocode-mempalace` (0★): Long-term memory plugin with semantic recall into system prompt.
- `mimocode-plusplus-v2`: Command guard hooks + path guard + policy engine.
- `rldyour-mimocode` (1★): Full config with build/plan/compose agents, persistent memory, skills, MCP, runtime validation.
- `MimoCodeHub` (1★): WebUI for MiMoCode CLI.

### 9.3 Relationship to OVAV

OVAV is built ON TOP of MiMoCode:
- OVAV's `.mimocode/` directory contains skills, agents, hooks
- OVAV's Go runtime (`go-runtime/`) provides governance, validators, session management
- OVAV's skills reference `.mimocode/skills/` for their base directory
- The `ovav-worktree-system` skill uses `owc`/`owd` commands defined in Go runtime

### 9.4 Key Insight

No public `github.com/xiaomi/mimocode` repository was found. MiMoCode appears to be distributed via internal Xiaomi channels (npm registry), with the community providing plugins and bridges. The GitHub topic page shows 19 community repositories.

### OVAV Implications
- OVAV is a **governor layer over MiMoCode**, not a replacement. This architecture is sound and mirrors how Anthropic's hooks/subagents layer over Claude.
- OVAV should monitor the `opencode-plusplus` project (108★) for reliability harness patterns.
- OVAV's memory system could be enhanced by evaluating `mimocode-mempalace` for semantic recall.

---

## 10. Actionable Recommendations for OVAV

### HIGH PRIORITY

1. **Formalize Agent Routing** (R-001)
   - Implement explicit routing workflow (Anthropic pattern) for OVAV's lead selection
   - Publish "agent cards" for each lead (capacity, domain, skills) → A2A AgentCard format
   - Source: Anthropic "Building effective agents" routing pattern + A2A AgentCard spec

2. **Add Hook-style Lifecycle Validation** (R-002)
   - Implement `PreToolUse` hooks for Bash safety (mirrors Anthropic Agent SDK + mimocode-plusplus-v2)
   - Implement `PostToolUse` audit logging (mirrors Anthropic Agent SDK hook example)
   - Source: Anthropic Agent SDK hooks documentation

3. **Adopt Guardrail Pattern for Cross-Area Validation** (R-003)
   - Per-lead input guardrails (blocking mode for cost optimization)
   - Per-lead output guardrails for final delivery validation
   - Source: OpenAI Agents SDK guardrails + tripwire documentation

### MEDIUM PRIORITY

4. **Evaluate Durable Execution / Checkpointing** (R-004)
   - Assess LangGraph-style checkpointing for OVAV's task execution
   - If agent execution fails mid-task, enable resume from last checkpoint
   - Source: LangGraph durable execution + Anthropic JSONL session model

5. **Implement Tiered Model Routing per Lead** (R-005)
   - Light/cheap model for routing/triage (existing role)
   - Premium model for complex implementation
   - Source: Anthropic routing pattern + OpenAI Agents SDK per-agent model config

6. **Publish OVAV Evaluation Methodology** (R-006)
   - OVAV's F0–F5 + evaluation pipeline is unique
   - Formalize as a benchmark methodology paper
   - Source: Industry gap — no framework has comprehensive multi-dimensional eval

### LOW PRIORITY / WATCH

7. **Monitor A2A Adoption** (R-007)
   - If OVAV needs external agent interoperability → adopt A2A AgentCard format
   - Source: A2A v1.0 spec (Linux Foundation), 25.1k stars

8. **Evaluate mimocode-mempalace** (R-008)
   - For enhanced semantic memory recall
   - Source: GitHub community project

9. **Adopt OpenTelemetry Tracing** (R-009)
   - MAF and LangGraph use OpenTelemetry
   - Source: MAF observability documentation

---

## Sources

1. Anthropic. "Building Effective Agents." Dec 2024. https://www.anthropic.com/research/building-effective-agents
2. Anthropic. "Agent SDK Overview." https://platform.claude.com/docs/en/agent-sdk/overview
3. OpenAI. "Agents SDK." https://github.com/openai/openai-agents-python (28.2k ★)
4. OpenAI. "Guardrails — OpenAI Agents SDK." https://openai.github.io/openai-agents-python/guardrails/
5. Microsoft. "Agent Framework." https://github.com/microsoft/agent-framework (12.5k ★)
6. Microsoft. "AutoGen." https://github.com/microsoft/autogen (60.1k ★, maintenance mode)
7. LangChain. "LangGraph." https://github.com/langchain-ai/langgraph (38.3k ★)
8. Google / Linux Foundation. "A2A Protocol." https://github.com/a2aproject/A2A (25.1k ★)
9. Google / Linux Foundation. "A2A Protocol Specification v1.0." https://a2a-protocol.org/latest/specification/
10. Anthropic. "Model Context Protocol Specification." https://github.com/modelcontextprotocol/modelcontextprotocol (8.7k ★)
11. OpenAI. "Swarm." https://github.com/openai/swarm (21.9k ★, superseded)
12. CrewAI. https://github.com/crewAIInc/crewAI (56.3k ★)
13. Anthropic. "Prompt Caching." https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
14. GitHub Topics: mimocode. https://github.com/topics/mimocode (19 repos)
15. whut09. "opencode-plusplus." https://github.com/whut09/opencode-plusplus (108 ★)
16. er-s-an. "OH-MY-MIMO-CODE." https://github.com/er-s-an/OH-MY-MIMO-CODE
17. cyberanrhy. "mimo-opencode-bridge." https://github.com/cyberanrhy/mimo-opencode-bridge
18. NDDev-it-com. "rldyour-mimocode." https://github.com/NDDev-it-com/rldyour-mimocode
19. Cipher208. "mimocode-plusplus-v2" (command guard hooks). https://github.com/Cipher208/mimocode-plusplus-v2
