// sheets_crypto.go — Vault-aware credential storage for ovav-sheets.
//
// Stores client_id + client_secret + refresh_token in AES-256-GCM,
// reusing go-runtime/internal/vault. NEVER writes plaintext secrets.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ovav/ovav/internal/vault"
)

const (
	credDirName      = "credentials"
	credFileName     = "google_oauth.enc"
	defaultKeyPath   = ".ovav/vault/vault.key"
	defaultCredsPath = ".ovav/vault/credentials/google_oauth.enc"
)

// Creds is the on-disk credential bundle. All fields except ProjectID are
// secrets; the vault key encrypts the whole struct before write.
type Creds struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	ProjectID    string   `json:"project_id"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	AccessToken  string   `json:"access_token,omitempty"`
	TokenExpiry  int64    `json:"token_expiry_unix,omitempty"`
	Scope        string   `json:"scope"`
	Scopes       []string `json:"scopes,omitempty"`
	RedirectURI  string   `json:"redirect_uri"`
}

type credStore struct {
	keyPath   string
	credsPath string
}

// NewCredStore returns a store rooted at repoRoot, defaulting to
// repoRoot/.ovav/vault/{vault.key, credentials/google_oauth.enc}.
func NewCredStore(repoRoot string) *credStore {
	return &credStore{
		keyPath:   filepath.Join(repoRoot, defaultKeyPath),
		credsPath: filepath.Join(repoRoot, defaultCredsPath),
	}
}

// Load reads and decrypts credentials. Returns ErrNoCreds when the file
// does not yet exist.
func (s *credStore) Load() (*Creds, error) {
	ct, err := os.ReadFile(s.credsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoCreds
		}
		return nil, fmt.Errorf("sheets: read creds: %w", err)
	}
	key, err := s.loadKey()
	if err != nil {
		return nil, err
	}
	pt, err := vault.Decrypt(ct, key)
	if err != nil {
		return nil, fmt.Errorf("sheets: decrypt creds: %w", err)
	}
	var c Creds
	if err := json.Unmarshal(pt, &c); err != nil {
		return nil, fmt.Errorf("sheets: parse creds: %w", err)
	}
	return &c, nil
}

// Save encrypts and writes credentials atomically (tmp + rename).
func (s *credStore) Save(c *Creds) error {
	if c.ClientID == "" || c.ClientSecret == "" {
		return errors.New("sheets: client_id and client_secret are required")
	}
	key, err := s.loadKey()
	if err != nil {
		return err
	}
	pt, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("sheets: marshal creds: %w", err)
	}
	ct, err := vault.Encrypt(pt, key)
	if err != nil {
		return fmt.Errorf("sheets: encrypt creds: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.credsPath), 0o700); err != nil {
		return fmt.Errorf("sheets: mkdir creds: %w", err)
	}
	tmp := s.credsPath + ".tmp"
	if err := os.WriteFile(tmp, ct, 0o600); err != nil {
		return fmt.Errorf("sheets: write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.credsPath); err != nil {
		return fmt.Errorf("sheets: rename: %w", err)
	}
	return nil
}

// loadKey reads the 32-byte vault key. Generates a new random one on first
// run, persisting it with 0600. The key file lives in .ovav/vault/ which
// must remain out of version control.
func (s *credStore) loadKey() ([]byte, error) {
	if k, err := os.ReadFile(s.keyPath); err == nil {
		if len(k) != vault.KeySize {
			return nil, fmt.Errorf("sheets: vault.key size %d, want %d", len(k), vault.KeySize)
		}
		return k, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("sheets: read key: %w", err)
	}
	k, err := vault.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("sheets: generate key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.keyPath), 0o700); err != nil {
		return nil, fmt.Errorf("sheets: mkdir vault: %w", err)
	}
	if err := os.WriteFile(s.keyPath, k, 0o600); err != nil {
		return nil, fmt.Errorf("sheets: write key: %w", err)
	}
	return k, nil
}

// ErrNoCreds signals that no encrypted credential bundle exists yet.
var ErrNoCreds = errors.New("sheets: no credentials stored (run `ovav sheets auth --client-id ... --client-secret ...`)")
