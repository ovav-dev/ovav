# TASK9 — Delegation: First Adversarial Audit (Red Team)

| Field | Value |
|---|---|
| Handoff ID | TASK9-RED-TEAM-AUDIT |
| Delegate To | **Kenji Tanaka** (Lead, Adversarial Intelligence) + Squad (Akiko, Ryu, Mei, Kaori, Hiroshi) |
| Delegated By | Thavren / Platform Engineering |
| Date | 2026-06-17 |
| Priority | P0 — Security Foundation |
| Branch | task/tasknext-ceo-task9 |
| Source Caps | caps.yaml L315-343 (T9 cap), L271-310 (v2.0.0 summary), L1128 (new_area: Kenji Tanaka) |

---

## 1. CONTEXT — Why This Audit Now

OVAV v2.0.0 is LIVE in production with 4 public surfaces:

| Surface | URL | Stack | Status |
|---|---|---|---|
| Landing | `ovav.dev` | Next.js 14 + Tailwind, CF Pages | HTTP 200 |
| Dashboard | `cpanel.ovav.dev` | React 18 + Vite + Go backend, Fly.io DFW | LIVE v2.0.0 |
| Documentation | `docs.ovav.dev` | Starlight (Astro), CF Pages | HTTP 200 |
| Status | `status.ovav.dev` | Better Uptime managed | HTTP 200 |

The Go runtime is production-ready:

| Metric | Value |
|---|---|
| Go validators | 61 (77.2% migration from Python) |
| Go packages | 16 |
| Go tests | 14/14 packages PASS (-race, 0 data races) |
| cPanel coverage | 67.4% (target: ≥80%) |
| cPanel API endpoints | 17 route handlers (auth, OAuth, profiles, validators, git, memory, agents, security, system, events, SSE, status) |
| CLI coverage | 34.5% (target: ≥60%) |
| Binary targets | 5/5 cross-compile (linux/darwin/windows) |
| Build | gofmt clean, go vet clean |

**The system has NEVER had a formal adversarial audit.** Every security validator is defensive — they check configuration, syntax, and policy compliance. No one has attacked OVAV with the mindset of a 2026 model-level adversary. This is that audit.

---

## 2. SCOPE — Adversarial Audit

Kenji Tanaka and squad execute a full-spectrum Red Team audit against OVAV v2.0.0. All testing is simulated, contained, and read-only by default. The scope is defined by Kenji's area contract (`.ovav/service_areas/areas/adversarial_intelligence.yaml`) and lead definition (`clients/opencode/agents/lead-kenji.md`).

### 2.1 Semantic Analysis (Akiko — Senior Red Team Operator)

| Focus | Details |
|---|---|
| **Prompt injection** | Can adversarial prompts injected via handoffs, API inputs, or agent messages bypass system instructions? |
| **Role confusion** | Can agents be tricked into acting outside their authorized functions? Test all 9 leads + 9 areas. |
| **Boundary bypass** | Can semantic ambiguity in handoff language be exploited to cross area boundaries? |
| **System prompt extraction** | Can any agent's system prompt be extracted through crafted conversation? |
| **Jailbreaking** | Can model-level jailbreak techniques bypass OVAV's hard stops? |
| **Hard stop evasion** | Test all hard stop responses across all leads — can they be semantically sidestepped? |

### 2.2 Boundary Testing (Kaori — Boundary Tester)

| Focus | Details |
|---|---|
| **Agent-to-agent isolation** | Verify no agent can invoke or influence another agent outside delegation rules. Cross-test all 68 agents. |
| **Permission escalation** | Attempt to escalate from team member → lead, lead → another area, area → CEO functions. |
| **Cross-area leakage** | Can information from one service area leak into another through handoffs, shared context, or memory surfaces? |
| **LAW-001 enforcement** | Verify Non-Invasion Area Boundary Law holds under adversarial pressure. |
| **Handoff protocol integrity** | Can handoffs be forged, intercepted, or modified to cross authority boundaries? |
| **Context economy bypass** | Can agents load more context than their tier permits? |

### 2.3 Race Condition Hunting (Mei — Race Condition Hunter)

| Focus | Details |
|---|---|
| **Validator concurrency** | Concurrent execution of validators — are there race windows where a validator passes but state changes before enforcement? |
| **Install pipeline** | Parallel install operations, backup/rollback race windows, manifest atomicity violations. |
| **SSE connections** | Concurrent SSE event streams — do multiple connections expose inconsistent state? |
| **cPanel API** | Concurrent auth sessions, token refresh races, OAuth state parameter races. |
| **TOCTOU** | Time-of-check to time-of-use across all file operations, config reads, and integrity checks. |
| **Goroutine leaks** | Any long-running goroutines without proper cancellation or cleanup? |

### 2.4 Architectural Audit (Ryu — Architectural Auditor)

| Focus | Details |
|---|---|
| **Single points of failure** | Identify components whose failure cascades to system-wide outage. |
| **Trust chain weaknesses** | Map the trust chain from CLI → validators → runtime → deploy → surfaces. Where are implicit trust assumptions? |
| **Circular dependencies** | Detect circular references in contracts, validators, or authority definitions. |
| **Contradictory rules** | Find rules that nullify each other (e.g., "X must always happen" + "Y blocks X"). |
| **Undocumented surfaces** | Are there API endpoints, file paths, or access patterns not declared in any contract? |
| **Authority conflicts** | Overlapping authority claims between areas or leads that could deadlock decisions. |

### 2.5 Drift Detection (Hiroshi — Drift Detector)

| Focus | Details |
|---|---|
| **Runtime vs. contracts** | Does actual runtime behavior match declared contracts? Test every contract clause. |
| **Personality drift** | Monitor agent output consistency over multiple invocations — tone, judgment criteria, response patterns. |
| **Version drift** | Are all deployed versions consistent? Check VERSION file vs. cPanel /health vs. binary ldflags. |
| **Config drift** | Does deployed config match committed config? Check secrets, env vars, OAuth redirects. |
| **Validator drift** | Do validators enforce what they claim? Test each validator against crafted violating inputs. |
| **Coverage regression** | Has test coverage silently dropped in any package since last measurement? |

---

## 3. SURFACES TO AUDIT

### 3.1 cPanel API (`cpanel.ovav.dev`) — PRIMARY TARGET

| Attack Vector | What to Test |
|---|---|
| **Auth flow** | JWT RS256 token forging, expiry bypass, session hijacking, token replay |
| **OAuth (Google/GitHub)** | CSRF state parameter manipulation (one-time use, 10min TTL), redirect URI poisoning, token exchange interception |
| **Rate limiting** | 5 attempts/min/IP bypass (X-Forwarded-For spoofing), distributed brute force |
| **CORS** | Cross-origin requests from unauthorized domains, preflight bypass |
| **All 17 endpoints** | Input fuzzing, SQL injection (if any DB), path traversal (URL-encoded), oversized payloads, null byte injection |
| **SSE events** | Connection flooding, event injection, long-lived connection abuse |
| **API versioning** | `/api/v1/` vs. `/` path confusion, version downgrade attacks |
| **Error handling** | Stack trace leakage, internal path disclosure in error messages |

### 3.2 CLI Validators — BYPASS ATTEMPTS

| Attack Vector | What to Test |
|---|---|
| **Validator bypass** | Can any of the 61 Go validators be bypassed with crafted input? |
| **Path resolution** | Symlink attacks, `..` traversal, root detection spoofing |
| **Integrity check evasion** | Modify files while preserving hash, race baseline generation |
| **Python validator shims** | Any remaining Python validators that lack enforcement parity with Go equivalents? |
| **validate CLI** | Can the `ovav validate` command be tricked into skipping validators? |

### 3.3 Install Pipeline — INTEGRITY ATTACK

| Attack Vector | What to Test |
|---|---|
| **Backup integrity** | Can backup be corrupted mid-operation? Partial backup restoration attacks. |
| **Rollback integrity** | Can rollback leave system in inconsistent state? Rollback to poisoned backup. |
| **Manifest tampering** | Modify install manifest between plan and apply phases. |
| **Safety gate bypass** | Disable or circumvent install safety gates (`--force`, env vars, signal timing). |
| **Config injection** | Inject malicious config during deploy phase. Environment variable poisoning. |
| **Boundary violation** | Install components outside declared boundaries (blocked surfaces). |

### 3.4 Agent Boundary Enforcement — CROSS-AGENT ATTACKS

| Attack Vector | What to Test |
|---|---|
| **Agent invocation** | Attempt to invoke non-authorized agents from any lead or area context. |
| **Permission matrix** | Cross-reference all 68 agents against permission_authority.json — any mismatches? |
| **Handoff forgery** | Create handoff documents that cross area boundaries without authorization. |
| **Context smuggling** | Hide instructions for other agents in shared context, comments, or artifact metadata. |
| **Subagent escalation** | Can a `mode:subagent` with `hidden:true` be surfaced to user or invoke other agents? |
| **OVAV governor bypass** | Attempt to disable or circumvent governor enforcement (output guard, BrevityRail, session gates). |

### 3.5 OAuth Flow — TOKEN & SESSION ATTACKS

| Attack Vector | What to Test |
|---|---|
| **Token spoofing** | Forge JWT RS256 tokens with modified claims (tier, role, expiry). |
| **CSRF** | Replay one-time CSRF state tokens, predict state values, cross-site request forgery. |
| **Redirect abuse** | Open redirect via OAuth callback, redirect to attacker-controlled domain. |
| **Session fixation** | Set session before authentication, hijack after login. |
| **Token leakage** | JWT in logs, error messages, client-side storage, Referer headers. |
| **OAuth provider spoof** | MITM between cPanel and Google/GitHub OAuth endpoints. |

---

## 4. RULES OF ENGAGEMENT

### 4.1 Mandatory Constraints

| Rule | Detail |
|---|---|
| **Read-only first** | NO destructive tests without explicit CEO waiver. Scan, probe, analyze — but do not modify production state. |
| **Sandbox preferred** | Prefer isolated test environments. Production testing only where sandbox cannot replicate the surface. |
| **Document everything** | Every finding must include: attack vector, reproduction steps, severity, affected component, suggested mitigation. |
| **Critical findings — immediate** | Any vulnerability rated CRITICAL or HIGH must be reported immediately via handoff to Thavren. Do not wait for final report. |
| **No self-remediation** | Kenji's area does NOT fix vulnerabilities. Findings go to Thavren (Platform Engineering) and Diana (Security Auditor). |
| **Logs immutable** | All adversarial operations logged to `.ovav/adversarial/logs/` with timestamps, operator, target, and result. |

### 4.2 Sovereign Area — NON-NEGOTIABLE

```
┌─────────────────────────────────────────────────────────────┐
│  KENJI'S AREA IS SOVEREIGN                                  │
│                                                             │
│  No other lead can override Red Team decisions about:       │
│  - What constitutes a vulnerability                         │
│  - Severity ratings                                         │
│  - Attack methodology                                       │
│  - Scope expansion within authorized surfaces               │
│  - Timing and cadence of testing                            │
│                                                             │
│  Other leads may DISPUTE findings (via formal handoff),     │
│  but they cannot BLOCK the audit.                           │
│                                                             │
│  Only the CEO can halt or constrain Red Team operations     │
│  via explicit CEO waiver.                                   │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Escalation Path

| Severity | Action | Timeline |
|---|---|---|
| **CRITICAL** — System compromise, data exposure, auth bypass | Immediate handoff to Thavren + Diana. CEO notified within 60 min. | < 1 hour |
| **HIGH** — Significant vulnerability, limited exploitability | Handoff to Thavren within the audit session. | Same session |
| **MEDIUM** — Defense-in-depth weakness, hardening opportunity | Documented in final report. | Final report |
| **LOW** — Cosmetic, informational, best-practice gap | Documented in final report with recommendation. | Final report |

### 4.4 Out of Scope

Kenji's squad does NOT:
- Test against third-party services (Stripe, Better Uptime, Cloudflare, Fly.io, GitHub)
- Execute social engineering against OVAV team members
- Perform DoS/DDoS against production surfaces
- Access secrets or credentials outside controlled audit environment
- Modify production code, config, or deployments

---

## 5. DELIVERABLES

### 5.1 Required Artifacts

| # | Deliverable | Description |
|---|---|---|
| 1 | **Red Team Audit Report** | Comprehensive report organized by surface and attack vector. Each finding includes: ID, severity (CRITICAL/HIGH/MEDIUM/LOW), attack vector, reproduction steps, affected component(s), evidence (logs/screenshots), suggested mitigation, CWE/OWASP mapping. |
| 2 | **Priority-Ranked Fix List** | All findings sorted by severity → exploitability → impact. Each entry has: finding ID, recommended fix owner (Thavren/Diana/Uriel/Dante), estimated effort (S/M/L), blocking status for launch. |
| 3 | **Penetration Test Logs** | Minimum 1 detailed pentest log per surface (5 total: cPanel API, CLI validators, install pipeline, agent boundaries, OAuth flow). Each log includes: target, methodology, tools used, attempts, successes, failures. |
| 4 | **Architectural Vulnerability Map** | Visual or structured map of trust chain weaknesses, single points of failure, and circular dependencies found. |
| 5 | **Drift Evidence Pack** | For each drift detected: contract clause vs. observed behavior, timestamp, reproduction evidence. |

### 5.2 Report Format

All findings must follow this structure:

```
FINDING-XXX: [Title]
Severity: CRITICAL | HIGH | MEDIUM | LOW
Surface: [cPanel API | CLI Validators | Install Pipeline | Agent Boundaries | OAuth Flow]
Attack Vector: [Description of how the vulnerability is exploited]
Reproduction:
  1. [Step by step]
  2. [With exact commands/inputs]
  3. [Expected vs. actual result]
Affected Components: [Files, endpoints, agents]
Evidence: [Logs, responses, screenshots]
Mitigation: [Recommended fix — for Thavren/Diana, not for Red Team]
CWE: [CWE-XXX]
OWASP: [A01-A10 if applicable]
```

---

## 6. TIMELINE

| Milestone | Target |
|---|---|
| **Start** | Immediately upon handoff receipt (2026-06-17) |
| **Phase 1 — Reconnaissance** | Surface mapping, endpoint discovery, agent inventory cross-reference, contract review. |
| **Phase 2 — Passive Testing** | Read-only probes, semantic analysis, boundary probing, config drift detection. |
| **Phase 3 — Active Testing** | Controlled exploit attempts (with CEO waiver if destructive), race condition hunting, OAuth attacks. |
| **Phase 4 — Report** | Compile findings, rank by severity, produce all 5 deliverables. |
| **Report Back** | Task10 or via formal handoff to Thavren. |

**Kenji controls the pace and order of operations.** The timeline above is guidance, not a schedule constraint. Quality of findings matters more than speed.

---

## 7. ACCEPTANCE CRITERIA

1. All 5 surfaces audited with at least 1 penetration test log each
2. Semantic analysis completed against all 9 lead system prompts and 9 area definitions
3. All 17 cPanel API endpoints probed for injection, auth bypass, and input validation weaknesses
4. All 61 Go validators tested for bypass potential (at minimum: spot-check 20% + all F0-F5 core validators)
5. Install pipeline tested for backup/rollback integrity under adversarial conditions
6. Agent boundary enforcement tested with cross-agent invocation attempts across all 9 areas
7. OAuth flow tested for CSRF, token spoofing, redirect abuse, and session fixation
8. All CRITICAL and HIGH findings reported immediately (not held for final report)
9. Priority-ranked fix list delivered with clear owner assignments
10. Architectural vulnerability map identifies all single points of failure
11. All findings have reproduction steps — no finding without repro
12. All adversarial activity logged to `.ovav/adversarial/logs/` with immutable audit trail

---

## 8. CONTEXT REFERENCES

| Artifact | Path | Relevant For |
|---|---|---|
| Kenji Lead Definition | `clients/opencode/agents/lead-kenji.md` | Authority scope, authorized functions, squad |
| Adversarial Intelligence Area | `clients/opencode/agents/area-adversarial-intelligence.md` | Area scope, rules of operation |
| Squad: Akiko | `clients/opencode/agents/team-akiko.md` | Semantic analysis specialist |
| Squad: Ryu | `clients/opencode/agents/team-ryu.md` | Architectural auditor |
| Squad: Mei | `clients/opencode/agents/team-mei.md` | Race condition hunter |
| Squad: Kaori | `clients/opencode/agents/team-kaori.md` | Boundary tester |
| Squad: Hiroshi | `clients/opencode/agents/team-hiroshi.md` | Autonomous pentester |
| Implementation Plan | `.ovav/plan/caps.yaml` | T9 cap (L315-343), v2.0.0 summary (L271-310) |
| cPanel OAuth (Go) | `go-runtime/cmd/cpanel/oauth.go` | OAuth Google + GitHub handlers |
| cPanel Auth (Go) | `go-runtime/cmd/cpanel/auth.go` | JWT + session management |
| cPanel Routes | `go-runtime/cmd/cpanel/routes.go` | All 17 API endpoint definitions |
| Validators (Go) | `go-runtime/internal/validators/validators.go` | Validator registry (61 validators) |
| Install Pipeline (Go) | `go-runtime/internal/install/` | Install, backup, rollback, manifest |
| Permission Authority | `.ovav/policy/permission_authority.json` | Agent permission matrix |
| OAuth Config | `config/oauth_config.yaml` | Provider client IDs + redirect URIs |
| Boundary Law | `.ovav/laws/ovav_laws.yaml` | LAW-001 Non-Invasion Area Boundary Law |
| Service Contracts | `.ovav/service_areas/areas/` | All 8+ area contracts |
| Topology Contracts | `.ovav/topology/` | Squad-to-area mappings |
| Rules of Engagement | `.ovav/adversarial/rules_of_engagement.yaml` | Red Team operational rules |
| Adversarial Logs | `.ovav/adversarial/logs/` | Immutable audit trail |

---

## 9. NOTES FOR KENJI

- **Es tu primera auditoría formal.** OVAV nunca ha sido atacado con mentalidad adversarial. Los validadores existentes son defensivos — verifican conformidad, no resistencia. Tu mirada es lo que falta.
- **El Go runtime es sólido pero no fue diseñado para resistir ataques.** 61 validadores, 0 data races, gofmt/vet limpio — pero nadie ha intentado romperlo. Los validadores mismos pueden ser el primer objetivo: ¿se pueden hacer pasar?
- **Los agentes son el perímetro más blando.** 68 agentes con system prompts, 9 leads con authority — cada uno es una superficie de ataque semántico. Akiko debería empezar por aquí.
- **cPanel OAuth ya fue auditado defensivamente** (Phase 3 hardening: CORS, CSRF, rate limiting, path traversal). Pero nadie lo ha atacado ofensivamente. Los CSRF state tokens son one-time con 10min TTL — ¿hay forma de predecirlos o reusarlos?
- **El install pipeline Go es nuevo** (port completo Python→Go, 2483 LOC, 81 tests). Es la infraestructura más crítica: si falla un rollback, OVAV queda en estado inconsistente. Mei debería priorizar race conditions aquí.
- **Tu área es soberana.** Si otro lead intenta limitar tu scope, redirigilos a este handoff. Solo el CEO puede frenarte. Si encontrás algo que asusta, reportalo inmediatamente — no esperes al informe final.
- **No arregles nada.** Encontrás, documentás, reportás. Thavren, Diana y Uriel implementan las mitigaciones. Tu valor está en lo que nadie más ve, no en lo que todos pueden arreglar.

---

*OVAV Governor System — Task9 Red Team Delegation*
*Delegated by: Thavren · Area: Platform Engineering*
*Delegated to: Kenji Tanaka · Area: Adversarial Intelligence (Red Team)*
