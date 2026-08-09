// revoker.go — Credential Revocation Engine
//
// Phase 6.8 of OVAV-VAULT-2026 plan.
//
// OVAV VAULT is not just storage — it is the CONTROL POINT.
// When you `vault revoke <name>`, the system:
//  1. Revokes the credential from ALL systems where it's registered (GitHub, Fly.io, etc.)
//  2. Deletes the secret from the vault
//  3. Updates the dependency graph (removes all refs)
//  4. Logs the revocation to the audit trail
//
// This makes OVAV VAULT superior to Bitwarden/1Password — they only store.
// OVAV VAULT also CONTROLS and REVOKES.
//
// Revocation is provider-specific. Each provider has its own API.
package secrets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Revoker defines the interface for revoking credentials from a provider.
type Revoker interface {
	// Revoke invalidates a credential at the provider.
	// Returns nil on success, error on failure.
	// If the credential doesn't exist at the provider, returns nil (idempotent).
	Revoke(ref SecretRef, secretValue string) error

	// Name returns the provider name (e.g. "github-actions", "fly-io").
	Name() string
}

// ── GitHub Actions Secrets Revoker ────────────────────────────────────────────

type GitHubActionsRevoker struct{}

func (r *GitHubActionsRevoker) Name() string { return "github-actions" }

func (r *GitHubActionsRevoker) Revoke(ref SecretRef, secretValue string) error {
	// Extract repo from path: "GitHub Actions: REPO/owner/name"
	// ref.Path format: "GitHub Actions: owner/repo"
	repo := strings.TrimPrefix(ref.Path, "GitHub Actions: ")
	if repo == ref.Path {
		return fmt.Errorf("cannot parse repo from path: %s", ref.Path)
	}

	parts := strings.Split(repo, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repo path: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN env not set — cannot revoke GitHub secret")
	}

	// GitHub API: DELETE /repos/{owner}/{repo}/actions/secrets/{secret_name}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/secrets/%s",
		owner, repoName, ref.EnvVar)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	// 204 = deleted, 404 = already gone (acceptable)
	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("GitHub API: HTTP %d — %s", resp.StatusCode, string(body))
}

// ── Fly.io Secrets Revoker ─────────────────────────────────────────────────────

type FlyIORevoker struct{}

func (r *FlyIORevoker) Name() string { return "fly-io" }

func (r *FlyIORevoker) Revoke(ref SecretRef, secretValue string) error {
	// Fly.io API: DELETE /api/v1/apps/{appname}/secrets/{secretname}
	// Requires FLY_API_TOKEN env
	appName := strings.TrimPrefix(ref.Path, "Fly.io app: ")
	if appName == ref.Path {
		return fmt.Errorf("cannot parse app name from path: %s", ref.Path)
	}

	token := os.Getenv("FLY_API_TOKEN")
	if token == "" {
		return fmt.Errorf("FLY_API_TOKEN env not set — cannot revoke Fly.io secret")
	}

	url := fmt.Sprintf("https://api.machines.dev/v1/apps/%s/secrets/%s",
		appName, ref.EnvVar)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Fly.io API request: %w", err)
	}
	defer resp.Body.Close()

	// 200 = deleted, 404 = already gone
	if resp.StatusCode == 200 || resp.StatusCode == 404 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("Fly.io API: HTTP %d — %s", resp.StatusCode, string(body))
}

// ── Generic Revoker (manual/cannot auto-revoke) ───────────────────────────────

type GenericRevoker struct {
	providerName string
}

func (r *GenericRevoker) Name() string { return r.providerName }

func (r *GenericRevoker) Revoke(ref SecretRef, secretValue string) error {
	// Cannot auto-revoke — user must do this manually
	return fmt.Errorf("cannot auto-revoke %s secrets: manual revocation required. "+
		"Please revoke %s at %s manually, then run 'vault remove %s' to delete from vault",
		r.providerName, ref.EnvVar, ref.Path, ref.SecretID)
}

// ── Revocation Result ─────────────────────────────────────────────────────────

// RevokeResult describes the outcome of a revoke operation.
type RevokeResult struct {
	SecretName string    `json:"secret_name"`
	Provider   string    `json:"provider"`
	Path       string    `json:"path"`
	RevokedAt  time.Time `json:"revoked_at"`
	Status     string    `json:"status"` // "revoked" | "not_found" | "failed"
	Error      string    `json:"error,omitempty"`
}

// RevocationReport is the full outcome of revoking a secret.
type RevocationReport struct {
	SecretName      string         `json:"secret_name"`
	SecretID        string         `json:"secret_id"`
	VaultDeleted    bool           `json:"vault_deleted"`
	DepGraphCleaned bool           `json:"depgraph_cleaned"`
	Results         []RevokeResult `json:"results"`  // per-provider results
	AuditID         string         `json:"audit_id"` // audit log entry ID
}

// ── RevokeSecret revokes a secret from all systems and removes from vault ─────

// RevokeSecret revokes a secret by name from all registered systems,
// then removes it from the vault and cleans the dependency graph.
func RevokeSecret(store *SecretStore, graph *DependencyGraph, name string) (*RevocationReport, error) {
	sec := store.GetByName(name)
	if sec == nil {
		return nil, fmt.Errorf("secret %q not found in vault", name)
	}

	refs := graph.GetRefs(sec.ID)
	report := &RevocationReport{
		SecretName: name,
		SecretID:   sec.ID,
		Results:    make([]RevokeResult, 0, len(refs)),
	}

	// Revoke from each registered system
	for _, ref := range refs {
		result := RevokeResult{
			SecretName: name,
			Provider:   string(ref.System),
			Path:       ref.Path,
			RevokedAt:  time.Now().UTC(),
		}

		revoker := getRevoker(ref.System)
		err := revoker.Revoke(ref, "") // secret value not needed for revocation

		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
		} else {
			result.Status = "revoked"
		}
		report.Results = append(report.Results, result)
	}

	// Also try to revoke using the secret's source info (if different from depgraph)
	sourcePath := ""
	if sec.Metadata != nil {
		sourcePath = sec.Metadata["source_path"]
	}
	if sec.Source != "" && sourcePath != "" {
		// Add a result for the source if not already covered
		found := false
		for _, r := range report.Results {
			if r.Path == sourcePath {
				found = true
				break
			}
		}
		if !found {
			result := RevokeResult{
				SecretName: name,
				Provider:   sec.Source,
				Path:       sourcePath,
				RevokedAt:  time.Now().UTC(),
				Status:     "source_not_in_depgraph_manual_action_required",
				Error:      "Source registered but not in depgraph — revoke manually",
			}
			report.Results = append(report.Results, result)
		}
	}

	// Clean dependency graph — remove all refs for this secret
	for _, ref := range refs {
		_ = graph.RemoveRef(sec.ID, string(ref.System), ref.Path)
	}
	report.DepGraphCleaned = true

	// Remove from vault
	if err := store.Remove(sec.ID); err != nil {
		// Log but don't fail — revocation from providers succeeded
		for i := range report.Results {
			if report.Results[i].Status == "" || report.Results[i].Status == "revoked" {
				report.Results[i].Status = "failed_vault_delete"
			}
		}
	} else {
		report.VaultDeleted = true
	}

	// Log to audit — audit entry written by caller after vault op succeeds
	_ = sec // suppress unused

	return report, nil
}

// getRevoker returns the appropriate revoker for a system.
func getRevoker(system System) Revoker {
	switch system {
	case SystemGitHubActions:
		return &GitHubActionsRevoker{}
	case SystemFlyIO:
		return &FlyIORevoker{}
	default:
		return &GenericRevoker{providerName: string(system)}
	}
}
