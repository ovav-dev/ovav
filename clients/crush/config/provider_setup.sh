# OVAV Provider Switching — Setup Script
# =============================================================================
# Run: source clients/crush/config/provider_setup.sh
# This script sets up environment variables and initializes OVAV provider config
# =============================================================================

# OVAV Root
export OVAV_ROOT="/home/braka/Systems/OVAV"

# =============================================================================
# PROVIDER: minimax_direct (YOUR subscription — RECOMMENDED)
# =============================================================================
# Your MiniMax monthly subscription — direct API access
# Key loaded from OVAV vault: ANTHROPIC_API_KEY
export ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-$(cat ~/.ovav/vault/tokens/ANTHROPIC_API_KEY 2>/dev/null || echo 'NOT_SET')}"
export ANTHROPIC_API_ENDPOINT="https://api.minimax.io/anthropic/v1"

# =============================================================================
# PROVIDER: hyper (CRUSH credits — FALLBACK)
# =============================================================================
# hyper.charm.land credits — use when MiniMax Direct is unavailable
# export HYPER_API_KEY="sk-hyper-..."  # Uncomment when needed

# =============================================================================
# PROVIDER: minimax_hyper (aihubmix via hyper)
# =============================================================================
# Uses MiniMax via aihubmix proxy through hyper credits
# export MINIMAX_API_KEY="sk-cp-..."  # Uncomment when needed

# =============================================================================
# Initialize OVAV Provider Config
# =============================================================================
# This makes OVAV remember your selection
if command -v ovav &> /dev/null; then
    # Set MiniMax Direct as default provider
    ovav provider use minimax_direct 2>/dev/null && echo "✅ OVAV provider configured: minimax_direct" || echo "⚠️  Run 'ovav provider use minimax_direct' manually"
else
    echo "⚠️  OVAV not in PATH — run 'ovav provider use minimax_direct' after installing"
fi

echo ""
echo "🔗 Provider Configuration Complete"
echo "   Active: minimax_direct (MiniMax Direct subscription)"
echo "   Endpoint: https://api.minimax.io/anthropic/v1"
echo ""
echo "To switch providers:"
echo "  ovav provider list          # See all providers"
echo "  ovav provider use hyper    # Use CRUSH credits"
echo "  ovav provider status       # Show current provider"
