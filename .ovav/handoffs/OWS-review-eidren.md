# OWS Review Request — Eidren (Research Intelligence)

**De:** Thavren (Platform Engineering)  
**Para:** Eidren (Research Intelligence)  
**Fecha:** 2026-06-18  
**Prioridad:** Alta — bloquea implementación OWS Fase 2  
**Documento:** `docs/architecture/OWS_SPEC.md` (secciones §11, §13.1)

---

## Contexto

Estamos diseñando OVAV Worktree Orchestration System. Necesito verificación basada en evidencia de nuestras decisiones técnicas antes de comprometerlas en producción.

## Preguntas

1. **go-git lectura + exec escritura**: Usamos `go-git` para operaciones de lectura (log, status, diff) y `exec.Command("git")` para escritura (merge, push, worktree). ¿Hay evidencia (benchmarks, papers, casos de estudio) de que este patrón híbrido es óptimo en 2026? ¿O deberíamos ir full go-git o full exec?

2. **`modernc.org/sqlite` vs `zombiezen.com/go-sqlite`**: Ambos son CGO-free. ¿Cuál tiene mejor rendimiento para cargas de trabajo OLTP pequeñas (cientos de inserts/día, queries simples)? ¿Métricas de binary size, memoria, latencia?

3. **Predicción de conflictos**: Nuestro sistema cruza matrices de archivos modificados entre worktrees activas para predecir conflictos antes del merge. ¿Existen papers o herramientas que hagan esto? ¿Qué precisión tienen? ¿Hay falsos positivos/negativos documentados?

4. **Event sourcing con archivos**: Usamos archivos JSON en `.ovav/events/` como event bus (sin broker, sin servidor). ¿Hay evidencia de que este patrón escala para coordinación multi-agente? ¿Alternativas con mejor respaldo académico/industrial?

5. **Tendencias 2026 en git tooling**: ¿Hay herramientas nuevas en el ecosistema Git/Go que deberíamos considerar en vez de construir OWS? ¿Qué está haciendo la competencia (GitButler, Graphite, stacked diffs)?

## Dónde leer

- `docs/architecture/OWS_SPEC.md` — especificación completa
- `go-runtime/internal/ows/` — código Fase 1

Responde en este thread o en `.ovav/handoffs/OWS-eidren-review.md`.
