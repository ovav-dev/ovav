// Package project — GOV-007: Sync notifications and diff detection.
//
// Provides the bridge between the sync projection pipeline and
// the cPanel broadcast hub for real-time Cockpit notifications.

package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SyncResult summarizes the outcome of a sync operation.
type SyncResult struct {
	Status     string       `json:"status"` // "success", "partial", "failed"
	Steps      []StepResult `json:"steps"`
	TotalSteps int          `json:"total_steps"`
	Failed     int          `json:"failed"`
	Duration   string       `json:"duration"`
	Timestamp  string       `json:"timestamp"`
	Changed    bool         `json:"changed"`
	Changes    []string     `json:"changes,omitempty"`
}

// StepResult tracks a single sync step outcome.
type StepResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "failed"
	Details string `json:"details,omitempty"`
	Count   int    `json:"count"`
}

// SyncAndDetectChanges runs the full sync pipeline and detects what changed.
// Returns a structured result suitable for broadcast to Cockpit.
func SyncAndDetectChanges(root string, verbose bool) (*SyncResult, error) {
	// HARD GATE: prevent Systems/Product cross-contamination.
	if err := ValidateTarget(root); err != nil {
		return nil, err
	}

	start := time.Now()
	result := &SyncResult{
		Steps:     make([]StepResult, 0),
		Timestamp: start.UTC().Format(time.RFC3339),
	}

	// Capture state before sync (git diff)
	beforeChanges := detectCurrentChanges(root)

	// Run full sync
	if err := Sync(root, verbose); err != nil {
		result.Status = "failed"
		result.Failed++
		result.Duration = time.Since(start).String()
		return result, err
	}

	// Capture state after sync
	afterChanges := detectCurrentChanges(root)
	result.Changed = beforeChanges != afterChanges
	if result.Changed {
		result.Changes = diffChanges(beforeChanges, afterChanges)
	}

	// Count steps
	result.Steps = append(result.Steps, StepResult{
		Name: "agents", Status: "ok",
	})
	result.Steps = append(result.Steps, StepResult{
		Name: "skills", Status: "ok",
	})
	result.Steps = append(result.Steps, StepResult{
		Name: "visual", Status: "ok",
	})
	result.Steps = append(result.Steps, StepResult{
		Name: "mimocode", Status: "ok",
	})

	result.TotalSteps = len(result.Steps)
	result.Status = "success"
	result.Duration = time.Since(start).String()

	return result, nil
}

// detectCurrentChanges captures the current git diff stat as a fingerprint.
func detectCurrentChanges(root string) string {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain", "--",
		"runtimes/", ".mimocode/", ".opencode/")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// diffChanges compares two git status fingerprints.
func diffChanges(before, after string) []string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	beforeSet := make(map[string]bool)
	for _, l := range beforeLines {
		l = strings.TrimSpace(l)
		if l != "" {
			beforeSet[l] = true
		}
	}

	changes := []string{}
	for _, l := range afterLines {
		l = strings.TrimSpace(l)
		if l != "" && !beforeSet[l] {
			changes = append(changes, l)
		}
	}
	return changes
}

// NotifyProductUpdate broadcasts a product update event to Cockpit via cPanel SSE.
// This is called by the cPanel product handler after sync completes.
// The notification payload is a JSON string that cPanel broadcasts as-is.
func NotifyProductUpdate(fromVer, toVer string, changedFiles []string) string {
	return fmt.Sprintf(`{
	"event": "product_update",
	"from_version": "%s",
	"to_version": "%s",
	"changed_files": %d,
	"message": "OVAV Product sync complete — restart to apply"
}`, fromVer, toVer, len(changedFiles))
}

// VERSION file path relative to OVAV root.
func versionFilePath(root string) string {
	return filepath.Join(root, "VERSION")
}

// ReadVersion reads the current VERSION from the OVAV root.
func ReadVersion(root string) string {
	data, err := os.ReadFile(versionFilePath(root))
	if err != nil {
		return "0.0.0"
	}
	return strings.TrimSpace(string(data))
}
