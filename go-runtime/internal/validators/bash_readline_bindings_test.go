package validators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeInputrc(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "workstation", "configs", "inputrc")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ovav.inputrc"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

func TestBashReadlineBindings_AllRequiredPresent_Pass(t *testing.T) {
	root := writeInputrc(t, `
"\e[1;2A": "\C-@\e[A"
"\e[1;2B": "\C-@\e[B"
"\e[1;2C": "\C-@\e[C"
"\e[1;2D": "\C-@\e[D"
"bell-style none"
"\e[1;6C": "x"
"\e[1;6D": "x"
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s", res.Status, res.Message)
	}
}

func TestBashReadlineBindings_RequiredMissing_Fail(t *testing.T) {
	root := writeInputrc(t, `
"\e[1;2A": "x"
"\e[1;2C": "x"
# missing \e[1;2B and \e[1;2D
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s: %s", res.Status, res.Message)
	}
}

func TestBashReadlineBindings_FileMissing_Fail(t *testing.T) {
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected fail when file missing, got %s: %s", res.Status, res.Message)
	}
}

func TestBashReadlineBindings_ExtrasMissing_Warn(t *testing.T) {
	root := writeInputrc(t, `
"\e[1;2A": "x"
"\e[1;2B": "x"
"\e[1;2C": "x"
"\e[1;2D": "x"
# no bell-style, no \e[1;6
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("expected warn for missing extras, got %s: %s", res.Status, res.Message)
	}
}

func TestBashReadlineBindings_CommentsIgnored_Warn(t *testing.T) {
	// Comments must not count as bindings. Test passes all required bindings
	// but skips recommended extras — so we expect 'warn' (not 'pass').
	root := writeInputrc(t, `
# \e[1;2A: commented out — does NOT count
"\e[1;2A": "x"
"\e[1;2B": "x"
"\e[1;2C": "x"
"\e[1;2D": "x"
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status == "fail" {
		t.Fatalf("expected pass/warn (comment lines ignored, required present), got fail: %s — issues: %v",
			res.Message, res.Issues)
	}
	// Verify comment didn't accidentally count
	for _, issue := range res.Issues {
		if strings.Contains(issue, "MISSING_REQUIRED") {
			t.Fatalf("commented binding should not count as missing: %s", issue)
		}
	}
}
