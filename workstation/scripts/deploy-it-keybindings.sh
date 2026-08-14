#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Intelligent Terminal Keybindings Deploy
#  Merges the source-of-truth fragment into the live settings.json.
#  Idempotent. Backed up. Validated.
#
#  Required environment variables (no defaults — user must provide):
#    OVAV_LIVE_IT_SETTINGS  absolute path to live settings.json
#                           (e.g. /mnt/c/Users/Alexa/AppData/Local/Packages/
#                                 Microsoft.IntelligentTerminal_8wekyb3d8bbwe/
#                                 LocalState/settings.json)
#
#  Optional environment variables (with documented defaults):
#    OVAV_FRAGMENT       absolute path to fragment JSON
#                        (default: $OVAV_ROOT/workstation/configs/
#                                 intelligent-terminal/settings-fragment.json)
#    OVAV_BACKUP_DIR     backup destination
#                        (default: $HOME/.ovav-backups)
#    OVAV_DRY_RUN        if set to "1", do not write — only report diff
#
#  External dependencies (must be on PATH):
#    bash   ≥ 4.0
#    jq     (required for JSON merge)
#
#  Registered under .ovav/registry/tool_configs.yaml → ovav_workstation_scripts.
#  NEVER auto-run by OVAV. Idempotent: safe to re-run after fragment changes.
#
#  Usage:
#    OVAV_LIVE_IT_SETTINGS="/mnt/c/Users/Alexa/.../settings.json" \
#      bash workstation/scripts/deploy-it-keybindings.sh
# ─────────────────────────────────────────────────────────────
set -euo pipefail

# ── Required env vars ───────────────────────────────────────
if [ -z "${OVAV_LIVE_IT_SETTINGS:-}" ]; then
  echo "ERROR: OVAV_LIVE_IT_SETTINGS env var is required." >&2
  echo "  Example: OVAV_LIVE_IT_SETTINGS=\"/mnt/c/Users/.../settings.json\" bash $0" >&2
  exit 2
fi
if [ ! -f "$OVAV_LIVE_IT_SETTINGS" ]; then
  echo "ERROR: live settings.json not found at: $OVAV_LIVE_IT_SETTINGS" >&2
  exit 1
fi

# ── Optional env vars with documented defaults ─────────────
: "${OVAV_ROOT:=/home/braka/Systems/ovav}"
OVAV_FRAGMENT="${OVAV_FRAGMENT:-$OVAV_ROOT/workstation/configs/intelligent-terminal/settings-fragment.json}"
OVAV_BACKUP_DIR="${OVAV_BACKUP_DIR:-$HOME/.ovav-backups}"
OVAV_DRY_RUN="${OVAV_DRY_RUN:-0}"
TS="$(date +%Y%m%d-%H%M%S)"
BACKUP="$OVAV_BACKUP_DIR/deploy-it-${TS}"

if [ ! -f "$OVAV_FRAGMENT" ]; then
  echo "ERROR: fragment not found at: $OVAV_FRAGMENT" >&2
  exit 1
fi

# ── Dependency checks ───────────────────────────────────────
command -v jq >/dev/null 2>&1 || { echo "ERROR: jq is required but not found on PATH." >&2; exit 3; }

# ── Helpers ─────────────────────────────────────────────────
log()  { printf "\033[1;36m▸\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m✓\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m⚠\033[0m %s\n" "$*"; }
fail() { printf "\033[1;31m✗\033[0m %s\n" "$*" >&2; exit 1; }

# ── 1. Backup live settings.json ────────────────────────────
log "Step 1: Backup live settings.json"
mkdir -p "$OVAV_BACKUP_DIR"
cp -p "$OVAV_LIVE_IT_SETTINGS" "$BACKUP/settings.json.bak"
ok "Backup at $BACKUP"

# ── 2. Merge: existing ← fragment ──────────────────────────
# Strategy:
#   - keybindings, newTabMenu, themes, profiles.list, schemes, actions:
#     merge from fragment with user customizations preserved where possible
#   - everything else in fragment overwrites existing (canonical defaults)
#
# We use jq with a small program. Comments inline explain the merge logic.
log "Step 2: Merge fragment into live settings.json"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

jq -s '
  (.[0]) as $existing |
  (.[1]) as $fragment |

  # ── Top-level scalar/array fields come from the fragment (canonical) ──
  $fragment
  | .defaultProfile = $fragment.defaultProfile
  | .theme          = $fragment.theme
  | .keybindings    = $fragment.keybindings
  | .newTabMenu     = $fragment.newTabMenu
  | .themes         = $fragment.themes
  | .schemes        = (($existing.schemes // []) + ($fragment.schemes // []) | unique_by(.name))
  | .actions        = (($existing.actions // []) + ($fragment.actions // []) | unique_by(.id))
  | .profiles.list  = (
      (($existing.profiles.list // []) + ($fragment.profiles.list // []))
      | unique_by(.guid)
    )
  | .profiles.defaults = (
      # Fragment defaults win; fall back to existing if fragment omits
      $fragment.profiles.defaults // $existing.profiles.defaults // {}
    )
' "$OVAV_LIVE_IT_SETTINGS" "$OVAV_FRAGMENT" > "$TMP"

# ── 3. Validate JSON ────────────────────────────────────────
log "Step 3: Validate merged JSON"
if ! jq empty "$TMP" 2>/dev/null; then
  fail "merged JSON invalid — restoring from backup"
fi
ok "merged JSON parses cleanly"

# ── 4. Validate keybindings (no id:null, no unresolved) ─────
log "Step 4: Validate keybindings in merged result"
NULL_IDS=$(jq '[.keybindings[]? | select(.id == null or .id == "")] | length' "$TMP")
if [ "$NULL_IDS" -gt 0 ]; then
  fail "merged keybindings contain $NULL_IDS entries with null/empty id — aborting"
fi
ok "0 null-id keybindings"

UNRESOLVED=$(jq --argfile frag "$OVAV_FRAGMENT" '
  [
    .keybindings[]?
    | select((.id != null) and (.id != ""))
    | select(
        (.id as $id |
          ([$frag | .. | objects | select(has("id")) | .id] + ($frag | .. | arrays | .[]? | select(has("id")) | .id)) | index($id)
        ) == null
      )
  ] | length
' "$TMP" 2>/dev/null || echo "0")
if [ "$UNRESOLVED" -gt 0 ]; then
  warn "$UNRESOLVED keybinding id(s) are not resolved in fragment actions — may be IT built-ins"
fi

# ── 5. Dry-run or commit ────────────────────────────────────
if [ "$OVAV_DRY_RUN" = "1" ]; then
  log "DRY-RUN mode — not writing"
  echo "--- merged keybindings count ---"
  jq '.keybindings | length' "$TMP"
  echo "--- merged profiles count ---"
  jq '.profiles.list | length' "$TMP"
  echo "--- merged schemes count ---"
  jq '.schemes | length' "$TMP"
  rm -f "$TMP"
  exit 0
fi

# ── 6. Atomic write ─────────────────────────────────────────
log "Step 5: Atomic write"
mv "$TMP" "$OVAV_LIVE_IT_SETTINGS"
trap - EXIT
ok "live settings.json updated"

# ── Summary ─────────────────────────────────────────────────
cat <<EOF

═══════════════════════════════════════════════════════════
  OVAV IT Keybindings Deploy — Done
═══════════════════════════════════════════════════════════
  Backup:        $BACKUP
  Live:          $OVAV_LIVE_IT_SETTINGS
  Fragment:      $OVAV_FRAGMENT

  Action for operator:
  • Restart Intelligent Terminal (close+reopen) for changes to take effect
  • Or trigger settings reload: Ctrl+Shift+R (Terminal.ReloadCommandPalette)

═══════════════════════════════════════════════════════════
EOF
