# OVAV — Git Flow Structure

**Fecha:** 2026-08-07  
**Versión:** 1.0  
**Estado:** ✅ DEFINITIVO

---

## ESTRUCTURA OFICIAL

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              REMOTO                                        │
│                                                                             │
│   ┌─────────────────────────┐         ┌─────────────────────────┐          │
│   │         MAIN            │         │        DEVELOP          │          │
│   │   (Producción Real)     │         │  (Integración)         │          │
│   │                         │         │                         │          │
│   │  • Versiones estables   │         │  • Cambios acumulados   │          │
│   │  • Tags                 │         │  • Features aprobadas    │          │
│   │  • Hotfixes             │         │  • Listo para release   │          │
│   │  • Production-ready      │         │                         │          │
│   └─────────────────────────┘         └─────────────────────────┘          │
│            ↑                                    │                           │
│            │                                    │                           │
│            │         ┌─────────────────────────┘                           │
│            │         │                                                     │
│            │         │  PR/Merge cuando está stable                       │
│            │         │                                                     │
│            └─────────┘                                                     │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              LOCAL                                         │
│                                                                             │
│   ┌─────────────────────────┐         ┌─────────────────────────┐          │
│   │         MAIN            │         │        DEVELOP           │          │
│   │   (Mirror del remoto)   │         │   (Rama principal)       │          │
│   │                         │         │                         │          │
│   │  • Push a main remoto   │         │  • Rama de desarrollo    │          │
│   │  • Tags de producción   │         │  • Desde aquí nacen      │          │
│   │  • Solo push stable     │         │    las worktrees        │          │
│   └─────────────────────────┘         │  • Acumula cambios       │          │
│            ↑                           │    revisados/sin errores │          │
│            │                           │  • Merge a main cuando   │          │
│            │                           │    está listo           │          │
│            │                           └─────────────────────────┘          │
│            │                                     │                           │
│            │                                     │  Worktrees               │
│            │                                     ↓                           │
│            │                           ┌─────────────────────────┐          │
│            │                           │  feature/task-xxx        │          │
│            │                           │  feature/auth-xxx        │          │
│            │                           │  feature/stripe-xxx     │          │
│            │                           │  ...                    │          │
│            │                           └─────────────────────────┘          │
│            │                                     ↑                           │
│            │                                     │ PR/Merge                  │
│            │                                     │                           │
│            └─────────────────────────────────────┘                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## WORKFLOW DETALLADO

### 1. Inicio de día (sincronizar)
```bash
# Asegurar que estamos en develop y actualizado
git checkout develop
git pull origin develop

# Asegurar que main está actualizado
git checkout main
git pull origin main
```

### 2. Crear tarea/feature
```bash
# Desde develop, crear worktree
git worktree add .worktrees/feature-auth-clerk -b feature/auth-clerk

# Trabajar en la worktree
cd .worktrees/feature-auth-clerk
# ... hacer cambios ...
git add .
git commit -m "feat(auth): integrate Clerk authentication"
git push -u origin feature/auth-clerk
```

### 3. Merge a develop (después de probar)
```bash
# Volver a develop principal
git checkout develop
git merge feature/auth-clerk

# Probar que todo funciona juntos
# ... testing ...

# Push a develop remoto
git push origin develop
```

### 4. Release a producción
```bash
# Cuando develop está estable y probado:

# 1. Merge develop → main
git checkout main
git merge develop

# 2. Crear tag
git tag -a v3.0.0 -m "Release v3.0.0 - OVAV SaaS Launch"

# 3. Push todo
git push origin main
git push origin develop
git push origin v3.0.0
```

### 5. Hotfix (emergencia)
```bash
# Desde main (producción)
git checkout main

# Crear hotfix branch
git worktree add .worktrees/hotfix-urgent-fix -b hotfix/urgent-fix

# Trabajar en hotfix
cd .worktrees/hotfix-urgent-fix
# ... fixes ...
git add .
git commit -m "fix(hotfix): critical bug fix"
git push -u origin hotfix/urgent-fix

# Merge directo a main
git checkout main
git merge hotfix/urgent-fix
git push origin main

# También merge a develop
git checkout develop
git merge hotfix/urgent-fix
git push origin develop
```

---

## REGLAS DE ORO

### ✅ HACER
- Siempre crear worktrees desde `develop`
- Siempre probar en `develop` antes de mergear a `main`
- Usar naming consistente: `feature/`, `fix/`, `hotfix/`, `docs/`
- Commit atómico: un cambio = un commit
- Push frecuente a remoto para backup

### ❌ NO HACER
- Nunca trabajar directamente en `main` o `develop`
- Nunca hacer `git add .` — stagear archivos específicos
- Nunca hacer force push a `main` o `develop`
- Nunca mergear código no probado
- Nunca dejar worktrees huérfanas (sin branch remota)

---

## LIMPIEZA DE WORKTREES

### Ver worktrees activas
```bash
git worktree list
```

### Eliminar worktree terminada
```bash
# 1. Eliminar directorio
rm -rf .worktrees/feature-auth-clerk

# 2. Limpiar git
git worktree prune

# 3. Eliminar branch si ya no se necesita
git branch -d feature/auth-clerk
git push origin --delete feature/auth-clerk
```

---

## ESTRUCTURA DE COMMITS

```
<tipo>(<área>): descripción corta

- Bullet point 1
- Bullet point 2

Tipos:
  feat    → Nueva funcionalidad
  fix     → Corrección de bug
  hotfix  → Fix urgente en producción
  refactor→ Refactorización
  docs    → Documentación
  style   → Formato (sin cambio de lógica)
  test    → Tests
  chore   → Mantenimiento, deps, configs
```

**Ejemplos:**
```
feat(auth): integrate Clerk authentication
fix(stripe): resolve webhook signature validation
hotfix(payment): critical checkout redirect fix
docs(api): update endpoint documentation
refactor(dashboard): simplify state management
```

---

## синхронизация (Sincronización)

### Actualizar develop con últimos cambios de main
```bash
git checkout develop
git merge main
# Resolver conflictos si hay
git push origin develop
```

### Actualizar feature branch con develop
```bash
git checkout feature/auth-clerk
git merge develop
# Resolver conflictos si hay
git push origin feature/auth-clerk
```

---

## VALIDACIÓN ANTES DE MERGE A MAIN

Antes de hacer merge a `main`, verificar:

- [ ] Todos los tests pasan
- [ ] No hay warnings de linter
- [ ] Documentación actualizada
- [ ] CHANGELOG.md actualizado
- [ ] Versión incrementada en package.json
- [ ] Credenciales removidas (no hardcoded)
- [ ] Variables de entorno documentadas

---

## SCRIPTS ÚTILES

### Inicio de sesión
```bash
#!/bin/bash
# scripts/git-start.sh
git checkout develop
git pull origin develop
git checkout main
git pull origin main
git worktree list
```

### Preparar release
```bash
#!/bin/bash
# scripts/git-release.sh
VERSION=$1
git checkout main
git merge develop
git tag -a $VERSION -m "Release $VERSION"
git push origin main develop $VERSION
```

---

*Documento creado: 2026-08-07*
*Versión: 1.0*
