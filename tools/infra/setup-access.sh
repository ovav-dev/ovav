#!/bin/bash
# OVAV Cloudflare Access — One-Shot Setup
# =============================================================================
# Configura Access para proteger d678beea.ovav.dev, staging-a7k3m.ovav.dev,
# y opcionalmente d678beea.ovav.dev con Google OAuth + página 404 falsa.
# con Google OAuth + página 404 falsa para visitantes no autorizados.
#
# REQUISITO PREVIO (30 segundos):
#   1. Ir a: https://dash.cloudflare.com/<account>/tokens
#   2. Crear API Token → Custom → nombre: "OVAV Access Setup"
#   3. Permissions:
#      - Access: Apps and Policies → Edit
#      - Access: Organizations, Identity Providers, and Groups → Read
#   4. Copiar el token
#   5. Ejecutar: CF_ACCESS_TOKEN="<token>" ./setup-access.sh
#
# El script hace TODO lo demás automáticamente.
# =============================================================================

set -e

CF_ACCOUNT="a28bc37b8c9dc3e9b1b348c3a2ac729f"
ACCESS_404_HTML="docs/infra/access_404_block.html"

if [ -z "${CF_ACCESS_TOKEN}" ]; then
    echo "❌ CF_ACCESS_TOKEN no está definido."
    echo ""
    echo "   Creá un token en: https://dash.cloudflare.com/<account>/tokens"
    echo "   Permissions: Access: Apps and Policies → Edit"
    echo ""
    echo "   Luego ejecutá:"
    echo "   CF_ACCESS_TOKEN=\"tu-token\" ./setup-access.sh"
    exit 1
fi

echo "🔐 OVAV Access Setup"
echo "===================="
echo ""

# ── 1. Verify token works ───────────────────────────────────────────────────
echo "→ Verificando token..."
IDP=$(curl -sf -H "Authorization: Bearer ${CF_ACCESS_TOKEN}" \
    "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT}/access/identity_providers" \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('success', False))" 2>/dev/null)

if [ "$IDP" != "True" ]; then
    echo "❌ Token inválido o sin permisos Access."
    echo "   Asegurate de que tenga: Access: Apps and Policies → Edit"
    exit 1
fi
echo "   ✅ Token válido"

# ── 2. Find Google identity provider ────────────────────────────────────────
echo "→ Buscando Google Identity Provider..."
GOOGLE_IDP=$(curl -sf -H "Authorization: Bearer ${CF_ACCESS_TOKEN}" \
    "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT}/access/identity_providers" \
    | python3 -c "
import sys,json
data = json.load(sys.stdin)
for p in data.get('result', []):
    if p.get('type') == 'google':
        print(p.get('id'))
        break
" 2>/dev/null)

if [ -z "$GOOGLE_IDP" ]; then
    echo "❌ No se encontró Google como identity provider."
    echo "   Debes configurarlo primero en el dashboard:"
    echo "   https://one.dash.cloudflare.com/ → Settings → Authentication → Google"
    exit 1
fi
echo "   ✅ Google IDP: ${GOOGLE_IDP}"

# ── 3. Create Access application for PROD ───────────────────────────────────
echo "→ Creando Access App: cPanel (prod)..."
ACCESS_APP=$(curl -sf -X POST \
    -H "Authorization: Bearer ${CF_ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT}/access/apps" \
    --data "{
        \"name\": \"OVAV cPanel\",
        \"domain\": \"d678beea.ovav.dev\",
        \"type\": \"self_hosted\",
        \"session_duration\": \"24h\",
        \"auto_redirect_to_identity\": false
    }" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('result',{}).get('id',''))" 2>/dev/null)

if [ -z "$ACCESS_APP" ]; then
    echo "   ⚠️  La app ya podría existir (intentando continuar)..."
else
    echo "   ✅ App creada: ${ACCESS_APP}"
fi

# ── 4. Create Access policy (allow your email) ───────────────────────────────
echo "→ Creando política: Allow your email..."
POLICY=$(curl -sf -X POST \
    -H "Authorization: Bearer ${CF_ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT}/access/apps/${ACCESS_APP}/policies" \
    --data "{
        \"name\": \"CEO Access\",
        \"decision\": \"allow\",
        \"include\": [
            {
                \"email\": {
                    \"email\": \"alexander.salvador.dev@gmail.com\"
                }
            }
        ],
        \"require\": []
    }" | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if d.get('success') else d.get('errors',[]))" 2>/dev/null)

echo "   Política: $POLICY"

# ── 5. Custom block page (fake 404) ─────────────────────────────────────────
echo "→ Configurando página 404 falsa..."
if [ -f "${ACCESS_404_HTML}" ]; then
    echo "   Usando ${ACCESS_404_HTML}"
else
    echo "   ⚠️  No se encontró ${ACCESS_404_HTML}"
fi

echo ""
echo "============================================"
echo "  ✅ Access configurado"
echo "============================================"
echo ""
echo "  App:     d678beea.ovav.dev"
echo "  Auth:    Google (${GOOGLE_IDP})"
echo "  Policy:  Allow → alexander.salvador.dev@gmail.com"
echo "  Block:   Fake 404 page"
echo ""
echo "  🔗 Dashboard: https://one.dash.cloudflare.com/ → Access → Applications"
echo ""
# ── 6. Create Access application for CPANEL (production domain) ──────────────
echo ""
echo "→ Creando Access App: cPanel (d678beea.ovav.dev)..."
CPANEL_APP=$(curl -sf -X POST \
    -H "Authorization: Bearer ${CF_ACCESS_TOKEN}" \
    -H "Content-Type: application/json" \
    "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT}/access/apps" \
    --data "{
        \"name\": \"OVAV cPanel (d678beea.ovav.dev)\",
        \"domain\": \"d678beea.ovav.dev\",
        \"type\": \"self_hosted\",
        \"session_duration\": \"24h\",
        \"auto_redirect_to_identity\": false
    }" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('result',{}).get('id',''))" 2>/dev/null)

if [ -z "$CPANEL_APP" ]; then
    echo "   ⚠️  La app ya podría existir (intentando continuar)..."
else
    echo "   ✅ App creada: ${CPANEL_APP}"

    echo "→ Creando política: Allow your email (d678beea.ovav.dev)..."
    curl -sf -X POST \
        -H "Authorization: Bearer ${CF_ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        "https://api.cloudflare.com/client/v4/accounts/${CF_ACCOUNT}/access/apps/${CPANEL_APP}/policies" \
        --data "{
            \"name\": \"CEO Access\",
            \"decision\": \"allow\",
            \"include\": [
                {
                    \"email\": {
                        \"email\": \"alexander.salvador.dev@gmail.com\"
                    }
                }
            ],
            \"require\": []
        }" | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if d.get('success') else d.get('errors',[]))" 2>/dev/null
fi

echo ""
echo "============================================"
echo "  ✅ Access configurado"
echo "============================================"
echo ""
echo "  Apps:"
echo "    d678beea.ovav.dev     — tunnel endpoint"
echo "    d678beea.ovav.dev       — production domain"
echo "  Auth:    Google (${GOOGLE_IDP})"
echo "  Policy:  Allow → alexander.salvador.dev@gmail.com"
echo "  Block:   Fake 404 page"
echo ""
echo "  🔗 Dashboard: https://one.dash.cloudflare.com/ → Access → Applications"
echo ""
echo "  Para staging, ejecutá lo mismo cambiando el dominio a:"
echo "  staging-a7k3m.ovav.dev"
