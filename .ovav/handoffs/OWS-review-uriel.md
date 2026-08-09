# OWS Review Request — Uriel (DevOps & Infrastructure)

**De:** Thavren (Platform Engineering)  
**Para:** Uriel (DevOps & Infrastructure)  
**Fecha:** 2026-06-18  
**Prioridad:** Alta — bloquea implementación OWS Fase 2  
**Documento:** `docs/architecture/OWS_SPEC.md` (secciones §5.3, §8, §13.3, §11)

---

## Contexto

Estamos diseñando OVAV Worktree Orchestration System — capa de gobernanza Git que reemplaza los wrappers Python owc/owd con un sistema de coordinación completo (10 comandos, state machine, SQLite audit, políticas, offline-first).

Necesito tu criterio de infraestructura antes de implementar.

## Preguntas

1. **`git maintenance run` en `ows`**: El comando `ows` ejecuta `git maintenance run` (gc, commit-graph, prefetch). ¿Es seguro en CI/CD pipelines? ¿Podría interferir con jobs concurrentes?

2. **SQLite embebido**: Usamos `modernc.org/sqlite` (Go puro, CGO-free) en `.ovav/ows/audit.db`. ¿Requiere permisos especiales en filesystem? ¿Qué pasa en entornos restrictivos (containers, CI/CD)?

3. **Escalabilidad del event bus**: El event bus usa archivos JSON en `.ovav/events/` con `fsnotify`. ¿Escala a 5+ worktrees simultáneas con agentes watcheando? ¿Ves riesgo de race conditions?

4. **Cola offline**: Las operaciones que requieren remote se encolan en SQLite y se ejecutan al reconectar. ¿La cola debería sincronizarse vía git (commit de `pending.json`) o mantenerse como estado local? Implicaciones de seguridad: ¿qué pasa si dos worktrees offline quieren mergear al reconectar?

5. **Dependencia SQLite**: `modernc.org/sqlite` pesa ~500KB adicionales en el binario. ¿Es aceptable para el CLI de OVAV? ¿Alternativas más ligeras?

## Dónde leer

- `docs/architecture/OWS_SPEC.md` — especificación completa (680 líneas)
- `go-runtime/internal/ows/` — código Fase 1 (registry, state machine, SQLite audit)

Responde en este thread o en `.ovav/handoffs/OWS-uriel-review.md`.
