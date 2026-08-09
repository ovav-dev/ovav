// Package identity implements OVAV identity and access management.
//
// Loads the identity registry from .ovav/registry/identities.yaml,
// verifies HMAC-SHA256 signatures, and provides permission checks.
// The registry is the single source of truth for who can access OVAV SYSTEMS.
//
// Architecture:
//
//	seed → PBKDF2(machine_id) → vault_key → SHA256(vault_key) = key_hash
//	key_hash → lookup in registry → identity with role/level/permissions
//
// Registry is git-tracked, human-readable YAML, HMAC-signed by CEO key.
package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Types ────────────────────────────────────────────────────────────────────

// Identity represents a single authorized person in the OVAV system.
type Identity struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Email       string   `yaml:"email" json:"email,omitempty"`
	Role        string   `yaml:"role" json:"role"`
	Level       int      `yaml:"level" json:"level"`
	KeyHash     string   `yaml:"key_hash" json:"key_hash"`
	Permissions []string `yaml:"permissions" json:"permissions"`
	Status      string   `yaml:"status" json:"status"`
}

// RoleDef describes a role's level and purpose.
type RoleDef struct {
	Level       int    `yaml:"level" json:"level"`
	Description string `yaml:"description" json:"description"`
}

// Signature holds the HMAC signature metadata for the registry.
type Signature struct {
	Algorithm string `yaml:"algorithm" json:"algorithm"`
	SignedBy  string `yaml:"signed_by" json:"signed_by"`
	SignedAt  string `yaml:"signed_at" json:"signed_at"`
	Value     string `yaml:"value" json:"value"`
}

// Registry is the full identity registry loaded from YAML.
type Registry struct {
	Version    int                `yaml:"version" json:"version"`
	Canonical  bool               `yaml:"canonical" json:"canonical"`
	UpdatedBy  string             `yaml:"updated_by" json:"updated_by"`
	Identities []Identity         `yaml:"identities" json:"identities"`
	Roles      map[string]RoleDef `yaml:"roles" json:"roles"`
	Signature  Signature          `yaml:"signature" json:"signature"`
}

// ── Registry loading ─────────────────────────────────────────────────────────

const registryRelPath = ".ovav/registry/identities.yaml"

// LoadRegistry reads and parses the identity registry from the repo root.
// It does NOT verify the signature — call VerifySignature separately with the CEO key.
func LoadRegistry(repoRoot string) (*Registry, error) {
	path := filepath.Join(repoRoot, registryRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("identity: cannot read registry: %w", err)
	}

	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("identity: invalid YAML: %w", err)
	}

	if reg.Version != 1 {
		return nil, fmt.Errorf("identity: unsupported registry version %d", reg.Version)
	}

	return &reg, nil
}

// ── Identity lookup ──────────────────────────────────────────────────────────

// FindIdentity searches the registry for an identity matching the given key_hash.
// Returns the identity if found and active, or an error otherwise.
func FindIdentity(reg *Registry, keyHash string) (*Identity, error) {
	if reg == nil {
		return nil, fmt.Errorf("identity: registry is nil")
	}
	if keyHash == "" {
		return nil, fmt.Errorf("identity: key_hash is empty")
	}

	for i := range reg.Identities {
		id := &reg.Identities[i]
		if strings.EqualFold(id.KeyHash, keyHash) {
			if id.Status != "active" {
				return nil, fmt.Errorf("identity: %s is %s (not active)", id.ID, id.Status)
			}
			return id, nil
		}
	}

	return nil, fmt.Errorf("identity: no matching identity for key_hash")
}

// ── Permission checks ────────────────────────────────────────────────────────

// HasPermission checks if an identity has a specific permission.
func HasPermission(id *Identity, permission string) bool {
	if id == nil {
		return false
	}
	for _, p := range id.Permissions {
		if p == permission || p == "full_system" {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if an identity has at least one of the given permissions.
func HasAnyPermission(id *Identity, permissions ...string) bool {
	for _, p := range permissions {
		if HasPermission(id, p) {
			return true
		}
	}
	return false
}

// ── Signature verification ───────────────────────────────────────────────────

// VerifySignature verifies the HMAC-SHA256 signature of the registry.
// The ceoKey is the raw CEO vault_key (32 bytes from PBKDF2).
// Signature covers all YAML content before the "signature:" block.
func VerifySignature(repoRoot string, ceoKey []byte) (bool, error) {
	path := filepath.Join(repoRoot, registryRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("identity: cannot read registry: %w", err)
	}

	content := string(data)

	// Split at the signature block — everything before "signature:" is signed content
	sigMarker := "\nsignature:"
	idx := strings.Index(content, sigMarker)
	if idx < 0 {
		// Try at start of file (unlikely but handle it)
		if strings.HasPrefix(content, "signature:") {
			return false, fmt.Errorf("identity: no content before signature block")
		}
		return false, fmt.Errorf("identity: signature block not found")
	}

	signedContent := content[:idx]

	// Parse the full registry to get the stored signature value
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return false, fmt.Errorf("identity: invalid YAML: %w", err)
	}

	if reg.Signature.Value == "" || reg.Signature.Value == "PLACEHOLDER" {
		return false, fmt.Errorf("identity: signature is placeholder — registry not yet signed")
	}

	// Compute HMAC-SHA256(ceoKey, signedContent)
	mac := hmac.New(sha256.New, ceoKey)
	mac.Write([]byte(signedContent))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	storedSig, err := hex.DecodeString(reg.Signature.Value)
	if err != nil {
		return false, fmt.Errorf("identity: stored signature is not valid hex: %w", err)
	}

	computedSig, err := hex.DecodeString(expectedSig)
	if err != nil {
		return false, fmt.Errorf("identity: computed signature encoding failed: %w", err)
	}

	return hmac.Equal(storedSig, computedSig), nil
}

// SignRegistry computes the HMAC-SHA256 signature for registry content.
// Returns the hex-encoded signature string.
// The ceoKey is the raw CEO vault_key (32 bytes from PBKDF2).
func SignRegistry(repoRoot string, ceoKey []byte) (string, error) {
	path := filepath.Join(repoRoot, registryRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("identity: cannot read registry: %w", err)
	}

	content := string(data)
	sigMarker := "\nsignature:"
	idx := strings.Index(content, sigMarker)
	if idx < 0 {
		return "", fmt.Errorf("identity: signature block not found")
	}

	signedContent := content[:idx]

	mac := hmac.New(sha256.New, ceoKey)
	mac.Write([]byte(signedContent))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ── Display helpers ──────────────────────────────────────────────────────────

// RoleLabel returns a human-readable role label with level.
func RoleLabel(id *Identity) string {
	roleUpper := strings.ToUpper(id.Role)
	return fmt.Sprintf("%s · Level %d", roleUpper, id.Level)
}

// WelcomeMessage formats the login welcome line for an identity.
func WelcomeMessage(id *Identity) string {
	return fmt.Sprintf("Welcome, %s [%s]", id.Name, RoleLabel(id))
}
