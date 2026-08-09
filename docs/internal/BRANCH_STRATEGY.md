# OVAV Branch Strategy

**Canonical document — Single source of truth for branch governance.**
**Aligned to: Conventional Commits 1.0.0 + GitHub Rulesets 2026 + Trunk-Based Development.**

---

## 1. Taxonomía de ramas

### 1.1 Ramas protegidas (PROTECTED)

| Nombre | Tier | Protección |
|---|---|---|
| `main`, `master` | ULTRA | Waiver + challenge + suite completa. Force bloqueado permanentemente. |
| `develop`, `development` | HIGH | Waiver con scope + merge-base check. |
| `prod`, `production`, `staging` | HIGH | Waiver requerido para writes. |
| `release/*` | HIGH (prefix) | Rama de estabilización pre-prod. Requiere waiver. |
| `hotfix/*` | HIGH (prefix) | Corrección urgente. Requiere waiver. Nace de main. |
| `prod/*`, `production/*` | HIGH (prefix) | Ramas de producción. Requiere waiver. |

### 1.2 Ramas de trabajo (WORK)

Taxonomía basada en Conventional Commits 1.0.0 con extensiones OVAV:

| Prefijo | Tipo CC | Propósito | Lifetime | Auto-detect |
|---|---|---|---|---|
| `feature/` | `feat:` | Nueva funcionalidad | 7d warn / 14d archive | ✅ |
| `fix/` | `fix:` | Corrección de bug | 7d warn / 14d archive | ✅ |
| `docs/` | `docs:` | Documentación | 14d warn / 30d archive | ✅ |
| `chore/` | `chore:` | Mantenimiento | 7d warn / 14d archive | ✅ |
| `refactor/` | `refactor:` | Refactorización | 7d warn / 14d archive | ✅ |
| `perf/` | `perf:` | Optimización de rendimiento | 7d warn / 14d archive | ✅ |
| `test/` | `test:` | Tests | 7d warn / 14d archive | ✅ |
| `ci/` | `ci:` | CI/CD | 7d warn / 14d archive | ✅ |
| `build/` | `build:` | Sistema de build | 7d warn / 14d archive | ✅ |
| `research/` | *(OVAV)* | Investigación, benchmarks | 30d warn / 60d archive | ✅ |
| `experiment/` | *(OVAV)* | Pruebas, prototipos, spikes | 14d warn / 21d delete | ✅ |
| `task/` | *(OVAV)* | Tarea genérica (legacy) | 7d warn / 14d archive | ✅ |

---

## 2. Flujo canónico

```
main ← develop ← feature/* ← task/*
```

### 2.1 Ciclo de vida completo

1. **Creación**: `ovav worktree create <name>` desde `develop`
2. **Validación**: `validate_branch_name()` + existencia previa + prefijo reconocido
3. **Trabajo**: Commits en la rama. Pre-flight checks opcionales.
4. **Merge**: `git merge <branch> --no-ff --no-edit` a `develop` (siempre merge commit)
5. **Cleanup**: `git branch -d <branch>` + `git push origin --delete <branch>`
6. **GC**: Ramas mergeadas se eliminan automáticamente en `on_session_start`

### 2.2 Reglas de creación

- **Todas las ramas de trabajo nacen de `develop`**
- **Nombre**: `tipo/descripcion-breve` en kebab-case, ASCII-only
- **Validación automática**: `validate_branch_name()` en todos los entry points
- **Prohibido**: `git checkout -b` y `git switch -c` directos (requieren harness)
- **Auto-registro**: Cada rama creada se registra en `branch_dependencies.yaml`

### 2.3 Convención de nombres

```
<tipo>/<segmento-o-descripcion>

Ejemplos:
  feature/branch-shield
  fix/login-timeout-auth
  docs/api-reference-v2
  research/benchmark-sapling-2026
  experiment/prototipo-ui-sapling
  task/s146-validacion-final
```

**Reglas**:
- ASCII-only (sin tildes, ñ, emojis)
- kebab-case (guiones, no underscores ni espacios)
- Máximo 128 caracteres total
- Componentes sin `-` al inicio
- Sin `..`, `.lock`, espacios, caracteres de control

---

## 3. Protección por tiers

### 3.1 ULTRA (main, master)

| Sin waiver | Con waiver |
|---|---|
| read, verify, status, inspect, diagnose, report, sync, checkout | write, commit, push, merge, stage, implement, mutate |
| Force-push/force-delete: **PERMANENTEMENTE BLOQUEADO** | |

**Requisitos del waiver ULTRA**:
- Scope obligatorio: `release`, `hotfix`, o `emergency`
- Razón mínima 20 caracteres
- Challenge suite completa
- Expira en 60 minutos máximo

### 3.2 HIGH (develop, prod, release/*, hotfix/*, etc.)

| Sin waiver | Con waiver |
|---|---|
| read, verify, status, inspect, diagnose, report, sync, checkout | write, commit, push, merge, stage, implement, mutate |
| Force bloqueado | |

### 3.3 STANDARD (todas las ramas de trabajo)

| Siempre permitido |
|---|
| read, verify, status, inspect, diagnose, report, sync, checkout, write, commit, push, merge, stage, implement, mutate |
| Pre-flight advisory (no bloqueante) |

---

## 4. Detección inteligente de tipo

El módulo `branch_type_mapper.py` detecta automáticamente el prefijo según:

| Prioridad | Fuente | Ejemplo |
|---|---|---|
| 1 | `user_hint` explícito | `--type research` |
| 2 | `work_type` del segmento | `"bugfix"` → `fix/` |
| 3 | `segment_metadata` (tags, category) | `"documentation"` → `docs/` |
| 4 | Perfil activo | Eidren → `research/`, Thavren → `feature/` |
| 5 | Keywords en descripción | "fix login bug" → `fix/` |
| 6 | Default | `task/` |

---

## 5. Garbage Collection

| Trigger | Acción |
|---|---|
| `on_session_start` | GC scan de ramas mergeadas |
| `on_session_close` | GC scan de ramas stale |
| Manual | `python3 tools/harnesses/h_gc_branches.py --execute` |

**Políticas de lifetime**:
- Ramas mergeadas a develop → auto-delete inmediato
- Ramas no mergeadas: warning al alcanzar `warning_days`, archive/delete al alcanzar `action_days`
- `research/` tiene lifetime extendido (30/60 días)
- `experiment/` es desechable (14/21 días)

---

## 6. Integración con otras superficies

| Superficie | Integración |
|---|---|
# Worktree resolver removed — worktree automation system eliminated 2026-06-11
| **Knowledge Compiler** | Detecta patrones de naming y sugiere mejoras |
| **Auto-triggers** | `before_branch_create`, `after_branch_create`, `before_branch_delete` |
| **Snapshots** | Registran branch activo en cada snapshot |
| **Workspace safety gate** | Verifica branch + repo + cwd antes de writes |
| **Git push gate** | HTTPS-only, workspace safety obligatorio |
| **Permisos** | `git checkout -b` DENY, `git branch -d/-D` DENY vía bash (requiere harness) |

---

## 7. Archivos clave

| Archivo | Rol |
|---|---|
| `tools/security/branch_types.py` | SSOT: definiciones canónicas de tipos, tiers, lifetime |
| `tools/security/branch_type_mapper.py` | Detección inteligente de tipo basada en contexto |
| `tools/security/validators.py` | `validate_branch_name()` — validación exhaustiva |
| `tools/validators/branch_shield.py` | Protección 3-tier (ULTRA/HIGH/STANDARD) |
| `tools/validators/check_protected_branch.py` | Gate de ramas protegidas con waiver CEO |
| `ovav worktree <create\|done>` | Lifecycle completo: create/validate/merge/delete |
| `tools/harnesses/h_gc_branches.py` | Garbage collection de ramas huérfanas |
| `tools/harnesses/workspace_safety_gate.py` | Safety gate pre-write |
# worktree_resolver.py reference removed — worktree automation system eliminated 2026-06-11
| `.ovav/registry/branch_dependencies.yaml` | Registro de dependencias entre ramas |

---

## 8. Evolución

Este documento reemplaza todas las definiciones fragmentadas que existían en 16+ listas distribuidas en 6+ archivos. Cualquier cambio a la taxonomía de ramas debe hacerse en `branch_types.py` (SSOT) y reflejarse aquí.

**Versión**: 2.0.0 — 2026-06-07
**Alineado a**: Conventional Commits 1.0.0, GitHub Rulesets, Trunk-Based Development
