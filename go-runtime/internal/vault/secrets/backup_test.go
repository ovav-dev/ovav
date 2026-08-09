package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── BackupFormat ──────────────────────────────────────────────────────────────

func TestBackupFormat_JSON(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	bf := BackupFormat{
		Version:     1,
		App:         "ovav-vault-secrets",
		CreatedAt:   now,
		MachineID:   "device-123",
		SecretCount: 3,
		Data:        []byte("encrypted"),
	}

	data, err := json.Marshal(bf)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var bf2 BackupFormat
	if err := json.Unmarshal(data, &bf2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if bf2.Version != 1 {
		t.Errorf("Version = %d, want 1", bf2.Version)
	}
	if bf2.MachineID != "device-123" {
		t.Errorf("MachineID = %q, want %q", bf2.MachineID, "device-123")
	}
	if bf2.SecretCount != 3 {
		t.Errorf("SecretCount = %d, want 3", bf2.SecretCount)
	}
}

func TestBackupFormat_Fields(t *testing.T) {
	now := time.Now()
	bf := BackupFormat{
		Version:     1,
		App:         "ovav-vault-secrets",
		CreatedAt:   now,
		MachineID:   "test-machine",
		SecretCount: 5,
		Data:        []byte("blob"),
	}

	if bf.App != "ovav-vault-secrets" {
		t.Errorf("App = %q, want %q", bf.App, "ovav-vault-secrets")
	}
	if bf.SecretCount != 5 {
		t.Errorf("SecretCount = %d, want 5", bf.SecretCount)
	}
}

// ── Backup round-trip ────────────────────────────────────────────────────────

func TestBackup_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("CF_TOKEN", TypeAPIToken, "cf", "manual", []byte("cf-secret")))
	store.Add(NewSecret("AWS_KEY", TypeCloudKey, "aws", "manual", []byte("aws-key")))

	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	err := Backup(store, vaultKey, path)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// Read the backup file
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var bf BackupFormat
	if err := json.Unmarshal(data, &bf); err != nil {
		t.Fatalf("Unmarshal backup: %v", err)
	}

	if bf.Version != 1 {
		t.Errorf("Version = %d, want 1", bf.Version)
	}
	if bf.SecretCount != 2 {
		t.Errorf("SecretCount = %d, want 2", bf.SecretCount)
	}
	if bf.MachineID == "" {
		t.Error("MachineID should be set")
	}
}

func TestBackup_Restore(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("CF_TOKEN", TypeAPIToken, "cf", "manual", []byte("cf-secret")))
	store.Add(NewSecret("AWS_KEY", TypeCloudKey, "aws", "manual", []byte("aws-key")))

	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	err := Backup(store, vaultKey, path)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// Restore
	restored, err := Restore(path, vaultKey)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if restored.Count() != 2 {
		t.Errorf("Restored store: Count = %d, want 2", restored.Count())
	}

	restoredSec := restored.GetByName("CF_TOKEN")
	if restoredSec == nil {
		t.Fatal("CF_TOKEN not found in restored store")
	}
	// Value is not serialized (json:"-"), so Hash is the verifiable field
	if restoredSec.Hash == "" {
		t.Error("CF_TOKEN Hash should not be empty after restore")
	}
	// Hash should match the hash of the original value
	expectedHash := ComputeHash([]byte("cf-secret"))
	if restoredSec.Hash != expectedHash {
		t.Errorf("CF_TOKEN Hash = %q, want %q", restoredSec.Hash, expectedHash)
	}
}

func TestBackup_Restore_WrongKey(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val")))

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1)
	}

	Backup(store, key1, path)

	_, err := Restore(path, key2)
	if err == nil {
		t.Error("Restore with wrong key: expected error")
	}
}

func TestBackup_Restore_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	os.WriteFile(path, []byte("not json at all"), 0600)

	key := make([]byte, 32)
	_, err := Restore(path, key)
	if err == nil {
		t.Error("Restore with invalid JSON: expected error")
	}
}

func TestBackup_Restore_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "badversion.json")

	bf := BackupFormat{
		Version:     999,
		App:         "ovav-vault-secrets",
		CreatedAt:   time.Now(),
		MachineID:   "test",
		SecretCount: 0,
		Data:        []byte("something"),
	}
	data, _ := json.Marshal(bf)
	os.WriteFile(path, data, 0600)

	key := make([]byte, 32)
	_, err := Restore(path, key)
	if err == nil {
		t.Error("Restore with unsupported version: expected error")
	}
}

// ── BackupInfo ───────────────────────────────────────────────────────────────

func TestBackupInfo(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val")))
	store.Add(NewSecret("S2", TypeCloudKey, "aws", "manual", []byte("val2")))
	store.Add(NewSecret("S3", TypeOAuthCreds, "github", "manual", []byte("val3")))

	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	Backup(store, vaultKey, path)

	info, err := BackupInfo(path)
	if err != nil {
		t.Fatalf("BackupInfo: %v", err)
	}

	if info.Version != 1 {
		t.Errorf("Version = %d, want 1", info.Version)
	}
	if info.SecretCount != 3 {
		t.Errorf("SecretCount = %d, want 3", info.SecretCount)
	}
	if info.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestBackupInfo_FileNotFound(t *testing.T) {
	_, err := BackupInfo("/nonexistent/path/backup.json")
	if err == nil {
		t.Error("BackupInfo nonexistent file: expected error")
	}
}

func TestBackupInfo_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(path, []byte("not json"), 0600)

	_, err := BackupInfo(path)
	if err == nil {
		t.Error("BackupInfo with invalid JSON: expected error")
	}
}

// ── Backup empty store ───────────────────────────────────────────────────────

func TestBackup_EmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.json")

	store := NewSecretStore()
	vaultKey := make([]byte, 32)

	err := Backup(store, vaultKey, path)
	if err != nil {
		t.Fatalf("Backup empty store: %v", err)
	}

	restored, err := Restore(path, vaultKey)
	if err != nil {
		t.Fatalf("Restore empty store: %v", err)
	}

	if restored.Count() != 0 {
		t.Errorf("Restored empty store: Count = %d, want 0", restored.Count())
	}
}

// ── Backup multiple secret types ───────────────────────────────────────────────

func TestBackup_SecretTypes(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("API_TOKEN", TypeAPIToken, "cf", "manual", []byte("token")))
	store.Add(NewSecret("OAUTH", TypeOAuthCreds, "github", "oauth", []byte("oauth-secret")))
	store.Add(NewSecret("DB", TypeDBCredential, "db", "manual", []byte("db-password")))
	store.Add(NewSecret("CLOUD", TypeCloudKey, "aws", "manual", []byte("cloud-key")))
	store.Add(NewSecret("ENCRYPT", TypeEncryptionKey, "local", "manual", []byte("enc-key")))
	store.Add(NewSecret("USER", TypeUserSecret, "local", "manual", []byte("user-secret")))
	store.Add(NewSecret("TUNNEL", TypeTunnelToken, "cf", "tunnel", []byte("tunnel-token")))

	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	Backup(store, vaultKey, path)

	restored, err := Restore(path, vaultKey)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if restored.Count() != 7 {
		t.Errorf("Restored count = %d, want 7", restored.Count())
	}
}

// ── Backup corrupted data ──────────────────────────────────────────────────────

func TestBackup_Restore_CorruptedData(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "corrupted.json")

	bf := BackupFormat{
		Version:     1,
		App:         "ovav-vault-secrets",
		CreatedAt:   time.Now(),
		MachineID:   "test",
		SecretCount: 1,
		Data:        []byte("not-valid-base64!!!"),
	}
	data, _ := json.Marshal(bf)
	os.WriteFile(path, data, 0600)

	key := make([]byte, 32)
	_, err := Restore(path, key)
	if err == nil {
		t.Error("Restore with corrupted data: expected error")
	}
}

// ── BackupInfo does not decrypt ───────────────────────────────────────────────

func TestBackupInfo_DoesNotDecrypt(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val")))

	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	Backup(store, vaultKey, path)

	// BackupInfo should work without the key
	info, err := BackupInfo(path)
	if err != nil {
		t.Fatalf("BackupInfo: %v", err)
	}
	if info.SecretCount != 1 {
		t.Errorf("SecretCount = %d, want 1", info.SecretCount)
	}
}

// ── Backup path handling ──────────────────────────────────────────────────────

func TestBackup_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "a", "b", "c")
	path := filepath.Join(nested, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val")))
	vaultKey := make([]byte, 32)

	err := Backup(store, vaultKey, path)
	if err != nil {
		t.Fatalf("Backup with nested path: %v", err)
	}

	if _, statErr := os.Stat(nested); statErr != nil {
		t.Errorf("Directory not created: %v", statErr)
	}
}

// ── Backup with zero key ─────────────────────────────────────────────────────

func TestBackup_Restore_ZeroKey(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val")))

	zeroKey := make([]byte, 32) // All zeros

	err := Backup(store, zeroKey, path)
	if err != nil {
		t.Fatalf("Backup with zero key: %v", err)
	}

	restored, err := Restore(path, zeroKey)
	if err != nil {
		t.Fatalf("Restore with zero key: %v", err)
	}

	if restored.Count() != 1 {
		t.Errorf("Restored count = %d, want 1", restored.Count())
	}
}

// ── Backup metadata preserved ─────────────────────────────────────────────────

func TestBackup_MetadataPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	sec := NewSecret("CF_TOKEN", TypeAPIToken, "cloudflare", "manual", []byte("secret-value"))
	sec.Tags = []string{"production", "critical"}
	store.Add(sec)

	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	Backup(store, vaultKey, path)

	restored, _ := Restore(path, vaultKey)
	restoredSec := restored.GetByName("CF_TOKEN")

	// Value is not serialized, but Hash should match
	expectedHash := ComputeHash([]byte("secret-value"))
	if restoredSec.Hash != expectedHash {
		t.Errorf("Hash = %q, want %q", restoredSec.Hash, expectedHash)
	}
	// Tags should be preserved
	if len(restoredSec.Tags) != 2 {
		t.Errorf("Tags len = %d, want 2", len(restoredSec.Tags))
	}
}

// ── WriteBackup ( Backup is the actual function name) ─────────────────────────

func TestBackupInfo_JSONRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.json")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val")))
	vaultKey := make([]byte, 32)
	for i := range vaultKey {
		vaultKey[i] = byte(i)
	}

	Backup(store, vaultKey, path)

	info, err := BackupInfo(path)
	if err != nil {
		t.Fatalf("BackupInfo: %v", err)
	}

	// Verify JSON round-trip on BackupFormat
	data, _ := json.Marshal(info)
	var info2 BackupFormat
	json.Unmarshal(data, &info2)

	if info2.Version != info.Version {
		t.Errorf("Version round-trip mismatch")
	}
	if info2.SecretCount != info.SecretCount {
		t.Errorf("SecretCount round-trip mismatch")
	}
}
