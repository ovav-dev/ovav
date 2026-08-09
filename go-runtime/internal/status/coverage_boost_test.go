package status

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── round1 ────────────────────────────────────────────────────────────────────

func TestRound1(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0.05, 0.0},
		{1.23, 1.2},
		{9.95, 9.9},
		{-0.05, 0.0},
	}
	for _, tt := range tests {
		if got := round1(tt.in); got != tt.want {
			t.Errorf("round1(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// ── checkGovernor: session marker freshness ────────────────────────────────────

func TestCheckGovernor_SessionMarkerFresh(t *testing.T) {
	dir := t.TempDir()
	ovav := filepath.Join(dir, ".ovav")
	runtime := filepath.Join(ovav, "runtime")
	os.MkdirAll(filepath.Join(ovav, "policy"), 0755)
	os.MkdirAll(filepath.Join(ovav, "service_areas"), 0755)
	os.MkdirAll(runtime, 0755)
	os.WriteFile(filepath.Join(ovav, "policy", "permission_authority.json"), []byte("{}"), 0644)

	// Write a fresh session marker (< 2 hours old)
	marker := filepath.Join(runtime, ".session_marker")
	os.WriteFile(marker, []byte("1"), 0644)

	e := New(dir)
	g := e.checkGovernor()
	if g.Status != "active" {
		t.Errorf("expected active, got status=%q", g.Status)
	}
	// SessionAgeMin is set when session marker is fresh; value may round to 0 for very fresh files
	if g.SessionAgeMin < 0 || g.SessionAgeMin > 120 {
		t.Errorf("SessionAgeMin = %v, want 0..120", g.SessionAgeMin)
	}
}

func TestCheckGovernor_SessionMarkerStale(t *testing.T) {
	dir := t.TempDir()
	ovav := filepath.Join(dir, ".ovav")
	runtime := filepath.Join(ovav, "runtime")
	os.MkdirAll(filepath.Join(ovav, "policy"), 0755)
	os.MkdirAll(filepath.Join(ovav, "service_areas"), 0755)
	os.MkdirAll(runtime, 0755)
	os.WriteFile(filepath.Join(ovav, "policy", "permission_authority.json"), []byte("{}"), 0644)

	// Write a stale session marker (> 2 hours old)
	marker := filepath.Join(runtime, ".session_marker")
	os.WriteFile(marker, []byte("1"), 0644)
	stale := time.Now().Add(-3 * time.Hour)
	os.Chtimes(marker, stale, stale)

	e := New(dir)
	g := e.checkGovernor()
	if g.Status != "active" || g.SessionAgeMin != 0 {
		t.Errorf("expected active without session age, got status=%q age=%v", g.Status, g.SessionAgeMin)
	}
}

// ── checkGovernor: degraded (missing policy or service_areas) ──────────────────

func TestCheckGovernor_Degraded(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
	// No policy dir → degraded
	e := New(dir)
	g := e.checkGovernor()
	if g.Status != "degraded" {
		t.Errorf("Governor.Status = %q, want degraded", g.Status)
	}
}

// ── checkMemory: govActive path ────────────────────────────────────────────────

func TestCheckMemory_GovernorActive(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtime, 0755)
	os.WriteFile(filepath.Join(runtime, "memory_governor_active"), []byte("1"), 0644)

	e := New(dir)
	m := e.checkMemory()
	if m.Status != "active" {
		t.Errorf("Memory.Status = %q, want active", m.Status)
	}
}

// ── checkMemory: memory dir exists but no active markers ───────────────────────

func TestCheckMemory_Inactive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "memory"), 0755)

	e := New(dir)
	m := e.checkMemory()
	if m.Status != "inactive" {
		t.Errorf("Memory.Status = %q, want inactive", m.Status)
	}
}

// ── checkMemory: no memory dir at all ─────────────────────────────────────────

func TestCheckMemory_Absent(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	m := e.checkMemory()
	if m.Status != "absent" {
		t.Errorf("Memory.Status = %q, want absent", m.Status)
	}
}

// ── checkIntegrity: valid JSON file ────────────────────────────────────────────

func TestCheckIntegrity_FromFile(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtime, 0755)

	data := `{"status":"pass","label":"Integrity: 5/5","icon":"🟢","total":5,"intact":5}`
	os.WriteFile(filepath.Join(runtime, "integrity_status.json"), []byte(data), 0644)

	e := New(dir)
	ig := e.checkIntegrity()
	if ig.Status != "pass" {
		t.Errorf("Integrity.Status = %q, want pass", ig.Status)
	}
	if ig.Total != 5 {
		t.Errorf("Integrity.Total = %d, want 5", ig.Total)
	}
}

// ── checkIntegrity: invalid JSON ───────────────────────────────────────────────

func TestCheckIntegrity_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtime, 0755)
	os.WriteFile(filepath.Join(runtime, "integrity_status.json"), []byte("not-json"), 0644)

	e := New(dir)
	ig := e.checkIntegrity()
	if ig.Status != "unknown" {
		t.Errorf("Integrity.Status = %q, want unknown", ig.Status)
	}
}

// ── checkIntegrity: no file → quick check ──────────────────────────────────────

func TestCheckIntegrity_NoFile_QuickCheck(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.0"), 0644)
	// Create the other critical files
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte("{}"), 0644)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "area_boundary_enforcement.yaml"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte(""), 0644)

	e := New(dir)
	ig := e.checkIntegrity()
	if ig.Status != "pass" {
		t.Errorf("Integrity.Status = %q, want pass (all files present)", ig.Status)
	}
}

// ── quickIntegrityCheck: some files missing ────────────────────────────────────

func TestQuickIntegrityCheck_SomeMissing(t *testing.T) {
	dir := t.TempDir()
	// Only create AGENTS.md, others missing
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)

	e := New(dir)
	ig := e.quickIntegrityCheck()
	if ig.Status != "fail" {
		t.Errorf("Integrity.Status = %q, want fail", ig.Status)
	}
	if len(ig.Compromised) == 0 {
		t.Error("expected compromised files list")
	}
}

// ── gitBranch: detached HEAD ──────────────────────────────────────────────────

func TestGitBranch_Detached(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc1234def5678"), 0644)

	e := New(dir)
	branch := e.gitBranch()
	if branch != "abc1234" {
		t.Errorf("gitBranch() = %q, want %q", branch, "abc1234")
	}
}

func TestGitBranch_ShortDetached(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc"), 0644)

	e := New(dir)
	branch := e.gitBranch()
	if branch != "detached" {
		t.Errorf("gitBranch() = %q, want detached", branch)
	}
}

func TestGitBranch_NoFile(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	branch := e.gitBranch()
	if branch != "unknown" {
		t.Errorf("gitBranch() = %q, want unknown", branch)
	}
}

// ── checkBranch: protected branch ─────────────────────────────────────────────

func TestCheckBranch_Protected(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	e := New(dir)
	b := e.checkBranch()
	if !b.Protected {
		t.Error("branch should be protected")
	}
	if b.Label != "🔒 main" {
		t.Errorf("Label = %q, want %q", b.Label, "🔒 main")
	}
}

func TestCheckBranch_Unprotected(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature-x"), 0644)

	e := New(dir)
	b := e.checkBranch()
	if b.Protected {
		t.Error("feature-x should not be protected")
	}
}

// ── checkTokens: valid JSON ───────────────────────────────────────────────────

func TestCheckTokens_FromFile(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtime, 0755)

	data := `{"total_all":1000,"total_input":600,"total_output":400,"measurements":5,"current_usage_pct":42.5,"bpe_verified":true}`
	os.WriteFile(filepath.Join(runtime, "session_token_stats.json"), []byte(data), 0644)

	e := New(dir)
	ts := e.checkTokens()
	if ts.TotalAll != 1000 {
		t.Errorf("TotalAll = %d, want 1000", ts.TotalAll)
	}
	if ts.BPEVerified != true {
		t.Error("BPEVerified should be true")
	}
}

func TestCheckTokens_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	runtime := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(runtime, 0755)
	os.WriteFile(filepath.Join(runtime, "session_token_stats.json"), []byte("bad"), 0644)

	e := New(dir)
	ts := e.checkTokens()
	if ts.TotalAll != 0 {
		t.Errorf("TotalAll = %d, want 0 (parse error)", ts.TotalAll)
	}
}

func TestCheckTokens_NoFile(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	ts := e.checkTokens()
	if ts.TotalAll != 0 {
		t.Errorf("TotalAll = %d, want 0 (no file)", ts.TotalAll)
	}
}

// ── WriteMarkers: governor active ─────────────────────────────────────────────

func TestWriteMarkers_GovernorActive(t *testing.T) {
	dir := t.TempDir()
	ovav := filepath.Join(dir, ".ovav")
	runtime := filepath.Join(ovav, "runtime")
	os.MkdirAll(filepath.Join(ovav, "policy"), 0755)
	os.MkdirAll(filepath.Join(ovav, "service_areas"), 0755)
	os.MkdirAll(runtime, 0755)
	os.WriteFile(filepath.Join(ovav, "policy", "permission_authority.json"), []byte("{}"), 0644)

	e := New(dir)
	if err := e.WriteMarkers(); err != nil {
		t.Fatalf("WriteMarkers() error: %v", err)
	}

	// Governor is active, so governor_active marker should exist
	govMarker := filepath.Join(runtime, "governor_active")
	if _, err := os.Stat(govMarker); err != nil {
		t.Error("governor_active marker should exist when governor is active")
	}

	// Verify integrity_status.json was written
	intFile := filepath.Join(runtime, "integrity_status.json")
	if _, err := os.Stat(intFile); err != nil {
		t.Error("integrity_status.json should exist after WriteMarkers")
	}
}

// ── WriteMarkers: degraded governor (no policy/service_areas) ──────────────────

func TestWriteMarkers_DegradedGovernor(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	if err := e.WriteMarkers(); err != nil {
		t.Fatalf("WriteMarkers() error: %v", err)
	}

	// WriteMarkers creates .ovav/runtime/, so governor is degraded (Active=true)
	// The governor_active marker should exist
	govMarker := filepath.Join(dir, ".ovav", "runtime", "governor_active")
	if _, err := os.Stat(govMarker); err != nil {
		t.Error("governor_active marker should exist when governor is degraded/active")
	}

	// Verify the main status JSON was written
	statusFile := filepath.Join(dir, ".ovav", "runtime", "ovav_status.json")
	if _, err := os.Stat(statusFile); err != nil {
		t.Errorf("status file not created: %v", err)
	}
}

// ── Aggregate: full flow with all subsystems ──────────────────────────────────

func TestAggregate_AllSubsystems(t *testing.T) {
	dir := t.TempDir()
	ovav := filepath.Join(dir, ".ovav")
	runtime := filepath.Join(ovav, "runtime")
	os.MkdirAll(filepath.Join(ovav, "policy"), 0755)
	os.MkdirAll(filepath.Join(ovav, "service_areas"), 0755)
	os.MkdirAll(runtime, 0755)
	os.MkdirAll(filepath.Join(ovav, "memory"), 0755)
	os.WriteFile(filepath.Join(ovav, "policy", "permission_authority.json"), []byte("{}"), 0644)

	// Memory active via gov marker
	os.WriteFile(filepath.Join(runtime, "memory_governor_active"), []byte("1"), 0644)

	// Token stats
	tokData := `{"total_all":200,"total_input":120,"total_output":80,"measurements":3,"current_usage_pct":15.0,"bpe_verified":false}`
	os.WriteFile(filepath.Join(runtime, "session_token_stats.json"), []byte(tokData), 0644)

	// Git branch
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	// Critical governance files for integrity check
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.0"), 0644)
	os.WriteFile(filepath.Join(ovav, "policy", "permission_authority.json"), []byte("{}"), 0644)
	os.MkdirAll(filepath.Join(ovav, "laws"), 0755)
	os.WriteFile(filepath.Join(ovav, "laws", "area_boundary_enforcement.yaml"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(ovav, "plan"), 0755)
	os.WriteFile(filepath.Join(ovav, "plan", "caps.yaml"), []byte(""), 0644)

	e := New(dir)
	p := e.Aggregate()

	if p.OVAV.Governor.Status != "active" {
		t.Errorf("Governor.Status = %q", p.OVAV.Governor.Status)
	}
	if p.OVAV.Memory.Status != "active" {
		t.Errorf("Memory.Status = %q", p.OVAV.Memory.Status)
	}
	if p.OVAV.Integrity.Status != "pass" {
		t.Errorf("Integrity.Status = %q", p.OVAV.Integrity.Status)
	}
	if p.OVAV.Branch.Branch != "main" {
		t.Errorf("Branch.Branch = %q", p.OVAV.Branch.Branch)
	}
	if p.OVAV.Tokens.TotalAll != 200 {
		t.Errorf("Tokens.TotalAll = %d", p.OVAV.Tokens.TotalAll)
	}
	if p.OVAV.Capsule.Active {
		t.Error("Capsule should always be inactive")
	}
}
