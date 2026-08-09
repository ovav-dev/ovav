# Worktree Continuity Protocol — OVAV v5.0
#
# Criterio manual para flujo simultáneo de 3+ worktrees.
# Comandos optimizados: owc, owcp, ows, owd, owx.

## Comandos esenciales

| Comando | Qué hace | Cuándo |
|---|---|---|
| `owc <feature>` | Crea worktree `task/<feature>` desde develop | Iniciar feature nueva |
| `owcp <hash>` | Cherry-pick 1 commit al bus develop | Compartir cambio sin mergear rama entera |
| `ows` | Status visual de TODAS las worktrees | Ver qué está pasando |
| `owd` | Merge a develop + push + cleanup (CEO-gated) | Feature terminada |
| `owx` | Limpiar worktrees huérfanas | Mantenimiento |

## Principio: develop es un BUS, no un destino

Los commits individuales viajan por develop sin requerir merge de ramas enteras.

## Escenario: 3 worktrees simultáneas

```
A: task/cpanel-v5       → backend Go
B: task/cpanel-frontend → React  
C: task/dominio-ovav    → DNS/SSL
```

### Situación 1: A tiene un cambio que B necesita AHORA

```fish
# Desde A
git commit -m "feat(CP): API contract v1"
owcp <hash>             # cherry-pick al bus develop

# Desde B
git pull origin develop # absorber el contrato
```

### Situación 2: A terminó todo

```fish
# Desde A
owd                     # merge + cleanup (CEO confirma)
```

### Situación 3: Ver estado general

```fish
ows
# Muestra:
#   ▶ cpanel-v5        ↑2 ↓0  cdb609c  feat(CP): Backend Go
#     cpanel-frontend  ↑0 ↓3  a1b2c3d  docs: initial setup
#     dominio-ovav     ↑0 ↓0  f9e8d7c  feat(DNS): records
```

## Flujo día a día

```
DÍA 1: owc cpanel-v5; owc cpanel-frontend; owc dominio-ovav
DÍA 2: A termina contrato → owcp <hash> → B recibe con pull
DÍA 3: A termina backend → owd → cleanup automático
DÍA 4: B termina frontend → owd
DÍA 5: C termina dominio → owd
```

## Lo que NUNCA hacer

```
❌ owd desde worktree NO lista para merge
❌ git push --force (bloqueado por OVAV)
❌ git merge sin pull previo
❌ Trabajar en develop directamente
```

## Rescate: cherry-pick falló

```fish
git status                    # ver archivos en conflicto
# resolver → git add <files> → git cherry-pick --continue
# abortar → git cherry-pick --abort; git checkout task/original
```
