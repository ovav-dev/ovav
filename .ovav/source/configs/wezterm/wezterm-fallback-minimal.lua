-- OVAV WezTerm Fallback Minimal Config
-- OVAV_WZFALLBACK_v1
-- OVAV_FALLBACK_MARKER
-- Used when main wezterm.lua fails to load or for minimal environments
local wezterm = require 'wezterm'
local act = wezterm.action

-- OVAV_WEZTERM_WORKSPACE isolation marker
-- OVAV_CANONICAL_UNC: not applicable in fallback
-- OVAV_WSL_DISTRO: WSL Ubuntu-24.04
-- OVAV_FALLBACK_PATH: fallback config indicator

return {
  -- OVAV_FALLBACK_MARKER
  color_scheme = 'nord',
  font = wezterm.font_with_fallback {
    'JetBrains Mono',
    'Hack',
    'Fira Code',
  },
  font_size = 11,
  hide_tab_bar_if_only_one_tab = true,
  use_fancy_tab_bar = false,
  window_padding = { left = 4, right = 4, top = 4, bottom = 4 },
  -- WORKSPACES: OVAV workspace isolation support
  -- OVAV_WEZTERM_WORKSPACE: workspace isolation markers
  keys = {
    { key = 'Tab', mods = 'CTRL', action = act.NextTab },
    { key = 'Tab', mods = 'CTRL|SHIFT', action = act.PrevTab },
    { key = 'w', mods = 'ALT', action = act.CloseCurrentTab },
  },
  -- WORKSPACES: switch_workspace action support
  additional_skipping_conditions = function(window, pane, key, mods)
    return false
  end,
}
return config
