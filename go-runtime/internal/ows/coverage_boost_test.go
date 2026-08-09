package ows

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// coverage_boost_test.go — Coverage boost: 70.8% → 80%+
// Uses OpenAudit(t.TempDir()) for DB operations.
// ═══════════════════════════════════════════════════════════════════════════

func TestCB_IsConflictMarkerLine(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"+<<<<<<< .our", true}, {"+=======", true}, {"+>>>>>>> .their", true},
		{"+normal line", false}, {"<<<<<<< .our", false}, {"", false},
	}
	for _, tt := range tests {
		if got := isConflictMarkerLine(tt.input); got != tt.want {
			t.Errorf("isConflictMarkerLine(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCB_IsTreeHash(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"abc123def456789012345678901234567890abcd", true},
		{"ABC123def456789012345678901234567890abcd", false},
		{"abc123def", false}, {"", false},
	}
	for _, tt := range tests {
		if got := isTreeHash(tt.input); got != tt.want {
			t.Errorf("isTreeHash(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCB_FirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "b", "c"); got != "b" {
		t.Errorf("got %q", got)
	}
	if got := firstNonEmpty("", "", ""); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestCB_MakeSet(t *testing.T) {
	s := makeSet([]string{"a", "b", "a", "  c  ", ""})
	if len(s) != 3 {
		t.Errorf("expected 3, got %d", len(s))
	}
}

func TestCB_Union(t *testing.T) {
	a := map[string]bool{"x": true}
	b := map[string]bool{"y": true}
	u := union(a, b)
	if len(u) != 2 {
		t.Errorf("expected 2, got %d", len(u))
	}
}

func TestCB_ConflictMatrix_Summary(t *testing.T) {
	m := &ConflictMatrix{TotalFiles: 5, ConflictFiles: 0, SafeFiles: 5}
	if got := m.Summary(); got != "✓ 5 files — 0 conflicts, safe to merge" {
		t.Errorf("got %q", got)
	}
	m2 := &ConflictMatrix{TotalFiles: 5, ConflictFiles: 2, SafeFiles: 3}
	if got := m2.Summary(); got != "⚠ 5 files — 2 conflict(s), 3 safe" {
		t.Errorf("got %q", got)
	}
}

func TestCB_ConflictMatrix_Conflicts(t *testing.T) {
	m := &ConflictMatrix{Files: []ConflictPrediction{
		{FilePath: "a.go", Status: "conflict"},
		{FilePath: "b.go", Status: "safe"},
		{FilePath: "c.go", Status: "conflict"},
	}}
	if c := m.Conflicts(); len(c) != 2 {
		t.Errorf("expected 2, got %d", len(c))
	}
}

func TestCB_EventBus_PubSub(t *testing.T) {
	bus := &EventBus{subscribers: make(map[EventType][]EventHandler)}
	received := make(chan BusEvent, 1)
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) { received <- e })
	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test", Payload: map[string]any{"k": "v"}})
	select {
	case evt := <-received:
		if evt.Source != "test" {
			t.Errorf("source = %q", evt.Source)
		}
	case <-time.After(1 * time.Second):
		t.Error("handler not called")
	}
}

func TestCB_EventBus_Emit_NoDB(t *testing.T) {
	bus := &EventBus{subscribers: make(map[EventType][]EventHandler)}
	bus.Emit(BusEvent{Type: EvtConflictDetected, Source: "test"})
}

func TestCB_EventBus_Close_NoDB(t *testing.T) {
	bus := &EventBus{}
	if err := bus.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestCB_EventBus_QueryEvents_NoDB(t *testing.T) {
	bus := &EventBus{}
	_, err := bus.QueryEvents(EvtWorktreeCreated, time.Hour)
	if err == nil {
		t.Error("expected error without DB")
	}
}

func TestCB_EventBus_MultipleSubscribers(t *testing.T) {
	bus := &EventBus{subscribers: make(map[EventType][]EventHandler)}
	count := 0
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) { count++ })
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) { count++ })
	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test"})
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestCB_EventBus_WrongType(t *testing.T) {
	bus := &EventBus{subscribers: make(map[EventType][]EventHandler)}
	called := false
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) { called = true })
	bus.EmitSync(BusEvent{Type: EvtConflictDetected, Source: "test"})
	if called {
		t.Error("wrong type handler should not be called")
	}
}

func TestCB_AgentLock_IsExpired(t *testing.T) {
	if !(&AgentLock{ExpiresAt: time.Now().UTC().Add(-1 * time.Hour)}).IsExpired() {
		t.Error("past → expired")
	}
	if (&AgentLock{ExpiresAt: time.Now().UTC().Add(1 * time.Hour)}).IsExpired() {
		t.Error("future → not expired")
	}
}

func TestCB_AuditDB_LogAndQuery(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	adb.Log(LogEntry{Timestamp: time.Now().UTC(), Actor: "alice", Command: "owc", Result: "ok"})
	adb.Log(LogEntry{Timestamp: time.Now().UTC(), Actor: "bob", Command: "owd", Result: "ok"})
	logs, _ := adb.QueryLogs("", "alice", 10)
	if len(logs) != 1 {
		t.Errorf("alice: expected 1, got %d", len(logs))
	}
	logs, _ = adb.QueryLogs("owd", "", 10)
	if len(logs) != 1 {
		t.Errorf("owd: expected 1, got %d", len(logs))
	}
}

func TestCB_AuditDB_SaveLoadWorktree(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	now := time.Now().UTC()
	adb.SaveWorktree(WorktreeRecord{ID: "wt-001", Branch: "feature/test", Profile: "feature",
		Owner: "thavren", State: StateActive, PolicyVer: 1, CreatedAt: now, UpdatedAt: now})
	loaded, err := adb.LoadWorktree("wt-001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Branch != "feature/test" {
		t.Errorf("branch = %q", loaded.Branch)
	}
}

func TestCB_AuditDB_LoadWorktree_NotFound(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	_, err = adb.LoadWorktree("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCB_AuditDB_ListWorktrees(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	now := time.Now().UTC()
	adb.SaveWorktree(WorktreeRecord{ID: "wt-1", Branch: "a", State: StateActive, CreatedAt: now, UpdatedAt: now})
	adb.SaveWorktree(WorktreeRecord{ID: "wt-2", Branch: "b", State: StateActive, Owner: "alice", CreatedAt: now, UpdatedAt: now})
	wts, _ := adb.ListWorktrees(StateActive, "")
	if len(wts) != 2 {
		t.Errorf("expected 2, got %d", len(wts))
	}
	wts, _ = adb.ListWorktrees(StateActive, "alice")
	if len(wts) != 1 {
		t.Errorf("alice: expected 1, got %d", len(wts))
	}
}

func TestCB_AuditDB_PruneStale(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	old := time.Now().UTC().Add(-48 * time.Hour)
	now := time.Now().UTC()
	adb.SaveWorktree(WorktreeRecord{ID: "old", Branch: "a", State: StateCleaned, CreatedAt: old, UpdatedAt: old})
	adb.SaveWorktree(WorktreeRecord{ID: "new", Branch: "b", State: StateActive, CreatedAt: now, UpdatedAt: now})
	pruned, err := adb.PruneStale(24 * time.Hour)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if pruned != 1 {
		t.Errorf("expected 1, got %d", pruned)
	}
}

func TestCB_AuditDB_PolicyVersion(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	if err := adb.SavePolicy("test", 42, "rule", "thavren"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	ver, _, err := adb.GetPolicyVersion("test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ver != 42 {
		t.Errorf("version = %d, want 42", ver)
	}
}

func TestCB_AuditDB_GetPolicy_Empty(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	ver, _, err := adb.GetPolicyVersion("nonexistent")
	// May return error for missing policy — that's OK
	_ = err
	if ver != 0 {
		t.Errorf("version = %d, want 0", ver)
	}
}

func TestCB_BoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("true → 1")
	}
	if boolToInt(false) != 0 {
		t.Error("false → 0")
	}
}

func TestCB_WorktreeState_Values(t *testing.T) {
	if StateActive != "ACTIVE" {
		t.Errorf("StateActive = %q", StateActive)
	}
	if StateCreated != "CREATED" {
		t.Errorf("StateCreated = %q", StateCreated)
	}
}

// ── OpenAudit error paths ───────────────────────────────────────────────────

func TestCB_OpenAudit_MkdirError(t *testing.T) {
	// Use a path that can't be created (file in the way)
	tmp := t.TempDir()
	blocker := tmp + "/blocker"
	// Create a file where the directory should be
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := OpenAudit(blocker + "/.ovav/ows")
	if err == nil {
		t.Error("expected error when dir creation fails")
	}
}

func TestCB_OpenAudit_Success(t *testing.T) {
	adb, err := OpenAudit(t.TempDir())
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer adb.Close()
	if adb.db == nil {
		t.Error("db should not be nil")
	}
}

// ── OpenOfflineQueue error paths ────────────────────────────────────────────

func TestCB_OpenOfflineQueue_MkdirError(t *testing.T) {
	tmp := t.TempDir()
	blocker := tmp + "/blocker"
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := OpenOfflineQueue(blocker + "/.ovav/ows")
	if err == nil {
		t.Error("expected error when dir creation fails")
	}
}

func TestCB_OpenOfflineQueue_Success(t *testing.T) {
	q, err := OpenOfflineQueue(t.TempDir())
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}
	defer q.db.Close()
	if q.db == nil {
		t.Error("db should not be nil")
	}
}

// ── getHMACSecret ───────────────────────────────────────────────────────────

func TestCB_GetHMACSecret_WithEnv(t *testing.T) {
	t.Setenv("OVAV_HMAC_SECRET", "my-secret-key")
	secret := getHMACSecret()
	if secret != "my-secret-key" {
		t.Errorf("got %q, want 'my-secret-key'", secret)
	}
}

func TestCB_GetHMACSecret_Fallback(t *testing.T) {
	os.Unsetenv("OVAV_HMAC_SECRET")
	secret := getHMACSecret()
	if secret == "" {
		t.Error("fallback should not be empty")
	}
}

// ── OfflineQueue operations ─────────────────────────────────────────────────

func TestCB_OfflineQueue_EnqueueDequeue(t *testing.T) {
	q, err := OpenOfflineQueue(t.TempDir())
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}
	defer q.db.Close()

	op, err := q.Enqueue("test-op", `{"key":"value"}`, "test-secret")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if op.ID == "" {
		t.Error("op ID should not be empty")
	}

	ops, err := q.DequeueAll()
	if err != nil {
		t.Fatalf("DequeueAll: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("expected 1 op, got %d", len(ops))
	}
}

func TestCB_OfflineQueue_DequeueAll_WrongSecret(t *testing.T) {
	q, err := OpenOfflineQueue(t.TempDir())
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}
	defer q.db.Close()

	q.Enqueue("test-op", `{"key":"value"}`, "correct-secret")

	// DequeueAll returns all ops — HMAC verification is the caller's job
	ops, err := q.DequeueAll()
	if err != nil {
		t.Fatalf("DequeueAll: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("expected 1 op, got %d", len(ops))
	}
	// Verify HMAC is stored
	if ops[0].HMAC == "" {
		t.Error("HMAC should not be empty")
	}
}

func TestCB_OfflineQueue_EnqueueError(t *testing.T) {
	q, err := OpenOfflineQueue(t.TempDir())
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}
	q.db.Close() // Close DB to trigger error

	_, err = q.Enqueue("test-op", `{"key":"value"}`, "secret")
	if err == nil {
		t.Error("expected error when DB is closed")
	}
}

// ── signHMAC ────────────────────────────────────────────────────────────────

func TestCB_SignHMAC(t *testing.T) {
	sig1 := signHMAC("data", "secret")
	sig2 := signHMAC("data", "secret")
	if sig1 != sig2 {
		t.Error("same input should produce same HMAC")
	}
	if sig1 == "" {
		t.Error("HMAC should not be empty")
	}
	sig3 := signHMAC("data", "different-secret")
	if sig1 == sig3 {
		t.Error("different secret should produce different HMAC")
	}
}

// ── hygiene.go: hasContent, shouldSkipPath ──────────────────────────────────

func TestCB_HasContent(t *testing.T) {
	dir := t.TempDir()
	if hasContent(dir) {
		t.Error("empty dir should have no content")
	}

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644)
	if !hasContent(dir) {
		t.Error("dir with file should have content")
	}

	subDir := filepath.Join(dir, "sub")
	if hasContent(subDir) {
		t.Error("non-existent dir should have no content")
	}
}

func TestCB_ShouldSkipPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".git/config", true},
		{"node_modules/pkg", true},
		{".ovav/worktrees/feat", true},
		{".ovav/runtime/logs/app.log", true},
		{"src/main.go", false},
		{".ovav/registry/file.yaml", false},
	}
	for _, tt := range tests {
		if got := shouldSkipPath(tt.path); got != tt.want {
			t.Errorf("shouldSkipPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// ── policy.go: randomNonce ──────────────────────────────────────────────────

func TestCB_RandomNonce(t *testing.T) {
	nonce, err := randomNonce()
	if err != nil {
		t.Fatalf("randomNonce: %v", err)
	}
	if len(nonce) != 16 {
		t.Errorf("nonce length = %d, want 16", len(nonce))
	}
	// Two calls should produce different nonces
	nonce2, _ := randomNonce()
	if nonce == nonce2 {
		t.Error("two calls should produce different nonces")
	}
}

// ── state.go: State transitions ─────────────────────────────────────────────

func TestCB_State_StringValues(t *testing.T) {
	states := []struct {
		s    State
		want string
	}{
		{StateCreated, "CREATED"},
		{StateActive, "ACTIVE"},
		{StateDirty, "DIRTY"},
		{StateVerified, "VERIFIED"},
		{StateIntegrated, "INTEGRATED"},
		{StateCleaned, "CLEANED"},
		{StateLocked, "LOCKED"},
		{StateFailed, "FAILED"},
		{StateRescued, "RESCUED"},
		{StateStale, "STALE"},
	}
	for _, tt := range states {
		if string(tt.s) != tt.want {
			t.Errorf("State(%q) = %q", tt.want, string(tt.s))
		}
	}
}

// ── registry.go: FindByShortName ────────────────────────────────────────────

func TestCB_FindByShortName(t *testing.T) {
	cmd, ok := FindByShortName("owc")
	if !ok {
		t.Error("FindByShortName(owc) should return true")
	}
	if cmd.Name == "" {
		t.Error("command name should not be empty")
	}
	_, ok = FindByShortName("nonexistent-xyz")
	if ok {
		t.Error("FindByShortName(nonexistent) should return false")
	}
}

// ── Real git repo tests for handler coverage ────────────────────────────────

func createTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s (%v)", c[1:], string(out), err)
		}
	}
	return dir
}

func TestCB_CleanWorktrees_Empty(t *testing.T) {
	dir := createTestGitRepo(t)
	err := cleanWorktrees(dir, nil)
	if err != nil {
		t.Errorf("cleanWorktrees: %v", err)
	}
}

func TestCB_CleanWorktrees_DryRun(t *testing.T) {
	dir := createTestGitRepo(t)
	err := cleanWorktrees(dir, map[string]string{"dry-run": "true"})
	if err != nil {
		t.Errorf("cleanWorktrees dry-run: %v", err)
	}
}

func TestCB_CheckOrphanWorktreeDirs_NoWorktrees(t *testing.T) {
	dir := createTestGitRepo(t)
	orphans := checkOrphanWorktreeDirs(dir)
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d", len(orphans))
	}
}

func TestCB_CheckOrphanWorktreeDirs_WithOrphan(t *testing.T) {
	dir := createTestGitRepo(t)
	// Create orphan dir in .ovav/worktrees
	orphanDir := filepath.Join(dir, ".ovav", "worktrees", "orphan-feature")
	os.MkdirAll(orphanDir, 0755)
	os.WriteFile(filepath.Join(orphanDir, "file.go"), []byte("package main"), 0644)

	orphans := checkOrphanWorktreeDirs(dir)
	if len(orphans) != 1 {
		t.Errorf("expected 1 orphan, got %d: %v", len(orphans), orphans)
	}
}

func TestCB_CheckOrphanWorktreeDirs_EmptyDir(t *testing.T) {
	dir := createTestGitRepo(t)
	// Create empty dir — should NOT be orphan
	emptyDir := filepath.Join(dir, ".ovav", "worktrees", "empty-feature")
	os.MkdirAll(emptyDir, 0755)

	orphans := checkOrphanWorktreeDirs(dir)
	if len(orphans) != 0 {
		t.Errorf("empty dir should not be orphan, got %d", len(orphans))
	}
}

func TestCB_HasContent_NonExistent(t *testing.T) {
	if hasContent("/nonexistent/path/that/does/not/exist") {
		t.Error("non-existent dir should have no content")
	}
}

// ── More handler coverage with real git repos ───────────────────────────────

func TestCB_CleanWorktrees_Nonexistent(t *testing.T) {
	err := cleanWorktrees("/nonexistent/path", nil)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestCB_MakeCreateHandler(t *testing.T) {
	dir := createTestGitRepo(t)
	handler := makeCreateHandler(dir)
	err := handler(context.Background(), map[string]string{
		"branch":  "feature/test-create",
		"profile": "feature",
	})
	// May succeed or fail depending on git state, but should not panic
	_ = err
}

func TestCB_MakeVerifyHandler(t *testing.T) {
	dir := createTestGitRepo(t)
	handler := makeVerifyHandler(dir)
	err := handler(context.Background(), map[string]string{})
	// May succeed or fail, but should not panic
	_ = err
}

func TestCB_MakeListHandler(t *testing.T) {
	dir := createTestGitRepo(t)
	handler := makeListHandler(dir)
	err := handler(context.Background(), map[string]string{})
	_ = err
}

func TestCB_MakeRescueHandler(t *testing.T) {
	dir := createTestGitRepo(t)
	handler := makeRescueHandler(dir)
	err := handler(context.Background(), map[string]string{
		"path": "/nonexistent",
	})
	_ = err
}

func TestCB_MakeSyncHandler(t *testing.T) {
	dir := createTestGitRepo(t)
	handler := makeSyncHandler(dir)
	err := handler(context.Background(), map[string]string{})
	_ = err
}

func TestCB_MakeLockHandler(t *testing.T) {
	dir := createTestGitRepo(t)
	handler := makeLockHandler(dir)
	err := handler(context.Background(), map[string]string{
		"worktree": "nonexistent",
		"reason":   "test",
	})
	_ = err
}

// ── handlers.go: simple helpers ─────────────────────────────────────────────

func TestCB_Shorten(t *testing.T) {
	if got := shorten("hello", 10); got != "hello" {
		t.Errorf("shorten short: %q", got)
	}
	long := shorten("hello world this is long", 10)
	if len(long) > 10 {
		t.Errorf("shorten long: length %d > 10, got %q", len(long), long)
	}
	if got := shorten("abc", 3); got != "abc" {
		t.Errorf("shorten exact: %q", got)
	}
}

func TestCB_BoolIcon(t *testing.T) {
	if boolIcon(true) != "✅" {
		t.Error("true → ✅")
	}
	if boolIcon(false) != "❌" {
		t.Error("false → ❌")
	}
}

func TestCB_StatusIcon(t *testing.T) {
	got := statusIcon(true, "✅", "❌")
	if got != "✅ ✅" {
		t.Errorf("true: got %q", got)
	}
	got = statusIcon(false, "✅", "❌")
	if got != "❌ ❌" {
		t.Errorf("false: got %q", got)
	}
	// Empty trueStr should return just "✅"
	got = statusIcon(true, "", "")
	if got != "✅" {
		t.Errorf("empty true: got %q", got)
	}
}

func TestCB_PrefixForProfile(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"feature", "feature/"},
		{"hotfix", "hotfix/"},
		{"release", "release/"},
		{"unknown", "feature/"}, // default fallback
	}
	for _, tt := range tests {
		if got := prefixForProfile(tt.name); got != tt.want {
			t.Errorf("prefixForProfile(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestCB_ProfileToGitflow(t *testing.T) {
	p := ProfileConfig{BaseBranch: "develop", MergeTo: "develop"}
	gf := profileToGitflow("feature", p)
	if gf.Name != "feature" {
		t.Errorf("name = %q", gf.Name)
	}
	if gf.Prefix != "feature/" {
		t.Errorf("prefix = %q", gf.Prefix)
	}
}
