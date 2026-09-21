#!/usr/bin/env bash
# load-vault.sh — populate OVAV vault with all secrets
# CRIT-001: never log tokens. Use file-based I/O via @ syntax.

set -euo pipefail

export PATH="/home/braka/go/bin:/home/braka/.local/bin:$PATH"
KEY_HEX=$(cat /home/braka/.config/ovav/vault.key)
export OVAV_SECRETS_KEY="$KEY_HEX"

# Use secure temp dir within /tmp/opencode (allowed by policy)
TMPDIR="/tmp/opencode/ovav-vault-load"
mkdir -p "$TMPDIR"
trap "rm -rf $TMPDIR" EXIT

added=0
failed=0

# Helper: silently add secret from file
add_secret() {
  local type="$1"
  local name="$2"
  local file="$3"
  if [ ! -f "$file" ]; then
    echo "skip: $name (no file)"
    return
  fi
  if ovav-vault-secrets add --type "$type" --name "$name" --value "@$file" >/dev/null 2>&1; then
    echo "ok: $name"
    added=$((added+1))
  else
    echo "fail: $name"
    failed=$((failed+1))
  fi
}

# 1. GitHub PAT (from gh auth)
echo "=== GitHub PAT ==="
gh auth token > "$TMPDIR/gh_pat" 2>/dev/null && add_secret api_token "github_pat_alexander" "$TMPDIR/gh_pat"

# 2. Local OVAV vault key (the encryption key itself)
echo "=== Vault key ==="
add_secret encryption_key "ovav_local_vault_key" /home/braka/.config/ovav/vault.key

# 3. CEO seed
echo "=== CEO seed ==="
add_secret user_secret "ovav_ceo_seed" /home/braka/.local/share/ovav/seed_export

# 4. MiniMax API key (from opencode auth)
echo "=== MiniMax ==="
if [ -f /home/braka/.local/share/opencode/auth.json ]; then
  # Extract MiniMax token carefully without logging
  jq -r '.["minimax-coding-plan"].access // .["minimax-coding-plan"].apiKey // empty' \
    /home/braka/.local/share/opencode/auth.json > "$TMPDIR/minimax" 2>/dev/null || true
  if [ -s "$TMPDIR/minimax" ]; then
    add_secret api_token "minimax_api_key" "$TMPDIR/minimax"
  fi
fi

# 5. OpenAI key (if present)
echo "=== OpenAI ==="
jq -r '.openai.access // .openai.apiKey // empty' \
  /home/braka/.local/share/opencode/auth.json > "$TMPDIR/openai" 2>/dev/null || true
[ -s "$TMPDIR/openai" ] && add_secret api_token "openai_api_key" "$TMPDIR/openai"

# 6. Filesystem secrets discovered
echo "=== Filesystem secrets ==="
ENV_FILES=$(find /home/braka/Systems/ovav /home/braka/Systems/work -name ".env*" -readable 2>/dev/null || true)
for f in $ENV_FILES; do
  # Extract API_KEY/TOKEN/SECRET variables
  grep -E "^[A-Z_]+_(API_KEY|API_TOKEN|SECRET|TOKEN|KEY)=" "$f" 2>/dev/null | while IFS= read -r line; do
    var=$(echo "$line" | cut -d= -f1)
    val=$(echo "$line" | cut -d= -f2-)
    if [ -n "$val" ] && [ "$val" != "your_key_here" ] && [ "$val" != "changeme" ]; then
      echo "$val" > "$TMPDIR/$var"
      add_secret user_secret "$var" "$TMPDIR/$var"
    fi
  done
done

echo ""
echo "=== Summary ==="
echo "added: $added"
echo "failed: $failed"
echo ""
echo "=== Vault contents ==="
ovav-vault-secrets list --json 2>&1 | jq -r '.[].name + " (" + .type + ")"' 2>/dev/null || \
  ovav-vault-secrets list 2>&1
