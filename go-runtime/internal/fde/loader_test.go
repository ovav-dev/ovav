package fde

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Loader tests ────────────────────────────────────────────────────

func TestLoadBrainPack_NotFound(t *testing.T) {
	_, err := LoadBrainPack("/nonexistent", "platform_engineering", "thavren")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestLoadBrainPack_PartialLoad(t *testing.T) {
	// Create a temp dir with only some brain files
	dir := t.TempDir()
	areaDir := filepath.Join(dir, ".ovav", "service_areas", "platform_engineering", "thavren")
	if err := os.MkdirAll(areaDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write only SELF_MODEL
	selfModelContent := `self_model:
  version: "1.0"
  operator: thavren
  service_area: platform_engineering
`
	if err := os.WriteFile(filepath.Join(areaDir, "SELF_MODEL.yaml"), []byte(selfModelContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load — should succeed with nil for missing files
	pack, err := LoadBrainPack(dir, "platform_engineering", "thavren")
	if err != nil {
		t.Fatalf("LoadBrainPack failed: %v", err)
	}
	if pack == nil {
		t.Fatal("pack should not be nil")
	}
	if pack.SelfModel == nil {
		t.Error("SelfModel should be loaded")
	}
	if pack.Criteria != nil {
		t.Error("Criteria should be nil (file missing)")
	}
}

func TestCriteria_All(t *testing.T) {
	c := &Criteria{Entries: []Criterion{{ID: "1"}, {ID: "2"}}}
	if all := c.All(); len(all) != 2 {
		t.Errorf("Entries All = %d, want 2", len(all))
	}

	c2 := &Criteria{Entries: nil, Items: []Criterion{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if all := c2.All(); len(all) != 3 {
		t.Errorf("Items All = %d, want 3", len(all))
	}
}

func TestCriteria_Count(t *testing.T) {
	c := &Criteria{Entries: []Criterion{{ID: "1"}, {ID: "2"}, {ID: "3"}}}
	if c.Count() != 3 {
		t.Errorf("Count = %d, want 3", c.Count())
	}
}

func TestEvolution_AllSessions(t *testing.T) {
	e := &Evolution{Sessions: []EvolutionEntry{{Summary: "s1"}, {Summary: "s2"}}}
	if all := e.AllSessions(); len(all) != 2 {
		t.Errorf("Sessions All = %d, want 2", len(all))
	}

	e2 := &Evolution{Sessions: nil, History: []EvolutionEntry{{Summary: "h1"}}}
	if all := e2.AllSessions(); len(all) != 1 {
		t.Errorf("History All = %d, want 1", len(all))
	}
}

func TestOperatingLevel_Statement(t *testing.T) {
	ol := &OperatingLevel{
		Description: "default description",
		Law:         nil,
	}
	if s := ol.Statement(); s != "default description" {
		t.Errorf("Statement = %q, want default description", s)
	}

	ol2 := &OperatingLevel{
		Description: "default",
		Law:         map[string]interface{}{"statement": "law statement"},
	}
	if s := ol2.Statement(); s != "law statement" {
		t.Errorf("Statement from Law = %q, want law statement", s)
	}
}

func TestFindFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(file, []byte("test: true"), 0644); err != nil {
		t.Fatal(err)
	}

	result := findFile(dir, []string{"missing.yaml", "test.yaml"})
	if result != file {
		t.Errorf("findFile = %q, want %q", result, file)
	}

	result2 := findFile(dir, []string{"missing.yaml", "alsomissing.yaml"})
	if result2 != "" {
		t.Errorf("findFile for missing = %q, want empty", result2)
	}
}
