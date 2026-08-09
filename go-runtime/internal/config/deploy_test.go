package config

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout during fn execution and returns captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

func TestShow_OutputsConfigInfo(t *testing.T) {
	out := captureStdout(t, func() {
		code := Show([]string{})
		if code != 0 {
			t.Errorf("Show() returned %d, want 0", code)
		}
	})

	if !strings.Contains(out, "OVAV Config") {
		t.Errorf("expected output to contain 'OVAV Config', got:\n%s", out)
	}
	if !strings.Contains(out, "Go version:") {
		t.Errorf("expected output to contain 'Go version:', got:\n%s", out)
	}
	if !strings.Contains(out, "OS/Arch:") {
		t.Errorf("expected output to contain 'OS/Arch:', got:\n%s", out)
	}
	if !strings.Contains(out, "===================================================") {
		t.Errorf("expected separator line with '=' repeated, got:\n%s", out)
	}
}

func TestShow_WithJSONFlag(t *testing.T) {
	out := captureStdout(t, func() {
		code := Show([]string{"--json"})
		if code != 0 {
			t.Errorf("Show(--json) returned %d, want 0", code)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", result["status"])
	}
	if result["command"] != "config" {
		t.Errorf("expected command=config, got %v", result["command"])
	}
	if result["go_runtime"] != true {
		t.Errorf("expected go_runtime=true, got %v", result["go_runtime"])
	}
	if _, ok := result["go_version"]; !ok {
		t.Error("expected go_version key in JSON output")
	}
	if _, ok := result["os"]; !ok {
		t.Error("expected os key in JSON output")
	}
	if _, ok := result["arch"]; !ok {
		t.Error("expected arch key in JSON output")
	}
}

func TestShow_DevModeConfigSources(t *testing.T) {
	oldVal := os.Getenv("OVAV_DEV")
	os.Setenv("OVAV_DEV", "1")
	defer func() {
		if oldVal == "" {
			os.Unsetenv("OVAV_DEV")
		} else {
			os.Setenv("OVAV_DEV", oldVal)
		}
	}()

	out := captureStdout(t, func() {
		code := Show([]string{})
		if code != 0 {
			t.Errorf("Show() returned %d, want 0", code)
		}
	})

	if !strings.Contains(out, "Configuration sources") {
		t.Errorf("expected 'Configuration sources' in OVAV_DEV=1 mode, got:\n%s", out)
	}
	if !strings.Contains(out, "Authority contract") {
		t.Errorf("expected 'Authority contract' status in OVAV_DEV=1 mode, got:\n%s", out)
	}
	if !strings.Contains(out, "Permission authority") {
		t.Errorf("expected 'Permission authority' status in OVAV_DEV=1 mode, got:\n%s", out)
	}
}

func TestShow_WithArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantJSON bool
	}{
		{"no args", []string{}, 0, false},
		{"json flag", []string{"--json"}, 0, true},
		{"irrelevant args", []string{"foo", "bar"}, 0, false},
		{"json with extra", []string{"--json", "extra"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				code := Show(tt.args)
				if code != tt.wantCode {
					t.Errorf("Show(%v) returned %d, want %d", tt.args, code, tt.wantCode)
				}
			})

			if len(out) == 0 {
				t.Error("expected non-empty output")
			}

			if tt.wantJSON {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); err != nil {
					t.Errorf("expected valid JSON for args %v, got error: %v", tt.args, err)
				}
			} else {
				if !strings.Contains(out, "OVAV Config") {
					t.Errorf("expected human-readable output for args %v", tt.args)
				}
			}
		})
	}
}
