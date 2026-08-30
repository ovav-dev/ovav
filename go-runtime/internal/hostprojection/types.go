// Package hostprojection provides governed, single-file host projections.
package hostprojection

import (
	"errors"
	"time"
)

var (
	// ErrUnsupported reports that mutation is unavailable on this platform.
	ErrUnsupported = errors.New("host projection mutation is supported only on Linux")
	// ErrLocked reports that another process owns the transaction lock.
	ErrLocked = errors.New("host projection transaction is locked")
	// ErrConcurrentChange reports that a file no longer matches its planned identity and content.
	ErrConcurrentChange = errors.New("host projection detected a concurrent change")
)

// DurabilityLevel describes whether file and directory fsync are fully supported.
type DurabilityLevel string

const (
	DurabilityFull        DurabilityLevel = "full"
	DurabilityDegraded    DurabilityLevel = "degraded"
	DurabilityUnsupported DurabilityLevel = "unsupported"
)

// Preview is immutable dry-run metadata. Plan performs no filesystem writes.
type Preview struct {
	Source             string
	Destination        string
	AllowedRoot        string
	BackupRoot         string
	BackupPath         string
	JournalPath        string
	LockPath           string
	PlannedAt          time.Time
	SourceSHA256       string
	OriginalSHA256     string
	DestinationExisted bool
	PlatformSupported  bool
	Durability         DurabilityLevel
	DurabilityDetail   string
}

// Result reports a mutation or recovery outcome, including degraded durability.
type Result struct {
	Operation        string
	Applied          bool
	RolledBack       bool
	Recovered        bool
	AlreadyComplete  bool
	JournalState     string
	JournalPath      string
	BackupPath       string
	Durability       DurabilityLevel
	DurabilityDetail string
}
