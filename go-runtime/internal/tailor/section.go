package tailor

import "fmt"

// ── Row ─────────────────────────────────────────────────────────────

// Row is a display-ready row for rendering in any UI.
// Types: "section", "item", "gap", "action".
type Row struct {
	Type  string `json:"type"`
	Group string `json:"group,omitempty"` // "plans", "tools", "roles"
	Kind  string `json:"kind,omitempty"`  // "plan", "tool", "role"
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Note  string `json:"note,omitempty"`

	Active       bool   `json:"active"`
	Allowed      bool   `json:"allowed"`
	RequiredPlan string `json:"required_label,omitempty"`
	DetectedNote string `json:"detected_note,omitempty"`
}

// ── SectionedRows ───────────────────────────────────────────────────

// SectionedRows returns the full list of display rows: section headers,
// items, gaps, and the final install action.
func (s *State) SectionedRows() []Row {
	var rows []Row

	sections := []struct {
		group string
		title string
		kind  string
	}{
		{"plans", "Plan", "plan"},
		{"tools", "Tools", "tool"},
		{"roles", "Roles", "role"},
	}

	for _, sec := range sections {
		rows = append(rows, Row{Type: "section", Label: sec.title})

		switch sec.group {
		case "plans":
			for _, p := range s.Plans {
				rows = append(rows, Row{
					Type:    "item",
					Group:   "plans",
					Kind:    "plan",
					ID:      p.ID,
					Label:   p.Label,
					Note:    p.Tagline,
					Active:  p.Active,
					Allowed: true,
				})
			}
		case "tools":
			for _, t := range s.Tools {
				allowed := s.IsAllowed(t.MinPlan)
				if !allowed {
					continue // hidden when not allowed
				}
				rows = append(rows, Row{
					Type:         "item",
					Group:        "tools",
					Kind:         "tool",
					ID:           t.ID,
					Label:        t.Label,
					Note:         t.Note,
					Active:       t.Active,
					Allowed:      allowed,
					RequiredPlan: PlanLabel(t.MinPlan),
					DetectedNote: t.DetectedNote,
				})
			}
		case "roles":
			for _, r := range s.Roles {
				allowed := s.IsAllowed(r.MinPlan)
				if !allowed {
					continue
				}
				rows = append(rows, Row{
					Type:         "item",
					Group:        "roles",
					Kind:         "role",
					ID:           r.ID,
					Label:        r.Label,
					Note:         r.Note,
					Active:       r.Active,
					Allowed:      allowed,
					RequiredPlan: PlanLabel(r.MinPlan),
				})
			}
		}
		rows = append(rows, Row{Type: "gap"})
	}

	// Install action
	rows = append(rows, Row{
		Type:    "action",
		Kind:    "install",
		ID:      "install",
		Label:   "Install OVAV",
		Note:    s.InstallSummary(),
		Allowed: s.SelectedPlan != "",
	})

	return rows
}

// ── SelectableRows ──────────────────────────────────────────────────

// SelectableRows returns only the interactive rows (items + actions),
// flattening out sections and gaps. Indices from this list are used
// by ToggleAt and RowHint.
func (s *State) SelectableRows() []Row {
	var rows []Row
	for _, r := range s.SectionedRows() {
		if r.Type == "item" || r.Type == "action" {
			rows = append(rows, r)
		}
	}
	return rows
}

// ── RowHint ─────────────────────────────────────────────────────────

// RowHint returns contextual help text for the selectable row at index.
func (s *State) RowHint(index int) string {
	rows := s.SelectableRows()
	if len(rows) == 0 {
		return "No options available."
	}
	row := rows[index%len(rows)]

	switch row.Type {
	case "action":
		if s.SelectedPlan != "" {
			return s.InstallSummary()
		}
		return "Choose a plan before installing."
	}

	switch row.Kind {
	case "plan":
		return fmt.Sprintf("Plan %s: %s", row.Label, row.Note)
	}

	if s.SelectedPlan == "" {
		return "Choose a plan to see what you can enable."
	}
	if !row.Allowed {
		return fmt.Sprintf("Blocked: requires %s.", row.RequiredPlan)
	}

	suffix := row.RequiredPlan
	if row.Kind == "tool" {
		suffix = row.DetectedNote
	}
	return fmt.Sprintf("%s · %s · %s", row.Label, row.Note, suffix)
}

// ── SelectableCount ─────────────────────────────────────────────────

// SelectableCount returns the number of selectable rows.
func (s *State) SelectableCount() int {
	return len(s.SelectableRows())
}
