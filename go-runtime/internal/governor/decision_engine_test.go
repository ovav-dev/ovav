package governor

import (
	"testing"
)

func TestDecide_IntegrityBroken(t *testing.T) {
	state := SystemState{
		IntegrityNeedRepair: true,
		IntegrityFailing:    []string{"F0_runtime", "F2_security"},
		IntegrityScore:      65.0,
		IntegrityStatus:     "degraded",
	}

	decisions := Decide(state)

	if len(decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(decisions))
	}

	d := decisions[0]
	if d.Priority != PriorityCritical {
		t.Errorf("priority = %s, want CRITICAL", d.Priority)
	}
	if d.Action != ActionRepair {
		t.Errorf("action = %s, want REPAIR", d.Action)
	}
	if d.Lead != LeadThavren {
		t.Errorf("lead = %s, want thavren", d.Lead)
	}
	if !HasCriticalDecisions(decisions) {
		t.Error("HasCriticalDecisions should be true")
	}
}

func TestDecide_HealthDegraded(t *testing.T) {
	state := SystemState{
		IntegrityNeedRepair: false,
		HealthNeedAttention: true,
		HealthWarnings:      []string{"high_coverage_gap"},
		HealthScore:         55.0,
		HealthStatus:        "degraded",
	}

	decisions := Decide(state)

	if len(decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(decisions))
	}

	d := decisions[0]
	if d.Priority != PriorityHigh {
		t.Errorf("priority = %s, want HIGH", d.Priority)
	}
	if d.Action != ActionDiagnose {
		t.Errorf("action = %s, want DIAGNOSE", d.Action)
	}
}

func TestDecide_ContractDrift(t *testing.T) {
	state := SystemState{
		ContractDrift:       true,
		ContractStaleFields: []string{"work_state HEAD mismatch", "contract_age_30h"},
	}

	decisions := Decide(state)

	if len(decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(decisions))
	}

	d := decisions[0]
	if d.Priority != PriorityHigh {
		t.Errorf("priority = %s, want HIGH", d.Priority)
	}
	if d.Action != ActionSync {
		t.Errorf("action = %s, want SYNC", d.Action)
	}
}

func TestDecide_GitChanges(t *testing.T) {
	state := SystemState{
		GitChanges:    5,
		GitNeedCommit: true,
		GitBranch:     "feature/test",
	}

	decisions := Decide(state)

	if len(decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(decisions))
	}

	d := decisions[0]
	if d.Priority != PriorityMedium {
		t.Errorf("priority = %s, want MEDIUM", d.Priority)
	}
	if d.Action != ActionStabilize {
		t.Errorf("action = %s, want STABILIZE", d.Action)
	}
}

func TestDecide_GitChangesBelowThreshold(t *testing.T) {
	// ≤3 changes → no STABILIZE decision
	state := SystemState{
		GitChanges:    2,
		GitNeedCommit: true,
		GitBranch:     "feature/test",
	}

	decisions := Decide(state)

	if len(decisions) != 0 {
		t.Errorf("want 0 decisions for ≤3 changes, got %d", len(decisions))
	}
}

func TestDecide_MultipleDecisions(t *testing.T) {
	state := SystemState{
		IntegrityNeedRepair: true,
		IntegrityFailing:    []string{"F0"},
		IntegrityScore:      50.0,
		IntegrityStatus:     "degraded",
		ContractDrift:       true,
		ContractStaleFields: []string{"stale"},
		GitChanges:          10,
		GitNeedCommit:       true,
		GitBranch:           "feature/big",
	}

	decisions := Decide(state)

	// Should have 3 decisions: CRITICAL repair, HIGH sync, MEDIUM stabilize
	// Health is skipped because integrity is broken
	if len(decisions) != 3 {
		t.Fatalf("want 3 decisions, got %d", len(decisions))
	}

	// Verify ordering: CRITICAL first
	if decisions[0].Priority != PriorityCritical {
		t.Errorf("first decision should be CRITICAL, got %s", decisions[0].Priority)
	}
	if decisions[1].Priority != PriorityHigh {
		t.Errorf("second decision should be HIGH, got %s", decisions[1].Priority)
	}
	if decisions[2].Priority != PriorityMedium {
		t.Errorf("third decision should be MEDIUM, got %s", decisions[2].Priority)
	}
}

func TestDecide_AllHealthy(t *testing.T) {
	state := SystemState{} // all zero values → healthy

	decisions := Decide(state)

	if len(decisions) != 0 {
		t.Errorf("want 0 decisions for healthy state, got %d", len(decisions))
	}
}

func TestDecide_HealthSkippedWhenIntegrityBroken(t *testing.T) {
	// When integrity is broken, health decisions are skipped
	// (repair comes first)
	state := SystemState{
		IntegrityNeedRepair: true,
		IntegrityFailing:    []string{"F0"},
		IntegrityScore:      30.0,
		IntegrityStatus:     "failing",
		HealthNeedAttention: true,
		HealthWarnings:      []string{"warn"},
		HealthScore:         40.0,
		HealthStatus:        "degraded",
	}

	decisions := Decide(state)

	// Only integrity repair, no health diagnose
	for _, d := range decisions {
		if d.Action == ActionDiagnose {
			t.Error("should not emit DIAGNOSE when integrity is broken")
		}
	}
	if len(decisions) != 1 {
		t.Errorf("want 1 decision (repair only), got %d", len(decisions))
	}
}

func TestFilterByPriority(t *testing.T) {
	decisions := []Decision{
		{Priority: PriorityCritical},
		{Priority: PriorityHigh},
		{Priority: PriorityHigh},
		{Priority: PriorityMedium},
	}

	critical := FilterByPriority(decisions, PriorityCritical)
	if len(critical) != 1 {
		t.Errorf("want 1 critical, got %d", len(critical))
	}

	high := FilterByPriority(decisions, PriorityHigh)
	if len(high) != 2 {
		t.Errorf("want 2 high, got %d", len(high))
	}
}

func TestCountByPriority(t *testing.T) {
	decisions := []Decision{
		{Priority: PriorityCritical},
		{Priority: PriorityHigh},
		{Priority: PriorityHigh},
		{Priority: PriorityMedium},
		{Priority: PriorityMedium},
		{Priority: PriorityMedium},
	}

	counts := CountByPriority(decisions)

	if counts[PriorityCritical] != 1 {
		t.Errorf("critical count = %d, want 1", counts[PriorityCritical])
	}
	if counts[PriorityHigh] != 2 {
		t.Errorf("high count = %d, want 2", counts[PriorityHigh])
	}
	if counts[PriorityMedium] != 3 {
		t.Errorf("medium count = %d, want 3", counts[PriorityMedium])
	}
	if counts[PriorityLow] != 0 {
		t.Errorf("low count = %d, want 0", counts[PriorityLow])
	}
}

func TestDecision_Timestamp(t *testing.T) {
	state := SystemState{
		IntegrityNeedRepair: true,
		IntegrityFailing:    []string{"F0"},
		IntegrityStatus:     "degraded",
	}

	decisions := Decide(state)

	if len(decisions) != 1 {
		t.Fatal("expected 1 decision")
	}

	if decisions[0].Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}
