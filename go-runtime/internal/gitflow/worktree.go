// Package gitflow — worktree resolution system.
// Uses `git worktree list --porcelain` as canonical source of truth.
// Makes OWD location-agnostic: works regardless of which worktree is CWD.
package gitflow

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolvedWorktree holds the result of worktree resolution.
type ResolvedWorktree struct {
	WorktreePath string // absolute path to the worktree directory
	Branch       string // short branch name e.g. "feature/sprint-1.7"
	MainRepoRoot string // main repo root (always set)
	IsWorktree   bool   // true if resolved to a worktree (not the main repo)
}

// porcelainEntry is a parsed entry from `git worktree list --porcelain`.
type porcelainEntry struct {
	Worktree string // absolute path to the worktree
	HEAD     string // commit hash
	Branch   string // full ref e.g. "refs/heads/feature/sprint-1.7"
}

// ResolveWorktree finds the worktree for a given branch.
// Uses git worktree list --porcelain as canonical source.
// Works regardless of CWD — no dependency on os.Getwd().
//
// branchHint can be:
//
//	""              → auto-detect from current HEAD
//	"feature/x"     → branch name (short or full)
//	"/path/to/wt"   → worktree path (validated against porcelain)
func ResolveWorktree(branchHint string) (*ResolvedWorktree, error) {
	// 1. List all worktree entries from porcelain
	entries, err := listWorktreeEntries()
	if err != nil {
		return nil, fmt.Errorf("resolve worktree: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("resolve worktree: no worktrees found")
	}

	// 2. Find main repo root from entries
	mainRepoRoot := findMainRepoRootFromEntries(entries)
	if mainRepoRoot == "" {
		return nil, fmt.Errorf("resolve worktree: cannot detect main repo root")
	}

	// 3. Resolve based on hint type
	if branchHint == "" {
		return resolveFromHEAD(entries, mainRepoRoot)
	}
	if looksLikeWorktreePath(branchHint) {
		return resolveFromPath(entries, branchHint, mainRepoRoot)
	}
	return resolveFromBranch(entries, branchHint, mainRepoRoot)
}

// findMainRepoRoot detects the main repo root from any path within the repo.
// Returns the absolute path to the main repo toplevel.
func findMainRepoRoot(anyPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = anyPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	commonDir := strings.TrimSpace(string(out))

	if commonDir == ".git" {
		// anyPath IS the main repo — resolve toplevel
		return gitToplevel(anyPath)
	}

	// Worktree: commonDir points to the .git dir in the main repo
	if filepath.IsAbs(commonDir) {
		// Absolute: /path/to/main/.git → /path/to/main
		return filepath.Dir(commonDir), nil
	}
	// Relative: resolve from anyPath
	absCommonDir := filepath.Join(anyPath, commonDir)
	absCommonDir = filepath.Clean(absCommonDir)
	return filepath.Dir(absCommonDir), nil
}

// parseWorktreeList parses git worktree list --porcelain output.
// Each entry has: worktree <path>, HEAD <hash>, branch <ref>.
// Entries are separated by blank lines.
func parseWorktreeList(porcelainOutput string) []porcelainEntry {
	var entries []porcelainEntry
	var current porcelainEntry
	inEntry := false

	for _, line := range strings.Split(porcelainOutput, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			// Flush previous entry
			if inEntry && current.Worktree != "" {
				entries = append(entries, current)
			}
			current = porcelainEntry{}
			current.Worktree = strings.TrimPrefix(line, "worktree ")
			inEntry = true
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch ")
		case line == "":
			// Blank line: flush current entry
			if inEntry && current.Worktree != "" {
				entries = append(entries, current)
				current = porcelainEntry{}
				inEntry = false
			}
		}
		// Other lines (bare, detached, prunable) are ignored
	}
	// Flush last entry (may not have trailing blank line)
	if inEntry && current.Worktree != "" {
		entries = append(entries, current)
	}
	return entries
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// listWorktreeEntries runs git worktree list --porcelain and parses the output.
func listWorktreeEntries() ([]porcelainEntry, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list --porcelain: %w", err)
	}
	return parseWorktreeList(string(out)), nil
}

// findMainRepoRootFromEntries identifies the main repo by checking git-common-dir.
// The main repo is the entry where git-common-dir returns ".git".
func findMainRepoRootFromEntries(entries []porcelainEntry) string {
	for _, e := range entries {
		cmd := exec.Command("git", "rev-parse", "--git-common-dir")
		cmd.Dir = e.Worktree
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(out)) == ".git" {
			return e.Worktree
		}
	}
	return ""
}

// resolveFromHEAD auto-detects the worktree from the current HEAD branch.
func resolveFromHEAD(entries []porcelainEntry, mainRepoRoot string) (*ResolvedWorktree, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("detect current branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return nil, fmt.Errorf("detached HEAD — specify a branch: owd <branch>")
	}

	return matchBranch(entries, branch, mainRepoRoot)
}

// resolveFromPath finds a worktree by its filesystem path.
func resolveFromPath(entries []porcelainEntry, pathHint string, mainRepoRoot string) (*ResolvedWorktree, error) {
	absPath, err := filepath.Abs(pathHint)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", pathHint, err)
	}
	absPath = filepath.Clean(absPath)

	for _, e := range entries {
		if e.Worktree == absPath {
			branch := strings.TrimPrefix(e.Branch, "refs/heads/")
			return &ResolvedWorktree{
				WorktreePath: e.Worktree,
				Branch:       branch,
				MainRepoRoot: mainRepoRoot,
				IsWorktree:   e.Worktree != mainRepoRoot,
			}, nil
		}
	}
	return nil, fmt.Errorf("path %q is not a registered worktree", absPath)
}

// resolveFromBranch finds a worktree by branch name.
// Supports: short name ("sprint-1.7"), partial ("feature/sprint-1.7"), full ref.
func resolveFromBranch(entries []porcelainEntry, branchHint string, mainRepoRoot string) (*ResolvedWorktree, error) {
	// 1. Exact match: short branch name
	fullRef := "refs/heads/" + branchHint
	for _, e := range entries {
		if e.Branch == fullRef {
			return entryToResolved(e, mainRepoRoot), nil
		}
	}

	// 2. Exact match: full ref passthrough
	for _, e := range entries {
		if e.Branch == branchHint {
			return entryToResolved(e, mainRepoRoot), nil
		}
	}

	// 3. Suffix match: "sprint-1.7" matches "feature/sprint-1.7"
	for _, e := range entries {
		short := strings.TrimPrefix(e.Branch, "refs/heads/")
		if strings.HasSuffix(short, "/"+branchHint) {
			return entryToResolved(e, mainRepoRoot), nil
		}
	}

	return nil, fmt.Errorf("no worktree found for branch %q", branchHint)
}

// matchBranch finds the entry matching a short branch name.
func matchBranch(entries []porcelainEntry, branch string, mainRepoRoot string) (*ResolvedWorktree, error) {
	fullRef := "refs/heads/" + branch
	for _, e := range entries {
		if e.Branch == fullRef {
			return entryToResolved(e, mainRepoRoot), nil
		}
	}
	return nil, fmt.Errorf("no worktree found for branch %q", branch)
}

// entryToResolved converts a porcelainEntry to a ResolvedWorktree.
func entryToResolved(e porcelainEntry, mainRepoRoot string) *ResolvedWorktree {
	branch := strings.TrimPrefix(e.Branch, "refs/heads/")
	return &ResolvedWorktree{
		WorktreePath: e.Worktree,
		Branch:       branch,
		MainRepoRoot: mainRepoRoot,
		IsWorktree:   e.Worktree != mainRepoRoot,
	}
}

// looksLikeWorktreePath returns true if the hint looks like a filesystem path.
func looksLikeWorktreePath(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.Contains(s, ".ovav/worktrees/")
}

// gitToplevel returns the absolute toplevel of the git repo at dir.
func gitToplevel(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return dir, nil // fallback: return dir as-is
	}
	return strings.TrimSpace(string(out)), nil
}
