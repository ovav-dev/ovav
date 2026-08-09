# Source Map — Lead Accountability Protocol

| ID | Source | Type | Tier | Key Data | Relevance |
|---|---|---|---|---|---|
| S1 | RARR (Gao et al., arXiv 2210.08726) | Paper (ACL 2023) + Implementation | T1 | Post-hoc attribution + revision; uses LM + web search; significantly improves attribution | Frente A: verificación post-modelo con corrección |
| S2 | Self-RAG (Asai et al., arXiv 2310.11511) | Paper (2023) + Implementation | T1 | 7B/13B models outperform ChatGPT on QA, reasoning, fact verification; adaptive retrieval + self-reflection tokens | Frente A: verificación con auto-crítica |
| S3 | SelfCheckGPT (Manakul et al., arXiv 2303.08896) | Paper (EMNLP 2023) + Code | T1 | Zero-resource hallucination detection; sampling consistency; higher AUC-PR than grey-box methods | Frente A: detección de alucinaciones sin DB externa |
| S4 | Guardrails AI (guardrailsai.com) | Product (Apache 2.0) | T2 | Input/Output Guards; Guardrails Hub con validators reutilizables; Guardrails Index (Feb 2025): benchmark de 24 guardrails en 6 categorías; integración MLflow (Mar 2026) | Frentes A+B: framework de validación en producción |
| S5 | NeMo Guardrails (NVIDIA, v0.21.0) | Product (Apache 2.0) + Paper (arXiv 2310.10501) | T1 | 5 tipos de rails (input, dialog, retrieval, execution, output); "self check facts" y "self check hallucination"; protección jailbreak; Colang DSL | Frentes A+B: toolkit programable con verificación factual |
| S6 | SAFE/LongFact (Wei et al., arXiv 2403.18802) | Paper (Google DeepMind, 2024) | T1 | Descompone respuestas long-form en facts individuales; evalúa cada fact via multi-step reasoning + Google Search; métrica F1 para factualidad | Frente A+C: evaluación automatizada de factualidad long-form |
| S7 | TruthfulQA (Lin et al., arXiv 2109.07958) | Paper (ACL 2022) + Benchmark público | T1 | 817 preguntas, 38 categorías; GPT-3 solo 58% truthful vs 94% humanos; modelos más grandes = menos truthful | Frente C: baseline de veracidad pre-verificación |
| S8 | HaluEval (Li et al., arXiv 2305.11747) | Paper (EMNLP 2023) + Benchmark | T1 | ChatGPT: ~19.5% respuestas con alucinaciones; LLMs existentes no detectan bien alucinaciones; conocimiento externo ayuda | Frente C: tasa de alucinación + efectividad de mitigación |
| S9 | Anthropic Core Views on AI Safety (anthropic.com) | Documento oficial (Mar 2023) | T2 | Enfoque: scaling supervision, mechanistic interpretability, process-oriented learning; Constitutional AI; "show, don't tell" | Frente B: filosofía de supervisión humana + AI |
| S10 | Google Fact Check Tools (toolbox.google.com/factcheck) | Producto/API | T2 | API para acceder a artículos de fact-checking de publishers reputados; base de datos de claims verificados | Frente A: fuente externa de verificación |

## Fuentes adicionales (menor peso)

| ID | Source | Nota |
|---|---|---|
| S11 | Claude 3.5 Sonnet model card | 64% agentic coding eval vs 38% Claude 3 Opus; relevante para mejora incremental pero no específicamente para verificación factual |
| S12 | Guardrails AI Blog (Snowglobe, Changi Airport) | Casos de uso en producción pero sin métricas de mejora cuantitativas publicadas |
