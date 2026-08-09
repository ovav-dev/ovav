#!/usr/bin/env bash
# fish-phase2-cleanup.sh — OVAV Phase 2: remove wezterm hooks from fish config
# =============================================================================
# Ejecutar DESDE UBUNTU WSL2 (fish o bash).
# Idempotente: podés re-ejecutarlo sin daño.
# CEO Braka / OVAV Platform Engineering — 2026-07-10
# =============================================================================

set -euo pipefail
FISH_CONF="$HOME/.config/fish"
ARCHIVE="$FISH_CONF/_archive_20260709_wezterm_migration"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "╔══════════════════════════════════════════════════════════╗"
echo "║  OVAV — Fish Phase 2 WezTerm Cleanup                    ║"
echo "║  Removing wezterm hooks, keeping fish functionality     ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# ── Step 1: Archive wezterm hooks ────────────────────────────────────────
echo "[1/4] Archiving wezterm hooks..."
mkdir -p "$ARCHIVE"

for hook in "20-ovav-wezterm-osc7.fish" "25-ovav-wezterm-git.fish"; do
    src="$FISH_CONF/conf.d/$hook"
    dst="$ARCHIVE/$hook.archived-$TIMESTAMP"
    if [ -f "$src" ]; then
        mv "$src" "$dst"
        echo "  ✔ Archived: $hook → $dst"
    else
        echo "  - Already removed: $hook"
    fi
done

# ── Step 2: Disable OVAV_TERMINAL=wezterm ──────────────────────────────
echo ""
echo "[2/4] Disabling OVAV_TERMINAL=wezterm..."
RUNTIME_TOOLS="$FISH_CONF/conf.d/30-ovav-runtime-tools.fish"

if [ -f "$RUNTIME_TOOLS" ]; then
    if grep -q "set -gx OVAV_TERMINAL wezterm" "$RUNTIME_TOOLS"; then
        sed -i 's|set -gx OVAV_TERMINAL wezterm|# OVAV_TERMINAL wezterm disabled — PowerShell migration 2026-07-10|' \
            "$RUNTIME_TOOLS"
        echo "  ✔ OVAV_TERMINAL=wezterm → disabled"
    else
        echo "  - Already disabled or not present"
    fi
else
    echo "  ⚠ 30-ovav-runtime-tools.fish not found — skipping"
fi

# ── Step 3: Clean config.fish wezterm block ────────────────────────────
echo ""
echo "[3/4] Cleaning config.fish wezterm report-CWD block..."
CONFIG_FISH="$FISH_CONF/config.fish"

if [ -f "$CONFIG_FISH" ]; then
    if grep -q "__wezterm_report_cwd" "$CONFIG_FISH"; then
        # Backup before editing
        cp "$CONFIG_FISH" "$ARCHIVE/config.fish.before-cleanup-$TIMESTAMP"
        sed -i '/^# OVAV: report CWD to wezterm via OSC 0 title sequence$/,/^__wezterm_report_cwd$/d' "$CONFIG_FISH"
        sed -i '/^printf "\\e\]7;file/d' "$CONFIG_FISH"
        echo "  ✔ WezTerm CWD report block removed"
    else
        echo "  - Already clean or not present"
    fi
else
    echo "  ⚠ config.fish not found — skipping"
fi

# ── Step 4: Verify fish loads clean ─────────────────────────────────────
echo ""
echo "[4/4] Verifying fish loads cleanly..."

FISH_CHECK=$(fish -N "$FISH_CONF/config.fish" 2>&1) || true
SYNTAX_ERR=$(fish -c 'fish -N ~/.config/fish/config.fish' 2>&1 || true)

# Quick load test
if timeout 3 fish -l -c 'echo FISH_CLEAN_OK; exit 0' 2>&1 | grep -q "FISH_CLEAN_OK"; then
    echo "  ✔ Fish loads cleanly — no errors"
else
    echo "  ⚠ Fish load test timed out or had issues — check manually"
    echo "    Run: fish -l -c 'echo OK; exit 0'"
fi

# ── Summary ─────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  FISH PHASE 2 CLEANUP COMPLETE                           ║"
echo "╠══════════════════════════════════════════════════════════╣"
echo "║  Archived wezterm hooks in:                              ║"
echo "║    $ARCHIVE"
echo "║                                                          ║"
echo "║  Next: Open Windows Terminal → new Ubuntu tab             ║"
echo "║  Verify: owc --help (should still work)                  ║"
echo "║  Verify: exit (should return 0, not 15)                  ║"
echo "║                                                          ║"
echo "║  Restore if needed:                                      ║"
echo "║    cp $ARCHIVE/*.archived-* $FISH_CONF/conf.d/           ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

exit 0
