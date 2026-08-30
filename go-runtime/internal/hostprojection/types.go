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

// SourceValidator validates the exact source snapshot captured by planning.
type SourceValidator func([]byte) error

// JournalAuthority is the minimal path authority recorded by a journal.
type JournalAuthority struct {
	Source      string
	Destination string
	AllowedRoot string
	BackupRoot  string
}

// JournalIdentity identifies the inspected journal inode without exposing a
// mutable recovery token.
type JournalIdentity struct {
	Device uint64
	Inode  uint64
}

// JournalInspection is an immutable-by-API token issued after trusted journal
// inspection. Its fields can be read only through value-returning methods.
type JournalInspection struct {
	authority   JournalAuthority
	identity    JournalIdentity
	journalPath string
	backupRoot  string
	lockPath    string
	digest      string
}

// Authority returns a copy of the inspected journal authority.
func (inspection JournalInspection) Authority() JournalAuthority { return inspection.authority }

// Identity returns a copy of the inspected journal filesystem identity.
func (inspection JournalInspection) Identity() JournalIdentity { return inspection.identity }

// Digest returns the inspected journal SHA-256 digest.
func (inspection JournalInspection) Digest() string { return inspection.digest }

// JournalPath returns the exact inspected journal path.
func (inspection JournalInspection) JournalPath() string { return inspection.journalPath }
