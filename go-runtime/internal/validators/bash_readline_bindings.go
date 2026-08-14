package validators

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BashReadlineBindings validates that the OVAV bash readline config
// (inputrc) provides shift+arrow bindings. bash 5.x readline ships
// with NO default bindings for shift+arrow (modifier 2), causing
// "shift+arrow → DABC + Windows beep" regression in WSL.
type BashReadlineBindings struct{}

func NewBashReadlineBindings() *BashReadlineBindings { return &BashReadlineBindings{} }

func (b *BashReadlineBindings) ID() string   { return "bash_readline_bindings" }
func (b *BashReadlineBindings) Name() string { return "Bash Readline Bindings Validator" }
func (b *BashReadlineBindings) Description() string {
	return "Validates OVAV inputrc provides shift+arrow (modifier 2) bindings"
}
func (b *BashReadlineBindings) Weight() int { return 6 }

// inputrcRelPath is the canonical path to the OVAV inputrc.
const inputrcRelPath = "workstation/configs/inputrc/ovav.inputrc"

// requiredShiftArrowBindings are the CSI sequences that bash 5.x doesn't
// ship by default but the OVAV inputrc must provide to fix the
// "shift+arrow → DABC" regression.
//
// Format: ESC [ 1 ; 2 X   (X = A/B/C/D for up/down/right/left)
//                ↑
//                modifier 2 = shift
var requiredShiftArrowBindings = []string{
	`"\e[1;2A"`, // shift+up
	`"\e[1;2B"`, // shift+down
	`"\e[1;2C"`, // shift+right
	`"\e[1;2D"`, // shift+left
}

// recommendedExtras are not strictly required but highly recommended
// to round out the readline experience. Reported as warnings.
var recommendedExtras = []string{
	`"\e[1;6C"`, // shift+ctrl+right (select word forward)
	`"\e[1;6D"`, // shift+ctrl+left (select word backward)
	"bell-style none", // suppress audible bell (WSL surfaces as Windows error sound)
}

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

	// Parse inputrc line by line. Readline parser is line-oriented and
	// uses `#` for comments (only at line start, not mid-line for values).
	// We do a simple text scan to find required bindings.
	var issues []string
	var warnings []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip blank lines and pure comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, key := range requiredShiftArrowBindings {
			if strings.HasPrefix(line, key) {
				seen[key] = true
			}
		}
		for _, substr := range recommendedExtras {
			if strings.Contains(line, substr) {
				seen[substr] = true
			}
		}
	}

	// Check required bindings
	for _, key := range requiredShiftArrowBindings {
		if !seen[key] {
			issues = append(issues, fmt.Sprintf("MISSING_REQUIRED: %s", key))
		}
	}

	// Check recommended extras (warnings only)
	for _, substr := range recommendedExtras {
		if !seen[substr] {
			warnings = append(warnings, fmt.Sprintf("MISSING_RECOMMENDED: %s", substr))
		}
	}

	sort.Strings(issues)
	sort.Strings(warnings)

	status := "pass"
	var msg string
	switch {
	case len(issues) > 0:
		status = "fail"
		msg = fmt.Sprintf("FAIL — %d missing required shift+arrow binding(s) in %s",
			len(issues), inputrcRelPath)
	case len(warnings) > 0:
		status = "warn"
		msg = fmt.Sprintf("WARN — inputrc has %d recommended-binding gap(s)",
			len(warnings))
	default:
		msg = fmt.Sprintf("PASS — inputrc has all %d required + %d recommended bindings",
			len(requiredShiftArrowBindings), len(recommendedExtras))
	}

	all := append(append([]string(nil), issues...), warnings...)
	return Result{
		ID:       b.ID(),
		Name:     b.Name(),
		Status:   status,
		Weight:   b.Weight(),
		Message:  msg,
		Issues:   all,
		Duration: time.Since(start),
	}
}

var _ Validator = (*BashReadlineBindings)(nil)
