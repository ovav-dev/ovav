// audit.go — Append-only audit log for vault access.
//
// Phase 4 of OVAV-VAULT-2026 plan.
//
// Records every vault operation in an append-only, encrypted audit log at:
//
//	~/.local/share/ovav/secrets.audit
//
// Each entry captures: timestamp, secret_id, action, source, machine_id.
// The audit log is NOT part of the main secrets vault — it has its own
// encrypted file to avoid leaking access patterns into the main vault blob.
//
// The log is append-only: LogEntry records are serialized to JSON, each
// prefixed with a newline, and appended to the file. On read, we parse
// each line as a JSON object.
package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	vaultpkg "github.com/ovav/ovav/internal/vault"
)

// AuditAction records the type of vault operation.
type AuditAction string

const (
	AuditGet      AuditAction = "get"
	AuditAdd      AuditAction = "add"
	AuditRemove   AuditAction = "remove"
	AuditList     AuditAction = "list"
	AuditHealth   AuditAction = "health"
	AuditDiscover AuditAction = "discover"
	AuditRotate   AuditAction = "rotate"
)

// LogEntry is a single audit record.
type LogEntry struct {
	Timestamp  time.Time   `json:"timestamp"`
	SecretID   string      `json:"secret_id,omitempty"` // empty for list/health
	SecretName string      `json:"secret_name,omitempty"`
	Action     AuditAction `json:"action"`
	Source     string      `json:"source"` // "cli", "api", "discover"
	MachineID  string      `json:"machine_id"`
	Count      int         `json:"count,omitempty"` // for list: number of secrets
}

// AuditLog manages the append-only audit log.
type AuditLog struct {
	mu   sync.Mutex
	path string
	key  []byte
}

// NewAuditLog opens or creates the audit log at the default path.
func NewAuditLog(key []byte) (*AuditLog, error) {
	return NewAuditLogPath(filepath.Join(os.Getenv("HOME"), ".local", "share", "ovav", "secrets.audit"), key)
}

// NewAuditLogPath opens or creates the audit log at a specific path.
func NewAuditLogPath(path string, key []byte) (*AuditLog, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("audit mkdir: %w", err)
	}
	return &AuditLog{path: path, key: key}, nil
}

// Append adds a new entry to the audit log.
// The entry is serialized to JSON and appended (encrypted) to the file.
func (a *AuditLog) Append(entry LogEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}
	// Prefix with newline for line-based parsing on read
	data = append([]byte("\n"), data...)

	ciphertext, err := vaultpkg.Encrypt(data, a.key)
	if err != nil {
		return fmt.Errorf("audit encrypt: %w", err)
	}

	f, err := os.OpenFile(a.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("audit open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(ciphertext); err != nil {
		return fmt.Errorf("audit write: %w", err)
	}

	return nil
}

// ReadAll decrypts and returns all audit log entries.
func (a *AuditLog) ReadAll() ([]LogEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ciphertext, err := os.ReadFile(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("audit read: %w", err)
	}

	if len(ciphertext) == 0 {
		return nil, nil
	}

	// Single encrypted blob — decrypt then parse lines
	plaintext, err := vaultpkg.Decrypt(ciphertext, a.key)
	if err != nil {
		// For backwards compat: might be multiple encrypted lines from old format
		return a.readLines(ciphertext)
	}

	return a.parseLines(plaintext)
}

// parseLines parses newline-delimited JSON log entries.
func (a *AuditLog) parseLines(data []byte) ([]LogEntry, error) {
	var entries []LogEntry
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// readLines reads the old line-by-line encrypted format (backwards compat).
func (a *AuditLog) readLines(ciphertext []byte) ([]LogEntry, error) {
	var entries []LogEntry
	lines := bytes.Split(ciphertext, []byte{'\n'})
	for _, line := range lines {
		if len(line) < 12 { // minimum: nonce + tag
			continue
		}
		plaintext, err := vaultpkg.Decrypt(line, a.key)
		if err != nil {
			continue // skip lines we can't decrypt
		}
		// Remove leading newline if present
		trimmed := bytes.TrimPrefix(plaintext, []byte{'\n'})
		if len(trimmed) == 0 {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal(trimmed, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Filepath is filepath.Join for audit paths.
func Filepath(elem ...string) string {
	return filepath.Join(elem...)
}

// RotationConfig holds per-type rotation thresholds.
type RotationConfig struct {
	// ThresholdDays is the number of days before a rotation reminder fires.
	ThresholdDays int
}

// DefaultRotationConfig returns rotation thresholds by secret type.
func DefaultRotationConfig() map[SecretType]RotationConfig {
	return map[SecretType]RotationConfig{
		TypeAPIToken:      {ThresholdDays: 90},
		TypeOAuthCreds:    {ThresholdDays: 180},
		TypeDBCredential:  {ThresholdDays: 180},
		TypeCloudKey:      {ThresholdDays: 365},
		TypeEncryptionKey: {ThresholdDays: 90},
		TypeUserSecret:    {ThresholdDays: 365},
		TypeTunnelToken:   {ThresholdDays: 30},
	}
}

// RotationReport holds rotation advice for one secret.
type RotationReport struct {
	SecretID      string     `json:"secret_id"`
	Name          string     `json:"name"`
	Type          SecretType `json:"type"`
	DaysSinceUsed int        `json:"days_since_used"`
	ThresholdDays int        `json:"threshold_days"`
	NeedsRotation bool       `json:"needs_rotation"`
	Reason        string     `json:"reason"`
}

// CheckRotations analyzes secrets and returns rotation advice.
func CheckRotations(store *SecretStore) []*RotationReport {
	cfg := DefaultRotationConfig()
	var reports []*RotationReport

	for _, sec := range store.List("") {
		threshold := 90 // default
		if c, ok := cfg[sec.Type]; ok {
			threshold = c.ThresholdDays
		}

		lastUsed := sec.CreatedAt
		if sec.LastUsed != nil {
			lastUsed = *sec.LastUsed
		}

		daysSince := int(time.Since(lastUsed).Hours() / 24)
		needsRotation := daysSince > threshold

		var reason string
		if needsRotation {
			reason = fmt.Sprintf("used %d days ago (threshold: %d)", daysSince, threshold)
		}

		reports = append(reports, &RotationReport{
			SecretID:      sec.ID,
			Name:          sec.Name,
			Type:          sec.Type,
			DaysSinceUsed: daysSince,
			ThresholdDays: threshold,
			NeedsRotation: needsRotation,
			Reason:        reason,
		})
	}

	return reports
}
