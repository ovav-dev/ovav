# DNS Configuration — cPanel Split (ovav.dev → landing + cpanel.ovav.dev)

**Date:** 2026-06-15 23:15 UTC-5
**Author:** Thavren (Platform Engineering)
**Status:** READY — requiere CEO/owner para aplicar en Cloudflare Dashboard

---

## Target Architecture

```
ovav.dev           → Cloudflare Pages (landing page)
cpanel.ovav.dev    → Fly.io (Go backend + cPanel SPA)
```

---

## Step 1: Cloudflare Pages — Landing Page

### Deploy landing page
```bash
cd .ovav/projects/ovav-product/webapp/frontend
pnpm build                              # → out/
# Deploy via Cloudflare Dashboard:
#   Pages → Create project → ovav-landing
#   Upload out/ directory
#   Framework preset: None (static HTML)
```

### DNS CNAME record
```
Type:   CNAME
Name:   @  (or ovav.dev)
Target: ovav-landing.pages.dev
Proxy:  Proxied (orange cloud)
TTL:    Auto
```

> ⚠️ If `ovav.dev` currently has an A/AAAA record pointing to Fly.io, remove it first.
> This will route the root domain to the landing page.

---

## Step 2: Fly.io — cPanel Backend

### Add certificate for subdomain
```bash
flyctl certs create cpanel.ovav.dev --app ovav-systems
```

### Set OAuth redirect URI secret
```bash
flyctl secrets set OAUTH_REDIRECT_URI=https://cpanel.ovav.dev --app ovav-systems
```

### Update OAuth provider redirect URIs (Google Cloud Console / GitHub OAuth Apps)
```
Google: https://cpanel.ovav.dev
GitHub: https://cpanel.ovav.dev
```
> ⚠️ IMPORTANTE: El redirect URI debe ser la URL base del SPA (donde vive Login.tsx),
> NO el endpoint API. El frontend lee ?code= de los query params de la URL.
> Si se usa /api/v1/auth/oauth/google, Google redirigiría al endpoint API
> (mostrando JSON) en vez de al SPA. Hacerlo ANTES del switch DNS.

### DNS CNAME record
```
Type:   CNAME
Name:   cpanel
Target: ovav-systems.fly.dev
Proxy:  Proxied (orange cloud)
TTL:    Auto
```

---

## Step 3: Deploy y Verificar

```bash
# Redeploy cPanel backend con OAUTH_REDIRECT_URI
flyctl deploy --app ovav-systems --config fly.toml

# Verificar landing page
curl -I https://ovav.dev

# Verificar cPanel
curl -I https://cpanel.ovav.dev
curl https://cpanel.ovav.dev/health

# Verificar OAuth config
curl https://cpanel.ovav.dev/api/v1/auth/config
```

---

## Rollback (si algo falla)

```bash
# Revertir DNS: ovav.dev → Fly.io
# O: apuntar ovav.dev de vuelta al IP de Fly.io
flyctl ips list --app ovav-systems
# → Usar esos IPs como A records para ovav.dev
```

---

## Dependencies

| Componente | Estado | Responsable |
|---|---|---|
| Landing page build | ✅ Listo (`out/`) | Dante |
| OAuth redirect URIs (código) | ✅ Actualizado | Thavren |
| OAuth provider console update | 🔴 Pendiente | CEO |
| CF Pages deploy | 🔴 Pendiente | CEO / Uriel |
| Fly.io cert `cpanel.ovav.dev` | 🔴 Pendiente | CEO / Uriel |
| DNS records | 🔴 Pendiente | CEO (Cloudflare owner) |
| `flyctl secrets set OAUTH_REDIRECT_URI` | 🔴 Pendiente | CEO / Uriel |
| `flyctl deploy` | 🔴 Pendiente | CEO / Uriel |
