-- ╔══════════════════════════════════════════════════════════════════════════════╗
-- ║              OVAV WEZTERM WINDOWS LOADER — PROXY v5 (C7.2.0)                 ║
-- ║                  %USERPROFILE%\.wezterm.lua · CAPA 7                          ║
-- ╚══════════════════════════════════════════════════════════════════════════════╝
--
-- PROXY ARCHITECTURE v4:
--   Este archivo vive en %USERPROFILE%\.wezterm.lua (Windows).
--   Carga la configuración canónica desde WSL vía UNC path dofile().
--   Si WSL no está disponible → fallback full con thema OVAV (no WSL).
--
--   Primary:   ~/.config/wezterm/wezterm.lua              (WSL canonical)
--   Windows:    %USERPROFILE%\.wezterm.lua                (este proxy)
--   Fallback:   %USERPROFILE%\.wezterm-fallback-full.lua (Windows-native, full OVAV)
--
-- INTEGRITY MARKERS (no modificar):
--   OVAV_WZPROXY_v3
--   OVAV_PROXY_MARKER
--   OVAV_CANONICAL_PATH_WSL
--   OVAV_CANONICAL_UNC
--   OVAV_FALLBACK_PATH
--   OVAV_CAPA7_CROSS_PLATFORM

local wezterm = require 'wezterm'
local config = wezterm.config_builder and wezterm.config_builder() or {}

-- ═══════════════════════════════════════════════════════════════════════════════
-- DETECCIÓN DE ENTORNO
-- ═══════════════════════════════════════════════════════════════════════════════

local is_windows = wezterm.target_triple:find('windows') ~= nil

-- WSL distro — configurable via variable de entorno
local wsl_distro = os.getenv('OVAV_WSL_DISTRO') or 'Ubuntu-24.04'
local wsl_canonical_path = string.format(
    '\\\\wsl.localhost\\%s\\home\\braka\\.config\\wezterm\\wezterm.lua',
    wsl_distro
)
-- Fallback adicional: ~/.wezterm.lua (symlink directo al repo OVAV)
local wsl_symlink_path = string.format(
    '\\\\wsl.localhost\\%s\\home\\braka\\.wezterm.lua',
    wsl_distro
)

-- Fallback path — full OVAV fallback con thema y keybindings completos
-- Este fallback NO requiere WSL — funciona 100% en Windows nativo
local fallback_path = os.getenv('USERPROFILE') .. '\\.wezterm-fallback-full.lua'

-- ═══════════════════════════════════════════════════════════════════════════════
-- CARGA CANÓNICA
-- ═══════════════════════════════════════════════════════════════════════════════

local canonical_loaded = false
local canonical_config = nil

if is_windows then
    -- Intentar cargar desde WSL UNC path
    local ok, result = pcall(function()
        return dofile(wsl_canonical_path)
    end)

    if ok and result then
        canonical_config = result
        canonical_loaded = true
        wezterm.log_info('OVAV WezTerm: canonical config loaded from WSL — ' .. wsl_canonical_path)
    else
        -- Fallback: intentar ~/.wezterm.lua (symlink directo al repo OVAV)
        wezterm.log_info('OVAV WezTerm: primary WSL path unreachable — trying symlink path')
        local ok2, result2 = pcall(function()
            return dofile(wsl_symlink_path)
        end)

        if ok2 and result2 then
            canonical_config = result2
            canonical_loaded = true
            wezterm.log_info('OVAV WezTerm: config loaded from WSL symlink — ' .. wsl_symlink_path)
        else
            wezterm.log_error('OVAV WezTerm: all WSL paths unreachable — loading fallback')
        end
    end
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- FALLBACK
-- ═══════════════════════════════════════════════════════════════════════════════

if not canonical_loaded then
    local ok, result = pcall(function()
        return dofile(fallback_path)
    end)

    if ok and result then
        canonical_config = result
        wezterm.log_info('OVAV WezTerm: full OVAV fallback loaded (Windows-native) — ' .. fallback_path)
    else
        wezterm.log_error('OVAV WezTerm: no config available — using minimal defaults')
        -- Minimal defaults para no quedar sin terminal
        config.font = wezterm.font('CaskaydiaCove Nerd Font Mono')
        config.font_size = 12.0
        config.color_scheme = 'Catppuccin Mocha'
        return config
    end
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- CONFIGURACIÓN CANÓNICA (desde WSL o fallback)
-- ═══════════════════════════════════════════════════════════════════════════════

return canonical_config
