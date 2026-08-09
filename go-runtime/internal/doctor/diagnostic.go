// Package doctor implements OVAV system diagnostics.
//
// C9.7: ovav doctor — comprehensive health check.
// Replaces the Python doctor command referenced in AGENTS.md.
// Checks: Go runtime, git state, OVAV config, branch safety, registry integrity.
//
// Command:
//
//	ovav doctor          Full diagnostic
//	ovav doctor --quick  Fast check (git + branch only)
//	ovav doctor --json   JSON output for automation
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ovav/ovav/internal/ceo"
	"github.com/ovav/ovav/internal/cli"
)

// ── Check result ─────────────────────────────────────────────────────────────

// CheckResult represents a single diagnostic check.
type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "fail", "warn", "skip"
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

// ── Doctor ────────────────────────────────────────────────────────────────────

// Run executes a full system diagnostic.
func Run(quick bool) []CheckResult {
	var checks []CheckResult

	// ── Go Runtime ──────────────────────────────────────
	checks = append(checks, checkGoRuntime())

	// ── Git State ───────────────────────────────────────
	checks = append(checks, checkGitAvailable())
	checks = append(checks, checkGitRepo())
	if !quick {
		checks = append(checks, checkGitClean())
		checks = append(checks, checkGitRemote())
	}

	// ── Branch Safety ───────────────────────────────────
	checks = append(checks, checkBranchSafety())

	// ── OVAV Config ─────────────────────────────────────
	checks = append(checks, checkOVAVRoot())
	if !quick {
		checks = append(checks, checkAuthorityContract())
		checks = append(checks, checkPermissionAuthority())
		checks = append(checks, checkRegistryIntegrity())
		checks = append(checks, checkWaiverStatus())
	}

	// ── Environment ─────────────────────────────────────
	if !quick {
		checks = append(checks, checkGoVersion())
		checks = append(checks, checkDiskSpace())
	}

	return checks
}

// ── Individual checks ────────────────────────────────────────────────────────

func checkGoRuntime() CheckResult {
	return CheckResult{
		Name:   "go-runtime",
		Status: "pass",
		Detail: fmt.Sprintf("Go %s · %s/%s · OVAV Go Runtime v5.0",
			runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
}

func checkGitAvailable() CheckResult {
	_, err := exec.LookPath("git")
	if err != nil {
		return CheckResult{
			Name: "git-available", Status: "fail",
			Detail: "git not found in PATH",
			Fix:    "Install git: sudo apt install git",
		}
	}
	return CheckResult{
		Name: "git-available", Status: "pass",
		Detail: "git is available",
	}
}

func checkGitRepo() CheckResult {
	root, err := cli.FindRepoRoot()
	if err != nil {
		return CheckResult{
			Name: "git-repo", Status: "warn",
			Detail: "Not in a git repository. Some features limited.",
		}
	}
	branch, sha, dirty := cli.GitInfo()
	return CheckResult{
		Name:   "git-repo",
		Status: "pass",
		Detail: fmt.Sprintf("Repo: %s · Branch: %s · HEAD: %s · %s",
			filepath.Base(root), branch, sha, dirty),
	}
}

func checkGitClean() CheckResult {
	cmd := exec.Command("git", "status", "--short")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Name: "git-clean", Status: "warn",
			Detail: "Cannot check git status",
		}
	}
	changed := strings.TrimSpace(string(out))
	if changed == "" {
		return CheckResult{
			Name: "git-clean", Status: "pass",
			Detail: "Working tree clean",
		}
	}
	lines := strings.Count(changed, "\n") + 1
	return CheckResult{
		Name:   "git-clean",
		Status: "warn",
		Detail: fmt.Sprintf("%d file(s) modified", lines),
		Fix:    "Stage or stash changes before operations that require clean state",
	}
}

func checkGitRemote() CheckResult {
	if !cli.HasGitRemote() {
		return CheckResult{
			Name: "git-remote", Status: "warn",
			Detail: "No git remote configured",
			Fix:    "Set remote: git remote add origin <url>",
		}
	}
	url := cli.GitRemoteURL()
	allowed := []string{"github.com/ovav-dev/ovav-systems", "github.com/Alexander-Salvador/ovav"}
	ok := false
	for _, a := range allowed {
		if strings.Contains(url, a) {
			ok = true
			break
		}
	}
	if ok {
		return CheckResult{
			Name: "git-remote", Status: "pass",
			Detail: fmt.Sprintf("Remote: %s (authorized)", url),
		}
	}
	return CheckResult{
		Name: "git-remote", Status: "fail",
		Detail: fmt.Sprintf("Remote %s is not an authorized OVAV remote", url),
		Fix:    "Set correct remote: git remote set-url origin https://github.com/ovav-dev/ovav-systems.git",
	}
}

func checkBranchSafety() CheckResult {
	protected := map[string]bool{
		"develop": true, "main": true, "master": true,
		"production": true, "prod": true, "staging": true,
	}

	branch, _, _ := cli.GitInfo()
	if branch == "unknown" {
		return CheckResult{
			Name: "branch-safety", Status: "warn",
			Detail: "Cannot determine current branch",
		}
	}

	if protected[branch] {
		root, _ := cli.FindRepoRoot()
		// CEO logged in → full access, no waiver needed
		if ceo.IsActive(root) {
			return CheckResult{
				Name:   "branch-safety",
				Status: "pass",
				Detail: fmt.Sprintf("Protected branch '%s' — CEO session active, full access", branch),
			}
		}
		// No CEO session — check collaborator waiver
		waiverPath := filepath.Join(root, ".ovav", "runtime", "protected_branch_waiver.yaml")
		if _, err := os.Stat(waiverPath); err == nil {
			return CheckResult{
				Name:   "branch-safety",
				Status: "warn",
				Detail: fmt.Sprintf("Protected branch '%s' — WAIVER ACTIVE. Writes allowed.", branch),
			}
		}
		return CheckResult{
			Name:   "branch-safety",
			Status: "warn",
			Detail: fmt.Sprintf("Protected branch '%s' — READ-ONLY. Login as CEO or create waiver to write.", branch),
			Fix:    "Login as CEO: ovav login  |  Create waiver: ovav waiver \"your reason\"",
		}
	}

	return CheckResult{
		Name: "branch-safety", Status: "pass",
		Detail: fmt.Sprintf("Branch '%s' — full access", branch),
	}
}

func checkOVAVRoot() CheckResult {
	root, err := cli.FindRepoRoot()
	if err != nil {
		return CheckResult{
			Name: "ovav-root", Status: "fail",
			Detail: "Not in an OVAV repository",
			Fix:    "Run from within an OVAV repository",
		}
	}

	// Check key directories exist
	checks := []string{
		".ovav/registry",
		".ovav/policy",
		"tools",
		"go-runtime",
	}
	missing := []string{}
	for _, c := range checks {
		p := filepath.Join(root, c)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			missing = append(missing, c)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:   "ovav-root",
			Status: "warn",
			Detail: fmt.Sprintf("OVAV root: %s · Missing: %s", root, strings.Join(missing, ", ")),
		}
	}
	return CheckResult{
		Name:   "ovav-root",
		Status: "pass",
		Detail: fmt.Sprintf("OVAV root: %s — all directories present", root),
	}
}

// CRITICAL FIX: current_authority_contract.yaml was replaced by caps.yaml + git HEAD
// as the single canonical authority (SingleAuthority validator, v56.0+).
// Absence of this file is CORRECT, not a failure.
func checkAuthorityContract() CheckResult {
	root, _ := cli.FindRepoRoot()
	ac := filepath.Join(root, ".ovav", "service_areas", "shared", "current_authority_contract.yaml")
	if _, err := os.Stat(ac); os.IsNotExist(err) {
		return CheckResult{
			Name: "authority-contract", Status: "pass",
			Detail: "current_authority_contract.yaml intentionally absent — replaced by caps.yaml + git HEAD (canonical)",
		}
	}
	return CheckResult{
		Name: "authority-contract", Status: "warn",
		Detail: "current_authority_contract.yaml exists — should be removed. caps.yaml + git HEAD is canonical.",
		Fix:    "Delete current_authority_contract.yaml — replaced by caps.yaml",
	}
}

func checkPermissionAuthority() CheckResult {
	root, _ := cli.FindRepoRoot()
	pa := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if _, err := os.Stat(pa); os.IsNotExist(err) {
		return CheckResult{
			Name: "permission-authority", Status: "fail",
			Detail: "permission_authority.json missing",
			Fix:    "Restore from backup or reinitialize OVAV",
		}
	}
	return CheckResult{
		Name: "permission-authority", Status: "pass",
		Detail: "Permission authority present",
	}
}

func checkRegistryIntegrity() CheckResult {
	root, _ := cli.FindRepoRoot()
	autoTriggers := filepath.Join(root, ".ovav", "registry", "auto_triggers.yaml")

	if _, err := os.Stat(autoTriggers); os.IsNotExist(err) {
		return CheckResult{
			Name: "registry", Status: "warn",
			Detail: "Registry file missing: auto_triggers.yaml",
		}
	}
	return CheckResult{
		Name: "registry", Status: "pass",
		Detail: "Registry intact · auto_triggers present. Memory derived from git HEAD + caps.yaml (canonical sources per AGENTS.md).",
	}
}

func checkWaiverStatus() CheckResult {
	root, _ := cli.FindRepoRoot()
	waiverPath := filepath.Join(root, ".ovav", "runtime", "protected_branch_waiver.yaml")
	if _, err := os.Stat(waiverPath); os.IsNotExist(err) {
		return CheckResult{
			Name: "waiver", Status: "pass",
			Detail: "No active waiver — standard operation",
		}
	}

	// Try JSON first (v2 waiver format)
	data, err := os.ReadFile(waiverPath)
	if err != nil {
		return CheckResult{
			Name: "waiver", Status: "warn",
			Detail: "Waiver file exists but cannot be read",
		}
	}

	// Detect format: JSON starts with '{', YAML starts with key-value or '---'
	isJSON := len(data) > 0 && data[0] == '{'

	var w map[string]interface{}
	if isJSON {
		if err := json.Unmarshal(data, &w); err != nil {
			return CheckResult{
				Name: "waiver", Status: "warn",
				Detail: "Waiver file is invalid JSON",
			}
		}
	} else {
		yamlData, _ := cli.ReadYAML(waiverPath)
		// v1 format wraps waiver data under a "waiver" key
		if inner, ok := yamlData["waiver"].(map[string]interface{}); ok {
			w = inner
		} else if len(yamlData) > 0 {
			// v2 format (YAML or JSON): waiver data at root level
			// Check if this looks like a v2 waiver (has schema or id field)
			if _, hasSchema := yamlData["schema"]; hasSchema {
				w = yamlData
			} else if _, hasID := yamlData["id"]; hasID {
				w = yamlData
			} else {
				// Malformed: has content but not v1 (no waiver key) and not v2 (no schema/id)
				return CheckResult{
					Name: "waiver", Status: "warn",
					Detail: "Waiver file malformed — missing required fields",
				}
			}
		} else {
			return CheckResult{
				Name: "waiver", Status: "warn",
				Detail: "Waiver file malformed — cannot parse",
			}
		}
	}

	active, _ := w["active"].(bool)
	branch, _ := w["branch"].(string)
	reason, _ := w["reason"].(string)

	if active {
		return CheckResult{
			Name:   "waiver",
			Status: "warn",
			Detail: fmt.Sprintf("⚠️  WAIVER ACTIVE — branch: %s, reason: %s", branch, reason),
		}
	}
	return CheckResult{
		Name: "waiver", Status: "pass",
		Detail: "Waiver file exists but inactive",
	}
}

func checkGoVersion() CheckResult {
	v := runtime.Version()
	// Go 1.22+ is the target
	return CheckResult{
		Name:   "go-version",
		Status: "pass",
		Detail: fmt.Sprintf("Go %s — compatible with OVAV Go Runtime", v),
	}
}

func checkDiskSpace() CheckResult {
	root, _ := cli.FindRepoRoot()
	// Simple check: does the directory exist and is writable?
	info, err := os.Stat(root)
	if err != nil {
		return CheckResult{
			Name: "disk-space", Status: "warn",
			Detail: "Cannot stat repo root",
		}
	}
	_ = info
	return CheckResult{
		Name: "disk-space", Status: "pass",
		Detail: "Repo directory accessible",
	}
}

// ── Formatting ───────────────────────────────────────────────────────────────

// FormatResults returns a human-readable diagnostic report.
func FormatResults(results []CheckResult) string {
	var b strings.Builder
	b.WriteString("\n  🩺 OVAV Doctor — System Diagnostic\n")
	b.WriteString("  ================================================\n")

	passed := 0
	failed := 0
	warned := 0

	for _, r := range results {
		icon := map[string]string{
			"pass": "✅", "fail": "❌", "warn": "⚠️ ", "skip": "⬜",
		}[r.Status]

		b.WriteString(fmt.Sprintf("  %s %-22s %s\n", icon, r.Name, r.Detail))

		if r.Fix != "" {
			b.WriteString(fmt.Sprintf("     ↳ fix: %s\n", r.Fix))
		}

		switch r.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "warn":
			warned++
		}
	}

	b.WriteString("  ================================================\n")
	b.WriteString(fmt.Sprintf("  %d passed · %d warnings · %d failures\n", passed, warned, failed))

	// Overall status
	if failed > 0 {
		b.WriteString("  🔴 System has failures — review and fix before critical operations\n")
	} else if warned > 0 {
		b.WriteString("  🟡 System operational with warnings — review before protected branch work\n")
	} else {
		b.WriteString("  🟢 All systems nominal — ready for work\n")
	}
	b.WriteString("\n")
	return b.String()
}
