package install

import (
	"os"
	"path/filepath"
	"strings"
)

// CheckTargetBoundary checks if a target path is within mode boundaries.
func CheckTargetBoundary(target string, mode Mode, root string) BoundaryResult {
	targetLower := strings.ToLower(target)

	// Home path detection
	home, err := os.UserHomeDir()
	if err == nil {
		absTarget, err := filepath.Abs(target)
		if err == nil && strings.HasPrefix(absTarget, home) {
			rootAbs, _ := filepath.Abs(root)
			if !strings.HasPrefix(absTarget, rootAbs) {
				return BoundaryResult{
					Status: "blocked",
					Target: target,
					Mode:   mode,
					Reason: "home_path_detected_outside_repo_root",
				}
			}
		}
	}

	// Check for unsafe selectors in relative path
	rel := relPath(target, root)
	for _, unsafe := range UnsafeSelectors {
		if strings.Contains(strings.ToLower(rel), unsafe) {
			return BoundaryResult{
				Status: "blocked",
				Target: target,
				Mode:   mode,
				Reason: "unsafe_selector_detected_in_relative_path: " + unsafe,
			}
		}
	}

	switch mode {
	case ModeDryRun:
		return BoundaryResult{Status: "ok", Target: target, Mode: mode, AllowsWrite: false}
	case ModeSandbox:
		allowed := isSourceLocalPath(target, root) && strings.Contains(targetLower, ".ovav/artifacts/")
		reason := ""
		if !allowed {
			reason = "target outside sandbox boundaries"
		}
		return BoundaryResult{Status: statusStr(allowed), Target: target, Mode: mode, AllowsWrite: true, Reason: reason}
	case ModeSourceLocalApply:
		allowed := isSafeTarget(target, mode, root)
		reason := ""
		if !allowed {
			reason = "target outside REPO_ROOT or ineligible surface"
		}
		return BoundaryResult{Status: statusStr(allowed), Target: target, Mode: mode, AllowsWrite: true, Reason: reason}
	default:
		return BoundaryResult{Status: "blocked", Target: target, Reason: "unknown_mode: " + string(mode)}
	}
}

// ValidateAllTargets validates a list of target paths against boundary rules.
func ValidateAllTargets(targets []string, mode Mode, root string) BoundaryReport {
	results := make([]BoundaryResult, 0, len(targets))
	blockedDetails := make([]BoundaryResult, 0)

	for _, t := range targets {
		result := CheckTargetBoundary(t, mode, root)
		results = append(results, result)
		if result.Status == "blocked" {
			blockedDetails = append(blockedDetails, result)
		}
	}

	status := "pass"
	if len(blockedDetails) > 0 {
		status = "fail"
	}

	return BoundaryReport{
		Status:         status,
		Mode:           mode,
		Total:          len(results),
		Allowed:        len(results) - len(blockedDetails),
		Blocked:        len(blockedDetails),
		BlockedDetails: blockedDetails,
		Results:        results,
	}
}

func statusStr(allowed bool) string {
	if allowed {
		return "ok"
	}
	return "blocked"
}
