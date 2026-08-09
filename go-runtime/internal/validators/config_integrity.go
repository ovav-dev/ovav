package validators

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ConfigIntegrity validates configuration files for syntax correctness,
// YAML validity, and canonical source integrity.
// Replaces: check_config_syntax.py, check_canonical_integrity.py,
// check_bootstrap_chain.py, check_f1_architecture.py
type ConfigIntegrity struct{}

func NewConfigIntegrity() *ConfigIntegrity { return &ConfigIntegrity{} }

func (c *ConfigIntegrity) ID() string   { return "config_integrity" }
func (c *ConfigIntegrity) Name() string { return "Config Integrity" }
func (c *ConfigIntegrity) Description() string {
	return "Validates YAML/JSON config syntax, canonical sources, and bootstrap chain"
}
func (c *ConfigIntegrity) Weight() int { return 15 }

// Required config files for OVAV operation.
var requiredConfigs = []struct {
	path   string
	label  string
	isJSON bool
}{
	{".ovav/plan/caps.yaml", "Canonical plan (caps.yaml)", false},
	{".ovav/laws/ovav_laws.yaml", "OVAV laws", false},
	{".ovav/policy/permission_authority.json", "Permission authority", true},
	{"opencode.json", "OpenCode config", true},
	{"AGENTS.md", "Agent instructions", false},
}

// Canonical source files — must not have duplicates.
var canonicalSources = []struct {
	path  string
	label string
}{
	{".ovav/plan/caps.yaml", "Implementation plan"},
	{".ovav/policy/permission_authority.json", "Permission authority"},
	{"AGENTS.md", "Agent instructions"},
}

func (c *ConfigIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check required config files exist and are non-empty
	for _, cfg := range requiredConfigs {
		fullPath := filepath.Join(root, cfg.path)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING: %s (%s)", cfg.path, cfg.label))
			continue
		}
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot stat %s: %v", cfg.path, err))
			continue
		}
		if info.Size() == 0 {
			issues = append(issues, fmt.Sprintf("EMPTY: %s (%s) — file is empty", cfg.path, cfg.label))
		}
		// Quick YAML/JSON syntax check
		if cfg.isJSON {
			data, _ := os.ReadFile(fullPath)
			if len(data) > 0 && data[0] != '{' && data[0] != '[' {
				issues = append(issues, fmt.Sprintf("SYNTAX: %s does not appear to be valid JSON", cfg.path))
			}
		}
	}

	// 2. Check canonical sources are not duplicated (deprecated copies)
	deprecatedPaths := map[string]string{
		"IMPLEMENTATION_PLAN.md":                           ".ovav/plan/caps.yaml (canonical)",
		"docs/implementation/07_IMPLEMENTATION_ROADMAP.md": ".ovav/plan/caps.yaml (canonical)",
		"current_authority_contract.yaml":                  ".ovav/plan/caps.yaml (canonical)",
		"derived_artifacts.yaml":                           ".ovav/plan/caps.yaml (canonical)",
	}
	for deprecated, canonical := range deprecatedPaths {
		if _, err := os.Stat(filepath.Join(root, deprecated)); err == nil {
			issues = append(issues, fmt.Sprintf("DEPRECATED: %s exists — use %s instead", deprecated, canonical))
		}
	}

	// 3. Check VERSION file consistency + cross-validate with git tags and source code
	versionPath := filepath.Join(root, "VERSION")
	var fileVersion string
	if data, err := os.ReadFile(versionPath); err == nil {
		fileVersion = strings.TrimSpace(string(data))
		if fileVersion == "" {
			issues = append(issues, "EMPTY: VERSION file is empty")
		} else if !strings.HasPrefix(fileVersion, "2.") && !strings.HasPrefix(fileVersion, "3.") {
			issues = append(issues, fmt.Sprintf("VERSION: expected 2.x.x or 3.x.x, got %s", fileVersion))
		}
	} else {
		issues = append(issues, "MISSING: VERSION file not found")
	}

	// 3a. Cross-check VERSION against latest git tag
	if fileVersion != "" {
		if tag, err := getLatestGitTag(root); err == nil && tag != "" {
			tagVersion := strings.TrimPrefix(tag, "v")
			// Strip product prefix if present (e.g., "pi-memory-v3.0.0" -> "3.0.0")
			if idx := strings.LastIndex(tagVersion, "-"); idx >= 0 {
				tagVersion = strings.TrimPrefix(tagVersion[idx+1:], "v")
			}
			if tagVersion != fileVersion {
				issues = append(issues, fmt.Sprintf(
					"VERSION_MISMATCH: VERSION file says '%s' but latest git tag is '%s' (expected '%s')",
					fileVersion, tag, tagVersion))
			}
		}
	}

	// 3b. Cross-check VERSION against cpanel shared.go Version constant
	if fileVersion != "" {
		sharedGoPath := filepath.Join(root, "go-runtime", "cmd", "cpanel", "shared.go")
		if sharedData, err := os.ReadFile(sharedGoPath); err == nil {
			re := regexp.MustCompile(`Version\s*=\s*"([^"]+)"`)
			if matches := re.FindStringSubmatch(string(sharedData)); len(matches) >= 2 {
				sharedVersion := matches[1]
				if sharedVersion != fileVersion {
					issues = append(issues, fmt.Sprintf(
						"VERSION_MISMATCH: VERSION file says '%s' but go-runtime/cmd/cpanel/shared.go has Version = '%s'",
						fileVersion, sharedVersion))
				}
			}
		}
	}

	// 3c. Check RELEASE_NOTES.md for references to deleted files
	releaseNotesPath := filepath.Join(root, "docs", "RELEASE_NOTES.md")
	if rnData, err := os.ReadFile(releaseNotesPath); err == nil {
		rnContent := string(rnData)
		staleRefs := map[string]string{
			"IMPLEMENTATION_PLAN.md":          "reemplazado por caps.yaml",
			"current_authority_contract.yaml": "reemplazado por caps.yaml",
			"derived_artifacts.yaml":          "deprecado en v2.0",
		}
		for ref, reason := range staleRefs {
			if strings.Contains(rnContent, ref) && !strings.Contains(rnContent, reason) {
				issues = append(issues, fmt.Sprintf(
					"STALE_REF: docs/RELEASE_NOTES.md references '%s' (%s) — add historical context marker",
					ref, reason))
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: c.ID(), Name: c.Name(), Status: "fail", Weight: c.Weight(),
			Message: fmt.Sprintf("FAIL config integrity — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: c.ID(), Name: c.Name(), Status: "pass", Weight: c.Weight(),
		Message:  "PASS config integrity — all required configs present and valid",
		Duration: time.Since(start),
	}
}

// getLatestGitTag returns the most recent git tag (e.g., "v2.1.0").
// Uses git tag sorted by creation date (all tags, not just reachable).
// dir specifies the working directory for git commands.
// Returns empty string if git is unavailable or no tags exist.
func getLatestGitTag(dir string) (string, error) {
	// Use git tag --sort=-creatordate to get all tags sorted by creation date.
	// This is more reliable than git describe (which only finds reachable tags).
	cmd := exec.Command("git", "-C", dir, "tag", "--sort=-creatordate")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0], nil
	}
	return "", fmt.Errorf("no tags found")
}

var _ Validator = (*ConfigIntegrity)(nil)
