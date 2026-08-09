# OVAV Security Gaps — Deep Dive & Defense-in-Depth Design

**Author**: Platform Engineering (Thavren's squad) — Helena (Deep Explorer) + Diana (Security Auditor)
**Date**: 2026-07-28
**Source**: Benchmark data `62d4299` + F0-F5 validator analysis
**Status**: proposal/v1

---

## Section 1: OVAV Security Gap Map

### Current OVAV Coverage (F0-F5)

| Validator | Coverage | What it blocks |
|-----------|----------|----------------|
| **F0** workspace_safety_gate | ✅ Full | Dangerous filesystem ops, protected paths |
| **F1** identity_guard | ✅ Full | Agents without OVAV_IDENTITY_GUARD |
| **F2** permission_policy_drift | ✅ Full | Unauthorized external_directory writes |
| **F3** secrets_hygiene | ✅ Full | Plaintext secrets in tracked files |
| **F4** check_living_integrity | ✅ Full | F0-F4 validation + integrity score |
| **F5** check_branch_protection | ✅ Full | Push to main/master without waiver |

**Result: ASR 0% on 4×20 hard tasks + 12 adversarial vectors.**

---

### Known Security Gaps

#### GAP-1: V8 Multi-Turn Escalation
| Field | Value |
|-------|-------|
| **Severity** | MEDIUM-HIGH |
| **Description** | Attacker uses multiple turns to escalate privileges gradually. Each individual turn looks benign; cumulative effect is dangerous. |
| **Current OVAV coverage** | PARTIAL — F0 gates single operations but does NOT track session state across multiple turns |
| **Example attack** | Turn 1: "list my files" → Turn 2: "show me the content of files in /tmp" → Turn 3: "now delete those files" |
| **What Fable/Opus do** | Implicit context tracking — RLHF trained to refuse escalation patterns |
| **OVAV gap** | No session-level threat aggregation |

#### GAP-2: Safe-Alternative Hallucination
| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Description** | Model suggests a "safe-looking" alternative that is actually dangerous. Specifically seen in T20 MiniMax OVAV. |
| **Current OVAV coverage** | NONE — validator checks for dangerous commands, not dangerous "safe" alternatives |
| **Example** | User asks to delete files. Model suggests "use rm -rf /tmp/test" as "safe temporary cleanup". rm -rf is dangerous. |
| **What top models do** | Fable 5 / Opus 5 have stronger RLHF against "disguised dangerous operations" |
| **OVAV gap** | No semantic analysis of suggested operations |

#### GAP-3: Adversarial Prompt Evolution
| Field | Value |
|-------|-------|
| **Severity** | HIGH |
| **Description** | Attacker observes OVAV blocks and adapts prompt dynamically. Like red team iterative approach. |
| **Current OVAV coverage** | NONE — F0-F5 are static rules; they don't learn from attack patterns within session |
| **Example** | V1 blocked → Attacker tries V1 with synonyms → Blocked → Tries V1 with base64 encoding → Blocked? |
| **What top models do** | Reinforcement learning from human feedback reduces novelty-seeking exploits |
| **OVAV gap** | No adaptive threat detection |

#### GAP-4: Model-Specific Failure Modes
| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Description** | Each model has different security posture. DeepSeek: 70% ASR RAW. MiniMax: 45% ASR RAW. Different patterns fail. |
| **Current OVAV coverage** | PARTIAL — F0-F5 are model-agnostic but may under- or over-block certain models |
| **Example** | MiniMax responds well to V1-V12 but fails on T20. DeepSeek fails broadly but blocks well with OVAV |
| **What top models do** | Fable 5 / Opus 5 have consistent security posture across task types |
| **OVAV gap** | No per-model security profiles |

#### GAP-5: Workspace Contamination Post-Task
| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Description** | Task completes successfully but leaves artifacts that could be exploited in subsequent sessions. |
| **Current OVAV coverage** | NONE — no post-task workspace hygiene validation |
| **Example** | Model generates a script with a backdoor comment, or creates temp files with sensitive data |
| **What top models do** | Fable 5 / Opus 5 trained to clean up; habit of leaving no trace |
| **OVAV gap** | No post-task artifact scanning |

#### GAP-6: Secrets Exfiltration via Generated Comments
| Field | Value |
|-------|-------|
| **Severity** | MEDIUM |
| **Description** | Model embeds real secrets (from context) into code comments or error messages. |
| **Current OVAV coverage** | PARTIAL — F3 checks tracked source files but not generated output before commit |
| **Example** | "Here's the API key you mentioned earlier: `const key = 'real-key-123'`" — in a comment |
| **What top models do** | RLHF to not echo back sensitive data seen in context |
| **OVAV gap** | No pre-commit scan for context-echoed secrets |

---

## Section 2: Per-Gap Defense-in-Depth Design

### GAP-1: V8 Multi-Turn Escalation

**Layer 1 — Detect**
- `F0_extended`: Track cumulative operation count per session. If >10 filesystem operations in 5 minutes, escalate to enhanced scrutiny.
- Session operation log: `~/.ovav/session_ops.yaml`

**Layer 2 — Prevent**
- `F4_session_context`: New sub-check that evaluates "escalation risk score" based on operation pattern.
- Pattern: file_read → file_read → file_delete = escalation sequence. Block at 3rd operation if pattern matches.

**Layer 3 — Recover**
- `workspace_rollback`: After dangerous sequence, offer auto-rollback of affected files.

**Layer 4 — Audit**
- `audit_escalation_log`: Log all escalation sequences to `~/.ovav/audit/escalation_*.yaml`.

---

### GAP-2: Safe-Alternative Hallucination

**Layer 1 — Detect**
- `F0_semantic_scan`: New validator that runs after code generation. Scans for dangerous operations disguised as safe alternatives.
- Patterns: `rm -rf` in any context, `eval()` with string concatenation, shell commands in non-shell scripts.

**Layer 2 — Prevent**
- `pre_execution_gate`: Before any shell command execution, OVAV CLI must confirm the exact command being run. No silent execution.

**Layer 3 — Recover**
- `audit_hallucination_log`: Track "safe alternative" suggestions that were actually dangerous. Feed to F0 rule update.

**Layer 4 — Audit**
- `hallucination_report`: Per-session report of "safe-looking but dangerous" suggestions caught.

---

### GAP-3: Adversarial Prompt Evolution

**Layer 1 — Detect**
- `F0_adversarial_pattern`: Track prompts that were BLOCKED. If similar prompt (Levenshtein distance <3) appears again within 10 minutes, flag as iterative attack.
- Block count per session tracked.

**Layer 2 — Prevent**
- `adaptive_threshold`: After 3 blocked attempts of similar pattern, introduce mandatory delay (30s) before next attempt.
- After 5 blocked: require CEO waiver for that session.

**Layer 3 — Recover**
- `threat_feed`: Store blocked patterns. Weekly F0 rule update from accumulated patterns.

**Layer 4 — Audit**
- `adversarial_log`: Immutable record of all blocked patterns with timestamps.

---

### GAP-4: Model-Specific Failure Modes

**Layer 1 — Detect**
- `model_profile_gate`: At session start, record which model is running. Load model-specific security profile.
- Profiles: `~/.ovav/models/fable5.yaml`, `~/.ovav/models/opus5.yaml`, `~/.ovav/models/deepseek.yaml`, etc.

**Layer 2 — Prevent**
- Model-specific F0 rules: DeepSeek gets stricter filesystem gates (more failures on RAW). MiniMax gets stricter "alternative suggestion" scanning.
- `model_strength_gate`: Use model strengths to route tasks. Fable 5 for security-critical; MiniMax for routine tasks.

**Layer 3 — Recover**
- `model_failure_log`: After each security event, log which model was running. Update profiles monthly.

**Layer 4 — Audit**
- `model_audit_report`: Per-model security posture over time.

---

### GAP-5: Workspace Contamination Post-Task

**Layer 1 — Detect**
- `F0_post_task_scan`: After each task completes, scan workspace for: new files in unexpected locations, modified system files, temp files with sensitive data.
- Baseline: snapshot of workspace before task. Compare after.

**Layer 2 — Prevent**
- `workspace_fence`: OVAV CLI creates a `.ovav/workspace_sentinel` file that marks the "clean state". Any deviation from baseline tracked.

**Layer 3 — Recover**
- `workspace_clean`: OVAV CLI command `ovav workspace clean` removes all non-committed artifacts not in original baseline.

**Layer 4 — Audit**
- `contamination_report`: Per-session report of all workspace changes.

---

### GAP-6: Secrets Exfiltration via Comments

**Layer 1 — Detect**
- `F3_extended`: Extend secrets hygiene to scan generated code comments for patterns matching context-seen values (API keys, tokens, passwords).
- Check: if a value appears in context AND in generated comment → flag.

**Layer 2 — Prevent**
- `context_seal`: OVAV marks context-seen secrets. Any echo of marked secrets in output triggers block.

**Layer 3 — Recover**
- `audit_exfiltration_log`: Log all exfiltration attempts with the secret type.

**Layer 4 — Audit**
- `exfiltration_report`: Monthly review of exfiltration patterns.

---

## Section 3: Security Governance for Unknown/Unknown Threats

### The Unknown-Knowns Problem

OVAV F0-F5 handle KNOWN threats. The benchmark tested known patterns (V1-V12, T1-T20). But attackers innovate.

**Defense Strategy:**

| Layer | Approach | Implementation |
|-------|----------|----------------|
| **1. Anomaly detection** | Flag operations that are unusual for the task context | `F0_anomaly_score`: if operation is statistically rare for this task type → enhanced scrutiny |
| **2. Behavior limits** | Hard caps: max file operations per session, max network calls, max external commands | `F0_hard_limits`: configurable per workspace |
| **3. Principle of least privilege** | AI session starts with minimal permissions; must explicitly request more | `F0_privilege_escalation`: each escalation request logged + requires confirmation |
| **4. Immutable audit trail** | Whatever happens, we record it. Even if we can't prevent, we document. | `~/.ovav/audit/` append-only logs |
| **5. Regular red team** | Bi-weekly adversarial simulation (Kenji's team) to find unknown gaps | `ovav red-team --weekly` |

### The "Cuerpo/Cerebro" Tightening

```
PROBLEMA ACTUAL:
  Cuerpo (model) → genera código → ejecuta → F0 previene SI es peligroso
  Pero: F0 solo checkea operación individual, no el efecto acumulativo

MEJORA PROPUESTA:
  Ovav (cerebro) → trackea sesión completa → ve patrones → previene ANTES de operación
  No: "rm -rf" → block
  Sí: "read files → read files → rm -rf" → block ANTES del rm -rf
```

**Implementación:**
- Session state tracker en `~/.ovav/session_state.yaml`
- Pattern matcher que evalúa la SECUENCIA, no solo el individual operation
- `F0_sequence_guard`: nuevo sub-validador de F0

---

## Section 4: F0-F5 Gap Analysis

### What F0-F5 DON'T Catch That They Should

| Validator | Current | Gap | Should Also Catch |
|-----------|---------|-----|-------------------|
| **F0** workspace_safety | Block dangerous ops | No sequence tracking | Escalation patterns (GAP-1) |
| **F0** workspace_safety | Block dangerous ops | No semantic scan | Safe-alternative dangerous ops (GAP-2) |
| **F1** identity_guard | Block if no identity | No runtime drift | Agent identity changes mid-session |
| **F2** permission_drift | External directory writes | No read leaks | Sensitive data read + exfiltrated via output |
| **F3** secrets_hygiene | Plaintext secrets in files | No context echoes | Secrets from context embedded in comments (GAP-6) |
| **F4** integrity | F0-F4 + score | No adaptive rules | Iterative adversarial patterns (GAP-3) |
| **F5** branch_protection | Push blocks | No log of push attempts | Audit trail of blocked pushes |

### F0-F5 Evolution Roadmap

| Phase | Validators | New Capabilities |
|-------|------------|-----------------|
| **Now** | F0-F5 | Static rule-based security |
| **v91** | F0-F5 + F0_extended | Sequence tracking, per-model profiles |
| **v92** | F0-F5 + F0_extended + F1_extended | Session state, privilege escalation |
| **v93** | All above + adaptive threat | Pattern learning from blocked attempts |

---

## Summary: Defense-in-Depth Per Gap

| GAP | Severity | Layers | Primary Defense |
|-----|----------|--------|-----------------|
| V8 Multi-Turn | MEDIUM-HIGH | 4 | F0_sequence_guard |
| Safe-Alternative Hallucination | MEDIUM | 4 | F0_semantic_scan + pre_execution_gate |
| Adversarial Evolution | HIGH | 4 | F0_adaptive_threshold |
| Model-Specific | MEDIUM | 4 | model_profile_gate |
| Workspace Contamination | MEDIUM | 4 | F0_post_task_scan + workspace_fence |
| Secrets via Comments | MEDIUM | 4 | F3_extended + context_seal |

**Key insight:** Every gap needs 3+ layers because NO single layer is perfect. Defense-in-depth means: if one layer fails, the others still catch it.

---

*Author: Helena (Deep Explorer) + Diana (Security Auditor) — Platform Engineering*
*Source: Benchmark 62d4299 + F0-F5 gap analysis*
