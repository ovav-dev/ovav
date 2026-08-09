// Package hooks implements the OVAV Git Hook System — native Go, zero external deps.
//
// Architecture:
//   - Thin bash shims → ovav hook run --stage <stage>
//   - Stage-filtered dispatcher → validators.RunStage()
//   - SHA-256 integrity verification on every execution
//   - Cross-platform: Linux/macOS (bash shims), Windows (batch shims)
//
// Integration points:
//   - owc (worktree create) → auto-installs hooks
//   - ovav infra bootstrap → installs hooks system-wide
//   - ovav hook run → called by git hooks automatically
//   - owv → shares same validator engine, different scope
//
// This package is part of the OVAV Workflow System (OWS).
// It does NOT duplicate owv validators — it dispatches to the same DefaultRegistry().
package hooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ovav/ovav/internal/validators"
)

// ── Types ──────────────────────────────────────────────────────────────────────

// Stage represents a git hook trigger point.
type Stage string

const (
	StagePreCommit    Stage = "pre-commit"
	StagePrePush      Stage = "pre-push"
	StagePostCheckout Stage = "post-checkout"
	StageCommitMsg    Stage = "commit-msg"
)

// AllStages returns all supported hook stages.
func AllStages() []Stage {
	return []Stage{StagePreCommit, StagePrePush, StagePostCheckout, StageCommitMsg}
}

// HookName returns the git hook filename for a stage.
func (s Stage) HookName() string {
	return string(s)
}

// StageLabel returns the human-readable label.
func (s Stage) Label() string {
	switch s {
	case StagePreCommit:
		return "Pre-commit"
	case StagePrePush:
		return "Pre-push"
	case StagePostCheckout:
		return "Post-checkout"
	case StageCommitMsg:
		return "Commit-msg"
	default:
		return string(s)
	}
}

// Manager manages git hooks for an OVAV repository.
type Manager struct {
	RepoRoot   string // path to the repo root (main repo, not worktree)
	OVAVBinary string // path to the ovav Go binary
}

// NewManager creates a hook manager for the given repository.
func NewManager(repoRoot string) *Manager {
	return &Manager{
		RepoRoot:   repoRoot,
		OVAVBinary: resolveOVAVBinary(repoRoot),
	}
}

// ── Installation ───────────────────────────────────────────────────────────────

// InstallResult holds the outcome of a hook installation.
type InstallResult struct {
	Stage   Stage  `json:"stage"`
	Status  string `json:"status"` // "installed", "updated", "skip", "fail"
	Path    string `json:"path"`
	Message string `json:"message,omitempty"`
}

// Install installs or updates all OVAV git hook shims in .git/hooks/.
// It is idempotent — running it multiple times is safe.
// Returns the list of installation results.
func (m *Manager) Install() ([]InstallResult, error) {
	hooksDir := m.hooksDir()
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return nil, fmt.Errorf("create hooks dir %s: %w", hooksDir, err)
	}

	var results []InstallResult
	for _, stage := range AllStages() {
		r := m.installHook(stage)
		results = append(results, r)
	}
	return results, nil
}

// installHook installs a single hook shim, replacing any existing broken symlinks.
func (m *Manager) installHook(stage Stage) InstallResult {
	hookPath := filepath.Join(m.hooksDir(), stage.HookName())
	result := InstallResult{Stage: stage, Path: hookPath}

	// Check if existing hook is an OVAV-managed shim
	existing, err := os.ReadFile(hookPath)
	if err == nil && isOVAVShim(string(existing)) {
		// Already installed — verify integrity
		expected := m.shimContent(stage)
		if string(existing) == expected {
			result.Status = "skip"
			result.Message = "already installed and intact"
			return result
		}
		// Content differs — update
		result.Status = "updated"
		result.Message = "updated to latest shim"
	} else {
		result.Status = "installed"
		result.Message = "fresh installation"
	}

	// Remove any broken symlinks
	if info, err := os.Lstat(hookPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		os.Remove(hookPath)
	}

	content := m.shimContent(stage)
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		result.Status = "fail"
		result.Message = fmt.Sprintf("write failed: %v", err)
	}
	return result
}

// Uninstall removes all OVAV-managed hook shims.
func (m *Manager) Uninstall() ([]InstallResult, error) {
	var results []InstallResult
	for _, stage := range AllStages() {
		hookPath := filepath.Join(m.hooksDir(), stage.HookName())
		r := InstallResult{Stage: stage, Path: hookPath}

		data, err := os.ReadFile(hookPath)
		if err != nil {
			r.Status = "skip"
			r.Message = "not found"
			results = append(results, r)
			continue
		}

		if isOVAVShim(string(data)) {
			if err := os.Remove(hookPath); err != nil {
				r.Status = "fail"
				r.Message = fmt.Sprintf("remove failed: %v", err)
			} else {
				r.Status = "removed"
			}
		} else {
			r.Status = "skip"
			r.Message = "not an OVAV hook — preserved"
		}
		results = append(results, r)
	}
	return results, nil
}

// ── Status ─────────────────────────────────────────────────────────────────────

// HookStatus represents the state of a single hook installation.
type HookStatus struct {
	Stage      Stage  `json:"stage"`
	Installed  bool   `json:"installed"`
	OVAV       bool   `json:"ovav_managed"`
	Executable bool   `json:"executable"`
	SHA256     string `json:"sha256,omitempty"`
	SHA256OK   bool   `json:"sha256_ok"`
	Message    string `json:"message,omitempty"`
}

// StatusReport holds the complete hook status for a repository.
type StatusReport struct {
	RepoRoot   string       `json:"repo_root"`
	HooksDir   string       `json:"hooks_dir"`
	OVAVBinary string       `json:"ovav_binary"`
	AllHealthy bool         `json:"all_healthy"`
	Hooks      []HookStatus `json:"hooks"`
	LastCheck  string       `json:"last_check"`
}

// Status checks the installation and integrity of all OVAV hooks.
func (m *Manager) Status() (*StatusReport, error) {
	report := &StatusReport{
		RepoRoot:   m.RepoRoot,
		HooksDir:   m.hooksDir(),
		OVAVBinary: m.OVAVBinary,
		LastCheck:  time.Now().UTC().Format(time.RFC3339),
	}

	allHealthy := true
	for _, stage := range AllStages() {
		hs := m.checkHookStatus(stage)
		report.Hooks = append(report.Hooks, hs)
		if !hs.Installed || !hs.OVAV || !hs.Executable || !hs.SHA256OK {
			allHealthy = false
		}
	}
	report.AllHealthy = allHealthy
	return report, nil
}

// checkHookStatus inspects a single hook file.
func (m *Manager) checkHookStatus(stage Stage) HookStatus {
	hookPath := filepath.Join(m.hooksDir(), stage.HookName())
	hs := HookStatus{Stage: stage}

	info, err := os.Stat(hookPath)
	if err != nil {
		hs.Message = "not installed"
		return hs
	}
	hs.Installed = true
	hs.Executable = info.Mode()&0111 != 0

	data, err := os.ReadFile(hookPath)
	if err != nil {
		hs.Message = fmt.Sprintf("cannot read: %v", err)
		return hs
	}

	content := string(data)
	hs.OVAV = isOVAVShim(content)

	// Compute SHA-256 of the shim
	h := sha256.Sum256(data)
	hs.SHA256 = hex.EncodeToString(h[:])

	// Verify against expected shim
	if hs.OVAV {
		expected := m.shimContent(stage)
		expectedHash := sha256.Sum256([]byte(expected))
		hs.SHA256OK = hex.EncodeToString(expectedHash[:]) == hs.SHA256
		if !hs.SHA256OK {
			hs.Message = "shim tampered or outdated — run 'ovav hook install' to repair"
		}
	} else {
		hs.SHA256OK = true // not our shim, can't verify
		hs.Message = "foreign hook — not OVAV-managed"
	}

	return hs
}

// ── Execution ──────────────────────────────────────────────────────────────────

// RunResult holds the outcome of a hook stage execution.
type RunResult struct {
	Stage    Stage               `json:"stage"`
	Passed   bool                `json:"passed"`
	Results  []validators.Result `json:"results,omitempty"`
	Duration time.Duration       `json:"duration_ms"`
	Error    string              `json:"error,omitempty"`
}

// RunStage executes validators for the given hook stage.
// It filters the DefaultRegistry to only run validators relevant to this stage
// (defined in auto_triggers.yaml). Validators run in parallel via goroutines.
func (m *Manager) RunStage(stage Stage) *RunResult {
	start := time.Now()
	result := &RunResult{Stage: stage}

	// Get the stage filter
	filter := GetStageFilter(stage)
	if len(filter) == 0 {
		result.Passed = true
		result.Duration = time.Since(start)
		return result
	}

	// Build a filtered registry — only validators for this stage
	registry := validators.DefaultRegistry()
	allValidators := registry.All()

	// Execute in parallel
	type jobResult struct {
		r validators.Result
	}
	jobs := make(chan jobResult, len(allValidators))
	var wg sync.WaitGroup

	for _, v := range allValidators {
		// Skip validators not in this stage's filter
		if !filter.Matches(v.ID()) {
			continue
		}
		wg.Add(1)
		go func(v validators.Validator) {
			defer wg.Done()
			jobs <- jobResult{r: v.Validate(context.Background(), m.RepoRoot)}
		}(v)
	}

	// Wait for all and collect
	go func() {
		wg.Wait()
		close(jobs)
	}()

	passed := true
	for j := range jobs {
		result.Results = append(result.Results, j.r)
		if j.r.Status == "fail" || j.r.Status == "error" {
			passed = false
		}
	}

	result.Passed = passed
	result.Duration = time.Since(start)
	return result
}

// ── Audit ───────────────────────────────────────────────────────────────────────

// AuditReport holds the results of a hook security audit.
type AuditReport struct {
	RepoRoot  string       `json:"repo_root"`
	HooksDir  string       `json:"hooks_dir"`
	Status    string       `json:"status"` // "clean", "tampered", "broken", "uninstalled"
	Hooks     []HookStatus `json:"hooks"`
	Threats   []string     `json:"threats,omitempty"`
	LastAudit string       `json:"last_audit"`
}

// Audit performs a security audit of all git hooks.
// It checks: installation, SHA-256 integrity, OVAV ownership, symlink safety, tampering.
func (m *Manager) Audit() (*AuditReport, error) {
	report := &AuditReport{
		RepoRoot:  m.RepoRoot,
		HooksDir:  m.hooksDir(),
		LastAudit: time.Now().UTC().Format(time.RFC3339),
	}

	threats := make([]string, 0)

	// Check each hook
	for _, stage := range AllStages() {
		hs := m.checkHookStatus(stage)
		report.Hooks = append(report.Hooks, hs)

		if !hs.Installed {
			threats = append(threats, fmt.Sprintf("%s: missing — validation bypass possible", stage.Label()))
		}
		if hs.Installed && !hs.OVAV {
			threats = append(threats, fmt.Sprintf("%s: foreign hook — not OVAV managed, unauditable", stage.Label()))
		}
		if hs.Installed && hs.OVAV && !hs.SHA256OK {
			threats = append(threats, fmt.Sprintf("%s: SHA-256 MISMATCH — shim tampered", stage.Label()))
		}
		if hs.Installed && !hs.Executable {
			threats = append(threats, fmt.Sprintf("%s: not executable — hook will be silently ignored", stage.Label()))
		}
	}

	// Check for symlinks (potential contamination vector)
	for _, stage := range AllStages() {
		hookPath := filepath.Join(m.hooksDir(), stage.HookName())
		info, err := os.Lstat(hookPath)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(hookPath)
			threats = append(threats, fmt.Sprintf("%s: is a SYMLINK → %s — cross-worktree contamination risk", stage.Label(), target))
		}
	}

	// Check for broken symlinks
	for _, stage := range AllStages() {
		hookPath := filepath.Join(m.hooksDir(), stage.HookName())
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			if _, lerr := os.Lstat(hookPath); lerr == nil {
				threats = append(threats, fmt.Sprintf("%s: BROKEN symlink — git will silently ignore this hook", stage.Label()))
			}
		}
	}

	report.Threats = threats

	switch {
	case len(report.Hooks) == 0:
		report.Status = "uninstalled"
	case len(threats) == 0:
		report.Status = "clean"
	case containsTampered(report.Hooks):
		report.Status = "tampered"
	default:
		report.Status = "broken"
	}

	return report, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

// hooksDir returns the path to .git/hooks/ for the repository.
// Handles both main repos (.git is a directory) and worktrees (.git is a file).
func (m *Manager) hooksDir() string {
	gitPath := filepath.Join(m.RepoRoot, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		return gitPath + "/hooks"
	}

	// Main repo: .git is a directory
	if info.IsDir() {
		return filepath.Join(gitPath, "hooks")
	}

	// Worktree: .git is a file containing "gitdir: /path/to/main/.git/worktrees/name"
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return gitPath + "/hooks"
	}

	content := strings.TrimSpace(string(data))
	if strings.HasPrefix(content, "gitdir: ") {
		gitDir := strings.TrimPrefix(content, "gitdir: ")
		// gitDir points to <main>/.git/worktrees/<name>
		// The real hooks are at <main>/.git/hooks/
		// Walk up from worktrees/<name> to .git, then to hooks
		mainGitDir := filepath.Dir(filepath.Dir(gitDir)) // worktrees/<name> → worktrees → .git
		return filepath.Join(mainGitDir, "hooks")
	}

	return gitPath + "/hooks"
}

// shimContent returns the thin bash/batch wrapper for a given stage.
func (m *Manager) shimContent(stage Stage) string {
	if runtime.GOOS == "windows" {
		return m.shimContentWindows(stage)
	}
	return m.shimContentUnix(stage)
}

// shimContentUnix returns the POSIX shell shim.
func (m *Manager) shimContentUnix(stage Stage) string {
	binary := m.OVAVBinary
	if binary == "" {
		binary = "ovav"
	}
	return fmt.Sprintf(`#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────
# OVAV Git Hook — %s
# Managed by: ovav hook install
# DO NOT EDIT — edits will be overwritten
# SHA-256 verified on every execution
# ──────────────────────────────────────────────────────────
set -euo pipefail
exec %s hook run --stage %s "$@"
`, stage.Label(), binary, string(stage))
}

// shimContentWindows returns the PowerShell/batch shim for Windows.
func (m *Manager) shimContentWindows(stage Stage) string {
	binary := m.OVAVBinary
	if binary == "" {
		binary = "ovav"
	}
	return fmt.Sprintf(`@echo off
REM OVAV Git Hook — %s
REM Managed by: ovav hook install
REM DO NOT EDIT
%s hook run --stage %s %%*
`, stage.Label(), binary, string(stage))
}

// isOVAVShim checks if a hook content is an OVAV-managed shim.
func isOVAVShim(content string) bool {
	return strings.Contains(content, "OVAV Git Hook") ||
		strings.Contains(content, "ovav hook run --stage")
}

// resolveOVAVBinary finds the ovav Go binary.
// Priority: 1) OVAV_BIN env  2) ~/.local/bin/ovav  3) go-runtime/build/ovav  4) $PATH/ovav
func resolveOVAVBinary(repoRoot string) string {
	if env := os.Getenv("OVAV_BIN"); env != "" {
		return env
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "ovav"),
		filepath.Join(repoRoot, "go-runtime", "build", "ovav"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Fallback: rely on $PATH
	if p, err := exec.LookPath("ovav"); err == nil {
		return p
	}

	return "ovav"
}

// containsTampered checks if any hook has been tampered.
func containsTampered(hooks []HookStatus) bool {
	for _, h := range hooks {
		if h.OVAV && !h.SHA256OK {
			return true
		}
	}
	return false
}

// ── Manifest ───────────────────────────────────────────────────────────────────

// Manifest returns a summary of hook configuration for display.
func (m *Manager) Manifest() map[string]interface{} {
	return map[string]interface{}{
		"repo_root":   m.RepoRoot,
		"ovav_binary": m.OVAVBinary,
		"stages":      AllStages(),
		"platform":    runtime.GOOS,
		"managed_by":  "ovav hook install",
	}
}
