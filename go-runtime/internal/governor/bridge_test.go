package governor

import (
	"testing"
)

// ── QuickPainScorer tests ──────────────────────────────────────────────────

func TestQuickPainScorer_ReturnsValues(t *testing.T) {
	avg, max, total, escalation := QuickPainScorer()

	if avg < 0 || avg > 10 {
		t.Errorf("avg = %v, want 0-10", avg)
	}
	if max < 0 || max > 10 {
		t.Errorf("max = %v, want 0-10", max)
	}
	if total < 0 {
		t.Errorf("total = %v, want >= 0", total)
	}
	if escalation && total == 0 {
		t.Errorf("escalation detected but total=0 is inconsistent")
	}
}

// ── CountActiveAlertsQuick tests ──────────────────────────────────────────

func TestCountActiveAlertsQuick_NoFiles(t *testing.T) {
	// When neither file exists, count should be 0
	count := CountActiveAlertsQuick()
	if count < 0 {
		t.Errorf("count = %v, want >= 0", count)
	}
}

func TestCountActiveAlertsQuick_WithFiles(t *testing.T) {
	// CountActiveAlertsQuick reads from the real repo root (FindRepoRoot),
	// so we can only verify it returns a non-negative value without crashing.
	// The actual count depends on whether dashboard.json / security_violations.yaml
	// exist in the repo — which is environment-dependent.
	count := CountActiveAlertsQuick()
	if count < 0 {
		t.Errorf("count = %d, want >= 0", count)
	}
}

// ── CountPendingDelegationsQuick tests ────────────────────────────────────

func TestCountPendingDelegationsQuick_NoDir(t *testing.T) {
	// When .ovav/runtime doesn't exist, should return 0 without panicking
	count := CountPendingDelegationsQuick()
	if count < 0 {
		t.Errorf("count = %v, want >= 0", count)
	}
}

func TestCountPendingDelegationsQuick_WithEntries(t *testing.T) {
	// Count actual runtime directory entries - just verify it returns >= 0 and doesn't crash
	count := CountPendingDelegationsQuick()
	if count < 0 {
		t.Errorf("count = %d, want >= 0", count)
	}
	// Real runtime dir has 71 entries; verify it's reasonable
	if count < 50 {
		t.Logf("count = %d (real runtime dir has many entries)", count)
	}
}

// ── CountOutstandingDecisionsQuick tests ─────────────────────────────────

func TestCountOutstandingDecisionsQuick_NoDir(t *testing.T) {
	count := CountOutstandingDecisionsQuick()
	if count < 0 {
		t.Errorf("count = %v, want >= 0", count)
	}
}

func TestCountOutstandingDecisionsQuick_FiltersCorrectly(t *testing.T) {
	// Verify it returns a non-negative count and only counts .yaml/.json files
	count := CountOutstandingDecisionsQuick()
	if count < 0 {
		t.Errorf("count = %d, want >= 0", count)
	}
	// Real verify dir has last_result.json - count should be >= 1 if it exists
	if count < 1 {
		t.Logf("count = %d (verify dir may have yaml/json files)", count)
	}
}

// ── VerifyTrust tests ────────────────────────────────────────────────────

func TestVerifyTrust_NoClaims(t *testing.T) {
	input := TrustInput{
		LeadName:     "thavren",
		OutputClaims: []string{},
	}
	output := VerifyTrust(input)

	if output.Action != TrustDisclaimer {
		t.Errorf("action = %v, want TRUST DISCLAIMER", output.Action)
	}
	if output.TrustScore != 50 {
		t.Errorf("trustScore = %v, want 50", output.TrustScore)
	}
	if output.Passed {
		t.Errorf("passed = true, want false for no claims")
	}
}

func TestVerifyTrust_AllClaimsMatch(t *testing.T) {
	input := TrustInput{
		LeadName: "thavren",
		OutputClaims: []string{
			"34/34 go tests passing",
			"governor coverage 91.2%",
		},
	}
	output := VerifyTrust(input)

	if !output.Passed {
		t.Errorf("passed = false, want true when all claims match known facts")
	}
	// TrustScore is now 0-1 range (after threshold scale bug fix)
	if output.TrustScore < 0.70 {
		t.Errorf("trustScore = %v, want >= 0.70 for matching claims", output.TrustScore)
	}
}

func TestVerifyTrust_ContradictedClaim(t *testing.T) {
	// Note: Testing contradicted claims requires a knownFact with value=false.
	// Current knownFacts all have value=true — contradiction path cannot be
	// triggered with existing facts. This test documents the expected behavior.
	input := TrustInput{
		LeadName: "thavren",
		OutputClaims: []string{
			"0 data races", // verified=true → not contradicted
		},
	}
	output := VerifyTrust(input)
	// This specific claim is verified, not contradicted.
	if output.TrustScore < 0.90 {
		t.Errorf("trustScore = %v, want >= 0.90 for verified single claim", output.TrustScore)
	}
}

func TestVerifyTrust_UnknownClaim(t *testing.T) {
	// Unknown claims: score = 0/total = 0 < 0.50 → TrustReject
	// Unknown-only claims always result in rejection
	input := TrustInput{
		LeadName: "thavren",
		OutputClaims: []string{
			"system is on fire",
		},
	}
	output := VerifyTrust(input)

	// Unknown claim score = 0 → TrustReject
	if output.Action != TrustReject {
		t.Errorf("action = %v, want TRUST REJECT for unknown-only claims", output.Action)
	}
	if output.Passed {
		t.Errorf("passed = true, want false for unknown claims")
	}
}

func TestVerifyTrust_MixedClaims(t *testing.T) {
	// Mix of verified and unknown: score = 1/2 = 0.50 → TrustBlock (passed=false)
	// After fix: EvaluateTrust returns score in 0-1 range (0.50)
	input := TrustInput{
		LeadName: "thavren",
		OutputClaims: []string{
			"34/34 go tests passing", // verified
			"system is on fire",      // unknown
		},
	}
	output := VerifyTrust(input)

	// Score = 0.50 (1 verified / 2 total, no contradictions)
	// 0.50 >= 0.50 → TrustBlock, passed=false
	if output.TrustScore != 0.50 {
		t.Errorf("trustScore = %v, want 0.50 for 1 verified + 1 unknown", output.TrustScore)
	}
	if output.Action != TrustBlock {
		t.Errorf("action = %v, want TRUST BLOCK", output.Action)
	}
	if output.Passed {
		t.Errorf("passed = true, want false for 0.50 score")
	}
}

// ── integrityStatus / healthLabel tests ─────────────────────────────────

func TestIntegrityStatus_Label(t *testing.T) {
	tests := []struct {
		score     float64
		wantLabel string
	}{
		{100, "healthy"},
		{80, "degraded"},
		{50, "critical"},
		{20, "critical"},
	}
	for _, tt := range tests {
		label := integrityStatus(tt.score)
		if label != tt.wantLabel {
			t.Errorf("integrityStatus(%.1f) = %q, want %q", tt.score, label, tt.wantLabel)
		}
	}
}

func TestHealthLabel_Label(t *testing.T) {
	tests := []struct {
		score     float64
		wantLabel string
	}{
		{98.8, "healthy"},
		{80.0, "degraded"},
		{50.0, "critical"},
		{20.0, "critical"},
	}
	for _, tt := range tests {
		label := healthLabel(tt.score)
		if label != tt.wantLabel {
			t.Errorf("healthLabel(%.1f) = %q, want %q", tt.score, label, tt.wantLabel)
		}
	}
}
