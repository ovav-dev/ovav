// Package install implements the OVAV Install Gateway — a unified install pipeline
// for applying governed changes to repository surfaces.
//
// Ported from tools/install_gateway/ (Python) to Go stdlib.
// Three operational modes:
//
//   - dry-run: plan/preview only, no writes
//   - sandbox: simulated writes within .ovav/artifacts/ sandbox
//   - source-local-apply: real writes bounded to REPO_ROOT with gate enforcement
//
// Pipeline: plan → manifest → safety → boundaries → backup → apply → verify → report.
// All functions accept mode as a parameter. Source-local-apply requires all gates to pass.
package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// applyMu protects concurrent ExecuteApply pipeline calls.
var applyMu sync.Mutex

// ── Mode type ─────────────────────────────────────────────────────────────────

// Mode represents the operational mode of the install gateway.
type Mode string

const (
	ModeDryRun           Mode = "dry-run"
	ModeSandbox          Mode = "sandbox"
	ModeSourceLocalApply Mode = "source-local-apply"
)

// ValidModes lists all supported operational modes.
var ValidModes = []Mode{ModeDryRun, ModeSandbox, ModeSourceLocalApply}

// ResolveMode normalizes and validates an apply mode string.
func ResolveMode(mode string) (Mode, error) {
	normalized := strings.ToLower(strings.ReplaceAll(mode, "_", "-"))
	switch {
	case normalized == "dry-run" || normalized == "dryrun":
		return ModeDryRun, nil
	case normalized == "sandbox":
		return ModeSandbox, nil
	case normalized == "source-local-apply" || normalized == "apply" || normalized == "source-local":
		return ModeSourceLocalApply, nil
	default:
		return "", fmt.Errorf("install: unknown mode %q — valid modes: %v", mode, ValidModes)
	}
}

// ── Apply-eligible surfaces (source-local-apply mode only) ────────────────────

// SourceLocalApplyEligible lists the only paths that source-local-apply may target.
var SourceLocalApplyEligible = []string{
	".ovav/registry/",
	"runtimes/opencode/agents/",
	".opencode/commands/",
	".ovav/source/skills/",
	".ovav/artifacts/",
	"tools/",
}

// ── Permanently blocked surfaces ──────────────────────────────────────────────

// PermanentlyBlockedSurfaces lists surfaces that are blocked in all modes.
var PermanentlyBlockedSurfaces = []string{
	"global_install",
	"user_home_config",
	"opencode_global_config",
	"plugin_install",
	"external_services",
	"ui_tui",
	"mcp_a2a",
}

// ── Unsafe selectors — always rejected ────────────────────────────────────────

// UnsafeSelectors are keywords that, if found in a target path, cause rejection.
var UnsafeSelectors = []string{
	"apply-now", "install-now", "global", "write-global", "real",
	"home", "user-config", "opencode-global",
	"external", "mcp", "a2a", "ui", "tui",
}

// ── Risk classification ───────────────────────────────────────────────────────

// RiskMap classifies risk levels to action categories.
var RiskMap = map[string]string{
	"repo-local":         "allow",
	"source-local-apply": "allow_with_gates",
	"user-config-risk":   "review_required",
	"user-local-risk":    "review_required",
	"global-risk":        "blocked",
	"unknown-risk":       "blocked",
	"sandbox":            "allow_dry_run",
}

// ── Gate definitions ──────────────────────────────────────────────────────────

// BackupGates lists all backup-related gates.
var BackupGates = []string{
	"explicit_approval",
	"deterministic_plan",
	"backup_plan_exists",
	"backup_executed",
	"backup_verified",
	"affected_manifest",
	"dry_run_preview",
	"strict_validation",
	"no_evidence_drift",
}

// RollbackGates lists all rollback-related gates.
var RollbackGates = []string{
	"restore_plan_exists",
	"rollback_completeness",
	"rollback_deterministic",
	"rollback_cannot_escalate",
	"rollback_sandbox_tested",
}

// ── Entry types ───────────────────────────────────────────────────────────────

// PlanEntry represents a single entry in an install plan.
type PlanEntry struct {
	TargetID     string `json:"target_id"`
	Target       string `json:"target"`
	Source       string `json:"source,omitempty"`
	TargetRisk   string `json:"target_risk"`
	Mode         Mode   `json:"mode"`
	WriteEnabled bool   `json:"write_enabled"`
}

// Plan is the result of building an install plan from a pack.
type Plan struct {
	Status       string      `json:"status"`
	PackID       string      `json:"pack_id"`
	PackNotes    string      `json:"pack_notes,omitempty"`
	Mode         Mode        `json:"mode"`
	RealApply    bool        `json:"real_apply"`
	DryRunOnly   bool        `json:"dry_run_only"`
	SandboxOnly  bool        `json:"sandbox_only"`
	Entries      []PlanEntry `json:"entries"`
	EntryCount   int         `json:"entry_count"`
	AllowedModes []string    `json:"allowed_modes,omitempty"`
	Error        string      `json:"error,omitempty"`
	Reason       string      `json:"reason,omitempty"`
}

// ManifestEntry represents a single entry in an install manifest.
type ManifestEntry struct {
	Target        string `json:"target"`
	TargetExists  bool   `json:"target_exists"`
	TargetRisk    string `json:"target_risk"`
	Operation     string `json:"operation"`
	NeedsBackup   bool   `json:"needs_backup"`
	NeedsRollback bool   `json:"needs_rollback"`
	WriteEnabled  bool   `json:"write_enabled"`
	Mode          Mode   `json:"mode"`
	Source        string `json:"source,omitempty"`
}

// Manifest is the result of building a manifest from a plan.
type Manifest struct {
	Status              string          `json:"status"`
	PackID              string          `json:"pack_id"`
	Mode                Mode            `json:"mode"`
	TotalEntries        int             `json:"total_entries"`
	BlockedEntries      int             `json:"blocked_entries"`
	ApplyEntries        int             `json:"apply_entries"`
	BackupRequiredCount int             `json:"backup_required_count"`
	DryRunOnly          bool            `json:"dry_run_only"`
	Entries             []ManifestEntry `json:"entries"`
	BlockedDetails      []ManifestEntry `json:"blocked_details"`
}

// SafetyEntry represents the safety evaluation of a single target.
type SafetyEntry struct {
	Target       string `json:"target"`
	Risk         string `json:"risk"`
	SafetyStatus string `json:"safety_status"`
	Mode         Mode   `json:"mode"`
	WriteAllowed bool   `json:"write_allowed"`
}

// SafetyReport is the result of a safety evaluation.
type SafetyReport struct {
	Status           string        `json:"status"`
	Mode             Mode          `json:"mode"`
	OverallSafety    string        `json:"overall_safety"`
	HasBlocked       bool          `json:"has_blocked"`
	HasReviewReq     bool          `json:"has_review_required"`
	NeedsBackup      bool          `json:"needs_backup"`
	NeedsRollback    bool          `json:"needs_rollback"`
	Entries          []SafetyEntry `json:"entries"`
	Issues           []string      `json:"issues"`
	RealApplyAllowed bool          `json:"real_apply_allowed"`
}

// BackupResult represents the result of backing up a single target.
type BackupResult struct {
	Target     string `json:"target"`
	BackupPath string `json:"backup_path,omitempty"`
	SourceHash string `json:"source_hash,omitempty"`
	BackupHash string `json:"backup_hash,omitempty"`
	Verified   bool   `json:"verified"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
}

// BackupReport is the result of executing a backup operation.
type BackupReport struct {
	Status            string         `json:"status"`
	Mode              Mode           `json:"mode"`
	Timestamp         string         `json:"timestamp"`
	BackupDir         string         `json:"backup_dir,omitempty"`
	BackupPerformed   bool           `json:"backup_performed"`
	TotalTargets      int            `json:"total_targets"`
	BackedUp          int            `json:"backed_up"`
	Skipped           int            `json:"skipped"`
	Failed            int            `json:"failed"`
	Blocked           int            `json:"blocked"`
	DryRunPreview     bool           `json:"dry_run_preview,omitempty"`
	TargetsIdentified int            `json:"targets_identified,omitempty"`
	Targets           []string       `json:"targets,omitempty"`
	Results           []BackupResult `json:"results"`
}

// ApplyResult represents the result of applying a single file operation.
type ApplyResult struct {
	Target    string `json:"target"`
	Operation string `json:"operation"`
	Written   bool   `json:"written"`
	Error     string `json:"error,omitempty"`
}

// ApplyReport is the result of file apply operations.
type ApplyReport struct {
	Status  string        `json:"status"`
	Mode    Mode          `json:"mode"`
	Total   int           `json:"total"`
	Written int           `json:"written"`
	Skipped int           `json:"skipped"`
	Results []ApplyResult `json:"results"`
}

// RollbackResult represents the result of rolling back a single file.
type RollbackResult struct {
	Target       string `json:"target"`
	BackupPath   string `json:"backup_path,omitempty"`
	ExpectedHash string `json:"expected_hash,omitempty"`
	RestoredHash string `json:"restored_hash,omitempty"`
	Verified     bool   `json:"verified"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
}

// RollbackReport is the result of executing a rollback operation.
type RollbackReport struct {
	Status            string           `json:"status"`
	Mode              Mode             `json:"mode"`
	Timestamp         string           `json:"timestamp"`
	RollbackPerformed bool             `json:"rollback_performed"`
	TotalTargets      int              `json:"total_targets"`
	Restored          int              `json:"restored"`
	Failed            int              `json:"failed"`
	CompletenessOK    bool             `json:"completeness_ok"`
	DryRunPreview     bool             `json:"dry_run_preview,omitempty"`
	TargetsAvailable  int              `json:"targets_available,omitempty"`
	Targets           []string         `json:"targets,omitempty"`
	Results           []RollbackResult `json:"results"`
}

// GateReport contains the gate satisfaction evaluation.
type GateReport struct {
	Backup         GateEval `json:"backup"`
	Rollback       GateEval `json:"rollback"`
	TotalSatisfied int      `json:"total_satisfied"`
	TotalGates     int      `json:"total_gates"`
}

// GateEval evaluates how many gates in a category are satisfied.
type GateEval struct {
	Total           int      `json:"total"`
	Satisfied       int      `json:"satisfied"`
	SatisfiedList   []string `json:"satisfied_list"`
	UnsatisfiedList []string `json:"unsatisfied"`
}

// VerifyResult represents verification of a single target.
type VerifyResult struct {
	Target string `json:"target"`
	Exists bool   `json:"exists"`
	Status string `json:"status"`
}

// VerifyReport is the result of post-apply verification.
type VerifyReport struct {
	Status   string         `json:"status"`
	Mode     Mode           `json:"mode"`
	Total    int            `json:"total"`
	Verified int            `json:"verified"`
	Missing  int            `json:"missing"`
	Results  []VerifyResult `json:"results"`
}

// BoundaryResult represents a boundary check for a single target.
type BoundaryResult struct {
	Status      string `json:"status"`
	Target      string `json:"target"`
	Mode        Mode   `json:"mode"`
	AllowsWrite bool   `json:"allows_write"`
	Reason      string `json:"reason,omitempty"`
}

// BoundaryReport is the result of validating all targets against boundaries.
type BoundaryReport struct {
	Status         string           `json:"status"`
	Mode           Mode             `json:"mode"`
	Total          int              `json:"total"`
	Allowed        int              `json:"allowed"`
	Blocked        int              `json:"blocked"`
	BlockedDetails []BoundaryResult `json:"blocked_details"`
	Results        []BoundaryResult `json:"results"`
}

// StageResults holds all pipeline stage outputs.
type StageResults struct {
	Plan       Plan           `json:"plan"`
	Manifest   Manifest       `json:"manifest"`
	Safety     SafetyReport   `json:"safety"`
	Boundaries BoundaryReport `json:"boundaries"`
	Backup     BackupReport   `json:"backup"`
	Apply      ApplyReport    `json:"apply"`
	Verify     VerifyReport   `json:"verify"`
	Gates      GateReport     `json:"gates"`
}

// ApplyGatewayReport is the top-level result of the full apply pipeline.
type ApplyGatewayReport struct {
	Status                string       `json:"status"`
	PackID                string       `json:"pack_id"`
	Mode                  Mode         `json:"mode"`
	SourceLocalApplyReady bool         `json:"source_local_apply_ready"`
	RealApplyPerformed    bool         `json:"real_apply_performed"`
	Stages                StageResults `json:"stages"`
	Errors                []string     `json:"errors"`
	BlockedSurfaces       []string     `json:"blocked_surfaces"`
}

// ── Evidence report types ─────────────────────────────────────────────────────

// EvidenceReport contains the result of writing evidence to disk.
type EvidenceReport struct {
	Status       string   `json:"status"`
	Segment      string   `json:"segment"`
	Timestamp    string   `json:"timestamp"`
	FilesWritten int      `json:"files_written"`
	Paths        []string `json:"paths"`
}

// UXPreview contains human-readable previews of install operations.
type UXPreview struct {
	Status           string `json:"status"`
	Mode             Mode   `json:"mode"`
	PlanPreview      string `json:"plan_preview"`
	RiskPreview      string `json:"risk_preview"`
	RollbackGuide    string `json:"rollback_guide"`
	RealApplyAllowed bool   `json:"real_apply_allowed"`
	GatesRequired    bool   `json:"gates_required"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// timestamp returns a UTC timestamp string suitable for filenames.
func timestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

// hashFile returns the SHA-256 hex digest of a file, or empty string if the file cannot be read.
func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// isSourceLocalPath checks if a path resolves within the given root.
func isSourceLocalPath(path string, root string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	return strings.HasPrefix(abs, rootAbs)
}

// isSafeTarget checks if a target path is safe for the given mode.
func isSafeTarget(target string, mode Mode, root string) bool {
	if mode == "" || mode == ModeDryRun || mode == ModeSandbox {
		return isSourceLocalPath(target, root)
	}
	if mode == ModeSourceLocalApply {
		if !isSourceLocalPath(target, root) {
			return false
		}
		abs, _ := filepath.Abs(target)
		rootAbs, _ := filepath.Abs(root)
		rel := strings.TrimPrefix(abs, rootAbs)
		rel = strings.TrimPrefix(rel, string(filepath.Separator))

		// Ensure rel has trailing separator for prefix matching
		relWithSep := rel + string(filepath.Separator)
		for _, surface := range SourceLocalApplyEligible {
			// Match if rel starts with surface OR surface starts with rel (directory match)
			if strings.HasPrefix(relWithSep, surface) || strings.HasPrefix(surface, relWithSep) {
				return true
			}
		}
		return false
	}
	return isSourceLocalPath(target, root)
}

// relPath returns the relative path within root, or the original if unresolvable.
func relPath(target string, root string) string {
	abs, err := filepath.Abs(target)
	if err != nil {
		return target
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return target
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return target
	}
	return rel
}
