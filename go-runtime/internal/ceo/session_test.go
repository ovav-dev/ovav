package ceo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setTestHMAC sets the OVAV_HMAC_SECRET env var for tests.
// Required because ceoSecret() now returns an error when the env var is not set
// (no hardcoded fallback — security improvement).
func setTestHMAC(t *testing.T) {
	t.Setenv("OVAV_HMAC_SECRET", "test-hmac-secret-for-ceo-tests-only")
}

func TestCreateAndLoad(t *testing.T) {
	setTestHMAC(t)
	dir := t.TempDir()

	// Create CEO session
	if err := Create(dir, 8); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify file exists
	markerPath := filepath.Join(dir, markerRelPath)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Fatal("marker file not created")
	}

	// Load and verify
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if !s.Active {
		t.Error("expected active=true")
	}
	if s.Operator != "ceo-alexander" {
		t.Errorf("expected operator=ceo-alexander, got %s", s.Operator)
	}
	if s.Signature == "" {
		t.Error("expected non-empty signature")
	}

	// Verify expiry is in the future
	expiry, err := time.Parse(time.RFC3339, s.ExpiresAt)
	if err != nil {
		t.Fatalf("invalid expiry format: %v", err)
	}
	if time.Now().UTC().After(expiry) {
		t.Error("session already expired")
	}
}

func TestIsActive_ValidSession(t *testing.T) {
	setTestHMAC(t)
	dir := t.TempDir()
	if err := Create(dir, 8); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if !IsActive(dir) {
		t.Error("IsActive() should return true for valid session")
	}
}

func TestIsActive_NoSession(t *testing.T) {
	dir := t.TempDir()
	if IsActive(dir) {
		t.Error("IsActive() should return false with no session")
	}
}

func TestIsActive_TamperedSignature(t *testing.T) {
	setTestHMAC(t)
	dir := t.TempDir()
	if err := Create(dir, 8); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Load and tamper with the signature
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	s.Signature = "deadbeef"
	// Write the tampered session back
	if err := s.write(dir); err != nil {
		t.Fatalf("write() failed: %v", err)
	}

	if IsActive(dir) {
		t.Error("IsActive() should return false with tampered signature")
	}
}

func TestIsActive_ExpiredSession(t *testing.T) {
	setTestHMAC(t)
	dir := t.TempDir()
	if err := Create(dir, -1); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if IsActive(dir) {
		t.Error("IsActive() should return false for expired session")
	}
}

func TestRevoke(t *testing.T) {
	setTestHMAC(t)
	dir := t.TempDir()
	if err := Create(dir, 8); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if err := Revoke(dir); err != nil {
		t.Fatalf("Revoke() failed: %v", err)
	}
	if IsActive(dir) {
		t.Error("IsActive() should return false after revoke")
	}
}

func TestRevoke_NoSession(t *testing.T) {
	dir := t.TempDir()
	if err := Revoke(dir); err != nil {
		t.Fatalf("Revoke() should not fail on missing file: %v", err)
	}
}

func TestLoad_NoSession(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Error("Load() should return error when no session exists")
	}
}

func TestLoad_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	markerPath := filepath.Join(dir, markerRelPath)
	os.MkdirAll(filepath.Dir(markerPath), 0755)
	os.WriteFile(markerPath, []byte("garbage"), 0600)

	_, err := Load(dir)
	if err == nil {
		t.Error("Load() should return error for corrupted file")
	}
}

func TestSession_RoundTrip(t *testing.T) {
	setTestHMAC(t)
	dir := t.TempDir()

	// Create
	if err := Create(dir, 8); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Round-trip: load → write → load again → verify identical
	s1, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() 1 failed: %v", err)
	}

	if err := s1.write(dir); err != nil {
		t.Fatalf("write() failed: %v", err)
	}

	s2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() 2 failed: %v", err)
	}

	if s2.Signature != s1.Signature {
		t.Error("signatures should be identical after round-trip")
	}
	if s2.Operator != s1.Operator {
		t.Error("operator should be identical")
	}
}

func TestSign_Deterministic(t *testing.T) {
	setTestHMAC(t)
	s := &Session{
		Active:    true,
		Operator:  "test",
		GrantedAt: "2026-01-01T00:00:00Z",
		ExpiresAt: "2026-12-31T23:59:59Z",
		Nonce:     "abc123",
	}
	sig1, _ := s.sign()
	sig2, _ := s.sign()
	if sig1 != sig2 {
		t.Error("sign() should be deterministic")
	}
}

func TestGenNonce(t *testing.T) {
	n1, err := genNonce()
	if err != nil {
		t.Fatalf("genNonce() failed: %v", err)
	}
	n2, err := genNonce()
	if err != nil {
		t.Fatalf("genNonce() 2 failed: %v", err)
	}
	if n1 == n2 {
		t.Error("genNonce() should produce unique values")
	}
	// Verify hex format (32 chars for 16 bytes)
	if len(n1) != 32 {
		t.Errorf("nonce length = %d, want 32", len(n1))
	}
}
