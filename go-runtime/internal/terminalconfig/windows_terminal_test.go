package terminalconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanWindowsTerminalPreservesUnrelatedSettings(t *testing.T) {
	current := []byte(`{
  "actions": [{"command":"copy","keys":"ctrl+c"}],
  "unrelated": {"keep": true},
  "profiles": {"defaults":{"font":{"size":11}},"list":[{"guid":"{11111111-1111-1111-1111-111111111111}","name":"Other"}]},
  "schemes": [{"name":"Other Scheme","background":"#000000"}]
}`)
	fragment := []byte(`{
	  "theme":{"light":"OVAV Day UI","dark":"OVAV Night UI"},
	  "actions":[{"command":"paste","keys":"ctrl+v"}],
	  "profiles":{"defaults":{"colorScheme":{"light":"OVAV Day","dark":"OVAV Night"}},"list":[{"guid":"{22222222-2222-2222-2222-222222222222}","name":"OVAV","commandline":"wsl.exe"}]},
  "schemes":[{"name":"OVAV Night","background":"#202124"},{"name":"OVAV Day","background":"#f7f4ee"}],
  "themes":[{"name":"OVAV Night UI"},{"name":"OVAV Day UI"}]
}`)

	plan, err := PlanWindowsTerminal(current, fragment, `C:\settings.json`, time.Date(2026, 8, 13, 12, 34, 56, 0, time.FixedZone("CEST", 2*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(plan.Merged) {
		t.Fatal("merged projection is invalid JSON")
	}
	if plan.Backup != `C:\settings.json.ovav-backup-20260813T103456Z` {
		t.Fatalf("unexpected backup plan: %s", plan.Backup)
	}

	text := string(plan.Merged)
	for _, preserved := range []string{`"actions"`, `"ctrl+c"`, `"unrelated"`, `"Other Scheme"`, `"Other"`, `"size": 11`} {
		if !strings.Contains(text, preserved) {
			t.Errorf("merged settings lost %s", preserved)
		}
	}
	for _, projected := range []string{`"OVAV Day UI"`, `"OVAV Night UI"`, `"OVAV Day"`, `"OVAV Night"`, `"ctrl+v"`} {
		if !strings.Contains(text, projected) {
			t.Errorf("merged settings missing %s", projected)
		}
	}
}

func TestWindowsTerminalFragmentUsesInstalledSchemaSurface(t *testing.T) {
	path := filepath.Join("..", "..", "..", ".ovav", "source", "configs", "windows-terminal", "ovav.fragment.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatal("fragment is invalid JSON")
	}
	text := string(data)
	for _, expected := range []string{
		`"light": "OVAV Day UI"`, `"dark": "OVAV Night UI"`,
		`"light": "OVAV Day"`, `"dark": "OVAV Night"`,
		`"useMica": true`, `Ubuntu-26.04`, `/home/braka/Systems/ovav`,
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("fragment missing %s", expected)
		}
	}
	for _, forbidden := range []string{`"actions"`, `Ubuntu-24.04`, `/home/braka/Systems/OVAV`} {
		if strings.Contains(text, forbidden) {
			t.Errorf("fragment contains forbidden replacement/stale setting %s", forbidden)
		}
	}
}

func TestRepositoryMergePlanAgainstFragment(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	fragment, err := os.ReadFile(filepath.Join(root, ".ovav", "source", "configs", "windows-terminal", "ovav.fragment.json"))
	if err != nil {
		t.Fatal(err)
	}
	current := []byte(`{
  "defaultProfile":"{11111111-1111-1111-1111-111111111111}",
  "actions":[{"command":"paste","keys":"ctrl+v"}],
	"profiles":{"defaults":{},"list":[{"guid":"{11111111-1111-1111-1111-111111111111}","name":"Existing"}]},
  "schemes":[],
  "themes":[]
}`)
	plan, err := PlanWindowsTerminal(current, fragment, `C:\settings.json`, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(plan.Merged) || !strings.Contains(string(plan.Merged), `"ctrl+v"`) {
		t.Fatal("repository projection is invalid or overwrote unrelated actions")
	}
}

func TestWriteWindowsTerminalPlannerSmokeFixture(t *testing.T) {
	current, err := os.ReadFile(filepath.Join("testdata", "windows_terminal_existing.json"))
	if err != nil {
		t.Fatal(err)
	}
	fragment, err := os.ReadFile(filepath.Join("..", "..", "..", ".ovav", "source", "configs", "windows-terminal", "ovav.fragment.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanWindowsTerminal(current, fragment, `C:\settings.json`, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestPlanWindowsTerminalRejectsInvalidJSON(t *testing.T) {
	if _, err := PlanWindowsTerminal([]byte(`{`), []byte(`{}`), "settings.json", time.Time{}); err == nil {
		t.Fatal("expected invalid installed settings to fail")
	}
	if _, err := PlanWindowsTerminal([]byte(`{}`), []byte(`{`), "settings.json", time.Time{}); err == nil {
		t.Fatal("expected invalid fragment to fail")
	}
}

func TestPlanWindowsTerminalRejectsInvalidWT124Structures(t *testing.T) {
	validCurrent := `{
  "profiles":{"defaults":{},"list":[{"guid":"{11111111-1111-1111-1111-111111111111}","name":"Existing"}]},
  "actions":[{"command":"copy","keys":"ctrl+c"}]
}`
	validFragment := `{
	  "theme":{"light":"OVAV Day UI","dark":"OVAV Night UI"},
  "profiles":{"defaults":{"colorScheme":{"light":"OVAV Day","dark":"OVAV Night"}},"list":[{"guid":"{22222222-2222-2222-2222-222222222222}","name":"OVAV","commandline":"wsl.exe"}]},
  "schemes":[{"name":"OVAV Day"},{"name":"OVAV Night"}],
  "themes":[{"name":"OVAV Day UI"},{"name":"OVAV Night UI"}]
}`
	tests := []struct {
		name     string
		current  string
		fragment string
		want     string
	}{
		{name: "profiles type", current: `{"profiles":[]}`, fragment: validFragment, want: "profiles must be an object"},
		{name: "scheme requires name", current: validCurrent, fragment: `{"profiles":{"list":[]},"schemes":[{"background":"#000000"}]}`, want: "schemes[0].name"},
		{name: "paired schemes", current: validCurrent, fragment: `{"profiles":{"defaults":{"colorScheme":{"light":"OVAV Day"}},"list":[]},"schemes":[{"name":"OVAV Day"}]}`, want: "colorScheme must pair light and dark"},
		{name: "paired themes", current: validCurrent, fragment: `{"theme":{"light":"OVAV Day UI"},"profiles":{"list":[]},"themes":[{"name":"OVAV Day UI"}]}`, want: "theme must pair light and dark"},
		{name: "theme reference", current: validCurrent, fragment: `{"theme":{"light":"Missing Day","dark":"OVAV Night UI"},"profiles":{"list":[]},"themes":[{"name":"OVAV Night UI"}]}`, want: "unknown theme"},
		{name: "profiles required", current: `{}`, fragment: validFragment, want: "profiles is required"},
		{name: "profile guid required", current: validCurrent, fragment: `{"profiles":{"list":[{"name":"OVAV","commandline":"wsl.exe"}]}}`, want: "guid is required"},
		{name: "profile guid", current: validCurrent, fragment: `{"profiles":{"list":[{"guid":"not-a-guid","name":"OVAV","commandline":"wsl.exe"}]}}`, want: "valid GUID"},
		{name: "default profile guid", current: `{"defaultProfile":"not-a-guid","profiles":{"list":[]}}`, fragment: validFragment, want: "defaultProfile must be a valid GUID"},
		{name: "profile command", current: validCurrent, fragment: `{"profiles":{"list":[{"guid":"{22222222-2222-2222-2222-222222222222}","name":"OVAV","commandline":42}]}}`, want: "commandline must be a string"},
		{name: "action command", current: `{"profiles":{"list":[]},"actions":[{"command":{"split":"vertical"}}]}`, fragment: `{"profiles":{"list":[]}}`, want: "command.action"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PlanWindowsTerminal([]byte(tt.current), []byte(tt.fragment), "settings.json", time.Time{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("PlanWindowsTerminal() error = %v, want %q", err, tt.want)
			}
		})
	}
}

// Test: Built-in Windows Terminal profiles (PowerShell, Command Prompt,
// Azure Cloud Shell) do NOT require commandline in the fragment. Windows
// Terminal supplies defaults at runtime. This mirrors the production
// fragment at workstation/configs/intelligent-terminal/settings-fragment.json
// which includes all three as named-only profiles.
func TestPlanWindowsTerminalBuiltinProfiles_NoCommandline_Pass(t *testing.T) {
	current := `{"profiles":{"list":[]}}`
	fragment := `{
		"profiles":{"defaults":{"colorScheme":{"light":"OVAV Day","dark":"OVAV Night"}},"list":[
			{"guid":"{22222222-2222-2222-2222-222222222222}","name":"OVAV","commandline":"wsl.exe"},
			{"guid":"{61c54bbd-c2c6-5271-96e7-009a87ff44bf}","name":"Windows PowerShell"},
			{"guid":"{0caa0dad-35be-5f56-a8ff-afceeeaa6101}","name":"Command Prompt"},
			{"guid":"{b453ae62-4e3d-5e58-b989-0a998ec441b8}","name":"Azure Cloud Shell","source":"Windows.Terminal.Azure"}
		]},
		"schemes":[{"name":"OVAV Day"},{"name":"OVAV Night"}]
	}`
	if _, err := PlanWindowsTerminal([]byte(current), []byte(fragment), "settings.json", time.Time{}); err != nil {
		t.Fatalf("expected built-in profiles without commandline to pass, got error: %v", err)
	}
}

// Test: Non-built-in profiles (OVAV-defined or third-party) still require
// commandline in the fragment. The strict-mode rule applies only to system
// profiles that ship with Windows Terminal by default.
func TestPlanWindowsTerminalNonBuiltin_NoCommandline_Fail(t *testing.T) {
	current := `{"profiles":{"list":[]}}`
	fragment := `{
		"profiles":{"defaults":{},"list":[
			{"guid":"{33333333-3333-3333-3333-333333333333}","name":"Custom"}
		]}
	}`
	_, err := PlanWindowsTerminal([]byte(current), []byte(fragment), "settings.json", time.Time{})
	if err == nil || !strings.Contains(err.Error(), "commandline is required") {
		t.Fatalf("expected commandline-required error for custom profile, got: %v", err)
	}
}

// Test: Profiles with a `source` field don't need commandline either —
// they're externally-sourced and Windows Terminal resolves them at runtime.
func TestPlanWindowsTerminalSourcedProfile_NoCommandline_Pass(t *testing.T) {
	current := `{"profiles":{"list":[]}}`
	fragment := `{
		"profiles":{"defaults":{},"list":[
			{"guid":"{44444444-4444-4444-4444-444444444444}","name":"External","source":"Windows.Terminal.Wsl"}
		]}
	}`
	if _, err := PlanWindowsTerminal([]byte(current), []byte(fragment), "settings.json", time.Time{}); err != nil {
		t.Fatalf("expected sourced profile without commandline to pass, got error: %v", err)
	}
}
