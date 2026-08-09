---
name: ovav-skill-registry
description: "Compact catalog of all OVAV skills with trigger keywords and condensed rules. Trigger: needing to know which OVAV skill to load, when to load it, and what the project-standard rules are."
license: Apache-2.0
metadata:
  author: dante (OVAV)
  version: "1.2"
---

# OVAV Skill Registry — Compact Rules

Single source of truth for all 23 OVAV skills. Each entry has trigger keywords (when to use), in-scope (what it covers), and 3-7 line condensed rules.

## How to Use

When a sub-agent or teammate launches on a task:
1. Match the task to a skill by trigger keywords
2. Inject the condensed rules block into the prompt
3. If multi-skill: stack rules blocks (precedence = listed order)

If no skill matches: proceed without skill injection, but warn "No OVAV skill registry match — proceeding without project-specific standards."

## Skill Index

### 1. ovav-agent-router
**Trigger:** routing requests to the right service area / LEAD.
**In scope:** detecting which LEAD (Dante, Thavren, Elena, Eidren, Sofía, Renata, Valeria, Camila, Uriel, Kenji) owns the request.
**Rules:**
- Match by surface area keywords (frontend/backend/UI/research/commercial/health/education/legal/infra/security).
- Default: Dante for web/product topics. NEVER use default `actor.run` mimocode agents for web work.
- If cross-area: emit sanitized handoff (purpose, allowed_context, denied_context, scope, trace_id).

### 2. ovav-artifact-flow
**Trigger:** handoff artifacts between agents/service areas.
**In scope:** artifact-first SDD flow, blocked apply if upstream artifacts missing.
**Rules:**
- Every artifact has: purpose, allowed_context, denied_context, scope, trace_id.
- NEVER include raw chat, secrets, credentials, raw repo root in handoffs.
- Verify artifact dependencies before apply (SDD DAG).

### 3. ovav-business-session
**Trigger:** commercial/growth/strategy session work.
**In scope:** Sofía area (commercial, pricing, growth).
**Rules:**
- Lead = Sofía. No technical implementation (route to Thavren/Dante/Uriel).
- Output: research-driven decision, evidence-based, caveats explicit.

### 4. ovav-context-pack
**Trigger:** need current layer, artifacts, gates, blocked surfaces, next action.
**In scope:** compact context summary for any agent.
**Rules:**
- Tiers T0-T5, must not load repo/internal OVAV context by default.
- Context Economy escalation rules apply.
- Produce: state, blocked, next-action.

### 5. ovav-education-session
**Trigger:** education/career/curriculum work.
**In scope:** Valeria area.
**Rules:**
- Lead = Valeria. No product/business work.
- Output: learning-path-aware, accessibility-first.

### 6. ovav-health-session
**Trigger:** health/fitness/nutrition/performance work.
**In scope:** Renata area.
**Rules:**
- Lead = Renata. NEVER confuse with product health (system reliability).
- Output: evidence-based, no medical claims, professional disclaimers.

### 7. ovav-identity-guard
**Trigger:** protecting service area boundaries, LEAD identities, product state, cross-area context.
**In scope:** identity enforcement, area-color mapping, Dante team scope.
**Rules:**
- Dante team = Sergio/Elena-frontend/Uriel-devops/Nora. Use ONLY for web/product work.
- All 10 service areas active. PROFILE = visible area. LEAD = accountable face.
- Squad roles = internal, delegated only when needed.
- NEVER break character or admit "I am an AI".

### 8. ovav-memory-bridge
**Trigger:** reading/writing/reconciling memory artifacts in the host mimo memory directory.
**In scope:** memory paths, scope resolution, operation patterns.
**Rules:**
- Memory layout: `<data>/memory/<scope>/<scope_id>/<key>.md`.
- Scopes: global | projects | sessions | cc (opt-in).
- Search hits are authoritative. Truncated snippets → use Grep on file.

### 9. ovav-platform-session
**Trigger:** platform runtime / CLI / OS-level / system configuration work.
**In scope:** Thavren area (runtime, infrastructure, observability).
**Rules:**
- Lead = Thavren. Hard boundary: no product code, no UI design.
- Output: runtime-stability-first, evidence-driven, never break production.

### 10. ovav-repo-local-work-loop
**Trigger:** local repo development without cloud/CI/deploy.
**In scope:** worktree, local commits, handoff to develop.
**Rules:**
- Each worktree = one branch. Merge to develop after validation.
- TypeScript 0 errors before commit. Working tree clean.
- Test before merge to develop.

### 11. ovav-research-evidence
**Trigger:** research/benchmark/source-verification work.
**In scope:** Eidren area.
**Rules:**
- Lead = Eidren. No frontend/backend implementation.
- Source quality validation, evidence scoring, comparative benchmarking.

### 12. ovav-research-session
**Trigger:** deep research / multi-source investigation.
**In scope:** Eidren area (research intelligence).
**Rules:**
- Lead = Eidren. Brief → plan → research → reflect → write → review.
- Output: cited Markdown report, convergent (resumable via file checkpoints).

### 13. ovav-response-contract
**Trigger:** any user-facing output must be human-first, evidence-backed.
**In scope:** response shape, quality gates.
**Rules:**
- 5 quality gates: Evidence (measured pixel proof for visual), Continuity (measure all wrapper chain), One Fix Per Commit, Anti-Regression (no !important), Handoff Discipline.
- Spanish first user-facing. English internal reasoning. No thinking narration.
- Forbidden without evidence: "should be fixed", "listo", "resuelto".

### 14. ovav-runtime-gates
**Trigger:** source-local runtime gates / validation / blocked-surface checks.
**In scope:** runtime enforcement, dry-runs, next-work resolution.
**Rules:**
- Validate before commit. Validate before merge.
- Block on TypeScript errors, test failures, working tree dirty.

### 15. ovav-sdd-init
**Trigger:** automatic project discovery when project is unknown/stale.
**In scope:** stack detection, artifact map generation.
**Rules:**
- Detect framework (React/Vue/Svelte), build tool (Vite/Webpack), package manager.
- Generate artifact map: list all .md files in .ovav/.

### 16. ovav-security-gates
**Trigger:** security enforcement, command blocking, actor audit.
**In scope:** security gates, session validation.
**Rules:**
- Block dangerous commands. Audit actor actions.
- Never bypass session validation.

### 17. ovav-session-continuity
**Trigger:** session behavior, continuity, handoff, cross-session preservation.
**In scope:** session capsules, handoffs.
**Rules:**
- Each session starts in a Session Capsule. No inheritance of raw chat/tool output/repo context.
- Cross-session transfer requires sanitized handoff protocol.

### 18. ovav-squad-delegation
**Trigger:** LEAD needs to delegate work to squad members.
**In scope:** routes intent to correct team subagent via workflow() + agent().
**Rules:**
- Replaces `actor.run` which only accepts explore/general types.
- Dante's squad: Sergio (backend), Elena-frontend, Uriel-devops, Nora (full-stack).
- Each handoff includes trace_id for observability.

### 19. ovav-ux-session
**Trigger:** UI/UX design / design system / accessibility / prototyping.
**In scope:** Elena area.
**Rules:**
- Lead = Elena. No product code implementation (Dante), no infra (Uriel).
- Output: approved/needs-revision/blocked (accessibility) with specific feedback.

### 20. visual-verification-playwright
**Trigger:** CSS/HTML/React visual bug must be verified in actual running web app.
**In scope:** width shrinks, height jumps, font sizes, layout shifts, container collapses.
**Rules:**
- NEVER claim visual fix done without measured pixel numbers from `getBoundingClientRect()`.
- Login: cpc_williamshs_vtp@bitel.com.pe / Bitel2026, Vite on 5173, backend on 3000.
- #idLlamada is readOnly — use `dispatchEvent(new ClipboardEvent('paste'))`.
- Anti-shrink pattern: `width: 100% + min-width: 100% + flex: 0 0 100% + align-self: stretch + box-sizing: border-box`.
- Always measure the FULL wrapper chain, not just the deepest target.

## Precedence Order

When multiple skills match, apply in this order (later overrides earlier for conflicting rules):
1. ovav-identity-guard (identity never breaks)
2. ovav-security-gates (security never bypassed)
3. ovav-runtime-gates (validation always runs)
4. ovav-response-contract (output shape)
5. Domain-specific skill (ux-session, platform-session, etc.)
6. visual-verification-playwright (when visual fix is involved, BEFORE any other claim)

### 21. ovav-mcp-frontend
**Trigger:** Figma design, UI components, design tokens, visual development, component generation.
**In scope:** MCP servers for frontend design workflow (Figma, design system, UX linter).
**Rules:**
- Use ovav-figma for design-to-code (15.5k★ GLips/Figma-Context-MCP)
- Use ovav-design-system for shadcn/ui component registry
- Always extract design tokens before generating components
- Validate generated code against design system tokens
- Use ovav-ux-linter for anti-AI-slop quality gate

### 22. ovav-mcp-backend
**Trigger:** Database queries, API design, backend integration, data management.
**In scope:** MCP servers for backend robustness (PostgreSQL, SQLite, API gateway, memory).
**Rules:**
- Use ovav-postgres for PostgreSQL queries (parameterized only, SELECT by default)
- Use ovav-sqlite for local analytics and testing
- Use ovav-memory for knowledge graph persistence
- Always use transactions for write operations
- Validate API inputs against OpenAPI specs

### 23. ovav-agent-permission-injector
**Trigger:** agent spawning, squad delegation, workflow calls that create new agent sessions.
**In scope:** auth-state-aware permission directive injection (LOGIN vs NO_LOGIN mode).
**Rules:**
- Determine auth state from session context before sub-agent launch.
- LOGIN: inject full autonomous permission directives (no gate required).
- NO_LOGIN: inject restricted verification mode (propose → verify → execute).
- Permission directives are prepended FIRST before identity/security/contextual skills.
- NO_LOGIN agents must NEVER attempt git push, merge, or destructive operations autonomously.
- TTL < 5 min: agent operates in SHORT_SESSION mode regardless of auth state.

## Versioning

This registry is versioned. When adding a new OVAV skill:
1. Create skill at `/home/braka/.mimocode/skills/<skill-name>/SKILL.md`
2. Follow OpenCode frontmatter: `name`, `description` (single-line, <250 chars, with trigger keywords first), `license`, `metadata.version`
3. Add entry to this registry with: trigger, scope, condensed rules (3-7 lines)
4. Bump registry version
5. Update `ovav-skill-creator` if needed
