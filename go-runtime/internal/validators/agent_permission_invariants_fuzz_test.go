package validators

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Property-based fuzzing for AgentPermissionInvariants ───────────────────

func TestAgentPermission_FuzzedInputs(t *testing.T) {
	// Property tests: the validator should handle arbitrary YAML frontmatter
	// without panicking, and should return consistent results.
	testCases := []struct {
		name       string
		thavrenFM  string
		areaFM     string
		wantPanic  bool
		wantStatus string // "pass", "fail"
	}{
		{
			name: "empty frontmatter",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			wantStatus: "pass",
		},
		{
			name: "missing permission block",
			thavrenFM: `---
name: Thavren
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
---`,
			wantStatus: "fail",
		},
		{
			name:       "completely empty YAML",
			thavrenFM:  "",
			areaFM:     "",
			wantPanic:  false, // parse error → fail, not panic
			wantStatus: "fail",
		},
		{
			name: "malformed YAML — no closing",
			thavrenFM: `---
name: Thavren
permission:` + "\n",
			areaFM: `---
name: Platform Engineering
---`,
			wantStatus: "fail",
		},
		{
			name: "external_directory wildcard wrong value",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---`,
			wantStatus: "fail",
		},
		{
			name: "area edit not allow",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: deny
  bash: {}
  external_directory:
    "*": deny
---`,
			wantStatus: "fail",
		},
		{
			name: "bash permissions mismatch",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash:
    sudo: deny
  external_directory:
    "*": allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash:
    sudo: allow
  external_directory:
    "*": deny
---`,
			wantStatus: "fail",
		},
		{
			name: "area no explicit allow",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
---`,
			wantStatus: "fail",
		},
		{
			name: "scope injection in name field",
			thavrenFM: `---
name: Thavren<script>malicious</script>
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			// Name field with HTML/script — should not panic
			wantStatus: "fail", // wrong name → fail
		},
		{
			name: "fabricated permission key",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
  fabricated_key: allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			// fabricated_key is not in requiredPermissionKeys → fail
			wantStatus: "fail",
		},
		{
			name: "empty string name",
			thavrenFM: `---
name: ""
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			wantStatus: "fail", // name mismatch
		},
		{
			name: "boolean instead of string in permission.edit",
			thavrenFM: `---
name: Thavren
permission:
  edit: true
  bash: {}
  external_directory:
    "*": allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			wantStatus: "fail", // edit is not "allow" string
		},
		{
			name: "deeply nested injection attempt in external_directory",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
    /etc/passwd: allow
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			wantStatus: "pass", // area passes — thavren path allows dangerous paths but that's a policy choice
		},
		{
			name: "null bytes in YAML",
			thavrenFM: "---" + "\n" +
				"name: Thavren\x00Extra" + "\n" +
				"permission:\n" +
				"  edit: allow\n" +
				"  bash: {}\n" +
				"  external_directory:\n" +
				"    \"*\": allow\n" +
				"---",
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			wantStatus: "fail", // name won't match
		},
		{
			name: "YAML list instead of map for external_directory",
			thavrenFM: `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    - item1
    - item2
---`,
			areaFM: `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`,
			wantStatus: "fail", // external_directory is not a map
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()

			// Write thavren file
			thavrenPath := filepath.Join(tmp, "lead-thavren.md")
			if tc.thavrenFM != "" {
				content := tc.thavrenFM + "\n# Body\nSome content here."
				if err := os.WriteFile(thavrenPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Write area file
			areaPath := filepath.Join(tmp, "area-platform-engineering.md")
			if tc.areaFM != "" {
				content := tc.areaFM + "\n# Body\nSome content here."
				if err := os.WriteFile(areaPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			v := NewAgentPermissionInvariants()

			// Run in a function to catch panics
			var result Result
			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()
				// Create subdirs so paths resolve
				clientsDir := filepath.Join(tmp, "clients", "opencode", "agents")
				os.MkdirAll(clientsDir, 0755)
				// Move files
				os.Rename(thavrenPath, filepath.Join(clientsDir, "lead-thavren.md"))
				os.Rename(areaPath, filepath.Join(clientsDir, "area-platform-engineering.md"))

				result = v.Validate(context.Background(), tmp)
			}()

			if tc.wantPanic && !panicked {
				t.Errorf("expected panic but none occurred")
			}
			if !tc.wantPanic && panicked {
				t.Errorf("unexpected panic")
			}
			if !tc.wantPanic && result.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", result.Status, tc.wantStatus)
			}
		})
	}
}

// ── Edge case: concurrent file modifications ──────────────────────────────

func TestAgentPermission_FileDisappearsDuringParse(t *testing.T) {
	tmp := t.TempDir()
	v := NewAgentPermissionInvariants()

	// Create files
	clientsDir := filepath.Join(tmp, "clients", "opencode", "agents")
	os.MkdirAll(clientsDir, 0755)

	thavrenPath := filepath.Join(clientsDir, "lead-thavren.md")
	areaPath := filepath.Join(clientsDir, "area-platform-engineering.md")

	validFM := `---
name: Thavren
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---`

	os.WriteFile(thavrenPath, []byte(validFM+"\n# Body"), 0644)
	os.WriteFile(areaPath, []byte(`---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---`+"\n# Body"), 0644)

	result := v.Validate(context.Background(), tmp)
	if result.Status != "pass" {
		t.Errorf("expected pass with valid files, got %s: %s", result.Status, result.Message)
	}

	// Delete thavren file
	os.Remove(thavrenPath)

	result2 := v.Validate(context.Background(), tmp)
	if result2.Status != "fail" {
		t.Errorf("expected fail after file deletion, got %s", result2.Status)
	}
}

// ── Property: setsEqual and mapsEqual helpers ─────────────────────────────

func TestPermission_SetsEqualEdgeCases(t *testing.T) {
	tests := []struct {
		a, b     map[string]bool
		expected bool
	}{
		{map[string]bool{"a": true, "b": true}, map[string]bool{"a": true, "b": true}, true},
		{map[string]bool{"a": true}, map[string]bool{"a": true, "b": true}, false},
		{map[string]bool{"a": true, "b": true}, map[string]bool{"a": true}, false},
		{map[string]bool{}, map[string]bool{}, true},
		{nil, map[string]bool{}, true}, // nil vs empty
	}
	for _, tt := range tests {
		if got := setsEqual(tt.a, tt.b); got != tt.expected {
			t.Errorf("setsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestPermission_MapsEqualEdgeCases(t *testing.T) {
	tests := []struct {
		a, b     map[string]interface{}
		expected bool
	}{
		{
			map[string]interface{}{"a": "1", "b": "2"},
			map[string]interface{}{"a": "1", "b": "2"},
			true,
		},
		{
			map[string]interface{}{"a": "1"},
			map[string]interface{}{"a": "1", "b": "2"},
			false,
		},
		{
			map[string]interface{}{"a": "1", "b": "2"},
			map[string]interface{}{"a": "1"},
			false,
		},
		{
			map[string]interface{}{"a": "1", "b": "2"},
			map[string]interface{}{"a": "1", "b": "3"},
			false,
		},
		{
			map[string]interface{}{"sudo": "allow"},
			map[string]interface{}{"sudo": "allow"},
			true,
		},
		{
			map[string]interface{}{"sudo": "allow"},
			map[string]interface{}{"sudo": "deny"},
			false,
		},
	}
	for _, tt := range tests {
		if got := mapsEqual(tt.a, tt.b); got != tt.expected {
			t.Errorf("mapsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
		}
	}
}

// ── Property: injection attempts in YAML values ───────────────────────────

func TestAgentPermission_SQLInjectionInName(t *testing.T) {
	tmp := t.TempDir()
	clientsDir := filepath.Join(tmp, "clients", "opencode", "agents")
	os.MkdirAll(clientsDir, 0755)

	injectionNames := []string{
		"Thavren' OR '1'='1",
		"Thavren\"; DROP TABLE permissions;--",
		"Thavren\nadmin: true",     // newline splits YAML key; name="Thavren" → matches → pass (correct)
		"Thavren\r\nadmin: true",   // same as above
		strings.Repeat("A", 10000), // very long name
	}

	for _, name := range injectionNames {
		wantPass := name == "Thavren\nadmin: true" || name == "Thavren\r\nadmin: true"
		t.Run(name[:min(30, len(name))], func(t *testing.T) {
			thavrenContent := `---
name: ` + name + `
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": allow
---` + "\n# Body"

			areaContent := `---
name: Platform Engineering
permission:
  edit: allow
  bash: {}
  external_directory:
    "*": deny
    /home/braka: allow
---` + "\n# Body"

			os.WriteFile(filepath.Join(clientsDir, "lead-thavren.md"), []byte(thavrenContent), 0644)
			os.WriteFile(filepath.Join(clientsDir, "area-platform-engineering.md"), []byte(areaContent), 0644)

			v := NewAgentPermissionInvariants()
			result := v.Validate(context.Background(), tmp)

			// For newline injection: YAML parser reads name as "Thavren" → pass (correct)
			// For other injections: name != "Thavren" → fail (correct)
			gotPass := result.Status == "pass"
			if gotPass != wantPass {
				t.Errorf("injection %q: got pass=%v, want pass=%v", name[:min(20, len(name))], gotPass, wantPass)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
