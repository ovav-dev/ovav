//go:build !linux

package hostprojection

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// Transaction is a fail-closed plan on non-Linux platforms.
type Transaction struct {
	mu      sync.Mutex
	preview Preview
}

// Plan returns inspectable metadata but performs no mutation or unsafe validation.
func Plan(source, destination, allowedRoot, backupRoot string, at time.Time) (*Transaction, error) {
	paths := []*string{&source, &destination, &allowedRoot, &backupRoot}
	for _, path := range paths {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return nil, fmt.Errorf("resolve plan path: %w", err)
		}
		*path = absolute
	}
	return &Transaction{preview: Preview{
		Source: source, Destination: destination, AllowedRoot: allowedRoot, BackupRoot: backupRoot,
		PlannedAt: at.UTC(), PlatformSupported: false, Durability: DurabilityUnsupported,
		DurabilityDetail: ErrUnsupported.Error(),
	}}, nil
}

func (t *Transaction) Preview() Preview { t.mu.Lock(); defer t.mu.Unlock(); return t.preview }

func (t *Transaction) Apply() (Result, error) {
	return Result{Operation: "apply", Durability: DurabilityUnsupported}, ErrUnsupported
}

func (t *Transaction) Rollback() (Result, error) {
	return Result{Operation: "rollback", Durability: DurabilityUnsupported}, ErrUnsupported
}

// Recover fails closed because host mutation is Linux-only.
func Recover(_, _, _ string) (Result, error) {
	return Result{Operation: "recover", Durability: DurabilityUnsupported}, ErrUnsupported
}
