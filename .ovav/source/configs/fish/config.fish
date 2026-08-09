# Fish Shell Configuration
# This file is sourced on every fish shell startup (interactive + login)

# Source all conf.d files in alphabetical order
for f in ~/.config/fish/conf.d/*.fish
    source $f
end
