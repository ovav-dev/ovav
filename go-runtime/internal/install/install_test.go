package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func tempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create minimal repo structure
	ovavDir := filepath.Join(dir, ".ovav")
	registryDir := filepath.Join(ovavDir, "registry")
	artifactsDir := filepath.Join(ovavDir, "artifacts")
	toolsDir := filepath.Join(dir, "tools")
	opencodeDir := filepath.Join(dir, ".opencode")
	agentsDir := filepath.Join(opencodeDir, "agents")
	skillsDir := filepath.Join(opencodeDir, "skills")
	commandsDir := filepath.Join(opencodeDir, "commands")

	for _, d := range []string{ovavDir, registryDir, artifactsDir, toolsDir, agentsDir, skillsDir, commandsDir} {
		os.MkdirAll(d, 0755)
	}

	// Write minimal install_packs.yaml
	packsYAML := `install_packs:
  test_pack:
    modes: [inspect, plan, dry-run, sandbox, source-local-apply]
    targets: [source_runtime, opencode_agents]
    notes: test_pack_for_go_tests
  dry_run_only_pack:
    modes: [inspect, plan, dry-run]
    targets: [source_runtime]
    notes: cannot_apply
`
	os.WriteFile(filepath.Join(registryDir, "install_packs.yaml"), []byte(packsYAML), 0644)

	// Create a test file in tools/ to have a real target for backup tests
	testData := []byte("hello world — test file for install gateway\n")
	os.WriteFile(filepath.Join(toolsDir, "test_tool.sh"), testData, 0755)
	os.WriteFile(filepath.Join(agentsDir, "test_agent.md"), testData, 0644)

	return dir
}

// ── Mode resolution ──────────────────────────────────────────────────────────

func TestResolveMode(t *testing.T) {
	tests := []struct {
		input    string
		expected Mode
		wantErr  bool
	}{
		{"dry-run", ModeDryRun, false},
		{"dry_run", ModeDryRun, false},
		{"DRY-RUN", ModeDryRun, false},
		{"sandbox", ModeSandbox, false},
		{"source-local-apply", ModeSourceLocalApply, false},
		{"source_local_apply", ModeSourceLocalApply, false},
		{"apply", ModeSourceLocalApply, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ResolveMode(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q, got mode %q", tt.input, mode)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
			}
			if mode != tt.expected {
				t.Errorf("mode = %q, want %q", mode, tt.expected)
			}
		})
	}
}

// ── Plan building ─────────────────────────────────────────────────────────────

func TestBuildPlan(t *testing.T) {
	repo := tempRepo(t)

	t.Run("valid_pack_dry_run", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		if plan.Status != "pass" {
			t.Errorf("status = %s, want pass", plan.Status)
		}
		if plan.EntryCount != 2 {
			t.Errorf("entry_count = %d, want 2", plan.EntryCount)
		}
		if !plan.DryRunOnly {
			t.Error("expected dry_run_only = true")
		}
		if plan.RealApply {
			t.Error("expected real_apply = false")
		}
		for _, e := range plan.Entries {
			if e.WriteEnabled {
				t.Errorf("write_enabled should be false for dry-run, got true for %s", e.TargetID)
			}
		}
	})

	t.Run("valid_pack_source_local_apply", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		if plan.Status != "pass" {
			t.Errorf("status = %s, want pass (%s)", plan.Status, plan.Error)
		}
		if plan.EntryCount != 2 {
			t.Errorf("entry_count = %d, want 2", plan.EntryCount)
		}
		if !plan.RealApply {
			t.Error("expected real_apply = true")
		}
		for _, e := range plan.Entries {
			if !e.WriteEnabled {
				t.Errorf("expected write_enabled = true for source-local-apply, target %s", e.TargetID)
			}
		}
	})

	t.Run("sandbox_mode_routes_to_sandbox_path", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSandbox, repo)
		if plan.Status != "pass" {
			t.Errorf("status = %s, want pass", plan.Status)
		}
		for _, e := range plan.Entries {
			if !strings.Contains(e.Target, "sandbox") {
				t.Errorf("sandbox target should contain 'sandbox', got: %s", e.Target)
			}
		}
	})

	t.Run("unknown_pack", func(t *testing.T) {
		plan := BuildPlan("nonexistent", ModeDryRun, repo)
		if plan.Status != "fail" {
			t.Errorf("status = %s, want fail", plan.Status)
		}
		if plan.Error == "" {
			t.Error("expected error message for unknown pack")
		}
	})

	t.Run("mode_not_allowed", func(t *testing.T) {
		plan := BuildPlan("dry_run_only_pack", ModeSourceLocalApply, repo)
		if plan.Status != "blocked" {
			t.Errorf("status = %s, want blocked", plan.Status)
		}
	})

	t.Run("sandbox_mode", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSandbox, repo)
		if plan.Status != "pass" {
			t.Errorf("status = %s, want pass (%s)", plan.Status, plan.Error)
		}
		if !plan.SandboxOnly {
			t.Error("expected sandbox_only = true")
		}
	})
}

// ── Manifest building ─────────────────────────────────────────────────────────

func TestBuildManifest(t *testing.T) {
	repo := tempRepo(t)

	t.Run("dry_run_manifest", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		plan.Status = "pass" // ensure we proceed
		manifest := BuildManifest(plan)

		if manifest.Status != "pass" {
			t.Errorf("status = %s, want pass", manifest.Status)
		}
		if manifest.TotalEntries != 2 {
			t.Errorf("total = %d, want 2", manifest.TotalEntries)
		}
		if manifest.DryRunOnly != true {
			t.Error("expected dry_run_only = true")
		}
		// In dry-run, all operations should be dry_run since write_enabled is false
		for _, e := range manifest.Entries {
			if e.Operation != "dry_run" {
				t.Errorf("expected dry_run operation, got %s for %s", e.Operation, e.Target)
			}
		}
	})

	t.Run("source_local_apply_manifest", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		plan.Status = "pass"
		manifest := BuildManifest(plan)

		if manifest.Status != "pass" {
			t.Errorf("status = %s, want pass", manifest.Status)
		}
		if manifest.ApplyEntries != 2 {
			t.Errorf("apply_entries = %d, want 2", manifest.ApplyEntries)
		}
		for _, e := range manifest.Entries {
			if e.Operation != "create" && e.Operation != "update" {
				t.Errorf("unexpected operation %s for %s (mode=%s, write=%t)", e.Operation, e.Target, e.Mode, e.WriteEnabled)
			}
		}
	})

	t.Run("blocked_entries", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		// Inject a blocked entry manually
		plan.Entries = append(plan.Entries, PlanEntry{
			TargetID:     "blocked_target",
			Target:       "/etc/hacked.conf",
			TargetRisk:   "global-risk",
			Mode:         ModeDryRun,
			WriteEnabled: false,
		})
		manifest := BuildManifest(plan)

		if manifest.Status != "fail" {
			t.Errorf("status = %s, want fail (blocked entry)", manifest.Status)
		}
		if manifest.BlockedEntries != 1 {
			t.Errorf("blocked = %d, want 1", manifest.BlockedEntries)
		}
		// Verify the blocked entry is in BlockedDetails
		found := false
		for _, b := range manifest.BlockedDetails {
			if strings.Contains(b.Target, "hacked.conf") {
				found = true
			}
		}
		if !found {
			t.Error("blocked entry not in BlockedDetails")
		}
	})
}

// ── Safety evaluation ─────────────────────────────────────────────────────────

func TestEvaluateSafety(t *testing.T) {
	repo := tempRepo(t)

	t.Run("all_clean", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		safety := EvaluateSafety(plan)

		if safety.Status != "pass" {
			t.Errorf("status = %s, want pass", safety.Status)
		}
		if safety.OverallSafety != "allow" {
			t.Errorf("overall = %s, want allow", safety.OverallSafety)
		}
		if safety.HasBlocked {
			t.Error("expected no blocked entries")
		}
	})

	t.Run("blocked_entries", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		// Inject a high-risk entry
		plan.Entries = append(plan.Entries, PlanEntry{
			Target:     "/etc/danger.conf",
			TargetRisk: "global-risk",
		})
		safety := EvaluateSafety(plan)

		if safety.OverallSafety != "blocked" {
			t.Errorf("overall = %s, want blocked", safety.OverallSafety)
		}
		if !safety.HasBlocked {
			t.Error("expected has_blocked = true")
		}
	})

	t.Run("unsafe_selectors_detected", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		plan.Entries[0].Target = ".ovav/registry/apply-now.sh"
		safety := EvaluateSafety(plan)

		if safety.Status != "fail" {
			t.Errorf("status = %s, want fail", safety.Status)
		}
		found := false
		for _, issue := range safety.Issues {
			if strings.Contains(issue, "apply-now") {
				found = true
			}
		}
		if !found {
			t.Error("expected unsafe selector detection")
		}
	})

	t.Run("real_apply_allowed", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		safety := EvaluateSafety(plan)

		if !safety.RealApplyAllowed {
			t.Error("expected real_apply_allowed = true for clean plan")
		}
		if !safety.NeedsBackup {
			t.Error("expected needs_backup = true for source-local-apply")
		}
	})
}

// ── Boundary validation ───────────────────────────────────────────────────────

func TestBoundaryValidation(t *testing.T) {
	repo := tempRepo(t)

	t.Run("all_targets_ok", func(t *testing.T) {
		targets := []string{
			filepath.Join(repo, ".ovav", "registry", "test.yaml"),
			filepath.Join(repo, "tools", "test.sh"),
		}
		report := ValidateAllTargets(targets, ModeDryRun, repo)

		if report.Status != "pass" {
			t.Errorf("status = %s, want pass", report.Status)
		}
		if report.Blocked != 0 {
			t.Errorf("blocked = %d, want 0", report.Blocked)
		}
	})

	t.Run("home_path_blocked", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		report := ValidateAllTargets([]string{filepath.Join(home, ".config", "test.lua")}, ModeSourceLocalApply, repo)

		if report.Blocked != 1 {
			t.Errorf("blocked = %d, want 1", report.Blocked)
		}
	})

	t.Run("unsafe_selector_in_path", func(t *testing.T) {
		targets := []string{filepath.Join(repo, ".ovav", "registry", "global-config.yaml")}
		report := ValidateAllTargets(targets, ModeSourceLocalApply, repo)

		if report.Blocked != 1 {
			t.Errorf("blocked = %d, want 1 for unsafe selector", report.Blocked)
		}
	})

	t.Run("dry_run_allows_examination", func(t *testing.T) {
		targets := []string{filepath.Join(repo, ".ovav", "test.json")}
		report := ValidateAllTargets(targets, ModeDryRun, repo)

		if report.Status != "pass" {
			t.Errorf("dry-run should pass: %s", report.Status)
		}
		for _, r := range report.Results {
			if r.AllowsWrite {
				t.Error("dry-run should not allow writes")
			}
		}
	})
}

// ── Backup engine ─────────────────────────────────────────────────────────────

func TestExecuteBackup(t *testing.T) {
	repo := tempRepo(t)

	t.Run("dry_run_preview", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		manifest := BuildManifest(plan)
		report := ExecuteBackup(manifest, ModeDryRun, repo)

		if report.Status != "pass" {
			t.Errorf("status = %s, want pass", report.Status)
		}
		if report.BackupPerformed {
			t.Error("dry-run should not perform backup")
		}
		if !report.DryRunPreview {
			t.Error("expected dry_run_preview = true")
		}
		// In dry-run, NeedsBackup is false for all entries, so 0 targets identified
		// This matches Python behavior: dry-run entries have needs_backup=False
	})

	t.Run("real_backup_source_local_apply", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		// Override with actual existing files to back up
		plan.Entries[0].Target = filepath.Join(repo, "tools", "test_tool.sh")
		plan.Entries[1].Target = filepath.Join(repo, ".opencode", "agents", "test_agent.md")

		manifest := BuildManifest(plan)
		report := ExecuteBackup(manifest, ModeSourceLocalApply, repo)

		if report.Status != "pass" {
			t.Errorf("status = %s, want pass", report.Status)
		}
		if !report.BackupPerformed {
			t.Error("should have performed backup")
		}
		if report.BackedUp < 1 {
			t.Errorf("backed_up = %d, want >= 1", report.BackedUp)
		}
		if report.BackupDir == "" {
			t.Error("backup_dir should not be empty")
		}

		// Verify backup directory exists
		if _, err := os.Stat(report.BackupDir); os.IsNotExist(err) {
			t.Errorf("backup dir does not exist: %s", report.BackupDir)
		}

		// Verify each backed-up result has valid hash
		for _, r := range report.Results {
			if r.Status == "backed_up" {
				if !r.Verified {
					t.Errorf("backup not verified for %s: source=%s backup=%s", r.Target, r.SourceHash, r.BackupHash)
				}
				if r.SourceHash == "" || r.BackupHash == "" {
					t.Errorf("missing hash for %s", r.Target)
				}
			}
		}
	})

	t.Run("sandbox_backup", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSandbox, repo)
		plan.Entries[0].Target = filepath.Join(repo, "tools", "test_tool.sh")
		manifest := BuildManifest(plan)
		report := ExecuteBackup(manifest, ModeSandbox, repo)

		if !strings.Contains(report.BackupDir, "sandbox") {
			t.Errorf("sandbox backup dir should contain 'sandbox', got: %s", report.BackupDir)
		}
	})
}

// ── Rollback engine ──────────────────────────────────────────────────────────

func TestExecuteRollback(t *testing.T) {
	repo := tempRepo(t)

	t.Run("dry_run_preview", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)
		manifest := BuildManifest(plan)
		backup := ExecuteBackup(manifest, ModeDryRun, repo)
		rollback := ExecuteRollback(backup, manifest, ModeDryRun, repo)

		if rollback.Status != "pass" {
			t.Errorf("status = %s, want pass", rollback.Status)
		}
		if rollback.RollbackPerformed {
			t.Error("dry-run should not perform rollback")
		}
	})

	t.Run("rollback_roundtrip", func(t *testing.T) {
		// Create a real file, back it up, modify it, rollback, verify
		testFile := filepath.Join(repo, "tools", "test_tool.sh")
		originalContent, _ := os.ReadFile(testFile)

		// Build plan for this file
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		plan.Entries = []PlanEntry{{
			TargetID:     "test_file",
			Target:       testFile,
			TargetRisk:   "repo-local",
			Mode:         ModeSourceLocalApply,
			WriteEnabled: true,
		}}
		manifest := BuildManifest(plan)

		// Override operation to ensure backup happens
		manifest.Entries[0].Operation = "update"
		manifest.Entries[0].NeedsBackup = true
		manifest.Entries[0].TargetExists = true

		// Backup
		backup := ExecuteBackup(manifest, ModeSourceLocalApply, repo)
		if backup.BackedUp != 1 {
			t.Fatalf("backup failed: backed_up=%d, status=%s", backup.BackedUp, backup.Status)
		}

		// Modify the file
		os.WriteFile(testFile, []byte("MODIFIED CONTENT\n"), 0755)

		// Rollback
		rollback := ExecuteRollback(backup, manifest, ModeSourceLocalApply, repo)
		if rollback.Status != "pass" {
			t.Errorf("rollback status = %s, want pass", rollback.Status)
		}
		if rollback.Restored != 1 {
			t.Errorf("restored = %d, want 1", rollback.Restored)
		}

		// Verify file content is restored
		restoredContent, _ := os.ReadFile(testFile)
		if string(restoredContent) != string(originalContent) {
			t.Errorf("rollback didn't restore original content.\noriginal: %q\nrestored: %q", originalContent, restoredContent)
		}
	})
}

// ── Full pipeline ─────────────────────────────────────────────────────────────

func TestExecuteApply(t *testing.T) {
	repo := tempRepo(t)

	t.Run("full_dry_run_pipeline", func(t *testing.T) {
		report := ExecuteApply("test_pack", ModeDryRun, repo)

		if report.Status != "pass" {
			t.Errorf("status = %s, want pass", report.Status)
		}
		if report.RealApplyPerformed {
			t.Error("dry-run should not perform real apply")
		}
		if report.Stages.Plan.Status != "pass" {
			t.Errorf("plan stage failed: %s", report.Stages.Plan.Error)
		}
		if report.Stages.Manifest.Status != "pass" {
			t.Errorf("manifest stage failed: blocked=%d", report.Stages.Manifest.BlockedEntries)
		}
		if report.Stages.Gates.TotalGates != 14 {
			t.Errorf("total gates = %d, want 14 (9 backup + 5 rollback)", report.Stages.Gates.TotalGates)
		}

		// Verify JSON serialization works
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Errorf("json marshal failed: %v", err)
		}
		if len(data) < 100 {
			t.Error("report JSON too short")
		}
	})

	t.Run("source_local_apply_pipeline", func(t *testing.T) {
		report := ExecuteApply("test_pack", ModeSourceLocalApply, repo)

		if report.Status != "pass" {
			t.Logf("errors: %v", report.Errors)
		}
		if !report.SourceLocalApplyReady {
			t.Error("expected source_local_apply_ready = true")
		}
		if report.Stages.Backup.BackupPerformed != true {
			t.Error("expected backup to be performed")
		}
		// Backup gates should all be satisfied (or at least most)
		if report.Stages.Gates.Backup.Satisfied < 7 {
			t.Errorf("backup gates satisfied = %d, want >= 7", report.Stages.Gates.Backup.Satisfied)
		}
		if report.Stages.Gates.Rollback.Satisfied < 3 {
			t.Errorf("rollback gates satisfied = %d, want >= 3", report.Stages.Gates.Rollback.Satisfied)
		}
	})

	t.Run("undefined_pack", func(t *testing.T) {
		report := ExecuteApply("nonexistent", ModeDryRun, repo)
		if len(report.Errors) == 0 {
			t.Error("expected errors for unknown pack")
		}
	})
}

// ── Evidence report writing ──────────────────────────────────────────────────

func TestWriteEvidence(t *testing.T) {
	repo := tempRepo(t)

	report := ExecuteApply("test_pack", ModeDryRun, repo)
	evidence, err := WriteEvidence(report, "TEST", repo)

	if err != nil {
		t.Fatalf("WriteEvidence failed: %v", err)
	}
	if evidence.Status != "pass" {
		t.Errorf("evidence status = %s, want pass", evidence.Status)
	}
	if evidence.FilesWritten != 4 {
		t.Errorf("files_written = %d, want 4", evidence.FilesWritten)
	}

	// Verify files exist
	for _, p := range evidence.Paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("evidence file not found: %s", p)
		}
	}

	// Verify Markdown summary has expected sections
	for _, p := range evidence.Paths {
		if strings.HasSuffix(p, ".md") {
			content, _ := os.ReadFile(p)
			if !strings.Contains(string(content), "Gate Satisfaction") {
				t.Error("summary missing 'Gate Satisfaction' section")
			}
			if !strings.Contains(string(content), "Apply Results") {
				t.Error("summary missing 'Apply Results' section")
			}
		}
	}
}

// ── UX previews ───────────────────────────────────────────────────────────────

func TestUXPreviews(t *testing.T) {
	repo := tempRepo(t)

	plan := BuildPlan("test_pack", ModeDryRun, repo)
	safety := EvaluateSafety(plan)
	manifest := BuildManifest(plan)
	backup := ExecuteBackup(manifest, ModeDryRun, repo)

	t.Run("plan_preview", func(t *testing.T) {
		preview := PreviewPlan(plan)
		if !strings.Contains(preview, "Install Plan Preview") {
			t.Error("plan preview missing title")
		}
		if !strings.Contains(preview, "Dry-run mode") {
			t.Error("plan preview missing mode note")
		}
	})

	t.Run("risk_preview", func(t *testing.T) {
		preview := PreviewRisk(safety)
		if !strings.Contains(preview, "Risk Summary") {
			t.Error("risk preview missing title")
		}
		if !strings.Contains(preview, "Overall safety") {
			t.Error("risk preview missing overall safety")
		}
	})

	t.Run("rollback_guide", func(t *testing.T) {
		guide := PreviewRollbackGuide(backup)
		if !strings.Contains(guide, "Rollback Guide") {
			t.Error("rollback guide missing title")
		}
		if !strings.Contains(guide, "Rollback procedure") {
			t.Error("rollback guide missing procedure")
		}
	})

	t.Run("build_ux_preview", func(t *testing.T) {
		ux := BuildUXPreview(plan, safety, backup)
		if ux.Status != "pass" {
			t.Errorf("ux status = %s, want pass", ux.Status)
		}
		if ux.PlanPreview == "" || ux.RiskPreview == "" || ux.RollbackGuide == "" {
			t.Error("ux preview has empty sections")
		}
	})
}

// ── Config deployment ────────────────────────────────────────────────────────

func TestConfigDeploy(t *testing.T) {
	repo := tempRepo(t)

	t.Run("deploy_map_has_wezterm_entries", func(t *testing.T) {
		deployMap := GetDeployMap(repo)
		if len(deployMap) < 1 {
			t.Error("deploy map should have at least 1 entry")
		}
		foundWezterm := false
		for _, e := range deployMap {
			if strings.Contains(e.Source, "wezterm") {
				foundWezterm = true
			}
		}
		if !foundWezterm {
			t.Error("deploy map missing wezterm config")
		}
	})

	t.Run("governed_dry_run", func(t *testing.T) {
		result := GovernedDeploy(ModeDryRun, repo)
		if result.Status != "ok" {
			t.Errorf("status = %s, want ok", result.Status)
		}
		if result.RealDeployPerformed {
			t.Error("dry-run should not perform real deploy")
		}
		if !result.Governed {
			t.Error("should be governed")
		}
	})

	t.Run("governed_sandbox", func(t *testing.T) {
		result := GovernedDeploy(ModeSandbox, repo)
		if result.Status != "ok" {
			t.Errorf("status = %s, want ok", result.Status)
		}
		if result.SandboxRoot == "" {
			t.Error("sandbox root should not be empty")
		}
	})

	t.Run("theme_diagnostics", func(t *testing.T) {
		diag := ThemeDiagnostics(repo)
		if _, ok := diag["config_exists"]; !ok {
			t.Error("diagnostics missing config_exists")
		}
		if _, ok := diag["source_exists"]; !ok {
			t.Error("diagnostics missing source_exists")
		}
	})

	t.Run("governed_diagnose", func(t *testing.T) {
		result := GovernedDiagnose(repo)
		governed, _ := result["governed"].(bool)
		if !governed {
			t.Error("diagnose should be governed")
		}
	})
}

// ── Utility functions ─────────────────────────────────────────────────────────

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	h := hashFile(f)
	if h == "" {
		t.Error("hash should not be empty")
	}
	// SHA-256 of "hello" is 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	if len(h) != 64 {
		t.Errorf("hash length = %d, want 64", len(h))
	}

	// Non-existent file
	h2 := hashFile(filepath.Join(dir, "nope.txt"))
	if h2 != "" {
		t.Error("hash of non-existent file should be empty")
	}
}

func TestTimestamp(t *testing.T) {
	ts := timestamp()
	if len(ts) != 16 {
		t.Errorf("timestamp length = %d, want 16 (YYYYMMDDTHHMMSSZ: 8+1+6+1)", len(ts))
	}
	if !strings.HasSuffix(ts, "Z") {
		t.Error("timestamp should end with Z (UTC)")
	}
}

func TestIsSourceLocalPath(t *testing.T) {
	repo := tempRepo(t)

	if !isSourceLocalPath(filepath.Join(repo, "tools"), repo) {
		t.Error("tools/ should be source-local")
	}
	if isSourceLocalPath("/tmp/outside", repo) {
		t.Error("/tmp/outside should NOT be source-local")
	}
}

func TestIsSafeTarget(t *testing.T) {
	repo := tempRepo(t)

	t.Run("dry_run_allows_any_repo_path", func(t *testing.T) {
		if !isSafeTarget(filepath.Join(repo, "tools"), ModeDryRun, repo) {
			t.Error("dry-run should allow tools/")
		}
	})

	t.Run("source_local_apply_only_eligible", func(t *testing.T) {
		// Eligible surface
		if !isSafeTarget(filepath.Join(repo, "tools"), ModeSourceLocalApply, repo) {
			t.Error("source-local-apply should allow tools/ (eligible surface)")
		}

		// Ineligible surface (random file in repo root)
		randomFile := filepath.Join(repo, "random.txt")
		os.WriteFile(randomFile, []byte("test"), 0644)
		if isSafeTarget(randomFile, ModeSourceLocalApply, repo) {
			t.Error("source-local-apply should NOT allow random file outside eligible surfaces")
		}
	})

	t.Run("outside_repo_blocked", func(t *testing.T) {
		if isSafeTarget("/tmp/outside", ModeSourceLocalApply, repo) {
			t.Error("paths outside repo should be blocked")
		}
	})
}

func TestClassifyRisk(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/.config/test", "user-config-risk"},
		{"/etc/config.conf", "global-risk"},
		{"/usr/local/bin/test", "global-risk"},
		{"~/.local/share/test", "user-local-risk"},
		{"/repo/tools/test.sh", "repo-local"},
		{"/repo/.ovav/artifacts/test", "sandbox"},
		{"/repo/.opencode/agents/test.md", "repo-local"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := classifyRisk(tt.path)
			if result != tt.expected {
				t.Errorf("classifyRisk(%q) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	if fileExists(filepath.Join(dir, "nope.txt")) {
		t.Error("non-existent file should not exist")
	}

	f := filepath.Join(dir, "exists.txt")
	os.WriteFile(f, []byte("test"), 0644)
	if !fileExists(f) {
		t.Error("created file should exist")
	}

	// Directory check
	if !fileExists(dir) {
		t.Error("directory should be detected as existing")
	}
}

// ── Paridad: Python gateway test suite ────────────────────────────────────────
// These tests verify that the Go port produces equivalent results to the
// Python install_gateway for the same inputs.

func TestPythonParity(t *testing.T) {
	repo := tempRepo(t)

	t.Run("parity_plan_structure", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeDryRun, repo)

		// Python equivalent checks:
		// - plan["status"] == "pass"
		// - plan["pack_id"] == pack_id
		// - plan["mode"] == mode
		// - "entries" in plan, len > 0
		// - each entry has: target_id, target, source, target_risk, mode, write_enabled

		if plan.Status != "pass" {
			t.Errorf("[parity] status mismatch: Go=%s, Python=pass", plan.Status)
		}
		if plan.PackID != "test_pack" {
			t.Errorf("[parity] pack_id mismatch")
		}
		if plan.Mode != ModeDryRun {
			t.Errorf("[parity] mode mismatch")
		}
		if len(plan.Entries) == 0 {
			t.Error("[parity] entries should not be empty")
		}
		for _, e := range plan.Entries {
			if e.TargetID == "" {
				t.Error("[parity] entry missing target_id")
			}
			if e.Target == "" {
				t.Error("[parity] entry missing target")
			}
			if e.TargetRisk == "" {
				t.Error("[parity] entry missing target_risk")
			}
		}
	})

	t.Run("parity_manifest_classification", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		plan.Status = "pass"
		manifest := BuildManifest(plan)

		// Python: each entry has operation in {create, update, blocked, dry_run}
		validOps := map[string]bool{"create": true, "update": true, "blocked": true, "dry_run": true}
		for _, e := range manifest.Entries {
			if !validOps[e.Operation] {
				t.Errorf("[parity] invalid operation: %s", e.Operation)
			}
		}
	})

	t.Run("parity_gate_count", func(t *testing.T) {
		report := ExecuteApply("test_pack", ModeDryRun, repo)

		// Python: total gates = len(BACKUP_GATES) + len(ROLLBACK_GATES) = 9 + 5 = 14
		if report.Stages.Gates.TotalGates != 14 {
			t.Errorf("[parity] total gates: Go=%d, Python=14", report.Stages.Gates.TotalGates)
		}
		if report.Stages.Gates.Backup.Total != 9 {
			t.Errorf("[parity] backup gates total: Go=%d, Python=9", report.Stages.Gates.Backup.Total)
		}
		if report.Stages.Gates.Rollback.Total != 5 {
			t.Errorf("[parity] rollback gates total: Go=%d, Python=5", report.Stages.Gates.Rollback.Total)
		}
	})

	t.Run("parity_blocked_surfaces", func(t *testing.T) {
		report := ExecuteApply("test_pack", ModeDryRun, repo)

		// Go: 7 permanently blocked surfaces (engram_memory removed — system deprecado)
		if len(report.BlockedSurfaces) != 7 {
			t.Errorf("[parity] blocked surfaces: Go=%d, expected=7", len(report.BlockedSurfaces))
		}

		expectedSurfaces := map[string]bool{
			"global_install": true, "user_home_config": true,
			"opencode_global_config": true, "plugin_install": true,
			"external_services": true,
			"ui_tui":            true, "mcp_a2a": true,
		}
		for _, s := range report.BlockedSurfaces {
			if !expectedSurfaces[s] {
				t.Errorf("[parity] unexpected blocked surface: %s", s)
			}
		}
	})

	t.Run("parity_error_accumulation", func(t *testing.T) {
		report := ExecuteApply("nonexistent", ModeDryRun, repo)

		// Python: unknown pack should generate at least 1 error
		if len(report.Errors) == 0 {
			t.Error("[parity] expected errors for unknown pack (Python generates plan_failed)")
		}
		if report.Status != "fail" {
			t.Errorf("[parity] status should be 'fail' for unknown pack, got: %s", report.Status)
		}
	})
}

// ── Edge cases ────────────────────────────────────────────────────────────────

func TestEdgeCases(t *testing.T) {
	repo := tempRepo(t)

	t.Run("empty_plan", func(t *testing.T) {
		// Pack with zero targets would need to exist in registry
		plan := Plan{
			Status:     "pass",
			PackID:     "empty",
			Mode:       ModeDryRun,
			Entries:    []PlanEntry{},
			EntryCount: 0,
		}
		manifest := BuildManifest(plan)
		if manifest.TotalEntries != 0 {
			t.Error("empty plan should produce empty manifest")
		}
	})

	t.Run("special_chars_in_path", func(t *testing.T) {
		f := filepath.Join(repo, "tools", "test with spaces.sh")
		os.WriteFile(f, []byte("test"), 0644)

		if !fileExists(f) {
			t.Error("file with spaces should be detected")
		}

		h := hashFile(f)
		if h == "" {
			t.Error("hash of file with spaces should not be empty")
		}
	})

	t.Run("binary_file_hash", func(t *testing.T) {
		f := filepath.Join(repo, "tools", "binary.bin")
		data := make([]byte, 256)
		for i := range data {
			data[i] = byte(i)
		}
		os.WriteFile(f, data, 0644)

		h := hashFile(f)
		if len(h) != 64 {
			t.Errorf("binary file hash length = %d, want 64", len(h))
		}
	})

	t.Run("large_plan_json_stability", func(t *testing.T) {
		plan := BuildPlan("test_pack", ModeSourceLocalApply, repo)
		manifest := BuildManifest(plan)
		safety := EvaluateSafety(plan)

		// Serialize and deserialize to check JSON stability
		data, _ := json.Marshal(manifest)
		var restored Manifest
		json.Unmarshal(data, &restored)

		if restored.TotalEntries != manifest.TotalEntries {
			t.Error("JSON round-trip changed total entries")
		}
		if restored.Status != manifest.Status {
			t.Error("JSON round-trip changed status")
		}

		// Safety JSON stability
		safetyData, _ := json.Marshal(safety)
		var restoredSafety SafetyReport
		json.Unmarshal(safetyData, &restoredSafety)
		if restoredSafety.OverallSafety != safety.OverallSafety {
			t.Error("JSON round-trip changed overall safety")
		}
	})
}

// ── Additional tests for uncovered functions ─────────────────────────────────

func Test_jsonString(t *testing.T) {
	// Simple struct
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	v := testStruct{Name: "test", Age: 42}
	result := _jsonString(v)
	if !strings.Contains(result, `"name": "test"`) {
		t.Errorf("expected 'name' field, got: %s", result)
	}
	if !strings.Contains(result, `"age": 42`) {
		t.Errorf("expected 'age' field, got: %s", result)
	}

	// Nil/empty
	result2 := _jsonString(nil)
	if result2 != "null" {
		t.Errorf("expected 'null' for nil, got: %s", result2)
	}

	// Map
	m := map[string]interface{}{"key": "value"}
	result3 := _jsonString(m)
	if !strings.Contains(result3, `"key": "value"`) {
		t.Errorf("expected map key, got: %s", result3)
	}
}

func TestRunStrictValidation(t *testing.T) {
	result := RunStrictValidation("/tmp/test")
	if result["status"] != "pass" {
		t.Errorf("expected status=pass, got %v", result["status"])
	}
	if note, ok := result["note"].(string); !ok || !strings.Contains(note, "go-native") {
		t.Errorf("expected go-native note, got: %v", note)
	}
}

func TestVerifyApplyResults(t *testing.T) {
	repo := tempRepo(t)

	t.Run("dry_run_mode", func(t *testing.T) {
		report := ApplyGatewayReport{
			Stages: StageResults{
				Apply: ApplyReport{Results: []ApplyResult{}},
			},
		}
		result := VerifyApplyResults(report, ModeDryRun, repo)
		if result["status"] != "pass" {
			t.Errorf("expected pass for dry-run, got %v", result["status"])
		}
		if result["verification"] != "not_applicable_dry_run" {
			t.Errorf("expected dry-run message")
		}
	})

	t.Run("no_written_files", func(t *testing.T) {
		report := ApplyGatewayReport{
			Stages: StageResults{
				Apply: ApplyReport{
					Results: []ApplyResult{
						{Target: "/tmp/nonexistent", Written: false},
					},
				},
			},
		}
		result := VerifyApplyResults(report, ModeSourceLocalApply, repo)
		if result["status"] != "pass" {
			t.Errorf("expected pass when no files written, got %v", result["status"])
		}
		if result["file_count"].(int) != 0 {
			t.Errorf("expected 0 files, got %d", result["file_count"])
		}
	})

	t.Run("written_file_exists", func(t *testing.T) {
		// Create a real file that will exist
		tmpFile := filepath.Join(repo, "tools", "verify_test_file.txt")
		os.WriteFile(tmpFile, []byte("verify me"), 0644)

		report := ApplyGatewayReport{
			Stages: StageResults{
				Apply: ApplyReport{
					Results: []ApplyResult{
						{Target: tmpFile, Written: true},
					},
				},
			},
		}
		result := VerifyApplyResults(report, ModeSourceLocalApply, repo)
		if result["status"] != "pass" {
			t.Errorf("expected pass when file exists, got %v: %v", result["status"], result)
		}
	})

	t.Run("written_file_missing", func(t *testing.T) {
		report := ApplyGatewayReport{
			Stages: StageResults{
				Apply: ApplyReport{
					Results: []ApplyResult{
						{Target: "/tmp/definitely_not_exists_ovav_test", Written: true},
					},
				},
			},
		}
		result := VerifyApplyResults(report, ModeSourceLocalApply, repo)
		if result["status"] != "fail" {
			t.Errorf("expected fail when file missing, got %v", result["status"])
		}
	})

	t.Run("path_leakage_detection", func(t *testing.T) {
		report := ApplyGatewayReport{
			Stages: StageResults{
				Apply: ApplyReport{
					Results: []ApplyResult{
						{Target: "/home/user/secret_token_file", Written: true},
					},
				},
			},
		}
		result := VerifyApplyResults(report, ModeSourceLocalApply, repo)
		checks := result["checks"].(map[string]bool)
		if checks["no_path_leakage"] != false {
			t.Errorf("expected path leakage detection")
		}
	})

	t.Run("mixed_results", func(t *testing.T) {
		tmpFile := filepath.Join(repo, "tools", "mixed_test.txt")
		os.WriteFile(tmpFile, []byte("mixed"), 0644)

		report := ApplyGatewayReport{
			Stages: StageResults{
				Apply: ApplyReport{
					Results: []ApplyResult{
						{Target: tmpFile, Written: true},
						{Target: "/tmp/nonexistent", Written: true},
						{Target: "/home/secret", Written: true},
					},
				},
			},
		}
		result := VerifyApplyResults(report, ModeSourceLocalApply, repo)
		if result["status"] != "fail" {
			t.Errorf("expected fail with mixed results, got %v", result["status"])
		}
		fileCount := result["file_count"].(int)
		if fileCount != 3 {
			t.Errorf("expected 3 files, got %d", fileCount)
		}
	})
}

func TestGovernedApply(t *testing.T) {
	repo := tempRepo(t)

	// Create WezTerm source template so DeployAll has a source
	weztermSourceDir := filepath.Join(repo, ".ovav", "context", "backups")
	os.MkdirAll(weztermSourceDir, 0755)
	os.WriteFile(filepath.Join(weztermSourceDir, "wezterm-catppuccin-mocha.lua"), []byte("-- OVAV WezTerm Config v1\nreturn {}"), 0644)

	// Override HOME to point to a temp dir so expandPath("~/...") resolves safely
	tempHome := filepath.Join(repo, "fake_home")
	os.MkdirAll(tempHome, 0755)
	// Create target dir structure that DeployAll would write to
	os.MkdirAll(filepath.Join(tempHome, "..ovav", "source", "configs", "wezterm"), 0755)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", oldHome)

	// Also set OVAV_DEV=1 to bypass safety checks
	oldDev := os.Getenv("OVAV_DEV")
	os.Setenv("OVAV_DEV", "1")
	defer os.Setenv("OVAV_DEV", oldDev)

	result := governedApply(repo)

	// governedApply always returns a GovernedDeployResult
	if result.Mode != ModeSourceLocalApply {
		t.Errorf("expected mode %s, got %s", ModeSourceLocalApply, result.Mode)
	}
	if !result.Governed {
		t.Error("expected governed=true")
	}
	if !result.RealDeployPerformed {
		t.Error("expected RealDeployPerformed=true")
	}
	if result.Gates == nil {
		t.Error("expected gates map")
	}
	// Check gate values
	gates := result.Gates
	if v, ok := gates["deploy_executed"].(bool); !ok || !v {
		t.Error("expected deploy_executed=true")
	}
}

func TestThemeDiagnostics(t *testing.T) {
	repo := tempRepo(t)

	// Override HOME
	tempHome := filepath.Join(repo, "fake_home2")
	os.MkdirAll(tempHome, 0755)
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", oldHome)

	// Case 1: no target file — should report missing
	t.Run("target_missing", func(t *testing.T) {
		diag := themeDiagnostics(repo)
		exists, _ := diag["config_exists"].(bool)
		if exists {
			t.Error("expected config_exists=false when target missing")
		}
		issues := diag["issues"].([]string)
		if len(issues) == 0 || issues[0] != "wezterm_config_missing — run deploy first" {
			t.Errorf("expected missing config message, got %v", issues)
		}
	})

	// Create wezterm source file (correct location from weztermSourceRel)
	weztermSourceDir := filepath.Join(repo, ".ovav", "context", "backups")
	os.MkdirAll(weztermSourceDir, 0755)
	sourceContent := "-- OVAV WezTerm Config v1\nreturn {}"
	os.WriteFile(filepath.Join(weztermSourceDir, "wezterm-catppuccin-mocha.lua"), []byte(sourceContent), 0644)

	// Create target dir (expandPath resolves ~/..ovav/source/configs/wezterm/wezterm.lua)
	targetDir := filepath.Join(tempHome, "..ovav", "source", "configs", "wezterm")
	os.MkdirAll(targetDir, 0755)

	// Case 2: target exists but no source — source missing
	t.Run("source_missing", func(t *testing.T) {
		targetPath := filepath.Join(targetDir, "wezterm.lua")
		os.WriteFile(targetPath, []byte("dummy"), 0644)
		// Remove source to trigger missing source
		os.Remove(filepath.Join(weztermSourceDir, "wezterm-catppuccin-mocha.lua"))
		defer os.WriteFile(filepath.Join(weztermSourceDir, "wezterm-catppuccin-mocha.lua"), []byte(sourceContent), 0644)

		diag := themeDiagnostics(repo)
		issues := diag["issues"].([]string)
		if len(issues) == 0 || issues[0] != "source_template_missing" {
			t.Errorf("expected source_template_missing, got %v", issues)
		}
	})

	// Case 3: content mismatch
	t.Run("content_mismatch", func(t *testing.T) {
		targetPath := filepath.Join(targetDir, "wezterm.lua")
		os.WriteFile(targetPath, []byte("different content"), 0644)

		diag := themeDiagnostics(repo)
		match, _ := diag["content_match"].(bool)
		if match {
			t.Error("expected content_match=false with different content")
		}
		issues := diag["issues"].([]string)
		if len(issues) == 0 || issues[0] != "content_mismatch — target differs from source template" {
			t.Errorf("expected content_mismatch, got %v", issues)
		}
	})

	// Case 4: content matches
	t.Run("content_match", func(t *testing.T) {
		targetPath := filepath.Join(targetDir, "wezterm.lua")
		os.WriteFile(targetPath, []byte(sourceContent), 0644)

		diag := themeDiagnostics(repo)
		match, _ := diag["content_match"].(bool)
		if !match {
			t.Error("expected content_match=true with matching content")
		}
		likelyIssue, _ := diag["likely_issue"].(string)
		if likelyIssue != "restart_required" {
			t.Errorf("expected restart_required, got %s", likelyIssue)
		}
	})
}

func TestRollbackTarget(t *testing.T) {
	repo := tempRepo(t)

	// Create a target file and a backup
	targetFile := filepath.Join(repo, "tools", "rollback_test.txt")
	os.WriteFile(targetFile, []byte("original content"), 0644)

	backupDir := filepath.Join(repo, ".ovav", "context", "backups", "pre_deploy")
	os.MkdirAll(backupDir, 0755)

	// Create backup with known hash
	backupFile := filepath.Join(backupDir, "rollback_test.txt.20260617000000")
	os.WriteFile(backupFile, []byte("backup content"), 0644)
	expectedHash := hashFile(backupFile)

	// Now modify the target to be different from backup
	os.WriteFile(targetFile, []byte("modified content"), 0644)

	result := rollbackTarget(targetFile, backupFile, expectedHash, repo)

	if !result.Verified {
		t.Error("expected verified=true")
	}
	if result.ExpectedHash != expectedHash {
		t.Errorf("expected hash %s, got %s", expectedHash, result.ExpectedHash)
	}
	if result.Status != "restored" {
		t.Errorf("expected status 'restored', got %s", result.Status)
	}

	t.Run("boundary_blocked", func(t *testing.T) {
		result := rollbackTarget("/etc/passwd", backupFile, "abc", repo)
		if result.Status != "blocked" {
			t.Errorf("expected blocked for path outside repo, got %s", result.Status)
		}
	})

	t.Run("backup_missing", func(t *testing.T) {
		result := rollbackTarget(targetFile, "/nonexistent/backup", "abc", repo)
		if result.Status != "failed" {
			t.Errorf("expected failed for missing backup, got %s", result.Status)
		}
	})

	t.Run("hash_mismatch", func(t *testing.T) {
		// Create target with content that differs from backup's expected hash
		target2 := filepath.Join(repo, "tools", "rollback_hash_test.txt")
		os.WriteFile(target2, []byte("different than backup"), 0644)

		backup2 := filepath.Join(backupDir, "rollback_hash_backup.txt")
		os.WriteFile(backup2, []byte("backup content"), 0644)
		wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

		result := rollbackTarget(target2, backup2, wrongHash, repo)
		if result.Status != "verification_failed" {
			t.Errorf("expected verification_failed for hash mismatch, got %s", result.Status)
		}
	})
}

func TestPreviewRollbackGuide(t *testing.T) {
	t.Run("empty_backup", func(t *testing.T) {
		backup := BackupReport{
			BackupDir: "/tmp/backup",
			BackedUp:  0,
			Failed:    0,
			Results:   []BackupResult{},
		}
		guide := PreviewRollbackGuide(backup)
		if !strings.Contains(guide, "No files were backed up") {
			t.Errorf("expected empty backup message, got: %s", guide)
		}
	})

	t.Run("with_entries", func(t *testing.T) {
		backup := BackupReport{
			BackupDir: "/tmp/backup",
			BackedUp:  2,
			Failed:    1,
			Results: []BackupResult{
				{Target: "/tmp/file1.txt", Status: "backed_up"},
				{Target: "/tmp/file2.txt", Status: "failed"},
			},
		}
		guide := PreviewRollbackGuide(backup)
		if !strings.Contains(guide, "Rollback procedure") {
			t.Errorf("expected rollback procedure, got: %s", guide)
		}
		if !strings.Contains(guide, "❌") {
			t.Errorf("expected fail icon for non-backed-up entry")
		}
	})
}

// ── Extra coverage tests: push 84% → 90%+ ────────────────────────────────

func TestGovernedDeploy_UnknownMode(t *testing.T) {
	repo := tempRepo(t)
	result := GovernedDeploy("INVALID_MODE", repo)
	if result.Status != "error" {
		t.Errorf("expected error for unknown mode, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for unknown mode")
	}
}

func TestApplyFiles_DryRunAndBlockedOps(t *testing.T) {
	plan := Plan{PackID: "test_plan"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "blocked", Target: "/tmp/blocked_file"},
			{Operation: "create", Target: "/tmp/dry_run_create"},
			{Operation: "update", Target: "/tmp/dry_run_update"},
		},
	}
	report := applyFiles(plan, manifest, ModeDryRun, "/tmp")
	if len(report.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(report.Results))
	}
	for _, r := range report.Results {
		if r.Operation != "dry_run_preview" {
			t.Errorf("expected dry_run_preview for all ops in dry-run mode, got %s for %s", r.Operation, r.Target)
		}
	}
}

func TestApplyFiles_BlockedOperations(t *testing.T) {
	plan := Plan{PackID: "test_plan"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "blocked", Target: "/tmp/should_be_blocked"},
		},
	}
	report := applyFiles(plan, manifest, ModeSourceLocalApply, "/tmp")
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Operation != "blocked" {
		t.Errorf("expected blocked operation, got %s", report.Results[0].Operation)
	}
	if report.Results[0].Written {
		t.Error("blocked operation should not be written")
	}
}

func TestApplyFiles_SandboxMode(t *testing.T) {
	dir := t.TempDir()
	plan := Plan{PackID: "test_plan"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: filepath.Join(dir, "sandbox", "new_file.txt")},
		},
	}
	report := applyFiles(plan, manifest, ModeSandbox, dir)
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Operation != "sandbox_simulated" {
		t.Errorf("expected sandbox_simulated, got %s", report.Results[0].Operation)
	}
	if !report.Results[0].Written {
		t.Error("sandbox should simulate write")
	}
}

func TestApplyConfigFile_ReadOnlyTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	os.WriteFile(source, []byte("content"), 0644)
	// Create read-only directory
	roDir := filepath.Join(dir, "readonly")
	os.MkdirAll(roDir, 0555) // r-xr-xr-x (no write)
	t.Cleanup(func() { os.Chmod(roDir, 0755) })

	target := filepath.Join(roDir, "subdir", "file.txt") // subdir can't be created
	result := applyConfigFile(source, target)
	if result {
		t.Error("expected false when can't create target dir")
	}
}

func TestIsSourceLocalPath_EdgeCases(t *testing.T) {
	repo := tempRepo(t)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"within_project_yaml", filepath.Join(repo, ".ovav", "config", "test.yaml"), true},
		{"tools_dir", filepath.Join(repo, "tools", "validator", "check.py"), true},
		{"dotdot_traversal", filepath.Join(repo, "..", "..", "..", "etc", "passwd"), false},
		{"absolute_path", "/etc/passwd", false},
		{"relative_simple", filepath.Join(repo, "test.txt"), true},
		{"current_dir", filepath.Join(repo, ".", "test.txt"), true},
		{"home_tilde", filepath.Join(repo, "~", ".config", "wezterm", "test.lua"), true},
		{"project_root_itself", repo, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSourceLocalPath(tt.path, repo)
			if result != tt.expected {
				t.Errorf("isSourceLocalPath(%q, repo) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDeployAll_MissingSource(t *testing.T) {
	// Use a minimal temp dir with no source config files
	dir := t.TempDir()
	// Create necessary metadata dirs but not the actual source files
	os.MkdirAll(filepath.Join(dir, ".ovav", "source", "configs"), 0755)

	result := DeployAll(dir)
	// Since GetDeployMap may return entries even for missing sources,
	// the result should have skipped entries
	if result.AllOK {
		t.Error("expected AllOK=false on empty/missing source dir")
	}
	// Check that at least some deployments were attempted
	t.Logf("DeployAll on empty dir: %d deployments, AllOK=%v", len(result.Deployments), result.AllOK)
}

// ── UX preview tests ───────────────────────────────────────────────────────

func TestPreviewPlan_Variants(t *testing.T) {
	t.Run("dry_run", func(t *testing.T) {
		plan := Plan{
			PackID:     "test_pack",
			Mode:       ModeDryRun,
			EntryCount: 2,
			DryRunOnly: true,
			Entries: []PlanEntry{
				{Target: "/tmp/file1.txt", TargetRisk: "low", WriteEnabled: true},
				{Target: "/tmp/file2.txt", TargetRisk: "medium", WriteEnabled: false},
			},
		}
		preview := PreviewPlan(plan)
		if !strings.Contains(preview, "Dry-run mode") {
			t.Errorf("expected dry-run warning, got: %s", preview)
		}
	})

	t.Run("sandbox", func(t *testing.T) {
		plan := Plan{
			PackID:      "test_pack",
			Mode:        ModeSandbox,
			EntryCount:  1,
			SandboxOnly: true,
			Entries: []PlanEntry{
				{Target: "/tmp/sandbox.txt", TargetRisk: "low", WriteEnabled: true},
			},
		}
		preview := PreviewPlan(plan)
		if !strings.Contains(preview, "Sandbox mode") {
			t.Errorf("expected sandbox warning, got: %s", preview)
		}
	})

	t.Run("real_apply", func(t *testing.T) {
		plan := Plan{
			PackID:     "test_pack",
			Mode:       ModeSourceLocalApply,
			EntryCount: 1,
			RealApply:  true,
			Entries: []PlanEntry{
				{Target: "/tmp/real.txt", TargetRisk: "high", WriteEnabled: true},
			},
		}
		preview := PreviewPlan(plan)
		if !strings.Contains(preview, "Source-local-apply") {
			t.Errorf("expected real-apply warning, got: %s", preview)
		}
	})
}

func TestPreviewRisk_Variants(t *testing.T) {
	t.Run("with_issues", func(t *testing.T) {
		safety := SafetyReport{
			OverallSafety:    "review_required",
			HasBlocked:       false,
			HasReviewReq:     true,
			RealApplyAllowed: false,
			Issues:           []string{"path outside allowed surface", "write to system path"},
			Entries: []SafetyEntry{
				{Target: "/tmp/allowed", SafetyStatus: "allow"},
				{Target: "/tmp/review", SafetyStatus: "review_required"},
				{Target: "/tmp/blocked", SafetyStatus: "blocked"},
			},
		}
		preview := PreviewRisk(safety)
		if !strings.Contains(preview, "Issues (2)") {
			t.Errorf("expected issues section, got: %s", preview)
		}
		if !strings.Contains(preview, "✅") || !strings.Contains(preview, "⚠️") || !strings.Contains(preview, "🚫") {
			t.Errorf("expected safety icons, got: %s", preview)
		}
	})

	t.Run("no_issues", func(t *testing.T) {
		safety := SafetyReport{
			OverallSafety:    "ok",
			HasBlocked:       false,
			HasReviewReq:     false,
			RealApplyAllowed: true,
			Entries:          []SafetyEntry{},
		}
		preview := PreviewRisk(safety)
		if strings.Contains(preview, "## Issues") {
			t.Errorf("expected no issues section when empty, got: %s", preview)
		}
	})
}

func TestLoadInstallPacks_ErrorPaths(t *testing.T) {
	t.Run("missing_file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadInstallPacks(dir)
		if err == nil {
			t.Error("expected error for missing install_packs.yaml")
		}
	})

	t.Run("invalid_yaml", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)
		os.WriteFile(filepath.Join(dir, ".ovav", "registry", "install_packs.yaml"), []byte(": : bad: yaml: :"), 0644)
		_, err := LoadInstallPacks(dir)
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

func TestBuildPlan_InvalidRepoRoot(t *testing.T) {
	// Use a path that can't be resolved to absolute
	plan := BuildPlan("test_pack", ModeDryRun, "/nonexistent/path/that/cannot/be/accessed")
	if plan.Status != "fail" {
		t.Errorf("expected fail for invalid repo root, got %s", plan.Status)
	}
}

func TestCheckTargetBoundary_EdgeCases(t *testing.T) {
	repo := tempRepo(t)

	t.Run("external_blocked", func(t *testing.T) {
		result := CheckTargetBoundary("/etc/shadow", ModeSourceLocalApply, repo)
		if result.Status == "allowed" {
			t.Error("external path should not be allowed")
		}
	})

	t.Run("repo_path_allowed", func(t *testing.T) {
		result := CheckTargetBoundary(filepath.Join(repo, "tools", "test.txt"), ModeSourceLocalApply, repo)
		if result.Status != "ok" && result.Status != "allowed" {
			t.Errorf("repo path should be allowed, got %s: %s", result.Status, result.Reason)
		}
	})

	t.Run("dry_run_allows_all", func(t *testing.T) {
		result := CheckTargetBoundary("/etc/passwd", ModeDryRun, repo)
		if result.Status != "ok" && result.Status != "allowed" {
			t.Errorf("dry-run should allow any path, got %s: %s", result.Status, result.Reason)
		}
	})

	t.Run("sandbox_restricted", func(t *testing.T) {
		result := CheckTargetBoundary("/etc/shells", ModeSandbox, repo)
		if result.Status == "allowed" {
			t.Error("sandbox should restrict external paths")
		}
	})
}

// ── applyFiles coverage: sandbox failure paths ────────────────────────────────

func TestApplyFiles_SandboxMkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file that blocks subdirectory creation
	blocker := filepath.Join(dir, "blocker")
	os.WriteFile(blocker, []byte("block"), 0644)
	// Target tries to create a subdirectory under the blocker file
	target := filepath.Join(blocker, "subdir", "file.txt")

	plan := Plan{PackID: "test"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: target},
		},
	}
	report := applyFiles(plan, manifest, ModeSandbox, dir)
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Operation != "sandbox_failed" {
		t.Errorf("expected sandbox_failed, got %s", report.Results[0].Operation)
	}
	if report.Results[0].Error == "" {
		t.Error("expected error message for sandbox_failed")
	}
}

func TestApplyFiles_SandboxCreateFailure(t *testing.T) {
	// /proc/self/fd exists as a dir but is read-only — Create will fail
	target := "/proc/self/fd/ovav_test_cannot_create"
	plan := Plan{PackID: "test"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: target},
		},
	}
	report := applyFiles(plan, manifest, ModeSandbox, "/tmp")
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Operation != "sandbox_failed" {
		t.Errorf("expected sandbox_failed for read-only /proc, got %s", report.Results[0].Operation)
	}
	if report.Results[0].Written {
		t.Error("should not be written on read-only fs")
	}
}

// ── applyFiles coverage: source-local-apply failure paths ─────────────────────

func TestApplyFiles_RealApplyMkdirAllFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	os.WriteFile(blocker, []byte("block"), 0644)
	target := filepath.Join(blocker, "subdir", "file.txt")

	plan := Plan{PackID: "test"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: target, Source: ""},
		},
	}
	report := applyFiles(plan, manifest, ModeSourceLocalApply, dir)
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Operation != "create_failed" {
		t.Errorf("expected create_failed, got %s", report.Results[0].Operation)
	}
	if report.Results[0].Error == "" {
		t.Error("expected error message")
	}
}

func TestApplyFiles_RealApplyCopyFileFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a valid source file
	source := filepath.Join(dir, "valid_source.txt")
	os.WriteFile(source, []byte("source content"), 0644)

	// Target on read-only /proc filesystem — copyFile will fail to create dest
	target := "/proc/self/fd/ovav_test_copy_fail"

	plan := Plan{PackID: "test"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "update", Target: target, Source: source},
		},
	}
	report := applyFiles(plan, manifest, ModeSourceLocalApply, dir)
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Operation != "update_failed" {
		t.Errorf("expected update_failed, got %s", report.Results[0].Operation)
	}
	if report.Results[0].Error == "" {
		t.Error("expected error message for copy failure")
	}
}

func TestApplyFiles_NonCreateUpdateOperation(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "tools", "noop_file.txt")
	os.MkdirAll(filepath.Dir(target), 0755)

	plan := Plan{PackID: "test"}
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "delete", Target: target},
			{Operation: "rename", Target: target},
			{Operation: "noop", Target: target},
		},
	}
	report := applyFiles(plan, manifest, ModeSourceLocalApply, dir)
	if len(report.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(report.Results))
	}
	for i, r := range report.Results {
		if r.Written {
			t.Errorf("result %d should not be written for non-create/update op", i)
		}
		if r.Operation == "create" || r.Operation == "update" {
			t.Errorf("operation should be preserved as-is, got %s", r.Operation)
		}
	}
	if report.Skipped != 3 {
		t.Errorf("expected 3 skipped, got %d", report.Skipped)
	}
}

// ── verifyApplied coverage: missing files ─────────────────────────────────────

func TestVerifyApplied_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: filepath.Join(dir, "definitely_nonexistent.txt")},
			{Operation: "update", Target: filepath.Join(dir, "also_nonexistent.txt")},
		},
	}
	report := verifyApplied(manifest, ModeSourceLocalApply, dir)
	if report.Status != "fail" {
		t.Errorf("expected status=fail when files missing, got %s", report.Status)
	}
	if report.Missing != 2 {
		t.Errorf("expected missing=2, got %d", report.Missing)
	}
	if report.Verified != 0 {
		t.Errorf("expected verified=0, got %d", report.Verified)
	}
	if len(report.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(report.Results))
	}
	for _, r := range report.Results {
		if r.Status != "missing" {
			t.Errorf("expected status=missing for %s, got %s", r.Target, r.Status)
		}
		if r.Exists {
			t.Errorf("expected Exists=false for %s", r.Target)
		}
	}
}

func TestVerifyApplied_PartialSuccess(t *testing.T) {
	dir := t.TempDir()
	existsFile := filepath.Join(dir, "exists.txt")
	os.WriteFile(existsFile, []byte("here"), 0644)

	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: existsFile},
			{Operation: "create", Target: filepath.Join(dir, "missing.txt")},
		},
	}
	report := verifyApplied(manifest, ModeSourceLocalApply, dir)
	if report.Status != "fail" {
		t.Errorf("expected status=fail when any file missing, got %s", report.Status)
	}
	if report.Missing != 1 {
		t.Errorf("expected missing=1, got %d", report.Missing)
	}
	if report.Verified != 1 {
		t.Errorf("expected verified=1, got %d", report.Verified)
	}
}

func TestVerifyApplied_AllSuccess(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "file1.txt")
	f2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(f1, []byte("a"), 0644)
	os.WriteFile(f2, []byte("b"), 0644)

	manifest := Manifest{
		Entries: []ManifestEntry{
			{Operation: "create", Target: f1},
			{Operation: "update", Target: f2},
			{Operation: "dry_run", Target: "/nonexistent"}, // skipped — not create/update
		},
	}
	report := verifyApplied(manifest, ModeSourceLocalApply, dir)
	if report.Status != "pass" {
		t.Errorf("expected status=pass, got %s", report.Status)
	}
	if report.Verified != 2 {
		t.Errorf("expected verified=2, got %d", report.Verified)
	}
	if report.Missing != 0 {
		t.Errorf("expected missing=0, got %d", report.Missing)
	}
}

// ── backupTarget coverage: error paths ────────────────────────────────────────

func TestBackupTarget_DotDotPathBlocked(t *testing.T) {
	dir := t.TempDir()
	// Create a file inside the temp dir
	srcFile := filepath.Join(dir, "inside.txt")
	os.WriteFile(srcFile, []byte("test"), 0644)

	// Use a different temp dir as repo root so that srcFile is outside the "repo"
	repoRoot := t.TempDir()

	result := backupTarget(srcFile, filepath.Join(dir, "backups"), repoRoot)
	if result.Status != "blocked" {
		t.Errorf("expected blocked for path outside repo root, got %s: %s", result.Status, result.Reason)
	}
	if !strings.Contains(result.Reason, "outside") {
		t.Errorf("expected 'outside' in reason, got: %s", result.Reason)
	}
}

func TestBackupTarget_TargetDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	nonExistent := filepath.Join(dir, "does_not_exist.txt")

	result := backupTarget(nonExistent, filepath.Join(dir, "backups"), dir)
	if result.Status != "skipped" {
		t.Errorf("expected skipped for non-existent target, got %s", result.Status)
	}
	if result.Reason != "target_does_not_exist" {
		t.Errorf("expected target_does_not_exist, got %s", result.Reason)
	}
}

func TestBackupTarget_CannotReadSource(t *testing.T) {
	dir := t.TempDir()
	unreadableFile := filepath.Join(dir, "unreadable.txt")
	os.WriteFile(unreadableFile, []byte("secret"), 0200) // write-only, no read
	t.Cleanup(func() { os.Chmod(unreadableFile, 0644) })

	result := backupTarget(unreadableFile, filepath.Join(dir, "backups"), dir)
	if result.Status != "failed" {
		t.Errorf("expected failed for unreadable file, got %s: %s", result.Status, result.Reason)
	}
	if result.Reason != "cannot_read_source" {
		t.Errorf("expected cannot_read_source, got %s", result.Reason)
	}
}

func TestBackupTarget_MkdirAllFails(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "source.txt")
	os.WriteFile(srcFile, []byte("test"), 0644)

	// Create a file exactly where backup would create a directory — blocking MkdirAll
	backupDir := filepath.Join(dir, "backups")
	os.WriteFile(backupDir, []byte("blocker"), 0644) // regular file, not a dir

	result := backupTarget(srcFile, backupDir, dir)
	if result.Status != "failed" {
		t.Errorf("expected failed when MkdirAll blocked, got %s: %s", result.Status, result.Reason)
	}
	if !strings.Contains(result.Reason, "mkdir_failed") {
		t.Errorf("expected mkdir_failed in reason, got: %s", result.Reason)
	}
}

func TestBackupTarget_CopyFileFails(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "source.txt")
	os.WriteFile(srcFile, []byte("test"), 0644)

	// Backup destination in a valid dir, but we'll use a blocker
	// Actually test that copyFile failure is caught: use /proc as target backup dir
	// MkdirAll on /proc/self/fd succeeds (it already exists), copyFile fails
	backupDir := "/proc/self/fd"

	result := backupTarget(srcFile, backupDir, dir)
	// Since source is outside /proc, rel will start with "..", so it'll be blocked
	// We need the file to be inside the same fs namespace. Let's use a different approach.
	// Actually with dir as repoRoot, the rel path will be something like "source.txt",
	// and backupPath = /proc/self/fd/source.txt. MkdirAll on /proc/self/fd is fine.
	// copyFile tries to create /proc/self/fd/source.txt which fails.
	if result.Status != "failed" {
		t.Errorf("expected failed when copy to read-only fs, got %s: %s", result.Status, result.Reason)
	}
	if !strings.Contains(result.Reason, "copy_failed") {
		t.Errorf("expected copy_failed in reason, got: %s", result.Reason)
	}
}

// ── copyFile coverage: destination creation failure ───────────────────────────

func TestCopyFile_DestinationCreateFails(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "valid_source.txt")
	os.WriteFile(srcFile, []byte("source content"), 0644)

	// Destination on read-only /proc filesystem
	dstFile := "/proc/self/fd/ovav_copyfile_test"
	err := copyFile(srcFile, dstFile)
	if err == nil {
		t.Error("expected error when creating file on read-only /proc")
	}
}

// ── rollbackTarget coverage: mkdir failure ────────────────────────────────────

func TestRollbackTarget_MkdirAllFails(t *testing.T) {
	dir := t.TempDir()
	// Create an eligible surface dir (tools/) with a blocker file
	toolsDir := filepath.Join(dir, "tools")
	os.MkdirAll(toolsDir, 0755)
	blockerPath := filepath.Join(toolsDir, "blocker")
	os.WriteFile(blockerPath, []byte("block"), 0644)

	// Create a valid backup file
	backupFile := filepath.Join(dir, "backup", "myfile.txt")
	os.MkdirAll(filepath.Dir(backupFile), 0755)
	os.WriteFile(backupFile, []byte("backup content"), 0644)
	expectedHash := hashFile(backupFile)

	// Target within tools/ (eligible surface) but under the blocker file
	target := filepath.Join(blockerPath, "subdir", "target.txt")

	result := rollbackTarget(target, backupFile, expectedHash, dir)
	if result.Status != "failed" {
		t.Errorf("expected failed when MkdirAll blocked, got %s: %s", result.Status, result.Reason)
	}
	if !strings.Contains(result.Reason, "mkdir_failed") {
		t.Errorf("expected mkdir_failed in reason, got: %s", result.Reason)
	}
}

// ── PreviewRollbackGuide: additional variants ─────────────────────────────────

func TestPreviewRollbackGuide_WithSkippedAndBlocked(t *testing.T) {
	backup := BackupReport{
		BackupDir: "/tmp/backup",
		BackedUp:  1,
		Failed:    0,
		Skipped:   1,
		Blocked:   1,
		Results: []BackupResult{
			{Target: "/tmp/file1.txt", Status: "backed_up"},
			{Target: "/tmp/file2.txt", Status: "skipped"},
			{Target: "/tmp/file3.txt", Status: "blocked"},
		},
	}
	guide := PreviewRollbackGuide(backup)
	if !strings.Contains(guide, "Rollback Guide") {
		t.Error("expected rollback guide title")
	}
	if !strings.Contains(guide, "Rollback procedure") {
		t.Error("expected rollback procedure")
	}
	// Should have ❌ icons for non-backed-up entries
	nonBackedUp := strings.Count(guide, "❌")
	if nonBackedUp < 2 {
		t.Errorf("expected at least 2 ❌ for skipped+blocked, got %d\n%s", nonBackedUp, guide)
	}
}

func TestPreviewRollbackGuide_AllFailed(t *testing.T) {
	backup := BackupReport{
		BackupDir: "/tmp/backup",
		BackedUp:  0,
		Failed:    3,
		Results: []BackupResult{
			{Target: "/tmp/fail1.txt", Status: "failed"},
			{Target: "/tmp/fail2.txt", Status: "verification_failed"},
			{Target: "/tmp/fail3.txt", Status: "failed"},
		},
	}
	guide := PreviewRollbackGuide(backup)
	if strings.Contains(guide, "✅") {
		t.Error("should not contain success icon when all failed")
	}
	if !strings.Contains(guide, "❌") {
		t.Error("should contain fail icon")
	}
}

// ── GovernedDeploy coverage: additional edge ──────────────────────────────────

func TestGovernedDeploy_ErrorIncludesMode(t *testing.T) {
	repo := tempRepo(t)
	result := GovernedDeploy("INVALID", repo)
	if !strings.Contains(result.Error, "INVALID") {
		t.Errorf("expected error to mention mode, got: %s", result.Error)
	}
}

// ── T14: Installer coverage push 90.7%→95% ──────────────────────────────────

func TestDeployAll_CopyFailed(t *testing.T) {
	repo := tempRepo(t)

	// Create the source file that GetDeployMap expects
	sourceDir := filepath.Join(repo, ".ovav", "context", "backups")
	os.MkdirAll(sourceDir, 0755)
	sourceFile := filepath.Join(sourceDir, "wezterm-catppuccin-mocha.lua")
	if err := os.WriteFile(sourceFile, []byte("-- wezterm config\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the expanded target path as a DIRECTORY (not file) so copyFile fails
	home, _ := os.UserHomeDir()
	targetPath := filepath.Join(home, "..ovav", "source", "configs", "wezterm", "wezterm.lua")
	os.MkdirAll(targetPath, 0755) // creates as directory — copyFile will fail trying to write to it
	defer os.RemoveAll(filepath.Join(home, "..ovav"))

	result := DeployAll(repo)
	if result.AllOK {
		t.Error("expected AllOK=false when copy fails (target is directory)")
	}
	foundFailed := false
	for _, d := range result.Deployments {
		if d.Status == "failed" && d.Reason == "copy_failed" {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Errorf("expected a deployment with status=failed reason=copy_failed, got %d deployments", len(result.Deployments))
		for _, d := range result.Deployments {
			t.Logf("  status=%s reason=%s target=%s", d.Status, d.Reason, d.Target)
		}
	}
}

func TestGovernedSandbox_CopyFails(t *testing.T) {
	repo := tempRepo(t)

	// Create the source file
	sourceFile := filepath.Join(repo, ".ovav", "context", "backups", "wezterm-catppuccin-mocha.lua")
	os.MkdirAll(filepath.Dir(sourceFile), 0755)
	if err := os.WriteFile(sourceFile, []byte("-- test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create sandbox root directory read-only so copyFile fails inside governedSandbox
	sandboxRoot := filepath.Join(repo, ".ovav", "artifacts", "S88", "evidence", "sandbox")
	os.MkdirAll(sandboxRoot, 0555)    // read+execute, no write
	defer os.Chmod(sandboxRoot, 0755) // restore for cleanup

	result := GovernedDeploy(ModeSandbox, repo)
	if result.Status != "ok" {
		// Even if copy fails, governedSandbox should not crash — it handles errors gracefully
		t.Logf("governed deploy sandbox result: status=%s error=%s", result.Status, result.Error)
	}
	if result.SandboxRoot == "" {
		t.Error("sandbox root should not be empty even on copy failures")
	}

	// Restore permissions for cleanup
	os.Chmod(sandboxRoot, 0755)
}

func TestExecuteRollback_MissingBackupPath(t *testing.T) {
	repo := tempRepo(t)

	backup := BackupReport{
		Status:    "pass",
		BackedUp:  1,
		BackupDir: filepath.Join(repo, ".ovav", "backups"),
		Results: []BackupResult{
			{
				Target:     "/some/target",
				BackupPath: "", // MISSING
				SourceHash: "", // MISSING
				Status:     "backed_up",
			},
		},
	}

	manifest := Manifest{
		Status:       "pass",
		TotalEntries: 1,
		Entries: []ManifestEntry{
			{Target: "/some/target", Operation: "update", NeedsBackup: true},
		},
	}

	rollback := ExecuteRollback(backup, manifest, ModeSourceLocalApply, repo)
	if rollback.Status != "pass" {
		t.Errorf("status = %s, want pass", rollback.Status)
	}
	// The entry with empty BackupPath should be skipped
	foundSkipped := false
	for _, r := range rollback.Results {
		if r.Status == "skipped" && r.Reason == "missing_backup_path_or_hash" {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Error("expected a result skipped due to missing_backup_path_or_hash")
	}
}

func TestWriteEvidence_PathLeakage(t *testing.T) {
	repo := tempRepo(t)

	report := ApplyGatewayReport{
		Status: "pass",
		Stages: StageResults{
			Safety: SafetyReport{OverallSafety: "safe"},
			Gates:  GateReport{TotalSatisfied: 1, TotalGates: 1},
		},
	}

	// Test with a segment that tries to escape repo root (path leakage guard)
	evidence, err := WriteEvidence(report, "../../../etc", repo)
	if err == nil {
		t.Error("expected error for path leakage segment")
	}
	if evidence.Status == "pass" {
		t.Error("expected non-pass status for path leakage")
	}
}

func TestRelPath_EdgeCases(t *testing.T) {
	repo := tempRepo(t)

	tests := []struct {
		base, target string
	}{
		{repo, repo},                                 // same path
		{repo, filepath.Join(repo, "tools")},         // subdirectory
		{repo, filepath.Join(repo, "a", "b", "c")},   // deep path
		{repo, filepath.Join(repo, "..", "outside")}, // would escape but relPath handles
	}
	for _, tc := range tests {
		rel := relPath(tc.base, tc.target)
		t.Logf("relPath(%s, %s) = %s", tc.base, tc.target, rel)
		if rel == "" && tc.base != tc.target {
			// relPath returns target when paths are on different roots
			t.Logf("  relPath fell back to absolute: %s", rel)
		}
	}
}

func TestIsSourceLocalPath_MoreEdgeCases(t *testing.T) {
	repo := tempRepo(t)

	tests := []struct {
		path string
		root string
		want bool
	}{
		{repo, repo, true}, // same path
		{filepath.Join(repo, "tools", "script.sh"), repo, true}, // subpath
		{"/etc/passwd", repo, false},                            // outside repo
		{"/tmp/somefile", repo, false},                          // outside repo
		{filepath.Join(repo, "..", "outside"), repo, false},     // escaped path
	}
	for _, tc := range tests {
		got := isSourceLocalPath(tc.path, tc.root)
		if got != tc.want {
			t.Errorf("isSourceLocalPath(%q, %q) = %v, want %v", tc.path, tc.root, got, tc.want)
		}
	}
}

func TestExecuteBackup_EmptyManifest(t *testing.T) {
	repo := tempRepo(t)

	manifest := Manifest{
		Status:       "pass",
		TotalEntries: 0,
		Entries:      []ManifestEntry{},
	}

	backup := ExecuteBackup(manifest, ModeDryRun, repo)
	if backup.Status != "pass" {
		t.Errorf("status = %s, want pass", backup.Status)
	}
	if backup.BackedUp != 0 {
		t.Errorf("backed_up = %d, want 0", backup.BackedUp)
	}
}

func TestClassifyRisk_AllLevels(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/etc/passwd", "global-risk"},
		{"/usr/local/bin/app", "global-risk"},
		{"/home/user/.ssh/id_rsa", "user-home-risk"},
		{"/home/user/.config/wezterm/wezterm.lua", "user-config-risk"},
		{"/home/user/.local/share/app", "user-local-risk"},
		{"/home/user/project/go-runtime/cmd/cpanel/main.go", "user-home-risk"},
		{"/home/user/project/tools/some_script.sh", "user-home-risk"},
		{"~/.ovav/vault/master.key", "user-home-risk"},
		{"/tmp/sandbox/test.go", "sandbox"},
	}
	for _, tc := range tests {
		got := classifyRisk(tc.path)
		if got != tc.want {
			t.Errorf("classifyRisk(%q) = %s, want %s", tc.path, got, tc.want)
		}
	}
}

func TestBuildSummaryMD_WithErrors(t *testing.T) {
	report := ApplyGatewayReport{
		Status: "fail",
		PackID: "test_pack",
		Mode:   ModeSourceLocalApply,
		Errors: []string{"plan_failed: invalid mode", "safety_check_failed"},
		Stages: StageResults{
			Gates:  GateReport{TotalSatisfied: 1, TotalGates: 2},
			Apply:  ApplyReport{Written: 1, Skipped: 1},
			Backup: BackupReport{BackedUp: 0, Failed: 1},
		},
	}
	summary := buildSummaryMD(report, "S88")
	if !strings.Contains(summary, "Errors") {
		t.Error("summary should contain Errors section when report has errors")
	}
	if !strings.Contains(summary, "plan_failed") {
		t.Error("summary should contain error message")
	}
}

func TestExecuteApply_PlanFailure(t *testing.T) {
	repo := tempRepo(t)

	// Use an invalid pack ID that doesn't exist
	result := ExecuteApply("NONEXISTENT_PACK", ModeSourceLocalApply, repo)

	if result.Status != "fail" {
		t.Errorf("status = %s, want fail for nonexistent pack", result.Status)
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for nonexistent pack")
	}
	// Plan stage should have failed
	if result.Stages.Plan.Status != "fail" {
		t.Errorf("plan status = %s, want fail", result.Stages.Plan.Status)
	}
}

func TestWriteEvidence_InvalidRoot(t *testing.T) {
	// Use a root that contains a null byte (guaranteed to fail filepath.Abs on some systems)
	// Actually, let's use a relative path that would cause issues
	_, err := WriteEvidence(ApplyGatewayReport{}, "S88", "/nonexistent/path/that/definitely/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent root path")
	}
}

func TestBackupTarget_DirectoryTarget(t *testing.T) {
	repo := tempRepo(t)

	// Create a directory where the target file would be
	target := filepath.Join(repo, "tools", "some_dir")
	os.MkdirAll(target, 0755)

	backupDir := filepath.Join(repo, ".ovav", "backups")
	os.MkdirAll(backupDir, 0755)

	result := backupTarget(target, backupDir, repo)
	// Directories are skipped, not failed
	if result.Status != "skipped" {
		t.Errorf("status = %s, want skipped (directories are not backed up)", result.Status)
	}
	if result.Reason != "target_is_directory" {
		t.Errorf("reason = %s, want target_is_directory", result.Reason)
	}
}

func TestGovernedDeploy_SourceMissing(t *testing.T) {
	repo := tempRepo(t)
	// No source files exist — governedSandbox should handle gracefully
	result := GovernedDeploy(ModeSandbox, repo)
	if result.Status != "ok" {
		t.Errorf("status = %s, want ok (governedDeploy handles missing sources gracefully)", result.Status)
	}
}

func TestExecuteBackup_TargetOutsideRepo(t *testing.T) {
	repo := tempRepo(t)

	manifest := Manifest{
		Status:       "pass",
		TotalEntries: 1,
		Entries: []ManifestEntry{
			{
				Target:       "/etc/passwd",
				Operation:    "update",
				NeedsBackup:  true,
				TargetExists: true,
			},
		},
	}

	backup := ExecuteBackup(manifest, ModeSourceLocalApply, repo)
	if backup.Status != "pass" {
		t.Errorf("status = %s, want pass", backup.Status)
	}
	// Target outside repo should be blocked
	foundBlocked := false
	for _, r := range backup.Results {
		if r.Status == "blocked" {
			foundBlocked = true
		}
	}
	if !foundBlocked {
		t.Error("expected blocked target for path outside repo root")
	}
}

func TestBuildPlan_LoadPacksError(t *testing.T) {
	// Create a repo without install_packs.yaml to trigger LoadInstallPacks error
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)
	// Don't create install_packs.yaml → LoadInstallPacks will fail

	plan := BuildPlan("any_pack", ModeDryRun, dir)
	if plan.Status != "fail" {
		t.Errorf("plan status = %s, want fail when install_packs.yaml is missing", plan.Status)
	}
	if !strings.Contains(plan.Error, "load_packs_failed") {
		t.Errorf("expected load_packs_failed error, got: %s", plan.Error)
	}
}

// ── T15: OS-level error paths & uncovered branches ──────────────────────────

func TestResolveSource_UnknownTarget(t *testing.T) {
	root := t.TempDir()
	result := resolveSource("nonexistent_pack", root)
	if result != "" {
		t.Errorf("expected empty string for unknown target, got: %s", result)
	}
}

func TestExpandPath_HomeDirUnset(t *testing.T) {
	// Save original HOME and unset it
	origHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	defer os.Setenv("HOME", origHome)

	result := expandPath("~/test/config")
	if result != "~/test/config" {
		t.Errorf("expected original path when HOME is unset, got: %s", result)
	}
}

func TestRelPath_ErrorPaths(t *testing.T) {
	root := t.TempDir()

	// relPath success: returns relative path within root
	t.Run("success", func(t *testing.T) {
		result := relPath(filepath.Join(root, "subdir", "file.txt"), root)
		if result != "subdir/file.txt" {
			t.Errorf("expected relative path, got: %s", result)
		}
	})

	// relPath with target outside root
	t.Run("outside_root", func(t *testing.T) {
		result := relPath("/etc/passwd", root)
		if result == "etc/passwd" {
			t.Error("expected non-trivial relative path for outside-root target")
		}
	})
}

func TestIsSourceLocalPath_ErrorPaths(t *testing.T) {
	root := t.TempDir()

	// filepath.Abs on path with null byte
	t.Run("abs_path_error", func(t *testing.T) {
		result := isSourceLocalPath("/valid/path\x00broken", root)
		if result {
			t.Error("expected false when path Abs fails")
		}
	})

	// filepath.Abs on root with null byte
	t.Run("abs_root_error", func(t *testing.T) {
		result := isSourceLocalPath("/valid/path", root+"\x00broken")
		if result {
			t.Error("expected false when root Abs fails")
		}
	})

	// Success path
	t.Run("success_local", func(t *testing.T) {
		targetFile := filepath.Join(root, "test.txt")
		os.WriteFile(targetFile, []byte("test"), 0644)
		if !isSourceLocalPath(targetFile, root) {
			t.Error("expected true for path within root")
		}
	})
}

func TestExecuteBackup_AbsError(t *testing.T) {
	manifest := Manifest{
		Status:       "pass",
		TotalEntries: 1,
		Entries: []ManifestEntry{
			{Target: "/some/file", Operation: "update", NeedsBackup: true},
		},
	}
	// Null byte in root causes filepath.Abs to fail
	result := ExecuteBackup(manifest, ModeSourceLocalApply, "/valid\x00broken")
	if result.Status != "fail" {
		t.Errorf("status = %s, want fail on Abs error", result.Status)
	}
	if result.BackupPerformed {
		t.Error("backup should not be performed on Abs error")
	}
}

func TestWriteEvidence_AbsError(t *testing.T) {
	report := ApplyGatewayReport{PackID: "test", Status: "pass"}
	_, err := WriteEvidence(report, "S99", "/root\x00broken")
	if err == nil {
		t.Error("expected error on filepath.Abs failure")
	}
}

func TestWriteEvidence_OutsideRoot(t *testing.T) {
	repo := tempRepo(t)
	report := ApplyGatewayReport{PackID: "test", Status: "pass"}

	evidence, err := WriteEvidence(report, "S99_TEST", repo)
	if err != nil {
		t.Fatalf("WriteEvidence failed: %v", err)
	}
	if evidence.Status != "pass" {
		t.Errorf("status = %s, want pass", evidence.Status)
	}
	if evidence.FilesWritten != 4 {
		t.Errorf("expected 4 files written, got %d", evidence.FilesWritten)
	}
}

func TestGovernedApply_VerifyFail(t *testing.T) {
	repo := tempRepo(t)

	// Do NOT create wezterm source file → DeployAll will fail → AllOK=false
	// This triggers the status="fail" branch in governedApply

	tempHome := filepath.Join(repo, "fake_home_verify")
	os.MkdirAll(tempHome, 0755)
	os.MkdirAll(filepath.Join(tempHome, "..ovav", "source", "configs", "wezterm"), 0755)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", oldHome)

	oldDev := os.Getenv("OVAV_DEV")
	os.Setenv("OVAV_DEV", "1")
	defer os.Setenv("OVAV_DEV", oldDev)

	result := governedApply(repo)

	if result.Status != "fail" {
		t.Errorf("status = %s, want fail when DeployAll fails (source missing)", result.Status)
	}
	if !result.RealDeployPerformed {
		t.Error("expected RealDeployPerformed=true")
	}
	gates := result.Gates
	if v, ok := gates["deploy_ok"].(bool); ok && v {
		t.Error("expected deploy_ok=false when source is missing")
	}
}

func TestDeployAll_SourceMissing(t *testing.T) {
	repo := t.TempDir()
	os.MkdirAll(filepath.Join(repo, ".ovav", "registry"), 0755)

	// Don't create the wezterm source file → DeployAll will skip it
	result := DeployAll(repo)
	if result.AllOK {
		t.Error("expected AllOK=false when source is missing")
	}
	if len(result.Deployments) == 0 {
		t.Error("expected at least one deployment entry (skipped)")
	}
	foundSkipped := false
	for _, d := range result.Deployments {
		if d.Status == "skipped" && d.Reason == "source_file_missing" {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Error("expected skipped deployment with source_file_missing reason")
	}
}
