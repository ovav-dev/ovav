#!/usr/bin/env bash
# OVAV Red Team R5 — Local Boundary Audit (Codeberg migration fallback)
#
# Replaces the GitHub Actions cron workflow (red-team-cron.yml) which no longer
# executes after migration from GitHub to Codeberg.
#
# Schedule: Weekly, Monday 08:00 UTC-5 (Lima)
#   crontab: 0 8 * * 1 /home/braka/Systems/OVAV/.ovav/worktrees/feature-sprint-1.8/tools/harnesses/run_red_team_audit.sh
#
# Manual:   ./tools/harnesses/run_red_team_audit.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORTS_DIR="$REPO_ROOT/.ovav/red_team/reports"

echo "🔴 OVAV Red Team R5 — Local Boundary Audit"
echo "   Repo: $REPO_ROOT"
echo ""

# ── Step 1: Build Go validate CLI ──────────────────────────────────────
echo "📦 Building validate CLI..."
cd "$REPO_ROOT/go-runtime"
go build -o "$REPO_ROOT/validate" ./internal/validators/cmd/validate/ 2>&1 || {
    echo "⚠️  Go build failed — trying pre-built binary..."
    if [ ! -f "$REPO_ROOT/validate" ]; then
        echo "❌ No validate binary available. Audit aborted."
        exit 1
    fi
}
echo "✅ validate CLI ready"

# ── Step 2: Run boundary audit ─────────────────────────────────────────
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
TIMESTAMP_FILE=$(date -u +"%Y-%m-%dT%H%M%SZ")

echo ""
echo "🔍 Running R5 boundary audit..."
AUDIT_OUTPUT=$("$REPO_ROOT/validate" --id red_team_audit --root "$REPO_ROOT" 2>&1) || true

# ── Step 3: Determine pass/fail ────────────────────────────────────────
if echo "$AUDIT_OUTPUT" | grep -qE "\bFAIL\b"; then
    STATUS="fail"
    ISSUE_COUNT=$(echo "$AUDIT_OUTPUT" | grep -c "    - " || echo "0")
    SEVERITY="CRITICAL"
elif echo "$AUDIT_OUTPUT" | grep -qE "(✅|pass)"; then
    STATUS="pass"
    ISSUE_COUNT=0
    SEVERITY="OK"
else
    STATUS="unknown"
    ISSUE_COUNT=0
    SEVERITY="UNKNOWN"
fi

# ── Step 4: Save report ────────────────────────────────────────────────
mkdir -p "$REPORTS_DIR"
REPORT_FILE="$REPORTS_DIR/audit-${TIMESTAMP_FILE}.json"

cat > "$REPORT_FILE" << EOF
{
  "audit_type": "R5_boundary_audit",
  "timestamp": "$TIMESTAMP",
  "status": "$STATUS",
  "issue_count": $ISSUE_COUNT,
  "trigger": "local_cron",
  "branch": "$(cd "$REPO_ROOT" && git branch --show-current)",
  "commit": "$(cd "$REPO_ROOT" && git rev-parse HEAD)",
  "evaluator": "Kenji Tanaka (Adversarial Intelligence)",
  "severity": "$SEVERITY"
}
EOF

# ── Step 5: Report summary ─────────────────────────────────────────────
echo ""
echo "══════════════════════════════════════════════"
if [ "$STATUS" = "pass" ]; then
    echo "  ✅ Red Team R5 — PASS"
else
    echo "  🔴 Red Team R5 — FAIL ($ISSUE_COUNT issues)"
fi
echo "  Report: $REPORT_FILE"
echo "══════════════════════════════════════════════"

# ── Step 6: Auto-commit report ─────────────────────────────────────────
cd "$REPO_ROOT"
git add .ovav/red_team/reports/ 2>/dev/null || true
if git diff --staged --quiet 2>/dev/null; then
    echo "📄 No report changes to commit"
else
    git commit -m "docs(red-team): automated R5 audit report — $STATUS" 2>/dev/null || {
        echo "⚠️  Could not auto-commit report (may need manual commit)"
    }
fi

# ── Step 7: Alert on failure ───────────────────────────────────────────
if [ "$STATUS" = "fail" ]; then
    echo ""
    echo "🚨 RED TEAM BOUNDARY AUDIT FAILED"
    echo "   Issues: $ISSUE_COUNT"
    echo "   Report: $REPORT_FILE"
    echo "   Action: Kenji Tanaka (Adversarial Intelligence) must investigate."
    exit 1
fi

exit 0
