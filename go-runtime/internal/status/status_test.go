package status

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	e := New("/tmp/fake")
	if e.projectRoot != "/tmp/fake" {
		t.Errorf("projectRoot = %q, want %q", e.projectRoot, "/tmp/fake")
	}
	want := filepath.Join("/tmp/fake", ".ovav", "runtime")
	if e.runtimeDir != want {
		t.Errorf("runtimeDir = %q, want %q", e.runtimeDir, want)
	}
}

func TestAggregate_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	payload := e.Aggregate()

	if payload.OVAV.Overall != "absent" {
		t.Errorf("Overall = %q, want %q (no .ovav dir)", payload.OVAV.Overall, "absent")
	}
	if payload.EngineVersion != "2.0.0-go" {
		t.Errorf("EngineVersion = %q, want %q", payload.EngineVersion, "2.0.0-go")
	}
	if payload.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", payload.ProjectRoot, dir)
	}
	if payload.GeneratedAt == "" {
		t.Error("GeneratedAt should not be empty")
	}
}

func TestAggregate_WithGovernorFiles(t *testing.T) {
	dir := t.TempDir()
	// Create minimal OVAV structure
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "service_areas"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte("{}"), 0644)

	e := New(dir)
	payload := e.Aggregate()

	if payload.OVAV.Governor.Status != "active" {
		t.Errorf("Governor.Status = %q, want %q", payload.OVAV.Governor.Status, "active")
	}
	if !payload.OVAV.Governor.Active {
		t.Error("Governor.Active should be true")
	}
}

func TestComputeOverall(t *testing.T) {
	e := New("/tmp/x")

	tests := []struct {
		name     string
		gStatus  string
		iStatus  string
		wantOver string
		wantIcon string
	}{
		{"absent", "absent", "pass", "absent", "⚫"},
		{"integrity fail", "active", "fail", "degraded", "🔴"},
		{"governor degraded", "degraded", "pass", "degraded", "🟡"},
		{"all good", "active", "pass", "active", "🟢"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := GovernorStatus{Status: tt.gStatus}
			i := IntegrityStatus{Status: tt.iStatus}
			overall, icon := e.computeOverall(g, i)
			if overall != tt.wantOver {
				t.Errorf("overall = %q, want %q", overall, tt.wantOver)
			}
			if icon != tt.wantIcon {
				t.Errorf("icon = %q, want %q", icon, tt.wantIcon)
			}
		})
	}
}

func TestSystemInfo(t *testing.T) {
	info := SystemInfo()
	if info["go_version"] == "" {
		t.Error("go_version should not be empty")
	}
	if info["os"] == "" {
		t.Error("os should not be empty")
	}
	if info["arch"] == "" {
		t.Error("arch should not be empty")
	}
}

func TestWriteMarkers(t *testing.T) {
	dir := t.TempDir()
	e := New(dir)
	if err := e.WriteMarkers(); err != nil {
		t.Fatalf("WriteMarkers() error: %v", err)
	}

	statusFile := filepath.Join(dir, ".ovav", "runtime", "ovav_status.json")
	if _, err := os.Stat(statusFile); err != nil {
		t.Errorf("status file not created: %v", err)
	}
}
