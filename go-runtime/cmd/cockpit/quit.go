package main

// ── Quit ───────────────────────────────────────────────────────────

func (m Model) renderQuit() string {
	return renderTitleBar("Exit") +
		"\n  Close OVAV Cockpit?\n\n" +
		"  [ Enter ]  Confirm\n" +
		"  [ Esc  ]  Cancel\n\n" +
		renderHelpBar("Enter: Quit  •  Esc: Go back  •  ?: Help")
}
