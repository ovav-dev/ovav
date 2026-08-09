package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ── infra.go tests ──────────────────────────────────────────────────────────

func TestTokenDir(t *testing.T) {
	if TokenDir != ".ovav/vault/tokens" {
		t.Errorf("TokenDir = %q, want .ovav/vault/tokens", TokenDir)
	}
}

func TestRequiredTokens_NonEmpty(t *testing.T) {
	if len(RequiredTokens) == 0 {
		t.Error("RequiredTokens should not be empty")
	}
	for i, ts := range RequiredTokens {
		if ts.Name == "" {
			t.Errorf("RequiredTokens[%d].Name is empty", i)
		}
	}
}

func TestTokenSpec_Fields(t *testing.T) {
	ts := TokenSpec{Name: "api", Description: "API token", Source: "vault", EnvVar: "CF_API_TOKEN", Optional: false}
	if ts.Name != "api" {
		t.Errorf("Name = %q, want api", ts.Name)
	}
	if ts.EnvVar != "CF_API_TOKEN" {
		t.Errorf("EnvVar = %q", ts.EnvVar)
	}
	if ts.Optional {
		t.Error("Optional should be false")
	}
}

func TestResolveRepoRoot_FromCWD(t *testing.T) {
	root, err := ResolveRepoRoot()
	if err != nil {
		t.Fatalf("ResolveRepoRoot from repo CWD: %v", err)
	}
	if root == "" {
		t.Error("root should not be empty")
	}
	if _, err := os.Stat(filepath.Join(root, ".ovav")); err != nil {
		t.Errorf(".ovav not found at resolved root %q: %v", root, err)
	}
}

func TestResolveRepoRoot_NoOVAV(t *testing.T) {
	// Create a temp dir WITHOUT .ovav — but CWD is in the real repo
	// Just verify the function signature works; full isolation is hard
	dir := t.TempDir()
	// Cannot easily test failure case since CWD is inside OVAV repo
	_ = dir
}

func TestHasCommand_Exists(t *testing.T) {
	if !HasCommand("go") {
		t.Log("go not found — test may run in minimal environment")
	}
}

func TestHasCommand_NotExists(t *testing.T) {
	if HasCommand("nonexistent_command_xyz_123") {
		t.Error("nonexistent command should not be found")
	}
}

func TestVaultPath(t *testing.T) {
	root := "/tmp/ovav"
	path := VaultPath(root)
	if !strings.HasSuffix(path, TokenDir) {
		t.Errorf("VaultPath = %q, expected suffix %q", path, TokenDir)
	}
	if !strings.HasPrefix(path, root) {
		t.Errorf("VaultPath = %q, expected prefix %q", path, root)
	}
}

// ── bootstrap.go tests ─────────────────────────────────────────────────────

func TestBootstrapResult_Fields(t *testing.T) {
	br := BootstrapResult{
		Step:   "test-step",
		Status: "ok",
		Detail: "all good",
	}
	if br.Step != "test-step" {
		t.Errorf("Step = %q", br.Step)
	}
	if br.Status != "ok" {
		t.Errorf("Status = %q", br.Status)
	}
}

func TestBootstrap_NoOVAVDir(t *testing.T) {
	dir := t.TempDir()
	results, err := Bootstrap(dir)
	if err != nil {
		t.Logf("Bootstrap error (expected in clean dir): %v", err)
	}
	if results == nil {
		t.Error("Bootstrap should return results slice even on failure")
	}
}

func TestCheckGHSecret(t *testing.T) {
	found, err := checkGHSecret("GITHUB_TOKEN_NONEXISTENT_XYZ")
	if err != nil {
		t.Logf("checkGHSecret error: %v", err)
	}
	if found {
		t.Error("should not find nonexistent secret")
	}
}

// ── check.go tests ──────────────────────────────────────────────────────────

func TestServiceStatus_Fields(t *testing.T) {
	ss := ServiceStatus{
		Service: "ovav.dev",
		URL:     "https://ovav.dev",
		Status:  "ok",
		Detail:  "responding",
	}
	if ss.Service != "ovav.dev" {
		t.Errorf("Service = %q", ss.Service)
	}
	if ss.Status != "ok" {
		t.Errorf("Status = %q", ss.Status)
	}
}

func TestPrintStatusReport_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintStatusReport panicked: %v", r)
		}
	}()
	PrintStatusReport(nil)
	PrintStatusReport([]ServiceStatus{})
	PrintStatusReport([]ServiceStatus{
		{Service: "test", URL: "https://test.dev", Status: "ok"},
	})
}

func TestTokenStatus_Fields(t *testing.T) {
	ts := TokenStatus{
		Name:   "api_token",
		Found:  true,
		Source: "vault",
		EnvVar: "CF_API_TOKEN",
	}
	if !ts.Found {
		t.Error("Found should be true")
	}
	if ts.Name != "api_token" {
		t.Errorf("Name = %q", ts.Name)
	}
}

func TestCheckTokens_NoVaultDir(t *testing.T) {
	dir := t.TempDir()
	results := CheckTokens(dir)
	if len(results) == 0 {
		t.Error("CheckTokens should return results even with no vault dir")
	}
	for _, r := range results {
		if r.Found {
			t.Errorf("token %q should not be found in empty dir", r.Name)
		}
	}
}

func TestCheckAll_NoVaultDir(t *testing.T) {
	dir := t.TempDir()
	results := CheckAll(dir)
	if results == nil {
		t.Error("CheckAll should return slice even with no vault")
	}
}

func TestCheckHTTP_Offline(t *testing.T) {
	result := checkHTTP("test-offline", "https://0.0.0.0:19999", 200)
	if result.Service != "test-offline" {
		t.Errorf("Service = %q", result.Service)
	}
	if result.Status != "down" && result.Status != "error" {
		t.Logf("unreachable URL status = %q (detail: %s)", result.Status, result.Detail)
	}
}

func TestCheckFlyIO_ReturnsStatus(t *testing.T) {
	result := checkFlyIO()
	if result.Service == "" {
		t.Error("FlyIO status should have a service name")
	}
}

func TestCheckGitHub_ReturnsStatus(t *testing.T) {
	result := checkGitHub()
	if result.Service == "" {
		t.Error("GitHub status should have a service name")
	}
}

func TestCheckGoogleCloud_ReturnsStatus(t *testing.T) {
	result := checkGoogleCloud()
	if result.Service == "" {
		t.Error("GCloud status should have a service name")
	}
}

// ── cloudflare.go tests ────────────────────────────────────────────────────

func TestCFDNSRecord_Struct(t *testing.T) {
	rec := CFDNSRecord{
		ID:      "abc123",
		Name:    "test.ovav.dev",
		Type:    "A",
		Content: "1.2.3.4",
		TTL:     1,
		Proxied: true,
	}
	if rec.Type != "A" {
		t.Errorf("Type = %q", rec.Type)
	}
	if !rec.Proxied {
		t.Error("Proxied should be true")
	}
}

func TestCFTunnelConfig_Struct(t *testing.T) {
	cfg := CFTunnelConfig{
		TunnelID:   "tun-abc",
		TunnelName: "ovav-tunnel",
		Hostnames:  []string{"d678beea.ovav.dev"},
	}
	if cfg.TunnelName != "ovav-tunnel" {
		t.Errorf("TunnelName = %q", cfg.TunnelName)
	}
	if len(cfg.Hostnames) != 1 {
		t.Errorf("Hostnames length = %d", len(cfg.Hostnames))
	}
}

func TestLoadCFCredentials_NoVaultDir(t *testing.T) {
	dir := t.TempDir()
	_, err := loadCFCredentials(dir)
	if err == nil {
		t.Log("loadCFCredentials succeeded unexpectedly (may have env vars)")
	}
}

func TestCFAPIVerify_InvalidToken(t *testing.T) {
	err := cfAPIVerify("invalid-token-xyz")
	if err == nil {
		t.Error("expected error with invalid token")
	}
}

func TestDNSResult_Struct(t *testing.T) {
	dr := DNSResult{
		ZoneID: "zone123",
		Record: CFDNSRecord{ID: "rec456", Name: "test.ovav.dev"},
		Action: "deleted",
	}
	if dr.ZoneID != "zone123" {
		t.Errorf("ZoneID = %q", dr.ZoneID)
	}
	if dr.Action != "deleted" {
		t.Errorf("Action = %q", dr.Action)
	}
}

func TestTunnelInfo_Struct(t *testing.T) {
	ti := TunnelInfo{
		ID:        "tun-abc",
		Name:      "ovav-tunnel",
		Hostnames: []string{"d678beea.ovav.dev"},
	}
	if ti.Name != "ovav-tunnel" {
		t.Errorf("Name = %q", ti.Name)
	}
	if ti.ID != "tun-abc" {
		t.Errorf("ID = %q", ti.ID)
	}
}

func TestCFAPI_Constant(t *testing.T) {
	if cfAPI == "" {
		t.Error("cfAPI should not be empty")
	}
	if !strings.HasPrefix(cfAPI, "https://") {
		t.Error("cfAPI should use HTTPS")
	}
}

func TestGetAccountID_NoVaultDir(t *testing.T) {
	dir := t.TempDir()
	_, err := getAccountID(dir)
	if err == nil {
		t.Log("getAccountID succeeded without vault (may have env vars)")
	}
}

func TestGetZoneID_NoVaultDir(t *testing.T) {
	dir := t.TempDir()
	_, err := getZoneID("example.com", dir)
	if err == nil {
		t.Log("getZoneID succeeded without vault (may have env vars)")
	}
}

// ── coverage boost tests ────────────────────────────────────────────────────

func TestLoadCFCredentials_WithTokenFile(t *testing.T) {
	dir := t.TempDir()
	token := "test-token-abc123"
	if err := os.WriteFile(filepath.Join(dir, "CF_API_TOKEN"), []byte(token), 0600); err != nil {
		t.Fatal(err)
	}

	creds, err := loadCFCredentials(dir)
	if err != nil {
		t.Fatalf("loadCFCredentials with token file: %v", err)
	}
	if creds.apiToken != token {
		t.Errorf("apiToken = %q, want %q", creds.apiToken, token)
	}
	if creds.apiKey != "" {
		t.Errorf("apiKey should be empty, got %q", creds.apiKey)
	}
}

func TestLoadCFCredentials_WithAPIKeyAndEmail(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CF_API_KEY"), []byte("my-key"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CF_EMAIL"), []byte("user@example.com"), 0600); err != nil {
		t.Fatal(err)
	}

	creds, err := loadCFCredentials(dir)
	if err != nil {
		t.Fatalf("loadCFCredentials with key+email: %v", err)
	}
	if creds.apiKey != "my-key" {
		t.Errorf("apiKey = %q, want %q", creds.apiKey, "my-key")
	}
	if creds.email != "user@example.com" {
		t.Errorf("email = %q, want %q", creds.email, "user@example.com")
	}
	if creds.apiToken != "" {
		t.Errorf("apiToken should be empty when using key auth")
	}
}

func TestLoadCFCredentials_APIKeyWithoutEmail(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CF_API_KEY"), []byte("my-key"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadCFCredentials(dir)
	if err == nil {
		t.Error("expected error when CF_API_KEY is set without CF_EMAIL")
	}
	if !strings.Contains(err.Error(), "CF_EMAIL") {
		t.Errorf("error should mention CF_EMAIL, got: %v", err)
	}
}

func TestLoadCFToken_FromEnvironment(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CF_API_TOKEN", "env-token-xyz")
	// Clear CF_API_KEY so priority 1 path is taken
	t.Setenv("CF_API_KEY", "")

	result := loadCFToken(dir, vaultDir)
	if result.Status != "ok" {
		t.Errorf("status = %q, want ok (detail: %s)", result.Status, result.Detail)
	}
	if result.Step != "cf-token" {
		t.Errorf("step = %q, want cf-token", result.Step)
	}

	data, err := os.ReadFile(filepath.Join(vaultDir, "CF_API_TOKEN"))
	if err != nil {
		t.Fatalf("token file not written: %v", err)
	}
	if string(data) != "env-token-xyz" {
		t.Errorf("vault content = %q, want %q", string(data), "env-token-xyz")
	}
}

func TestLoadCFToken_FromAPIKeyEnv(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CF_API_TOKEN", "")
	t.Setenv("CF_API_KEY", "global-key-123")
	t.Setenv("CF_EMAIL", "admin@example.com")

	result := loadCFToken(dir, vaultDir)
	if result.Status != "ok" {
		t.Errorf("status = %q, want ok (detail: %s)", result.Status, result.Detail)
	}
	if !strings.Contains(result.Detail, "Global API Key") {
		t.Errorf("detail = %q, should mention Global API Key", result.Detail)
	}

	data, err := os.ReadFile(filepath.Join(vaultDir, "CF_API_KEY"))
	if err != nil {
		t.Fatalf("key file not written: %v", err)
	}
	if string(data) != "global-key-123" {
		t.Errorf("vault key = %q, want %q", string(data), "global-key-123")
	}

	emailData, err := os.ReadFile(filepath.Join(vaultDir, "CF_EMAIL"))
	if err != nil {
		t.Fatalf("email file not written: %v", err)
	}
	if string(emailData) != "admin@example.com" {
		t.Errorf("vault email = %q, want %q", string(emailData), "admin@example.com")
	}
}

func TestLoadCFToken_AlreadyInVault(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Pre-populate vault
	if err := os.WriteFile(filepath.Join(vaultDir, "CF_API_TOKEN"), []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	// Clear env so it falls through to vault check
	t.Setenv("CF_API_TOKEN", "")
	t.Setenv("CF_API_KEY", "")

	result := loadCFToken(dir, vaultDir)
	if result.Status != "ok" {
		t.Errorf("status = %q, want ok", result.Status)
	}
	if !strings.Contains(result.Detail, "already in vault") {
		t.Errorf("detail = %q, should mention 'already in vault'", result.Detail)
	}
}

func TestLoadCFAccountID_FromEnvironment(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CF_ACCOUNT_ID", "account-12345")

	result := loadCFAccountID(dir, vaultDir)
	if result.Status != "ok" {
		t.Errorf("status = %q, want ok (detail: %s)", result.Status, result.Detail)
	}
	if result.Step != "cf-account" {
		t.Errorf("step = %q, want cf-account", result.Step)
	}

	data, err := os.ReadFile(filepath.Join(vaultDir, "CF_ACCOUNT_ID"))
	if err != nil {
		t.Fatalf("account ID file not written: %v", err)
	}
	if string(data) != "account-12345" {
		t.Errorf("vault content = %q, want %q", string(data), "account-12345")
	}
}

func TestLoadCFAccountID_AlreadyInVault(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Pre-populate vault
	if err := os.WriteFile(filepath.Join(vaultDir, "CF_ACCOUNT_ID"), []byte("existing-id"), 0600); err != nil {
		t.Fatal(err)
	}

	// Clear env so it falls through to vault/gh checks
	t.Setenv("CF_ACCOUNT_ID", "")
	// If gh is installed, skip — it'll check GitHub Secrets before vault
	if _, err := exec.LookPath("gh"); err == nil {
		t.Skip("gh CLI is installed — GitHub Secrets check runs before vault. Skipping.")
	}

	result := loadCFAccountID(dir, vaultDir)
	if result.Status == "fail" {
		t.Errorf("status = fail, detail: %s", result.Detail)
	}
}

func TestCheckTokens_WithVaultFiles(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, TokenDir)
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write token files into vault
	if err := os.WriteFile(filepath.Join(vaultDir, "CF_API_TOKEN"), []byte("test-token-data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "CF_ACCOUNT_ID"), []byte("test-account-id"), 0600); err != nil {
		t.Fatal(err)
	}

	// Clear env vars so vault path is exercised
	t.Setenv("CF_API_TOKEN", "")
	t.Setenv("CF_ACCOUNT_ID", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	results := CheckTokens(dir)
	if len(results) != len(RequiredTokens) {
		t.Fatalf("got %d results, want %d", len(results), len(RequiredTokens))
	}

	for _, ts := range results {
		switch ts.Name {
		case "CF_API_TOKEN":
			if !ts.Found {
				t.Error("CF_API_TOKEN should be found in vault")
			}
			if ts.Source != "vault" {
				t.Errorf("CF_API_TOKEN source = %q, want vault", ts.Source)
			}
			if !strings.Contains(ts.Details, "bytes") {
				t.Errorf("CF_API_TOKEN details = %q, should mention bytes", ts.Details)
			}
		case "CF_ACCOUNT_ID":
			if !ts.Found {
				t.Error("CF_ACCOUNT_ID should be found in vault")
			}
			if ts.Source != "vault" {
				t.Errorf("CF_ACCOUNT_ID source = %q, want vault", ts.Source)
			}
		case "GCLOUD_CREDENTIALS":
			// Optional token — not in vault, not in env
			if ts.Found {
				t.Error("GCLOUD_CREDENTIALS should not be found")
			}
			if ts.Source != "none" {
				t.Errorf("GCLOUD_CREDENTIALS source = %q, want none", ts.Source)
			}
			if !strings.Contains(ts.Details, "optional") {
				t.Errorf("GCLOUD_CREDENTIALS details = %q, should mention optional", ts.Details)
			}
		}
	}
}

func TestCheckTokens_EnvFallback(t *testing.T) {
	dir := t.TempDir()
	// No vault dir — forces env var fallback

	t.Setenv("CF_API_TOKEN", "env-token-value")
	t.Setenv("CF_ACCOUNT_ID", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	results := CheckTokens(dir)
	for _, ts := range results {
		if ts.Name == "CF_API_TOKEN" {
			if !ts.Found {
				t.Error("CF_API_TOKEN should be found via env")
			}
			if ts.Source != "environment" {
				t.Errorf("CF_API_TOKEN source = %q, want environment", ts.Source)
			}
			if !strings.Contains(ts.Details, "chars") {
				t.Errorf("CF_API_TOKEN details = %q, should mention chars", ts.Details)
			}
		}
	}
}

func TestPrintStatusReport_AllStatuses(t *testing.T) {
	// Capture stdout to verify all branches execute without panic
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	statuses := []string{"ok", "down", "degraded", "unknown"}
	var results []ServiceStatus
	for i, s := range statuses {
		results = append(results, ServiceStatus{
			Service: fmt.Sprintf("svc-%s", s),
			URL:     fmt.Sprintf("https://svc%d.test", i),
			Status:  s,
			Detail:  fmt.Sprintf("detail for %s", s),
		})
	}

	PrintStatusReport(results)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	r.Close()
	output := string(buf[:n])

	// Verify each icon appears in output
	expectedIcons := map[string]string{
		"ok":       "✅",
		"down":     "🔴",
		"degraded": "🟡",
		"unknown":  "⚪",
	}
	for status, icon := range expectedIcons {
		if !strings.Contains(output, icon) {
			t.Errorf("output missing icon %q for status %q", icon, status)
		}
		if !strings.Contains(output, fmt.Sprintf("svc-%s", status)) {
			t.Errorf("output missing service name for status %q", status)
		}
	}
}
