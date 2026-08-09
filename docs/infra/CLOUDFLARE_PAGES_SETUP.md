# Cloudflare Pages — Configuración de Entornos (Staging + Previews)

> **Para:** CEO  
> **Ejecutar en:** Cloudflare Dashboard → Workers & Pages  
> **Tiempo estimado:** 15 minutos

---

## Paso 1: Configurar staging.ovav.dev (branch alias)

### Objetivo
Cada push a `develop` → visible automáticamente en `staging.ovav.dev`

### En Cloudflare Dashboard

```
Workers & Pages → ovav-landing → Settings → Domains & Routes
```

**Agregar dominio:**

| Campo | Valor |
|-------|-------|
| Domain | `staging.ovav.dev` |
| Branch | `develop` |

> Cloudflare Pages automáticamente desplegará el branch `develop` en `staging.ovav.dev` cuando se haga push. Sin CI extra, sin scripts.

### Verificar DNS (si no existe)

```
DNS → Records → Add record
  Type:   CNAME
  Name:   staging
  Target: ovav-landing.pages.dev
  Proxy:  ✅ (orange cloud)
```

---

## Paso 2: Desactivar auto-deploy en producción

### Objetivo
`main` no se despliega automáticamente — solo por tag o manual.

### En Cloudflare Dashboard

```
Workers & Pages → ovav-landing → Settings → Builds & deployments
```

| Campo | Valor |
|-------|-------|
| Production branch | `main` |
| **Enable automatic production branch deployments** | ❌ **DESACTIVADO** |

> Con esto, pushear a `main` NO dispara deploy. Solo se despliega producción cuando se crea un tag `v*` (vía GitHub Actions) o manualmente.

---

## Paso 3: Verificar Preview Deployments

Cada branch que NO es `main` ni `develop` recibe automáticamente un preview URL:

```
Workers & Pages → ovav-landing → Settings → Builds & deployments
```

| Campo | Valor |
|-------|-------|
| Preview branches | ✅ **All non-Production branches** (default) |

**Resultado:** Cada `task/*` branch que pushees → `https://<hash>.ovav-landing.pages.dev`

---

## Paso 4: (Opcional) Proteger previews con Access

```
Settings → General → Enable access policy
```

Restringe los preview URLs a tu cuenta Cloudflare. Recomendado si el repo es público.

---

## Resumen post-configuración

| Branch | URL | Deploy |
|--------|-----|--------|
| `main` | `ovav.dev` | ❌ Manual (solo tag v*) |
| `develop` | `staging.ovav.dev` | ✅ Automático |
| `task/*` | `<hash>.ovav-landing.pages.dev` | ✅ Automático |
