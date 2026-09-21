// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
package ows

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ── Hygiene Report ───────────────────────────────────────────────────────────

// HygieneResult holds the results of a workspace hygiene scan.
// All findings are advisory unless Blocking=true.
type HygieneResult struct {
	UntrackedFiles      []string         `json:"untracked_files,omitempty"`
	UnstagedModified    []string         `json:"unstaged_modified,omitempty"`
	DirtyAfterMerge     []string         `json:"dirty_after_merge,omitempty"`
	StaleLockFiles      []StaleLock      `json:"stale_locks,omitempty"`
	BrokenSymlinks      []string         `json:"broken_symlinks,omitempty"`
	OrphanWorktreeDirs  []string         `json:"orphan_worktrees,omitempty"`
	GeneratedFileDrift  []GeneratedDrift `json:"generated_drift,omitempty"`
	LargeUntrackedFiles []LargeFile      `json:"large_untracked,omitempty"`
	GitConfigWarnings   []string         `json:"git_config_warnings,omitempty"`
	AuditTrailWarning   string           `json:"audit_trail_warning,omitempty"`
	BlockingIssues      int              `json:"blocking_issues"`
	WarningIssues       int              `json:"warning_issues"`
	TotalIssues         int              `json:"total_issues"`
	Clean               bool             `json:"clean"`
}

type StaleLock struct {
	Path    string `json:"path"`
	Age     string `json:"age"`
	Expired bool   `json:"expired"`
}

type GeneratedDrift struct {
	File      string `json:"file"`
	Canonical string `json:"canonical"`
	Detail    string `json:"detail"`
}

type LargeFile struct {
	Path string `json:"path"`
	Size string `json:"size"`
	Ext  string `json:"ext"`
}

// ── Check 1: Untracked files ────────────────────────────────────────────────

func checkUntrackedFiles(repoRoot string) []string {
	return gitLines(repoRoot, "ls-files", "--others", "--exclude-standard")
}

// ── Check 2: Modified unstaged files ─────────────────────────────────────────

func checkUnstagedModified(repoRoot string) []string {
	return gitLines(repoRoot, "diff", "--name-only")
}

// ── Check 3: Dirty files after merge (files that belong to NO current worktree) ──

func checkDirtyAfterMerge(repoRoot string) []string {
	// Files modified/unstaged in the repo root that are not part of any
	// active worktree — likely forgotten changes from a previous session.
	unstaged := checkUnstagedModified(repoRoot)
	untracked := checkUntrackedFiles(repoRoot)

	// Get list of active worktree paths
	activeWorktrees := make(map[string]bool)
	lines := gitLines(repoRoot, "worktree", "list", "--porcelain")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			p := strings.TrimPrefix(line, "worktree ")
			activeWorktrees[p] = true
		}
	}

	var dirty []string
	for _, f := range append(unstaged, untracked...) {
		fullPath := filepath.Join(repoRoot, f)
		// Check if this file lives inside an active worktree
		inActiveWorktree := false
		for wt := range activeWorktrees {
			if strings.HasPrefix(fullPath, wt+string(filepath.Separator)) {
				inActiveWorktree = true
				break
			}
		}
		if !inActiveWorktree {
			dirty = append(dirty, f)
		}
	}
	return dirty
}

// ── Check 4: Stale lock files ────────────────────────────────────────────────

func checkStaleLocks(repoRoot string) []StaleLock {
	locksDir := filepath.Join(repoRoot, ".ovav", "locks")
	entries, err := os.ReadDir(locksDir)
	if err != nil {
		return nil
	}

	var stale []StaleLock
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		age := time.Since(info.ModTime())
		expired := age > 24*time.Hour
		if expired {
			stale = append(stale, StaleLock{
				Path:    filepath.Join(".ovav", "locks", e.Name()),
				Age:     age.Round(time.Minute).String(),
				Expired: true,
			})
		}
	}
	return stale
}

// ── Check 5: Broken symlinks ─────────────────────────────────────────────────

func checkBrokenSymlinks(repoRoot string) []string {
	var broken []string
	filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip .git, node_modules, .ovav/worktrees, .mimocode (runtime workspace)
		rel, _ := filepath.Rel(repoRoot, path)
		if strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, "node_modules") ||
			strings.HasPrefix(rel, ".ovav/worktrees") || strings.HasPrefix(rel, ".mimocode") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil
			}
			// Resolve relative to symlink location
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}
			if _, err := os.Stat(target); os.IsNotExist(err) {
				broken = append(broken, rel)
			}
		}
		return nil
	})
	return broken
}

// ── Check 6: Orphaned worktree directories ───────────────────────────────────

func checkOrphanWorktreeDirs(repoRoot string) []string {
	wtDir := filepath.Join(repoRoot, ".ovav", "worktrees")
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		return nil
	}

	// Get git-registered worktrees
	active := make(map[string]bool)
	lines := gitLines(repoRoot, "worktree", "list", "--porcelain")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			p := strings.TrimPrefix(line, "worktree ")
			active[filepath.Base(p)] = true
		}
	}

	var orphaned []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !active[e.Name()] {
			// Check if directory has actual content (not just empty)
			dirPath := filepath.Join(wtDir, e.Name())
			if hasContent(dirPath) {
				orphaned = append(orphaned, filepath.Join(".ovav", "worktrees", e.Name()))
			}
		}
	}
	return orphaned
}

// ── Check 7: Generated file drift ────────────────────────────────────────────

var generatedPaths = map[string]string{
	"runtimes/opencode/agents":    "ovav/agents/*.yaml (convert engine)",
	"runtimes/claude-code/agents": "ovav/agents/*.yaml (convert engine)",
	"runtimes/cursor/rules":       "ovav/agents/*.yaml (convert engine)",
	"clients/opencode/plugins":    ".ovav/visual/ (project sync)",
	"clients/opencode/agents":     "runtimes/opencode/agents/ (symlink)",
	".opencode/themes":            ".ovav/visual/theme/ (project sync)",
}

func checkGeneratedFileDrift(repoRoot string) []GeneratedDrift {
	var drift []GeneratedDrift

	for genPath, canonicalDesc := range generatedPaths {
		fullPath := filepath.Join(repoRoot, genPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		// Check for unstaged modifications in generated directories
		unstaged := checkUnstagedModified(repoRoot)
		for _, f := range unstaged {
			if strings.HasPrefix(f, genPath+"/") || f == genPath {
				if info.IsDir() {
					drift = append(drift, GeneratedDrift{
						File:      f,
						Canonical: canonicalDesc,
						Detail:    "Regenerable — run 'ovav project sync' to restore",
					})
				}
			}
		}

		// Check for untracked files in generated directories
		untracked := checkUntrackedFiles(repoRoot)
		for _, f := range untracked {
			if strings.HasPrefix(f, genPath+"/") {
				drift = append(drift, GeneratedDrift{
					File:      f,
					Canonical: canonicalDesc,
					Detail:    "Untracked generated file — may be stale or orphaned",
				})
			}
		}
	}
	return drift
}

// ── Check 8: Large untracked files ───────────────────────────────────────────

var largeFileExts = map[string]bool{
	".zip": true, ".tar": true, ".gz": true, ".exe": true,
	".bin": true, ".dll": true, ".so": true, ".dylib": true,
	".pdf": true, ".mp4": true, ".mov": true, ".png": true,
	".jpg": true, ".wasm": true,
}

func checkLargeUntracked(repoRoot string) []LargeFile {
	untracked := checkUntrackedFiles(repoRoot)
	var large []LargeFile

	for _, f := range untracked {
		ext := strings.ToLower(filepath.Ext(f))
		if !largeFileExts[ext] {
			continue
		}
		info, err := os.Stat(filepath.Join(repoRoot, f))
		if err != nil {
			continue
		}
		if info.Size() > 1024*1024 { // >1MB
			size := fmt.Sprintf("%.1f MB", float64(info.Size())/(1024*1024))
			large = append(large, LargeFile{
				Path: f,
				Size: size,
				Ext:  ext,
			})
		}
	}
	return large
}

// ── Check 9: Git config health ───────────────────────────────────────────────

func checkGitConfig(repoRoot string) []string {
	var warnings []string

	email := strings.TrimSpace(gitOutput(repoRoot, "config", "--local", "user.email"))
	if email == "" {
		warnings = append(warnings, "git user.email not configured locally — commits will lack attribution")
	}

	name := strings.TrimSpace(gitOutput(repoRoot, "config", "--local", "user.name"))
	if name == "" {
		warnings = append(warnings, "git user.name not configured locally — commits will lack attribution")
	}

	return warnings
}

// ── Check 10: Audit trail hygiene ────────────────────────────────────────────

func checkAuditTrail(repoRoot string) string {
	// .ovav/audit/ should be in .gitignore to prevent trail.jsonl leaks
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return ""
	}
	content := string(data)
	if !strings.Contains(content, ".ovav/audit/") && !strings.Contains(content, ".ovav/audit") {
		return ".ovav/audit/ not in .gitignore — trail.jsonl may leak to develop"
	}
	return ""
}

// ── Main Scan ────────────────────────────────────────────────────────────────

// WorkspaceHygieneScan runs all workspace hygiene checks against the repository.
// Returns a HygieneResult with all findings categorized and prioritized.
func WorkspaceHygieneScan(repoRoot string) *HygieneResult {
	r := &HygieneResult{Clean: true}

	// 1. Untracked files (not in gitignore, never staged)
	r.UntrackedFiles = checkUntrackedFiles(repoRoot)
	r.WarningIssues += len(r.UntrackedFiles)

	// 2. Modified unstaged files (risk of loss on checkout)
	r.UnstagedModified = checkUnstagedModified(repoRoot)
	r.WarningIssues += len(r.UnstagedModified)

	// 3. Dirty files after merge (orphaned from previous worktrees)
	r.DirtyAfterMerge = checkDirtyAfterMerge(repoRoot)
	r.WarningIssues += len(r.DirtyAfterMerge)

	// 4. Stale lock files (past TTL)
	r.StaleLockFiles = checkStaleLocks(repoRoot)
	r.WarningIssues += len(r.StaleLockFiles)

	// 5. Broken symlinks
	r.BrokenSymlinks = checkBrokenSymlinks(repoRoot)
	r.WarningIssues += len(r.BrokenSymlinks)

	// 6. Orphaned worktree directories
	r.OrphanWorktreeDirs = checkOrphanWorktreeDirs(repoRoot)
	r.WarningIssues += len(r.OrphanWorktreeDirs)

	// 7. Generated file drift
	r.GeneratedFileDrift = checkGeneratedFileDrift(repoRoot)
	r.WarningIssues += len(r.GeneratedFileDrift)

	// 8. Large untracked files (blocking — should never be committed)
	r.LargeUntrackedFiles = checkLargeUntracked(repoRoot)
	r.BlockingIssues += len(r.LargeUntrackedFiles)

	// 9. Git config health
	r.GitConfigWarnings = checkGitConfig(repoRoot)
	r.WarningIssues += len(r.GitConfigWarnings)

	// 10. Audit trail hygiene
	r.AuditTrailWarning = checkAuditTrail(repoRoot)
	if r.AuditTrailWarning != "" {
		r.WarningIssues++
	}

	r.TotalIssues = r.BlockingIssues + r.WarningIssues
	if r.TotalIssues > 0 {
		r.Clean = false
	}
	return r
}

// Report prints a human-readable hygiene report.
func (r *HygieneResult) Report() string {
	if r.Clean {
		return "✅ Workspace hygiene — clean"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🧹 Workspace Hygiene — %d issue(s) [%d blocking, %d warnings]\n",
		r.TotalIssues, r.BlockingIssues, r.WarningIssues))
	b.WriteString(strings.Repeat("─", 55) + "\n")

	if len(r.LargeUntrackedFiles) > 0 {
		b.WriteString("\n🚨 LARGE UNTRACKED FILES (blocking):\n")
		for _, lf := range r.LargeUntrackedFiles {
			b.WriteString(fmt.Sprintf("   %s (%s)\n", lf.Path, lf.Size))
		}
	}

	if len(r.DirtyAfterMerge) > 0 {
		b.WriteString("\n⚠️  DIRTY FILES (outside worktrees):\n")
		for _, f := range r.DirtyAfterMerge {
			b.WriteString(fmt.Sprintf("   %s\n", f))
		}
	}

	if len(r.UnstagedModified) > 0 {
		b.WriteString("\n⚠️  MODIFIED UNSTAGED (risk of loss):\n")
		for _, f := range r.UnstagedModified {
			b.WriteString(fmt.Sprintf("   %s\n", f))
		}
	}

	if len(r.GeneratedFileDrift) > 0 {
		b.WriteString("\n⚠️  GENERATED FILE DRIFT (regenerable):\n")
		for _, gd := range r.GeneratedFileDrift {
			b.WriteString(fmt.Sprintf("   %s ← %s\n     %s\n", gd.File, gd.Canonical, gd.Detail))
		}
	}

	if len(r.StaleLockFiles) > 0 {
		b.WriteString("\n⚠️  STALE LOCKS (expired >24h):\n")
		for _, sl := range r.StaleLockFiles {
			b.WriteString(fmt.Sprintf("   %s — age %s\n", sl.Path, sl.Age))
		}
	}

	if len(r.BrokenSymlinks) > 0 {
		b.WriteString("\n⚠️  BROKEN SYMLINKS:\n")
		for _, s := range r.BrokenSymlinks {
			b.WriteString(fmt.Sprintf("   %s\n", s))
		}
	}

	if len(r.OrphanWorktreeDirs) > 0 {
		b.WriteString("\n⚠️  ORPHAN WORKTREE DIRS:\n")
		for _, d := range r.OrphanWorktreeDirs {
			b.WriteString(fmt.Sprintf("   %s\n", d))
		}
	}

	if len(r.GitConfigWarnings) > 0 {
		b.WriteString("\n⚠️  GIT CONFIG:\n")
		for _, w := range r.GitConfigWarnings {
			b.WriteString(fmt.Sprintf("   %s\n", w))
		}
	}

	if r.AuditTrailWarning != "" {
		b.WriteString(fmt.Sprintf("\n⚠️  AUDIT TRAIL: %s\n", r.AuditTrailWarning))
	}

	if len(r.UntrackedFiles) > 0 {
		b.WriteString(fmt.Sprintf("\n💡 Untracked files: %d (use 'ovav hygiene --verbose' to list)\n", len(r.UntrackedFiles)))
	}

	return b.String()
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// gitLines runs a git command and returns output split by newlines.
func gitLines(repoRoot string, args ...string) []string {
	out := gitOutput(repoRoot, args...)
	if out == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// gitOutput runs git -C repoRoot <args> and returns stdout.
func gitOutput(repoRoot string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// hasContent returns true if a directory contains any files or subdirectories.
func hasContent(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// blockedPaths returns directories to skip during file walking.
var blockedPaths = regexp.MustCompile(`^(\.git|node_modules|\.ovav/worktrees|\.ovav/runtime/logs)`)

func shouldSkipPath(rel string) bool {
	return blockedPaths.MatchString(rel)
}
