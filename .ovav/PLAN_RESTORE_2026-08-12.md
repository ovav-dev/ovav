# Plan de Referencia — Limpieza y Arquitectura OVAV
**Fecha:** 2026-08-12
**Estado:** Listo para revisión post-restore

---

## ✅ Checklist de Limpieza Ejecutada

- [x] git clean -xfd ejecutado (2.8GB liberados)
- [x] Remote corregido: `ovav-systems` → `ovav`
- [x] Binarios trackeados verificados: `bin/ovav`, `bin/ovav-cockpit`
- [x] Vault tokens: ELIMINADOS (backup manual requerido post-restore)
- [x] Repo limpio: 0 cambios, 0 untracked

---

## 🔐 Estado de Accesos

| Servicio | Estado | Nota |
|----------|--------|------|
| GitHub (gh CLI) | ✅ OK | Token con scope `repo`, `workflow` |
| Cloudflare (wrangler) | ✅ OK | OAuth `alexander.salvador.dev@gmail.com` |
| Git Push a `develop` | ❌ BLOQUEADO | Rama protegida, requiere PR |
| Git Push a `feature/*` | ❌ BLOQUEADO | Gate local |

**Acción post-restore:** Hacer PR o deshabilitar protección de `develop` temporalmente.

---

## 📐 Arquitectura Actual vs Target

### ACTUAL (confusa)
```
OVAV (repo único, 1.6GB)
├── go-runtime/        # Go CLI + validators + cockpit
├── tools/cpanel/      # React admin (d678beea.ovav.dev)
├── web/
│   ├── frontend/      # Next.js (NO DEPLOYADO - MISTERIO)
│   └── backend/        # FastAPI (NO DEPLOYADO)
├── apps/docs/         # Astro → docs.ovav.dev ✅
├── docs/              # Markdown docs (no deployado)
├── clients/           # OpenCode/MiMo configs
└── .ovav/            # Governance configs
```

### PROBLEMAS DETECTADOS
1. `wrangler.toml` referencia `landing/frontend/out/` que NO EXISTE en repo
2. `web/frontend` (Next.js) no está deployado - ¿fuente en otro repo?
3. `web/backend` (FastAPI) no está deployado
4. Mismatch entre lo que hay y lo que está en producción

---

## 🎯 Arquitectura Target — Discusión Pendiente

### Opción A: Monorepo Total
```
ovav-dev/ovav/
├── packages/
│   ├── runtime-go/      # Go runtime (ovav, cockpit, cpanel)
│   ├── frontend-cpanel/ # React admin (d678beea.ovav.dev)
│   └── governance-py/   # Python tools
├── skills/              # OpenCode + MiMoCode skills
├── agents/              # Agent definitions
└── .ovav/              # Governance configs
```
**Pros:** Todo junto, CI/CD único
**Cons:** Deploy acoplado

### Opción B: Separación Inteligente
```
ovav-dev/ovav/          # Core runtime + governance
ovav-dev/ovav-web/      # Marketing + Auth + User Dashboard
ovav-dev/ovav-docs/     # Documentación
```
**Pros:** Escalabilidad, equipos independientes
**Cons:** Sincronización cross-repo

### Opción C: Monorepo con Turborepo
```
ovav-dev/ovav/
├── apps/
│   ├── web-marketing/   # Landing + Auth (Next.js)
│   ├── web-admin/       # cPanel (React)
│   └── docs/           # Documentación (Astro)
├── packages/
│   ├── runtime-go/      # Go core
│   └── shared-types/    # Types compartidos
└── .ovav/              # Governance
```
**Pros:** Mejor de ambos mundos
**Cons:** Complejidad de build

---

## 🌐 Preguntas Clave para Resolver Post-Restore

1. **¿`landing/frontend/out/` está en otro repo?** Investigar `ovav-landing` o similar
2. **¿`web/backend` FastAPI para qué se usa?** No hay deploy matching
3. **¿`d678beea.ovav.dev` usa qué backend?** Fly.io + Go backend
4. **¿Migrar `d678beea` → `app.ovav.dev`?** Decisión de branding
5. **¿1 dominio o subdomains?** `ovav.dev` + `app.ovav.dev` + `docs.ovav.dev`

---

## 📋 Acciones Pendientes

### Prioridad INMEDIATA (post-restore)
1. [ ] Regenerar vault tokens (eliminados por git clean)
2. [ ] Hacer PR a `develop` con los 87 commits adelantados
3. [ ] Investigar `landing/frontend/out/` source

### Prioridad ALTA (semana 1)
4. [ ] Decidir estructura: Opción A, B o C
5. [ ] Reorganizar repo según decisión
6. [ ] Sincronizar `web/` con lo deployado

### Prioridad MEDIA (semana 2)
7. [ ] Implementar Turborepo/Nx si Opción C
8. [ ] Migrar docs a repo separado si Opción B
9. [ ] Evaluar MCP compatibility

---

## 🔧 Comandos Útiles Post-Restore

```bash
# Ver qué cambió localmente
git status && git log --oneline origin/develop..develop

# Regenerar vault tokens
ovav vault gen-key
# затем вручную восстановить API keys

# Hacer PR
gh pr create --base develop --head feature/cleanup-2026-08-12

# Push a feature branch
git push origin develop:feature/tu-nombre --force-with-lease
```

---

## 📊 Tamaños Post-Limpieza

| Componente | Tamaño |
|------------|--------|
| go-runtime | 53MB |
| tools | 832KB |
| web | 2.2MB |
| apps/docs | 132KB |
| .ovav | 14MB |
| clients | 892KB |
| **TOTAL** | **~70MB** |

Repo total: 1.6GB (incluye .git con 79572 objetos, 285MB en packs)

---

*Generado: 2026-08-12 por Thavren (Platform Engineering)*
