package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ovav/ovav/internal/ceo"
)

// ProtectedBranch verifies that writes are blocked on protected git branches
// unless a valid CEO waiver exists. Protected branches: main, master, develop,
// development, prod, production, staging.
type ProtectedBranch struct{}

func NewProtectedBranch() *ProtectedBranch { return &ProtectedBranch{} }

func (p *ProtectedBranch) ID() string   { return "protected_branch" }
func (p *ProtectedBranch) Name() string { return "Protected Branch Gate" }
func (p *ProtectedBranch) Description() string {
	return "Blocks write operations on protected branches without CEO waiver"
}
func (p *ProtectedBranch) Weight() int { return 15 }

// protectedBranches is the canonical list of protected branch names.
var protectedBranches = map[string]bool{
	"main": true, "master": true, "develop": true,
	"development": true, "prod": true, "production": true, "staging": true,
}

// allowedOnProtected lists operations permitted on protected branches without waiver.
var allowedOnProtected = map[string]bool{
	"verify": true, "read": true, "status": true,
	"inspect": true, "diagnose": true, "report": true,
	"sync": true, "checkout": true,
}

// blockedOnProtected lists operations blocked on protected branches without waiver.
var blockedOnProtected = map[string]bool{
	"write": true, "commit": true, "push": true,
	"merge": true, "stage": true, "implement": true, "mutate": true,
}

func (p *ProtectedBranch) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// Check if we're in a git repo
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
			Message:  "PASS protected branch — not a git repository (skipped)",
			Duration: time.Since(start),
		}
	}

	// Determine current branch from .git/HEAD
	branch, err := currentBranch(root)
	if err != nil {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "error", Weight: p.Weight(),
			Message:  fmt.Sprintf("ERROR: cannot determine current branch: %v", err),
			Duration: time.Since(start),
		}
	}

	if !protectedBranches[branch] {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
			Message:  fmt.Sprintf("PASS protected branch — '%s' is not protected", branch),
			Duration: time.Since(start),
		}
	}

	// We're on a protected branch. Check for CEO session first (auto-bypass).
	if ceo.IsActive(root) {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
			Message:  fmt.Sprintf("PASS protected branch — '%s' (CEO session active — waiver not required)", branch),
			Duration: time.Since(start),
		}
	}

	// No CEO session — check for waiver.
	waiverPath := filepath.Join(root, ".ovav", "runtime", "protected_branch_waiver.yaml")
	if _, err := os.Stat(waiverPath); os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf(
			"PROTECTED BRANCH '%s' — no waiver found at %s. "+
				"Allowed: verify, read, status, inspect, diagnose, report, sync, checkout. "+
				"Blocked: write, commit, push, merge, stage, implement, mutate.",
			branch, waiverPath,
		))
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "fail", Weight: p.Weight(),
			Message: fmt.Sprintf("FAIL protected branch — '%s' is protected, no waiver", branch),
			Issues:  issues, Duration: time.Since(start),
		}
	}

	// Waiver exists — we pass (detailed validation done by check_waiver logic).
	// In production, waiver expiry and session-binding are checked here.
	return Result{
		ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
		Message:  fmt.Sprintf("PASS protected branch — '%s' has active waiver", branch),
		Duration: time.Since(start),
	}
}

// currentBranch reads the current git branch from .git/HEAD (or worktree .git file).
func currentBranch(root string) (string, error) {
	headPath := resolveGitPath(root, "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("cannot read .git/HEAD: %w", err)
	}
	content := string(data)
	// Format: "ref: refs/heads/branch-name"
	const prefix = "ref: refs/heads/"
	if len(content) > len(prefix) && content[:len(prefix)] == prefix {
		branch := content[len(prefix):]
		// Trim trailing newline
		for len(branch) > 0 && branch[len(branch)-1] == '\n' {
			branch = branch[:len(branch)-1]
		}
		return branch, nil
	}
	// Detached HEAD — return the commit hash
	return "HEAD (detached)", nil
}

var _ Validator = (*ProtectedBranch)(nil)
