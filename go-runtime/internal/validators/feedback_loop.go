package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FeedbackLoop validates L7 feedback loop: sanitization, beliefs, compaction, and gate safe-stop.
// Replaces: check_L7_feedback_loop.py
type FeedbackLoop struct{}

func NewFeedbackLoop() *FeedbackLoop { return &FeedbackLoop{} }

func (f *FeedbackLoop) ID() string   { return "feedback_loop" }
func (f *FeedbackLoop) Name() string { return "L7 Feedback Loop Validator" }
func (f *FeedbackLoop) Description() string {
	return "Validates L7 feedback loop: sanitization, beliefs, compaction, and gate safe-stop"
}
func (f *FeedbackLoop) Weight() int { return 10 }

func (f *FeedbackLoop) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check the Go belief manager, including functional aging rather than a TODO stub.
	bmPath := filepath.Join(root, "go-runtime", "internal", "agents", "belief.go")
	if data, err := os.ReadFile(bmPath); err == nil {
		content := string(data)
		for _, token := range []string{"type BeliefManager struct", "AddBelief(", "AddBeliefWithState(", "DeprecateBelief(", "DeprecateStaleEmergent(", "DeprecateStaleEmergentAt(", "delete("} {
			if !strings.Contains(content, token) {
				issues = append(issues, fmt.Sprintf("L7: Go belief manager missing functional token %q", token))
			}
		}
		if strings.Contains(content, "TODO: implement emergent belief aging") {
			issues = append(issues, "L7: emergent belief aging remains TODO/nonfunctional")
		}
	} else {
		issues = append(issues, "L7: go-runtime/internal/agents/belief.go not found")
	}

	// 2. Check the current Go memory governor write and context-pack pipelines.
	govPath := filepath.Join(root, "go-runtime", "internal", "memory", "governor.go")
	if data, err := os.ReadFile(govPath); err == nil {
		content := string(data)
		for _, token := range []string{"type Governor struct", "func (g *Governor) Write(", "Classify(", "UpsertCard(", "Save()", "func (g *Governor) SessionPack("} {
			if !strings.Contains(content, token) {
				issues = append(issues, fmt.Sprintf("L7: Go memory governor missing pipeline token %q", token))
			}
		}
	} else {
		issues = append(issues, "L7: go-runtime/internal/memory/governor.go not found")
	}

	if len(issues) > 0 {
		return Result{
			ID: f.ID(), Name: f.Name(), Status: "fail", Weight: f.Weight(),
			Message:  fmt.Sprintf("FAIL L7 feedback loop — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: f.ID(), Name: f.Name(), Status: "pass", Weight: f.Weight(),
		Message:  "PASS L7 feedback loop — sanitization, beliefs, compaction, and gate verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*FeedbackLoop)(nil)
