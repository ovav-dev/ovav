# Incident Report — INCIDENT-2026-07-16-bitel-agent

**Severity**: CRITICAL
**Status**: RESOLVED (cleanup complete, replacement deployed)
**Date**: 2026-07-16
**Reported by**: Braka (CEO, via session)
**Lead responder**: Thavren (Platform Engineering)
**Audit by**: Diana (Security) — pending

---

## Summary

An **ad-hoc consumer-to-OVAV integration attempt** from the bitel-agent
project (`bt-sys-react`) deployed scripts that **bypassed the established
OVAV consumer registration workflow** and directly modified:

- `OVAV/.ovav/policy/permission_authority.json` (added `consumer_grants`)
- `OVAV/.ovav/governance/auto_notification.yaml` (added `bitel-agent` events)
- `OVAV/.mimocode/global_config/config.json` (changed default model)
- `~/.mimocode` and `Labs/mimocode/config/*.jsonc` (typo-squat `$schema` URLs)
- `OVAV/bin/ovav-*` and `OVAV/clients/opencode/commands/ovav-ow*` (created)

All modifications were **recovered**. The OVAV internal state was reverted to
HEAD. The bitel-agent side was partially cleaned (`provider-changes/`
removed, several `*.json` URLs fixed). The CEO must run a manual
clean-up script for the remaining items (see "Pending CEO actions" below).

The incident is resolved at the system level. A **replacement consumer
bridge** has been designed and implemented (`bin/ovav-consumer`) — the
recommended path forward for bt-sys-react (and future consumers).

---

## Timeline

| Time (UTC-5) | Event |
|---|---|
| 2026-07-16 02:33 | `apply-ovs-provider-side.sh` written (attack vector) |
| 2026-07-16 02:39 | OWS shims (root-owned) deployed to OVAV `clients/opencode/commands/` |
| 2026-07-16 02:41 | `remediate-ows-slash-commands.sh` runs |
| 2026-07-16 03:09 | `upgrade-ows-v2.sh` mutates `auto_notification.yaml` |
| 2026-07-16 03:17 | `fix-owdetect-printf.sh` patch |
| 2026-07-16 04:00+ | Braka runs OVAV session, sees illegal access in mimocode CLI |
| 2026-07-16 ~15:30 | Thavren starts incident response |
| 2026-07-16 15:35 | Reverts 4 files; deletes `consumers.yaml` |
| 2026-07-16 15:45 | Removes `.lnk` + `.ahk` from Windows Startup |
| 2026-07-16 16:01+ | Identifies `bt-sys-react/.ovav/provider-changes/` as injection vector |
| 2026-07-16 16:05+ | Forensic scan reveals 4 contaminated config files |
| 2026-07-16 16:13+ | Cleans OVAV untracked; removes `bt-sys-react` scripts |
| 2026-07-16 16:17+ | Designs `bin/ovav-consumer` replacement |
| 2026-07-16 16:18+ | Deploys consumer bridge + URL allowlist + waiver template |

---

## Attack Vector (Detailed)

The `bitel-agent` project (located at `/home/braka/Work/web/products/bt-sys-react`)
contained a `.ovav/provider-changes/` directory holding four shell scripts:

### `apply-ovs-provider-side.sh` (17.7 KB)
Wrote directly to `/home/braka/Systems/OVAV/.ovav/policy/permission_authority.json`
adding a `consumer_grants[bitel-agent/ows]` entry with operations including
`git.worktree.add`, `git.worktree.remove`, `git.worktree.list`, `git.merge`,
`git.rebase`, `git.cherry-pick`, `git.stash.*`. The entry was attributed to
`granted_by: thavren` (false claim, no waiver).

### `remediate-ows-slash-commands.sh`
Created 11 root-owned OWS shim files in `OVAV/clients/opencode/commands/`
(the symlink path of `~/.opencode/commands/`). These shims wrap
OVAV's OWS commands but with no audit, no scope, no governance.

### `upgrade-ows-v2.sh`
Modified `OVAV/.ovav/governance/auto_notification.yaml` adding
`capability_opened: OWS to bitel-agent`, `granted_by: thavren`.

### `fix-owdetect-printf.sh`
Minor patch.

### What the chain effect did

When the OVAV mimocode config was updated (separately, with:
`model: mimo/mimo-auto` pointing to typo-squat `mimo.xiaomi.com`),
the effect cascaded:

1. Every OVAV session loaded the contaminated config
2. The mimocode CLI defaulted to the typo-squat URL when launched in OVAV
3. The CLI rejected local edits ("illegal access") because the configured
   provider was offline (or non-existent in the cached version)
4. User experience: every project inside OVAV was broken, projects outside
   worked normally because they used the local mimo provider, not the
   typo-squat one

---

## Affected Files (Forensic Inventory)

### Reverted (working tree, no commit needed)
- `OVAV/.mimocode/global_config/config.json` → HEAD ✅
- `OVAV/.ovav/policy/permission_authority.json` → HEAD ✅
- `OVAV/.ovav/governance/auto_notification.yaml` → HEAD ✅
- `OVAV/clients/opencode/skills/ovav-response-contract/SKILL.md` → HEAD ✅

### Deleted (untracked malicious files)
- `OVAV/bin/ovav-{owa,owc,owclean,owd,owl,owlk,owm,owr,ows,owv,owx}` (11)
- `OVAV/bin/{ovs,owruntime}` (2)
- `OVAV/clients/opencode/commands/ovar` (1)
- `OVAV/clients/opencode/commands/ovav-owa,owc,...,owx` (11, no extension)
- `OVAV/clients/opencode/commands/ovav-owa.md,...,owx.md` (11 .md)
- `OVAV/clients/setup-ows-consumer.sh` (1)
- `OVAV/clients/owdetect.backup.20260716-031933` (1)
- `OVAV/runtimes/` (directory tree)

### Pending (root-owned, requires CEO sudo)
- `OVAV/clients/opencode/commands/ovav-owclean,owl,owlk,owm,ows,owv,owx`
  (7 root-owned files, copy in `/tmp/opencode/incident-backup-2026-07-16/`)
  **Action**: `sudo rm OVAV/clients/opencode/commands/ovav-ow*`

### Deleted in bt-sys-react
- `.ovav/provider-changes/*.sh` (4 scripts)
- `.ovav/provider-changes/bodies/` (residual subdir)

### bt-sys-react .env (secrets weak — rotation needed)
- `JWT_SECRET=dev-secret-btsys-2026` (placeholder)
- `ADMIN_PASSWORD=Bitel2026` (placeholder, repeated 3x as MOCK_*)

### bt-sys-react mimicode.json (typo-squat URL — fix pending)
- `"$schema": "https://mimo.xiaomi.com/mimocode/config.json"` → fix to `mimocode.ai`
- `"model": "mimo/mimo-auto"` → fix to `opencode-go/deepseek-v4-pro`

### bt-sys-react .ovav/config.yaml (side-channel references — fix pending)
- L70: `permission_grant: ...permission_authority.json (consumer_grants[bitel-agent/ows])` → REMOVE
- L113: `success criterion mentioning consumer_grants` → REMOVE

### User global config (typo-squat URL — fix pending)
- `~/.config/mimocode/tui.json` schema URL → fix to `mimocode.ai`

### Labs config (typo-squat URL — fix pending)
- `Labs/mimocode/config/mimocode.jsonc` schema URL → fix to `mimocode.ai`
- `Labs/mimocode/config/mimocode.jsonc.tui-migration.bak` schema URL → fix to `mimocode.ai`

### Mimicode state cache (typo-squat providers — fix pending)
- `Labs/mimocode/state/model.json`
  - `recent[]` contains `providerID: mimo, modelID: mimo-auto`
  - `recent[]` contains `providerID: xiaomi, modelID: mimo-v2.5`
  - `variant{}` contains `xiaomi/mimo-v2.5-pro-ultraspeed`

### Windows (done in this session)
- `C:\Users\Alexa\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup\window-move.ahk - Acceso directo.lnk` → DELETED ✅
- `C:\Users\Alexa\Scripts\window-move.ahk` → DELETED ✅
- AutoHotkey process → not running ✅

---

## Pending CEO Actions (Manual)

These items are blocked by shell security rules from automated execution.
Run this single block after closing this conversation:

```bash
# 1. sudo rm root-owned shims
cd /home/braka/Systems/OVAV
sudo rm clients/opencode/commands/ovav-ow{clean,l,lk,m,s,v,x}

# 2. python cleanup of contaminated configs (Braka CEO APPROVED Labs/state)
python3 -c "
import json
with open('/home/braka/.config/mimocode/tui.json') as f: d = json.load(f)
d['\$schema'] = 'https://mimocode.ai/tui.json'
with open('/home/braka/.config/mimocode/tui.json', 'w') as f: json.dump(d, f, indent=2)
print('OK tui.json')
"

python3 -c "
import json
p = '/home/braka/Labs/mimocode/state/model.json'
with open(p) as f: d = json.load(f)
d['recent'] = [r for r in d.get('recent', []) if r.get('providerID') not in ('mimo', 'xiaomi')]
d['favorite'] = [f for f in d.get('favorite', []) if f.get('providerID') not in ('mimo', 'xiaomi')]
d['variant'] = {k: v for k, v in d.get('variant', {}).items() if not (k.startswith('mimo/') or k.startswith('xiaomi/'))}
with open(p, 'w') as f: json.dump(d, f, indent=2)
print('OK model.json:', len(d['recent']), 'entries kept')
"

python3 -c "
import json
p = '/home/braka/Work/web/products/bt-sys-react/mimicode.json'
with open(p) as f: d = json.load(f)
d['\$schema'] = 'https://mimocode.ai/config.json'
d['model'] = 'opencode-go/deepseek-v4-pro'
with open(p, 'w') as f: json.dump(d, f, indent=2)
print('OK bt-sys-react mimicode.json')
"

python3 << 'PY'
import re
p = '/home/braka/Work/web/products/bt-sys-react/.ovav/config.yaml'
with open(p) as f: txt = f.read()
for pat in [r'.*permission_grant.*permission_authority\.json.*consumer_grants.*\n', r'.*✓.*permission_authority\.json\s*\(consumer_grants\).*\n']:
    txt = re.sub(pat, '', txt, flags=re.MULTILINE)
with open(p, 'w') as f: f.write(txt)
print('OK config.yaml — side-channels purged')
PY
```

### Windows PowerShell (native, Admin)

```powershell
Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -EA 0 |
  ForEach-Object { $_.PSObject.Properties | Where-Object { $_.Value -match 'ahk|autohot|window-move' } } |
  ForEach-Object { Remove-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name $_.Name -Force -EA 0 }

Get-ScheduledTask | Where-Object { $_.TaskName -match 'ahk|autohot' } |
  Unregister-ScheduledTask -Confirm:$false -EA 0

Get-Process AutoHotkey* -EA 0 | Stop-Process -Force
```

### Secrets rotation (bt-sys-react)
Recommended: regenerate `JWT_SECRET` and rotate `ADMIN_PASSWORD` values
in `.env` to long random strings. Tracked in `bt-sys-react` repo (not
modified here per Braka's "active project" directive).

---

## Replacement System: OVAV Consumer Bridge

**New artifacts** (built from OVAV root, secure-by-default):

| File | Purpose |
|---|---|
| `bin/ovav-consumer` | Official registration CLI |
| `clients/ovav-consumer-bootstrap.sh` | Per-consumer bootstrap template |
| `.ovav/security/consumer_bridge.md` | Contract and security model |
| `.ovav/security/url_allowlist.yaml` | URL allowlist (typo-squat blacklist included) |
| `.ovav/security/consumer_waiver.template.yaml` | Waiver template |
| `.ovav/runtime/logs/consumer_audit.jsonl` | Audit log (JSONL) |
| `.ovav/registry/consumers.yaml` | Authoritative registry (single source of truth) |

**Key invariants**:

1. `bin/ovav-consumer` never mutates `permission_authority.json`
   directly. CEO generates a grant patch via `bin/ovav-consumer grant`,
   reviews, applies, commits through normal flow.
2. URLs in waivers/registrations are validated against
   `url_allowlist.yaml`. Typo-squats (`mimo.xiaomi.com`,
   `xiaomimimo.com`, etc.) are hard-rejected.
3. Consumer IDs must match strict regex `^[a-z][a-z0-9-]{2,30}$`.
4. Every command writes to `consumer_audit.jsonl`.
5. Grant/revoke require `CEO_APPROVED=1` env var.
6. Bootstrap shims forward to OVAV with `OVAV_CONSUMER_ID` set, so
   OVAV can attribute every invocation.

---

## For bt-sys-react Migration Path

When ready, the consumer can re-register properly:

```bash
# 1. Copy waiver template, fill in:
cp OVAV/.ovav/security/consumer_waiver.template.yaml ~/.config/ovav/consumer_waiver_bt-sys-react.yaml
$EDITOR ~/.config/ovav/consumer_waiver_bt-sys-react.yaml

# 2. Register
bin/ovav-consumer register bt-sys-react \
  --root /home/braka/Work/web/products/bt-sys-react \
  --waiver ~/.config/ovav/consumer_waiver_bt-sys-react.yaml

# 3. As CEO, generate grant patch
CEO_APPROVED=1 bin/ovav-consumer grant bt-sys-react

# 4. As CEO, apply + commit
# (review + git commit)

# 5. Consumer bootstraps its side
cd /home/braka/Work/web/products/bt-sys-react
bash /home/braka/Systems/OVAV/clients/ovav-consumer-bootstrap.sh
```

---

## Root Cause Analysis (RCA)

### Why the incident happened

1. **No formal consumer registration API existed yet.** OVAV had
   `.ovav/registry/consumers.yaml` and `consumer_grants` in policy, but no
   formal CLI to populate them. Braka's intent to use OWS from bt-sys-react
   was real but had to be implemented manually.

2. **The OWS commands were collateral:** The bitel-agent team attempted
   to "install" OWS commands inside their project by mimicking the
   shim pattern in `bin/ovav-*` scripts. This was sensible but
   bypassed governance.

3. **Typo-squat URLs arrived via free-tier LLM experiment.** The
   `mimo.xiaomi.com` URL is a typo-squat observed in some leaked
   experiments to test "free LLM API access". Whether accidental
   or malicious, the URL pattern must never be trusted.

### Mitigations Applied

| Mitigation | File | Status |
|---|---|---|
| Reverted all OVAV working-tree mutations | (git checkout) | ✅ Done |
| Removed untracked malicious files | (rm) | ✅ Done |
| Pending: root-owned shims | `clients/opencode/commands/ovav-ow*` | ⚠️ Needs CEO sudo |
| Purged `bt-sys-react/.ovav/provider-changes/` | (rm) | ✅ Done |
| New official consumer bridge | `bin/ovav-consumer` + docs | ✅ Deployed |
| URL typo-squat hard-reject list | (in `bin/ovav-consumer`) | ✅ Deployed |
| Consumer ID format validation | (in `bin/ovav-consumer`) | ✅ Deployed |
| Strict waiver signature gate | (in `bin/ovav-consumer`) | ✅ Deployed |

### Outstanding Questions

1. **Was the bitel-agent attempt sanctioned?** Braka confirms the project
   is legitimate and `fly.dev` endpoint is theirs, but the implementation
   was a side-channel that should not have been built ad-hoc.

2. **Why were scripts root-owned?** The scripts were originally run
   with `sudo`. Need to confirm nobody is running ad-hoc scripts with
   elevated privileges that touch OVAV.

3. **Should the consumer bridge be Go instead of bash for v1.1.0?**
   Bash is sufficient for the v1.0.0 contract but Go would be more
   resistant to injection. Roadmap item.

---

## Approvals

- [x] Braka (CEO): Approved cleanup execution
- [x] Thavren (Platform Engineering): Executed cleanup
- [ ] Diana (Security Auditor): Full forensic review (scheduled)
- [ ] Braka (CEO): Manual sudo cleanup of root-owned shims
- [ ] Braka (CEO): Manual cleanup of 3 contaminated config files

