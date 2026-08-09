package validators

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// CapsChronosAlignment validates that caps.yaml's updated_at timestamp
// is within tolerance of git HEAD. This enforces the temporal authority
// chain: chronos_gate > git HEAD > caps.yaml timestamps.
// Added v41.0 after caps authority dilution incident where current_state
// reported v37.0 while git HEAD was v40.0.
type CapsChronosAlignment struct{}

func NewCapsChronosAlignment() *CapsChronosAlignment { return &CapsChronosAlignment{} }

func (v *CapsChronosAlignment) ID() string   { return "caps_chronos_alignment" }
func (v *CapsChronosAlignment) Name() string { return "Caps-Chronos Temporal Alignment" }
func (v *CapsChronosAlignment) Description() string {
	return "Ensures caps.yaml updated_at is within 2h of git HEAD commit date"
}
func (v *CapsChronosAlignment) Weight() int { return 20 }

func (v *CapsChronosAlignment) Validate(ctx context.Context, root string) Result {
	root = resolveRepoRoot(root)
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")

	// Get git HEAD commit date
	headDate, err := getGitHeadDate(root)
	if err != nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Cannot determine git HEAD date",
			Issues: []string{err.Error()},
		}
	}

	// Parse caps.yaml updated_at
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

	updatedAt, ok := caps["updated_at"]
	if !ok {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Missing required field: updated_at",
			Issues: []string{"caps.yaml must have an 'updated_at' field with UTC-5 timestamp"},
		}
	}

	capsDate, err := parseCapsDate(fmt.Sprint(updatedAt))
	if err != nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Cannot parse updated_at",
			Issues: []string{err.Error()},
		}
	}

	// Check tolerance: caps must be within 2 hours of git HEAD
	drift := headDate.Sub(capsDate)
	if drift < 0 {
		drift = -drift
	}

	maxDrift := 2 * time.Hour
	if drift > maxDrift {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status:  "fail",
			Message: fmt.Sprintf("caps.yaml stale: updated_at is %v from git HEAD (max 2h)", drift.Round(time.Minute)),
			Issues:  []string{fmt.Sprintf("caps.updated_at=%s, git HEAD=%s", capsDate.Format(time.RFC3339), headDate.Format(time.RFC3339))},
		}
	}

	return Result{
		ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
		Status:  "pass",
		Message: fmt.Sprintf("caps.yaml within tolerance: drift=%v (max 2h)", drift.Round(time.Minute)),
	}
}

// getGitHeadDate returns the commit date of HEAD.
func getGitHeadDate(repoRoot string) (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%aI")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
}

// parseCapsDate parses the caps.yaml updated_at field which uses format:
// "2026-06-19 03:05 UTC-5" or similar variants.
func parseCapsDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	// Try "2026-06-19 03:05 UTC-5" — replace UTC-X with offset
	normalized := strings.Replace(s, "UTC-5", "-0500", 1)
	normalized = strings.Replace(normalized, "UTC-4", "-0400", 1)
	if t, err := time.Parse("2006-01-02 15:04 -0700", normalized); err == nil {
		return t, nil
	}
	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try with space-separated offset "2026-06-19 03:05 -0500"
	if t, err := time.Parse("2006-01-02 15:04 -0700", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}
