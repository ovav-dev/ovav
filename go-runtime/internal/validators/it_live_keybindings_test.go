package validators

import (
	"os"
	"path/filepath"
	"testing"
)

// writeLiveSettings is a test helper that writes a minimal live IT
// settings.json to a custom path and returns it. The path is also
// exposed via OVAV_LIVE_IT_SETTINGS so the validator can find it.
func writeLiveSettings(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write live settings: %v", err)
	}
	t.Setenv("OVAV_LIVE_IT_SETTINGS", path)
	return path
}

func TestITLiveKeybindings_NoLivePath_Skip(t *testing.T) {
	// No env var, no default path → SKIP
	t.Setenv("OVAV_LIVE_IT_SETTINGS", "")
	// Also clear default paths by pointing to a non-existent location
	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "skip" {
		t.Fatalf("expected skip when no live path, got %s: %s", res.Status, res.Message)
	}
}

func TestITLiveKeybindings_ValidLiveSettings_Pass(t *testing.T) {
	writeLiveSettings(t, `{
		"keybindings": [
			{"id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c"},
			{"id": "Terminal.OpenNewTab", "keys": "ctrl+t"}
		]
	}`)

	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s — issues: %v", res.Status, res.Message, res.Issues)
	}
}

func TestITLiveKeybindings_NullID_Fail(t *testing.T) {
	writeLiveSettings(t, `{
		"keybindings": [
			{"id": null, "keys": "ctrl+c"},
			{"id": "Terminal.CopyToClipboard", "keys": "ctrl+shift+c"}
		]
	}`)

	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s: %s", res.Status, res.Message)
	}
}

func TestITLiveKeybindings_UnresolvedID_Fail(t *testing.T) {
	writeLiveSettings(t, `{
		"keybindings": [
			{"id": "Terminal.NotReal", "keys": "ctrl+n"}
		]
	}`)

	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected fail for unresolved id, got %s: %s", res.Status, res.Message)
	}
}

func TestITLiveKeybindings_InvalidJSON_Fail(t *testing.T) {
	writeLiveSettings(t, `{ not valid json`)
	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected fail on invalid JSON, got %s: %s", res.Status, res.Message)
	}
}

func TestITLiveKeybindings_EmptyKeybindings_Warn(t *testing.T) {
	writeLiveSettings(t, `{"keybindings": []}`)

	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "warn" {
		t.Fatalf("expected warn for empty keybindings, got %s: %s", res.Status, res.Message)
	}
}

func TestITLiveKeybindings_CustomActionID_Resolves(t *testing.T) {
	writeLiveSettings(t, `{
		"keybindings": [
			{"id": "OVAV.tab", "keys": "ctrl+alt+t"}
		],
		"actions": [
			{"id": "OVAV.tab", "command": {"action": "newTab"}}
		]
	}`)

	v := NewITLiveKeybindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "pass" {
		t.Fatalf("custom action should resolve, got %s: %s — issues: %v", res.Status, res.Message, res.Issues)
	}
}
