# OVAV — Runtime Tools (fzf, zoxide, atuin, eza, fd, bat, rg, delta, btm, etc.)
# Canonical source: config/fish/30-ovav-runtime-tools.fish
# Deploy target: ~/.config/fish/conf.d/30-ovav-runtime-tools.fish
# Purpose: Configure interactive tools used across OVAV sessions.
#          All aliases are conditional on tool availability (graceful degradation).

set -gx FZF_DEFAULT_OPTS "--height=42% --layout=reverse --border=rounded --info=inline --cycle"
if command -q fd
    set -gx FZF_DEFAULT_COMMAND "fd --type f --hidden --exclude .git 2>/dev/null"
else if command -q fdfind
    set -gx FZF_DEFAULT_COMMAND "fdfind --type f --hidden --exclude .git 2>/dev/null"
end

if status is-interactive
    if command -q zoxide
        zoxide init fish | source
    end
    if command -q atuin
        atuin init fish | source
    end
    
    # OVAV Keybindings — safe bindings that don't conflict with terminal defaults
    # Only bind \ee (edit command buffer) if not already bound
    bind --query \ee >/dev/null 2>&1; or bind \ee edit_command_buffer
end

if command -q eza
    alias ls='eza --icons=always --group-directories-first --color=always'
    alias ll='eza -lah --icons=always --group-directories-first --git --color=always'
    alias la='eza -a --icons=always --group-directories-first --color=always'
    alias lt='eza --tree --level=2 --icons=always --group-directories-first --git --color=always'
    alias fl='eza -lah --icons=always --group-directories-first --git --color=always'
    alias files='eza -lah --icons=always --group-directories-first --git --color=always'
    alias tree2='eza --tree --level=2 --icons=always --group-directories-first --git --color=always'
else
    alias ls='command ls --color=auto -F'
    alias ll='command ls -lah --color=auto -F'
    alias la='command ls -A --color=auto -F'
end

if command -q fd
    alias ff='fd --hidden --exclude .git'
    alias ffd='fd --type d --hidden --exclude .git'
else if command -q fdfind
    alias ff='fdfind --hidden --exclude .git'
    alias ffd='fdfind --type d --hidden --exclude .git'
end

if command -q batcat
    alias bat='batcat'
end
if command -q bat
    alias preview='bat --style=numbers --color=always'
    alias catp='bat --paging=never --style=plain'
end
if command -q rg
    alias grep='rg'
end
if command -q delta
    set -gx GIT_PAGER delta
    alias gdiff='git diff | delta'
    alias gshow='git show --stat --oneline --decorate'
end
if command -q btm
    alias sysmon='btm'
else if command -q bottom
    alias sysmon='bottom'
end
if command -q dust
    alias usage='dust'
end
if command -q duf
    alias disks='duf'
end
if command -q procs
    alias psx='procs'
end
if command -q hyperfine
    alias bench='hyperfine'
end
if command -q tldr
    alias helpx='tldr'
else if command -q tealdeer
    alias helpx='tealdeer'
end
