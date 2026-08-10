# OVAV + Crush — MiniMax Direct Setup (Permanent)
# =============================================================================
# Run: source clients/crush/config/minimax_direct_env.sh
# Or add to ~/.bashrc or ~/.zshrc
# =============================================================================

# Your MiniMax Subscription (NOT hypercredits)
export ANTHROPIC_API_KEY="sk-cp-I9CVx22_-wxYz2tr7P2YFHZ-MeENpjGlTvAgm9MpUsPdZxZWqVLjnEKbx8yl-3Tw7vtLvA_oQLH7igXJQqhR2QsUvHm653c6U_aby9NCc__vU7bnQu1QTF8"
export ANTHROPIC_API_ENDPOINT="https://api.minimax.io/anthropic/v1"

# OVAV Root
export OVAV_ROOT="/home/braka/Systems/OVAV"

# Hypercredits still available if needed (97 remaining)
# But these env vars make calls go to MiniMax Direct

echo "✅ MiniMax Direct connected"
echo "   Using YOUR subscription (not hypercredits)"
echo "   Endpoint: api.minimax.io"