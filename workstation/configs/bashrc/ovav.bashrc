# ════════════════════════════════════════════════════════════════════════
#  OVAV INTELLIGENT TERMINAL 2026 — Bash Runtime (ble.sh-aware)
#
#  Architecture:
#    1. PATH (canonical, portable)
#    2. OVAV identity (auto-detected from script location)
#    3. ble.sh FIRST (when present) — line editor of record
#    4. Atuin (history) — bind Ctrl+R via ble-bind if ble.sh, else bind
#    5. fzf (fuzzy) — bind Ctrl+T / Alt-C via ble-bind if ble.sh, else bind
#    6. Starship (prompt) — overrides PS1
#    7. Mise (toolchain) — runtime version manager
#    8. Zoxide (navigation) — replaces cd
#    9. OpenCode (agent CLI) — env vars
#   10. IT shell-integration v3 (OSC 133/9;9/9001) — LAST so PROMPT_COMMAND is finalized
#   11. PROMPT_COMMAND chain — tab title + starship_precmd via IT user-PC hook
#   12. Aliases, clipboard bridge, IT-restart check
#
#  Keybinding ownership (CRITICAL — see issue noted 2026-08-16):
#    - IT (Windows Terminal) MUST NOT capture Ctrl+T/W/C/V — bash owns these.
#    - Atuin owns Ctrl+R.
#    - fzf owns Ctrl+T (file) and Alt+C (cd).
#    - ble.sh owns all other readline keys; IT shell-integration owns OSC.
#
#  Idempotent — safe to source multiple times. Each tool detects its own state.
# ════════════════════════════════════════════════════════════════════════

# ───────────────────────────────────────────────────────────────────────
#  1. PATH (canonical, portable — no hardcoded user paths)
# ───────────────────────────────────────────────────────────────────────
for _ovav_path in "$HOME/.local/bin" "$HOME/.atuin/bin" "$HOME/.opencode/bin" "$HOME/.local/share/mise/shims"; do
    case ":$PATH:" in
        *":$_ovav_path:"*) ;;
        *) export PATH="$PATH:$_ovav_path" ;;
    esac
done
unset _ovav_path

# Clear bash command hash — login shells cache negative lookups for
# binaries not in PATH at first check.
hash -r 2>/dev/null || true

# ───────────────────────────────────────────────────────────────────────
#  2. OVAV IDENTITY (auto-detected, no hardcoded paths)
#     Resolution order:
#       (a) existing OVAV_ROOT if valid
#       (b) parent dir of this file's grandparent
#       (c) ancestor search for known marker (caps.yaml)
# ───────────────────────────────────────────────────────────────────────
_ovav_detect_root() {
    if [ -n "${OVAV_ROOT:-}" ] && [ -d "${OVAV_ROOT}" ] && [ -f "${OVAV_ROOT}/.ovav/plan/caps.yaml" ]; then
        return 0
    fi
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]:-$(dirname "$0")}")" 2>/dev/null && pwd || echo "")"
    if [ -n "$script_dir" ] && [ -f "$script_dir/../../.ovav/plan/caps.yaml" ]; then
        OVAV_ROOT="$(cd "$script_dir/../.." && pwd)"
        export OVAV_ROOT
        return 0
    fi
    if [ -f "/home/braka/Systems/ovav/.ovav/plan/caps.yaml" ]; then
        export OVAV_ROOT="/home/braka/Systems/ovav"
        return 0
    fi
    return 1
}
if ! _ovav_detect_root; then
    echo "OVAV: WARNING — workspace root not detected; tooling may be degraded" 1>&2
fi
unset -f _ovav_detect_root

if [ -n "${OVAV_ROOT:-}" ]; then
    export OVAV_WORKSTATION="${OVAV_ROOT}/workstation"
fi

# ───────────────────────────────────────────────────────────────────────
#  COLOR / TERMINAL
# ───────────────────────────────────────────────────────────────────────
export COLORTERM=truecolor
export TERM=xterm-256color

# ───────────────────────────────────────────────────────────────────────
#  3. BLE.SH FIRST — line editor of record (when present)
#     ble.sh REPLACES readline; subsequent `bind` calls are ignored
#     unless wrapped in ble-bind. We load it FIRST so all keybindings
#     after this point can detect BLE_ATTACHED and adapt.
# ───────────────────────────────────────────────────────────────────────
_BLE_LOADED=""
if [ -f "$HOME/.local/share/blesh/ble.sh" ]; then
    # ble.sh refuses to load in subshells (e.g., bash -c, non-tty stdin).
    # We guard with [ -t 0 ] && [ -t 1 ] to ensure we have a real terminal.
    if [ -t 0 ] && [ -t 1 ]; then
        source "$HOME/.local/share/blesh/ble.sh" 2>/dev/null && _BLE_LOADED="yes"
        [ -f "$HOME/.blerc" ] && source "$HOME/.blerc" 2>/dev/null || true
    fi
fi
export _BLE_LOADED

# ───────────────────────────────────────────────────────────────────────
#  Helper: bind a key to a function, with or without ble.sh
# ───────────────────────────────────────────────────────────────────────
_ovav_bind_key() {
    local key="$1" widget="$2"
    if [ "$_BLE_LOADED" = "yes" ] && type ble-bind >/dev/null 2>&1; then
        # ble.sh active — use ble-bind for widget/function dispatch
        ble-bind -f "$key" "$widget" 2>/dev/null || true
    else
        # Plain bash — use readline bind (only works when no ble.sh)
        bind "\"$widget\"" 2>/dev/null || true
    fi
}

# ───────────────────────────────────────────────────────────────────────
#  4. ATUIN — history search (Ctrl+R)
#     Atuin's `init bash` uses readline `bind` to wire Ctrl+R. When ble.sh
#     is active, those binds are ignored. We disable Atuin's internal bind
#     and wire Ctrl+R ourselves via ble-bind.
# ───────────────────────────────────────────────────────────────────────
if command -v atuin >/dev/null 2>&1; then
    # Disable Atuin's readline bind for Ctrl+R — we handle it below
    export __atuin_bind_ctrl_r=false
    export __atuin_bind_up_arrow=false
    # Load Atuin init (defines __atuin_widget_run, Atuin preexec, etc.)
    eval "$(atuin init bash --disable-up-arrow 2>/dev/null)"
    # Wire Ctrl+R → Atuin search (widget 0 in emacs keymap)
    if type __atuin_widget_run >/dev/null 2>&1; then
        _ovav_bind_key 'C-r' '__atuin_widget_run 0'
    fi
fi

# ───────────────────────────────────────────────────────────────────────
#  5. FZF — fuzzy file picker (Ctrl+T) and cd picker (Alt+C)
#     Note: fzf --bash ALSO binds Ctrl+R to __fzf_history__ by default.
#     Atuin owns Ctrl+R — we explicitly unbind it after fzf init.
# ───────────────────────────────────────────────────────────────────────
if command -v fzf >/dev/null 2>&1; then
    eval "$(fzf --bash 2>/dev/null)"
    export FZF_DEFAULT_COMMAND='fd --type f --hidden --follow --exclude .git 2>/dev/null || find . -type f -not -path "*/\.git/*"'
    export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
    export FZF_ALT_C_COMMAND="$FZF_DEFAULT_COMMAND"
    # Adaptive color palette: prefers terminal ANSI semantic colors
    export FZF_DEFAULT_OPTS="$FZF_DEFAULT_OPTS \
      --color=bg+:#2A3A5C,bg:-1,spinner:#5EEAD4,hl:#F2CC60 \
      --color=fg:#C9D1E0,header:#4A5568,info:#C099FF,pointer:#5EEAD4 \
      --color=marker:#7EE787,fg+:#E8EEF8,prompt:#6EA8FE,hl+:#F5D88A \
      --color=selected-bg:#2A3A5C,border:#4A5568 \
      --no-bold --layout=reverse --height=60%"
    # Wire Ctrl+T → fzf-file-widget (file picker)
    if type fzf-file-widget >/dev/null 2>&1; then
        _ovav_bind_key 'C-t' 'fzf-file-widget'
    fi
    # Wire Alt+C → fzf-cd-widget (cd picker) — only when bash supports
    if type fzf-cd-widget >/dev/null 2>&1; then
        _ovav_bind_key 'M-c' 'fzf-cd-widget'
    fi
fi
unset -f _ovav_bind_key

# ───────────────────────────────────────────────────────────────────────
#  6. STARSHIP — premium minimal prompt (must come BEFORE mise so its
#     precmd hook is available when IT shell-integration captures it)
# ───────────────────────────────────────────────────────────────────────
if [ -x "$HOME/.local/bin/starship" ] || type starship >/dev/null 2>&1; then
    export STARSHIP_CONFIG="${OVAV_WORKSTATION:-$HOME/.config}/configs/starship/starship.toml"
    case "${INTELLIGENT_TERMINAL_THEME:-}" in
        light) export STARSHIP_PALETTE="ovav-day" ;;
        *)     export STARSHIP_PALETTE="ovav-night" ;;
    esac
    eval "$(starship init bash 2>/dev/null)"
fi

# ───────────────────────────────────────────────────────────────────────
#  7. MISE — runtime version manager (node/python/go)
# ───────────────────────────────────────────────────────────────────────
if command -v mise >/dev/null 2>&1; then
    eval "$(mise activate bash 2>/dev/null)"
fi

# ───────────────────────────────────────────────────────────────────────
#  8. ZOXIDE — smart cd (registered as `z` after init)
# ───────────────────────────────────────────────────────────────────────
if command -v zoxide >/dev/null 2>&1; then
    eval "$(zoxide init bash 2>/dev/null)"
fi

# ───────────────────────────────────────────────────────────────────────
#  9. OPENCODE — agent CLI on PATH (canonical Linux)
# ───────────────────────────────────────────────────────────────────────
if [ -x "$HOME/.opencode/bin/opencode" ]; then
    export OPENCODE_CONFIG_DIR="${XDG_CONFIG_DIR:-$HOME/.config}/opencode"
fi

# ───────────────────────────────────────────────────────────────────────
#  10. INTELLIGENT TERMINAL SHELL INTEGRATION (v3 — official)
#     Loaded AFTER mise so it has the final word on PROMPT_COMMAND.
#     It captures __IT_SHELLINTEG_USER_PC and chains it via
#     __it_shellinteg_prompt (the wrapper). This is the ONLY supported
#     way to integrate with IT shell-integration; setting PROMPT_COMMAND
#     directly clobbers IT's wrapper.
# ───────────────────────────────────────────────────────────────────────
if [ -f "$HOME/.intelligent-terminal/shell-integration_v3.sh" ]; then
    . "$HOME/.intelligent-terminal/shell-integration_v3.sh"
fi

# ───────────────────────────────────────────────────────────────────────
#  11. PROMPT CHAIN — tab title + starship_precmd via IT user-PC hook
# ───────────────────────────────────────────────────────────────────────
_ovav_tab_title() {
    local git_branch=""
    if command -v git >/dev/null 2>&1; then
        git_branch="$(git symbolic-ref --short HEAD 2>/dev/null || echo '')"
    fi
    local short_path="${PWD/#$HOME/~}"
    local tab_text="⬢ OVAV · ${short_path}"
    if [[ -n "$git_branch" ]]; then
        tab_text="${tab_text} · ${git_branch}"
    fi
    printf '\033]0;%s\007' "$tab_text"
}
if [ -n "${__IT_SHELLINTEG_USER_PC:-}" ]; then
    export __IT_SHELLINTEG_USER_PC="_ovav_tab_title;starship_precmd"
elif [ -n "${__it_shellinteg_user_pc:-}" ]; then
    export __it_shellinteg_user_pc="_ovav_tab_title;starship_precmd"
else
    PROMPT_COMMAND="_ovav_tab_title;starship_precmd"
fi

# ───────────────────────────────────────────────────────────────────────
#  12. ALIASES, CLIPBOARD BRIDGE, IT RESTART CHECK
# ───────────────────────────────────────────────────────────────────────
# Modern tool aliases (only when modern tool is installed)
if command -v eza &>/dev/null; then
    alias ll='eza -la --icons --git'
    alias lt='eza --tree --level=2 --icons'
    alias la='eza -a --icons'
    alias l='eza --icons'
fi
command -v bat &>/dev/null && alias cat='bat --style=numbers,changes,header --theme=Tokyo-Night'
command -v fd &>/dev/null && alias find='fd'
command -v btop &>/dev/null && alias top='btop'
command -v nvim &>/dev/null && alias vim='nvim'
command -v codium &>/dev/null && alias code='codium'

# Editor — prefer Neovim when available
export EDITOR='nvim'
export VISUAL='nvim'

# Productivity aliases
alias gs='git status'
alias gp='git pull'
alias gco='git checkout'
alias gb='git branch'
alias gl='git log --oneline -20'
alias ov='ovav'
alias ovs='ovav status'
alias ovv='ovav validate'
alias ovd='ovav doctor --quick'
alias ocd='opencode'
alias ocr='opencode --continue'
[ -n "${OVAV_ROOT:-}" ] && alias ovproj='cd $OVAV_ROOT'

# Clipboard bridge (WSL ↔ Windows)
if command -v clip.exe >/dev/null 2>&1; then
    ovclip() {
        if [ $# -gt 0 ]; then
            printf '%s\n' "$*" | clip.exe
        else
            clip.exe
        fi
    }
    ovpaste() {
        powershell.exe -NoProfile -Command 'Get-Clipboard' 2>/dev/null
    }
    ovsessions() {
        opencode session list --format json "$@"
    }
    export -f ovclip ovpaste ovsessions 2>/dev/null
fi

# OVAV runtime binary path
if [ -x "$HOME/.local/bin/ovav" ]; then
    export OVAV_BIN="$HOME/.local/bin/ovav"
fi

# ───────────────────────────────────────────────────────────────────────
#  IT RESTART CHECK — warns if settings.json is newer than IT process.
#  Runs only in interactive shells with a TTY.
# ───────────────────────────────────────────────────────────────────────
_ov_it_check() {
  if [ -t 0 ] && command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -Command '
      $settings = "C:\Users\Alexa\AppData\Local\Packages\Microsoft.IntelligentTerminal_8wekyb3d8bbwe\LocalState\settings.json"
      $proc = Get-Process -Name "WindowsTerminal" -ErrorAction SilentlyContinue | Select-Object -First 1
      if ($proc -and (Test-Path $settings)) {
        $st = (Get-Item $settings).LastWriteTime
        if ($st -gt $proc.StartTime) {
          Write-Host ""
          Write-Host "  ⚠️  OVAV: IT settings updated. Restart IT (Ctrl+Shift+W, reopen) to apply."
          Write-Host ""
        }
      }
    ' 2>/dev/null
  fi
}
case "$-" in *i*) _ov_it_check ;; esac

# ════════════════════════════════════════════════════════════════════════
#  END OVAV bashrc — sourced from ~/.bashrc
# ════════════════════════════════════════════════════════════════════════