// Package permissions provides deterministic permission policy simulation.
package permissions

import (
	"errors"
	"fmt"
	"strings"
)

// Governor identifies a canonical permission governor.
type Governor string

const (
	GovernorBash       Governor = "bash"
	GovernorSystemPath Governor = "system_path"
	GovernorPlugin     Governor = "plugin"
	GovernorNewState   Governor = "new_state"
)

// ErrPolicyContradiction indicates incompatible expectations for one policy input.
var ErrPolicyContradiction = errors.New("permission simulation: policy contradiction")

// SimulationCase describes one deterministic permission expectation.
type SimulationCase struct {
	Name          string
	Governor      Governor
	Input         string
	Operation     string
	Operator      string
	PluginType    string
	ExpectAllowed bool
}

// SimulationResult is the structured outcome for one simulation case.
type SimulationResult struct {
	Name            string   `json:"name"`
	Governor        Governor `json:"governor"`
	Input           string   `json:"input"`
	Allowed         bool     `json:"allowed"`
	ExpectedAllowed bool     `json:"expected_allowed"`
	Passed          bool     `json:"passed"`
	Contradiction   bool     `json:"contradiction,omitempty"`
	MatchedRule     string   `json:"matched_rule,omitempty"`
	Reason          string   `json:"reason"`
}

// DefaultSimulationCases returns the canonical smoke expectations.
func DefaultSimulationCases() []SimulationCase {
	return []SimulationCase{
		{Name: "bash allow", Governor: GovernorBash, Input: "git status", Operator: "andres", ExpectAllowed: true},
		{Name: "bash deny", Governor: GovernorBash, Input: "sudo id", Operator: "andres", ExpectAllowed: false},
		{Name: "system path allow", Governor: GovernorSystemPath, Input: "/etc/hosts", Operation: "read", ExpectAllowed: true},
		{Name: "system path deny", Governor: GovernorSystemPath, Input: "/proc/sys/kernel", Operation: "write", ExpectAllowed: false},
		{Name: "plugin allow", Governor: GovernorPlugin, Input: "local", Operator: "thavren", PluginType: "native", ExpectAllowed: true},
		{Name: "plugin deny", Governor: GovernorPlugin, Input: "external", Operator: "thavren", PluginType: "mcp_server", ExpectAllowed: false},
		{Name: "new state allow", Governor: GovernorNewState, Input: "revocable", ExpectAllowed: true},
		{Name: "new state deny", Governor: GovernorNewState, Input: "inherited", ExpectAllowed: false},
		{Name: "unknown defaults deny", Governor: GovernorNewState, Input: "unknown", ExpectAllowed: false},
	}
}

// SimulateCases evaluates cases in input order and returns every result. It
// reports an error when an expectation fails or two cases assert contradictory
// outcomes for the same normalized governor input.
func SimulateCases(cases []SimulationCase) ([]SimulationResult, error) {
	results := make([]SimulationResult, 0, len(cases))
	expectations := make(map[string]bool, len(cases))
	contradictions := make(map[string]bool)
	var failures []error

	for _, test := range cases {
		key := simulationKey(test)
		if expected, ok := expectations[key]; ok && expected != test.ExpectAllowed {
			contradictions[key] = true
		} else {
			expectations[key] = test.ExpectAllowed
		}
	}

	for i, test := range cases {
		key := simulationKey(test)
		result, err := simulateCase(test)
		result.Contradiction = contradictions[key]
		if err != nil {
			result.Passed = false
			failures = append(failures, fmt.Errorf("case %q: %w", caseName(test, i), err))
		} else if result.Contradiction {
			result.Passed = false
			failures = append(failures, fmt.Errorf("%w: case %q conflicts on %s", ErrPolicyContradiction, caseName(test, i), key))
		} else {
			result.Passed = result.Allowed == result.ExpectedAllowed
			if !result.Passed {
				failures = append(failures, fmt.Errorf("case %q: expected allowed=%t, got %t", caseName(test, i), result.ExpectedAllowed, result.Allowed))
			}
		}
		results = append(results, result)
	}

	return results, errors.Join(failures...)
}

// Simulate preserves the original API and runs the canonical smoke suite.
func Simulate() error {
	_, err := SimulateCases(DefaultSimulationCases())
	return err
}

func simulateCase(test SimulationCase) (SimulationResult, error) {
	result := SimulationResult{
		Name:            test.Name,
		Governor:        test.Governor,
		Input:           test.Input,
		ExpectedAllowed: test.ExpectAllowed,
	}

	switch test.Governor {
	case GovernorBash:
		decision := NewBashCommandGovernor().Check(test.Input, test.Operator)
		result.Allowed = decision.Allowed
		result.MatchedRule = decision.MatchedRule
		result.Reason = decision.Reason
	case GovernorSystemPath:
		decision := NewSystemPathGovernor("").Check(test.Input, test.Operation)
		result.Allowed = decision.Allowed
		result.MatchedRule = decision.RequiresGate
		result.Reason = decision.Reason
	case GovernorPlugin:
		decision := NewPluginGovernor().AuthorizePlugin(test.Operator, test.Input, test.PluginType)
		result.Allowed = decision.Allowed
		result.Reason = decision.Reason
	case GovernorNewState:
		decision := NewNewStatesGovernor().Check(test.Input)
		result.Allowed = decision.Allowed
		result.Reason = decision.Reason
	default:
		result.Reason = "unknown governor — default deny"
		return result, fmt.Errorf("unknown governor %q", test.Governor)
	}

	return result, nil
}

func simulationKey(test SimulationCase) string {
	return strings.Join([]string{
		string(test.Governor),
		strings.ToLower(strings.TrimSpace(test.Input)),
		strings.ToLower(strings.TrimSpace(test.Operation)),
		strings.ToLower(strings.TrimSpace(test.Operator)),
		strings.ToLower(strings.TrimSpace(test.PluginType)),
	}, "|")
}

func caseName(test SimulationCase, index int) string {
	if test.Name != "" {
		return test.Name
	}
	return fmt.Sprintf("case-%d", index)
}
