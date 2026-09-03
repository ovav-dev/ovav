package ows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ── F4: Worktree Routing Engine ────────────────────────────────────────────

// RouteMode defines the routing strategy for moving commits between branches.
type RouteMode string

const (
	RouteCherryPick RouteMode = "cherry-pick" // selective commit transfer
	RoutePatch      RouteMode = "patch"       // all commits as a single patch
	RouteHotfix     RouteMode = "hotfix"      // apply to main + develop simultaneously
	RouteEmergency  RouteMode = "emergency"   // bypass policies with waiver
)

// RouteResult captures the outcome of a routing operation.
type RouteResult struct {
	Mode      RouteMode `json:"mode"`
	SourceRef string    `json:"source_ref"`
	TargetRef string    `json:"target_ref"`
	Commits   []string  `json:"commits_transferred"`
	Skipped   []string  `json:"commits_skipped,omitempty"`
	Conflicts []string  `json:"conflicts,omitempty"`
	Success   bool      `json:"success"`
}

// Route transfers commits from source to target using the specified mode.
// This implements the `owx` (ovav worktree route) command.
func Route(ctx context.Context, repoRoot, sourceRef, targetRef string, mode RouteMode) (*RouteResult, error) {
	// Detect CI environment — some modes require interactive confirmation
	if IsCI() && mode == RouteEmergency {
		return nil, fmt.Errorf("emergency routing requires interactive confirmation — blocked in CI")
	}

	result := &RouteResult{
		Mode:      mode,
		SourceRef: sourceRef,
		TargetRef: targetRef,
	}

	// Get commits unique to source (not in target)
	commits, err := uniqueCommits(repoRoot, sourceRef, targetRef)
	if err != nil {
		return nil, fmt.Errorf("enumerate commits: %w", err)
	}

	switch mode {
	case RouteCherryPick:
		return routeCherryPick(ctx, repoRoot, sourceRef, targetRef, commits, result)
	case RoutePatch:
		return routePatch(ctx, repoRoot, sourceRef, targetRef, commits, result)
	case RouteHotfix:
		return routeHotfix(ctx, repoRoot, sourceRef, targetRef, commits, result)
	case RouteEmergency:
		return routeEmergency(ctx, repoRoot, sourceRef, targetRef, commits, result)
	default:
		return nil, fmt.Errorf("unknown route mode: %s", mode)
	}
}

// Abort rolls back an in-progress operation. Implements `owa`.
func Abort(repoRoot string) error {
	// Check if there's a cherry-pick or rebase in progress
	if isInProgress(repoRoot, "CHERRY_PICK_HEAD") {
		out, err := runGitOutput(repoRoot, "cherry-pick", "--abort")
		if err != nil {
			return fmt.Errorf("abort cherry-pick: %w\n%s", err, out)
		}
		return nil
	}
	if isInProgress(repoRoot, "REBASE_HEAD") {
		out, err := runGitOutput(repoRoot, "rebase", "--abort")
		if err != nil {
			return fmt.Errorf("abort rebase: %w\n%s", err, out)
		}
		return nil
	}
	if isInProgress(repoRoot, "MERGE_HEAD") {
		out, err := runGitOutput(repoRoot, "merge", "--abort")
		if err != nil {
			return fmt.Errorf("abort merge: %w\n%s", err, out)
		}
		return nil
	}
	return fmt.Errorf("no operation in progress to abort")
}

// IsCI detects if running in a CI/CD environment.
func IsCI() bool {
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_HOME", "BUILD_ID", "DRONE"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// ── Route Implementations ────────────────────────────────────────────────

func routeCherryPick(ctx context.Context, repoRoot, sourceRef, targetRef string, commits []string, result *RouteResult) (*RouteResult, error) {
	// Save current branch
	currentBranch, err := currentBranch(repoRoot)
	if err != nil {
		return nil, err
	}

	// Switch to target
	if err := checkout(repoRoot, targetRef); err != nil {
		return nil, fmt.Errorf("checkout target %s: %w", targetRef, err)
	}

	for _, commit := range commits {
		select {
		case <-ctx.Done():
			// Restore original branch before returning
			_ = checkout(repoRoot, currentBranch)
			return result, ctx.Err()
		default:
		}

		out, err := runGitOutput(repoRoot, "cherry-pick", "--no-commit", commit)
		if err != nil {
			// Conflict — abort this cherry-pick and skip
			_ = runGit(repoRoot, "cherry-pick", "--abort")
			result.Skipped = append(result.Skipped, commit)
			result.Conflicts = append(result.Conflicts, out)
			continue
		}
		result.Commits = append(result.Commits, commit)
	}

	// Commit if there are staged changes
	if hasStagedChanges(repoRoot) {
		runGitCmdNoFail(repoRoot, "commit", "-m", fmt.Sprintf("owx: cherry-pick from %s", sourceRef))
	}

	// Restore original branch
	_ = checkout(repoRoot, currentBranch)
	result.Success = len(result.Skipped) == 0
	return result, nil
}

func routePatch(ctx context.Context, repoRoot, sourceRef, targetRef string, commits []string, result *RouteResult) (*RouteResult, error) {
	currentBranch, err := currentBranch(repoRoot)
	if err != nil {
		return nil, err
	}

	// A fast-forward preserves the complete source history and avoids asking
	// git apply to materialize large repositories containing binaries, renames,
	// and merge commits. It is the lossless form of patch routing whenever the
	// target is already an ancestor of the source.
	if isAncestor(repoRoot, targetRef, sourceRef) {
		if targetRef == sourceRef {
			result.Commits = commits
			result.Success = true
			return result, nil
		}
		if err := checkout(repoRoot, targetRef); err != nil {
			return nil, fmt.Errorf("checkout target: %w", err)
		}
		if _, err := runGitOutput(repoRoot, "merge", "--ff-only", sourceRef); err != nil {
			_ = checkout(repoRoot, currentBranch)
			return nil, fmt.Errorf("fast-forward source: %w", err)
		}
		if err := checkout(repoRoot, currentBranch); err != nil {
			return nil, fmt.Errorf("restore source branch: %w", err)
		}
		result.Commits = commits
		result.Success = true
		return result, nil
	}

	// Generate patch from all commits
	patch, err := runGitOutput(repoRoot, "format-patch", "--stdout", targetRef+".."+sourceRef)
	if err != nil {
		return nil, fmt.Errorf("generate patch: %w", err)
	}

	// Switch to target
	if err := checkout(repoRoot, targetRef); err != nil {
		return nil, fmt.Errorf("checkout target: %w", err)
	}

	// Apply patch
	cmd := exec.Command("git", "apply", "--check")
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(patch)
	if err := cmd.Run(); err != nil {
		result.Conflicts = append(result.Conflicts, "patch does not apply cleanly: "+err.Error())
		_ = checkout(repoRoot, currentBranch)
		return result, nil
	}

	// Apply for real
	cmd = exec.Command("git", "apply")
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(patch)
	if err := cmd.Run(); err != nil {
		_ = checkout(repoRoot, currentBranch)
		return nil, fmt.Errorf("apply patch: %w", err)
	}

	result.Commits = commits
	runGitCmdNoFail(repoRoot, "add", "-A")
	runGitCmdNoFail(repoRoot, "commit", "-m", fmt.Sprintf("owx: patch from %s (%d commits)", sourceRef, len(commits)))

	_ = checkout(repoRoot, currentBranch)
	result.Success = true
	return result, nil
}

// isAncestor reports whether ancestor is reachable from descendant. A false
// result includes the normal divergent-history case and command failure; the
// caller then uses the regular patch/conflict path.
func isAncestor(repoRoot, ancestor, descendant string) bool {
	_, err := runGitOutput(repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

func routeHotfix(ctx context.Context, repoRoot, sourceRef, targetRef string, commits []string, result *RouteResult) (*RouteResult, error) {
	// Hotfix: apply to main AND develop simultaneously
	targets := []string{"main", "develop"}
	for _, tgt := range targets {
		tgtResult, err := routeCherryPick(ctx, repoRoot, sourceRef, tgt, commits, &RouteResult{Mode: RouteCherryPick})
		if err != nil {
			return nil, fmt.Errorf("hotfix to %s: %w", tgt, err)
		}
		result.Commits = append(result.Commits, tgtResult.Commits...)
		if !tgtResult.Success {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("hotfix to %s: conflicts in %v", tgt, tgtResult.Skipped))
		}
	}
	result.Success = len(result.Conflicts) == 0
	return result, nil
}

func routeEmergency(ctx context.Context, repoRoot, sourceRef, targetRef string, commits []string, result *RouteResult) (*RouteResult, error) {
	// OWS-B4 FIX: Validate waiver before allowing emergency routing.
	// The old implementation called routePatch() with NO waiver check,
	// despite the comment claiming "force-push allowed with waiver".
	waiver, err := LoadWaiver(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("emergency routing requires a valid CEO waiver: %w", err)
	}
	if err := ValidateWaiver(waiver, "owx", targetRef); err != nil {
		return nil, fmt.Errorf("emergency waiver invalid: %w", err)
	}
	fmt.Printf("  ⚠️  EMERGENCY ROUTE authorized by waiver %s (expires %s)\n",
		waiver.ID, time.Unix(waiver.ExpiresAt, 0).Format(time.RFC3339))
	return routePatch(ctx, repoRoot, sourceRef, targetRef, commits, result)
}

// ── Git Helpers ──────────────────────────────────────────────────────────

func uniqueCommits(repoRoot, sourceRef, targetRef string) ([]string, error) {
	out, err := runGitOutput(repoRoot, "log", "--oneline", "--format=%H", targetRef+".."+sourceRef)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

func currentBranch(repoRoot string) (string, error) {
	out, err := runGitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return out, nil
}

func checkout(repoRoot, ref string) error {
	return runGit(repoRoot, "checkout", ref)
}

func isInProgress(repoRoot, markerFile string) bool {
	// Check the actual git directory — handles worktrees where .git is a file
	// pointing to the main repo's .git directory
	gitDir := repoRoot + "/.git"
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		// Worktree: .git is a file containing "gitdir: <path>"
		data, err := os.ReadFile(gitDir)
		if err != nil {
			return false
		}
		content := strings.TrimSpace(string(data))
		if strings.HasPrefix(content, "gitdir:") {
			gitDir = strings.TrimSpace(strings.TrimPrefix(content, "gitdir:"))
		}
	}
	_, err = os.Stat(gitDir + "/" + markerFile)
	return err == nil
}

func hasStagedChanges(repoRoot string) bool {
	out, err := runGitOutput(repoRoot, "diff", "--cached", "--name-only")
	return err == nil && out != ""
}

func runGitCmdNoFail(repoRoot string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	_ = cmd.Run() // best-effort, errors ignored
}

func runGit(repoRoot string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}
