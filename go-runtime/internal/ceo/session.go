// Package ceo manages CEO session authentication for OVAV security bypass.
//
// When a valid CEO session is active, protected branch gates, waiver
// requirements, and other security checks are automatically bypassed.
// Without a CEO session, full security enforcement applies.
//
// OVAV Signature: internal/ceo — stabilized 2026-08-02
// Security fix: ceoSecret() now requires OVAV_HMAC_SECRET env var (no fallback).
package ceo

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Marker path relative to repo root.
const markerRelPath = ".ovav/runtime/ceo_session.yaml"

// Session represents an active CEO session marker.
type Session struct {
	Active    bool   `yaml:"active"`
	GrantedAt string `yaml:"granted_at"`
	ExpiresAt string `yaml:"expires_at"`
	Operator  string `yaml:"operator"`
	Nonce     string `yaml:"nonce"`
	Signature string `yaml:"signature"`
}

// ── Public API ─────────────────────────────────────────────────────────

// IsActive returns true if a valid, non-expired CEO session marker exists.
func IsActive(repoRoot string) bool {
	s, err := Load(repoRoot)
	if err != nil {
		return false
	}
	return s.Valid()
}

// Create creates a new CEO session marker valid for ttlHours hours.
// Returns an error if signing fails or the marker cannot be written.
func Create(repoRoot string, ttlHours int) error {
	nonce, err := genNonce()
	if err != nil {
		return fmt.Errorf("ceo session: nonce generation failed: %w", err)
	}

	now := time.Now().UTC()
	s := Session{
		Active:    true,
		GrantedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(time.Duration(ttlHours) * time.Hour).Format(time.RFC3339),
		Operator:  "ceo-alexander",
		Nonce:     nonce,
	}

	sig, err := s.sign()
	if err != nil {
		return fmt.Errorf("ceo session: signing failed: %w", err)
	}
	s.Signature = sig

	return s.write(repoRoot)
}

// Revoke removes the CEO session marker.
func Revoke(repoRoot string) error {
	p := markerPath(repoRoot)
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ceo session: revoke failed: %w", err)
	}
	return nil
}

// ── Session Logic ──────────────────────────────────────────────────────

// Valid checks expiry and HMAC signature of the session.
func (s *Session) Valid() bool {
	if !s.Active {
		return false
	}

	expiry, err := time.Parse(time.RFC3339, s.ExpiresAt)
	if err != nil {
		return false
	}
	if time.Now().UTC().After(expiry) {
		return false
	}

	expected, err := s.sign()
	if err != nil {
		return false
	}

	return hmac.Equal([]byte(s.Signature), []byte(expected))
}

// ── File I/O ───────────────────────────────────────────────────────────

// Load reads and parses the CEO session marker from disk.
func Load(repoRoot string) (*Session, error) {
	data, err := os.ReadFile(markerPath(repoRoot))
	if err != nil {
		return nil, err
	}
	return unmarshal(data)
}

func markerPath(repoRoot string) string {
	return filepath.Join(repoRoot, markerRelPath)
}

func (s *Session) write(repoRoot string) error {
	p := markerPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	yaml := s.marshal()
	return os.WriteFile(p, []byte(yaml), 0600)
}

// ── HMAC Signing ───────────────────────────────────────────────────────

// ceoSecret returns the HMAC secret for CEO sessions.
// Reuses the same OVAV_HMAC_SECRET as the OWS waiver system.
//
// SECURITY: OVAV requires OVAV_HMAC_SECRET to be set in all environments.
// There is NO fallback key — if the env var is not set, this returns an error.
// This prevents a known dev key from being used in production by accident.
func ceoSecret() ([]byte, error) {
	s := os.Getenv("OVAV_HMAC_SECRET")
	if s == "" {
		return nil, fmt.Errorf("OVAV_HMAC_SECRET environment variable not set — CEO sessions require a secret. Set OVAV_HMAC_SECRET in your environment")
	}
	return []byte(s), nil
}

func (s *Session) sign() (string, error) {
	secret, err := ceoSecret()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%s:%s:%s:%s", s.Operator, s.GrantedAt, s.ExpiresAt, s.Nonce)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ── Serialization ──────────────────────────────────────────────────────

func (s *Session) marshal() string {
	return fmt.Sprintf(`# OVAV CEO Session — auto-generated, do not edit manually
active: %v
granted_at: "%s"
expires_at: "%s"
operator: "%s"
nonce: "%s"
signature: "%s"
`, s.Active, s.GrantedAt, s.ExpiresAt, s.Operator, s.Nonce, s.Signature)
}

func unmarshal(data []byte) (*Session, error) {
	s := &Session{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"`)
		switch key {
		case "active":
			s.Active = val == "true"
		case "granted_at":
			s.GrantedAt = val
		case "expires_at":
			s.ExpiresAt = val
		case "operator":
			s.Operator = val
		case "nonce":
			s.Nonce = val
		case "signature":
			s.Signature = val
		}
	}
	if s.Operator == "" {
		return nil, fmt.Errorf("ceo session: missing operator field")
	}
	return s, nil
}

// ── Utilities ──────────────────────────────────────────────────────────

func genNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
