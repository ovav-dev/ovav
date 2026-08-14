package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BashReadlineBindings validates that the OVAV bash readline config
// (inputrc) is present, syntactically valid, and provides the conveniences
// bash 5.x doesn't ship by default.
//
// IMPORTANT (2026-08-14): the validator no longer requires shift+arrow
// bindings in inputrc. CEO confirmed shift+arrow should be handled by the
// TERMINAL (visual selection in IT), not by bash readline (set-mark + motion).
// The inputrc now leaves shift+arrow unbound so IT can intercept it natively.
type BashReadlineBindings struct{}

func NewBashReadlineBindings() *BashReadlineBindings { return &BashReadlineBindings{} }

func (b *BashReadlineBindings) ID() string   { return "bash_readline_bindings" }
func (b *BashReadlineBindings) Name() string { return "Bash Readline Bindings Validator" }
func (b *BashReadlineBindings) Description() string {
	return "Validates OVAV inputrc is present and has recommended bash conveniences"
}
func (b *BashReadlineBindings) Weight() int { return 6 }

// inputrcRelPath is the canonical path to the OVAV inputrc.
const inputrcRelPath = "workstation/configs/inputrc/ovav.inputrc"

// recommendedTokens are conveniences bash 5.x doesn't ship by default
// but improve the interactive shell experience. Missing any of these
// produces a WARN (not FAIL) — they're recommendations, not hard
// requirements.
var recommendedTokens = []string{
	"bell-style none",                // suppress WSL→Windows error sound
	"enable-bracketed-paste",         // prevent pasted text from being interpreted
	"\\e[1;5C",                        // ctrl+right word forward (bash 5.x has this; explicit for documentation)
	"\\e[1;5D",                        // ctrl+left word backward
}

// shiftArrowExplicitlyUnbound is a special marker we check for in inputrc.
// IT must NOT see shift+arrow bound to bash — otherwise IT's visual selection
// is overridden by bash's silent set-mark.
var shiftArrowExplicitlyUnboundMarker = "# Shift+arrow: deliberately UNBOUND"

func (b *BashReadlineBindings) Validate(_ context.Context, root string) Result {
	start := time.Now()

	path := filepath.Join(root, inputrcRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{
			ID:       b.ID(),
			Name:     b.Name(),
			Status:   "fail",
			Weight:   b.Weight(),
			Message:  fmt.Sprintf("FAIL — cannot read inputrc: %v (path: %s)", err, inputrcRelPath),
			Duration: time.Since(start),
		}
	}

	content := string(data)
	var warnings []string

	// Check recommended tokens
	for _, token := range recommendedTokens {
		if !strings.Contains(content, token) {
			warnings = append(warnings, fmt.Sprintf("MISSING_RECOMMENDED: %s", token))
		}
	}

	// Check that shift+arrow is explicitly unbound (so IT can intercept)
	if !strings.Contains(content, shiftArrowExplicitlyUnboundMarker) {
		warnings = append(warnings,
			fmt.Sprintf("MISSING_MARKER: %q — comment explaining why shift+arrow is unbound",
				shiftArrowExplicitlyUnboundMarker))
	}

	sort.Strings(warnings)

	status := "pass"
	msg := fmt.Sprintf("PASS — inputrc present, %d recommended binding(s) verified",
		len(recommendedTokens)-len(warnings)+len(warnings))
	if len(warnings) > 0 {
		// Check if the inputrc has at least the marker (otherwise warn, not fail)
		if strings.Contains(content, shiftArrowExplicitlyUnboundMarker) {
			status = "warn"
			msg = fmt.Sprintf("WARN — inputrc has %d recommended-binding gap(s)",
				len(warnings))
		} else {
			// Missing the marker AND likely has no recommended bindings
			// → this might be an old inputrc without the new architecture
			status = "warn"
			msg = fmt.Sprintf("WARN — inputrc missing 'shift+arrow unbound' marker (%d gaps)",
				len(warnings))
		}
	}

	return Result{
		ID:       b.ID(),
		Name:     b.Name(),
		Status:   status,
		Weight:   b.Weight(),
		Message:  msg,
		Issues:   warnings,
		Duration: time.Since(start),
	}
}

var _ Validator = (*BashReadlineBindings)(nil)
