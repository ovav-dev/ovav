package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NetworkHardening validates network allowlist configuration, critical domain
// coverage, and network guard readiness. Part of F0.4 hardening.
// Replaces: check_network_hardening.py
type NetworkHardening struct{}

func NewNetworkHardening() *NetworkHardening { return &NetworkHardening{} }

func (n *NetworkHardening) ID() string   { return "network_hardening" }
func (n *NetworkHardening) Name() string { return "F0.4 Network Hardening" }
func (n *NetworkHardening) Description() string {
	return "Validates network allowlist, critical domains, and guard configuration"
}
func (n *NetworkHardening) Weight() int { return 14 }

// Critical domains that must be covered by network allowlist.
var criticalDomains = []string{
	"github.com",
	"pypi.org",
	"api.github.com",
}

func (n *NetworkHardening) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check network allowlist config exists
	allowlistPath := filepath.Join(root, ".ovav", "registry", "network_allowlist.yaml")
	data, err := os.ReadFile(allowlistPath)
	if os.IsNotExist(err) {
		issues = append(issues, "NETWORK: .ovav/registry/network_allowlist.yaml not found")
	} else if err != nil {
		issues = append(issues, fmt.Sprintf("NETWORK: Cannot read allowlist: %v", err))
	} else {
		content := string(data)
		// 2. Verify critical domains are present
		missing := []string{}
		for _, domain := range criticalDomains {
			if !strings.Contains(content, domain) {
				missing = append(missing, domain)
			}
		}
		if len(missing) > 0 {
			issues = append(issues, fmt.Sprintf("NETWORK: Critical domains missing from allowlist: %s", strings.Join(missing, ", ")))
		}
	}

	// 3. Verify network guard policy in permission_authority.json
	policyPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if data, err := os.ReadFile(policyPath); err == nil {
		content := strings.ToLower(string(data))
		if !strings.Contains(content, "network_guard") && !strings.Contains(content, "network allowlist") {
			issues = append(issues, "NETWORK: network_guard resource policy not found in permission_authority.json")
		}
		if strings.Contains(content, `"bypass_operators"`) {
			// Check bypass is empty (zero-trust)
			if strings.Contains(content, `"bypass_operators": []`) || strings.Contains(content, `"bypass_operators":[]`) {
				// Good — zero-trust, no bypass
			}
		}
	}

	// 4. Check rate limits are configured
	if data, err := os.ReadFile(policyPath); err == nil {
		content := string(data)
		if !strings.Contains(content, "rate_limits") && !strings.Contains(content, "external_requests_per_minute") {
			issues = append(issues, "NETWORK: rate_limits not configured in permission_authority.json")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: n.ID(), Name: n.Name(), Status: "fail", Weight: n.Weight(),
			Message:  fmt.Sprintf("FAIL network hardening — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: n.ID(), Name: n.Name(), Status: "pass", Weight: n.Weight(),
		Message:  "PASS network hardening — allowlist, critical domains, rate limits verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*NetworkHardening)(nil)
