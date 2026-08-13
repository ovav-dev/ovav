-- OVAV adaptive Day/Night palette. Projection requires an installed WezTerm.
local wezterm = require 'wezterm'

local palettes = {
  light = {
    foreground = '#202124',
    background = '#f7f4ee',
    cursor_bg = '#2f685c',
    cursor_fg = '#f7f4ee',
    selection_bg = '#cbc3b7',
    selection_fg = '#202124',
    ansi = { '#202124', '#97392f', '#356b39', '#785500', '#285d82', '#8b4050', '#2f685c', '#555b63' },
    brights = { '#4f545c', '#a64238', '#447b47', '#886318', '#32658d', '#984b5a', '#397568', '#202124' },
  },
  dark = {
    foreground = '#d9dadd',
    background = '#202124',
    cursor_bg = '#72b8aa',
    cursor_fg = '#202124',
    selection_bg = '#42474f',
    selection_fg = '#f7f4ee',
    ansi = { '#202124', '#e17f74', '#86c78a', '#e0b86b', '#79aee0', '#df8997', '#72b8aa', '#d9dadd' },
    brights = { '#979ba3', '#ef9a91', '#9bd39e', '#eac87f', '#91bce4', '#e8a1ac', '#8dccbf', '#f7f4ee' },
  },
}

local function get_system_theme()
  local override = os.getenv 'OVAV_COLOR_SCHEME'
  if override == 'light' or override == 'dark' then
    return override
  end

  local appearance = wezterm.gui.get_appearance()
  if appearance and appearance:find 'Light' then
    return 'light'
  end
  return 'dark'
end

local function apply_ovav_theme()
  return { colors = palettes[get_system_theme()] }
end

return {
  get_theme = get_system_theme,
  apply_theme = apply_ovav_theme,
  palettes = palettes,
}
