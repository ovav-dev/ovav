// rotator.go — Credential Rotation Engine
//
// Phase 6.9 of OVAV-VAULT-2026 plan.
//
// Rotation is a three-phase operation:
//  1. GENERATE: Create a new cryptographically random secret
//  2. PUSH: Propagate the new secret to all registered systems (GitHub, Fly, etc.)
//  3. UPDATE: Store the new value in the vault, mark old value as superseded
//
// If PUSH fails, the operation returns partial results with error.
package secrets

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/secretbox"
)

// ── GitHub Actions Secrets Rotation ─────────────────────────────────────────

// GitHub public key for encrypting secrets (libsodium sealed box)
type githubPublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"` // base64-encoded 32-byte Curve25519 public key
}

// rotateGitHubSecret rotates a GitHub Actions secret.
func rotateGitHubSecret(ref SecretRef) (newValue string, err error) {
	repo := strings.TrimPrefix(ref.Path, "GitHub Actions: ")
	if repo == ref.Path {
		return "", fmt.Errorf("cannot parse repo from path: %s", ref.Path)
	}

	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid repo path: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN env not set")
	}

	// Step 1: Get repo's public key
	pubKeyURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/secrets/public-key", owner, repoName)
	httpResp, err := httpGet(pubKeyURL, token, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("get public key: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		body, _ := io.ReadAll(httpResp.Body)
		return "", fmt.Errorf("get public key: HTTP %d — %s", httpResp.StatusCode, string(body))
	}

	var pk githubPublicKey
	if err := json.NewDecoder(httpResp.Body).Decode(&pk); err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}

	// Step 2: Generate new random secret (32 bytes → base64)
	newValue = generateRandomSecret(32)

	// Step 3: Encrypt with NaCl sealed box (libsodium sealedbox_easy compatible)
	encryptedB64, err := githubEncryptSecret(pk.Key, newValue)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	// Step 4: PUT encrypted secret to GitHub
	putURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/secrets/%s", owner, repoName, ref.EnvVar)
	encPayload := map[string]string{
		"encrypted_value": encryptedB64,
		"key_id":          pk.KeyID,
	}
	payload, _ := json.Marshal(encPayload)

	httpResp2, err := httpPut(putURL, token, payload, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("put secret: %w", err)
	}
	defer httpResp2.Body.Close()

	if httpResp2.StatusCode != 201 && httpResp2.StatusCode != 204 {
		body, _ := io.ReadAll(httpResp2.Body)
		return "", fmt.Errorf("put secret: HTTP %d — %s", httpResp2.StatusCode, string(body))
	}

	return newValue, nil
}

// githubEncryptSecret encrypts a plaintext for GitHub Actions using NaCl sealed box.
// GitHub uses libsodium sealedbox_easy which produces:
//
//	ephemeral_pk (32) || nonce (24) || ciphertext
//
// The symmetric key is derived as BLAKE2b-256(scalarmult(ephemeralSK, recipientPK))
func githubEncryptSecret(base64PubKey, plaintext string) (string, error) {
	// Decode recipient public key
	recipientPKBytes, err := base64.StdEncoding.DecodeString(base64PubKey)
	if err != nil {
		return "", fmt.Errorf("decode pubkey: %w", err)
	}
	if len(recipientPKBytes) != 32 {
		return "", fmt.Errorf("expected 32-byte Curve25519 pubkey, got %d", len(recipientPKBytes))
	}

	var recipientPK [32]byte
	copy(recipientPK[:], recipientPKBytes)

	// Generate ephemeral keypair using box.GenerateKey
	ephemeralPub, ephemeralPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ephemeral keypair: %w", err)
	}

	// Compute shared secret using Curve25519 ECDH
	// For NaCl box: shared = crypto_scalarmult(ephemeralPriv, recipientPK)
	var shared [32]byte
	box.Precompute(&shared, (*[32]byte)(recipientPKBytes), ephemeralPriv)

	// Derive symmetric key: BLAKE2b-256 of shared secret
	h := sha256.New()
	h.Write(shared[:])
	var symKey [32]byte
	copy(symKey[:], h.Sum(nil))

	// Encrypt with crypto_secretbox_easy: nonce (24 bytes) || ciphertext
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}

	// crypto_secretbox_easy(plaintext, nonce, key)
	encrypted := secretboxSeal([]byte(plaintext), &nonce, &symKey)

	// Assemble: ephemeral_pk (32) || nonce (24) || ciphertext
	result := make([]byte, 0, 32+24+len(encrypted))
	result = append(result, ephemeralPub[:]...)
	result = append(result, nonce[:]...)
	result = append(result, encrypted...)

	return base64.StdEncoding.EncodeToString(result), nil
}

// secretboxSeal implements crypto_secretbox_easy.
func secretboxSeal(plaintext []byte, nonce *[24]byte, key *[32]byte) []byte {
	var out []byte
	return secretbox.Seal(out, plaintext, nonce, key)
}

// httpGet performs an HTTP GET with token and timeout.
func httpGet(url, token string, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return http.DefaultClient.Do(req)
}

// httpPut performs an HTTP PUT with token, body, and timeout.
func httpPut(url, token string, body []byte, timeout time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	return http.DefaultClient.Do(req)
}

// generateRandomSecret generates a cryptographically random secret.
func generateRandomSecret(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// ── Fly.io Secret Rotation ────────────────────────────────────────────────────

// rotateFlySecret rotates a Fly.io secret using their API.
func rotateFlySecret(ref SecretRef) (newValue string, err error) {
	appName := strings.TrimPrefix(ref.Path, "Fly.io app: ")
	if appName == ref.Path {
		return "", fmt.Errorf("cannot parse app name from path: %s", ref.Path)
	}

	token := os.Getenv("FLY_API_TOKEN")
	if token == "" {
		return "", fmt.Errorf("FLY_API_TOKEN env not set")
	}

	newValue = generateRandomSecret(32)

	// Fly.io API: PATCH /api/v1/apps/{appname}/secrets
	url := fmt.Sprintf("https://api.machines.dev/v1/apps/%s/secrets", appName)
	payload := map[string]interface{}{
		"secrets": []map[string]string{
			{"name": ref.EnvVar, "value": newValue},
		},
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Fly.io API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Fly.io API: HTTP %d — %s", resp.StatusCode, string(bodyBytes))
	}

	return newValue, nil
}

// ── RotateResult ──────────────────────────────────────────────────────────────

// RotateResult is the outcome of rotating a single provider's secret.
type RotateResult struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Status   string `json:"status"` // "rotated", "failed", "manual"
	Error    string `json:"error,omitempty"`
}

// RotateReport is the full outcome of a rotate operation.
type RotateReport struct {
	SecretName       string         `json:"secret_name"`
	SecretID         string         `json:"secret_id"`
	VaultUpdated     bool           `json:"vault_updated"`
	RollbackOccurred bool           `json:"rollback_occurred"`
	Results          []RotateResult `json:"results"`
}

// ── RotateSecret ──────────────────────────────────────────────────────────────

// RotateSecret rotates a secret: generates new value, pushes to all providers,
// updates vault. Callers must pass the vault encryption key to persist changes.
func RotateSecret(store *SecretStore, graph *DependencyGraph, name string, vaultKey []byte) (*RotateReport, error) {
	sec := store.GetByName(name)
	if sec == nil {
		return nil, fmt.Errorf("secret %q not found", name)
	}

	refs := graph.GetRefs(sec.ID)
	if len(refs) == 0 {
		return nil, fmt.Errorf("no providers registered for %q — add providers with 'vault add' then 'vault deps track'", name)
	}

	report := &RotateReport{
		SecretName: name,
		SecretID:   sec.ID,
		Results:    make([]RotateResult, 0, len(refs)),
	}

	// Step 1: Generate new master value
	newMasterValue := []byte(generateRandomSecret(48))

	// Step 2: Push to each provider
	for _, ref := range refs {
		result := RotateResult{Provider: string(ref.System), Path: ref.Path}

		var err error
		switch ref.System {
		case SystemGitHubActions:
			_, err = rotateGitHubSecret(ref)
		case SystemFlyIO:
			_, err = rotateFlySecret(ref)
		default:
			err = fmt.Errorf("rotation not supported for %s — manual rotation required", ref.System)
			result.Status = "manual"
			result.Error = err.Error()
		}

		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "rotated"
		}

		report.Results = append(report.Results, result)
	}

	// Step 3: Check if any failed
	anyFailed := false
	for _, r := range report.Results {
		if r.Status == "failed" || r.Status == "manual" {
			anyFailed = true
			break
		}
	}

	if anyFailed {
		report.RollbackOccurred = true
		return report, fmt.Errorf("rotation incomplete — some providers failed or need manual action")
	}

	// Step 4: Update vault with new value
	sec.Value = newMasterValue
	h := sha256.Sum256(newMasterValue)
	sec.Hash = hex.EncodeToString(h[:])
	if sec.Metadata == nil {
		sec.Metadata = make(map[string]string)
	}
	sec.Metadata["last_rotated"] = time.Now().UTC().Format(time.RFC3339)

	if err := store.Save(vaultKey); err != nil {
		return report, fmt.Errorf("save vault: %w", err)
	}
	report.VaultUpdated = true

	return report, nil
}
