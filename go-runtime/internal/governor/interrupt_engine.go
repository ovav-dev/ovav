// Interrupt Engine — Real-time trust oversight of lead outputs.
//
// Replaces tools/governor/interrupt_engine.py (445 LOC Python).
// Provides:
//   - OversightFunc: wraps a lead function with trust gate evaluation
//   - ReasoningGuard: verifies individual claims during reasoning
//   - InterruptEngine: singleton monitor for multiple leads
//
// Stdlib only. Thread-safe.

package governor

import (
	"fmt"
	"sync"
)

// ── OVAVInterrupt error ─────────────────────────────────────────────────────

// OVAVInterrupt is returned when OVAV blocks a lead's output.
// This is not a technical error — it is a governor order.
type OVAVInterrupt struct {
	GateResult GateResult
	Message    string
}

func (e *OVAVInterrupt) Error() string {
	return e.Message
}

// ── Oversight function ──────────────────────────────────────────────────────

// OversightConfig configures the trust oversight wrapper.
type OversightConfig struct {
	Lead        string
	MaxRetries  int
	AutoCorrect bool
	Feed        *SessionFeed // optional, nil = no event publishing
}

// OversightResult holds the result of an oversigned function call.
type OversightResult struct {
	Output     string
	GateResult GateResult
	Attempts   int
}

// OversightFunc evaluates a lead's output through the trust gate.
//
// Replaces the Python @ovav_oversight decorator.
// The generateFn produces the output. Claims are extracted and verified.
// If trust < threshold, retries up to MaxRetries before returning OVAVInterrupt.
func OversightFunc(cfg OversightConfig, generateFn func() (string, []string), knownFacts map[string]bool) (*OversightResult, error) {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.Lead == "" {
		cfg.Lead = "thavren"
	}

	for attempt := 0; attempt < cfg.MaxRetries; attempt++ {
		output, claims := generateFn()

		verified, contradicted, _, contradictions := ValidateClaims(claims, knownFacts)
		total := len(claims)
		result := EvaluateTrust(cfg.Lead, total, verified, contradicted, contradictions)

		if result.Passed {
			finalOutput := output
			if result.Action == TrustDisclaimer {
				finalOutput = fmt.Sprintf("[OVAV: trust=%.2f] %s", result.TrustScore, output)
			}

			if cfg.Feed != nil {
				cfg.Feed.PublishEvent(
					fmt.Sprintf("OVAV allows delivery from %s: trust=%.3f", cfg.Lead, result.TrustScore),
					"decision", "ovav", "info", "",
				)
			}

			return &OversightResult{
				Output:     finalOutput,
				GateResult: result,
				Attempts:   attempt + 1,
			}, nil
		}

		// Failed — publish interrupt event
		if cfg.Feed != nil && attempt < cfg.MaxRetries-1 {
			cfg.Feed.PublishEvent(
				fmt.Sprintf("OVAV INTERRUPTS %s: trust=%.3f, contradictions=%d. Attempt %d/%d",
					cfg.Lead, result.TrustScore, result.ClaimsContradicted, attempt+1, cfg.MaxRetries),
				"alert", "ovav", "warn", "",
			)
		}

		// Last attempt — total block
		if attempt == cfg.MaxRetries-1 {
			return nil, &OVAVInterrupt{
				GateResult: result,
				Message: fmt.Sprintf(
					"OVAV TOTAL BLOCK — %d failed attempts. Trust: %.3f. Contradictions: %d. Lead must investigate before retrying.",
					cfg.MaxRetries, result.TrustScore, result.ClaimsContradicted,
				),
			}
		}

		// Non-last attempt — return interrupt for caller to handle
		return nil, &OVAVInterrupt{
			GateResult: result,
			Message: fmt.Sprintf(
				"OVAV INTERRUPT — Trust Gate: %.3f. Contradictions: %v. Correct and retry (%d/%d).",
				result.TrustScore, contradictions, attempt+1, cfg.MaxRetries,
			),
		}
	}

	return nil, &OVAVInterrupt{
		Message: "OVAV: max retries exhausted",
	}
}

// ── Reasoning Guard ─────────────────────────────────────────────────────────

// ClaimCheck holds the result of verifying a single claim during reasoning.
type ClaimCheck struct {
	Verified       bool     `json:"verified"`
	Trust          float64  `json:"trust"`
	Status         string   `json:"status"`
	Contradictions []string `json:"contradictions,omitempty"`
}

// ReasoningGuard verifies individual claims during lead reasoning.
//
// Replaces Python ReasoningGuard class.
// Usage:
//
//	guard := NewReasoningGuard("thavren", knownFacts, nil)
//	check := guard.Check("OVAV uses Go for its core")
//	if !check.Verified { /* exclude this claim */ }
type ReasoningGuard struct {
	Lead           string
	KnownFacts     map[string]bool
	Feed           *SessionFeed
	VerifiedClaims []ClaimCheck
	RejectedClaims []ClaimCheck
}

// NewReasoningGuard creates a new reasoning guard.
func NewReasoningGuard(lead string, knownFacts map[string]bool, feed *SessionFeed) *ReasoningGuard {
	if lead == "" {
		lead = "thavren"
	}
	return &ReasoningGuard{
		Lead:       lead,
		KnownFacts: knownFacts,
		Feed:       feed,
	}
}

// Check verifies a single claim during reasoning.
// Replaces Python ReasoningGuard.check().
func (g *ReasoningGuard) Check(claimText string) ClaimCheck {
	verified, contradicted, _, contradictions := ValidateClaims([]string{claimText}, g.KnownFacts)
	total := 1
	result := EvaluateTrust(g.Lead, total, verified, contradicted, contradictions)

	check := ClaimCheck{
		Verified:       result.Passed,
		Trust:          result.TrustScore,
		Status:         string(result.Action),
		Contradictions: contradictions,
	}

	if result.Passed {
		g.VerifiedClaims = append(g.VerifiedClaims, check)
	} else {
		g.RejectedClaims = append(g.RejectedClaims, check)
	}

	return check
}

// VerifyOutput evaluates a complete output before delivery (final layer).
// Replaces Python ReasoningGuard.verify_output().
func (g *ReasoningGuard) VerifyOutput(output string, claims []string) GateResult {
	verified, contradicted, _, contradictions := ValidateClaims(claims, g.KnownFacts)
	total := len(claims)
	return EvaluateTrust(g.Lead, total, verified, contradicted, contradictions)
}

// Summary returns a summary of the guard session.
func (g *ReasoningGuard) Summary() string {
	total := len(g.VerifiedClaims) + len(g.RejectedClaims)
	if len(g.RejectedClaims) > 0 {
		return fmt.Sprintf("ReasoningGuard: %s rejected %d/%d claims during reasoning — detected BEFORE output",
			g.Lead, len(g.RejectedClaims), total)
	}
	return fmt.Sprintf("ReasoningGuard: %s verified %d claims — all clean", g.Lead, total)
}

// ── Interrupt Engine (singleton monitor) ────────────────────────────────────

// InterruptEngine monitors multiple leads and decides whether to intervene.
//
// Replaces Python InterruptEngine class.
// Thread-safe singleton.
type InterruptEngine struct {
	mu             sync.RWMutex
	knownFacts     map[string]bool
	activeSessions map[string]bool
	feed           *SessionFeed
}

var (
	interruptInstance *InterruptEngine
	interruptOnce     sync.Once
)

// NewInterruptEngine returns the singleton InterruptEngine.
func NewInterruptEngine(knownFacts map[string]bool, feed *SessionFeed) *InterruptEngine {
	interruptOnce.Do(func() {
		interruptInstance = &InterruptEngine{
			knownFacts:     knownFacts,
			activeSessions: make(map[string]bool),
			feed:           feed,
		}
	})
	return interruptInstance
}

// NewInterruptEngineForTest creates a non-singleton instance for testing.
func NewInterruptEngineForTest(knownFacts map[string]bool, feed *SessionFeed) *InterruptEngine {
	return &InterruptEngine{
		knownFacts:     knownFacts,
		activeSessions: make(map[string]bool),
		feed:           feed,
	}
}

// MonitorResult holds the result of monitoring a lead's output.
type MonitorResult struct {
	Interrupt bool       `json:"interrupt"`
	Message   string     `json:"message"`
	Result    GateResult `json:"result"`
}

// Monitor evaluates a lead's output and decides whether to intervene.
// Replaces Python InterruptEngine.monitor().
func (e *InterruptEngine) Monitor(lead, output string, claims []string) MonitorResult {
	e.mu.Lock()
	e.activeSessions[lead] = true
	e.mu.Unlock()

	verified, contradicted, _, contradictions := ValidateClaims(claims, e.knownFacts)
	total := len(claims)
	result := EvaluateTrust(lead, total, verified, contradicted, contradictions)

	if !result.Passed {
		return MonitorResult{
			Interrupt: true,
			Message: fmt.Sprintf(
				"OVAV GOVERNOR INTERRUPT\n  LEAD: %s\n  Trust: %.3f (threshold: %.2f)\n  Contradictions: %d\n  Action: %s\n  %s",
				lead, result.TrustScore, result.ThresholdApplied, result.ClaimsContradicted, result.Action, result.Summary,
			),
			Result: result,
		}
	}

	if result.Action == TrustDisclaimer {
		return MonitorResult{
			Interrupt: false,
			Message:   fmt.Sprintf("OVAV: output accepted with disclaimer (trust=%.3f)", result.TrustScore),
			Result:    result,
		}
	}

	return MonitorResult{
		Interrupt: false,
		Message:   fmt.Sprintf("OVAV: output verified (trust=%.3f)", result.TrustScore),
		Result:    result,
	}
}

// InterruptStatus holds the current state of the interrupt engine.
type InterruptStatus struct {
	ActiveSessions int     `json:"active_sessions"`
	GateThreshold  float64 `json:"gate_threshold"`
}

// GetStatus returns the current state of the interrupt engine.
// Replaces Python InterruptEngine.get_status().
func (e *InterruptEngine) GetStatus() InterruptStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return InterruptStatus{
		ActiveSessions: len(e.activeSessions),
		GateThreshold:  TrustThresholdDisclaimer, // 0.75
	}
}
