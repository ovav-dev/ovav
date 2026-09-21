# OVAV interactive shell visual integration. Sourcing is silent in scripts.
[[ $- == *i* ]] || return 0

export COLORTERM="${COLORTERM:-truecolor}"

if command -v starship >/dev/null 2>&1; then
  export STARSHIP_CONFIG="${STARSHIP_CONFIG:-$HOME/.config/ovav/starship.toml}"
  eval "$(starship init bash)"
fi
