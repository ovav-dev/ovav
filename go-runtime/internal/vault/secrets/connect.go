// connect.go — OVAV CONNECT: AI Provider API Key Management
//
// Phase 6.3 of OVAV-VAULT-2026 plan.
//
// OVAV CONNECT tracks AI provider API keys (OpenAI, Anthropic, OpenRouter, Azure)
// inside the vault. It replaces scattered env vars with unified, encrypted storage
// and provides spend tracking, usage logs, and rotation reminders.
//
// OVAV CONNECT keys are stored INSIDE the main secrets.vault alongside all other
// secrets. They are NOT in a separate file. This ensures:
//   - Same encryption (vault key = PBKDF2(seed, machineID))
//   - Same audit log (every access logged)
//   - Same backup/restore (handled by the vault subsystem)
//   - Same sync (handled by the sync subsystem)
package secrets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	vaultpkg "github.com/ovav/ovav/internal/vault"
)

// ConnectKey represents an AI provider API key tracked by OVAV CONNECT.
// It is a view type derived from the underlying Secret record.
type ConnectKey struct {
	ID           string            `json:"id"`
	Provider     string            `json:"provider"`
	Name         string            `json:"name"`
	EncryptedB64 string            `json:"encrypted_b64"` // base64(AES-256-GCM(raw_key)) — persisted
	EnvVar       string            `json:"env_var"`
	Model        string            `json:"model,omitempty"`
	OrgID        string            `json:"org_id,omitempty"`
	AddedAt      time.Time         `json:"added_at"`
	LastUsed     *time.Time        `json:"last_used,omitempty"`
	LastChecked  *time.Time        `json:"last_checked,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	MonthlyLimit int               `json:"monthly_limit_cents"` // $500.00 = 50000
	CurrentSpend int               `json:"current_spend_cents"`
	Status       string            `json:"status"` // "active" | "expired" | "quota_exceeded" | "unknown"
	Tags         []string          `json:"tags,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ProviderConfig holds per-provider OVAV CONNECT configuration.
type ProviderConfig struct {
	Provider      string
	EnvVar        string
	DisplayName   string
	Color         string
	SpendEndpoint string
	AuthHeader    string
}

// KnownProviders maps provider names to their configuration.
var KnownProviders = map[string]ProviderConfig{
	"openai": {
		Provider:      "openai",
		EnvVar:        "OPENAI_API_KEY",
		DisplayName:   "OpenAI",
		Color:         "#10A37F",
		SpendEndpoint: "https://api.openai.com/v1/dashboard/billing/summary",
		AuthHeader:    "Bearer",
	},
	"anthropic": {
		Provider:      "anthropic",
		EnvVar:        "ANTHROPIC_API_KEY",
		DisplayName:   "Anthropic",
		Color:         "#CC785C",
		SpendEndpoint: "https://console.anthropic.com/settings/usage",
		AuthHeader:    "x-api-key",
	},
	"openrouter": {
		Provider:      "openrouter",
		EnvVar:        "OPENROUTER_API_KEY",
		DisplayName:   "OpenRouter",
		Color:         "#9M8DDB",
		SpendEndpoint: "https://openrouter.ai/api/v1/credits",
		AuthHeader:    "Bearer",
	},
	"azure": {
		Provider:      "azure",
		EnvVar:        "AZURE_OPENAI_KEY",
		DisplayName:   "Azure OpenAI",
		Color:         "#0078D4",
		SpendEndpoint: "",
		AuthHeader:    "api-key",
	},
}

// ConnectStore manages OVAV CONNECT keys.
type ConnectStore struct {
	store    *SecretStore
	vaultKey []byte
}

// NewConnectStore creates a connect store backed by the main vault.
func NewConnectStore(store *SecretStore, vaultKey []byte) *ConnectStore {
	return &ConnectStore{store: store, vaultKey: vaultKey}
}

// AddConnectKey adds a new connect key to the vault.
func (cs *ConnectStore) AddConnectKey(provider, name, rawKey, envVar string) error {
	encrypted, err := vaultpkg.Encrypt([]byte(rawKey), cs.vaultKey)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	secretName := fmt.Sprintf("CONNECT/%s", strings.ToUpper(provider))
	sec := NewSecret(secretName, TypeAPIToken, "connect", "connect", []byte(rawKey))
	sec.Tags = []string{"connect", provider}
	sec.Metadata["provider"] = provider
	sec.Metadata["env_var"] = envVar
	sec.Metadata["display_name"] = name
	sec.Metadata["encrypted_b64"] = base64.StdEncoding.EncodeToString(encrypted)
	if envVar != "" {
		sec.Metadata["env_var"] = envVar
	}

	if err := cs.store.Add(sec); err != nil {
		return fmt.Errorf("add secret: %w", err)
	}

	return nil
}

// ListConnectKeys returns all connect keys from the vault.
func (cs *ConnectStore) ListConnectKeys() ([]*ConnectKey, error) {
	var keys []*ConnectKey
	for _, sec := range cs.store.List("") {
		if sec.Provider != "connect" {
			continue
		}
		ck := secretToConnectKey(sec)
		keys = append(keys, ck)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].AddedAt.After(keys[j].AddedAt)
	})
	return keys, nil
}

// GetConnectKey returns a single connect key by provider.
func (cs *ConnectStore) GetConnectKey(provider string) (*ConnectKey, error) {
	sec := cs.store.GetByName(fmt.Sprintf("CONNECT/%s", strings.ToUpper(provider)))
	if sec == nil {
		return nil, fmt.Errorf("connect: key for provider %q not found", provider)
	}
	return secretToConnectKey(sec), nil
}

// DecryptKey decrypts and returns the raw API key for a provider.
// Returns error if key is not found or cannot be decrypted.
func (cs *ConnectStore) DecryptKey(provider string) (string, error) {
	sec := cs.store.GetByName(fmt.Sprintf("CONNECT/%s", strings.ToUpper(provider)))
	if sec == nil {
		return "", fmt.Errorf("connect: key for provider %q not found", provider)
	}

	b64, ok := sec.Metadata["encrypted_b64"]
	if !ok || b64 == "" {
		return "", fmt.Errorf("connect: encrypted key not found in vault")
	}

	encrypted, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("connect: decode encrypted key: %w", err)
	}

	raw, err := vaultpkg.Decrypt(encrypted, cs.vaultKey)
	if err != nil {
		return "", fmt.Errorf("connect: decrypt: %w", err)
	}

	// Update last used
	now := time.Now()
	sec.LastUsed = &now

	return string(raw), nil
}

// TrackUsage records that a connect key was used now.
func (cs *ConnectStore) TrackUsage(provider string) error {
	sec := cs.store.GetByName(fmt.Sprintf("CONNECT/%s", strings.ToUpper(provider)))
	if sec == nil {
		return fmt.Errorf("connect: key for provider %q not found", provider)
	}
	now := time.Now()
	sec.LastUsed = &now
	return nil
}

// SetMonthlyLimit sets the monthly spend limit for a provider.
func (cs *ConnectStore) SetMonthlyLimit(provider string, cents int) error {
	sec := cs.store.GetByName(fmt.Sprintf("CONNECT/%s", strings.ToUpper(provider)))
	if sec == nil {
		return fmt.Errorf("connect: key for provider %q not found", provider)
	}
	if sec.Metadata == nil {
		sec.Metadata = make(map[string]string)
	}
	sec.Metadata["monthly_limit_cents"] = fmt.Sprintf("%d", cents)
	return nil
}

// ConnectSpendEntry is one row in the spend report.
type ConnectSpendEntry struct {
	Provider        string     `json:"provider"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	MonthlyLimit    int        `json:"monthly_limit_cents"`
	CurrentSpend    int        `json:"current_spend_cents"`
	LastChecked     *time.Time `json:"last_checked,omitempty"`
	MonthlyLimitFmt string     `json:"monthly_limit_fmt"`
	CurrentSpendFmt string     `json:"current_spend_fmt"`
	UsagePct        float64    `json:"usage_pct"`
	Warning         bool       `json:"warning"`
}

// SpendReport returns a spend summary for all connect keys.
func (cs *ConnectStore) SpendReport() ([]*ConnectSpendEntry, error) {
	keys, err := cs.ListConnectKeys()
	if err != nil {
		return nil, err
	}

	var report []*ConnectSpendEntry
	for _, key := range keys {
		entry := &ConnectSpendEntry{
			Provider:        key.Provider,
			Name:            key.Name,
			Status:          key.Status,
			MonthlyLimit:    key.MonthlyLimit,
			CurrentSpend:    key.CurrentSpend,
			LastChecked:     key.LastChecked,
			MonthlyLimitFmt: formatCents(key.MonthlyLimit),
			CurrentSpendFmt: formatCents(key.CurrentSpend),
		}

		if key.MonthlyLimit > 0 {
			entry.UsagePct = float64(key.CurrentSpend) / float64(key.MonthlyLimit) * 100
		}

		if key.Status == "quota_exceeded" || (key.MonthlyLimit > 0 && entry.UsagePct >= 90) {
			entry.Warning = true
		}

		report = append(report, entry)
	}
	return report, nil
}

// SyncSpend queries the provider API and updates stored spend data.
func (cs *ConnectStore) SyncSpend(provider string) error {
	cfg, ok := KnownProviders[provider]
	if !ok {
		return fmt.Errorf("connect: unknown provider %q", provider)
	}

	rawKey, err := cs.DecryptKey(provider)
	if err != nil {
		return fmt.Errorf("connect: get key: %w", err)
	}

	switch provider {
	case "openrouter":
		return cs.syncOpenRouterSpend(cfg, rawKey)
	default:
		return fmt.Errorf("connect: spend sync for %q not yet implemented", provider)
	}
}

func (cs *ConnectStore) syncOpenRouterSpend(cfg ProviderConfig, rawKey string) error {
	req, err := http.NewRequest("GET", cfg.SpendEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+rawKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("openrouter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("openrouter: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Type     string  `json:"type"`
			Credits  float64 `json:"credits"`
			Currency string  `json:"currency"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("openrouter: parse: %w", err)
	}

	var totalCents int
	for _, c := range result.Data {
		if c.Type == "paid" && c.Currency == "USD" {
			totalCents = int(c.Credits * 100)
		}
	}

	sec := cs.store.GetByName(fmt.Sprintf("CONNECT/%s", strings.ToUpper(cfg.Provider)))
	if sec != nil {
		if sec.Metadata == nil {
			sec.Metadata = make(map[string]string)
		}
		sec.Metadata["current_spend_cents"] = fmt.Sprintf("%d", totalCents)
		now := time.Now()
		sec.LastUsed = &now
	}

	return nil
}

// DetectAndVaultEnvKeys scans env for known API keys and vaults them.
func (cs *ConnectStore) DetectAndVaultEnvKeys() ([]*ConnectKey, error) {
	var added []*ConnectKey

	envProviders := []struct {
		envVar   string
		provider string
		name     string
	}{
		{"OPENAI_API_KEY", "openai", "OpenAI API Key"},
		{"ANTHROPIC_API_KEY", "anthropic", "Anthropic API Key"},
		{"OPENROUTER_API_KEY", "openrouter", "OpenRouter API Key"},
		{"AZURE_OPENAI_KEY", "azure", "Azure OpenAI Key"},
	}

	for _, p := range envProviders {
		rawKey := os.Getenv(p.envVar)
		if rawKey == "" {
			continue
		}

		// Skip if already vaulted
		sec := cs.store.GetByName(fmt.Sprintf("CONNECT/%s", strings.ToUpper(p.provider)))
		if sec != nil {
			continue
		}

		if err := cs.AddConnectKey(p.provider, p.name, rawKey, p.envVar); err != nil {
			continue
		}

		ck, _ := cs.GetConnectKey(p.provider)
		if ck != nil {
			added = append(added, ck)
		}
	}

	return added, nil
}

// secretToConnectKey converts a vault Secret (provider=connect) to a ConnectKey.
func secretToConnectKey(sec *Secret) *ConnectKey {
	ck := &ConnectKey{
		ID:       sec.ID,
		Name:     sec.Metadata["display_name"],
		AddedAt:  sec.CreatedAt,
		LastUsed: sec.LastUsed,
		Tags:     sec.Tags,
		Metadata: sec.Metadata,
	}

	if p := sec.Metadata["provider"]; p != "" {
		ck.Provider = p
	} else {
		ck.Provider = providerFromName(sec.Name)
	}

	if v := sec.Metadata["env_var"]; v != "" {
		ck.EnvVar = v
	}
	if v := sec.Metadata["model"]; v != "" {
		ck.Model = v
	}
	if v := sec.Metadata["org_id"]; v != "" {
		ck.OrgID = v
	}
	if v := sec.Metadata["status"]; v != "" {
		ck.Status = v
	} else {
		ck.Status = "active"
	}
	if v := sec.Metadata["encrypted_b64"]; v != "" {
		ck.EncryptedB64 = v
	}

	var monthlyLimit, currentSpend int
	if v := sec.Metadata["monthly_limit_cents"]; v != "" {
		fmt.Sscanf(v, "%d", &monthlyLimit)
	}
	if v := sec.Metadata["current_spend_cents"]; v != "" {
		fmt.Sscanf(v, "%d", &currentSpend)
	}
	ck.MonthlyLimit = monthlyLimit
	ck.CurrentSpend = currentSpend

	if ck.Name == "" {
		ck.Name = sec.Name
	}

	return ck
}

// providerFromName extracts "openai" from "CONNECT/OPENAI".
func providerFromName(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		return strings.ToLower(parts[1])
	}
	return ""
}

func formatCents(cents int) string {
	return fmt.Sprintf("$%.2f", float64(cents)/100)
}
