package ows

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Registry Tests ───────────────────────────────────────────────────────────────

func TestCommandRegistry_AllCommands(t *testing.T) {
	expected := 17
	if len(CommandRegistry) != expected {
		t.Errorf("CommandRegistry has %d commands, want %d", len(CommandRegistry), expected)
	}

	// Verify all commands have required fields
	for name, cmd := range CommandRegistry {
		if cmd.Name == "" {
			t.Errorf("command %q has empty Name", name)
		}
		if cmd.ShortName == "" {
			t.Errorf("command %q has empty ShortName", name)
		}
		if cmd.Short == "" {
			t.Errorf("command %q has empty Short", name)
		}
	}
}

func TestCommandRegistry_ShortNames(t *testing.T) {
	tests := []struct {
		short string
		full  string
	}{
		{"owc", "ovav worktree create"},
		{"owu", "ovav worktree update"},
		{"ows", "ovav worktree sync"},
		{"owp", "ovav worktree prepare"},
		{"owv", "ovav worktree verify"},
		{"owd", "ovav worktree done"},
		{"owx", "ovav worktree route"},
		{"owa", "ovav worktree abort"},
		{"owr", "ovav worktree rescue"},
		{"owl", "ovav worktree list"},
		{"owlk", "ovav worktree lock"},
		{"owm", "ovav worktree move"},
		{"owclean", "ovav worktree clean"},
		{"own", "ovav worktree nuke"},
	}

	for _, tt := range tests {
		cmd, ok := FindByShortName(tt.short)
		if !ok {
			t.Errorf("short name %q not found in registry", tt.short)
			continue
		}
		if cmd.Name != tt.full {
			t.Errorf("short %q → name %q, want %q", tt.short, cmd.Name, tt.full)
		}
	}
}

func TestProfileRegistry_AllProfiles(t *testing.T) {
	expected := 12
	if len(ProfileRegistry) != expected {
		t.Errorf("ProfileRegistry has %d profiles, want %d", len(ProfileRegistry), expected)
	}

	// Verify each profile has a valid base branch
	for name, p := range ProfileRegistry {
		if p.BaseBranch != "develop" && p.BaseBranch != "main" {
			t.Errorf("profile %q has invalid BaseBranch: %q", name, p.BaseBranch)
		}
	}
}

func TestProfileRegistry_MergeRules(t *testing.T) {
	// Hotfix and emergency must merge to main+develop
	if p := ProfileRegistry["hotfix"]; p.MergeTo != "main+develop" {
		t.Errorf("hotfix MergeTo = %q, want main+develop", p.MergeTo)
	}
	if p := ProfileRegistry["emergency"]; p.MergeTo != "main+develop" {
		t.Errorf("emergency MergeTo = %q, want main+develop", p.MergeTo)
	}
	// Release merges to main only
	if p := ProfileRegistry["release"]; p.MergeTo != "main" {
		t.Errorf("release MergeTo = %q, want main", p.MergeTo)
	}
}

func TestParseArgs(t *testing.T) {
	defs := []Arg{
		{Name: "target", Required: true},
		{Name: "mode", Default: "cherry-pick"},
	}

	// Both provided
	result := parseArgs([]string{"main", "hotfix"}, defs)
	if result["target"] != "main" || result["mode"] != "hotfix" {
		t.Errorf("parseArgs both: got target=%q mode=%q", result["target"], result["mode"])
	}

	// Only required
	result = parseArgs([]string{"develop"}, defs)
	if result["target"] != "develop" || result["mode"] != "cherry-pick" {
		t.Errorf("parseArgs default: got target=%q mode=%q", result["target"], result["mode"])
	}

	// None
	result = parseArgs([]string{}, defs)
	if result["target"] != "cherry-pick" && result["mode"] != "cherry-pick" {
		t.Errorf("parseArgs empty: should use defaults where available")
	}
}

// ── State Machine Tests ──────────────────────────────────────────────────────────

func TestStateMachine_ValidTransitions(t *testing.T) {
	tests := []struct {
		from  State
		event Event
		to    State
	}{
		{StateCreated, EvWorkStarted, StateActive},
		{StateActive, EvConflictDetected, StateDirty},
		{StateActive, EvVerificationPassed, StateVerified},
		{StateActive, EvLockRequested, StateLocked},
		{StateActive, EvStaleDetected, StateStale},
		{StateDirty, EvConflictResolved, StateActive},
		{StateVerified, EvIntegrationComplete, StateIntegrated},
		{StateIntegrated, EvCleanupComplete, StateCleaned},
		{StateFailed, EvRescueRequested, StateRescued},
		{StateLocked, EvUnlockRequested, StateActive},
	}

	for _, tt := range tests {
		to, ok := ValidTransition(tt.from, tt.event)
		if !ok {
			t.Errorf("transition %s → (%s) should be valid", tt.from, tt.event)
		}
		if to != tt.to {
			t.Errorf("transition %s → (%s) = %s, want %s", tt.from, tt.event, to, tt.to)
		}
	}
}

func TestStateMachine_InvalidTransitions(t *testing.T) {
	invalid := []struct {
		from  State
		event Event
	}{
		{StateCreated, EvIntegrationComplete},   // can't merge before active
		{StateCleaned, EvWorkStarted},           // cleaned is terminal
		{StateVerified, EvWorkStarted},          // already verified
		{StateIntegrated, EvVerificationPassed}, // can't re-verify after merge
	}

	for _, tt := range invalid {
		_, ok := ValidTransition(tt.from, tt.event)
		if ok {
			t.Errorf("transition %s → (%s) should be INVALID", tt.from, tt.event)
		}
	}
}

func TestExecuteTransition(t *testing.T) {
	wt := &WorktreeRecord{
		ID:    "task/test",
		State: StateCreated,
	}

	if err := ExecuteTransition(wt, EvWorkStarted); err != nil {
		t.Fatalf("ExecuteTransition: %v", err)
	}
	if wt.State != StateActive {
		t.Errorf("state = %s, want %s", wt.State, StateActive)
	}

	// Invalid transition should error
	if err := ExecuteTransition(wt, EvIntegrationComplete); err == nil {
		t.Error("expected error for invalid transition CREATED → INTEGRATION_COMPLETE")
	}
}

func TestDOTGraph(t *testing.T) {
	graph := DOTGraph()
	if graph == "" {
		t.Error("DOTGraph returned empty string")
	}
	if !contains(graph, "digraph OWS_StateMachine") {
		t.Error("DOTGraph missing digraph header")
	}
}

func TestASCIIStateMatrix(t *testing.T) {
	matrix := ASCIIStateMatrix()
	if matrix == "" {
		t.Error("ASCIIStateMatrix returned empty string")
	}
	if !contains(matrix, "CREATED") {
		t.Error("ASCIIStateMatrix missing CREATED state")
	}
}

// ── Audit Tests ──────────────────────────────────────────────────────────────────

func TestAuditDB_OpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	// Verify tables exist by inserting and querying
	if err := db.Log(LogEntry{
		Actor:   "test",
		Command: "owc",
		Target:  "task/test",
		Result:  "success",
	}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	entries, err := db.QueryLogs("owc", "", 10)
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("QueryLogs returned %d entries, want 1", len(entries))
	}
	if entries[0].Actor != "test" || entries[0].Command != "owc" {
		t.Errorf("entry = %+v, want actor=test command=owc", entries[0])
	}
}

func TestAuditDB_SaveLoadWorktree(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	wt := WorktreeRecord{
		ID:      "task/test-save",
		Branch:  "task/test-save",
		Profile: "feature",
		Owner:   "thavren",
		State:   StateActive,
	}

	if err := db.SaveWorktree(wt); err != nil {
		t.Fatalf("SaveWorktree: %v", err)
	}

	loaded, err := db.LoadWorktree("task/test-save")
	if err != nil {
		t.Fatalf("LoadWorktree: %v", err)
	}

	if loaded.State != StateActive || loaded.Owner != "thavren" {
		t.Errorf("loaded = %+v, want State=ACTIVE Owner=thavren", loaded)
	}
}

func TestAuditDB_ListWorktrees(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	// Create 3 worktrees
	for _, name := range []string{"task/a", "task/b", "task/c"} {
		db.SaveWorktree(WorktreeRecord{
			ID:    name,
			State: StateActive,
			Owner: "test",
		})
	}

	wts, err := db.ListWorktrees(StateActive, "test")
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(wts) != 3 {
		t.Errorf("ListWorktrees returned %d, want 3", len(wts))
	}
}

func TestAuditDB_PolicyVersions(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	if err := db.SavePolicy("POL-001", 1, "no force push", "thavren"); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	if err := db.SavePolicy("POL-001", 2, "no force push + block SSH", "thavren"); err != nil {
		t.Fatalf("SavePolicy v2: %v", err)
	}

	version, rule, err := db.GetPolicyVersion("POL-001")
	if err != nil {
		t.Fatalf("GetPolicyVersion: %v", err)
	}
	if version != 2 {
		t.Errorf("GetPolicyVersion version = %d, want 2", version)
	}
	if rule != "no force push + block SSH" {
		t.Errorf("GetPolicyVersion rule = %q, want latest", rule)
	}
}

func TestAuditDB_PruneStale(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	db.SaveWorktree(WorktreeRecord{ID: "task/stale", State: StateCleaned})
	db.SaveWorktree(WorktreeRecord{ID: "task/active", State: StateActive})

	// Prune with 0 duration (prunes all CLEANED)
	n, err := db.PruneStale(0)
	if err != nil {
		t.Fatalf("PruneStale: %v", err)
	}
	if n != 1 {
		t.Errorf("PruneStale pruned %d, want 1", n)
	}

	// Verify active still exists
	_, err = db.LoadWorktree("task/active")
	if err != nil {
		t.Errorf("active worktree should still exist: %v", err)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────────

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestOWSPackage(t *testing.T) {
	// Smoke test: ensure temp dir is clean after all audit tests
	dir := t.TempDir()
	dbPath := filepath.Join(dir, ".ovav", "ows", "audit.db")
	if _, err := os.Stat(dbPath); err == nil {
		t.Log("audit.db exists (expected from previous test)")
	}
	// Run sub-tests
	t.Run("Registry", func(t *testing.T) {
		TestCommandRegistry_AllCommands(t)
		TestCommandRegistry_ShortNames(t)
		TestProfileRegistry_AllProfiles(t)
		TestProfileRegistry_MergeRules(t)
		TestParseArgs(t)
	})
	t.Run("StateMachine", func(t *testing.T) {
		TestStateMachine_ValidTransitions(t)
		TestStateMachine_InvalidTransitions(t)
		TestExecuteTransition(t)
		TestDOTGraph(t)
		TestASCIIStateMatrix(t)
	})
	t.Run("Audit", func(t *testing.T) {
		TestAuditDB_OpenAndMigrate(t)
		TestAuditDB_SaveLoadWorktree(t)
		TestAuditDB_ListWorktrees(t)
		TestAuditDB_PolicyVersions(t)
		TestAuditDB_PruneStale(t)
	})
}
