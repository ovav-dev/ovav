#!/bin/bash
# =============================================================================
# OVAV Web Full-Stack Development Launcher
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$PROJECT_DIR/web/backend"
FRONTEND_DIR="$PROJECT_DIR/web/frontend"
PID_DIR="/tmp/ovav-dev-pids"

mkdir -p "$PID_DIR"
BACKEND_PID="$PID_DIR/backend.pid"
FRONTEND_PID="$PID_DIR/frontend.pid"

BACKEND_PORT=8080
FRONTEND_PORT=3000
BACKEND_LOG="/tmp/ovav-backend.log"
FRONTEND_LOG="/tmp/ovav-frontend.log"

# Colors
C='\033[0;36m'
G='\033[0;32m'
R='\033[0;31m'
Y='\033[1;33m'
N='\033[0m'

log()  { echo -e "${G}[OVAV]${N} $1"; }
info() { echo -e "${C}[INFO]${N} $1"; }
warn() { echo -e "${Y}[WARN]${N} $1"; }
err()  { echo -e "${R}[ERROR]${N} $1"; }

# Stop services
do_stop() {
    log "Stopping services..."
    [ -f "$BACKEND_PID" ] && kill -9 $(cat "$BACKEND_PID") 2>/dev/null && rm -f "$BACKEND_PID"
    pkill -9 -f "uvicorn.*$BACKEND_PORT" 2>/dev/null || true
    [ -f "$FRONTEND_PID" ] && kill -9 $(cat "$FRONTEND_PID") 2>/dev/null && rm -f "$FRONTEND_PID"
    pkill -9 -f "next-server" 2>/dev/null || true
    echo "✅"
}

# Status check
do_status() {
    echo ""
    echo -e "${C}═══════════════════════════════════════════${N}"
    echo -e "${C}         OVAV Web Full-Stack Status${N}"
    echo -e "${C}═══════════════════════════════════════════${N}"
    echo ""
    
    local bk=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$BACKEND_PORT/ 2>/dev/null || echo "000")
    local fe=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$FRONTEND_PORT/ 2>/dev/null || echo "000")
    
    [ "$bk" = "200" ] && echo -e "  🔌 Backend  :$BACKEND_PORT  ${G}✓ HTTP $bk${N}" || echo -e "  🔌 Backend  :$BACKEND_PORT  ${R}✗ HTTP $bk${N}"
    [ "$fe" = "200" ] && echo -e "  🌐 Frontend :$FRONTEND_PORT ${G}✓ HTTP $fe${N}" || echo -e "  🌐 Frontend :$FRONTEND_PORT ${R}✗ HTTP $fe${N}"
    
    echo ""
    echo -e "${C}═══════════════════════════════════════════${N}"
    echo ""
}

# View logs
do_logs() {
    echo -e "${C}--- Backend Log ---${N}"
    tail -30 "$BACKEND_LOG" 2>/dev/null || echo "No log"
    echo ""
    echo -e "${C}--- Frontend Log ---${N}"
    tail -30 "$FRONTEND_LOG" 2>/dev/null || echo "No log"
}

# Clean and stop
do_clean() {
    do_stop
    rm -rf "$FRONTEND_DIR/.next" 2>/dev/null || true
    log "Clean complete!"
}

# Start all services
do_start() {
    log "🚀 Starting OVAV Web Full-Stack..."
    do_stop
    echo ""
    
    # Backend
    log "🔌 Starting Backend (FastAPI)..."
    cat > /tmp/ovav-backend-start.sh << 'EOF'
#!/bin/bash
cd /home/braka/Systems/OVAV/web/backend
cd /home/braka/Systems/OVAV/web/backend && exec uvicorn app.main:app --host 127.0.0.1 --port 8080
EOF
    chmod +x /tmp/ovav-backend-start.sh
    setsid bash /tmp/ovav-backend-start.sh >> "$BACKEND_LOG" 2>&1 &
    echo $! > "$BACKEND_PID"
    
    for i in {1..15}; do
        curl -s -o /dev/null http://127.0.0.1:$BACKEND_PORT/ 2>/dev/null && break
        sleep 1
    done
    echo -e "   ${G}✓ Backend ready (PID: $(cat $BACKEND_PID))${N}"
    
    # Frontend
    echo ""
    log "🌐 Starting Frontend (Next.js)..."
    rm -rf "$FRONTEND_DIR/.next"
    
    cat > /tmp/ovav-frontend-start.sh << 'EOF'
#!/bin/bash
cd /home/braka/Systems/OVAV/web/frontend
export NEXT_TELEMETRY_DISABLED=1
exec pnpm dev --port 3000
EOF
    chmod +x /tmp/ovav-frontend-start.sh
    setsid bash /tmp/ovav-frontend-start.sh >> "$FRONTEND_LOG" 2>&1 &
    echo $! > "$FRONTEND_PID"
    
    for i in {1..30}; do
        curl -s -o /dev/null http://127.0.0.1:$FRONTEND_PORT/ 2>/dev/null && break
        [ $((i % 5)) -eq 0 ] && echo "   Waiting... ($i/30)"
        sleep 1
    done
    echo -e "   ${G}✓ Frontend ready (PID: $(cat $FRONTEND_PID))${N}"
    
    echo ""
    echo -e "${G}═══════════════════════════════════════════${N}"
    echo -e "${G}  ✅ OVAV Web Started!${N}"
    echo -e "${G}═══════════════════════════════════════════${N}"
    echo ""
    echo -e "  🌐 Frontend: ${C}http://localhost:$FRONTEND_PORT${N}"
    echo -e "  🔌 Backend:  ${C}http://localhost:$BACKEND_PORT${N}"
    echo ""
    do_status
}

# Help
do_help() {
    echo ""
    echo -e "${C}OVAV Web Full-Stack Commands${N}"
    echo ""
    echo "  ./bin/pr dev      - Start all services"
    echo "  ./bin/pr stop     - Stop services"
    echo "  ./bin/pr status   - Check status"
    echo "  ./bin/pr clean    - Clean caches"
    echo "  ./bin/pr logs     - View logs"
    echo "  ./bin/pr help     - Show this help"
    echo ""
    echo "  pnpm run dev      - Start all (via pnpm)"
    echo "  pnpm run status   - Check status"
    echo ""
}

# Main
case "${1:-dev}" in
    dev|start)     do_start ;;
    stop)          do_stop ;;
    status)        do_status ;;
    clean)         do_clean ;;
    logs)          do_logs ;;
    help|-h|--help) do_help ;;
    *)             err "Unknown: $1" && do_help && exit 1 ;;
esac
