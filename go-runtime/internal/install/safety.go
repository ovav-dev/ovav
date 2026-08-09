package install

import (
	"sort"
	"strings"
)

// EvaluateSafety evaluates the safety of an install plan for the given mode.
func EvaluateSafety(plan Plan) SafetyReport {
	entries := plan.Entries
	issues := make([]string, 0)
	safetyEntries := make([]SafetyEntry, 0, len(entries))

	hasBlocked := false
	hasReview := false
	needsBackup := false
	needsRollback := false

	for _, entry := range entries {
		target := entry.Target
		risk := entry.TargetRisk
		if risk == "" {
			risk = classifyRisk(target)
		}
		safetyStatus := RiskMap[risk]
		if safetyStatus == "" {
			safetyStatus = "blocked"
		}

		switch safetyStatus {
		case "blocked":
			hasBlocked = true
		case "review_required":
			hasReview = true
		}

		if plan.Mode == ModeSourceLocalApply {
			needsBackup = true
			needsRollback = true
		}

		// Check for unsafe selectors in the target path
		for _, unsafe := range UnsafeSelectors {
			if strings.Contains(strings.ToLower(target), unsafe) {
				issues = append(issues, "unsafe_selectors_in_target: "+unsafe)
				safetyStatus = "blocked"
				hasBlocked = true
			}
		}

		writeAllowed := safetyStatus == "allow" && plan.Mode != ModeDryRun

		safetyEntries = append(safetyEntries, SafetyEntry{
			Target:       target,
			Risk:         risk,
			SafetyStatus: safetyStatus,
			Mode:         plan.Mode,
			WriteAllowed: writeAllowed,
		})
	}

	// Deduplicate and sort issues
	issueSet := make(map[string]bool)
	for _, i := range issues {
		issueSet[i] = true
	}
	uniqueIssues := make([]string, 0, len(issueSet))
	for i := range issueSet {
		uniqueIssues = append(uniqueIssues, i)
	}
	sort.Strings(uniqueIssues)

	// Overall safety
	overall := "allow"
	if hasBlocked {
		overall = "blocked"
	} else if hasReview {
		overall = "review_required"
	}

	status := "pass"
	if len(uniqueIssues) > 0 {
		status = "fail"
	}

	return SafetyReport{
		Status:           status,
		Mode:             plan.Mode,
		OverallSafety:    overall,
		HasBlocked:       hasBlocked,
		HasReviewReq:     hasReview,
		NeedsBackup:      needsBackup,
		NeedsRollback:    needsRollback,
		Entries:          safetyEntries,
		Issues:           uniqueIssues,
		RealApplyAllowed: plan.Mode == ModeSourceLocalApply && overall == "allow",
	}
}
