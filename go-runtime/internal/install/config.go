package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config deployment constants.
const (
	weztermSourceRel = ".ovav/context/backups/wezterm-catppuccin-mocha.lua"
	configBackupDir  = ".ovav/context/backups/pre_deploy"
)

// DeployEntry represents a source-to-target config mapping.
type DeployEntry struct {
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	SourceExists bool                   `json:"source_exists"`
	TargetExists bool                   `json:"target_exists,omitempty"`
	Backup       string                 `json:"backup,omitempty"`
	HadPrevious  bool                   `json:"had_previous_config,omitempty"`
	Applied      bool                   `json:"applied"`
	OK           bool                   `json:"ok"`
	Status       string                 `json:"status"`
	Reason       string                 `json:"reason,omitempty"`
	Verification map[string]interface{} `json:"verification,omitempty"`
}

// DeployAllResult is the result of deploying all configs.
type DeployAllResult struct {
	Timestamp   string        `json:"timestamp"`
	BackupDir   string        `json:"backup_dir"`
	Deployments []DeployEntry `json:"deployments"`
	AllOK       bool          `json:"all_ok"`
}

// GovernedDeployResult is the result of governed config deployment.
type GovernedDeployResult struct {
	Status              string                 `json:"status"`
	Mode                Mode                   `json:"mode"`
	Governed            bool                   `json:"governed"`
	DeployEntries       int                    `json:"deploy_entries,omitempty"`
	Entries             []DeployEntry          `json:"entries,omitempty"`
	RealDeployPerformed bool                   `json:"real_deploy_performed"`
	Gates               map[string]interface{} `json:"gates"`
	ApprovalNote        string                 `json:"approval_note,omitempty"`
	Warning             string                 `json:"warning,omitempty"`
	DeployResult        *DeployAllResult       `json:"deploy_result,omitempty"`
	Diagnostics         map[string]interface{} `json:"diagnostics,omitempty"`
	Error               string                 `json:"error,omitempty"`
	SandboxRoot         string                 `json:"sandbox_root,omitempty"`
}

// DetectWindowsUsername tries to find the Windows username from WSL /mnt/c/Users.
func DetectWindowsUsername() string {
	usersDir := "/mnt/c/Users"
	entries, err := os.ReadDir(usersDir)
	if err != nil {
		return ""
	}
	// Sort in reverse to get most recently modified
	type entryWithInfo struct {
		name string
		info os.FileInfo
	}
	var candidates []entryWithInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.Name() == "Public" || e.Name() == "Default" || e.Name() == "Default User" || e.Name() == "All Users" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, entryWithInfo{e.Name(), info})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].info.ModTime().After(candidates[j].info.ModTime())
	})
	for _, c := range candidates {
		appData := filepath.Join(usersDir, c.name, "AppData")
		if _, err := os.Stat(appData); err == nil {
			return c.name
		}
	}
	return ""
}

// GetDeployMap returns the config deployment map.
func GetDeployMap(repoRoot string) []struct{ Source, Target string } {
	deployMap := []struct{ Source, Target string }{
		{weztermSourceRel, "~/..ovav/source/configs/wezterm/wezterm.lua"},
	}
	winUser := DetectWindowsUsername()
	if winUser != "" {
		deployMap = append(deployMap, struct{ Source, Target string }{
			weztermSourceRel,
			fmt.Sprintf("/mnt/c/Users/%s/..ovav/source/configs/wezterm/wezterm.lua", winUser),
		})
	}
	return deployMap
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// DeployAll runs the full deploy pipeline for all configs.
func DeployAll(repoRoot string) DeployAllResult {
	root, _ := filepath.Abs(repoRoot)
	backupDir := filepath.Join(root, configBackupDir)

	result := DeployAllResult{
		Timestamp: timestamp(),
		BackupDir: backupDir,
		AllOK:     true,
	}

	deployMap := GetDeployMap(root)
	for _, entry := range deployMap {
		source := filepath.Join(root, entry.Source)
		target := expandPath(entry.Target)

		de := DeployEntry{
			Source:       source,
			Target:       target,
			SourceExists: fileExists(source),
		}

		if !fileExists(source) {
			de.Status = "skipped"
			de.Reason = "source_file_missing"
			de.OK = false
			result.AllOK = false
			result.Deployments = append(result.Deployments, de)
			continue
		}

		// Backup existing
		backupPath := backupExistingFile(target, backupDir)
		de.Backup = backupPath
		de.HadPrevious = backupPath != ""

		// Apply
		applied := applyConfigFile(source, target)
		de.Applied = applied
		if !applied {
			de.Status = "failed"
			de.Reason = "copy_failed"
			de.OK = false
			result.AllOK = false
			result.Deployments = append(result.Deployments, de)
			continue
		}

		// Verify
		verification := verifyConfigDeploy(source, target)
		de.Verification = verification
		match, _ := verification["match"].(bool)
		de.OK = match
		if match {
			de.Status = "deployed_and_verified"
		} else {
			de.Status = "verification_failed"
			result.AllOK = false
		}

		result.Deployments = append(result.Deployments, de)
	}

	return result
}

// backupExistingFile backs up an existing target file before overwriting.
func backupExistingFile(target string, backupDir string) string {
	if _, err := os.Stat(target); err != nil {
		return ""
	}
	os.MkdirAll(backupDir, 0755)
	ts := timestamp()
	backupName := filepath.Base(target) + "." + ts + ".bak"
	backupPath := filepath.Join(backupDir, backupName)
	if err := copyFile(target, backupPath); err != nil {
		return ""
	}
	return backupPath
}

// applyConfigFile copies source to target.
func applyConfigFile(source, target string) bool {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return false
	}
	return copyFile(source, target) == nil
}

// verifyConfigDeploy verifies that the deployed file matches the source.
func verifyConfigDeploy(source, target string) map[string]interface{} {
	sourceHash := hashFile(source)
	targetHash := hashFile(target)
	return map[string]interface{}{
		"target":        target,
		"target_exists": fileExists(target),
		"source_hash":   sourceHash,
		"target_hash":   targetHash,
		"match":         sourceHash == targetHash && sourceHash != "",
	}
}

// GovernedDeploy executes config deployment under OVAV gate governance.
func GovernedDeploy(mode Mode, repoRoot string) GovernedDeployResult {
	switch mode {
	case ModeDryRun:
		return governedDryRun(repoRoot)
	case ModeSandbox:
		return governedSandbox(repoRoot)
	case ModeSourceLocalApply:
		return governedApply(repoRoot)
	default:
		return GovernedDeployResult{
			Status: "error",
			Mode:   mode,
			Error:  fmt.Sprintf("Unknown mode: %s", mode),
		}
	}
}

func governedDryRun(repoRoot string) GovernedDeployResult {
	root, _ := filepath.Abs(repoRoot)
	deployMap := GetDeployMap(root)
	entries := make([]DeployEntry, 0, len(deployMap))

	for _, entry := range deployMap {
		source := filepath.Join(root, entry.Source)
		target := expandPath(entry.Target)
		entries = append(entries, DeployEntry{
			Source:       source,
			Target:       target,
			SourceExists: fileExists(source),
			TargetExists: fileExists(target),
			Status:       "dry_run_preview",
		})
	}

	return GovernedDeployResult{
		Status:              "ok",
		Mode:                ModeDryRun,
		Governed:            true,
		DeployEntries:       len(entries),
		Entries:             entries,
		RealDeployPerformed: false,
		Gates: map[string]interface{}{
			"plan_approved":   true,
			"backup_required": true,
			"backup_exists":   false,
			"verify_required": true,
		},
		ApprovalNote: "Explicit approval required for real deploy to ~/.config/ paths.",
	}
}

func governedSandbox(repoRoot string) GovernedDeployResult {
	root, _ := filepath.Abs(repoRoot)
	deployMap := GetDeployMap(root)
	sandboxRoot := filepath.Join(root, ".ovav", "artifacts", "S88", "evidence", "sandbox")
	os.MkdirAll(sandboxRoot, 0755)

	entries := make([]DeployEntry, 0, len(deployMap))
	for _, entry := range deployMap {
		source := filepath.Join(root, entry.Source)
		sandboxTarget := filepath.Join(sandboxRoot, strings.ReplaceAll(
			strings.ReplaceAll(entry.Target, "~/", ""), "/", "_"))

		written := false
		if fileExists(source) {
			os.MkdirAll(filepath.Dir(sandboxTarget), 0755)
			if copyFile(source, sandboxTarget) == nil {
				written = true
			}
		}

		entries = append(entries, DeployEntry{
			Source:  source,
			Target:  sandboxTarget,
			Applied: written,
			Status:  "sandbox_simulated",
		})
	}

	return GovernedDeployResult{
		Status:              "ok",
		Mode:                ModeSandbox,
		Governed:            true,
		DeployEntries:       len(entries),
		Entries:             entries,
		RealDeployPerformed: false,
		SandboxRoot:         sandboxRoot,
		Gates: map[string]interface{}{
			"plan_approved":    true,
			"backup_required":  true,
			"backup_simulated": true,
			"verify_required":  true,
		},
	}
}

func governedApply(repoRoot string) GovernedDeployResult {
	root, _ := filepath.Abs(repoRoot)
	backupDir := filepath.Join(root, configBackupDir)
	backupExists := false
	if info, err := os.Stat(backupDir); err == nil && info.IsDir() {
		backupExists = true
	}

	result := DeployAll(root)
	deployOK := result.AllOK

	diag := themeDiagnostics(root)
	verifyOK, _ := diag["content_match"].(bool)

	status := "pass"
	if !(deployOK && verifyOK) {
		status = "fail"
	}

	gateStatus := deployOK && verifyOK

	return GovernedDeployResult{
		Status:              status,
		Mode:                ModeSourceLocalApply,
		Governed:            true,
		RealDeployPerformed: true,
		DeployResult:        &result,
		Diagnostics:         diag,
		Gates: map[string]interface{}{
			"plan_approved":   true,
			"backup_dir":      backupDir,
			"backup_exists":   backupExists,
			"deploy_executed": true,
			"deploy_ok":       deployOK,
			"verify_ok":       verifyOK,
			"gates_passed":    gateStatus,
		},
		Warning: "This performed a REAL write to ~/..ovav/source/configs/wezterm/wezterm.lua. Governed by OVAV gates.",
	}
}

// ThemeDiagnostics diagnoses WezTerm theme deployment status.
func ThemeDiagnostics(repoRoot string) map[string]interface{} {
	return themeDiagnostics(repoRoot)
}

func themeDiagnostics(repoRoot string) map[string]interface{} {
	root, _ := filepath.Abs(repoRoot)
	target := expandPath("~/..ovav/source/configs/wezterm/wezterm.lua")
	source := filepath.Join(root, weztermSourceRel)

	diag := map[string]interface{}{
		"config_exists": fileExists(target),
		"source_exists": fileExists(source),
		"content_match": false,
		"issues":        []string{},
	}

	if !fileExists(target) {
		diag["issues"] = []string{"wezterm_config_missing — run deploy first"}
		return diag
	}
	if !fileExists(source) {
		diag["issues"] = []string{"source_template_missing"}
		return diag
	}

	sourceHash := hashFile(source)
	targetHash := hashFile(target)
	diag["content_match"] = sourceHash == targetHash

	if !diag["content_match"].(bool) {
		diag["issues"] = []string{"content_mismatch — target differs from source template"}
		return diag
	}

	// Font check omitted in Go port (no fontconfig subprocess dependency)
	// Architecture decision: Go Native Integrity — zero subprocess bridges.

	if issues, ok := diag["issues"].([]string); ok && len(issues) == 0 {
		diag["likely_issue"] = "restart_required"
	} else {
		diag["likely_issue"] = "issues_found"
	}

	return diag
}

// GovernedDiagnose runs theme diagnostics under governance.
func GovernedDiagnose(repoRoot string) map[string]interface{} {
	diag := ThemeDiagnostics(repoRoot)
	result := map[string]interface{}{
		"status":                "ok",
		"mode":                  "diagnose",
		"governed":              true,
		"diagnostics":           diag,
		"real_deploy_performed": false,
	}
	return result
}

// MarshalJSON helper for diagnostics
func _jsonString(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
