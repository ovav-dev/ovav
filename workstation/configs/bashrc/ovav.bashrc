# OVAV INTELLIGENT TERMINAL 2026
# Bash runtime additions — sourced from ~/.bashrc
# This snippet is appended to ~/.bashrc by workstation/install.sh

# ─────────────────────────────────────────────────────────────
#  OVAV IDENTITY
# ─────────────────────────────────────────────────────────────
export OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
export OVAV_WORKSTATION="${OVAV_ROOT}/workstation"

# ─────────────────────────────────────────────────────────────
#  PATH (canonical Linux install per rule #11, #33)
#  Includes /usr/local/bin, OpenCode, Atuin, OVAV locals
# ─────────────────────────────────────────────────────────────
case ":$PATH:" in
  *":/usr/local/bin:"*) ;;
  *) export PATH="/usr/local/bin:/home/braka/.opencode/bin:/home/braka/.atuin/bin:$HOME/.local/bin:$PATH" ;;
esac

# ─────────────────────────────────────────────────────────────
#  COLOR / TERMINAL
# ─────────────────────────────────────────────────────────────
export COLORTERM=truecolor
export TERM=xterm-256color

# ─────────────────────────────────────────────────────────────
#  ATUIN — history (NO pty-proxy per rule #15)
#  Critical: must be in PATH BEFORE checking command -v
#  (Atuin ships its own binary at ~/.atuin/bin, NOT in /usr/local/bin)
# ─────────────────────────────────────────────────────────────
if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin init bash --disable-up-arrow 2>/dev/null)"
fi

# ─────────────────────────────────────────────────────────────
#  ZOXIDE — navigation
# ─────────────────────────────────────────────────────────────
if command -v zoxide >/dev/null 2>&1; then
  eval "$(zoxide init bash 2>/dev/null)"
fi

# ─────────────────────────────────────────────────────────────
#  FZF — fuzzy primitives (files, processes, branches)
#  Ctrl-R is OWNED by Atuin. fzf provides Ctrl-T, Alt-C, and
#  custom invocations only.
# ─────────────────────────────────────────────────────────────
if command -v fzf >/dev/null 2>&1; then
  eval "$(fzf --bash 2>/dev/null)"
  # Explicit ownership: fzf owns Ctrl-T and Alt-C, NOT Ctrl-R
  # (Atuin owns Ctrl-R via 'atuin init bash' above)
  export FZF_DEFAULT_COMMAND='fd --type f --hidden --follow --exclude .git 2>/dev/null || find . -type f -not -path "*/\.git/*"'
  export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
  export FZF_ALT_C_COMMAND='fd --type d --hidden --follow --exclude .git 2>/dev/null || find . -type d -not -path "*/\.git/*"'
  # Adaptive color palette: prefers terminal ANSI semantic colors
  export FZF_DEFAULT_OPTS="$FZF_DEFAULT_OPTS \
    --color=bg+:#2A3A5C,bg:-1,spinner:#5EEAD4,hl:#F2CC60 \
    --color=fg:#C9D1E0,header:#4A5568,info:#C099FF,pointer:#5EEAD4 \
    --color=marker:#7EE787,fg+:#E8EEF8,prompt:#6EA8FE,hl+:#F5D88A \
    --color=selected-bg:#2A3A5C,border:#4A5568 \
    --no-bold --layout=reverse --height=60%"
fi

# ─────────────────────────────────────────────────────────────
#  STARSHIP — premium minimal prompt
# ─────────────────────────────────────────────────────────────
if command -v starship >/dev/null 2>&1; then
  export STARSHIP_CONFIG="${OVAV_ROOT}/workstation/configs/starship/starship.toml"
  # Theme sync — follow Intelligent Terminal theme if exposed
  case "${INTELLIGENT_TERMINAL_THEME:-}" in
    light) export STARSHIP_PALETTE="ovav-day" ;;
    *)     export STARSHIP_PALETTE="ovav-night" ;;
  esac
  eval "$(starship init bash 2>/dev/null)"
fi

# ─────────────────────────────────────────────────────────────
#  OPENCODE — agent CLI on PATH (canonical Linux)
# ─────────────────────────────────────────────────────────────
if [ -x "$HOME/.opencode/bin/opencode" ]; then
  export OPENCODE_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/opencode"
fi

# ─────────────────────────────────────────────────────────────
#  MISE — runtime version manager (node/python/go)
# ─────────────────────────────────────────────────────────────
if command -v mise >/dev/null 2>&1; then
  eval "$(mise activate bash 2>/dev/null)"
fi

# ─────────────────────────────────────────────────────────────
#  OVAV RUNTIME (Go CLI)
# ─────────────────────────────────────────────────────────────
if [ -x "$HOME/.local/bin/ovav" ]; then
  export OVAV_BIN="$HOME/.local/bin/ovav"
fi

# ─────────────────────────────────────────────────────────────
#  ALIASES — productivity (non-critical, no overrides)
# ─────────────────────────────────────────────────────────────
alias ll='ls -lah --color=auto'
alias la='ls -A --color=auto'
alias l='ls -CF --color=auto'
alias gs='git status'
alias gp='git pull'
alias gco='git checkout'
alias gb='git branch'
alias gl='git log --oneline -20'
alias ov='ovav'
alias ovs='ovav status'
alias ovv='ovav validate'
alias ocd='opencode'
alias ocr='opencode --continue'

# OVAV-aware: jump to project
alias ovproj='cd $OVAV_ROOT'

# ─────────────────────────────────────────────────────────────
#  INTELLIGENT TERMINAL SHELL INTEGRATION (v3 — official)
#  Sourced after all prompt hooks so OSC 133 has final word.
# ─────────────────────────────────────────────────────────────
if [ -f "$HOME/.intelligent-terminal/shell-integration_v3.sh" ]; then
  . "$HOME/.intelligent-terminal/shell-integration_v3.sh"
fi

# ─────────────────────────────────────────────────────────────
#  CLEANUP — REMOVED legacy MiMoCode artifact
#  Was: export MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS=1
#  Reason: MiMoCode is not OVAV. Dangerous bypass does not apply.
# ─────────────────────────────────────────────────────────────