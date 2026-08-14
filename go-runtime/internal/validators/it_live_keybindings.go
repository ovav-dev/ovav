package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// ITLiveKeybindings validates the LIVE Intelligent Terminal settings.json
// (not just the repo fragment). Detects the same issues as ITKeybindings
// (NULL_ID, UNRESOLVED_ID, EMPTY_KEYS) but on the file the user actually
// runs IT with.
//
// Why this exists: commit bc1fb2b (2026-08-14) fixed 17 broken keybindings
// in workstation/configs/intelligent-terminal/settings-fragment.json — but
// the LIVE /mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.Intelligent
// Terminal_8wekyb3d8bbwe/LocalState/settings.json was never re-deployed.
// it_keybindings validator still PASSed because it only checked the fragment.
// Result: IT still had id:null entries that broke escape-sequence parsing,
// causing shift+arrow to type literal "D" and trigger Windows beep.
//
// Resolution: run workstation/scripts/deploy-it-keybindings.sh after any
// fragment change. This validator catches the drift before CEO notices.
type ITLiveKeybindings struct{}

func NewITLiveKeybindings() *ITLiveKeybindings { return &ITLiveKeybindings{} }

func (l *ITLiveKeybindings) ID() string   { return "it_live_keybindings" }
func (l *ITLiveKeybindings) Name() string { return "IT Live Keybindings Validator" }
func (l *ITLiveKeybindings) Description() string {
	return "Validates the LIVE Intelligent Terminal settings.json (drift detection vs fragment)"
}
func (l *ITLiveKeybindings) Weight() int { return 8 }

// defaultLivePaths are the WSL-mount locations where IT settings live.
// If any of them exists, we validate it. None of these paths are required —
// if no live settings.json is found, the validator returns SKIP (not FAIL).
var defaultLivePaths = []string{
	"/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json",
}

// resolveLivePath picks the first existing live settings.json.
// Override via OVAV_LIVE_IT_SETTINGS env var — when set, ONLY that path is
// considered (no fallback to defaults). This lets tests and CI pin the path
// explicitly. When unset, falls back to defaultLivePaths.
func resolveLivePath(root string) string {
	if v, ok := os.LookupEnv("OVAV_LIVE_IT_SETTINGS"); ok {
		if v == "" {
			return "" // explicit empty → no live path
		}
		if _, err := os.Stat(v); err == nil {
			return v
		}
		return "" // explicit path doesn't exist → no live path
	}
	for _, p := range defaultLivePaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	_ = root // root is unused; kept for future expansion (worktree-relative paths)
	return ""
}

type liveKeybindingFragment struct {
	Keybindings []struct {
		ID   string `json:"id"`
		Keys string `json:"keys"`
	} `json:"keybindings"`
	Actions []struct {
		ID string `json:"id"`
	} `json:"actions"`
}

func (l *ITLiveKeybindings) Validate(_ context.Context, root string) Result {
	start := time.Now()

	livePath := resolveLivePath(root)
	if livePath == "" {
		return Result{
			ID:       l.ID(),
			Name:     l.Name(),
			Status:   "skip",
			Weight:   l.Weight(),
			Message:  "SKIP — no live IT settings.json found (set OVAV_LIVE_IT_SETTINGS to override)",
			Duration: time.Since(start),
		}
	}

	data, err := os.ReadFile(livePath)
	if err != nil {
		return Result{
			ID:       l.ID(),
			Name:     l.Name(),
			Status:   "fail",
			Weight:   l.Weight(),
			Message:  fmt.Sprintf("FAIL — cannot read live IT settings: %v", err),
			Duration: time.Since(start),
		}
	}

	var frag liveKeybindingFragment
	if err := json.Unmarshal(data, &frag); err != nil {
		return Result{
			ID:       l.ID(),
			Name:     l.Name(),
			Status:   "fail",
			Weight:   l.Weight(),
			Message:  fmt.Sprintf("FAIL — invalid JSON in %s: %v", livePath, err),
			Duration: time.Since(start),
		}
	}

	// Build resolved-id set: builtins ∪ live-defined actions
	resolved := map[string]bool{}
	for id := range itBuiltinActions {
		resolved[id] = true
	}
	for _, a := range frag.Actions {
		if a.ID != "" {
			resolved[a.ID] = true
		}
	}

	var nullIDs []string
	var unresolved []string
	keysSeen := map[string]string{}
	nonEmpty := 0

	for i, kb := range frag.Keybindings {
		entry := fmt.Sprintf("entry #%d (keys=%q)", i, kb.Keys)
		if strings.TrimSpace(kb.Keys) == "" {
			nullIDs = append(nullIDs, fmt.Sprintf("%s (EMPTY_KEYS)", entry))
			continue
		}
		nonEmpty++
		if strings.TrimSpace(kb.ID) == "" {
			nullIDs = append(nullIDs, fmt.Sprintf("%s keys=%q", entry, kb.Keys))
			continue
		}
		if !resolved[kb.ID] {
			unresolved = append(unresolved, fmt.Sprintf("%s id=%q", entry, kb.ID))
		}
		if prev, ok := keysSeen[kb.Keys]; ok && prev != kb.ID {
			// duplicate-key already covered by ITKeybindings validator
			_ = prev
		}
		keysSeen[kb.Keys] = kb.ID
	}

	if len(frag.Keybindings) == 0 {
		return Result{
			ID:       l.ID(),
			Name:     l.Name(),
			Status:   "warn",
			Weight:   l.Weight(),
			Message:  fmt.Sprintf("WARN — live IT settings has 0 keybindings (IT will use defaults)"),
			Duration: time.Since(start),
		}
	}

	var issues []string
	for _, msg := range nullIDs {
		issues = append(issues, "NULL_ID: "+msg)
	}
	for _, msg := range unresolved {
		issues = append(issues, "UNRESOLVED_ID: "+msg)
	}
	sort.Strings(issues)

	if len(issues) > 0 {
		return Result{
			ID:       l.ID(),
			Name:     l.Name(),
			Status:   "fail",
			Weight:   l.Weight(),
			Message: fmt.Sprintf(
				"FAIL — %d keybinding issue(s) in live IT settings (%s). Re-run: workstation/scripts/deploy-it-keybindings.sh",
				len(issues), livePath),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID:       l.ID(),
		Name:     l.Name(),
		Status:   "pass",
		Weight:   l.Weight(),
		Message:  fmt.Sprintf("PASS — %d live keybindings validated", nonEmpty),
		Duration: time.Since(start),
	}
}

var _ Validator = (*ITLiveKeybindings)(nil)
