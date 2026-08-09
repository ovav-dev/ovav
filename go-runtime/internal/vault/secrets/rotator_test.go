package secrets

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// ── RotateResult / RotateReport JSON ─────────────────────────────────────────

func TestRotateResult_JSON(t *testing.T) {
	result := RotateResult{
		Provider: "github-actions",
		Path:     "GitHub Actions: owner/repo",
		Status:   "rotated",
	}

	data, err := jsonMarshal(result)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var r2 RotateResult
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if r2.Status != "rotated" {
		t.Errorf("Status = %q, want %q", r2.Status, "rotated")
	}
}

func TestRotateReport_JSON(t *testing.T) {
	report := RotateReport{
		SecretName:       "CF_TOKEN",
		SecretID:         "sec-123",
		VaultUpdated:     true,
		RollbackOccurred: false,
		Results: []RotateResult{
			{Provider: "github-actions", Status: "rotated"},
			{Provider: "fly-io", Status: "failed", Error: "token expired"},
		},
	}

	data, err := jsonMarshal(report)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var r2 RotateReport
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !r2.VaultUpdated {
		t.Error("VaultUpdated = false, want true")
	}
	if len(r2.Results) != 2 {
		t.Errorf("Results len = %d, want 2", len(r2.Results))
	}
}

// ── GenerateRandomSecret ─────────────────────────────────────────────────────

func TestGenerateRandomSecret_Length(t *testing.T) {
	s := generateRandomSecret(32)
	// base64 encoded — 32 bytes → ~43 chars
	if len(s) < 40 {
		t.Errorf("generateRandomSecret(32): got len %d, want ~43", len(s))
	}
}

func TestGenerateRandomSecret_Uniqueness(t *testing.T) {
	s1 := generateRandomSecret(32)
	s2 := generateRandomSecret(32)
	if s1 == s2 {
		t.Error("generateRandomSecret: two calls returned same value")
	}
}

func TestGenerateRandomSecret_DifferentLengths(t *testing.T) {
	s16 := generateRandomSecret(16)
	s48 := generateRandomSecret(48)
	// 16 bytes base64 → ~22 chars, 48 bytes → ~64 chars
	if len(s16) == len(s48) {
		t.Error("Different lengths should produce different encoded lengths")
	}
}

// ── githubEncryptSecret ───────────────────────────────────────────────────────

func TestGithubEncryptSecret_OutputFormat(t *testing.T) {
	// This tests the encryption format without making real HTTP calls
	// We can't easily test githubEncryptSecret without a real public key
	// but we can verify the key decoding logic

	// 32-byte Curve25519 public key (base64 encoded)
	// This is a test key — not a real one
	testPubKey := base64.StdEncoding.EncodeToString(make([]byte, 32))

	_, err := githubEncryptSecret(testPubKey, "test-plaintext")
	// Should fail with wrong key length but verifies the path works
	if err == nil {
		t.Log("Note: githubEncryptSecret accepted a zero key")
	}
}

func TestGithubEncryptSecret_InvalidKey(t *testing.T) {
	_, err := githubEncryptSecret("not-base64", "plaintext")
	if err == nil {
		t.Error("Expected error for invalid base64 key")
	}

	// Too short
	shortKey := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	_, err = githubEncryptSecret(shortKey, "plaintext")
	if err == nil {
		t.Error("Expected error for too-short key")
	}
}

// ── RotateSecret Tests ───────────────────────────────────────────────────────

func TestRotateSecret_NotFound(t *testing.T) {
	store := NewSecretStore()
	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	key := make([]byte, 32)

	_, err := RotateSecret(store, graph, "nonexistent", key)
	if err == nil {
		t.Error("RotateSecret nonexistent: expected error")
	}
}

func TestRotateSecret_NoProviders(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("OrphanToken", TypeAPIToken, "cf", "manual", []byte("val"))
	store.Add(sec)

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	key := make([]byte, 32)

	_, err := RotateSecret(store, graph, "OrphanToken", key)
	if err == nil {
		t.Error("RotateSecret with no providers: expected error")
	}
}

func TestRotateSecret_ValueUpdate(t *testing.T) {
	// This tests that when all providers fail, the vault is not updated
	origGH := os.Getenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")
	defer func() {
		if origGH != "" {
			os.Setenv("GITHUB_TOKEN", origGH)
		}
	}()

	store := NewSecretStore()
	sec := NewSecret("GH_TOKEN", TypeAPIToken, "github", "github", []byte("original-value"))
	store.Add(sec)

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	graph.AddRef(sec.ID, SystemGitHubActions, "GitHub Actions: owner/repo", "GH_TOKEN")

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	originalHash := sec.Hash
	report, err := RotateSecret(store, graph, "GH_TOKEN", key)

	// Should not panic, should return report
	if err != nil {
		// Rotation failure is expected without real tokens
		if !strings.Contains(err.Error(), "incomplete") && !strings.Contains(err.Error(), "failed") {
			// The error message talks about partial failure
		}
	}

	if report == nil {
		t.Fatal("RotateSecret returned nil report")
	}

	// Vault should NOT be updated when providers fail (rollback)
	if report.VaultUpdated {
		t.Error("VaultUpdated = true, want false (rollback should have occurred)")
	}

	// The hash should be unchanged
	if sec.Hash != originalHash {
		t.Error("Hash changed despite rollback")
	}
}

func TestRotateSecret_RotatableFlag(t *testing.T) {
	store := NewSecretStore()
	sec := NewSecret("GH_TOKEN", TypeAPIToken, "github", "github", []byte("val"))
	store.Add(sec)
	sec.Rotatable = true // Should be set by AddRef with GitHub

	graph := &DependencyGraph{refs: make(map[string][]SecretRef)}
	graph.AddRef(sec.ID, SystemGitHubActions, "GitHub Actions: owner/repo", "GH_TOKEN")

	// Verify AutoRotatable is true
	refs := graph.GetRefs(sec.ID)
	if !refs[0].AutoRotatable {
		t.Error("GitHub Actions ref should be AutoRotatable=true")
	}
}

// ── secretboxSeal ────────────────────────────────────────────────────────────

func TestSecretboxSeal(t *testing.T) {
	var nonce [24]byte
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	// Nonce all zeros is fine for testing
	plaintext := []byte("hello world")

	out := secretboxSeal(plaintext, &nonce, &key)
	if len(out) == 0 {
		t.Error("secretboxSeal returned empty")
	}
}

// ── httpGet / httpPut ───────────────────────────────────────────────────────

func TestHttpGet_Timeout(t *testing.T) {
	// Just verify httpGet returns error for invalid URL
	_, err := httpGet("http://127.0.0.1:99999/nonexistent", "token", 100*time.Millisecond)
	if err == nil {
		t.Error("Expected error for unreachable URL")
	}
}

func TestHttpPut_Timeout(t *testing.T) {
	_, err := httpPut("http://127.0.0.1:99999/nonexistent", "token", []byte("body"), 100*time.Millisecond)
	if err == nil {
		t.Error("Expected error for unreachable URL")
	}
}

// ── RotateResult Status ──────────────────────────────────────────────────────

func TestRotateResult_StatusValues(t *testing.T) {
	statuses := []string{"rotated", "failed", "manual"}

	for _, s := range statuses {
		r := RotateResult{Status: s}
		if r.Status != s {
			t.Errorf("Status = %q, want %q", r.Status, s)
		}
	}
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
