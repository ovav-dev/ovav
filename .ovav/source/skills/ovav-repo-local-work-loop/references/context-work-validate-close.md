# Context → Work → Validate → Close Loop

## 1. Context (automatic on session start)

- Run `context` discovery silently.
- Greeting/simple tasks: load only identity_baseline + session_baseline.
- Implementation tasks: add implementation_context + registry_guard.
- Research tasks: add research_evidence.
- Safety-related: add safety_gate.
- Do NOT load full artifacts, full registries or full harness catalog for simple tasks.

## 2. Work (route to correct service area)

- Platform/system/runtime/config/OpenCode → Platform Engineering (lead: Thavren).
- Research/evidence/benchmark/source verification → OVAV Research Intelligence / Eidren.
- If ambiguous: ask one compact Spanish clarification.
- Use Harness Intelligence Router for harness family selection.

## 3. Validate (automatic before close)

- Low-risk: compact checks only.
- Implementation: segment validation + registry check + OpenCode safety.
- Closure: strict validation + artifact drift + local Git safety.
- Run checkers: check_repo_local_work_loop as primary gate.
- Preserve historical prerequisite checks only when current validators depend on them.

## 4. Close (local-only, after validation)

- Build closure evidence card: what changed, evidence paths, validation, risks, next.
- Dry-run close-layer. No remote/origin workflow.
- Restore historical artifact drift after validators.
- Stage current-segment files only. Never stage broad folders blindly.
- Do NOT commit unless user explicitly approves.
- Never push, never delete branches, never create next branches.
