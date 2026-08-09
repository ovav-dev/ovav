#!/bin/bash
# OVAV Test Nexus — Pre-push gate
# Blocks push if any OTN check fails

echo "🔒 OVAV Test Nexus — Running pre-push checks..."

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

go run -C go-runtime ./cmd/otn_gate
EXIT=$?

if [ $EXIT -ne 0 ]; then
    echo ""
    echo "🚫 PUSH BLOCKED — OTN gate failed."
    echo "   Fix failures and try again."
    echo "   Evidence: .ovav/evidence/push_reports/"
    exit 1
fi

echo "✅ OTN gate passed. Push allowed."
exit 0
