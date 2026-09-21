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

// Test 1: New architecture (shift+arrow unbound + marker) → PASS
func TestBashReadlineBindings_NewArchitecture_Pass(t *testing.T) {
	root := writeInputrc(t, `
# Shift+arrow: deliberately UNBOUND
"bell-style none"
"\e[1;5C": forward-word
"\e[1;5D": backward-word
enable-bracketed-paste
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s — issues: %v", res.Status, res.Message, res.Issues)
	}
}

// Test 2: New architecture but missing recommended bindings → WARN
func TestBashReadlineBindings_NewArchitecture_MissingRecommended_Warn(t *testing.T) {
	root := writeInputrc(t, `
# Shift+arrow: deliberately UNBOUND
"\e[1;5C": forward-word
"\e[1;5D": backward-word
# missing bell-style, enable-bracketed-paste
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("expected warn for missing recommended, got %s: %s", res.Status, res.Message)
	}
}

// Test 3: Old architecture (shift+arrow bound) → WARN with marker check
// (Doesn't fail, but warns that the marker explaining shift+arrow is unbound is missing)
func TestBashReadlineBindings_OldArchitecture_Warn(t *testing.T) {
	root := writeInputrc(t, `
"\e[1;2A": "\C-@\e[A"
"\e[1;2B": "\C-@\e[B"
"\e[1;2C": "\C-@\e[C"
"\e[1;2D": "\C-@\e[D"
"bell-style none"
"\e[1;5C": forward-word
"\e[1;5D": backward-word
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("expected warn for missing marker (old architecture), got %s: %s",
			res.Status, res.Message)
	}
	// Verify warning mentions the marker
	foundMarker := false
	for _, issue := range res.Issues {
		if strings.Contains(issue, "MISSING_MARKER") {
			foundMarker = true
		}
	}
	if !foundMarker {
		t.Fatalf("expected MISSING_MARKER warning, got issues: %v", res.Issues)
	}
}

// Test 4: File missing → FAIL
func TestBashReadlineBindings_FileMissing_Fail(t *testing.T) {
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), t.TempDir())
	if res.Status != "fail" {
		t.Fatalf("expected fail when file missing, got %s: %s", res.Status, res.Message)
	}
}

// Test 5: Comments document the tokens — validator uses simple text contains,
// so even commented tokens count as present. This is intentional because
// comments are documentary: a comment mentioning "bell-style none" indicates
// the operator intended to include it.
func TestBashReadlineBindings_CommentsCountAsDocumentation(t *testing.T) {
	root := writeInputrc(t, `
# \e[1;5C: documented but not active
# bell-style none: documented
# Shift+arrow: deliberately UNBOUND
"\e[1;5C": forward-word
"\e[1;5D": backward-word
enable-bracketed-paste
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "pass" {
		t.Fatalf("expected pass (commented tokens count as documentation), got %s: %s — issues: %v",
			res.Status, res.Message, res.Issues)
	}
}

// Test 6: Real-fragment regression — section header uses Unicode box-drawing
// chars (─) for aesthetics; the canonical marker line must still be present
// verbatim so the validator can detect it. Mirrors the actual production
// fragment workstation/configs/inputrc/ovav.inputrc.
func TestBashReadlineBindings_RealFragment_Pass(t *testing.T) {
	root := writeInputrc(t, `
# ─── OVAV readline config ────────────────────────────────────
set bell-style none
set enable-bracketed-paste on

"\e[1;5C": forward-word
"\e[1;5D": backward-word

# ── Shift+arrow: deliberately UNBOUND ─────────────────────────
# Shift+arrow: deliberately UNBOUND
# bash readline should NOT bind shift+arrow — that's the terminal's job.
"\e[1;3C": forward-word
"\e[1;3D": backward-word
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "pass" {
		t.Fatalf("real-fragment regression: expected pass, got %s: %s — issues: %v",
			res.Status, res.Message, res.Issues)
	}
}

// Test 7: Real-fragment regression — section header with box-drawing chars
// but WITHOUT the canonical marker line MUST warn. Catches the drift where
// the section header is present but the validator's literal marker is missing.
func TestBashReadlineBindings_RealFragment_MissingMarker_Warn(t *testing.T) {
	root := writeInputrc(t, `
set bell-style none
set enable-bracketed-paste on

"\e[1;5C": forward-word
"\e[1;5D": backward-word

# ── Shift+arrow: deliberately UNBOUND ─────────────────────────
"\e[1;3C": forward-word
"\e[1;3D": backward-word
`)
	v := NewBashReadlineBindings()
	res := v.Validate(t.Context(), root)
	if res.Status != "warn" {
		t.Fatalf("real-fragment missing marker: expected warn, got %s: %s",
			res.Status, res.Message)
	}
}
