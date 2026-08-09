package governor

import (
	"testing"
)

func TestEvaluateTrust_Deliver(t *testing.T) {
	result := EvaluateTrust("thavren", 10, 10, 0, nil)

	if !result.Passed {
		t.Error("should pass")
	}
	if result.Action != TrustDeliver {
		t.Errorf("action = %s, want DELIVER", result.Action)
	}
	if result.TrustScore != 1.0 {
		t.Errorf("score = %.2f, want 1.0", result.TrustScore)
	}
	if result.ClaimsUnknown != 0 {
		t.Errorf("unknown = %d, want 0", result.ClaimsUnknown)
	}
}

func TestEvaluateTrust_Disclaimer(t *testing.T) {
	// 9 of 10 verified, 1 contradicted → base 0.9 - 0.15 = 0.75 → at DISCLAIMER threshold
	result := EvaluateTrust("thavren", 10, 9, 1, []string{"bad claim"})

	if !result.Passed {
		t.Error("should pass with disclaimer")
	}
	if result.Action != TrustDisclaimer {
		t.Errorf("action = %s, want DISCLAIMER (score=%.2f)", result.Action, result.TrustScore)
	}
}

func TestEvaluateTrust_Block(t *testing.T) {
	// 5 of 10 verified, 2 contradicted → base 0.5 - 0.30 = 0.20
	result := EvaluateTrust("thavren", 10, 5, 2, []string{"c1", "c2"})

	if result.Passed {
		t.Error("should be blocked")
	}
	if result.Action != TrustReject {
		t.Errorf("action = %s, want REJECT", result.Action)
	}
}

func TestEvaluateTrust_BorderlineDisclaimer(t *testing.T) {
	// 9 of 10 verified, 1 unknown → score 0.9 → DELIVER (at threshold)
	result := EvaluateTrust("thavren", 10, 9, 0, nil)

	if !result.Passed {
		t.Error("should pass at deliver threshold")
	}
	if result.Action != TrustDeliver {
		t.Errorf("action = %s, want DELIVER", result.Action)
	}
}

func TestEvaluateTrust_NoClaims(t *testing.T) {
	result := EvaluateTrust("thavren", 0, 0, 0, nil)

	if !result.Passed {
		t.Error("should pass with no claims")
	}
	if result.TrustScore != 1.0 {
		t.Errorf("score = %.2f, want 1.0", result.TrustScore)
	}
}

func TestEvaluateTrust_AllContradicted(t *testing.T) {
	// 0 of 5 verified, 5 contradicted → 0 - 0.75 = 0
	result := EvaluateTrust("thavren", 5, 0, 5, []string{"a", "b", "c", "d", "e"})

	if result.Passed {
		t.Error("should be rejected")
	}
	if result.TrustScore != 0 {
		t.Errorf("score = %.2f, want 0", result.TrustScore)
	}
	if result.Action != TrustReject {
		t.Errorf("action = %s, want REJECT", result.Action)
	}
}

func TestValidateClaims(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV runs on Go 1.25":        true,
		"caps.yaml is canonical":      true,
		"Python is the main language": false,
		"OWS supports 11 commands":    true,
	}

	claims := []string{
		"OVAV runs on Go 1.25",
		"caps.yaml is canonical",
		"Python is the main language", // false → contradicted
		"AI models are conscious",     // unknown
	}

	verified, contradicted, unknown, contradictions := ValidateClaims(claims, knownFacts)

	if verified != 2 {
		t.Errorf("verified = %d, want 2", verified)
	}
	if contradicted != 1 {
		t.Errorf("contradicted = %d, want 1", contradicted)
	}
	if unknown != 1 {
		t.Errorf("unknown = %d, want 1", unknown)
	}
	if len(contradictions) != 1 {
		t.Errorf("contradictions = %d, want 1", len(contradictions))
	}
	if contradictions[0] != "Python is the main language" {
		t.Errorf("contradiction = %q", contradictions[0])
	}
}

func TestValidateClaims_Empty(t *testing.T) {
	knownFacts := map[string]bool{"a": true}
	v, c, u, _ := ValidateClaims(nil, knownFacts)
	if v != 0 || c != 0 || u != 0 {
		t.Errorf("empty claims should return 0,0,0")
	}
}

func TestGateResult_Timestamp(t *testing.T) {
	result := EvaluateTrust("thavren", 1, 1, 0, nil)
	if result.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
	if result.Lead != "thavren" {
		t.Errorf("lead = %s, want thavren", result.Lead)
	}
}
