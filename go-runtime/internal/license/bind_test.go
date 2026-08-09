package license

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"
)

func TestMachineID(t *testing.T) {
	id, err := MachineID()
	if err != nil {
		t.Fatalf("MachineID: %v", err)
	}
	if id == "" {
		t.Error("MachineID returned empty string")
	}
	// Machine ID should be stable
	id2, _ := MachineID()
	if id != id2 {
		t.Errorf("MachineID not stable: %q vs %q", id, id2)
	}
}

func TestDeriveKey(t *testing.T) {
	key1, err := DeriveKey("test-license-key-12345", "test-machine-id")
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	if len(key1) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(key1))
	}

	// Same inputs → same key
	key2, _ := DeriveKey("test-license-key-12345", "test-machine-id")
	if !bytes.Equal(key1, key2) {
		t.Error("same inputs should produce same key")
	}

	// Different inputs → different keys
	key3, _ := DeriveKey("different-key", "test-machine-id")
	if bytes.Equal(key1, key3) {
		t.Error("different license keys should produce different derived keys")
	}

	key4, _ := DeriveKey("test-license-key-12345", "different-machine")
	if bytes.Equal(key1, key4) {
		t.Error("different machine IDs should produce different derived keys")
	}
}

func TestDeriveKeyEmpty(t *testing.T) {
	_, err := DeriveKey("", "machine-id")
	if err == nil {
		t.Error("expected error for empty license key")
	}
}

func TestBindAndVerify(t *testing.T) {
	lic := &License{
		Key:       "ovav-test-license-2026",
		Holder:    "Test User",
		Email:     "test@ovav.dev",
		IssuedAt:  time.Now().UTC().Add(-24 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(365 * 24 * time.Hour),
		Tier:      "pro",
	}

	// Bind
	result, err := Bind(lic)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if !result.Bound {
		t.Error("expected bound=true")
	}
	if result.VaultHash == "" {
		t.Error("expected vault key hash")
	}

	// Verify with correct hash
	verify := Verify(lic, result.VaultHash)
	if !verify.Valid {
		t.Errorf("expected valid license, got: %s", verify.Message)
	}
	if verify.DaysLeft < 364 {
		t.Errorf("expected ~365 days, got %d", verify.DaysLeft)
	}

	// Verify with wrong hash
	verify2 := Verify(lic, "badhash")
	if verify2.Valid {
		t.Error("expected invalid for wrong hash")
	}
}

func TestVerifyExpired(t *testing.T) {
	lic := &License{
		Key:       "expired-key",
		Holder:    "Test",
		Email:     "test@ovav.dev",
		IssuedAt:  time.Now().UTC().Add(-400 * 24 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-30 * 24 * time.Hour), // expired 30 days ago
		Tier:      "trial",
	}

	// Bind and get the hash
	result, _ := Bind(lic)
	verify := Verify(lic, result.VaultHash)
	if verify.Valid {
		t.Error("expected expired license to be invalid")
	}
	if verify.DaysLeft >= 0 {
		t.Error("expected negative days left for expired license")
	}
}

func TestEncodeDecodeLicense(t *testing.T) {
	original := &License{
		Key:       "encode-test-key",
		Holder:    "Encode Test",
		Email:     "encode@ovav.dev",
		IssuedAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt: time.Now().UTC().Truncate(time.Second).Add(365 * 24 * time.Hour),
		Tier:      "pro",
	}

	encoded := EncodeLicenseKey(original)
	if encoded == "" {
		t.Error("encoded key is empty")
	}

	decoded, err := DecodeLicenseKey(encoded)
	if err != nil {
		t.Fatalf("DecodeLicenseKey: %v", err)
	}

	if decoded.Key != original.Key {
		t.Errorf("key mismatch: %q vs %q", decoded.Key, original.Key)
	}
	if decoded.Holder != original.Holder {
		t.Errorf("holder mismatch: %q vs %q", decoded.Holder, original.Holder)
	}
	if decoded.Email != original.Email {
		t.Errorf("email mismatch: %q vs %q", decoded.Email, original.Email)
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	// Verify key derivation is deterministic across calls
	k1, _ := DeriveKey("abc", "def")
	k2, _ := DeriveKey("abc", "def")

	if !bytes.Equal(k1, k2) {
		t.Error("DeriveKey must be deterministic")
	}

	hash := sha256.Sum256(k1)
	if len(hash) != 32 {
		t.Errorf("expected 32-byte hash, got %d", len(hash))
	}
}
