package validators

import (
	"context"
	"testing"
)

func TestAdversarialVerification_ID(t *testing.T) {
	v := &AdversarialVerification{}
	if v.ID() != "adversarial_verification" {
		t.Errorf("expected adversarial_verification, got %s", v.ID())
	}
}

func TestAdversarialVerification_Name(t *testing.T) {
	v := &AdversarialVerification{}
	if v.Name() != "Adversarial Verification" {
		t.Errorf("expected Adversarial Verification, got %s", v.Name())
	}
}

func TestAdversarialVerification_Weight(t *testing.T) {
	v := &AdversarialVerification{}
	if v.Weight() != 5 {
		t.Errorf("expected weight 5, got %d", v.Weight())
	}
}

func TestAdversarialVerification_Validate(t *testing.T) {
	v := &AdversarialVerification{}
	result := v.Validate(context.Background(), ".")
	if result.ID != "adversarial_verification" {
		t.Errorf("expected validator adversarial_verification, got %s", result.ID)
	}
}

func TestAdversarialVerification_Judge(t *testing.T) {
	v := &AdversarialVerification{}

	// 3 passes = verified
	jurors := []Juror{
		{Name: "j1", Pass: true, Reason: "ok"},
		{Name: "j2", Pass: true, Reason: "ok"},
		{Name: "j3", Pass: true, Reason: "ok"},
	}
	if v.judge(jurors) != "verified" {
		t.Error("expected verified with 3 passes")
	}

	// 2 passes = verified
	jurors[2].Pass = false
	if v.judge(jurors) != "verified" {
		t.Error("expected verified with 2 passes")
	}

	// 1 pass = rejected
	jurors[1].Pass = false
	if v.judge(jurors) != "rejected" {
		t.Error("expected rejected with 1 pass")
	}

	// 0 passes = rejected
	jurors[0].Pass = false
	if v.judge(jurors) != "rejected" {
		t.Error("expected rejected with 0 passes")
	}
}

func TestAdversarialVerification_EvidenceJuror(t *testing.T) {
	v := &AdversarialVerification{}

	// No evidence = fail
	claim := Claim{Description: "test"}
	juror := v.jurorEvidence(claim)
	if juror.Pass {
		t.Error("expected fail when no evidence")
	}

	// With evidence = pass
	claim.Evidence = "some evidence"
	juror = v.jurorEvidence(claim)
	if !juror.Pass {
		t.Error("expected pass with evidence")
	}
}

func TestAdversarialVerification_FormatJurors(t *testing.T) {
	jurors := []Juror{
		{Name: "j1", Pass: true, Reason: "ok"},
		{Name: "j2", Pass: false, Reason: "fail"},
	}
	output := FormatJurors(jurors)
	if output == "" {
		t.Error("expected non-empty output")
	}
}
