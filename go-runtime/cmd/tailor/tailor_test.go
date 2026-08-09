package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ovav/ovav/internal/tailor"
)

// ── toggleByID ──────────────────────────────────────────────────────

func TestToggleByID_TogglesActiveState(t *testing.T) {
	s := tailor.NewState(nil)

	var targetID string
	for _, tool := range s.Tools {
		if tool.Active && s.IsAllowed(tool.MinPlan) {
			targetID = tool.ID
			break
		}
	}
	if targetID == "" {
		t.Skip("no active+allowed tool found to toggle")
	}

	wasActive := false
	for _, tool := range s.Tools {
		if tool.ID == targetID {
			wasActive = tool.Active
			break
		}
	}

	toggleByID(s, targetID, io.Discard)

	for _, tool := range s.Tools {
		if tool.ID == targetID {
			if tool.Active == wasActive {
				t.Errorf("toggleByID(%q) did not change Active state (was %v, still %v)",
					targetID, wasActive, tool.Active)
			}
			return
		}
	}
	t.Errorf("tool %q not found after toggle", targetID)
}

func TestToggleByID_NotFoundDoesNotPanic(t *testing.T) {
	s := tailor.NewState(nil)
	toggleByID(s, "nonexistent-item-xyz", io.Discard)
}

func TestToggleByID_NumericIndex(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) < 2 {
		t.Skip("need at least 2 selectable rows")
	}
	toggleByID(s, "1", io.Discard)
	toggleByID(s, "2", io.Discard)
}

func TestToggleByID_OutOfRangeIndex(t *testing.T) {
	s := tailor.NewState(nil)
	toggleByID(s, "99999", io.Discard)
}

func TestToggleByID_GatedToolMessage(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	var gatedID string
	for _, tool := range s.Tools {
		if tool.MinPlan != "" && tool.MinPlan != "nucleo" && !s.IsAllowed(tool.MinPlan) {
			gatedID = tool.ID
			break
		}
	}
	if gatedID == "" {
		t.Skip("no gated tool in nucleo plan")
	}
	toggleByID(s, gatedID, io.Discard)
}

func TestToggleByID_GatedRoleMessage(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	var gatedID string
	for _, role := range s.Roles {
		if role.MinPlan != "" && role.MinPlan != "nucleo" && !s.IsAllowed(role.MinPlan) {
			gatedID = role.ID
			break
		}
	}
	if gatedID == "" {
		t.Skip("no gated role in nucleo plan")
	}
	var buf bytes.Buffer
	toggleByID(s, gatedID, &buf)
	out := buf.String()
	if !strings.Contains(out, "requires plan") {
		t.Errorf("expected 'requires plan' message for gated role, got %q", out)
	}
}

func TestToggleByID_GatedToolMessage_WithOutput(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	var gatedID string
	for _, tool := range s.Tools {
		if tool.MinPlan != "" && tool.MinPlan != "nucleo" && !s.IsAllowed(tool.MinPlan) {
			gatedID = tool.ID
			break
		}
	}
	if gatedID == "" {
		t.Skip("no gated tool in nucleo plan")
	}
	var buf bytes.Buffer
	toggleByID(s, gatedID, &buf)
	out := buf.String()
	if !strings.Contains(out, "requires plan") {
		t.Errorf("expected 'requires plan' message, got %q", out)
	}
}

func TestToggleByID_NonNumeric_InvalidID(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	toggleByID(s, "abc123", &buf)
	out := buf.String()
	if !strings.Contains(out, "Item not found") {
		t.Errorf("expected 'Item not found', got %q", out)
	}
}

// ── printStatus ─────────────────────────────────────────────────────

func TestPrintStatus_DoesNotPanic(t *testing.T) {
	s := tailor.NewState(nil)
	printStatus(io.Discard, s)
}

func TestPrintStatus_WithPlanGating(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	printStatus(io.Discard, s)
}

func TestPrintStatus_WithLastMessage(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) > 0 {
		s.ToggleAt(0)
	}
	printStatus(io.Discard, s)
}

func TestPrintStatus_Output(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "Plan:") {
		t.Error("expected output to contain 'Plan:'")
	}
}

func TestPrintStatus_ToolDetected(t *testing.T) {
	s := tailor.NewState(map[string]bool{"opencode": true, "git": true})
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "[detected]") {
		t.Error("expected output to contain '[detected]' for detected tools")
	}
}

func TestPrintStatus_ToolNotDetected(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	// Non-detected tools should not show [detected]
	if strings.Contains(out, "[detected]") {
		t.Error("expected no '[detected]' in output for non-detected tools")
	}
}

func TestPrintStatus_RoleAllowed(t *testing.T) {
	s := tailor.NewState(nil)
	// nucleo allows platform_engineering (min_plan=nucleo)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "Platform Engineering") {
		t.Error("expected 'Platform Engineering' in output")
	}
}

func TestPrintStatus_RoleGated(t *testing.T) {
	s := tailor.NewState(nil)
	// nucleo does NOT allow security_architecture (min_plan=command)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "requires Command") {
		t.Errorf("expected 'requires Command' for gated role, got:\n%s", out)
	}
}

func TestPrintStatus_ToolGated(t *testing.T) {
	s := tailor.NewState(nil)
	// nucleo does NOT allow nvim (min_plan=studio)
	if err := s.SelectPlan("nucleo"); err != nil {
		t.Fatalf("SelectPlan(nucleo): %v", err)
	}
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "requires Studio") {
		t.Errorf("expected 'requires Studio' for gated tool, got:\n%s", out)
	}
}

func TestPrintStatus_ToolAllowedNotDetected(t *testing.T) {
	// Default state with no detections — all nucleo tools are allowed but not detected
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	// OpenCode is nucleo-min, active by default — should show without [detected]
	if !strings.Contains(out, "OpenCode") {
		t.Error("expected 'OpenCode' in output")
	}
}

func TestPrintStatus_WithLastMessageContent(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("studio"); err != nil {
		t.Fatalf("SelectPlan(studio): %v", err)
	}
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "Studio selected") {
		t.Errorf("expected last message about plan selection, got:\n%s", out)
	}
}

// ── printPreview ────────────────────────────────────────────────────

func TestPrintPreview_NoChanges(t *testing.T) {
	s := tailor.NewState(nil)
	changes := s.PreviewChanges()
	if len(changes) != 0 {
		t.Errorf("expected 0 pending changes on fresh state, got %d", len(changes))
	}
}

func TestPrintPreview_WithChanges(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	s.ToggleAt(0)
	changes := s.PreviewChanges()
	if len(changes) == 0 {
		t.Skip("no pending changes")
	}
	printPreview(io.Discard, s)
}

func TestPrintPreview_NilChanges(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	printPreview(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "No pending changes.") {
		t.Errorf("expected 'No pending changes.' for nil changes, got %q", out)
	}
}

func TestPrintPreview_RemovingChange(t *testing.T) {
	s := tailor.NewState(nil)
	// Find an inactive item, activate it
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	for i, row := range rows {
		if row.Type == "item" && row.Kind != "plan" && !row.Active {
			s.ToggleAt(i) // activate it
			break
		}
	}
	s.ApplySelection() // snapshot with the item active
	// Re-query rows and find an active item to deactivate
	rows2 := s.SelectableRows()
	for i, row := range rows2 {
		if row.Type == "item" && row.Kind != "plan" && row.Active {
			s.ToggleAt(i) // deactivate it
			break
		}
	}
	changes := s.PreviewChanges()
	if len(changes) == 0 {
		t.Skip("no removal changes")
	}
	var buf bytes.Buffer
	printPreview(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "- ") {
		t.Errorf("expected removal marker '- ' in preview, got:\n%s", out)
	}
}

func TestPrintPreview_AddingChange(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	s.ApplySelection()
	// Toggle an inactive item on
	for i, row := range rows {
		if row.Type == "item" && !row.Active {
			s.ToggleAt(i)
			break
		}
	}
	changes := s.PreviewChanges()
	if len(changes) == 0 {
		t.Skip("no adding changes")
	}
	var buf bytes.Buffer
	printPreview(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "+ ") {
		t.Errorf("expected adding marker '+ ' in preview, got:\n%s", out)
	}
}

func TestPrintPreview_Output(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	s.ToggleAt(0)
	var buf bytes.Buffer
	printPreview(&buf, s)
	out := buf.String()
	if out == "" {
		t.Error("expected non-empty preview output")
	}
}

// ── printResults ────────────────────────────────────────────────────

func TestPrintResults_DoesNotPanic(t *testing.T) {
	printResults(io.Discard, []tailor.ResultRow{
		{Label: "Plan", Value: "Studio"},
		{Label: "Tools", Value: "5"},
	})
}

func TestPrintResults_Empty(t *testing.T) {
	printResults(io.Discard, nil)
	printResults(io.Discard, []tailor.ResultRow{})
}

func TestPrintResults_Output(t *testing.T) {
	var buf bytes.Buffer
	printResults(&buf, []tailor.ResultRow{
		{Label: "Plan", Value: "Studio"},
		{Label: "Tools", Value: "5"},
	})
	out := buf.String()
	if !strings.Contains(out, "Studio") {
		t.Error("expected output to contain 'Studio'")
	}
}

// ── printHelp ───────────────────────────────────────────────────────

func TestPrintHelp_DoesNotPanic(t *testing.T) {
	printHelp(io.Discard)
}

func TestPrintHelp_Output(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf)
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Error("expected output to contain 'Usage:'")
	}
}

// ── dispatch ────────────────────────────────────────────────────────

func TestDispatch_Status(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"status"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(buf.String(), "Plan:") {
		t.Error("expected output to contain 'Plan:'")
	}
}

func TestDispatch_NoArgs(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, nil, &buf)
	if err == nil {
		t.Error("expected error for no args")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestDispatch_EmptyArgs(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{}, &buf)
	if err == nil {
		t.Error("expected error for empty args")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestDispatch_SelectPlan(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"select", "studio"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if s.SelectedPlan != "studio" {
		t.Errorf("expected plan 'studio', got %q", s.SelectedPlan)
	}
}

func TestDispatch_PlanAlias(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"plan", "command"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if s.SelectedPlan != "command" {
		t.Errorf("expected plan 'command', got %q", s.SelectedPlan)
	}
}

func TestDispatch_SelectNoPlanArg(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"select"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("expected exit code 1 for missing plan arg, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Error("expected usage message")
	}
}

func TestDispatch_PlanNoArg(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"plan"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestDispatch_SelectInvalidPlan(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"select", "nonexistent"}, &buf)
	if err == nil {
		t.Error("expected error for invalid plan")
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestDispatch_Toggle(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	var targetID string
	for _, r := range rows {
		if r.Type == "item" {
			targetID = r.ID
			break
		}
	}
	if targetID == "" {
		t.Skip("no toggleable item found")
	}
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"toggle", targetID}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestDispatch_ToggleNoArg(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"toggle"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("expected exit code 1 for missing toggle arg, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Error("expected usage message")
	}
}

func TestDispatch_Help(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"help"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Error("expected output to contain 'Usage:'")
	}
}

func TestDispatch_HelpShort(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"-h"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestDispatch_HelpLong(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"--help"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestDispatch_Preview(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"preview"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestDispatch_Apply(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"apply"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestDispatch_Unknown(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"unknown"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Errorf("expected exit code 1 for unknown command, got %d", code)
	}
	if !strings.Contains(buf.String(), "Unknown command:") {
		t.Error("expected output to contain 'Unknown command:'")
	}
}

// ── handleLine ──────────────────────────────────────────────────────

func TestHandleLine_Status(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "status", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected handleLine to return true (continue)")
	}
	if !strings.Contains(buf.String(), "Plan:") {
		t.Error("expected output to contain 'Plan:'")
	}
}

func TestHandleLine_StatusAlias_S(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "s", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true for 's'")
	}
	if !strings.Contains(buf.String(), "Plan:") {
		t.Error("expected Plan: in output")
	}
}

func TestHandleLine_Quit(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "quit", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cont {
		t.Error("expected handleLine to return false (quit)")
	}
	if !strings.Contains(buf.String(), "Goodbye") {
		t.Error("expected output to contain 'Goodbye'")
	}
}

func TestHandleLine_QuitAlias_Q(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "q", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cont {
		t.Error("expected continue=false for 'q'")
	}
}

func TestHandleLine_QuitAlias_Exit(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "exit", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cont {
		t.Error("expected continue=false for 'exit'")
	}
}

func TestHandleLine_Empty(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected handleLine to return true for empty line")
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty line, got %q", buf.String())
	}
}

func TestHandleLine_PlanSelect(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "plan studio", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Studio selected") {
		t.Errorf("expected 'Studio selected', got %q", buf.String())
	}
}

func TestHandleLine_SelectPlan(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "select command", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Command selected") {
		t.Errorf("expected 'Command selected', got %q", buf.String())
	}
}

func TestHandleLine_PlanNoArg(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "plan", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("expected usage message, got %q", buf.String())
	}
}

func TestHandleLine_SelectNoArg(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "select", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("expected usage message, got %q", buf.String())
	}
}

func TestHandleLine_PlanInvalid(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "plan badplan", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true even on error")
	}
	if !strings.Contains(buf.String(), "Error:") {
		t.Errorf("expected error message, got %q", buf.String())
	}
}

func TestHandleLine_Toggle(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	var targetID string
	for _, r := range rows {
		if r.Type == "item" {
			targetID = r.ID
			break
		}
	}
	if targetID == "" {
		t.Skip("no toggleable item")
	}
	var buf bytes.Buffer
	cont, err := handleLine(s, "toggle "+targetID, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
}

func TestHandleLine_ToggleAlias_T(t *testing.T) {
	s := tailor.NewState(nil)
	rows := s.SelectableRows()
	if len(rows) == 0 {
		t.Skip("no selectable rows")
	}
	var targetID string
	for _, r := range rows {
		if r.Type == "item" {
			targetID = r.ID
			break
		}
	}
	if targetID == "" {
		t.Skip("no toggleable item")
	}
	var buf bytes.Buffer
	cont, err := handleLine(s, "t "+targetID, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
}

func TestHandleLine_ToggleNoArg(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "toggle", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Errorf("expected usage message, got %q", out)
	}
	if !strings.Contains(out, "Available items:") {
		t.Errorf("expected item listing, got %q", out)
	}
}

func TestHandleLine_ToggleNoArg_ListsItems(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	_, err := handleLine(s, "toggle", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// Should list available items
	if !strings.Contains(out, "[") {
		t.Errorf("expected bracketed items, got %q", out)
	}
}

func TestHandleLine_ToggleNoArg_ShowsActiveMark(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	_, err := handleLine(s, "toggle", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	// Should contain either [✓] or [ ] for items
	if !strings.Contains(out, "[") || !strings.Contains(out, "]") {
		t.Errorf("expected active marks in item listing, got %q", out)
	}
}

func TestHandleLine_Preview(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "preview", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "No pending changes.") {
		t.Errorf("expected 'No pending changes.' in output, got %q", buf.String())
	}
}

func TestHandleLine_PreviewAlias_P(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "p", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
}

func TestHandleLine_Apply(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "apply", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Configuration applied") {
		t.Errorf("expected 'Configuration applied', got %q", buf.String())
	}
}

func TestHandleLine_ApplyAlias_A(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "a", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Configuration applied") {
		t.Errorf("expected 'Configuration applied', got %q", buf.String())
	}
}

func TestHandleLine_Help(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "help", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	if !strings.Contains(buf.String(), "Commands:") {
		t.Errorf("expected 'Commands:', got %q", buf.String())
	}
}

func TestHandleLine_HelpAlias_H(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "h", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
}

func TestHandleLine_HelpAlias_QuestionMark(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "?", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
}

func TestHandleLine_Unknown(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "foobar", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true for unknown command")
	}
	out := buf.String()
	if !strings.Contains(out, "Unknown: foobar") {
		t.Errorf("expected 'Unknown: foobar', got %q", out)
	}
	if !strings.Contains(out, "type 'help'") {
		t.Errorf("expected help hint, got %q", out)
	}
}

func TestHandleLine_WhitespaceOnly(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer
	cont, err := handleLine(s, "   ", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true for whitespace-only line")
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

// ── Integration tests ───────────────────────────────────────────────

func TestDispatch_FullFlow(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer

	// Select plan
	code, err := dispatch(s, []string{"select", "studio"}, &buf)
	if err != nil || code != 0 {
		t.Fatalf("select studio: code=%d err=%v", code, err)
	}

	// Toggle a tool
	rows := s.SelectableRows()
	var toolID string
	for _, r := range rows {
		if r.Type == "item" && r.Kind == "tool" {
			toolID = r.ID
			break
		}
	}
	if toolID != "" {
		buf.Reset()
		code, err = dispatch(s, []string{"toggle", toolID}, &buf)
		if err != nil || code != 0 {
			t.Fatalf("toggle %s: code=%d err=%v", toolID, code, err)
		}
	}

	// Preview
	buf.Reset()
	code, err = dispatch(s, []string{"preview"}, &buf)
	if err != nil || code != 0 {
		t.Fatalf("preview: code=%d err=%v", code, err)
	}

	// Apply
	buf.Reset()
	code, err = dispatch(s, []string{"apply"}, &buf)
	if err != nil || code != 0 {
		t.Fatalf("apply: code=%d err=%v", code, err)
	}

	// Status
	buf.Reset()
	code, err = dispatch(s, []string{"status"}, &buf)
	if err != nil || code != 0 {
		t.Fatalf("status: code=%d err=%v", code, err)
	}
}

func TestHandleLine_FullInteractiveFlow(t *testing.T) {
	s := tailor.NewState(nil)
	var buf bytes.Buffer

	// Select plan
	cont, err := handleLine(s, "plan studio", &buf)
	if err != nil || !cont {
		t.Fatalf("plan studio: cont=%v err=%v", cont, err)
	}

	// Toggle
	rows := s.SelectableRows()
	var toolID string
	for _, r := range rows {
		if r.Type == "item" && r.Kind == "tool" {
			toolID = r.ID
			break
		}
	}
	if toolID != "" {
		buf.Reset()
		cont, err = handleLine(s, "toggle "+toolID, &buf)
		if err != nil || !cont {
			t.Fatalf("toggle %s: cont=%v err=%v", toolID, cont, err)
		}
	}

	// Preview
	buf.Reset()
	cont, err = handleLine(s, "preview", &buf)
	if err != nil || !cont {
		t.Fatalf("preview: cont=%v err=%v", cont, err)
	}

	// Apply
	buf.Reset()
	cont, err = handleLine(s, "apply", &buf)
	if err != nil || !cont {
		t.Fatalf("apply: cont=%v err=%v", cont, err)
	}

	// Quit
	buf.Reset()
	cont, err = handleLine(s, "quit", &buf)
	if err != nil {
		t.Fatalf("quit: err=%v", err)
	}
	if cont {
		t.Error("expected quit to return false")
	}
}

func TestDispatch_PreviewWithChanges(t *testing.T) {
	s := tailor.NewState(nil)
	s.ApplySelection() // create snapshot
	// Toggle first selectable tool item
	rows := s.SelectableRows()
	for i, r := range rows {
		if r.Type == "item" && r.Kind == "tool" {
			s.ToggleAt(i)
			break
		}
	}
	var buf bytes.Buffer
	code, err := dispatch(s, []string{"preview"}, &buf)
	if err != nil || code != 0 {
		t.Fatalf("preview: code=%d err=%v", code, err)
	}
	if !strings.Contains(buf.String(), "Pending changes:") {
		t.Errorf("expected 'Pending changes:', got %q", buf.String())
	}
}

func TestHandleLine_PreviewWithChanges(t *testing.T) {
	s := tailor.NewState(nil)
	s.ApplySelection() // create snapshot
	// Toggle first selectable item
	rows := s.SelectableRows()
	for i, r := range rows {
		if r.Type == "item" && r.Kind == "tool" {
			s.ToggleAt(i)
			break
		}
	}
	var buf bytes.Buffer
	cont, err := handleLine(s, "preview", &buf)
	if err != nil || !cont {
		t.Fatalf("preview: cont=%v err=%v", cont, err)
	}
	if !strings.Contains(buf.String(), "Pending changes:") {
		t.Errorf("expected 'Pending changes:', got %q", buf.String())
	}
}

func TestPrintStatus_ActiveRoleAllowed(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("command"); err != nil {
		t.Fatalf("SelectPlan(command): %v", err)
	}
	// Activate a role that is allowed by command plan
	for i, role := range s.Roles {
		if s.IsAllowed(role.MinPlan) {
			s.Roles[i].Active = true
			break
		}
	}
	var buf bytes.Buffer
	printStatus(&buf, s)
	out := buf.String()
	if !strings.Contains(out, "✓") {
		t.Errorf("expected '✓' marker for active role, got:\n%s", out)
	}
}

func TestHandleLine_ToggleListing_ActiveItem(t *testing.T) {
	s := tailor.NewState(nil)
	if err := s.SelectPlan("command"); err != nil {
		t.Fatalf("SelectPlan(command): %v", err)
	}
	// Activate a tool so the listing shows [✓]
	for i, tool := range s.Tools {
		if s.IsAllowed(tool.MinPlan) {
			s.Tools[i].Active = true
			break
		}
	}
	var buf bytes.Buffer
	cont, err := handleLine(s, "toggle", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cont {
		t.Error("expected continue=true")
	}
	out := buf.String()
	if !strings.Contains(out, "✓") {
		t.Errorf("expected '✓' mark in toggle listing, got:\n%s", out)
	}
}

func TestHandleLine_ApplyWithChanges(t *testing.T) {
	s := tailor.NewState(nil)
	s.ApplySelection() // snapshot initial
	for i, tool := range s.Tools {
		if tool.Active {
			s.ToggleAt(i)
			break
		}
	}
	var buf bytes.Buffer
	cont, err := handleLine(s, "apply", &buf)
	if err != nil || !cont {
		t.Fatalf("apply: cont=%v err=%v", cont, err)
	}
	if !strings.Contains(buf.String(), "Configuration applied") {
		t.Errorf("expected 'Configuration applied', got %q", buf.String())
	}
}
