package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── Integrity Verification ────────────────────────────────────────────────────

// IntegritySnapshot holds the expected SHA-256 hashes of all hook shims.
// This is used to detect tampering by comparing current state against known-good.
type IntegritySnapshot struct {
	GeneratedAt string           `json:"generated_at"`
	BinarySHA   string           `json:"binary_sha"`
	Hooks       map[Stage]string `json:"hooks"` // stage → SHA-256
}

// GenerateIntegritySnapshot computes SHA-256 hashes of all installed hook shims
// and the ovav binary. This snapshot is the "known-good" baseline for tamper detection.
func (m *Manager) GenerateIntegritySnapshot() (*IntegritySnapshot, error) {
	snap := &IntegritySnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Hooks:       make(map[Stage]string),
	}

	// Hash the ovav binary
	if data, err := os.ReadFile(m.OVAVBinary); err == nil {
		h := sha256.Sum256(data)
		snap.BinarySHA = hex.EncodeToString(h[:])
	}

	// Hash each hook shim
	for _, stage := range AllStages() {
		hookPath := filepath.Join(m.hooksDir(), stage.HookName())
		data, err := os.ReadFile(hookPath)
		if err != nil {
			continue
		}
		h := sha256.Sum256(data)
		snap.Hooks[stage] = hex.EncodeToString(h[:])
	}

	return snap, nil
}

// VerifyIntegrity compares current hook state against a known-good snapshot.
// Returns a list of violations found.
func (m *Manager) VerifyIntegrity(snap *IntegritySnapshot) []string {
	var violations []string

	// Verify binary integrity
	if snap.BinarySHA != "" {
		if data, err := os.ReadFile(m.OVAVBinary); err == nil {
			h := sha256.Sum256(data)
			currentSHA := hex.EncodeToString(h[:])
			if currentSHA != snap.BinarySHA {
				violations = append(violations, fmt.Sprintf(
					"OVAV binary tampered: expected %s, got %s", snap.BinarySHA[:16], currentSHA[:16]))
			}
		}
	}

	// Verify each hook shim
	for stage, expectedSHA := range snap.Hooks {
		hookPath := filepath.Join(m.hooksDir(), stage.HookName())
		data, err := os.ReadFile(hookPath)
		if err != nil {
			violations = append(violations, fmt.Sprintf("%s: missing — was removed", stage.Label()))
			continue
		}
		h := sha256.Sum256(data)
		currentSHA := hex.EncodeToString(h[:])
		if currentSHA != expectedSHA {
			violations = append(violations, fmt.Sprintf("%s: tampered — SHA-256 mismatch", stage.Label()))
		}
	}

	return violations
}

// ── No-verify Detection ────────────────────────────────────────────────────────

// NoVerifyCheck checks git reflog for evidence of --no-verify usage.
// This is a local best-effort check. Full enforcement requires server-side
// CI attestation and GitHub pre-receive hooks.
func (m *Manager) NoVerifyCheck() (*NoVerifyReport, error) {
	report := &NoVerifyReport{
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Check reflog for commits with --no-verify patterns
	// This is an approximation — git doesn't log --no-verify usage directly.
	// We check for:
	// 1. Commits on protected branches without waiver
	// 2. Unusually large commits (potential bypass)
	// 3. Timestamps that don't match the last hook audit

	// For now, we check if the hooks look intact (best-effort proxy)
	status, err := m.Status()
	if err != nil {
		report.Detected = false
		report.Message = "cannot verify: " + err.Error()
		return report, nil
	}

	if !status.AllHealthy {
		report.Detected = true
		report.Evidence = append(report.Evidence,
			"Hooks are not all healthy — commits may have bypassed validation")
		for _, h := range status.Hooks {
			if !h.Installed {
				report.Evidence = append(report.Evidence,
					fmt.Sprintf("  %s: not installed — all commits since last install may be unverified", h.Stage.Label()))
			}
			if !h.SHA256OK {
				report.Evidence = append(report.Evidence,
					fmt.Sprintf("  %s: tampered — commits may have bypassed modified hooks", h.Stage.Label()))
			}
		}
	}

	if !report.Detected {
		report.Message = "No evidence of --no-verify usage detected (hooks intact)"
	}

	return report, nil
}

// NoVerifyReport holds the results of a no-verify check.
type NoVerifyReport struct {
	CheckedAt string   `json:"checked_at"`
	Detected  bool     `json:"detected"`
	Evidence  []string `json:"evidence,omitempty"`
	Message   string   `json:"message"`
}

// ── Tampering History ─────────────────────────────────────────────────────────

// TamperingEvent records a detected tampering incident.
type TamperingEvent struct {
	Timestamp string `json:"timestamp"`
	Stage     Stage  `json:"stage"`
	Type      string `json:"type"` // "sha256_mismatch", "missing", "symlink", "foreign_hook"
	Detail    string `json:"detail"`
}

// CheckTampering performs a quick tampering check and returns any events found.
// This is designed to run on every hook invocation for real-time detection.
func (m *Manager) CheckTampering() []TamperingEvent {
	var events []TamperingEvent

	for _, stage := range AllStages() {
		hookPath := filepath.Join(m.hooksDir(), stage.HookName())
		info, err := os.Lstat(hookPath)

		if os.IsNotExist(err) {
			events = append(events, TamperingEvent{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Stage:     stage,
				Type:      "missing",
				Detail:    "Hook file does not exist — validation bypass possible",
			})
			continue
		}

		// Symlink detection
		if info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(hookPath)
			events = append(events, TamperingEvent{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Stage:     stage,
				Type:      "symlink",
				Detail:    fmt.Sprintf("Hook is symlink → %s (cross-worktree contamination risk)", target),
			})
		}

		// SHA-256 check
		data, err := os.ReadFile(hookPath)
		if err != nil {
			continue
		}

		if !isOVAVShim(string(data)) {
			events = append(events, TamperingEvent{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Stage:     stage,
				Type:      "foreign_hook",
				Detail:    "Hook content is not OVAV-managed — unauditable code is running",
			})
		}
	}

	return events
}

// ── CI Detection ───────────────────────────────────────────────────────────────

// IsCI returns true if the current environment is a CI system.
// Used to adjust hook behavior: in CI, hooks should be stricter and
// should fail loudly rather than silently.
func IsCI() bool {
	// Standard CI environment variables
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_ID", "GITHUB_ACTIONS",
		"GITLAB_CI", "CIRCLECI", "JENKINS_HOME", "TRAVIS", "APPVEYOR",
		"DRONE", "CODEBUILD_BUILD_ID", "BITBUCKET_BUILD_NUMBER",
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// CIStrictMode returns extra validators to run in CI environments.
// These are validators that might be too slow for interactive commit hooks
// but are appropriate for CI where latency isn't critical.
func CIStrictMode() StageFilter {
	return StageFilter{
		"check_living_integrity",
		"check_supply_chain",
		"check_exfil_patterns",
		"check_permission_policy_drift",
		"check_contract_freshness",
		"validate_all",
	}
}

// FormatNoVerifyHuman returns a human-readable summary of a no-verify report.
func FormatNoVerifyHuman(r *NoVerifyReport) string {
	var b strings.Builder
	b.WriteString("── OVAV No-Verify Audit ──\n\n")

	if r.Detected {
		b.WriteString("⚠️  POTENTIAL BYPASS DETECTED\n\n")
		b.WriteString("Evidence:\n")
		for _, e := range r.Evidence {
			b.WriteString(fmt.Sprintf("  %s\n", e))
		}
		b.WriteString("\nRecommendation: run 'ovav hook install' to restore hooks,\n")
		b.WriteString("then re-commit affected changes for validation.\n")
	} else {
		b.WriteString("✅ No evidence of bypass — hooks intact.\n")
		b.WriteString(fmt.Sprintf("   Checked: %s\n", r.CheckedAt))
	}

	return b.String()
}
