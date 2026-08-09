package hooks

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Stage Filter ───────────────────────────────────────────────────────────────

// StageFilter maps hook stages to validator IDs based on auto_triggers.yaml.
// This ensures hooks and owv use the SAME configuration source — no duplication.
type StageFilter []string

// normalizeName strips Python auto_triggers naming conventions to get a
// comparable core name that can be matched against Go validator IDs.
func normalizeName(name string) string {
	// Strip common prefixes
	for _, prefix := range []string{"check_", "validate_"} {
		if strings.HasPrefix(name, prefix) {
			name = strings.TrimPrefix(name, prefix)
			break
		}
	}
	// Strip common suffixes
	for _, suffix := range []string{"_gate", "_validator", "_check"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

// Matches returns true if the validator ID should run for this stage.
// It handles naming convention differences between auto_triggers.yaml
// (uses Python-style names like "check_secrets_hygiene") and Go validators
// (use concise names like "secrets_hygiene").
func (sf StageFilter) Matches(validatorID string) bool {
	normalizedID := normalizeName(validatorID)
	for _, name := range sf {
		// Exact match
		if name == validatorID {
			return true
		}
		// Normalized comparison (handles check_*, validate_*, *_gate, *_validator)
		if normalizeName(name) == normalizedID {
			return true
		}
	}
	return false
}

// ── Canonical stage-to-validator mapping ──────────────────────────────────────

// autoTriggersStageMap maps git hook stages to their auto_trigger events.
var autoTriggersStageMap = map[Stage]string{
	StagePreCommit: "before_git_stage",
	StagePrePush:   "before_git_push",
}

// GetStageFilter returns the list of validator IDs that should run for a given stage.
// It reads from auto_triggers.yaml (canonical source) with a hardcoded fallback
// for when the file is unavailable or the system is bootstrapping.
func GetStageFilter(stage Stage) StageFilter {
	// Try to read from auto_triggers.yaml
	if filter := loadStageFilterFromYAML(stage); len(filter) > 0 {
		return filter
	}
	// Hardcoded fallback for bootstrap / missing config
	return hardcodedStageFilter(stage)
}

// loadStageFilterFromYAML reads the auto_triggers.yaml file and extracts
// validator names for the given stage's corresponding event.
func loadStageFilterFromYAML(stage Stage) StageFilter {
	eventName, ok := autoTriggersStageMap[stage]
	if !ok {
		return nil
	}

	// Find the repo root from the current working directory
	// The hooks package doesn't have a direct repo root reference here,
	// but the Manager does. We use a simple search from CWD.
	cwd, _ := os.Getwd()
	repoRoot := findRepoRoot(cwd)
	if repoRoot == "" {
		return nil
	}

	yamlPath := filepath.Join(repoRoot, ".ovav", "registry", "auto_triggers.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil
	}

	// Parse the YAML to extract validator names for this event
	return parseAutoTriggersForEvent(data, eventName)
}

// parseAutoTriggersForEvent parses auto_triggers.yaml looking for a specific event.
// It extracts the list of validator names (the module/function references).
func parseAutoTriggersForEvent(data []byte, eventName string) StageFilter {
	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}

	router, ok := root["router"].(map[string]interface{})
	if !ok {
		return nil
	}

	eventList, ok := router[eventName]
	if !ok {
		return nil
	}

	// The value can be []interface{} in parsed YAML
	list, ok := eventList.([]interface{})
	if !ok {
		return nil
	}

	var filter StageFilter
	for _, item := range list {
		if s, ok := item.(string); ok {
			filter = append(filter, s)
		}
	}
	return filter
}

// findRepoRoot walks up from the given path looking for a .git directory.
func findRepoRoot(from string) string {
	for {
		if _, err := os.Stat(filepath.Join(from, ".git")); err == nil {
			return from
		}
		parent := filepath.Dir(from)
		if parent == from {
			return ""
		}
		from = parent
	}
}

// ── Hardcoded fallback ─────────────────────────────────────────────────────────
// These are the minimum validators that MUST run for each stage.
// When auto_triggers.yaml is unavailable (bootstrap, corrupted config),
// this hardcoded list ensures security validation still runs.
// UPDATED: Keep in sync with auto_triggers.yaml. Last sync: 2026-06-19.

func hardcodedStageFilter(stage Stage) StageFilter {
	switch stage {
	case StagePreCommit:
		return StageFilter{
			"check_protected_branch",
			"check_secrets_hygiene",
			"validate_workspace_safety_gate",
			"branch_shield",
			"surface_governor",
		}
	case StagePrePush:
		return StageFilter{
			"check_secrets_hygiene",
			"validate_workspace_safety_gate",
			"branch_shield",
			"check_release_gate",
			"pre_push_intelligence",
		}
	default:
		return nil
	}
}
