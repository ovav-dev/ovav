package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareITKeybindings_NoDrift(t *testing.T) {
	fragment := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"},
			{"keys": "ctrl+c", "id": "Terminal.CopyToClipboard"}
		]
	}`)
	live := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"},
			{"keys": "ctrl+c", "id": "Terminal.CopyToClipboard"}
		]
	}`)
	items, err := compareITKeybindings(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 drift items, got %d: %v", len(items), items)
	}
}

func TestCompareITKeybindings_MissingInLive(t *testing.T) {
	fragment := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"},
			{"keys": "ctrl+shift+p", "id": "Terminal.OpenNewTab"}
		]
	}`)
	live := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"}
		]
	}`)
	items, err := compareITKeybindings(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 drift item, got %d: %v", len(items), items)
	}
	if items[0].Type != DriftMissingInLive {
		t.Fatalf("expected missing_in_live, got %s", items[0].Type)
	}
	if items[0].Path != "keybindings[ctrl+shift+p]" {
		t.Fatalf("unexpected path: %s", items[0].Path)
	}
	if items[0].SuggestedFix == "" {
		t.Fatal("expected suggested fix")
	}
}

func TestCompareITKeybindings_ModifiedAction(t *testing.T) {
	fragment := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"}
		]
	}`)
	live := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.OtherAction"}
		]
	}`)
	items, err := compareITKeybindings(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 drift item, got %d", len(items))
	}
	if items[0].Type != DriftModified {
		t.Fatalf("expected modified, got %s", items[0].Type)
	}
}

func TestCompareITKeybindings_AddedInLive(t *testing.T) {
	fragment := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"}
		]
	}`)
	live := []byte(`{
		"keybindings": [
			{"keys": "ctrl+v", "id": "Terminal.PasteFromClipboard"},
			{"keys": "ctrl+x", "id": "Terminal.CustomAction"}
		]
	}`)
	items, err := compareITKeybindings(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 drift item (ctrl+x added in live), got %d", len(items))
	}
	if items[0].Type != DriftMissingInFragment {
		t.Fatalf("expected missing_in_fragment, got %s", items[0].Type)
	}
}

func TestCompareBashInputrc_NoDrift(t *testing.T) {
	fragment := []byte("# OVAV inputrc\n\"\\C-x\\C-e\": edit-and-execute-command\n")
	live := []byte("# live inputrc\n\"\\C-x\\C-e\": edit-and-execute-command\n")
	items, err := compareBashInputrc(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 drift items (same binding), got %d: %v", len(items), items)
	}
}

func TestCompareBashInputrc_MissingBinding(t *testing.T) {
	fragment := []byte("\"\\C-x\\C-e\": edit-and-execute-command\n\"\\C-x\\C-r\": re-read-init-file\n")
	live := []byte("\"\\C-x\\C-e\": edit-and-execute-command\n")
	items, err := compareBashInputrc(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 drift item, got %d: %v", len(items), items)
	}
	if items[0].Type != DriftMissingInLive {
		t.Fatalf("expected missing_in_live, got %s", items[0].Type)
	}
}

func TestDriftTarget_ResolveLivePath(t *testing.T) {
	t.Setenv("OVAV_LIVE_IT_SETTINGS", "/tmp/test-settings.json")
	target := DriftTarget{
		ID:      "test",
		LiveAbs: "/default/path",
		LiveEnv: "OVAV_LIVE_IT_SETTINGS",
	}
	got := target.resolveLivePath()
	if got != "/tmp/test-settings.json" {
		t.Fatalf("env override failed: got %q", got)
	}
}

func TestDriftTarget_ResolveLivePath_Tilde(t *testing.T) {
	target := DriftTarget{
		LiveAbs: "~/.inputrc",
	}
	got := target.resolveLivePath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".inputrc")
	if got != want {
		t.Fatalf("tilde expansion failed: got %q, want %q", got, want)
	}
}

func TestBuildDriftReport(t *testing.T) {
	// Use a temp dir with valid fragment + live files
	root := t.TempDir()
	fragDir := filepath.Join(root, "workstation", "configs", "intelligent-terminal")
	if err := os.MkdirAll(fragDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fragFile := filepath.Join(fragDir, "settings-fragment.json")
	if err := os.WriteFile(fragFile, []byte(`{"keybindings":[{"keys":"ctrl+v","id":"Terminal.PasteFromClipboard"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set live path via env to a known file
	liveFile := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(liveFile, []byte(`{"keybindings":[{"keys":"ctrl+v","id":"Terminal.PasteFromClipboard"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OVAV_LIVE_IT_SETTINGS", liveFile)

	report, err := buildDriftReport(root, "it-keybindings")
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalTargets != 1 {
		t.Fatalf("expected 1 target (filtered), got %d", report.TotalTargets)
	}
	if len(report.Targets) != 1 {
		t.Fatalf("expected 1 target report, got %d", len(report.Targets))
	}
	if report.DriftedTargets != 0 {
		t.Fatalf("expected 0 drift, got %d", report.DriftedTargets)
	}
}

func TestAppendCatalog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "drift.jsonl")
	entry := DriftCatalogEntry{
		Timestamp:      "2026-08-14T19:38:00Z",
		TotalTargets:   5,
		DriftedTargets: 2,
		TotalItems:     7,
	}
	appendCatalog(path, entry)
	appendCatalog(path, entry)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	var parsed DriftCatalogEntry
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.TotalItems != 7 {
		t.Fatalf("expected 7 items, got %d", parsed.TotalItems)
	}
}

func TestComparePinnedBaseline_Drift(t *testing.T) {
	fragment := []byte(`{"files":{"AGENTS.md":"OLDHASH","caps.yaml":"OLDCAPS"}}`)
	live := []byte(`{"files":{"AGENTS.md":"NEWHASH","caps.yaml":"NEWCAPS","new.md":"x"}}`)
	items, err := comparePinnedBaseline(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 drift items (AGENTS.md, caps.yaml mismatch), got %d: %v", len(items), items)
	}
	for _, item := range items {
		if item.Type != DriftModified {
			t.Fatalf("expected modified, got %s", item.Type)
		}
	}
}