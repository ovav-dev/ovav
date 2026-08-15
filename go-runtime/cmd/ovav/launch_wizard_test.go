package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── LaunchGate unit tests ──────────────────────────────────────────────────

func TestGetLaunchGates_NotEmpty(t *testing.T) {
	gates := getLaunchGates()
	if len(gates) == 0 {
		t.Fatal("no gates registered")
	}
	for _, g := range gates {
		if g.ID == "" {
			t.Fatal("gate without ID")
		}
		if g.Description == "" {
			t.Errorf("gate %s without description", g.ID)
		}
		if g.Check == nil {
			t.Errorf("gate %s without Check function", g.ID)
		}
	}
}

func TestGetLaunchGates_HasCEOGates(t *testing.T) {
	gates := getLaunchGates()
	ceoCount := 0
	for _, g := range gates {
		if g.CEORequired {
			ceoCount++
		}
	}
	// Per ADR-014: at least 2 CEO gates (pin + verify)
	if ceoCount < 2 {
		t.Fatalf("expected at least 2 CEO gates, got %d", ceoCount)
	}
}

func TestGatePinnedBaseline_NotExists(t *testing.T) {
	gates := getLaunchGates()
	var gate LaunchGate
	for _, g := range gates {
		if g.ID == "pinned_baseline" {
			gate = g
			break
		}
	}

	// Without baseline.pinned.json
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "integrity_backups"), 0o755)

	passed, _, _ := gate.Check(root)
	if passed {
		t.Fatal("gate should fail when baseline.pinned.json missing")
	}
	if !gate.CEORequired {
		t.Fatal("pinned_baseline should be CEO-required")
	}
}

func TestGatePinnedBaseline_Exists(t *testing.T) {
	gates := getLaunchGates()
	var gate LaunchGate
	for _, g := range gates {
		if g.ID == "pinned_baseline" {
			gate = g
			break
		}
	}

	// Create baseline.pinned.json
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "integrity_backups"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "integrity_backups", "baseline.pinned.json"),
		[]byte(`{"schema":"v1","algorithm":"sha256","files":{}}`), 0o644)

	passed, detail, _ := gate.Check(root)
	if !passed {
		t.Fatalf("expected pass, got fail: %s", detail)
	}
}

func TestGateEvidenceCaptured(t *testing.T) {
	gates := getLaunchGates()
	var gate LaunchGate
	for _, g := range gates {
		if g.ID == "evidence_captured" {
			gate = g
			break
		}
	}

	root := t.TempDir()

	// Empty dir → fail
	passed, _, _ := gate.Check(root)
	if passed {
		t.Fatal("gate should fail with no evidence files")
	}

	// Create 4 files → pass
	evidenceDir := filepath.Join(root, ".ovav", "registry", "launch_evidence")
	os.MkdirAll(evidenceDir, 0o755)
	for i := 0; i < 4; i++ {
		os.WriteFile(filepath.Join(evidenceDir, "file-"+string(rune('a'+i))+".txt"),
			[]byte("evidence"), 0o644)
	}
	passed, _, _ = gate.Check(root)
	if !passed {
		t.Fatal("expected pass with 4 evidence files")
	}
}

func TestGateEvidenceCaptured_Partial(t *testing.T) {
	gates := getLaunchGates()
	var gate LaunchGate
	for _, g := range gates {
		if g.ID == "evidence_captured" {
			gate = g
			break
		}
	}

	root := t.TempDir()
	evidenceDir := filepath.Join(root, ".ovav", "registry", "launch_evidence")
	os.MkdirAll(evidenceDir, 0o755)
	// Only 2 files (less than threshold of 4)
	for i := 0; i < 2; i++ {
		os.WriteFile(filepath.Join(evidenceDir, "file.txt"),
			[]byte("evidence"), 0o644)
	}

	passed, detail, _ := gate.Check(root)
	if passed {
		t.Fatal("expected fail with only 2 evidence files")
	}
	if !contains(detail, "2 evidence files") {
		t.Logf("detail: %s", detail)
	}
}

func TestGateProductionReady(t *testing.T) {
	gates := getLaunchGates()
	var gate LaunchGate
	for _, g := range gates {
		if g.ID == "production_ready" {
			gate = g
			break
		}
	}

	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)

	// Status: launch_verification_blocked → fail
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("status: launch_verification_blocked\n"), 0o644)
	passed, _, _ := gate.Check(root)
	if passed {
		t.Fatal("expected fail when status is launch_verification_blocked")
	}

	// Status: production_ready → pass
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("status: production_ready\n"), 0o644)
	passed, _, _ = gate.Check(root)
	if !passed {
		t.Fatal("expected pass when status is production_ready")
	}
}

func TestGateProductionReady_MissingCapsFile(t *testing.T) {
	gates := getLaunchGates()
	var gate LaunchGate
	for _, g := range gates {
		if g.ID == "production_ready" {
			gate = g
			break
		}
	}

	root := t.TempDir()
	// No caps.yaml
	passed, detail, _ := gate.Check(root)
	if passed {
		t.Fatal("expected fail when caps.yaml missing")
	}
	if !contains(detail, "caps.yaml") {
		t.Fatalf("expected detail to mention caps.yaml, got: %s", detail)
	}
}

// ── Wizard helper tests ───────────────────────────────────────────────────

func TestOverallEmoji(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"ready", "✅"},
		{"needs-ceo-attention", "⏳"},
		{"blocked", "❌"},
		{"unknown", "❌"},
	}
	for _, tc := range tests {
		got := overallEmoji(tc.status)
		if got != tc.want {
			t.Errorf("overallEmoji(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestGateEmoji(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"pass", "✅"},
		{"fail", "❌"},
		{"skipped", "⏳"},
		{"unknown", "⏳"},
	}
	for _, tc := range tests {
		got := gateEmoji(tc.status)
		if got != tc.want {
			t.Errorf("gateEmoji(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestExtractFailureSummary(t *testing.T) {
	input := "✅ All passed\n❌ some validator failed\nOK"
	got := extractFailureSummary(input)
	if !contains(got, "failed") {
		t.Fatalf("expected summary mentioning failure, got: %s", got)
	}
}

func TestExtractFailureSummary_NoFailures(t *testing.T) {
	input := "all passed\nno issues"
	got := extractFailureSummary(input)
	if got != "validators failed" {
		t.Fatalf("expected default message, got: %s", got)
	}
}

func TestExtractSmokeFailures(t *testing.T) {
	input := "long output\nSummary: 19 passed, 2 failed"
	got := extractSmokeFailures(input)
	if !contains(got, "Summary:") {
		t.Fatalf("expected Summary in extract, got: %s", got)
	}
}

// ── Launch CLI tests ───────────────────────────────────────────────────────

// TestCmdLaunch_DispatchNoArgs defined in launch_cli_test.go

func TestCmdLaunch_DispatchHelp2(t *testing.T) {
	code := cmdLaunch([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdLaunch_DispatchUnknownGate(t *testing.T) {
	code := cmdLaunch([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestCmdLaunch_Roadmap(t *testing.T) {
	code := cmdLaunch([]string{"roadmap"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdLaunch_VerifyNoWaiver(t *testing.T) {
	code := cmdLaunch([]string{"verify"})
	if code != 1 {
		t.Fatalf("expected 1 (no waiver), got %d", code)
	}
}

func TestRunLaunchCEODecide_NoGate(t *testing.T) {
	code := runLaunchCEODecide([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no gate), got %d", code)
	}
}

func TestRunLaunchCEODecide_NoReason(t *testing.T) {
	code := runLaunchCEODecide([]string{"--gate=pin"})
	if code != 1 {
		t.Fatalf("expected 1 (no reason), got %d", code)
	}
}

func TestRunLaunchCEODecide_UnknownGate(t *testing.T) {
	code := runLaunchCEODecide([]string{"--gate=unknown", "--reason=test"})
	if code != 1 {
		t.Fatalf("expected 1 (unknown gate), got %d", code)
	}
}

// ── ReadinessReport tests ─────────────────────────────────────────────────

func TestReadinessReport_DetermineOverall(t *testing.T) {
	gates := getLaunchGates()
	report := ReadinessReport{}
	for _, g := range gates {
		passed, _, ceoRequired := g.Check(t.TempDir())
		// For test, we just use a stub result
		report.Gates = append(report.Gates, GateReport{
			ID:          g.ID,
			Description: g.Description,
			Status:      ternary(passed, "pass", "fail"),
			CEORequired: ceoRequired,
		})
	}
	// Just verify structure (overall is computed separately)
	if len(report.Gates) != len(gates) {
		t.Fatalf("expected %d gates, got %d", len(gates), len(report.Gates))
	}
}

func TestSaveReadinessReport(t *testing.T) {
	root := t.TempDir()
	report := ReadinessReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RepoRoot:  root,
		Overall:   "ready",
	}
	if err := saveReadinessReport(root, report); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	path := filepath.Join(root, ".ovav", "registry", "launch_readiness.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

// ── Helper ─────────────────────────────────────────────────────────────────

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
