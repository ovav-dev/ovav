#!/bin/bash
# =============================================================================
# OVAV Web Watchdog v2 - Persistencia real con re-spawn y registro OVAV
# =============================================================================

PROJECT_DIR="/home/braka/Systems/OVAV"
BACKEND_DIR="$PROJECT_DIR/web/backend"
FRONTEND_DIR="$PROJECT_DIR/web/frontend"
PID_DIR="/tmp/ovav-dev-pids"
NERVE_BUS="$PROJECT_DIR/.ovav/runtime/nerve_bus.json"
PROCESS_REG="$PROJECT_DIR/.ovav/runtime/process_registry.json"

mkdir -p "$PID_DIR"

BACKEND_PID="$PID_DIR/backend.pid"
FRONTEND_PID="$PID_DIR/frontend.pid"

BACKEND_PORT=8000
FRONTEND_PORT=3000
BACKEND_LOG="/tmp/ovav-backend.log"
FRONTEND_LOG="/tmp/ovav-frontend.log"

# =============================================================================
# Registro en nerve_bus de OVAV
# =============================================================================
ovav_notify() {
    local event=$1
    local msg=$2
    echo "[$(date -Iseconds)] [$event] $msg" >> /tmp/ovav-nerve-notify.log
}

# =============================================================================
# Registro de proceso
# =============================================================================
register_process() {
    local name=$1
    local pid=$2
    local port=$3
    
    mkdir -p "$(dirname "$PROCESS_REG")"
    local ts=$(date -Iseconds)
    
    if [ -f "$PROCESS_REG" ]; then
        local tmp=$(mktemp)
        if command -v jq &>/dev/null; then
            jq --arg n "$name" --arg p "$pid" --arg pt "$port" --arg t "$ts" \
               '.processes[$n] = {pid: ($p|tonumber), port: ($pt|tonumber), registered: $t, status: "running"}' \
               "$PROCESS_REG" > "$tmp" && mv "$tmp" "$PROCESS_REG"
        fi
    else
        echo "{\"processes\": {\"$name\": {\"pid\": $pid, \"port\": $port, \"registered\": \"$ts\", \"status\": \"running\"}}}" > "$PROCESS_REG"
    fi
    
    ovav_notify "PROCESS_REGISTER" "$name PID=$pid PORT=$port"
}

# =============================================================================
# Verificar si proceso está vivo
# =============================================================================
is_alive() {
    local pid=$1
    [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}

# =============================================================================
# Iniciar Backend
# =============================================================================
start_backend() {
    # Verificar si ya está corriendo
    local current=$(cat "$BACKEND_PID" 2>/dev/null)
    if is_alive "$current"; then
        echo "[$(date +%H:%M:%S)] Backend ya corriendo: $current"
        return 0
    fi
    
    echo "[$(date +%H:%M:%S)] Iniciando Backend..."
    ovav_notify "PROCESS_START" "backend starting"
    
    # Crear wrapper que ejecuta en el directorio correcto
    cat > /tmp/ovav-backend-run.sh << 'EOF'
#!/bin/bash
cd /home/braka/Systems/OVAV/web/backend
exec python3 -m uvicorn app.main:app --host 127.0.0.1 --port 8000
EOF
    chmod +x /tmp/ovav-backend-run.sh
    
    # Iniciar con setsid para nueva sesión
    setsid bash /tmp/ovav-backend-run.sh </dev/null >> "$BACKEND_LOG" 2>&1 &
    local new_pid=$!
    
    # Guardar PID
    echo $new_pid > "$BACKEND_PID"
    
    # Esperar a que esté listo
    for i in {1..15}; do
        curl -s -o /dev/null http://127.0.0.1:$BACKEND_PORT/ 2>/dev/null && break
        sleep 1
    done
    
    register_process "backend" "$new_pid" "$BACKEND_PORT"
    echo "[$(date +%H:%M:%S)] Backend iniciado: $new_pid"
}

# =============================================================================
# Iniciar Frontend
# =============================================================================
start_frontend() {
    local current=$(cat "$FRONTEND_PID" 2>/dev/null)
    if is_alive "$current"; then
        echo "[$(date +%H:%M:%S)] Frontend ya corriendo: $current"
        return 0
    fi
    
    echo "[$(date +%H:%M:%S)] Iniciando Frontend..."
    ovav_notify "PROCESS_START" "frontend starting"
    
    # Limpiar cache
    rm -rf "$FRONTEND_DIR/.next" 2>/dev/null
    
    cat > /tmp/ovav-frontend-run.sh << 'EOF'
#!/bin/bash
cd /home/braka/Systems/OVAV/web/frontend
export NEXT_TELEMETRY_DISABLED=1
export NODE_ENV=development
exec pnpm dev --port 3000
EOF
    chmod +x /tmp/ovav-frontend-run.sh
    
    setsid bash /tmp/ovav-frontend-run.sh </dev/null >> "$FRONTEND_LOG" 2>&1 &
    local new_pid=$!
    
    echo $new_pid > "$FRONTEND_PID"
    
    for i in {1..30}; do
        curl -s -o /dev/null http://127.0.0.1:$FRONTEND_PORT/ 2>/dev/null && break
        [ $((i % 5)) -eq 0 ] && echo "[$(date +%H:%M:%S)] Esperando frontend... ($i/30)"
        sleep 1
    done
    
    register_process "frontend" "$new_pid" "$FRONTEND_PORT"
    echo "[$(date +%H:%M:%S)] Frontend iniciado: $new_pid"
}

# =============================================================================
# Detener todo
# =============================================================================
stop_all() {
    echo "[$(date +%H:%M:%S)] Deteniendo servicios..."
    
    [ -f "$BACKEND_PID" ] && kill -9 $(cat "$BACKEND_PID") 2>/dev/null && rm -f "$BACKEND_PID"
    [ -f "$FRONTEND_PID" ] && kill -9 $(cat "$FRONTEND_PID") 2>/dev/null && rm -f "$FRONTEND_PID"
    
    pkill -9 -f "uvicorn.*8000" 2>/dev/null || true
    pkill -9 -f "next-server" 2>/dev/null || true
    
    ovav_notify "PROCESS_STOP" "all services stopped"
    echo "[$(date +%H:%M:%S)] Servicios detenidos"
}

# =============================================================================
# Status
# =============================================================================
status() {
    echo ""
    echo "=== OVAV Web Status ==="
    echo ""
    
    local bk_http=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$BACKEND_PORT/ 2>/dev/null || echo "000")
    local fe_http=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:$FRONTEND_PORT/ 2>/dev/null || echo "000")
    
    [ "$bk_http" = "200" ] && echo "🔌 Backend :$BACKEND_PORT  ✓ $(cat $BACKEND_PID 2>/dev/null || echo '?')" || echo "🔌 Backend :$BACKEND_PORT  ✗ ($bk_http)"
    [ "$fe_http" = "200" ] && echo "🌐 Frontend :$FRONTEND_PORT ✓ $(cat $FRONTEND_PID 2>/dev/null || echo '?')" || echo "🌐 Frontend :$FRONTEND_PORT ✗ ($fe_http)"
    
    echo ""
}

# =============================================================================
# WATCHDOG LOOP - Mantener vivos con re-spawn
# =============================================================================
watchdog_loop() {
    echo "[$(date +%H:%M:%S)] OVAV Watchdog iniciado - modo perpetuo"
    ovav_notify "WATCHDOG_START" "process watchdog active"
    
    while true; do
        # Check Backend
        if ! is_alive "$(cat "$BACKEND_PID" 2>/dev/null)"; then
            echo "[$(date +%H:%M:%S)] ⚠ Backend muerto, reiniciando..."
            ovav_notify "PROCESS_RESTART" "backend restarting"
            start_backend
        fi
        
        # Check Frontend
        if ! is_alive "$(cat "$FRONTEND_PID" 2>/dev/null)"; then
            echo "[$(date +%H:%M:%S)] ⚠ Frontend muerto, reiniciando..."
            ovav_notify "PROCESS_RESTART" "frontend restarting"
            start_frontend
        fi
        
        sleep 10
    done
}

# =============================================================================
# MAIN
# =============================================================================
case "${1:-status}" in
    start|dev)
        stop_all 2>/dev/null
        start_backend
        start_frontend
        echo ""
        echo "✅ OVAV Web corriendo"
        echo "🌐 Frontend: http://localhost:$FRONTEND_PORT"
        echo "🔌 Backend:  http://localhost:$BACKEND_PORT"
        echo ""
        echo "Ejecuta './bin/ovav-watchdog.sh watchdog' para iniciar watchdog perpetuo"
        ;;
    watchdog)
        start_backend
        start_frontend
        watchdog_loop
        ;;
    stop)
        stop_all
        ;;
    status)
        status
        ;;
    restart)
        stop_all
        start_backend
        start_frontend
        ;;
    *)
        echo "Uso: $0 {start|stop|status|restart|watchdog}"
        ;;
esac
