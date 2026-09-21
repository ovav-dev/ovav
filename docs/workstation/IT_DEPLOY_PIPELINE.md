# IT Keybindings Deploy Pipeline

**Status:** Active (2026-08-14)
**Purpose:** Synchronize the OVAV IT keybindings fragment into the live Intelligent Terminal settings.json on the user's machine.

## Why this exists

The OVAV repo has `workstation/configs/intelligent-terminal/settings-fragment.json`
as the **source-of-truth** for IT keybindings. IT itself reads/writes its own
`settings.json` (e.g. `/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json`).

Until 2026-08-14 there was no explicit deploy step. Fragment changes were
silently drifting from the live state. CEO reported `shift+arrow → D + Windows
beep` regression because the fragment was fixed in commit `bc1fb2b` but the live
file still had 13 `id:null` + 4 wrong-action entries — IT's canonicalization
choked on the malformed bindings, escape sequences were corrupted, and the
`D`/`B`/`A`/`C` trailing bytes of arrow escape sequences leaked through as
literal characters.

## Components

### 1. Source of truth: `workstation/configs/intelligent-terminal/settings-fragment.json`

Validated by `it_keybindings` validator (#72 in `ovav validate`). Catches
NULL_ID, UNRESOLVED_ID, EMPTY_KEYS, DUPLICATE_KEY in the **fragment**.

### 2. Live state: `~/AppData/Local/Packages/Microsoft.IntelligentTerminal_*/LocalState/settings.json`

Validated by `it_live_keybindings` validator (#73 in `ovav validate`). Catches
the same issues in the **live file** — drift detection.

### 3. Deploy: `workstation/scripts/deploy-it-keybindings.sh`

Surgical merge of fragment into live file:
- Idempotent (safe to re-run)
- Backed up (timestamped in `$HOME/.ovav-backups/deploy-it-YYYYMMDD-HHMMSS/`)
- Validates merged JSON parses cleanly
- Validates 0 null-id keybindings in merged result (catches bad fragments)
- Dry-run mode via `OVAV_DRY_RUN=1`

### 4. Audit trail: `workstation/configs/intelligent-terminal/DEPLOY_LOG.md`

Records every successful deploy with timestamp, operator, paths, and result.

## Deploy pipeline flow

```
┌─────────────────────────�
│ developer edits        │
│ fragment in repo       │
└──────────┬──────────────┘
           │
           ▼
   [ ovav validate ]    ◄── catches issues BEFORE deploy
   it_keybindings: PASS │
           │
           ▼
   [ commit fix ]       ◄── atomic commit per concern
           │
           ▼
   [ merge to develop ] �── fast-forward
           │
           ▼
   [ rebuild bin/ovav ] ◄── gitignored binary
           │
           ▼
   [ run deploy ]       ◄── OVAV_LIVE_IT_SETTINGS=... bash deploy-it-keybindings.sh
   it_live_keybindings:  │
   PASS expected        │
           │
           ▼
   [ update DEPLOY_LOG ] ◄── append row with timestamp + result
           │
           ▼
   [ restart IT ]       ◄── or Ctrl+Shift+R (Terminal.ReloadCommandPalette)
```

## When to run the deploy

| Trigger | Action |
|---------|--------|
| New keybinding added to fragment | Re-run deploy |
| Keybinding id fixed in fragment | Re-run deploy |
| Custom action added/renamed in fragment | Re-run deploy |
| IT version upgrade (new built-in actions) | Re-run deploy (built-in map may need update) |
| `ovav validate` reports `it_live_keybindings` FAIL | **MUST re-run deploy** |
| `ovav validate` reports `it_keybindings` PASS but `it_live_keybindings` FAIL | **Drift detected** — re-run deploy |

## How to run the deploy

### Standard (Linux + WSL)

```bash
# In an OVAV worktree, with WSL access to /mnt/c/Users/Alexa/
OVAV_LIVE_IT_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json" \
  bash workstation/scripts/deploy-it-keybindings.sh
```

### Dry-run (preview only, no write)

```bash
OVAV_LIVE_IT_SETTINGS="/mnt/c/Users/Alexa/.../settings.json" \
  OVAV_DRY_RUN=1 \
  bash workstation/scripts/deploy-it-keybindings.sh
```

### Custom backup location

```bash
OVAV_LIVE_IT_SETTINGS="/mnt/c/Users/Alexa/.../settings.json" \
  OVAV_BACKUP_DIR="/custom/path/backups" \
  bash workstation/scripts/deploy-it-keybindings.sh
```

### After deploy: trigger IT reload

The live IT will pick up changes when:
- User restarts Intelligent Terminal (close + reopen), OR
- User presses `Ctrl+Shift+R` (bound to `Terminal.ReloadCommandPalette`)

IT also auto-detects settings.json changes within ~30 seconds and reloads,
but explicit reload is faster.

## Safety guarantees

The deploy script will **ABORT** if:

1. `OVAV_LIVE_IT_SETTINGS` is unset or path doesn't exist → exit 2/1
2. `jq` is not on PATH → exit 3
3. Merged JSON fails `jq empty` validation → exit 1 (original restored from backup)
4. Merged keybindings have any null/empty id → exit 1 (no write)
5. Live file becomes unwritable → `mv` fails, exit propagates

If anything goes wrong, the live file is **never modified** (atomic write at the
end via `mv` of a temp file).

## Validator integration

`ovav validate` runs both validators by default:

```
✅ IT Keybindings Validator       PASS — 47 keybindings validated  (fragment)
✅ IT Live Keybindings Validator  PASS — 47 live keybindings validated  (live)
```

If `it_live_keybindings` FAILS, the message is:

```
FAIL — N keybinding issue(s) in live IT settings (/path/...). 
Re-run: workstation/scripts/deploy-it-keybindings.sh
```

This is intentional: the validator names the remediation step directly.

## References

- **Fragment:** `workstation/configs/intelligent-terminal/settings-fragment.json`
- **Deploy script:** `workstation/scripts/deploy-it-keybindings.sh`
- **Audit log:** `workstation/configs/intelligent-terminal/DEPLOY_LOG.md`
- **Contract:** `docs/workstation/IT_KEYBINDINGS_CONTRACT.md`
- **Validator #72 (fragment):** `go-runtime/internal/validators/it_keybindings.go`
- **Validator #73 (live):** `go-runtime/internal/validators/it_live_keybindings.go`
- **Install script (calls deploy):** `workstation/scripts/install.sh`
- **Policy:** `config/workstation/ovav-workstation-scripts.yaml`
- **Registry:** `.ovav/registry/tool_configs.yaml` → `ovav_workstation_scripts`
