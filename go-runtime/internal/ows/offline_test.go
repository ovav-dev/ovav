package ows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// offline_test.go — Tests for offline.go, resolve.go, driver.go
// F7: Offline-First Operations + F8: AI Conflict Resolution
// ═══════════════════════════════════════════════════════════════════════════

// ── Helper: open a temp offline queue ──────────────────────────────────────

func openTestQueue(t *testing.T) *OfflineQueue {
	t.Helper()
	dir := t.TempDir()
	q, err := OpenOfflineQueue(dir)
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}
	t.Cleanup(func() { q.Close() })
	return q
}

// ═══════════════════════════════════════════════════════════════════════════
// offline.go — OfflineQueue
// ═══════════════════════════════════════════════════════════════════════════

func TestOpenOfflineQueue_CreatesSchema(t *testing.T) {
	dir := t.TempDir()
	q, err := OpenOfflineQueue(dir)
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}
	defer q.Close()

	// Verify the DB file was created
	dbPath := filepath.Join(dir, ".ovav", "ows", "offline_queue.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected DB file at %s", dbPath)
	}
}

func TestOpenOfflineQueue_Idempotent(t *testing.T) {
	dir := t.TempDir()

	q1, err := OpenOfflineQueue(dir)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	q1.Close()

	// Second open should succeed (schema already exists)
	q2, err := OpenOfflineQueue(dir)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer q2.Close()
}

func TestEnqueue_Basic(t *testing.T) {
	q := openTestQueue(t)

	op, err := q.Enqueue("push", `{"branch":"feature/x"}`, "test-secret")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	if op.ID == "" {
		t.Error("op.ID should not be empty")
	}
	if op.OpType != "push" {
		t.Errorf("OpType = %q, want push", op.OpType)
	}
	if op.Payload != `{"branch":"feature/x"}` {
		t.Errorf("Payload = %q, want raw JSON", op.Payload)
	}
	if op.HMAC == "" {
		t.Error("HMAC should not be empty")
	}
	if op.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", op.RetryCount)
	}
	if op.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestEnqueue_UniqueIDs(t *testing.T) {
	q := openTestQueue(t)

	op1, _ := q.Enqueue("push", "{}", "s")
	op2, _ := q.Enqueue("merge", "{}", "s")

	if op1.ID == op2.ID {
		t.Errorf("IDs should be unique: %s == %s", op1.ID, op2.ID)
	}
}

func TestDequeueAll_Empty(t *testing.T) {
	q := openTestQueue(t)

	ops, err := q.DequeueAll()
	if err != nil {
		t.Fatalf("DequeueAll: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("expected 0 ops, got %d", len(ops))
	}
}

func TestDequeueAll_ReturnsAll(t *testing.T) {
	q := openTestQueue(t)

	q.Enqueue("push", "{}", "s")
	q.Enqueue("merge", "{}", "s")
	q.Enqueue("sync", "{}", "s")

	ops, err := q.DequeueAll()
	if err != nil {
		t.Fatalf("DequeueAll: %v", err)
	}
	if len(ops) != 3 {
		t.Errorf("expected 3 ops, got %d", len(ops))
	}
}

func TestDequeueAll_Ordered(t *testing.T) {
	q := openTestQueue(t)

	q.Enqueue("first", "{}", "s")
	q.Enqueue("second", "{}", "s")
	q.Enqueue("third", "{}", "s")

	ops, err := q.DequeueAll()
	if err != nil {
		t.Fatalf("DequeueAll: %v", err)
	}
	if len(ops) < 3 {
		t.Fatalf("expected 3 ops, got %d", len(ops))
	}
	// Ordered by created_at ASC — first inserted is first
	if ops[0].OpType != "first" {
		t.Errorf("ops[0].OpType = %q, want first", ops[0].OpType)
	}
	if ops[2].OpType != "third" {
		t.Errorf("ops[2].OpType = %q, want third", ops[2].OpType)
	}
}

func TestDequeueAll_PreservesHMAC(t *testing.T) {
	q := openTestQueue(t)
	secret := "my-secret"

	enqueued, _ := q.Enqueue("push", `{"data":1}`, secret)

	ops, _ := q.DequeueAll()
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].HMAC != enqueued.HMAC {
		t.Errorf("HMAC mismatch: enqueued=%s dequeued=%s", enqueued.HMAC, ops[0].HMAC)
	}
}

func TestVerifyHMAC_ValidSecret(t *testing.T) {
	q := openTestQueue(t)
	secret := "test-hmac-secret"

	op, _ := q.Enqueue("push", `{"x":1}`, secret)

	if !op.VerifyHMAC(secret) {
		t.Error("VerifyHMAC should return true with correct secret")
	}
}

func TestVerifyHMAC_WrongSecret(t *testing.T) {
	q := openTestQueue(t)

	op, _ := q.Enqueue("push", `{"x":1}`, "correct-secret")

	if op.VerifyHMAC("wrong-secret") {
		t.Error("VerifyHMAC should return false with wrong secret")
	}
}

func TestVerifyHMAC_TamperedPayload(t *testing.T) {
	q := openTestQueue(t)
	secret := "s"

	op, _ := q.Enqueue("push", `{"x":1}`, secret)

	// Tamper with payload
	op.Payload = `{"x":999}`

	if op.VerifyHMAC(secret) {
		t.Error("VerifyHMAC should return false after payload tampering")
	}
}

func TestVerifyHMAC_TamperedOpType(t *testing.T) {
	q := openTestQueue(t)
	secret := "s"

	op, _ := q.Enqueue("push", "{}", secret)
	op.OpType = "merge"

	if op.VerifyHMAC(secret) {
		t.Error("VerifyHMAC should return false after OpType tampering")
	}
}

func TestComplete_RemovesOp(t *testing.T) {
	q := openTestQueue(t)

	op, _ := q.Enqueue("push", "{}", "s")

	err := q.Complete(op.ID)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	ops, _ := q.DequeueAll()
	if len(ops) != 0 {
		t.Errorf("expected 0 ops after Complete, got %d", len(ops))
	}
}

func TestComplete_NonexistentID(t *testing.T) {
	q := openTestQueue(t)

	// Should not error even if ID doesn't exist
	err := q.Complete("nonexistent-id")
	if err != nil {
		t.Errorf("Complete with nonexistent ID should not error: %v", err)
	}
}

func TestIncrementRetry(t *testing.T) {
	q := openTestQueue(t)

	op, _ := q.Enqueue("push", "{}", "s")

	if err := q.IncrementRetry(op.ID); err != nil {
		t.Fatalf("IncrementRetry: %v", err)
	}

	ops, _ := q.DequeueAll()
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", ops[0].RetryCount)
	}
}

func TestIncrementRetry_Multiple(t *testing.T) {
	q := openTestQueue(t)

	op, _ := q.Enqueue("push", "{}", "s")

	q.IncrementRetry(op.ID)
	q.IncrementRetry(op.ID)
	q.IncrementRetry(op.ID)

	ops, _ := q.DequeueAll()
	if ops[0].RetryCount != 3 {
		t.Errorf("RetryCount = %d, want 3", ops[0].RetryCount)
	}
}

func TestClose_NoError(t *testing.T) {
	dir := t.TempDir()
	q, err := OpenOfflineQueue(dir)
	if err != nil {
		t.Fatalf("OpenOfflineQueue: %v", err)
	}

	if err := q.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// offline.go — HMAC Utilities
// ═══════════════════════════════════════════════════════════════════════════

func TestSignHMAC_Deterministic(t *testing.T) {
	sig1 := signHMAC("hello world", "secret")
	sig2 := signHMAC("hello world", "secret")

	if sig1 != sig2 {
		t.Errorf("same inputs should produce same HMAC: %s vs %s", sig1, sig2)
	}
}

func TestSignHMAC_DifferentData(t *testing.T) {
	sig1 := signHMAC("data-a", "secret")
	sig2 := signHMAC("data-b", "secret")

	if sig1 == sig2 {
		t.Error("different data should produce different HMACs")
	}
}

func TestSignHMAC_DifferentSecret(t *testing.T) {
	sig1 := signHMAC("data", "secret-1")
	sig2 := signHMAC("data", "secret-2")

	if sig1 == sig2 {
		t.Error("different secrets should produce different HMACs")
	}
}

func TestSignHMAC_OutputFormat(t *testing.T) {
	sig := signHMAC("test", "key")

	// SHA-256 HMAC → 64 hex chars
	if len(sig) != 64 {
		t.Errorf("HMAC hex length = %d, want 64", len(sig))
	}
	// All hex chars
	for _, c := range sig {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("non-hex char %q in HMAC output", string(c))
		}
	}
}

func TestGetHMACSecret_NotEmpty(t *testing.T) {
	secret := getHMACSecret()
	if secret == "" {
		t.Error("getHMACSecret should return non-empty string")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// resolve.go — AI Conflict Resolution
// ═══════════════════════════════════════════════════════════════════════════

func TestPlanResolution_SafeFile(t *testing.T) {
	matrix := &ConflictMatrix{
		Files: []ConflictPrediction{
			{
				FilePath:     "safe.go",
				Status:       "safe",
				SourceRanges: []LineRange{{Start: 1, End: 5}},
				TargetRanges: []LineRange{{Start: 10, End: 15}},
			},
		},
	}

	plan := PlanResolution(matrix)

	if len(plan.Resolutions) != 1 {
		t.Fatalf("expected 1 resolution, got %d", len(plan.Resolutions))
	}

	r := plan.Resolutions[0]
	if r.Strategy != ResolveUnion {
		t.Errorf("strategy = %q, want union", r.Strategy)
	}
	if r.Confidence != 0.95 {
		t.Errorf("confidence = %.2f, want 0.95", r.Confidence)
	}
	if r.Reasoning == "" {
		t.Error("reasoning should not be empty")
	}
	if plan.AutoApplied != 1 {
		t.Errorf("AutoApplied = %d, want 1", plan.AutoApplied)
	}
	if plan.NeedsReview != 0 {
		t.Errorf("NeedsReview = %d, want 0", plan.NeedsReview)
	}
}

func TestPlanResolution_ConlictSmall(t *testing.T) {
	matrix := &ConflictMatrix{
		Files: []ConflictPrediction{
			{
				FilePath:      "conflict.go",
				Status:        "conflict",
				OverlapRanges: []LineRange{{Start: 5, End: 10}}, // 1 range ≤ 2
			},
		},
	}

	plan := PlanResolution(matrix)

	r := plan.Resolutions[0]
	if r.Strategy != ResolvePreferSource {
		t.Errorf("strategy = %q, want prefer_source", r.Strategy)
	}
	if r.Confidence != 0.6 {
		t.Errorf("confidence = %.2f, want 0.6", r.Confidence)
	}
	if plan.NeedsReview != 1 {
		t.Errorf("NeedsReview = %d, want 1", plan.NeedsReview)
	}
}

func TestPlanResolution_ConflictLarge(t *testing.T) {
	matrix := &ConflictMatrix{
		Files: []ConflictPrediction{
			{
				FilePath: "big_conflict.go",
				Status:   "conflict",
				OverlapRanges: []LineRange{
					{Start: 1, End: 5},
					{Start: 10, End: 15},
					{Start: 20, End: 25}, // 3 ranges > 2
				},
			},
		},
	}

	plan := PlanResolution(matrix)

	r := plan.Resolutions[0]
	if r.Strategy != ResolveInteractive {
		t.Errorf("strategy = %q, want interactive", r.Strategy)
	}
	if r.Confidence != 0.3 {
		t.Errorf("confidence = %.2f, want 0.3", r.Confidence)
	}
	if !strings.Contains(r.Reasoning, "interactive") {
		t.Errorf("reasoning should mention interactive: %s", r.Reasoning)
	}
}

func TestPlanResolution_SourceOnly(t *testing.T) {
	matrix := &ConflictMatrix{
		Files: []ConflictPrediction{
			{FilePath: "src_only.go", Status: "source_only"},
		},
	}

	plan := PlanResolution(matrix)

	r := plan.Resolutions[0]
	if r.Strategy != ResolveUnion {
		t.Errorf("strategy = %q, want union", r.Strategy)
	}
	if r.Confidence != 1.0 {
		t.Errorf("confidence = %.2f, want 1.0", r.Confidence)
	}
	if plan.AutoApplied != 1 {
		t.Errorf("AutoApplied = %d, want 1", plan.AutoApplied)
	}
}

func TestPlanResolution_TargetOnly(t *testing.T) {
	matrix := &ConflictMatrix{
		Files: []ConflictPrediction{
			{FilePath: "tgt_only.go", Status: "target_only"},
		},
	}

	plan := PlanResolution(matrix)

	r := plan.Resolutions[0]
	if r.Strategy != ResolveUnion {
		t.Errorf("strategy = %q, want union", r.Strategy)
	}
	if r.Confidence != 1.0 {
		t.Errorf("confidence = %.2f, want 1.0", r.Confidence)
	}
	if !strings.Contains(r.Reasoning, "target_only") {
		t.Errorf("reasoning should mention target_only: %s", r.Reasoning)
	}
}

func TestPlanResolution_MixedFiles(t *testing.T) {
	matrix := &ConflictMatrix{
		Files: []ConflictPrediction{
			{FilePath: "safe.go", Status: "safe",
				SourceRanges: []LineRange{{Start: 1, End: 3}},
				TargetRanges: []LineRange{{Start: 10, End: 12}}},
			{FilePath: "conflict.go", Status: "conflict",
				OverlapRanges: []LineRange{{Start: 5, End: 8}}},
			{FilePath: "src.go", Status: "source_only"},
			{FilePath: "big.go", Status: "conflict",
				OverlapRanges: []LineRange{
					{Start: 1, End: 2}, {Start: 4, End: 5}, {Start: 7, End: 8},
				}},
		},
	}

	plan := PlanResolution(matrix)

	if len(plan.Resolutions) != 4 {
		t.Fatalf("expected 4 resolutions, got %d", len(plan.Resolutions))
	}
	if plan.AutoApplied != 2 {
		t.Errorf("AutoApplied = %d, want 2 (safe + source_only)", plan.AutoApplied)
	}
	if plan.NeedsReview != 2 {
		t.Errorf("NeedsReview = %d, want 2 (small conflict + big conflict)", plan.NeedsReview)
	}
	if plan.Matrix != matrix {
		t.Error("plan.Matrix should reference original matrix")
	}
}

func TestPlanResolution_EmptyMatrix(t *testing.T) {
	matrix := &ConflictMatrix{Files: []ConflictPrediction{}}

	plan := PlanResolution(matrix)

	if len(plan.Resolutions) != 0 {
		t.Errorf("expected 0 resolutions, got %d", len(plan.Resolutions))
	}
	if plan.AutoApplied != 0 {
		t.Errorf("AutoApplied = %d, want 0", plan.AutoApplied)
	}
	if plan.NeedsReview != 0 {
		t.Errorf("NeedsReview = %d, want 0", plan.NeedsReview)
	}
}

func TestResolutionPlan_Summary_AllAuto(t *testing.T) {
	plan := &ResolutionPlan{
		Resolutions: make([]ConflictResolution, 3),
		AutoApplied: 3,
		NeedsReview: 0,
	}

	summary := plan.Summary()
	if !strings.Contains(summary, "all auto-resolved") {
		t.Errorf("summary should mention all auto-resolved: %s", summary)
	}
	if !strings.Contains(summary, "0 conflicts") {
		t.Errorf("summary should mention 0 conflicts: %s", summary)
	}
}

func TestResolutionPlan_Summary_WithReview(t *testing.T) {
	plan := &ResolutionPlan{
		Resolutions: make([]ConflictResolution, 5),
		AutoApplied: 3,
		NeedsReview: 2,
	}

	summary := plan.Summary()
	if !strings.Contains(summary, "need review") {
		t.Errorf("summary should mention need review: %s", summary)
	}
	if !strings.Contains(summary, "3 auto-applied") {
		t.Errorf("summary should mention 3 auto-applied: %s", summary)
	}
	if !strings.Contains(summary, "2 need review") {
		t.Errorf("summary should mention 2 need review: %s", summary)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// resolve.go — formatRanges helper
// ═══════════════════════════════════════════════════════════════════════════

func TestFormatRanges_Empty(t *testing.T) {
	got := formatRanges(nil)
	if got != "[]" {
		t.Errorf("formatRanges(nil) = %q, want []", got)
	}

	got2 := formatRanges([]LineRange{})
	if got2 != "[]" {
		t.Errorf("formatRanges([]) = %q, want []", got2)
	}
}

func TestFormatRanges_SingleLine(t *testing.T) {
	ranges := []LineRange{{Start: 5, End: 5}}
	got := formatRanges(ranges)
	if got != "[5]" {
		t.Errorf("formatRanges = %q, want [5]", got)
	}
}

func TestFormatRanges_Range(t *testing.T) {
	ranges := []LineRange{{Start: 10, End: 20}}
	got := formatRanges(ranges)
	if got != "[10-20]" {
		t.Errorf("formatRanges = %q, want [10-20]", got)
	}
}

func TestFormatRanges_Multiple(t *testing.T) {
	ranges := []LineRange{
		{Start: 1, End: 5},
		{Start: 10, End: 10},
		{Start: 20, End: 30},
	}
	got := formatRanges(ranges)
	want := "[1-5,10,20-30]"
	if got != want {
		t.Errorf("formatRanges = %q, want %q", got, want)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// resolve.go — ResolutionStrategy constants
// ═══════════════════════════════════════════════════════════════════════════

func TestResolutionStrategy_Values(t *testing.T) {
	tests := []struct {
		strategy ResolutionStrategy
		want     string
	}{
		{ResolvePreferSource, "prefer_source"},
		{ResolvePreferTarget, "prefer_target"},
		{ResolveUnion, "union"},
		{ResolveInteractive, "interactive"},
	}

	for _, tt := range tests {
		if string(tt.strategy) != tt.want {
			t.Errorf("strategy = %q, want %q", tt.strategy, tt.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// driver.go — SQLite Driver Registration
// ═══════════════════════════════════════════════════════════════════════════

func TestDriverName_Constant(t *testing.T) {
	if DriverName != "sqlite" {
		t.Errorf("DriverName = %q, want sqlite", DriverName)
	}
}
