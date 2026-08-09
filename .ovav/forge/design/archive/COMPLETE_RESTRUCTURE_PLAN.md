# OVAV Complete Restructure Plan v1.0

> **Autoridad:** Thavren — Platform Engineering Lead
> **Fecha diseño:** 2026-06-03
> **Estado:** Diseño aprobado, ejecución pendiente
> **Branch:** task/implementaciones

---

## Objetivo

Reorganizar TODO OVAV desde la raíz `/home/braka/Systems/OVAV/` con arquitectura multi-target (OpenCode, Claude Code, VSCode, Pi). Nada fuera de lugar, segmentación completa, blindaje contra desorden futuro.

## Principios

1. `.ovav/` = TODO desarrollo y gobernanza interna OVAV
2. `.opencode/` = SOLO proyección generada, nunca fuente
3. `tools/` = motor de ejecución (sin cambios estructurales mayores)
4. Raíz = mínimo indispensable (opencode.json, AGENTS.md, VERSION, README.md, .gitignore)
5. Multi-target desde el diseño: adapters por CLI, no acoplamiento

---

## Estructura objetivo

```
OVAV/
├── .ovav/                              ← CEREBRO OVAV (todo interno)
│   │
│   ├── source/                         ← DESARROLLO universal
│   │   ├── agents/                     ← areas/, leads/, teams/
│   │   ├── skills/                     ← 11 skills fuente
│   │   ├── plugins/                    ← plugins por target
│   │   │   ├── opencode/tui/status/monitor/
│   │   │   ├── claude-code/
│   │   │   ├── vscode/
│   │   │   └── pi/
│   │   ├── configs/                    ← wezterm, nvim, shell
│   │   └── visual/                     ← theme, branding, monitoring
│   │
│   ├── forge/                          ← RELEASE ENGINE (9-gate multi-target)
│   │   ├── pipeline.py                 ← orchestrator principal
│   │   ├── targets/                    ← manifiestos por CLI
│   │   │   ├── opencode.yaml
│   │   │   ├── claude-code.yaml
│   │   │   ├── vscode.yaml
│   │   │   └── pi.yaml
│   │   ├── adapters/                   ← lógica de conversión por CLI
│   │   │   ├── opencode/{agents,skills,plugins,visual}.py
│   │   │   ├── claude-code/
│   │   │   ├── vscode/
│   │   │   └── pi/
│   │   ├── releases/                   ← versionado por target
│   │   │   ├── opencode/
│   │   │   ├── claude-code/
│   │   │   ├── vscode/
│   │   │   └── pi/
│   │   └── design/                     ← este documento y specs
│   │
│   ├── registry/                       ← consolidar registry/ raíz aquí
│   ├── runtime/                        ← sesiones, locks, logs, evidence
│   ├── governance/                     ← laws, policy, security, topology, evaluation
│   ├── memory/                         ← ledger, context, handoffs
│   ├── health/                         ← reportes
│   ├── artifacts/                      ← histórico S0-S151, BUILD, RC, M, L
│   ├── cache/                          ← checkpoints, research
│   ├── quarantine/                     ← archivos denegados
│   └── lockdown/                       ← bloqueos activos
│
├── .opencode/                          ← PROYECCIÓN (solo archivos generados)
│   ├── agents/                         ← desde .ovav/source/agents/
│   ├── skills/                         ← desde .ovav/source/skills/
│   ├── commands/                       ← generado
│   ├── plugins/                        ← desde .ovav/source/plugins/opencode/
│   └── themes/                         ← desde .ovav/source/visual/
│
├── tools/                              ← motor de ejecución
├── bin/                                ← CLI (ovav, ovav-shell, ovav-logo)
├── docs/                               ← documentación
├── tests/                              ← tests
├── schemas/                            ← JSON schemas
├── assets/                             ← PNGs, íconos (consolidar sueltos raíz)
│
├── opencode.json                       ← requerido por OpenCode en raíz
├── AGENTS.md                           ← requerido por OpenCode en raíz
├── VERSION                             ← versión
├── README.md                           ← público
└── .gitignore
```

---

## Plan de migración — 9 fases

### F0 — Fix model watchdog + agregar trigger
- **Problema:** `model_watchdog.py` lee `config["agent"]["ovav-platform-engineering"]["model"]` que no existe. El modelo real está en `config["model"]` raíz.
- **Problema:** `auto_triggers.yaml` no tiene trigger de model exhaustion. El watchdog nunca se gatilla automático.
- **Fix:** Re-escribir watchdog para leer/escribir `config["model"]`. Agregar trigger `on_model_error`.
- **Archivos:** `tools/agent_runtime/model_watchdog.py`, `.ovav/registry/auto_triggers.yaml`
- **Riesgo:** Bajo

### F1 — Crear `.ovav/source/` + mover agentes, skills
- Mover `.ovav/source/agents/` → `.ovav/source/agents/`
- Mover skills fuente a `.ovav/source/skills/`
- Actualizar referencias en `project_opencode.py`, `discover_skills.py`, etc.
- **Archivos:** ~30
- **Riesgo:** Medio

### F2 — Crear `.ovav/forge/` + mover pipeline, releases
- Mover `tools/visual/release_pipeline.py` → `.ovav/forge/pipeline.py`
- Mover `.ovav/forge/releases/opencode/` → `.ovav/forge/releases/opencode/`
- Crear `targets/` y `adapters/` con estructura inicial
- Extraer lógica de `project_opencode_visual.py` → `adapters/opencode/visual.py`
- **Archivos:** ~10
- **Riesgo:** Medio

### F3 — Mover `registry/` raíz → `.ovav/registry/`
- Mover 30+ YAMLs de `registry/` → `.ovav/registry/`
- Actualizar TODAS las referencias en `tools/`, `bin/`, `.ovav/`
- **Archivos:** 30+ yamls + referencias en 35+ Python files
- **Riesgo:** Alto

### F4 — Consolidar `runtime/` + `config/` + sueltos raíz
- `runtime/` raíz → `.ovav/runtime/`
- `config/` raíz → `.ovav/source/configs/`
- `.ovav/lab/statusline-inspect/` → `.ovav/lab/`
- `OVAV_LIVING_INTELLIGENCE_*.md` → `docs/lab/`
- **Archivos:** ~15
- **Riesgo:** Medio

### F5 — Mover fuentes plugins a `.ovav/source/plugins/`
- `.opencode/packages/ovav-tui/` → `.ovav/source/plugins/opencode/tui/`
- `.opencode/packages/statusline-ref/` → `.ovav/lab/statusline-ref/`
- `.ovav/source/plugins/opencode/ovav-status.js` → `.ovav/source/plugins/opencode/status/`
- `.ovav/source/plugins/opencode/ovav-monitor.js` → `.ovav/source/plugins/opencode/monitor/`
- **Archivos:** ~15
- **Riesgo:** Medio

### F6 — Actualizar paths hardcodeados
- Buscar y reemplazar TODOS los paths rotos por la migración
- Script de búsqueda: `rg "\.opencode/|registry/|\.ovav/visual/releases|\.ovav/source/agents/" tools/ bin/ .ovav/`
- Validar cada archivo individualmente
- **Archivos:** 35+ Python files
- **Riesgo:** Alto

### F7 — Re-proyectar y validar
- Ejecutar `project_opencode.py` → regenerar `.opencode/agents/`
- Ejecutar `project_opencode_visual.py` → regenerar `.opencode/themes/`, `plugins/`, `tui.json`
- Ejecutar `discover_skills.py` → regenerar skill registry
- Ejecutar `release_pipeline.py --dry-run` → validar pipeline
- **Riesgo:** Alto

### F8 — Limpiar raíz + smoke test completo
- Mover PNGs sueltos → `assets/`
- Verificar `.gitignore` actualizado
- `ovav validate` + `ovav doctor`
- `python3 tools/validators/validate_all.py`
- Verificar que `opencode.json` sigue siendo válido
- **Riesgo:** Bajo

---

## Issues previos detectados

### Model watchdog roto
- `get_current_model()` retorna "unknown" porque busca en path equivocado
- Sin trigger automático en `auto_triggers.yaml`
- Fix pendiente en F0

### Team permissions
- Thavren es el gatekeeper. Equipo pide autorización → Thavren aprueba.
- No permisos blanket. Cada acceso requiere aprobación explícita del lead.
- Policy definida en `.ovav/policy/permission_authority.json`

### poison_guard_hardening timeout
- `python3 tools/ovav_runtime.py validate` timed out a 45s durante push gate
- Puede necesitar timeout mayor o investigación de causa raíz

### check_delegation_runtime fixtures
- Fixtures inválidos pasan validate_all inesperadamente
- Requiere revisión de `check_invalid_fixtures.py`

---

## Criterio aprendido

**Nunca cerrar sesión sin documentar decisiones de arquitectura.** Si diseñamos algo que requiere múltiples sesiones para ejecutar, el diseño debe persistir en `.ovav/forge/design/` o en el ledger. La conversación no es almacenamiento confiable.
