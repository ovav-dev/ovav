#!/usr/bin/env bash
# =============================================================================
# OVAV OWS Worktree — runtime dispatcher
# v3.1.0 — OWS-HARDENING-v0.1.0 SU-1: GNU-style --help/-h parser
#         12 profiles, 6-stage validation pipeline,
#         4 compliance levels, 3 owd modes, conflict prediction, tier gates.
# =============================================================================
set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/OVAV}"
OW_LIB="$OVAV_ROOT/bin/ovav-owlib.sh"
[ -f "$OW_LIB" ] && source "$OW_LIB"

OVAV_CONSUMER_ID="${OVAV_CONSUMER_ID:-anonymous}"
OVAV_CONSUMER_ROOT="${OVAV_CONSUMER_ROOT:-$(pwd)}"
OVAV_CONSUMER_TIER="${OVAV_CONSUMER_TIER:-business}"
OW_AUDIT_LOG="$OVAV_ROOT/.ovav/runtime/logs/consumer_audit.jsonl"

CMD="$(basename "$0" | sed -E 's/^(ovav-)?ow//; s/^-//')"
# When invoked directly as ovav-ow-runtime.sh, use first arg as command (test mode)
case "$CMD" in
  runtime.sh|runtime|"" )
    CMD="${1:-}"
    if [ -n "$CMD" ]; then shift; fi
    ;;
esac
[ -z "$CMD" ] && CMD="c"

# ──── SU-1: Universal --help/-h/--version interception (pre-tier) ──────────
# Help must never be gated by tier (discoverability invariant).
# Scan $@ for any help/version flag before tier evaluation.
OW_HELP_REQUESTED=0
for _arg in "$@"; do
  case "$_arg" in
    -h|--help|--version)
      OW_HELP_REQUESTED=1
      break
      ;;
  esac
done

# ──── SU-1: GNU-style --help/-h parser (BLOCKING, before any command logic)
# Reusable across all 11 ow-* commands. Prevents OWS-GAP-01:
#   `owc --help` previously created a branch named "--help" because
#   the dispatcher passed $1 unfiltered to branch detection.
# This parser intercepts standard GNU flags BEFORE any dispatch logic.
ow_show_help() {
  local cmd_name="$1"
  case "$cmd_name" in
    c)
      cat <<'EOF'
owc — crear worktree OVAV con inteligencia convencional commit

Uso:
  owc <nombre-branch> [--carry] [--profile <tipo>] [--suggest] [--auto]
  owc --help | -h

Argumentos:
  <nombre-branch>     Branch/worktree a crear (o descripción plana)
  [descripción]       Descripción (usada para inferir tipo y perfil)

Modos inteligentes:
  --suggest           Solo mostrar branch name sugerido (no crear)
  --auto <desc>       Crear branch automáticamente desde descripción

Perfiles auto-detectados:
  feature/*   fix/*    hotfix/*  release/*  docs/*
  spike/*     research/*  emergency/*  refactor/*
  chore/*     ci/*     build/*   test/*    deps/*

Detección convencional commit:
  owc feat(auth): add OAuth2 login
  owc fix core race condition          # auto-detecta tipo=fix

Ejemplos:
  owc feature/login-page
  owc --auto "add vault secret storage"
  owc --suggest "fix critical auth bug"
  owc hotfix/critical-auth-bug

Ver también:
  owd  — finalizar + publicar
  owl  — listar worktrees
  owv  — verificar antes de finalizar
EOF
      ;;
    d)
      cat <<'EOF'
owd — finalizar worktree: merge + cleanup + push

Uso:
  owd [compliance] [--resume]
  owd --help | -h
  owd --version

Compliance levels (autodetectados desde el branch):
  standard    — merge develop
  strict      — 6 validators required
  waiver      — CEO pre-approval file required

Flag --resume: continúa push pendiente desde state machine (SU-4)

Ejemplos:
  owd
  owd strict
  owd --resume

Estados de merge:
  READY       — todos los gates pasaron
  CONFLICT    — abort con cleanup (SU-2)
  PUSH_PENDING— push falló, re-ejecutable idempotente
EOF
      ;;
    l)
      cat <<'EOF'
owl — listar worktrees OVAV con detección de zombies

Uso:
  owl [--zombie-only] [--json]
  owl --help | -h

Flags:
  --zombie-only   Lista solo worktrees cuya branch upstream ya no existe
  --json          Output en JSON para tooling

Ejemplos:
  owl
  owl --zombie-only
EOF
      ;;
    v)
      cat <<'EOF'
owv — verificar worktree (gate pre-merge)

Uso:
  owv [--verbose]
  owv --help | -h

Stages (en orden):
  1. Conflicts prediction
  2. Secrets sweep
  3. Forbidden files
  4. Stack-specific validation
  5. Hygiene scan
  6. GPG signature check (compliance=strict)

Con --verbose muestra REASON + remediation hint por stage.
EOF
      ;;
    s)
      cat <<'EOF'
ows — sincronizar worktree (prune + gc)
Uso: ows [--help]
EOF
      ;;
    m)
      cat <<'EOF'
owm — mover worktree a otra ubicación
Uso: owm <src> <dest> [--help]
EOF
      ;;
    clean)
      cat <<'EOF'
owclean — limpiar worktrees huérfanos y z
Uso: owclean [--dry-run] [--help]
EOF
      ;;
    prep)
      cat <<'EOF'
owprep — preparar/regenerar worktree-config.json
Uso: owprep [--help]
EOF
      ;;
    suggest)
      cat <<'EOF'
owsuggest — sugerir comandos OWS según contexto
Uso: owsuggest [--explain] [--help]
EOF
      ;;
    p)
      cat <<'EOF'
owp — push con auto-clean de regenerables (full auto-discard)
Uso: owp [--rebase] [--help]
EOF
      ;;
    a|r|x|lk)
      cat <<'EOF'
ow{a,r,x,lk} — operaciones tier 2-4 (gated)

Uso:
  owa <branch>           — abort worktree (pro tier)
  owr <branch>           — retry último owd (pro tier)
  owlk <branch>          — lock/unlock worktree (pro tier)
  owx <src> <dest>       — cherry-pick cross-branch (enterprise)

Su-11 (siguiente sprint) libera self-recovery para free tier.
EOF
      ;;
    *)
      cat <<'EOF'
OVAV Worktree System (OWS) — comandos disponibles

Uso: ow<comando> [--help] [--version]

Worktree lifecycle (free tier):
  owc   — crear worktree con perfil auto-detectado
  owd   — finalizar (merge + cleanup + push)
  owl   — listar worktrees + zombies
  owv   — verificar pre-merge
  ows   — sincronizar (prune)
  owclean — limpiar huérfanos
  owm   — mover worktree
  owprep — preparar config
  owsuggest — sugerir comandos
  owp   — push con auto-clean

Recovery (pro+ tier — su-11 libera self):
  owa   — abort
  owr   — retry
  owlk  — lock/unlock

Cross-tier (enterprise):
  owx   — cherry-pick entre branches

Docs: docs/worktree/OWS_USER_GUIDE.md
EOF
      ;;
  esac
}

ow_parse_common_flags() {
  # Reusable GNU-style parser. Returns 0 (help shown) on --help/-h/--version.
  # All OTHER flags pass through unchanged — this parser only intercepts
  # the three universal flags. Per-handler flags (--compliance, --json, etc.)
  # are handled by each handler after this returns.
  while [ $# -gt 0 ]; do
    case "$1" in
      -h|--help)
        ow_show_help "$CMD"
        return 0
        ;;
      --version)
        echo "ow$CMD v3.1.0 (OVAV worktree runtime)"
        return 0
        ;;
      --)
        shift
        break
        ;;
      *)
        # Stop parsing at first non-flag arg
        break
        ;;
    esac
  done
  return 2  # 2 = "no help/version flag seen, continue with $@ intact"
}

# ──── Tier check (mandatory gate) ─────────────────────────────────────────
ow_tier_level() {
  case "$1" in
    free)       echo 1 ;;
    pro)        echo 2 ;;
    business)   echo 3 ;;
    enterprise) echo 4 ;;
    *)          echo 0 ;;  # anonymous/unknown
  esac
}

ow_required_level_for() {
  case "$1" in
    c|d|l|v|s|clean|m|owprep|owsuggest|prep|suggest|p) echo 1 ;;  # free tier
    a|r)              echo 2 ;;  # pro tier
    x)                echo 4 ;;  # enterprise tier (cherry-pick / cross-branch)
    lk)               echo 2 ;;  # pro tier (lock)
    *) echo 99 ;;
  esac
}

USER_TIER=$(ow_tier_level "$OVAV_CONSUMER_TIER")
REQ_TIER=$(ow_required_level_for "$CMD")
if [ "$USER_TIER" -lt "$REQ_TIER" ] && [ "${OW_HELP_REQUESTED:-0}" != "1" ]; then
  # SU-11: self-recovery boundary — free tier can abort/retry/lock OWN worktrees
  SELF_RECOVERY=0
  case "$CMD" in
    a|r|lk)
      # Check if target worktree belongs to this consumer
      if [ -n "${1:-}" ] && [ -d "${1:-}" ]; then
        TARGET_CONSUMER=$(cd "${1}" 2>/dev/null && git config --worktree ovav.consumer_id 2>/dev/null || echo "")
        if [ "$TARGET_CONSUMER" = "$OVAV_CONSUMER_ID" ]; then
          SELF_RECOVERY=1
        fi
      fi
      ;;
  esac
  if [ "$SELF_RECOVERY" = "0" ]; then
    echo "BLOCKED ow$CMD requires tier $OVAV_CONSUMER_TIER < required level $REQ_TIER" >&2
    exit 1
  fi
fi

# ──── Helpers ─────────────────────────────────────────────────────────────
STACK="$(ow_detect_stack "$OVAV_CONSUMER_ROOT" 2>/dev/null || echo unknown)"
WT_CONFIG="$OVAV_CONSUMER_ROOT/.ovav/worktree-config.json"

ow_log_audit_safe() {
  local cmd="$1" args="$2" consumer_id="$3" root="$4" stack="$5" branch_type="$6" decision="$7" reason="$8"
  mkdir -p "$(dirname "$OW_AUDIT_LOG")" 2>/dev/null
  local ts; ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf '{"ts":"%s","actor":"%s","command":"ow%s","args":"%s","consumer_id":"%s","consumer_root":"%s","stack":"%s","branch_type":"%s","decision":"%s","reason":"%s"}\n' \
    "$ts" "${USER:-unknown}" "$cmd" "$args" "$consumer_id" "$root" "$stack" "$branch_type" "$decision" "$reason" \
    >> "$OW_AUDIT_LOG"
}

# ──── Global help/version interception ────────────────────────────────────
# Must run BEFORE case statement to prevent handlers from consuming flags.
# Also must run AFTER tier check is bypassed (OW_HELP_REQUESTED already set).
ow_parse_common_flags "$@" && exit 0

# ──── Commands ────────────────────────────────────────────────────────────
case "$CMD" in
  c)  # owc — create worktree with profile detection + conventional commit intelligence
    CARRY_UNCOMMITTED=0
    FORCE_PROFILE=""
    SUGGEST_ONLY=0
    AUTO_CONVENTIONAL=0
    while [ $# -gt 0 ]; do
      case "$1" in
        --carry)          CARRY_UNCOMMITTED=1; shift ;;
        --profile)        FORCE_PROFILE="${2:-}"; shift 2 ;;
        --suggest)        SUGGEST_ONLY=1; shift ;;
        --auto|-a)        AUTO_CONVENTIONAL=1; shift ;;
        --)               shift; break ;;
        *)                break ;;
      esac
    done

    # ── Smart name detection ─────────────────────────────────────────────────
    # Description-only mode: no slash, no known prefix, no ticket ID
    if [ -z "${1:-}" ]; then
      echo "ERROR: branch name required" >&2
      echo "Usage: owc <branch-name> [--carry] [--profile <type>] [--suggest] [--auto]" >&2
      exit 2
    fi

    name="${1:-}"
    desc="${2:-}"

    # Description detection: plain English → conventional commit branch
    if [[ ! "$name" =~ / ]] && \
       [[ ! "$name" =~ ^(feature|feat|fix|hotfix|release|docs|refactor|spike|research|chore|ci|build|test|deps|perf|security|migration|enterprise|emergency|patch)- ]] && \
       [[ ! "$name" =~ ^[A-Z]{2,10}-[0-9]+$ ]] && \
       [[ ! "$name" =~ ^#[0-9]+$ ]]; then
      suggested="$(ow_suggest_branch_name "$name" "feature")"
      echo "[ovav-owc] Branch name auto-generated from description:"
      echo ""
      echo "  Description: $name"
      echo "  Suggested:   $suggested"
      echo ""
      if [ "$AUTO_CONVENTIONAL" = "1" ]; then
        name="$suggested"
        echo "[ovav-owc] Using: $name (--auto mode)"
      elif [ "$SUGGEST_ONLY" = "1" ]; then
        echo "Use: owc '$suggested'  OR  owc --auto '$name'"
        exit 0
      else
        echo "Use --auto to create with this name:"
        echo "  owc '$suggested'"
        echo "  owc --auto '$name'"
        echo ""
        echo "Examples:"
        echo "  owc feature/login              # explicit feature branch"
        echo "  owc --auto 'add vault secret' # auto from description"
        exit 0
      fi
    fi

    # Validate branch name is safe for git
    if ! echo "$name" | grep -qE '^[a-zA-Z0-9_./-]+$'; then
      echo "ERROR: branch name contains invalid characters: $name" >&2
      echo "Hint: use --auto mode for plain descriptions, or --suggest to preview" >&2
      exit 2
    fi

    if [ ! -d "$OVAV_CONSUMER_ROOT/.git" ]; then
      echo "ERROR: not in a git repo ($OVAV_CONSUMER_ROOT)" >&2
      exit 2
    fi

    # SU-5: forced profile overrides auto-detection
    if [ -n "$FORCE_PROFILE" ]; then
      case "$FORCE_PROFILE" in
        feature|fix|hotfix|release|docs|refactor|spike|research|chore|ci|build|test|deps|perf|security|migration|enterprise|emergency|patch|main)
          PROFILE="$FORCE_PROFILE" ;;
        *) echo "ERROR: unknown profile '$FORCE_PROFILE'. Valid: feature,hotfix,release,..." >&2; exit 2 ;;
      esac
      echo "[ovav-owc] profile=$PROFILE (forced override)"
    else
      PROFILE="$(ow_detect_profile "$name" "$desc")"
    fi
    BASE_BRANCH="$(ow_base_branch "$PROFILE")"
    TTL_HOURS="$(ow_default_ttl_hours "$PROFILE")"
    REVIEWER_REQ="$(ow_required_reviewer "$PROFILE")"

    # main branch: don't worktree, just checkout
    if [ "$PROFILE" = "main" ]; then
      echo "[ovav-owc] type=main — checking out directly"
      (cd "$OVAV_CONSUMER_ROOT" && git checkout "$name" 2>&1 | head -3)
      ow_log_audit_safe "c" "$name" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "allowed" "checkout main"
      exit 0
    fi

    target_path="$(ow_compute_target_path "$name" "$OVAV_CONSUMER_ROOT")"
    echo "[ovav-owc] profile=$PROFILE base=$BASE_BRANCH ttl=${TTL_HOURS}h target=$target_path stack=$STACK"

    # Branch exists check (avoid duplication)
    if (cd "$OVAV_CONSUMER_ROOT" && git rev-parse --verify "$name" >/dev/null 2>&1); then
      echo "ERROR: branch '$name' already exists"
      ow_log_audit_safe "c" "$name" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "denied" "branch exists"
      exit 3
    fi

    mkdir -p "$(dirname "$target_path")"

    # Conflict prediction: skip during owc — branch doesn't exist yet,
    # it's created FROM base_branch so conflicts are impossible at creation.
    # Prediction is only meaningful for owd (merge back) and owl (status).
    # Create the worktree + branch
    if (cd "$OVAV_CONSUMER_ROOT" && git worktree add -b "$name" "$target_path" "$BASE_BRANCH" 2>&1 | tail -3); then
      # Create .ovav after worktree exists (was pre-created before, causing
      # 'directory already exists' fatal from git worktree add)
      mkdir -p "$target_path/.ovav"
      # Configure git rerere for merge conflict reuse (worktree-scoped)
      (cd "$target_path" && git config --worktree rerere.enabled true 2>/dev/null || true)

      # Spike: tag creation time for TTL cleanup
      if [ "$PROFILE" = "spike" ]; then
        echo "$TTL_HOURS" > "$target_path/.ovav/.spike-ttl-hours"
        echo "$(date -u +%s)" > "$target_path/.ovav/.spike-created-at"
        echo "[ovav-owc] spike TTL: $TTL_HOURS hours (auto-cleanup via owclean)"
      fi

      # Post-create hook (if user-defined)
      if [ -x "$target_path/.ovav/hooks/post-create" ]; then
        "$target_path/.ovav/hooks/post-create" "$name" "$PROFILE" "$target_path" 2>&1 | head -5 || true
      fi

      # SU-3: carry uncommitted changes from parent to worktree
      if [ "$CARRY_UNCOMMITTED" = "1" ]; then
        echo "  [carry] Stashing uncommitted changes from $OVAV_CONSUMER_ROOT..."
        (cd "$OVAV_CONSUMER_ROOT" && git stash push -m "owc-carry: $name" 2>&1 | tail -3)
        STASH_REF=$(cd "$OVAV_CONSUMER_ROOT" && git stash list | head -1 | cut -d: -f1)
        if [ -n "$STASH_REF" ]; then
          echo "  [carry] Applying stash in $target_path..."
          (cd "$target_path" && git stash apply "$STASH_REF" 2>&1 | tail -3) || {
            echo "  [carry] WARN: stash apply had conflicts. Stash left pending."
            echo "  [carry] Resolve in $target_path, then: git stash drop"
          }
        else
          echo "  [carry] No uncommitted changes to carry."
        fi
      fi

      # Auto-cd (printed; shell wrapper evaluates it)
      echo "PWD_NOW=$target_path"

      ow_log_audit_safe "c" "$name" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "allowed" "create: $name -> $target_path"

      cat <<EOF
[ovav-owc created] $name
  Profile: $PROFILE  base: $BASE_BRANCH  ttl: ${TTL_HOURS}h
  Reviewer required: $REVIEWER_REQ
  Path:    $target_path
  Stack:   $STACK
EOF
    else
      ow_log_audit_safe "c" "$name" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "denied" "worktree creation failed"
      exit 4
    fi
    ;;

  d)  # owd — finalize & publish with 6-stage validation
    # Parse options + branch / path
    COMPLIANCE="standard"
    REVIEWER=""
    TARGET_BRANCH=""
    TARGET_PATH=""
    while [ $# -gt 0 ]; do
      case "$1" in
        --compliance) COMPLIANCE="${2:-standard}"; shift 2 ;;
        --reviewer)   REVIEWER="${2:-}"; shift 2 ;;
        --*)          echo "Unknown flag: $1" >&2; exit 2 ;;
        *)            if [ -z "$TARGET_BRANCH" ]; then TARGET_BRANCH="$1"; else TARGET_PATH="$1"; fi; shift ;;
      esac
    done

    # Resolve target worktree
    if [ -z "$TARGET_BRANCH" ]; then
      # Auto-detect: assume CWD is the worktree
      TARGET_PATH="$(pwd)"
      TARGET_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
    elif [ -d "$TARGET_BRANCH" ]; then
      TARGET_PATH="$TARGET_BRANCH"
      TARGET_BRANCH="$(cd "$TARGET_BRANCH" && git rev-parse --abbrev-ref HEAD)"
    fi

    [ ! -d "$TARGET_PATH" ] && echo "ERROR: path $TARGET_PATH not found" >&2 && exit 5

    PROFILE="$(ow_detect_profile "$TARGET_BRANCH")"
    PROFILE_BASE="$(ow_base_branch "$PROFILE")"
    REVIEWER_LVL="$(ow_required_reviewer_level "$COMPLIANCE")"

    echo "[ovav-owd] profile=$PROFILE branch=$TARGET_BRANCH compliance=$COMPLIANCE reviewer_level=$REVIEWER_LVL"

    FAIL_STAGES=""
    # ──── Stage 1: Conflict Prediction ─────────────────────────────────
    echo ""
    echo "  [S1/6] Conflict prediction vs $PROFILE_BASE:"
    CONFLICT="$(ow_predict_conflicts "$TARGET_BRANCH" "$PROFILE_BASE")"
    case "$CONFLICT" in
      CLEAN) echo "    PASS: clean merge" ;;
      CONFLICTS:*) echo "    FAIL: $CONFLICT"; FAIL_STAGES="$FAIL_STAGES 1" ;;
      *) echo "    SKIP: $CONFLICT (base missing?)" ;;
    esac

    # ──── Stage 2: Secrets Sweep ─────────────────────────────────────
    echo "  [S2/6] Secrets sweep:"
    SECRETS="$(ow_secrets_sweep "$TARGET_PATH")"
    if [[ "$SECRETS" == "FOUND: 0" ]]; then
      echo "    PASS: no secrets detected"
    else
      echo "    FAIL: $SECRETS"
      FAIL_STAGES="$FAIL_STAGES 2"
    fi

    # ──── Stage 3: Forbidden Files ────────────────────────────────────
    echo "  [S3/6] Forbidden files:"
    FFB="$(ow_forbidden_files "$TARGET_PATH")"
    if [[ "$FFB" == "FORBIDDEN: 0"* ]] && [[ "$FFB" == *"BIG_FILES: 0"* ]]; then
      echo "    PASS: no forbidden files"
    else
      echo "    FAIL: $FFB"
      FAIL_STAGES="$FAIL_STAGES 3"
    fi

    # ──── Stage 4: Stack Validation ──────────────────────────────────
    if [ "$COMPLIANCE" = "strict" ] || [ "$COMPLIANCE" = "maximum" ]; then
      echo "  [S4/6] Stack validation ($STACK):"
      SV="$(ow_stack_validate "$TARGET_PATH" "$STACK")"
      echo "    RESULT: $SV"
      [[ "$SV" == *FAIL* ]] && FAIL_STAGES="$FAIL_STAGES 4"
    else
      echo "  [S4/6] Stack validation: SKIP (compliance=$COMPLIANCE)"
    fi

    # ──── Stage 5: Hygiene Scan ──────────────────────────────────────
    if [ "$COMPLIANCE" = "strict" ] || [ "$COMPLIANCE" = "maximum" ]; then
      echo "  [S5/6] Hygiene scan:"
      HG="$(ow_hygiene_scan "$TARGET_PATH")"
      echo "    $HG"
      [[ "$HG" == *"HYGIENE_ISSUES: 0"* ]] || FAIL_STAGES="$FAIL_STAGES 5"
    else
      echo "  [S5/6] Hygiene scan: SKIP (compliance=$COMPLIANCE)"
    fi

    # ──── Stage 6: GPG Signatures ────────────────────────────────────
    if [ "$COMPLIANCE" = "maximum" ]; then
      echo "  [S6/6] GPG signature check:"
      GPG_COUNT="$(ow_gpg_check "$TARGET_PATH")"
      echo "    Signed commits: $GPG_COUNT"
      [ "$GPG_COUNT" = "0" ] && FAIL_STAGES="$FAIL_STAGES 6"
    else
      echo "  [S6/6] GPG signature: SKIP (compliance=$COMPLIANCE)"
    fi

    # ──── Reviewer Check (strict+ requires) ──────────────────────────
    if [ "$REVIEWER_LVL" != "none" ]; then
      if [ -z "$REVIEWER" ]; then
        echo "  REVIEWER REQUIRED ($REVIEWER_LVL) but not provided. Use --reviewer NAME"
        FAIL_STAGES="$FAIL_STAGES R"
      else
        echo "  Reviewer provided: $REVIEWER"
      fi
    fi

    # ──── Final gate ─────────────────────────────────────────────────
    echo ""
    if [ -n "$FAIL_STAGES" ]; then
      echo "BLOCKED: stages failed (S$FAIL_STAGES). Fix and re-run."
      ow_log_audit_safe "d" "$TARGET_BRANCH $COMPLIANCE" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "denied" "stages failed: $FAIL_STAGES"
      exit 6
    fi

    echo "All gates PASS — proceeding to merge & push"
    if [ "$PROFILE" = "hotfix" ] || [ "$PROFILE" = "emergency" ]; then
      echo "  (hotfix/emergency: merging into BOTH $PROFILE_BASE and develop)"
    fi

    # ──── ACTUAL merge + push via OVAV's git hooks system ────────
    # OVAV native hooks (pre-push, post-commit, etc.) handle all safety.
    cd "$OVAV_CONSUMER_ROOT" 2>/dev/null
    echo "  [merge] $TARGET_BRANCH → $PROFILE_BASE (no-ff)"
    # Check if base branch is already checked out in another worktree
    BASE_WT=$(git worktree list 2>/dev/null | grep "\[$PROFILE_BASE\]$" | head -1 | awk '{print $1}')
    if [ -n "$BASE_WT" ] && [ "$BASE_WT" != "$(pwd)" ]; then
      # Base branch is in another worktree — merge there directly
      (cd "$BASE_WT" && git merge --no-ff -m "owd(merge): $TARGET_BRANCH → $PROFILE_BASE" "$TARGET_BRANCH" 2>&1 | tail -5)
    else
      git checkout "$PROFILE_BASE" 2>&1 | head -3
      git merge --no-ff -m "owd(merge): $TARGET_BRANCH → $PROFILE_BASE" "$TARGET_BRANCH" 2>&1 | tail -5
    fi
    if [ $? -ne 0 ]; then
      # SU-2: rollback cleanup — restore staged + unstaged tracked, preserve untracked
      echo "  [rollback] Cleaning up after merge conflict..."
      git restore --staged . 2>/dev/null || true
      git checkout -- . 2>/dev/null || true
      echo "  [rollback] Working tree restored. Untracked files preserved."
      ow_log_audit_safe "d" "$TARGET_BRANCH $COMPLIANCE" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "denied" "merge conflict (cleaned)"
      echo "BLOCKED: merge conflict. Working tree cleaned. Resolve manually."
      exit 7
    fi
    echo "  [push] origin $PROFILE_BASE (hooks enabled)"
    # SU-4: push state machine — if push fails, save state for resume
    PUSH_STATE_DIR="$OVAV_ROOT/.ovav/runtime/owd_state"
    PUSH_STATE_FILE="$PUSH_STATE_DIR/$TARGET_BRANCH.yaml"
    mkdir -p "$PUSH_STATE_DIR"
    # Push from the worktree where base branch is checked out
    PUSH_WT="${BASE_WT:-$(pwd)}"
    if ! (cd "$PUSH_WT" && git push origin "$PROFILE_BASE" 2>&1 | tail -5); then
      # Save push state for resume
      cat > "$PUSH_STATE_FILE" <<STATEEOF
state: PUSH_PENDING
local_commits: $(git log --oneline "$PROFILE_BASE" --not origin/"$PROFILE_BASE" 2>/dev/null | head -5 | awk '{print $1}' | tr '\n' ',')
remote_branch: origin/$PROFILE_BASE
merge_commit: $(git rev-parse HEAD 2>/dev/null)
ttl: 24h
created_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
STATEEOF
      ow_log_audit_safe "d" "$TARGET_BRANCH $COMPLIANCE" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "denied" "push failed (state saved for resume)"
      echo "BLOCKED: push failed. State saved to $PUSH_STATE_FILE"
      echo "  Resume with: owd --resume"
      exit 8
    fi
    # Push succeeded — clean any pending state
    rm -f "$PUSH_STATE_FILE" 2>/dev/null || true

    # Cleanup worktree + delete branch
    cd "$OVAV_CONSUMER_ROOT" 2>/dev/null
    git worktree remove --force "$TARGET_PATH" 2>&1 | head -2
    git branch -d "$TARGET_BRANCH" 2>&1 | head -2

    # After successful finalize: redirect back to default branch (develop by default)
    local default_branch="develop"
    if git show-ref --verify --quiet refs/heads/main 2>/dev/null; then
      default_branch="main"
    elif git show-ref --verify --quiet refs/heads/master 2>/dev/null; then
      default_branch="master"
    fi
    git checkout "$default_branch" 2>&1 | head -2

    ow_log_audit_safe "d" "$TARGET_BRANCH $COMPLIANCE" "$OVAV_CONSUMER_ID" "$OVAV_CONSUMER_ROOT" "$STACK" "$PROFILE" "allowed" "completed"
    # Emit PWD_NOW so shell wrapper does the actual `cd` for the user
    echo "PWD_NOW=$OVAV_CONSUMER_ROOT"
    echo ""
    echo "[ovav-owd] Done. Back on $default_branch."
    ;;

  l)  # owl — list with predictions
    # SU-6: zombie detection flags
    ZOMBIE_ONLY=0
    if [ "${1:-}" = "--zombie-only" ]; then ZOMBIE_ONLY=1; shift; fi
    if [ "${1:-}" = "--json" ]; then
      # JSON-only output (no header)
      (cd "$OVAV_CONSUMER_ROOT" && git worktree list --porcelain 2>/dev/null) | python3 -c "import sys,json
data = []
block = {}
for line in sys.stdin:
    line = line.rstrip()
    if not line:
        if block:
            data.append(block)
            block = {}
    elif line.startswith('worktree '): block['path'] = line[9:]
    elif line.startswith('HEAD '): block['head'] = line[5:13]
    elif line.startswith('branch '): block['branch'] = line[7:]
if block:
    data.append(block)
print(json.dumps(data, indent=2))" 2>/dev/null
      exit 0
    fi
    echo "[ovav-owl] worktrees in $OVAV_CONSUMER_ROOT:"
    ZOMBIE_COUNT=0
    while IFS= read -r wt_line; do
      wt_path=$(echo "$wt_line" | awk '{print $1}')
      wt_head=$(echo "$wt_line" | awk '{print $2}')
      wt_branch=$(echo "$wt_line" | awk '{print $3}' | sed 's/\[//' | sed 's/\]//')
      # Skip main repo line
      [ -z "$wt_branch" ] && continue
      # SU-6: check if remote tracking branch exists
      is_zombie=0
      if [ -n "$wt_branch" ]; then
        remote_gone=$(cd "$OVAV_CONSUMER_ROOT" && git branch -vv 2>/dev/null | grep "$wt_branch" | grep -c "\[.*: gone\]" || true)
        [ "$remote_gone" -gt 0 ] && is_zombie=1
      fi
      if [ "$ZOMBIE_ONLY" = "1" ] && [ "$is_zombie" = "0" ]; then
        continue
      fi
      if [ "$is_zombie" = "1" ]; then
        echo "  [ZOMBIE] $wt_path ($wt_branch)"
        ZOMBIE_COUNT=$((ZOMBIE_COUNT+1))
      else
        echo "  $wt_path ($wt_branch)"
      fi
    done < <(cd "$OVAV_CONSUMER_ROOT" && git worktree list 2>/dev/null)
    if [ "$ZOMBIE_ONLY" = "1" ]; then
      echo ""
      echo "  Found $ZOMBIE_COUNT zombie(s). Run 'owclean' to prune."
    fi
    echo ""
    if [ "${1:-}" = "--history" ]; then
      echo "Audit (last 20 operations):"
      tail -20 "$OW_AUDIT_LOG" 2>/dev/null | python3 -c "import sys,json
for line in sys.stdin:
    line = line.strip()
    if not line: continue
    try:
        e = json.loads(line)
        print(f\"  {e.get('ts','')}  ow{e.get('command','')}  {e.get('decision','')}  {e.get('reason','')[:60]}\")
    except: pass" 2>/dev/null || tail -20 "$OW_AUDIT_LOG" 2>/dev/null
    else
      # Default: include conflict predictions
      echo "Conflict predictions vs develop:"
      while IFS= read -r line; do
        wtpath=$(echo "$line" | awk '{print $1}')
        wtbranch=$(echo "$line" | grep -oE '\[[^]]+\]' | tr -d '[]')
        if [ -n "$wtbranch" ] && [ "$wtbranch" != "detached" ]; then
          cfl=$(ow_predict_conflicts "$wtbranch" "develop" 2>/dev/null || echo UNKNOWN)
          printf "  %-50s branch=%-25s conflict=%s\n" "$wtpath" "$wtbranch" "$cfl"
        fi
      done < <(cd "$OVAV_CONSUMER_ROOT" && git worktree list 2>/dev/null)
    fi
    ;;

  v)  # owv — verify (no merge)
    # SU-7: --verbose flag for actionable errors
    OWV_VERBOSE=0
    if [ "${1:-}" = "--verbose" ]; then OWV_VERBOSE=1; shift; fi
    echo "[ovav-owv] stack=$STACK — running validators..."
    OWV_FAIL=0
    # S1: Integrity
    if (cd "$OVAV_ROOT" && go run -C go-runtime ./cmd/ovav/ status 2>&1 | grep -q "Integrity: 100.0%"); then
      echo "  PASS: OVAV integrity 100%"
    else
      echo "  FAIL: integrity not 100%"
      [ "$OWV_VERBOSE" = "1" ] && echo "    → Fix: go run -C go-runtime ./cmd/ovav/ defend scan"
      OWV_FAIL=1
    fi
    # S2: Stack-specific
    case "$STACK" in
      go)
        VET_OUT=$(cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && go vet ./... 2>&1 || true)
        if [ -z "$VET_OUT" ]; then
          echo "  PASS: go vet clean"
        else
          echo "  FAIL: go vet errors"
          [ "$OWV_VERBOSE" = "1" ] && echo "    → Fix: run 'go vet ./...' and address each error"
          OWV_FAIL=1
        fi
        ;;
      typescript) [ -f "$OVAV_CONSUMER_ROOT/package.json" ] && echo "  (npm scripts in package.json)" ;;
      python)    echo "  (pyproject.toml / requirements.txt)" ;;
      rust)      (cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && cargo check --quiet 2>&1 | head -3) ;;
    esac
    # S3: Secrets
    SECRETS=$(ow_secrets_sweep "$OVAV_CONSUMER_ROOT" 2>/dev/null || echo "FOUND: 0")
    if [[ "$SECRETS" == "FOUND: 0" ]]; then
      echo "  PASS: no secrets"
    else
      echo "  FAIL: $SECRETS"
      [ "$OWV_VERBOSE" = "1" ] && echo "    → Fix: remove secrets from tracked files"
      OWV_FAIL=1
    fi
    if [ "$OWV_FAIL" = "1" ]; then
      echo ""
      echo "  Some validators failed. Fix issues above, then re-run: owv"
      exit 1
    fi
    echo "  Tip: run 'owd' to finalize when ready."
    ;;

  s)  # ows — sync + maintenance
    MODE="${1:-default}"
    cd "$OVAV_CONSUMER_ROOT" 2>/dev/null
    case "$MODE" in
      default)
        git fetch origin 2>&1 | head -3
        git worktree prune 2>&1 | head -2
        git worktree list 2>&1 | head -5
        ;;
      --rebase)
        git fetch origin 2>&1 | head -3
        git rebase origin/develop 2>&1 | head -5
        ;;
      --full)
        git fetch origin 2>&1 | head -3
        git rebase origin/develop 2>&1 | head -5
        git worktree prune 2>&1 | head -2
        git gc --auto 2>&1 | head -2
        ;;
    esac
    ;;

  clean)  # owclean — spike TTL + orphan cleanup
    local_repos_root="$OVAV_CONSUMER_ROOT"
    cd "$local_repos_root" 2>/dev/null || cd /
    pruned=0
    git worktree prune 2>&1 | head -2

    # Spike TTL cleanup (from global worktree list, not local — spikes may be anywhere)
    while IFS= read -r wt_dir; do
      if [ -f "$wt_dir/.ovav/.spike-created-at" ]; then
        created=$(cat "$wt_dir/.ovav/.spike-created-at" 2>/dev/null || echo 0)
        ttl=$(cat "$wt_dir/.ovav/.spike-ttl-hours" 2>/dev/null || echo 48)
        now=$(date -u +%s)
        age=$(( (now - created) / 3600 ))
        if [ "$age" -gt "$ttl" ]; then
          echo "  Removing expired spike (age ${age}h > ${ttl}h): $wt_dir"
          # Worktree must be removed from its own worktree root
          wt_parent=$(dirname "$wt_dir")
          wt_branch=$(basename "$wt_dir")
          # Find git dir containing this worktree's metadata
          (cd "$wt_parent/.." 2>/dev/null && git worktree remove --force "$wt_dir" 2>/dev/null) || rm -rf "$wt_dir"
          pruned=$((pruned + 1))
        else
          echo "  Spike OK (age ${age}h <= ${ttl}h): $wt_dir"
        fi
      fi
    done < <(git worktree list --porcelain 2>/dev/null | grep '^worktree ' | awk '{print $2}')

    # Auto-clean prunable in current repo
    (cd "$local_repos_root" && git worktree list --porcelain 2>/dev/null) | awk 'NF==4 && $NF=="true" {print $1}' | while read -r path; do
      (cd "$local_repos_root" && git worktree remove --force "$path" 2>/dev/null) && pruned=$((pruned + 1)) && echo "  removed prunable: $path"
    done
    echo "  Total pruned: $pruned"
    ;;

  m)  # owm — move with validation
    if [ -z "${1:-}" ] || [ -z "${2:-}" ]; then
      echo "Usage: owm <src> <dest>" >&2; exit 1
    fi
    (cd "$OVAV_CONSUMER_ROOT" && git worktree move "$1" "$2") 2>&1 | head -3
    ;;

  x)  # owx — cross-branch cherry-pick or route changes
    src="${1:?Usage: owx <src-branch> <commit|target>}"
    sha="${2:-}"
    PROFILE="$(ow_detect_profile "$src")"
    TARGET_DIR="$(ow_compute_target_path "cherry-pick-$src-$(date +%s)" "$OVAV_CONSUMER_ROOT")"
    mkdir -p "$(dirname "$TARGET_DIR")"
    (cd "$OVAV_CONSUMER_ROOT" && git worktree add -b "cherrypick-$src-$(date +%s)" "$TARGET_DIR" "$src" 2>&1 | head -3)
    if [ -n "$sha" ]; then
      (cd "$TARGET_DIR" && git cherry-pick "$sha" 2>&1 | head -5)
    fi
    echo "PWD_NOW=$TARGET_DIR"
    ;;

  a)  # owa — abort
    cur=$(cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "none")
    echo "[ovav-owa] Abort: clean up '$(pwd)' — branch=$cur"
    echo "  Run: git worktree remove --force \$(pwd)"
    ;;

  r)  # owr — rescue
    echo "[ovav-owr] Rescue: broken worktrees:"
    (cd "$OVAV_CONSUMER_ROOT" && git worktree list --porcelain 2>/dev/null | grep -B1 "prunable" | head -10)
    echo "  Run: ovav-owclean to auto-prune"
    ;;

  lk)  # owlk — lock
    wt="${1:-$(pwd)}"
    (cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && git worktree lock --reason "OVAV lock by $OVAV_CONSUMER_ID at $(date -u +%Y-%m-%dT%H:%M:%SZ)" "$wt") 2>&1 | head -3
    ;;

  suggest)  # SU-10: owsuggest with --explain
    EXPLAIN=0
    if [ "${1:-}" = "--explain" ]; then EXPLAIN=1; shift; fi
    # Generate suggestions based on repo state
    echo "[owsuggest] Analyzing $OVAV_CONSUMER_ROOT..."
    BRANCHES=$(cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && git branch -a 2>/dev/null | wc -l || echo 0)
    UNCOMMITTED=$(cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && git status --porcelain 2>/dev/null | wc -l || echo 0)
    STASH_COUNT=$(cd "$OVAV_CONSUMER_ROOT" 2>/dev/null && git stash list 2>/dev/null | wc -l || echo 0)
    echo ""
    if [ "$UNCOMMITTED" -gt 0 ]; then
      echo "  → owc feature/<name>  (create worktree for your $UNCOMMITTED uncommitted changes)"
      [ "$EXPLAIN" = "1" ] && echo "    Why: you have $UNCOMMITTED uncommitted files. owc creates an isolated branch."
    fi
    if [ "$STASH_COUNT" -gt 0 ]; then
      echo "  → ows  (sync: fetch + prune stashed work)"
      [ "$EXPLAIN" = "1" ] && echo "    Why: you have $STASH_COUNT stashed changes that may need attention."
    fi
    if [ "$BRANCHES" -gt 5 ]; then
      echo "  → owclean  (prune $BRANCHES branches)"
      [ "$EXPLAIN" = "1" ] && echo "    Why: $BRANCHES branches may include orphans from previous worktrees."
    fi
    echo "  → owd  (finalize current worktree when ready)"
    [ "$EXPLAIN" = "1" ] && echo "    Why: merges your feature branch to develop with 6-stage validation."
    echo ""
    ;;

  *)
    echo "Usage: ow{c,d,l,v,s,clean,m,x,a,r,lk,prep} [args]" >&2
    exit 2
    ;;
esac
