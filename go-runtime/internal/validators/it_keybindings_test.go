package validators

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFragment is a test helper that writes a minimal IT settings-fragment.json
// to a temp directory and returns the root path.
func writeFragment(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "workstation", "configs", "intelligent-terminal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings-fragment.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write fragment: %v", err)
	}
	return root
}

func TestITKeybindings_ValidFragment_Pass(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c"},
			{"id": "Terminal.MoveFocusUp", "keys": "alt+up"},
			{"id": "OVAV.tab", "keys": "ctrl+alt+t"}
		],
		"actions": [
			{"id": "OVAV.tab", "command": {"action": "newTab"}}
		]
	}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s — issues: %v", res.Status, res.Message, res.Issues)
	}
}

func TestITKeybindings_NullID_Fail(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": null, "keys": "ctrl+c"},
			{"id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c"}
		]
	}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s: %s", res.Status, res.Message)
	}
	if len(res.Issues) == 0 {
		t.Fatal("expected at least one issue, got none")
	}
}

func TestITKeybindings_EmptyID_Fail(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "", "keys": "ctrl+c"}
		]
	}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for empty id, got %s: %s", res.Status, res.Message)
	}
}

func TestITKeybindings_UnresolvedID_Fail(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "Terminal.NotARealAction", "keys": "ctrl+n"}
		]
	}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for unresolved id, got %s: %s", res.Status, res.Message)
	}
}

func TestITKeybindings_FragmentMissing_Fail(t *testing.T) {
	v := NewITKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected fail when fragment missing, got %s: %s", res.Status, res.Message)
	}
}

func TestITKeybindings_InvalidJSON_Fail(t *testing.T) {
	root := writeFragment(t, `{ "keybindings": "this is not an array" }`)
	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail on invalid JSON, got %s: %s", res.Status, res.Message)
	}
}

func TestITKeybindings_DuplicateKey_Warn(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c"},
			{"id": "Terminal.PasteFromClipboard", "keys": "ctrl+shift+c"}
		]
	}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("expected warn for duplicate key, got %s: %s — issues: %v", res.Status, res.Message, res.Issues)
	}
}

func TestITKeybindings_EmptyKeys_Fail(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "Terminal.CopyToClipboard", "keys": ""}
		]
	}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for empty keys, got %s: %s", res.Status, res.Message)
	}
}

func TestITKeybindings_EmptyKeybindingsList_Warn(t *testing.T) {
	root := writeFragment(t, `{"keybindings": []}`)

	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("expected warn for empty keybindings list, got %s: %s", res.Status, res.Message)
	}
}

func TestITKeybindings_AllBuiltinActionsResolve(t *testing.T) {
	// Sanity: every entry in itBuiltinActions should parse as a non-empty string.
	// (Catches typos like "" keys, accidental duplicates in the map.)
	seen := make(map[string]bool)
	for id := range itBuiltinActions {
		if id == "" {
			t.Fatal("itBuiltinActions contains empty id")
		}
		if seen[id] {
			t.Fatalf("itBuiltinActions contains duplicate id: %q", id)
		}
		seen[id] = true
	}
	if len(seen) < 50 {
		t.Fatalf("expected at least 50 built-in IT actions, got %d (if you intentionally added more actions, bump this threshold)", len(seen))
	}
}

// TestITKeybindings_OVAVPaneManagement_Pass — regression for the round-3
// CEO-mandated pane management scheme:
//
//	alt+a            → Terminal.SplitPaneDown   (open new pane, coherent default)
//	alt+shift+a      → Terminal.SplitPaneRight  (open new pane, alternate direction)
//	alt+x            → Terminal.ClosePane        (close focused pane)
//	ctrl+shift+z     → Terminal.TogglePaneZoom   (maximize/restore pane)
//
// This test ensures all four resolve against the builtin-actions allowlist and
// don't collide with each other or with previously-validated alt+arrow focus bindings.
func TestITKeybindings_OVAVPaneManagement_Pass(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "Terminal.SplitPaneDown",  "keys": "alt+a"},
			{"id": "Terminal.SplitPaneRight", "keys": "alt+shift+a"},
			{"id": "Terminal.ClosePane",      "keys": "alt+x"},
			{"id": "Terminal.TogglePaneZoom", "keys": "ctrl+shift+z"},
			{"id": "Terminal.MoveFocusUp",    "keys": "alt+up"},
			{"id": "Terminal.MoveFocusDown",  "keys": "alt+down"},
			{"id": "Terminal.MoveFocusLeft",  "keys": "alt+left"},
			{"id": "Terminal.MoveFocusRight", "keys": "alt+right"}
		]
	}`)
	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "pass" {
		t.Fatalf("OVAV pane management keys must pass; got %s: %s — issues: %v",
			res.Status, res.Message, res.Issues)
	}
}

// TestITKeybindings_DuplicateKey_DifferentAction_Warn — two keybindings that
// share a key combination but map to different actions MUST be flagged so the
// IT doesn't silently drop one. Catches accidental key collisions during
// fragment edits. The validator emits a WARN (not FAIL) because Windows
// Terminal silently drops the duplicate — it's a UX bug, not a security bug.
func TestITKeybindings_DuplicateKey_DifferentAction_Warn(t *testing.T) {
	root := writeFragment(t, `{
		"keybindings": [
			{"id": "Terminal.ClosePane",      "keys": "alt+x"},
			{"id": "Terminal.OpenSystemMenu", "keys": "alt+x"}
		]
	}`)
	v := NewITKeybindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("expected warn for duplicate key alt+x mapped to different actions, got %s: %s",
			res.Status, res.Message)
	}
}
