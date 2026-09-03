#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════════════
#  OVAV Area Signing — Canonical Tooling
#  Cada lead (10 áreas) firma con SSH ed25519. Cada commit verifica.
#  Co-firma: lead + área + agents que trabajaron.
# ════════════════════════════════════════════════════════════════════════════
#
#  Usage:
#    ./ovav-area-signing.sh setup            # generates 10 SSH ed25519 keys
#                                            # updates allowed_signers_full
#                                            # configures git
#    ./ovav-area-signing.sh switch <lead>    # switch active signing key
#    ./ovav-area-signing.sh status           # which key is currently active
#    ./ovav-area-signing.sh verify <commit>  # verify commit signature
#    ./ovav-area-signing.sh list             # list all leads and their pubkeys
#    ./ovav-area-signing.sh restore          # re-install global config (idempotent)
#
#  Spec: human-AI commit signing — each area commits signed.
#         Co-authors: lead + contributing agents via commit trailers.
#
# (C) 2026 OVAV AGENTS — Thavren (Platform Engineering)

set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
KEYS_DIR="$HOME/.ssh/ovav_signing"
ALLOWED_SIGNERS="$OVAV_ROOT/.ovav/allowed_signers_full"
LEADS=(camila dante eidren elena kenji renata sofia thavren uriel valeria)

declare -A LEAD_AREA
LEAD_AREA[camila]="legal_compliance"
LEAD_AREA[dante]="digital_product"
LEAD_AREA[eidren]="research_intelligence"
LEAD_AREA[elena]="ux_design"
LEAD_AREA[kenji]="adversarial_intelligence"
LEAD_AREA[renata]="health_performance"
LEAD_AREA[sofia]="commercial_growth"
LEAD_AREA[thavren]="platform_engineering"
LEAD_AREA[uriel]="devops_infrastructure"
LEAD_AREA[valeria]="education_career"

log()  { printf '\033[1;36m▸\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m✓\033[0m %s\n' "$*"; }
err()  { printf '\033[1;31m✗\033[0m %s\n' "$*" >&2; }

cmd_setup() {
    log "Setting up OVAV area signing infrastructure"
    mkdir -p "$KEYS_DIR"
    chmod 700 "$KEYS_DIR"

    log "Generating SSH ed25519 keys for ${#LEADS[@]} leads"
    for lead in "${LEADS[@]}"; do
        if [ ! -f "$KEYS_DIR/ovav_${lead}" ]; then
            ssh-keygen -t ed25519 -C "ovav-${lead}-signing@ovav.worktree" \
                -f "$KEYS_DIR/ovav_${lead}" -N "" -q
            chmod 600 "$KEYS_DIR/ovav_${lead}"
            chmod 644 "$KEYS_DIR/ovav_${lead}.pub"
            ok "key $lead"
        else
            ok "key $lead (cached)"
        fi
    done

    log "Regenerating $ALLOWED_SIGNERS"
    : > "$ALLOWED_SIGNERS"
    for lead in "${LEADS[@]}"; do
        local pubfile="$KEYS_DIR/ovav_${lead}.pub"
        if [ -f "$pubfile" ]; then
            local lead_cap
            lead_cap="$(awk 'BEGIN{FS=OFS=""}{for(i=1;i<=NF;i++)if(i==1||$i~/[ -]/)$i=toupper($i);print}' <<< "$lead")"
            local area="${LEAD_AREA[$lead]}"
            local line
            line="$(awk -v id="${lead}@ovav.worktree" -v area="${lead_cap} [${area}]" \
                '{printf "%s %s %s %s\n", id, $1, $2, area}' "$pubfile")"
            printf '%s\n' "$line" >> "$ALLOWED_SIGNERS"
        fi
    done
    ok "allowed_signers_full refreshed (10 entries)"

    log "Configuring git for SSH signing (global)"
    git config --global gpg.format ssh
    git config --global gpg.ssh.allowedSignersFile "$ALLOWED_SIGNERS"
    git config --global commit.gpgsign true
    git config --global tag.gpgsign true
    ok "git global: gpg.format=ssh · commit.gpgsign=true · tag.gpgsign=true"

    log "Adding keys to ssh-agent"
    if [ -z "${SSH_AUTH_SOCK:-}" ]; then
        err "SSH_AUTH_SOCK not set; start ssh-agent first: eval \$(ssh-agent -s)"
        return 1
    fi
    for lead in "${LEADS[@]}"; do
        ssh-add "$KEYS_DIR/ovav_${lead}" 2>/dev/null \
            && ok "ssh-add $lead" \
            || err "ssh-add $lead failed"
    done
    ok "Setup complete."
}

cmd_switch() {
    local lead="${1:-}"
    if [ -z "$lead" ]; then
        err "usage: ovav-area-signing.sh switch <lead>"
        err "leads: ${LEADS[*]}"
        return 2
    fi
    if [[ " ${LEADS[*]} " != *" $lead "* ]]; then
        err "Unknown lead: $lead"
        err "leads: ${LEADS[*]}"
        return 2
    fi
    local keyfile="$KEYS_DIR/ovav_${lead}"
    local area="${LEAD_AREA[$lead]}"
    local lead_cap
    lead_cap="$(awk 'BEGIN{FS=OFS=""}{for(i=1;i<=NF;i++)if(i==1||$i~/[ -]/)$i=toupper($i);print}' <<< "$lead")"
    local user_name="Thavren (Platform Engineering)"   # default; updateable
    local user_email="${lead}@ovav.worktree"

    git config --local user.signingkey "$keyfile"
    git config --local gpg.format ssh
    git config --local commit.gpgsign true
    # Identity: who is signing
    git config --local user.name "$lead_cap"
    git config --local user.email "$user_email"
    # Per-lead Co-authored-by + area
    export OVAV_ACTIVE_LEAD="$lead"
    export OVAV_ACTIVE_AREA="$area"
    ok "switched: $lead_cap <$user_email>  [area=$area]"
    ok "  signingkey=$keyfile"
}

cmd_status() {
    local lead="${OVAV_ACTIVE_LEAD:-?}"
    local area="${OVAV_ACTIVE_AREA:-?}"
    local signingkey
    signingkey="$(git config --get user.signingkey 2>/dev/null || echo '(none)')"
    local signname
    signname="$(git config --get user.name 2>/dev/null || echo '(unset)')"
    local signmail
    signmail="$(git config --get user.email 2>/dev/null || echo '(unset)')"
    local git_root
    git_root="$(git rev-parse --show-toplevel 2>/dev/null || echo '(not in a git repo)')"
    printf '\n  active lead     = %s\n' "$lead"
    printf '  active area     = %s\n' "$area"
    printf '  user.name       = %s\n' "$signname"
    printf '  user.email      = %s\n' "$signmail"
    printf '  user.signingkey = %s\n' "$signingkey"
    printf '  git root        = %s\n\n' "$git_root"
}

cmd_verify() {
    local target="${1:-HEAD}"
    if [ ! -d "$OVAV_ROOT" ]; then
        err "Run from inside the OVAV repository."
        return 1
    fi
    ( cd "$OVAV_ROOT" && git verify-commit "$target" 2>&1 || true )
}

cmd_list() {
    log "OVAV area signing keys"
    printf '%-10s %-26s %s\n' 'LEAD' 'AREA' 'PUBLIC KEY (fingerprint)'
    printf '%-10s %-26s %s\n' '----' '----' '---------------------'
    for lead in "${LEADS[@]}"; do
        local pubfile="$KEYS_DIR/ovav_${lead}.pub"
        if [ -f "$pubfile" ]; then
            local fp
            fp="$(ssh-keygen -lf "$pubfile" 2>/dev/null | awk '{print $2}')"
            printf '%-10s %-26s %s\n' "$lead" "${LEAD_AREA[$lead]}" "$fp"
        fi
    done
}

cmd_restore() {
    cmd_setup
}

case "${1:-}" in
    setup)   cmd_setup ;;
    switch)  cmd_switch "${2:-}" ;;
    status)  cmd_status ;;
    verify)  cmd_verify "${2:-HEAD}" ;;
    list)    cmd_list ;;
    restore) cmd_restore ;;
    ""|help|-h|--help)
        printf '%s\n' "OVAV area signing — see header comment for usage"
        exit 0
        ;;
    *)  err "unknown subcommand: $1"; exit 2 ;;
esac
