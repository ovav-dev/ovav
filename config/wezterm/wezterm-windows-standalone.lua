-- ╔══════════════════════════════════════════════════════════════════════════════╗
-- ║               OVAV WEZTERM WINDOWS — Standalone Nightly                   ║
-- ║            config/wezterm/wezterm-windows-standalone.lua · v2.0.0          ║
-- ║                                                                            ║
-- ║  Para WezTerm nightly Windows 11 (20260805+).                             ║
-- ║  Tema OVAV v2.0.0 — paleta canónica desde .ovav/visual/theme/theme.yaml   ║
-- ║  Workspace isolation, tab bar con acentos OVAV, status bar minimal.       ║
-- ║                                                                            ║
-- ║  Fuente canónica: config/wezterm/wezterm.lua                               ║
-- ║  Dependencias: Nerd Font (CascadiaCode ya instalado en Windows)           ║
-- ╚══════════════════════════════════════════════════════════════════════════════╝

local wezterm = require 'wezterm'
local act = wezterm.action
local config = wezterm.config_builder and wezterm.config_builder() or {}

-- ═══════════════════════════════════════════════════════════════════════════════
-- PALETA OVAV v2.0.0 — eye-friendly, bajo contraste, sesiones largas
-- ═══════════════════════════════════════════════════════════════════════════════
local P = {
  -- Superficie — modo oscuro
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

  -- Workspace dim backgrounds
  ws_home_dim   = '#1e2a1e',
  ws_system_dim = '#25281e',
  ws_dev_dim    = '#1e2528',
  ws_ovav_dim   = '#281e22',

  -- Semánticos OVAV
  success  = '#7eb77f',
  error    = '#d4756b',
  warning  = '#d4a85c',
  info     = '#6d9bc3',

  -- ANSI — paleta completa OVAV
  black    = '#242424',
  red      = '#d4756b',
  green    = '#7eb77f',
  yellow   = '#c4a65a',
  blue     = '#6d9bc3',
  magenta  = '#c47d8a',
  cyan     = '#5a7d8a',
  white    = '#d4d4d4',

  -- Bright ANSI
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
-- WORKSPACES — WSL2 paths (no Nerd Font icons, ASCII fallback)
-- ═══════════════════════════════════════════════════════════════════════════════
local WORKSPACES = {
  home = {
    key = '1', label = 'HOME', icon = '[H]',
    accent = '#7eb77f', accent_dim = '#1e2a1e',
    cwd = '/home/braka',
    prefixes = { '/home/braka' },
  },
  system = {
    key = '2', label = 'SYS', icon = '[S]',
    accent = '#d4a85c', accent_dim = '#25281e',
    cwd = '/home/braka/.config',
    prefixes = { '/home/braka/.config' },
  },
  ovav = {
    key = '3', label = 'OVAV', icon = '[O]',
    accent = '#c47d8a', accent_dim = '#281e22',
    cwd = '/home/braka/Systems/OVAV',
    prefixes = { '/home/braka/Systems/OVAV' },
  },
}

-- ═══════════════════════════════════════════════════════════════════════════════
-- HELPERS
-- ═══════════════════════════════════════════════════════════════════════════════
local function starts_with(s, prefix)
  return s and prefix and s:sub(1, #prefix) == prefix
end

local function detect_workspace(cwd)
  if not cwd or cwd == '' then return 'home' end
  for name, ws in pairs(WORKSPACES) do
    if ws.prefixes then
      for _, p in ipairs(ws.prefixes) do
        if starts_with(cwd, p) then return name end
      end
    end
  end
  return 'home'
end

local function abbrev_path(cwd)
  if not cwd then return '' end
  -- WSL/Linux path normalization
  if cwd == '/home/braka' then return '~' end
  if starts_with(cwd, '/home/braka') then
    return '~' .. cwd:sub(#'/home/braka' + 1)
  end
  return cwd
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- WORKSPACE SWITCHING
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
  },
}

wezterm.GLOBAL.ovav_workspace = wezterm.GLOBAL.ovav_workspace or 'home'

local function apply_ws_colors(window, ws_name)
  local wsc = WS_COLORS[ws_name] or WS_COLORS['home']
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
      },
    })
  end)
end

local function switch_workspace(window, pane, ws_name)
  local ws = WORKSPACES[ws_name]
  if not ws then return end
  wezterm.GLOBAL.ovav_workspace = ws_name
  apply_ws_colors(window, ws_name)
  -- Spawn a new tab in WSL domain with the workspace's CWD
  -- Use cd in the command since cwd param may not work for WSL domains
  window:perform_action(act.SpawnCommandInNewTab({
    domain = { DomainName = 'WSL:Ubuntu-24.04' },
    args = { 'fish', '-c', 'cd ' .. ws.cwd .. ' && fish -l' },
  }), pane)
end

-- ═══════════════════════════════════════════════════════════════════════════════
-- TAB BAR — OVAV styled with workspace accent
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('format-tab-title', function(tab, tabs, panes, _, hover, _)
  local tab_id = tostring(tab.tab_id)
  if not wezterm.GLOBAL['ovav_tab_ws_' .. tab_id] then
    wezterm.GLOBAL['ovav_tab_ws_' .. tab_id] = wezterm.GLOBAL.ovav_workspace or 'home'
  end
  local ws_name = wezterm.GLOBAL['ovav_tab_ws_' .. tab_id] or 'home'
  local ws = WORKSPACES[ws_name] or WORKSPACES['home']
  local accent = ws.accent or P.fg

  -- Count tabs in workspace
  local tab_index = 1
  local same_ws_count = 0
  if tabs then
    for _, t in ipairs(tabs) do
      local tws = wezterm.GLOBAL['ovav_tab_ws_' .. tostring(t.tab_id)] or wezterm.GLOBAL.ovav_workspace or 'home'
      if tws == ws_name then
        same_ws_count = same_ws_count + 1
        if t.tab_id == tab.tab_id then tab_index = same_ws_count end
      end
    end
  end

  local label = ws.label
  if same_ws_count > 1 then label = ws.label .. ' ' .. tab_index end

  if tab.is_active then
    return {
      { Foreground = { Color = accent } }, { Text = ws.icon .. ' ' },
      { Foreground = { Color = P.fg_bright } }, { Text = label .. ' ' },
    }
  else
    return {
      { Foreground = { Color = P.fg_muted } }, { Text = ws.icon .. ' ' .. label .. ' ' },
    }
  end
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- STATUS BAR — minimal, workspace + time
-- ═══════════════════════════════════════════════════════════════════════════════
wezterm.on('update-status', function(window, pane)
  local ws_name = wezterm.GLOBAL.ovav_workspace or 'home'
  local ws = WORKSPACES[ws_name] or WORKSPACES['home']

  local right = {
    { Foreground = { Color = P.surface1 } }, { Text = '' },
    { Background = { Color = P.surface1 } },
    { Foreground = { Color = ws.accent } }, { Text = ws.icon .. ' ' },
    { Foreground = { Color = P.fg } }, { Text = ws.label .. '  ' },
    { Foreground = { Color = P.fg_dim } }, { Text = wezterm.strftime('%H:%M') .. '  ' },
    { Background = { Color = P.bg } },
    { Foreground = { Color = P.surface1 } }, { Text = '' },
  }
  window:set_right_status(wezterm.format(right))
  window:set_left_status('')
end)

-- ═══════════════════════════════════════════════════════════════════════════════
-- COLORES — paleta OVAV completa
-- ═══════════════════════════════════════════════════════════════════════════════
config.colors = {
  foreground = P.fg,
  background = P.bg,
  cursor_bg = P.green,
  cursor_fg = P.bg,
  cursor_border = P.green,
  selection_fg = P.white,
  selection_bg = P.surface2,
  scrollbar_thumb = P.ws_home_dim,
  split = P.ws_home_dim,
  visual_bell = P.green,
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
-- APARIENCIA — Windows optimized
-- ═══════════════════════════════════════════════════════════════════════════════
config.font = wezterm.font_with_fallback({
  'Cascadia Code', 'Cascadia Mono', 'JetBrains Mono', 'Consolas',
  'Symbols Nerd Font Mono', 'Noto Color Emoji',
})
config.font_size = 11.0
config.line_height = 1.20
config.bold_brightens_ansi_colors = true
config.window_background_opacity = 0.97
config.win32_system_backdrop = 'Disable'
config.window_padding = { left = 8, right = 8, top = 6, bottom = 6 }
config.window_decorations = 'RESIZE'
config.inactive_pane_hsb = { saturation = 1.0, brightness = 1.0 }
config.audible_bell = 'Disabled'
config.exit_behavior = 'CloseOnCleanExit'
config.window_close_confirmation = 'NeverPrompt'

-- ═══════════════════════════════════════════════════════════════════════════════
-- WSL2 DOMAIN — connect WezTerm Windows → WSL2 Ubuntu-24.04 fish shell
-- ═══════════════════════════════════════════════════════════════════════════════
-- WSL2 domain configuration
config.wsl_domains = {
  {
    name = 'WSL:Ubuntu-24.04',
    distribution = 'Ubuntu-24.04',
    username = 'braka',
    default_prog = { 'fish', '-l' },
  },
}

-- Use WSL domain by default for new tabs/pane spawns
config.default_domain = 'WSL:Ubuntu-24.04'

-- gui-startup: wezterm on Windows always starts with cmd.exe pane.
-- Workaround: spawn WSL pane, then user can close the cmd.exe pane manually.
-- New tabs (Ctrl+T) will use the WSL domain by default.



-- ═══════════════════════════════════════════════════════════════════════════════
-- RENDER
-- ═══════════════════════════════════════════════════════════════════════════════
config.front_end = 'WebGpu'
config.animation_fps = 2
config.max_fps = 120
config.cursor_blink_rate = 0
config.text_blink_rate = 0
config.scrollback_lines = 10000
config.default_cursor_style = 'SteadyBar'
config.cursor_thickness = '1.8px'
config.adjust_window_size_when_changing_font_size = false

-- ═══════════════════════════════════════════════════════════════════════════════
-- TAB BAR
-- ═══════════════════════════════════════════════════════════════════════════════
config.enable_tab_bar = true
config.use_fancy_tab_bar = true
config.show_tabs_in_tab_bar = true
config.show_new_tab_button_in_tab_bar = true
config.show_tab_index_in_tab_bar = false
config.tab_bar_at_bottom = false
config.tab_max_width = 32
config.hide_tab_bar_if_only_one_tab = false

-- ═══════════════════════════════════════════════════════════════════════════════
-- HYPERLINKS — OVAV GitHub references
-- ═══════════════════════════════════════════════════════════════════════════════
config.hyperlink_rules = wezterm.default_hyperlink_rules()
table.insert(config.hyperlink_rules, { regex = [[\b#(\d+)\b]], format = 'https://github.com/ovav-dev/ovav-systems/issues/$1', highlight = 1 })
table.insert(config.hyperlink_rules, { regex = [[\b!(\d+)\b]], format = 'https://github.com/ovav-dev/ovav-systems/pull/$1', highlight = 1 })

-- ═══════════════════════════════════════════════════════════════════════════════
-- KEY BINDINGS — Windows optimized (no WSL)
-- ═══════════════════════════════════════════════════════════════════════════════
config.keys = {
  -- Workspace switching with Alt+1-4
  { key = '1', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    switch_workspace(w, p, 'home')
  end) },
  { key = '2', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    switch_workspace(w, p, 'system')
  end) },
  { key = '3', mods = 'ALT', action = wezterm.action_callback(function(w, p)
    switch_workspace(w, p, 'ovav')
  end) },

  -- Pane navigation
  { key = 'h', mods = 'ALT', action = act.ActivatePaneDirection 'Left' },
  { key = 'j', mods = 'ALT', action = act.ActivatePaneDirection 'Down' },
  { key = 'k', mods = 'ALT', action = act.ActivatePaneDirection 'Up' },
  { key = 'l', mods = 'ALT', action = act.ActivatePaneDirection 'Right' },

  -- Pane resize
  { key = 'h', mods = 'ALT|SHIFT', action = act.AdjustPaneSize { 'Left', 3 } },
  { key = 'j', mods = 'ALT|SHIFT', action = act.AdjustPaneSize { 'Down', 3 } },
  { key = 'k', mods = 'ALT|SHIFT', action = act.AdjustPaneSize { 'Up', 3 } },
  { key = 'l', mods = 'ALT|SHIFT', action = act.AdjustPaneSize { 'Right', 3 } },

  -- Splits
  { key = 't', mods = 'ALT', action = act.SplitHorizontal { domain = 'CurrentPaneDomain' } },
  { key = 'D', mods = 'ALT|SHIFT', action = act.SplitVertical { domain = 'CurrentPaneDomain' } },

  -- Tab management
  { key = 't', mods = 'CTRL|SHIFT', action = act.SpawnTab('DefaultDomain') },
  { key = 'w', mods = 'CTRL|SHIFT', action = act.CloseCurrentTab { confirm = true } },
  { key = 'Tab', mods = 'CTRL', action = act.ActivateTabRelative(1) },
  { key = 'Tab', mods = 'CTRL|SHIFT', action = act.ActivateTabRelative(-1) },
  { key = '1', mods = 'CTRL|SHIFT', action = act.ActivateTab(0) },
  { key = '2', mods = 'CTRL|SHIFT', action = act.ActivateTab(1) },
  { key = '3', mods = 'CTRL|SHIFT', action = act.ActivateTab(2) },
  { key = '4', mods = 'CTRL|SHIFT', action = act.ActivateTab(3) },

  -- Copy mode
  { key = 'x', mods = 'CTRL|SHIFT', action = act.ActivateCopyMode },

  -- Reload
  { key = 'r', mods = 'CTRL|SHIFT', action = act.ReloadConfiguration },
}

return config
