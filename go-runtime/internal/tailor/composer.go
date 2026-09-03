package tailor

import "fmt"

// ── Plan gating ─────────────────────────────────────────────────────

// IsAllowed returns true if the current selected plan satisfies minPlan.
// An empty minPlan means "always allowed".
func (s *State) IsAllowed(minPlan string) bool {
	if minPlan == "" {
		return true
	}
	return planRank(s.SelectedPlan) >= planRank(minPlan)
}

// ── SelectPlan ──────────────────────────────────────────────────────

// SelectPlan sets the active plan and disables items not allowed at this level.
// Returns a human-readable status message.
func (s *State) SelectPlan(planID string) error {
	found := false
	for i := range s.Plans {
		if s.Plans[i].ID == planID {
			found = true
			s.Plans[i].Active = true
		} else {
			s.Plans[i].Active = false
		}
	}
	if !found {
		return fmt.Errorf("plan %q is not available", planID)
	}

	s.SelectedPlan = planID
	s.disableDisallowed()

	msg := fmt.Sprintf("Plan %s selected. Compatible options unlocked.", PlanLabel(planID))
	s.LastMessage = msg
	return nil
}

// disableDisallowed turns off items whose min_plan is above the current selection.
func (s *State) disableDisallowed() {
	for i := range s.Tools {
		if !s.IsAllowed(s.Tools[i].MinPlan) {
			s.Tools[i].Active = false
		}
	}
	for i := range s.Roles {
		if !s.IsAllowed(s.Roles[i].MinPlan) {
			s.Roles[i].Active = false
		}
	}
}

// enableAllowed turns ON items whose min_plan is at or below the current
// selection. Mirrors disableDisallowed so initial state, plan upgrades,
// and explicit Apply all leave the composer in a consistent state where
// allowed items are Active=true and disallowed items are Active=false.
func (s *State) enableAllowed() {
	for i := range s.Tools {
		if s.IsAllowed(s.Tools[i].MinPlan) {
			s.Tools[i].Active = true
		}
	}
	for i := range s.Roles {
		if s.IsAllowed(s.Roles[i].MinPlan) {
			s.Roles[i].Active = true
		}
	}
}

// ApplyAllowed applies both enable + disable to keep the composer state
// consistent with the currently-selected plan. Used by initial-state
// constructors and any caller that has set SelectedPlan directly.
func (s *State) ApplyAllowed() {
	s.enableAllowed()
	s.disableDisallowed()
}

// ── ToggleAt ────────────────────────────────────────────────────────

// ToggleAt toggles the selectable row at the given index.
// The index refers to the flattened list from SelectableRows().
// Returns a human-readable message describing what happened.
func (s *State) ToggleAt(index int) string {
	rows := s.SelectableRows()
	if len(rows) == 0 {
		return "No options available."
	}

	row := rows[index%len(rows)]

	// Actions (install)
	if row.Type == "action" {
		if s.SelectedPlan == "" {
			msg := "Choose a plan first to continue."
			s.LastMessage = msg
			return msg
		}
		msg := "Ready to confirm installation. Press Enter."
		s.LastMessage = msg
		return msg
	}

	// Plan selection
	if row.Kind == "plan" {
		if err := s.SelectPlan(row.ID); err != nil {
			return err.Error()
		}
		return s.LastMessage
	}

	// Must have a plan selected
	if s.SelectedPlan == "" {
		msg := "Choose a plan first to continue."
		s.LastMessage = msg
		return msg
	}

	// Item must be allowed
	if !row.Allowed {
		msg := fmt.Sprintf("%s requires plan %s.", row.Label, PlanLabel(row.RequiredPlan))
		s.LastMessage = msg
		return msg
	}

	// Toggle the item
	switch row.Group {
	case "tools":
		for i := range s.Tools {
			if s.Tools[i].ID == row.ID {
				s.Tools[i].Active = !s.Tools[i].Active
				if s.Tools[i].Active {
					s.LastMessage = fmt.Sprintf("%s: included", row.Label)
				} else {
					s.LastMessage = fmt.Sprintf("%s: removed", row.Label)
				}
				return s.LastMessage
			}
		}
	case "roles":
		for i := range s.Roles {
			if s.Roles[i].ID == row.ID {
				s.Roles[i].Active = !s.Roles[i].Active
				if s.Roles[i].Active {
					s.LastMessage = fmt.Sprintf("%s: included", row.Label)
				} else {
					s.LastMessage = fmt.Sprintf("%s: removed", row.Label)
				}
				return s.LastMessage
			}
		}
	}

	return "Unknown item."
}

// ── Item lookup ─────────────────────────────────────────────────────

// findTool returns a pointer to the tool with the given ID, or nil.
func (s *State) findTool(id string) *Item {
	for i := range s.Tools {
		if s.Tools[i].ID == id {
			return &s.Tools[i]
		}
	}
	return nil
}

// findRole returns a pointer to the role with the given ID, or nil.
func (s *State) findRole(id string) *Item {
	for i := range s.Roles {
		if s.Roles[i].ID == id {
			return &s.Roles[i]
		}
	}
	return nil
}

// ActiveToolCount returns how many tools are currently active.
func (s *State) ActiveToolCount() int {
	n := 0
	for _, t := range s.Tools {
		if t.Active {
			n++
		}
	}
	return n
}

// ActiveRoleCount returns how many roles are currently active.
func (s *State) ActiveRoleCount() int {
	n := 0
	for _, r := range s.Roles {
		if r.Active {
			n++
		}
	}
	return n
}

// ActiveToolLabels returns the labels of active tools.
func (s *State) ActiveToolLabels() []string {
	var labels []string
	for _, t := range s.Tools {
		if t.Active {
			labels = append(labels, t.Label)
		}
	}
	return labels
}

// ActiveRoleLabels returns the labels of active roles.
func (s *State) ActiveRoleLabels() []string {
	var labels []string
	for _, r := range s.Roles {
		if r.Active {
			labels = append(labels, r.Label)
		}
	}
	return labels
}
