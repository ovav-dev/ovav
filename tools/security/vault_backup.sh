#!/usr/bin/env bash
# =============================================================================
# OVAV Vault Backup — encrypted vault snapshot with integrity verification
# Usage: bash tools/security/vault_backup.sh [--check] [--backup-dir DIR]
#
# Steps:
#   1. Scan vault with `ovav vault scan`
#   2. Encrypt assets with `ovav vault encrypt --key .ovav/vault/master.key`
#   3. Copy .ovav/vault/*.enc to backup_dir
#   4. Create timestamped tarball: backups/vault_YYYYMMDD_HHMMSS.tar.gz
#   5. Verify integrity (extract + confirm .enc files present)
#
# Flags:
#   --check          Dry-run: verify vault state, do NOT write backups
#   --backup-dir DIR Override default backup directory (default: .ovav/vault/backups/)
# =============================================================================
set -euo pipefail

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# ── Resolve repo root (script lives in tools/security/) ──────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

# ── Defaults ──────────────────────────────────────────────────────────────────
VAULT_DIR=".ovav/vault"
MASTER_KEY="$VAULT_DIR/master.key"
BACKUP_DIR="$VAULT_DIR/backups"
CHECK_MODE=false
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_NAME="vault_${TIMESTAMP}"

# ── Parse arguments ───────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --check)
            CHECK_MODE=true
            shift
            ;;
        --backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [--check] [--backup-dir DIR]"
            echo ""
            echo "  --check          Dry-run: verify vault state only, no writes"
            echo "  --backup-dir DIR Custom backup directory (default: .ovav/vault/backups/)"
            exit 0
            ;;
        *)
            echo -e "${RED}ERROR: Unknown flag: $1${NC}" >&2
            exit 1
            ;;
    esac
done

# ── Helpers ───────────────────────────────────────────────────────────────────
info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*" >&2; }

cleanup() {
    local exit_code=$?
    # Remove temporary extraction dir if it exists
    if [[ -n "${VERIFY_DIR:-}" && -d "$VERIFY_DIR" ]]; then
        rm -rf "$VERIFY_DIR"
    fi
    # On failure, remove partial backup tarball
    if [[ $exit_code -ne 0 && -n "${TARBALL:-}" && -f "$TARBALL" ]]; then
        warn "Removing partial backup: $TARBALL"
        rm -f "$TARBALL"
    fi
    exit $exit_code
}
trap cleanup EXIT

# ── Preflight checks ─────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}=== OVAV Vault Backup ===${NC}"
echo -e "  Timestamp : ${TIMESTAMP}"
echo -e "  Mode      : $(if $CHECK_MODE; then echo 'DRY-RUN (--check)'; else echo 'FULL BACKUP'; fi)"
echo -e "  Vault dir : ${VAULT_DIR}"
echo -e "  Backup dir: ${BACKUP_DIR}"
echo ""

# 1. Vault directory must exist
if [[ ! -d "$VAULT_DIR" ]]; then
    fail "Vault directory not found: $VAULT_DIR"
    fail "Run 'ovav vault init' first (see SEG-0 C0-C3)."
    exit 1
fi
ok "Vault directory exists"

# 2. Master key must exist
if [[ ! -f "$MASTER_KEY" ]]; then
    fail "Master key not found: $MASTER_KEY"
    fail "Run 'ovav vault init' to generate the master key."
    exit 1
fi
ok "Master key present: $MASTER_KEY"

# ── Step 1: Scan vault ───────────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[1/5] Scanning vault...${NC}"

if command -v ovav &>/dev/null; then
    if ! ovav vault scan 2>&1; then
        fail "ovav vault scan failed"
        exit 1
    fi
    ok "Vault scan complete"
else
    warn "'ovav' CLI not in PATH — falling back to filesystem scan"
    ENC_COUNT=$(find "$VAULT_DIR" -maxdepth 1 -name '*.enc' -type f 2>/dev/null | wc -l)
    info "Found ${ENC_COUNT} .enc file(s) in vault"
    if [[ "$ENC_COUNT" -eq 0 ]]; then
        warn "No encrypted assets found — vault may be empty"
    fi
fi

# ── Step 2: Encrypt assets ───────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[2/5] Encrypting assets...${NC}"

if command -v ovav &>/dev/null; then
    if ! ovav vault encrypt --key "$MASTER_KEY" 2>&1; then
        fail "ovav vault encrypt failed"
        exit 1
    fi
    ok "Encryption complete"
else
    warn "'ovav' CLI not in PATH — skipping encryption step"
    warn "Ensure all vault assets are encrypted before backup"
fi

# ── Check mode exit ──────────────────────────────────────────────────────────
if $CHECK_MODE; then
    echo ""
    echo -e "${GREEN}=== DRY-RUN COMPLETE ===${NC}"
    ENC_COUNT=$(find "$VAULT_DIR" -maxdepth 1 -name '*.enc' -type f 2>/dev/null | wc -l)
    info "Encrypted files: ${ENC_COUNT}"
    info "Master key     : present"
    info "Vault directory: OK"
    if [[ "$ENC_COUNT" -gt 0 ]]; then
        ok "Vault is ready for backup"
    else
        warn "Vault has no .enc files — nothing to back up"
    fi
    exit 0
fi

# ── Step 3: Copy .enc files to backup staging ────────────────────────────────
echo ""
echo -e "${YELLOW}[3/5] Staging encrypted assets...${NC}"

mkdir -p "$BACKUP_DIR"

STAGING_DIR="$(mktemp -d "${BACKUP_DIR}/.staging_XXXXXX")"
STAGE_COUNT=0

for enc_file in "$VAULT_DIR"/*.enc; do
    [[ -e "$enc_file" ]] || continue
    cp "$enc_file" "$STAGING_DIR/"
    STAGE_COUNT=$((STAGE_COUNT + 1))
done

# Also include vault metadata if present
for meta_file in "$VAULT_DIR"/vault_meta.json "$VAULT_DIR"/master.key; do
    [[ -e "$meta_file" ]] && cp "$meta_file" "$STAGING_DIR/"
done

if [[ "$STAGE_COUNT" -eq 0 ]]; then
    fail "No .enc files found to stage"
    rm -rf "$STAGING_DIR"
    exit 1
fi
ok "Staged ${STAGE_COUNT} encrypted file(s)"

# ── Step 4: Create timestamped tarball ───────────────────────────────────────
echo ""
echo -e "${YELLOW}[4/5] Creating backup tarball...${NC}"

TARBALL="${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"

# Build tarball from staging directory contents (relative paths)
if ! tar -czf "$TARBALL" -C "$STAGING_DIR" .; then
    fail "tar creation failed"
    rm -rf "$STAGING_DIR"
    exit 1
fi

TARBALL_SIZE=$(du -h "$TARBALL" | cut -f1)
ok "Tarball created: ${TARBALL} (${TARBALL_SIZE})"

# Clean staging dir
rm -rf "$STAGING_DIR"

# ── Step 5: Verify integrity ─────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}[5/5] Verifying backup integrity...${NC}"

VERIFY_DIR="$(mktemp -d "${BACKUP_DIR}/.verify_XXXXXX")"

if ! tar -xzf "$TARBALL" -C "$VERIFY_DIR"; then
    fail "Failed to extract tarball for verification"
    exit 1
fi

VERIFY_ENC_COUNT=$(find "$VERIFY_DIR" -maxdepth 1 -name '*.enc' -type f | wc -l)

if [[ "$VERIFY_ENC_COUNT" -ne "$STAGE_COUNT" ]]; then
    fail "Integrity check FAILED: expected ${STAGE_COUNT} .enc files, found ${VERIFY_ENC_COUNT}"
    exit 1
fi

# Verify master key survived the round-trip
if [[ ! -f "$VERIFY_DIR/master.key" ]]; then
    fail "Integrity check FAILED: master.key missing from backup"
    exit 1
fi

ok "Integrity verified: ${VERIFY_ENC_COUNT} .enc file(s) + master.key"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}=== BACKUP COMPLETE ===${NC}"
echo -e "  Tarball   : ${TARBALL}"
echo -e "  Size      : ${TARBALL_SIZE}"
echo -e "  Files     : ${STAGE_COUNT} encrypted + metadata"
echo -e "  Integrity : VERIFIED"
echo -e "  Timestamp : ${TIMESTAMP}"
echo ""
echo -e "${CYAN}To restore:${NC}"
echo -e "  tar -xzf ${TARBALL} -C ${VAULT_DIR}/"
echo ""

# ── Retention policy: keep last 10 backups ───────────────────────────────────
BACKUP_COUNT=$(find "$BACKUP_DIR" -maxdepth 1 -name 'vault_*.tar.gz' -type f | wc -l)
if [[ "$BACKUP_COUNT" -gt 10 ]]; then
    REMOVE_COUNT=$((BACKUP_COUNT - 10))
    info "Retention policy: keeping last 10 backups, removing ${REMOVE_COUNT} oldest"
    find "$BACKUP_DIR" -maxdepth 1 -name 'vault_*.tar.gz' -type f -printf '%T+ %p\n' \
        | sort \
        | head -n "$REMOVE_COUNT" \
        | awk '{print $2}' \
        | xargs rm -f
    ok "Old backups cleaned"
fi
