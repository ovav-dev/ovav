package tracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestTrackedProviderMarshalNeverPersistsAPIKey(t *testing.T) {
	const secret = "test-secret-must-not-be-written"

	data, err := yaml.Marshal([]*TrackedProvider{{
		ID:        "minimax-test",
		Type:      "minimax",
		APIKey:    secret,
		APIKeyEnv: "MINIMAX_API_KEY",
		Enabled:   true,
	}})
	if err != nil {
		t.Fatalf("marshal provider: %v", err)
	}

	serialized := string(data)
	if strings.Contains(serialized, secret) {
		t.Fatal("serialized provider contains API key")
	}
	if !strings.Contains(serialized, "api_key_env: MINIMAX_API_KEY") {
		t.Fatal("serialized provider lost API key environment reference")
	}
}

func TestTrackedProviderResolveAPIKeyFromEnvironment(t *testing.T) {
	t.Setenv("MINIMAX_API_KEY", "environment-only-key")
	p := &TrackedProvider{APIKey: "legacy-key", APIKeyEnv: "MINIMAX_API_KEY"}

	if got := p.ResolveAPIKey(); got != "environment-only-key" {
		t.Fatalf("ResolveAPIKey() = %q, want environment value", got)
	}
}

func TestSaveProvidersDoesNotPersistResolvedAPIKey(t *testing.T) {
	const secret = "test-secret-must-not-be-written"
	dir := t.TempDir()
	tk := New(dir)
	tk.providers["minimax-test"] = &TrackedProvider{
		ID:        "minimax-test",
		Type:      "minimax",
		APIKey:    secret,
		APIKeyEnv: "MINIMAX_API_KEY",
		Enabled:   true,
		AddedAt:   time.Now(),
	}

	if err := tk.SaveProviders(); err != nil {
		t.Fatalf("SaveProviders() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "providers.yaml"))
	if err != nil {
		t.Fatalf("read saved providers: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatal("saved providers contain API key")
	}
}
