package governor

import (
	"fmt"
	"strings"
	"time"
)

// ── Trust Gate Types ─────────────────────────────────────────────────────

// TrustAction represents the action determined by trust evaluation.
type TrustAction string

const (
	TrustDeliver    TrustAction = "DELIVER"
	TrustDisclaimer TrustAction = "DISCLAIMER"
	TrustBlock      TrustAction = "BLOCK"
	TrustReject     TrustAction = "REJECT"
)

// Trust thresholds calibrated from experiment E0 (F1=0.952).
const (
	TrustThresholdDeliver    = 0.90
	TrustThresholdDisclaimer = 0.75
	TrustThresholdBlock      = 0.50
)

// GateResult represents the result of a trust gate evaluation.
type GateResult struct {
	Passed             bool        `json:"passed"`
	TrustScore         float64     `json:"trust_score"`
	ThresholdApplied   float64     `json:"threshold_applied"`
	Action             TrustAction `json:"action"`
	ClaimsTotal        int         `json:"claims_total"`
	ClaimsVerified     int         `json:"claims_verified"`
	ClaimsContradicted int         `json:"claims_contradicted"`
	ClaimsUnknown      int         `json:"claims_unknown"`
	Contradictions     []string    `json:"contradictions,omitempty"`
	Summary            string      `json:"summary"`
	Lead               string      `json:"lead"`
	Timestamp          time.Time   `json:"timestamp"`
}

// ── Trust Gate Evaluation ────────────────────────────────────────────────

// EvaluateTrust evaluates an output against trust thresholds.
//
// Claims are statements made by a lead that need verification.
// Verified: confirmed against OVAV knowledge base.
// Contradicted: conflicts with OVAV knowledge.
// Unknown: cannot be verified.
//
// Replaces tools/governor/trust_gate.py (305 LOC Python).
func EvaluateTrust(lead string, claimsTotal, claimsVerified, claimsContradicted int, contradictions []string) GateResult {
	claimsUnknown := claimsTotal - claimsVerified - claimsContradicted
	if claimsUnknown < 0 {
		claimsUnknown = 0
	}

	// Trust score: ratio of verified claims to total, penalized by contradictions.
	// Base = verified / total. Each contradiction reduces the score by 0.15.
	var score float64
	if claimsTotal > 0 {
		base := float64(claimsVerified) / float64(claimsTotal)
		penalty := float64(claimsContradicted) * 0.15
		score = base - penalty
		if score < 0 {
			score = 0
		}
	} else {
		score = 1.0 // no claims → no trust issues
	}

	// Determine action and threshold
	var action TrustAction
	var threshold float64
	var passed bool

	switch {
	case score >= TrustThresholdDeliver:
		action = TrustDeliver
		threshold = TrustThresholdDeliver
		passed = true
	case score >= TrustThresholdDisclaimer:
		action = TrustDisclaimer
		threshold = TrustThresholdDisclaimer
		passed = true
	case score >= TrustThresholdBlock:
		action = TrustBlock
		threshold = TrustThresholdBlock
		passed = false
	default:
		action = TrustReject
		threshold = TrustThresholdBlock
		passed = false
	}

	return GateResult{
		Passed:             passed,
		TrustScore:         score,
		ThresholdApplied:   threshold,
		Action:             action,
		ClaimsTotal:        claimsTotal,
		ClaimsVerified:     claimsVerified,
		ClaimsContradicted: claimsContradicted,
		ClaimsUnknown:      claimsUnknown,
		Contradictions:     contradictions,
		Summary:            buildTrustSummary(action, score, lead),
		Lead:               lead,
		Timestamp:          time.Now().UTC(),
	}
}

func buildTrustSummary(action TrustAction, score float64, lead string) string {
	switch action {
	case TrustDeliver:
		return fmt.Sprintf("PASS — Trust score %.2f ≥ %.2f. Lead %s output approved for delivery.", score, TrustThresholdDeliver, lead)
	case TrustDisclaimer:
		return fmt.Sprintf("PASS with disclaimer — Trust score %.2f. Lead %s output delivered with caution notice.", score, lead)
	case TrustBlock:
		return fmt.Sprintf("BLOCKED — Trust score %.2f < %.2f. Lead %s must correct output.", score, TrustThresholdDisclaimer, lead)
	default:
		return fmt.Sprintf("REJECTED — Trust score %.2f < %.2f. Lead %s output rejected. Regenerate.", score, TrustThresholdBlock, lead)
	}
}

// ── Trust Helpers ────────────────────────────────────────────────────────

// ValidateClaims checks a list of claims against known facts.
// Returns the count of verified, contradicted, and unknown claims.
// This is a simplified deterministic version — the Python equivalent used
// ML-based model integrity checks.
func ValidateClaims(claims []string, knownFacts map[string]bool) (verified, contradicted, unknown int, contradictions []string) {
	for _, claim := range claims {
		claim = strings.TrimSpace(claim)
		if claim == "" {
			continue
		}
		if fact, ok := knownFacts[claim]; ok {
			if fact {
				verified++
			} else {
				contradicted++
				contradictions = append(contradictions, claim)
			}
		} else {
			unknown++
		}
	}
	return
}
