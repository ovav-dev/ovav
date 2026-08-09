package audit

import (
	"context"
	"testing"
	"time"
)

func TestQueryBuilder(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	// Seed 3 log entries.
	ctx1 := WithActor(context.Background(), "alice")
	logger.LogImmediate(ctx1, OpRead, "status", "ok", nil)

	ctx2 := WithActor(context.Background(), "bob")
	logger.LogImmediate(ctx2, OpWrite, "git_push", "ok", nil)

	ctx3 := WithActor(context.Background(), "alice")
	logger.LogImmediate(ctx3, OpAdmin, "vault_unlock", "ok", nil)

	// Filter by actor.
	aliceEntries, err := logger.Query().FilterByActor("alice").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(aliceEntries) != 2 {
		t.Errorf("alice entries = %d, want 2", len(aliceEntries))
	}

	// Filter by op.
	writeEntries, err := logger.Query().FilterByOp("WRITE").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(writeEntries) != 1 {
		t.Errorf("WRITE entries = %d, want 1", len(writeEntries))
	}

	// Combined filter — alice + WRITE = 0.
	aliceWrite, err := logger.Query().FilterByActor("alice").FilterByOp("WRITE").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(aliceWrite) != 0 {
		t.Errorf("alice+WRITE entries = %d, want 0", len(aliceWrite))
	}

	// Filter by time range.
	now := time.Now().UTC()
	hourAgo := now.Add(-1 * time.Hour)
	timeEntries, err := logger.Query().FilterByTimeRange(hourAgo, now.Add(1*time.Hour)).Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(timeEntries) < 3 {
		t.Errorf("time-range entries = %d, want >=3", len(timeEntries))
	}
}

func TestFilterCombinations(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	// Empty query on fresh logger — empty result.
	entries, err := logger.Query().Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("empty query entries = %d, want 0", len(entries))
	}

	// Non-existent actor.
	entries, err = logger.Query().FilterByActor("nobody").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("nobody entries = %d, want 0", len(entries))
	}
}
