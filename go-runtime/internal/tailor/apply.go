package tailor

import (
	"fmt"
	"strings"
)

// ── ApplySelection ──────────────────────────────────────────────────

// ApplySelection commits the current state as the new applied snapshot.
// Returns structured result rows describing what was applied.
func (s *State) ApplySelection() []ResultRow {
	changes := s.PreviewChanges()
	snap := s.Snapshot()
	s.Applied = &snap
	s.LastAppliedAt = nowUTC()
	s.LastMessage = "Configuration ready to install."

	activeTools := s.ActiveToolLabels()
	activeRoles := s.ActiveRoleLabels()

	toolStr := "none"
	if len(activeTools) > 0 {
		toolStr = strings.Join(activeTools, ", ")
	}
	roleStr := "none"
	if len(activeRoles) > 0 {
		roleStr = strings.Join(activeRoles, ", ")
	}

	nChanges := 0
	if changes != nil {
		nChanges = len(changes)
	}

	pri := 3
	if nChanges == 0 {
		pri = 5
	}

	return []ResultRow{
		{Label: "Status", Value: "configuration applied", Priority: 3},
		{Label: "Plan", Value: PlanLabel(s.SelectedPlan), Priority: 3},
		{Label: "Changes", Value: fmt.Sprintf("%d", nChanges), Priority: pri},
		{Label: "Tools", Value: toolStr, Priority: 2},
		{Label: "Roles", Value: roleStr, Priority: 2},
		{Label: "Next", Value: "Install OVAV from Tailor", Priority: 5},
	}
}

// ── InstallSummary ──────────────────────────────────────────────────

// InstallSummary returns a one-line summary of what will be installed.
func (s *State) InstallSummary() string {
	plan := PlanLabel(s.SelectedPlan)
	tools := s.ActiveToolCount()
	roles := s.ActiveRoleCount()
	return fmt.Sprintf("%s · %d tools · %d roles", plan, tools, roles)
}

// ── InstallConfirmationRows ─────────────────────────────────────────

// InstallConfirmationRows returns the rows shown on the final confirmation screen.
func (s *State) InstallConfirmationRows() []ResultRow {
	if s.SelectedPlan == "" {
		return []ResultRow{{Label: "Plan", Value: "choose a plan to continue", Priority: 1}}
	}

	toolStr := "none"
	if labels := s.ActiveToolLabels(); len(labels) > 0 {
		toolStr = strings.Join(labels, ", ")
	}
	roleStr := "none"
	if labels := s.ActiveRoleLabels(); len(labels) > 0 {
		roleStr = strings.Join(labels, ", ")
	}

	return []ResultRow{
		{Label: "Plan", Value: PlanLabel(s.SelectedPlan), Priority: 3},
		{Label: "Tools", Value: toolStr, Priority: 2},
		{Label: "Roles", Value: roleStr, Priority: 2},
		{Label: "Installation", Value: "governed pipeline with confirmation", Priority: 3},
		{Label: "Safety", Value: "preview · backup · verify", Priority: 5},
	}
}
