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
	return planUnsupported(source, destination, allowedRoot, backupRoot, at, nil, false)
}

// PlanValidated returns an inspectable non-mutating plan off Linux after
// no-follow regular-file identity and source-content validation.
func PlanValidated(source, destination, allowedRoot, backupRoot string, at time.Time, validate SourceValidator) (*Transaction, error) {
	return planUnsupported(source, destination, allowedRoot, backupRoot, at, validate, true)
}

func planUnsupported(source, destination, allowedRoot, backupRoot string, at time.Time, validate SourceValidator, inspectSource bool) (*Transaction, error) {
	paths := []*string{&source, &destination, &allowedRoot, &backupRoot}
	for _, path := range paths {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return nil, fmt.Errorf("resolve plan path: %w", err)
		}
		*path = absolute
	}
	var sourceSHA256 string
	if inspectSource {
		var err error
		sourceSHA256, err = validateSourceNoFollow(source, validate)
		if err != nil {
			return nil, err
		}
	}
	return &Transaction{preview: Preview{
		Source: source, Destination: destination, AllowedRoot: allowedRoot, BackupRoot: backupRoot,
		SourceSHA256: sourceSHA256,
		PlannedAt:    at.UTC(), PlatformSupported: false, Durability: DurabilityUnsupported,
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

// InspectJournal is unavailable off Linux because recovery mutation is Linux-only.
func InspectJournal(_, _ string) (JournalInspection, error) {
	return JournalInspection{}, ErrUnsupported
}

// RecoverInspected is unavailable off Linux because recovery mutation is Linux-only.
func RecoverInspected(_ JournalInspection, _ JournalAuthority) (Result, error) {
	return Result{Operation: "recover", Durability: DurabilityUnsupported}, ErrUnsupported
}
