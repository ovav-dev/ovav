// internal/security/branch_types.go
// ============================================================
// OVAV Branch Types — Security Surface Bridge (Go native)
// ============================================================
// Migrated from tools/git/branch/types.py (SSOT)
// All branch taxonomy, lifetime policies, tier assignments,
// and helper functions live here.
// ============================================================

package security

import (
	"regexp"
	"strings"
)

// Protected branches (never force-push, never delete)
var ProtectedBranches = map[string]struct{}{
	"main":      {},
	"master":    {},
	"develop":   {},
	"staging":   {},
	"prod":      {},
	"production": {},
}

// Protected prefixes (patterns that create protected branches)
var ProtectedPrefixes = map[string]struct{}{
	"main":      {},
	"master":    {},
	"develop":   {},
	"staging":   {},
	"prod":      {},
	"production": {},
	"release/":  {},
	"hotfix/":   {},
}

// UltraBranches — highest criticality (main, master only)
var UltraBranches = map[string]struct{}{
	"main":   {},
	"master": {},
}

// HighBranches — high criticality (release/hotfix branches)
var HighBranches = map[string]struct{}{
	"release/": {},
	"hotfix/":  {},
}

// ProductionBranches — directly affect production
var ProductionBranches = map[string]struct{}{
	"main":       {},
	"master":     {},
	"prod":       {},
	"production": {},
}

// PrimaryBranches — main development targets
var PrimaryBranches = map[string]struct{}{
	"main":     {},
	"master":   {},
	"develop":  {},
	"staging":  {},
}

// WorkBranchPrefixes
var WorkBranchPrefixes = map[string]struct{}{
	"feature/":   {},
	"fix/":       {},
	"hotfix/":    {},
	"refactor/":  {},
	"docs/":      {},
	"test/":      {},
	"ci/":        {},
}

// AllBranchKeys
var AllBranchKeys = map[string]struct{}{
	"ultra":       {},
	"high":        {},
	"production":  {},
	"primary":     {},
	"work":        {},
	"experimental": {},
}

// CCBranchMap — Conventional commits branch map
var CCBranchMap = map[string]string{
	"feature":  "feature/",
	"fix":      "fix/",
	"hotfix":   "hotfix/",
	"refactor": "refactor/",
	"docs":     "docs/",
	"test":     "test/",
	"ci":       "ci/",
	"build":    "build/",
	"chore":    "chore/",
	"perf":     "perf/",
	"security": "security/",
}

// OVAVBranchMap — OVAV-specific branch map
var OVAVBranchMap = map[string]string{
	"feature":   "feature/",
	"fix":       "fix/",
	"hotfix":    "hotfix/",
	"feature/":  "feature/",
	"fix/":      "fix/",
}

// TaskPrefixes
var TaskPrefixes = map[string]struct{}{
	"T":     {},
	"TASK":  {},
	"BUG":   {},
	"FIX":   {},
	"HOTFIX": {},
}

// ValidTaskPatterns — compiled regex for task patterns
var ValidTaskPatterns = regexp.MustCompile(`^(T|TASK|BUG|FIX|HOTFIX)[-_]\d+(\.\d+)*$`)

// LifetimePolicies — in days
var LifetimePolicies = map[string]int{
	"ultra":        0,   // Never expires
	"high":         30,  // 30 days
	"production":   90,  // 90 days
	"primary":      180, // 6 months
	"work":         14,  // 2 weeks
	"experimental": 7,   // 1 week
}

// PrefixTierMap — prefix to tier mapping
var PrefixTierMap = map[string]string{
	"main":      "ultra",
	"master":    "ultra",
	"release/":  "high",
	"hotfix/":   "high",
	"prod":      "production",
	"production": "production",
	"develop":   "primary",
	"staging":   "primary",
	"feature/":  "work",
	"fix/":      "work",
	"refactor/": "work",
	"docs/":     "work",
	"test/":     "work",
	"ci/":       "work",
}

// IsWorkBranch checks if branch is a work branch (feature/fix/etc.)
func IsWorkBranch(branch string) bool {
	for prefix := range WorkBranchPrefixes {
		if strings.HasPrefix(branch, prefix) {
			return true
		}
	}
	return false
}

// IsProtectedBranch checks if branch is a protected branch
func IsProtectedBranch(branch string) bool {
	if _, ok := ProtectedBranches[branch]; ok {
		return true
	}
	for prefix := range ProtectedPrefixes {
		if strings.HasPrefix(branch, prefix) {
			return true
		}
	}
	return false
}

// IsPrimaryBranch checks if branch is a primary branch
func IsPrimaryBranch(branch string) bool {
	_, ok := PrimaryBranches[branch]
	return ok
}

// GetBranchTier gets the tier of a branch
func GetBranchTier(branch string) string {
	if _, ok := UltraBranches[branch]; ok {
		return "ultra"
	}
	if _, ok := ProductionBranches[branch]; ok {
		return "production"
	}
	for prefix := range HighBranches {
		if strings.HasPrefix(branch, prefix) {
			return "high"
		}
	}
	if _, ok := PrimaryBranches[branch]; ok {
		return "primary"
	}
	if IsWorkBranch(branch) {
		return "work"
	}
	return "experimental"
}

// GetBranchPrefix gets the prefix of a branch
func GetBranchPrefix(branch string) string {
	for prefix := range WorkBranchPrefixes {
		if strings.HasPrefix(branch, prefix) {
			return prefix
		}
	}
	return ""
}

// GetBranchTypeKey gets the branch type key for a branch
func GetBranchTypeKey(branch string) string {
	return GetBranchTier(branch)
}

// GetLifetimePolicy gets the lifetime policy in days for a branch
func GetLifetimePolicy(branch string) int {
	tier := GetBranchTier(branch)
	if days, ok := LifetimePolicies[tier]; ok {
		return days
	}
	return 7 // default experimental
}

// ValidateBranchPrefix validates that a branch has a correct prefix
func ValidateBranchPrefix(branch string) bool {
	if IsProtectedBranch(branch) {
		return true
	}
	if IsWorkBranch(branch) {
		return true
	}
	return false
}
