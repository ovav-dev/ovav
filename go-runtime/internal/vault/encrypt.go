// Package vault implements AES-256-GCM encryption/decryption for OVAV assets.
//
// C9.3: Vault runtime. Usa crypto/aes y crypto/cipher de stdlib.
// Sin dependencias externas. Clave derivada de PBKDF2 (internal/license/).
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	KeySize   = 32 // AES-256 key length
	NonceSize = 12 // GCM nonce size
	TagSize   = 16 // GCM auth tag size
)

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns: nonce(12) || tag(16) || ciphertext.
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("vault: key must be %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("vault: generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext produced by Encrypt.
// Expects: nonce(12) || tag(16) || ciphertext.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("vault: key must be %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: cipher.NewGCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("vault: ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("vault: decryption failed (wrong key or corruption): %w", err)
	}

	return plaintext, nil
}

// GenerateKey generates a random 32-byte AES-256 key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("vault: generate key: %w", err)
	}
	return key, nil
}

// EncryptFile encrypts a file to an output path.
func EncryptFile(inputPath, outputPath string, key []byte) error {
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("vault: read input: %w", err)
	}
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, ciphertext, 0644)
}

// DecryptFile decrypts a file to an output path.
func DecryptFile(inputPath, outputPath string, key []byte) error {
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("vault: read input: %w", err)
	}
	plaintext, err := Decrypt(ciphertext, key)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, plaintext, 0644)
}
