package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── RoleLabel ────────────────────────────────────────────────────────────────

func TestRoleLabel(t *testing.T) {
	tests := []struct {
		name string
		id   *Identity
		want string
	}{
		{"lead", &Identity{Role: "lead", Level: 4}, "LEAD · Level 4"},
		{"ceo", &Identity{Role: "ceo", Level: 5}, "CEO · Level 5"},
		{"implementer", &Identity{Role: "implementer", Level: 1}, "IMPLEMENTER · Level 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleLabel(tt.id)
			if got != tt.want {
				t.Errorf("RoleLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── WelcomeMessage ───────────────────────────────────────────────────────────

func TestWelcomeMessage(t *testing.T) {
	tests := []struct {
		name string
		id   *Identity
		want string
	}{
		{"ceo", &Identity{Name: "Thavren", Role: "ceo", Level: 5}, "Welcome, Thavren [CEO · Level 5]"},
		{"implementer", &Identity{Name: "Kael", Role: "implementer", Level: 1}, "Welcome, Kael [IMPLEMENTER · Level 1]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WelcomeMessage(tt.id)
			if got != tt.want {
				t.Errorf("WelcomeMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── HasPermission ────────────────────────────────────────────────────────────

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name       string
		id         *Identity
		permission string
		want       bool
	}{
		{"has exact", &Identity{Permissions: []string{"read", "write"}}, "read", true},
		{"missing", &Identity{Permissions: []string{"read", "write"}}, "admin", false},
		{"full_system grants any", &Identity{Permissions: []string{"full_system"}}, "anything", true},
		{"nil identity", nil, "read", false},
		{"empty permissions", &Identity{Permissions: nil}, "read", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.id, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── HasAnyPermission ─────────────────────────────────────────────────────────

func TestHasAnyPermission(t *testing.T) {
	tests := []struct {
		name        string
		id          *Identity
		permissions []string
		want        bool
	}{
		{"has one of many", &Identity{Permissions: []string{"read"}}, []string{"read", "write"}, true},
		{"has none", &Identity{Permissions: []string{"read"}}, []string{"write", "admin"}, false},
		{"full_system", &Identity{Permissions: []string{"full_system"}}, []string{"anything"}, true},
		{"empty check list", &Identity{Permissions: []string{"read"}}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasAnyPermission(tt.id, tt.permissions...)
			if got != tt.want {
				t.Errorf("HasAnyPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── FindIdentity ─────────────────────────────────────────────────────────────

func TestFindIdentity(t *testing.T) {
	reg := &Registry{Identities: []Identity{
		{ID: "kael", KeyHash: "aaa", Status: "active"},
		{ID: "vella", KeyHash: "bbb", Status: "suspended"},
	}}

	tests := []struct {
		name    string
		reg     *Registry
		hash    string
		wantID  string
		wantErr bool
	}{
		{"nil registry", nil, "abc", "", true},
		{"empty hash", reg, "", "", true},
		{"not found", reg, "zzz", "", true},
		{"inactive", reg, "bbb", "", true},
		{"active found", reg, "aaa", "kael", false},
		{"case insensitive", reg, "AAA", "kael", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := FindIdentity(tt.reg, tt.hash)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.ID != tt.wantID {
				t.Errorf("got ID %q, want %q", id.ID, tt.wantID)
			}
		})
	}
}

// ── LoadRegistry ─────────────────────────────────────────────────────────────

func writeTestRegistry(t *testing.T, root string, content string) {
	t.Helper()
	dir := filepath.Join(root, ".ovav", "registry")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "identities.yaml"), []byte(content), 0644)
}

const validRegistryYAML = `version: 1
canonical: true
updated_by: ceo
identities:
  - id: thavren
    name: Thavren
    role: ceo
    level: 5
    key_hash: abc123
    permissions:
      - full_system
    status: active
roles:
  ceo:
    level: 5
    description: Chief Executive Officer
signature:
  algorithm: HMAC-SHA256
  signed_by: ceo
  signed_at: "2025-01-01T00:00:00Z"
  value: PLACEHOLDER
`

func TestLoadRegistry_Valid(t *testing.T) {
	root := t.TempDir()
	writeTestRegistry(t, root, validRegistryYAML)

	reg, err := LoadRegistry(root)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if reg.Version != 1 {
		t.Errorf("version: got %d, want 1", reg.Version)
	}
	if len(reg.Identities) != 1 {
		t.Fatalf("identities: got %d, want 1", len(reg.Identities))
	}
	if reg.Identities[0].ID != "thavren" {
		t.Errorf("identity ID: got %q, want %q", reg.Identities[0].ID, "thavren")
	}
	if reg.Roles["ceo"].Level != 5 {
		t.Errorf("ceo role level: got %d, want 5", reg.Roles["ceo"].Level)
	}
}

func TestLoadRegistry_MissingFile(t *testing.T) {
	root := t.TempDir()
	_, err := LoadRegistry(root)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadRegistry_InvalidYAML(t *testing.T) {
	root := t.TempDir()
	writeTestRegistry(t, root, "not: [valid: yaml: {{{")
	_, err := LoadRegistry(root)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadRegistry_UnsupportedVersion(t *testing.T) {
	root := t.TempDir()
	writeTestRegistry(t, root, "version: 99\nidentities: []\n")
	_, err := LoadRegistry(root)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
	if !strings.Contains(err.Error(), "unsupported registry version") {
		t.Errorf("error should mention version: %v", err)
	}
}

// ── SignRegistry + VerifySignature ──────────────────────────────────────────

func TestSignAndVerifyRegistry(t *testing.T) {
	root := t.TempDir()
	ceoKey := []byte("test-ceo-key-32-bytes-long-paddi")

	// Write registry with placeholder signature
	writeTestRegistry(t, root, validRegistryYAML)

	// Sign it
	sig, err := SignRegistry(root, ceoKey)
	if err != nil {
		t.Fatalf("SignRegistry: %v", err)
	}
	if sig == "" {
		t.Fatal("SignRegistry returned empty signature")
	}

	// Now update the YAML with the real signature
	content := strings.Replace(validRegistryYAML, "value: PLACEHOLDER", "value: "+sig, 1)
	writeTestRegistry(t, root, content)

	// Verify
	valid, err := VerifySignature(root, ceoKey)
	if err != nil {
		t.Fatalf("VerifySignature: %v", err)
	}
	if !valid {
		t.Error("expected valid signature")
	}
}

func TestVerifySignature_WrongKey(t *testing.T) {
	root := t.TempDir()
	ceoKey := []byte("test-ceo-key-32-bytes-long-paddi")
	wrongKey := []byte("wrong-key-32-bytes-long-padding!")

	writeTestRegistry(t, root, validRegistryYAML)

	sig, _ := SignRegistry(root, ceoKey)
	content := strings.Replace(validRegistryYAML, "value: PLACEHOLDER", "value: "+sig, 1)
	writeTestRegistry(t, root, content)

	valid, err := VerifySignature(root, wrongKey)
	if err != nil {
		t.Fatalf("VerifySignature: %v", err)
	}
	if valid {
		t.Error("expected invalid signature with wrong key")
	}
}

func TestVerifySignature_Placeholder(t *testing.T) {
	root := t.TempDir()
	writeTestRegistry(t, root, validRegistryYAML)

	_, err := VerifySignature(root, []byte("key"))
	if err == nil {
		t.Error("expected error for PLACEHOLDER signature")
	}
}

func TestSignRegistry_MissingFile(t *testing.T) {
	root := t.TempDir()
	_, err := SignRegistry(root, []byte("key"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSignRegistry_NoSignatureBlock(t *testing.T) {
	root := t.TempDir()
	writeTestRegistry(t, root, "version: 1\nidentities: []\n")
	_, err := SignRegistry(root, []byte("key"))
	if err == nil {
		t.Error("expected error when signature block is missing")
	}
}

func TestSignRegistry_HMACCorrectness(t *testing.T) {
	root := t.TempDir()
	ceoKey := []byte("test-key-for-hmac-verification!!!")
	writeTestRegistry(t, root, validRegistryYAML)

	sig, err := SignRegistry(root, ceoKey)
	if err != nil {
		t.Fatalf("SignRegistry: %v", err)
	}

	// Manually compute expected HMAC
	content := validRegistryYAML
	sigMarker := "\nsignature:"
	idx := strings.Index(content, sigMarker)
	signedContent := content[:idx]
	mac := hmac.New(sha256.New, ceoKey)
	mac.Write([]byte(signedContent))
	expected := hex.EncodeToString(mac.Sum(nil))

	if sig != expected {
		t.Errorf("HMAC mismatch:\n  got:  %s\n  want: %s", sig, expected)
	}
}

// ── Audit: InitAudit / LogAudit / CloseAudit ─────────────────────────────────

func TestAuditLifecycle(t *testing.T) {
	root := t.TempDir()

	// Init
	err := InitAudit(root)
	if err != nil {
		t.Fatalf("InitAudit: %v", err)
	}

	// Log an entry
	entry := AuditEntry{
		IdentityID: "thavren",
		Name:       "Thavren",
		Role:       "ceo",
		Level:      5,
		Action:     "login",
		MachineID:  "test-machine",
		Success:    true,
	}
	err = LogAudit(entry)
	if err != nil {
		t.Fatalf("LogAudit: %v", err)
	}

	// Close
	CloseAudit()

	// Verify file was written
	auditPath := filepath.Join(root, ".ovav", "registry", "audit.jsonl")
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var parsed AuditEntry
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("parse audit line: %v", err)
	}
	if parsed.IdentityID != "thavren" {
		t.Errorf("identity_id: got %q, want %q", parsed.IdentityID, "thavren")
	}
	if parsed.Timestamp == "" {
		t.Error("timestamp should have been auto-set")
	}
}

func TestLogAudit_NotInitialized(t *testing.T) {
	// Reset global state
	auditMu.Lock()
	prevFile := auditFile
	auditFile = nil
	auditMu.Unlock()
	defer func() {
		auditMu.Lock()
		auditFile = prevFile
		auditMu.Unlock()
	}()

	err := LogAudit(AuditEntry{Action: "login"})
	if err == nil {
		t.Error("expected error when audit not initialized")
	}
}

func TestAuditMultipleEntries(t *testing.T) {
	root := t.TempDir()

	err := InitAudit(root)
	if err != nil {
		t.Fatalf("InitAudit: %v", err)
	}

	for i := 0; i < 3; i++ {
		err = LogAudit(AuditEntry{
			IdentityID: "test",
			Action:     "login",
			Success:    true,
		})
		if err != nil {
			t.Fatalf("LogAudit[%d]: %v", i, err)
		}
	}

	CloseAudit()

	auditPath := filepath.Join(root, ".ovav", "registry", "audit.jsonl")
	data, _ := os.ReadFile(auditPath)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestAuditFilePermissions(t *testing.T) {
	root := t.TempDir()

	err := InitAudit(root)
	if err != nil {
		t.Fatalf("InitAudit: %v", err)
	}
	CloseAudit()

	auditPath := filepath.Join(root, ".ovav", "registry", "audit.jsonl")
	info, err := os.Stat(auditPath)
	if err != nil {
		t.Fatalf("stat audit file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions: got %o, want %o", info.Mode().Perm(), 0600)
	}
}

// ── Audit convenience constructors ───────────────────────────────────────────

func TestNewLoginEntry(t *testing.T) {
	id := &Identity{ID: "thavren", Name: "Thavren", Role: "ceo", Level: 5}
	entry := NewLoginEntry(id, "machine-1")

	if entry.Action != "login" {
		t.Errorf("action: got %q, want %q", entry.Action, "login")
	}
	if !entry.Success {
		t.Error("login entry should be successful")
	}
	if entry.IdentityID != "thavren" {
		t.Errorf("identity_id: got %q", entry.IdentityID)
	}
	if entry.MachineID != "machine-1" {
		t.Errorf("machine_id: got %q", entry.MachineID)
	}
	if entry.Timestamp == "" {
		t.Error("timestamp should be set")
	}
}

func TestNewLogoutEntry(t *testing.T) {
	id := &Identity{ID: "kael", Name: "Kael", Role: "implementer", Level: 1}
	entry := NewLogoutEntry(id, "machine-2", 2*time.Hour+30*time.Minute)

	if entry.Action != "logout" {
		t.Errorf("action: got %q, want %q", entry.Action, "logout")
	}
	if !entry.Success {
		t.Error("logout entry should be successful")
	}
	if entry.Duration == "" {
		t.Error("duration should be set")
	}
}

func TestNewFailedLoginEntry(t *testing.T) {
	entry := NewFailedLoginEntry("machine-3", "invalid key_hash")

	if entry.Action != "login_failed" {
		t.Errorf("action: got %q, want %q", entry.Action, "login_failed")
	}
	if entry.Success {
		t.Error("failed login should not be successful")
	}
	if entry.Reason != "invalid key_hash" {
		t.Errorf("reason: got %q", entry.Reason)
	}
	if entry.IdentityID != "unknown" {
		t.Errorf("identity_id: got %q, want %q", entry.IdentityID, "unknown")
	}
}

// ── formatDuration ───────────────────────────────────────────────────────────

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes+seconds", 5*time.Minute + 15*time.Second, "5m15s"},
		{"hours+minutes", 2*time.Hour + 45*time.Minute, "2h45m"},
		{"zero", 0, "0s"},
		{"exactly one minute", time.Minute, "1m0s"},
		{"exactly one hour", time.Hour, "1h0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.d)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
