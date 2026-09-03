// Package identity — Audit logging for OVAV identity events.
//
// Writes append-only JSONL to .ovav/registry/audit.jsonl.
// Each line is a self-contained JSON object. Never deleted, never modified.
// Thread-safe via sync.Mutex on file writes.

package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── Audit entry ──────────────────────────────────────────────────────────────

// AuditEntry records a single identity event (login, logout, failed attempt).
type AuditEntry struct {
	Timestamp  string `json:"timestamp"`
	IdentityID string `json:"identity_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Level      int    `json:"level"`
	Action     string `json:"action"` // "login", "logout", "login_failed"
	MachineID  string `json:"machine_id"`
	IP         string `json:"ip,omitempty"`
	Duration   string `json:"duration,omitempty"` // human-readable, logout only
	Success    bool   `json:"success"`
	Reason     string `json:"reason,omitempty"` // failure reason
}

// ── Audit logger ─────────────────────────────────────────────────────────────

const auditRelPath = AuditRelativePath

var (
	auditMu   sync.Mutex
	auditFile *os.File
	auditRoot string
)

// InitAudit opens the audit log file for appending.
// Must be called before LogAudit. Thread-safe.
func InitAudit(repoRoot string) error {
	auditMu.Lock()
	defer auditMu.Unlock()

	auditRoot = repoRoot
	path := filepath.Join(repoRoot, auditRelPath)

	// Ensure directory exists with restricted permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("audit: cannot create directory: %w", err)
	}

	// Open for append, create if not exists, write-only
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("audit: cannot open log file: %w", err)
	}

	// Ensure file permissions are correct (even if file already existed)
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return fmt.Errorf("audit: cannot set file permissions: %w", err)
	}

	auditFile = f
	return nil
}

// CloseAudit closes the audit log file.
func CloseAudit() {
	auditMu.Lock()
	defer auditMu.Unlock()

	if auditFile != nil {
		auditFile.Close()
		auditFile = nil
	}
}

// LogAudit appends an audit entry to the JSONL log.
// Thread-safe. Returns error if the file is not initialized or write fails.
func LogAudit(entry AuditEntry) error {
	auditMu.Lock()
	defer auditMu.Unlock()

	if auditFile == nil {
		return fmt.Errorf("audit: log file not initialized — call InitAudit first")
	}

	// Ensure timestamp is set
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit: JSON marshal failed: %w", err)
	}

	// Write JSON line + newline
	line = append(line, '\n')
	if _, err := auditFile.Write(line); err != nil {
		return fmt.Errorf("audit: write failed: %w", err)
	}

	// Sync to disk for durability
	if err := auditFile.Sync(); err != nil {
		return fmt.Errorf("audit: sync failed: %w", err)
	}

	return nil
}

// ── Convenience constructors ─────────────────────────────────────────────────

// NewLoginEntry creates an audit entry for a successful login.
func NewLoginEntry(id *Identity, machineID string) AuditEntry {
	return AuditEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		IdentityID: id.ID,
		Name:       id.Name,
		Role:       id.Role,
		Level:      id.Level,
		Action:     "login",
		MachineID:  machineID,
		Success:    true,
	}
}

// NewLogoutEntry creates an audit entry for a logout event.
func NewLogoutEntry(id *Identity, machineID string, duration time.Duration) AuditEntry {
	return AuditEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		IdentityID: id.ID,
		Name:       id.Name,
		Role:       id.Role,
		Level:      id.Level,
		Action:     "logout",
		MachineID:  machineID,
		Duration:   formatDuration(duration),
		Success:    true,
	}
}

// NewFailedLoginEntry creates an audit entry for a failed login attempt.
func NewFailedLoginEntry(machineID, reason string) AuditEntry {
	return AuditEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		IdentityID: "unknown",
		Name:       "unknown",
		Role:       "none",
		Level:      0,
		Action:     "login_failed",
		MachineID:  machineID,
		Success:    false,
		Reason:     reason,
	}
}

// formatDuration produces a compact human-readable duration.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
