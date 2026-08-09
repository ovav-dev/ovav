# Propuesta: Workflow Git OVAV v3.0

> **De:** Thavren · Platform Engineering  
> **Para:** CEO · Revisión  
> **Estado:** PROPUESTA — no implementado

---

## Problema actual

El workflow `owc`/`owd` actual funciona pero tiene fugas:

```bash
# Hoy: mezcla de comandos OVAV + git puro
owc feat-x          # ✅ wrapper OVAV
git add file.go     # ❌ git puro
git commit -m "..." # ❌ git puro  
owd                 # ⚠️ requiere waiver si lo corre agente
git push origin develop  # ❌ manual, olvidable
# ... merge a main? → ❌ nunca se hace
# ... tag de versión? → ❌ nunca se hace
```

**Fugas detectadas:**
1. `owd` detecta TTY y bloquea agentes → hay que hacer waiver cada vez
2. No hay validación pre-push (secrets, tests, gofmt)
3. `main` queda stale porque nadie mergea develop→main
4. No hay versionado automático
5. Agentes y CEO usan git puro mezclado con wrappers

---

## Workflow propuesto: `ovav git <comando>`

Un solo entrypoint que reemplaza TODOS los comandos git sueltos:

```
ovav git start <feature>    → crea branch + worktree
ovav git status             → estado completo del repo
ovav git save "<mensaje>"   → stage + commit con formato forzado
ovav git push               → push con validación pre-flight
ovav git merge              → merge a develop (con gates)
ovav git release <version>  → merge a main + tag + CHANGELOG
```

---

## Comandos detallados

### `ovav git start <feature>`
```bash
# Reemplaza: owc
$ ovav git start fix-cpanel-oauth

  ✓ Branch creado: task/fix-cpanel-oauth (desde develop)
  ✓ Worktree: .ovav/worktrees/task-fix-cpanel-oauth/
  ✓ Hook pre-commit instalado
  
  Next: ovav git save "tu mensaje" → ovav git push
```

### `ovav git status`
```bash
# Reemplaza: git status + git log dispersos
$ ovav git status

  Branch:     task/tasknext-ceo-task9
  Base:       develop (3 commits ahead)
  Changes:    12 modified, 5 new, 28 deleted
  Validators: 58/61 PASS
  Tests:      16/16 PASS (0 races)
  Next:       ovav git save "mensaje" → ovav git push
```

### `ovav git save "<mensaje>"`
```bash
# Reemplaza: git add + git commit
# FORMA el mensaje automáticamente con tipo + scope
$ ovav git save "corregido OAuth exchange mock"

  ┌─────────────────────────────────────────┐
  │ Pre-commit gate:                        │
  │  ✓ gofmt          0 files               │
  │  ✓ go vet         clean                 │
  │  ✓ secrets scan   0 found               │
  │  ✓ test quick     12/12 PASS            │
  │  ✓ branch shield  task branch OK        │
  └─────────────────────────────────────────┘
  
  Commit: fix(cpanel): corregido OAuth exchange mock
  [task/fix-cpanel-oauth a3f29b1]
```

**Formato forzado del mensaje:**
```
<tipo>(<scope>): <descripción>

Tipos: feat, fix, docs, chore, refactor, test, security
Scope: cpanel, validators, cli, cockpit, install, vault, docs, arch, config
```

Si el mensaje no sigue el formato → **BLOCKED**.

### `ovav git push`
```bash
# Reemplaza: git push origin <branch>
# Ejecuta validación completa antes de pushear
$ ovav git push

  ┌─────────────────────────────────────────┐
  │ Pre-push gate:                          │
  │  ✓ go test -race  16/16 PASS           │
  │  ✓ validate_all    58/61 PASS          │
  │  ✓ secrets_hygiene 0 found             │
  │  ✓ supply_chain    SBOM OK             │
  │  ✓ contract_fresh  all fresh           │
  └─────────────────────────────────────────┘
  
  Push: task/fix-cpanel-oauth → origin
  Preview: https://a3f29b1c.ovav-landing.pages.dev
```

### `ovav git merge`
```bash
# Reemplaza: owd (sin bloqueo TTY para agentes)
# El waiver se maneja a nivel OVAV, no TTY
$ ovav git merge

  Merge: task/fix-cpanel-oauth → develop
  ✓ Fast-forward OK
  ✓ Push develop → origin
  
  Staging deploy: https://staging.ovav.dev (auto)
```

**Regla de waiver:** Si lo ejecuta un agente → requiere waiver `.ovav/runtime/merge_waiver.yaml`.  
Si lo ejecuta el CEO → sin waiver (detección por usuario git config).

### `ovav git release <version>`
```bash
# Reemplaza: merge manual develop→main + git tag + changelog
$ ovav git release 2.1.0

  ┌─────────────────────────────────────────┐
  │ Release gate:                           │
  │  ✓ develop up-to-date with remote       │
  │  ✓ all validators PASS (58/61 OK)      │
  │  ✓ tests 16/16 PASS (0 races)          │
  │  ✓ no uncommitted changes               │
  │  ✓ CHANGELOG.md updated                 │
  └─────────────────────────────────────────┘
  
  Merge develop → main
  Tag: v2.1.0
  Push: main + tags → origin
  
  Production deploy triggered:
    ovav.dev         (CF Pages — manual)
    cpanel.ovav.dev  (Fly.io — auto on tag)
  
  ✓ GitHub Release creado: v2.1.0
```

---

## Comparativa: antes vs. después

| Situación | Hoy | Propuesto |
|-----------|-----|-----------|
| Crear branch | `owc feat` + worktree manual | `ovav git start feat` |
| Committear | `git add` + `git commit` suelto | `ovav git save "msg"` con formato forzado |
| Pushear | `git push origin` (sin gates) | `ovav git push` (tests + validators + secrets) |
| Merge a develop | `owd` (bloquea agentes por TTY) | `ovav git merge` (waiver file, no TTY) |
| Release | ❌ No existe | `ovav git release 2.1.0` (merge + tag + changelog) |
| Ver estado | `git status` + `git log` sueltos | `ovav git status` (panel unificado) |

---

## Implementación

### Opción A: Shell script (`tools/cli/ovav_git.sh`)
- Ventaja: simple, sin dependencias
- Desventaja: menos robusto que Go

### Opción B: Comando Go (`cmd/ovav/git.go` con subcomandos Cobra)
- Ventaja: integrado al runtime Go, testable, cross-platform
- Desventaja: más código

### Recomendación: **Opción B** (Go)
Ya tenemos `cmd/ovav/` con subcomandos. Agregar `ovav git *` es natural y consistente con el resto del CLI (`ovav profile`, `ovav sbom`, `ovav validate`).

---

## Riesgos y mitigaciones

| Riesgo | Mitigación |
|--------|-----------|
| `ovav git push` muy lento (tests + validators) | Flag `--quick` para skip tests en iteraciones rápidas |
| Agente se saltea los gates | `ovav git` es el ÚNICO entrypoint — se bloquea `git` puro vía pre-commit hook |
| Waiver bypass | El waiver file tiene firma HMAC, no se puede falsificar |

---

## Plan de rollout

1. **Fase 1**: Implementar `ovav git start`, `ovav git status`, `ovav git save`
2. **Fase 2**: Implementar `ovav git push` (con pre-push gates)
3. **Fase 3**: Implementar `ovav git merge` (reemplaza `owd`)
4. **Fase 4**: Implementar `ovav git release` (tag + changelog + deploy trigger)
5. **Fase 5**: Deprecar `owc`/`owd` y bloquear `git push` directo

---

> **Decisión requerida:** ¿Autorizás implementar Fase 1-2 en Task10?
