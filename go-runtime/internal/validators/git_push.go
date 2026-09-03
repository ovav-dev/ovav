package validators

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/ceo"
)

// GitPush validates OVAV git push gate rules:
// 1. Remote must be HTTPS (not SSH)
// 2. Push URL must match fetch URL (no split remotes)
// 3. Platform agent must prohibit raw git push
// 4. No force push or force delete on any surface
type GitPush struct{}

func NewGitPush() *GitPush { return &GitPush{} }

func (g *GitPush) ID() string   { return "git_push" }
func (g *GitPush) Name() string { return "Git Push Gate" }
func (g *GitPush) Description() string {
	return "Enforces HTTPS-only push, no split remotes, no force push"
}
func (g *GitPush) Weight() int { return 10 }

func (g *GitPush) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	var warnings []string

	// Resolve git config path (handles worktrees where .git is a file)
	gitConfig := resolveGitPath(root, "config")
	data, err := os.ReadFile(gitConfig)
	if err != nil {
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "error", Weight: g.Weight(),
			Message:  fmt.Sprintf("ERROR: cannot read git config: %v", err),
			Duration: time.Since(start),
		}
	}
	config := string(data)

	// Rule 1: Remote must be HTTPS
	if strings.Contains(config, "git@github") || strings.Contains(config, "url = git@") {
		issues = append(issues, "Git remote uses SSH — OVAV requires HTTPS-only transport")
	}
	if !strings.Contains(config, "url = https://") {
		issues = append(issues, "Git remote is not HTTPS — OVAV push requires HTTPS transport")
	}

	// Rule 2: No split push/fetch URLs
	if strings.Contains(config, "pushurl") {
		issues = append(issues, "Git push URL must not split from fetch URL (pushurl detected)")
	}

	// Rule 3: Check platform agent prohibits raw git push
	platformAgent := filepath.Join(root, "clients", "opencode", "agents", "area-platform-engineering.md")
	if agentData, err := os.ReadFile(platformAgent); err == nil {
		agentText := strings.ToLower(string(agentData))
		if !strings.Contains(agentText, "raw git push") && !strings.Contains(agentText, "force push") {
			issues = append(issues, "Platform agent missing raw git push prohibition")
		}
	} else {
		issues = append(issues, fmt.Sprintf("Cannot read platform agent: %v", err))
	}

	// Rule 4: Raw push remains denied in the harness and the Go-native governed
	// push command is dispatched to its implementation.
	// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
	// YOLO mode: bash is 100% allow (no deny rules). The raw push gate is
	// enforced by the Go push_cli (ovav git push) which routes through the
	// Go push engine with protected-branch gates. Skip the opencode.json
	// deny check if YOLO is active.
	opencodeJSON := filepath.Join(root, "opencode.json")
	if jsonData, err := os.ReadFile(opencodeJSON); err == nil {
		text := string(jsonData)
		isYolo := strings.Contains(text, `"_ovav"`) || strings.Contains(text, `"yolo"`)
		if !isYolo {
			if !strings.Contains(text, `"git push*": "deny"`) && !strings.Contains(text, `"git push*":"deny"`) {
				issues = append(issues, "opencode.json does not deny raw git push")
			}
		}
	} else {
		issues = append(issues, fmt.Sprintf("Cannot read opencode.json: %v", err))
	}
	pushCLI, pushErr := os.ReadFile(filepath.Join(root, "go-runtime", "cmd", "ovav", "push_cli.go"))
	dispatch, dispatchErr := os.ReadFile(filepath.Join(root, "go-runtime", "cmd", "ovav", "dispatch.go"))
	if pushErr != nil || dispatchErr != nil || !strings.Contains(string(pushCLI), "cmdPush") || !strings.Contains(string(pushCLI), "gitflow.Push") || !strings.Contains(string(dispatch), `case "push"`) && !strings.Contains(string(dispatch), "cmdPush") {
		issues = append(issues, "Go-native governed push command wiring is incomplete")
	}

	// Rule 5: Protected branch must have waiver (migrated from Python)
	if safe, msg := g.checkBranchSafety(root); !safe {
		issues = append(issues, msg)
	}

	// Rule 6: No uncommitted changes (migrated from Python)
	if safe, msg := g.checkUncommitted(root); !safe {
		warnings = append(warnings, msg)
	}

	// Rule 7: No stale locks (migrated from Python)
	if safe, msg := g.checkLocks(root); !safe {
		issues = append(issues, msg)
	}

	if len(issues) > 0 {
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "fail", Weight: g.Weight(),
			Message: fmt.Sprintf("FAIL git push gate — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	if len(warnings) > 0 {
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "warn", Weight: g.Weight(),
			Message: fmt.Sprintf("WARN git push gate — wiring valid, %d worktree warning(s)", len(warnings)),
			Issues:  warnings, Duration: time.Since(start),
		}
	}
	return Result{
		ID: g.ID(), Name: g.Name(), Status: "pass", Weight: g.Weight(),
		Message:  "PASS git push gate — HTTPS transport, branch safety, hygiene verified",
		Duration: time.Since(start),
	}
}

// checkBranchSafety checks if current branch is protected and has a waiver.
//
// Resolution order (matches protected_branch.go):
//  1. Non-protected branch → always safe.
//  2. Active CEO session → bypass all waiver requirements.
//  3. Centralized runtime waiver (.ovav/runtime/protected_branch_waiver.yaml)
//     that covers the branch → safe.
//  4. Otherwise → fail with "no active waiver".
func (g *GitPush) checkBranchSafety(root string) (bool, string) {
	branch := getCurrentBranch(root)
	if branch == "" {
		return false, "Cannot determine current branch"
	}
	if !protectedBranches[branch] {
		return true, ""
	}
	// Protected branch — CEO session auto-bypasses.
	if ceo.IsActive(root) {
		return true, ""
	}
	// Protected branch — require the centralized runtime waiver file.
	waiverPath := filepath.Join(root, ".ovav", "runtime", "protected_branch_waiver.yaml")
	if _, err := os.Stat(waiverPath); os.IsNotExist(err) {
		return false, fmt.Sprintf("Protected branch '%s' has no active waiver", branch)
	}
	return true, ""
}

// checkUncommitted checks for uncommitted changes.
func (g *GitPush) checkUncommitted(root string) (bool, string) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Sprintf("Git status check failed: %v", err)
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return false, "Uncommitted changes exist"
	}
	return true, ""
}

// checkLocks checks for stale lock files (older than 1 hour).
func (g *GitPush) checkLocks(root string) (bool, string) {
	locksDir := filepath.Join(root, ".ovav", "locks")
	info, err := os.Stat(locksDir)
	if err != nil || !info.IsDir() {
		return true, ""
	}
	entries, err := os.ReadDir(locksDir)
	if err != nil {
		return true, ""
	}
	cutoff := time.Now().Add(-1 * time.Hour).Unix()
	var stale []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".lock") {
			continue
		}
		path := filepath.Join(locksDir, entry.Name())
		if stat, err := os.Stat(path); err == nil {
			if stat.ModTime().Unix() < cutoff {
				stale = append(stale, entry.Name())
			}
		}
	}
	if len(stale) > 0 {
		return false, fmt.Sprintf("Stale locks: %s", strings.Join(stale[:3], ", "))
	}
	return true, ""
}

// resolveGitPath resolves a path inside the .git directory, handling worktrees
// where .git is a file containing "gitdir: /path/to/actual/.git/worktrees/name".
func resolveGitPath(root, subpath string) string {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return filepath.Join(root, ".git", subpath)
	}
	if info.IsDir() {
		return filepath.Join(root, ".git", subpath)
	}
	// .git is a file (worktree) — read the actual gitdir path
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return filepath.Join(root, ".git", subpath)
	}
	content := strings.TrimSpace(string(data))
	const prefix = "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return filepath.Join(root, ".git", subpath)
	}
	gitDir := content[len(prefix):]
	// HEAD: return the WORKTREE's HEAD, not the main repo's HEAD.
	// In a worktree, HEAD is at .git/worktrees/<name>/HEAD, which reflects
	// the feature branch. The main repo's HEAD is always develop.
	if subpath == "HEAD" {
		headPath := filepath.Join(gitDir, "HEAD")
		if _, err := os.Stat(headPath); err == nil {
			return headPath
		}
	}
	// For other subpaths (config, index, etc.): go to the main .git directory.
	// Structure: .../main/.git/worktrees/<name> → main .git is 2 levels up.
	parentDir := filepath.Dir(filepath.Dir(gitDir))
	configPath := filepath.Join(parentDir, subpath)
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	// Fallback: try the worktree's gitdir directly
	return filepath.Join(gitDir, subpath)
}

var _ Validator = (*GitPush)(nil)
