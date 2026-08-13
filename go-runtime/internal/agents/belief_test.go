package agents

import (
	"testing"
	"time"
)

func TestBeliefManagerDeprecatesOnlyStaleEmergentBeliefs(t *testing.T) {
	now := time.Now()
	bm := NewBeliefManager()
	bm.AddBelief("revocable-old", "keep")
	if err := bm.AddEmergentBeliefAt("emergent-old", "expire", now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := bm.AddEmergentBeliefAt("emergent-current", "keep", now.Add(-6*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	bm.DeprecateStaleEmergentAt(7*24*time.Hour, now)

	if !bm.IsDeprecated("emergent-old") {
		t.Fatal("expected stale emergent belief to be deprecated")
	}
	if bm.IsDeprecated("emergent-current") {
		t.Fatal("current emergent belief must remain active")
	}
	if bm.IsDeprecated("revocable-old") {
		t.Fatal("revocable beliefs must not be aged as emergent")
	}
}

func TestBeliefManagerRejectsUnknownState(t *testing.T) {
	bm := NewBeliefManager()
	if err := bm.AddBeliefWithState("unsafe", "value", BeliefState("unknown"), time.Now()); err == nil {
		t.Fatal("expected unknown belief state to fail closed")
	}
}

func TestAddBeliefPreservesBackwardsCompatibleAPI(t *testing.T) {
	bm := NewBeliefManager()
	bm.AddBelief("existing", 42)

	value, ok := bm.Belief("existing")
	if !ok || value != 42 {
		t.Fatalf("expected existing AddBelief API to store value, got %v, %t", value, ok)
	}
}
