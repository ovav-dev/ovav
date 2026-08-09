// Package secrets implements the OVAV Secrets Subsystem.
//
// C9.3 extended: Secrets management for OVAV SYSTEMS infrastructure.
// Competes with Bitwarden/1Password in security, operates exclusively
// within OVAV ecosystem. No external dependencies. AES-256-GCM.
//
// Architecture:
//   - SecretStore: encrypted JSON blob at ~/.local/share/ovav/secrets.vault
//   - Same AES-256-GCM key as existing vault (vault_key_export)
//   - Secret types: api_token, oauth_creds, db_credential, cloud_key,
//     encryption_key, user_secret, tunnel_token
//
// Decoupling: if this package is removed, ovav central continues
// functioning with its profiles/agents/skills assets intact.
package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ovav/ovav/internal/vault"
	"github.com/ovav/ovav/internal/vault/secrets/types"
)

// ── Secret Types (re-exported from types package) ───────────────────────────

// SecretType classifies a secret by its security and operational characteristics.
type SecretType = types.SecretType

const (
	TypeAPIToken      = types.TypeAPIToken
	TypeOAuthCreds    = types.TypeOAuthCreds
	TypeDBCredential  = types.TypeDBCredential
	TypeCloudKey      = types.TypeCloudKey
	TypeEncryptionKey = types.TypeEncryptionKey
	TypeUserSecret    = types.TypeUserSecret
	TypeTunnelToken   = types.TypeTunnelToken
)

// AllTypes returns all known secret types.
func AllTypes() []SecretType { return types.AllTypes() }

// InferType attempts to infer the SecretType from a secret name.
func InferType(name string) SecretType { return types.InferType(name) }

// ComputeHash computes a SHA256 hash of the secret value.
var ComputeHash = types.ComputeHash

// Secret represents a single credential stored in the vault.
type Secret = types.Secret

// Metadata holds arbitrary provider-specific key-value pairs.
type Metadata = types.Metadata

// NewSecret creates a new Secret with a generated UUID and current timestamp.
func NewSecret(name string, secretType SecretType, provider, source string, value []byte) *Secret {
	return &Secret{
		ID:        uuid.New().String(),
		Name:      name,
		Type:      secretType,
		Provider:  provider,
		Source:    source,
		Hash:      ComputeHash(value),
		CreatedAt: time.Now().UTC(),
		Rotatable: false,
		Tags:      []string{},
		Metadata:  Metadata{},
	}
}

// ── SecretStore ─────────────────────────────────────────────────────────────

// SecretStore manages the collection of secrets with mutex-based concurrency safety.
type SecretStore struct {
	mu      sync.RWMutex
	secrets map[string]*Secret // id -> Secret
}

// NewSecretStore creates an empty SecretStore.
func NewSecretStore() *SecretStore {
	return &SecretStore{
		secrets: make(map[string]*Secret),
	}
}

// Add inserts a new secret into the store.
func (s *SecretStore) Add(secret *Secret) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.secrets[secret.ID]; exists {
		return errors.New("secret with this ID already exists")
	}
	s.secrets[secret.ID] = secret
	return nil
}

// Get returns a secret by ID, or nil if not found.
func (s *SecretStore) Get(id string) *Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.secrets[id]
}

// GetByName returns a secret by exact name match, or nil if not found.
func (s *SecretStore) GetByName(name string) *Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sec := range s.secrets {
		if sec.Name == name {
			return sec
		}
	}
	return nil
}

// List returns all secrets, optionally filtered by type.
func (s *SecretStore) List(filterType SecretType) []*Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Secret
	for _, sec := range s.secrets {
		if filterType == "" || sec.Type == filterType {
			result = append(result, sec)
		}
	}
	return result
}

// Remove deletes a secret by ID.
func (s *SecretStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.secrets[id]; !exists {
		return errors.New("secret not found")
	}
	delete(s.secrets, id)
	return nil
}

// Count returns the total number of secrets.
func (s *SecretStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.secrets)
}

// UpdateUsage updates the LastUsed timestamp for a secret.
func (s *SecretStore) UpdateUsage(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sec, exists := s.secrets[id]; exists {
		now := time.Now().UTC()
		sec.LastUsed = &now
	}
}

// ── Persistence ─────────────────────────────────────────────────────────────

// StoreFormat is the JSON format for the encrypted secrets vault.
type StoreFormat struct {
	Version  int       `json:"version"`   // Schema version — currently 1
	StoredAt time.Time `json:"stored_at"` // When the vault was written
	Secrets  []*Secret `json:"secrets"`   // All stored secrets
}

// ToJSON serializes the store to JSON (caller must encrypt the result).
func (s *SecretStore) ToJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	format := StoreFormat{
		Version:  1,
		StoredAt: time.Now().UTC(),
		Secrets:  make([]*Secret, 0, len(s.secrets)),
	}
	for _, sec := range s.secrets {
		format.Secrets = append(format.Secrets, sec)
	}

	return json.MarshalIndent(format, "", "  ")
}

// FromJSON deserializes a store from JSON (caller must decrypt first).
func FromJSON(data []byte) (*SecretStore, error) {
	var format StoreFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return nil, err
	}

	store := NewSecretStore()
	for _, sec := range format.Secrets {
		store.secrets[sec.ID] = sec
	}
	return store, nil
}

// ── File I/O with encryption ────────────────────────────────────────────────

// VaultDir is the directory where the secrets vault is stored.
var VaultDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "ovav")

// VaultFileName is the encrypted vault file name.
const VaultFileName = "secrets.vault"

// FullPath returns the full path to the secrets vault.
func FullPath() string {
	return filepath.Join(VaultDir, VaultFileName)
}

// Save encrypts and writes the store to the default vault path.
// It uses the provided AES-256 key (32 bytes).
func (s *SecretStore) Save(key []byte) error {
	return s.SaveToPath(FullPath(), key)
}

// SaveToPath encrypts and writes the store to a specific path.
func (s *SecretStore) SaveToPath(path string, key []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("secrets: mkdir %s: %w", dir, err)
	}

	// Serialize to JSON
	jsonData, err := s.ToJSON()
	if err != nil {
		return fmt.Errorf("secrets: serialize: %w", err)
	}

	// Encrypt with AES-256-GCM (same algorithm as vault package)
	ciphertext, err := vault.Encrypt(jsonData, key)
	if err != nil {
		return fmt.Errorf("secrets: encrypt: %w", err)
	}

	// Write encrypted blob
	if err := os.WriteFile(path, ciphertext, 0600); err != nil {
		return fmt.Errorf("secrets: write %s: %w", path, err)
	}

	return nil
}

// ErrNotFound is returned when the vault file does not exist.
var ErrNotFound = errors.New("secrets: vault file not found")

// Load reads and decrypts the store from the default vault path.
// Returns ErrNotFound if the vault file does not exist.
func Load(key []byte) (*SecretStore, error) {
	return LoadFromPath(FullPath(), key)
}

// LoadFromPath reads and decrypts the store from a specific path.
func LoadFromPath(path string, key []byte) (*SecretStore, error) {
	ciphertext, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("secrets: read %s: %w", path, err)
	}

	jsonData, err := vault.Decrypt(ciphertext, key)
	if err != nil {
		return nil, fmt.Errorf("secrets: decrypt: %w", err)
	}

	store, err := FromJSON(jsonData)
	if err != nil {
		return nil, fmt.Errorf("secrets: parse: %w", err)
	}

	return store, nil
}
