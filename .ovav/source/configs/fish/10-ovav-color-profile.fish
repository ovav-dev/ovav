# ═══════════════════════════════════════════════════════════════════════════════
# OVAV Fish Color Profile — Minimalist Modern 2026
# This file is sourced by fish at startup to configure syntax colors.
# Time-based day/night detection for fish_color_normal.
# ═══════════════════════════════════════════════════════════════════════════════

# Detect time-based theme
set -l hour (date +%H)
set -l is_day (test "$hour" -ge 7 -a "$hour" -lt 19; and echo "yes"; or echo "no")

if test "$is_day" = "yes"
    # ── DAY MODE ─────────────────────────────────────────────────────────────
    # Background: #f0f2f7 (very light gray)
    # fish_color_normal: dark gray for maximum readability
    set -g fish_color_normal      "#505050"
    set -g fish_color_param       "#505050"
    set -g fish_color_command     "#505050"
    set -g fish_color_keyword     "#505050"
    set -g fish_color_end         "#505050"
    set -g fish_color_statement   "#505050"
    set -g fish_color_option      "#505050"
    set -g fish_color_redirection "#505050"
    set -g fish_color_operator   "#505050"
    set -g fish_color_escape     "#505050"
else
    # ── NIGHT MODE ───────────────────────────────────────────────────────────
    # Background: #0f0f1a (very dark)
    # fish_color_normal: light blue-white for maximum readability
    set -g fish_color_normal      "#c0caf5"
    set -g fish_color_param       "#c0caf5"
    set -g fish_color_command     "#c0caf5"
    set -g fish_color_keyword     "#c0caf5"
    set -g fish_color_end         "#c0caf5"
    set -g fish_color_statement   "#c0caf5"
    set -g fish_color_option      "#c0caf5"
    set -g fish_color_redirection "#c0caf5"
    set -g fish_color_operator   "#c0caf5"
    set -g fish_color_escape     "#c0caf5"
end

# ── COLORS SAME FOR BOTH MODES ───────────────────────────────────────────

# Errors: muted red (safety accent — ONLY for invalid tokens)
set -g fish_color_error          "#e05050"
set -g fish_color_cancel         "#e05050"

# Comments: muted neutral gray, italic
set -g fish_color_comment        "#888899"   --italics

# Strings: subtle teal
set -g fish_color_quote         "#6090a0"

# Autosuggestion: slightly dimmer
set -g fish_color_autosuggestion "#777788"

# Selection / search
set -g fish_color_selection     "#c0caf5"   --background=283457
set -g fish_color_search_match  "#ffbf3f"   --background=283457

# Path validity
set -g fish_color_valid_path     "#6090a0"  --underline
set -g fish_color_history_current "#9090a0"

# Status / host (prompt bar colors — less critical)
set -g fish_color_status         "#e05050"   --bold
set -g fish_color_cwd            "#c0caf5"
set -g fish_color_cwd_root       "#e05050"
set -g fish_color_user           "#6090a0"
set -g fish_color_host           "#6090a0"
set -g fish_color_host_remote    "#e05050"
