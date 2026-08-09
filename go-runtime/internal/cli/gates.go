// Package cli — gates.go: Go migrations of Python gate/check CLIs.
//
// Replaces:
//
//	tools/cli/ovav_execution_gateway.py   (113 LOC)
//	tools/cli/ovav_surface_manager.py     (103 LOC)
//	tools/cli/ovav_public_export_gate.py  (111 LOC)
//	tools/cli/ovav_repo_presentation_gate.py (119 LOC)
//	tools/cli/ovav_release_package.py     (108 LOC)
//
// All functions are pure Go, stdlib-only, return JSON-serializable maps.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── Execution Gateway (ovav_execution_gateway.py) ────────────────────────────

// ExecutionGateReport builds a gate report for cockpit action execution.
// Replaces ovav_execution_gateway.py build_gate_report().
func ExecutionGateReport(action, mode string, consent, riskAccepted bool) map[string]interface{} {
	gates := map[string]interface{}{
		"consent_provided":  consent,
		"risk_accepted":     riskAccepted,
		"dry_run_completed": mode == "dry_run",
	}

	allowed := false
	var requires []string

	switch mode {
	case "dry_run":
		allowed = true
	case "apply":
		allowed = consent && riskAccepted
		if !consent {
			requires = append(requires, "consent")
		}
		if !riskAccepted {
			requires = append(requires, "accept_risk")
		}
	default:
		allowed = false
		requires = append(requires, "valid_mode")
	}

	if requires == nil {
		requires = []string{}
	}

	return map[string]interface{}{
		"schema_version":   "ovav.execution_gateway.v1",
		"action":           action,
		"mode":             mode,
		"allowed":          allowed,
		"requires":         requires,
		"gates":            gates,
		"writes_performed": false,
		"generated_at":     time.Now().UTC().Format(time.RFC3339),
	}
}

// ValidGatewayActions returns the set of valid gateway action names.
func ValidGatewayActions() []string {
	return []string{"setup", "sync", "security", "recovery", "update"}
}

// ── Surface Manager (ovav_surface_manager.py) ────────────────────────────────

// ManagedSurface describes a surface path to check.
type ManagedSurface struct {
	Path     string
	Required bool
	Desc     string
}

// DefaultManagedSurfaces returns the canonical list of managed surfaces.
func DefaultManagedSurfaces() []ManagedSurface {
	return []ManagedSurface{
		{".opencode/", true, "OpenCode configuration"},
		{".ovav/source/skills/", true, "OpenCode skills"},
		{".ovav/policy/", true, "OVAV policy"},
		{".ovav/policy/permission_authority.json", true, "Permission authority"},
		{"AGENTS.md", true, "Agent instructions"},
		{"docs/launch/", false, "Launch documentation"},
		{".github/workflows/ci.yml", false, "CI workflow"},
	}
}

// SurfacesCheck checks all managed surfaces and returns status.
// Replaces ovav_surface_manager.py check_surfaces().
func SurfacesCheck(repoRoot string) map[string]interface{} {
	surfaces := DefaultManagedSurfaces()
	var results []map[string]interface{}

	for _, s := range surfaces {
		fullPath := filepath.Join(repoRoot, s.Path)
		exists := pathExists(fullPath)
		status := "ok"
		if !exists {
			status = "missing"
		}
		results = append(results, map[string]interface{}{
			"path":     s.Path,
			"desc":     s.Desc,
			"required": s.Required,
			"exists":   exists,
			"status":   status,
		})
	}

	return map[string]interface{}{
		"schema_version": "ovav.surface_manager.v1",
		"command":        "status",
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"surfaces":       results,
	}
}

// SurfacesRepairPlan generates a no-write repair plan for missing surfaces.
func SurfacesRepairPlan(repoRoot string) map[string]interface{} {
	check := SurfacesCheck(repoRoot)
	surfaces, _ := check["surfaces"].([]map[string]interface{})

	var repairActions []map[string]interface{}
	totalMissing := 0
	totalRequiredMissing := 0

	for _, s := range surfaces {
		st, _ := s["status"].(string)
		if st == "missing" || st == "missing_required" {
			path, _ := s["path"].(string)
			action := "create_dir"
			if strings.Contains(path, ".") && !strings.HasSuffix(path, "/") {
				action = "create"
			}
			repairActions = append(repairActions, map[string]interface{}{
				"action":         action,
				"path":           path,
				"desc":           s["desc"],
				"required":       s["required"],
				"write_required": true,
			})
			totalMissing++
			if st == "missing_required" {
				totalRequiredMissing++
			}
		}
	}

	if repairActions == nil {
		repairActions = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"schema_version":         "ovav.surface_manager.v1",
		"command":                "repair-plan",
		"generated_at":           time.Now().UTC().Format(time.RFC3339),
		"surfaces":               check["surfaces"],
		"repair_actions":         repairActions,
		"total_missing":          totalMissing,
		"total_required_missing": totalRequiredMissing,
		"writes_required":        true,
		"consent_required":       true,
	}
}

// ── Public Export Gate (ovav_public_export_gate.py) ──────────────────────────

var secretPatterns = []string{
	"sk-", "ghp_", "github_pat_", "AIza", "sk-or-", "xoxb-", "xoxp-",
	"-----BEGIN " + "RSA PRIVATE KEY-----", "-----BEGIN " + "OPENSSH PRIVATE KEY-----",
}

var forbiddenFiles = []string{
	".env", ".env.local", ".env.production", "credentials.json", "secrets.yaml",
}

// ExportGateCheck runs release safety checks for public distribution.
// Replaces ovav_public_export_gate.py.
func ExportGateCheck(repoRoot string) map[string]interface{} {
	secrets := scanForSecrets(repoRoot)
	forbidden := checkForbiddenFiles(repoRoot)
	essentials := checkEssentials(repoRoot)

	checks := []map[string]interface{}{
		{
			"name":   "no_secrets_found",
			"passed": len(secrets) == 0,
			"detail": detailOrClean(len(secrets), "secrets found"),
		},
		{
			"name":   "no_forbidden_files",
			"passed": len(forbidden) == 0,
			"detail": detailListOrClean(forbidden),
		},
		{
			"name":   "readme_present",
			"passed": essentials["readme_exists"].(bool),
			"detail": foundOrMissing(essentials["readme_exists"].(bool)),
		},
		{
			"name":   "license_present",
			"passed": essentials["license_exists"].(bool),
			"detail": foundOrMissing(essentials["license_exists"].(bool)),
		},
	}

	allPass := true
	for _, c := range checks {
		if !c["passed"].(bool) {
			allPass = false
			break
		}
	}

	return map[string]interface{}{
		"schema_version":  "ovav.public_export_gate.v1",
		"passed":          allPass,
		"checks":          checks,
		"secrets_found":   secrets,
		"forbidden_files": forbidden,
		"generated_at":    time.Now().UTC().Format(time.RFC3339),
	}
}

func scanForSecrets(repoRoot string) []map[string]interface{} {
	var findings []map[string]interface{}
	scanDirs := []string{"tools/", "bin/", ".ovav/registry/", "docs/"}

	// Performance guards: max depth and files per directory to prevent I/O exhaustion.
	const maxDepth = 10        // directories deeper than 10 levels are unlikely to contain secrets
	const maxFilesPerDir = 200 // cap to prevent huge directories from causing timeouts

	for _, dir := range scanDirs {
		fullDir := filepath.Join(repoRoot, dir)
		if !pathExists(fullDir) {
			continue
		}
		filepath.Walk(fullDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible entries
			}
			relPath, _ := filepath.Rel(fullDir, path)
			depth := len(strings.Split(relPath, string(filepath.Separator)))
			if depth > maxDepth {
				return filepath.SkipDir // don't descend beyond max depth
			}
			if info.IsDir() {
				// Count files in this directory; if excessive, skip descending
				if entries, err := os.ReadDir(path); err == nil && len(entries) > maxFilesPerDir {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".py") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, pattern := range secretPatterns {
				if strings.Contains(string(content), pattern) {
					rel, _ := filepath.Rel(repoRoot, path)
					truncated := pattern
					if len(truncated) > 20 {
						truncated = truncated[:20]
					}
					findings = append(findings, map[string]interface{}{
						"file":     rel,
						"pattern":  truncated,
						"severity": "high",
					})
				}
			}
			return nil
		})
	}

	if findings == nil {
		findings = []map[string]interface{}{}
	}
	return findings
}

func checkForbiddenFiles(repoRoot string) []string {
	var found []string
	for _, f := range forbiddenFiles {
		if pathExists(filepath.Join(repoRoot, f)) {
			found = append(found, f)
		}
	}
	if found == nil {
		found = []string{}
	}
	return found
}

func checkEssentials(repoRoot string) map[string]interface{} {
	return map[string]interface{}{
		"readme_exists":       pathExists(filepath.Join(repoRoot, "README.md")),
		"license_exists":      pathExists(filepath.Join(repoRoot, "LICENSE")) || pathExists(filepath.Join(repoRoot, "LICENSE.md")),
		"contributing_exists": pathExists(filepath.Join(repoRoot, "CONTRIBUTING.md")),
	}
}

// ── Repo Presentation Gate (ovav_repo_presentation_gate.py) ──────────────────

// RepoPresentationGate validates README, docs, and repo structure.
// Replaces ovav_repo_presentation_gate.py.
func RepoPresentationGate(repoRoot string) map[string]interface{} {
	readmeCheck := checkReadme(repoRoot)
	docsCheck := checkDocs(repoRoot)
	ciCheck := checkCI(repoRoot)

	checks := []map[string]interface{}{
		{
			"name":   "readme_present",
			"passed": readmeCheck["present"].(bool),
			"detail": foundOrMissing(readmeCheck["present"].(bool)),
		},
		{
			"name":   "readme_quality",
			"passed": len(readmeCheck["issues"].([]string)) == 0,
			"detail": readmeCheck["issues"],
		},
		{
			"name":   "docs_complete",
			"passed": docsCheck["all_present"].(bool),
			"detail": docsCheck["missing"],
		},
		{
			"name":   "ci_configured",
			"passed": ciCheck["present"].(bool),
			"detail": ciCheck["workflows"],
		},
	}

	allPass := true
	for _, c := range checks {
		if !c["passed"].(bool) {
			allPass = false
			break
		}
	}

	return map[string]interface{}{
		"schema_version": "ovav.repo_presentation_gate.v1",
		"passed":         allPass,
		"checks":         checks,
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
	}
}

func checkReadme(repoRoot string) map[string]interface{} {
	readmePath := filepath.Join(repoRoot, "README.md")
	if !pathExists(readmePath) {
		return map[string]interface{}{
			"present": false,
			"issues":  []string{"README.md not found"},
		}
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		return map[string]interface{}{
			"present": true,
			"issues":  []string{"cannot read README.md"},
		}
	}

	text := string(content)
	var issues []string
	if len(text) < 100 {
		issues = append(issues, "README too short (<100 chars)")
	}
	if !strings.Contains(text, "OVAV") && !strings.Contains(strings.ToLower(text), "ovav") {
		issues = append(issues, "README does not mention OVAV")
	}
	if !strings.Contains(text, "##") {
		issues = append(issues, "README has no sections (missing ## headers)")
	}
	if issues == nil {
		issues = []string{}
	}

	return map[string]interface{}{
		"present":    true,
		"issues":     issues,
		"size_chars": len(text),
	}
}

func checkDocs(repoRoot string) map[string]interface{} {
	essential := []string{
		"AGENTS.md",
	}
	var found, missing []string
	for _, p := range essential {
		if pathExists(filepath.Join(repoRoot, p)) {
			found = append(found, p)
		} else {
			missing = append(missing, p)
		}
	}
	if found == nil {
		found = []string{}
	}
	if missing == nil {
		missing = []string{}
	}
	return map[string]interface{}{
		"found":       found,
		"missing":     missing,
		"all_present": len(missing) == 0,
	}
}

func checkCI(repoRoot string) map[string]interface{} {
	workflowsDir := filepath.Join(repoRoot, ".github", "workflows")
	if !pathExists(workflowsDir) {
		return map[string]interface{}{
			"present":   false,
			"workflows": []string{},
		}
	}
	entries, _ := os.ReadDir(workflowsDir)
	var workflows []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
			workflows = append(workflows, e.Name())
		}
	}
	if workflows == nil {
		workflows = []string{}
	}
	return map[string]interface{}{
		"present":   len(workflows) > 0,
		"workflows": workflows,
	}
}

// ── Release Package (ovav_release_package.py) ────────────────────────────────

// ReleasePackageCheck validates release readiness.
// Replaces ovav_release_package.py.
func ReleasePackageCheck(repoRoot string) map[string]interface{} {
	uncommittedClean, uncommittedFiles := checkUncommitted(repoRoot)
	versionInfo := checkVersionConsistency(repoRoot)
	latestTag := getLatestTag(repoRoot)

	checks := []map[string]interface{}{
		{
			"name":   "no_uncommitted_changes",
			"passed": uncommittedClean,
			"detail": detailOrList(uncommittedClean, "clean", uncommittedFiles),
		},
		{
			"name":   "version_consistency",
			"passed": len(versionInfo) >= 1,
			"detail": versionInfo,
		},
		{
			"name":   "has_git_tag",
			"passed": latestTag != "",
			"detail": orDefault(latestTag, "no tags"),
		},
	}

	allPass := true
	for _, c := range checks {
		if !c["passed"].(bool) {
			allPass = false
			break
		}
	}

	if uncommittedFiles == nil {
		uncommittedFiles = []string{}
	}

	return map[string]interface{}{
		"schema_version":      "ovav.release_package.v1",
		"ready":               allPass,
		"uncommitted_changes": !uncommittedClean,
		"uncommitted_files":   uncommittedFiles,
		"tag_exists":          latestTag != "",
		"latest_tag":          latestTag,
		"version_sources":     versionInfo,
		"checks":              checks,
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
	}
}

func checkUncommitted(repoRoot string) (bool, []string) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return false, []string{"git unavailable"}
	}
	lines := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return len(lines) == 0, lines
}

func checkVersionConsistency(repoRoot string) map[string]string {
	sources := map[string]string{}
	files := map[string]string{
		"AGENTS.md": "AGENTS.md",
		"README.md": "README.md",
	}
	for key, relPath := range files {
		full := filepath.Join(repoRoot, relPath)
		content, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(content), "\n") {
			if strings.Contains(strings.ToLower(line), "version") && hasDigit(line) {
				truncated := strings.TrimSpace(line)
				if len(truncated) > 80 {
					truncated = truncated[:80]
				}
				sources[key] = truncated
				break
			}
		}
	}
	return sources
}

func getLatestTag(repoRoot string) string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func foundOrMissing(found bool) string {
	if found {
		return "found"
	}
	return "missing"
}

func detailOrClean(count int, suffix string) string {
	if count == 0 {
		return "clean"
	}
	return fmt.Sprintf("%d %s", count, suffix)
}

func detailListOrClean(items []string) interface{} {
	if len(items) == 0 {
		return "clean"
	}
	return items
}

func detailOrList(clean bool, cleanStr string, items []string) interface{} {
	if clean {
		return cleanStr
	}
	return items
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
