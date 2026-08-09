// providers/github.go — GitHub Secrets discovery via GitHub API.
//
// Uses `gh api` to list organization/repo secrets without reading values
// (GitHub does not expose secret values via API — they are encrypted at rest).
//
// Discovery records: name, inferred type, last_updated, source="github_secrets".
// For full ingestion, user must provide values via `vault secrets add`.
package providers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/vault/secrets/types"
)

// GitHubDiscoveryResult holds metadata about a GitHub secret.
type GitHubDiscoveryResult struct {
	Name      string // e.g. "CLOUDFLARE_API_TOKEN"
	Type      types.SecretType
	UpdatedAt time.Time `json:"updated_at"`
	Source    string    // always "github_secrets"
}

// DiscoverGitHubSecrets lists secrets from a GitHub repository.
// Uses `gh api` — requires `gh` CLI authenticated.
func DiscoverGitHubSecrets(owner, repo string) ([]GitHubDiscoveryResult, error) {
	// gh api returns JSON array of {name, updated_at, etc.}
	// We use --jq to extract just name and updated_at.
	// Note: --jq with array produces NDJSON; use map() to get proper JSON array.
	cmd := exec.Command("gh", "api",
		fmt.Sprintf("repos/%s/%s/actions/secrets", owner, repo),
		"--jq", "[.secrets[] | {name: .name, updated_at: .updated_at}]")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api: %w", err)
	}

	var raw []struct {
		Name      string `json:"name"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing gh api output: %w", err)
	}

	results := make([]GitHubDiscoveryResult, 0, len(raw))
	for _, s := range raw {
		updatedAt, _ := time.Parse(time.RFC3339, s.UpdatedAt)
		results = append(results, GitHubDiscoveryResult{
			Name:      s.Name,
			Type:      types.InferType(s.Name),
			UpdatedAt: updatedAt,
			Source:    "github_secrets",
		})
	}

	return results, nil
}

// DiscoverGitHubOrg lists all secrets for each repo in an org.
// It skips repos where the authenticated token lacks access.
func DiscoverGitHubOrg(org string) (map[string][]GitHubDiscoveryResult, error) {
	// Get all repos in the org
	cmd := exec.Command("gh", "api", fmt.Sprintf("orgs/%s/repos", org),
		"--jq", ".[].name")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing org repos: %w", err)
	}

	var repoNames []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			repoNames = append(repoNames, line)
		}
	}

	all := make(map[string][]GitHubDiscoveryResult)
	for _, repo := range repoNames {
		secrets, err := DiscoverGitHubSecrets(org, repo)
		if err != nil {
			// Skip repos we don't have access to
			continue
		}
		if len(secrets) > 0 {
			all[repo] = secrets
		}
	}

	return all, nil
}
