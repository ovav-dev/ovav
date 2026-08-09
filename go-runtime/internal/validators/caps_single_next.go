package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CapsSingleNext validates that caps.yaml has exactly one active next_phase.
// Multiple "next" pointers in the plan matrix cause governance confusion.
// Added v41.0 after caps authority dilution incident (multiple next pointers
// in historical task sections competed with canonical next_phase).
type CapsSingleNext struct{}

func NewCapsSingleNext() *CapsSingleNext { return &CapsSingleNext{} }

func (v *CapsSingleNext) ID() string   { return "caps_single_next" }
func (v *CapsSingleNext) Name() string { return "Caps Single Next Pointer" }
func (v *CapsSingleNext) Description() string {
	return "Ensures caps.yaml has exactly one active next_phase — no conflicting next pointers"
}
func (v *CapsSingleNext) Weight() int { return 25 }

func (v *CapsSingleNext) Validate(ctx context.Context, root string) Result {
	root = resolveRepoRoot(root)
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")

	data, err := os.ReadFile(capsPath)
	if err != nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Cannot read caps.yaml",
			Issues: []string{err.Error()},
		}
	}

	var caps map[string]interface{}
	if err := yaml.Unmarshal(data, &caps); err != nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Cannot parse caps.yaml",
			Issues: []string{err.Error()},
		}
	}

	// Check 1: next_phase must exist
	nextPhase, ok := caps["next_phase"]
	if !ok {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Missing required field: next_phase",
			Issues: []string{"caps.yaml must have a 'next_phase' field"},
		}
	}

	nextPhaseStr, ok := nextPhase.(string)
	if !ok || nextPhaseStr == "" {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Invalid next_phase",
			Issues: []string{"next_phase must be a non-empty string"},
		}
	}

	// Check 2: No conflicting "next" pointers anywhere in the document
	conflicts := findNextConflicts(string(data))
	if len(conflicts) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status:  "fail",
			Message: fmt.Sprintf("Found %d conflicting 'next' pointers in caps.yaml", len(conflicts)),
			Issues:  conflicts,
		}
	}

	return Result{
		ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
		Status:  "pass",
		Message: fmt.Sprintf("Single next_phase=%q — no conflicting pointers", nextPhaseStr),
	}
}

// findNextConflicts scans caps.yaml raw text for stale "next" fields
// that could compete with the canonical next_phase.
func findNextConflicts(raw string) []string {
	var conflicts []string
	lines := strings.Split(raw, "\n")
	inHeader := true

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// The canonical next_phase is in the header (first section)
		if inHeader && strings.HasPrefix(trimmed, "next_phase:") {
			inHeader = false
			continue
		}

		// Detect conflicting next pointers in other sections
		if strings.HasPrefix(trimmed, "next:") {
			val := strings.TrimPrefix(trimmed, "next:")
			val = strings.TrimSpace(val)
			if val != "" && val != "null" {
				conflicts = append(conflicts, fmt.Sprintf("'next: %s'", val))
			}
		}
		if strings.HasPrefix(trimmed, "next_task:") {
			conflicts = append(conflicts, "'next_task:' field present")
		}
		if strings.HasPrefix(trimmed, "next_phase:") && !inHeader {
			conflicts = append(conflicts, "'next_phase:' duplicated outside header")
		}
		if strings.HasPrefix(trimmed, "next_task_suggestion:") {
			conflicts = append(conflicts, "'next_task_suggestion:' competes with canonical next_phase")
		}
	}

	return conflicts
}
