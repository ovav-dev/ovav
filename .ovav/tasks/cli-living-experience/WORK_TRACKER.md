# CLI Living Experience — Work Tracker

**Rama:** `task/cli-living-experience-rc9`
**Inicio:** 2026-05-27
**Objetivo:** RC9.0 — cockpit vivo, interactivo, premium

## Estado

| # | Segmento | Issue | Estado | Commit |
|---|---|---|---|---|
| — | Refactor CLI (pre-segmentos) | — | ✅ completado | `792bed2` |
| 1 | Purga opciones internas | [#15](https://github.com/Alexander-Salvador/OVAV/issues/15) | ✅ completado | `f23833b` |
| 2 | Launch install real | [#16](https://github.com/Alexander-Salvador/OVAV/issues/16) | ✅ completado | `5c06560` |
| 3 | Tailor toggles vivos | [#17](https://github.com/Alexander-Salvador/OVAV/issues/17) | ✅ completado | working tree |
| 4 | Recovery + Update real | [#18](https://github.com/Alexander-Salvador/OVAV/issues/18) | ✅ completado | working tree |
| 5 | Capa visual premium | [#19](https://github.com/Alexander-Salvador/OVAV/issues/19) | ✅ completado | `6b8f63c` |
| 6 | Pantallas resultado | [#20](https://github.com/Alexander-Salvador/OVAV/issues/20) | ✅ completado | working tree |
| 7 | Guía contextual | [#21](https://github.com/Alexander-Salvador/OVAV/issues/21) | ✅ completado | `6b8f63c` |
| 8 | Pulido + smoke + cierre | [#22](https://github.com/Alexander-Salvador/OVAV/issues/22) | ✅ smoke source-local completado | working tree |

## Refactor completado (2026-05-27)

- `bin/ovav`: eliminadas 13 capas de `main()` encadenadas (de 1708 → 1353 líneas)
- `tools/cli/router.py`: NUEVO — command router centralizado (319 líneas, 13 rutas)
- `tools/cli/_legacy/`: 10 archivos obsoletos archivados (experience_engine, smokes huérfanos, RC5 tests)
- Validación: `python3 tools/ovav_runtime.py validate` → OK

## Siguiente acción

SEG 6 quedó implementado y validado. Cierre source-local RC9 ejecutado con smokes CLI + runtime/OpenCode validators en PASS. No declarar production/global-ready; final launch/tag externo sigue bajo autoridad de Final Launch Verification.

## Decisiones

- **2026-05-27 (refactor):** Router centralizado reemplaza 13 capas de main. Nuevos comandos se agregan en `tools/cli/router.py`, no en `bin/ovav`.
- **2026-05-27 (SEG 3):** Tailor ahora tiene toggles vivos en sesión: Space alterna herramientas/roles/plan, Enter abre preview, confirmación aplica estado en memoria y se preserva al navegar. La lógica quedó separada en `tools/cli/ovav_tailor_composer.py`. Revisión de feedback: plan obligatorio primero, opciones atenuadas por plan, planes `Núcleo/Studio/Command`, footer dinámico e instalación con confirmación.
- **2026-05-27 (SEG 4):** Update ahora hace preview real desde Git/upstream local y plan `ovav update --plan`; aplica solo con cambios disponibles, working tree limpio, backup pre-update, merge `ff-only` y verificación. Recovery lista backups reales con `rollback --list --json`, muestra preview y restaura solo con doble Enter + flags de consentimiento.
- **2026-05-27 (RC9 polish):** Se limpian placeholders del cockpit, se sube `VERSION` a `v1.0.0-rc9` y se registra evidencia source-local en `.ovav/artifacts/RC9_0/CLI_POLISH_SMOKE_EVIDENCE.md`; no implica cierre final ni claim global.
- **2026-05-27 (SEG 6):** Pantallas de resultado estructuradas implementadas con contrato `ovav.cockpit_result.v1`; install, Tailor, update y recovery terminan en `render_result()` con cambios, verificación y próximo paso. Evidencia: `.ovav/artifacts/CLI_LIVING_EXPERIENCE/SEG_06_RESULT_SCREENS_EVIDENCE.md`.
- **2026-05-27 (anterior):** Base unificada. Solo cockpit curses. Experience engine archivado en `_legacy/`.

## Archivos del plan

- `.ovav/tasks/cli-living-experience/SEGMENT_PLAN.md`
- `.ovav/tasks/cli-living-experience/ISSUE_01` a `08` — specs detalladas (targets corregidos a `ovav_first_run_cockpit.py`)
- `.ovav/tasks/cli-living-experience/WORK_TRACKER.md` — este archivo
- `tools/cli/router.py` — command router (nuevo)
