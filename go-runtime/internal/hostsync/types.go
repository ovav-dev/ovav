// Package hostsync maps exact repository-owned profiles onto governed host
// projection transactions. It exposes no arbitrary source or destination API.
package hostsync

import (
	"time"

	"github.com/ovav/ovav/internal/hostprojection"
)

const resultSchema = "ovav.host_sync_result.v1"

// Profile is public, immutable registry metadata.
type Profile struct {
	Name           string `json:"name"`
	SourceRelative string `json:"source_relative"`
	MigrationID    string `json:"migration_id,omitempty"`
	Windows        bool   `json:"windows"`
}

// Request selects one allowlisted plan/apply profile or an approved rollback.
type Request struct {
	RepoRoot         string
	Home             string
	WindowsHome      string
	Profile          string
	Apply            bool
	ApproveHostWrite bool
	RollbackJournal  string
	Now              time.Time
}

// Result is a stable JSON-ready projection outcome.
type Result struct {
	SchemaVersion      string                         `json:"schema_version"`
	Operation          string                         `json:"operation"`
	Mode               string                         `json:"mode"`
	Profile            string                         `json:"profile"`
	Source             string                         `json:"source,omitempty"`
	Destination        string                         `json:"destination"`
	AllowedRoot        string                         `json:"allowed_root"`
	BackupRoot         string                         `json:"backup_root"`
	BackupPath         string                         `json:"backup_path,omitempty"`
	JournalPath        string                         `json:"journal_path"`
	SourceSHA256       string                         `json:"source_sha256,omitempty"`
	OriginalSHA256     string                         `json:"original_sha256,omitempty"`
	DestinationExisted bool                           `json:"destination_existed"`
	Approved           bool                           `json:"approved"`
	WritesPerformed    bool                           `json:"writes_performed"`
	Applied            bool                           `json:"applied"`
	RolledBack         bool                           `json:"rolled_back"`
	Recovered          bool                           `json:"recovered"`
	AlreadyComplete    bool                           `json:"already_complete"`
	JournalState       string                         `json:"journal_state,omitempty"`
	Durability         hostprojection.DurabilityLevel `json:"durability"`
	DurabilityDetail   string                         `json:"durability_detail,omitempty"`
}
