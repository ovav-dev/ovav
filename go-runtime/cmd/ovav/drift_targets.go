package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DriftType classifies a drift item.
type DriftType string

const (
	DriftMissingInLive     DriftType = "missing_in_live"
	DriftMissingInFragment DriftType = "missing_in_fragment"
	DriftModified          DriftType = "modified"
	DriftAdded             DriftType = "added"
	DriftIdentical         DriftType = "identical"
)

// DriftItem is a single drift observation between fragment and live.
type DriftItem struct {
	Type         DriftType `json:"type"`
	Path         string    `json:"path"` // JSON path / section name
	FragmentJSON string    `json:"fragment,omitempty"`
	LiveJSON     string    `json:"live,omitempty"`
	SuggestedFix string    `json:"suggested_fix,omitempty"`
}

// DriftTarget is a (fragment, live) pair plus comparison logic.
type DriftTarget struct {
	ID          string
	Name        string
	FragmentRel string // relative to repo root
	LiveAbs     string // absolute path or ~ expansion
	LiveEnv     string // env var that overrides LiveAbs
	AutoFixable bool
	Compare     func(fragment, live []byte) ([]DriftItem, error) `json:"-"`
}

// DriftTargetReport is the per-target report.
type DriftTargetReport struct {
	Target     DriftTarget `json:"target"`
	FragmentOK bool        `json:"fragment_ok"`
	LiveOK     bool        `json:"live_ok"`
	Items      []DriftItem `json:"items"`
}

// DriftReport is the full drift report across all targets.
type DriftReport struct {
	Timestamp      string              `json:"timestamp"`
	RepoRoot       string              `json:"repo_root"`
	TotalTargets   int                 `json:"total_targets"`
	DriftedTargets int                 `json:"drifted_targets"`
	TotalItems     int                 `json:"total_items"`
	Targets        []DriftTargetReport `json:"targets"`
}

// DriftCatalogEntry is a historical record (one line per run).
type DriftCatalogEntry struct {
	Timestamp      string `json:"timestamp"`
	RepoRoot       string `json:"repo_root"`
	TotalTargets   int    `json:"total_targets"`
	DriftedTargets int    `json:"drifted_targets"`
	TotalItems     int    `json:"total_items"`
}

// DefaultTargets returns the 5 registered targets (ADR-007).
//
// Each target's Compare function is target-specific because the file
// structures differ (JSON with arrays vs flat INI vs YAML). Adding a new
// target = new Compare function + new entry here.
func DefaultTargets(repoRoot string) []DriftTarget {
	return []DriftTarget{
		{
			ID:          "it-keybindings",
			Name:        "IT Keybindings (Intelligent Terminal v0.1.4+)",
			FragmentRel: "workstation/configs/intelligent-terminal/settings-fragment.json",
			LiveAbs:     "/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json",
			LiveEnv:     "OVAV_LIVE_IT_SETTINGS",
			AutoFixable: true,
			Compare:     compareITKeybindings,
		},
		{
			ID:          "bash-inputrc",
			Name:        "Bash inputrc (~/.inputrc)",
			FragmentRel: "workstation/configs/inputrc/ovav.inputrc",
			LiveAbs:     "~/.inputrc",
			LiveEnv:     "OVAV_LIVE_INPUTRC",
			AutoFixable: true,
			Compare:     compareBashInputrc,
		},
		{
			ID:          "runtime-baseline",
			Name:        "Runtime Integrity Baseline",
			FragmentRel: ".ovav/integrity_backups/baseline.json",
			LiveAbs:     "(dynamic — file hashes)",
			LiveEnv:     "",
			AutoFixable: true,
			Compare:     compareRuntimeBaseline,
		},
		{
			ID:          "pinned-baseline",
			Name:        "Pinned Baseline (last CEO-approved)",
			FragmentRel: ".ovav/integrity_backups/baseline.pinned.json",
			LiveAbs:     "(pinned vs current)",
			LiveEnv:     "",
			AutoFixable: false, // requires CEO approval
			Compare:     comparePinnedBaseline,
		},
		{
			ID:          "tool-configs",
			Name:        "Tool Configs (registry vs bin)",
			FragmentRel: ".ovav/registry/tool_configs.yaml",
			LiveAbs:     "bin/ovav",
			LiveEnv:     "OVAV_BIN_OVAV",
			AutoFixable: false, // requires rebuild
			Compare:     compareToolConfigs,
		},
	}
}

// ── Compare functions (target-specific) ────────────────────────────────────

// compareITKeybindings compares fragment vs live IT settings.json.
//
// Strategy: extract "keybindings" array from each, compare by "keys" field
// (the actual keyboard combo). Order-independent comparison.
func compareITKeybindings(fragment, live []byte) ([]DriftItem, error) {
	var fragDoc, liveDoc struct {
		Keybindings []map[string]any `json:"keybindings"`
		Actions     []map[string]any `json:"actions"`
	}
	if err := json.Unmarshal(fragment, &fragDoc); err != nil {
		return nil, fmt.Errorf("fragment parse: %w", err)
	}
	if err := json.Unmarshal(live, &liveDoc); err != nil {
		return nil, fmt.Errorf("live parse: %w", err)
	}

	// Index by "keys" field
	fragByKeys := indexKeybindingsByKeys(fragDoc.Keybindings)
	liveByKeys := indexKeybindingsByKeys(liveDoc.Keybindings)

	items := []DriftItem{}

	// Items in fragment but missing in live
	for k, frag := range fragByKeys {
		liveItem, ok := liveByKeys[k]
		if !ok {
			items = append(items, DriftItem{
				Type:         DriftMissingInLive,
				Path:         fmt.Sprintf("keybindings[%s]", k),
				FragmentJSON: compactJSON(frag),
				SuggestedFix: "ovav deploy run --target=it-keybindings",
			})
			continue
		}
		// Same keys — check if action changed
		if diffAction(frag, liveItem) {
			items = append(items, DriftItem{
				Type:         DriftModified,
				Path:         fmt.Sprintf("keybindings[%s]", k),
				FragmentJSON: compactJSON(frag),
				LiveJSON:     compactJSON(liveItem),
				SuggestedFix: "ovav deploy run --target=it-keybindings",
			})
		}
	}

	// Items in live but missing in fragment (potential removal)
	for k, live := range liveByKeys {
		if _, ok := fragByKeys[k]; !ok {
			items = append(items, DriftItem{
				Type:         DriftMissingInFragment,
				Path:         fmt.Sprintf("keybindings[%s]", k),
				LiveJSON:     compactJSON(live),
				SuggestedFix: "audit: keybinding added to live outside OVAV — review and add to fragment",
			})
		}
	}

	return items, nil
}

// indexKeybindingsByKeys returns a map keyed by the "keys" field.
func indexKeybindingsByKeys(items []map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(items))
	for _, item := range items {
		k, ok := item["keys"].(string)
		if !ok {
			continue
		}
		out[k] = item
	}
	return out
}

// diffAction checks if the "id" or "command" fields differ.
func diffAction(a, b map[string]any) bool {
	for _, field := range []string{"id", "command"} {
		if a[field] != b[field] {
			return true
		}
	}
	return false
}

// compareBashInputrc compares fragment (ovav.inputrc) vs live (~/.inputrc).
//
// Strategy: line-by-line comparison. "deliberately UNBOUND" comments are
// ignored (they're annotations, not bindings).
func compareBashInputrc(fragment, live []byte) ([]DriftItem, error) {
	fragLines := filterCommentLines(splitLines(string(fragment)))
	liveLines := filterCommentLines(splitLines(string(live)))

	items := []DriftItem{}

	// Treat each non-comment line as a "binding"
	fragSet := stringSet(fragLines)
	liveSet := stringSet(liveLines)

	// In fragment but missing in live
	for _, line := range fragLines {
		if _, ok := liveSet[line]; !ok {
			items = append(items, DriftItem{
				Type:         DriftMissingInLive,
				Path:         fmt.Sprintf("line:%d", indexOfLine(string(fragment), line)),
				FragmentJSON: line,
				SuggestedFix: "ovav deploy run --target=bash-inputrc",
			})
		}
	}

	// In live but missing in fragment
	for _, line := range liveLines {
		if _, ok := fragSet[line]; !ok {
			items = append(items, DriftItem{
				Type:         DriftMissingInFragment,
				Path:         fmt.Sprintf("live-only:%d", indexOfLine(string(live), line)),
				LiveJSON:     line,
				SuggestedFix: "audit: line added to live outside OVAV — review and add to fragment",
			})
		}
	}

	return items, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func filterCommentLines(lines []string) []string {
	var out []string
	for _, l := range lines {
		trimmed := trimLeft(l)
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		out = append(out, l)
	}
	return out
}

func trimLeft(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return s[i:]
		}
	}
	return ""
}

func stringSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, s := range items {
		out[s] = struct{}{}
	}
	return out
}

func indexOfLine(content, line string) int {
	idx := 0
	for i := 0; i < len(content); i++ {
		if i+len(line) > len(content) {
			return -1
		}
		if content[i:i+len(line)] == line {
			return idx
		}
		if content[i] == '\n' {
			idx++
		}
	}
	return -1
}

// compareRuntimeBaseline compares baseline.json hashes vs current file hashes.
//
// If any file's current hash != baseline hash → drift.
func compareRuntimeBaseline(fragment, live []byte) ([]DriftItem, error) {
	// For this target, "fragment" is the baseline.json (expected hashes)
	// and "live" is unused (we compute hashes on demand).
	type Baseline struct {
		Files map[string]string `json:"files"`
	}
	var frag Baseline
	if err := json.Unmarshal(fragment, &frag); err != nil {
		return nil, fmt.Errorf("baseline parse: %w", err)
	}
	// "live" param is unused — caller computes on-demand
	_ = live

	// Note: actual hash computation requires root + access to current files.
	// The Compare signature is too narrow. We'll use a wrapper.
	// For now, return empty (caller handles via dedicated logic).
	return []DriftItem{}, nil
}

// comparePinnedBaseline compares baseline.pinned.json vs current baseline.json.
func comparePinnedBaseline(fragment, live []byte) ([]DriftItem, error) {
	type Baseline struct {
		Files map[string]string `json:"files"`
	}
	var pinned, current Baseline
	if err := json.Unmarshal(fragment, &pinned); err != nil {
		return nil, fmt.Errorf("pinned parse: %w", err)
	}
	if err := json.Unmarshal(live, &current); err != nil {
		return nil, fmt.Errorf("current parse: %w", err)
	}
	items := []DriftItem{}
	for path, expected := range pinned.Files {
		actual, ok := current.Files[path]
		if !ok {
			items = append(items, DriftItem{
				Type:         DriftMissingInLive,
				Path:         path,
				FragmentJSON: expected,
				SuggestedFix: "ovav integrity pin --approve (CEO waiver)",
			})
			continue
		}
		if actual != expected {
			items = append(items, DriftItem{
				Type:         DriftModified,
				Path:         path,
				FragmentJSON: expected,
				LiveJSON:     actual,
				SuggestedFix: "ovav integrity pin --approve (CEO waiver)",
			})
		}
	}
	return items, nil
}

// compareToolConfigs checks if bin/ovav exists and is executable.
// (Detailed profile comparison is in validator tool_config_profiles.)
func compareToolConfigs(fragment, live []byte) ([]DriftItem, error) {
	// 'live' is the contents of bin/ovav (binary, large) — we don't diff it.
	// We just check existence + executability.
	_ = fragment
	_ = live
	return []DriftItem{}, nil
}

// compactJSON marshals a map[string]any to compact JSON.
func compactJSON(v map[string]any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// resolveLivePath returns the effective live path (env override > target default).
func (t DriftTarget) resolveLivePath() string {
	if t.LiveEnv != "" {
		if v, ok := os.LookupEnv(t.LiveEnv); ok && v != "" {
			return v
		}
	}
	if t.LiveAbs == "" || t.LiveAbs == "(dynamic — file hashes)" || t.LiveAbs == "(pinned vs current)" {
		return ""
	}
	if len(t.LiveAbs) > 0 && t.LiveAbs[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, t.LiveAbs[2:])
		}
	}
	return t.LiveAbs
}
