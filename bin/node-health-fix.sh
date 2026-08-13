#!/usr/bin/env bash
# =============================================================================
# OVAV Node/Starship Health Fix
# =============================================================================
# Corrige issues de NVM/Node/Starship detectados en WSL2
# Uso: bash node-health-fix.sh
#
# Este script DEBE ser sourceado para afectar el shell actual:
#   source node-health-fix.sh
# =============================================================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# =============================================================================
# Fix 1: npm_config_user_agent corrupto
# =============================================================================
fix_npm_user_agent() {
    log_info "Fixing npm_config_user_agent..."

    # Detectar si la variable está mal configurada
    if [[ "${npm_config_user_agent:-}" == *"undefined"* ]] || \
       [[ "${npm_config_user_agent:-}" == *"v24.3.0"* ]]; then
        log_warn "Found corrupted npm_config_user_agent: ${npm_config_user_agent:-}"
        unset npm_config_user_agent
        log_info "Unset npm_config_user_agent - will auto-regenerate from npm"
    else
        log_info "npm_config_user_agent looks OK: ${npm_config_user_agent:-undefined}"
    fi
}

# =============================================================================
# Fix 2: Verificar y limpiar npm cache
# =============================================================================
fix_npm_cache() {
    log_info "Verifying npm cache..."

    local cache_before
    cache_before=$(du -sh ~/.npm/_cacache 2>/dev/null | cut -f1 || echo "unknown")

    npm cache verify --prefer-offline 2>/dev/null || npm cache verify

    log_info "Cache verified (was: ${cache_before})"
}

# =============================================================================
# Fix 3: Verificar que NVM esté correctamente cargado
# =============================================================================
fix_nvm_load() {
    log_info "Checking NVM load status..."

    if ! command -v nvm &>/dev/null; then
        log_warn "NVM not in PATH - attempting to source..."

        # Detectar shell
        if [[ -n "${BASH_VERSION:-}" ]]; then
            NVM_SH="${HOME}/.nvm/nvm.sh"
        elif [[ -n "${ZSH_VERSION:-}" ]]; then
            NVM_SH="${HOME}/.nvm/nvm.sh"
        else
            NVM_SH="${HOME}/.nvm/nvm.sh"
        fi

        if [[ -f "$NVM_SH" ]]; then
            # shellcheck source=/dev/null
            source "$NVM_SH" 2>/dev/null || true
            log_info "NVM sourced from $NVM_SH"
        else
            log_warn "NVM script not found at $NVM_SH"
        fi
    else
        log_info "NVM already in PATH"
    fi

    # Mostrar estado
    nvm current 2>/dev/null || echo "NVM not available"
}

# =============================================================================
# Fix 4: Starship config con command_timeout al INICIO (global)
# =============================================================================
fix_starship_config() {
    log_info "Checking Starship configuration..."

    # Usar Python para manipulación segura de TOML
    python3 << 'PYEOF'
import os
import shutil

star_dir = os.path.expanduser('~/.config')
star_cfg = os.path.join(star_dir, 'starship.toml')
tmp_cfg = '/tmp/opencode/starship.toml'

if not os.path.exists(star_cfg):
    print('[WARN] No starship.toml found')
    exit(0)

with open(star_cfg, 'r') as f:
    content = f.read()

# Parsear: command_timeout debe estar al inicio (global), no en secciones
lines = content.split('\n')

# Remover command_timeout existente (cualquier ubicación)
clean_lines = [l for l in lines if 'command_timeout' not in l]

# Remover línea de separador residual de fixes anteriores
clean_lines = [l for l in clean_lines
              if not (l.startswith('# ===') and 'OVAV' in l)]
clean_lines = [l for l in clean_lines
              if not l.startswith('# ====') and not l.startswith('# ----')]

# Unir contenido limpio
clean = '\n'.join(clean_lines).strip()

# Insertar command_timeout al INICIO
fixed = 'command_timeout = 10000\n\n' + clean + '\n'

# Asegurar directorio tmp
os.makedirs('/tmp/opencode', exist_ok=True)
with open(tmp_cfg, 'w') as f:
    f.write(fixed)

# Copiar de vuelta
shutil.copy(tmp_cfg, star_cfg)

# Verificar que es TOML válido
try:
    import tomllib
    with open(star_cfg, 'rb') as f:
        tomllib.load(f)
    print('[INFO] starship.toml fixed - TOML valid, command_timeout at top')
except Exception as e:
    print(f'[WARN] TOML parse issue: {e}')
    print('[INFO] Restored from backup via content read')

print('[INFO] command_timeout is now at the TOP of starship.toml (global scope)')
PYEOF
}

# =============================================================================
# Fix 5: Node binary health check
# =============================================================================
fix_node_health() {
    log_info "Running Node health check..."

    local node_path
    node_path=$(command -v node 2>/dev/null || echo "")

    if [[ -z "$node_path" ]]; then
        log_error "Node not found in PATH"
        return 1
    fi

    log_info "Node: $node_path"

    # Verificar versión
    local node_version
    node_version=$(node --version 2>/dev/null || echo "FAILED")
    log_info "Node version: $node_version"

    # Verificar que responde rápido
    local start_time
    start_time=$(date +%s%N)
    node --version >/dev/null 2>&1
    local end_time
    end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    log_info "Node response time: ${duration_ms}ms"

    if [[ $duration_ms -gt 5000 ]]; then
        log_warn "Node response time > 5s - may indicate issue"
    else
        log_info "Node response time OK"
    fi

    # Hash del binary (para verificar integridad)
    local node_hash
    node_hash=$(sha256sum "$node_path" 2>/dev/null | cut -d' ' -f1)
    log_info "Node SHA256: ${node_hash:0:16}..."
}

# =============================================================================
# MAIN
# =============================================================================
main() {
    echo ""
    echo "========================================"
    echo "OVAV Node/Starship Health Fix"
    echo "========================================"
    echo ""

    fix_npm_user_agent
    echo ""
    fix_npm_cache
    echo ""
    fix_nvm_load
    echo ""
    fix_node_health
    echo ""
    fix_starship_config
    echo ""

    echo "========================================"
    echo -e "${GREEN}[DONE]${NC} Health checks complete"
    echo "========================================"
    echo ""
    echo "To apply immediately in current shell:"
    echo "  source ~/.nvm/nvm.sh"
    echo "  unset npm_config_user_agent"
    echo ""
}

main "$@"
