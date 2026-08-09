// Package infra provides OVAV infrastructure operations — DNS, tunnels,
// OAuth, token management, and connectivity verification.
//
// Commands:
//
//	ovav infra bootstrap   Install dependencies + load credentials
//	ovav infra check        Verify connectivity to all services
//	ovav infra dns          Manage Cloudflare DNS records
//	ovav infra tunnel       Manage Cloudflare tunnels
//	ovav infra tokens       Manage infrastructure credentials in vault
//
// Architecture:
//   - Uses OVAV vault (AES-256-GCM) for credential storage
//   - Cloudflare API v4 for DNS and tunnel operations
//   - gcloud CLI for Google Cloud OAuth (optional)
//   - All operations are idempotent and verify results
package infra

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// TokenDir is the vault directory for infrastructure tokens.
const TokenDir = ".ovav/vault/tokens"

// RequiredTokens lists the tokens needed for full infrastructure control.
var RequiredTokens = []TokenSpec{
	{
		Name:        "CF_API_TOKEN",
		Description: "Cloudflare API token (Zone:DNS:Edit + Account:Argo Tunnel:Edit)",
		Source:      "GitHub Secrets or Cloudflare Dashboard → API Tokens",
		EnvVar:      "CF_API_TOKEN",
	},
	{
		Name:        "CF_ACCOUNT_ID",
		Description: "Cloudflare account ID",
		Source:      "Cloudflare Dashboard → Overview → Account ID",
		EnvVar:      "CF_ACCOUNT_ID",
	},
	{
		Name:        "GCLOUD_CREDENTIALS",
		Description: "Google Cloud service account JSON (for OAuth redirect URI management)",
		Source:      "Google Cloud Console → IAM → Service Accounts",
		EnvVar:      "GOOGLE_APPLICATION_CREDENTIALS",
		Optional:    true,
	},
}

// TokenSpec describes a required infrastructure token.
type TokenSpec struct {
	Name        string
	Description string
	Source      string
	EnvVar      string
	Optional    bool
}

// ResolveRepoRoot finds the OVAV repository root.
func ResolveRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside an OVAV repository")
		}
		dir = parent
	}
}

// HasCommand checks if a CLI tool is installed.
func HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// VaultPath returns the path to the token vault directory.
func VaultPath(root string) string {
	return filepath.Join(root, TokenDir)
}
