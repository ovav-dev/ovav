package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentMemory_NewAndStore(t *testing.T) {
	root := t.TempDir()

	am, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	card := Card{
		Topic:           "test topic",
		Summary:         "test summary",
		OperationalRule: "test rule",
		Priority:        "HIGH",
		Tags:            []string{"test", "unit"},
	}

	result, err := am.Store(card, StoreOptions{
		AgentID: "tester",
	})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	if result.Card.ID == "" {
		t.Error("Card ID should be non-empty after store")
	}
	if result.Card.ProposedBy != "tester" {
		t.Errorf("ProposedBy = %q, want tester", result.Card.ProposedBy)
	}
	// Commit may be empty in temp dir (no git repo); that's acceptable
	_ = result.Card.Commit
	if result.Card.DeprecatedAt == "" {
		t.Error("Evidence hash should be set")
	}
	if result.Card.Status != StatusActive {
		t.Errorf("Status = %q, want active", result.Card.Status)
	}
}

func TestAgentMemory_Recall(t *testing.T) {
	root := t.TempDir()

	am, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	// Store 3 cards
	for i := 0; i < 3; i++ {
		_, err := am.Store(Card{
			Topic:           "recall test",
			Summary:         "summary for recall test",
			OperationalRule: "rule for recall test",
			Priority:        "NORMAL",
			Tags:            []string{"recall", "test"},
		}, StoreOptions{AgentID: "tester"})
		if err != nil {
			t.Fatalf("Store card %d: %v", i, err)
		}
	}

	results := am.Recall(RecallOptions{
		Query: "recall test",
		Limit: 10,
	})

	if len(results.Cards) == 0 {
		t.Fatal("Recall returned 0 cards, want 3")
	}

	if results.Authenticity.Total != len(results.Cards) {
		t.Errorf("Authenticity.Total = %d, want %d", results.Authenticity.Total, len(results.Cards))
	}
}

func TestAgentMemory_Recent(t *testing.T) {
	root := t.TempDir()

	am, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	// Store 5 cards
	for i := 0; i < 5; i++ {
		_, err := am.Store(Card{
			Topic:           "recent test",
			Summary:         "summary",
			OperationalRule: "rule",
		}, StoreOptions{AgentID: "tester"})
		if err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	recent := am.Recent(3)
	if len(recent) != 3 {
		t.Errorf("Recent(3) = %d, want 3", len(recent))
	}

	recentAll := am.Recent(0)
	if len(recentAll) != 5 {
		t.Errorf("Recent(0) = %d, want 5", len(recentAll))
	}
}

func TestAgentMemory_Stats(t *testing.T) {
	root := t.TempDir()

	am, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	_, err = am.Store(Card{
		Topic:           "stats test 1",
		Summary:         "summary",
		OperationalRule: "rule",
		Tags:            []string{"test", "alpha"},
	}, StoreOptions{AgentID: "alice"})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	_, err = am.Store(Card{
		Topic:           "stats test 2",
		Summary:         "summary",
		OperationalRule: "rule",
		Tags:            []string{"test", "beta"},
	}, StoreOptions{AgentID: "bob"})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	total, active, byAgent, byTag := am.Stats()

	if total["total"] != 2 {
		t.Errorf("total[total] = %d, want 2", total["total"])
	}
	if active["active"] != 2 {
		t.Errorf("active[active] = %d, want 2", active["active"])
	}
	if byAgent["alice"] != 1 {
		t.Errorf("byAgent[alice] = %d, want 1", byAgent["alice"])
	}
	if byAgent["bob"] != 1 {
		t.Errorf("byAgent[bob] = %d, want 1", byAgent["bob"])
	}
	if byTag["test"] != 2 {
		t.Errorf("byTag[test] = %d, want 2", byTag["test"])
	}
}

func TestAgentMemory_Verify(t *testing.T) {
	root := t.TempDir()

	am, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	result, err := am.Store(Card{
		Topic:           "verify test",
		Summary:         "verify summary",
		OperationalRule: "verify rule",
	}, StoreOptions{AgentID: "tester"})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	report := am.Verify([]Card{result.Card})

	if report.Total != 1 {
		t.Errorf("report.Total = %d, want 1", report.Total)
	}
	if report.Verified == 0 {
		t.Error("report.Verified should be >= 1 for valid card")
	}
}

func TestAgentMemory_FilePersistence(t *testing.T) {
	root := t.TempDir()

	// Store a card
	am1, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	_, err = am1.Store(Card{
		Topic:           "persist test",
		Summary:         "persist summary",
		OperationalRule: "persist rule",
	}, StoreOptions{AgentID: "tester"})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Load in new instance
	am2, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory (reload): %v", err)
	}

	recent := am2.Recent(0)
	if len(recent) != 1 {
		t.Errorf("Recent after reload = %d, want 1", len(recent))
	}
	if recent[0].Topic != "persist test" {
		t.Errorf("Topic = %q, want persist test", recent[0].Topic)
	}
}

func TestAgentMemory_VerifyOnDisk(t *testing.T) {
	root := t.TempDir()

	// Directly write a memory file (simulating external persistence)
	memPath := filepath.Join(root, ".ovav", "runtime", "agent_memory.yaml")
	if err := os.MkdirAll(filepath.Dir(memPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	memYAML := `version: 1
purpose: test
last_updated: "2026-07-31T00:00:00Z"
cards:
  - id: test-card-1
    status: active
    priority: HIGH
    topic: on disk test
    tags: [test]
    summary: stored on disk
    operational_rule: rule from disk
    last_confirmed: "2026-07-31"
    commit: 45f8b36a0144046aedf6b468a01f4fc0e7e52292
    proposed_by: disk_writer
    deprecated_at: e3f7d06c0aa2e65487b6142fd213fb17bf4e77efe752181e4072f7d6fa7ae2aa
`
	if err := os.WriteFile(memPath, []byte(memYAML), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	am, err := NewAgentMemory(root, true)
	if err != nil {
		t.Fatalf("NewAgentMemory: %v", err)
	}

	cards := am.Recent(0)
	if len(cards) != 1 {
		t.Fatalf("Recent = %d, want 1", len(cards))
	}

	report := am.Verify(cards)
	if report.Verified == 0 {
		t.Error("card from disk should have verified evidence hash")
	}
}

func Test_sha256Hash(t *testing.T) {
	h1 := sha256Hash("hello")
	h2 := sha256Hash("hello")
	h3 := sha256Hash("world")

	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("SHA-256 hex = %d chars, want 64", len(h1))
	}
}

func Test_generateCardID(t *testing.T) {
	// generateCardID uses time.Now().UnixNano() so is non-deterministic
	// Verify it produces non-empty unique-looking IDs
	id := generateCardID("thavren", "some summary")
	if id == "" {
		t.Error("CardID should not be empty")
	}
	// ID format: 3-char prefix + "-" + 12-char hash
	if len(id) < 10 {
		t.Errorf("CardID %q too short, expected at least 10 chars", id)
	}
	// Different agents should produce different prefixes
	idEidren := generateCardID("eidren", "some summary")
	if idEidren == id {
		t.Error("different agents should produce different IDs")
	}
}
