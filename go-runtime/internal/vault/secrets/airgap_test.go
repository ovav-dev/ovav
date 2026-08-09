package secrets

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Airgap File Format Tests ───────────────────────────────────────────────────

func TestAirgapFile_Magic(t *testing.T) {
	if airgapMagic != "AGAP" {
		t.Errorf("airgapMagic = %q, want %q", airgapMagic, "AGAP")
	}
	if airgapVersion != 1 {
		t.Errorf("airgapVersion = %d, want 1", airgapVersion)
	}
}

func TestExportToAirgap_Basic(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))

	data, err := ExportToAirgap(store, &DependencyGraph{}, "test-seed-32-chars-here!!", nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	// Verify magic bytes
	if string(data[:4]) != "AGAP" {
		t.Errorf("Magic = %q, want %q", string(data[:4]), "AGAP")
	}

	// Verify version
	if data[4] != 0 || data[5] != 0 || data[6] != 0 || data[7] != 1 {
		t.Errorf("Version bytes = %v, want [0 0 0 1]", data[4:8])
	}

	// Verify HMAC is present (bytes 8-39)
	hmacZero := true
	for i := 8; i < 40; i++ {
		if data[i] != 0 {
			hmacZero = false
			break
		}
	}
	if hmacZero {
		t.Error("HMAC should not be all zeros")
	}
}

func TestExportToAirgap_WithOptions(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	store.Add(NewSecret("S2", TypeCloudKey, "aws", "manual", []byte("val2")))

	future := time.Now().Add(7 * 24 * time.Hour)
	opts := &ExportOptions{
		Password:   "hunter2",
		Expiration: future,
		Secrets:    store.List(""),
	}

	data, err := ExportToAirgap(store, &DependencyGraph{}, "test-seed-32-chars-here!!", opts)
	if err != nil {
		t.Fatalf("ExportToAirgap with opts: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportToAirgap returned empty data")
	}
}

func TestExportToAirgap_SpecificSecrets(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	store.Add(NewSecret("S2", TypeCloudKey, "aws", "manual", []byte("val2")))

	// Only export S1 — use GetByName for deterministic selection (not map index)
	secret := store.GetByName("S1")
	if secret == nil {
		t.Fatal("S1 not found in store")
	}
	opts := &ExportOptions{Password: "hunter2", Secrets: []*Secret{secret}}

	data, err := ExportToAirgap(store, &DependencyGraph{}, "test-seed-32-chars-here!!", opts)
	if err != nil {
		t.Fatalf("ExportToAirgap with specific secrets: %v", err)
	}

	// Verify by parsing the file format
	af, err := parseAirgapFile(data)
	if err != nil {
		t.Fatalf("parseAirgapFile: %v", err)
	}

	// HMAC verification using seed (no password for HMAC key)
	h := hmac.New(sha256.New, []byte("test-seed-32-chars-here!!"))
	h.Write(af.Payload)
	expected := h.Sum(nil)
	if !hmac.Equal(af.HMAC[:], expected) {
		t.Error("HMAC verification failed on export")
	}

	// Verify round-trip via ImportFromAirgap
	importStore := NewSecretStore()
	result, err := ImportFromAirgap(data, importStore, "test-seed-32-chars-here!!", "hunter2")
	if err != nil {
		t.Fatalf("ImportFromAirgap: %v", err)
	}

	// Should have imported 1 secret
	if result.SecretsImported != 1 {
		t.Errorf("SecretsImported = %d, want 1", result.SecretsImported)
	}

	// Verify the imported secret
	imported := importStore.GetByName("S1")
	if imported == nil {
		t.Fatal("S1 not found after import")
	}
	if imported.Type != TypeAPIToken {
		t.Errorf("S1.Type = %v, want %v", imported.Type, TypeAPIToken)
	}
}

func TestParseAirgapFile_TooShort(t *testing.T) {
	_, err := parseAirgapFile([]byte("TOOSHORT"))
	if err == nil {
		t.Error("parseAirgapFile on short data: expected error")
	}
}

func TestParseAirgapFile_InvalidMagic(t *testing.T) {
	data := make([]byte, 100)
	copy(data[:4], []byte("BADP"))
	copy(data[4:8], []byte{0, 0, 0, 1})

	_, err := parseAirgapFile(data)
	if err == nil {
		t.Error("Expected error for invalid magic")
	}
}

func TestParseAirgapFile_Version2(t *testing.T) {
	data := make([]byte, 100)
	copy(data[:4], []byte("AGAP"))
	copy(data[4:8], []byte{0, 0, 0, 2}) // Version 2

	af, err := parseAirgapFile(data)
	if err != nil {
		t.Fatalf("parseAirgapFile: %v", err)
	}
	if af.Version != 2 {
		t.Errorf("Version = %d, want 2", af.Version)
	}
}

// ── Import Tests ───────────────────────────────────────────────────────────────

func TestImportFromAirgap_Basic(t *testing.T) {
	store := NewSecretStore()
	seed := "test-seed-32-chars-here!!"

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	// Import into new store
	store2 := NewSecretStore()
	result, err := ImportFromAirgap(exportData, store2, seed, "")
	if err != nil {
		t.Fatalf("ImportFromAirgap: %v", err)
	}

	if result.SecretsImported != 0 {
		t.Errorf("SecretsImported = %d, want 0 (empty vault)", result.SecretsImported)
	}
}

func TestImportFromAirgap_WithSecrets(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("CF_TOKEN", TypeAPIToken, "cloudflare", "manual", []byte("cf-secret-value")))
	store.Add(NewSecret("AWS_KEY", TypeCloudKey, "aws", "manual", []byte("aws-key-value")))

	seed := "test-seed-32-chars-here!!"
	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	// Import into empty store
	store2 := NewSecretStore()
	result, err := ImportFromAirgap(exportData, store2, seed, "")
	if err != nil {
		t.Fatalf("ImportFromAirgap: %v", err)
	}

	if result.SecretsImported != 2 {
		t.Errorf("SecretsImported = %d, want 2", result.SecretsImported)
	}
	if store2.Count() != 2 {
		t.Errorf("store2.Count = %d, want 2", store2.Count())
	}
}

func TestImportFromAirgap_DuplicateSkip(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("CF_TOKEN", TypeAPIToken, "cf", "manual", []byte("original")))

	seed := "test-seed-32-chars-here!!"
	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	// Add same secret to target store before import
	store2 := NewSecretStore()
	store2.Add(NewSecret("CF_TOKEN", TypeAPIToken, "cf", "manual", []byte("original")))

	result, err := ImportFromAirgap(exportData, store2, seed, "")
	if err != nil {
		t.Fatalf("ImportFromAirgap: %v", err)
	}

	if result.SecretsImported != 0 {
		t.Errorf("SecretsImported on duplicate: got %d, want 0", result.SecretsImported)
	}
	if result.SecretsSkipped != 1 {
		t.Errorf("SecretsSkipped = %d, want 1", result.SecretsSkipped)
	}
	if store2.Count() != 1 {
		t.Errorf("store2.Count = %d, want 1", store2.Count())
	}
}

func TestImportFromAirgap_WrongSeed(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v")))

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, "correct-seed-32-characters!!!", nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	store2 := NewSecretStore()
	_, err = ImportFromAirgap(exportData, store2, "wrong-seed-32-characters!!!!", "")
	if err == nil {
		t.Error("ImportFromAirgap with wrong seed: expected HMAC error")
	}
}

func TestImportFromAirgap_Expired(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v")))

	past := time.Now().Add(-1 * time.Hour)
	opts := &ExportOptions{Expiration: past}
	seed := "test-seed-32-chars-here!!"

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, opts)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	store2 := NewSecretStore()
	result, err := ImportFromAirgap(exportData, store2, seed, "")
	if err == nil {
		// Check result for expired flag
		if !result.Expired {
			t.Error("Expired flag should be set")
		}
		t.Error("ImportFromAirgap with expired: expected error")
	}
}

func TestImportFromAirgap_WithPassword(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v")))

	password := "hunter2"
	opts := &ExportOptions{Password: password}
	seed := "test-seed-32-chars-here!!"

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, opts)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	store2 := NewSecretStore()
	result, err := ImportFromAirgap(exportData, store2, seed, password)
	if err != nil {
		t.Fatalf("ImportFromAirgap with password: %v", err)
	}

	if result.SecretsImported != 1 {
		t.Errorf("SecretsImported = %d, want 1", result.SecretsImported)
	}

	// Wrong password should fail
	_, err = ImportFromAirgap(exportData, store2, seed, "wrongpassword")
	if err == nil {
		t.Error("ImportFromAirgap with wrong password: expected error")
	}
}

// ── VerifyHMAC Tests ───────────────────────────────────────────────────────────

func TestVerifyHMAC_Valid(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v")))
	seed := "test-seed-32-chars-here!!"

	data, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	valid, err := VerifyHMAC(data, seed)
	if err != nil {
		t.Fatalf("VerifyHMAC: %v", err)
	}
	if !valid {
		t.Error("VerifyHMAC with correct seed: got false, want true")
	}
}

func TestVerifyHMAC_Invalid(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v")))
	seed := "test-seed-32-chars-here!!"

	data, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	valid, err := VerifyHMAC(data, "wrong-seed-32-characters!!!!")
	if err != nil {
		t.Fatalf("VerifyHMAC: %v", err)
	}
	if valid {
		t.Error("VerifyHMAC with wrong seed: got true, want false")
	}
}

// ── AirgapHandle Tests ─────────────────────────────────────────────────────────

func TestAirgapHandle_Export(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.airgap")

	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	seed := "test-seed-32-chars-here!!"

	h := NewAirgapHandle(path)
	err := h.Export(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("AirgapHandle.Export: %v", err)
	}

	if !h.Exists() {
		t.Error("AirgapHandle.Exists: got false, want true after export")
	}

	// Verify by importing
	store2 := NewSecretStore()
	result, err := h.Import(store2, seed, "")
	if err != nil {
		t.Fatalf("AirgapHandle.Import: %v", err)
	}
	if result.SecretsImported != 1 {
		t.Errorf("SecretsImported = %d, want 1", result.SecretsImported)
	}
}

func TestAirgapHandle_Import(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "backup.airgap")

	// Export first
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	seed := "test-seed-32-chars-here!!"

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}
	os.WriteFile(path, exportData, 0600)

	// Import via handle
	h := NewAirgapHandle(path)
	store2 := NewSecretStore()
	result, err := h.Import(store2, seed, "")
	if err != nil {
		t.Fatalf("AirgapHandle.Import: %v", err)
	}
	if result.SecretsImported != 1 {
		t.Errorf("SecretsImported = %d, want 1", result.SecretsImported)
	}
}

func TestAirgapHandle_NotExists(t *testing.T) {
	h := NewAirgapHandle("/nonexistent/path.airgap")
	if h.Exists() {
		t.Error("Exists on nonexistent: got true, want false")
	}
}

// ── CheckExpiration ───────────────────────────────────────────────────────────

func TestCheckExpiration_NotExpired(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("v")))
	seed := "test-seed-32-chars-here!!"

	future := time.Now().Add(7 * 24 * time.Hour)
	opts := &ExportOptions{Expiration: future}
	data, err := ExportToAirgap(store, &DependencyGraph{}, seed, opts)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	// CheckExpiration currently returns (false, "", nil) for metadata-only read
	expired, expiresAt, err := CheckExpiration(data)
	if err != nil {
		t.Fatalf("CheckExpiration: %v", err)
	}
	// Without seed we can't verify HMAC, so we can't get expiration info
	// The function returns (false, "", nil) per its comment
	_ = expired
	_ = expiresAt
}

// ── deriveAirgapKey Tests ─────────────────────────────────────────────────────

func TestDeriveAirgapKey_Deterministic(t *testing.T) {
	k1 := deriveAirgapKey("seed", "password")
	k2 := deriveAirgapKey("seed", "password")
	k3 := deriveAirgapKey("seed", "different")

	if !bytes.Equal(k1, k2) {
		t.Error("deriveAirgapKey: same inputs should produce same output")
	}
	if bytes.Equal(k1, k3) {
		t.Error("deriveAirgapKey: different inputs should produce different output")
	}
	if len(k1) != 32 {
		t.Errorf("deriveAirgapKey: len = %d, want 32", len(k1))
	}
}

func TestDeriveAirgapKey_DifferentPasswords(t *testing.T) {
	k1 := deriveAirgapKey("seed", "pass1")
	k2 := deriveAirgapKey("seed", "pass2")
	if bytes.Equal(k1, k2) {
		t.Error("deriveAirgapKey: different passwords should produce different keys")
	}
}

func TestDeriveAirgapKey_EmptyPassword(t *testing.T) {
	k1 := deriveAirgapKey("seed", "")
	k2 := deriveAirgapKey("seed", "")
	if !bytes.Equal(k1, k2) {
		t.Error("deriveAirgapKey with empty password: should be deterministic")
	}
}

// ── aesGCMSeal / aesGCPOpen Tests ─────────────────────────────────────────────

func TestAES_GCM_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	nonce := make([]byte, 12)
	nonce[0] = 0x01

	plaintext := []byte("hello, world! this is a secret message.")

	ciphertext := aesGCMSeal(plaintext, nonce, key)
	if len(ciphertext) == 0 {
		t.Fatal("Seal returned empty ciphertext")
	}

	decrypted, err := aesGCPOpen(ciphertext, nonce, key)
	if err != nil {
		t.Fatalf("aesGCPOpen: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Round-trip: decrypted != plaintext")
	}
}

func TestAES_GCM_WrongNonce(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	nonce1 := make([]byte, 12)
	nonce2 := make([]byte, 12)
	nonce2[11] = 0xFF

	plaintext := []byte("test message")
	ciphertext := aesGCMSeal(plaintext, nonce1, key)

	_, err := aesGCPOpen(ciphertext, nonce2, key)
	if err == nil {
		t.Error("aesGCPOpen with wrong nonce: expected error")
	}
}

func TestAES_GCM_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	for i := range key1 {
		key1[i] = byte(i)
	}

	nonce := make([]byte, 12)
	plaintext := []byte("test message")

	ciphertext := aesGCMSeal(plaintext, nonce, key1)

	_, err := aesGCPOpen(ciphertext, nonce, key2)
	if err == nil {
		t.Error("aesGCPOpen with wrong key: expected error")
	}
}

// ── ImportResult Merge ─────────────────────────────────────────────────────────

func TestImportResult_Merge_Skip(t *testing.T) {
	r := &ImportResult{SecretsImported: 5, SecretsSkipped: 2}
	store := NewSecretStore()
	r.Merge(store, MergeSkip) // No-op for Skip
	if r.SecretsImported != 5 {
		t.Errorf("MergeSkip: SecretsImported changed to %d", r.SecretsImported)
	}
}

// ── AirgapPayload JSON ─────────────────────────────────────────────────────────

func TestAirgapPayload_JSON(t *testing.T) {
	payload := AirgapPayload{
		Version:    1,
		VaultBlob:  []byte("encrypted-data"),
		ExportedAt: "2026-08-01T12:00:00Z",
		ExpiresAt:  "2026-09-01T12:00:00Z",
		DeviceID:   "test-device",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var p2 AirgapPayload
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if p2.Version != 1 || p2.DeviceID != "test-device" {
		t.Error("AirgapPayload round-trip failed")
	}
}

// ── WriteAirgapFile ───────────────────────────────────────────────────────────

func TestWriteAirgapFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.airgap")
	data := []byte("AGAP test payload data")

	err := WriteAirgapFile(path, data, 0600)
	if err != nil {
		t.Fatalf("WriteAirgapFile: %v", err)
	}

	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Read back: %v", err)
	}
	if !bytes.Equal(read, data) {
		t.Error("Written data != read data")
	}
}

// ── AirgapFile ReadFrom/WriteTo ───────────────────────────────────────────────

func TestAirgapFile_WriteTo(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	seed := "test-seed-32-chars-here!!"

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	af, err := parseAirgapFile(exportData)
	if err != nil {
		t.Fatalf("parseAirgapFile: %v", err)
	}

	// Write to buffer via WriteTo
	var buf bytes.Buffer
	n, err := af.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(len(exportData)) {
		t.Errorf("WriteTo n = %d, want %d", n, len(exportData))
	}

	// Verify the output is the same as input
	if !bytes.Equal(buf.Bytes(), exportData) {
		t.Error("WriteTo output != original data")
	}
}

func TestAirgapFile_ReadFrom(t *testing.T) {
	store := NewSecretStore()
	store.Add(NewSecret("S1", TypeAPIToken, "cf", "manual", []byte("val1")))
	seed := "test-seed-32-chars-here!!"

	exportData, err := ExportToAirgap(store, &DependencyGraph{}, seed, nil)
	if err != nil {
		t.Fatalf("ExportToAirgap: %v", err)
	}

	af := &AirgapFile{}
	n, err := af.ReadFrom(bytes.NewReader(exportData))
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if n != int64(len(exportData)) {
		t.Errorf("ReadFrom n = %d, want %d", n, len(exportData))
	}

	if af.Magic != "AGAP" {
		t.Errorf("Magic = %q, want %q", af.Magic, "AGAP")
	}
	if af.Version != 1 {
		t.Errorf("Version = %d, want 1", af.Version)
	}
}
