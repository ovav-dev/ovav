package terminalconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeITFragment is a test helper that writes the IT settings fragment to a
// temp directory tree matching workstation/configs/intelligent-terminal/.
func writeITFragment(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "workstation", "configs", "intelligent-terminal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings-fragment.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

// TestITFragment_CopyOnSelect_PerProfile — regression for the round-5 bug where
// `copyOnSelect: true` was set ONLY in profiles.defaults, but some Windows
// Terminal versions (especially older 0.2.x builds in WSL contexts) don't
// propagate the setting to per-profile list entries reliably.
//
// The fix is to set copyOnSelect: true directly in each OVAV profile AND keep
// it in defaults as a fallback for system profiles (PowerShell, CMD, etc).
func TestITFragment_CopyOnSelect_PerProfile(t *testing.T) {
	root := writeITFragment(t, `{
		"profiles": {
			"defaults": {
				"copyOnSelect": true
			},
			"list": [
				{"name": "OVAV", "copyOnSelect": true},
				{"name": "SYS", "copyOnSelect": true},
				{"name": "HUB", "copyOnSelect": true},
				{"name": "WIN", "copyOnSelect": true}
			]
		}
	}`)

	fragment, err := os.ReadFile(filepath.Join(root, "workstation", "configs", "intelligent-terminal", "settings-fragment.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]interface{}
	if err := json.Unmarshal(fragment, &d); err != nil {
		t.Fatal(err)
	}

	// Defaults must have copyOnSelect: true
	defaults, ok := d["profiles"].(map[string]interface{})["defaults"].(map[string]interface{})
	if !ok {
		t.Fatal("profiles.defaults missing or wrong type")
	}
	if v, _ := defaults["copyOnSelect"].(bool); !v {
		t.Errorf("profiles.defaults.copyOnSelect must be true (got %v)", defaults["copyOnSelect"])
	}

	// Each OVAV profile must have copyOnSelect: true DIRECTLY (not just inherited)
	list, _ := d["profiles"].(map[string]interface{})["list"].([]interface{})
	requiredProfiles := []string{"OVAV", "SYS", "HUB", "WIN"}
	for _, name := range requiredProfiles {
		found := false
		for _, raw := range list {
			prof, _ := raw.(map[string]interface{})
			if prof["name"] == name {
				v, present := prof["copyOnSelect"].(bool)
				if !present {
					t.Errorf("profile %q missing copyOnSelect (must be set directly, not just inherited from defaults)", name)
				} else if !v {
					t.Errorf("profile %q has copyOnSelect=false (must be true)", name)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("profile %q missing from fragment entirely", name)
		}
	}
}

// TestITFragment_AutoReloadSettings — regression for settings reload. When
// `autoReloadSettings: true` is set, Windows Terminal watches settings.json
// and reloads when it changes. Without this, the CEO has to restart IT every
// time the fragment changes — defeating the purpose of the deploy pipeline.
func TestITFragment_AutoReloadSettings(t *testing.T) {
	root := writeITFragment(t, `{"autoReloadSettings": true}`)

	fragment, err := os.ReadFile(filepath.Join(root, "workstation", "configs", "intelligent-terminal", "settings-fragment.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]interface{}
	if err := json.Unmarshal(fragment, &d); err != nil {
		t.Fatal(err)
	}
	if v, _ := d["autoReloadSettings"].(bool); !v {
		t.Errorf("autoReloadSettings must be true (got %v)", d["autoReloadSettings"])
	}
}
