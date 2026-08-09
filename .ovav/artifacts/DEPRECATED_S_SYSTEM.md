# ⚠️ S* Artifact System — DEPRECATED

**Los directorios `S0` a `S151` pertenecen a un sistema de numeración de segmentos ANTIGUO.**

Estos artefactos son evidencia histórica, NO autoridad actual.

> **⚠️ ESTE ARCHIVO ESTÁ DEPRECADO.** La autoridad real del sistema reside en `.ovav/plan/caps.yaml` + `git HEAD`.
> Cualquier afirmación de autoridad en este archivo es histórica y NO vinculante.

No deben usarse para:
- Determinar el BUILD o segmento activo
- Definir next_work o estado del sistema
- Resolver rutas de implementación

## Documento de referencia consolidado

Toda la historia del sistema S*, sus BUILDs y su migración al sistema actual está documentada en:
→ **`.ovav/docs/HISTORICAL_S_SYSTEM_REFERENCE.md`**

## Sistema actual

- **BUILD tracking:** B18-B23 (B23 = Tool Readiness Matrix, cerrado). Sistema de BUILD en pausa.
- **Ruta estratégica:** `.ovav/plan/caps.yaml` — única fuente de verdad del plan (datos canónicos).
- **Autoridad canónica:** `caps.yaml` (plan data) + `git HEAD` (temporal data). Cadena: chronos (git) → git HEAD → caps.yaml.
- **Handoff:** `.ovav/context/CURRENT_HANDOFF.md` (generado, sin autoridad)
- **Lab/ideas:** `.ovav/lab/ideas.yaml`

## Si encuentras una referencia S*

Si un archivo activo (fuera de `.ovav/artifacts/S*/`) referencia un segmento S*:
1. El archivo está desactualizado.
2. La referencia debe migrarse al sistema B* o eliminarse.
3. Si el contenido es valioso, consolídalo en `HISTORICAL_S_SYSTEM_REFERENCE.md`.

## Archivos que SÍ son autoridad actual

- `BUILD23_TOOL_READINESS_BOUNDARY/` — B23, baseline activo
- `FINAL_LAUNCH_VERIFICATION/` — verificación de launch activa
- `L5/`, `L6/`, `L7/` — capas de inteligencia activas
- `M1/`, `M2/` — puentes de proyección global

## Corte de deprecación en runtime

El runtime (`runtime_next_work.py`, `todo_progress_runtime.py`) ahora trunca segmentos S* en **S30**. S31+ son excluidos de reportes de progreso/todo para evitar confusión.

**Última actualización:** 2026-06-10 — Limpieza post-S*, corte S30 implementado en runtime.
