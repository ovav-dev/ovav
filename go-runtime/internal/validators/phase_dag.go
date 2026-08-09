package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// PhaseDAG validates the phase_dag.yaml phase order and transitions.
// Replaces: validate_phase_dag.py
type PhaseDAG struct{}

func NewPhaseDAG() *PhaseDAG { return &PhaseDAG{} }

func (p *PhaseDAG) ID() string   { return "validate_phase_dag" }
func (p *PhaseDAG) Name() string { return "Phase DAG Validator" }
func (p *PhaseDAG) Description() string {
	return "Validates phase_dag.yaml phase order, transitions, and blocking rules"
}
func (p *PhaseDAG) Weight() int { return 7 }

var expectedPhases = []string{"init", "explore", "proposal", "spec", "design", "tasks", "apply", "verify", "archive"}

func (p *PhaseDAG) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	dagPath := filepath.Join(root, ".ovav", "registry", "phase_dag.yaml")
	data, err := os.ReadFile(dagPath)
	if err != nil {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "skip", Weight: p.Weight(),
			Message:  fmt.Sprintf("SKIP — phase_dag.yaml not found: %v", err),
			Duration: time.Since(start),
		}
	}

	var doc struct {
		PhaseDAG map[string]interface{} `yaml:"phase_dag"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "fail", Weight: p.Weight(),
			Message:  fmt.Sprintf("FAIL — YAML parse error: %v", err),
			Issues:   []string{fmt.Sprintf("YAML parse error: %v", err)},
			Duration: time.Since(start),
		}
	}

	pd := doc.PhaseDAG
	if pd == nil {
		issues = append(issues, "phase_dag.yaml must contain a phase_dag mapping")
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "fail", Weight: p.Weight(),
			Message:  "FAIL — missing phase_dag mapping",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	// Check phase names match expected — derive order from "next" chain, not map iteration
	phaseNames := make([]string, 0, len(expectedPhases))
	// Walk the chain from "init" following "next" to derive deterministic order
	visited := map[string]bool{}
	current := "init"
	for current != "" && !visited[current] {
		visited[current] = true
		phaseNames = append(phaseNames, current)
		if entry, ok := pd[current].(map[string]interface{}); ok {
			nextRaw := entry["next"]
			switch v := nextRaw.(type) {
			case []interface{}:
				if len(v) > 0 {
					if s, ok := v[0].(string); ok {
						current = s
					} else {
						current = ""
					}
				} else {
					current = ""
				}
			default:
				current = ""
			}
		} else {
			current = ""
		}
	}
	if !stringSlicesEqual(phaseNames, expectedPhases) {
		issues = append(issues, fmt.Sprintf("phase order must be %v; found %v", expectedPhases, phaseNames))
	}

	// Check each phase has correct next
	for i, phaseName := range expectedPhases {
		entry, ok := pd[phaseName].(map[string]interface{})
		if !ok {
			issues = append(issues, fmt.Sprintf("missing phase entry: %s", phaseName))
			continue
		}
		nextRaw := entry["next"]
		var nextList []string
		switch v := nextRaw.(type) {
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					nextList = append(nextList, s)
				}
			}
		}

		if i == len(expectedPhases)-1 { // "archive" — no next
			if len(nextList) != 0 {
				issues = append(issues, "archive must not have a next phase")
			}
		} else {
			expectedNext := expectedPhases[i+1]
			if len(nextList) != 1 || nextList[0] != expectedNext {
				issues = append(issues, fmt.Sprintf("%s must flow to %s", phaseName, expectedNext))
			}
		}
	}

	// Check blocking_rules
	if br, ok := pd["blocking_rules"].(map[string]interface{}); ok {
		applyReqs := toStringSlice(br["apply_requires"])
		expectedApply := []string{"proposal", "spec", "design", "tasks"}
		if !stringSlicesEqual(applyReqs, expectedApply) {
			issues = append(issues, "apply_requires must block missing proposal/spec/design/tasks")
		}
		verifyReqs := toStringSlice(br["verify_requires"])
		if !stringSlicesEqual(verifyReqs, []string{"apply_log"}) {
			issues = append(issues, "verify_requires must require apply_log")
		}
		archiveReqs := toStringSlice(br["archive_requires"])
		if !stringSlicesEqual(archiveReqs, []string{"verify_report"}) {
			issues = append(issues, "archive_requires must require verify_report")
		}
	} else {
		issues = append(issues, "blocking_rules must be a mapping")
	}

	if len(issues) > 0 {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "fail", Weight: p.Weight(),
			Message:  fmt.Sprintf("FAIL — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
		Message:  "PASS — phase_dag.yaml valid",
		Duration: time.Since(start),
	}
}

func toStringSlice(v interface{}) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

var _ Validator = (*PhaseDAG)(nil)
