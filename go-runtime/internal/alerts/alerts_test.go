package alerts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndResolve(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root)

	// Create an alert
	alert, err := m.Create(CatSecrets, SevCritical, "API key leaked", "Found plaintext API key", "config.yaml", 42)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if alert.ID == "" {
		t.Fatal("alert ID should not be empty")
	}
	if alert.Category != CatSecrets {
		t.Errorf("Category = %q, want %q", alert.Category, CatSecrets)
	}
	if alert.Severity != SevCritical {
		t.Errorf("Severity = %q, want %q", alert.Severity, SevCritical)
	}

	// Verify file was persisted
	alertPath := filepath.Join(root, ".ovav", "alerts", alert.ID+".yaml")
	if _, err := os.Stat(alertPath); err != nil {
		t.Fatalf("alert file not found at %s: %v", alertPath, err)
	}

	// Active should return 1 alert
	active, err := m.Active()
	if err != nil {
		t.Fatalf("Active() error: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active alert, got %d", len(active))
	}
	if active[0].Title != "API key leaked" {
		t.Errorf("Title = %q, want %q", active[0].Title, "API key leaked")
	}

	// Resolve it
	if err := m.Resolve(alert.ID); err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// File should be removed
	if _, err := os.Stat(alertPath); !os.IsNotExist(err) {
		t.Error("alert file should have been removed after Resolve()")
	}

	// Active should return 0
	active, err = m.Active()
	if err != nil {
		t.Fatalf("Active() after resolve error: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active alerts after resolve, got %d", len(active))
	}
}

func TestCount_BySeverity(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root)

	m.Create(CatSecrets, SevCritical, "crit1", "", "", 0)
	m.Create(CatSecrets, SevCritical, "crit2", "", "", 0)
	m.Create(CatBranch, SevHigh, "high1", "", "", 0)
	m.Create(CatConfig, SevLow, "low1", "", "", 0)

	counts, err := m.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if counts[SevCritical] != 2 {
		t.Errorf("CRITICAL count = %d, want 2", counts[SevCritical])
	}
	if counts[SevHigh] != 1 {
		t.Errorf("HIGH count = %d, want 1", counts[SevHigh])
	}
	if counts[SevLow] != 1 {
		t.Errorf("LOW count = %d, want 1", counts[SevLow])
	}
}

func TestHasBlocking(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root)

	// No alerts → not blocking
	blocking, err := m.HasBlocking()
	if err != nil {
		t.Fatalf("HasBlocking() error: %v", err)
	}
	if blocking {
		t.Error("expected not blocking with 0 alerts")
	}

	// LOW alert → not blocking
	m.Create(CatConfig, SevLow, "low", "", "", 0)
	blocking, _ = m.HasBlocking()
	if blocking {
		t.Error("expected not blocking with only LOW alerts")
	}

	// CRITICAL alert → blocking
	m.Create(CatSecrets, SevCritical, "crit", "", "", 0)
	blocking, _ = m.HasBlocking()
	if !blocking {
		t.Error("expected blocking with CRITICAL alert")
	}
}

func TestResolve_NonExistentIsNoop(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root)

	// Resolving a non-existent alert should not error
	err := m.Resolve("DOES_NOT_EXIST")
	if err != nil {
		t.Errorf("Resolve(non-existent) should not error, got: %v", err)
	}
}

func TestActive_EmptyDir(t *testing.T) {
	root := t.TempDir()
	m := NewManager(root)

	// No alerts dir at all
	active, err := m.Active()
	if err != nil {
		t.Fatalf("Active() error: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(active))
	}
}
