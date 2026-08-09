# Cleanup Checklist — OVAV

**Fecha:** 2026-08-07  
**Status:** En progreso

---

## ✅ LOCAL (Completado)

### Archivos Eliminados
- [x] `landing/CHANGELOG.md`
- [x] `landing/CONTENT_REQUEST_VALERIA.md`
- [x] `landing/COPY_REQUEST_SOFIA.md`
- [x] `landing/DESIGN_BRIEF_ELENA.md`
- [x] `landing/FROM_OVAV_WEB_REPO.md`
- [x] `landing/HANDOFF_RESPONSE_FROM_THAVREN.md`
- [x] `landing/nginx.conf`
- [x] `landing/docker-compose.yml`
- [x] `apps/landing/` (vacío)
- [x] `apps/cpanel/` (migrado a tools/cpanel)
- [x] `services/` (vacío)
- [x] `.owav/worktrees/feature-owd-test/`

### Archivos Creados
- [x] `.gitignore` actualizado

### Directories Estructura Final
```
ovav-systems/
├── landing/
│   ├── frontend/          ← Landing (Next.js)
│   └── backend/          ← API (FastAPI)
├── apps/
│   └── docs/             ← Documentación
├── tools/
│   └── cpanel/          ← Admin panel
├── go-runtime/          ← Core del sistema
├── docs/                 ← Documentos de arquitectura
├── scripts/              ← Scripts de utilidad
└── infra/               ← Terraform, etc.
```

---

## ⏳ CLOUDFLARE (Pendiente - Requiere acción manual)

### Dashboard: https://dash.cloudflare.com/a28bc37b8c9dc3e9b1b348c3a2ac729f/pages

### Eliminar Proyectos
1. **ovav-systems**
   - Razón: Deployment history lleno, duplicado de ovav-landing
   - Acción: Dashboard → ovav-systems → Settings → Delete

2. **Deployments innecesarios de ovav-landing**
   - `develop.ovav-landing.pages.dev` → ELIMINAR
   - Otros previews → ELIMINAR
   - Mantener solo producción: `ovav.dev`

### Mantener
- [ ] `ovav-landing` → ovav.dev (producción)
- [ ] `ovav-docs` → docs.ovav.dev

### DNS Records a Limpiar
```
TYPE    NAME            DELETE
────────────────────────────────
CNAME   staging         → staging.ovav.dev (si no se usa)
CNAME   preview         → *.ovav-dev.pages.dev (si no se usa)
A        @              → Cloudflare Pages IP (OK)
CNAME   www             → ovav.dev (OK)
CNAME   docs            → docs.ovav.dev (OK)
```

---

## ⏳ FLY.IO (Verificar)

### Apps en Fly.io
Verificar en: `https://fly.io/apps`

### Eliminar si existe
- `ovav-portal` → Duplicado de cPanel

### Mantener
- [ ] `ovav-backend` o similar → API backend

---

## ⏳ GITHUB (Pendiente)

### Repositorios
| Repo | Status | Acción |
|------|--------|--------|
| ovav-systems | ✅ OK | Mantener |
| ovav-docs | ❌ ARCHIVED | Mantener archived |
| ovav-web | ❌ ARCHIVED | Mantener archived |
| ovav-portal | ❌ ARCHIVED | Mantener archived |

### Actions a Limpiar
- Ir a: https://github.com/ovav-dev/ovav-systems/settings/actions
- Delete old workflow runs

---

## 🚀 FLUJO DE DESPLIEGUE CORRECTO

### LOCAL → PRODUCCIÓN

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DESARROLLO LOCAL                                │
│                                                                          │
│  1. Trabajar en local                                                  │
│     pnpm dev                                                           │
│                                                                          │
│  2. Verificar todo funciona                                             │
│     localhost:3000  → Landing                                          │
│     localhost:8080  → API                                             │
│     localhost:4321  → Docs                                            │
│                                                                          │
│  3. Si TODO funciona → EVALUAR enviar a producción                      │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
                               │
                               ▼ ¿OK?
┌─────────────────────────────────────────────────────────────────────────┐
│                         EVALUACIÓN                                     │
│                                                                          │
│  ¿Está listo para producción?                                          │
│  ├── [ ] Landing funciona correctamente                                 │
│  ├── [ ] Pricing configurado                                           │
│  ├── [ ] Auth funcionando (si aplica)                                  │
│  ├── [ ] No hay errores en consola                                     │
│  └── [ ] Tested en local                                               │
│                                                                          │
│  ¿SÍ? → git push → GitHub Actions → Producción                       │
│  ¿NO? → Continuar desarrollo local                                     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Ramas

```
main                    ← Producción (protected)
│
└── develop           ← Staging (auto-deploy preview)
    │
    └── feature/*     ← Desarrollo (PR → develop)
```

### Deploy Automático

```yaml
# Solo push a main = producción
# Push a develop = staging/preview
```

---

## 📋 CHECKLIST PRE-DEPLOY

Antes de hacer `git push` a main:

- [ ] Todo funciona en local
- [ ] No hay console errors
- [ ] Responsive en mobile
- [ ] Auth (Clerk) configurado
- [ ] Stripe configurado
- [ ] DNS apunta correctamente

---

## 🔧 COMANDOS ÚTILES

```bash
# Limpiar builds
rm -rf landing/frontend/.next
rm -rf landing/frontend/node_modules
rm -rf apps/docs/node_modules

# Reinstalar
pnpm install
pnpm dev

# Ver estado git
git status

# Ver qué se va a push
git diff --stat origin/main
```

---

*Document: 2026-08-07*
