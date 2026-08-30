package validators

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const validYOLOPolicy = `{
  "schema_version":"ovav.permission_authority.v3",
  "authority":"OVAV SYSTEMS is canonical. CEO Alexander Salvador governs. OVAV AGENTS operates with total freedom when installed.",
  "_ovav_yolo":{"enabled":true,"applied":"2026-08-19","bash_default":"allow","edit_default":"allow","write_default":"allow","read_default":"allow","external_directory_default":"allow","doom_loop_default":"allow"}
}`

func TestYOLOPolicyRequiresCompleteMarkerAndCanonicalHEAD(t *testing.T) {
	var policy map[string]interface{}
	if err := json.Unmarshal([]byte(validYOLOPolicy), &policy); err != nil {
		t.Fatal(err)
	}
	if !hasCanonicalYOLOMarker(policy) {
		t.Fatal("complete canonical YOLO marker rejected")
	}

	for _, field := range []string{"enabled", "applied", "bash_default", "edit_default", "write_default", "read_default", "external_directory_default", "doom_loop_default"} {
		t.Run("missing_"+field, func(t *testing.T) {
			var candidate map[string]interface{}
			if err := json.Unmarshal([]byte(validYOLOPolicy), &candidate); err != nil {
				t.Fatal(err)
			}
			delete(candidate["_ovav_yolo"].(map[string]interface{}), field)
			if hasCanonicalYOLOMarker(candidate) {
				t.Fatalf("incomplete marker accepted without %s", field)
			}
		})
	}
	t.Run("invalid_applied_date", func(t *testing.T) {
		var candidate map[string]interface{}
		if err := json.Unmarshal([]byte(validYOLOPolicy), &candidate); err != nil {
			t.Fatal(err)
		}
		candidate["_ovav_yolo"].(map[string]interface{})["applied"] = "not-a-date"
		if hasCanonicalYOLOMarker(candidate) {
			t.Fatal("invalid applied date accepted")
		}
	})

	root := t.TempDir()
	policyPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if err := os.MkdirAll(filepath.Dir(policyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, []byte(validYOLOPolicy), 0o600); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, root, "init")
	runGitTest(t, root, "config", "user.email", "test@example.com")
	runGitTest(t, root, "config", "user.name", "Test")
	runGitTest(t, root, "add", ".ovav/policy/permission_authority.json")
	runGitTest(t, root, "commit", "-m", "policy")
	if !isYOLOPolicy(root, policy) {
		t.Fatal("committed canonical YOLO policy rejected")
	}
	if err := os.WriteFile(policyPath, []byte(validYOLOPolicy+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if isYOLOPolicy(root, policy) {
		t.Fatal("uncommitted YOLO policy drift accepted")
	}
}

func TestHostConfigDriftRequiresExactCanonicalConfigProjection(t *testing.T) {
	tests := []struct {
		name       string
		hostConfig string
		wantStatus string
	}{
		{name: "exact", hostConfig: `{"_ovav":true,"permission":{"*":"allow"}}`, wantStatus: "pass"},
		{name: "marker spoof", hostConfig: `{"_ovav":true,"permission":{"*":"allow"},"provider":{"evil":{}}}`, wantStatus: "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			home := t.TempDir()
			t.Setenv("HOME", home)
			canonicalContent := `{"_ovav":true,"permission":{"*":"allow"}}`
			canonical := filepath.Join(root, "opencode.json")
			host := filepath.Join(home, ".config", "opencode", "opencode.json")
			for _, path := range []string{canonical, host} {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(canonical, []byte(canonicalContent), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(host, []byte(tt.hostConfig), 0o600); err != nil {
				t.Fatal(err)
			}

			result := NewHostConfigDrift().Validate(context.Background(), root)
			if result.Status != tt.wantStatus {
				t.Fatalf("status=%s, want %s: %v", result.Status, tt.wantStatus, result.Issues)
			}
		})
	}
}

func TestCanonicalYOLOPolicyValidatorsPass(t *testing.T) {
	root, err := findOVAVRoot()
	if err != nil {
		t.Fatal(err)
	}

	tests := []Validator{
		NewCredentialGovernance(),
		NewSecurityHardening(),
		NewAdvancedHardening(),
	}
	for _, validator := range tests {
		t.Run(validator.ID(), func(t *testing.T) {
			result := validator.Validate(context.Background(), root)
			if result.Status != "pass" {
				t.Fatalf("canonical YOLO policy rejected: %s: %v", result.Message, result.Issues)
			}
		})
	}
}

func TestHostConfigDriftAllowsExactCanonicalAgentProjection(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	content := []byte("canonical OVAV agent projection\n")

	canonical := filepath.Join(root, ".opencode", "agents", "area-platform-engineering.md")
	host := filepath.Join(home, ".config", "opencode", "agents", "area-platform-engineering.md")
	for _, path := range []string{canonical, host} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	result := NewHostConfigDrift().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("exact canonical projection flagged as intrusion: %s: %v", result.Message, result.Issues)
	}
}

func TestHostConfigDriftRejectsChangedAndSymlinkedAgentProjection(t *testing.T) {
	tests := []struct {
		name  string
		plant func(*testing.T, string)
	}{
		{
			name: "changed content",
			plant: func(t *testing.T, host string) {
				if err := os.WriteFile(host, []byte("tampered\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink",
			plant: func(t *testing.T, host string) {
				if err := os.Symlink("missing-target", host); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			home := t.TempDir()
			t.Setenv("HOME", home)
			canonical := filepath.Join(root, ".opencode", "agents", "area-platform-engineering.md")
			host := filepath.Join(home, ".config", "opencode", "agents", "area-platform-engineering.md")
			for _, dir := range []string{filepath.Dir(canonical), filepath.Dir(host)} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(canonical, []byte("canonical\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			tt.plant(t, host)

			result := NewHostConfigDrift().Validate(context.Background(), root)
			if result.Status != "fail" {
				t.Fatalf("unsafe host projection accepted: %s: %v", result.Message, result.Issues)
			}
		})
	}
}
