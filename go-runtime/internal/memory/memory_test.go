package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadLedger_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ledger, err := LoadLedger(dir)
	if err != nil {
		t.Fatalf("LoadLedger on empty dir: %v", err)
	}
	if ledger == nil {
		t.Fatal("expected non-nil ledger")
	}
	if ledger.Version != 3 {
		t.Errorf("expected version 3, got %d", ledger.Version)
	}
	if len(ledger.Cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(ledger.Cards))
	}
}

func TestLoadLedger_Existing(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	if err := os.MkdirAll(registry, 0755); err != nil {
		t.Fatal(err)
	}

	content := `active_context_ledger:
  version: 3
  purpose: test
  cards:
    - id: test-card-1
      status: active
      summary: A test card
      operational_rule: Test rule
      last_confirmed: "2026-06-30"
  last_updated: "2026-06-30T00:00:00Z"
`
	ledgerPath := filepath.Join(registry, "active_context_ledger.yaml")
	if err := os.WriteFile(ledgerPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ledger, err := LoadLedger(dir)
	if err != nil {
		t.Fatalf("LoadLedger: %v", err)
	}
	if len(ledger.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(ledger.Cards))
	}
	if ledger.Cards[0].ID != "test-card-1" {
		t.Errorf("expected test-card-1, got %s", ledger.Cards[0].ID)
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	if err := os.MkdirAll(registry, 0755); err != nil {
		t.Fatal(err)
	}

	// Load empty → add card → save → reload → verify
	ledger, err := LoadLedger(dir)
	if err != nil {
		t.Fatal(err)
	}

	ledger.UpsertCard(Card{
		ID:              "saved-card",
		Status:          StatusActive,
		Summary:         "Persisted card",
		OperationalRule: "Must survive round-trip",
		Tags:            []string{"test"},
		LastConfirmed:   "2026-06-30",
	})

	if err := ledger.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload
	reloaded, err := LoadLedger(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Cards) != 1 {
		t.Fatalf("expected 1 card after reload, got %d", len(reloaded.Cards))
	}
	if reloaded.Cards[0].ID != "saved-card" {
		t.Errorf("expected saved-card, got %s", reloaded.Cards[0].ID)
	}
}

func TestActiveCards_FiltersByStatus(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "a", Status: StatusActive},
			{ID: "b", Status: StatusCompleted},
			{ID: "c", Status: StatusDeprecated},
			{ID: "d", Status: StatusInProgress},
			{ID: "e", Status: StatusPending},
		},
	}

	active := ledger.ActiveCards()
	if len(active) != 2 {
		t.Fatalf("expected 2 active cards, got %d", len(active))
	}

	ids := make(map[string]bool)
	for _, c := range active {
		ids[c.ID] = true
	}
	if !ids["a"] || !ids["d"] {
		t.Errorf("expected cards a and d to be active, got %v", ids)
	}
}

func TestUpsertCard_New(t *testing.T) {
	ledger := &Ledger{}
	ledger.UpsertCard(Card{ID: "new", Status: StatusActive, Summary: "new"})
	if len(ledger.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(ledger.Cards))
	}
}

func TestUpsertCard_Update(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "x", Status: StatusActive, Summary: "old"},
		},
	}
	ledger.UpsertCard(Card{ID: "x", Status: StatusCompleted, Summary: "updated"})
	if len(ledger.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(ledger.Cards))
	}
	if ledger.Cards[0].Summary != "updated" {
		t.Errorf("expected 'updated', got %s", ledger.Cards[0].Summary)
	}
	if ledger.Cards[0].Status != StatusCompleted {
		t.Errorf("expected completed status, got %s", ledger.Cards[0].Status)
	}
}

func TestClassifier_BlocksSecret(t *testing.T) {
	c := NewClassifier(true)
	card := Card{
		ID:      "secret-card",
		Summary: "Contains API_KEY for production",
	}
	result := c.Classify(card)
	if result.Allow {
		t.Error("expected secret content to be blocked")
	}
}

func TestClassifier_AllowsPublic(t *testing.T) {
	c := NewClassifier(true)
	card := Card{
		ID:      "harmless-card",
		Summary: "Go tool catalog has 43 tools registered",
	}
	result := c.Classify(card)
	if !result.Allow {
		t.Error("expected harmless content to be allowed")
	}
}

func TestClassifier_ProductBlocksSensitive(t *testing.T) {
	c := NewClassifier(false) // Product mode — no sensitive
	card := Card{
		ID:      "internal-card",
		Summary: "Backend endpoint at api.ovav.dev internal config",
	}
	result := c.Classify(card)
	if result.Allow {
		t.Error("expected sensitive content to be blocked in Product mode")
	}
}

func TestRecall_ByTags(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "c1", Status: StatusActive, Tags: []string{"security", "context_cut"}, Summary: "s1", OperationalRule: "r1"},
			{ID: "c2", Status: StatusActive, Tags: []string{"ui", "context_cut"}, Summary: "s2", OperationalRule: "r2"},
			{ID: "c3", Status: StatusCompleted, Tags: []string{"security"}, Summary: "s3", OperationalRule: "r3"},
		},
	}

	r := NewRecall(ledger)
	results := r.ByTags([]string{"security"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result for security tag, got %d", len(results))
	}
	if results[0].Card.ID != "c1" {
		t.Errorf("expected c1, got %s", results[0].Card.ID)
	}
}

func TestRecall_ByQuery(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "vault", Status: StatusActive, Tags: []string{"security"}, Summary: "Vault Encryption 100% Go", OperationalRule: "Use ovav vault"},
			{ID: "cockpit", Status: StatusActive, Tags: []string{"ui"}, Summary: "Cockpit TUI in Go with Bubble Tea", OperationalRule: "Dev in go-runtime/"},
			{ID: "doc", Status: StatusActive, Tags: []string{"docs"}, Summary: "Architecture documentation", OperationalRule: "Update ARCHITECTURE.md"},
		},
	}

	r := NewRecall(ledger)
	results := r.ByQuery("vault encryption", 5)

	if len(results) < 1 {
		t.Fatal("expected at least 1 result for 'vault encryption'")
	}
	if results[0].Card.ID != "vault" {
		t.Errorf("expected vault card first, got %s", results[0].Card.ID)
	}
}

func TestRecall_CriticalCards(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "c1", Status: StatusActive, Priority: "CRITICAL", Summary: "s1", OperationalRule: "r1"},
			{ID: "c2", Status: StatusActive, Priority: "HIGH", Summary: "s2", OperationalRule: "r2"},
			{ID: "c3", Status: StatusActive, Priority: "CRITICAL", Summary: "s3", OperationalRule: "r3"},
		},
	}

	r := NewRecall(ledger)
	critical := r.CriticalCards()

	if len(critical) != 2 {
		t.Fatalf("expected 2 critical cards, got %d", len(critical))
	}
}

func TestGovernor_WritePipeline(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	if err := os.MkdirAll(registry, 0755); err != nil {
		t.Fatal(err)
	}

	g, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	card := Card{
		ID:              "gov-test",
		Status:          StatusActive,
		Summary:         "Go memory governor test card",
		OperationalRule: "Test operational rule",
		Tags:            []string{"test"},
		LastConfirmed:   time.Now().Format("2006-01-02"),
	}

	if err := g.Write(card); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Verify it was persisted
	reloaded, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	found := reloaded.Ledger().CardByID("gov-test")
	if found == nil {
		t.Fatal("card not found after reload")
	}
	if found.Summary != card.Summary {
		t.Errorf("summary mismatch: got %s", found.Summary)
	}
}

func TestGovernor_WriteRejectsSecret(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registry, 0755)

	g, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	card := Card{
		ID:              "secret",
		Status:          StatusActive,
		Summary:         "Contains API_KEY=sk-12345",
		OperationalRule: "Use this secret",
	}

	err = g.Write(card)
	if err == nil {
		t.Error("expected Write to reject secret content")
	}
}

func TestGovernor_WriteRejectsEmptyFields(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registry, 0755)

	g, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	// Missing summary
	err = g.Write(Card{ID: "bad", Status: StatusActive, OperationalRule: "r"})
	if err == nil {
		t.Error("expected Write to reject card without summary")
	}

	// Missing operational_rule
	err = g.Write(Card{ID: "bad2", Status: StatusActive, Summary: "s"})
	if err == nil {
		t.Error("expected Write to reject card without operational_rule")
	}
}

func TestGovernor_SessionPack(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registry, 0755)

	g, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	// Add critical card
	g.ledger.UpsertCard(Card{
		ID: "critical-1", Status: StatusActive, Priority: "CRITICAL",
		Summary: "Critical system rule", OperationalRule: "Must follow",
		LastConfirmed: time.Now().Format("2006-01-02"),
	})

	pack := g.SessionPack()
	if len(pack.Cards) < 1 {
		t.Error("expected at least 1 card in session pack")
	}
	if len(pack.OperationalRules) < 1 {
		t.Error("expected at least 1 operational rule")
	}
}
