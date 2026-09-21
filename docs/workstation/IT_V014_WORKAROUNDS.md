# IT v0.1.4 Workarounds Catalog

**Status:** Active (until IT GA release)
**Date:** 2026-08-14
**Related:** ADR-005 (Phase 3: IT upgrade path)

## Why this document exists

Intelligent Terminal (IT) v0.1.4 is an **alpha release** (`LocalCache/Local/IntelligentTerminal/hooks-upgrade-state.json`
shows version 0.1.4). CEO chose to keep IT and document workarounds rather
than downgrade to Windows Terminal.

This catalog lists known IT v0.1.4 issues + workarounds, so future
maintainers know what's compensating for IT bugs vs what's intentional.

## How to use

When you encounter a keybinding/UX issue in IT:

1. **Check this catalog first.** Maybe there's a known workaround.
2. **If not documented,** reproduce with `cat -v` (raw byte capture) and add
   to this catalog.
3. **When IT GA ships,** remove the workarounds in batches.

## Known issues + workarounds

### Issue 1: IT v0.1.4 doesn't render visual selection for shift+arrow (CONFIRMED)

**Symptom:** Pressing shift+arrow in IT doesn't visually highlight text.
The terminal doesn't enter selection mode.

**Workaround:** None at IT level. CEO confirmed IT v0.1.4 ships with
shift+arrow handling, but it appears broken or inactive in this version.

**Detection:** `bind -p | grep -E "1;2[ABCD]"` should show no bash bindings
(in IT default behavior, IT intercepts before bash).

**Status:** UNRESOLVED (IT alpha bug). Escalate to Phase 3 (IT upgrade path).

### Issue 2: IT v0.1.4 may strip unrecognized keybindings (CONFIRMED via test)

**Symptom:** Some custom keybindings disappear after IT canonicalizes
settings.json. Suspected: keys not in IT's known set get stripped.

**Workaround:** Document every keybinding we use. If a key disappears,
add explicit `Terminal.X` action reference in fragment.

**Detection:** `ovav validate` (it_keybindings + it_live_keybindings
validators catch missing entries).

**Status:** PARTIAL (validators detect; no prevention).

### Issue 3: IT v0.1.4 may ignore ctrl+v (CONFIRMED by CEO)

**Symptom:** Pressing Ctrl+V in IT bash inserts `^V` literal (bash
readline quoted-insert).

**Workaround:** Add `ctrl+v` → `Terminal.PasteFromClipboard` binding to
the fragment. (Done in commit `dc4289b`.)

**Detection:** `ovav validate` checks if `ctrl+v` is in the fragment.

**Status:** WORKAROUND APPLIED.

### Issue 4: IT settings.json has stale settingsHash in state.json (OBSERVED)

**Symptom:** After deploy, `state.json`'s `settingsHash` may not match the
actual hash of `settings.json`. IT may not auto-detect changes.

**Workaround:** CEO must restart IT completely (close all windows) or
press Ctrl+Shift+R to force reload. Document this prominently in
operator-facing docs.

**Detection:** Compare `state.json::settingsHash` with computed hash of
`settings.json`. Drift = manual restart needed.

**Status:** OPERATOR ACTION REQUIRED. Phase 1 (anti-drift core) should add
auto-reload via Win32 API.

### Issue 5: IT may canonicalize settings.json on save (SUSPECTED)

**Symptom:** Direct python `open(LIVE, 'w')` writes 48 entries that persist
for 60+ seconds. But after deploy script runs, IT might revert to 47
in some scenarios (not yet reproduced reliably).

**Workaround:** Direct write via python `Path.replace()` between sibling
files (cross-FS safe). Used in `_deploy-write-live.py` (commit `eb066cd`).

**Detection:** Compare live keybinding count to fragment count.

**Status:** UNCONFIRMED. May not be IT behavior — may be measurement
artifact.

## Workaround tracking

| Workaround | Commit | Status | Remove when |
|-----------|--------|--------|-------------|
| ctrl+v binding added | `dc4289b` | Applied | IT v0.1.4+ handles ctrl+v natively |
| shift+arrow unbound in bash readline | `dc4289b` | Applied | IT visual selection works |
| `_deploy-write-live.py` (python helper) | `eb066cd` | Applied | WSL cross-FS bug fixed (Linux kernel ≥ ?) |
| Path-aware fragment resolution | `eb066cd` | Applied | Always (it's correct architecture) |
| Operator restart after deploy | (manual) | Required | Phase 1: auto-reload via Win32 API |
| Manual `bind -f ~/.inputrc` for new bindings | (manual) | Required | Phase 2: auto-remediation |

## Upgrade path: IT v0.1.4 → IT GA

When IT releases a GA version:

1. **Identify IT GA version** — check `Microsoft Store → Intelligent Terminal`
   or `hooks-upgrade-state.json` (after update).
2. **Test all keybindings** — run E2E test that exercises every entry in
   the fragment. See `workstation/tests/test-it-keybindings.sh` (TODO:
   create in Phase 3).
3. **Remove workarounds** in reverse order:
   - Remove ctrl+v workaround if IT GA has it
   - Restore shift+arrow bindings in inputrc (with new architecture)
   - Re-evaluate `_deploy-write-live.py` necessity
4. **Run validator** — `ovav validate` should pass 74/74.
5. **Update this document** — mark workarounds as REMOVED, link to commit.

## Tracking new issues

When you find a new IT v0.1.4 issue:

```markdown
### Issue N: <short description>

**Symptom:** <what CEO/user sees>
**Workaround:** <how OVAV compensates>
**Detection:** <how to test for this>
**Status:** <APPLIED | UNRESOLVED | PARTIAL>
```

Then update the tracking table.

## References

- **ADR-005:** Anti-drift architecture plan (Phase 3 covers IT upgrade)
- **docs/workstation/IT_DEPLOY_PIPELINE.md:** Deploy workflow
- **docs/workstation/IT_KEYBINDINGS_CONTRACT.md:** Keybinding contract
- **`workstation/scripts/deploy-it-keybindings.sh`:** Surgical deploy
- **`go-runtime/internal/validators/it_keybindings.go`:** Fragment validator
- **`go-runtime/internal/validators/it_live_keybindings.go`:** Live validator
