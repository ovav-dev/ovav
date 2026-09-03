package validators

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ChangedFilesScope holds the set of changed files for scoped validation.
// When non-nil, validators WITHOUT a scope declaration are SKIPPED
// (they cannot be filtered to changed files).
// Set by SetChangedFiles() called from the validate CLI.
var ChangedFilesScope []string

// SetChangedFiles configures the changed-files scope for validator filtering.
// Call before running validators with --changed-files.
func SetChangedFiles(files []string) {
	ChangedFilesScope = files
}

// ClearChangedFiles resets the changed-files scope.
func ClearChangedFiles() {
	ChangedFilesScope = nil
}

// ValidatorScope maps validator IDs to their relevant path scopes.
// A validator only runs if at least one of its scope paths was changed
// in the current branch (vs the parent branch).
// If a scope path doesn't exist in the worktree, the validator is skipped entirely.
var ValidatorScope = map[string][]string{
	"agent_governance":          {".ovav/service_areas/", "go-runtime/internal/runtimes/claude-code/agents/"},
	"agent_runtime_enforcement": {"go-runtime/internal/agents/", "go-runtime/internal/validators/context_firewall_v2.go", ".ovav/service_areas/shared/"},
	"squad_normalization":       {"go-runtime/internal/agents/", "go-runtime/internal/validators/context_firewall_v2.go", ".ovav/registry/", ".ovav/service_areas/shared/"},
	"tool_config_profiles":      {"tools/cli/"},
	"service_area_router":       {".ovav/service_areas/"},
	"agent_ux_visual_delivery":  {".ovav/service_areas/"},
	"cross_target_consistency":  {".ovav/service_areas/"},
	"feedback_loop":             {"go-runtime/internal/agents/belief.go", "go-runtime/internal/memory/governor.go"},
	"rego_policies":             {"go-runtime/internal/permissions/", ".ovav/registry/rego_policies/", ".ovav/policy/rego/"},
	"f1_architecture":           {"go-runtime/internal/permissions/", "go-runtime/internal/security/bootstrap.go", ".ovav/policy/"},
	"context_firewall_v2":       {".ovav/service_areas/"},
	"context_firewall":          {".ovav/service_areas/"},
	// caps_chronos_alignment: only runs when caps.yaml was modified in the branch.
	// During owd pre-merge, if caps.yaml was not touched by the feature branch,
	// the validator is skipped since caps.yaml will be updated post-merge from develop.
	"caps_chronos_alignment": {".ovav/plan/caps.yaml"},
}

// ScopeCheck determines if a validator should run based on changed files.
// Returns (shouldRun, statusMessage).
// If shouldRun=false, the validator should return a SKIP result.
//
// When ChangedFilesScope is set (scoped validation mode), validators
// WITHOUT a scope declaration are SKIPPED since they cannot be filtered
// to specific changed files.
func ScopeCheck(validatorID, root string) (bool, string) {
	scopes, ok := ValidatorScope[validatorID]
	if !ok || len(scopes) == 0 {
		// No scope declared.
		// In scoped mode (ChangedFilesScope set), skip validators that can't be scoped.
		if len(ChangedFilesScope) > 0 {
			return false, "SKIPPED (no scope declared) — cannot filter to changed files"
		}
		return true, "" // no ChangedFilesScope → always run
	}

	// Check if scope path exists AND has git-tracked files in this worktree's HEAD
	scopePath := filepath.Join(root, scopes[0])
	if _, err := os.Stat(scopePath); err != nil {
		return false, "SKIPPED (not applicable) — " + scopePath + " does not exist in this worktree"
	}

	// Check if scope path has any git-tracked files in this worktree's committed state.
	// If the scope path is empty in git HEAD, the validator is not applicable to this worktree.
	trackedCmd := exec.Command("git", "ls-files", scopes[0])
	trackedCmd.Dir = root
	trackedOut, trackedErr := trackedCmd.Output()
	if trackedErr != nil || strings.TrimSpace(string(trackedOut)) == "" {
		// No files tracked in this scope path → validator not applicable for this worktree
		return false, "SKIPPED (not applicable) — " + scopes[0] + " has no git-tracked files in this worktree"
	}

	// Get current branch
	branch := getCurrentBranch(root)
	if branch == "" {
		return true, "" // can't determine branch → run
	}

	// Determine parent branch
	parent := "develop"
	if strings.HasPrefix(branch, "feature/") || strings.HasPrefix(branch, "bugfix/") {
		parent = "develop"
	} else if strings.HasPrefix(branch, "hotfix/") {
		parent = "main"
	}

	// If we're on the base branch itself (develop, main), always run —
	// scope filtering only makes sense for feature branches where we compare
	// against the parent's new changes. On base branch, we need validators
	// like context_firewall to run to populate the domain tracker.
	if branch == parent {
		return true, ""
	}

	// Determine the set of changed files to check against.
	// If ChangedFilesScope is set (scoped validation from owv --changed-only),
	// use that list directly instead of re-running git diff.
	var changedFiles []string
	if len(ChangedFilesScope) > 0 {
		changedFiles = ChangedFilesScope
	} else {
		// Get changed files vs parent via git diff
		cmd := exec.Command("git", "diff", "--name-only", parent+"..."+branch)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return true, "" // can't diff → run
		}
		changedFiles = strings.Split(strings.TrimSpace(string(out)), "\n")
		changedFiles = append(changedFiles, worktreeChangedFiles(root)...)
	}

	for _, line := range changedFiles {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, scope := range scopes {
			if strings.HasPrefix(line, scope) {
				return true, ""
			}
		}
	}

	return false, "SKIPPED (scope) — no relevant files changed in this branch"
}

func worktreeChangedFiles(root string) []string {
	var files []string
	for _, args := range [][]string{{"diff", "--name-only"}, {"ls-files", "--others", "--exclude-standard"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err == nil {
			files = append(files, strings.Split(strings.TrimSpace(string(out)), "\n")...)
		}
	}
	return files
}

func getCurrentBranch(root string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = root
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}
