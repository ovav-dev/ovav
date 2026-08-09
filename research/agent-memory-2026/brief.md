# Brief: Passive Live Memory Ingestion in AI Agents

**Date**: 2026-08-06
**Depth**: standard
**Workspace**: /home/braka/Systems/OVAV/research/agent-memory-2026/

## Research question
How do modern AI agent systems implement passive live memory — collecting user data in real-time without explicit "remember this" prompts — and what architectural patterns, weighting schemes, trigger mechanisms, and privacy controls govern that ingestion?

## Scope
Focus: production agent systems (Cline, Cursor, Claude Code/GitHub Copilot, Copilot, Roo Code, etc.).
Context: living memory / persistent context systems for developer tooling.
Boundary: real, shipped implementations preferred over academic prototypes.

## Research angles

### Angle 1 — Data collection: what, why, how
What data do production agents passively collect? (conversations, file changes, decisions, errors, tool calls, git events?) How is it stored? What is the user-visible model?

### Angle 2 — Memory weighting & trigger systems
How do agents weight memory by importance, recency, or context relevance? What triggers a memory write — tool calls, file edits, decision points, explicit checkpoints? Event-driven patterns.

### Angle 3 — Privacy, sensitivity & opt-out
How do these systems handle data that should NOT be stored? Privacy/sensitivity labels, exclusion patterns, per-memory-type opt-out, audit trails.

### Angle 4 — Open-source / academic alternatives
Beyond proprietary agents, what frameworks or research systems exist for passive memory ingestion in agents? LangChain memory, MemGPT, Letta, smol agents, etc.

## Assumptions
- Target is a developer/knowledge-worker agent context, not a consumer chatbot.
- The output is a design input for OVAV MEMORY.
- English-language sources preferred, but Chinese-context systems (e.g. Chinese LLM agents) are relevant if found.
