#!/bin/bash
# OVAV Web Full-Stack Service Manager
# Usage: ./web/start.sh {start|stop|restart|status|logs}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-status}" in
    start|dev)
        echo "🔌 Starting Backend..."
        systemctl --user start ovav-backend
        echo "🌐 Starting Frontend..."
        systemctl --user start ovav-frontend
        sleep 2
        echo ""
        echo "✅ Services started!"
        echo "   Frontend: http://localhost:3000"
        echo "   Backend:  http://localhost:8080"
        ;;
    stop)
        systemctl --user stop ovav-frontend ovav-backend
        echo "✅ Stopped"
        ;;
    restart)
        systemctl --user restart ovav-backend ovav-frontend
        echo "✅ Restarted"
        ;;
    status)
        echo ""
        echo "=== OVAV Web Status ==="
        echo ""
        systemctl --user status ovav-backend --no-pager 2>/dev/null | head -5
        echo ""
        systemctl --user status ovav-frontend --no-pager 2>/dev/null | head -5
        echo ""
        curl -s -o /dev/null -w "🔌 Backend :8080  %{http_code}\n" http://localhost:8080/ 2>/dev/null
        curl -s -o /dev/null -w "🌐 Frontend :3000 %{http_code}\n" http://localhost:3000/ 2>/dev/null
        ;;
    logs)
        echo "=== Backend Logs ==="
        journalctl --user -u ovav-backend -n 20 --no-pager
        echo ""
        echo "=== Frontend Logs ==="
        journalctl --user -u ovav-frontend -n 20 --no-pager
        ;;
    *)
        echo "Usage: ./web/start.sh {start|stop|restart|status|logs}"
        ;;
esac
