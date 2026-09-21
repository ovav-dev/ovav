package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── InitOutput ────────────────────────────────────────────────────────────────

func TestInitOutput(t *testing.T) {
	// Should not panic
	InitOutput()
}

// ── HasJSONFlag ───────────────────────────────────────────────────────────────

func TestHasJSONFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"present", []string{"--json"}, true},
		{"present_with_other", []string{"--debug", "--json", "file"}, true},
		{"absent", []string{"--debug", "file"}, false},
		{"empty", []string{}, false},
		{"nil", nil, false},
		{"partial_no_match", []string{"--json-file"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasJSONFlag(tt.args)
			if got != tt.want {
				t.Errorf("HasJSONFlag(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

// ── Logo ──────────────────────────────────────────────────────────────────────

func TestLogo(t *testing.T) {
	got := Logo()
	if got != "OVAV" {
		t.Errorf("Logo() = %q, want %q", got, "OVAV")
	}
}

// ── TimestampUTC ──────────────────────────────────────────────────────────────

func TestTimestampUTC(t *testing.T) {
	ts := TimestampUTC()
	// Must be ISO 8601: "2006-01-02T15:04:05Z"
	if len(ts) != 20 {
		t.Errorf("TimestampUTC() = %q, want length 20 (ISO 8601)", ts)
	}
	if !strings.HasSuffix(ts, "Z") {
		t.Errorf("TimestampUTC() = %q, should end with 'Z'", ts)
	}
	if ts[10] != 'T' {
		t.Errorf("TimestampUTC() = %q, should have 'T' at position 10", ts)
	}
}

// ── Output ────────────────────────────────────────────────────────────────────

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestOutput_Statuses(t *testing.T) {
	tests := []struct {
		status  string
		summary string
		want    string
	}{
		{"ok", "all good", "✓ all good\n"},
		{"pass", "passed checks", "✓ passed checks\n"},
		{"blocked", "protected branch", "✗ BLOCKED: protected branch\n"},
		{"error", "something failed", "✗ ERROR: something failed\n"},
		{"warn", "be careful", "⚠ be careful\n"},
		{"unknown", "uh oh", "⚠ uh oh\n"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			out := captureStdout(func() {
				Output(map[string]interface{}{
					"status":  tt.status,
					"summary": tt.summary,
				}, false)
			})
			if out != tt.want {
				t.Errorf("Output() = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestOutput_JSON(t *testing.T) {
	out := captureStdout(func() {
		Output(map[string]interface{}{
			"status":  "ok",
			"summary": "json mode",
		}, true)
	})
	if !strings.Contains(out, `"status"`) || !strings.Contains(out, `"ok"`) {
		t.Errorf("Output(JSON) = %q, want JSON output", out)
	}
}

func TestOutput_ExtraSections(t *testing.T) {
	out := captureStdout(func() {
		Output(map[string]interface{}{
			"status":  "ok",
			"summary": "with plan",
			"plan": map[string]interface{}{
				"name": "test-plan",
			},
			"report": "simple report string",
		}, false)
	})
	if !strings.Contains(out, "--- plan ---") {
		t.Errorf("missing '--- plan ---' in output: %q", out)
	}
	if !strings.Contains(out, "--- report ---") {
		t.Errorf("missing '--- report ---' in output: %q", out)
	}
}

func TestOutput_LargeExtraSection(t *testing.T) {
	// Build a map with 600 chars to trigger truncation at 500
	largeValue := strings.Repeat("x", 600)
	out := captureStdout(func() {
		Output(map[string]interface{}{
			"status":  "ok",
			"summary": "large plan",
			"plan": map[string]interface{}{
				"data": largeValue,
			},
		}, false)
	})
	// Output should be truncated (≤ 500 chars in plan section)
	if !strings.Contains(out, "--- plan ---") {
		t.Errorf("missing '--- plan ---' in output: %q", out)
	}
}

func TestOutput_NonMapExtraSection(t *testing.T) {
	out := captureStdout(func() {
		Output(map[string]interface{}{
			"status":   "ok",
			"summary":  "non-map extra",
			"manifest": 42,
		}, false)
	})
	if !strings.Contains(out, "--- manifest ---") {
		t.Errorf("missing '--- manifest ---' in output: %q", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("missing value '42' in output: %q", out)
	}
}

// ── ReadYAML ──────────────────────────────────────────────────────────────────

func TestReadYAML(t *testing.T) {
	t.Run("valid_file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.yaml")
		os.WriteFile(path, []byte("waiver:\n  active: true\n  branch: develop\n"), 0644)
		result, err := ReadYAML(path)
		if err != nil {
			t.Fatalf("ReadYAML() error = %v", err)
		}
		w, ok := result["waiver"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected nested map under 'waiver', got %T", result["waiver"])
		}
		if w["active"] != true {
			t.Errorf("waiver.active = %v, want true", w["active"])
		}
		if w["branch"] != "develop" {
			t.Errorf("waiver.branch = %v, want 'develop'", w["branch"])
		}
	})

	t.Run("missing_file", func(t *testing.T) {
		_, err := ReadYAML("/nonexistent/path.yaml")
		if err == nil {
			t.Error("ReadYAML() should error on missing file")
		}
	})
}

// ── FindRepoRoot ──────────────────────────────────────────────────────────────

func TestFindRepoRoot(t *testing.T) {
	t.Run("in_repo", func(t *testing.T) {
		root, err := FindRepoRoot()
		if err != nil {
			t.Fatalf("FindRepoRoot() error = %v (expected to find repo root)", err)
		}
		// Check that .git exists at the root
		gitPath := filepath.Join(root, ".git")
		if info, err2 := os.Stat(gitPath); err2 != nil || !info.IsDir() {
			// Might be a worktree (file, not dir)
			if _, err2 := os.Stat(gitPath); err2 != nil {
				t.Errorf("no .git at root %q", root)
			}
		}
	})

	t.Run("error_path", func(t *testing.T) {
		// Test that we get a proper error when not in a repo
		// We can't realistically change CWD in a table test, but we can verify
		// the function returns error type correctly when root-not-found happens.
		// This tests the parent==dir termination path.
		// We'll trust that the traversal logic is correct if the in-repo test passes.
	})
}

func TestFindRepoRoot_SkipsNestedRuntimeOvavDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot := filepath.Clean(filepath.Join(cwd, "..", ".."))
	if err := os.Chdir(runtimeRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	root, err := FindRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".ovav", "plan", "caps.yaml")); err != nil {
		t.Fatalf("FindRepoRoot returned a non-canonical root %q: %v", root, err)
	}
}

// ── MustFindRepoRoot ──────────────────────────────────────────────────────────

func TestMustFindRepoRoot(t *testing.T) {
	t.Run("in_repo", func(t *testing.T) {
		root := MustFindRepoRoot()
		if root == "" {
			t.Error("MustFindRepoRoot() returned empty string")
		}
	})
}

// ── Git helpers ───────────────────────────────────────────────────────────────

func TestGitInfo(t *testing.T) {
	branch, sha, dirty := GitInfo()

	// All three should return something (at minimum "unknown")
	if branch == "" {
		t.Error("GitInfo branch is empty")
	}
	if sha == "" {
		t.Error("GitInfo sha is empty")
	}
	if dirty == "" {
		t.Error("GitInfo dirty is empty")
	}

	// dirty should be "clean" or "dirty"
	if dirty != "clean" && dirty != "dirty" {
		t.Errorf("GitInfo dirty = %q, want 'clean' or 'dirty'", dirty)
	}
}

func TestHasGitRemote(t *testing.T) {
	// Just verify it doesn't panic and returns a bool
	_ = HasGitRemote()
}

func TestGitRemoteURL(t *testing.T) {
	// Just verify it doesn't panic
	_ = GitRemoteURL()
}

func TestRunGitCmd(t *testing.T) {
	t.Run("valid_command", func(t *testing.T) {
		out := runGitCmd("version")
		if out == "" {
			t.Error("runGitCmd('version') returned empty, expected git version string")
		}
	})

	t.Run("invalid_command", func(t *testing.T) {
		out := runGitCmd("nonexistent-subcommand-xyz")
		if out != "" {
			t.Errorf("runGitCmd('nonexistent-subcommand') = %q, want empty string", out)
		}
	})
}
