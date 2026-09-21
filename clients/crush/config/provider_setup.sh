# OVAV Provider Switching — Setup Script
# =============================================================================
# Run: source clients/crush/config/provider_setup.sh
# This script sets up environment variables and initializes OVAV provider config
# =============================================================================

# OVAV Root
export OVAV_ROOT="/home/braka/Systems/OVAV"

# =============================================================================
# PROVIDER: openai (YOUR GPT subscription — DEFAULT)
# =============================================================================
# GPT via OpenAI API/browser.
export OPENAI_API_ENDPOINT="${OPENAI_API_ENDPOINT:-https://api.openai.com/v1}"
export OPENAI_MODEL="openai/gpt-5.6-luna"

# =============================================================================
# PROVIDER: minimax-coding-plan (YOUR MiniMax subscription — FALLBACK)
# =============================================================================
# MiniMax M3 is the fallback/small model configured for OVAV.
# export MINIMAX_API_KEY="..."  # Set through the vault/environment, never here
export MINIMAX_MODEL="minimax-coding-plan/MiniMax-M3"

# =============================================================================
# Initialize OVAV Provider Config
# =============================================================================
# This makes OVAV remember your selection
if command -v ovav &> /dev/null; then
    # Set GPT as default provider; MiniMax remains the configured fallback.
    ovav provider use openai 2>/dev/null && echo "✅ OVAV provider configured: openai" || echo "⚠️  Run 'ovav provider use openai' manually"
else
    echo "⚠️  OVAV not in PATH — run 'ovav provider use openai' after installing"
fi

echo ""
echo "🔗 Provider Configuration Complete"
echo "   Active: openai (GPT via API/browser)"
echo "   Fallback/small: minimax-coding-plan/MiniMax-M3"
echo ""
echo "To switch providers:"
echo "  ovav provider list          # See all providers"
echo "  ovav provider use minimax_direct  # Use MiniMax subscription directly"
echo "  ovav provider status       # Show current provider"
