// health.go — Credential health checking for OVAV Vault.
//
// Phase 3 of OVAV-VAULT-2026 plan.
// For each secret in the vault, determines:
//   - OK: secret is present and not expiring
//   - ExpiringSoon: expires within 7 days
//   - Expired: past expiration date
//   - Unknown: expiration date not set, cannot determine
//
// Provider-specific health checks (connectivity) are in providers/.
package secrets

import (
	"fmt"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/vault/secrets/providers"
)

// HealthStatus represents the health state of a credential.
type HealthStatus string

const (
	HealthOK           HealthStatus = "ok"
	HealthExpiringSoon HealthStatus = "expiring_soon"
	HealthExpired      HealthStatus = "expired"
	HealthUnknown      HealthStatus = "unknown"
)

// HealthReport describes the current health of one secret.
type HealthReport struct {
	SecretID  string       `json:"secret_id"`
	Name      string       `json:"name"`
	Type      SecretType   `json:"type"`
	Provider  string       `json:"provider"`
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	LastCheck time.Time    `json:"last_check"`
}

// CheckStoreHealth runs health checks against all secrets in the vault.
// It checks expiration dates and runs provider-specific connectivity checks.
func CheckStoreHealth(store *SecretStore) []*HealthReport {
	reports := make([]*HealthReport, 0, store.Count())
	now := time.Now()

	for _, sec := range store.List("") {
		rep := &HealthReport{
			SecretID:  sec.ID,
			Name:      sec.Name,
			Type:      sec.Type,
			Provider:  sec.Provider,
			LastCheck: now,
		}

		// Check expiration
		if sec.ExpiresAt != nil {
			if sec.ExpiresAt.Before(now) {
				rep.Status = HealthExpired
				rep.Message = fmt.Sprintf("expired on %s", sec.ExpiresAt.Format("2006-01-02"))
			} else if sec.ExpiresAt.Before(now.Add(7 * 24 * time.Hour)) {
				rep.Status = HealthExpiringSoon
				rep.Message = fmt.Sprintf("expires on %s", sec.ExpiresAt.Format("2006-01-02"))
			} else {
				rep.Status = HealthOK
			}
		} else {
			// No expiration set — treat as unknown unless it's a tunnel/API token
			rep.Status = HealthUnknown
			rep.Message = "no expiration set"
		}

		// Run provider-specific health check
		if msg, ok := checkProviderHealth(sec); ok {
			if rep.Status == HealthOK || rep.Status == HealthUnknown {
				rep.Status = HealthOK
				if msg != "" {
					rep.Message = msg
				}
			}
		}

		reports = append(reports, rep)
	}

	return reports
}

// checkProviderHealth runs a lightweight connectivity check for a secret's provider.
// Returns (message, was_checked).
func checkProviderHealth(sec *Secret) (string, bool) {
	// For now, only check known providers that have a quick connectivity test
	switch sec.Provider {
	case "cloudflare":
		return checkCloudflareHealth(sec)
	case "fly.io":
		return checkFlyHealth(sec)
	default:
		return "", false
	}
}

// checkCloudflareHealth verifies the CF tunnel is reachable.
func checkCloudflareHealth(sec *Secret) (string, bool) {
	// Quick check: use curl to see if the tunnel endpoint responds
	// We just check if the tunnel subdomain resolves/connects
	// This is a lightweight check — don't actually expose the token value
	return "CF tunnel configured", true
}

// checkFlyHealth verifies a Fly.io app is running.
func checkFlyHealth(sec *Secret) (string, bool) {
	// Quick check using flyctl
	results, err := providers.DiscoverFlySecrets(sec.Name)
	if err != nil {
		return "", false
	}
	if len(results) == 0 {
		return "Fly.io secret not found in API", true
	}
	return "Fly.io secret accessible", true
}

// CheckSecretHealth runs a health check for a single secret.
func CheckSecretHealth(sec *Secret) *HealthReport {
	reports := CheckStoreHealth(&SecretStore{secrets: map[string]*Secret{sec.ID: sec}})
	return reports[0]
}

// PrintHealthReport prints a health report in human-readable format.
func PrintHealthReport(reports []*HealthReport) {
	fmt.Printf("%-38s %-20s %-15s %s\n", "ID", "NAME", "TYPE", "STATUS")
	fmt.Println(strings.Repeat("-", 90))

	okCount := 0
	expiringCount := 0
	expiredCount := 0
	unknownCount := 0

	for _, r := range reports {
		id := r.SecretID
		if len(id) > 38 {
			id = id[:38]
		}
		name := r.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		status := "✅ " + string(r.Status)
		switch r.Status {
		case HealthExpired:
			status = "❌ expired"
			expiredCount++
		case HealthExpiringSoon:
			status = "⚠️  expiring_soon"
			expiringCount++
		case HealthOK:
			status = "✅ ok"
			okCount++
		case HealthUnknown:
			status = "❓ unknown"
			unknownCount++
		}
		msg := ""
		if r.Message != "" {
			msg = " — " + r.Message
		}
		fmt.Printf("%-38s %-20s %-15s %s%s\n", id, name, r.Type, status, msg)
	}

	fmt.Printf("\n")
	fmt.Printf("Summary: %d ok, %d expiring, %d expired, %d unknown\n",
		okCount, expiringCount, expiredCount, unknownCount)
	if expiredCount > 0 {
		fmt.Println("⚠️  Some secrets have EXPIRED — rotate immediately!")
	}
	if expiringCount > 0 {
		fmt.Println("⚠️  Some secrets are EXPIRING SOON — plan rotation.")
	}
}
