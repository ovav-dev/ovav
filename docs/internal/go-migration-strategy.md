# OVAV Go Migration Strategy

> **Regla**: Solo migrar a Go lo que esté 100% limpio, seguro y bien documentado.
> Python sigue siendo canónico para desarrollo activo y herramientas internas.

## Estado actual

| Runtime | Archivos | Estado |
|---------|---------|--------|
| Python (`tools/`) | ~500 | Canónico, recién reorganizado |
| Go (`go-runtime/`) | 23 | CLI pública + cPanel, limpio |

## Criterios de migración

Un módulo Python está listo para migrar a Go cuando:
1. ✅ API estable (sin cambios en 2+ semanas)
2. ✅ 100% test coverage
3. ✅ Documentación completa (docstrings + ARCHITECTURE.md)
4. ✅ Sin dependencias externas (stdlib only como el runtime Go actual)
5. ✅ Usado en CLI pública o cPanel (no solo interno)

## Módulos candidatos (orden de prioridad)

### Fase 1 — Ya en Go ✅
- `cmd/ovav/` — CLI pública: status, profile, config, tools, update, version
- `cmd/cpanel/` — Servidor HTTP control panel
- `internal/doctor/` — Diagnóstico del sistema
- `internal/tools/` — Catálogo de herramientas (43 registradas)

### Fase 2 — Próximos candidatos
- `tools/git/branch/types.py` → `internal/git/branch/` (constantes, sin lógica compleja)
- `tools/git/branch/check_protected.py` → `internal/git/branch/` (waiver check)
- `tools/git/push/push_gate.py` → `internal/git/push/` (pre-push validation)

### Fase 3 — Mediano plazo
- `tools/git/worktree/position.py` → `cmd/ovav/` comando `worktree` (cuando API sea estable)
- `tools/git/stage/intelligence.py` → `internal/git/stage/`
- `tools/economy/` → `internal/economy/`

### Fase 4 — Largo plazo
- `tools/validators/` → Go nativos (sin exec Python)
- `tools/governor/` → Go nativos
- `tools/security/` → Go nativos

## No migrar (se queda en Python)

- `tools/cli/` TUI/cockpit — Python es mejor para TUI
- `tools/harnesses/` — Harnesses de desarrollo, no producción
- `tools/memory/` — Sistema interno, no expuesto [DEPRECATED — memory system removed 2026-06-11]
- `tools/research/` — Tools de Eidren, no producción

## Métricas de progreso

- **Objetivo**: 80% del runtime productivo en Go
- **Timeline**: Sin fecha fija. Migrar solo cuando el módulo cumpla los 5 criterios.
