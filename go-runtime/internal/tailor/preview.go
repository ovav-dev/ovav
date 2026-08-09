package tailor

// ── Change ──────────────────────────────────────────────────────────

// Change represents a diff between the current state and the last applied snapshot.
type Change struct {
	Label   string `json:"label"`
	Summary string `json:"summary"`
	After   bool   `json:"after"` // true = adding, false = removing
}

// ── PreviewChanges ──────────────────────────────────────────────────

// PreviewChanges returns a list of changes that will be applied
// compared to the last Applied snapshot.
// Returns nil if nothing has been applied yet (initial state).
func (s *State) PreviewChanges() []Change {
	if s.Applied == nil {
		return nil
	}

	applied := s.Applied
	current := s.Snapshot()
	var changes []Change

	// Plan change
	if applied.SelectedPlan != current.SelectedPlan {
		changes = append(changes, Change{
			Label:   "Plan",
			Summary: PlanLabel(applied.SelectedPlan) + " → " + PlanLabel(current.SelectedPlan),
			After:   current.SelectedPlan != "",
		})
	}

	// Tool changes
	for _, tool := range s.Tools {
		before := applied.Tools[tool.ID]
		after := current.Tools[tool.ID]
		if before != after {
			action := "remove"
			if after {
				action = "include"
			}
			changes = append(changes, Change{
				Label:   "Tool · " + tool.Label,
				Summary: action + " " + tool.Label,
				After:   after,
			})
		}
	}

	// Role changes
	for _, role := range s.Roles {
		before := applied.Roles[role.ID]
		after := current.Roles[role.ID]
		if before != after {
			action := "remove"
			if after {
				action = "include"
			}
			changes = append(changes, Change{
				Label:   "Role · " + role.Label,
				Summary: action + " " + role.Label,
				After:   after,
			})
		}
	}

	return changes
}

// HasChanges returns true if there are pending changes vs the applied snapshot.
func (s *State) HasChanges() bool {
	changes := s.PreviewChanges()
	return len(changes) > 0
}

// ── ResultRow ───────────────────────────────────────────────────────

// ResultRow is a structured row for rendering apply/preview results.
// Priority: 1=critical, 2=info, 3=success, 5=low.
type ResultRow struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Priority int    `json:"priority"`
}

// PreviewResultRows renders PreviewChanges as display-ready rows.
func (s *State) PreviewResultRows() []ResultRow {
	changes := s.PreviewChanges()
	if s.SelectedPlan == "" {
		return []ResultRow{{Label: "Plan", Value: "choose a plan to continue", Priority: 1}}
	}
	if changes == nil || len(changes) == 0 {
		return []ResultRow{
			{Label: "Changes", Value: "no pending changes", Priority: 5},
			{Label: "Plan", Value: PlanLabel(s.SelectedPlan), Priority: 3},
		}
	}
	rows := make([]ResultRow, 0, len(changes))
	for _, ch := range changes {
		pri := 3
		if !ch.After {
			pri = 5
		}
		rows = append(rows, ResultRow{Label: "Change", Value: ch.Summary, Priority: pri})
	}
	// Limit to 10
	if len(rows) > 10 {
		rows = rows[:10]
	}
	return rows
}
