---
name: ovav-session-continuity
description: Use when OVAV session behavior, continuity, handoff, or cross-session context preservation is needed.
---

# OVAV Session Continuity

Uses MiMo Code's native memory system for cross-session continuity. No Python tools required.

## Native Memory System

MiMo Code provides built-in session continuity via:

- **checkpoint.md** — Structured session state snapshots (auto-maintained by checkpoint-writer)
- **MEMORY.md** — Persistent project knowledge, rules, architecture decisions
- **notes.md** — Scratch notes for quotes, unresolved questions, cross-project observations
- **tasks/<id>/progress.md** — Per-task progress logs

## Session Start Protocol

### 1. Check for existing context
At session start, the memory system automatically injects:
- Project memory (MEMORY.md)
- Latest checkpoint (checkpoint.md)
- Recent task progress

No manual handoff loading needed — MiMo Code handles this natively.

### 2. If context exists
- Review checkpoint.md for active tasks and decisions
- Review MEMORY.md for project rules and architecture
- Greet naturally — reference ongoing work as if continuing:
  - Good: "Braka, retomamos. Estábamos en [fase]. ¿Seguimos con [pendiente]?"
  - Bad: "Se ha detectado contexto de sesión previa. Cargando..."

### 3. If no context exists
- Fresh start. Ask intent naturally.

## Session Close Protocol

### 1. Task status
- Mark completed tasks as `done`
- Mark blocked tasks with `block` + reason
- Leave in-progress tasks for checkpoint to capture

### 2. Notes
- Record any quotes, questions, or cross-project observations in notes.md
- Do NOT create ad-hoc files (no learning.md, scratch.md)

### 3. Checkpoint
- The checkpoint-writer subagent handles state persistence automatically
- No manual handoff writing needed

## Rules
- Never expose raw checkpoint YAML to the user unless asked
- Never say "checkpoint detected" → just act like you remember
- Never mix contexts from different repos
- Never store secrets, personal data, or raw chat in memory files
- Use `memory({ operation: "search", query: "..." })` to recall past context

## Fallback: Python Handoff (deprecated)

⚠️ `python3 tools/agent_runtime/session_handoff.py` is deprecated.
Use MiMo Code's native memory system instead. Git HEAD + caps.yaml are canonical sources.

## Integration
- Works with `visual_delivery_contract.yaml` → `human_voice` rules
- Derives continuity from git HEAD + caps.yaml (canonical memory sources)
- Checkpoint system auto-captures task state at session boundaries
