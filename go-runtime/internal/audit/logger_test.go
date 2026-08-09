package audit

import (
	"context"
	"os"
	"testing"
)

func TestOpLevelString(t *testing.T) {
	tests := []struct {
		level    OpLevel
		expected string
	}{
		{OpRead, "READ"},
		{OpWrite, "WRITE"},
		{OpAdmin, "ADMIN"},
		{OpLevel(99), "OPLEVEL(99)"},
	}
	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.expected {
			t.Errorf("OpLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestOpLevelIsValid(t *testing.T) {
	if !OpRead.IsValid() {
		t.Error("OpRead should be valid")
	}
	if !OpWrite.IsValid() {
		t.Error("OpWrite should be valid")
	}
	if !OpAdmin.IsValid() {
		t.Error("OpAdmin should be valid")
	}
	if OpLevel(-1).IsValid() {
		t.Error("OpLevel(-1) should be invalid")
	}
	if OpLevel(99).IsValid() {
		t.Error("OpLevel(99) should be invalid")
	}
}

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()
	ctx = WithActor(ctx, "test-actor")
	ctx = WithResource(ctx, "test-resource")
	ctx = WithOp(ctx, OpWrite)

	if got := ActorFrom(ctx); got != "test-actor" {
		t.Errorf("ActorFrom = %q, want %q", got, "test-actor")
	}
	if got := ResourceFrom(ctx); got != "test-resource" {
		t.Errorf("ResourceFrom = %q, want %q", got, "test-resource")
	}
	if got := OpFrom(ctx); got != OpWrite {
		t.Errorf("OpFrom = %v, want %v", got, OpWrite)
	}
}

func TestContextDefaults(t *testing.T) {
	ctx := context.Background()
	if got := ActorFrom(ctx); got != "system" {
		t.Errorf("ActorFrom(default) = %q, want %q", got, "system")
	}
	if got := ResourceFrom(ctx); got != "unknown" {
		t.Errorf("ResourceFrom(default) = %q, want %q", got, "unknown")
	}
	if got := OpFrom(ctx); got != OpRead {
		t.Errorf("OpFrom(default) = %v, want %v", got, OpRead)
	}
}

func TestNewLogger(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestLogBasic(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	ctx := WithActor(context.Background(), "unit-test")
	ctx = WithResource(ctx, "test-resource")

	err = logger.Log(ctx, OpWrite, "git_push", "ok", nil, func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Log() error = %v", err)
	}

	// Query and verify.
	entries, err := logger.Query().FilterByActor("unit-test").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least 1 log entry")
	}
}

func TestLogImmediate(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	ctx := WithActor(context.Background(), "immediate-test")

	err = logger.LogImmediate(ctx, OpRead, "status_check", "ok", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("LogImmediate() error = %v", err)
	}

	entries, err := logger.Query().FilterByActor("immediate-test").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least 1 log entry from LogImmediate")
	}
}

func TestLogFnError(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	ctx := WithActor(context.Background(), "error-test")
	wantErr := os.ErrNotExist

	err = logger.Log(ctx, OpAdmin, "file_read", "error", nil, func() error {
		return wantErr
	})
	// Log records the fn error as result string, does not return it.
	if err != nil {
		t.Errorf("Log() returned unexpected error = %v", err)
	}

	entries, err := logger.Query().FilterByActor("error-test").Execute()
	if err != nil {
		t.Errorf("Query.Execute() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Result == "ok" {
		t.Error("expected result to contain error description")
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Dir(dir))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	// Second close should be safe.
	if err := logger.Close(); err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
	// Log after close should fail.
	ctx := WithActor(context.Background(), "post-close")
	err = logger.Log(ctx, OpRead, "test", "ok", nil, nil)
	if err == nil {
		t.Error("expected error after close, got nil")
	}
}
