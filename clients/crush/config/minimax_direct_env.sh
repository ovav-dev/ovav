# OVAV + Crush — MiniMax Direct Setup (Permanent)
# =============================================================================
# Run: source clients/crush/config/minimax_direct_env.sh
# Or add to ~/.bashrc or ~/.zshrc
#
# DEPRECATED: Use 'ovav provider use minimax_direct' instead.
# This file is kept for backward compatibility.
# =============================================================================

# Your MiniMax Subscription (NOT hypercredits)
export ANTHROPIC_API_KEY="sk-cp-I9CVx22_-wxYz2tr7P2YFHZ-MeENpjGlTvAgm9MpUsPdZxZWqVLjnEKbx8yl-3Tw7vtLvA_oQLH7igXJQqhR2QsUvHm653c6U_aby9NCc__vU7bnQu1QTF8"
export ANTHROPIC_API_ENDPOINT="https://api.minimax.io/anthropic/v1"

# OVAV Root
export OVAV_ROOT="/home/braka/Systems/OVAV"

# Set OVAV provider to minimax_direct (persisted to ~/.ovav/provider.json)
if command -v ovav > /dev/null 2>&1; then
    ovav provider use minimax_direct 2>/dev/null || true
fi

# Clear hyper credits env var to avoid conflicts
unset HYPER_API_KEY 2>/dev/null || true

echo "MiniMax Direct connected"
echo "   Using YOUR subscription (not hypercredits)"
echo "   Endpoint: api.minimax.io"
echo ""
echo "New command: 'ovav provider use <name>' to switch providers"
