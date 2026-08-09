// token_vault.go — encrypt/decrypt infra tokens with OVAV vault key.
// Bridges ovav login (PBKDF2 key) and ovav infra (token directory).
package infra

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const tokenEncFile = "tokens.enc"

// DecryptTokensFromVault decrypts tokens.enc → tokens/ using the login vault key.
// Called by ovav login after vault key is derived.
func DecryptTokensFromVault(keyHex string) error {
	if keyHex == "" {
		return nil
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil // silently skip
	}

	root, err := ResolveRepoRoot()
	if err != nil {
		return nil
	}

	encPath := filepath.Join(root, TokenDir, tokenEncFile)
	data, err := os.ReadFile(encPath)
	if err != nil {
		return nil // no tokens.enc yet — first time setup
	}

	plaintext, err := decryptAESGCM(data, key)
	if err != nil {
		return fmt.Errorf("decrypt tokens.enc: %w", err)
	}

	var tokens map[string]string
	if err := json.Unmarshal(plaintext, &tokens); err != nil {
		return fmt.Errorf("parse tokens.enc: %w", err)
	}

	tokensDir := filepath.Join(root, TokenDir)
	os.MkdirAll(tokensDir, 0700)
	for name, value := range tokens {
		os.WriteFile(filepath.Join(tokensDir, name), []byte(strings.TrimSpace(value)), 0600)
	}
	return nil
}

// EncryptTokensToVault reads tokens/ → encrypts → tokens.enc using OVAV_VAULT_KEY.
// Called by ovav infra bootstrap after tokens are written.
func EncryptTokensToVault(root string) error {
	keyHex := os.Getenv("OVAV_VAULT_KEY")
	if keyHex == "" {
		return nil
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil
	}

	tokensDir := VaultPath(root)
	entries, err := os.ReadDir(tokensDir)
	if err != nil {
		return nil
	}

	tokens := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tokensDir, e.Name()))
		if err != nil {
			continue
		}
		tokens[e.Name()] = strings.TrimSpace(string(data))
	}
	if len(tokens) == 0 {
		return nil
	}

	plaintext, _ := json.Marshal(tokens)
	ciphertext, err := encryptAESGCM(plaintext, key)
	if err != nil {
		return fmt.Errorf("encrypt tokens: %w", err)
	}

	encPath := filepath.Join(root, TokenDir, tokenEncFile)
	return os.WriteFile(encPath, ciphertext, 0600)
}

func encryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	return gcm.Open(nil, ciphertext[:ns], ciphertext[ns:], nil)
}
