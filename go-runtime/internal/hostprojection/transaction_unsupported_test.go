//go:build !linux

package hostprojection

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMutationFailsClosedOffLinux(t *testing.T) {
	tx, err := Plan("source", "destination", "allowed", "backup", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	if tx.Preview().PlatformSupported {
		t.Fatal("PlatformSupported = true off Linux")
	}
	if _, err := tx.Apply(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Apply() error = %v, want ErrUnsupported", err)
	}
	if _, err := tx.Rollback(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Rollback() error = %v, want ErrUnsupported", err)
	}
	if _, err := Recover("journal", "allowed", "backup"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Recover() error = %v, want ErrUnsupported", err)
	}
}

func TestPlanValidatedReadsAndValidatesSourceOffLinux(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, []byte("valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	tx, err := PlanValidated(source, "destination", "allowed", "backup", time.Unix(2, 0), func(content []byte) error {
		called = true
		if string(content) != "valid" {
			t.Fatalf("validator content = %q", content)
		}
		return nil
	})
	if err != nil || !called || tx.Preview().SourceSHA256 == "" {
		t.Fatalf("PlanValidated() = %+v, called=%v, err=%v", tx, called, err)
	}
	symlink := filepath.Join(filepath.Dir(source), "source-link")
	if err := os.Symlink(source, symlink); err == nil {
		if _, err := PlanValidated(symlink, "destination", "allowed", "backup", time.Now(), func([]byte) error { return nil }); err == nil {
			t.Fatal("PlanValidated accepted symlink source")
		}
	}
}
