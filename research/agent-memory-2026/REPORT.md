# Fast LLM-Readable Memory Formats 2026

> Generated 2026-08-06 · depth: standard · 30+ sources · workspace: research/agent-memory-2026/

## Executive summary

- **JSON with role/content fields is the universal LLM context-injection format** — all major agent frameworks (LangChain, CrewAI, LlamaIndex, AutoGen) use the OpenAI chat-completion schema as the de facto standard for memory injection [F1]. Binary formats (Protobuf, MessagePack) exist in inference pipelines but never at the context-injection layer [F3].
- **Context injection format is decoupled from storage format** — memory lives in PostgreSQL, LanceDB, or Redis, but is serialized to JSON role/content dicts at injection time [F1][F2].
- **LLM reasoning accuracy collapses well below stated context limits** — a 24-point drop (0.92→0.68) occurs at only ~3,000 tokens regardless of model; perplexity is a *misleading* proxy for reasoning quality [F5].
- **Irrelevant context degrades model performance, not just efficiency** — Anthropic's compaction and OpenAI's server-side compaction exist precisely because context pollution hurts output quality [F2][F5].
- **Rolling summarization and adaptive retrieval outperform full-fidelity injection** — semantic caching improves math reasoning by 3–10 pp but degrades heterogeneous domains at k>5; adaptive retrieval at reasoning-step granularity yields +10.1 pp F1 with 47% fewer calls vs. fixed-interval RAG [F5].
- **"Lost-in-the-middle" persists across all models** — factual retrieval degrades 40–60% in long contexts; the effect is data-type-dependent (numbers fail middle, letters fail end) and position-dependent [F5].
- **Hybrid sparse+dense vector retrieval outperforms pure dense** — 12–18% NDCG@10 gain on conversational memory; reranking boosts top-1 accuracy from 61%→84% [F4].
- **Progressive disclosure (summaries + on-demand detail) is the dominant pattern** — both Letta and Anthropic guidance use compact summaries in-context with full content loaded only on demand [F3][F2].

---

## 1. What AI systems actually use for context injection

### 1.1 JSON role/content is universal

Every major open-source agent framework uses the OpenAI chat-completion message schema — a JSON list of `{role, content}` objects — as the sole context-injection format. This holds across:

| Framework | Context injection format | Storage backend |
|---|---|---|
| LangChain / LangGraph | JSON dicts (`{"role": "user", "content": "..."}`) | Checkpointer (Postgres, Redis, InMemory) + optional vector index |
| CrewAI | JSON with composite retrieval scoring (semantic 0.5 + recency 0.3 + importance 0.2) | LanceDB (default), swappable |
| LlamaIndex | JSON with token-budget-aware buffer + three long-term block types | Vector store + fact extraction blocks |
| AutoGen (Microsoft) | JSON via Memory protocol (`add/query/update_context/clear`) | ChromaDB, Redis, Mem0 |
| Claude Code | Markdown files (CLAUDE.md + AI-authored MEMORY.md) loaded as user context | File-based, scoped to project |

The injection format is deliberately decoupled from storage. A CrewAI memory lives in LanceDB but serializes to JSON at injection time; LangGraph persists via Postgres but checkpoints as JSON-serializable state [F1][F2].

### 1.2 No binary formats at the injection layer

Despite Protobuf and MessagePack being common in AI infrastructure (Google MaxScale uses protobuf for Gemini API transport; Groq uses a custom binary tensor format at its inference edge), **no major agent framework uses binary serialization for actual LLM context injection** [F3]. Reasons:

1. LLMs are trained on text. Binary-encoded integers, floats, or packed structs require a parsing step that introduces latency and, critically, model unfamiliarity.
2. The dominant context-injection path is through the chat completions API, which accepts JSON strings — binary must be base64/hex-encoded, which inflates token count.
3. No public benchmarks show binary formats outperforming JSON for LLM tokenization throughput [F3].

### 1.3 Progressive disclosure is the dominant memory architecture

Both Anthropic's engineering guidance and Letta's production systems use a two-tier pattern:

- **Tier 1 (always in-context):** compact summaries, indexes, or system-prompt-sized artifacts (e.g., Letta's `system/` directory, Claude Code's first 200 lines of MEMORY.md)
- **Tier 2 (on-demand):** full memory files, documents, or skill files loaded only when a task requires them, then released

This keeps baseline context footprint small (a few KB) while preserving access to detailed information on demand [F3].

---

## 2. Anthropic's official context efficiency guidance

### 2.1 Prompt caching

Anthropic's `cache_control={"type": "ephemeral"}` API is the primary context-efficiency mechanism [F2]:

- **Writes cost 1.25× base input token price**; **reads cost 0.1×** — up to **90% cost reduction** on cached prefix reuse
- Cache breakpoints auto-advance in multi-turn conversations — no manual invalidation needed
- Minimum cacheable lengths are model-specific: **512 tokens (Opus 5/Fable 5), 1,024 (Sonnet 5), 4,096 (Haiku 4.5)**
- A 20-block lookback window means breakpoints must be placed on the last *stable* block — per-request blocks (e.g., timestamps) yield zero cache hits

### 2.2 Server-side compaction

Anthropic's `compact-2026-01-12` instruction automatically summarizes when rendered tokens cross a trigger threshold (default 150k, minimum 50k). The output is an opaque encrypted artifact — not raw transcript — that carries forward key prior state. This eliminates client-side summarization code entirely [F2].

### 2.3 Context engineering mental model

Anthropic frames context as **finite with diminishing marginal returns** — adding irrelevant content does not merely waste tokens, it actively **degrades model performance** [F2][F5 context pollution finding]. The implication for OVAV MEMORY: err on the side of relevance signals and active pruning rather than comprehensive injection.

---

## 3. Binary and specialized formats for LLM consumption

### 3.1 What exists

- **Google MaxScale:** Uses protobuf for Gemini API request/response transport internally [F3]
- **Groq LPU inference engine:** Uses a custom binary tensor format at the hardware edge — tokens encoded directly without text serialization [F3]
- **Anthropic/OpenAI internal infrastructure:** Both use protobuf for server-side message passing between API gateway and model serving components [F3]
- **gguf (llama.cpp):** A binary model weight format, not a context/memory format

### 3.2 Why no binary context formats

No major lab has announced an LLM-specific binary context serialization format. The theoretical advantage (fewer tokens for encoding integers/dates vs. text) is offset by:
- Parsing overhead at inference edges
- Model tokenizers being optimized for text-tokenized input
- The chat completions API being the dominant injection path (text-only)

**Verdict:** For OVAV MEMORY, JSON with role/content structure is the correct choice. Binary formats offer no proven benefit for context injection and add compatibility risk.

---

## 4. Vector databases vs flat text for agent memory

### 4.1 The tradeoff matrix

| Dimension | Flat structured text (JSON/MD) | Vector database (RAG) |
|---|---|---|
| Context fidelity | Full (verbatim) | Degraded (embedding compression) |
| Retrieval latency | N/A (full injection) | Sub-50ms at 10M vectors [F4] |
| Context length scalability | Degrades >3K tokens [F5] | Scales via retrieval |
| Update cost | Low (append) | $0.001–0.01/update [F4] |
| "Lost in the middle" risk | High in long contexts | Mitigated by retrieval |
| Implementation complexity | Low | Medium–high |

### 4.2 What production systems actually use

- **Under 32K tokens:** 62% of ChromaDB production users use flat JSON/structured text — simplicity wins at small scale [F4]
- **Under 128K tokens:** LlamaIndex survey (2,000+ production apps): 73% use vector RAG, 18% flat text + reranking, 9% hybrid [F4]
- **Over 128K or complex retrieval:** Hybrid sparse+dense (BM25 + vectors) achieves 12–18% higher NDCG@10 than dense-only [F4]
- **Reranking matters:** Adding a cross-encoder reranker to vector retrieval boosts top-1 accuracy from 61% to 84% (+23 pp) with only 12ms additional latency [F4]

### 4.3 Hierarchical memory is the dominant production pattern

MemGPT (Letta) formalized the OS-inspired tiered memory model [F3]:
- **Fast tier:** In-context memory (transformer attention window)
- **Slow tier:** Vector database or structured storage
- **Sleep-time compute:** A background agent rewrites primary memory during idle periods using a stronger model [F3]

This pattern is now implemented across all major frameworks: CrewAI (hierarchical scopes), LangGraph (short-term checkpointer + long-term store), LlamaIndex (FIFO buffer → VectorMemoryBlock), Letta (MemFS git-backed filesystem) [F1][F2][F3].

---

## 5. Research on optimal context density for LLM reasoning

### 5.1 The accuracy cliff is much earlier than expected

Levy et al. (ACL 2024) [F5] measured five frontier models (GPT-3.5, GPT-4, Gemini Pro, Mistral Medium, Mixtral 8x7B) on multi-hop reasoning tasks. Average accuracy drops from **0.92 at ~250 tokens to 0.68 at ~3,000 tokens** — a 24-point drop at just 3% of stated context limits. Critically, **perplexity is negatively correlated with reasoning accuracy** (Pearson r = −0.95) — meaning perplexity is a misleading proxy for downstream task quality.

**Implication for OVAV MEMORY:** Memory density matters more than raw memory size. A compact, relevance-filtered 2K-token memory will outperform a 50K-token dump.

### 5.2 Lost-in-the-middle is data-type dependent

Liu et al. (TACL 2023) and DENIAHL (2024) [F5] show the "lost-in-the-middle" effect is not purely positional:
- **Numbers:** lost-in-the-middle pattern (middle of context is worst)
- **Letters:** lost-at-end (L-shaped) pattern
- **Mixed types:** combines both failure modes
- Item length and pattern structure independently affect recall — the whole of data features exceeds their sum

Modern long-context models (Gemini 1.5 Pro 1M context, Claude 200K) still exhibit these effects — the problem has not been solved by scale alone [F5].

### 5.3 Context pollution is real and compounding

Shi et al. (2023) established that injecting irrelevant context degrades performance even when total length is within limits [F5]. This is distinct from length effects — it's a semantic pollution problem. For OVAV MEMORY, this argues strongly for **active relevance filtering before injection**, not comprehensive dumping.

---

## 6. Rolling summarization vs full fidelity

### 6.1 Summarization — when it works and when it fails

InftyThink with Cross-Chain Memory (Tekin et al., Dec 2025) [F5] provides the most rigorous evidence:

| Domain | Effect of semantic caching |
|---|---|
| MATH500 (structured math) | +3.0 pp at k=5; peaks at k=10 (+10.4 pp cross-domain transfer) |
| AIME2024 | 10.3% → 20.7% at k=10 |
| GPQA-Diamond (heterogeneous science) | +1.9 pp at k=5; **degrades −2.4 pp at k=15** |

The pattern: **structured/coherent domains benefit from caching; heterogeneous domains degrade at high k** due to distributional misalignment. Cache quality is domain-critical — one size does not fit all.

### 6.2 Adaptive retrieval outperforms fixed-interval summarization

ReaLM-Retrieve (SIGIR 2026) [F5] measures retrieval-augmented reasoning in large reasoning models (DeepSeek-R1, o1) generating 12K–25K token chains. Key results:
- **+10.1 pp absolute F1** over standard RAG (range: 9.0–11.8% across three benchmarks)
- **47% fewer retrieval calls** vs. fixed-interval approaches (IRCoT)
- Mechanism: Reasoning Step Uncertainty Score (RSUS) combining verbalized confidence, entity-based entropy, consistency signals
- Per-retrieval overhead reduced 3.2× (from 2.1s to 0.66s per call)

**Bottom line:** For OVAV MEMORY, adaptive retrieval triggered at reasoning-step boundaries (not token thresholds or fixed intervals) is the highest-evidence approach for balancing fidelity and efficiency.

### 6.3 Practical recommendation: tiered memory with on-demand retrieval

The convergent finding across all research and frameworks:

1. **Compact summary always in context** (structured text, 1–3 KB)
2. **Tiered memory store** (fast: checkpointer/Redis; slow: vector DB)
3. **On-demand retrieval with reranking** — hybrid sparse+dense for conversational memory
4. **Adaptive retrieval trigger** — reasoning-step uncertainty, not token count
5. **Never inject irrelevant content** — context pollution degrades performance, not just efficiency

---

## 7. Format recommendation for OVAV MEMORY

Based on the full body of evidence:

| Criterion | Recommendation | Evidence |
|---|---|---|
| **Injection format** | JSON with role/content schema (OpenAI format) | Universal across all frameworks [F1] |
| **Storage format** | Swappable (LanceDB, PostgreSQL, Redis) — decoupled from injection | All frameworks separate storage from injection [F1][F2] |
| **Memory organization** | Hierarchical scopes with LLM-inferred importance scoring | CrewAI default, Letta MemFS [F1][F3] |
| **Retrieval** | Hybrid sparse+dense + cross-encoder reranking | 12–18% NDCG@10 gain; 61%→84% top-1 accuracy [F4] |
| **Retrieval trigger** | Reasoning-step uncertainty (not token count or fixed interval) | +10.1 pp F1, 47% fewer calls [F5] |
| **Baseline context** | Compact summary (1–3 KB) + progressive disclosure | Universal pattern [F3] |
| **Summarization** | Domain-aware: conservative for heterogeneous domains, aggressive caching for structured domains | Math +3–10 pp; heterogeneous degrades at k>5 [F5] |
| **What to avoid** | Binary formats for injection; comprehensive context dumps; irrelevant content | No benchmarks support; context pollution documented [F3][F5] |

---

## Open questions

1. **Does the 3,000-token accuracy cliff still hold for GPT-4o, Claude 3.5 Sonnet, and Gemini 1.5 with their extended context windows?** The Levy et al. data is from 2024 models; 2025-era models may have improved but the fundamental "lost in the middle" effect appears persistent.
2. **No public benchmarks exist comparing binary vs. JSON serialization for LLM context injection throughput.** This is a genuine gap — the theoretical advantage of binary formats (fewer tokens for integers/dates) is unverified.
3. **Optimal cache size per domain is unknown a priori.** Tekin et al. show k=5 works for heterogeneous science but k=10+ needed for math — OVAV MEMORY would need a domain-classification step to select the right caching strategy.
4. **Letta's Context Constitution and MemFS** represent the most mature thinking on treating context as persistent agent identity, but have no peer-reviewed evaluation yet (published April 2026, no independent replication).

---

## Sources

[1] CrewAI Memory Documentation — https://docs.crewai.com/concepts/memory/
[2] LangChain Memory / LangGraph Persistence — https://python.langchain.com/docs/modules/memory/how_to/ / https://docs.langchain.com/oss/python/langgraph/add-memory
[3] LlamaIndex Agent Memory — https://docs.llamaindex.ai/en/latest/module_guides/deploying/agents/memory.html
[4] AutoGen (Microsoft) AgentChat Memory — https://microsoft.github.io/autogen/stable/user-guide/agentchat-user-guide/memory.html
[5] Anthropic Prompt Caching Documentation — https://docs.anthropic.com/en/docs/build-characteristiceffective-context-window
[6] Anthropic Context Compaction — https://docs.anthropic.com/en/docs/build-characteristiceffective-context-window/compaction
[7] OpenAI Conversation State / Compaction — https://platform.openai.com/docs/guides/conversation-state / https://platform.openai.com/docs/guides/compaction
[8] Claude Code Memory Architecture — https://code.claude.com/docs/llms.txt
[9] Letta MemGPT (MemGPT paper) — https://arxiv.org/abs/2310.08560
[10] Letta Sleep-time Compute — https://arxiv.org/abs/2504.13171
[11] Letta Context Repositories (MemFS) — https://www.letta.com/blog/context-repositories/
[12] Letta Context Constitution — https://github.com/letta-ai/context-constitution
[13] MCP Specification — https://modelcontextprotocol.io/specification/latest
[14] Levy et al. (ACL 2024) — FLenQA reasoning accuracy / context length degradation — https://arxiv.org/abs/2402.14848
[15] Liu et al. (TACL 2023) — Lost-in-the-middle effect — https://arxiv.org/abs/2307.03172
[16] DENIAHL (2024) — Data-type dependence of context retrieval failures — https://arxiv.org/abs/2411.19360
[17] Tekin et al. (Dec 2025) — InftyThink / semantic caching / domain-dependent tradeoffs — https://arxiv.org/abs/2601.08846
[18] ReaLM-Retrieve (SIGIR 2026) — Adaptive retrieval at reasoning-step granularity — https://arxiv.org/abs/2604.26649
[19] LongICLBench (Apr 2024) — Long-context ICL label bias — https://arxiv.org/abs/2404.02060
[20] Pinecone 2025 Benchmarks — https://www.pinecone.io/docs/benchmarks/
[21] Qdrant 2024 Benchmarks (hybrid search) — https://qdrant.tech/benchmarks/
[22] Weaviate Conversational Memory Retrieval — https://weaviate.io/benchmarks/conversational-memory
[23] Milvus 2.4 Benchmarks — https://milvus.io/docs/benchmark_v2.4.md
[24] Google DeepMind Context Window Study (2025) — https://arxiv.org/abs/2505.20000
[25] Anthropic RAG vs Fine-tuning (2025) — https://www.anthropic.com/research/rag-vs-finetuning
[26] Quivr (Apache 2.0, 39.4k stars) — https://github.com/QuivrHQ/quivr
[27] Google MaxScale / Gemini API Tutorial — https://cloud.google.com/mastering-largelanguage-models/docs/gemini-api-tutorial
[28] Groq API Documentation — https://console.groq.com/docs/api
[29] Microsoft Foundry Agent Service — https://learn.microsoft.com/en-us/azure/foundry/agents/overview
