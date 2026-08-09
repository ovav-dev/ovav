# OVAV — Decisiones Arquitectónicas y Benchmarks (Bloques H-I)

> Análisis: 2026-06-10 · v1.0.0 → v2.0.0+

---

## 🔷 BLOQUE H: Decisiones Arquitectónicas

### H1: Sidecar OVAV — CLI agent propio (I-029)

**¿Qué es sidecar?**
Un CLI agent OVAV nativo que corre al lado de OpenCode (o lo reemplaza).
OpenCode es el runtime actual. Sidecar sería nuestra propia implementación.

**Análisis costo/beneficio:**
| Factor | OpenCode (actual) | Sidecar propio |
|--------|-------------------|----------------|
| Control | Medio (dependemos de su API) | Total |
| Mantenimiento | Cero (lo mantiene Anthropic) | Alto (nosotros) |
| Features OVAV-first | Adaptadas | Nativas |
| Riesgo | Cambios de API, deprecaciones | Bugs propios |
| Tiempo estimado | 0 (ya funciona) | 6-12 meses |

**Recomendación:** POSPONER hasta v3.0.0. OpenCode funciona bien como runtime. El esfuerzo de construir un CLI agent propio no se justifica mientras OpenCode cubra nuestras necesidades. Si OpenCode cambia su API de forma incompatible o depreca features que necesitamos, reevaluar.

**Señales para activar sidecar:**
- OpenCode depreca plugin API que usamos
- OpenCode limita el control que necesitamos sobre modelos
- Necesitamos features que OpenCode no soporta (multi-model nativo, routing propio)

---

### H2: WSL2 vs Windows 11 (I-032)

**Arquitectura actual:**
- OVAV en WSL2: `/home/braka/Systems/OVAV/`
- Aislado de docs globales de Windows
- Worktrees para desarrollo paralelo

**Análisis:**
| Opción | Ventaja | Desventaja |
|--------|---------|------------|
| **WSL2 (actual)** | Aislamiento, Linux nativo, git rápido | Sin acceso a apps Windows |
| **Windows 11 nativo** | Acceso a todo el filesystem, apps Windows | Menos isolation, WSL2 overhead |
| **Híbrido** | Lo mejor de ambos | Complejidad de sincronización |

**Recomendación:** MANTENER WSL2 para OVAV sistema. Es la arquitectura correcta:
- OVAV como gobernador necesita Linux (bash, Python nativo, git rápido)
- El aislamiento actual es CORRECTO — OVAV no debe mezclarse con docs personales
- Si se necesita acceso a Windows, usar `/mnt/c/` desde WSL2

---

### H3: OVAV como Collective Intelligence (I-006)

**Visión:** OVAV no es solo un governor — es un sistema de inteligencia colectiva con múltiples agentes especializados colaborando autónomamente.

**Estado actual:** 70 agentes definidos, 6 áreas de servicio, SNV activo. La base existe.

**Prerrequisitos para CI:**
1. Gateway OVAV (I-021) — ✅ completado
2. Memoria compartida entre agentes (I-017) — ✅ integrada
3. Routing inteligente (I-023) — ✅ completado
4. KPIs por LEAD (I-031) — ✅ definidos

**Próximo paso:** Dejar que los agentes colaboren en tareas reales. La CI no se construye — emerge del uso.

---

## ⬜ BLOQUE I: Benchmarks e Investigación

### I1: OVAV roles vs Claude Code (I-028)

**Benchmark propuesto:**
| Dimensión | OVAV (Thavren + DeepSeek V4 Pro) | Claude Code |
|-----------|----------------------------------|-------------|
| Arquitectura de sistemas | Especializado (Thavren) | Generalista |
| Seguridad | 33 tools de defensa | Generalista |
| Memoria | SNV + Memory Governor | Por sesión |
| Costo | $0.55/M input (DeepSeek Pro) | $3/M input (Sonnet) |

**Lead:** Eidren (Evidence & Decision Intelligence)
**Estado:** 🔬 exploration — requiere sesión dedicada de benchmarking

---

### I2: DeepSeek compressed sparse attention (I-026)

**Qué es:** DeepSeek V4 usa compressed sparse attention para reducir el costo de atención en contextos largos, manteniendo calidad.

**Cómo aplicarlo en OVAV:**
- No podemos modificar el modelo, pero podemos optimizar qué le enviamos
- Prompt caching (I-015) — ✅ strategy definida
- Context cut cuando cambia la tarea (I-017) — ✅ integrado
- Token budget enforcement (L6) — ✅ activo

**Conclusión:** Ya aplicamos los principios. No necesitamos replicar CSA — necesitamos usar DeepSeek como proveedor primario.

---

### I3: Clean uninstall (I-003)

**Requisitos:**
1. Remover `.opencode/` generado por OVAV
2. Remover `.ovav/` del proyecto
3. Remover configuraciones de shell
4. Opcional: remover credenciales del vault

**Estado:** 🔬 exploration — baja prioridad. Solo necesario si se migra a sidecar (H1) o se abandona OVAV.

**Implementación:** ~50 líneas de bash. No requiere herramienta compleja.

---

_Análisis: 2026-06-10 · Thavren — Platform Engineering & DX Lead_
