# Research Brief: Live Memory — Why Existing Systems Fail & How Proactive Injection Wins

## Question
The user observes that despite multiple concurrent memory layers (AI model context, harness persistent memory, external memory systems), AI models still lose track of solutions already implemented. Why does this happen? And does "live proactive injection" solve it in a way existing reactive retrieval systems cannot?

## Problem Statement
Three memory layers exist simultaneously:
1. AI model memory (context window) — loses thread at capacity %
2. Harness persistent memory (MiMoCode memory.md, Claude Code, etc.) — reactive, not live
3. External memory layers (Engram, etc.) — "persistent" but not truly live

Despite all three, when the model struggles with `agends.md` errors, none retrieves the solution that was already implemented. Why?

User proposes: OVAV MEMORY as a truly "live" memory — always running in harness background, monitoring active context, proactively injecting when similarity is detected at the moment of struggle.

## Scope
In:
- Why reactive retrieval fails at moment of need (contextual retrieval failure)
- Proactive vs reactive memory injection patterns
- Background monitoring in AI agent harnesses
- Specific integration path for MiMoCode harness
- Real gains: what can live injection prevent that existing systems cannot?

Out:
- General memory architecture (already covered in previous research)
- Implementation details (design phase)

## Angles

1. **Contextual retrieval failure** — why do RAG/retrieval systems fail to retrieve at the moment the model needs them? What does research say about "lost in the middle" at the moment of active struggle?

2. **Proactive vs reactive memory** — is there research on proactive (push) vs reactive (pull) memory injection? Does any framework do this?

3. **Background monitoring in harnesses** — how can a memory system monitor an agent's context passively, without explicit tool calls? What are the technical approaches?

4. **MiMoCode harness integration** — how does MiMoCode load/expose agent context? What hooks exist for background monitoring?

5. **Real gains quantification** — what specifically would live injection prevent? Can this be quantified vs existing memory systems?

## Depth: standard
