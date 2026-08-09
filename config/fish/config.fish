# Fish Shell Configuration
# This file is sourced on every fish shell startup (interactive + login)

# Source all conf.d files in alphabetical order
for f in ~/.config/fish/conf.d/*.fish
    source $f
end

# Source OVAV aliases if they exist
if test -f ~/.config/fish/aliases.fish
    source ~/.config/fish/aliases.fish
end

# Source OVAV commands if they exist (cross-terminal shortcuts)
if test -f ~/.config/fish/ovav-commands.fish
    source ~/.config/fish/ovav-commands.fish
end
