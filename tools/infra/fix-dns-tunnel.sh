#!/bin/bash
# OVAV — DNS Tunnel Fix
# =============================================================================
# Problem: d678beea.ovav.dev y staging-a7k3m.ovav.dev no tienen registros DNS.
# El túnel cloudflared está conectado pero sin DNS → inaccesible.
#
# Este script crea los CNAME records faltantes y opcionalmente enruta
# d678beea.ovav.dev a través del túnel para defense-in-depth.
#
# REQUISITOS:
#   CF_API_TOKEN — Cloudflare API token con permisos DNS:Edit + Argo Tunnel:Edit
#   Opcional: CF_ACCOUNT_ID — se auto-detecta si no se especifica
#
# USO:
#   CF_API_TOKEN="<token>" ./fix-dns-tunnel.sh
#   CF_API_TOKEN="<token>" ./fix-dns-tunnel.sh --route-cpanel
# =============================================================================

set -euo pipefail

PROD_TUNNEL_CNAME="3b90d21c-ce98-41f9-883b-09980289560d.cfargotunnel.com"
STAG_TUNNEL_CNAME="c1780111-4a6a-431f-9813-10675645ada6.cfargotunnel.com"
PROD_TUNNEL_ID="3b90d21c-ce98-41f9-883b-09980289560d"
STAG_TUNNEL_ID="c1780111-4a6a-431f-9813-10675645ada6"
ZONE_NAME="ovav.dev"

ROUTE_CPANEL=false
for arg in "$@"; do
    case "$arg" in
        --route-cpanel) ROUTE_CPANEL=true ;;
    esac
done

# ── Validate token ────────────────────────────────────────────────────────────
if [ -z "${CF_API_TOKEN:-}" ]; then
    echo "❌ CF_API_TOKEN no definido."
    echo ""
    echo "   Creá un token en: https://dash.cloudflare.com/profile/api-tokens"
    echo "   Permisos: Zone:DNS:Edit + Account:Argo Tunnel:Edit"
    echo ""
    echo "   Luego: CF_API_TOKEN=\"tu-token\" $0"
    exit 1
fi

echo "🔧 OVAV DNS Tunnel Fix"
echo "======================"

# ── Auto-detect account ID ────────────────────────────────────────────────────
if [ -z "${CF_ACCOUNT_ID:-}" ]; then
    echo "→ Detectando Account ID..."
    CF_ACCOUNT_ID=$(curl -sf -H "Authorization: Bearer ${CF_API_TOKEN}" \
        "https://api.cloudflare.com/client/v4/accounts?page=1&per_page=1" \
        | python3 -c "import sys,json; print(json.load(sys.stdin)['result'][0]['id'])" 2>/dev/null)
    if [ -z "$CF_ACCOUNT_ID" ]; then
        echo "❌ No se pudo detectar Account ID. Especificá CF_ACCOUNT_ID."
        exit 1
    fi
fi
echo "   Account: ${CF_ACCOUNT_ID}"

# ── Get zone ID ───────────────────────────────────────────────────────────────
echo "→ Obteniendo Zone ID para ${ZONE_NAME}..."
ZONE_ID=$(curl -sf -H "Authorization: Bearer ${CF_API_TOKEN}" \
    "https://api.cloudflare.com/client/v4/zones?name=${ZONE_NAME}" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['result'][0]['id'])" 2>/dev/null)
if [ -z "$ZONE_ID" ]; then
    echo "❌ No se encontró la zona ${ZONE_NAME}."
    exit 1
fi
echo "   Zone: ${ZONE_ID}"

# ── Create DNS: d678beea.ovav.dev → prod tunnel ──────────────────────────────
echo ""
echo "→ Creando DNS: d678beea.ovav.dev → prod tunnel..."
RESP=$(curl -sf -X POST \
    -H "Authorization: Bearer ${CF_API_TOKEN}" \
    -H "Content-Type: application/json" \
    "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
    --data "{
        \"type\": \"CNAME\",
        \"name\": \"d678beea.ovav.dev\",
        \"content\": \"${PROD_TUNNEL_CNAME}\",
        \"ttl\": 1,
        \"proxied\": true
    }")
SUCCESS=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)
if [ "$SUCCESS" = "True" ]; then
    echo "   ✅ d678beea.ovav.dev → tunnel (proxied)"
else
    ERROR=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('errors',[{}])[0].get('message',''))" 2>/dev/null)
    if echo "$ERROR" | grep -qi "already exists"; then
        echo "   ⚠️  Ya existe (actualizando)..."
        # Get existing record ID and update
        REC_ID=$(curl -sf -H "Authorization: Bearer ${CF_API_TOKEN}" \
            "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records?type=CNAME&name=d678beea.ovav.dev" \
            | python3 -c "import sys,json; print(json.load(sys.stdin)['result'][0]['id'])" 2>/dev/null)
        if [ -n "$REC_ID" ]; then
            curl -sf -X PUT \
                -H "Authorization: Bearer ${CF_API_TOKEN}" \
                -H "Content-Type: application/json" \
                "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records/${REC_ID}" \
                --data "{\"type\":\"CNAME\",\"name\":\"d678beea.ovav.dev\",\"content\":\"${PROD_TUNNEL_CNAME}\",\"ttl\":1,\"proxied\":true}" \
                | python3 -c "import sys,json; print('   ✅ Actualizado' if json.load(sys.stdin).get('success') else '   ❌ Falló')"
        fi
    else
        echo "   ❌ Error: ${ERROR}"
    fi
fi

# ── Create DNS: staging-a7k3m.ovav.dev → staging tunnel ──────────────────────
echo "→ Creando DNS: staging-a7k3m.ovav.dev → staging tunnel..."
RESP=$(curl -sf -X POST \
    -H "Authorization: Bearer ${CF_API_TOKEN}" \
    -H "Content-Type: application/json" \
    "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
    --data "{
        \"type\": \"CNAME\",
        \"name\": \"staging-a7k3m.ovav.dev\",
        \"content\": \"${STAG_TUNNEL_CNAME}\",
        \"ttl\": 1,
        \"proxied\": true
    }")
SUCCESS=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)
if [ "$SUCCESS" = "True" ]; then
    echo "   ✅ staging-a7k3m.ovav.dev → staging tunnel (proxied)"
else
    ERROR=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('errors',[{}])[0].get('message',''))" 2>/dev/null)
    if echo "$ERROR" | grep -qi "already exists"; then
        echo "   ⚠️  Ya existe (actualizando)..."
        REC_ID=$(curl -sf -H "Authorization: Bearer ${CF_API_TOKEN}" \
            "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records?type=CNAME&name=staging-a7k3m.ovav.dev" \
            | python3 -c "import sys,json; print(json.load(sys.stdin)['result'][0]['id'])" 2>/dev/null)
        if [ -n "$REC_ID" ]; then
            curl -sf -X PUT \
                -H "Authorization: Bearer ${CF_API_TOKEN}" \
                -H "Content-Type: application/json" \
                "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records/${REC_ID}" \
                --data "{\"type\":\"CNAME\",\"name\":\"staging-a7k3m.ovav.dev\",\"content\":\"${STAG_TUNNEL_CNAME}\",\"ttl\":1,\"proxied\":true}" \
                | python3 -c "import sys,json; print('   ✅ Actualizado' if json.load(sys.stdin).get('success') else '   ❌ Falló')"
        fi
    else
        echo "   ❌ Error: ${ERROR}"
    fi
fi

# ── Optionally route d678beea.ovav.dev through tunnel ───────────────────────────
if $ROUTE_CPANEL; then
    echo ""
    echo "→ Enrutando d678beea.ovav.dev a través del túnel..."
    
    # Add d678beea.ovav.dev to tunnel ingress
    echo "   Actualizando tunnel ingress (prod)..."
    RESP=$(curl -sf -X PUT \
        -H "Authorization: Bearer ${CF_API_TOKEN}" \
        -H "Content-Type: application/json" \
        "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT_ID}/cfd_tunnel/${PROD_TUNNEL_ID}/configurations" \
        --data '{"config":{"ingress":[
            {"hostname":"d678beea.ovav.dev","service":"http://localhost:5858"},
            {"hostname":"d678beea.ovav.dev","service":"http://localhost:5858"},
            {"service":"http_status:404"}
        ]}}')
    SUCCESS=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)
    if [ "$SUCCESS" = "True" ]; then
        echo "   ✅ Ingress actualizado: d678beea + d678beea.ovav.dev"
    else
        echo "   ⚠️  Ingress update: $(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('errors',[{}])[0].get('message',''))" 2>/dev/null)"
    fi
    
    # Update DNS: d678beea.ovav.dev → tunnel
    echo "   Actualizando DNS: d678beea.ovav.dev → prod tunnel..."
    # Get existing record
    REC_ID=$(curl -sf -H "Authorization: Bearer ${CF_API_TOKEN}" \
        "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records?type=CNAME&name=d678beea.ovav.dev" \
        | python3 -c "import sys,json; r=json.load(sys.stdin)['result']; print(r[0]['id'] if r else '')" 2>/dev/null)
    
    if [ -n "$REC_ID" ]; then
        RESP=$(curl -sf -X PUT \
            -H "Authorization: Bearer ${CF_API_TOKEN}" \
            -H "Content-Type: application/json" \
            "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records/${REC_ID}" \
            --data "{\"type\":\"CNAME\",\"name\":\"d678beea.ovav.dev\",\"content\":\"${PROD_TUNNEL_CNAME}\",\"ttl\":1,\"proxied\":true}")
        SUCCESS=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)
        if [ "$SUCCESS" = "True" ]; then
            echo "   ✅ d678beea.ovav.dev → prod tunnel"
        else
            echo "   ❌ Error actualizando d678beea.ovav.dev"
        fi
    else
        # It might be an A record, create CNAME instead
        # First, find and delete any existing A/AAAA records for cpanel
        EXISTING=$(curl -sf -H "Authorization: Bearer ${CF_API_TOKEN}" \
            "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records?name=d678beea.ovav.dev" \
            | python3 -c "import sys,json; r=json.load(sys.stdin)['result']; [print(f'{x[\"id\"]}:{x[\"type\"]}') for x in r]" 2>/dev/null)
        
        if [ -n "$EXISTING" ]; then
            echo "   ⚠️  Encontrados registros existentes: $EXISTING"
            echo "   Para migrar d678beea.ovav.dev al túnel, eliminá primero los registros A existentes"
            echo "   y creá un CNAME manualmente en el dashboard."
        else
            RESP=$(curl -sf -X POST \
                -H "Authorization: Bearer ${CF_API_TOKEN}" \
                -H "Content-Type: application/json" \
                "https://api.cloudflare.com/client/v4/zones/${ZONE_ID}/dns_records" \
                --data "{\"type\":\"CNAME\",\"name\":\"d678beea.ovav.dev\",\"content\":\"${PROD_TUNNEL_CNAME}\",\"ttl\":1,\"proxied\":true}")
            SUCCESS=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('success', False))" 2>/dev/null)
            echo "   $( [ "$SUCCESS" = "True" ] && echo '✅ d678beea.ovav.dev → tunnel' || echo '❌ Error' )"
        fi
    fi
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "============================================"
echo "  ✅ DNS Tunnel Fix Aplicado"
echo "============================================"
echo ""
echo "  d678beea.ovav.dev      → prod tunnel"
echo "  staging-a7k3m.ovav.dev → staging tunnel"
if $ROUTE_CPANEL; then
    echo "  d678beea.ovav.dev        → prod tunnel"
fi
echo ""
echo "  ⏳ Propagación DNS: 1-5 minutos"
echo ""
echo "  ⚠️  PENDIENTE MANUAL: Cloudflare Access"
echo "     Ejecutar: CF_ACCESS_TOKEN=\"<token>\" ./tools/infra/setup-access.sh"
echo "     Protege d678beea.ovav.dev con Google OAuth (CEO-only)"
echo "============================================"
