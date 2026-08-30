//go:build !linux

package hostprojection

import (
	"errors"
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
