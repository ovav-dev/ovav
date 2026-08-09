package governor

import (
	"testing"
)

func TestIntegrityMeshHealth(t *testing.T) {
	tests := []struct {
		name       string
		pass       int
		fail       int
		total      int
		wantScore  float64
		wantStatus HealthStatus
	}{
		{"all_pass", 10, 0, 10, 100.0, StatusHealthy},
		{"one_fail_high_score", 9, 1, 10, 90.0, StatusDegraded},
		{"half_fail", 5, 5, 10, 50.0, StatusCritical},
		{"all_fail", 0, 10, 10, 0.0, StatusCritical},
		{"empty", 0, 0, 0, 0.0, StatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := IntegrityMeshHealth(tt.pass, tt.fail, tt.total)
			if r.Score != tt.wantScore {
				t.Errorf("score = %.1f, want %.1f", r.Score, tt.wantScore)
			}
			if r.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", r.Status, tt.wantStatus)
			}
		})
	}
}

func TestSelfDiagnosisHealth(t *testing.T) {
	tests := []struct {
		name       string
		ok         int
		warn       int
		crit       int
		wantStatus HealthStatus
	}{
		{"all_ok", 10, 0, 0, StatusHealthy},
		{"one_warning", 9, 1, 0, StatusDegraded},
		{"one_critical", 9, 0, 1, StatusCritical},
		{"empty", 0, 0, 0, StatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := SelfDiagnosisHealth(tt.ok, tt.warn, tt.crit)
			if r.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", r.Status, tt.wantStatus)
			}
		})
	}
}

func TestSelfDiagnosisHealth_Scoring(t *testing.T) {
	// 5 OK + 2 warnings + 3 critical = 10 total
	// Score = (5 + 2*0.5) / 10 * 100 = 60%
	r := SelfDiagnosisHealth(5, 2, 3)
	if r.Score != 60.0 {
		t.Errorf("score = %.1f, want 60.0", r.Score)
	}
}

func TestPainScorerHealth(t *testing.T) {
	tests := []struct {
		name               string
		avgPain            float64
		maxPain            float64
		totalEvents        int
		escalationDetected bool
		wantScore          float64
		wantStatus         HealthStatus
	}{
		{"no_events", 0, 0, 0, false, 100.0, StatusHealthy},
		{"low_pain", 10, 20, 5, false, 90.0, StatusHealthy},
		{"medium_pain", 55, 60, 10, false, 45.0, StatusDegraded},
		{"high_pain", 60, 85, 20, false, 40.0, StatusCritical}, // maxPain >= 80
		{"escalation", 30, 40, 5, true, 70.0, StatusCritical},
		{"extreme_pain", 100, 100, 5, false, 0.0, StatusCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := PainScorerHealth(tt.avgPain, tt.maxPain, tt.totalEvents, tt.escalationDetected)
			if r.Score != tt.wantScore {
				t.Errorf("score = %.1f, want %.1f", r.Score, tt.wantScore)
			}
			if r.Status != tt.wantStatus {
				t.Errorf("status = %s, want %s", r.Status, tt.wantStatus)
			}
		})
	}
}

func TestComputeUnifiedHealth_AllHealthy(t *testing.T) {
	reports := []SubsystemReport{
		IntegrityMeshHealth(10, 0, 10),
		SelfDiagnosisHealth(10, 0, 0),
		PainScorerHealth(10, 20, 10, false),
	}

	result := ComputeUnifiedHealth(reports...)

	if result.Overall != StatusHealthy {
		t.Errorf("overall = %s, want healthy", result.Overall)
	}
	// Weighted: 100*0.40 + 100*0.35 + 90*0.25 = 40+35+22.5 = 97.5
	if result.CompositeScore != 97.5 {
		t.Errorf("composite = %.1f, want 97.5", result.CompositeScore)
	}
	if result.RuleEnforced != "All subsystems healthy" {
		t.Errorf("rule = %q, want 'All subsystems healthy'", result.RuleEnforced)
	}
}

func TestComputeUnifiedHealth_IntegrityDegraded(t *testing.T) {
	reports := []SubsystemReport{
		IntegrityMeshHealth(8, 2, 10),       // degraded: 80%
		SelfDiagnosisHealth(10, 0, 0),       // healthy: 100%
		PainScorerHealth(10, 20, 10, false), // healthy
	}

	result := ComputeUnifiedHealth(reports...)

	// HARD RULE: integrity degraded → overall degraded
	if result.Overall != StatusDegraded {
		t.Errorf("overall = %s, want degraded", result.Overall)
	}
	if !contains(result.RuleEnforced, "Integrity Mesh") {
		t.Errorf("rule should mention Integrity Mesh, got %q", result.RuleEnforced)
	}
}

func TestComputeUnifiedHealth_SelfDiagnosisCritical(t *testing.T) {
	reports := []SubsystemReport{
		IntegrityMeshHealth(10, 0, 10), // healthy
		SelfDiagnosisHealth(8, 0, 2),   // 2 critical → critical
		PainScorerHealth(10, 20, 10, false),
	}

	result := ComputeUnifiedHealth(reports...)

	if result.Overall != StatusCritical {
		t.Errorf("overall = %s, want critical", result.Overall)
	}
}

func TestComputeUnifiedHealth_PainScorerDegraded(t *testing.T) {
	reports := []SubsystemReport{
		IntegrityMeshHealth(10, 0, 10),      // healthy
		SelfDiagnosisHealth(10, 0, 0),       // healthy
		PainScorerHealth(55, 60, 10, false), // degraded
	}

	result := ComputeUnifiedHealth(reports...)

	if result.Overall != StatusDegraded {
		t.Errorf("overall = %s, want degraded", result.Overall)
	}
	if !contains(result.RuleEnforced, "PainScorer") {
		t.Errorf("rule should mention PainScorer, got %q", result.RuleEnforced)
	}
}

func TestComputeUnifiedHealth_Empty(t *testing.T) {
	result := ComputeUnifiedHealth()

	if result.Overall != StatusUnknown {
		t.Errorf("overall = %s, want unknown", result.Overall)
	}
	if result.CompositeScore != 0 {
		t.Errorf("composite = %.1f, want 0", result.CompositeScore)
	}
}

func TestComputeUnifiedHealth_MissingSubsystem(t *testing.T) {
	// Only 2 of 3 subsystems report
	reports := []SubsystemReport{
		IntegrityMeshHealth(10, 0, 10),
		SelfDiagnosisHealth(10, 0, 0),
		// PainScorer missing
	}

	result := ComputeUnifiedHealth(reports...)

	if result.Overall != StatusHealthy {
		t.Errorf("overall = %s, want healthy (missing pain_scorer is OK)", result.Overall)
	}

	// Weighted: 100*0.40 + 100*0.35 = 40+35 = 75 / 0.75 = 100
	if result.CompositeScore != 100.0 {
		t.Errorf("composite = %.1f, want 100.0", result.CompositeScore)
	}
}

func TestSortSubsystems(t *testing.T) {
	result := UnifiedResult{
		Subsystems: map[string]SubsystemReport{
			"integrity_mesh": {},
			"self_diagnosis": {},
			"pain_scorer":    {},
		},
	}

	names := SortSubsystems(result)

	if len(names) != 3 {
		t.Fatalf("want 3 names, got %d", len(names))
	}
	expected := []string{"integrity_mesh", "pain_scorer", "self_diagnosis"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		val      float64
		decimals int
		want     float64
	}{
		{97.55, 1, 97.6},
		{97.54, 1, 97.5},
		{100.0, 1, 100.0},
		{0.0, 1, 0.0},
		{67.567, 2, 67.57},
	}
	for _, tt := range tests {
		got := roundTo(tt.val, tt.decimals)
		if got != tt.want {
			t.Errorf("roundTo(%.3f, %d) = %.2f, want %.2f", tt.val, tt.decimals, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
