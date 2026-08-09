// Package memory implements OVAV operational memory for Systems.
//
// Replaces the deprecated Python tools/memory/ system (55 files, 6,203 LOC)
// with a clean Go implementation. Manages the active_context_ledger.yaml:
// read, write, classify, recall, and session continuity.
//
// Architecture:
//   - Ledger: YAML-backed card store (.ovav/registry/active_context_ledger.yaml)
//   - Classifier: privacy gate for writes
//   - Recall: tag + recency-based query
//   - Governor: orchestration pipeline (classify → validate → write)
//
// Integration: session_greeting loads ledger at session start via LoadLedger().
package memory

import "time"

// PrivacyTag classifies information sensitivity.
type PrivacyTag string

const (
	PrivacyPublicProject   PrivacyTag = "public_project"
	PrivacyInternalProject PrivacyTag = "internal_project"
	PrivacySensitiveLocal  PrivacyTag = "sensitive_local"
	PrivacySecret          PrivacyTag = "secret"
	PrivacyIdentity        PrivacyTag = "identity_or_personal"
)

// CardStatus represents the lifecycle state of a ledger card.
type CardStatus string

const (
	StatusActive     CardStatus = "active"
	StatusCompleted  CardStatus = "completed"
	StatusDeprecated CardStatus = "deprecated"
	StatusPending    CardStatus = "pending_ceo"
	StatusInProgress CardStatus = "in_progress"
)

// Card is a single entry in the operational memory ledger.
type Card struct {
	ID              string     `yaml:"id" json:"id"`
	Status          CardStatus `yaml:"status" json:"status"`
	Priority        string     `yaml:"priority,omitempty" json:"priority,omitempty"`
	Type            string     `yaml:"type,omitempty" json:"type,omitempty"`
	Topic           string     `yaml:"topic,omitempty" json:"topic,omitempty"`
	Tags            []string   `yaml:"tags,flow" json:"tags,omitempty"`
	Summary         string     `yaml:"summary" json:"summary"`
	OperationalRule string     `yaml:"operational_rule" json:"operational_rule"`
	Confidence      string     `yaml:"confidence,omitempty" json:"confidence,omitempty"`
	LastConfirmed   string     `yaml:"last_confirmed,omitempty" json:"last_confirmed,omitempty"`
	Commit          string     `yaml:"commit,omitempty" json:"commit,omitempty"`
	ProposedBy      string     `yaml:"proposed_by,omitempty" json:"proposed_by,omitempty"`
	SupersededBy    string     `yaml:"superseded_by,omitempty" json:"superseded_by,omitempty"`
	DeprecatedAt    string     `yaml:"deprecated_at,omitempty" json:"deprecated_at,omitempty"`
	SessionID       string     `yaml:"session_id,omitempty" json:"session_id,omitempty"`       // autonomous session that created this card
	OperationID     string     `yaml:"operation_id,omitempty" json:"operation_id,omitempty"`   // source operation/subsystem
	EvidenceRefs    []string   `yaml:"evidence_refs,omitempty" json:"evidence_refs,omitempty"` // related evidence files or URLs

	// Internal tracking (not persisted to YAML)
	loadedAt time.Time `yaml:"-" json:"-"`
}

// Ledger is the top-level structure of active_context_ledger.yaml.
type Ledger struct {
	Version    int    `yaml:"version" json:"version"`
	Purpose    string `yaml:"purpose" json:"purpose"`
	RestoredAt string `yaml:"restored_at,omitempty" json:"restored_at,omitempty"`
	RestoredBy string `yaml:"restored_by,omitempty" json:"restored_by,omitempty"`
	Cards      []Card `yaml:"cards" json:"cards"`
	LastUpdate string `yaml:"last_updated" json:"last_updated"`

	// Internal tracking
	path     string    `yaml:"-" json:"-"`
	loadedAt time.Time `yaml:"-" json:"-"`
}

// RecallResult contains a matched card with relevance metadata.
type RecallResult struct {
	Card      Card    `json:"card"`
	Relevance float64 `json:"relevance"` // 0.0–1.0
	Reason    string  `json:"reason"`    // why this card was recalled
}

// ContextPack is a compact summary for agent context injection.
type ContextPack struct {
	Source           string    `json:"source"`
	Cards            []Card    `json:"cards"`
	OperationalRules []string  `json:"operational_rules"`
	GeneratedAt      time.Time `json:"generated_at"`
}
