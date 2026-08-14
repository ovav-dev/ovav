# IT v0.2 Keybinding Contract

**Status:** Active (enforced by `it_keybindings` validator, ID=72 in `ovav validate`)
**Fragment path:** `workstation/configs/intelligent-terminal/settings-fragment.json`
**Date:** 2026-08-14

## Why this contract exists

On 2026-08-14, `ovav validate` was passing 71/71 but keybindings were silently
broken in practice. A previous fix (commit `e2860ec`) had written "47 explicit
keybindings to bypass IT canonicalization" but 13 had `"id": null` (no action)
and 4 had wrong action IDs (directional pane movement bound to `MovePaneToTab0`).

**The lesson:** A green validator suite does not mean features work. Visual
configurations like keybindings need structural validation that catches semantic
errors, not just parse errors.

This contract is enforced by the `it_keybindings` Go validator
(`go-runtime/internal/validators/it_keybindings.go`). Any future commit that
violates these rules fails `ovav validate` BEFORE IT can silently drop the
broken entries.

## Rules (enforced by validator)

### Rule 1 — Every keybinding MUST have a non-null, non-empty `id`

```jsonc
// ❌ FAILS validator (NULL_ID)
{ "id": null, "keys": "ctrl+c" }
{ "id": "",   "keys": "ctrl+c" }

// ✅ PASSES validator
{ "id": "Terminal.CopyToClipboard", "keys": "ctrl+c" }
```

**Why:** IT parses keybindings but only executes those with a recognized `id`.
Entries with `id: null` are silently ignored — the key is "bound" but does
nothing.

### Rule 2 — Every `id` MUST resolve to a known action

Resolution order:

1. Built-in IT v0.2 actions (full list in `itBuiltinActions` map in the validator)
2. Custom actions defined in the same fragment's `actions` array

```jsonc
// ❌ FAILS validator (UNRESOLVED_ID) — typo, not a real action
{ "id": "Terminal.CopyToClip", "keys": "ctrl+c" }

// ✅ PASSES validator — custom action defined in fragment
{
  "id": "OVAV.tab",
  "keys": "ctrl+alt+t"
}
// ...and later in the same fragment:
{ "id": "OVAV.tab", "command": { "action": "newTab", "profile": "OVAV" } }
```

### Rule 3 — Every entry MUST have a non-empty `keys` field

```jsonc
// ❌ FAILS validator (EMPTY_KEYS)
{ "id": "Terminal.CopyToClipboard", "keys": "" }

// ✅ PASSES validator
{ "id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c" }
```

### Rule 4 (warning) — Same `keys` value bound to multiple distinct IDs

```jsonc
// ⚠️ WARNS validator (DUPLICATE_KEY) — likely a copy-paste mistake
{ "id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c" }
{ "id": "Terminal.PasteFromClipboard", "keys": "ctrl+shift+c" }
```

This is allowed by IT (later entry wins) but is almost always an error.

### Rule 5 (warning) — Empty keybindings list

```jsonc
// ⚠️ WARNS validator (EMPTY_KEYBINDINGS) — IT will use built-in defaults
{ "keybindings": [] }
```

If you intentionally want IT defaults, remove the `keybindings` key entirely.

## IT v0.2 Built-in action reference (canonical subset)

### Clipboard

| Action | Notes |
|--------|-------|
| `Terminal.CopyToClipboard` | Copy selection. IT passes `Ctrl+C` to shell when no selection. |
| `Terminal.PasteFromClipboard` | Paste from clipboard. |

### Tabs

| Action | Notes |
|--------|-------|
| `Terminal.OpenNewTab` | New tab (default profile). |
| `Terminal.CloseTab` | Close current tab. |
| `Terminal.ClosePane` | Close focused pane. |
| `Terminal.CloseOtherTabs` | Close all tabs except current. |
| `Terminal.NextTab` / `Terminal.PrevTab` | Cycle tabs. |
| `Terminal.SwitchToTab0` … `Terminal.SwitchToTab7` | Direct switch. |

### Pane management

| Action | Notes |
|--------|-------|
| `Terminal.SplitVertical` / `Terminal.SplitHorizontal` | Split. |
| `Terminal.SplitPaneUp` / `Down` / `Left` / `Right` | Directional split. |
| `Terminal.MovePaneUp` / `Down` / `Left` / `Right` | Move focused pane. |
| `Terminal.MoveFocusUp` / `Down` / `Left` / `Right` | Move focus. |
| `Terminal.SwapPaneUp` / `Down` / `Left` / `Right` | Swap with neighbor. |
| `Terminal.MovePaneToTab0` … `Terminal.MovePaneToTab8` | Specific tab (rare; use directional MovePane* for normal shortcuts). |
| `Terminal.TogglePaneZoom` | Zoom/unzoom pane. |

### Font

| Action | Notes |
|--------|-------|
| `Terminal.IncreaseFontSize` | +1 step. |
| `Terminal.DecreaseFontSize` | -1 step. |
| `Terminal.ResetFontSize` | Back to defaults.font.size. |

### UI

| Action | Notes |
|--------|-------|
| `Terminal.ToggleFullscreen` | Fullscreen toggle. |
| `Terminal.ToggleAlwaysOnTop` | Window always-on-top. |
| `Terminal.ToggleCommandPalette` | Open palette. |
| `Terminal.ReloadCommandPalette` | Refresh palette cache (rare). |
| `Terminal.OpenSettingsFile` | Open settings.json in default editor. |
| `Terminal.OpenSystemMenu` | Window system menu. |

### Search

| Action | Notes |
|--------|-------|
| `Terminal.FindText` | Open find dialog. |

The full set (70+) lives in `itBuiltinActions` map in
`go-runtime/internal/validators/it_keybindings.go`. Add new built-in IDs there
when upgrading to newer IT versions.

## Testing the contract

```bash
# Run only this validator
go run -C go-runtime ./cmd/ovav/ validate it_keybindings

# Run the Go test suite for this validator
go test -C go-runtime ./internal/validators/ -run TestITKeybindings -v
```

## References

- **Validator:** `go-runtime/internal/validators/it_keybindings.go`
- **Tests:** `go-runtime/internal/validators/it_keybindings_test.go`
- **Fragment:** `workstation/configs/intelligent-terminal/settings-fragment.json`
- **Fix commits:** `bc1fb2b` (17 broken bindings), `a66bc1d` (validator added)
- **ADR:** see also `docs/architecture/ADR-005-it-keybindings-regression.md` (proposed)

## Change history

| Date | Change | Commit |
|------|--------|--------|
| 2026-08-14 | Initial contract + validator | `a66bc1d` |
| 2026-08-14 | Repair 17 broken keybindings | `bc1fb2b` |
