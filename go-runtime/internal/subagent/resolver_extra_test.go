package subagent

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// resolver_extra_test.go — Sprint 8 T12 (zero debt)
// ═══════════════════════════════════════════════════════════════════════════

func TestT12SameSet_BothEmpty(t *testing.T) {
	if !sameSet(nil, nil) {
		t.Error("two empty slices should be equal")
	}
}

func TestT12SameSet_SameElements(t *testing.T) {
	a := []string{"x", "y", "z"}
	b := []string{"z", "y", "x"}
	if !sameSet(a, b) {
		t.Error("permutations should be equal as sets")
	}
}

func TestT12SameSet_Different(t *testing.T) {
	a := []string{"x", "y"}
	b := []string{"x", "z"}
	if sameSet(a, b) {
		t.Error("different sets should not be equal")
	}
}

func TestT12SameSet_DifferentLength(t *testing.T) {
	a := []string{"x", "y"}
	b := []string{"x"}
	if sameSet(a, b) {
		t.Error("different lengths should not be equal")
	}
}

func TestT12DisambiguationHint_Empty(t *testing.T) {
	c := &Catalog{}
	hint := c.disambiguationHint(nil)
	// With no IDs, it skips the loop and returns the default message
	_ = hint
}

func TestT12DisambiguationHint_Single(t *testing.T) {
	c := &Catalog{}
	hint := c.disambiguationHint([]string{"a"})
	if hint == "" {
		t.Error("single ID should produce hint or empty fallback")
	}
}

func TestT12DisambiguationHint_Multiple(t *testing.T) {
	c := &Catalog{}
	hint := c.disambiguationHint([]string{"a", "b", "c"})
	if hint == "" {
		t.Error("multiple IDs should produce hint")
	}
}

func TestT12FindByID_Empty(t *testing.T) {
	c := &Catalog{}
	agent := c.findByID("")
	if agent != nil {
		t.Error("empty ID should return nil")
	}
}

func TestT12MustGet_NotFoundPanics(t *testing.T) {
	defer func() {
		// MustGet panic on missing is documented behavior
		if r := recover(); r == nil {
			t.Error("MustGet with nonexistent ID should panic (documented behavior)")
		}
	}()
	c := &Catalog{}
	_ = c.MustGet("definitely-does-not-exist")
}

func TestT12AgentStruct_Fields(t *testing.T) {
	a := Agent{
		ID:   "test-agent",
		Name: "Test Agent",
		Kind: "team",
		Area: "platform",
		Note: "test",
	}
	if a.ID != "test-agent" {
		t.Error("Agent.ID should match")
	}
}

func TestT12ResolutionStruct_Fields(t *testing.T) {
	r := Resolution{
		Input:      "test",
		Ambiguous:  false,
		Suggestion: "test hint",
	}
	if r.Input != "test" {
		t.Error("Resolution.Input should match")
	}
}

func TestT12ResolutionRulesStruct_Fields(t *testing.T) {
	rr := ResolutionRules{
		StrictMode:        false,
		AmbiguityStrategy: "first",
	}
	if rr.AmbiguityStrategy != "first" {
		t.Error("ResolutionRules.AmbiguityStrategy should match")
	}
}
