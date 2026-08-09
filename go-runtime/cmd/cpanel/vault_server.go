// OVAV cPanel — Vault Server (per-user secret storage).
//
// Stores secrets per authenticated user. Values are encrypted with the user's
// vault key (derived from password + email via PBKDF2).
//
// Tier-based slot limits:
//   free: 5 secrets
//   premium: 50 secrets
//   enterprise: unlimited
//
// All endpoints require Bearer JWT with user_id claim.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── Secret model ──────────────────────────────────────────────────────────────

type VaultSecret struct {
	ID         string            `json:"id"`          // SHA256(name+type+source)[:16]
	Name       string            `json:"name"`        // e.g. "GITHUB_TOKEN"
	Type       string            `json:"type"`        // api_token, oauth_creds, db_credential, cloud_key, encryption_key, user_secret, tunnel_token
	Value      string            `json:"value"`       // plaintext (encrypted at rest)
	Metadata   map[string]string `json:"metadata"`    // encrypted_b64, provider, last_rotated, expires_at
	Source     string            `json:"source"`      // "manual", "github", "fly", "openrouter"
	SourcePath string            `json:"source_path"` // "GitHub Actions: owner/repo"
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// VaultStore is the per-user vault storage.
type VaultStore struct {
	UserID  string         `json:"user_id"`
	Secrets []*VaultSecret `json:"secrets"` // indexed by Name
	Updated time.Time      `json:"updated"`
}

// ── Vault key derivation ──────────────────────────────────────────────────────

// encryptVaultValue encrypts plaintext with AES-256-GCM using the user's vault key.
func encryptVaultValue(plaintext []byte, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decryptVaultValue decrypts a base64-encoded ciphertext using AES-256-GCM.
func decryptVaultValue(encoded string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// ── Vault store ───────────────────────────────────────────────────────────────

// userVaultStore maps userID → vault data (in-memory, persisted to disk).
var (
	vaultStores = make(map[string]*vaultData)
	vaultMu     = sync.RWMutex{}
)

type vaultData struct {
	mu    sync.RWMutex
	store *VaultStore
	key   []byte // vault encryption key (derived from password)
	dirty bool
	path  string
}

func getVaultData(userID string) (*vaultData, error) {
	vaultMu.RLock()
	vd, exists := vaultStores[userID]
	vaultMu.RUnlock()
	if exists {
		return vd, nil
	}

	vaultMu.Lock()
	defer vaultMu.Unlock()
	// Double-check after acquiring write lock
	if vd, exists = vaultStores[userID]; exists {
		return vd, nil
	}

	vd = &vaultData{
		store: &VaultStore{UserID: userID, Secrets: []*VaultSecret{}},
		path:  vaultStorePath(userID),
	}
	vd.load()
	vaultStores[userID] = vd
	return vd, nil
}

func vaultStorePath(userID string) string {
	return filepath.Join(RepoRoot, ".ovav", "vault", "stores", userID+".vault")
}

func (vd *vaultData) load() {
	vd.mu.Lock()
	defer vd.mu.Unlock()

	data, err := os.ReadFile(vd.path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		return
	}
	var store VaultStore
	if err := json.Unmarshal(data, &store); err != nil {
		return
	}
	vd.store = &store
}

func (vd *vaultData) save() error {
	vd.mu.RLock()
	store := vd.store
	vd.mu.RUnlock()

	os.MkdirAll(filepath.Dir(vd.path), 0700)
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vd.path, data, 0600)
}

// ── Vault operations ─────────────────────────────────────────────────────────

// ListSecrets returns metadata for all secrets (no values).
func ListSecrets(userID string) ([]*VaultSecret, error) {
	vd, err := getVaultData(userID)
	if err != nil {
		return nil, err
	}
	vd.mu.RLock()
	defer vd.mu.RUnlock()

	secrets := make([]*VaultSecret, len(vd.store.Secrets))
	for i, s := range vd.store.Secrets {
		// Return a copy without the plaintext value
		sec := *s
		sec.Value = "" // never expose value in list
		if sec.Metadata == nil {
			sec.Metadata = make(map[string]string)
		}
		if encrypted := sec.Metadata["encrypted_b64"]; encrypted != "" {
			sec.Metadata["encrypted_b64"] = "***" // mask ciphertext in list
		}
		secrets[i] = &sec
	}
	return secrets, nil
}

// GetSecret returns a secret by name (including value, decrypted).
func GetSecret(userID, name string) (*VaultSecret, error) {
	vd, err := getVaultData(userID)
	if err != nil {
		return nil, err
	}
	vd.mu.RLock()
	defer vd.mu.RUnlock()

	for _, s := range vd.store.Secrets {
		if s.Name == name {
			// Decrypt the value if encrypted
			sec := *s
			if encrypted := sec.Metadata["encrypted_b64"]; encrypted != "" && vd.key != nil {
				if decrypted, err := decryptVaultValue(encrypted, vd.key); err == nil {
					sec.Value = string(decrypted)
				}
			}
			return &sec, nil
		}
	}
	return nil, fmt.Errorf("secret not found")
}

// AddSecret adds a new secret. Returns error if slot limit reached.
func AddSecret(userID, name, value, secretType, source, sourcePath string, metadata map[string]string) (*VaultSecret, error) {
	store := getUserStore()
	limit := store.SlotLimit(userID)

	vd, err := getVaultData(userID)
	if err != nil {
		return nil, err
	}
	vd.mu.Lock()
	defer vd.mu.Unlock()

	// Check slot limit
	userTier := "free"
	if user := store.GetByID(userID); user != nil {
		userTier = user.Tier
	}
	if len(vd.store.Secrets) >= limit {
		return nil, fmt.Errorf("slot limit reached (%d secrets, tier: %s)", limit, userTier)
	}

	// Check duplicate name
	for _, s := range vd.store.Secrets {
		if s.Name == name {
			return nil, fmt.Errorf("secret '%s' already exists", name)
		}
	}

	// Encrypt value if key available
	encryptedValue := ""
	if vd.key != nil {
		enc, err := encryptVaultValue([]byte(value), vd.key)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %w", err)
		}
		encryptedValue = enc
	}

	now := time.Now().UTC()
	secretIDHash := sha256Hash([]byte(name + secretType + source))
	secret := &VaultSecret{
		ID:         encodeBase64URL(secretIDHash[:])[:16],
		Name:       name,
		Type:       secretType,
		Value:      value, // plaintext in memory
		Metadata:   metadata,
		Source:     source,
		SourcePath: sourcePath,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if encryptedValue != "" {
		if secret.Metadata == nil {
			secret.Metadata = make(map[string]string)
		}
		secret.Metadata["encrypted_b64"] = encryptedValue
		secret.Value = "" // clear plaintext after encrypting
	}

	vd.store.Secrets = append(vd.store.Secrets, secret)
	vd.store.Updated = now
	vd.dirty = true

	if err := vd.save(); err != nil {
		// Rollback
		vd.store.Secrets = vd.store.Secrets[:len(vd.store.Secrets)-1]
		vd.dirty = true
		return nil, fmt.Errorf("save failed: %w", err)
	}

	// Return without plaintext value
	sec := *secret
	sec.Value = ""
	return &sec, nil
}

// RemoveSecret deletes a secret by name.
func RemoveSecret(userID, name string) error {
	vd, err := getVaultData(userID)
	if err != nil {
		return err
	}
	vd.mu.Lock()
	defer vd.mu.Unlock()

	for i, s := range vd.store.Secrets {
		if s.Name == name {
			vd.store.Secrets = append(vd.store.Secrets[:i], vd.store.Secrets[i+1:]...)
			vd.store.Updated = time.Now().UTC()
			vd.dirty = true
			return vd.save()
		}
	}
	return fmt.Errorf("secret not found")
}

// SetVaultKey sets the vault encryption key for a user session.
// Called after user login to enable decrypt.
func SetVaultKey(userID string, key []byte) {
	vd, err := getVaultData(userID)
	if err != nil {
		return
	}
	vd.mu.Lock()
	defer vd.mu.Unlock()
	vd.key = key
}

// CountSecrets returns the number of secrets a user has.
func CountSecrets(userID string) int {
	vd, err := getVaultData(userID)
	if err != nil {
		return 0
	}
	vd.mu.RLock()
	defer vd.mu.RUnlock()
	return len(vd.store.Secrets)
}

// SlotLimit returns the secret slot limit for a user's tier.
func SlotLimit(userID string) int {
	store := getUserStore()
	return store.SlotLimit(userID)
}

// HasFeature checks if a user tier has a specific feature.
func HasVaultFeature(userID, feature string) bool {
	store := getUserStore()
	return store.HasFeature(userID, feature)
}
