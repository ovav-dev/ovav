#!/bin/bash
# =============================================================================
# OVAV Process Manager - Usa systemd para persistencia real
# =============================================================================

PROJECT_DIR="/home/braka/Systems/OVAV"
PID_DIR="/tmp/ovav-dev-pids"
REGISTRY="$PROJECT_DIR/.ovav/runtime/process_registry.json"

mkdir -p "$PID_DIR"

# Colors
C='\033[0;36m'
G='\033[0;32m'
R='\033[0;31m'
Y='\033[1;33m'
N='\033[0m'

log()  { echo -e "${G}[OVAV]${N} $1"; }
info() { echo -e "${C}[INFO]${N} $1"; }
err()  { echo -e "${R}[ERROR]${N} $1"; }

# =============================================================================
# Register en OVAV runtime
# =============================================================================
register_ovav() {
    local name=$1
    local pid=$2
    local port=$3
    
    mkdir -p "$(dirname "$REGISTRY")"
    local ts=$(date -Iseconds)
    
    if [ -f "$REGISTRY" ]; then
        local tmp=$(mktemp)
        if command -v jq &>/dev/null; then
            jq --arg n "$name" --arg p "$pid" --arg pt "$port" \
               '.processes[$n] = {pid: ($p|tonumber), port: ($pt|tonumber), registered: "'$ts'", status: "running"}' \
               "$REGISTRY" > "$tmp" && mv "$tmp" "$REGISTRY"
        fi
    else
        echo "{\"processes\": {\"$name\": {\"pid\": $pid, \"port\": $port, \"registered\": \"$ts\", \"status\": \"running\"}}}" > "$REGISTRY"
    fi
}

# =============================================================================
# Start con systemd
# =============================================================================
systemd_start() {
    info "Starting via systemd user service..."
    
    # Instalar servicio si no existe
    local service_file="$HOME/.config/systemd/user/ovav-web.service"
    mkdir -p "$HOME/.config/systemd/user"
    
    cat > "$service_file" << 'EOF'
[Unit]
Description=OVAV Web Development Stack
After=network.target

[Service]
Type=simple
WorkingDirectory=/home/braka/Systems/OVAV
ExecStart=/bin/bash /home/braka/Systems/OVAV/bin/ovav-watchdog.sh start
ExecStop=/bin/bash /home/braka/Systems/OVAV/bin/ovav-watchdog.sh stop
Restart=on-failure
RestartSec=5
Environment=HOME=/home/braka
Environment=USER=braka
StandardOutput=append:/tmp/ovav-web-stdout.log
StandardError=append:/tmp/ovav-web-stderr.log

[Install]
WantedBy=default.target
EOF
    
    # Recargar systemd
    systemctl --user daemon-reload 2>/dev/null || true
    
    # Iniciar servicio
    systemctl --user start ovav-web 2>/dev/null && \
        log "✅ Started via systemd" || \
        err "Failed to start systemd service"
}

# =============================================================================
# Status
# =============================================================================
do_status() {
    echo ""
    echo -e "${C}═══════════════════════════════════════════${N}"
    echo -e "${C}         OVAV Web Status${N}"
    echo -e "${C}═══════════════════════════════════════════${N}"
    echo ""
    
    # Check processes
    local bk_http fe_http
    bk_http=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8000/ 2>/dev/null || echo "000")
    fe_http=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:3000/ 2>/dev/null || echo "000")
    
    [ "$bk_http" = "200" ] && echo -e "  🔌 Backend :8000  ${G}✓ HTTP $bk_http${N}" || echo -e "  🔌 Backend :8000  ${R}✗ HTTP $bk_http${N}"
    [ "$fe_http" = "200" ] && echo -e "  🌐 Frontend :3000 ${G}✓ HTTP $fe_http${N}" || echo -e "  🌐 Frontend :3000 ${R}✗ HTTP $fe_http${N}"
    
    echo ""
    
    # Systemd status
    if systemctl --user is-active ovav-web &>/dev/null; then
        echo -e "  📦 Systemd: ${G}active${N}"
    else
        echo -e "  📦 Systemd: ${Y}inactive${N}"
    fi
    
    # OVAV Registry
    if [ -f "$REGISTRY" ]; then
        echo -e "  📋 Registry: ${G}exists${N}"
        cat "$REGISTRY" | head -c 200
        echo ""
    else
        echo -e "  📋 Registry: ${Y}not found${N}"
    fi
    
    echo ""
}

# =============================================================================
# Stop
# =============================================================================
do_stop() {
    log "Stopping services..."
    
    # Stop systemd
    systemctl --user stop ovav-web 2>/dev/null || true
    
    # Kill processes
    pkill -9 -f "uvicorn.*8000" 2>/dev/null || true
    pkill -9 -f "next-server" 2>/dev/null || true
    
    # Clean PIDs
    rm -f "$PID_DIR"/*.pid
    
    log "✅ Stopped"
}

# =============================================================================
# Main
# =============================================================================
case "${1:-status}" in
    start|dev)
        systemd_start
        sleep 2
        do_status
        ;;
    stop)
        do_stop
        ;;
    status)
        do_status
        ;;
    *)
        echo "Usage: $0 {start|stop|status}"
        ;;
esac
