# PLAN S12 — Lead Accountability Protocol

> **Fecha**: 2026-06-06
> **Autor**: Thavren — Platform Engineering Lead
> **Investigación**: Eidren — Research Intelligence Lead (4 artefactos de evidencia)
> **Estado**: Plan aprobado, pendiente implementación

---

## Evidencia fundacional

Antes de diseñar, los hechos duros:

| Dato | Fuente |
|---|---|
| ChatGPT alucina en ~19.5% de respuestas | HaluEval (EMNLP 2023) |
| GPT-3 solo 58% truthful vs 94% humanos | TruthfulQA (ACL 2022) |
| Modelos más grandes fueron **menos** truthful | TruthfulQA |
| SelfCheckGPT (muestreo múltiple) detecta alucinaciones sin fuente externa | EMNLP 2023 |
| RARR corrige claims con búsqueda externa (5% mejora factual) | ACL 2023 |
| SAFE (Google DeepMind) verifica claims descompuestos contra Google Search | 2024 |
| **No existen datos de tasa de falsos positivos en ningún sistema público** | — |

**Conclusión**: La tecnología existe. Los patrones están validados. Pero no hay producto. OVAV debe construir su propio Trust Gate. Y los falsos positivos son más peligrosos que los falsos negativos — marcar como falso algo verdadero destruye confianza.

---

## Arquitectura separada: OVAV ↔ LEAD

```
┌─────────────────────────────────────────────────────────┐
│                    OVAV (Gobernador)                     │
│                                                         │
│  Interrupt Engine    Trust Gate     Accountability Log  │
│  (monitorea)         (decide)       (registra)          │
│                                                         │
│  "Thavren, detente.                   "LEAD entregó     │
│   Trust 0.3 en claim X"               sin verificar     │
│                                        3 veces hoy"     │
└──────────────┬──────────────────────────────────────────┘
               │ monitorea / interrumpe / audita
               ▼
┌─────────────────────────────────────────────────────────┐
│                  LEAD (Thavren / Eidren)                 │
│                                                         │
│  Verification Loop:                                     │
│   1. Genera output (vía modelo)                         │
│   2. Pasa por MIL (automático)                          │
│   3. Si trust < 0.85 → no entrega                       │
│   4. Investiga claims fallidos                          │
│      - Thavren: claims técnicos, paths, versiones       │
│      - Eidren: claims externos, fuentes, benchmarks     │
│   5. Corrige output                                     │
│   6. Re-verifica con MIL                                │
│   7. Trust ≥ 0.85 → entrega                             │
└─────────────────────────────────────────────────────────┘
```

---

## PARTE 1 — Lo que necesita OVAV (Sistema)

### S12-A: Interrupt Engine

OVAV monitorea en tiempo real. Si un LEAD va a entregar output no verificado, OVAV interrumpe.

**Mecanismo**: Hook en el flujo de salida del LEAD. Antes de que el output llegue al usuario, pasa por MIL. Si trust < 0.85, OVAV emite una interrupción visible.

**Archivos**:
- `tools/governor/interrupt_engine.py` — motor de interrupción
- Integración en `session_feed.py` — registrar interrupciones

**Comportamiento**:
```
OVAV: "⚠️ INTERRUPCIÓN — Trust Gate: 0.3/1.0"
OVAV: "Claim contradicho: 'OVAV tiene 10 millones de usuarios'"
OVAV: "Acción requerida: investigar y corregir antes de entregar"
```

### S12-B: Trust Gate

OVAV decide si un output es apto para entrega. No es binario — es un score calibrado con umbrales.

**Thresholds**:
| Trust | Acción |
|---|---|
| ≥ 0.90 | ✅ Entrega automática |
| 0.85 — 0.89 | ⚠️ Entrega con disclaimer |
| 0.50 — 0.84 | 🛑 Interrupción — LEAD debe corregir |
| < 0.50 | 🔴 Bloqueo total — LEAD debe regenerar |

**Archivos**:
- `tools/governor/trust_gate.py` — gate de confianza

### S12-C: Accountability Log

OVAV registra cada incidencia. Si un LEAD entrega sin verificar repetidamente, escala al CEO.

**Métricas**:
- Entregas sin verificar (trust < 0.85 entregado igual)
- Interrupciones generadas
- Correcciones aplicadas
- Trust score promedio por LEAD
- Tiempo promedio de corrección

**Archivos**:
- `.ovav/logs/accountability.jsonl` — log de accountability

---

## PARTE 2 — Lo que necesitan los LEADs

### S12-D: Verification Loop (Thavren)

Cuando OVAV interrumpe, Thavren NO puede entregar. Debe ejecutar el ciclo:

1. **Identificar** claims fallidos (MIL ya los marcó)
2. **Investigar**: 
   - Claims técnicos → verificar en filesystem, git, código OVAV
   - Claims factuales → verificar en Knowledge Compiler
   - Claims externos → delegar a Eidren (Research Mesh)
3. **Corregir**: reescribir claims con datos verificados
4. **Re-verificar**: pasar output corregido por MIL
5. **Entregar**: solo si trust ≥ 0.85

**Archivos**:
- `tools/governor/verification_loop.py` — flujo de corrección
- Integración con `delegation_protocol.py` — si necesita research, delega a Eidren

### S12-E: Research Escalation (Eidren)

Cuando Thavren encuentra un claim que requiere investigación externa, delega a Eidren.

**Flujo**:
```
Thavren: "Claim X requiere verificación externa"
  → delegation_protocol.dispatch_task(lead="eidren", ...)
    → Eidren investiga con Research Mesh
      → Eidren reporta hallazgo con fuentes
        → Thavren corrige claim
```

**Archivos**:
- Ya existe: `delegation_protocol.py`
- Nuevo: integración del ciclo Thavren→Eidren→Thavren

---

## PARTE 3 — Experimento controlado (ANTES de implementar)

**Riesgo crítico**: Sin datos de falsos positivos, cualquier threshold es especulación.

**Protocolo**:
1. Seleccionar 20 outputs reales de LEAD (mezcla de correctos + con alucinaciones)
2. Ejecutar MIL sobre cada uno → trust scores
3. Verificar manualmente cada claim marcado como falso
4. Medir:
   - Verdaderos positivos (MIL detectó alucinación real)
   - Falsos positivos (MIL marcó como falso algo verdadero)
   - Verdaderos negativos (MIL dejó pasar output correcto)
   - Falsos negativos (MIL dejó pasar alucinación)
5. Calibrar threshold basado en datos reales

**Duración estimada**: 30 minutos (ejecución + análisis)
**Archivos**: `.ovav/research/lead_accountability_protocol/experiment_results.yaml`

---

## Orden de implementación

| Fase | Qué | Dependencia | Tiempo |
|---|---|---|---|
| **E0** | Experimento controlado (20 outputs) | Ninguna | 30 min |
| **S12-A** | Interrupt Engine | E0 (threshold calibrado) | 20 min |
| **S12-B** | Trust Gate | E0 | 15 min |
| **S12-D** | Verification Loop (Thavren) | S12-A + S12-B | 25 min |
| **S12-E** | Research Escalation (Eidren) | S12-D | 15 min |
| **S12-C** | Accountability Log | S12-A | 15 min |

**Total**: ~2 horas con experimento incluido

---

## Lo que NO haremos (hype sin evidencia)

- ❌ Fact-checker universal automático (imposible sin internet viva)
- ❌ Zero-latency verification (requiere muestreo múltiple)
- ❌ Un solo método (múltiples capas necesarias)
- ❌ Copiar frameworks pesados (OVAV nativo)
- ❌ Ignorar falsos positivos (más dañinos que falsos negativos)
