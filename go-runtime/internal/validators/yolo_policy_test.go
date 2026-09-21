package validators

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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

func TestHostConfigDriftAgentProjections(t *testing.T) {
	tests := []struct {
		name          string
		plant         func(*testing.T, string, string, string)
		fault         func(*testing.T, string, string) func(string)
		requiresLinux bool
		wantStatus    string
	}{
		{
			name: "normal regular copy accepted",
			plant: func(t *testing.T, host, _, _ string) {
				if err := os.MkdirAll(filepath.Dir(host), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(host, []byte("canonical\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: "pass",
		},
		{
			name: "canonical directory symlink accepted",
			plant: func(t *testing.T, host, runtimeDir, _ string) {
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(runtimeDir, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			requiresLinux: true,
			wantStatus:    "pass",
		},
		{
			name: "symlink identity replacement rejected",
			plant: func(t *testing.T, host, runtimeDir, _ string) {
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(runtimeDir, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			fault: func(t *testing.T, host, runtimeDir string) func(string) {
				called := false
				t.Cleanup(func() {
					if !called {
						t.Error("symlink replacement fault seam was not reached")
					}
				})
				return func(stage string) {
					if stage != "before_projection_recheck" {
						return
					}
					called = true
					if err := os.Remove(filepath.Dir(host)); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(runtimeDir, filepath.Dir(host)); err != nil {
						t.Fatal(err)
					}
				}
			},
			requiresLinux: true,
			wantStatus:    "fail",
		},
		{
			name: "directory identity replacement rejected",
			plant: func(t *testing.T, host, runtimeDir, _ string) {
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(runtimeDir, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			fault: func(t *testing.T, host, runtimeDir string) func(string) {
				called := false
				t.Cleanup(func() {
					if !called {
						t.Error("directory replacement fault seam was not reached")
					}
				})
				return func(stage string) {
					if stage != "before_projection_recheck" {
						return
					}
					called = true
					if err := os.Rename(runtimeDir, runtimeDir+".replaced"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(runtimeDir, 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(runtimeDir, filepath.Base(host)), []byte("canonical\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			},
			requiresLinux: true,
			wantStatus:    "fail",
		},
		{
			name: "outside directory symlink rejected",
			plant: func(t *testing.T, host, _, outsideDir string) {
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(outsideDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(outsideDir, filepath.Base(host)), []byte("canonical\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outsideDir, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: "fail",
		},
		{
			name: "nested directory symlink rejected",
			plant: func(t *testing.T, host, runtimeDir, outsideDir string) {
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				alias := filepath.Join(outsideDir, "agents-alias")
				if err := os.Symlink(runtimeDir, alias); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(alias, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: "fail",
		},
		{
			name: "direct file symlink rejected",
			plant: func(t *testing.T, host, runtimeDir, _ string) {
				if err := os.MkdirAll(filepath.Dir(host), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(runtimeDir, filepath.Base(host)), host); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: "fail",
		},
		{
			name: "non regular target rejected",
			plant: func(t *testing.T, host, runtimeDir, _ string) {
				target := filepath.Join(runtimeDir, filepath.Base(host))
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(target, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(runtimeDir, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: "fail",
		},
		{
			name: "mismatched canonical content rejected",
			plant: func(t *testing.T, host, runtimeDir, _ string) {
				if err := os.WriteFile(filepath.Join(runtimeDir, filepath.Base(host)), []byte("mismatch\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(filepath.Dir(host)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(runtimeDir, filepath.Dir(host)); err != nil {
					t.Fatal(err)
				}
			},
			wantStatus: "fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.requiresLinux && runtime.GOOS != "linux" {
				t.Skip("descriptor-anchored directory projections are Linux-only")
			}
			root := t.TempDir()
			mainRoot := t.TempDir()
			home := t.TempDir()
			t.Setenv("HOME", home)
			canonical := filepath.Join(root, ".opencode", "agents", "area-platform-engineering.md")
			host := filepath.Join(home, ".config", "opencode", "agents", "area-platform-engineering.md")
			runtimeDir := filepath.Join(mainRoot, "go-runtime", "internal", "runtimes", "opencode", "agents")
			worktreeGitDir := filepath.Join(mainRoot, ".git", "worktrees", "test-worktree")
			for _, dir := range []string{filepath.Dir(canonical), runtimeDir, worktreeGitDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(canonical, []byte("canonical\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(runtimeDir, filepath.Base(host)), []byte("canonical\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: "+worktreeGitDir+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			tt.plant(t, host, runtimeDir, t.TempDir())

			validator := NewHostConfigDrift()
			if tt.fault != nil {
				validator.projectionFault = tt.fault(t, host, runtimeDir)
			}
			result := validator.Validate(context.Background(), root)
			if result.Status != tt.wantStatus {
				t.Fatalf("status=%s, want %s: %s: %v", result.Status, tt.wantStatus, result.Message, result.Issues)
			}
		})
	}
}

func TestMainRepoRootNoGit(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, string) string
		wantErr bool
	}{
		{
			name: "normal git directory",
			setup: func(t *testing.T, root string) string {
				if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
					t.Fatal(err)
				}
				return root
			},
		},
		{
			name: "worktree git file",
			setup: func(t *testing.T, root string) string {
				mainRoot := t.TempDir()
				gitDir := filepath.Join(mainRoot, ".git", "worktrees", "test-worktree")
				if err := os.MkdirAll(gitDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: "+gitDir+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return mainRoot
			},
		},
		{
			name: "malformed gitdir",
			setup: func(t *testing.T, root string) string {
				if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: malformed\nextra\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			want := tt.setup(t, root)
			got, err := mainRepoRootNoGit(root)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("mainRepoRootNoGit()=%q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("mainRepoRootNoGit()=%q, want %q", got, want)
			}
		})
	}
}
