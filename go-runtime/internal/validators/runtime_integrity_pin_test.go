package validators

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPinIntegrityBaselineCopiesCurrentBaselineAtomically(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseline := IntegrityBaseline{
		Schema:    IntegrityBaselineSchema,
		Algorithm: "sha256",
		Files:     map[string]string{"opencode.json": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}
	data, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "baseline.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := PinIntegrityBaseline(root); err != nil {
		t.Fatal(err)
	}
	pinned, err := os.ReadFile(filepath.Join(dir, "baseline.pinned.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got IntegrityBaseline
	if err := json.Unmarshal(pinned, &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != baseline.Schema || got.Files["opencode.json"] != baseline.Files["opencode.json"] {
		t.Fatalf("pinned baseline = %+v, want %+v", got, baseline)
	}
}
