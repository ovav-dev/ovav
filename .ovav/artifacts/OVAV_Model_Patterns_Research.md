# OVAV Model Patterns Research
## Fable 5 & Opus 5 — Architectural & Code-Quality Intelligence for Governance Absorption

**Produced by:** OVAV Research Intelligence (Eidren's team)
**Date:** 2026-07-28
**Worktree:** feature/bencharmk-attack (HEAD: 54023d9, hace 28 min)
**Classification:** Internal — Governance Intelligence

---

## Section 1: Fable 5 & Opus 5 Documented Capabilities

### 1.1 Model Positioning

**Source:** Anthropic official documentation (docs.anthropic.com, published June 2026)

- **Claude Fable 5** (`claude-fable-5`): Anthropic's most capable widely released model. Built for "the most demanding reasoning and long-horizon agentic work." $10/M input, $50/M output. 1M token context, 128k max output. Released June 9, 2026. Price/performance leader on Artificial Analysis Intelligence Index (score 60/61).
- **Claude Opus 5** (`claude-opus-5`): Flagship model for "demanding reasoning, coding, and long-horizon agentic work." Particularly strong at end-to-end software tasks, code review, bug finding, and coordinating parallel subagents. $5/M input, $25/M output. Released July 24, 2026. Tops the Artificial Analysis Intelligence Index at score 61.

**Source:** OpenRouter product pages; Artificial Analysis Intelligence Index v4.1 (18 evaluations including Terminal-Bench v2.1, SciCode, GPQA Diamond, Humanity's Last Exam).

---

### 1.2 Adaptive Thinking — The Core Architectural Behavior

**Source:** Anthropic Extended Thinking documentation (docs.anthropic.com/en/docs/build-with-claude/thinking)

Both Fable 5 and Opus 5 use **adaptive thinking** (always on — cannot be disabled on Fable 5; disabled only at `effort ≤ high` on Opus 5). This is the single most architecturally significant capability:

- Thinking blocks (encrypted `signature` field) are preserved across tool-use turns. The model reasons between tool calls ("interleaved thinking") — it can reason about a tool result before making the next tool call.
- Raw chain-of-thought is **never returned** — only summarized thinking (`display: "summarized"`) or an empty field (`display: "omitted"`, the default). This prevents reasoning extraction attacks.
- On Fable 5, attempting to elicit internal reasoning as response text triggers `stop_details.category: "reasoning_extraction"` refusal.
- Thinking blocks from prior turns are **kept in context** by default (keep-all regime on Opus 5/Fable 5), enabling cache hits across multi-step agentic workflows.

**OVAV Relevance:** OVAV has no validator that inspects whether an AI model is actually using interleaved thinking in multi-step tasks, or whether thinking blocks are being preserved correctly across tool calls. This is a **governance blind spot** for code quality.

---

### 1.3 Effort Parameter — Token Budget Intelligence

**Source:** Anthropic Effort documentation (docs.anthropic.com/en/docs/build-with-claude/effort)

Fable 5 and Opus 5 expose an `effort` parameter with 5 levels: `low`, `medium`, `high` (default), `xhigh`, `max`.

Key behavioral facts:
- **`xhigh`**: "Extended capability for long-horizon work." Recommended starting point for coding and agentic use cases over 30 minutes. Token budgets in the millions.
- **`max`**: Absolute maximum capability, no token constraints. Reserved for genuinely frontier problems.
- Effort controls **all tokens** — text responses, tool calls, and thinking. At `xhigh`/`max`, thinking cannot be disabled.
- On Opus 5 at `xhigh` or `max`, `thinking: {type: "disabled"}` returns a 400 error.
- Lower effort (`low`, `medium`) still performs well and often exceeds `xhigh` on prior models.

**OVAV Relevance:** OVAV has no mechanism to enforce that models are invoked at appropriate effort levels for the task type. A coding agent could be run at `low` effort for a complex multi-file implementation — and OVAV would not know. The `model_budget_policy.yaml` exists but does not encode effort-level governance per task class.

---

### 1.4 End-to-End Software Task Execution

**Source:** Anthropic product descriptions; OpenRouter Fable 5 page ("autonomous knowledge work," "long-running, complex, and asynchronous tasks," "executing well-scoped tasks with few mistakes, automatically self-correcting through verification loops")

The documented capabilities that distinguish Fable 5/Opus 5 from smaller models:
1. **Verification loops**: The model "automatically self-corrects through verification loops" — it checks its own outputs before declaring done.
2. **Few mistakes on well-scoped tasks**: Demonstrates task boundary awareness.
3. **Long-horizon execution**: Can execute tasks over hours/days without frequent human check-ins.
4. **Parallel subagent coordination**: Opus 5 specifically noted for "coordinating parallel subagents."

**OVAV Relevance:** OVAV's validators do not verify whether a coding agent performed self-verification (e.g., ran tests, checked types, validated output) before declaring completion. The benchmark ASR of 0% with OVAV vs 45-70% RAW is about dangerous task refusal — not about whether the model verified its own work.

---

### 1.5 Safety Classifiers & Refusal Behavior (Fable 5)

**Source:** Anthropic "Redeploying Fable 5" blog post (June 30, 2026); official Fable 5/Mythos 5 introduction

Fable 5 includes safety classifiers that can **decline requests** at `stop_reason: "refusal"` (HTTP 200, not an error). Key architectural facts:

- Safety margin approach: classifiers trigger on ambiguous requests even if probably benign, to avoid missing genuinely harmful ones. This causes false positives on routine cybersecurity tasks.
- Classifier improvements are trained in response to specific bypass techniques (e.g., the Amazon research report bypass was addressed with a new classifier blocking 99%+ of that technique).
- **Every model tested** (including Haiku 4.5, Sonnet 4.6, Opus 4.6-4.8) could produce the same exploit demonstration as Fable 5 for the reported case — demonstrating that Fable 5's unique value is not capability but the combination of high capability + calibrated refusals.
- Refusals return structured metadata: `stop_details.category` (e.g., `"reasoning_extraction"`) so applications can handle them programmatically.

**OVAV Relevance:** OVAV has no mechanism to handle model refusals gracefully. If a governed model refuses a legitimate coding task, OVAV has no fallback routing, no retry logic, and no notification. This is a critical gap for production reliability.

---

### 1.6 Tool Use & Context Management

**Source:** Anthropic Thinking + Tool Use documentation

- Tool use with thinking: thinking blocks appear **between** tool calls (interleaved). Claude can reason about a tool result before deciding the next action.
- Thinking blocks must be passed back **complete and unmodified** in tool-use turns — the API rejects modified thinking blocks with a 400 error.
- On keep-all models (Fable 5, Opus 5, Sonnet 5): prior turns' thinking is preserved in context and cached incrementally across the assistant turn.
- Context clearing via `clear_thinking_20251015` context-editing strategy.
- Prompt caching interactions: configuration changes (`effort`, `thinking` type) **invalidate cache breakpoints**.

**OVAV Relevance:** OVAV does not validate whether tool-use conversations are being managed correctly (thinking blocks preserved, cache breakpoints respected). This is invisible to OVAV's current governance layer.

---

### 1.7 Data Type & Schema Rigor

**Source:** Anthropic documentation (general capability descriptions); OpenRouter model pages

Fable 5 and Opus 5 are documented to handle:
- **Structured outputs** with strong type adherence
- **Schema-guided generation** via tool definitions
- **Multi-modal inputs** (text + image) with consistent type handling
- 1M token context with consistent type/schema awareness across long documents

**Evidence from benchmark data:** Terminal-Bench v2.1 (agentic coding + terminal use), SciCode (coding), and GPQA Diamond (scientific reasoning) all require rigorous type handling. The top scores on these benchmarks by Opus 5/Fable 5 indicate strong typed-output capabilities.

**OVAV Relevance:** OVAV has no validator that checks whether generated code has proper type annotations, schema definitions, or validation layers. Code quality is currently measured only by coverage and secrets hygiene — not by type rigor.

---

### 1.8 Long-Context Coherence

**Source:** Anthropic context window documentation; AA-LCR benchmark description

- 1M token context window (same tokenizer across Fable 5, Opus 5, Sonnet 5 — ~30% more tokens than pre-Opus 4.7 models for the same text).
- AA-LCR (Artificial Analysis Long Context Reasoning) is one of the 9 evaluations in the Intelligence Index.
- "AA-LCR measures long context reasoning" — Opus 5 scores highest, indicating the ability to maintain coherence, reference resolution, and cross-document consistency over very long contexts.

**OVAV Relevance:** OVAV has no mechanism to enforce that long-context tasks maintain coherence (e.g., cross-referencing earlier sections of a large file, maintaining consistent variable naming across a 50k-token context). This is invisible to current validators.

---

### 1.9 Self-Correction & Error Recovery

**Source:** OpenRouter Fable 5 description ("automatically self-correcting through verification loops"); Anthropic Effort documentation ("at lower effort levels, Claude scopes its work to what was asked rather than doing more than requested")

Fable 5 in particular demonstrates:
- **Verification loops**: It checks intermediate results before proceeding.
- **Error recovery**: When something fails, it backtracks and tries alternative approaches.
- **Scoped task awareness**: At lower effort, it respects task boundaries rather than overreaching.

**OVAV Relevance:** OVAV has no validator that checks whether a coding agent self-corrected after producing a failing test, or whether it recovered from an error gracefully. The benchmark only measures dangerous task refusal — not error recovery quality.

---

### 1.10 Security-Aware Coding

**Source:** Anthropic "Redeploying Fable 5" — Fable 5's safety classifiers, the safety margin concept, the jailbreak severity framework (capability gain, breadth, ease of weaponization, discoverability)

Fable 5 has explicit security-aware behaviors:
- Classifiers detect when code being requested could be used for exploitation (even if the request is routine defensive work).
- The model refuses borderline cases out of abundance of caution.
- Jailbreak severity is scored across 4 dimensions: capability gain, breadth, ease of weaponization, discoverability.

**OVAV Relevance:** OVAV's security validators check for secrets hygiene and exfiltration patterns but do not assess whether code being generated contains potentially dangerous patterns (e.g., SQL concatenation, eval usage, insecure deserialization) unless those patterns are in an explicit denylist.

---

## Section 2: Top-Tier Model Behaviors NOT Governed by OVAV

Based on Sections 1.1–1.10, the following 13 specific behaviors are exhibited by Fable 5/Opus 5 but are **not enforced or governed** by OVAV's current validator set (F0–F5, from `validators.go`).

| # | Gap | Description | Current OVAV Coverage |
|---|-----|-------------|----------------------|
| G1 | **Interleaved thinking enforcement** | No validator checks whether multi-step coding tasks actually use interleaved tool-call thinking (thinking between tool results) | None |
| G2 | **Self-verification requirement** | No validator checks whether code agents ran tests, type checks, or linters before declaring completion | None |
| G3 | **Effort-level calibration** | No governance of `effort` parameter setting per task class; complex tasks could run at `low` effort | None (model_budget_policy.yaml exists but is not enforced per-task) |
| G4 | **Refusal handling** | No mechanism to handle `stop_reason: "refusal"` — no fallback routing, no retry, no user notification | None |
| G5 | **Context preservation discipline** | No validator checks whether thinking blocks are preserved correctly across tool-use turns | None |
| G6 | **Type/schema rigor** | No validator checks whether generated code has proper type annotations, schema definitions, or validation | None |
| G7 | **Long-context coherence** | No validator checks cross-referencing consistency in long-context tasks (1M token window) | None |
| G8 | **Architectural decision logging** | No requirement to document architectural decisions (ADR-style) in code artifacts | None |
| G9 | **Multi-file coherence** | No validator checks whether changes across multiple files are architecturally consistent | None (only individual file safety) |
| G10 | **Cost-efficiency awareness** | No governance of token usage relative to task complexity; no task budgets enforcement | Partial (model_budget_policy.yaml) |
| G11 | **Error recovery quality** | No validator checks whether agents recovered gracefully from errors vs. giving up | None |
| G12 | **Secure coding patterns** | Validators check denylist patterns but do not assess whether code contains insecure patterns (SQL injection risk, eval usage, insecure crypto) | Partial (secrets hygiene only) |
| G13 | **Multi-turn planning coherence** | No validator checks whether the agent's plan across turns remains coherent and does not contradict prior decisions | None |

---

## Section 3: OVAV Governance Mechanisms to Absorb These Behaviors

This section proposes concrete mechanisms for each gap. Mechanisms are designed to be **model-agnostic** — they govern behavior regardless of which AI model (Fable 5, Opus 5, DeepSeek-v4-pro, Qwen-3.7-max, etc.) is acting as the "cuerpo."

---

### G1: Interleaved Thinking Enforcement

**Proposed Validator: `interleaved_thinking_guard`**

**Trigger:** Any tool-use task with more than 2 sequential tool calls.

**Mechanism:**
```
BEFORE: tool_result_returned
CHECK:
  - Is there a thinking block from the assistant in the current turn?
  - Does the next assistant message contain new thinking (not just tool_use or text)?
IF missing:
  - Emit warning: "Thinking block not preserved between tool calls"
  - If >5 tool calls without intervening thinking: BLOCK with "Interleaved thinking required for multi-step tasks"
```

**Why this absorbs Fable 5/Opus 5 intelligence:** These models use interleaved thinking automatically. A weaker model that skips thinking between tool calls is exhibiting behavior that Fable 5/Opus 5 would not. By enforcing interleaved thinking, OVAV forces all models to behave like top-tier models on this dimension.

**Governance file:** `.ovav/validators/interleaved_thinking_guard.go`

---

### G2: Self-Verification Requirement

**Proposed Validator: `self_verification_gate`**

**Trigger:** Any `edit` or `write` operation on a code file that is not a fixture or test.

**Mechanism:**
```
BEFORE: edit/write committed
CHECK:
  - Was a verification step run after the last code change?
  - Acceptable verification: `go test`, `ruff check`, `mypy`, `gofmt -d`, `cargo check`, etc.
  - Verification must run AFTER the last edit, not before
IF no verification:
  - Emit warning requiring verification step
  - BLOCK commit until verification evidence is presented
```

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5 "automatically self-corrects through verification loops." This validator forces all models to exhibit the same self-verification behavior.

**Governance file:** `.ovav/validators/self_verification_gate.go`

---

### G3: Effort-Level Calibration

**Proposed Enhancement to `model_budget_policy.yaml`**

**Mechanism:**
```yaml
effort_governance:
  task_effort_requirements:
    architecture_design: "xhigh"        #必须有xhigh
    heavy_implementation: "xhigh"
    code_review: "high"
    security_audit: "high"
    testing_qa: "high"
    small_patches: "medium"
    summarization: "low"
    deep_exploration: "high"
    quick_diagnosis: "low"
  
  enforcement:
    check_effort_on_invoke: true
    downgrade_block_threshold: 2  # 2 consecutive low-effort on xhigh task = block
```

**Why this absorbs Fable 5/Opus 5 intelligence:** The effort parameter is how Fable 5/Opus 5 allocate reasoning resources. By encoding effort requirements per task class, OVAV ensures all models spend appropriate tokens — not just Fable 5/Opus 5 which do this adaptively.

---

### G4: Refusal Handling

**Proposed Validator: `refusal_handler`**

**Trigger:** Any response with `stop_reason: "refusal"`.

**Mechanism:**
```
ON refusal detected:
  1. Log refusal category (from stop_details.category)
  2. Route to fallback model (one tier lower in model_body_ladder)
  3. If fallback also refuses: escalate to human notification
  4. Record refusal event in session ledger
  5. DO NOT surface refusal to user without sanitization
```

**Governance file:** `.ovav/validators/refusal_handler.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5's refusal behavior is a feature, not a bug. OVAV must handle it gracefully. This mechanism turns a failure mode into a governed fallback path.

---

### G5: Context Preservation Discipline

**Proposed Validator: `thinking_block_preservation`**

**Trigger:** Every tool-use turn.

**Mechanism:**
```
BEFORE: assistant message with tool_use confirmed
CHECK:
  - Are all prior thinking blocks from this assistant turn present and unmodified?
  - If thinking block was modified or dropped: 400 error from API
  - OVAV validator should detect this before API call
  - Track thinking block count per conversation turn
  - Alert if thinking blocks are not growing across multi-turn tasks
```

**Governance file:** `.ovav/validators/thinking_block_preservation.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5 and Opus 5 preserve all thinking blocks by default. Weaker models may inadvertently drop thinking context. This validator ensures the context management discipline that top-tier models exhibit naturally.

---

### G6: Type/Schema rigor

**Proposed Validator: `type_schema_governor`**

**Trigger:** Any new Go/Python/TypeScript file, or any file exceeding 200 lines.

**Mechanism:**
```
BEFORE: first commit of any code file
CHECK:
  - Go: must have `//go:generate` or explicit type annotations on all exported APIs
  - Python: must have type hints on all function signatures (enforced via `mypy --strict`)
  - TypeScript: must have explicit interface or type for all exported functions
  - Schema files (JSON/YAML): must have JSON Schema validation section
  - If types are absent: warning + require explicit annotation or BYPASS waiver from lead
```

**Governance file:** `.ovav/validators/type_schema_governor.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5/Opus 5 produce code with strong type adherence (evidenced by top SciCode and Terminal-Bench scores). This validator extends type rigor to all models.

---

### G7: Long-Context Coherence

**Proposed Validator: `long_context_coherence`**

**Trigger:** Any task with context window utilization > 30% of available tokens.

**Mechanism:**
```
ON context_usage > 30%:
  CHECK:
    - Are variable names consistent with earlier definitions in the context window?
    - Are there conflicting assignments to the same variable in different parts of the context?
    - Does the most recent output reference artifacts that are still in context?
  WARN on:
    - Variable shadowing without explicit re-declaration
    - References to "the X we defined earlier" where X is no longer in context
    - Function calls to functions not present in current context
```

**Governance file:** `.ovav/validators/long_context_coherence.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Opus 5 scores highest on AA-LCR (long context reasoning). This validator enforces the same coherence discipline across all models.

---

### G8: Architectural Decision Logging

**Proposed Governance Rule: `adr_enforcement`**

**Mechanism:**
```yaml
adr_enforcement:
  trigger: Any change to 3+ files simultaneously that changes dependencies, interfaces, or data flow
  
  requirement:
    - File: `docs/adr/NNN-name-of-decision.md`
    - Required sections: Context, Decision, Consequences, Alternatives Considered
    - Must be committed before or alongside the code change
    - Must reference the git commit hash of the decision
    
  validator: `architecture_decision_logger`
  BLOCK if: 3+ files changed without corresponding ADR
```

**Governance file:** `.ovav/validators/adr_enforcement.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Top-tier models are documented to approach architecture decisions systematically. ADR enforcement ensures this systematic approach is applied regardless of which model is used.

---

### G9: Multi-File Coherence

**Proposed Validator: `multi_file_coherence`**

**Trigger:** Any commit touching 3+ files.

**Mechanism:**
```
BEFORE: git commit (3+ files)
CHECK:
  - Import/require statements across changed files: are they consistent?
    (no orphaned imports, no missing imports, no circular dependencies introduced)
  - Interface/implementation pairs: if interface changes, does implementation change?
  - Type consistency: if type T is used in file A and file B, are the definitions identical?
  - API surface: if a function signature changes, are all call sites updated?
  - Test coherence: if implementation changes, do tests still cover the same paths?
  
  TOOL: run `go vet ./...` or equivalent language linter across changed files together
  WARN on: cross-file type mismatches, orphaned functions, inconsistent error handling
```

**Governance file:** `.ovav/validators/multi_file_coherence.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Opus 5 is specifically noted for "end-to-end software tasks" and "coordinating parallel subagents" — implying cross-file coherence. This validator enforces that capability.

---

### G10: Cost-Efficiency Awareness

**Proposed Enhancement: `token_budget_enforcer`**

**Mechanism:**
```yaml
token_budget_enforcer:
  per_task_budgets:
    small_patch: 2048        # max output tokens
    medium_task: 16384
    large_task: 65536
    architecture_design: 128000  # xhigh effort can use millions; cap here
    deep_research: 256000
    
  track:
    - input_tokens_per_task
    - output_tokens_per_task
    - thinking_tokens_per_task (if visible)
    - cache_hit_rate
    
  alert_if:
    - output_tokens > 2x task budget without justification
    - cache_hit_rate < 30% (indicates poor prompt caching)
    
  BLOCK if:
    - cumulative session cost exceeds session budget AND no waiver
```

**Governance file:** `.ovav/validators/token_budget_enforcer.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5's effort parameter is a cost-efficiency lever. OVAV encoding task budgets ensures all models are cost-conscious, not just those with explicit effort controls.

---

### G11: Error Recovery Quality

**Proposed Validator: `error_recovery_audit`**

**Trigger:** Any task that produces a runtime error, test failure, or build failure in its output.

**Mechanism:**
```
ON build/test failure detected:
  CHECK:
    - Did the agent attempt to fix the error?
    - How many iterations of fix-attempt cycle occurred?
    - Did the agent explain what went wrong (root cause analysis)?
    - Did the agent verify the fix before reporting success?
    
  SCORE:
    - 0: No fix attempted
    - 1: Fix attempted but not verified
    - 2: Fix attempted and verified with test/run
    - 3: Fix + root cause explanation + verification
    
  BLOCK if: score = 0 on a production code change
  WARN if: score = 1 on any code change
```

**Governance file:** `.ovav/validators/error_recovery_audit.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5 "automatically self-corrects through verification loops." This validator measures whether that self-correction actually occurred.

---

### G12: Secure Coding Patterns

**Proposed Enhancement: `secure_coding_guardian`**

**Trigger:** Any code file change.

**Mechanism:**
```yaml
secure_coding_patterns:
  BLOCK (hard):
    - `eval(` (any language)
    - `exec(` (Python)
    - `os.system(` without input sanitization
    - Hardcoded credentials: detect via secrets_hygiene (already exists)
    - SQL concatenation: `f"SELECT * FROM {var}"` without parameterized query
    - Insecure random: `random.random()` for security-sensitive operations
    - YAML load without SafeLoader
    - `json.load()` from untrusted source without validation
    
  WARN (soft):
    - Usage of `//go:inline` without justification
    - Reflection usage in Go
    - `any` type in TypeScript without explicit cast
    - `// nolint:` comments without explanation
    
  require_documentation:
    - Any cryptography usage must have a comment citing the algorithm rationale
    - Any network call must have error handling documented
```

**Governance file:** `.ovav/validators/secure_coding_guardian.go` (enhance existing security hardening validator)

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5's safety classifiers are trained to detect potentially dangerous code patterns. This validator encodes that awareness at the OVAV level, making it model-agnostic.

---

### G13: Multi-Turn Planning Coherence

**Proposed Validator: `planning_coherence_guard`**

**Trigger:** Any session with 5+ turns on the same task.

**Mechanism:**
```
ON turn 5+ of same task:
  CHECK:
    - Has the agent stated an overall plan?
    - Does the current action align with the stated plan?
    - Has the plan changed? If so, was the change explained?
    
  BLOCK if:
    - Agent takes an action that contradicts a prior stated constraint
    - Agent introduces a new architectural pattern that contradicts earlier discussion
    
  REQUIRE:
    - At turn 3: agent must articulate "my approach is: [summary]"
    - At turn 7+: any deviation from prior approach requires explicit "revisiting plan because: [reason]"
```

**Governance file:** `.ovav/validators/planning_coherence_guard.go`

**Why this absorbs Fable 5/Opus 5 intelligence:** Fable 5/Opus 5 approach complex tasks with "long-horizon" planning. This validator enforces that planning discipline across all models, preventing drift and contradiction.

---

## Section 4: "Cuerpo y Cerebro" Architecture

### The Metaphor

```
┌─────────────────────────────────────────────────────────────────────┐
│                         OVAV CEREBRO                                │
│  ───────────────────────────────────────────────────────────────── │
│  Validators (F0-F5), Gates, Policies, Registry, Session Guard,       │
│  Chronos, Output Guard, Compliance, ADR enforcement, Budget,        │
│  Effort calibration, Refusal handling, Thinking block tracking      │
│                                                                      │
│  "The brain does not write code. It governs HOW code gets written." │
│  "The brain does not execute. It decides WHAT can execute."        │
│  "The brain does not model. It decides WHICH model runs."          │
└─────────────────────────────────────────────────────────────────────┘
                              │ sends models task bundles
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        AI MODEL (CUERPO)                             │
│  ───────────────────────────────────────────────────────────────── │
│  Fable 5 / Opus 5 / DeepSeek-v4-pro / Qwen-3.7-max / MiniMax-M3   │
│                                                                      │
│  "The body generates code. It executes tool calls. It thinks."     │
│  "The body is replaceable — any capable model can serve."           │
│  "The body is interchangeable. The brain is permanent."            │
└─────────────────────────────────────────────────────────────────────┘
                              │ produces code, artifacts
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       VERIFICATION LAYER                             │
│  ───────────────────────────────────────────────────────────────── │
│  Self-verification gate, test runners, linters, type checkers       │
│  These are the SENSORY FEEDBACK the brain uses to validate output   │
└─────────────────────────────────────────────────────────────────────┘
```

### How OVAV Absorbs Top-Tier Model Intelligence

**Source:** OVAV architecture (validators.go, session_guard_policy.yaml, model_body_ladder.yaml, capabilities.yaml); Anthropic Fable 5/Opus 5 documentation

The key insight is that Fable 5/Opus 5 intelligence comes from **architectural behaviors**, not proprietary magic:

| Top-Tier Behavior | How OVAV Absorbs It |
|-------------------|---------------------|
| Interleaved thinking | Enforce with `interleaved_thinking_guard` — forces all models to think between tool calls |
| Self-verification loops | Enforce with `self_verification_gate` — requires tests/linters to run before commit |
| Effort calibration | Encode in `model_budget_policy.yaml` — map task types to effort levels |
| Refusal handling | Route with `refusal_handler` — fallback routing + logging on refusal |
| Context preservation | Monitor with `thinking_block_preservation` — ensures thinking continuity |
| Type/schema rigor | Validate with `type_schema_governor` — forces typed code from all models |
| Long-context coherence | Check with `long_context_coherence` — cross-references within context window |
| Architectural decisions | Require with `adr_enforcement` — forces documentation of decisions |
| Multi-file coherence | Validate with `multi_file_coherence` — cross-file consistency checks |
| Token budgets | Enforce with `token_budget_enforcer` — per-task token caps |
| Error recovery | Audit with `error_recovery_audit` — measures fix quality |
| Secure coding | Guard with `secure_coding_guardian` — blocks dangerous patterns |
| Planning coherence | Guard with `planning_coherence_guard` — prevents plan drift |

### The Model Body Ladder Integration

OVAV's `model_body_ladder.yaml` already routes models by task type. The intelligence-absorption mechanisms above complement routing by **ensuring that any model, regardless of tier, is forced to exhibit top-tier behavioral patterns**.

The 14 OpenCode Go models (from `model_body_ladder.yaml`) are:
- **Lead tier:** DeepSeek-v4-pro, Qwen-3.7-max, MiniMax-M3, GLM-5.1
- **Strong tier:** Qwen-3.7-plus, GLM-5, MiniMax-M2.7, Kimi-K2.6
- **Deep tier:** Mimo-v2.5-pro, Kimi-K2.5, MiniMax-M2.5
- **Fast tier:** DeepSeek-v4-flash, Qwen-3.6-plus, Mimo-v2.5

None of these are Fable 5 or Opus 5 (Anthropic models are explicitly excluded per `no_anthropic: true` in `model_body_ladder.yaml`). Yet with the governance mechanisms proposed in Section 3, OVAV can make any model in the ladder **behave** with the structural discipline of Fable 5/Opus 5 on the dimensions that matter for code quality and safety.

### Closing the Loop: The Benchmark Evidence

The OVAV benchmark already demonstrates that governance works: **ASR 0% with OVAV vs 45-70% RAW** on dangerous task refusal. This proves that OVAV can govern outcomes that raw models cannot self-govern.

The mechanisms in Section 3 extend this same principle to code quality dimensions:
- **Code quality ASR** (self-verification rate) should approach 0% without governance and 100% with `self_verification_gate`
- **Type rigor ASR** should approach 0% without `type_schema_governor`
- **Multi-file coherence ASR** should approach 0% without `multi_file_coherence`

OVAV's mission is to be the **cerebro** — the cognitive governance layer — that forces bodies (models) to behave intelligently, regardless of which body is running. Fable 5 and Opus 5 provide the behavioral specification. OVAV encodes that specification as enforceable governance.

### Summary Table: Gaps vs Mechanisms

| Gap | Proposed Validator | OVAV Component |
|-----|-------------------|----------------|
| G1 Interleaved thinking | `interleaved_thinking_guard` | new validator |
| G2 Self-verification | `self_verification_gate` | new validator |
| G3 Effort calibration | `model_budget_policy.yaml` enhancement | policy + runtime check |
| G4 Refusal handling | `refusal_handler` | new validator |
| G5 Context preservation | `thinking_block_preservation` | new validator |
| G6 Type/schema rigor | `type_schema_governor` | new validator |
| G7 Long-context coherence | `long_context_coherence` | new validator |
| G8 ADR logging | `adr_enforcement` | new validator |
| G9 Multi-file coherence | `multi_file_coherence` | new validator |
| G10 Token budgets | `token_budget_enforcer` | new validator |
| G11 Error recovery | `error_recovery_audit` | new validator |
| G12 Secure coding | `secure_coding_guardian` | enhance existing |
| G13 Planning coherence | `planning_coherence_guard` | new validator |

---

## Source Citations

1. Anthropic, "Models Overview," docs.anthropic.com/en/docs/about-claude/models — Fable 5/Opus 5 specs, pricing, context windows
2. Anthropic, "Introducing Claude Fable 5 and Claude Mythos 5," docs.anthropic.com/en/docs/about-claude/models/introducing-claude-fable-5-and-claude-mythos-5 — capabilities, adaptive thinking, refusals, safety classifiers
3. Anthropic, "Thinking," docs.anthropic.com/en/docs/build-with-claude/thinking — interleaved thinking, thinking block preservation, keep-all regime, signature encryption
4. Anthropic, "Effort," docs.anthropic.com/en/docs/build-with-claude/effort — effort levels, xhigh/max guidance, token control
5. Anthropic, "Redeploying Fable 5," anthropic.com/news/redeploying-fable-5 — safety classifiers, safety margin, jailbreak severity framework, self-correction, model comparison (Haiku/Sonnet/Opus/GPT/Kimi)
6. Artificial Analysis, "Intelligence Index v4.1," artificialanalysis.ai/models — Opus 5 score 61, Fable 5 score 60, benchmark descriptions (Terminal-Bench v2.1, SciCode, GPQA Diamond, AA-LCR, AA-Briefcase)
7. OpenRouter, "Claude Opus 5," openrouter.ai/models/anthropic/claude-opus-5 — end-to-end software tasks, parallel subagents
8. OpenRouter, "Claude Fable 5," openrouter.ai/models/anthropic/claude-fable-5 — autonomous knowledge work, verification loops, long-horizon tasks
9. OVAV `validators.go` — existing validator registry (F0-F5, 50+ validators)
10. OVAV `session_guard_policy.yaml` — injection patterns, governance files, response protocols
11. OVAV `model_body_ladder.yaml` — 14 OpenCode Go models, routing by task type, no_anthropic: true
12. OVAV `capabilities.yaml` (platform_engineering) — default tools, denied_without_grant, context classes
13. OVAV `quality_standards.yaml` — integration factor, phase approval criteria

---

*Document generated by OVAV Research Intelligence (Eidren's team)*
*Classification: Internal — Governance Intelligence*
*Next action: Present to Thavren for Platform Engineering absorption into validator roadmap*
