# OVAV Production Cleanup Instructions

**Fecha:** 2026-08-07  
**Estado:** REQUERIDO PARA PRODUCCIÓN  
**Propósito:** Eliminar todas las rutas innecesarias, dejar solo 3 puntos de entrada oficiales

---

## Dominios Oficiales (MANTENER)

| Dominio | Tipo | Propósito |
|---------|------|-----------|
| `ovav.dev` | Cloudflare Pages | Landing page público (PRODUCCIÓN) |
| `www.ovav.dev` | CNAME → ovav.dev | Redirect |
| `docs.ovav.dev` | Cloudflare Pages | Documentación pública |
| `d678beea.ovav.dev` | Fly.io + Cloudflare Tunnel | Backend cPanel (admin, no público) |
| `status.ovav.dev` | Cloudflare Pages/Worker | Status page público |

---

## Acciones Requeridas en Cloudflare Dashboard

### 1. Eliminar DNS Records

Ir a: **Cloudflare Dashboard → ovav.dev → DNS → Records**

**ELIMINAR estos records:**

```
TYPE    NAME              CONTENT           PROXIED
─────────────────────────────────────────────────────────────
A       get               (redirect)        No
A       api               (eliminar)        No
A       mcp               (eliminar)        No
A       cdn               (eliminar)        No
A       staging           (eliminar)        No
A       test              (eliminar)        No
A       custom            (eliminar)        No
A       releases          (eliminar)        No
CNAME   TU-SUBDOMINIO     (eliminar)        No
A       cpanel            (ya eliminado)    -
```

**MANTENER:**
```
TYPE    NAME              CONTENT                    PROXIED
─────────────────────────────────────────────────────────────
AAAA    @                  (Cloudflare Pages IP)      Yes
CNAME   www                ovav.dev                   Yes (proxy)
CNAME   docs               (Cloudflare Pages)         Yes
CNAME   d678beea           (Cloudflare Tunnel)        No
CNAME   status             (Better Uptime/custom)     Yes
```

### 2. Configurar Redirect 301 para get.ovav.dev

Ir a: **Cloudflare Dashboard → ovav.dev → Rules → Redirect Rules**

Crear regla:
```
SI: hostname == get.ovav.dev
ENTONCES: Redirect to https://ovav.dev{path}
Tipo: 301 (Permanent)
```

### 3. Eliminar Cloudflare Pages Preview Deployments

Ir a: **Cloudflare Dashboard → Workers & Pages → ovav-landing → Settings → Builds and deployments**

**Desactivar:**
- [ ] Preview deployments
- [ ] Branch control (eliminar branch `develop`)

**Eliminar historial:**
- Ir a **Deployments** tab → **Options** (•••) → **Delete deployment**

### 4. Eliminar Dominios Personalizados Innecesarios

Ir a: **Cloudflare Dashboard → Workers & Pages → ovav-landing → Settings → Custom domains**

**ELIMINAR:**
- ❌ `get.ovav.dev` (redirigir a ovav.dev)
- ❌ Cualquier otro dominio no listado arriba

**MANTENER:**
- ✅ `ovav.dev` (producción)

---

## Acciones Requeridas en Fly.io

### Verificar Apps Activas

```bash
fly apps list
```

**DEBERÍAS VER:**
```
NAME                OWNER    STATUS
ovav-systems        private  running
```

**ELIMINAR si existen:**
- ❌ `ovav-systems-staging` (si no se usa)
- ❌ `ovav-cpanel` (legacy)
- ❌ Cualquier app con nombre similar

### Verificar Secrets

```bash
fly secrets list -a ovav-systems
```

**MANTENER:**
- `TUNNEL_TOKEN` (Cloudflare tunnel)

### Eliminar dominios personalizados en Fly.io

```bash
# Ver dominios certificados
fly certs list -a ovav-systems

# Eliminar dominios legacy
fly certs delete ovav-cpanel.fly.dev -a ovav-systems 2>/dev/null || echo "No existe"
fly certs delete api.ovav.dev -a ovav-systems 2>/dev/null || echo "No existe"
```

---

## Verificación Post-Limpieza

### Probar desde navegador (incógnito):

```bash
# DEBEN CARGAR:
✅ https://ovav.dev
✅ https://docs.ovav.dev
✅ https://status.ovav.dev
✅ https://d678beea.ovav.dev/health (autenticado)

# DEBEN FALLAR o redirigir:
❌ https://get.ovav.dev         → Redirect a ovav.dev
❌ https://api.ovav.dev         → 404 o NXDOMAIN
❌ https://mcp.ovav.dev         → NXDOMAIN
❌ https://cdn.ovav.dev         → NXDOMAIN
❌ https://develop.ovav-landing.pages.dev → 404
```

### Probar desde curl:

```bash
# Verificar redirect de get.ovav.dev
curl -I https://get.ovav.dev 2>/dev/null | grep -E "HTTP|Location"

# Verificar que api.ovav.dev no existe
curl -I https://api.ovav.dev 2>/dev/null | grep -E "HTTP"

# Verificar que d678beea responde
curl -s https://d678beea.ovav.dev/health | jq .
```

---

## Checklist Final

- [ ] DNS: Solo 5 records activos (ovav.dev, www, docs, d678beea, status)
- [ ] DNS: Redirect 301 para get.ovav.dev → ovav.dev
- [ ] DNS: api.ovav.dev, mcp.ovav.dev, cdn.ovav.dev eliminados
- [ ] Cloudflare Pages: Preview deployments desactivados
- [ ] Cloudflare Pages: Solo ovav.dev como custom domain
- [ ] Fly.io: Solo app `ovav-systems` activa
- [ ] Fly.io: Certificados legacy eliminados
- [ ] Verificación: 3 URLs oficiales funcionan

---

## Scripts de Verificación Automatizada

```bash
#!/bin/bash
# Verificar estado de producción

echo "=== OVAV Production URL Check ==="
echo ""

check_url() {
    local url=$1
    local name=$2
    local expected=$3
    
    result=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
    
    if [[ "$result" == "$expected" ]] || [[ "$expected" == "2xx" && "$result" =~ ^2 ]]; then
        echo "✅ $name: $url ($result)"
    elif [[ "$expected" == "redirect" && "$result" =~ ^(301|302)$ ]]; then
        echo "✅ $name: $url (redirect $result)"
    else
        echo "❌ $name: $url (got $result, expected $expected)"
    fi
}

# URLs oficiales
check_url "https://ovav.dev" "Landing" "2xx"
check_url "https://docs.ovav.dev" "Docs" "2xx"
check_url "https://d678beea.ovav.dev/health" "Backend" "2xx"
check_url "https://status.ovav.dev" "Status" "2xx"

echo ""
echo "=== URLs que DEBEN fallar/redirigir ==="

check_url "https://get.ovav.dev" "Get (debe redirigir)" "redirect"
check_url "https://api.ovav.dev" "API (debe fallar)" "000"
check_url "https://mcp.ovav.dev" "MCP (debe fallar)" "000"
check_url "https://cdn.ovav.dev" "CDN (debe fallar)" "000"

echo ""
echo "=== Preview deployments ==="

check_url "https://develop.ovav-landing.pages.dev" "Develop preview" "000"
```

---

## Contacto de Emergencia

Si algo sale mal:
- **Producción anterior (backup):** Git revert a commit anterior
- **Fly.io rollback:** `fly deploy --image <previous-image>`
- **Cloudflare Pages rollback:** Deploy desde commit conocido en GitHub

---

**Última actualización:** 2026-08-07
**Responsable:** Thavren (Platform Engineering)
