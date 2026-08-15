#!/usr/bin/env bash
# Bashrc surgical fix — uses LINE NUMBERS from grep, not fragile regex
set -euo pipefail

BASHRC="$HOME/.bashrc"
TS=$(date +%Y%m%d-%H%M%S)
BACKUP="$HOME/.ovav-backups/manual-${TS}"
mkdir -p "$BACKUP"
cp -p "$BASHRC" "$BACKUP/bashrc-pre-fix.bak"

# Find the OVAV block markers as line numbers
START=$(grep -n "^# ── OVAV WORKSTATION 2026 ──" "$BASHRC" | head -1 | cut -d: -f1)
END=$(grep -n "^# ── END OVAV WORKSTATION ──" "$BASHRC" | head -1 | cut -d: -f1)

if [ -z "$START" ] || [ -z "$END" ]; then
  echo "ERROR: OVAV markers not found" >&2
  exit 1
fi

# Everything between 1 and START-1 = KEEP (but clean legacy inits)
# Lines START..END = KEEP OVAV block (but clean MIMOCODE refs)
# Lines END+1..end = KEEP

# Step 1: Extract pre-OVAV, OVAV, post-OVAV
PRE=$(sed -n "1,$((START-1))p" "$BASHRC")
OVAV=$(sed -n "${START},${END}p" "$BASHRC")
POST=$(sed -n "$((END+1)),\$p" "$BASHRC")

# Step 2: Clean PRE — remove legacy init blocks by anchor markers
clean_pre() {
  awk '
    BEGIN { skip = 0; depth = 0 }
    # Anchor 1: legacy Starship prompt section
    /^# Starship prompt$/ { skip = 1; next }
    skip && /^# Optional aliases:/ { skip = 0; print "    # [REMOVED legacy Starship init — moved to OVAV block]"; next }
    skip { next }
    # Anchor 2: legacy Atuin env helper (single line, just skip)
    /^# Atuin integration is installed only/ { skip = 1; next }
    /^if command -v atuin/ && skip { skip = 0; next }
    /^fi$/ && skip { skip = 0; next }
    skip { next }
    # Anchor 3: legacy zoxide init
    /^if command -v zoxide/ { skip = 1; next }
    /^fi$/ && skip { skip = 0; next }
    skip { next }
    # Anchor 4: legacy fzf --bash
    /^if command -v fzf/ { skip = 1; next }
    /^fi$/ && skip { skip = 0; next }
    skip { next }
    # Anchor 5: MIMOCODE total freedom line + its export
    /^# OVAV SYSTEMS — MIMOCODE TOTAL FREEDOM$/ { skip = 1; next }
    /^export MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS=1$/ { skip = 0; next }
    /^MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS/ { next }
    skip { next }
    { print }
  '
}

clean_ovav() {
  # Remove any MIMOCODE references from OVAV block too
  grep -v "MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS"
}

PRE_CLEAN=$(echo "$PRE" | clean_pre)
OVAV_CLEAN=$(echo "$OVAV" | clean_ovav)

# Reassemble
{
  echo "$PRE_CLEAN"
  echo "$OVAV_CLEAN"
  echo "$POST"
} > "$BASHRC.new"

mv "$BASHRC.new" "$BASHRC"

# Report
echo "── bashrc fixed ──"
echo "  backup: $BACKUP/bashrc-pre-fix.bak"
echo ""
printf "  %-30s %s\n" "Total lines:" "$(wc -l < $BASHRC)"
printf "  %-30s %s\n" "Atuin init:" "$(grep -c 'atuin init' $BASHRC)"
printf "  %-30s %s\n" "Starship init:" "$(grep -c 'starship init' $BASHRC)"
printf "  %-30s %s\n" "fzf --bash:" "$(grep -c 'fzf --bash' $BASHRC)"
printf "  %-30s %s\n" "zoxide init:" "$(grep -c 'zoxide init' $BASHRC)"
printf "  %-30s %s\n" "MIMOCODE refs:" "$(grep -c MIMOCODE $BASHRC || true)"
printf "  %-30s %s\n" "IT shell integration:" "$(grep -c 'intelligent-terminal/shell-integration' $BASHRC)"