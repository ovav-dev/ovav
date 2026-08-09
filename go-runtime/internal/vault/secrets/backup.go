// backup.go — Encrypted vault backup and restore.
//
// Phase 5 of OVAV-VAULT-2026 plan.
//
// Provides backup and restore functionality for the secrets vault.
// Backups are encrypted JSON files using the same AES-256-GCM algorithm
// as the vault itself.
//
// Backup format:
//
//	{
//	  "version": 1,
//	  "app": "ovav-vault-secrets",
//	  "created_at": "2026-08-03T...",
//	  "machine_id": "...",
//	  "secret_count": N,
//	  "data": "<vault.Encrypt(JSON)>"
//	}
//
// The outer JSON is plaintext metadata (for indexing/display).
// Only the "data" field is encrypted.
//
// Usage:
//
//	# Create backup
//	ovav-vault-secrets backup --path ~/.local/share/ovav/secrets-backup-$(date +%Y%m%d).enc
//
//	# Restore from backup
//	ovav-vault-secrets restore --path ~/.local/share/ovav/secrets-backup-20260803.enc
package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	vaultpkg "github.com/ovav/ovav/internal/vault"
)

// BackupFormat is the plaintext header for a backup file.
type BackupFormat struct {
	Version     int       `json:"version"`
	App         string    `json:"app"`
	CreatedAt   time.Time `json:"created_at"`
	MachineID   string    `json:"machine_id"`
	SecretCount int       `json:"secret_count"`
	Data        []byte    `json:"data"` // encrypted JSON blob
}

// Backup creates an encrypted backup of the vault to the given path.
// The backup includes all secrets and is encrypted with the vault key.
func Backup(store *SecretStore, key []byte, path string) error {
	// Serialize the store
	jsonData, err := store.ToJSON()
	if err != nil {
		return fmt.Errorf("backup serialize: %w", err)
	}

	// Encrypt the vault data
	ciphertext, err := vaultpkg.Encrypt(jsonData, key)
	if err != nil {
		return fmt.Errorf("backup encrypt: %w", err)
	}

	// Build backup header
	hostname, _ := os.Hostname()
	backup := BackupFormat{
		Version:     1,
		App:         "ovav-vault-secrets",
		CreatedAt:   time.Now().UTC(),
		MachineID:   hostname,
		SecretCount: store.Count(),
		Data:        ciphertext,
	}

	// Serialize header (plaintext)
	header, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("backup header marshal: %w", err)
	}

	// Write backup file
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("backup mkdir: %w", err)
	}
	if err := os.WriteFile(path, header, 0600); err != nil {
		return fmt.Errorf("backup write: %w", err)
	}

	return nil
}

// Restore decrypts and loads a backup from the given path.
// It replaces the current vault (use with caution — backs up current first).
func Restore(path string, key []byte) (*SecretStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("backup read: %w", err)
	}

	var backup BackupFormat
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("backup parse: %w", err)
	}

	if backup.Version != 1 {
		return nil, fmt.Errorf("unsupported backup version: %d", backup.Version)
	}

	// Decrypt the data
	plaintext, err := vaultpkg.Decrypt(backup.Data, key)
	if err != nil {
		return nil, fmt.Errorf("backup decrypt: %w (wrong key?)", err)
	}

	// Load the store
	store, err := FromJSON(plaintext)
	if err != nil {
		return nil, fmt.Errorf("backup parse vault: %w", err)
	}

	return store, nil
}

// BackupInfo reads and returns the plaintext header of a backup file
// without decrypting the vault data.
func BackupInfo(path string) (*BackupFormat, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("backup read: %w", err)
	}

	var backup BackupFormat
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("backup parse: %w", err)
	}

	return &backup, nil
}
