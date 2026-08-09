package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCaps_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	capsDir := filepath.Join(dir, ".ovav", "plan")
	os.MkdirAll(capsDir, 0755)

	yamlContent := `
version: "1.0"
updated_at: "2026-06-20"
updated_by: "thavren"
plan_version: "v1.9"
strategy: "Go+TS Native"
stack_target:
  go: "1.22"
  typescript: "5.0"
  python: "remove"
caps:
  CAP-001:
    name: "Governor Core"
    type: "SISTEMA"
    status: "done"
    pct: 100
    items: 5
    merge: "abc123"
    merged_at: "2026-06-15"
    summary: "Core governor system"
  CAP-002:
    name: "Context Ledger"
    type: "FEATURE"
    status: "done"
    pct: 100
    items: 3
    merge: "def456"
    merged_at: "2026-06-16"
    summary: "Ledger system"
pending:
  - id: "PEND-001"
    name: "Budget Engine"
    type: "FEATURE"
    status: "pending"
    pct: 30
    order: 1
    deps: ["CAP-001"]
    worktree: "task/budget"
    stack: "Go"
    tasks: ["Design", "Implement"]
    summary: "Budget tracking"
`
	os.WriteFile(filepath.Join(capsDir, "caps.yaml"), []byte(yamlContent), 0644)

	caps, err := LoadCaps(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caps == nil {
		t.Fatal("expected non-nil caps")
	}
	if caps.PlanVersion != "v1.9" {
		t.Errorf("expected plan version v1.9, got %q", caps.PlanVersion)
	}
	if caps.Strategy != "Go+TS Native" {
		t.Errorf("expected strategy Go+TS Native, got %q", caps.Strategy)
	}
	if caps.StackTarget.Go != "1.22" {
		t.Errorf("expected Go 1.22, got %q", caps.StackTarget.Go)
	}
	if len(caps.Caps) != 2 {
		t.Errorf("expected 2 caps, got %d", len(caps.Caps))
	}
	if cap, ok := caps.Caps["CAP-001"]; !ok {
		t.Error("missing CAP-001")
	} else {
		if cap.Name != "Governor Core" {
			t.Errorf("expected Governor Core, got %q", cap.Name)
		}
		if cap.Status != "done" {
			t.Errorf("expected done status, got %q", cap.Status)
		}
		if cap.Pct != 100 {
			t.Errorf("expected 100%%, got %d", cap.Pct)
		}
	}
	if len(caps.Pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(caps.Pending))
	}
	if caps.Pending[0].Name != "Budget Engine" {
		t.Errorf("expected Budget Engine, got %q", caps.Pending[0].Name)
	}
	if len(caps.Pending[0].Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(caps.Pending[0].Tasks))
	}
}

func TestLoadCaps_MissingFile(t *testing.T) {
	dir := t.TempDir()
	caps, err := LoadCaps(dir)
	if err == nil {
		t.Error("expected error for missing file")
	}
	if caps != nil {
		t.Error("expected nil caps for missing file")
	}
}

func TestLoadCaps_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	capsDir := filepath.Join(dir, ".ovav", "plan")
	os.MkdirAll(capsDir, 0755)
	os.WriteFile(filepath.Join(capsDir, "caps.yaml"), []byte("{{invalid yaml:::"), 0644)

	caps, err := LoadCaps(dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if caps != nil {
		t.Error("expected nil caps for invalid YAML")
	}
}

func TestLoadCaps_EmptyCaps(t *testing.T) {
	dir := t.TempDir()
	capsDir := filepath.Join(dir, ".ovav", "plan")
	os.MkdirAll(capsDir, 0755)
	os.WriteFile(filepath.Join(capsDir, "caps.yaml"), []byte("version: '1.0'\n"), 0644)

	caps, err := LoadCaps(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caps == nil {
		t.Fatal("expected non-nil caps")
	}
	if len(caps.Caps) != 0 {
		t.Errorf("expected 0 caps, got %d", len(caps.Caps))
	}
	if len(caps.Pending) != 0 {
		t.Errorf("expected 0 pending, got %d", len(caps.Pending))
	}
}

func TestCapsDataStruct(t *testing.T) {
	// Verify struct fields are accessible
	c := Cap{
		Name:     "test",
		Type:     "FEATURE",
		Status:   "done",
		Pct:      100,
		Items:    5,
		Merge:    "abc",
		MergedAt: "2026-06-15",
		Summary:  "summary",
	}
	if c.Name != "test" {
		t.Error("Cap Name field not accessible")
	}

	p := PendingCap{
		ID:       "P1",
		Name:     "pending",
		Type:     "FEATURE",
		Status:   "pending",
		Pct:      50,
		Order:    1,
		Deps:     []string{"dep1"},
		Worktree: "task/x",
		Commit:   "abc",
		Summary:  "sum",
		Stack:    "Go",
		Tasks:    []string{"t1", "t2"},
	}
	if p.ID != "P1" {
		t.Error("PendingCap ID field not accessible")
	}
	if len(p.Tasks) != 2 {
		t.Error("PendingCap Tasks field not accessible")
	}

	st := StackTarget{Go: "1.22", TypeScript: "5.0", Python: "remove"}
	if st.Go != "1.22" {
		t.Error("StackTarget Go field not accessible")
	}
}
