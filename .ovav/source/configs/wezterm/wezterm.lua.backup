-- ╔══════════════════════════════════════════════════════════════════════════════╗
-- ║                    OVAV WEZTERM — CONFIGURACIÓN CANÓNICA                     ║
-- ║                   .ovav/visual/wezterm/config.lua · v1.0.0                   ║
-- ╚══════════════════════════════════════════════════════════════════════════════╝
--  Fuente canónica. Se proyecta a:
--    • ~/.config/wezterm/wezterm.lua      (WSL / Linux)
--    • %USERPROFILE%\.wezterm.lua         (Windows vía proxy)
--
--  Arquitectura:
--    4 workspaces aislados (Alt+1..4) con identidad visual por color.
--    Status bar con path + git (izq) y workspace + reloj (der).
--    Tab bar con acentos de workspace.
--    Splits direccionales, rotación de panes, zoom, navegación de tabs.
--
--  Dependencias:
--    • OVAV Theme v2.0.0  → .ovav/visual/theme/theme.yaml
--    • Nerd Font           → CaskaydiaCove Nerd Font Mono
--    • WSL                 → wsl.exe para spawn

local wezterm = require 'wezterm'
local act = wezterm.action
local config = wezterm.config_builder and wezterm.config_builder() or {}

-- ═══════════════════════════════════════════════════════════════════════════════
-- PALETA OVAV v2.0.0 — eye-friendly, bajo contraste, sesiones largas
-- ═══════════════════════════════════════════════════════════════════════════════
local P = {
  -- Superficie
  bg        = '#242424',
  bg_panel  = '#2d2d2d',
  bg_hover  = '#454545',

  -- Texto
  fg        = '#d4d4d4',
  fg_bright = '#e8e8e8',
  fg_dim    = '#a0a0a0',
  fg_muted  = '#707070',

  -- Workspace accents
  ws_home   = '#7eb77f',
  ws_system = '#d4a85c',
  ws_dev    = '#5a7d8a',
  ws_ovav   = '#c47d8a',

  -- Workspace dim (tab backgrounds)
  ws_home_dim   = '#1e2a1e',
  ws_system_dim = '#25281e',
  ws_dev_dim    = '#1e2528',
  ws_ovav_dim   = '#281e22',

  -- Semánticos
  success  = '#7eb77f',
  error    = '#d4756b',
  warning  = '#d4a85c',
  info     = '#6d9bc3',

  -- ANSI
  black    = '#242424',
  red      = '#d4756b',
  green    = '#7eb77f',
  yellow   = '#c4a65a',
  blue     = '#6d9bc3',
  magenta  = '#c47d8a',
  cyan     = '#5a7d8a',
  white    = '#d4d4d4',

  -- Brights
  black_h  = '#383838',
  red_h    = '#d4756b',
  green_h  = '#7eb77f',
  yellow_h = '#c4a65a',
  blue_h   = '#6d9bc3',
  magenta_h= '#c47d8a',
  cyan_h   = '#6d9b8e',
  white_h  = '#d4d4d4',

  -- Surface extendida
  surface0 = '#2d2d2d',
  surface1 = '#383838',
  surface2 = '#454545',
  overlay0 = '#505050',
  overlay1 = '#383838',
  overlay2 = '#707070',
}

-- ═══════════════════════════════════════════════════════════════════════════════
-- WORKSPACES — 4 sesiones aisladas con identidad visual
-- ═══════════════════════════════════════════════════════════════════════════════
local WORKSPACES = {
  home = {
    key = '1', label = 'HOME', icon = '󰉋',
    accent = '#7eb77f', accent_dim = '#1e2a1e', cursor = '#7eb77f',
    cwd = '~',
    exact = { '/home/braka' },
    prefixes = {},
  },
  system = {
    key = '2', label = 'SYS', icon = '󰒓',
    accent = '#d4a85c', accent_dim = '#25281e', cursor = '#d4a85c',
    cwd = '/home/braka/.config',
    exact = {},
    prefixes = { '/home/braka/.config', '/home/braka/.local', '/etc' },
  },
  devbrk = {
    key = '3', label = 'DEV', icon = '󰲋',
    accent = '#5a7d8a', accent_dim = '#1e2528', cursor = '#5a7d8a',
    cwd = '/home/braka/dev/work',
    exact = {},
    prefixes = { '/home/braka/dev/work' },
    worktrees = { '/home/braka/dev/work' },
  },
  ovav = {
    key = '4', label = 'OVAV', icon = '󱚣',
    accent = '#c47d8a', accent_dim = '#281e22', cursor = '#c47d8a',
    cwd = '/home/braka/Systems/OVAV',
    exact = {},
    prefixes = { '/home/braka/Systems/OVAV' },
    worktrees = { '/home/braka/Systems/OVAV' },
  },
}

local C = { bg = P.bg, accent = P.ws_home, accent_dim = P.ws_home_dim, cursor = '#7eb77f' }
local function set_ws_palette(ws_name)
  local ws = WORKSPACES[ws_name] or WORKSPACES['home']
  C.accent = ws.accent; C.accent_dim = ws.accent_dim; C.cursor = ws.cursor
end

wezterm.GLOBAL.ovav_workspace = wezterm.GLOBAL.ovav_workspace or 'home'
wezterm.GLOBAL.focus_mode = wezterm.GLOBAL.focus_mode or false

-- ═══════════════════════════════════════════════════════════════════════════════
-- HELPERS
-- ═══════════════════════════════════════════════════════════════════════════════
local function normalize_uri(uri)
  if not uri then return '' end
  if type(uri) == 'userdata' and uri.file_path then return uri.file_path end
  local s = tostring(uri):gsub('^file://[^/]*', ''):gsub('%%20', ' '):gsub('^/C:/', 'C:/')
  return s
end

local function pane_cwd(pane)
  local pane_id = tostring(pane:pane_id())
  -- 1. OSC 0 title — emitted by fish __wezterm_report_cwd on startup + PWD change
  local ok1, title = pcall(function() return pane:get_title() end)
  if ok1 and title and title ~= '' then
    if title:match('^/') then wezterm.GLOBAL['ovav_cwd_' .. pane_id] = title; return title end
  end
  -- 2. Foreground process info — reliable fallback
  local ok2, info = pcall(function() return pane:get_foreground_process_info() end)
  if ok2 and info and info.cwd then
    local c = normalize_uri(info.cwd)
    if c ~= '' and c:match('^/') then wezterm.GLOBAL['ovav_cwd_' .. pane_id] = c; return c end
  end
  -- 3. Direct API — reads OSC 7
  local ok3, uri = pcall(function() return pane:get_current_working_dir() end)
  if ok3 and uri then
    local c = normalize_uri(uri)
    if c ~= '' and c:match('^/') then wezterm.GLOBAL['ovav_cwd_' .. pane_id] = c; return c end
  end
  -- 4. Fallback: last known CWD for this pane
  return wezterm.GLOBAL['ovav_cwd_' .. pane_id] or ''
end

local function starts_with(s, prefix)
  return s and prefix and s:sub(1, #prefix) == prefix
end

local function detect_workspace(cwd)
  if not cwd or cwd == '' then return 'home' end
  for name, ws in pairs(WORKSPACES) do
    if ws.exact then for _, p in ipairs(ws.exact) do if cwd == p then return name end end end
    if ws.prefixes then for _, p in ipairs(ws.prefixes) do if starts_with(cwd, p) then return name end end end
  end
  return 'home'
end

local function abbrev_path(cwd, ws_root)
  local rel = cwd
  if ws_root and ws_root ~= '~' and starts_with(cwd, ws_root) then
    rel = cwd:sub(#ws_root + 1):gsub('^/', '')
    if rel == '' then return '·' end
  end
  local home = os.getenv('HOME') or '/home/braka'
  if cwd == home then return '~' end
  local segs = {}
  for seg in rel:gmatch('[^/]+') do table.insert(segs, seg) end
  if #segs == 0 then return rel end
  if #segs == 1 then
    local s = segs[1]; if #s > 18 then s = s:sub(1, 15) .. '…' end; return s
  end
  local last, prev = segs[#segs], segs[#segs - 1]
  if #prev > 10 then prev = prev:sub(1, 7) .. '…' end
  local label = prev .. '/' .. last
  if #label > 28 then label = label:sub(1, 25) .. '…' end
  return label
end

local function git_branch(path)
  local b = path:match('/%.worktrees/([^/]+)'); if b then return b end
  for _, pfx in ipairs({'feature/', 'fix/', 'task/', 'hotfix/'}) do
    b = path:match('/(' .. pfx .. '[^/]+)'); if b then return b end
  end
  return nil
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- SPAWN
-- ═══════════════════════════════════════════════════════════════════════════════
local DISTRO = os.getenv('OVAV_WSL_DISTRO') or 'Ubuntu-24.04'

local function wsl_args(cwd)
  return { 'wsl.exe', '-d', DISTRO, '--cd', cwd, '--', 'fish', '-l' }
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- WORKSPACE SWITCHING + COLOR OVERRIDE
-- ═══════════════════════════════════════════════════════════════════════════════
local WS_COLORS = {
  home = {
    foreground = '#d4d4d4',
    background = '#1e2a1e',
    cursor_bg = '#7eb77f',
    cursor_fg = '#1e2a1e',
    cursor_border = '#7eb77f',
    selection_bg = '#2d3a2d',
    ansi = { '#1e2a1e', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#5a7d8a', '#d4d4d4' },
    brights = { '#2d3a2d', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#6d9b8e', '#e8e8e8' },
    tab_bar = { background = '#1e2a1e', active_tab = { bg_color = '#1e2a1e', fg_color = '#d4d4d4' }, inactive_tab = { bg_color = '#1e2a1e', fg_color = '#707070' }, inactive_tab_hover = { bg_color = '#2d3a2d', fg_color = '#d4d4d4' }, new_tab = { bg_color = '#1e2a1e', fg_color = '#707070' }, new_tab_hover = { bg_color = '#2d3a2d', fg_color = '#d4d4d4' } },
  },
  system = {
    foreground = '#d4d4d4',
    background = '#25281e',
    cursor_bg = '#d4a85c',
    cursor_fg = '#25281e',
    cursor_border = '#d4a85c',
    selection_bg = '#353820',
    ansi = { '#25281e', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#5a7d8a', '#d4d4d4' },
    brights = { '#353820', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#6d9b8e', '#e8e8e8' },
    tab_bar = { background = '#25281e', active_tab = { bg_color = '#25281e', fg_color = '#d4d4d4' }, inactive_tab = { bg_color = '#25281e', fg_color = '#707070' }, inactive_tab_hover = { bg_color = '#353820', fg_color = '#d4d4d4' }, new_tab = { bg_color = '#25281e', fg_color = '#707070' }, new_tab_hover = { bg_color = '#353820', fg_color = '#d4d4d4' } },
  },
  devbrk = {
    foreground = '#d4d4d4',
    background = '#1e2528',
    cursor_bg = '#5a7d8a',
    cursor_fg = '#1e2528',
    cursor_border = '#5a7d8a',
    selection_bg = '#2d3538',
    ansi = { '#1e2528', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#5a7d8a', '#d4d4d4' },
    brights = { '#2d3538', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#6d9b8e', '#e8e8e8' },
    tab_bar = { background = '#1e2528', active_tab = { bg_color = '#1e2528', fg_color = '#d4d4d4' }, inactive_tab = { bg_color = '#1e2528', fg_color = '#707070' }, inactive_tab_hover = { bg_color = '#2d3538', fg_color = '#d4d4d4' }, new_tab = { bg_color = '#1e2528', fg_color = '#707070' }, new_tab_hover = { bg_color = '#2d3538', fg_color = '#d4d4d4' } },
  },
  ovav = {
    foreground = '#d4d4d4',
    background = '#281e22',
    cursor_bg = '#c47d8a',
    cursor_fg = '#281e22',
    cursor_border = '#c47d8a',
    selection_bg = '#382d32',
    ansi = { '#281e22', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#5a7d8a', '#d4d4d4' },
    brights = { '#382d32', '#d4756b', '#7eb77f', '#c4a65a', '#6d9bc3', '#c47d8a', '#6d9b8e', '#e8e8e8' },
    tab_bar = { background = '#281e22', active_tab = { bg_color = '#281e22', fg_color = '#d4d4d4' }, inactive_tab = { bg_color = '#281e22', fg_color = '#707070' }, inactive_tab_hover = { bg_color = '#382d32', fg_color = '#d4d4d4' }, new_tab = { bg_color = '#281e22', fg_color = '#707070' }, new_tab_hover = { bg_color = '#382d32', fg_color = '#d4d4d4' } },
  },
}

local function apply_ws_colors(window, ws_name)
  local ws = WORKSPACES[ws_name]; if not ws then return end
  local wsc = WS_COLORS[ws_name] or WS_COLORS['home']
  set_ws_palette(ws_name)
  pcall(function()
    window:set_config_overrides({
      colors = {
        foreground = wsc.foreground,
        background = wsc.background,
        cursor_bg = wsc.cursor_bg,
        cursor_fg = wsc.cursor_fg,
        cursor_border = wsc.cursor_border,
        selection_bg = wsc.selection_bg,
        ansi = wsc.ansi,
        brights = wsc.brights,
        split = ws.accent_dim,
        scrollbar_thumb = ws.accent_dim,
        visual_bell = ws.accent,
        tab_bar = wsc.tab_bar,
      },
    })
  end)
end

local function switch_workspace(window, pane, ws_name)
  local ws = WORKSPACES[ws_name]; if not ws then return end
  wezterm.GLOBAL.ovav_workspace = ws_name
  apply_ws_colors(window, ws_name)
  window:perform_action(act.SwitchToWorkspace({
    name = 'ovav-' .. ws_name,
    spawn = { args = wsl_args(ws.cwd) },
  }), pane)
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- SMART PANE SPLIT
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('ovav-smart-pane', function(window, pane)
  local cwd = pane_cwd(pane)
  if cwd == '' then cwd = '~' end
  local direction, size = 'Right', { Percent = 40 }
  local ok2, dims = pcall(function() return pane:get_dimensions() end)
  if ok2 and dims and dims.pixel_width and dims.pixel_width < 900 then
    direction, size = 'Down', { Percent = 50 }
  end
  window:perform_action(act.SplitPane({ direction = direction, size = size, command = { args = wsl_args(cwd) } }), pane)
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- FOCUS MODE
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('ovav-toggle-focus', function(window, _)
  wezterm.GLOBAL.focus_mode = not (wezterm.GLOBAL.focus_mode or false)
  pcall(function() window:set_config_overrides({ enable_tab_bar = not wezterm.GLOBAL.focus_mode }) end)
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- FOCUS MODE
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('ovav-toggle-opacity', function(window, _)
  local cur = wezterm.GLOBAL.bg_opacity or 1.0; local nxt = cur == 1.0 and 0.88 or 1.0
  wezterm.GLOBAL.bg_opacity = nxt
  pcall(function() window:set_config_overrides({ window_background_opacity = nxt }) end)
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- TAB BAR — native tab bar with Chrome-like close buttons
-- format-tab-title supplies icon + label; close X is native (on hover)
-- ═══════════════════════════════════════════════════════════════════════════════
-- ═══════════════════════════════════════════════════════════════════════════════
-- TAB BAR — static workspace labels, numbered per session
-- Each tab born at its workspace root directory.
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('format-tab-title', function(tab, tabs, panes, _, hover, _)
  -- Tag this tab with current workspace (first-call only)
  local tab_id = tostring(tab:tab_id())
  if not wezterm.GLOBAL['ovav_tab_ws_' .. tab_id] then
    wezterm.GLOBAL['ovav_tab_ws_' .. tab_id] = wezterm.GLOBAL.ovav_workspace or 'home'
  end
  local ws_name = wezterm.GLOBAL['ovav_tab_ws_' .. tab_id] or 'home'
  local ws = WORKSPACES[ws_name] or WORKSPACES['home']
  local icon, accent = ws.icon or '', ws.accent or P.fg

  -- Count how many tabs in this workspace for numbering
  local tab_index = 1
  local same_ws_count = 0
  if tabs then
    for _, t in ipairs(tabs) do
      local tws = wezterm.GLOBAL['ovav_tab_ws_' .. tostring(t:tab_id())] or wezterm.GLOBAL.ovav_workspace or 'home'
      if tws == ws_name then
        same_ws_count = same_ws_count + 1
        if t:tab_id() == tab:tab_id() then tab_index = same_ws_count end
      end
    end
  end

  local label = ws.label
  if same_ws_count > 1 then label = ws.label .. ' ' .. tab_index end

  -- Pane count
  local pane_count = 0
  if panes then for _ in pairs(panes) do pane_count = pane_count + 1 end end
  local count_suffix = pane_count > 1 and (' ⊞' .. pane_count) or ''

  if tab:is_active() then
    return {
      { Foreground = { Color = accent } }, { Text = icon .. ' ' },
      { Foreground = { Color = P.fg_bright } }, { Text = label .. count_suffix .. ' ' },
    }
  else
    return {
      { Foreground = { Color = P.fg_muted } }, { Text = icon .. ' ' .. label .. count_suffix .. ' ' },
    }
  end
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- STATUS BAR
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('update-status', function(window, pane)
  if wezterm.GLOBAL.focus_mode then
    window:set_left_status(''); window:set_right_status(''); return
  end
  local cwd = pane_cwd(pane); local ws_name = detect_workspace(cwd)
  local ws = WORKSPACES[ws_name] or WORKSPACES['home']
  wezterm.GLOBAL.ovav_workspace = ws_name

  -- Mode check
  local active_table = window:active_key_table()

  if active_table == 'resize' then
    right = {
      { Foreground = { Color = P.info } }, { Text = '' },
      { Background = { Color = P.info } }, { Foreground = { Color = C.bg } }, { Text = ' RESIZE  ←h ↓j ↑k →l ' },
      { Foreground = { Color = P.info } }, { Text = '' },
      { Text = ' ' },
      { Foreground = { Color = P.surface1 } }, { Text = '' },
      { Background = { Color = P.surface1 } },
      { Foreground = { Color = C.accent } }, { Text = ws.icon .. ' ' },
      { Foreground = { Color = P.fg } }, { Text = ws.label .. '  ' },
      { Foreground = { Color = P.fg_dim } }, { Text = wezterm.strftime('%H:%M') .. '  ' },
      { Background = { Color = C.bg } }, { Foreground = { Color = P.surface1 } }, { Text = '' },
    }
    window:set_right_status(wezterm.format(right))
    window:set_left_status('')
    return
  elseif active_table == 'leader' then
    right = {
      { Foreground = { Color = P.green } }, { Text = '' },
      { Background = { Color = P.green } }, { Foreground = { Color = C.bg } }, { Text = ' ←h ↓j ↑k →l ' },
      { Foreground = { Color = P.green } }, { Text = '' }, { Text = ' ' },
      { Foreground = { Color = P.info } }, { Text = '' },
      { Background = { Color = P.info } }, { Foreground = { Color = C.bg } }, { Text = ' s:split v:vert ' },
      { Foreground = { Color = P.info } }, { Text = '' }, { Text = ' ' },
      { Foreground = { Color = P.ws_ovav } }, { Text = '' },
      { Background = { Color = P.ws_ovav } }, { Foreground = { Color = C.bg } }, { Text = ' 1 2 3 4 ' },
      { Foreground = { Color = P.ws_ovav } }, { Text = '' }, { Text = ' ' },
      { Foreground = { Color = P.warning } }, { Text = '' },
      { Background = { Color = P.warning } }, { Foreground = { Color = C.bg } }, { Text = ' g:home n q w z r ? ' },
      { Foreground = { Color = P.warning } }, { Text = '' }, { Text = '  ' },
      { Foreground = { Color = P.surface1 } }, { Text = '' },
      { Background = { Color = P.surface1 } },
      { Foreground = { Color = C.accent } }, { Text = ws.icon .. ' ' },
      { Foreground = { Color = P.fg } }, { Text = ws.label .. '  ' },
      { Foreground = { Color = P.fg_dim } }, { Text = wezterm.strftime('%H:%M') .. '  ' },
      { Background = { Color = C.bg } }, { Foreground = { Color = P.surface1 } }, { Text = '' },
    }
    window:set_right_status(wezterm.format(right))
    window:set_left_status('')
    return
  end

  -- NORMAL MODE
  right = {
    { Background = { Color = C.bg } },
    { Foreground = { Color = P.surface1 } }, { Text = '' },
    { Background = { Color = P.surface1 } },
    { Foreground = { Color = C.accent } }, { Text = ws.icon .. ' ' },
    { Foreground = { Color = P.fg } }, { Text = ws.label .. '  ' },
    { Foreground = { Color = P.fg_dim } }, { Text = wezterm.strftime('%H:%M') .. '  ' },
    { Background = { Color = C.bg } },
    { Foreground = { Color = P.surface1 } }, { Text = '' },
  }
  window:set_right_status(wezterm.format(right))
  window:set_left_status('')
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- APARIENCIA
-- ═══════════════════════════════════════════════════════════════════════════════
config.font = wezterm.font_with_fallback({
  'CaskaydiaCove Nerd Font Mono', 'CaskaydiaCove Nerd Font',
  'JetBrainsMono Nerd Font', 'Symbols Nerd Font Mono', 'Noto Color Emoji',
})
config.font_size = 12.0
config.line_height = 1.10
config.cell_width = 1.0
config.bold_brightens_ansi_colors = true
config.freetype_load_target = 'Light'
config.freetype_render_target = 'HorizontalLcd'
config.text_background_opacity = 1.0
config.win32_system_backdrop = 'Disable'
config.window_padding = { left = 14, right = 14, top = 8, bottom = 8 }
config.window_decorations = 'RESIZE'
config.inactive_pane_hsb = { saturation = 1.0, brightness = 1.0 }
config.audible_bell = 'SystemBeep'
config.exit_behavior = 'CloseOnCleanExit'
config.window_close_confirmation = 'NeverPrompt'

-- ═══════════════════════════════════════════════════════════════════════════════
-- RENDER
-- ═══════════════════════════════════════════════════════════════════════════════
config.front_end = 'OpenGL'
config.webgpu_preferred_adapter = nil
config.enable_wayland = false
config.animation_fps = 2
config.max_fps = 120
config.cursor_blink_rate = 0
config.text_blink_rate = 0
config.scrollback_lines = 20000
config.default_cursor_style = 'SteadyBar'
config.cursor_thickness = '1.8px'
config.adjust_window_size_when_changing_font_size = false

-- ═══════════════════════════════════════════════════════════════════════════════
-- COLORES
-- ═══════════════════════════════════════════════════════════════════════════════
config.colors = {
  foreground = P.fg, background = P.bg, cursor_bg = P.green, cursor_fg = P.bg,
  cursor_border = P.green, selection_fg = P.white, selection_bg = P.surface2,
  scrollbar_thumb = P.ws_home_dim, split = P.ws_home_dim, visual_bell = P.green,
  ansi = { P.black, P.red, P.green, P.yellow, P.blue, P.magenta, P.cyan, P.white },
  brights = { P.black_h, P.red_h, P.green_h, P.yellow_h, P.blue_h, P.magenta_h, P.cyan_h, P.white_h },
  indexed = {
    [16] = P.black, [17] = P.surface1, [18] = P.surface2, [19] = P.overlay0,
    [20] = P.overlay1, [21] = P.overlay2, [22] = P.fg_muted, [23] = P.fg_dim,
  },
  tab_bar = {
    background = P.bg,
    active_tab = { bg_color = P.ws_home_dim, fg_color = P.green, intensity = 'Bold' },
    inactive_tab = { bg_color = P.bg, fg_color = P.fg_muted },
    inactive_tab_hover = { bg_color = P.bg_hover, fg_color = P.fg },
    inactive_tab_edge = P.surface1,
    new_tab = { bg_color = P.bg, fg_color = P.fg_dim },
    new_tab_hover = { bg_color = P.bg_hover, fg_color = P.fg, intensity = 'Bold' },
  },
}

-- ═══════════════════════════════════════════════════════════════════════════════
-- TAB BAR SETTINGS
-- ═══════════════════════════════════════════════════════════════════════════════
config.enable_tab_bar = true
config.use_fancy_tab_bar = true   -- Native tab bar: Chrome-like close buttons on each tab
config.show_tabs_in_tab_bar = true
config.show_new_tab_button_in_tab_bar = true
config.show_tab_index_in_tab_bar = false
config.tab_bar_at_bottom = false
config.tab_max_width = 32
config.hide_tab_bar_if_only_one_tab = false
config.selection_word_boundary = ' \t\n{}[]()"\'`,;:│=<>'

-- ═══════════════════════════════════════════════════════════════════════════════
-- QUICK SELECT + HYPERLINKS
-- ═══════════════════════════════════════════════════════════════════════════════
config.quick_select_patterns = {
  '[0-9a-f]{7,40}', '[#][0-9]+', '[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+[#][0-9]+',
  '(~?[/][a-zA-Z0-9_./-]+)', '\\d{1,3}[.]\\d{1,3}[.]\\d{1,3}[.]\\d{1,3}',
  '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}',
  'sha256:[0-9a-f]{64}', 'v?[0-9]+\\.[0-9]+\\.[0-9]+', '\\d{4}-\\d{2}-\\d{2}',
  '[a-zA-Z0-9_.-]+[.][a-zA-Z]+[:]\\d+[:]\\d+', 'File\\s+"[^"]+",\\s*line\\s+\\d+',
  '[a-zA-Z0-9_/.-]+_test[.]go[:]\\d+', 'E\\d{3,5}', '[A-Z_]{3,20}_ERROR',
}
config.hyperlink_rules = wezterm.default_hyperlink_rules()
table.insert(config.hyperlink_rules, { regex = [[\b#(\d+)\b]], format = 'https://github.com/ovav-dev/ovav-systems/issues/$1', highlight = 1 })
table.insert(config.hyperlink_rules, { regex = [[\b!(\d+)\b]], format = 'https://github.com/ovav-dev/ovav-systems/pull/$1', highlight = 1 })

-- ═══════════════════════════════════════════════════════════════════════════════
-- WORKSPACE ISOLATION
-- ═══════════════════════════════════════════════════════════════════════════════
config.default_workspace = os.getenv('OVAV_WEZTERM_WORKSPACE') or 'ovav-home'
config.default_prog = wsl_args('~')

-- ═══════════════════════════════════════════════════════════════════════════════
-- KEY BINDINGS
-- ═══════════════════════════════════════════════════════════════════════════════
config.keys = {
  -- Leader key — Ctrl+O locks terminal in command mode until Escape or Ctrl+O again
  { key = 'o', mods = 'CTRL', action = act.ActivateKeyTable({ name = 'leader', one_shot = false, prevent_fallback = true }) },

  -- Workspace switching with Alt+1-4 (tracks previous for toggle)
  { key = '1', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
    switch_workspace(w, p, 'home')
  end) },
  { key = '2', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
    switch_workspace(w, p, 'system')
  end) },
  { key = '3', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
    switch_workspace(w, p, 'devbrk')
  end) },
  { key = '4', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
    switch_workspace(w, p, 'ovav')
  end) },

  -- Passthrough
  { key = 'x', mods = 'CTRL', action = act.SendKey({ key = 'x', mods = 'CTRL' }) },

  -- New tab (Ctrl+T / Alt+T) — spawns at current workspace root
  { key = 't', mods = 'CTRL', action = wezterm.action_callback(function(w, p)
    local ws_name = wezterm.GLOBAL.ovav_workspace or 'home'
    local ws = WORKSPACES[ws_name] or WORKSPACES['home']
    w:perform_action(act.SpawnCommandInNewTab({ args = wsl_args(ws.cwd) }), p)
    wezterm.GLOBAL['ovav_pending_ws'] = ws_name
  end) },
  { key = 't', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    local ws_name = wezterm.GLOBAL.ovav_workspace or 'home'
    local ws = WORKSPACES[ws_name] or WORKSPACES['home']
    w:perform_action(act.SpawnCommandInNewTab({ args = wsl_args(ws.cwd) }), p)
    wezterm.GLOBAL['ovav_pending_ws'] = ws_name
  end) },

  -- Close tab
  { key = 'w', mods = 'CTRL', action = act.CloseCurrentTab({ confirm = false }) },
  { key = 'F4', mods = 'CTRL', action = act.CloseCurrentTab({ confirm = false }) },

  -- Navigate tabs
  { key = 'Tab', mods = 'CTRL', action = act.ActivateTabRelative(1) },
  { key = 'Tab', mods = 'CTRL|SHIFT', action = act.ActivateTabRelative(-1) },

  -- Focus mode
  { key = 'f', mods = 'ALT|SHIFT', action = act.EmitEvent('ovav-toggle-focus') },

  -- Panes — smart split + directional
  { key = 'a', mods = 'ALT', action = act.EmitEvent('ovav-smart-pane') },
  { key = 's', mods = 'ALT', action = act.SplitPane({ direction = 'Down', size = { Percent = 50 } }) },
  { key = 'h', mods = 'ALT|SHIFT', action = act.SplitPane({ direction = 'Left',  size = { Percent = 40 } }) },
  { key = 'j', mods = 'ALT|SHIFT', action = act.SplitPane({ direction = 'Down',  size = { Percent = 50 } }) },
  { key = 'k', mods = 'ALT|SHIFT', action = act.SplitPane({ direction = 'Up',    size = { Percent = 50 } }) },
  { key = 'l', mods = 'ALT|SHIFT', action = act.SplitPane({ direction = 'Right', size = { Percent = 40 } }) },
  { key = 'H', mods = 'ALT', action = act.SplitPane({ direction = 'Left',  size = { Percent = 40 } }) },
  { key = 'J', mods = 'ALT', action = act.SplitPane({ direction = 'Down',  size = { Percent = 50 } }) },
  { key = 'K', mods = 'ALT', action = act.SplitPane({ direction = 'Up',    size = { Percent = 50 } }) },
  { key = 'L', mods = 'ALT', action = act.SplitPane({ direction = 'Right', size = { Percent = 40 } }) },

  -- Zoom pane
  { key = 'Enter', mods = 'ALT', action = act.TogglePaneZoomState },

  -- Rotate panes
  { key = '.', mods = 'ALT', action = act.RotatePanes('Clockwise') },
  { key = ',', mods = 'ALT', action = act.RotatePanes('CounterClockwise') },

  -- Close pane
  { key = 'w', mods = 'ALT', action = act.CloseCurrentPane({ confirm = false }) },
  { key = 'x', mods = 'ALT', action = act.CloseCurrentPane({ confirm = false }) },

  -- Pane navigation (Alt+Arrows)
  { key = 'LeftArrow',  mods = 'ALT', action = act.ActivatePaneDirection('Left') },
  { key = 'RightArrow', mods = 'ALT', action = act.ActivatePaneDirection('Right') },
  { key = 'UpArrow',    mods = 'ALT', action = act.ActivatePaneDirection('Up') },
  { key = 'DownArrow',  mods = 'ALT', action = act.ActivatePaneDirection('Down') },

  -- Pane resize (Alt+Shift+Arrows)
  { key = 'LeftArrow',  mods = 'ALT|SHIFT', action = act.AdjustPaneSize({ 'Left', 4 }) },
  { key = 'RightArrow', mods = 'ALT|SHIFT', action = act.AdjustPaneSize({ 'Right', 4 }) },
  { key = 'UpArrow',    mods = 'ALT|SHIFT', action = act.AdjustPaneSize({ 'Up', 2 }) },
  { key = 'DownArrow',  mods = 'ALT|SHIFT', action = act.AdjustPaneSize({ 'Down', 2 }) },

  -- Launcher / Search
  { key = 'l', mods = 'ALT', action = act.ShowLauncher },
  { key = 'p', mods = 'CTRL|SHIFT', action = act.ActivateCommandPalette },
  { key = ' ', mods = 'CTRL|SHIFT', action = act.QuickSelect },
  { key = 'f', mods = 'CTRL', action = act.Search({ CaseInSensitiveString = '' }) },

  -- Clipboard
  { key = 'v', mods = 'CTRL', action = act.PasteFrom('Clipboard') },
  { key = 'c', mods = 'CTRL|SHIFT', action = act.CopyTo('Clipboard') },

  -- Font size
  { key = '=', mods = 'CTRL',  action = act.IncreaseFontSize },
  { key = '+', mods = 'CTRL',  action = act.IncreaseFontSize },
  { key = '-', mods = 'CTRL',  action = act.DecreaseFontSize },
  { key = '0', mods = 'CTRL',  action = act.ResetFontSize },

  -- Kitty protocol (OpenCode input_newline)
  { key = 'Enter', mods = 'SHIFT', action = act.SendString('\x1b[13;2u') },
}

-- ═══════════════════════════════════════════════════════════════════════════════
-- LEADER KEY TABLE — Ctrl+O activa modo comando (estilo Zellij)
-- Barra de estado muestra guia visual de teclas coloreada por grupo.
-- Sub-modo resize: lider + r → hjkl redimensiona panes. Escape para volver.
-- ` (backtick): toggle al ultimo workspace.
-- ═══════════════════════════════════════════════════════════════════════════════
config.key_tables = {
  leader = {
    -- Exit leader mode
    { key = 'Escape', action = act.PopKeyTable },
    { key = 'o', mods = 'CTRL', action = act.PopKeyTable },

    -- Pane navigation (vim-style + arrows)
    { key = 'h', action = act.ActivatePaneDirection('Left') },
    { key = 'j', action = act.ActivatePaneDirection('Down') },
    { key = 'k', action = act.ActivatePaneDirection('Up') },
    { key = 'l', action = act.ActivatePaneDirection('Right') },
    { key = 'LeftArrow',  action = act.ActivatePaneDirection('Left') },
    { key = 'DownArrow',  action = act.ActivatePaneDirection('Down') },
    { key = 'UpArrow',    action = act.ActivatePaneDirection('Up') },
    { key = 'RightArrow', action = act.ActivatePaneDirection('Right') },

    -- Splits
    { key = 's', action = act.EmitEvent('ovav-smart-pane') },
    { key = 'v', action = act.SplitPane({ direction = 'Right', size = { Percent = 40 } }) },
    { key = 'H', action = act.SplitPane({ direction = 'Left',  size = { Percent = 40 } }) },
    { key = 'J', action = act.SplitPane({ direction = 'Down',  size = { Percent = 50 } }) },
    { key = 'K', action = act.SplitPane({ direction = 'Up',    size = { Percent = 50 } }) },
    { key = 'L', action = act.SplitPane({ direction = 'Right', size = { Percent = 40 } }) },

    -- Pane management
    { key = 'q', action = act.CloseCurrentPane({ confirm = false }) },
    { key = 'z', action = act.TogglePaneZoomState },
    { key = 'r', action = act.ActivateKeyTable({ name = 'resize', one_shot = false, replace_current = false, until_unknown = true }) },
    { key = 'R', action = act.RotatePanes('Clockwise') },

    -- Tab management
    { key = 'n', action = wezterm.action_callback(function(w, p)
      local ws_name = wezterm.GLOBAL.ovav_workspace or 'home'
      local ws = WORKSPACES[ws_name] or WORKSPACES['home']
      w:perform_action(act.SpawnCommandInNewTab({ args = wsl_args(ws.cwd) }), p)
      wezterm.GLOBAL['ovav_pending_ws'] = ws_name
    end) },
    { key = 'w', action = act.CloseCurrentTab({ confirm = false }) },
    { key = 'Tab', action = act.ActivateTabRelative(1) },
    { key = 'p', action = act.ActivateTabRelative(-1) },

    -- Workspace switching
    { key = '1', action = wezterm.action_callback(function(w, p)
      wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
      switch_workspace(w, p, 'home')
    end) },
    { key = '2', action = wezterm.action_callback(function(w, p)
      wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
      switch_workspace(w, p, 'system')
    end) },
    { key = '3', action = wezterm.action_callback(function(w, p)
      wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
      switch_workspace(w, p, 'devbrk')
    end) },
    { key = '4', action = wezterm.action_callback(function(w, p)
      wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
      switch_workspace(w, p, 'ovav')
    end) },
    -- Toggle last workspace
    { key = '`', action = wezterm.action_callback(function(w, p)
      local prev = wezterm.GLOBAL.ovav_prev_workspace
      if prev and prev ~= '' and WORKSPACES[prev] then
        wezterm.GLOBAL.ovav_prev_workspace = wezterm.GLOBAL.ovav_workspace
        switch_workspace(w, p, prev)
      end
    end) },

    -- Toggles
    { key = 'f', action = act.EmitEvent('ovav-toggle-focus') },

    -- Tools
    -- Go home: new tab at home directory
    { key = 'g', action = wezterm.action_callback(function(w, p)
      w:perform_action(act.SpawnCommandInNewTab({ args = wsl_args('~') }), p)
    end) },
    { key = '?', action = wezterm.action_callback(function(w, _)
      w:toast_notification('ovav-leader', string.format([[
←↓↑→  h j k l          navegar panes
split  s v H J K L      dividir panes
resize r                redimensionar (hjkl)
zoom   z                maximizar pane
close  q (pane) w (tab) cerrar
tab    n (new) Tab/prev siguiente/anterior
ws     1 2 3 4          workspaces
       ` (backtick)     toggle anterior
goto   g                tab nueva en home
foco   f                pantalla completa
buscar /                buscar texto
cmd    Space            paleta comandos
copy   y  paste P       portapapeles
font   = + - 0          tamaño fuente
rotar  R                rotar panes
ayuda  ?                esta lista
salir  Esc / Ctrl+O     salir modo comandos
]]), nil, 12000)
    end) },
    { key = '/', action = act.Search({ CaseInSensitiveString = '' }) },
    { key = ' ', action = act.ActivateCommandPalette },
    { key = 'y', action = act.CopyTo('Clipboard') },
    { key = 'P', action = act.PasteFrom('Clipboard') },

    -- Font size
    { key = '=', action = act.IncreaseFontSize },
    { key = '-', action = act.DecreaseFontSize },
    { key = '0', action = act.ResetFontSize },
  },
  -- Sub-modo: resize panes (leader + r). Escape vuelve a lider.
  resize = {
    { key = 'Escape', action = act.PopKeyTable },
    { key = 'h', action = act.AdjustPaneSize({ 'Left', 4 }) },
    { key = 'j', action = act.AdjustPaneSize({ 'Down', 2 }) },
    { key = 'k', action = act.AdjustPaneSize({ 'Up', 2 }) },
    { key = 'l', action = act.AdjustPaneSize({ 'Right', 4 }) },
    { key = 'LeftArrow',  action = act.AdjustPaneSize({ 'Left', 4 }) },
    { key = 'DownArrow',  action = act.AdjustPaneSize({ 'Down', 2 }) },
    { key = 'UpArrow',    action = act.AdjustPaneSize({ 'Up', 2 }) },
    { key = 'RightArrow', action = act.AdjustPaneSize({ 'Right', 4 }) },
  },
}

-- ═══════════════════════════════════════════════════════════════════════════════
-- MOUSE BINDINGS
-- ═══════════════════════════════════════════════════════════════════════════════
config.mouse_bindings = {
  { event = { Up = { streak = 1, button = 'Left' } }, mods = 'NONE', action = act.CompleteSelection('Clipboard') },
  { event = { Up = { streak = 2, button = 'Left' } }, mods = 'NONE', action = act.CompleteSelection('Clipboard') },
  { event = { Up = { streak = 3, button = 'Left' } }, mods = 'NONE', action = act.CompleteSelection('Clipboard') },
  { event = { Down = { streak = 1, button = 'Left' } }, mods = 'CTRL', action = act.OpenLinkAtMouseCursor },
  { event = { Down = { streak = 1, button = { WheelUp = 1 } } }, mods = 'CTRL', action = act.IncreaseFontSize, alt_screen = 'Any', mouse_reporting = false },
  { event = { Down = { streak = 1, button = { WheelDown = 1 } } }, mods = 'CTRL', action = act.DecreaseFontSize, alt_screen = 'Any', mouse_reporting = false },
  { event = { Up = { streak = 1, button = 'Right' } }, mods = 'NONE', action = act.PasteFrom('Clipboard'), alt_screen = 'Any', mouse_reporting = false },
}

-- ═══════════════════════════════════════════════════════════════════════════════
-- GUI STARTUP — initial window layout
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('gui-startup', function(cmd)
  local ws_name = os.getenv('OVAV_WEZTERM_WORKSPACE') or 'ovav-home'
  wezterm.GLOBAL.ovav_workspace = 'home'  -- ensure workspace tracking starts immediately
  local _, _, _ = wezterm.mux.spawn_window({
    workspace = ws_name,
    cwd = wezterm.home_dir,
  })
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- EVENT HANDLERS
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('ovav-ws-home',   function(w, p) switch_workspace(w, p, 'home') end)
wezterm.on('ovav-ws-system', function(w, p) switch_workspace(w, p, 'system') end)
wezterm.on('ovav-ws-devbrk', function(w, p) switch_workspace(w, p, 'devbrk') end)
wezterm.on('ovav-ws-ovav',   function(w, p) switch_workspace(w, p, 'ovav') end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- + BUTTON — new tab at current pane CWD (Chrome-like behavior)
-- ═══════════════════════════════════════════════════════════════════════════════
return config
