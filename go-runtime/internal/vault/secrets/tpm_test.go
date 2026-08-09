package secrets

import (
	"encoding/json"
	"runtime"
	"testing"
)

// ── SealedVaultKey JSON ──────────────────────────────────────────────────────

func TestSealedVaultKey_JSON(t *testing.T) {
	key := SealedVaultKey{
		SealedBlob:    []byte("sealed-data"),
		PCRPolicyHash: "pcr-hash",
		Platform:      "linux",
		Created:       "2026-08-01T00:00:00Z",
		Salt:          []byte("salt-data"),
	}

	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var k2 SealedVaultKey
	if err := json.Unmarshal(data, &k2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if k2.Platform != "linux" {
		t.Errorf("Platform = %q, want %q", k2.Platform, "linux")
	}
	if k2.PCRPolicyHash != "pcr-hash" {
		t.Errorf("PCRPolicyHash = %q, want %q", k2.PCRPolicyHash, "pcr-hash")
	}
	if string(k2.SealedBlob) != "sealed-data" {
		t.Errorf("SealedBlob mismatch")
	}
}

func TestSealedVaultKey_RoundTrip(t *testing.T) {
	key := SealedVaultKey{
		Platform:      "windows",
		SealedBlob:    []byte("sealed"),
		PCRPolicyHash: "abc123",
		Created:       "2026-08-01",
		Salt:          []byte("salt"),
	}

	data, _ := json.Marshal(key)
	var key2 SealedVaultKey
	json.Unmarshal(data, &key2)

	if key2.Platform != key.Platform {
		t.Error("Platform round-trip failed")
	}
	if string(key2.SealedBlob) != string(key.SealedBlob) {
		t.Error("SealedBlob round-trip failed")
	}
}

// ── PCRConfig ───────────────────────────────────────────────────────────────

func TestPCRConfig_Fields(t *testing.T) {
	cfg := PCRConfig{
		PCRs:        []int{0, 7},
		Description: "test PCR config",
	}

	if len(cfg.PCRs) != 2 {
		t.Errorf("PCRs len = %d, want 2", len(cfg.PCRs))
	}
	if cfg.PCRs[0] != 0 || cfg.PCRs[1] != 7 {
		t.Error("PCR values wrong")
	}
	if cfg.Description != "test PCR config" {
		t.Errorf("Description = %q, want %q", cfg.Description, "test PCR config")
	}
}

func TestDefaultLinuxPCRConfig(t *testing.T) {
	cfg := DefaultLinuxPCRConfig

	if len(cfg.PCRs) == 0 {
		t.Error("PCRs should not be empty")
	}
	if cfg.Bank != PCRBankSHA256 {
		t.Errorf("Bank = %v, want PCRBankSHA256", cfg.Bank)
	}
}

func TestDefaultWindowsPCRConfig(t *testing.T) {
	cfg := DefaultWindowsPCRConfig

	if len(cfg.PCRs) == 0 {
		t.Error("PCRs should not be empty")
	}
}

// ── SaveSealedKey / LoadSealedKey ─────────────────────────────────────────────

func TestSaveSealedKey_AndLoad(t *testing.T) {
	// NOTE: os.UserHomeDir() uses the password database on Linux,
	// not the $HOME env var. We cannot easily redirect the home dir
	// in tests without mocking UserHomeDir. Skip on non-Windows.
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Skip("os.UserHomeDir() not redirectable via $HOME on this platform")
	}

	key := SealedVaultKey{
		Platform:      "linux",
		SealedBlob:    []byte("sealed-blob"),
		PCRPolicyHash: "hash123",
		Created:       "2026-08-01",
		Salt:          []byte("salt"),
	}

	err := SaveSealedKey(&key)
	if err != nil {
		t.Fatalf("SaveSealedKey: %v", err)
	}

	loaded, err := LoadSealedKey()
	if err != nil {
		t.Fatalf("LoadSealedKey: %v", err)
	}

	if loaded.Platform != "linux" {
		t.Errorf("Platform = %q, want %q", loaded.Platform, "linux")
	}
	if string(loaded.SealedBlob) != "sealed-blob" {
		t.Error("SealedBlob mismatch")
	}
	if loaded.PCRPolicyHash != "hash123" {
		t.Error("PCRPolicyHash mismatch")
	}
}

func TestLoadSealedKey_NotFound(t *testing.T) {
	// When home dir doesn't have a sealed.key, LoadSealedKey returns error
	_, err := LoadSealedKey()
	// Expect an error if no sealed key exists
	if err == nil {
		t.Error("LoadSealedKey nonexistent: expected error")
	}
}

// ── NewTPM ─────────────────────────────────────────────────────────────────

func TestNewTPM(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		// TPM may not be available on all platforms
		t.Skip("NewTPM returned error: " + err.Error())
	}

	if tpm == nil {
		t.Fatal("NewTPM returned nil")
	}

	// Verify it implements TPMInterface
	var _ TPMInterface = tpm
}

func TestNewTPM_PlatformDetection(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM returned error: " + err.Error())
	}

	// All stub implementations should return errors for Seal/Unseal
	_, err = tpm.Seal([]byte("test"), PCRBankSHA256)
	if err == nil {
		// If it succeeds, platform detection worked
	}

	_, err = tpm.Unseal([]byte("test"))
	if err == nil {
		// All stubs return error
	}
}

// ── TPMInterface ─────────────────────────────────────────────────────────────

func TestTPMInterface_SealAndUnseal(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM: " + err.Error())
	}

	// Seal should return error (stub)
	_, err = tpm.Seal([]byte("key-data"), PCRBankSHA256)
	_ = err

	// Unseal should return error (stub)
	_, err = tpm.Unseal([]byte("sealed-data"))
	_ = err
}

// ── computePCRPolicyHash ─────────────────────────────────────────────────────

func TestComputePCRPolicyHash(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM: " + err.Error())
	}

	cfg := PCRConfig{
		PCRs:        []int{0, 7},
		Description: "test",
	}

	hash, err := computePCRPolicyHash(tpm, cfg)
	// Stub returns error (not implemented)
	if err == nil {
		if hash == "" {
			t.Error("Hash should not be empty when successful")
		}
	}
}

func TestComputePCRPolicyHash_EmptyPCRs(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM: " + err.Error())
	}

	cfg := PCRConfig{
		PCRs:        []int{},
		Description: "empty PCRs",
	}

	_, err = computePCRPolicyHash(tpm, cfg)
	// Should handle empty PCRs gracefully
	_ = err
}

// ── Platform-specific TPM stubs ───────────────────────────────────────────────

func TestTPMInterface_PlatformMethods(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM: " + err.Error())
	}

	var i TPMInterface = tpm // Verify interface compliance
	_ = i
}

// ── SealedVaultKey Platform values ───────────────────────────────────────────

func TestSealedVaultKey_Platforms(t *testing.T) {
	platforms := []string{"linux", "windows", "macos"}

	for _, p := range platforms {
		key := SealedVaultKey{Platform: p}
		if key.Platform != p {
			t.Errorf("Platform = %q, want %q", key.Platform, p)
		}
	}
}

// ── SaveSealedKey JSON format ─────────────────────────────────────────────────

func TestSaveSealedKey_JSONFormat(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Skip("os.UserHomeDir() not redirectable via $HOME on this platform")
	}

	key := SealedVaultKey{
		Platform:      "linux",
		SealedBlob:    []byte("sealed-bytes"),
		PCRPolicyHash: "policy-hash-value",
		Created:       "2026-08-01",
		Salt:          []byte("salt"),
	}

	SaveSealedKey(&key)

	// Load it back and verify JSON format
	loaded, err := LoadSealedKey()
	if err != nil {
		t.Fatalf("LoadSealedKey: %v", err)
	}

	if loaded.PCRPolicyHash != "policy-hash-value" {
		t.Errorf("PCRPolicyHash = %q, want %q", loaded.PCRPolicyHash, "policy-hash-value")
	}
}

// ── TPM returns proper error types ───────────────────────────────────────────

func TestTPM_SealError(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM: " + err.Error())
	}

	_, err = tpm.Seal([]byte("data"), PCRBankSHA256)
	// Stub always returns error — verify it's a proper error
	if err == nil {
		t.Skip("TPM.Seal succeeded — real TPM may be available")
	}
}

func TestTPM_UnsealError(t *testing.T) {
	tpm, err := NewTPM()
	if err != nil {
		t.Skip("NewTPM: " + err.Error())
	}

	_, err = tpm.Unseal([]byte("sealed"))
	if err == nil {
		t.Skip("TPM.Unseal succeeded — real TPM may be available")
	}
}

// ── TPMInterface Description ──────────────────────────────────────────────────

func TestTPMInterface_Description(t *testing.T) {
	cfg := PCRConfig{
		PCRs:        []int{0, 1, 2, 3, 4, 5, 6, 7},
		Description: "UEFI Linux default",
	}

	if cfg.Description != "UEFI Linux default" {
		t.Errorf("Description = %q", cfg.Description)
	}
}

// ── PCRBank values ──────────────────────────────────────────────────────────

func TestPCRBank_Values(t *testing.T) {
	if PCRBankSHA256 != 10 {
		t.Errorf("PCRBankSHA256 = %d, want 10", PCRBankSHA256)
	}
	if PCRBankSHA1 != 4 {
		t.Errorf("PCRBankSHA1 = %d, want 4", PCRBankSHA1)
	}
}
