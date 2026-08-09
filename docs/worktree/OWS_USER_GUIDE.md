# OWS — OVAV Worktree System · Guía de Uso v3.0

> **Para:** Desarrolladores y operadores OVAV
> **Versión:** v3.0 — Junio 2026
> **Git requerido:** 2.43+ (recomendado 2.54+)
> **Ubicación canónica:** `docs/worktree/OWS_USER_GUIDE.md`

---

## ¿Qué es OWS?

OWS gobierna cómo trabajás en OVAV. No reemplaza Git — lo automatiza.
Cada tarea vive en su propio worktree aislado. Cuando terminás, OWS verifica,
mergea, limpia y publica. Sin ensuciar el repositorio principal.

**Ciclo diario:** `owc` → trabajás → `owd`

---

## Comandos esenciales (los que usás todos los días)

### `owc` — Crear worktree

```fish
owc feature/login-redesign     # Rama feature/login-redesign desde develop
owc fix/bug-123                # Rama fix/bug-123 desde develop
owc hotfix/critical            # Rama hotfix/critical desde main (mergea a main+develop)
owc spike/test-new-lib         # Rama spike/test-new-lib (no mergea, 48h TTL)
```

**Qué hace:**
- Crea rama local (NUNCA remoto — cero ruido)
- Crea worktree en `.ovav/worktrees/<nombre>`
- Instala hooks + configura git rerere + maintenance
- Predice conflictos vs develop antes de crear
- Auto-cd al worktree (vía fish alias)

**Perfiles disponibles:** `feature`, `fix`, `hotfix`, `release`, `docs`, `refactor`, `spike`, `research`, `migration`, `enterprise`, `emergency`, `patch`

---

### `owd` — Finalizar y publicar

```fish
owd                           # Auto-detecta worktree actual
owd feature/login-redesign    # Desde cualquier directorio, por nombre de rama
owd .ovav/worktrees/feature-* # Por path explícito
owd --compliance strict       # Gates estrictos (GPG + reviewer requeridos)
owd --reviewer "Alexander"    # Requerido en modo strict+
```

**Qué hace (en orden):**
1. Resuelve el worktree (3 modos: auto, por branch, por path)
2. Ejecuta pipeline completo de verificación:
   - Conflict prediction (`git merge-tree`)
   - Secrets sweep (27 patrones regex)
   - Forbidden files (.env, .pem, .key, binarios)
   - Validación Go (vet + fmt + test + validators)
   - Hygiene scan (archivos grandes, config git)
   - GPG signatures (strict+)
   - Reviewer requerido (strict+)
3. Merge local feature → develop (sin push aún)
4. Limpia worktree + borra rama feature
5. Push develop → origin

**Niveles de compliance:** `quick` | `standard` | `strict` | `maximum`

---

### `owl` — Listar worktrees

```fish
owl                  # Lista con conflict predictions vs develop
owl --history        # Audit trail (últimas 20 operaciones)
owl --json           # Formato máquina
```

---

### `owv` — Verificar sin mergear

```fish
owv                  # go vet + gofmt + go test + hygiene + validators
```

Ideal para verificar antes de `owd` sin comprometer el merge.

---

## Comandos de mantenimiento

### `ows` — Sincronizar y mantener

```fish
ows                  # Fetch + maintenance + prune
ows --rebase         # Fetch + rebase (antes owu)
ows --full           # Todo: fetch + rebase + maintenance + prune
```

### `owclean` — Limpiar worktrees huérfanos

```fish
owclean              # Limpia worktrees sin branch, >30d inactivos, spikes caducados
owclean --dry-run    # Ver sin borrar
```

### `owm` — Mover worktree

```fish
owm /nuevo/path      # Mueve worktree a otra ubicación
```

---

## Comandos avanzados

### `owx` — Ruta de cambios entre ramas

```fish
owx --target develop --mode cherry-pick    # Cherry-pick commits a develop
owx --target main --mode hotfix            # Hotfix: main + develop simultáneo
```

### `owa` — Abortar operación en progreso

```fish
owa                  # Aborta merge/rebase en curso
```

### `owr` — Rescatar trabajo perdido

```fish
owr                  # Recupera trabajo de branches/stashes huérfanos
```

### `owlk` — Bloquear worktree

```fish
owlk                 # Bloquea worktree para coordinación multi-agente
owlk --unlock        # Desbloquea
```

---

## Escenarios de uso diario

### Feature normal

```fish
$ owc feature/payment-v2
$ cd .ovav/worktrees/feature-payment-v2
# ... codificar, commitear ...
$ owv                 # verificar
$ owd                 # merge + cleanup + push
```

### Bug fix urgente desde cualquier lado

```fish
$ owc fix/crash-login
$ cd .ovav/worktrees/fix-crash-login
# ... fix ...
$ owd
```

### Hotfix en producción

```fish
$ owc hotfix/security-patch
$ cd .ovav/worktrees/hotfix-security-patch
# ... fix urgente ...
$ owd --compliance strict --reviewer "Alexander"
# Mergea a main Y develop, strict gates
```

### Cerrar trabajo desde el repo principal

```fish
$ cd ~/Systems/OVAV          # develop, main repo
$ owd feature/payment-v2     # owd encuentra el worktree solo
# Mismo resultado que desde el worktree
```

### Spike (exploración, no mergea)

```fish
$ owc spike/test-graphql
$ cd .ovav/worktrees/spike-test-graphql
# ... experimentar 48h ...
# No necesita owd — owclean lo limpia a las 48h
```

---

## Qué NO hacer

- ❌ `git push origin feature/...` — owd lo hace
- ❌ `git checkout develop && git merge feature/...` — owd lo hace
- ❌ `git branch -d feature/...` — owd lo limpia
- ❌ `rm -rf .ovav/worktrees/...` — owclean o owd lo hacen
- ❌ `git push --force` — bloqueado por hooks
- ❌ `owd` desde develop/main → usar `owd <branch-name>`

---

## Referencia rápida

| Comando | Acción |
|---|---|
| `owc <nombre>` | Crear worktree + branch |
| `owd [branch]` | Merge + cleanup + push |
| `owl` | Listar worktrees |
| `owv` | Verificar sin mergear |
| `ows` | Sync + mantenimiento |
| `owclean` | Limpiar huérfanos |
| `owx --target <T>` | Ruta de cambios |
| `owa` | Abortar operación |
| `owr` | Rescatar trabajo |
| `owlk` | Bloquear worktree |
| `owm <path>` | Mover worktree |
