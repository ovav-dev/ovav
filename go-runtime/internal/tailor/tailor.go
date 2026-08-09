// Package tailor provides the OVAV Tailor Composer engine.
//
// Port of tools/cli/ovav_tailor_composer.py (294 LOC Python).
// Models workstation setup as a plan-gated state machine:
//
//	Plans (nucleo < studio < command) unlock tools and roles.
//
// The library is UI-agnostic. Consumers (CLI, Cockpit TUI) render
// the sectioned rows however they prefer.
package tailor

import (
	"time"
)

// ── Plan ────────────────────────────────────────────────────────────

// PlanOrder defines the canonical ordering and rank for workstation plans.
// Rank is used for gating: a plan unlocks items whose min_plan rank
// is <= the selected plan's rank.
var PlanOrder = []string{"nucleo", "studio", "command"}

// Plan represents a workstation plan level.
type Plan struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Icon    string `json:"icon"`
	Tagline string `json:"tagline"`
	Active  bool   `json:"active"`
}

// DefaultPlans returns the canonical OVAV workstation plans.
func DefaultPlans() []Plan {
	return []Plan{
		{ID: "nucleo", Label: "Core", Icon: "◆", Tagline: "local base · free"},
		{ID: "studio", Label: "Studio", Icon: "✦", Tagline: "editor + sessions"},
		{ID: "command", Label: "Command", Icon: "⬢", Tagline: "advanced operation"},
	}
}

// ── Item ────────────────────────────────────────────────────────────

// ItemKind is either "tool" or "role".
type ItemKind string

const (
	KindTool ItemKind = "tool"
	KindRole ItemKind = "role"
)

// Item represents a selectable tool or role option.
type Item struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Note         string   `json:"note"`
	Kind         ItemKind `json:"kind"`
	MinPlan      string   `json:"min_plan"`
	Active       bool     `json:"active"`
	Detected     bool     `json:"detected,omitempty"`
	DetectedNote string   `json:"detected_note,omitempty"`
}

// DefaultTools returns the canonical OVAV tool specs.
func DefaultTools(detected map[string]bool) []Item {
	tools := []Item{
		{ID: "opencode", Label: "OpenCode", Note: "governed AI workspace", Kind: KindTool, MinPlan: "nucleo"},
		{ID: "git", Label: "Git", Note: "safe versioning", Kind: KindTool, MinPlan: "nucleo"},
		{ID: "nvim", Label: "Neovim", Note: "technical editing", Kind: KindTool, MinPlan: "studio"},
		{ID: "zellij", Label: "Zellij", Note: "live sessions", Kind: KindTool, MinPlan: "studio"},
		{ID: "fish", Label: "Fish", Note: "productive shell", Kind: KindTool, MinPlan: "studio"},
	}
	for i := range tools {
		if detected != nil && detected[tools[i].ID] {
			tools[i].Detected = true
			tools[i].DetectedNote = "detected"
		} else {
			tools[i].DetectedNote = "prepared"
		}
	}
	return tools
}

// DefaultRoles returns the canonical OVAV role specs.
func DefaultRoles() []Item {
	return []Item{
		{ID: "platform_engineering", Label: "Platform Engineering", Note: "systems, CLI and runtime", Kind: KindRole, MinPlan: "nucleo"},
		{ID: "research_intelligence", Label: "Research Intelligence", Note: "evidence and benchmarks", Kind: KindRole, MinPlan: "studio"},
		{ID: "security_architecture", Label: "Security Architecture", Note: "risk, permissions and safety", Kind: KindRole, MinPlan: "command"},
	}
}

// ── Snapshot ────────────────────────────────────────────────────────

// Snapshot captures the applied state at a point in time.
// Used to diff changes in PreviewChanges.
type Snapshot struct {
	SelectedPlan string          `json:"selected_plan"`
	Plans        map[string]bool `json:"plans"`
	Tools        map[string]bool `json:"tools"`
	Roles        map[string]bool `json:"roles"`
}

// ── State ───────────────────────────────────────────────────────────

// State is the full mutable Tailor composer state.
// All mutation methods are defined in composer.go, preview.go, apply.go.
type State struct {
	Plans []Plan `json:"plans"`
	Tools []Item `json:"tools"`
	Roles []Item `json:"roles"`

	SelectedPlan string `json:"selected_plan"`

	// Applied is the last snapshot that was "applied".
	// nil means nothing has been applied yet — initial state.
	Applied *Snapshot `json:"applied,omitempty"`

	LastAppliedAt string `json:"last_applied_at,omitempty"`
	LastMessage   string `json:"last_message"`
}

// NewState creates the initial Tailor state. detectedTools is a map of
// tool IDs that are already present on the system (from environment scan).
// Pass nil if no detection is available.
func NewState(detectedTools map[string]bool) *State {
	plans := DefaultPlans()
	// Nucleo is active by default
	for i := range plans {
		if plans[i].ID == "nucleo" {
			plans[i].Active = true
		}
	}

	tools := DefaultTools(detectedTools)
	roles := DefaultRoles()

	s := &State{
		Plans:        plans,
		Tools:        tools,
		Roles:        roles,
		SelectedPlan: "nucleo",
		LastMessage:  "Choose a plan to unlock your setup.",
	}

	// Capture initial snapshot as "applied"
	snapshot := s.Snapshot()
	s.Applied = &snapshot

	return s
}

// Snapshot returns an immutable capture of the current state.
func (s *State) Snapshot() Snapshot {
	snap := Snapshot{
		SelectedPlan: s.SelectedPlan,
		Plans:        make(map[string]bool, len(s.Plans)),
		Tools:        make(map[string]bool, len(s.Tools)),
		Roles:        make(map[string]bool, len(s.Roles)),
	}
	for _, p := range s.Plans {
		snap.Plans[p.ID] = p.Active
	}
	for _, t := range s.Tools {
		snap.Tools[t.ID] = t.Active
	}
	for _, r := range s.Roles {
		snap.Roles[r.ID] = r.Active
	}
	return snap
}

// ── Plan ranking ────────────────────────────────────────────────────

// planRank returns the numeric rank of a plan ID.
// -1 if not found.
func planRank(planID string) int {
	for i, id := range PlanOrder {
		if id == planID {
			return i
		}
	}
	return -1
}

// PlanLabel returns the human-readable label for a plan ID.
func PlanLabel(planID string) string {
	for _, p := range DefaultPlans() {
		if p.ID == planID {
			return p.Label
		}
	}
	if planID == "" {
		return "no plan"
	}
	return planID
}

// ── Helpers ─────────────────────────────────────────────────────────

// nowUTC is a test hook: override in tests to freeze time.
var nowUTC = func() string {
	return time.Now().UTC().Format(time.RFC3339)
}
