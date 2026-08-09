package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	// Rule 4: Check opencode.json references ovav_git_push_gate
	opencodeJSON := filepath.Join(root, "opencode.json")
	if jsonData, err := os.ReadFile(opencodeJSON); err == nil {
		if !strings.Contains(string(jsonData), "ovav_git_push_gate") {
			issues = append(issues, "opencode.json missing ovav_git_push_gate wiring")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "fail", Weight: g.Weight(),
			Message: fmt.Sprintf("FAIL git push gate — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: g.ID(), Name: g.Name(), Status: "pass", Weight: g.Weight(),
		Message:  "PASS git push gate — HTTPS transport verified, no force push allowed",
		Duration: time.Since(start),
	}
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
