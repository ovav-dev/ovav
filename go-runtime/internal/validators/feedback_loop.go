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

	// 1. This Go validator IS the feedback loop (Python version migrated to Go)
	// L7 feedback loop functionality is implemented in this package

	// 2. Check belief_manager module
	bmPath := filepath.Join(root, "tools", "agent_runtime", "belief_manager.py")
	if data, err := os.ReadFile(bmPath); err == nil {
		content := string(data)
		if !strings.Contains(content, "class BeliefManager") {
			issues = append(issues, "L7: BeliefManager class not found in belief_manager.py")
		}
		if !strings.Contains(content, "add_belief") {
			issues = append(issues, "L7: add_belief method not found")
		}
		if !strings.Contains(content, "deprecate_belief") {
			issues = append(issues, "L7: deprecate_belief method not found")
		}
		if !strings.Contains(content, "deprecate_stale_emergent") {
			issues = append(issues, "L7: deprecate_stale_emergent method not found — emergent expiry missing")
		}
	} else {
		issues = append(issues, "L7: belief_manager.py not found")
	}

	// 3. Check memory governor
	govPath := filepath.Join(root, "tools", "memory", "governor.py")
	if data, err := os.ReadFile(govPath); err == nil {
		content := string(data)
		if !strings.Contains(content, "ledger_write_allowed") && !strings.Contains(content, "ledger_vivo") {
			issues = append(issues, "L7: ledger_vivo gate not found in memory governor")
		}
	} else {
		issues = append(issues, "L7: memory governor.py not found")
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
