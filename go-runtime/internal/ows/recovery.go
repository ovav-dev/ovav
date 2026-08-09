package ows

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ── F5: Recovery & Atomic Operations ──────────────────────────────────────

// RescueResult captures the outcome of a recovery operation.
type RescueResult struct {
	RecoveredCommits   []string `json:"recovered_commits"`
	RecoveredBranches  []string `json:"recovered_branches"`
	RecoveredWorktrees []string `json:"recovered_worktrees"`
}

// Rescue scans reflog and orphaned branches/worktrees for recoverable data.
// Implements `owr` (ovav worktree rescue).
func Rescue(repoRoot string) (*RescueResult, error) {
	result := &RescueResult{}

	// 1. Scan reflog for recently lost commits (last 30 days)
	reflogOut, err := runGitOutput(repoRoot, "reflog", "--format=%H %s", "--since=30.days.ago")
	if err == nil && reflogOut != "" {
		lines := strings.Split(strings.TrimSpace(reflogOut), "\n")
		for _, line := range lines {
			if strings.Contains(line, "reset: moving") || strings.Contains(line, "branch: deleted") {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) > 0 {
					result.RecoveredCommits = append(result.RecoveredCommits, parts[0])
				}
			}
		}
	}

	// 2. Scan for orphaned branches (not merged, no remote tracking)
	branchOut, err := runGitOutput(repoRoot, "branch", "--no-merged", "develop")
	if err == nil && branchOut != "" {
		lines := strings.Split(strings.TrimSpace(branchOut), "\n")
		for _, line := range lines {
			name := strings.TrimSpace(strings.TrimPrefix(line, "*"))
			if name != "" && name != "develop" && name != "main" {
				result.RecoveredBranches = append(result.RecoveredBranches, name)
			}
		}
	}

	// 3. Scan for orphaned worktrees
	worktreeOut, err := runGitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err == nil {
		result.RecoveredWorktrees = parseWorktreeList(worktreeOut)
	}

	return result, nil
}

// Sync performs full repository maintenance, prunes stale worktrees,
// and cleans up auto-cleanup profiles (spike, research).
// Implements `ows` (ovav worktree sync).
func Sync(repoRoot string) error {
	// 1. Fetch all remotes
	if err := runGit(repoRoot, "fetch", "--all", "--prune"); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	// 2. Git maintenance (aggressive: gc, commit-graph, prefetch)
	maintenanceCmds := [][]string{
		{"gc", "--auto", "--aggressive"},
		{"commit-graph", "write", "--reachable", "--changed-paths"},
		{"maintenance", "run", "--task=gc", "--task=commit-graph", "--task=prefetch"},
	}
	for _, args := range maintenanceCmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		_ = cmd.Run()
	}

	// 3. Prune stale worktrees (7 days) AND auto-cleanup profiles
	staleCount := pruneStaleWorktrees(repoRoot, 7*24*time.Hour)
	if staleCount > 0 {
		fmt.Printf("  🧹 Pruned %d stale worktree(s)\n", staleCount)
	}
	return nil
}

// pruneStaleWorktrees removes worktrees that match auto-cleanup criteria:
// - Profile with AutoCleanup: true (spike, research) AND last modified > 7 days
// - Any worktree last modified > 30 days (abandoned)
func pruneStaleWorktrees(repoRoot string, maxAge time.Duration) int {
	out, err := runGitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return 0
	}

	worktrees := parseWorktreePaths(out)
	now := time.Now()
	pruned := 0

	for _, wtPath := range worktrees {
		if wtPath == repoRoot {
			continue
		}

		info, err := os.Stat(wtPath)
		if err != nil {
			continue
		}

		age := now.Sub(info.ModTime())

		// Auto-cleanup profiles: spike and research worktrees get removed after 7 days
		isAutoCleanup := strings.Contains(wtPath, "spike-") || strings.Contains(wtPath, "research-")

		// Abandoned worktrees: any worktree inactive > 30 days
		isAbandoned := age > 30*24*time.Hour

		// Stale auto-cleanup: spike/research > 7 days
		isStaleAutoCleanup := isAutoCleanup && age > maxAge

		if isStaleAutoCleanup || isAbandoned {
			reason := "stale"
			if isAbandoned {
				reason = "abandoned"
			}
			fmt.Printf("  🧹 Removing %s worktree: %s (age: %.0fd)\n", reason, wtPath, age.Hours()/24)
			if err := runGit(repoRoot, "worktree", "remove", "--force", wtPath); err == nil {
				pruned++
			}
		}
	}

	// Also prune git metadata
	runGit(repoRoot, "worktree", "prune")
	return pruned
}

// parseWorktreeList parses `git worktree list --porcelain` output.
func parseWorktreeList(out string) []string {
	var worktrees []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			worktrees = append(worktrees, strings.TrimPrefix(line, "worktree "))
		}
	}
	return worktrees
}

// parseWorktreePaths returns just the paths from porcelain output.
func parseWorktreePaths(out string) []string {
	return parseWorktreeList(out)
}

// VerifyResult holds structured output from a full verification run.
type VerifyResult struct {
	GoTestPass    bool    `json:"go_test_pass"`
	GoVetPass     bool    `json:"go_vet_pass"`
	GofmtPass     bool    `json:"gofmt_pass"`
	CoveragePass  bool    `json:"coverage_pass"`
	CoveragePct   float64 `json:"coverage_pct"`
	ValidatePass  int     `json:"validate_pass"`
	ValidateFail  int     `json:"validate_fail"`
	ValidateTotal int     `json:"validate_total"`
	ValidateRan   bool    `json:"validate_ran"`
	HygieneClean  bool    `json:"hygiene_clean"`
	HygieneIssues int     `json:"hygiene_issues"`
	Passed        bool    `json:"passed"`
	Detail        string  `json:"detail"`
	Scoped        bool    `json:"scoped"` // true if running in scoped mode (changed-files set)
}

// Verify runs the full OVAV validation pipeline and returns structured results.
// Implements `owv` (ovav worktree verify).
// Stack-aware: detects project type and runs validators from correct directories.
// Consumer-grade: works for Go, TypeScript, Python, Rust, monorepos, and unknown stacks.
//
// When changedFiles is non-empty, runs in scoped mode: only tests packages
// containing changed .go files, passes --changed-files to validators, and
// filters hygiene checks to only changed files.
func Verify(repoRoot string, changedFiles []string, quick ...bool) (*VerifyResult, error) {
	var issues []string
	r := &VerifyResult{Scoped: len(changedFiles) > 0}
	scope := LoadScopeConfig()

	// Guard: invalid repo root — fail all checks
	if _, err := os.Stat(repoRoot); os.IsNotExist(err) {
		r.GoVetPass = false
		r.GofmtPass = false
		r.GoTestPass = false
		r.HygieneClean = false
		r.Passed = false
		r.Detail = "repo root does not exist: " + repoRoot
		return r, nil
	}

	// Phase 0: Stack Detection
	stack := DetectStacks(repoRoot)
	primary := stack.PrimaryStack()
	_ = primary // used for logging in handler

	// Determine verification level from branch profile
	currentBranch, _ := currentBranch(repoRoot)
	profile := DetectProfileFromBranch(currentBranch)
	verifyLevel := LevelForProfile(profile)

	// For OVAV's own repo (has go-runtime), keep legacy behavior
	ovavGoRuntime := repoRoot + "/go-runtime"
	isOVAVRepo := false
	if _, err := os.Stat(filepath.Join(ovavGoRuntime, "go.mod")); err == nil {
		isOVAVRepo = true
	}

	// Phase 1: Go Validation (if Go stack detected)
	if stack.HasGo() || isOVAVRepo {
		goDirs := stack.GoDirs()
		// OWS-v2: Apply scope filtering when changedFiles is provided
		if len(changedFiles) > 0 {
			goDirs = scope.ScopedGoDirs(repoRoot, goDirs, changedFiles)
			r.Scoped = true
		}
		if isOVAVRepo && len(goDirs) == 0 {
			// Legacy: OVAV repo uses go-runtime as the Go dir
			goDirs = []string{"go-runtime"}
		}

		for _, goDir := range goDirs {
			goRoot := filepath.Join(repoRoot, goDir)

			// 1a. go vet
			vetCmd := exec.Command("go", "vet", "./...")
			vetCmd.Dir = goRoot
			if out, err := vetCmd.CombinedOutput(); err != nil {
				r.GoVetPass = false
				issues = append(issues, fmt.Sprintf("go vet FAILED in %s: %s", goDir, truncateOutput(string(out), 200)))
			} else {
				r.GoVetPass = true
			}

			// 1b. gofmt
			fmtCmd := exec.Command("gofmt", "-l", ".")
			fmtCmd.Dir = goRoot
			out, _ := fmtCmd.Output()
			if strings.TrimSpace(string(out)) != "" {
				r.GofmtPass = false
				unformatted := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
				issues = append(issues, fmt.Sprintf("gofmt: %d unformatted file(s) in %s", unformatted, goDir))
			} else {
				r.GofmtPass = true
			}

			// 1c. go test -race (standard+ verification)
			// OWS-v2: quick mode skips go test (owd pre-merge gate — speed over full coverage)
			// When changedFiles provided, use AffectedPackages for scoped testing
			var testArgs []string
			if len(quick) > 0 && quick[0] {
				r.GoTestPass = true
			} else {
				// Build test args: always include -count=1
				testArgs = []string{"test", "-count=1"}
				if verifyLevel >= VerifyStandard {
					testArgs = append(testArgs, "-race")
				}
				// Scope to affected packages when changedFiles provided
				if len(changedFiles) > 0 {
					if affected := AffectedPackages(repoRoot, goDir, changedFiles); len(affected) > 0 {
						// Convert absolute package paths to relative paths for go test
						for _, pkg := range affected {
							rel, err := filepath.Rel(goRoot, pkg)
							if err == nil {
								testArgs = append(testArgs, rel)
							}
						}
						fmt.Printf("  🔬 scoped test: %d package(s)\n", len(affected))
					}
				}
				// Fallback to all packages if no scope or no affected packages
				if len(testArgs) == 2 {
					testArgs = append(testArgs, "./...")
				}
				testCmd := exec.Command("go", testArgs...)
				testCmd.Dir = goRoot
				if out, err := testCmd.CombinedOutput(); err != nil {
					r.GoTestPass = false
					issues = append(issues, fmt.Sprintf("go test FAILED in %s: %s", goDir, truncateOutput(string(out), 200)))
				} else {
					r.GoTestPass = true
				}
			}

			// 1d. Coverage gate (strict+ only)
			if verifyLevel >= VerifyStrict {
				coverFile := filepath.Join(os.TempDir(), "ovav-owv-coverage.out")
				coverCmd := exec.Command("go", "test", "-count=1", "-coverprofile="+coverFile, "./...")
				coverCmd.Dir = goRoot
				if _, err := coverCmd.CombinedOutput(); err == nil {
					totalCmd := exec.Command("go", "tool", "cover", "-func="+coverFile)
					totalCmd.Dir = goRoot
					if totalOut, err := totalCmd.Output(); err == nil {
						lines := strings.Split(strings.TrimSpace(string(totalOut)), "\n")
						if len(lines) > 0 {
							lastLine := lines[len(lines)-1]
							fields := strings.Fields(lastLine)
							if len(fields) >= 2 {
								pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
								if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
									r.CoveragePct = pct
									r.CoveragePass = pct >= 70.0
									if !r.CoveragePass {
										issues = append(issues, fmt.Sprintf("coverage %.1f%% below 70%% threshold in %s", pct, goDir))
									}
								}
							}
						}
					}
				}
			}

			// Only process first Go dir (primary)
			break
		}
	} else {
		// No Go detected — skip Go checks, don't fail
		r.GoVetPass = true
		r.GofmtPass = true
		r.GoTestPass = true
	}

	// Phase 2: OVAV Validators (if available)
	// Scope filtering: pass changedFiles so validators can skip irrelevant checks
	validateDir := ovavGoRuntime + "/internal/validators/cmd/validate"
	if info, err := os.Stat(validateDir); err == nil && info.IsDir() {
		valArgs := []string{"run", "."}
		if len(changedFiles) > 0 {
			valArgs = append(valArgs, "--changed-files", strings.Join(changedFiles, ","))
			valArgs = append(valArgs, "--root", repoRoot)
		}
		valCmd := exec.Command("go", valArgs...)
		valCmd.Dir = validateDir
		valOut, _ := valCmd.CombinedOutput()
		r.ValidateRan = true
		r.ValidatePass, r.ValidateFail = parseValidateOutput(string(valOut))
		r.ValidateTotal = r.ValidatePass + r.ValidateFail
		fmt.Print(string(valOut))
	}

	// Phase 3: Workspace Hygiene (always runs)
	hygiene := WorkspaceHygieneScan(repoRoot)
	r.HygieneClean = hygiene.Clean
	r.HygieneIssues = hygiene.TotalIssues
	if !hygiene.Clean {
		issues = append(issues, fmt.Sprintf("hygiene: %d issue(s) found", hygiene.TotalIssues))
		fmt.Print(hygiene.Report())
	}

	// Phase 4: Stack-specific validators
	for _, s := range stack.Stacks {
		if s.Type == StackTSReact || s.Type == StackTSNode || s.Type == StackTSVue {
			tsDir := filepath.Join(repoRoot, s.Dir)
			if _, err := os.Stat(filepath.Join(tsDir, "tsconfig.json")); err == nil {
				typeCmd := exec.Command("pnpm", "exec", "tsc", "--noEmit")
				typeCmd.Dir = tsDir
				if out, err := typeCmd.CombinedOutput(); err != nil {
					issues = append(issues, fmt.Sprintf("typescript typecheck FAILED in %s: %s", s.Dir, truncateOutput(string(out), 200)))
				}
			}
		}
		if s.Type == StackPython {
			pyDir := filepath.Join(repoRoot, s.Dir)
			ruffCmd := exec.Command("ruff", "check", ".")
			ruffCmd.Dir = pyDir
			if out, err := ruffCmd.CombinedOutput(); err != nil {
				issues = append(issues, fmt.Sprintf("ruff check FAILED in %s: %s", s.Dir, truncateOutput(string(out), 200)))
			}
		}
		if s.Type == StackRust {
			rustDir := filepath.Join(repoRoot, s.Dir)
			cargoCmd := exec.Command("cargo", "check")
			cargoCmd.Dir = rustDir
			if out, err := cargoCmd.CombinedOutput(); err != nil {
				issues = append(issues, fmt.Sprintf("cargo check FAILED in %s: %s", s.Dir, truncateOutput(string(out), 200)))
			}
		}
	}

	// Determine overall pass
	r.Passed = r.GoVetPass && r.GofmtPass && r.GoTestPass && r.HygieneClean
	if r.ValidateRan && r.ValidateFail > 0 {
		r.Passed = false
	}
	if len(issues) > 0 {
		r.Detail = strings.Join(issues, "; ")
	} else {
		r.Detail = "all checks passed"
	}

	// OWS-B6 FIX: Persist results so checkVerifiedGate() can read them.
	verifyDir := filepath.Join(repoRoot, ".ovav", "verify")
	if err := os.MkdirAll(verifyDir, 0755); err == nil {
		resultPath := filepath.Join(verifyDir, "last_result.json")
		data, _ := json.MarshalIndent(r, "", "  ")
		_ = os.WriteFile(resultPath, data, 0644)
	}

	return r, nil
}

// truncateOutput truncates a string to maxLen characters.
func truncateOutput(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// parseValidateOutput extracts pass/fail counts from validate CLI output.
// Looks for "Results: X passed, Y failed" pattern.
func parseValidateOutput(out string) (passed, failed int) {
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "passed") && strings.Contains(line, "failed") {
			// "── Results: 60 passed, 17 failed ──"
			line = strings.TrimPrefix(line, "── Results: ")
			line = strings.TrimSuffix(line, " ──")
			line = strings.TrimSpace(line)
			parts := strings.Split(line, ", ")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasSuffix(p, " passed") {
					fmt.Sscanf(p, "%d passed", &passed)
				}
				if strings.HasSuffix(p, " failed") {
					fmt.Sscanf(p, "%d failed", &failed)
				}
			}
			return
		}
	}
	return 0, 0
}
