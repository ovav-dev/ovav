# OVAV Production URL Audit — 2026-08-07

## Estado: ✅ 100% LIMPIO

### URLs Oficiales (Únicos puntos de entrada)

| URL | Tipo | Estado | Verificación |
|-----|------|--------|-------------|
| `https://ovav.dev` | Cloudflare Pages | ✅ 200 | Landing page público |
| `https://www.ovav.dev` | CNAME → ovav.dev | ✅ 200 | Redirect a landing |
| `https://docs.ovav.dev` | Cloudflare Pages | ✅ 200 | Documentación |
| `https://d678beea.ovav.dev` | Fly.io + Tunnel | ✅ 302 | Backend cPanel (auth) |
| `https://d678beea.ovav.dev/health` | Fly.io + Tunnel | ✅ 200 | Health check |
| `https://status.ovav.dev` | Better Uptime | ✅ 200 | Status page público |

### URLs Eliminadas/Redirigidas

| URL | Estado | Nota |
|-----|--------|------|
| `api.ovav.dev` | ❌ NXDOMAIN | Eliminada de DNS |
| `get.ovav.dev` | ❌ NXDOMAIN | Eliminada de DNS |
| `mcp.ovav.dev` | ❌ NXDOMAIN | Nunca existió |
| `cdn.ovav.dev` | ❌ NXDOMAIN | Nunca existió |
| `cpanel.ovav.dev` | ❌ NXDOMAIN | Eliminada por seguridad |
| `develop.ovav-landing.pages.dev` | ❌ ELIMINADO | Deploy borrado |
| `*.ovav-landing.pages.dev` | ❌ Desactivado | Preview deployments OFF |

### Cloudflare Pages Projects

#### ovav-landing
- **Production deploy:** `c002e5d0` (2026-06-16)
- **Aliases:** `ovav.dev`, `www.ovav.dev`
- **Preview deployments:** ❌ DESACTIVADOS

#### ovav-docs
- **GitHub:** `ovav-dev/ovav-docs`
- **Production branch:** `main`
- **Preview deployments:** ❌ DESACTIVADOS

### DNS Records Activos

```
TYPE    NAME                CONTENT                              PROXIED
────────────────────────────────────────────────────────────────────────────
CNAME   ovav.dev            ovav-landing.pages.dev               Yes
CNAME   www.ovav.dev        ovav-landing.pages.dev               Yes
CNAME   docs.ovav.dev       ovav-docs.pages.dev                  Yes
CNAME   d678beea.ovav.dev   (Cloudflare Tunnel)                 No
CNAME   status.ovav.dev     statuspage.betteruptime.com          No
MX      ovav.dev            route*.mx.cloudflare.net             No
TXT     ovav.dev            v=spf1...                           No
DKIM    _domainkey.ovav.dev ( DKIM keys)                        No
```

### Cloudflare Tunnel

```
Name: ovav-cpanel-prod
ID: 3b90d21c-ce98-41f9-883b-09980289560d
Connections: 4 (DFW)
Status: ✅ HEALTHY
```

---

## Arquitectura de Producción

```
                    ┌─────────────────────────────────────┐
                    │     Cloudflare Edge (CDN/WAF)        │
                    └──────────────┬──────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         │                         │                         │
         ▼                         ▼                         ▼
┌─────────────────┐     ┌─────────────────┐      ┌─────────────────┐
│  ovav-landing   │     │   ovav-docs    │      │  Cloudflare     │
│  (Pages)        │     │   (Pages)      │      │  Tunnel         │
│                 │     │                │      │                 │
│  ovav.dev  ────┼────►│  docs.ovav.dev │      │ d678beea.ovav. │
│  www.ovav.dev ─┘     │                │      │ dev            │
└─────────────────┘     └────────────────┘      └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │   Fly.io        │
                                                 │   ovav-systems  │
                                                 │                 │
                                                 │   Port 5858     │
                                                 │   (cPanel)      │
                                                 └─────────────────┘
```

---

## Verificación Rápida

```bash
#!/bin/bash
echo "=== OVAV Production Health Check ==="

urls=(
  "https://ovav.dev:200"
  "https://docs.ovav.dev:200"
  "https://d678beea.ovav.dev/health:200"
  "https://status.ovav.dev:200"
)

fail=0
for url_status in "${urls[@]}"; do
  url="${url_status%:*}"
  expected="${url_status##*:}"
  
  result=$(curl -s -o /dev/null -w "%{http_code}" "$url")
  
  if [[ "$result" == "$expected" ]]; then
    echo "✅ $url"
  else
    echo "❌ $url (got $result, expected $expected)"
    fail=1
  fi
done

# Verificar que urls eliminadas NO responden
echo ""
echo "=== URLs que deben estar muertas ==="

dead_urls=(
  "https://api.ovav.dev"
  "https://get.ovav.dev"
  "https://mcp.ovav.dev"
  "https://cdn.ovav.dev"
  "https://cpanel.ovav.dev"
)

for url in "${dead_urls[@]}"; do
  result=$(curl -s -o /dev/null -w "%{http_code}" "$url" --max-time 5)
  if [[ "$result" == "000" ]] || [[ "$result" =~ ^[45] ]]; then
    echo "✅ $url (dead: $result)"
  else
    echo "❌ $url (still alive: $result)"
    fail=1
  fi
done

exit $fail
```

---

## Nota sobre Cache de CDN

Después de eliminar los preview deployments, Cloudflare puede servir contenido cacheado
durante ~5 minutos. Si `develop.ovav-landing.pages.dev` aún responde, es normal.

Para forzar purge de cache (requiere token con permisos):
```
DELETE /zones/{zone_id}/purge_cache
Body: {"files": ["https://develop.ovav-landing.pages.dev/"]}
```

---

**Última verificación:** 2026-08-07  
**Estado:** ✅ PRODUCCIÓN LIMPIA Y ESTABLE
