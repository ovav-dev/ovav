# OVAV + Crush — MiniMax Direct Setup (Permanent)
# =============================================================================
# Compatible with Bash and Zsh. For Fish shell, run manually:
#   set -gx ANTHROPIC_API_KEY "sk-cp-I9CVx22_..."
#   set -gx ANTHROPIC_API_ENDPOINT "https://api.minimax.io/anthropic/v1"
#   ovav provider use minimax_direct
# =============================================================================

# Your MiniMax Subscription (NOT hypercredits)
set -gx ANTHROPIC_API_KEY "sk-cp-I9CVx22_-wxYz2tr7P2YFHZ-MeENpjGlTvAgm9MpUsPdZxZWqVLjnEKbx8yl-3Tw7vtLvA_oQLH7igXJQqhR2QsUvHm653c6U_aby9NCc__vU7bnQu1QTF8"
set -gx ANTHROPIC_API_ENDPOINT "https://api.minimax.io/anthropic/v1"

# OVAV Root
set -gx OVAV_ROOT "/home/braka/Systems/OVAV"

# Set OVAV provider to minimax_direct (persisted to ~/.ovav/provider.json)
if type -q ovav
    ovav provider use minimax_direct
end

# Clear hyper credits env var to avoid conflicts
if set -q HYPER_API_KEY
    set -e HYPER_API_KEY
end

echo "MiniMax Direct connected"
echo "   Using YOUR subscription (not hypercredits)"
echo "   Endpoint: api.minimax.io"
echo ""
echo "New command: 'ovav provider use <name>' to switch providers"
