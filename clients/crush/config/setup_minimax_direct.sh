#!/bin/bash
# OVAV + Crush — MiniMax Direct Connection Setup
# =============================================================================
# This loads your MiniMax subscription directly (NOT hypercredits)
# Run: source this file OR add to your shell profile
#
# DEPRECATED: Use 'ovav provider use minimax_direct' instead.
# =============================================================================

# MiniMax Direct Subscription (your personal subscription)
# Key loaded from OVAV vault: ANTHROPIC_API_KEY
export ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-$(cat ~/.ovav/vault/tokens/ANTHROPIC_API_KEY 2>/dev/null || echo 'NOT_SET')}"
export ANTHROPIC_API_ENDPOINT="https://api.minimax.io/anthropic/v1"

# OVAV root
export OVAV_ROOT="/home/braka/Systems/OVAV"

# Set OVAV provider to minimax_direct
if command -v ovav &> /dev/null; then
    ovav provider use minimax_direct 2>/dev/null || true
fi

# Clear hyper credits to avoid conflicts
unset HYPER_API_KEY 2>/dev/null || true

echo "✅ MiniMax Direct connected"
echo "   Using YOUR subscription (not hypercredits)"
echo "   Endpoint: api.minimax.io"
echo ""
echo "💡 New: Use 'ovav provider use hyper' to switch to CRUSH credits"