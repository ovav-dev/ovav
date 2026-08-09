#!/usr/bin/env fish
# ═══════════════════════════════════════════════════════════════════════
# OVAV SYSTEMS — Auto-Lock Gate v1.0
# ═══════════════════════════════════════════════════════════════════════
# Se ejecuta automáticamente al entrar al directorio OVAV.
# Si no hay sesión activa de OVAV Systems, bloquea el acceso y fuerza login.
#
# Instalación: symlink a ~/.config/fish/conf.d/99-ovav-systems-lock.fish
# ═══════════════════════════════════════════════════════════════════════

# ── Configuración ─────────────────────────────────────────────────────────

# Directorio raíz de OVAV Systems (ajustar si cambia)
set -g OVAV_SYSTEMS_ROOT "/home/braka/Systems/OVAV"

# Tiempo máximo de sesión antes de forzar re-login (segundos)
set -g OVAV_SESSION_MAX_AGE 86400  # 24 horas

# Archivo de sesión
set -g OVAV_SESSION_FILE "$HOME/.local/share/ovav/session"

# ── Vault Key + Seed Auto-Load ─────────────────────────────────────────
# ovav login writes vault key AND seed to secure temp files (0600).
# Vault key: local vault unlock
# Seed: cross-device sync (sync engine needs seed to derive SyncKey)
# Hook loads both automatically — no manual commands needed.

set -g OVAV_VAULT_KEY_FILE "$HOME/.local/share/ovav/vault_key_export"
set -g OVAV_SEED_FILE "$HOME/.local/share/ovav/seed_export"

function __ovav_vault_key_load
    if not test -f "$OVAV_VAULT_KEY_FILE"
        return 1
    end
    # Ya cargada y no ha cambiado
    if set -q OVAV_VAULT_KEY_LOADED
        return 0
    end
    set -l key (cat "$OVAV_VAULT_KEY_FILE" 2>/dev/null | string trim)
    if test -n "$key"
        set -gx OVAV_VAULT_KEY "$key"
        set -g OVAV_VAULT_KEY_LOADED 1
        rm -f "$OVAV_VAULT_KEY_FILE"
        return 0
    end
    return 1
end

function __ovav_seed_load
    if not test -f "$OVAV_SEED_FILE"
        return 1
    end
    # Ya cargada
    if set -q OVAV_SEED_LOADED
        return 0
    end
    set -l seed (cat "$OVAV_SEED_FILE" 2>/dev/null | string trim)
    if test -n "$seed"
        set -gx OVAV_SEED "$seed"
        set -g OVAV_SEED_LOADED 1
        rm -f "$OVAV_SEED_FILE"
        return 0
    end
    return 1
end

function __ovav_auth_load
    __ovav_vault_key_load
    __ovav_seed_load
end

# ── Verificación de sesión ─────────────────────────────────────────────────

function __ovav_session_active
    if not test -f "$OVAV_SESSION_FILE"
        return 1
    end

    # Verificar que la sesión no expiró
    set -l session_age 0
    if command -q python3
        set session_age (python3 -c "
import json, time, os
try:
    with open(os.path.expanduser('$OVAV_SESSION_FILE')) as f:
        s = json.load(f)
    created = time.mktime(time.strptime(s['created_at'][:19], '%Y-%m-%dT%H:%M:%S'))
    print(int(time.time() - created))
except:
    print(999999)
" 2>/dev/null)
    else if command -q date
        # Fallback: ver por fecha de modificación del archivo
        set -l mtime (stat -c %Y "$OVAV_SESSION_FILE" 2>/dev/null)
        if test -n "$mtime"
            set session_age (math (date +%s) - $mtime)
        end
    end

    if test "$session_age" -gt "$OVAV_SESSION_MAX_AGE"
        return 1
    end

    return 0
end

# ── Auto-lock en cd ───────────────────────────────────────────────────────

function __ovav_directory_lock --on-variable PWD
    # Solo actuar si estamos dentro del directorio OVAV
    if not string match -q "$OVAV_SYSTEMS_ROOT*" "$PWD"
        return
    end

    # Si ya estamos en proceso de login, no interrumpir
    if set -q OVAV_LOGIN_IN_PROGRESS
        return
    end

    # Auto-cargar vault key + seed si existen
    __ovav_auth_load

    # Verificar sesión activa
    if __ovav_session_active
        # Sesión activa — verificar antigüedad y advertir si está por expirar
        set -l session_age 0
        if test -f "$OVAV_SESSION_FILE"
            set -l mtime (stat -c %Y "$OVAV_SESSION_FILE" 2>/dev/null)
            if test -n "$mtime"
                set session_age (math (date +%s) - $mtime)
            end
        end

        set -l remaining (math $OVAV_SESSION_MAX_AGE - $session_age)
        if test "$remaining" -lt 3600  # menos de 1 hora
            set -l minutes (math floor $remaining / 60)
            echo "⏰  OVAV session expires in {$minutes}min — run 'ovav login' to renew"
        end
        return
    end

    # ── SIN SESIÓN: BLOQUEAR ───────────────────────────────────────────
    echo ""
    echo "🔒  OVAV Systems — Sealed Governor"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  This is a locked development workstation."
    echo "  Authentication required to proceed."
    echo ""
    echo "  Unlock: ovav login"
    echo ""

    # Bloquear: regresar al home
    # Comentado por defecto — descomentar para bloqueo físico total
    # cd $HOME
end

# ── Vault key auto-load en cada prompt ────────────────────────────────────

function __ovav_vault_prompt_load --on-event fish_prompt
    if not string match -q "$OVAV_SYSTEMS_ROOT*" "$PWD"
        return
    end
    __ovav_auth_load
end

# ── Bloqueo en nueva terminal ─────────────────────────────────────────────

function __ovav_startup_lock --on-event fish_prompt
    # Solo ejecutar una vez por sesión
    if set -q OVAV_STARTUP_CHECK_DONE
        return
    end
    set -g OVAV_STARTUP_CHECK_DONE 1

    # Si no estamos en OVAV, no hacer nada
    if not string match -q "$OVAV_SYSTEMS_ROOT*" "$PWD"
        return
    end

    # Auto-cargar vault key + seed al iniciar terminal
    __ovav_auth_load

    # Si no hay sesión, bloquear inmediatamente
    if not __ovav_session_active
        echo ""
        echo "🔒  OVAV Systems — Authentication Required"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  A new terminal opened inside OVAV Systems."
        echo "  This workspace is locked."
        echo ""
        echo "  Authenticate: ovav login"
        echo ""
    end
end

# ── Limpieza automática de sesión expirada ─────────────────────────────────

function __ovav_auto_logout --on-event fish_exit
    if not __ovav_session_active
        return
    end

    # Limpiar OVAV_VAULT_KEY del entorno
    if set -q OVAV_VAULT_KEY
        set -ge OVAV_VAULT_KEY
    end
end
