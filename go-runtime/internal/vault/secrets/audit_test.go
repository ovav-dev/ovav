package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── AuditLog Tests ─────────────────────────────────────────────────────────────

func TestAuditLog_AppendAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	log, err := NewAuditLogPath(path, key)
	if err != nil {
		t.Fatalf("NewAuditLogPath: %v", err)
	}

	entry := LogEntry{
		Timestamp:  time.Now().UTC(),
		SecretID:   "sec-123",
		SecretName: "CF_TOKEN",
		Action:     AuditGet,
		Source:     "cli",
		MachineID:  "test-machine",
		Count:      1,
	}

	if err := log.Append(entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := log.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("ReadAll: got %d entries, want 1", len(entries))
	}

	if entries[0].SecretID != "sec-123" {
		t.Errorf("SecretID = %q, want %q", entries[0].SecretID, "sec-123")
	}
	if entries[0].Action != AuditGet {
		t.Errorf("Action = %v, want %v", entries[0].Action, AuditGet)
	}
	if entries[0].Source != "cli" {
		t.Errorf("Source = %q, want %q", entries[0].Source, "cli")
	}
}

func TestAuditLog_AppendOnly(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")
	key := make([]byte, 32)

	log, _ := NewAuditLogPath(path, key)

	// Append one entry and verify it persists
	entry := LogEntry{SecretID: "s1", Action: AuditAdd, Source: "cli", MachineID: "m1"}
	if err := log.Append(entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	read, err := log.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(read) != 1 {
		t.Errorf("ReadAll: got %d entries, want 1", len(read))
	}
	if read[0].SecretID != "s1" {
		t.Errorf("SecretID = %q, want %q", read[0].SecretID, "s1")
	}
}

func TestAuditLog_EmptyLog(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.log")
	key := make([]byte, 32)

	log, _ := NewAuditLogPath(path, key)
	entries, err := log.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll on empty: %v", err)
	}
	if entries != nil {
		t.Errorf("ReadAll on empty: got %v, want nil", entries)
	}
}

func TestAuditLog_AppendSetsTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")
	key := make([]byte, 32)

	log, _ := NewAuditLogPath(path, key)

	before := time.Now().UTC().Add(-time.Second)

	entry := LogEntry{SecretID: "s1", Action: AuditAdd, Source: "cli", MachineID: "m1"}
	if err := log.Append(entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	after := time.Now().UTC().Add(time.Second)

	entries, _ := log.ReadAll()
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	if entries[0].Timestamp.Before(before) || entries[0].Timestamp.After(after) {
		t.Errorf("Timestamp not set correctly: %v", entries[0].Timestamp)
	}
}

func TestAuditLog_InvalidMagic(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.log")
	key := make([]byte, 32)

	// Write unrecognizable data (not valid encrypted audit log)
	os.WriteFile(path, []byte("NOTAGAPMAGIC"), 0600)

	log, _ := NewAuditLogPath(path, key)
	entries, err := log.ReadAll()
	// Undecryptable data yields empty entries (no error, just no parsable entries)
	if err != nil {
		t.Fatalf("ReadAll: unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ReadAll: got %d entries, want 0 for invalid data", len(entries))
	}
}

// ── RotationConfig / CheckRotations Tests ───────────────────────────────────────

func TestDefaultRotationConfig(t *testing.T) {
	cfg := DefaultRotationConfig()

	if cfg[TypeAPIToken].ThresholdDays != 90 {
		t.Errorf("APIToken threshold = %d, want 90", cfg[TypeAPIToken].ThresholdDays)
	}
	if cfg[TypeOAuthCreds].ThresholdDays != 180 {
		t.Errorf("OAuthCreds threshold = %d, want 180", cfg[TypeOAuthCreds].ThresholdDays)
	}
	if cfg[TypeEncryptionKey].ThresholdDays != 90 {
		t.Errorf("EncryptionKey threshold = %d, want 90", cfg[TypeEncryptionKey].ThresholdDays)
	}
	if cfg[TypeTunnelToken].ThresholdDays != 30 {
		t.Errorf("TunnelToken threshold = %d, want 30", cfg[TypeTunnelToken].ThresholdDays)
	}
}

func TestCheckRotations_NeedsRotation(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("OldToken", TypeAPIToken, "github", "manual", []byte("value"))
	// CreatedAt is now, but we set LastUsed to 100 days ago
	old := time.Now().Add(-100 * 24 * time.Hour)
	sec.LastUsed = &old
	store.Add(sec)

	reports := CheckRotations(store)
	if len(reports) != 1 {
		t.Fatalf("CheckRotations: got %d reports, want 1", len(reports))
	}

	if !reports[0].NeedsRotation {
		t.Error("NeedsRotation: got false, want true (100 days > 90 threshold)")
	}
	if reports[0].ThresholdDays != 90 {
		t.Errorf("ThresholdDays = %d, want 90", reports[0].ThresholdDays)
	}
}

func TestCheckRotations_FreshSecret(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("FreshToken", TypeAPIToken, "github", "manual", []byte("value"))
	// CreatedAt is now (LastUsed nil), threshold is 90 days
	store.Add(sec)

	reports := CheckRotations(store)
	if reports[0].NeedsRotation {
		t.Error("NeedsRotation: got true, want false (just created)")
	}
}

func TestCheckRotations_LastUsedOverridesCreated(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("RecentlyUsed", TypeAPIToken, "github", "manual", []byte("value"))
	// Created 200 days ago
	old := time.Now().Add(-200 * 24 * time.Hour)
	sec.CreatedAt = old
	// But last used 10 days ago
	recent := time.Now().Add(-10 * 24 * time.Hour)
	sec.LastUsed = &recent
	store.Add(sec)

	reports := CheckRotations(store)
	if reports[0].NeedsRotation {
		t.Error("NeedsRotation: got true, want false (last used 10 days ago < 90)")
	}
}

// ── LogEntry JSON ─────────────────────────────────────────────────────────────

func TestLogEntry_JSON(t *testing.T) {
	entry := LogEntry{
		Timestamp:  time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		SecretID:   "sec-abc",
		SecretName: "MY_SECRET",
		Action:     AuditRotate,
		Source:     "api",
		MachineID:  "machine-1",
		Count:      5,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var e2 LogEntry
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if e2.Action != AuditRotate {
		t.Errorf("Action = %v, want %v", e2.Action, AuditRotate)
	}
	if e2.SecretName != "MY_SECRET" {
		t.Errorf("SecretName = %q, want %q", e2.SecretName, "MY_SECRET")
	}
}

// ── AuditLogPath ─────────────────────────────────────────────────────────────

func TestNewAuditLog_UsesDefaultPath(t *testing.T) {
	// Save and restore HOME
	orig := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", orig)

	key := make([]byte, 32)
	log, err := NewAuditLog(key)
	if err != nil {
		t.Fatalf("NewAuditLog: %v", err)
	}

	expected := filepath.Join(tmpDir, ".local", "share", "ovav", "secrets.audit")
	if log.path != expected {
		t.Errorf("path = %q, want %q", log.path, expected)
	}
}

func TestNewAuditLogPath_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "a", "b", "c")
	path := filepath.Join(nested, "audit.log")
	key := make([]byte, 32)

	log, err := NewAuditLogPath(path, key)
	if err != nil {
		t.Fatalf("NewAuditLogPath: %v", err)
	}
	if log == nil {
		t.Fatal("log is nil")
	}

	// Verify dir was created
	if _, err := os.Stat(nested); err != nil {
		t.Errorf("Directory not created: %v", err)
	}
}

// ── Filepath helper ───────────────────────────────────────────────────────────

func TestFilepath(t *testing.T) {
	result := Filepath("a", "b", "c")
	if result != filepath.Join("a", "b", "c") {
		t.Errorf("Filepath = %q, want %q", result, filepath.Join("a", "b", "c"))
	}
}

// ── AuditAction constants ─────────────────────────────────────────────────────

func TestAuditAction_Values(t *testing.T) {
	if AuditGet != "get" {
		t.Errorf("AuditGet = %q, want %q", AuditGet, "get")
	}
	if AuditAdd != "add" {
		t.Errorf("AuditAdd = %q, want %q", AuditAdd, "add")
	}
	if AuditRemove != "remove" {
		t.Errorf("AuditRemove = %q, want %q", AuditRemove, "remove")
	}
	if AuditList != "list" {
		t.Errorf("AuditList = %q, want %q", AuditList, "list")
	}
	if AuditHealth != "health" {
		t.Errorf("AuditHealth = %q, want %q", AuditHealth, "health")
	}
	if AuditDiscover != "discover" {
		t.Errorf("AuditDiscover = %q, want %q", AuditDiscover, "discover")
	}
	if AuditRotate != "rotate" {
		t.Errorf("AuditRotate = %q, want %q", AuditRotate, "rotate")
	}
}

// ── parseLines ───────────────────────────────────────────────────────────────

func TestAuditLog_parseLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")
	key := make([]byte, 32)
	log, _ := NewAuditLogPath(path, key)

	// Build multi-line JSON data manually (simulate newline-delimited format)
	line1, _ := json.Marshal(LogEntry{SecretID: "s1", Action: AuditAdd, Source: "cli", MachineID: "m"})
	line2, _ := json.Marshal(LogEntry{SecretID: "s2", Action: AuditGet, Source: "api", MachineID: "m"})
	data := []byte("\n" + string(line1) + "\n" + string(line2) + "\n")

	entries, err := log.parseLines(data)
	if err != nil {
		t.Fatalf("parseLines: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("parseLines: got %d entries, want 2", len(entries))
	}
}

func TestAuditLog_parseLines_SkipsMalformed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")
	key := make([]byte, 32)
	log, _ := NewAuditLogPath(path, key)

	line1, _ := json.Marshal(LogEntry{SecretID: "s1", Action: AuditAdd, Source: "cli", MachineID: "m"})
	goodLine := "\n" + string(line1) + "\n"
	badLine := "\nNOT JSON\n"
	data := []byte(goodLine + badLine)

	entries, err := log.parseLines(data)
	if err != nil {
		t.Fatalf("parseLines: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("parseLines: got %d entries, want 1 (malformed skipped)", len(entries))
	}
}

func TestAuditLog_parseLines_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")
	key := make([]byte, 32)
	log, _ := NewAuditLogPath(path, key)

	data := []byte("\n\n\n")
	entries, err := log.parseLines(data)
	if err != nil {
		t.Fatalf("parseLines: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("parseLines: got %d entries, want 0 for empty lines", len(entries))
	}
}
