-- OVAV Theme Auto-Detection for WezTerm
-- Source: .ovav/source/configs/theme/auto.wezterm.lua
-- Detects system theme and applies OVAV dark/light accordingly
-- =============================================================================

local wezterm = require 'wezterm'
local scheme_dark = "Catppuccin Mocha"  -- Fallback dark
local scheme_light = "Catppuccin Latte" -- Fallback light

-- Try to detect GNOME dark mode preference
local function get_system_theme()
  -- Check environment for theme hints
  local scheme = os.getenv("OVAV_COLOR_SCHEME")
  if scheme then
    return scheme
  end

  -- Default to dark for OVAV (eye-friendly for long sessions)
  return "dark"
end

-- Apply theme based on detected preference
local function apply_ovav_theme()
  local theme = get_system_theme()

  if theme == "light" then
    return {
      colors = {
        foreground = "#242424",
        background = "#f5f5f5",
        ansi = { "#242424", "#d4756b", "#7eb77f", "#c4a65a", "#6d9bc3", "#c47d8a", "#5a7d8a", "#d4d4d4" },
        brights = { "#383838", "#d4756b", "#7eb77f", "#c4a65a", "#6d9bc3", "#c47d8a", "#6d9b8e", "#e8e8e8" },
      }
    }
  else
    return {
      colors = {
        foreground = "#d4d4d4",
        background = "#242424",
        ansi = { "#242424", "#d4756b", "#7eb77f", "#c4a65a", "#6d9bc3", "#c47d8a", "#5a7d8a", "#d4d4d4" },
        brights = { "#383838", "#d4756b", "#7eb77f", "#c4a65a", "#6d9bc3", "#c47d8a", "#6d9b8e", "#e8e8e8" },
      }
    }
  end
end

return {
  get_theme = get_system_theme,
  apply_theme = apply_ovav_theme,
}
