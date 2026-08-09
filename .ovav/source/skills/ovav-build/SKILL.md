# OVAV Build Mode — Skill

> **Versión:** 1.0 | **Fecha:** 2026-07-30 | **Lead:** Thavren

---

## Nombre
`ovav-build`

## Descripción
Use cuando hay un PLAN.md aprobado para ejecutar. Ejecuta task por task con subagentes especializados, TWO-STAGE REVIEW (spec compliance + code quality), y TDD enforce.

**NO USA `actor.run`** — usa `workflow + agent(subagent_type="team-*")` con el squad real.

---

## HARD GATE
```
NO IMPLEMENTACIÓN SIN PLAN.APPROVED.md
NO SKIP DE TDD
NO CLAIM SIN VERIFICATION EVIDENCE
```

---

## El Proceso

### CAPA 1: Setup
1. Leer PLAN.md completo
2. Verificar que existe `PLAN.APPROVED.md` (CEO approval)
3. Si no existe → HARD STOP hasta approval
4. Extraer: file structure, phases, tasks, squad assignments
5. Crear worktree con `owc <feature-name>`

### CAPA 2: File Structure Mapping
Antes de ejecutar cualquier task, mapear TODOS los archivos:

```
Archivos a crear/modificar:
├── frontend/src/api/client.ts          # Axios instance + interceptors
├── frontend/src/stores/authStore.ts   # Zustand auth state
├── backend/internal/api/handlers/auth.go  # bcrypt + JWT handlers
... (un archivo por línea con responsabilidad de 1 línea)
```

**Regla:** Si no puedes listar los archivos, no puedes hacer el plan de implementación.

### CAPA 3: Task Execution (por phase, por task)

**Por cada task:**
1. Marcar `in_progress`
2. **TDD RED:** Escribir test que falla
3. **TDD GREEN:** Código mínimo para pasar
4. **TDD REFACTOR:** Limpiar sin romper tests
5. **SPEC REVIEW Phase 1:** ¿El código implementa lo que el spec dice?
6. **SPEC REVIEW Phase 2:** ¿Hay gaps o silent omissions?
7. **CODE QUALITY REVIEW:** ¿El código es mantenible, tipado, documentado?
8. Si todo pasa → `task done`
9. Si falla → volver a RED

### CAPA 4: Verification Final
Después de cada phase:
- `go test ./...` (backend)
- `pnpm test` (frontend)
- `pnpm build` (frontend)
- Si algo falla → HARD STOP hasta fix

### CAPA 5: Integration Testing
Después de todas las phases:
- E2E test suite completa
- Verificar todos los flujos end-to-end

---

## Squad Delegation (CORRECTO)

**BUG B WORKAROUND:** `actor` solo acepta `explore`/`general`. El workaround es:

```
1. Crear subagent con contexto inline del squad member
2. Usar workflow (no actor directo)
3. Pasar --task <TID> para binding
```

Ejemplo:
```bash
# NO USAR ESTO (Bug B):
actor run general "implement auth" --task T1

# USAR ESTO:
workflow + agent(subagent_type="general") con context injection del squad member
```

Para squads reales, delegar a:
- **Marco** (Systems Architect): architecture, DB modeling, API contracts
- **Andrés** (Implementador Senior): refactors, tests, código production-grade
- **Lucas** (Implementador Junior): fixtures, patches, tasks pequeños
- **Diana** (Security Auditor): permissions, secrets, git safety
- **Pablo** (Code Reviewer): validation, patterns, consistency
- **Clara** (QA Engineer): tests, regression detection, edge cases

---

## TDD Iron Law

```
RED: Write failing test
GREEN: Minimal code to pass
REFACTOR: Clean without breaking
```

**Nunca escribir código de producción sin test que falla primero.**
**Nunca guardar código sin verificar que todos los tests pasan.**

---

## Two-Stage Review

### Stage 1: Spec Compliance
- ¿Qué dice el spec que debe hacer?
- ¿El código hace exactamente eso?
- ¿Hay silent omissions?

### Stage 2: Code Quality
- ¿Está tipado correctamente?
- ¿Sigue los patterns del codebase?
- ¿Tiene tests?
- ¿Es mantenible?

---

## Verification Evidence

| Claim | Command | Required |
|---|---|---|
| Tests pass | `go test ./...` / `pnpm test` | Full output + exit 0 |
| Build succeeds | `go build` / `pnpm build` | Exit 0 |
| Linter clean | `go vet` / `pnpm lint` | 0 errors |
| E2E passes | `playwright test` | Full report |

**Claim without evidence = lying.**

---

## Archivo de Plan

OVAV genera plans en `plans/<project>/`:

```
plans/
└── <project>/
    ├── DESIGN.md           # Concepto + arquitectura + componentes
    ├── PLAN.md            # Implementation plan con tasks
    ├── PLAN.APPROVED.md   # CEO approval (requerido para BUILD)
    └── DELIVERED.md       # Report final post-BUILD
```

---

## Integración con Skills

- **Pre-BUILD:** `ovav-brainstorm` → genera DESIGN.md
- **Pre-BUILD:** CEO approval → PLAN.APPROVED.md
- **BUILD:** `ovav-build` → ejecuta PLAN.md
- **Post-BUILD:** `ovav-verify` → verification final
- **Merge:** `owd` → merge a main

---

## Metadata

- **Ubicación:** `.ovav/source/skills/ovav-build/SKILL.md`
- **Skill ID:** `ovav-build`
- **Trigger:** Plan approved + `owc <feature>` completado
- **Predecesor:** `ovav-brainstorm`
- **Sucesor:** `ovav-verify`
