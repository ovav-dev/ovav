package infra

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ServiceStatus represents the connectivity status of an external service.
type ServiceStatus struct {
	Service string `json:"service"`
	Status  string `json:"status"` // "ok", "degraded", "down", "unknown"
	URL     string `json:"url"`
	Detail  string `json:"detail,omitempty"`
}

// CheckAll verifies connectivity to all OVAV infrastructure services.
func CheckAll(root string) []ServiceStatus {
	vaultDir := VaultPath(root)
	var results []ServiceStatus

	// 1. Cloudflare API
	results = append(results, checkCloudflare(vaultDir))

	// 2. OVAV surfaces
	results = append(results, checkHTTP("ovav.dev", "https://ovav.dev", 200))
	results = append(results, checkHTTP("d678beea.ovav.dev (health)", "https://d678beea.ovav.dev/health", 200))

	// 3. DNS — verify cpanel.ovav.dev is GONE
	results = append(results, checkDNSGone("cpanel.ovav.dev", vaultDir))

	// 4. Tunnel — verify d678beea is routed
	results = append(results, checkTunnelRouting("d678beea.ovav.dev", vaultDir))

	// 5. Fly.io
	results = append(results, checkFlyIO())

	// 6. GitHub
	results = append(results, checkGitHub())

	// 7. Google Cloud (if gcloud installed)
	if HasCommand("gcloud") {
		results = append(results, checkGoogleCloud())
	}

	return results
}

func checkCloudflare(vaultDir string) ServiceStatus {
	if err := verifyCloudflareConnectivity(vaultDir); err != nil {
		return ServiceStatus{"Cloudflare API", "down", cfAPI, err.Error()}
	}
	return ServiceStatus{"Cloudflare API", "ok", cfAPI, "authenticated"}
}

func checkHTTP(name, url string, expectStatus int) ServiceStatus {
	resp, err := http.Get(url)
	if err != nil {
		return ServiceStatus{name, "down", url, err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == expectStatus || (expectStatus == 200 && resp.StatusCode < 400) {
		return ServiceStatus{name, "ok", url, fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	// Check if behind Cloudflare Access (302 to login)
	if resp.StatusCode == 302 {
		loc := resp.Header.Get("Location")
		if strings.Contains(loc, "cloudflareaccess") {
			return ServiceStatus{name, "degraded", url, "behind Cloudflare Access (need service token for automated checks)"}
		}
	}

	return ServiceStatus{name, "degraded", url, fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, expectStatus)}
}

func checkDNSGone(hostname, vaultDir string) ServiceStatus {
	result, _ := CheckDNSRecord("ovav.dev", hostname, vaultDir)
	switch result.Action {
	case "not_found":
		return ServiceStatus{"DNS: " + hostname, "ok", hostname, "NXDOMAIN — correctly removed ✓"}
	case "found":
		return ServiceStatus{"DNS: " + hostname, "degraded", hostname,
			fmt.Sprintf("STILL EXISTS → %s (proxied=%v) — must be deleted", result.Record.Content, result.Record.Proxied)}
	default:
		return ServiceStatus{"DNS: " + hostname, "unknown", hostname, "cannot verify (no CF_API_TOKEN?)"}
	}
}

func checkTunnelRouting(hostname, vaultDir string) ServiceStatus {
	msg, err := VerifyTunnelHostname(hostname, vaultDir)
	if err != nil {
		return ServiceStatus{"Tunnel: " + hostname, "unknown", hostname, err.Error()}
	}
	if strings.Contains(msg, "NOT in") {
		return ServiceStatus{"Tunnel: " + hostname, "degraded", hostname, msg}
	}
	return ServiceStatus{"Tunnel: " + hostname, "ok", hostname, msg}
}

func checkFlyIO() ServiceStatus {
	resp, err := http.Get("https://d678beea.ovav.dev/health")
	if err != nil {
		return ServiceStatus{"Fly.io", "down", "d678beea.ovav.dev", err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		return ServiceStatus{"Fly.io", "ok", "d678beea.ovav.dev", fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	return ServiceStatus{"Fly.io", "degraded", "d678beea.ovav.dev", fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

func checkGitHub() ServiceStatus {
	req, _ := http.NewRequest("GET", "https://api.github.com/rate_limit", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ServiceStatus{"GitHub API", "down", "api.github.com", err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		var rl struct {
			Rate struct {
				Remaining int `json:"remaining"`
			} `json:"rate"`
		}
		json.NewDecoder(resp.Body).Decode(&rl)
		return ServiceStatus{"GitHub API", "ok", "api.github.com", fmt.Sprintf("%d requests remaining", rl.Rate.Remaining)}
	}
	return ServiceStatus{"GitHub API", "degraded", "api.github.com", fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

func checkGoogleCloud() ServiceStatus {
	return ServiceStatus{"Google Cloud", "unknown", "gcloud", "run 'gcloud auth list' to verify"}
}

// PrintStatusReport formats a check result as a readable table.
func PrintStatusReport(results []ServiceStatus) {
	fmt.Println()
	fmt.Println("  OVAV Infrastructure Status")
	fmt.Println("  ═══════════════════════════")
	for _, r := range results {
		icon := "✅"
		switch r.Status {
		case "down":
			icon = "🔴"
		case "degraded":
			icon = "🟡"
		case "unknown":
			icon = "⚪"
		}
		fmt.Printf("  %s %-35s %s\n", icon, r.Service, r.Detail)
	}
	fmt.Println()
}

// TokenStatus represents the availability of a token.
type TokenStatus struct {
	Name    string
	Found   bool
	Source  string
	EnvVar  string
	Details string
}

// CheckTokens verifies which tokens are available.
func CheckTokens(root string) []TokenStatus {
	vaultDir := VaultPath(root)
	var results []TokenStatus

	for _, spec := range RequiredTokens {
		ts := TokenStatus{Name: spec.Name, EnvVar: spec.EnvVar}

		// Check vault
		if data, err := os.ReadFile(filepath.Join(vaultDir, spec.Name)); err == nil && len(data) > 0 {
			ts.Found = true
			ts.Source = "vault"
			ts.Details = fmt.Sprintf("%d bytes", len(data))
		} else if val := os.Getenv(spec.EnvVar); val != "" {
			ts.Found = true
			ts.Source = "environment"
			ts.Details = fmt.Sprintf("%d chars", len(val))
		} else {
			ts.Found = false
			ts.Source = "none"
			if spec.Optional {
				ts.Details = "optional — not required"
			} else {
				ts.Details = fmt.Sprintf("missing — %s", spec.Source)
			}
		}

		results = append(results, ts)
	}

	return results
}
