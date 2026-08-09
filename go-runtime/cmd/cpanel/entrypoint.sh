#!/bin/sh
# OVAV cPanel — Cloudflared Tunnel Entrypoint v5.0
# =============================================================================
# 1. Start cloudflared FIRST (--token nativo, sin Python, sin config files)
# 2. Start cPanel (Fly.io health check)
# =============================================================================
# Fix v5.0 (2026-06-17):
#   - cloudflared --token flag: nativo, elimina dependencia de Python
#   - Python NO instalado en alpine:3.20 → v4.0 fallaba silenciosamente
#   - Orden: tunnel primero (background), cPanel después (foreground)
#   - Sin archivos JSON/YAML, sin base64 decode manual

set -e

echo "🔐 OVAV cPanel — Tunnel Entrypoint v5.0"
echo "============================================"

# ── 1. Start cloudflared tunnel FIRST ──────────────────────────────────────────
if [ -n "${TUNNEL_TOKEN}" ]; then
    echo "→ Starting cloudflared tunnel (--token mode)..."
    /usr/local/bin/cloudflared tunnel --no-autoupdate run \
        --token "${TUNNEL_TOKEN}" \
        > /tmp/cloudflared.log 2>&1 &
    CLOUDFLARED_PID=$!
    echo "   cloudflared PID: ${CLOUDFLARED_PID}"

    # Brief wait to catch immediate failures
    sleep 3
    if kill -0 ${CLOUDFLARED_PID} 2>/dev/null; then
        echo "   Tunnel process alive ✅"
    else
        echo "   ⚠️  Tunnel failed to start. Check /tmp/cloudflared.log"
        CLOUDFLARED_PID=""
    fi
else
    echo "⚠️  TUNNEL_TOKEN not set (dev/local mode — no tunnel)"
    CLOUDFLARED_PID=""
fi

# ── 2. Start cPanel ───────────────────────────────────────────────────────────
echo "→ Starting cPanel on ${OVAV_LISTEN_ADDR:-0.0.0.0}:${PORT:-5858}..."
/app/cpanel &
CPANEL_PID=$!
echo "   cPanel PID: ${CPANEL_PID}"
sleep 1

echo "   ✅ cPanel ready on :${PORT:-5858}"
echo "============================================"

# ── 3. Signal handling ───────────────────────────────────────────────────────
cleanup() {
    echo "→ Shutting down..."
    kill -TERM ${CPANEL_PID} 2>/dev/null || true
    [ -n "${CLOUDFLARED_PID}" ] && kill -TERM ${CLOUDFLARED_PID} 2>/dev/null || true
    wait ${CPANEL_PID} 2>/dev/null || true
    echo "✓ Shutdown complete"
}
trap cleanup INT TERM
wait ${CPANEL_PID}
