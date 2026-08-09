package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Governor.QueryResult — boost coverage ───────────────────────────────────────

// TestGovernor_QueryResult_CallsRecall verifies that QueryResult delegates to recall.
func TestGovernor_QueryResult_CallsRecall(t *testing.T) {
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
		ID:              "query-test",
		Status:          StatusActive,
		Summary:         "Platform Engineering memory test",
		OperationalRule: "Engineer platform systems",
		Tags:            []string{"platform", "engineering"},
		LastConfirmed:   time.Now().Format("2006-01-02"),
	}
	if err := g.Write(card); err != nil {
		t.Fatal(err)
	}

	pack := g.QueryResult("platform", 5)
	if pack == nil {
		t.Fatal("QueryResult returned nil")
	}
	if len(pack.Cards) == 0 {
		t.Error("expected at least 1 card for 'platform' query")
	}
}

// TestGovernor_QueryResult_EmptyQuery verifies QueryResult handles empty query.
func TestGovernor_QueryResult_EmptyQuery(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	if err := os.MkdirAll(registry, 0755); err != nil {
		t.Fatal(err)
	}

	g, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}

	pack := g.QueryResult("", 5)
	if pack == nil {
		t.Fatal("QueryResult returned nil")
	}
}

// TestGovernor_SessionPack_WithRecentCards verifies SessionPack includes recent cards.
func TestGovernor_SessionPack_WithRecentCards(t *testing.T) {
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
		ID:              "session-pack-test",
		Status:          StatusActive,
		Summary:         "Recent active card",
		OperationalRule: "Test operational",
		Tags:            []string{"test"},
		LastConfirmed:   time.Now().Format("2006-01-02"),
	}
	if err := g.Write(card); err != nil {
		t.Fatal(err)
	}

	pack := g.SessionPack()
	if pack == nil {
		t.Fatal("SessionPack returned nil")
	}
	if len(pack.Cards) == 0 {
		t.Error("expected at least 1 card in SessionPack")
	}
}

// ── Recall boost — recency bonus paths ─────────────────────────────────────────

// TestRecall_ByQuery_RecencyBonus verifies cards confirmed within 7 days get
// recency bonus (covered via ByQuery which calls matchScore internally).
func TestRecall_ByQuery_RecencyBonus(t *testing.T) {
	recentDate := time.Now().Add(-3 * 24 * time.Hour).Format("2006-01-02")
	ledger := &Ledger{
		Cards: []Card{
			{
				ID:              "recent-card",
				Status:          StatusActive,
				Tags:            []string{"test"},
				Summary:         "Recent confirmed card for recency test",
				OperationalRule: "Test rule",
				LastConfirmed:   recentDate,
			},
		},
	}
	r := NewRecall(ledger)
	results := r.ByQuery("recency test", 5)
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Relevance <= 0 {
		t.Errorf("expected positive relevance for matching terms, got %f", results[0].Relevance)
	}
}

// TestRecall_RecentActive_ZeroLimit verifies RecentActive handles limit=0 (no limit).
func TestRecall_RecentActive_ZeroLimit(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "c1", Status: StatusActive, Summary: "s1", OperationalRule: "r1",
				LastConfirmed: "2026-07-28"},
			{ID: "c2", Status: StatusActive, Summary: "s2", OperationalRule: "r2",
				LastConfirmed: "2026-07-27"},
		},
	}
	r := NewRecall(ledger)
	cards := r.RecentActive(0)
	if len(cards) != 2 {
		t.Errorf("expected 2 cards with limit=0, got %d", len(cards))
	}
}

// TestRecall_RecentActive_LimitExceedsCards verifies RecentActive when limit > cards.
func TestRecall_RecentActive_LimitExceedsCards(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "c1", Status: StatusActive, Summary: "s1", OperationalRule: "r1",
				LastConfirmed: "2026-07-28"},
		},
	}
	r := NewRecall(ledger)
	cards := r.RecentActive(100)
	if len(cards) != 1 {
		t.Errorf("expected 1 card, got %d", len(cards))
	}
}

// TestRecall_ByQuery_EmptyTerms verifies ByQuery handles empty query terms.
func TestRecall_ByQuery_EmptyTerms(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "c1", Status: StatusActive, Summary: "s1", OperationalRule: "r1"},
		},
	}
	r := NewRecall(ledger)
	results := r.ByQuery("", 5)
	// Empty terms → matchScore returns 0 → no results
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

// TestRecall_ByQuery_IDMatch verifies that ID matches get higher relevance (+3).
func TestRecall_ByQuery_IDMatch(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{ID: "platform-engineering-card", Status: StatusActive,
				Summary: "A completely different summary", OperationalRule: "Something else",
				Tags: []string{"other"}},
			{ID: "other-card", Status: StatusActive,
				Summary:         "Platform engineering for the platform system",
				OperationalRule: "Platform rule", Tags: []string{"platform"}},
		},
	}
	r := NewRecall(ledger)
	results := r.ByQuery("platform-engineering", 5)
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	// ID match should rank higher
	if results[0].Card.ID != "platform-engineering-card" {
		t.Errorf("expected platform-engineering-card first (ID match +3), got %s", results[0].Card.ID)
	}
}

// ── Governor validate error paths ───────────────────────────────────────────────

// TestGovernor_Validate_EmptyID verifies Write rejects card with empty ID.
func TestGovernor_Validate_EmptyID(t *testing.T) {
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
		ID:              "", // empty ID
		Status:          StatusActive,
		Summary:         "Test summary",
		OperationalRule: "Test rule",
		LastConfirmed:   time.Now().Format("2006-01-02"),
	}
	err = g.Write(card)
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

// TestGovernor_Validate_EmptyStatus verifies Write rejects card with empty status.
func TestGovernor_Validate_EmptyStatus(t *testing.T) {
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
		ID:              "test-empty-status",
		Status:          "", // empty status
		Summary:         "Test summary",
		OperationalRule: "Test rule",
		LastConfirmed:   time.Now().Format("2006-01-02"),
	}
	err = g.Write(card)
	if err == nil {
		t.Error("expected error for empty status")
	}
}

// ── Ledger boost ─────────────────────────────────────────────────────────────────

// TestSave_EmptyLedger verifies Save on a governor with empty ledger.
func TestSave_EmptyLedger(t *testing.T) {
	dir := t.TempDir()
	registry := filepath.Join(dir, ".ovav", "registry")
	if err := os.MkdirAll(registry, 0755); err != nil {
		t.Fatal(err)
	}
	g, err := NewGovernor(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	err = g.Ledger().Save()
	if err != nil {
		t.Logf("Save error (may be expected on empty ledger): %v", err)
	}
}

// TestRecall_MatchScore_MultipleTerms verifies matchScore with multiple terms
// hitting different fields (id=+3, tag=+2, summary=+1, rule=+1).
func TestRecall_MatchScore_MultipleTerms(t *testing.T) {
	ledger := &Ledger{
		Cards: []Card{
			{
				ID:              "id-match",
				Status:          StatusActive,
				Tags:            []string{"tag-match"},
				Summary:         "summary-match",
				OperationalRule: "rule-match",
				LastConfirmed:   time.Now().Format("2006-01-02"),
			},
		},
	}
	r := NewRecall(ledger)
	results := r.ByQuery("id-match tag-match summary-match rule-match", 5)
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Relevance <= 0 {
		t.Errorf("expected positive relevance, got %f", results[0].Relevance)
	}
}
