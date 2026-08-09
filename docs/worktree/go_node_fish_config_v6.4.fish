# OVAV v6.4 — Go 1.23 + Node 20 LTS fish config
# Instalar: cp docs/worktree/go_node_fish_config_v6.4.fish ~/.config/fish/conf.d/36-ovav-go-node.fish
# Luego: source ~/.config/fish/conf.d/36-ovav-go-node.fish

# ── Go 1.23.0 (user-space, via go1.23.0 download) ──────────────────────────
# Prepend ~/go/bin to PATH so go1.23.0 symlink takes priority over /usr/bin/go
if test -d "$HOME/go/bin"
    fish_add_path --prepend "$HOME/go/bin"
end

# ── Node 20 LTS (user-space, via nvm) ───────────────────────────────────────
set -gx NVM_DIR "$HOME/.nvm"
if test -f "$NVM_DIR/nvm.sh"
    # nvm is a bash script; fish needs bass or manual sourcing
    # Use nvm.fish wrapper or source via bass
    function nvm
        bass source "$NVM_DIR/nvm.sh" --no-use ';' nvm $argv
    end
end

# Auto-use default Node version on shell start
if test -f "$NVM_DIR/nvm.sh"
    set -g default_node_version (cat "$NVM_DIR/alias/default" 2>/dev/null)
    if test -n "$default_node_version"
        set -gx PATH "$NVM_DIR/versions/node/v$default_node_version/bin" $PATH
    end
end
