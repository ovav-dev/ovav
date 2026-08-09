#!/bin/bash
# OVAV Services Manager - Usa systemd user services

case "${1:-status}" in
    start)
        systemctl --user start ovav-backend.service ovav-frontend.service
        echo "✅ Servicios iniciados"
        ;;
    stop)
        systemctl --user stop ovav-frontend.service ovav-backend.service
        echo "✅ Servicios detenidos"
        ;;
    restart)
        systemctl --user restart ovav-backend.service ovav-frontend.service
        echo "✅ Servicios reiniciados"
        ;;
    status)
        echo ""
        systemctl --user status ovav-backend.service 2>/dev/null | head -5
        echo ""
        systemctl --user status ovav-frontend.service 2>/dev/null | head -5
        echo ""
        curl -s -o /dev/null -w "Backend (8080): %{http_code}\n" http://localhost:8080/ 2>/dev/null
        curl -s -o /dev/null -w "Frontend (3000): %{http_code}\n" http://localhost:3000/ 2>/dev/null
        ;;
    logs)
        echo "=== Backend Log ==="
        tail -20 /tmp/ovav-backend-svc.log 2>/dev/null
        echo ""
        echo "=== Frontend Log ==="
        tail -20 /tmp/ovav-frontend-svc.log 2>/dev/null
        ;;
    *)
        echo "Uso: $0 {start|stop|restart|status|logs}"
        ;;
esac
