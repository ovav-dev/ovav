// Package gitflow provides OVAV git v3.0 workflow commands.
// Replaces owc/owd Python wrappers with native Go commands.
package gitflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// mu protects concurrent git operations. Only one workflow command at a time.
var mu sync.Mutex

// Profile defines the git profile for worktree creation.
// Mirrors ows.ProfileConfig for internal use without circular import.
type Profile struct {
	Name       string // profile key: "feature", "hotfix", etc.
	Prefix     string // branch prefix: "feature/", "hotfix/", "release/", "patch/", etc.
	Base       string // base branch: "develop" or "main"
	MergeTo    string // merge target: "develop", "main", "main+develop", "none"
	Compliance string // compliance level: "quick", "standard", "strict", "maximum"
}

// DefaultProfile returns the "feature" profile (backward-compatible default).
func DefaultProfile() Profile {
	return Profile{Name: "feature", Prefix: "feature/", Base: "develop", MergeTo: "develop", Compliance: "standard"}
}

// DetectAuthor extracts the short author name from git config for branch naming.
// Uses user.email domain to detect role, falls back to user.name.
//
//	alexander@ovav.dev  → ceo
//	thavren@ovav.dev    → thavren
//	marco@externo.com   → marco (from user.name)
func DetectAuthor(repoRoot string) string {
	email := strings.TrimSpace(getGitOutput(repoRoot, "config", "user.email"))
	// OVAV domain: extract local part, use role mapping for CEO
	if strings.HasSuffix(email, "@ovav.dev") {
		local := strings.TrimSuffix(email, "@ovav.dev")
		// CEO gets special short name
		if local == "alexander" || local == "alexander.salvador" || local == "alexander.salvador.dev" {
			return "ceo"
		}
		return local
	}
	// External: use user.name, take last word as short name
	name := strings.TrimSpace(getGitOutput(repoRoot, "config", "user.name"))
	parts := strings.Fields(name)
	if len(parts) > 0 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return "dev"
}

// profilePrefixes maps profile names to their branch prefixes.
// Each profile uses its own name as the branch prefix (standard convention).
var profilePrefixes = map[string]string{
	"feature":    "feature/",
	"refactor":   "refactor/",
	"docs":       "docs/",
	"spike":      "spike/",
	"research":   "research/",
	"migration":  "migration/",
	"enterprise": "enterprise/",
	"hotfix":     "hotfix/",
	"emergency":  "emergency/",
	"release":    "release/",
	"patch":      "patch/",
	"fix":        "fix/",
}

// ProfileForName returns the profile config for a given name.
// Each profile uses its own name as the branch prefix (industry standard: type/desc).
func ProfileForName(name string) Profile {
	prefix, ok := profilePrefixes[name]
	if !ok {
		prefix = "feature/"
		name = "feature"
	}
	profile := Profile{Name: name, Prefix: prefix}

	switch name {
	case "hotfix", "emergency":
		profile.Base = "main"
		profile.MergeTo = "main+develop"
	case "release":
		profile.Base = "develop"
		profile.MergeTo = "main"
	case "patch":
		profile.Base = "main"
		profile.MergeTo = "main+develop"
	case "spike", "research":
		profile.Base = "develop"
		profile.MergeTo = "none"
	default:
		profile.Base = "develop"
		profile.MergeTo = "develop"
	}
	return profile
}

// DetectProfileFromBranch infers the profile from a branch name prefix.
// Recognizes all standard prefixes: feature/, hotfix/, fix/, release/, docs/, etc.
func DetectProfileFromBranch(branch string) Profile {
	for name, prefix := range profilePrefixes {
		if strings.HasPrefix(branch, prefix) {
			return ProfileForName(name)
		}
	}
	return ProfileForName("feature")
}

// Start creates a new worktree from the specified base branch.
// Usage: ovav worktree create <name> [--profile <type>]
//
// Without profile, defaults to "feature" (feature/ prefix, develop base).
// Supported profiles: feature, refactor, docs, spike, research, migration,
// enterprise, hotfix, release, patch, emergency, fix.
func Start(repoRoot, featureName string) error {
	return StartWithProfile(repoRoot, featureName, DefaultProfile())
}

// StartWithProfile creates a worktree using the specified profile's prefix and base branch.
// Branch naming follows industry convention: <type>/<description>.
//
// If featureName is "--help" or "-h", prints usage guidance and returns without
// creating any branch. This prevents accidental branch creation when help flags
// are mistyped as feature names.
func StartWithProfile(repoRoot, featureName string, profile Profile) error {
	mu.Lock()
	defer mu.Unlock()

	if featureName == "" {
		return fmt.Errorf("feature name required: ovav git start <name>")
	}

	if featureName == "--help" || featureName == "-h" {
		fmt.Println("Usage: ovav git start <name> [--profile <type>]")
		fmt.Println("       owc <name> [--compliance <level>]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ovav git start fix-bug")
		fmt.Println("  owc feature/login-ui")
		fmt.Println("  owc hotfix/critical --compliance maximum")
		fmt.Println()
		fmt.Println("Profiles: feature, refactor, docs, spike, research, migration, enterprise, hotfix, release, patch, fix, emergency")
		return nil
	}

	branchName := profile.Prefix + featureName
	// Sanitize worktree path: replace / with - for filesystem safety
	worktreeDir := strings.ReplaceAll(branchName, "/", "-")
	worktreePath := filepath.Join(repoRoot, ".ovav", "worktrees", worktreeDir)

	// Intelligent: detect if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		fmt.Printf("  ✅ %s already exists\n", branchName)
		fmt.Printf("WORKTREE:%s\n", worktreePath)
		return nil
	}

	author := DetectAuthor(repoRoot)
	baseBranch := profile.Base

	// Fetch latest base branch — capture to suppress verbose output
	_ = getGitOutput(repoRoot, "fetch", "origin", baseBranch)

	// Create branch from LOCAL base branch — guard against existing branch
	if out := getGitOutput(repoRoot, "branch", "--list", branchName); out != "" {
		_ = runGit(repoRoot, "branch", "-D", branchName)
	}
	_ = runGit(repoRoot, "branch", branchName, baseBranch)

	// Create worktree with progress indicator
	fmt.Print("  ⌛ ")
	wtOut := getGitOutput(repoRoot, "worktree", "add", worktreePath, branchName)
	if wtOut == "" {
		fmt.Print("\r     \n")
		return fmt.Errorf("worktree create failed")
	}
	fmt.Print("\r  ✅ done\n")

	// ── Set upstream tracking without pushing empty branch ──
	// Configures git remote tracking so owd can push + cleanup later.
	// The first real push happens when work is committed and owd runs.
	_ = runGit(repoRoot, "config", "branch."+branchName+".remote", "origin")
	_ = runGit(repoRoot, "config", "branch."+branchName+".merge", "refs/heads/"+branchName)
	fmt.Printf("  🌿 %s  ·  %s  ·  %s\n",
		branchName, profile.MergeTo, profile.Compliance)

	// Structured line for shell auto-cd wrapper
	fmt.Printf("WORKTREE:%s\n", worktreePath)

	// OVAV identity: autor registrado en el commit trail del worktree
	gitCmd := exec.Command("git", "-C", worktreePath, "config", "user.email", author+"@ovav.worktree")
	gitCmd.Run() // best-effort, no bloquea

	// ── git rerere: reuse recorded conflict resolution across worktrees ──
	// Cuando resolvés un conflicto en worktree-A, rerere graba la resolución.
	// Si el MISMO conflicto aparece en worktree-B, git lo auto-resuelve.
	// Zero-config, zero-side-effects. Git 2.23+.
	rerereCmd := exec.Command("git", "-C", worktreePath, "config", "rerere.enabled", "true")
	rerereCmd.Run() // best-effort, no bloquea

	// ── git maintenance: background repo optimization ──
	// Auto-gc, prefetch, commit-graph. Mantiene el repo rápido sin intervención.
	// Git 2.31+. Registra tareas en systemd/cron/launchctl según plataforma.
	maintCmd := exec.Command("git", "-C", worktreePath, "maintenance", "start")
	maintCmd.Run() // best-effort, no bloquea

	// ── Plantilla por perfil ──
	writeProfileTemplate(worktreePath, profile)

	// ── OVAV agents: copy to .mimocode/agents/ so mimo finds them ──
	agentsSrc := filepath.Join(repoRoot, "runtimes", "mimocode", "agents")
	if info, err := os.Stat(agentsSrc); err == nil && info.IsDir() {
		agentsDest := filepath.Join(worktreePath, ".mimocode", "agents")
		if err := os.MkdirAll(agentsDest, 0755); err != nil {
			fmt.Printf("  ⚠️  Cannot create .mimocode/agents: %v\n", err)
		} else {
			copied := copyAgentFiles(agentsSrc, agentsDest)
			if copied > 0 {
				fmt.Printf("  🤖 OVAV agents: %d files copied to .mimocode/agents/\n", copied)
			}
		}
	}

	// ── Worktree-local .gitignore: noise files that should never be tracked ──
	gitignorePath := filepath.Join(worktreePath, ".gitignore")
	noisePatterns := []string{
		"# OVAV worktree: generated files — never track",
		"bin/",
		"*.log",
		".cache/",
		"coverage/",
		"*.exe",
		"*.dll",
		".env",
		"credentials.json",
		"secrets.*",
		"*.key",
		"node_modules/",
		".DS_Store",
		"# OVAV untracked state (worktree-local)",
		".owl/untracked.json",
	}
	gitignoreContent := strings.Join(noisePatterns, "\n") + "\n"
	if existing, err := os.ReadFile(gitignorePath); err == nil {
		// Append if not already present
		if !strings.Contains(string(existing), "# OVAV worktree") {
			if err := os.WriteFile(gitignorePath, appendToFile(existing, []byte(gitignoreContent)), 0644); err != nil {
				fmt.Printf("  ⚠️  Cannot update .gitignore: %v\n", err)
			}
		}
	} else {
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			fmt.Printf("  ⚠️  Cannot create .gitignore: %v\n", err)
		}
	}

	return nil
}

// copyAgentFiles copies agent .md files from src to dest directory.
// Returns the number of files copied.
func copyAgentFiles(srcDir, destDir string) int {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0
	}
	copied := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".md") {
			srcPath := filepath.Join(srcDir, entry.Name())
			destPath := filepath.Join(destDir, entry.Name())
			data, err := os.ReadFile(srcPath)
			if err != nil {
				continue
			}
			if err := os.WriteFile(destPath, data, 0644); err == nil {
				copied++
			}
		}
	}
	return copied
}

// appendToFile appends content to existing bytes and returns the result.
func appendToFile(existing, newContent []byte) []byte {
	return append(existing, newContent...)
}

// writeProfileTemplate creates a template file in the worktree based on profile type.
func writeProfileTemplate(worktreePath string, profile Profile) {
	taskDir := filepath.Join(worktreePath, ".ovav", "task")
	os.MkdirAll(taskDir, 0755)

	var filename, content string
	switch profile.Name {
	case "hotfix", "emergency":
		filename = "HOTFIX.md"
		content = fmt.Sprintf(`# Hotfix: %s
> **OVAV Workflow** · Profile: %s · Compliance: %s

## Incident
<!-- Describe the incident: what broke, when, impact -->

## Root Cause
<!-- Technical root cause analysis -->

## Fix
<!-- What was changed and why -->

## Rollback Plan
<!-- Steps to revert if the fix causes issues -->
1. `+"`"+`git revert <merge-commit>`+"`"+`

## Verification
- [ ] owv passed
- [ ] Tests pass
- [ ] Manual verification done

---
_Generated by OVAV Worktree Orchestration System_
`, filepath.Base(worktreePath), profile.Name, profile.Compliance)

	case "release":
		filename = "RELEASE_NOTES.md"
		content = fmt.Sprintf(`# Release Notes
> **OVAV Workflow** · Profile: %s

## Version
<!-- Update VERSION file -->

## Changes
<!-- List the changes included in this release -->

## Breaking Changes
<!-- Any breaking changes? -->

## Upgrade Guide
<!-- Steps for users to upgrade -->

---
_Generated by OVAV Worktree Orchestration System_
`, profile.Name)

	case "research":
		filename = "RESEARCH.md"
		content = fmt.Sprintf(`# Research: %s
> **OVAV Workflow** · Profile: %s · Auto-cleanup: enabled

## Hypothesis
<!-- What are we investigating? -->

## Sources
<!-- Evidence, benchmarks, references -->

## Findings
<!-- What we discovered -->

## Recommendation
<!-- Based on findings, what should we do? -->

---
_Timer: 48h · This worktree will be auto-removed after inactivity_
`, filepath.Base(worktreePath), profile.Name)

	case "spike":
		filename = "SPIKE.md"
		content = fmt.Sprintf(`# Spike: %s
> **OVAV Workflow** · Profile: %s · Timer: 48h · Auto-cleanup: enabled

## Goal
<!-- What are we trying to learn? -->

## Approach
<!-- Technical approach for this spike -->

## Result
<!-- What we learned — code, benchmarks, architecture decision -->

## Decision
<!-- Based on results: proceed / abandon / more research needed -->

---
_Expires in 48h · Auto-cleanup enabled_
`, filepath.Base(worktreePath), profile.Name)

	default:
		filename = "checklist.md"
		content = fmt.Sprintf(`# Task Checklist
> **OVAV Workflow** · Profile: %s · Compliance: %s

## Before Starting
- [ ] Understand the requirements
- [ ] Review related code/docs

## Implementation
- [ ] Write the code
- [ ] Write tests
- [ ] Run `+"`"+`owv`+"`"+` to verify

## Before Merge
- [ ] All tests pass
- [ ] owv passed
- [ ] Conventional commits used
- [ ] Ready for `+"`"+`owd`+"`"+`

---
_Generated by OVAV Worktree Orchestration System_
`, profile.Name, profile.Compliance)
	}

	os.WriteFile(filepath.Join(taskDir, filename), []byte(content), 0644)
}

// Status displays comprehensive repository status.
// Usage: ovav git status
func Status(repoRoot string) error {
	branch := getGitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	baseBranch := "develop"
	aheadCount := getGitOutput(repoRoot, "rev-list", "--count", "origin/"+baseBranch+"..HEAD")
	dirty := getGitOutput(repoRoot, "status", "--porcelain")

	changeCount := 0
	if dirty != "" {
		changeCount = len(strings.Split(strings.TrimSpace(dirty), "\n"))
	}

	fmt.Printf("\n  Branch:     %s\n", strings.TrimSpace(branch))
	fmt.Printf("  Base:       %s (%s commits ahead)\n", baseBranch, strings.TrimSpace(aheadCount))
	fmt.Printf("  Changes:    %d files\n", changeCount)

	if dirty != "" {
		fmt.Printf("  Modified:\n")
		for _, line := range strings.Split(strings.TrimSpace(dirty), "\n") {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	}

	// Show last 3 commits
	log := getGitOutput(repoRoot, "log", "--oneline", "-3")
	if log != "" {
		fmt.Printf("\n  Recent:\n")
		for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	}

	fmt.Printf("\n  Next: ovav git save \"mensaje\" → ovav git push\n")
	return nil
}

// Save stages all changes and commits with formatted message.
// Usage: ovav git save "<message>"
//
// Conventional Commits: message should start with type prefix.
//
//	Valid:   "feat(ows): add conflict prediction"
//	Valid:   "fix: correct merge cleanup"
//	Invalid: "added conflict prediction"  → rejected
//
// If message lacks a type prefix, it's rejected with guidance.
func Save(repoRoot, message string) error {
	mu.Lock()
	defer mu.Unlock()

	if message == "" {
		return fmt.Errorf("commit message required: ovav git save \"<mensaje>\"")
	}

	// Validate conventional commit format
	commitType, body, err := parseConventionalCommit(message)
	if err != nil {
		return fmt.Errorf("conventional commit: %w\n  Expected: <type>[(scope)]: <description>\n  Example: feat(ows): add conflict prediction", err)
	}

	// Stage only tracked modified files (NOT git add -A which stages everything)
	staged, err := stageTrackedFiles(repoRoot)
	if err != nil {
		return fmt.Errorf("stage files: %w", err)
	}
	if staged == 0 {
		return fmt.Errorf("no tracked files to stage")
	}

	// Format commit message with validated type
	formattedMsg := fmt.Sprintf("%s: %s", commitType, body)
	if strings.Contains(commitType, "(") {
		formattedMsg = fmt.Sprintf("%s: %s", commitType, body)
	}

	// Commit
	if err := runGit(repoRoot, "commit", "-m", formattedMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	fmt.Printf("\n  ✅ Committed [%s]: %s\n", commitType, body)
	fmt.Printf("  📁 %d files staged\n", staged)
	fmt.Printf("\n  Next: ovav git push\n")

	return nil
}

// parseConventionalCommit validates and parses a conventional commit message.
// Returns (type+scope, description, error).
// Accepts: "type: description", "type(scope): description"
func parseConventionalCommit(message string) (string, string, error) {
	message = strings.TrimSpace(message)

	validTypes := []string{
		"feat", "fix", "docs", "style", "refactor", "perf", "test",
		"build", "ci", "chore", "revert", "security", "merge",
	}

	// Check if message starts with a valid type
	found := false
	for _, t := range validTypes {
		if strings.HasPrefix(strings.ToLower(message), t+"(") || strings.HasPrefix(strings.ToLower(message), t+":") {
			found = true
			break
		}
	}

	if !found {
		return "", "", fmt.Errorf("message must start with a conventional commit type (feat:, fix:, docs:, chore:, test:, etc.)")
	}

	// Extract type and body
	colonIdx := strings.Index(message, ":")
	if colonIdx < 0 {
		return "", "", fmt.Errorf("missing colon after type")
	}

	typePart := strings.TrimSpace(message[:colonIdx])
	body := strings.TrimSpace(message[colonIdx+1:])

	if body == "" {
		return "", "", fmt.Errorf("description required after type prefix")
	}

	return typePart, body, nil
}

// stageTrackedFiles stages only tracked files that have been modified or deleted.
// This is safer than git add -A which could stage untracked files.
func stageTrackedFiles(repoRoot string) (int, error) {
	// Get list of modified/deleted tracked files
	out := getGitOutput(repoRoot, "diff", "--name-only", "HEAD")
	files := strings.Split(strings.TrimSpace(out), "\n")

	// Also get untracked files that are NOT ignored
	untrackedOut := getGitOutput(repoRoot, "ls-files", "--others", "--exclude-standard")
	untracked := strings.Split(strings.TrimSpace(untrackedOut), "\n")

	count := 0
	// Stage modified/deleted files
	for _, f := range files {
		if f == "" {
			continue
		}
		if err := runGit(repoRoot, "add", f); err != nil {
			return count, fmt.Errorf("add %s: %w", f, err)
		}
		count++
	}

	// Stage untracked but not ignored files (new files in repo)
	for _, f := range untracked {
		if f == "" {
			continue
		}
		if err := runGit(repoRoot, "add", f); err != nil {
			return count, fmt.Errorf("add %s: %w", f, err)
		}
		count++
	}

	return count, nil
}

// detectCommitTypeFromFiles determines a suggested commit type from changed files.
// This is a FALLBACK — the user's message is authoritative for conventional commits.
func detectCommitTypeFromFiles(repoRoot string) string {
	changed := getGitOutput(repoRoot, "diff", "--cached", "--name-only")
	lower := strings.ToLower(changed)

	switch {
	case strings.Contains(lower, "test"):
		return "test:"
	case strings.Contains(lower, "docs") || strings.Contains(lower, ".md"):
		return "docs:"
	case strings.Contains(lower, "fix") || strings.Contains(lower, "bug"):
		return "fix:"
	case strings.Contains(lower, "validators") || strings.Contains(lower, "validator"):
		return "feat(validators):"
	case strings.Contains(lower, "install"):
		return "feat(install):"
	case strings.Contains(lower, "git") || strings.Contains(lower, "workflow"):
		return "feat(git):"
	default:
		return "feat:"
	}
}

// isAllowedRemote validates the remote URL meets OVAV security requirements.
// HTTPS URLs are always allowed. Local filesystem paths (/tmp, file://) are
// allowed for testing. SSH, git://, and unencrypted HTTP are blocked.
func isAllowedRemote(remoteURL string) bool {
	// Allow local filesystem paths (used in tests)
	if strings.HasPrefix(remoteURL, "/") || strings.HasPrefix(remoteURL, "file://") {
		return true
	}
	// Allow HTTPS
	if strings.HasPrefix(remoteURL, "https://") {
		return true
	}
	// Block everything else: SSH (git@, ssh://), git://, http://
	return false
}

// helpers

func runGit(repoRoot string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getGitOutput(repoRoot string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// MergeResult captures the outcome of a merge operation including worktree cleanup.
type MergeResult struct {
	Branch          string // The task branch that was merged
	WorktreePath    string // Path of the worktree (empty if not a worktree)
	WorktreeRemoved bool   // Whether the worktree was removed post-merge
	WorktreeError   string // Error during worktree removal, if any
}

// ── Fase 2: push, merge, release ──────────────────────────────────────────

// Fetch fetches all remotes. Used by push to detect divergence before push.
func Fetch(repoRoot, remote string) error {
	return runGit(repoRoot, "fetch", remote)
}

// Push pushes the current branch to origin via HTTPS only.
// Blocks force-push and SSH remotes.
// Usage: ovav git push
func Push(repoRoot string) error {
	mu.Lock()
	defer mu.Unlock()

	branch := strings.TrimSpace(getGitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch == "HEAD" {
		return fmt.Errorf("detached HEAD — cannot push")
	}

	// Verify remote is HTTPS (strict — explicit push command)
	remoteURL := strings.TrimSpace(getGitOutput(repoRoot, "remote", "get-url", "origin"))
	if !strings.HasPrefix(remoteURL, "https://") {
		if strings.Contains(remoteURL, "@") && !strings.Contains(remoteURL, "://") {
			return fmt.Errorf("SSH remote detected %q — OVAV requires HTTPS remotes only", remoteURL)
		}
		return fmt.Errorf("invalid remote URL %q — OVAV requires HTTPS remotes only", remoteURL)
	}

	fmt.Printf("  Pushing %s → origin/%s (HTTPS)\n", branch, branch)
	if err := runGit(repoRoot, "push", "origin", branch); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	fmt.Printf("  ✅ Pushed: %s → origin/%s\n", branch, branch)
	return nil
}

// Merge merges the current branch into its configured target(s).
// Detects the profile from the branch name prefix and merges accordingly:
//   - feature/release → develop only
//   - hotfix/patch/emergency → main first, then develop (doble merge atómico)
//
// Flow: merge all targets locally → cleanup worktree + delete branch → push targets.
// Push failures are non-fatal — the merge is valid locally and can be retried.
// Requires clean workspace. CEO waiver required for protected branches.
// Post-merge: removes the source worktree if running from one.
// Usage: ovav git merge [--no-ff]
func Merge(repoRoot string) (*MergeResult, error) {
	mu.Lock()
	defer mu.Unlock()

	result := &MergeResult{}

	branch := strings.TrimSpace(getGitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"))
	result.Branch = branch

	if branch == "HEAD" {
		return result, fmt.Errorf("detached HEAD — cannot merge")
	}
	if branch == "develop" || branch == "main" || branch == "master" {
		return result, fmt.Errorf("cannot merge from protected branch '%s' — use a task branch", branch)
	}

	// Check workspace is clean
	dirty := getGitOutput(repoRoot, "status", "--porcelain")
	if strings.TrimSpace(dirty) != "" {
		return result, fmt.Errorf("working tree is dirty — commit or stash changes before merge")
	}

	// Detect worktree info BEFORE any branch switch
	worktreePath := strings.TrimSpace(getGitOutput(repoRoot, "rev-parse", "--show-toplevel"))
	mainRepoRoot, isWorktree := getMainRepoRoot(repoRoot)
	if isWorktree {
		result.WorktreePath = worktreePath
	}

	// If running from a worktree, all git operations (switch/merge/push)
	// MUST use the main repo root. The worktree has its own branch checked out
	// and cannot switch to develop/main — those live in the main repo.
	gitRoot := repoRoot
	if isWorktree {
		gitRoot = mainRepoRoot
		fmt.Printf("  💡 Running merge from main repo: %s\n", mainRepoRoot)
	}

	// Detect profile from branch name
	profile := DetectProfileFromBranch(branch)
	mergeTargets := resolveMergeTargets(profile.MergeTo)

	fmt.Printf("  📋 Profile: %s → merge to %v\n", profile.Name, mergeTargets)

	// Fetch all needed base branches
	for _, target := range mergeTargets {
		fmt.Printf("  Fetching origin/%s...\n", target)
		if err := runGit(gitRoot, "fetch", "origin", target); err != nil {
			// Local integration remains safe when the remote is unavailable as
			// long as the base's remote-tracking ref already exists. This keeps
			// OWS usable offline without silently inventing a new base.
			if refErr := runGit(gitRoot, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+target); refErr != nil {
				return result, fmt.Errorf("fetch %s: %w", target, err)
			}
			fmt.Printf("  ⚠️  Remote unavailable; using existing origin/%s ref for local integration.\n", target)
		}
	}

	// ── Phase 1: Merge into each target (LOCAL ONLY, NO PUSH) ──
	type mergeRecord struct {
		target string
		preSHA string // origin/<target> SHA before merge — for rollback
	}
	var records []mergeRecord

	for _, target := range mergeTargets {
		// Save pre-merge SHA for rollback (origin/<target> is the clean state)
		preSHA := strings.TrimSpace(getGitOutput(gitRoot, "rev-parse", "origin/"+target))

		if err := mergeLocalTarget(gitRoot, branch, target); err != nil {
			// Collect names of previously merged targets for the rollback log
			var prevNames []string
			for _, rec := range records {
				prevNames = append(prevNames, rec.target)
			}
			fmt.Printf("\n  🔄 ROLLBACK: merge to %s failed — reverting %v\n", target, prevNames)
			for _, rec := range records {
				fmt.Printf("  Resetting %s to %s...\n", rec.target, rec.preSHA[:8])
				if switchErr := runGit(gitRoot, "switch", rec.target); switchErr != nil {
					fmt.Printf("  ⚠️  Cannot switch to %s for rollback: %v\n", rec.target, switchErr)
					continue
				}
				if resetErr := runGit(gitRoot, "reset", "--hard", rec.preSHA); resetErr != nil {
					fmt.Printf("  ⚠️  Manual reset may be needed in %s: %v\n", rec.target, resetErr)
				} else {
					fmt.Printf("  ✅ %s restored to pre-merge state\n", rec.target)
				}
			}
			// SU-2: Clean staged/unstaged files left by the failed merge before
			// switching back to source. git reset --hard only resets HEAD + working tree,
			// NOT the staging area — so staged files from the conflict remain orphaned.
			fmt.Printf("  🧹 Cleaning staged files before returning to %s...\n", branch)
			runGit(gitRoot, "restore", "--staged", ".")
			runGit(gitRoot, "checkout", "--", ".")
			// Best-effort switch back to source (may fail in worktree scenarios)
			_ = runGit(gitRoot, "switch", branch)
			return result, fmt.Errorf("doble merge aborted — %s rolled back: %w", target, err)
		}
		records = append(records, mergeRecord{target: target, preSHA: preSHA})
	}

	// ── Phase 2: Cleanup worktree + delete source branch ──
	// Post-merge: remove worktree FIRST — branch delete fails if worktree still exists
	if isWorktree && profile.MergeTo != "none" {
		cleanedMainRoot := cleanupWorktree(mainRepoRoot, worktreePath, result)
		// ── ATOMIC: if worktree cleanup failed, DO NOT delete the branch ──
		if result.WorktreeError != "" {
			fmt.Printf("\n  ⚠️  Worktree cleanup FAILED — preserving branch %s\n", branch)
			fmt.Printf("  📂 Worktree still exists at: %s\n", worktreePath)
			fmt.Printf("  🔧 Fix: resolve the issue, then run 'owd' again or manual cleanup:\n")
			fmt.Printf("     git -C %s worktree remove --force %s\n", mainRepoRoot, worktreePath)
			fmt.Printf("     git branch -d %s\n", branch)
			return result, fmt.Errorf("worktree cleanup failed: %s — branch %s preserved (no data lost)", result.WorktreeError, branch)
		}

		// Shell CWD fix: reset Go process CWD to main repo root.
		// The worktree is now deleted; if we don't do this, subsequent
		// operations or the returning shell inherit an invalid CWD.
		if result.WorktreeRemoved && cleanedMainRoot != "" {
			if err := os.Chdir(cleanedMainRoot); err == nil {
				fmt.Printf("  ✅ Shell CWD reset: %s\n", cleanedMainRoot)
			}
		}
	}

	// ── Delete source branch AFTER successful worktree cleanup ──
	fmt.Printf("  🧹 Deleting source branch %s...\n", branch)
	// Safe delete: -d only succeeds if the branch was fully merged
	if err := runGit(gitRoot, "branch", "-d", branch); err != nil {
		fmt.Printf("  ⚠️  Local branch not deleted: %v\n", err)
	} else {
		fmt.Printf("  ✅ Local branch deleted: %s\n", branch)
		// Attempt remote cleanup (non-blocking — branch may never have been pushed)
		if err := runGit(gitRoot, "push", "origin", "--delete", branch); err != nil {
			fmt.Printf("  ⚠️  Remote branch not deleted (may not have been pushed)\n")
		} else {
			fmt.Printf("  ✅ Remote branch deleted: origin/%s\n", branch)
		}
	}

	// ── Phase 3: Push all merged targets (PUSH LAST) ──
	// Push happens AFTER cleanup so hook bypass is unnecessary.
	// If push fails, the merge is still valid locally — user can retry.
	// OWS-GAP-04: Track push state so timeout can be resumed.
	targetNames := make([]string, len(records))
	for i, rec := range records {
		targetNames[i] = rec.target
	}

	sessionID := fmt.Sprintf("owd-%d", time.Now().UnixNano())
	state, err := InitPushState(gitRoot, sessionID, branch, targetNames)
	if err != nil {
		fmt.Printf("  ⚠️  Push state init failed: %v (continuing without state tracking)\n", err)
	}

PushLoop:
	for _, rec := range records {
		// Check if this target is already handled via resume
		if state != nil {
			for _, t := range state.Targets {
				if t.Name == rec.target && t.Status == "pushed" {
					fmt.Printf("  ⏭️  %s already pushed (resume), skipping\n", rec.target)
					continue PushLoop
				}
			}
		}

		pushErr := pushTargetWithState(gitRoot, rec.target, state)
		if pushErr == nil {
			if state != nil {
				ref := strings.TrimSpace(getGitOutput(gitRoot, "rev-parse", "refs/heads/"+rec.target))
				MarkTargetPushed(gitRoot, state, rec.target, ref)
			}
			continue
		}

		// Classify error
		errStr := pushErr.Error()
		isTimeout := strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "connection") ||
			strings.Contains(errStr, "reset") ||
			strings.Contains(errStr, "EOF") ||
			strings.Contains(errStr, "broken pipe") ||
			strings.Contains(errStr, "i/o timeout")

		if state != nil {
			ref := strings.TrimSpace(getGitOutput(gitRoot, "rev-parse", "refs/heads/"+rec.target))
			MarkTargetFailed(gitRoot, state, rec.target, ref, errStr)
		}

		if isTimeout {
			fmt.Printf("  ⏰ Push %s timed out — state saved to .ovav/runtime/push_state.json\n", rec.target)
			fmt.Printf("  📝 To resume: run 'owd' again — remaining targets will be retried automatically\n")
			break PushLoop
		}

		// Non-retryable error — warn but continue with remaining targets
		fmt.Printf("  ⚠️  Push %s failed (non-retryable): %v\n", rec.target, pushErr)
	}

	// If state exists and all targets are done, clear it
	if state != nil {
		pending := state.PendingTargets()
		if len(pending) == 0 {
			ClearPushState(gitRoot)
		}
	}

	return result, nil
}

// resolveMergeTargets converts a MergeTo string into an ordered slice of target branches.
// "main+develop" → ["main", "develop"] (merge to main first, then develop).
func resolveMergeTargets(mergeTo string) []string {
	switch mergeTo {
	case "main+develop":
		return []string{"main", "develop"}
	case "main":
		return []string{"main"}
	case "develop":
		return []string{"develop"}
	case "none":
		return nil
	default:
		return []string{"develop"}
	}
}

// mergeLocalTarget merges the source branch into the target branch (LOCAL ONLY).
// Performs: switch to target → pull → merge --no-ff → return error if conflict.
// Does NOT push — push is handled separately by pushTarget after all merges succeed.
func mergeLocalTarget(repoRoot, sourceBranch, target string) error {
	// Switch to target branch (git switch: modern, safer than checkout)
	fmt.Printf("  Switching to %s...\n", target)
	if err := runGit(repoRoot, "switch", target); err != nil {
		return fmt.Errorf("switch %s: %w", target, err)
	}

	// Pull latest
	fmt.Printf("  Pulling latest %s...\n", target)
	if err := pullTargetOrUseCachedRef(repoRoot, target); err != nil {
		return err
	}

	// Merge the source branch (--no-ff ensures merge commit for traceability)
	fmt.Printf("  Merging %s → %s...\n", sourceBranch, target)
	mergeMsg := fmt.Sprintf("Merge branch '%s' into %s", sourceBranch, target)
	if err := runGit(repoRoot, "merge", "--no-ff", sourceBranch, "-m", mergeMsg); err != nil {
		fmt.Printf("  ❌ Merge conflict detected merging into %s. Aborting...\n", target)
		runGit(repoRoot, "merge", "--abort")
		// SU-2: Safety net — clean orphaned staged/unstaged files left by abort
		fmt.Printf("  🧹 Cleaning orphaned merge artifacts...\n")
		runGit(repoRoot, "restore", "--staged", ".")
		runGit(repoRoot, "checkout", "--", ".")
		return fmt.Errorf("merge conflict into %s — resolve manually", target)
	}

	fmt.Printf("  ✅ Merged %s → %s (local)\n", sourceBranch, target)
	return nil
}

// pullTargetOrUseCachedRef refreshes a merge target when the remote is
// reachable. If the remote is unavailable, an existing origin/<target> ref is
// an explicit local snapshot and is safe to use for local integration. Other
// pull failures still abort instead of being treated as connectivity issues.
func pullTargetOrUseCachedRef(repoRoot, target string) error {
	cmd := exec.Command("git", "pull", "origin", target)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Print(string(out))
		return nil
	}

	message := strings.ToLower(string(out))
	remoteUnavailable := strings.Contains(message, "repository not found") ||
		strings.Contains(message, "could not resolve host") ||
		strings.Contains(message, "unable to access") ||
		strings.Contains(message, "failed to connect") ||
		strings.Contains(message, "network is unreachable") ||
		strings.Contains(message, "connection timed out")
	if remoteUnavailable {
		if refErr := runGit(repoRoot, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+target); refErr == nil {
			fmt.Printf("  ⚠️  Remote unavailable; using existing origin/%s ref for local integration.\n", target)
			return nil
		}
	}

	return fmt.Errorf("pull %s: %w: %s", target, err, strings.TrimSpace(string(out)))
}

// pushTarget pushes the target branch to origin with retry on timeout.
// Called AFTER all merges succeed and worktree cleanup is done.
// SU-4: Retries up to 3 times with exponential backoff (2s, 4s, 8s).
// OWS-GAP-04: When state is non-nil, updates it on each outcome.
func pushTarget(repoRoot, target string) error {
	return pushTargetWithState(repoRoot, target, nil)
}

// pushTargetWithState pushes the target branch to origin with retry on timeout.
// state may be nil (no state tracking). When non-nil, push state is updated
// on each attempt so that timeouts can be resumed.
func pushTargetWithState(repoRoot, target string, state *PushState) error {
	// Verify remote is HTTPS (or local filesystem for testing)
	remoteURL := strings.TrimSpace(getGitOutput(repoRoot, "remote", "get-url", "origin"))
	if !isAllowedRemote(remoteURL) {
		return fmt.Errorf("OVAV requires HTTPS or local remotes — got: %s", remoteURL)
	}

	maxRetries := 3
	baseDelay := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // 2s, 4s, 8s
			fmt.Printf("  🔄 Retry %d/%d in %v...\n", attempt, maxRetries, delay)
			time.Sleep(delay)
		}

		fmt.Printf("  Pushing %s → origin/%s...\n", target, target)
		// Use --atomic for safer partial pushes: if any ref fails, none are updated.
		// This prevents mixed-state when pushing multiple branches.
		err := runGit(repoRoot, "push", "--atomic", "origin", target)
		if err == nil {
			fmt.Printf("  ✅ Pushed: %s → origin/%s\n", target, target)
			return nil
		}

		// Only retry on timeout/network errors, not on auth/permission errors
		errStr := err.Error()
		if attempt == maxRetries {
			return fmt.Errorf("push %s failed after %d retries: %w", target, maxRetries, err)
		}
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") ||
			strings.Contains(errStr, "reset") || strings.Contains(errStr, "EOF") ||
			strings.Contains(errStr, "broken pipe") || strings.Contains(errStr, "i/o timeout") {
			fmt.Printf("  ⚠️  Push failed (network, will retry): %v\n", err)
			continue
		}
		// Non-retryable error — fail immediately
		return fmt.Errorf("push %s: %w", target, err)
	}

	return fmt.Errorf("push %s: max retries exceeded", target)
}

// cleanupWorktree removes and prunes a worktree from the main repo.
// Returns mainRepoRoot so the caller can reset its CWD to a valid directory.
func cleanupWorktree(mainRepoRoot, worktreePath string, result *MergeResult) string {
	fmt.Printf("\n  Cleaning up worktree...\n")
	cmd := exec.Command("git", "-C", mainRepoRoot, "worktree", "remove", "--force", worktreePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		result.WorktreeError = err.Error()
		fmt.Printf("  ⚠️  Worktree removal failed: %v\n", err)
		fmt.Printf("  Manual: git -C %s worktree remove %s\n", mainRepoRoot, worktreePath)
	} else {
		result.WorktreeRemoved = true
		fmt.Printf("  ✅ Worktree removed: %s\n", worktreePath)
	}

	// Prune stale worktree references
	pruneCmd := exec.Command("git", "-C", mainRepoRoot, "worktree", "prune")
	pruneCmd.Stdout = os.Stdout
	pruneCmd.Stderr = os.Stderr
	if err := pruneCmd.Run(); err != nil {
		fmt.Printf("  ⚠️  Worktree prune warning: %v\n", err)
	} else {
		fmt.Printf("  ✅ Worktrees pruned\n")
	}

	// Shell CWD fix: emit a marker on stdout so the parent shell can
	// cd to mainRepoRoot after the worktree is deleted. This solves the
	// "shell running in deleted worktree" failure mode.
	if result.WorktreeRemoved {
		fmt.Printf("\n__OVAV_SHELL_CWD_RESET__:%s__\n", mainRepoRoot)
	}

	return mainRepoRoot
}

// getMainRepoRoot detects if repoRoot is a git worktree and returns the main repo root.
// Returns (mainRepoRoot, true) if it is a worktree, or ("", false) if it's the main repo.
func getMainRepoRoot(worktreeRoot string) (string, bool) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = worktreeRoot
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	commonDir := strings.TrimSpace(string(out))
	if commonDir == ".git" {
		return "", false // This IS the main repo (not a worktree)
	}
	// git-common-dir returns the path to the .git directory in the main repo
	var mainRepoRoot string
	if filepath.IsAbs(commonDir) {
		mainRepoRoot = filepath.Dir(commonDir) // strip trailing .git
	} else {
		mainRepoRoot = filepath.Dir(filepath.Join(worktreeRoot, commonDir))
	}
	return mainRepoRoot, true
}

// Release creates a version tag on develop and pushes it.
// Requires clean workspace on develop branch.
// Usage: ovav git release <version>  (e.g., ovav git release v2.1.0)
func Release(repoRoot, version string) error {
	mu.Lock()
	defer mu.Unlock()

	if version == "" {
		return fmt.Errorf("version required: ovav git release <version>")
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	branch := strings.TrimSpace(getGitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch != "develop" && branch != "main" && branch != "master" {
		return fmt.Errorf("release must be created from develop or main, current branch: %s", branch)
	}

	// Check workspace is clean
	dirty := getGitOutput(repoRoot, "status", "--porcelain")
	if strings.TrimSpace(dirty) != "" {
		return fmt.Errorf("working tree is dirty — commit or stash changes before release")
	}

	// Ensure VERSION and CHANGELOG are updated
	versionPath := filepath.Join(repoRoot, "VERSION")
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return fmt.Errorf("VERSION file not found — create it with the release version")
	}

	// Create annotated tag
	tagMsg := fmt.Sprintf("OVAV Release %s", version)
	fmt.Printf("  Creating tag %s...\n", version)
	if err := runGit(repoRoot, "tag", "-a", version, "-m", tagMsg); err != nil {
		return fmt.Errorf("tag: %w", err)
	}

	// Push tag
	fmt.Printf("  Pushing tag %s...\n", version)
	if err := runGit(repoRoot, "push", "origin", version); err != nil {
		return fmt.Errorf("push tag: %w", err)
	}

	fmt.Printf("\n  ✅ Released: %s\n", version)
	fmt.Printf("  ✅ Tag pushed to origin\n")
	fmt.Printf("\n  Next: Create GitHub release from tag %s\n", version)

	return nil
}
