package governor

import (
	"errors"
	"testing"
)

// ── OversightFunc tests ─────────────────────────────────────────────────────

func TestOversightFunc_CleanOutput(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV is a governor": true,
		"Go is used":         true,
	}

	result, err := OversightFunc(
		OversightConfig{Lead: "thavren"},
		func() (string, []string) {
			return "OVAV is a governor written in Go", []string{"OVAV is a governor", "Go is used"}
		},
		knownFacts,
	)

	if err != nil {
		t.Fatalf("should not error on clean output: %v", err)
	}
	if result.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", result.Attempts)
	}
	if !result.GateResult.Passed {
		t.Error("gate result should pass")
	}
}

func TestOversightFunc_ContradictedOutput(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV was created by Microsoft": false, // contradicted
	}

	_, err := OversightFunc(
		OversightConfig{Lead: "thavren", MaxRetries: 1},
		func() (string, []string) {
			return "OVAV was created by Microsoft", []string{"OVAV was created by Microsoft"}
		},
		knownFacts,
	)

	if err == nil {
		t.Fatal("should error on contradicted output")
	}

	var interrupt *OVAVInterrupt
	if !errors.As(err, &interrupt) {
		t.Fatalf("error should be OVAVInterrupt, got %T", err)
	}
	if interrupt.GateResult.Passed {
		t.Error("gate result should not pass")
	}
}

func TestOversightFunc_DisclaimerPrefix(t *testing.T) {
	// Score between 0.75 and 0.90 → DISCLAIMER
	knownFacts := map[string]bool{
		"true claim":  true,
		"false claim": false,
	}

	// 3 claims: 2 verified, 1 contradicted → base 0.667 - 0.15 = 0.517 → BLOCK
	// Need: 4 claims: 3 verified, 0 contradicted, 1 unknown → 0.75 → DISCLAIMER
	knownFacts2 := map[string]bool{
		"a": true,
		"b": true,
		"c": true,
	}

	result, err := OversightFunc(
		OversightConfig{Lead: "thavren"},
		func() (string, []string) {
			return "output", []string{"a", "b", "c", "unknown_claim"}
		},
		knownFacts2,
	)

	// 3 verified, 0 contradicted, 1 unknown → score = 0.75 → DISCLAIMER
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if result.GateResult.Action != TrustDisclaimer {
		t.Errorf("action = %s, want DISCLAIMER (score=%.2f)", result.GateResult.Action, result.GateResult.TrustScore)
	}

	_ = knownFacts // suppress unused
}

func TestOversightFunc_WithFeed(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	knownFacts := map[string]bool{"true": true}

	_, err := OversightFunc(
		OversightConfig{Lead: "thavren", Feed: feed},
		func() (string, []string) {
			return "ok", []string{"true"}
		},
		knownFacts,
	)

	if err != nil {
		t.Fatalf("should not error: %v", err)
	}

	events := feed.ReadFeed(0, 50, nil)
	if len(events) < 1 {
		t.Error("expected at least 1 feed event from oversight")
	}
}

// ── ReasoningGuard tests ────────────────────────────────────────────────────

func TestReasoningGuard_VerifiedClaim(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV uses Go": true,
	}

	guard := NewReasoningGuard("thavren", knownFacts, nil)
	check := guard.Check("OVAV uses Go")

	if !check.Verified {
		t.Error("claim should be verified")
	}
	if len(guard.VerifiedClaims) != 1 {
		t.Errorf("verified claims = %d, want 1", len(guard.VerifiedClaims))
	}
}

func TestReasoningGuard_RejectedClaim(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV has 500 AWS servers": false,
	}

	guard := NewReasoningGuard("thavren", knownFacts, nil)
	check := guard.Check("OVAV has 500 AWS servers")

	if check.Verified {
		t.Error("claim should be rejected")
	}
	if len(guard.RejectedClaims) != 1 {
		t.Errorf("rejected claims = %d, want 1", len(guard.RejectedClaims))
	}
}

func TestReasoningGuard_MixedClaims(t *testing.T) {
	knownFacts := map[string]bool{
		"true fact":  true,
		"false fact": false,
	}

	guard := NewReasoningGuard("thavren", knownFacts, nil)

	guard.Check("true fact")
	guard.Check("false fact")
	guard.Check("true fact")

	if len(guard.VerifiedClaims) != 2 {
		t.Errorf("verified = %d, want 2", len(guard.VerifiedClaims))
	}
	if len(guard.RejectedClaims) != 1 {
		t.Errorf("rejected = %d, want 1", len(guard.RejectedClaims))
	}
}

func TestReasoningGuard_Summary(t *testing.T) {
	knownFacts := map[string]bool{"a": true}
	guard := NewReasoningGuard("thavren", knownFacts, nil)
	guard.Check("a")

	summary := guard.Summary()
	if summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestReasoningGuard_VerifyOutput(t *testing.T) {
	knownFacts := map[string]bool{
		"good": true,
		"bad":  false,
	}

	guard := NewReasoningGuard("thavren", knownFacts, nil)
	result := guard.VerifyOutput("output text", []string{"good", "good"})

	if !result.Passed {
		t.Error("output with all verified claims should pass")
	}
}

// ── InterruptEngine tests ───────────────────────────────────────────────────

func TestInterruptEngine_CleanOutput(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV is a governor created by Alexander Salvador.": true,
	}

	engine := NewInterruptEngineForTest(knownFacts, nil)
	result := engine.Monitor("thavren", "OVAV is a governor.", []string{"OVAV is a governor created by Alexander Salvador."})

	if result.Interrupt {
		t.Errorf("should not interrupt clean output: %s", result.Message)
	}
}

func TestInterruptEngine_ContradictedOutput(t *testing.T) {
	knownFacts := map[string]bool{
		"OVAV was created by Microsoft in 2019.": false,
	}

	engine := NewInterruptEngineForTest(knownFacts, nil)
	result := engine.Monitor("thavren", "OVAV was created by Microsoft.", []string{"OVAV was created by Microsoft in 2019."})

	if !result.Interrupt {
		t.Error("should interrupt contradicted output")
	}
	if result.Result.Passed {
		t.Error("gate result should not pass")
	}
}

func TestInterruptEngine_GetStatus(t *testing.T) {
	knownFacts := map[string]bool{"a": true}
	engine := NewInterruptEngineForTest(knownFacts, nil)

	status := engine.GetStatus()
	if status.GateThreshold != 0.75 {
		t.Errorf("threshold = %.2f, want 0.75", status.GateThreshold)
	}
}

func TestInterruptEngine_ActiveSessions(t *testing.T) {
	knownFacts := map[string]bool{"a": true}
	engine := NewInterruptEngineForTest(knownFacts, nil)

	engine.Monitor("thavren", "output", []string{"a"})
	engine.Monitor("eidren", "output", []string{"a"})

	status := engine.GetStatus()
	if status.ActiveSessions != 2 {
		t.Errorf("active_sessions = %d, want 2", status.ActiveSessions)
	}
}

func TestOVAVInterrupt_ErrorInterface(t *testing.T) {
	err := &OVAVInterrupt{
		Message: "test interrupt",
	}

	if err.Error() != "test interrupt" {
		t.Errorf("error message = %q, want 'test interrupt'", err.Error())
	}

	// Verify it satisfies error interface
	var e error = err
	if e == nil {
		t.Error("should satisfy error interface")
	}
}

func TestNewInterruptEngineForTest(t *testing.T) {
	knownFacts := map[string]bool{"test": true}
	feed := NewSessionFeed(t.TempDir())

	ie := NewInterruptEngineForTest(knownFacts, feed)
	if ie == nil {
		t.Fatal("NewInterruptEngineForTest returned nil")
	}
	if ie.knownFacts["test"] != true {
		t.Errorf("knownFacts[test] = %v, want true", ie.knownFacts["test"])
	}
	if ie.feed == nil {
		t.Error("feed should not be nil")
	}
}

func TestNewInterruptEngine_Singleton(t *testing.T) {
	knownFacts := map[string]bool{"singleton": true}
	feed := NewSessionFeed(t.TempDir())

	// NewInterruptEngine uses sync.Once — multiple calls return same instance
	ie1 := NewInterruptEngine(knownFacts, feed)
	ie2 := NewInterruptEngine(knownFacts, feed)

	if ie1 != ie2 {
		t.Error("NewInterruptEngine should return same instance (singleton)")
	}
	// Verify it initialized correctly
	if ie1.knownFacts["singleton"] != true {
		t.Errorf("knownFacts[singleton] = %v, want true", ie1.knownFacts["singleton"])
	}
}
