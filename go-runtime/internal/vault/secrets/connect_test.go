package secrets

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// ── ConnectKey Tests ──────────────────────────────────────────────────────────

func TestConnectKey_JSON(t *testing.T) {
	ck := ConnectKey{
		ID:           "ck-123",
		Provider:     "openai",
		Name:         "OpenAI Production",
		EncryptedB64: "ZW5jcnlwdGVk",
		EnvVar:       "OPENAI_API_KEY",
		Model:        "gpt-4o",
		OrgID:        "org-abc",
		AddedAt:      time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		Status:       "active",
		MonthlyLimit: 50000,
		CurrentSpend: 12500,
	}

	data, err := json.Marshal(ck)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var ck2 ConnectKey
	if err := json.Unmarshal(data, &ck2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if ck2.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", ck2.Provider, "openai")
	}
	if ck2.Status != "active" {
		t.Errorf("Status = %q, want %q", ck2.Status, "active")
	}
}

func TestConnectKey_SpendFormatting(t *testing.T) {
	entry := ConnectSpendEntry{
		Provider:        "openai",
		Name:            "OpenAI Key",
		Status:          "active",
		MonthlyLimit:    50000,
		CurrentSpend:    45000,
		MonthlyLimitFmt: formatCents(50000),
		CurrentSpendFmt: formatCents(45000),
		UsagePct:        90.0,
		Warning:         true,
	}

	if entry.MonthlyLimitFmt != "$500.00" {
		t.Errorf("MonthlyLimitFmt = %q, want %q", entry.MonthlyLimitFmt, "$500.00")
	}
	if entry.CurrentSpendFmt != "$450.00" {
		t.Errorf("CurrentSpendFmt = %q, want %q", entry.CurrentSpendFmt, "$450.00")
	}
	if entry.UsagePct != 90.0 {
		t.Errorf("UsagePct = %f, want 90.0", entry.UsagePct)
	}
	if !entry.Warning {
		t.Error("Warning = false, want true (90% usage)")
	}
}

func TestFormatCents(t *testing.T) {
	tests := []struct {
		cents int
		want  string
	}{
		{0, "$0.00"},
		{1, "$0.01"},
		{99, "$0.99"},
		{100, "$1.00"},
		{55099, "$550.99"},
		{50000, "$500.00"},
		{-100, "$-1.00"}, // fmt.Sprintf("$%.2f", -1.0) = "$-1.00"
	}

	for _, tc := range tests {
		got := formatCents(tc.cents)
		if got != tc.want {
			t.Errorf("formatCents(%d) = %q, want %q", tc.cents, got, tc.want)
		}
	}
}

// ── KnownProviders ─────────────────────────────────────────────────────────────

func TestKnownProviders(t *testing.T) {
	expected := []string{"openai", "anthropic", "openrouter", "azure"}
	for _, name := range expected {
		cfg, ok := KnownProviders[name]
		if !ok {
			t.Errorf("KnownProviders missing %q", name)
			continue
		}
		if cfg.Provider != name {
			t.Errorf("Provider = %q, want %q", cfg.Provider, name)
		}
		if cfg.EnvVar == "" {
			t.Errorf("EnvVar for %q is empty", name)
		}
	}
}

func TestKnownProviders_OpenAI(t *testing.T) {
	cfg := KnownProviders["openai"]
	if cfg.Color != "#10A37F" {
		t.Errorf("OpenAI color = %q, want #10A37F", cfg.Color)
	}
	if cfg.SpendEndpoint == "" {
		t.Error("OpenAI SpendEndpoint is empty")
	}
}

func TestKnownProviders_OpenRouter(t *testing.T) {
	cfg := KnownProviders["openrouter"]
	if cfg.AuthHeader != "Bearer" {
		t.Errorf("OpenRouter auth = %q, want %q", cfg.AuthHeader, "Bearer")
	}
}

// ── ConnectStore ─────────────────────────────────────────────────────────────

func TestConnectStore_AddConnectKey(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	err := cs.AddConnectKey("openai", "Test Key", "sk-test-key-value", "OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("AddConnectKey: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("store.Count = %d, want 1", store.Count())
	}
}

func TestConnectStore_ListConnectKeys(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	cs.AddConnectKey("openai", "OpenAI Key", "sk-openai", "OPENAI_API_KEY")
	cs.AddConnectKey("anthropic", "Anthropic Key", "sk-ant", "ANTHROPIC_API_KEY")

	keys, err := cs.ListConnectKeys()
	if err != nil {
		t.Fatalf("ListConnectKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("ListConnectKeys: got %d, want 2", len(keys))
	}
}

func TestConnectStore_GetConnectKey(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	cs.AddConnectKey("openai", "OpenAI Key", "sk-openai", "OPENAI_API_KEY")

	ck, err := cs.GetConnectKey("openai")
	if err != nil {
		t.Fatalf("GetConnectKey: %v", err)
	}
	if ck.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", ck.Provider, "openai")
	}
}

func TestConnectStore_GetConnectKey_NotFound(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	cs := NewConnectStore(store, key)

	_, err := cs.GetConnectKey("nonexistent")
	if err == nil {
		t.Error("GetConnectKey for nonexistent: expected error")
	}
}

func TestConnectStore_SetMonthlyLimit(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	cs.AddConnectKey("openai", "OpenAI Key", "sk-openai", "OPENAI_API_KEY")

	err := cs.SetMonthlyLimit("openai", 50000)
	if err != nil {
		t.Fatalf("SetMonthlyLimit: %v", err)
	}

	ck, _ := cs.GetConnectKey("openai")
	if ck.MonthlyLimit != 50000 {
		t.Errorf("MonthlyLimit = %d, want 50000", ck.MonthlyLimit)
	}
}

func TestConnectStore_TrackUsage(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	cs.AddConnectKey("openai", "OpenAI Key", "sk-openai", "OPENAI_API_KEY")

	before := time.Now()
	err := cs.TrackUsage("openai")
	if err != nil {
		t.Fatalf("TrackUsage: %v", err)
	}

	ck, _ := cs.GetConnectKey("openai")
	if ck.LastUsed == nil {
		t.Fatal("LastUsed should be set after TrackUsage")
	}
	if ck.LastUsed.Before(before) {
		t.Error("LastUsed should be >= time before call")
	}
}

func TestConnectStore_DecryptKey(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	cs.AddConnectKey("openai", "OpenAI Key", "sk-secret-key", "OPENAI_API_KEY")

	decrypted, err := cs.DecryptKey("openai")
	if err != nil {
		t.Fatalf("DecryptKey: %v", err)
	}
	if decrypted != "sk-secret-key" {
		t.Errorf("Decrypted = %q, want %q", decrypted, "sk-secret-key")
	}
}

func TestConnectStore_SpendReport(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	cs.AddConnectKey("openai", "OpenAI Key", "sk-openai", "OPENAI_API_KEY")
	cs.SetMonthlyLimit("openai", 50000)

	// Set some spend
	sec := store.GetByName("CONNECT/OPENAI")
	sec.Metadata["current_spend_cents"] = "12500"

	report, err := cs.SpendReport()
	if err != nil {
		t.Fatalf("SpendReport: %v", err)
	}
	if len(report) != 1 {
		t.Errorf("SpendReport entries = %d, want 1", len(report))
	}
	if report[0].UsagePct != 25.0 {
		t.Errorf("UsagePct = %f, want 25.0", report[0].UsagePct)
	}
}

// ── providerFromName ─────────────────────────────────────────────────────────

func TestProviderFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"CONNECT/OPENAI", "openai"},
		{"CONNECT/ANTHROPIC", "anthropic"},
		{"CONNECT/OPENROUTER", "openrouter"},
		{"CONNECT/AZURE", "azure"},
		{"CONNECT/GITHUB", "github"},
		{"INVALID", ""},
		{"OPENAI", ""},
	}

	for _, tc := range tests {
		got := providerFromName(tc.name)
		if got != tc.want {
			t.Errorf("providerFromName(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// ── DetectAndVaultEnvKeys ────────────────────────────────────────────────────

func TestConnectStore_DetectAndVaultEnvKeys(t *testing.T) {
	orig := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "sk-test-openai-key")
	defer func() {
		if orig != "" {
			os.Setenv("OPENAI_API_KEY", orig)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	added, err := cs.DetectAndVaultEnvKeys()
	if err != nil {
		t.Fatalf("DetectAndVaultEnvKeys: %v", err)
	}

	if len(added) == 0 {
		t.Error("DetectAndVaultEnvKeys: expected to detect at least 1 key")
	}

	if store.Count() < 1 {
		t.Errorf("store.Count = %d, want >= 1", store.Count())
	}
}

func TestConnectStore_DetectAndVaultEnvKeys_AlreadyExists(t *testing.T) {
	orig := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "sk-test-openai-key")
	defer func() {
		if orig != "" {
			os.Setenv("OPENAI_API_KEY", orig)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	store := NewSecretStore()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	cs := NewConnectStore(store, key)
	// Add it first — DetectAndVaultEnvKeys should skip it
	cs.AddConnectKey("openai", "OpenAI Key", "sk-openai", "OPENAI_API_KEY")

	added, err := cs.DetectAndVaultEnvKeys()
	if err != nil {
		t.Fatalf("DetectAndVaultEnvKeys: %v", err)
	}

	if len(added) != 0 {
		t.Errorf("DetectAndVaultEnvKeys when key exists: got %d, want 0", len(added))
	}
}

// ── secretToConnectKey ───────────────────────────────────────────────────────

func TestSecretToConnectKey(t *testing.T) {
	sec := NewSecret("CONNECT/OPENAI", TypeAPIToken, "connect", "connect", []byte("sk-secret"))
	sec.Metadata["provider"] = "openai"
	sec.Metadata["display_name"] = "Production Key"
	sec.Metadata["env_var"] = "OPENAI_API_KEY"
	sec.Metadata["monthly_limit_cents"] = "50000"
	sec.Metadata["current_spend_cents"] = "12500"

	ck := secretToConnectKey(sec)

	if ck.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", ck.Provider, "openai")
	}
	if ck.Name != "Production Key" {
		t.Errorf("Name = %q, want %q", ck.Name, "Production Key")
	}
	if ck.EnvVar != "OPENAI_API_KEY" {
		t.Errorf("EnvVar = %q, want %q", ck.EnvVar, "OPENAI_API_KEY")
	}
	if ck.MonthlyLimit != 50000 {
		t.Errorf("MonthlyLimit = %d, want 50000", ck.MonthlyLimit)
	}
	if ck.CurrentSpend != 12500 {
		t.Errorf("CurrentSpend = %d, want 12500", ck.CurrentSpend)
	}
	if ck.Status != "active" {
		// Default when not set
		t.Errorf("Status = %q, want %q", ck.Status, "active")
	}
}

func TestSecretToConnectKey_NoDisplayName(t *testing.T) {
	sec := NewSecret("CONNECT/OPENAI", TypeAPIToken, "connect", "connect", []byte("sk-secret"))
	sec.Metadata["provider"] = "openai"
	// No display_name set

	ck := secretToConnectKey(sec)
	// Should fall back to secret name
	if ck.Name != "CONNECT/OPENAI" {
		t.Errorf("Name = %q, want %q (fallback to secret name)", ck.Name, "CONNECT/OPENAI")
	}
}

// ── ProviderConfig ────────────────────────────────────────────────────────────

func TestProviderConfig_Fields(t *testing.T) {
	cfg := KnownProviders["openai"]
	if cfg.DisplayName == "" {
		t.Error("DisplayName should not be empty")
	}
	if cfg.AuthHeader == "" {
		t.Error("AuthHeader should not be empty")
	}
}

// ── ConnectSpendEntry ─────────────────────────────────────────────────────────

func TestConnectSpendEntry_Warning90(t *testing.T) {
	entry := ConnectSpendEntry{
		MonthlyLimit: 10000,
		CurrentSpend: 9500,
		UsagePct:     95.0,
		Warning:      true,
	}

	if entry.UsagePct != 95.0 {
		t.Errorf("UsagePct = %f, want 95.0", entry.UsagePct)
	}
	if !entry.Warning {
		t.Error("Warning should be true when >= 90%")
	}
}

func TestConnectSpendEntry_QuotaExceeded(t *testing.T) {
	entry := ConnectSpendEntry{
		Status:  "quota_exceeded",
		Warning: true,
	}

	if entry.Status != "quota_exceeded" {
		t.Errorf("Status = %q, want %q", entry.Status, "quota_exceeded")
	}
}

// ── ConnectStore DecryptKey NotFound ─────────────────────────────────────────

func TestConnectStore_DecryptKey_NotFound(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	cs := NewConnectStore(store, key)

	_, err := cs.DecryptKey("nonexistent")
	if err == nil {
		t.Error("DecryptKey for nonexistent: expected error")
	}
}

// ── ConnectStore TrackUsage NotFound ─────────────────────────────────────────

func TestConnectStore_TrackUsage_NotFound(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	cs := NewConnectStore(store, key)

	err := cs.TrackUsage("nonexistent")
	if err == nil {
		t.Error("TrackUsage for nonexistent: expected error")
	}
}

// ── NewConnectStore ───────────────────────────────────────────────────────────

func TestNewConnectStore(t *testing.T) {
	store := NewSecretStore()
	key := make([]byte, 32)
	cs := NewConnectStore(store, key)
	if cs == nil {
		t.Fatal("NewConnectStore returned nil")
	}
	if cs.store != store {
		t.Error("store not set correctly")
	}
	if cs.vaultKey == nil {
		t.Error("vaultKey should not be nil")
	}
}
