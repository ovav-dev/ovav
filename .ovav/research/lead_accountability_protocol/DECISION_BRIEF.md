# Decision Brief — Lead Accountability Protocol

**Para:** CEO, OVAV
**De:** Eidren, Research Intelligence Lead
**Fecha:** 2026-06-06
**Confianza general:** MEDIA-ALTA
**Urgencia:** Alta

---

## Resumen ejecutivo

Existe tecnología madura para construir un sistema de verificación post-modelo. Los patrones están claros: (1) guardrails programables en el pipeline LLM, (2) verificación automática contra evidencia externa, (3) auto-verificación por consistencia. Lo que **no existe** es un producto off-the-shelf que haga todo esto junto. OVAV tendrá que componerlo.

El gap más crítico: **no hay datos publicados de tasas de falsos positivos** en sistemas de verificación. Un protocolo de accountability que marque como falso algo verdadero es peor que no tener protocolo. Esto debe ser diseñado con extremo cuidado.

---

## Top 5 sistemas/patrones con evidencia de efectividad

### 1. Guardrails programables en pipeline (NeMo Guardrails)
**Qué es:** 5 tipos de rails (input, dialog, retrieval, execution, output) que interceptan cada etapa del pipeline LLM. Incluye "self check facts" y "self check hallucination" como rails de output predefinidos.
**Evidencia:** Paper académico (arXiv 2310.10501), Apache 2.0, v0.21.0 en producción, documentación extensa de NVIDIA.
**Relevancia para OVAV:** El patrón de "output rails" es exactamente lo que necesita el Accountability Protocol: interceptar el output del LEAD antes de que llegue al usuario, verificarlo, y bloquear/desviar si no cumple el threshold.
**Recomendación:** **ADOPTAR** el patrón de output rails. No necesariamente usar NeMo (dependencia pesada), pero sí el concepto de intercepción programable post-generación.

### 2. Verificación contra evidencia externa (RARR / SAFE)
**Qué es:** RARR busca atribución automáticamente para cada claim del LLM y corrige el contenido no soportado. SAFE (Google DeepMind) descompone respuestas long-form en facts individuales y verifica cada uno contra Google Search.
**Evidencia:** RARR publicado en ACL 2023. SAFE publicado por Google DeepMind (2024). Ambos con implementación.
**Relevancia para OVAV:** El Accountability Protocol necesita verificar claims técnicos contra documentación real, código, o fuentes externas. El patrón RARR/SAFE (descomponer → buscar evidencia → verificar → corregir) es directamente aplicable.
**Recomendación:** **ADAPTAR** el patrón. Para OVAV, la "evidencia externa" no es Google Search sino el repositorio local, la documentación del proyecto, y los artefactos de OVAV mismo. Esto es más acotado y más verificable que búsqueda web abierta.

### 3. Auto-verificación por consistencia (SelfCheckGPT)
**Qué es:** Muestrea el LLM múltiples veces con la misma pregunta. Si las respuestas son consistentes entre sí, el contenido es probablemente factual. Si divergen, es probablemente alucinación. Zero-resource: no requiere base de datos externa.
**Evidencia:** EMNLP 2023. AUC-PR significativamente superior a métodos grey-box.
**Relevancia para OVAV:** Es el método más ligero: no requiere infraestructura externa, solo múltiples llamados al modelo. Ideal como primera capa de verificación antes de verificaciones más costosas.
**Recomendación:** **ADOPTAR** como capa rápida de detección. Si el LEAD genera output inconsistente entre muestras → señal de alarma inmediata.

### 4. Self-RAG: modelo que sabe cuándo verificar
**Qué es:** Entrena al LLM para decidir autónomamente cuándo necesita recuperar evidencia y para auto-criticar sus respuestas mediante "reflection tokens". 7B/13B models superan a ChatGPT en fact verification.
**Evidencia:** Paper 2023 con implementación. Resultados en Open-domain QA, reasoning, y fact verification.
**Relevancia para OVAV:** Si OVAV pudiera entrenar/fine-tunear a sus LEADs para que internamente sepan cuándo un claim necesita verificación, el protocolo sería más eficiente. Pero requiere acceso al training del modelo, lo cual no es viable con modelos API.
**Recomendación:** **MONITOREAR**. El patrón de "reflection tokens" es prometedor pero requiere modelos propios. Para OVAV con modelos API, es más práctico implementar la verificación como capa externa (patrones 1-3).

### 5. Guardrails determinísticos + scorers (Guardrails AI + MLflow)
**Qué es:** Framework Python con validadores reutilizables (Guardrails Hub). Integración reciente con MLflow (Mar 2026) para usar validadores como "scorers" determinísticos en pipelines de evaluación. Guardrails Index (Feb 2025) benchmarkea 24 guardrails en 6 categorías.
**Evidencia:** Apache 2.0, 3,238 commits, adopción enterprise (Snowglobe, Changi Airport, MasterClass).
**Relevancia para OVAV:** El concepto de "validadores como scorers" es directamente aplicable: cada output del LEAD recibe un score de confianza en múltiples dimensiones (factualidad, completitud, consistencia, seguridad). Si el score < 85%, se bloquea.
**Recomendación:** **ADAPTAR** el concepto de scoring multidimensional. No necesariamente usar Guardrails AI (dependencia Python), pero sí el patrón de validadores componibles con scores.

---

## Lo que funciona vs lo que es hype

### ✅ FUNCIONA (evidencia sólida)
- **Guardrails en el pipeline LLM**: Patrón probado. NeMo y Guardrails AI lo demuestran.
- **Verificación automática contra fuentes**: RARR y SAFE lo prueban académicamente. El principio es sólido.
- **Detección de alucinaciones por consistencia**: SelfCheckGPT funciona sin infraestructura externa.
- **Scoring determinístico**: Guardrails AI + MLflow muestran que validadores determinísticos (regex, PII, toxicidad) son fiables.

### ⚠️ HYPE (evidencia débil o inexistente)
- **"AI quality gates off-the-shelf"**: No existe. Hay que construirlos.
- **"100% automated fact-checking"**: Incluso Google Fact Check Tools depende de verificación humana previa.
- **"Human-in-the-loop" como producto**: Cada empresa lo construye ad-hoc. No hay estándar.
- **"Zero-latency verification"**: Todo método de verificación añade latencia. RARR y SAFE requieren búsquedas. SelfCheckGPT requiere múltiples samples.
- **Métricas públicas de mejora en producción**: Casi inexistentes. Las empresas no publican estos números.

---

## Recomendación concreta para OVAV

### Qué construir

**Un "Trust Gate" de 3 capas para el Lead Accountability Protocol:**

```
LEAD genera output
    ↓
[Capa 1] SelfCheck → Consistencia interna
    ↓ (pasa si consistente)
[Capa 2] Evidence Check → Verificación contra docs/repo local
    ↓ (pasa si respaldado)
[Capa 3] Trust Score → Scoring multidimensional ≥ 85%
    ↓ (pasa si score suficiente)
ENTREGAR al usuario
```

**Capa 1 — SelfCheck (adopción directa de SelfCheckGPT):**
- Muestrear el modelo 3-5 veces con el mismo prompt.
- Medir consistencia semántica entre respuestas.
- Si divergen significativamente → bloquear y forzar reinvestigación.
- **Costo:** 3-5x tokens. **Beneficio:** detecta alucinaciones sin infraestructura externa.
- **Confianza en esta capa:** MEDIA. Es una señal, no una garantía.

**Capa 2 — Evidence Check (adaptación de RARR/SAFE):**
- Descomponer el output del LEAD en claims individuales.
- Para cada claim, buscar evidencia en: repositorio local, documentación OVAV, artefactos del proyecto.
- Si un claim no tiene respaldo → marcarlo.
- **Costo:** Búsqueda local + llamados adicionales al LLM. **Beneficio:** verificación con evidencia concreta.
- **Confianza en esta capa:** ALTA cuando la evidencia existe. No aplica para claims sobre conocimiento externo.

**Capa 3 — Trust Score (adaptación de Guardrails AI scorers):**
- Validadores determinísticos: consistencia, completitud, no contradicción con estado conocido.
- Cada validador produce un score 0-1.
- Score agregado debe ser ≥ 85% para liberar el output.
- Si no alcanza → devolver al LEAD con indicaciones específicas de qué corregir.
- **Confianza en esta capa:** MEDIA-ALTA. Depende de la calidad de los validadores.

### Qué evitar

1. **No construir un fact-checker universal**: Acotar la verificación a lo que OVAV puede verificar contra su propio conocimiento (repo, docs, artefactos). No intentar verificar claims sobre el mundo externo sin fuentes.

2. **No depender de un solo método**: La combinación de capas (consistencia + evidencia + scoring) es más robusta que cualquier método individual.

3. **No asumir que 85% es alcanzable en todos los contextos**: Habrá preguntas donde la evidencia es ambigua. El protocolo debe degradar con gracia: "no puedo verificar esto con suficiente confianza, pero aquí está lo que sé".

4. **No ignorar los falsos positivos**: Un sistema que bloquea output correcto es más dañino que uno que deja pasar output incorrecto. El threshold de 85% debe ser calibrado con datos reales. Si no hay datos, empezar con threshold bajo (60-70%) y subir gradualmente.

5. **No copiar implementaciones externas sin adaptar**: NeMo Guardrails y Guardrails AI son frameworks Python pesados. Para OVAV, que opera a nivel de sistema, probablemente convenga una implementación más ligera y nativa.

### Próximo paso inmediato

**Experimento controlado:** Tomar 20 outputs reales de Thavren/Eidren, aplicar las 3 capas manualmente (simuladas), medir:
- Cuántos outputs fueron correctamente bloqueados (verdaderos positivos)
- Cuántos outputs correctos fueron bloqueados (falsos positivos) ← **crítico**
- Cuántos outputs incorrectos pasaron (falsos negativos)
- Tiempo/latencia añadida

Sin este experimento, cualquier implementación es especulación.

---

## Fuentes verificables

| ID | Fuente | Link |
|---|---|---|
| S1 | RARR (Gao et al., ACL 2023) | https://arxiv.org/abs/2210.08726 |
| S2 | Self-RAG (Asai et al., 2023) | https://arxiv.org/abs/2310.11511 |
| S3 | SelfCheckGPT (Manakul et al., EMNLP 2023) | https://arxiv.org/abs/2303.08896 |
| S4 | Guardrails AI | https://github.com/guardrails-ai/guardrails |
| S5 | NeMo Guardrails (NVIDIA) | https://github.com/NVIDIA/NeMo-Guardrails |
| S6 | SAFE/LongFact (Wei et al., DeepMind 2024) | https://arxiv.org/abs/2403.18802 |
| S7 | TruthfulQA (Lin et al., ACL 2022) | https://arxiv.org/abs/2109.07958 |
| S8 | HaluEval (Li et al., EMNLP 2023) | https://arxiv.org/abs/2305.11747 |
| S9 | Anthropic Core Views on AI Safety | https://www.anthropic.com/research/core-views-on-ai-safety |
| S10 | Google Fact Check Tools | https://toolbox.google.com/factcheck/explorer |

---

## Notas finales

- **Confianza de este brief:** MEDIA-ALTA. Las fuentes son sólidas (ACL, EMNLP, DeepMind, productos Apache 2.0 activos), pero los números exactos de mejora requieren lectura completa de los papers (los abstracts no contienen todas las tablas de resultados).
- **Mayor riesgo identificado:** Tasa de falsos positivos desconocida. Esto puede matar la confianza del usuario en el sistema más rápido que las alucinaciones.
- **Siguiente paso recomendado:** Experimento controlado con 20 outputs reales antes de diseñar la arquitectura final.
