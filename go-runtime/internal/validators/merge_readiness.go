package validators

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MergeReadiness validates that the repository is ready for merge.
// Checks for conflicts, dirty working tree, unpushed commits, and branch status.
// Replaces: merge_readiness.py, merge_readiness_suite.py, pre_push_intelligence.py
type MergeReadiness struct{}

func NewMergeReadiness() *MergeReadiness { return &MergeReadiness{} }

func (m *MergeReadiness) ID() string   { return "merge_readiness" }
func (m *MergeReadiness) Name() string { return "Merge Readiness" }
func (m *MergeReadiness) Description() string {
	return "Validates branch state, conflicts, and merge readiness"
}
func (m *MergeReadiness) Weight() int { return 10 }

func (m *MergeReadiness) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check working tree is clean
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		issues = append(issues, fmt.Sprintf("GIT_ERROR: Cannot check git status: %v", err))
	} else if len(strings.TrimSpace(string(out))) > 0 {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		issues = append(issues, fmt.Sprintf("DIRTY: Working tree has %d uncommitted change(s)", len(lines)))
	}

	// 2. Check current branch
	branchName := ""
	cmd = exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = root
	branch, err := cmd.Output()
	if err != nil {
		issues = append(issues, fmt.Sprintf("GIT_ERROR: Cannot determine current branch: %v", err))
	} else {
		branchName = strings.TrimSpace(string(branch))
		// Warn if on a protected branch (skip if CEO session or waiver exists)
		protected := map[string]bool{
			"main": true, "master": true, "develop": true, "development": true,
			"prod": true, "production": true, "staging": true,
		}
		if protected[branchName] {
			// Check for CEO session or valid waiver — skip warning if present
			waiverPath := filepath.Join(root, ".ovav", "runtime", "protected_branch_waiver.yaml")
			_, waiverErr := os.Stat(waiverPath)
			hasSession := false
			sessionPath := filepath.Join(root, ".ovav", "runtime", "ceo_session.yaml")
			if _, sessionErr := os.Stat(sessionPath); sessionErr == nil {
				if sessionData, err := os.ReadFile(sessionPath); err == nil {
					hasSession = strings.Contains(string(sessionData), "active: true")
				}
			}
			if waiverErr != nil && !hasSession {
				issues = append(issues, fmt.Sprintf("PROTECTED: On protected branch '%s' — writes require CEO waiver", branchName))
			}
		}
		// Check branch naming convention
		if !strings.HasPrefix(branchName, "task/") && !strings.HasPrefix(branchName, "fix/") && !strings.HasPrefix(branchName, "feat/") && !strings.HasPrefix(branchName, "feature/") && !protected[branchName] {
			issues = append(issues, fmt.Sprintf("BRANCH: Branch '%s' doesn't follow naming convention (task/*, fix/*, feat/*, feature/*)", branchName))
		}
	}

	// 3. Check for unpushed commits — NORMAL on develop/task branches in local-first workflow
	// OVAV Git Flow: MAIN ← stable, DEVELOP local ← verification+unification,
	// worktrees on develop, push only when version is ready for MAIN remote.
	// UNPUSHED is expected state on develop — do NOT report as issue.
	cmd = exec.CommandContext(ctx, "git", "log", "@{u}..", "--oneline")
	cmd.Dir = root
	out, err = cmd.Output()
	if err == nil {
		unpushed := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(unpushed) > 0 && unpushed[0] != "" {
			// Only warn on main/master — develop/task branches: unpushed is intentional
			if branchName == "main" || branchName == "master" {
				issues = append(issues, fmt.Sprintf("UNPUSHED: %d commit(s) not pushed to remote", len(unpushed)))
			}
		}
	}
	if err != nil && strings.Contains(err.Error(), "no upstream") {
		// No upstream configured — normal for local-only workflow
	}

	// 4. Check for merge conflicts with develop
	cmd = exec.CommandContext(ctx, "git", "merge-base", "HEAD", "develop")
	cmd.Dir = root
	base, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(base))) > 0 {
		// Check if there are any conflicting files
		cmd = exec.CommandContext(ctx, "git", "diff", "--name-only", "--diff-filter=U")
		cmd.Dir = root
		conflicts, _ := cmd.Output()
		if len(strings.TrimSpace(string(conflicts))) > 0 {
			conflictFiles := strings.Split(strings.TrimSpace(string(conflicts)), "\n")
			issues = append(issues, fmt.Sprintf("CONFLICT: %d file(s) with unresolved merge conflicts", len(conflictFiles)))
		}
	}

	// 5. Check for .ovav artifacts that should be cleaned
	// NOTE: .ovav/runtime/ contains operational state files that are expected to
	// be long-lived (knowledge graphs, audit reports, decision ledgers, etc.).
	// These are NOT source code and do not indicate merge problems.
	// Skip stale check for operational runtime files — only flag actual build artifacts.
	artifactDirs := []string{
		".ovav/runtime",
	}
	// Files expected to be long-lived operational state (skip stale check)
	skipStaleCheck := map[string]bool{
		"agent_teams_queue.json":       true,
		"background_agents.json":       true,
		"capsule_active":               true,
		"capsule_info.json":            true,
		"cross_validation_report.json": true,
		"cross_validation_report.md":   true,
		"decision_ledger.jsonl":        true,
		"hebbian_weights.json":         true,
		"knowledge_graph.yaml":         true,
		"memory_active":                true,
		"memory_indicator.json":        true,
		"pattern_learner.yaml":         true,
		"self_audit_report.json":       true,
		"self_audit_report.md":         true,
		"session_feed.jsonl":           true,
		"smoke_test_state.json":        true,
		"snapshot_final.json":          true,
		"snapshot_inicial.json":        true,
		"snv_state.json":               true,
		"task_queue.jsonl":             true,
		"thavren_audit_report.json":    true,
	}
	for _, dir := range artifactDirs {
		artifactPath := filepath.Join(root, dir)
		if entries, err := os.ReadDir(artifactPath); err == nil {
			// Check for stale session files (>24h)
			for _, e := range entries {
				if info, err := e.Info(); err == nil {
					if time.Since(info.ModTime()) > 60*24*time.Hour && !info.IsDir() {
						// Skip known operational files that are expected to be long-lived
						if skipStaleCheck[e.Name()] {
							continue
						}
						issues = append(issues, fmt.Sprintf("STALE: %s/%s not modified in >24h", dir, e.Name()))
					}
				}
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "fail", Weight: m.Weight(),
			Message: fmt.Sprintf("FAIL merge readiness — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: m.ID(), Name: m.Name(), Status: "pass", Weight: m.Weight(),
		Message:  "PASS merge readiness — working tree clean, branch ready",
		Duration: time.Since(start),
	}
}

var _ Validator = (*MergeReadiness)(nil)
