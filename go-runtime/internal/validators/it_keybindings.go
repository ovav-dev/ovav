package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ITKeybindings validates the OVAV Intelligent Terminal (IT v0.2) keybindings fragment.
// Detects keybindings with null/empty IDs, unresolved action IDs (not built-in and
// not defined in the fragment's actions array), and duplicate key combos.
//
// Replaces: previously no validator existed. The bug fixed on 2026-08-14
// (commit e2860ec) wrote 47 keybindings with 13 having "id": null and 4 having
// wrong action IDs (MovePaneToTab0 for directional movement). Without this
// validator, that regression was only caught when CEO manually tested.
type ITKeybindings struct{}

func NewITKeybindings() *ITKeybindings { return &ITKeybindings{} }

func (k *ITKeybindings) ID() string   { return "it_keybindings" }
func (k *ITKeybindings) Name() string { return "IT Keybindings Validator" }
func (k *ITKeybindings) Description() string {
	return "Validates OVAV IT v0.2 keybindings: null/empty IDs, unresolved actions, duplicates"
}
func (k *ITKeybindings) Weight() int { return 8 }

// itFragmentRelPath is the canonical path to the IT settings fragment.
const itFragmentRelPath = "workstation/configs/intelligent-terminal/settings-fragment.json"

// itBuiltinActions lists all built-in IT v0.2 action IDs. Any keybinding id
// not in this set must be defined in the fragment's "actions" array, otherwise
// IT will silently drop the keybinding on first parse.
var itBuiltinActions = map[string]bool{
	// Clipboard
	"Terminal.CopyToClipboard":            true,
	"Terminal.PasteFromClipboard":         true,
	// Tabs
	"Terminal.OpenNewTab":                 true,
	"Terminal.CloseTab":                   true,
	"Terminal.ClosePane":                  true,
	"Terminal.CloseOtherTabs":             true,
	"Terminal.CloseWindow":                true,
	"Terminal.NextTab":                    true,
	"Terminal.PrevTab":                    true,
	"Terminal.SwitchToTab0":               true,
	"Terminal.SwitchToTab1":               true,
	"Terminal.SwitchToTab2":               true,
	"Terminal.SwitchToTab3":               true,
	"Terminal.SwitchToTab4":               true,
	"Terminal.SwitchToTab5":               true,
	"Terminal.SwitchToTab6":               true,
	"Terminal.SwitchToTab7":               true,
	"Terminal.MoveTabToNextWindow":        true,
	"Terminal.MoveTabToPrevWindow":        true,
	// Panes
	"Terminal.SplitVertical":              true,
	"Terminal.SplitHorizontal":            true,
	"Terminal.SplitPaneRight":             true,
	"Terminal.SplitPaneDown":              true,
	"Terminal.SplitPaneUp":                true,
	"Terminal.SplitPaneLeft":              true,
	"Terminal.MovePaneUp":                 true,
	"Terminal.MovePaneDown":               true,
	"Terminal.MovePaneLeft":               true,
	"Terminal.MovePaneRight":              true,
	"Terminal.MovePaneToTab0":             true, // legacy — moves pane to tab 0
	"Terminal.MovePaneToTab1":             true,
	"Terminal.MovePaneToTab2":             true,
	"Terminal.MovePaneToTab3":             true,
	"Terminal.MovePaneToTab4":             true,
	"Terminal.MovePaneToTab5":             true,
	"Terminal.MovePaneToTab6":             true,
	"Terminal.MovePaneToTab7":             true,
	"Terminal.MovePaneToTab8":             true,
	"Terminal.MoveFocusUp":                true,
	"Terminal.MoveFocusDown":              true,
	"Terminal.MoveFocusLeft":              true,
	"Terminal.MoveFocusRight":             true,
	"Terminal.SwapPaneUp":                 true,
	"Terminal.SwapPaneDown":               true,
	"Terminal.SwapPaneLeft":               true,
	"Terminal.SwapPaneRight":              true,
	"Terminal.TogglePaneZoom":             true,
	"Terminal.TogglePaneReadOnly":         true,
	// Scroll
	"Terminal.ScrollUp":                   true,
	"Terminal.ScrollDown":                 true,
	"Terminal.ScrollPageUp":               true,
	"Terminal.ScrollPageDown":             true,
	"Terminal.ScrollToTop":                true,
	"Terminal.ScrollToBottom":             true,
	// Font
	"Terminal.IncreaseFontSize":           true,
	"Terminal.DecreaseFontSize":           true,
	"Terminal.ResetFontSize":              true,
	// Search
	"Terminal.FindText":                   true,
	// UI
	"Terminal.ToggleFullscreen":           true,
	"Terminal.ToggleAlwaysOnTop":          true,
	"Terminal.ToggleCommandPalette":       true,
	"Terminal.ReloadCommandPalette":       true,
	"Terminal.OpenSettingsFile":           true,
	"Terminal.OpenSystemMenu":             true,
	"Terminal.OpenAbout":                  true,
	// Misc
	"Terminal.DiscardCommandHistory":      true,
	"Terminal.SelectAll":                  true,
	"Terminal.MarkOutput":                 true,
	"Terminal.ClearBuffer":                true,
	"Terminal.ClearScrollback":            true,
	"Terminal.Experimental_RenameTab":     true,
	"Terminal.Experimental_SetTabColor":   true,
	"Terminal.OpenNewTabDropdown":         true,
	"Terminal.OpenTabSearch":              true,
	"Terminal.Detach":                     true,
	"Terminal.AdjustOpacity":              true,
}

// keybindingFragment is the minimal shape we read from settings-fragment.json.
// We intentionally only decode the fields we need.
type keybindingFragment struct {
	Keybindings []struct {
		ID   string `json:"id"`
		Keys string `json:"keys"`
	} `json:"keybindings"`
	Actions []struct {
		ID string `json:"id"`
	} `json:"actions"`
}

func (k *ITKeybindings) Validate(_ context.Context, root string) Result {
	start := time.Now()
	var issues []string
	var warnings []string

	fragPath := filepath.Join(root, itFragmentRelPath)
	data, err := os.ReadFile(fragPath)
	if err != nil {
		return Result{ID: k.ID(), Name: k.Name(), Status: "fail", Weight: k.Weight(),
			Message: fmt.Sprintf("FAIL — cannot read IT fragment: %v", err),
			Duration: time.Since(start)}
	}

	var frag keybindingFragment
	if err := json.Unmarshal(data, &frag); err != nil {
		return Result{ID: k.ID(), Name: k.Name(), Status: "fail", Weight: k.Weight(),
			Message: fmt.Sprintf("FAIL — invalid JSON in %s: %v", itFragmentRelPath, err),
			Duration: time.Since(start)}
	}

	// Build resolved-id set: builtin ∪ fragment-defined actions
	resolved := map[string]bool{}
	for id := range itBuiltinActions {
		resolved[id] = true
	}
	for _, a := range frag.Actions {
		if a.ID != "" {
			resolved[a.ID] = true
		}
	}

	// Detect null/empty IDs and unresolved IDs
	var nullIDs []string
	var unresolved []string
	keysSeen := map[string]string{} // keys → id (for duplicate detection)

	for i, kb := range frag.Keybindings {
		entry := fmt.Sprintf("entry #%d (keys=%q)", i, kb.Keys)
		if strings.TrimSpace(kb.Keys) == "" {
			issues = append(issues, fmt.Sprintf("EMPTY_KEYS: %s", entry))
			continue
		}
		if strings.TrimSpace(kb.ID) == "" {
			nullIDs = append(nullIDs, fmt.Sprintf("%s keys=%q", entry, kb.Keys))
			continue
		}
		if !resolved[kb.ID] {
			unresolved = append(unresolved, fmt.Sprintf("%s id=%q", entry, kb.ID))
		}
		// Duplicate-key detection (same keys value bound to multiple distinct ids)
		if prev, ok := keysSeen[kb.Keys]; ok && prev != kb.ID {
			warnings = append(warnings, fmt.Sprintf(
				"DUPLICATE_KEY: keys=%q bound to both id=%q and id=%q (later overrides earlier)",
				kb.Keys, prev, kb.ID))
		}
		keysSeen[kb.Keys] = kb.ID
	}

	// Warn if fragment has no keybindings at all (suspicious — IT will use defaults)
	if len(frag.Keybindings) == 0 {
		warnings = append(warnings, "EMPTY_KEYBINDINGS: fragment declares 0 keybindings — IT will use built-in defaults (unintended)")
	}

	for _, msg := range nullIDs {
		issues = append(issues, "NULL_ID: "+msg)
	}
	for _, msg := range unresolved {
		issues = append(issues, "UNRESOLVED_ID: "+msg)
	}

	sort.Strings(issues)
	sort.Strings(warnings)

	status := "pass"
	msg := fmt.Sprintf("PASS — %d keybindings validated", len(frag.Keybindings))
	if len(issues) > 0 {
		status = "fail"
		msg = fmt.Sprintf("FAIL — %d keybinding issue(s) in %s", len(issues), itFragmentRelPath)
	} else if len(warnings) > 0 {
		status = "warn"
		msg = fmt.Sprintf("WARN — %d keybinding warning(s) in %s", len(warnings), itFragmentRelPath)
	}

	all := append(append([]string(nil), issues...), warnings...)
	return Result{
		ID:       k.ID(),
		Name:     k.Name(),
		Status:   status,
		Weight:   k.Weight(),
		Message:  msg,
		Issues:   all,
		Duration: time.Since(start),
	}
}

var _ Validator = (*ITKeybindings)(nil)
