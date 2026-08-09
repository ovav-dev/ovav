package governor

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ── Decision Types ───────────────────────────────────────────────────────

// Priority represents the urgency level of a decision.
type Priority string

const (
	PriorityCritical Priority = "CRITICAL"
	PriorityHigh     Priority = "HIGH"
	PriorityMedium   Priority = "MEDIUM"
	PriorityLow      Priority = "LOW"
)

// priorityOrder defines the sort order for priorities (highest first).
var priorityOrder = map[Priority]int{
	PriorityCritical: 0,
	PriorityHigh:     1,
	PriorityMedium:   2,
	PriorityLow:      3,
}

// Action represents the type of action to take.
type Action string

const (
	ActionRepair    Action = "REPAIR"
	ActionDiagnose  Action = "DIAGNOSE"
	ActionSync      Action = "SYNC"
	ActionStabilize Action = "STABILIZE"
)

// Domain represents the governance domain.
type Domain string

const (
	DomainSystem Domain = "system"
)

// Lead represents the assigned operator for a decision.
type Lead string

const (
	LeadThavren Lead = "thavren"
	LeadOVAV    Lead = "ovav"
)

// Decision represents a single autonomous decision emitted by the engine.
type Decision struct {
	Priority        Priority  `json:"priority"`
	Action          Action    `json:"action"`
	Target          string    `json:"target"`
	Lead            Lead      `json:"lead"`
	Domain          Domain    `json:"domain"`
	Reason          string    `json:"reason"`
	ExpectedOutcome string    `json:"expected_outcome"`
	Timestamp       time.Time `json:"timestamp"`
}

// SystemState holds the current system assessment for decision-making.
type SystemState struct {
	IntegrityNeedRepair bool
	IntegrityFailing    []string
	IntegrityScore      float64
	IntegrityStatus     string

	HealthNeedAttention bool
	HealthWarnings      []string
	HealthScore         float64
	HealthStatus        string

	ContractDrift       bool
	ContractStaleFields []string

	GitChanges    int
	GitNeedCommit bool
	GitBranch     string
}

// ── Decision Engine ──────────────────────────────────────────────────────

// Decide analyzes system state and emits autonomous decisions.
//
// Rules (matching Python decision_engine.py):
//
//	CRITICAL — integrity broken → REPAIR
//	HIGH     — health degraded → DIAGNOSE
//	HIGH     — contract drift  → SYNC
//	MEDIUM   — >3 uncommitted  → STABILIZE
//
// Decisions are deterministic — no ML model, just rules + thresholds.
func Decide(state SystemState) []Decision {
	var decisions []Decision
	now := time.Now().UTC()

	// CRITICAL: Integrity broken
	if state.IntegrityNeedRepair {
		decisions = append(decisions, Decision{
			Priority:        PriorityCritical,
			Action:          ActionRepair,
			Target:          "integrity_mesh",
			Lead:            LeadThavren,
			Domain:          DomainSystem,
			Reason:          fmt.Sprintf("Integrity Mesh %s (%.0f%%). Failing: %s", state.IntegrityStatus, state.IntegrityScore, strings.Join(state.IntegrityFailing, ", ")),
			ExpectedOutcome: "Integrity Mesh 100% HEALTHY",
			Timestamp:       now,
		})
	}

	// HIGH: Health degraded (only if integrity is OK)
	if state.HealthNeedAttention && !state.IntegrityNeedRepair {
		decisions = append(decisions, Decision{
			Priority:        PriorityHigh,
			Action:          ActionDiagnose,
			Target:          "system_health",
			Lead:            LeadThavren,
			Domain:          DomainSystem,
			Reason:          fmt.Sprintf("Self-diagnosis %s (%.0f%%). Warnings: %s", state.HealthStatus, state.HealthScore, strings.Join(state.HealthWarnings, ", ")),
			ExpectedOutcome: "Self-diagnosis 100% HEALTHY",
			Timestamp:       now,
		})
	}

	// HIGH: Contract drift
	if state.ContractDrift {
		decisions = append(decisions, Decision{
			Priority:        PriorityHigh,
			Action:          ActionSync,
			Target:          "authority_contract",
			Lead:            LeadThavren,
			Domain:          DomainSystem,
			Reason:          fmt.Sprintf("Contract drift: %s", strings.Join(state.ContractStaleFields, ", ")),
			ExpectedOutcome: "Contract synced with reality",
			Timestamp:       now,
		})
	}

	// MEDIUM: Uncommitted changes (>3 significant files)
	if state.GitNeedCommit && state.GitChanges > 3 {
		decisions = append(decisions, Decision{
			Priority:        PriorityMedium,
			Action:          ActionStabilize,
			Target:          "working_tree",
			Lead:            LeadThavren,
			Domain:          DomainSystem,
			Reason:          fmt.Sprintf("%s: %d uncommitted changes", state.GitBranch, state.GitChanges),
			ExpectedOutcome: "Working tree clean, changes committed",
			Timestamp:       now,
		})
	}

	// Sort by priority (CRITICAL first)
	sort.Slice(decisions, func(i, j int) bool {
		return priorityOrder[decisions[i].Priority] < priorityOrder[decisions[j].Priority]
	})

	return decisions
}

// ── Helpers ──────────────────────────────────────────────────────────────

// HasCriticalDecisions returns true if any decision is CRITICAL.
func HasCriticalDecisions(decisions []Decision) bool {
	for _, d := range decisions {
		if d.Priority == PriorityCritical {
			return true
		}
	}
	return false
}

// FilterByPriority returns decisions matching a given priority.
func FilterByPriority(decisions []Decision, p Priority) []Decision {
	var filtered []Decision
	for _, d := range decisions {
		if d.Priority == p {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// CountByPriority returns a map of priority → count.
func CountByPriority(decisions []Decision) map[Priority]int {
	counts := make(map[Priority]int)
	for _, d := range decisions {
		counts[d.Priority]++
	}
	return counts
}
