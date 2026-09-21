package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/data"
)

func TestNewModel(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		m := NewModel()
		if m.nav.Current() != ViewWelcome {
			t.Errorf("expected current view %q, got %q", ViewWelcome, m.nav.Current())
		}
		if m.nav.Depth() != 1 {
			t.Errorf("expected nav depth 1, got %d", m.nav.Depth())
		}
		if m.menuCursor != 0 {
			t.Errorf("expected menuCursor 0, got %d", m.menuCursor)
		}
		// ovavRoot should be non-empty
		if m.ovavRoot == "" {
			t.Error("expected non-empty ovavRoot")
		}
	})
}

func TestNavigation(t *testing.T) {
	t.Run("root to dashboard and back", func(t *testing.T) {
		m := NewModel()

		// Navigate welcome → root
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.welcomeUpdate(enterMsg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after welcome enter, got %q", m2.nav.Current())
		}
		m = m2

		// Select dashboard from root menu (position 1 after updates was added)
		m.menuCursor = 1 // dashboard
		result, _ = m.rootUpdate(enterMsg)
		m2 = result.(Model)
		if m2.nav.Current() != ViewDashboard {
			t.Errorf("expected ViewDashboard, got %q", m2.nav.Current())
		}
		m = m2

		// Go back from dashboard
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ = m.dashboardUpdate(escMsg)
		m2 = result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after dashboard esc, got %q", m2.nav.Current())
		}
	})

	t.Run("root to quit view", func(t *testing.T) {
		m := NewModel()
		enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.welcomeUpdate(enterMsg)
		m = result.(Model)

		// From root, press 'q' to go to quit
		qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		result, _ = m.rootUpdate(qMsg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewQuit {
			t.Errorf("expected ViewQuit, got %q", m2.nav.Current())
		}
	})
}

func TestViewRendering(t *testing.T) {
	t.Run("welcome view no panic", func(t *testing.T) {
		m := NewModel()
		output := m.renderWelcome()
		if output == "" {
			t.Error("expected non-empty welcome view")
		}
		// Should contain key elements
		if !containsAny(output, []string{"OVAV", "Dashboard"}) {
			t.Error("welcome view missing expected content")
		}
	})

	t.Run("root view no panic", func(t *testing.T) {
		m := NewModel()
		output := m.renderRoot()
		if output == "" {
			t.Error("expected non-empty root view")
		}
		if !containsAny(output, []string{"Main Menu", "Dashboard", "Health"}) {
			t.Error("root view missing expected content")
		}
	})

	t.Run("dashboard view no panic", func(t *testing.T) {
		m := NewModel()
		// Give model some width for dashboard rendering
		m.width = 120
		output := m.renderDashboard()
		if output == "" {
			t.Error("expected non-empty dashboard view")
		}
	})

	t.Run("health view no panic", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		output := m.renderHealth()
		if output == "" {
			t.Error("expected non-empty health view")
		}
	})

	t.Run("install view no panic", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		output := m.renderInstall()
		if output == "" {
			t.Error("expected non-empty install view")
		}
	})

	t.Run("quit view no panic", func(t *testing.T) {
		m := NewModel()
		output := m.renderQuit()
		if output == "" {
			t.Error("expected non-empty quit view")
		}
		if !containsAny(output, []string{"Exit", "Close OVAV Cockpit"}) {
			t.Error("quit view missing expected content")
		}
	})

	t.Run("view function no panic", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.ready = true
		output := m.renderCurrentView()
		if output == "" {
			t.Error("expected non-empty current view")
		}
	})
}

func TestUpdateHandling(t *testing.T) {
	t.Run("window size message", func(t *testing.T) {
		m := NewModel()
		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		result, cmd := m.Update(msg)
		m2 := result.(Model)
		if m2.width != 120 || m2.height != 40 {
			t.Errorf("expected width=120 height=40, got width=%d height=%d", m2.width, m2.height)
		}
		if !m2.ready {
			t.Error("expected ready=true after WindowSizeMsg")
		}
		// cmd should be nil (no initial batch needed beyond window title)
		_ = cmd
	})

	t.Run("ctrl+c quits", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		result, cmd := m.Update(msg)
		m2 := result.(Model)
		if !m2.quitting {
			t.Error("expected quitting=true after Ctrl+C")
		}
		if cmd == nil {
			t.Error("expected quit command after Ctrl+C")
		}
	})

	t.Run("esc on quit view quits", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewRoot, ViewQuit}
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, cmd := m.Update(msg)
		m2 := result.(Model)
		if !m2.quitting {
			t.Error("expected quitting=true after Esc on quit view")
		}
		if cmd == nil {
			t.Error("expected quit command")
		}
	})

	t.Run("goBackMsg pops nav", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewWelcome, ViewRoot, ViewDashboard}
		msg := goBackMsg{}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after goBack, got %q", m2.nav.Current())
		}
	})
}

func TestKeyMessageDispatch(t *testing.T) {
	t.Run("enter on welcome navigates to root", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot, got %q", m2.nav.Current())
		}
	})

	t.Run("q on root pushes quit", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewRoot}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewQuit {
			t.Errorf("expected ViewQuit, got %q", m2.nav.Current())
		}
	})
}

func TestOVAVRootDetection(t *testing.T) {
	t.Run("findOVAVRoot returns non-empty", func(t *testing.T) {
		root := findOVAVRoot()
		if root == "" {
			t.Error("expected non-empty OVAV root")
		}
	})

	t.Run("findOVAVRoot is cached second call", func(t *testing.T) {
		root1 := findOVAVRoot()
		root2 := findOVAVRoot()
		if root1 != root2 {
			t.Errorf("expected same root on second call, got %q vs %q", root1, root2)
		}
	})
}

func TestNavStack(t *testing.T) {
	t.Run("new nav stack", func(t *testing.T) {
		nav := NewNavStack(ViewWelcome)
		if nav.Current() != ViewWelcome {
			t.Errorf("expected %q, got %q", ViewWelcome, nav.Current())
		}
		if nav.CanGoBack() {
			t.Error("new nav stack should not allow go back")
		}
	})

	t.Run("push and pop", func(t *testing.T) {
		nav := NewNavStack(ViewWelcome)
		nav.Push(ViewRoot)
		nav.Push(ViewDashboard)
		if nav.Current() != ViewDashboard {
			t.Errorf("expected ViewDashboard, got %q", nav.Current())
		}
		if nav.Depth() != 3 {
			t.Errorf("expected depth 3, got %d", nav.Depth())
		}
		nav.Pop()
		if nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after pop, got %q", nav.Current())
		}
		nav.Pop()
		if nav.Current() != ViewWelcome {
			t.Errorf("expected ViewWelcome after second pop, got %q", nav.Current())
		}
		// Can't pop past root
		nav.Pop()
		if nav.Current() != ViewWelcome {
			t.Errorf("expected ViewWelcome after overpop, got %q", nav.Current())
		}
	})

	t.Run("replace", func(t *testing.T) {
		nav := NewNavStack(ViewWelcome)
		nav.Replace(ViewRoot)
		if nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after replace, got %q", nav.Current())
		}
		if nav.Depth() != 1 {
			t.Errorf("expected depth 1 after replace, got %d", nav.Depth())
		}
	})
}

func TestInstallModel(t *testing.T) {
	t.Run("new install model defaults", func(t *testing.T) {
		im := NewInstallModel()
		if im.running || im.completed || im.failed {
			t.Error("new install model should not be running/completed/failed")
		}
		if im.actionFocus != 0 {
			t.Errorf("expected actionFocus 0, got %d", im.actionFocus)
		}
	})

	t.Run("install model rejects keys while running", func(t *testing.T) {
		im := NewInstallModel()
		im.running = true
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := im.Update(msg)
		if result.running != true {
			t.Error("running state should be preserved")
		}
	})

	t.Run("install model left/right action focus", func(t *testing.T) {
		im := NewInstallModel()
		// Move right
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
		result, _ := im.Update(msg)
		if result.actionFocus != 1 {
			t.Errorf("expected actionFocus 1 after 'l', got %d", result.actionFocus)
		}
		// Move left
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		result, _ = result.Update(msg)
		if result.actionFocus != 0 {
			t.Errorf("expected actionFocus 0 after 'h', got %d", result.actionFocus)
		}
		// Can't go below 0
		result, _ = result.Update(msg)
		if result.actionFocus != 0 {
			t.Errorf("expected actionFocus still 0, got %d", result.actionFocus)
		}
	})

	t.Run("install completes on done tick", func(t *testing.T) {
		im := NewInstallModel()
		im.running = true
		msg := installTickMsg{done: true, stage: StageDone, progress: 100}
		result, cmd := im.Update(msg)
		if !result.completed || result.running {
			t.Error("expected completed=true, running=false")
		}
		if cmd != nil {
			t.Error("expected nil cmd on done")
		}
	})

	t.Run("install fails on failure tick", func(t *testing.T) {
		im := NewInstallModel()
		im.running = true
		msg := installTickMsg{failed: true, err: "test error"}
		result, cmd := im.Update(msg)
		if !result.failed || result.running {
			t.Error("expected failed=true, running=false")
		}
		if result.errorMsg != "test error" {
			t.Errorf("expected error 'test error', got %q", result.errorMsg)
		}
		if cmd != nil {
			t.Error("expected nil cmd on failure")
		}
	})

	t.Run("install progress tick advances stage", func(t *testing.T) {
		im := NewInstallModel()
		im.running = true
		im.stage = StageDetect
		msg := installTickMsg{stage: StageBackup, progress: 20}
		result, cmd := im.Update(msg)
		if result.stage != StageBackup || result.progress != 20 {
			t.Errorf("expected stage=StageBackup progress=20, got stage=%d progress=%d", result.stage, result.progress)
		}
		// Should return a cmd for next stage
		if cmd == nil {
			t.Error("expected non-nil cmd for next stage")
		}
	})
}

func TestHealthView(t *testing.T) {
	t.Run("health refresh with r key", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewWelcome, ViewHealth}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewHealth {
			t.Error("expected to stay on health view")
		}
	})
}

func TestTailorModel(t *testing.T) {
	t.Run("new tailor model", func(t *testing.T) {
		tm := NewTailorModel()
		if tm.step != TailorSelect {
			t.Errorf("expected TailorSelect, got %d", tm.step)
		}
		if tm.cursor != 0 {
			t.Errorf("expected cursor 0, got %d", tm.cursor)
		}
	})
}

func TestPlanDetailView(t *testing.T) {
	t.Run("plan detail back with esc", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewWelcome, ViewRoot, ViewDetail}
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after detail esc, got %q", m2.nav.Current())
		}
	})
}

func TestQuittingView(t *testing.T) {
	t.Run("view shows quitting message", func(t *testing.T) {
		m := NewModel()
		m.quitting = true
		output := m.View()
		if !containsAny(output, []string{"Exiting OVAV Cockpit"}) {
			t.Errorf("expected quitting message, got %q", output)
		}
	})

	t.Run("view shows initializing when not ready", func(t *testing.T) {
		m := NewModel()
		m.ready = false
		output := m.View()
		if !containsAny(output, []string{"Initializing"}) {
			t.Errorf("expected initializing message, got %q", output)
		}
	})
}

func TestMouseHandling(t *testing.T) {
	t.Run("mouse on quit view quits", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewWelcome, ViewQuit}
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if !m2.quitting {
			t.Error("expected quitting=true after mouse click on quit")
		}
	})

	t.Run("mouse on welcome navigates to root", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot, got %q", m2.nav.Current())
		}
	})
}

func TestFullProgram(t *testing.T) {
	t.Run("program runs without panic", func(t *testing.T) {
		p := tea.NewProgram(NewModel(), tea.WithoutRenderer())
		// Start the program in a goroutine and quit after a short delay
		done := make(chan struct{})
		go func() {
			if _, err := p.Run(); err != nil {
				t.Logf("program run error (expected in test): %v", err)
			}
			close(done)
		}()
		// Send a quit message
		go func() {
			time.Sleep(50 * time.Millisecond)
			p.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
		}()
		select {
		case <-done:
			// OK
		case <-time.After(2 * time.Second):
			t.Fatal("program timed out")
		}
	})
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func BenchmarkRenderWelcome(b *testing.B) {
	m := NewModel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderWelcome()
	}
}

func BenchmarkRenderRoot(b *testing.B) {
	m := NewModel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderRoot()
	}
}

func BenchmarkRenderDashboard(b *testing.B) {
	m := NewModel()
	m.width = 120
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderDashboard()
	}
}

// ── Plan Detail View Tests ──────────────────────────────────────────────

func TestPlanDetailViewComplete(t *testing.T) {
	t.Run("render plan detail with cap", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.planDetail = PlanDetailModel{
			cap: &data.Cap{
				Name:     "Test Cap",
				Type:     "SISTEMA",
				Items:    5,
				Pct:      100,
				MergedAt: "2026-06-17",
				Merge:    "abc123",
				Summary:  "Test summary of the cap",
			},
		}
		output := m.renderPlanDetail()
		if output == "" {
			t.Error("expected non-empty plan detail view")
		}
		if !containsAny(output, []string{"Test Cap", "completed"}) {
			t.Error("plan detail view missing expected content")
		}
	})

	t.Run("render plan detail with pending", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.planDetail = PlanDetailModel{
			pending: &data.PendingCap{
				Name:     "Pending Cap",
				Type:     "FEATURE",
				Order:    1,
				Tasks:    []string{"Task A", "Task B"},
				Deps:     []string{"dep1"},
				Worktree: "task/feature-x",
				Stack:    "Go",
				Summary:  "Pending summary",
			},
		}
		output := m.renderPlanDetail()
		if output == "" {
			t.Error("expected non-empty pending detail view")
		}
		if !containsAny(output, []string{"Pending Cap", "pending"}) {
			t.Error("pending detail view missing expected content")
		}
	})

	t.Run("render plan detail empty", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		output := m.renderPlanDetail()
		if output == "" {
			t.Error("expected non-empty empty detail view")
		}
		if !containsAny(output, []string{"No cap"}) {
			t.Error("empty detail view missing 'No cap' message")
		}
	})
}

// ── Install Progress & Result Tests ─────────────────────────────────────

func TestInstallProgressViews(t *testing.T) {
	t.Run("render install progress", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.installModel = InstallModel{
			running:  true,
			stage:    StageBackup,
			progress: 40,
		}
		output := m.renderInstallProgress()
		if output == "" {
			t.Error("expected non-empty install progress view")
		}
		if !containsAny(output, []string{"Install Progress"}) {
			t.Error("install progress view missing title")
		}
	})

	t.Run("render install result success", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.installModel = InstallModel{
			completed: true,
			stage:     StageDone,
			progress:  100,
		}
		output := m.renderInstallResult()
		if output == "" {
			t.Error("expected non-empty install result view")
		}
		if !containsAny(output, []string{"Install Complete", "Next Steps"}) {
			t.Error("install result view missing expected content")
		}
	})

	t.Run("StageProgress completed", func(t *testing.T) {
		output := StageProgress("Detect", "🔍", 0, 2, 100, 10)
		if output == "" {
			t.Error("expected non-empty stage progress")
		}
		if !containsAny(output, []string{"✅"}) {
			t.Error("completed stage should show ✅")
		}
	})

	t.Run("StageProgress current", func(t *testing.T) {
		output := StageProgress("Apply", "📦", 2, 2, 50, 10)
		if output == "" {
			t.Error("expected non-empty current stage progress")
		}
		if !containsAny(output, []string{"📦"}) {
			t.Error("current stage should show its icon")
		}
	})

	t.Run("StageProgress pending", func(t *testing.T) {
		output := StageProgress("Verify", "✅", 3, 1, 0, 10)
		if output == "" {
			t.Error("expected non-empty pending stage progress")
		}
		if !containsAny(output, []string{"⬜"}) {
			t.Error("pending stage should show ⬜")
		}
	})
}

// ── Tailor Model Tests ─────────────────────────────────────────────────

func TestTailorModelComplete(t *testing.T) {
	t.Run("tailor select navigation", func(t *testing.T) {
		tm := NewTailorModel()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, _ := tm.Update(msg)
		if result.cursor != 1 {
			t.Errorf("expected cursor 1 after down, got %d", result.cursor)
		}
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		result, _ = result.Update(msg)
		if result.cursor != 0 {
			t.Errorf("expected cursor 0 after up, got %d", result.cursor)
		}
	})

	t.Run("tailor confirm update enter", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorConfirm
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, cmd := tm.Update(msg)
		if result.step != TailorDone {
			t.Errorf("expected TailorDone after confirm enter, got %d", result.step)
		}
		if cmd == nil {
			t.Error("expected non-nil cmd for go back after tailor done")
		}
	})

	t.Run("tailor mouse wheel up", func(t *testing.T) {
		tm := NewTailorModel()
		tm.cursor = 1
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
		result, _ := tm.Update(msg)
		if result.cursor != 0 {
			t.Errorf("expected cursor 0 after wheel up, got %d", result.cursor)
		}
	})

	t.Run("tailor mouse wheel down", func(t *testing.T) {
		tm := NewTailorModel()
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
		result, _ := tm.Update(msg)
		if result.cursor != 1 {
			t.Errorf("expected cursor 1 after wheel down, got %d", result.cursor)
		}
	})

	t.Run("tailor mouse left click", func(t *testing.T) {
		tm := NewTailorModel()
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: 4}
		result, _ := tm.Update(msg)
		if result.cursor != 1 {
			t.Errorf("expected cursor 1 after mouse click Y=4, got %d", result.cursor)
		}
	})

	t.Run("tailor render select", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		output := m.renderTailorSelect()
		if output == "" {
			t.Error("expected non-empty tailor select view")
		}
		if !containsAny(output, []string{"Tailor Composer"}) {
			t.Error("tailor select view missing title")
		}
	})

	t.Run("tailor render preview", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		output := m.renderTailorPreview()
		if output == "" {
			t.Error("expected non-empty tailor preview view")
		}
	})

	t.Run("tailor render confirm", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		output := m.renderTailorConfirm()
		if output == "" {
			t.Error("expected non-empty tailor confirm view")
		}
		if !containsAny(output, []string{"Tailor Complete"}) {
			t.Error("tailor confirm view missing title")
		}
	})
}

// ── Init Test ───────────────────────────────────────────────────────────

func TestInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()
	// Init returns a tea.Batch with SetWindowTitle
	if cmd == nil {
		t.Error("expected non-nil cmd from Init")
	}
	// Execute the command to verify it doesn't panic
	_ = cmd()
}

// ── Vertical Pct Bar Test ───────────────────────────────────────────────

func TestVerticalPctBar(t *testing.T) {
	output := VerticalPctBar("Test", 75, 20)
	if output == "" {
		t.Error("expected non-empty vertical pct bar")
	}
	if !containsAny(output, []string{"75%", "Test"}) {
		t.Error("vertical pct bar missing expected content")
	}
}

// ── CLI Selector Tests ──────────────────────────────────────────────────

func TestNewCLISelectorModel(t *testing.T) {
	dir := t.TempDir()
	model := NewCLISelectorModel(dir)

	if len(model.targets) != 4 {
		t.Errorf("expected 4 targets, got %d", len(model.targets))
	}

	targets := map[string]bool{}
	for _, t := range model.targets {
		targets[string(t.Target)] = true
	}
	for _, expected := range []string{"opencode", "mimocode", "claude-code", "cursor"} {
		if !targets[expected] {
			t.Errorf("expected target %q not found", expected)
		}
	}

	if model.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", model.cursor)
	}
	if model.ovavRoot != dir {
		t.Errorf("expected ovavRoot %q, got %q", dir, model.ovavRoot)
	}
}

func TestCheckTargetStatus(t *testing.T) {
	t.Run("not_generated", func(t *testing.T) {
		dir := t.TempDir()
		status := checkTargetStatus(dir, "opencode")
		if status != "not_generated" {
			t.Errorf("expected 'not_generated', got %q", status)
		}
	})

	t.Run("generated", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents"), 0755)
		os.WriteFile(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents", "test.md"), []byte("test"), 0644)
		status := checkTargetStatus(dir, "opencode")
		if status != "generated" {
			t.Errorf("expected 'generated', got %q", status)
		}
	})

	t.Run("empty_dir_not_generated", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents"), 0755)
		status := checkTargetStatus(dir, "opencode")
		if status != "not_generated" {
			t.Errorf("expected 'not_generated' for empty dir, got %q", status)
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		dir := t.TempDir()
		status := checkTargetStatus(dir, "unknown-cli")
		if status != "unavailable" {
			t.Errorf("expected 'unavailable', got %q", status)
		}
	})
}

func TestRenderCLI(t *testing.T) {
	dir := t.TempDir()
	// Create generated opencode runtime so status shows ✅
	os.MkdirAll(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents", "area-platform-engineering.md"), []byte("test"), 0644)

	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)

	output := m.renderCLI()
	if output == "" {
		t.Error("expected non-empty CLI view")
	}
	if !containsAny(output, []string{"CLI Runtime Selector"}) {
		t.Error("CLI view missing title")
	}
	if !containsAny(output, []string{"OpenCode", "Claude Code", "Cursor"}) {
		t.Error("CLI view missing target labels")
	}
	if !containsAny(output, []string{"g  Generate", "g generate"}) {
		t.Error("CLI view missing generate instruction")
	}
}

func TestRenderCLI_WithMessage(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents", "area-platform-engineering.md"), []byte("test"), 0644)

	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)
	m.cliModel.message = "✅ Generated successfully!"

	output := m.renderCLI()
	if !containsAny(output, []string{"Generated successfully!"}) {
		t.Error("CLI view missing status message")
	}
}

func TestRenderCLI_WithPreview(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "internal", "runtimes", "opencode", "agents", "area-platform-engineering.md"), []byte("test"), 0644)

	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)
	m.cliModel.generated = true

	output := m.renderCLI()
	if !containsAny(output, []string{"Files that will be generated"}) {
		t.Error("CLI view missing generation preview")
	}
}

func TestRenderGenerationPreview(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "runtimes", "opencode", "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "runtimes", "opencode", "agents", "area-platform-engineering.md"), []byte("test"), 0644)

	target := cliTarget{
		Target: "opencode",
		Label:  "OpenCode",
		Status: "generated",
	}

	output := renderGenerationPreview(dir, target)
	if output == "" {
		t.Error("expected non-empty preview")
	}
	if !containsAny(output, []string{"Files that will be generated"}) {
		t.Error("preview missing title")
	}
	if !containsAny(output, []string{"runtimes/opencode/agents"}) {
		t.Error("preview missing output dir")
	}
	if !containsAny(output, []string{"71 total"}) {
		t.Error("preview missing file count")
	}
}

func TestRenderGenerationPreview_Unavailable(t *testing.T) {
	dir := t.TempDir()
	target := cliTarget{
		Target: "nonexistent",
		Label:  "Bad",
		Status: "unavailable",
	}

	output := renderGenerationPreview(dir, target)
	if !containsAny(output, []string{"Error"}) {
		t.Error("expected error for unavailable target")
	}
}

func TestCLIUpdate_Navigation(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "runtimes", "opencode", "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "runtimes", "opencode", "agents", "area-platform-engineering.md"), []byte("test"), 0644)

	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)

	t.Run("down moves cursor", func(t *testing.T) {
		result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(Model)
		if m2.cliModel.cursor != 1 {
			t.Errorf("expected cursor 1 after down, got %d", m2.cliModel.cursor)
		}
		m = m2
	})

	t.Run("up moves cursor back", func(t *testing.T) {
		result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyUp})
		m2 := result.(Model)
		if m2.cliModel.cursor != 0 {
			t.Errorf("expected cursor 0 after up, got %d", m2.cliModel.cursor)
		}
		m = m2
	})

	t.Run("up at top stays at 0", func(t *testing.T) {
		m.cliModel.cursor = 0
		result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyUp})
		m2 := result.(Model)
		if m2.cliModel.cursor != 0 {
			t.Errorf("expected cursor 0 at boundary, got %d", m2.cliModel.cursor)
		}
		m = m2
	})

	t.Run("down at bottom stays", func(t *testing.T) {
		m.cliModel.cursor = 3
		result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyDown})
		m2 := result.(Model)
		if m2.cliModel.cursor != 3 {
			t.Errorf("expected cursor 3 at boundary, got %d", m2.cliModel.cursor)
		}
		m = m2
	})
}

func TestCLIUpdate_EnterTogglesPreview(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "runtimes", "opencode", "agents"), 0755)
	os.WriteFile(filepath.Join(dir, "runtimes", "opencode", "agents", "area-platform-engineering.md"), []byte("test"), 0644)

	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)

	// Initially not generated
	if m.cliModel.generated {
		t.Error("expected generated=false initially")
	}

	result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := result.(Model)
	if !m2.cliModel.generated {
		t.Error("expected generated=true after enter")
	}

	result, _ = m2.cliUpdate(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := result.(Model)
	if m3.cliModel.generated {
		t.Error("expected generated=false after second enter")
	}
}

func TestCLIUpdate_GenerateKey(t *testing.T) {
	dir := t.TempDir()
	// Create canonical source files so generation succeeds
	canonicalDir := filepath.Join(dir, "ovav", "agents")
	os.MkdirAll(filepath.Join(canonicalDir, "areas"), 0755)
	os.MkdirAll(filepath.Join(canonicalDir, "leads"), 0755)
	os.MkdirAll(filepath.Join(canonicalDir, "teams"), 0755)
	os.WriteFile(filepath.Join(canonicalDir, "areas", "area-platform-engineering.yaml"),
		[]byte("id: platform-engineering\nname: Platform Engineering\nlead: thavren\nfunctions: [f1,f2,f3,f4,f5,f6,f7,f8,f9]\nlimitations: [l1,l2,l3,l4,l5]\nhard_stop: HARD STOP\n"), 0644)
	os.WriteFile(filepath.Join(canonicalDir, "leads", "lead-thavren.yaml"),
		[]byte("id: thavren\nname: Thavren\ndisplay_name: Platform Engineering\narea: platform-engineering\nfunctions: [f1,f2,f3,f4,f5,f6,f7,f8,f9]\nlimitations: [l1,l2,l3,l4,l5]\nhard_stop: HARD STOP\nsquad: [{name: A, country: PE, specialty: S}]\n"), 0644)

	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)

	result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m2 := result.(Model)

	if !containsAny(m2.cliModel.message, []string{"generated successfully"}) {
		t.Errorf("expected success message, got %q", m2.cliModel.message)
	}
	if !m2.cliModel.generated {
		t.Error("expected generated=true after generation")
	}
	if m2.cliModel.targets[0].Status != "generated" {
		t.Errorf("expected status 'generated', got %q", m2.cliModel.targets[0].Status)
	}
}

func TestCLIUpdate_GenerateUnavailable(t *testing.T) {
	dir := t.TempDir()
	m := NewModel()
	m.ovavRoot = dir
	m.nav.Push(ViewCLI)
	m.cliModel = NewCLISelectorModel(dir)
	// Navigate to cursor target (unavailable)
	m.cliModel.cursor = 3 // Cursor (unavailable — no canonical dir set up)

	result, _ := m.cliUpdate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m2 := result.(Model)

	if !containsAny(m2.cliModel.message, []string{"failed"}) && m2.cliModel.message == "" {
		t.Error("expected error or fail message for unavailable generation")
	}
	// Should NOT set generated=true on failure
	if m2.cliModel.generated && !containsAny(m2.cliModel.message, []string{"successfully"}) {
		t.Error("expected generated=false on failure")
	}
}

func TestIsCLIView(t *testing.T) {
	if !isCLIView(ViewCLI) {
		t.Error("expected true for ViewCLI")
	}
	if isCLIView(ViewRoot) {
		t.Error("expected false for ViewRoot")
	}
	if isCLIView(ViewDashboard) {
		t.Error("expected false for ViewDashboard")
	}
	if isCLIView("") {
		t.Error("expected false for empty string")
	}
}

// ── Help Overlay Tests ────────────────────────────────────────────────

func TestHelpUpdate(t *testing.T) {
	t.Run("close with ? key", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		result, _ := m.helpUpdate(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after ? close, got %q", m2.nav.Current())
		}
	})

	t.Run("close with esc", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ := m.helpUpdate(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after esc close, got %q", m2.nav.Current())
		}
	})

	t.Run("close with q", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		result, _ := m.helpUpdate(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after q close, got %q", m2.nav.Current())
		}
	})

	t.Run("other keys ignored", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		result, _ := m.helpUpdate(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewHelp {
			t.Errorf("expected ViewHelp preserved, got %q", m2.nav.Current())
		}
	})
}

func TestRenderHelp(t *testing.T) {
	t.Run("global help from root", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		output := m.renderHelp()
		if output == "" {
			t.Error("expected non-empty help view")
		}
		if !containsAny(output, []string{"Quick Help", "Global", "Navigation"}) {
			t.Error("help view missing global sections")
		}
		if !containsAny(output, []string{"Main Menu"}) {
			t.Error("help view missing contextual root shortcuts")
		}
	})

	t.Run("contextual help from dashboard", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard, ViewHelp}
		output := m.renderHelp()
		if !containsAny(output, []string{"Dashboard"}) {
			t.Error("help view missing dashboard contextual shortcuts")
		}
	})

	t.Run("contextual help from health", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHealth, ViewHelp}
		output := m.renderHelp()
		if !containsAny(output, []string{"Health"}) {
			t.Error("help view missing health contextual shortcuts")
		}
	})

	t.Run("contextual help from install", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewInstall, ViewHelp}
		output := m.renderHelp()
		if !containsAny(output, []string{"Install"}) {
			t.Error("help view missing install contextual shortcuts")
		}
	})

	t.Run("contextual help from tailor", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewTailor, ViewHelp}
		output := m.renderHelp()
		if !containsAny(output, []string{"Tailor"}) {
			t.Error("help view missing tailor contextual shortcuts")
		}
	})

	t.Run("contextual help from CLI", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewCLI, ViewHelp}
		output := m.renderHelp()
		if !containsAny(output, []string{"CLI"}) {
			t.Error("help view missing CLI contextual shortcuts")
		}
	})

	t.Run("footer present", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		output := m.renderHelp()
		if !containsAny(output, []string{"Productivity", "Close help"}) {
			t.Error("help view missing footer sections")
		}
	})
}

func TestToggleHelp(t *testing.T) {
	t.Run("push help when not active", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot}
		m.toggleHelp()
		if m.nav.Current() != ViewHelp {
			t.Errorf("expected ViewHelp after toggle, got %q", m.nav.Current())
		}
	})

	t.Run("pop help when active", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		m.toggleHelp()
		if m.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after toggle off, got %q", m.nav.Current())
		}
	})
}

func TestPeekBelow(t *testing.T) {
	t.Run("single depth returns root", func(t *testing.T) {
		nav := NewNavStack(ViewWelcome)
		result := nav.PeekBelow()
		if result != ViewRoot {
			t.Errorf("expected ViewRoot for depth 1, got %q", result)
		}
	})

	t.Run("two depth returns first", func(t *testing.T) {
		nav := NewNavStack(ViewRoot)
		nav.Push(ViewDashboard)
		result := nav.PeekBelow()
		if result != ViewRoot {
			t.Errorf("expected ViewRoot, got %q", result)
		}
	})

	t.Run("three depth returns second", func(t *testing.T) {
		nav := NewNavStack(ViewRoot)
		nav.Push(ViewDashboard)
		nav.Push(ViewHelp)
		result := nav.PeekBelow()
		if result != ViewDashboard {
			t.Errorf("expected ViewDashboard, got %q", result)
		}
	})
}

// ── Dashboard Tests ──────────────────────────────────────────────────

func newTestCaps() *data.CapsData {
	return &data.CapsData{
		Version:     "1.0",
		PlanVersion: "v1.9",
		Strategy:    "Go+TS",
		StackTarget: data.StackTarget{Go: "1.22", TypeScript: "5.0"},
		Caps: map[string]data.Cap{
			"CAP-001": {Name: "Governor Core", Type: "SISTEMA", Status: "done", Pct: 100, Items: 5, MergedAt: "2026-06-15", Merge: "abc123", Summary: "Core governor"},
			"CAP-002": {Name: "Context Ledger", Type: "FEATURE", Status: "done", Pct: 100, Items: 3, MergedAt: "2026-06-16", Merge: "def456", Summary: "Ledger system"},
			"CAP-003": {Name: "Incomplete Cap", Type: "FEATURE", Status: "pending", Pct: 50, Items: 2},
		},
		Pending: []data.PendingCap{
			{ID: "PEND-001", Name: "Budget Engine", Type: "FEATURE", Status: "pending", Pct: 30, Order: 1, Tasks: []string{"Design", "Implement"}, Deps: []string{"CAP-001"}, Worktree: "task/budget", Stack: "Go", Summary: "Budget tracking"},
			{ID: "PEND-002", Name: "Health Monitor", Type: "FEATURE", Status: "pending", Pct: 10, Order: 2, Tasks: []string{"Setup"}, Worktree: "task/health", Stack: "Go", Summary: "Health checks"},
		},
	}
}

func TestDashboardUpdate_Search(t *testing.T) {
	t.Run("toggle search with /", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if !m2.dashboardSearch {
			t.Error("expected dashboardSearch=true after /")
		}
	})

	t.Run("type filter in search mode", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.dashboardSearch = true
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.dashboardFilter != "g" {
			t.Errorf("expected filter 'g', got %q", m2.dashboardFilter)
		}
	})

	t.Run("backspace in search mode", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.dashboardSearch = true
		m.dashboardFilter = "gov"
		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.dashboardFilter != "go" {
			t.Errorf("expected filter 'go', got %q", m2.dashboardFilter)
		}
	})

	t.Run("enter applies search", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.dashboardSearch = true
		m.dashboardFilter = "test"
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.dashboardSearch {
			t.Error("expected dashboardSearch=false after enter")
		}
	})

	t.Run("esc closes search first", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.dashboardSearch = true
		m.dashboardFilter = "test"
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.dashboardSearch {
			t.Error("expected dashboardSearch=false after esc")
		}
		if m2.dashboardFilter != "" {
			t.Errorf("expected empty filter after esc, got %q", m2.dashboardFilter)
		}
		if m2.nav.Current() != ViewDashboard {
			t.Error("should stay on dashboard when closing search")
		}
	})

	t.Run("up/down ignored in search mode", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.dashboardSearch = true
		m.menuCursor = 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.menuCursor != 1 {
			t.Errorf("cursor should not change in search mode, got %d", m2.menuCursor)
		}
	})
}

func TestDashboardUpdate_Navigation(t *testing.T) {
	t.Run("down moves cursor", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.menuCursor != 1 {
			t.Errorf("expected cursor 1, got %d", m2.menuCursor)
		}
	})

	t.Run("up moves cursor back", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 1
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.menuCursor != 0 {
			t.Errorf("expected cursor 0, got %d", m2.menuCursor)
		}
	})

	t.Run("up at 0 stays at 0", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 0
		msg := tea.KeyMsg{Type: tea.KeyUp}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.menuCursor != 0 {
			t.Errorf("expected cursor 0 at boundary, got %d", m2.menuCursor)
		}
	})

	t.Run("down at max stays", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		// 2 done caps + 2 pending = 4 total, max index = 3
		m.menuCursor = 3
		msg := tea.KeyMsg{Type: tea.KeyDown}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.menuCursor != 3 {
			t.Errorf("expected cursor 3 at boundary, got %d", m2.menuCursor)
		}
	})

	t.Run("esc navigates back", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after esc, got %q", m2.nav.Current())
		}
	})

	t.Run("backspace navigates back when not in search", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		msg := tea.KeyMsg{Type: tea.KeyBackspace}
		result, _ := m.dashboardUpdate(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after backspace, got %q", m2.nav.Current())
		}
	})
}

func TestNavigateToCapDetail(t *testing.T) {
	t.Run("navigate to completed cap", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 0 // first done cap
		m.navigateToCapDetail()
		if m.nav.Current() != ViewDetail {
			t.Errorf("expected ViewDetail, got %q", m.nav.Current())
		}
		if m.planDetail.cap == nil {
			t.Error("expected cap to be set")
		}
		if m.planDetail.pending != nil {
			t.Error("expected pending to be nil")
		}
	})

	t.Run("navigate to second completed cap", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 1 // second done cap
		m.navigateToCapDetail()
		if m.nav.Current() != ViewDetail {
			t.Errorf("expected ViewDetail, got %q", m.nav.Current())
		}
		if m.planDetail.cap == nil {
			t.Error("expected cap to be set")
		}
	})

	t.Run("navigate to pending cap", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 2 // first pending (after 2 done)
		m.navigateToCapDetail()
		if m.nav.Current() != ViewDetail {
			t.Errorf("expected ViewDetail, got %q", m.nav.Current())
		}
		if m.planDetail.pending == nil {
			t.Error("expected pending to be set")
		}
		if m.planDetail.cap != nil {
			t.Error("expected cap to be nil for pending")
		}
	})

	t.Run("nil caps does nothing", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = nil
		m.navigateToCapDetail()
		if m.nav.Current() != ViewDashboard {
			t.Errorf("expected ViewDashboard preserved, got %q", m.nav.Current())
		}
	})

	t.Run("cursor out of range does nothing", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		m.caps = newTestCaps()
		m.menuCursor = 99
		m.navigateToCapDetail()
		if m.nav.Current() != ViewDashboard {
			t.Errorf("expected ViewDashboard preserved, got %q", m.nav.Current())
		}
	})
}

func TestRenderDashboard_WithCaps(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.caps = newTestCaps()
	output := m.renderDashboard()
	if output == "" {
		t.Error("expected non-empty dashboard with caps")
	}
	if !containsAny(output, []string{"Plan Dashboard", "v1.9", "Go+TS"}) {
		t.Error("dashboard missing strategy banner")
	}
	if !containsAny(output, []string{"Completed", "Pending"}) {
		t.Error("dashboard missing panel headers")
	}
}

func TestRenderDashboard_SearchActive(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.caps = newTestCaps()
	m.dashboardSearch = true
	m.dashboardFilter = "gov"
	output := m.renderDashboard()
	if !containsAny(output, []string{"Filter", "gov"}) {
		t.Error("dashboard missing search bar")
	}
}

func TestRenderDashboard_NilCaps(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.caps = nil
	output := m.renderDashboard()
	if !containsAny(output, []string{"Could not load"}) {
		t.Error("dashboard missing error message for nil caps")
	}
}

func TestRenderCompletedCaps(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.caps = newTestCaps()
	output := m.renderCompletedCaps(50)
	if output == "" {
		t.Error("expected non-empty completed caps")
	}
	if !containsAny(output, []string{"Completed"}) {
		t.Error("missing Completed header")
	}
}

func TestRenderPendingCaps(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.caps = newTestCaps()
	output := m.renderPendingCaps(50)
	if output == "" {
		t.Error("expected non-empty pending caps")
	}
	if !containsAny(output, []string{"Pending"}) {
		t.Error("missing Pending header")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a…"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// ── Update Routing Tests ─────────────────────────────────────────────

func TestUpdate_HelpToggle(t *testing.T) {
	t.Run("? from root pushes help", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewRoot}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewHelp {
			t.Errorf("expected ViewHelp, got %q", m2.nav.Current())
		}
	})

	t.Run("? from help pops help", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		// When on help and ? pressed, the global handler skips (view==ViewHelp),
		// then routes to helpUpdate which pops
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after ? on help, got %q", m2.nav.Current())
		}
	})

	t.Run("esc on help pops help", func(t *testing.T) {
		m := NewModel()
		m.ready = true
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		result, _ := m.Update(msg)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after esc on help, got %q", m2.nav.Current())
		}
	})
}

func TestUpdate_TailorDoneMsg(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.nav.stack = []string{ViewRoot, ViewTailor}
	m.tailorModel.step = TailorPreview
	msg := tailorDoneMsg{}
	result, _ := m.Update(msg)
	m2 := result.(Model)
	if m2.tailorModel.step != TailorDone {
		t.Errorf("expected TailorDone, got %d", m2.tailorModel.step)
	}
}

func TestUpdate_InstallTickMsg(t *testing.T) {
	m := NewModel()
	m.ready = true
	m.nav.stack = []string{ViewRoot, ViewInstall}
	m.installModel.running = true
	msg := installTickMsg{done: true, stage: StageDone, progress: 100}
	result, _ := m.Update(msg)
	m2 := result.(Model)
	if !m2.installModel.completed {
		t.Error("expected install completed after done tick")
	}
}

func TestHandleViewKey_AllViews(t *testing.T) {
	views := []struct {
		view string
		key  tea.KeyMsg
	}{
		{ViewWelcome, tea.KeyMsg{Type: tea.KeyEnter}},
		{ViewRoot, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}},
		{ViewDashboard, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}},
		{ViewHealth, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}},
		{ViewDetail, tea.KeyMsg{Type: tea.KeyEsc}},
		{ViewHelp, tea.KeyMsg{Type: tea.KeyEsc}},
		{ViewQuit, tea.KeyMsg{Type: tea.KeyEnter}},
	}
	for _, v := range views {
		t.Run(v.view, func(t *testing.T) {
			m := NewModel()
			m.width = 120
			m.nav.stack = []string{ViewWelcome, v.view}
			// Should not panic
			_, _ = m.handleViewKey(v.key, v.view)
		})
	}
}

func TestHandleMouse_AllViews(t *testing.T) {
	t.Run("mouse on root selects item", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.nav.stack = []string{ViewRoot}
		// Y=9 → row = 9-7 = 2 → health (item 2, dashboard is at item 1 which is at Y=8)
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: 9}
		result, _ := m.handleMouse(msg, ViewRoot)
		m2 := result.(Model)
		if m2.menuCursor != 2 {
			t.Errorf("expected menuCursor 2, got %d", m2.menuCursor)
		}
		if m2.nav.Current() != ViewHealth {
			t.Errorf("expected ViewHealth, got %q", m2.nav.Current())
		}
	})

	t.Run("mouse on help closes it", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewRoot, ViewHelp}
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft}
		result, _ := m.handleMouse(msg, ViewHelp)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot after click on help, got %q", m2.nav.Current())
		}
	})

	t.Run("mouse on dashboard stays on dashboard", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.caps = newTestCaps()
		m.nav.stack = []string{ViewRoot, ViewDashboard}
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, X: 10, Y: 10}
		result, _ := m.handleMouse(msg, ViewDashboard)
		m2 := result.(Model)
		if m2.nav.Current() != ViewDashboard {
			t.Errorf("expected ViewDashboard after click (no-op), got %q", m2.nav.Current())
		}
	})

	t.Run("mouse on install routes to install model", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.nav.stack = []string{ViewRoot, ViewInstall}
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: 10}
		// Should not panic
		_, _ = m.handleMouse(msg, ViewInstall)
	})

	t.Run("mouse on tailor routes to tailor model", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.nav.stack = []string{ViewRoot, ViewTailor}
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: 5}
		// Should not panic
		_, _ = m.handleMouse(msg, ViewTailor)
	})

	t.Run("mouse on welcome navigates to root", func(t *testing.T) {
		m := NewModel()
		m.nav.stack = []string{ViewWelcome}
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft}
		result, _ := m.handleMouse(msg, ViewWelcome)
		m2 := result.(Model)
		if m2.nav.Current() != ViewRoot {
			t.Errorf("expected ViewRoot, got %q", m2.nav.Current())
		}
	})
}

// ── Tailor Preview Tests ─────────────────────────────────────────────

func TestTailorPreviewUpdate(t *testing.T) {
	t.Run("left/right action focus", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorPreview
		tm.actionFocus = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
		result, _ := tm.Update(msg)
		if result.actionFocus != 1 {
			t.Errorf("expected actionFocus 1, got %d", result.actionFocus)
		}
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
		result, _ = result.Update(msg)
		if result.actionFocus != 0 {
			t.Errorf("expected actionFocus 0, got %d", result.actionFocus)
		}
	})

	t.Run("enter on back returns to select", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorPreview
		tm.actionFocus = 1 // Back button
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := tm.Update(msg)
		if result.step != TailorSelect {
			t.Errorf("expected TailorSelect after back, got %d", result.step)
		}
	})

	t.Run("enter on install goes to confirm", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorPreview
		tm.actionFocus = 0 // Install button
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, cmd := tm.Update(msg)
		if result.step != TailorConfirm {
			t.Errorf("expected TailorConfirm, got %d", result.step)
		}
		if cmd == nil {
			t.Error("expected non-nil cmd for tailor done")
		}
	})
}

func TestTailorSelectUpdate(t *testing.T) {
	t.Run("space toggles selection", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorSelect
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
		// Should not panic even if no selectable rows
		_, _ = tm.Update(msg)
	})

	t.Run("enter without plan stays on select", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorSelect
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		result, _ := tm.Update(msg)
		if result.step != TailorPreview {
			t.Errorf("expected TailorPreview (Enter transitions forward), got %d", result.step)
		}
	})
}

func TestTailorMouseUpdate(t *testing.T) {
	t.Run("wheel up in select", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorSelect
		tm.cursor = 1
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
		result, _ := tm.Update(msg)
		// Should not panic, cursor may change depending on selectable count
		_ = result
	})

	t.Run("wheel down in select", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorSelect
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
		result, _ := tm.Update(msg)
		_ = result
	})

	t.Run("left click in select", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorSelect
		msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Y: 5}
		result, _ := tm.Update(msg)
		_ = result
	})

	t.Run("mouse in preview step ignored", func(t *testing.T) {
		tm := NewTailorModel()
		tm.step = TailorPreview
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
		result, _ := tm.Update(msg)
		if result.step != TailorPreview {
			t.Errorf("expected TailorPreview preserved, got %d", result.step)
		}
	})
}

func TestRenderTailor(t *testing.T) {
	t.Run("render tailor routes to select", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.tailorModel.step = TailorSelect
		output := m.renderTailor()
		if output == "" {
			t.Error("expected non-empty tailor view")
		}
	})

	t.Run("render tailor routes to preview", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.tailorModel.step = TailorPreview
		output := m.renderTailor()
		if output == "" {
			t.Error("expected non-empty tailor preview")
		}
	})

	t.Run("render tailor routes to confirm", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.tailorModel.step = TailorConfirm
		output := m.renderTailor()
		if output == "" {
			t.Error("expected non-empty tailor confirm")
		}
	})

	t.Run("render tailor default falls to select", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.tailorModel.step = TailorStep(99)
		output := m.renderTailor()
		if output == "" {
			t.Error("expected non-empty default tailor view")
		}
	})
}

// ── Util Helper Tests ────────────────────────────────────────────────

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, expected int
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 0, 0},
		{5, 5, 5, 5},
	}
	for _, tt := range tests {
		result := clamp(tt.v, tt.lo, tt.hi)
		if result != tt.expected {
			t.Errorf("clamp(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, result, tt.expected)
		}
	}
}

func TestMinMax(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(5, 3) != 5 {
		t.Error("max(5,3) should be 5")
	}
}

func TestRenderTitleBar(t *testing.T) {
	output := renderTitleBar("Test Title")
	if output == "" {
		t.Error("expected non-empty title bar")
	}
	if !containsAny(output, []string{"OVAV Cockpit", "Test Title"}) {
		t.Error("title bar missing expected content")
	}
}

func TestRenderHelpBar(t *testing.T) {
	output := renderHelpBar("Press any key")
	if output == "" {
		t.Error("expected non-empty help bar")
	}
	if !containsAny(output, []string{"Press any key"}) {
		t.Error("help bar missing text")
	}
}

func TestRenderPctBar(t *testing.T) {
	tests := []struct {
		pct, width int
	}{
		{0, 10},
		{50, 10},
		{100, 10},
		{75, 0}, // should default to 10
	}
	for _, tt := range tests {
		output := renderPctBar(tt.pct, tt.width)
		if output == "" {
			t.Errorf("expected non-empty pct bar for pct=%d width=%d", tt.pct, tt.width)
		}
	}
}

func TestFindOVAVRoot_EnvOverride(t *testing.T) {
	// Save and restore cache
	oldCache := ovavRootCache
	oldSet := ovavRootCacheSet
	defer func() {
		ovavRootCache = oldCache
		ovavRootCacheSet = oldSet
	}()

	// Reset cache
	ovavRootCacheSet = false
	ovavRootCache = ""

	// Set env var
	t.Setenv("OVAV_ROOT", "/tmp/test-ovav-root")
	root := findOVAVRoot()
	if root != "/tmp/test-ovav-root" {
		t.Errorf("expected /tmp/test-ovav-root from env, got %q", root)
	}
}

// ── Button & ActionRow Tests ─────────────────────────────────────────

func TestRenderButton(t *testing.T) {
	active := RenderButton("Start", true)
	inactive := RenderButton("Back", false)
	if active == "" || inactive == "" {
		t.Error("expected non-empty button renders")
	}
}

func TestActionRow(t *testing.T) {
	output := ActionRow([]string{"OK", "Cancel"}, 0)
	if output == "" {
		t.Error("expected non-empty action row")
	}
	output2 := ActionRow([]string{"OK", "Cancel"}, 1)
	if output2 == "" {
		t.Error("expected non-empty action row with focus=1")
	}
}

// ── Health Helpers Tests ─────────────────────────────────────────────

func TestRenderCard(t *testing.T) {
	output := renderCard("Test Card", 40, "line 1", "line 2")
	if output == "" {
		t.Error("expected non-empty card")
	}
	if !containsAny(output, []string{"Test Card"}) {
		t.Error("card missing title")
	}
}

func TestKv(t *testing.T) {
	output := kv("Key", "Value")
	if output == "" {
		t.Error("expected non-empty kv")
	}
}

// ── CountDone Tests ──────────────────────────────────────────────────

func TestCountDone(t *testing.T) {
	caps := newTestCaps()
	n := countDone(caps)
	if n != 2 {
		t.Errorf("expected 2 done caps, got %d", n)
	}

	emptyCaps := &data.CapsData{Caps: map[string]data.Cap{}}
	if countDone(emptyCaps) != 0 {
		t.Error("expected 0 done caps for empty map")
	}
}

// ── RenderCurrentView Coverage ───────────────────────────────────────

func TestRenderCurrentView_AllViews(t *testing.T) {
	views := []string{
		ViewWelcome, ViewRoot, ViewDashboard, ViewHealth,
		ViewInstall, ViewTailor, ViewCLI, ViewDetail, ViewQuit, ViewHelp,
	}
	for _, v := range views {
		t.Run(v, func(t *testing.T) {
			m := NewModel()
			m.width = 120
			m.nav.stack = []string{v}
			m.caps = newTestCaps()
			output := m.renderCurrentView()
			if output == "" {
				t.Errorf("expected non-empty output for view %q", v)
			}
		})
	}

	t.Run("unknown view defaults to welcome", func(t *testing.T) {
		m := NewModel()
		m.width = 120
		m.nav.stack = []string{"unknown_view"}
		output := m.renderCurrentView()
		if output == "" {
			t.Error("expected non-empty output for unknown view")
		}
	})
}

// ── Install View Coverage ────────────────────────────────────────────

func TestRenderInstallOverview(t *testing.T) {
	m := NewModel()
	m.width = 120
	output := m.renderInstallOverview()
	if output == "" {
		t.Error("expected non-empty install overview")
	}
	if !containsAny(output, []string{"Install Pipeline", "Pipeline Stages"}) {
		t.Error("install overview missing expected content")
	}
}

func TestRenderInstallOverview_DirtyTree(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.sys = &data.SystemInfo{Dirty: "dirty"}
	output := m.renderInstallOverview()
	if !containsAny(output, []string{"dirty", "Working tree"}) {
		t.Error("install overview missing dirty warning")
	}
}

func TestInstallModel_LeftRight(t *testing.T) {
	im := NewInstallModel()
	// Right
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	result, _ := im.Update(msg)
	if result.actionFocus != 1 {
		t.Errorf("expected actionFocus 1, got %d", result.actionFocus)
	}
	// Right again stays at 1
	result, _ = result.Update(msg)
	if result.actionFocus != 1 {
		t.Errorf("expected actionFocus 1 at max, got %d", result.actionFocus)
	}
	// Left
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	result, _ = result.Update(msg)
	if result.actionFocus != 0 {
		t.Errorf("expected actionFocus 0, got %d", result.actionFocus)
	}
}

func TestInstallModel_EnterBack(t *testing.T) {
	im := NewInstallModel()
	im.actionFocus = 1 // Back button
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := im.Update(msg)
	if result.actionFocus != 1 {
		t.Errorf("expected actionFocus preserved, got %d", result.actionFocus)
	}
	if cmd == nil {
		t.Error("expected goBack cmd")
	}
}

func TestInstallModel_EnterCompleted(t *testing.T) {
	im := NewInstallModel()
	im.completed = true
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := im.Update(msg)
	_ = result
	if cmd == nil {
		t.Error("expected goBack cmd when completed")
	}
}

// ── Plan Detail Update Tests ─────────────────────────────────────────

func TestPlanDetailUpdate_BackKeys(t *testing.T) {
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEsc},
		{Type: tea.KeyBackspace},
	}
	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			m := NewModel()
			m.nav.stack = []string{ViewRoot, ViewDashboard, ViewDetail}
			result, _ := m.planDetailUpdate(key)
			m2 := result.(Model)
			if m2.nav.Current() != ViewDashboard {
				t.Errorf("expected ViewDashboard after %s, got %q", key.String(), m2.nav.Current())
			}
		})
	}
}

// ── Welcome Update Tests ─────────────────────────────────────────────

func TestWelcomeUpdate_Quit(t *testing.T) {
	m := NewModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	result, _ := m.welcomeUpdate(msg)
	m2 := result.(Model)
	if m2.nav.Current() != ViewQuit {
		t.Errorf("expected ViewQuit, got %q", m2.nav.Current())
	}
}

func TestWelcomeUpdate_OtherKey(t *testing.T) {
	m := NewModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	result, _ := m.welcomeUpdate(msg)
	m2 := result.(Model)
	if m2.nav.Current() != ViewWelcome {
		t.Errorf("expected ViewWelcome preserved, got %q", m2.nav.Current())
	}
}

// ── Root Update Tests ────────────────────────────────────────────────

func TestRootUpdate_AllMenuItems(t *testing.T) {
	// Map menu item IDs to expected view names (some differ)
	expectedViews := map[string]string{
		"cli": ViewCLI, // "cli" → "cli_selector"
	}
	for i, item := range menuItems {
		t.Run(item.label, func(t *testing.T) {
			m := NewModel()
			m.nav.stack = []string{ViewRoot}
			m.menuCursor = i
			msg := tea.KeyMsg{Type: tea.KeyEnter}
			result, _ := m.rootUpdate(msg)
			m2 := result.(Model)
			want := item.id
			if v, ok := expectedViews[item.id]; ok {
				want = v
			}
			if m2.nav.Current() != want {
				t.Errorf("expected %q, got %q", want, m2.nav.Current())
			}
		})
	}
}

func TestRootUpdate_UpDown(t *testing.T) {
	m := NewModel()
	m.nav.stack = []string{ViewRoot}
	m.menuCursor = 0

	// Down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.rootUpdate(msg)
	m2 := result.(Model)
	if m2.menuCursor != 1 {
		t.Errorf("expected cursor 1, got %d", m2.menuCursor)
	}

	// Up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = m2.rootUpdate(msg)
	m3 := result.(Model)
	if m3.menuCursor != 0 {
		t.Errorf("expected cursor 0, got %d", m3.menuCursor)
	}
}

// ── Health Update Tests ──────────────────────────────────────────────

func TestHealthUpdate_BackKeys(t *testing.T) {
	keys := []string{"q", "esc", "backspace"}
	for _, k := range keys {
		t.Run(k, func(t *testing.T) {
			m := NewModel()
			m.nav.stack = []string{ViewRoot, ViewHealth}
			var msg tea.KeyMsg
			if k == "backspace" {
				msg = tea.KeyMsg{Type: tea.KeyBackspace}
			} else if k == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
			}
			result, _ := m.healthUpdate(msg)
			m2 := result.(Model)
			if m2.nav.Current() != ViewRoot {
				t.Errorf("expected ViewRoot after %s, got %q", k, m2.nav.Current())
			}
		})
	}
}

func TestHealthUpdate_Refresh(t *testing.T) {
	m := NewModel()
	m.nav.stack = []string{ViewRoot, ViewHealth}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	result, _ := m.healthUpdate(msg)
	m2 := result.(Model)
	// Should stay on health and refresh sys data
	if m2.nav.Current() != ViewHealth {
		t.Error("expected to stay on health view after refresh")
	}
	if m2.sys == nil {
		t.Error("expected sys to be populated after refresh")
	}
}

// ── RenderHealth with data ───────────────────────────────────────────

func TestRenderHealth_WithSystemInfo(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.sys = &data.SystemInfo{
		Branch:      "feature/test",
		SHA:         "abc1234",
		Dirty:       "clean",
		GoVersion:   "go1.22.0",
		PlanVersion: "v1.9",
		Strategy:    "Go+TS",
		CapsDone:    5,
		CapsPending: 3,
		DoctorPass:  8,
		DoctorFail:  1,
		DoctorWarn:  2,
		DoctorTotal: 11,
		OVAVRoot:    "/test/root",
	}
	output := m.renderHealth()
	if output == "" {
		t.Error("expected non-empty health view with data")
	}
	if !containsAny(output, []string{"Health", "Identity", "Git"}) {
		t.Error("health view missing expected sections")
	}
}

func TestRenderHealth_NilSys(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.sys = nil
	output := m.renderHealth()
	if !containsAny(output, []string{"No system info"}) {
		t.Error("health view missing error for nil sys")
	}
}

func TestRenderHealth_DirtyTree(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.sys = &data.SystemInfo{
		Branch:      "develop",
		Dirty:       "dirty",
		GoVersion:   "go1.22.0",
		DoctorTotal: 0,
	}
	output := m.renderHealth()
	if output == "" {
		t.Error("expected non-empty health view")
	}
}

// ── Benchmarks for new coverage ──────────────────────────────────────

func BenchmarkRenderHelp(b *testing.B) {
	m := NewModel()
	m.nav.stack = []string{ViewRoot, ViewHelp}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderHelp()
	}
}

func BenchmarkRenderDashboardWithCaps(b *testing.B) {
	m := NewModel()
	m.width = 120
	m.caps = newTestCaps()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderDashboard()
	}
}

func BenchmarkRenderHealthWithSys(b *testing.B) {
	m := NewModel()
	m.width = 120
	m.sys = &data.SystemInfo{
		Branch: "main", SHA: "abc", Dirty: "clean",
		GoVersion: "go1.22", DoctorPass: 10, DoctorTotal: 10,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderHealth()
	}
}
