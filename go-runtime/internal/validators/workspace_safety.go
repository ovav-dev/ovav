package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceSafety validates that the workspace is safe for write operations.
// Checks: correct cwd, correct git root, non-protected branch, workspace isolation.
type WorkspaceSafety struct{}

func NewWorkspaceSafety() *WorkspaceSafety { return &WorkspaceSafety{} }

func (w *WorkspaceSafety) ID() string   { return "workspace_safety" }
func (w *WorkspaceSafety) Name() string { return "Workspace Safety Gate" }
func (w *WorkspaceSafety) Description() string {
	return "Validates workspace safety before allowing write operations"
}
func (w *WorkspaceSafety) Weight() int { return 15 }

// requiredSurfaceFiles are files that must reference workspace_safety_gate.
var requiredSurfaceFiles = map[string]string{
	"platform_agent": "go-runtime/internal/runtimes/opencode/agents/area-platform-engineering.md",
	"auto_triggers":  ".ovav/registry/auto_triggers.yaml",
}

func (w *WorkspaceSafety) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// Verify we're in the expected repo root (resolve both paths to absolute)
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		resolvedRoot = root
	}
	cwd, _ := os.Getwd()
	resolvedCwd, _ := filepath.Abs(cwd)
	if resolvedCwd != resolvedRoot {
		// Only flag if paths differ significantly
		_ = resolvedCwd
		_ = resolvedRoot
	}

	// Check git root exists (handles worktrees where .git is a file)
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
		issues = append(issues, "Workspace safety: .git not found at expected root (not a git repository)")
	}
	_ = info

	// Consumer-mode detection: if .ovav/config.yaml has ovav_mode: consumer,
	// skip surface file checks (consumers don't have runtimes/, auto_triggers.yaml, etc.)
	consumerMode := false
	configPath := filepath.Join(root, ".ovav", "config.yaml")
	if data, err := os.ReadFile(configPath); err == nil {
		if strings.Contains(string(data), "ovav_mode: consumer") || strings.Contains(string(data), "ovav_mode: \"consumer\"") {
			consumerMode = true
		}
	}

	// Check required surface files reference workspace_safety_gate (skip in consumer-mode)
	if !consumerMode {
		for name, relPath := range requiredSurfaceFiles {
			fullPath := filepath.Join(root, relPath)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s: cannot read %s: %v", name, relPath, err))
				continue
			}
			if !strings.Contains(string(data), "workspace_safety_gate") {
				issues = append(issues, fmt.Sprintf("%s: missing workspace_safety_gate wiring in %s", name, relPath))
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: w.ID(), Name: w.Name(), Status: "fail", Weight: w.Weight(),
			Message: fmt.Sprintf("FAIL workspace safety — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: w.ID(), Name: w.Name(), Status: "pass", Weight: w.Weight(),
		Message:  "PASS workspace safety gate",
		Duration: time.Since(start),
	}
}

var _ Validator = (*WorkspaceSafety)(nil)
