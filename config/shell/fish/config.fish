# OVAV root fish config — canonical
# WSL fish lo carga desde ~/.config/fish/conf.d/ovav.fish
# Este archivo documenta la capa base

if status is-interactive
    # zoxide — cd inteligente
    if type -q zoxide
        zoxide init fish | source
    end

    # atuin — history sync
    if type -q atuin
        atuin init fish | source
    end
end

# Paths OVAV
set -gx OVAV_ROOT /home/braka/Systems/OVAV
set -gx PATH $OVAV_ROOT/bin $PATH

# ─── OVAV commit aliases ────────────────────────────────────────────────────
# owci  — commit interactivo con identidad automática (manual=CEO, agent=LEAD)
# owca  — commit -a -S con firma SSH
# owcs  — status corto
alias owcs 'git status --short ---branch'
