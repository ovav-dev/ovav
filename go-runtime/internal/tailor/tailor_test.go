package tailor

import (
	"testing"
)

// ── NewState ─────────────────────────────────────────────────────────

func TestNewState_InitialState(t *testing.T) {
	s := NewState(nil)

	if s.SelectedPlan != "nucleo" {
		t.Errorf("expected nucleo, got %s", s.SelectedPlan)
	}
	if len(s.Plans) != 3 {
		t.Errorf("expected 3 plans, got %d", len(s.Plans))
	}
	if len(s.Tools) != 5 {
		t.Errorf("expected 5 tools, got %d", len(s.Tools))
	}
	if len(s.Roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(s.Roles))
	}
	if s.Applied == nil {
		t.Fatal("expected initial snapshot to be set")
	}
	if s.Applied.SelectedPlan != "nucleo" {
		t.Errorf("expected snapshot plan nucleo, got %s", s.Applied.SelectedPlan)
	}
}

func TestNewState_DetectedTools(t *testing.T) {
	detected := map[string]bool{"opencode": true, "git": true}
	s := NewState(detected)

	opencode := s.findTool("opencode")
	if opencode == nil {
		t.Fatal("expected opencode tool")
	}
	if !opencode.Detected {
		t.Error("expected opencode to be detected")
	}
	if opencode.DetectedNote != "detected" {
		t.Errorf("expected 'detected', got %s", opencode.DetectedNote)
	}

	nvim := s.findTool("nvim")
	if nvim == nil {
		t.Fatal("expected nvim tool")
	}
	if nvim.Detected {
		t.Error("expected nvim NOT to be detected")
	}
	if nvim.DetectedNote != "prepared" {
		t.Errorf("expected 'prepared', got %s", nvim.DetectedNote)
	}
}

// ── PlanRank ─────────────────────────────────────────────────────────

func TestPlanRank(t *testing.T) {
	tests := []struct {
		id   string
		want int
	}{
		{"nucleo", 0},
		{"studio", 1},
		{"command", 2},
		{"nonexistent", -1},
		{"", -1},
	}
	for _, tt := range tests {
		got := planRank(tt.id)
		if got != tt.want {
			t.Errorf("planRank(%q) = %d, want %d", tt.id, got, tt.want)
		}
	}
}

func TestPlanLabel(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"nucleo", "Core"},
		{"studio", "Studio"},
		{"command", "Command"},
		{"", "no plan"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := PlanLabel(tt.id)
		if got != tt.want {
			t.Errorf("PlanLabel(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

// ── SelectPlan ──────────────────────────────────────────────────────

func TestSelectPlan_Valid(t *testing.T) {
	s := NewState(nil)

	err := s.SelectPlan("studio")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.SelectedPlan != "studio" {
		t.Errorf("expected studio, got %s", s.SelectedPlan)
	}

	// Nucleo plan should be inactive, studio active
	for _, p := range s.Plans {
		if p.ID == "studio" && !p.Active {
			t.Error("expected studio plan to be active")
		}
		if p.ID == "nucleo" && p.Active {
			t.Error("expected nucleo plan to be inactive")
		}
	}
}

func TestSelectPlan_Invalid(t *testing.T) {
	s := NewState(nil)
	err := s.SelectPlan("enterprise")
	if err == nil {
		t.Error("expected error for invalid plan")
	}
}

func TestSelectPlan_DisablesDisallowed(t *testing.T) {
	s := NewState(nil)

	// At nucleo: nvim requires studio, so if we try to enable it, then
	// switch to nucleo, it should become disabled.
	// First switch to studio to enable nvim
	s.SelectPlan("studio")
	nvim := s.findTool("nvim")
	nvim.Active = true

	// Switch back to nucleo
	s.SelectPlan("nucleo")
	nvim = s.findTool("nvim")
	if nvim.Active {
		t.Error("expected nvim to be disabled when switching to nucleo")
	}
}

// ── IsAllowed ───────────────────────────────────────────────────────

func TestIsAllowed(t *testing.T) {
	s := NewState(nil)
	s.SelectedPlan = "nucleo"

	if !s.IsAllowed("nucleo") {
		t.Error("nucleo should allow nucleo items")
	}
	if !s.IsAllowed("") {
		t.Error("empty min_plan should always be allowed")
	}
	if s.IsAllowed("studio") {
		t.Error("nucleo should NOT allow studio items")
	}
	if s.IsAllowed("command") {
		t.Error("nucleo should NOT allow command items")
	}

	s.SelectedPlan = "studio"
	if !s.IsAllowed("studio") {
		t.Error("studio should allow studio items")
	}
	if s.IsAllowed("command") {
		t.Error("studio should NOT allow command items")
	}

	s.SelectedPlan = "command"
	if !s.IsAllowed("command") {
		t.Error("command should allow command items")
	}
}

// ── ToggleAt ────────────────────────────────────────────────────────

func TestToggleAt_ToggleTool(t *testing.T) {
	s := NewState(nil)

	// Find the index of opencode in selectable rows
	rows := s.SelectableRows()
	var opencodeIdx int
	for i, r := range rows {
		if r.ID == "opencode" {
			opencodeIdx = i
			break
		}
	}

	msg := s.ToggleAt(opencodeIdx)
	if msg != "OpenCode: included" {
		t.Errorf("expected 'OpenCode: included', got %q", msg)
	}

	opencode := s.findTool("opencode")
	if !opencode.Active {
		t.Error("expected opencode to be active after toggle")
	}

	// Toggle again
	msg = s.ToggleAt(opencodeIdx)
	if msg != "OpenCode: removed" {
		t.Errorf("expected 'OpenCode: removed', got %q", msg)
	}
	if opencode.Active {
		t.Error("expected opencode to be inactive after second toggle")
	}
}

func TestToggleAt_SelectPlan(t *testing.T) {
	s := NewState(nil)

	// Plan rows come first in selectable rows
	rows := s.SelectableRows()
	var studioIdx int
	for i, r := range rows {
		if r.ID == "studio" {
			studioIdx = i
			break
		}
	}

	msg := s.ToggleAt(studioIdx)
	if s.SelectedPlan != "studio" {
		t.Errorf("expected studio, got %s", s.SelectedPlan)
	}
	if msg != "Plan Studio selected. Compatible options unlocked." {
		t.Errorf("unexpected message: %q", msg)
	}
}

func TestToggleAt_BlockedItem(t *testing.T) {
	// Items below the current plan are hidden entirely from selectable rows.
	// Test the action row with no plan selected:
	s := NewState(nil)
	s.SelectedPlan = "" // no plan
	rows := s.SelectableRows()
	// Find the install action row
	var actionIdx int
	for i, r := range rows {
		if r.Type == "action" {
			actionIdx = i
			break
		}
	}
	msg := s.ToggleAt(actionIdx)
	if msg != "Choose a plan first to continue." {
		t.Errorf("expected plan-first message, got %q", msg)
	}
}

// ── SectionedRows ───────────────────────────────────────────────────

func TestSectionedRows_Structure(t *testing.T) {
	s := NewState(nil)
	rows := s.SectionedRows()

	// Check we have all section headers
	hasPlanSection := false
	hasToolSection := false
	hasRoleSection := false
	hasAction := false

	for _, r := range rows {
		if r.Type == "section" && r.Label == "Plan" {
			hasPlanSection = true
		}
		if r.Type == "section" && r.Label == "Tools" {
			hasToolSection = true
		}
		if r.Type == "section" && r.Label == "Roles" {
			hasRoleSection = true
		}
		if r.Type == "action" {
			hasAction = true
		}
	}

	if !hasPlanSection {
		t.Error("missing Plan section")
	}
	if !hasToolSection {
		t.Error("missing Tools section")
	}
	if !hasRoleSection {
		t.Error("missing Roles section")
	}
	if !hasAction {
		t.Error("missing Install action")
	}
}

func TestSectionedRows_HidesDisallowedTools(t *testing.T) {
	s := NewState(nil)
	// At nucleo, tools requiring studio+ should be hidden
	rows := s.SectionedRows()

	for _, r := range rows {
		if r.Type == "item" && r.Kind == "tool" {
			if r.ID == "nvim" || r.ID == "zellij" || r.ID == "fish" {
				// These require studio — should be hidden at nucleo
				// Wait, nvim requires studio: PlanOrder index 1. nucleo is 0.
				// So nvim is NOT allowed at nucleo. It should be hidden.
				t.Errorf("tool %s should be hidden at nucleo plan", r.ID)
			}
		}
	}

	// Switch to studio — now studio tools should appear
	s.SelectPlan("studio")
	rows = s.SectionedRows()
	foundNvim := false
	for _, r := range rows {
		if r.Type == "item" && r.Kind == "tool" && r.ID == "nvim" {
			foundNvim = true
		}
	}
	if !foundNvim {
		t.Error("expected nvim to appear at studio plan")
	}
}

func TestSectionedRows_HidesDisallowedRoles(t *testing.T) {
	s := NewState(nil)
	rows := s.SectionedRows()

	for _, r := range rows {
		if r.Type == "item" && r.Kind == "role" {
			if r.ID == "security_architecture" {
				t.Errorf("role security_architecture should be hidden at nucleo")
			}
		}
	}
}

// ── SelectableRows ──────────────────────────────────────────────────

func TestSelectableRows_OnlyItemsAndActions(t *testing.T) {
	s := NewState(nil)
	rows := s.SelectableRows()
	for _, r := range rows {
		if r.Type != "item" && r.Type != "action" {
			t.Errorf("unexpected row type %q in selectable rows", r.Type)
		}
	}
}

// ── Snapshot ────────────────────────────────────────────────────────

func TestSnapshot(t *testing.T) {
	s := NewState(nil)
	snap := s.Snapshot()

	if snap.SelectedPlan != "nucleo" {
		t.Errorf("expected nucleo, got %s", snap.SelectedPlan)
	}
	if len(snap.Plans) != 3 {
		t.Errorf("expected 3 plans, got %d", len(snap.Plans))
	}
	if !snap.Plans["nucleo"] {
		t.Error("expected nucleo to be active in snapshot")
	}
	if snap.Plans["studio"] {
		t.Error("expected studio to be inactive in snapshot")
	}
}

// ── PreviewChanges ──────────────────────────────────────────────────

func TestPreviewChanges_NoChanges(t *testing.T) {
	s := NewState(nil)
	changes := s.PreviewChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestPreviewChanges_PlanChange(t *testing.T) {
	s := NewState(nil)
	s.SelectPlan("studio")
	changes := s.PreviewChanges()
	if len(changes) < 1 {
		t.Fatal("expected at least 1 change")
	}
	if changes[0].Label != "Plan" {
		t.Errorf("expected Plan change, got %s", changes[0].Label)
	}
}

func TestPreviewChanges_ToolChange(t *testing.T) {
	s := NewState(nil)
	s.SelectPlan("studio")
	// Enable nvim
	nvim := s.findTool("nvim")
	nvim.Active = true

	changes := s.PreviewChanges()
	found := false
	for _, c := range changes {
		if c.Label == "Tool · Neovim" {
			found = true
			if !c.After {
				t.Error("expected nvim change to be an addition")
			}
		}
	}
	if !found {
		t.Error("expected nvim change in preview")
	}
}

func TestPreviewChanges_NilApplied(t *testing.T) {
	s := NewState(nil)
	s.Applied = nil
	changes := s.PreviewChanges()
	if changes != nil {
		t.Errorf("expected nil changes when Applied is nil, got %d", len(changes))
	}
}

// ── ApplySelection ──────────────────────────────────────────────────

func TestApplySelection(t *testing.T) {
	s := NewState(nil)
	s.SelectPlan("studio")
	nvim := s.findTool("nvim")
	nvim.Active = true

	results := s.ApplySelection()
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if s.Applied == nil {
		t.Fatal("expected Applied to be set")
	}
	if s.Applied.SelectedPlan != "studio" {
		t.Errorf("expected applied plan studio, got %s", s.Applied.SelectedPlan)
	}
	if !s.Applied.Tools["nvim"] {
		t.Error("expected nvim to be active in applied snapshot")
	}
	if s.LastAppliedAt == "" {
		t.Error("expected LastAppliedAt to be set")
	}
}

// ── InstallSummary ──────────────────────────────────────────────────

func TestInstallSummary(t *testing.T) {
	s := NewState(nil)
	summary := s.InstallSummary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

// ── InstallConfirmationRows ─────────────────────────────────────────

func TestInstallConfirmationRows(t *testing.T) {
	s := NewState(nil)
	rows := s.InstallConfirmationRows()
	if len(rows) == 0 {
		t.Fatal("expected confirmation rows")
	}
}

func TestInstallConfirmationRows_NoPlan(t *testing.T) {
	s := NewState(nil)
	s.SelectedPlan = ""
	rows := s.InstallConfirmationRows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Priority != 1 {
		t.Errorf("expected priority 1, got %d", rows[0].Priority)
	}
}

// ── RowHint ─────────────────────────────────────────────────────────

func TestRowHint(t *testing.T) {
	s := NewState(nil)
	rows := s.SelectableRows()

	// Plan row hint
	var planIdx int
	for i, r := range rows {
		if r.Kind == "plan" && r.ID == "nucleo" {
			planIdx = i
			break
		}
	}
	hint := s.RowHint(planIdx)
	if hint == "" {
		t.Error("expected non-empty hint for plan row")
	}

	// Tool row hint
	var toolIdx int
	for i, r := range rows {
		if r.Kind == "tool" {
			toolIdx = i
			break
		}
	}
	hint = s.RowHint(toolIdx)
	if hint == "" {
		t.Error("expected non-empty hint for tool row")
	}
}

// ── Edge Cases ──────────────────────────────────────────────────────

func TestToggleAt_OutOfRange(t *testing.T) {
	s := NewState(nil)
	// Modulo arithmetic should handle this
	msg := s.ToggleAt(999)
	if msg == "" {
		t.Error("expected non-empty message even for out-of-range index")
	}
}

func TestSelectableRows_AfterPlanSwitch(t *testing.T) {
	s := NewState(nil)
	before := len(s.SelectableRows())

	s.SelectPlan("command")
	after := len(s.SelectableRows())

	if after <= before {
		t.Errorf("expected more selectable rows at command plan: before=%d, after=%d", before, after)
	}
}

func TestHasChanges(t *testing.T) {
	s := NewState(nil)
	if s.HasChanges() {
		t.Error("expected no changes in fresh state")
	}

	s.SelectPlan("studio")
	if !s.HasChanges() {
		t.Error("expected changes after plan switch")
	}

	s.ApplySelection()
	if s.HasChanges() {
		t.Error("expected no changes after apply")
	}
}

func TestActiveToolCount(t *testing.T) {
	s := NewState(nil)
	initial := s.ActiveToolCount()

	// Enable a tool
	opencode := s.findTool("opencode")
	opencode.Active = true
	if s.ActiveToolCount() != initial+1 {
		t.Error("expected tool count to increase")
	}
}

func TestActiveRoleCount(t *testing.T) {
	s := NewState(nil)
	initial := s.ActiveRoleCount()

	// Enable a role
	pe := s.findRole("platform_engineering")
	pe.Active = true
	if s.ActiveRoleCount() != initial+1 {
		t.Error("expected role count to increase")
	}
}

func TestPreviewResultRows_Empty(t *testing.T) {
	s := NewState(nil)
	s.SelectedPlan = ""
	rows := s.PreviewResultRows()
	if len(rows) != 1 || rows[0].Priority != 1 {
		t.Error("expected warning row when no plan selected")
	}
}

func TestPreviewResultRows_NoChanges(t *testing.T) {
	s := NewState(nil)
	rows := s.PreviewResultRows()
	if len(rows) == 0 {
		t.Error("expected rows even with no changes")
	}
}

// ── Plan-gating: command unlocks everything ─────────────────────────

func TestCommandUnlocksAll(t *testing.T) {
	s := NewState(nil)
	s.SelectPlan("command")

	// All tools should be visible
	rows := s.SectionedRows()
	toolCount := 0
	for _, r := range rows {
		if r.Type == "item" && r.Kind == "tool" {
			toolCount++
		}
	}
	if toolCount != 5 {
		t.Errorf("expected 5 tools visible at command, got %d", toolCount)
	}

	// All roles should be visible
	roleCount := 0
	for _, r := range rows {
		if r.Type == "item" && r.Kind == "role" {
			roleCount++
		}
	}
	if toolCount != 5 {
		t.Errorf("expected 3 roles visible at command, got %d", roleCount)
	}
}
