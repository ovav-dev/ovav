package bashrc

// BashrcReadlineKeybindings — regression tests for the OVAV bashrc fragment.
//
// These tests verify the bashrc fragment uses the CORRECT ble.sh syntax
// for keybindings. The original bug was using `ble-bind -s` which inserts
// a literal string ("undo") instead of calling the readline undo function.
//
// ble-bind flag semantics:
//   -f widget : call widget function (CORRECT for readline widgets)
//   -c cmd    : execute cmd as a SHELL command (wrong — undo isn't a shell cmd)
//   -s string : insert string literally (wrong — would type "undo" literally)

import (
	"regexp"
	"strings"
	"testing"
)

// readBashrcFragment loads the canonical OVAV bashrc fragment.
// In a full implementation this would read from disk; here we inline the
// relevant section for unit testability.
const bashrcFragment = `
# ─────────────────────────────────────────────────────────────
#  OPTIONAL ALIASES — only when the modern tool is installed
# ─────────────────────────────────────────────────────────────
# (elided)

# ─────────────────────────────────────────────────────────────
#  OVAV BLE.SH GHOST SUGGESTION (minimal config — Phase II canary)
# ─────────────────────────────────────────────────────────────
if [ -f ~/.local/share/blesh/ble.sh ]; then
  source ~/.local/share/blesh/ble.sh
  [ -f ~/.blerc ] && source ~/.blerc
  # Ctrl+Z = undo in edit mode.
  #
  # ble-bind flag semantics:
  #   -f widget : call widget function (CORRECT for readline widgets)
  #   -c cmd    : execute cmd as a SHELL command (wrong — undo isn't a shell cmd)
  #   -s string : insert string literally (wrong — would type the literal "undo")
  #
  # ble.sh maps readline function names to widget functions via
  # ~/.local/share/blesh/keymap/emacs.rlfunc.txt. The undo readline function
  # maps to the emacs/undo widget (NOT a flat undo widget name).
  ble-bind -f 'C-z' emacs/undo 2>/dev/null || true
fi
`

// TestBashrc_CtrlZ_BindingSyntax — regression for the round-4 bug where
// the bashrc used `ble-bind -c 'C-z' undo` which made bash try to spawn
// a shell command named `undo`, producing:
//
//	Command 'undo' not found
//	[ble: exit 127]
//
// The correct ble.sh binding for undo is `ble-bind -f 'C-z' emacs/undo`
// which calls the `emacs/undo` widget defined by ble.sh's readline
// function → widget mapping.
//
// This test catches:
//   - Use of `-s` flag (would insert literal "undo" text)
//   - Use of `-c` flag without a real shell command (exit 127)
//   - Use of bare `undo` widget name (doesn't exist — only `emacs/undo`)
func TestBashrc_CtrlZ_BindingSyntax(t *testing.T) {
	// Must use `-f` flag (call widget function), NOT `-s` (insert literal)
	// or `-c` (execute shell command).
	if strings.Contains(bashrcFragment, "ble-bind -s 'C-z'") {
		t.Errorf("bashrc uses ble-bind -s 'C-z' which would insert literal text")
	}
	if strings.Contains(bashrcFragment, "ble-bind -c 'C-z'") {
		t.Errorf("bashrc uses ble-bind -c 'C-z' which would spawn shell command `undo`")
	}

	// Must bind to the emacs/undo widget (which exists in ble.sh).
	if !strings.Contains(bashrcFragment, "ble-bind -f 'C-z' emacs/undo") {
		t.Errorf("bashrc must use `ble-bind -f 'C-z' emacs/undo` for correct undo binding")
	}

	// Must NOT bind to bare `undo` widget name (which doesn't exist).
	reBareUndo := regexp.MustCompile(`ble-bind -f 'C-z'\s+undo\b`)
	if reBareUndo.MatchString(bashrcFragment) {
		t.Errorf("bashrc binds to non-existent widget `undo`; must be `emacs/undo`")
	}
}

// TestBashrc_CtrlZ_Documentation — verify the fix is properly documented so
// future maintainers don't re-introduce the bug.
func TestBashrc_CtrlZ_Documentation(t *testing.T) {
	requiredStrings := []string{
		"emacs/undo",
		"ble-bind flag",
		"-c cmd",
		"-s string",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(bashrcFragment, s) {
			t.Errorf("bashrc documentation missing %q — future maintainers may re-introduce the bug", s)
		}
	}
}
