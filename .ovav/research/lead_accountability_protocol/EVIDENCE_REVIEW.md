# Evidence Review — Lead Accountability Protocol

## Frente A — Verificación post-modelo

### Lo que sí existe y funciona

**1. Verificación con evidencia externa (RARR, SAFE)**
RARR demuestra que es factible tomar output de cualquier LLM, buscar atribución automáticamente, y corregir el contenido no soportado. SAFE (Google DeepMind, 2024) extiende esto a respuestas long-form: descompone en facts individuales, verifica cada uno contra Google Search, produce métrica F1. Ambos están publicados en venues top (ACL, DeepMind).

**2. Auto-verificación interna (Self-RAG, SelfCheckGPT)**
Self-RAG entrena al modelo para decidir cuándo recuperar evidencia y auto-criticar sus respuestas usando "reflection tokens". SelfCheckGPT usa consistencia entre múltiples muestreos: si el modelo conoce el tema, las muestras convergen; si alucina, divergen. Ambos funcionan sin base de datos externa (SelfCheckGPT) o con recuperación adaptativa (Self-RAG).

**3. Guardrails programables (Guardrails AI, NeMo Guardrails)**
Ambos son Apache 2.0, en producción, con comunidades activas. NeMo tiene 5 tipos de rails con hooks en cada etapa del pipeline LLM. Guardrails AI lanzó en Feb 2025 el Guardrails Index — benchmark de 24 guardrails. Ambos permiten definir reglas determinísticas + verificaciones basadas en LLM.

**4. Fact-checking externo (Google Fact Check Tools)**
API que indexa artículos de fact-checking de publishers verificados. Utilizable como fuente de verdad externa, aunque su cobertura es limitada a claims que ya han sido verificados por humanos.

### Lo que es hype o no tiene evidencia suficiente

- **"AI quality gates" como producto empaquetado**: No existe un producto off-the-shelf que sea "enchufar y verificar". Los sistemas existentes (Guardrails AI, NeMo) son frameworks que requieren configuración sustancial.
- **Verificación en tiempo real con latencia cero**: Todos los sistemas introducen latencia. RARR requiere búsqueda web + llamado adicional al LLM. SelfCheckGPT requiere múltiples samples.
- **Fact-checking 100% automatizado**: Google Fact Check Tools depende de fact-checkers humanos previos. No cubre claims nuevos en tiempo real.

---

## Frente B — Ciclos de corrección humana + AI

### Lo que sí existe y funciona

**1. Guardrails con supervisión humana (Guardrails AI + NeMo)**
Ambos permiten flujos donde el output del LLM pasa por validadores determinísticos + basados en LLM antes de llegar al usuario. Si un validador falla, se puede re-promptear al LLM, pedir corrección, o escalar a un humano. La integración Guardrails AI + MLflow (Mar 2026) permite usar validadores como "scorers" en pipelines de evaluación.

**2. Anthropic: Constitutional AI en runtime**
Anthropic no publica su implementación interna exacta, pero su filosofía documentada (Mar 2023) describe un enfoque multicapa: scaling supervision (humanos supervisan + AI ayuda a escalar), mechanistic interpretability (entender qué hace el modelo internamente), y process-oriented learning (el modelo aprende procesos verificables, no solo respuestas). Claude muestra en la práctica mayor adherencia a factualidad que modelos comparables. Sin embargo, los detalles de implementación runtime no son públicos.

**3. NeMo Guardrails: "self check facts" y "self check hallucination"**
Estos son rails de output que le piden al LLM que verifique sus propias afirmaciones. Son configurables y programables. El usuario define qué verificar y cómo. Esto es un ciclo automático de corrección, no humano, pero es el mecanismo base sobre el que se puede construir supervisión humana.

### Lo que es hype o no tiene evidencia suficiente

- **"Human-in-the-loop" como producto**: La mayoría de las implementaciones son ad-hoc por empresa. No hay un framework estándar. Cada organización construye su propio pipeline.
- **Casos de estudio públicos con métricas**: Muy escasos. El caso de Changi Airport con Snowglobe/Guardrails AI existe, pero no publica métricas de mejora de factualidad — solo de seguridad conversacional.
- **"AI propone, humano verifica, AI corrige" como ciclo cerrado**: En la práctica, el humano suele ser el cuello de botella. Los sistemas que escalan bien usan verificación automática como primer filtro y solo escalan a humano los casos borderline.

---

## Frente C — Métricas reales de mejora

### Datos duros

| Métrica | Valor | Fuente | Confianza |
|---|---|---|---|
| TruthfulQA baseline (GPT-3, sin verificación) | 58% truthful | S7 (Lin et al., ACL 2022) | ALTA |
| TruthfulQA humanos | 94% truthful | S7 | ALTA |
| ChatGPT tasa de alucinación | ~19.5% de respuestas | S8 (HaluEval, EMNLP 2023) | ALTA |
| Mejora con conocimiento externo en detección | Significativa (LLMs detectan mejor alucinaciones con contexto externo) | S8 | MEDIA (cualitativo) |
| SelfCheckGPT vs grey-box (AUC-PR) | "Considerably higher" | S3 (SelfCheckGPT, EMNLP 2023) | MEDIA (sin número exacto en abstract) |
| Self-RAG vs ChatGPT (QA, reasoning, fact verification) | "Significantly outperforms" | S2 (Self-RAG, 2023) | MEDIA (sin número exacto en abstract) |
| RARR mejora de atribución | "Significantly improves" | S1 (RARR, ACL 2023) | MEDIA (sin número exacto en abstract) |

### Lo que NO sabemos (gaps críticos)

1. **Tasa de falsos positivos**: ¿Con qué frecuencia un sistema de verificación marca como falso algo que es verdadero? Ninguna de las fuentes revisadas publica este número de forma clara. Es el gap más peligroso para un "accountability protocol".

2. **Mejora cuantitativa neta**: ¿Cuánto mejora la confiabilidad al añadir verificación? Los papers dicen "significantly improves" pero los abstracts no contienen el número exacto. Se requiere lectura completa de los papers para extraer las tablas de resultados.

3. **Costo en latencia**: RARR requiere búsqueda web + llamado extra al LLM. SelfCheckGPT requiere múltiples samples. SAFE requiere búsqueda Google + razonamiento multi-step. Ningún paper mide latencia en producción.

4. **Efectividad en dominios específicos**: TruthfulQA cubre 38 categorías generales. No hay evidencia de cómo funcionan estos sistemas en dominios técnicos especializados (ej. arquitectura de software, sistemas distribuidos).

5. **Degradación con el tiempo**: ¿Los guardrails se vuelven obsoletos? ¿Los validadores necesitan actualización constante? No hay datos.
