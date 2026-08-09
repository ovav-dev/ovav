package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/hooks"
)

// BootstrapResult holds the outcome of a bootstrap operation.
type BootstrapResult struct {
	Step   string
	Status string // "ok", "skip", "fail"
	Detail string
}

// Bootstrap installs required dependencies and loads credentials.
// It is idempotent — running it multiple times is safe.
func Bootstrap(root string) ([]BootstrapResult, error) {
	var results []BootstrapResult

	// Step 1: Ensure vault directory exists
	vaultDir := VaultPath(root)
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return results, fmt.Errorf("vault dir: %w", err)
	}
	results = append(results, BootstrapResult{"vault-dir", "ok", vaultDir})

	// Step 2: Check/install gcloud CLI
	if HasCommand("gcloud") {
		results = append(results, BootstrapResult{"gcloud", "ok", "already installed"})
	} else {
		r := installGcloud()
		results = append(results, r)
	}

	// Step 3: Load CF_API_TOKEN from environment or GitHub Secrets
	r := loadCFToken(root, vaultDir)
	results = append(results, r)

	// Step 4: Load CF_ACCOUNT_ID
	r = loadCFAccountID(root, vaultDir)
	results = append(results, r)

	// Step 5: Verify connectivity
	if err := verifyCloudflareConnectivity(vaultDir); err != nil {
		results = append(results, BootstrapResult{"cf-verify", "fail", err.Error()})
	} else {
		results = append(results, BootstrapResult{"cf-verify", "ok", "Cloudflare API reachable"})
	}

	// Step 6: Install git hooks
	hookMgr := hooks.NewManager(root)
	if _, err := hookMgr.Install(); err != nil {
		results = append(results, BootstrapResult{"git-hooks", "fail", err.Error()})
	} else {
		results = append(results, BootstrapResult{"git-hooks", "ok", "OVAV git hooks installed"})
	}

	// Step 7: Encrypt tokens to tokens.enc (if vault key available)
	if err := EncryptTokensToVault(root); err != nil {
		results = append(results, BootstrapResult{"tokens-enc", "fail", err.Error()})
	} else {
		results = append(results, BootstrapResult{"tokens-enc", "ok", "tokens encrypted to vault"})
	}

	return results, nil
}

// installGcloud attempts to install the Google Cloud CLI.
func installGcloud() BootstrapResult {
	// Check common paths first
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, "google-cloud-sdk", "bin", "gcloud"),
		"/usr/local/google-cloud-sdk/bin/gcloud",
		"/snap/bin/gcloud",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return BootstrapResult{"gcloud", "ok", fmt.Sprintf("found at %s", p)}
		}
	}

	// CRITICAL FIX (C1): Removed sudo exec — privilege escalation vector.
	// Bootstrap must NOT auto-elevate. The user must install dependencies explicitly.
	// Previously: exec.Command("sudo", "snap", "install", ...) — bypassed all permission gates.
	// Now: detection only. If gcloud is needed, the user installs it manually.

	return BootstrapResult{"gcloud", "skip", "not installed — install manually: https://cloud.google.com/sdk/docs/install"}
}

// loadCFToken loads the Cloudflare API token from environment or GitHub Secrets.
// Supports both API Token (CF_API_TOKEN) and Global API Key (CF_API_KEY + CF_EMAIL).
func loadCFToken(root, vaultDir string) BootstrapResult {
	// Priority 1: environment variable — API Token
	if token := os.Getenv("CF_API_TOKEN"); token != "" {
		if err := os.WriteFile(filepath.Join(vaultDir, "CF_API_TOKEN"), []byte(token), 0600); err != nil {
			return BootstrapResult{"cf-token", "fail", err.Error()}
		}
		return BootstrapResult{"cf-token", "ok", "loaded API Token from environment"}
	}

	// Priority 2: environment variable — Global API Key
	if key := os.Getenv("CF_API_KEY"); key != "" {
		if err := os.WriteFile(filepath.Join(vaultDir, "CF_API_KEY"), []byte(key), 0600); err != nil {
			return BootstrapResult{"cf-token", "fail", err.Error()}
		}
		if email := os.Getenv("CF_EMAIL"); email != "" {
			os.WriteFile(filepath.Join(vaultDir, "CF_EMAIL"), []byte(email), 0600)
		}
		return BootstrapResult{"cf-token", "ok", "loaded Global API Key from environment"}
	}

	// Priority 3: already in vault
	if _, err := os.Stat(filepath.Join(vaultDir, "CF_API_TOKEN")); err == nil {
		return BootstrapResult{"cf-token", "ok", "API Token already in vault"}
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "CF_API_KEY")); err == nil {
		return BootstrapResult{"cf-token", "ok", "Global API Key already in vault"}
	}

	// Priority 4: GitHub Secrets via gh CLI
	if HasCommand("gh") {
		exists, err := checkGHSecret("CLOUDFLARE_API_TOKEN")
		if err == nil && exists {
			return BootstrapResult{"cf-token", "fail",
				"Token exists in GitHub Secrets but cannot be read. Export locally: export CF_API_TOKEN=<token>"}
		}
	}

	return BootstrapResult{"cf-token", "fail",
		"No Cloudflare credentials found. Export CF_API_TOKEN or CF_API_KEY + CF_EMAIL, or create token at https://dash.cloudflare.com/profile/api-tokens"}
}

// loadCFAccountID loads the Cloudflare account ID.
func loadCFAccountID(root, vaultDir string) BootstrapResult {
	if id := os.Getenv("CF_ACCOUNT_ID"); id != "" {
		os.WriteFile(filepath.Join(vaultDir, "CF_ACCOUNT_ID"), []byte(id), 0600)
		return BootstrapResult{"cf-account", "ok", "loaded from environment"}
	}

	if HasCommand("gh") {
		exists, err := checkGHSecret("CLOUDFLARE_ACCOUNT_ID")
		if err == nil && exists {
			return BootstrapResult{"cf-account", "fail",
				"CF_ACCOUNT_ID exists in GitHub Secrets — cannot read. Export locally: export CF_ACCOUNT_ID=<id>"}
		}
	}

	if _, err := os.Stat(filepath.Join(vaultDir, "CF_ACCOUNT_ID")); err == nil {
		return BootstrapResult{"cf-account", "ok", "already in vault"}
	}

	return BootstrapResult{"cf-account", "skip", "not available — some operations may use API to auto-detect"}
}

// checkGHSecret checks if a secret exists in GitHub Actions.
func checkGHSecret(name string) (bool, error) {
	cmd := exec.Command("gh", "secret", "list", "--json", "name")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), name), nil
}

// verifyCloudflareConnectivity tests that the Cloudflare API is reachable.
func verifyCloudflareConnectivity(vaultDir string) error {
	creds, err := loadCFCredentials(vaultDir)
	if err != nil {
		return err
	}
	if creds.apiToken != "" {
		return cfAPIVerify(creds.apiToken)
	}
	return cfAPIVerify(creds.apiKey)
}
