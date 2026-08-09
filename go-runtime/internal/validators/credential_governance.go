package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CredentialGovernance validates credential governance, vault integrity,
// provider configuration, and scope isolation (product vs personal).
// Replaces: check_c3_2_credential_governance.py
type CredentialGovernance struct{}

func NewCredentialGovernance() *CredentialGovernance { return &CredentialGovernance{} }

func (c *CredentialGovernance) ID() string   { return "credential_governance" }
func (c *CredentialGovernance) Name() string { return "Credential Governance" }
func (c *CredentialGovernance) Description() string {
	return "Validates credential vault, provider config, scope isolation, and budget tracking"
}
func (c *CredentialGovernance) Weight() int { return 18 }

// Provider models expected in OVAV configuration.
var expectedProviders = []string{"deepseek", "openai", "anthropic", "qwen"}

// Required vault files.
var requiredVaultFiles = []string{
	"encrypt.go",
	"assets.go",
}

func (c *CredentialGovernance) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Validate Go vault package exists and has required files
	vaultDir := filepath.Join(root, "go-runtime", "internal", "vault")
	if info, err := os.Stat(vaultDir); os.IsNotExist(err) || !info.IsDir() {
		issues = append(issues, "VAULT: go-runtime/internal/vault/ directory not found")
	} else {
		for _, f := range requiredVaultFiles {
			vf := filepath.Join(vaultDir, f)
			if _, err := os.Stat(vf); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("VAULT: missing required file: internal/vault/%s", f))
			}
		}
		// Check _test.go exists for coverage
		testFile := filepath.Join(vaultDir, "encrypt_test.go")
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			issues = append(issues, "VAULT: missing encrypt_test.go — tests required for security-critical package")
		}
	}

	// 2. Validate permission_authority.json credential sections
	policyPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if data, err := os.ReadFile(policyPath); err == nil {
		var policy map[string]interface{}
		if err := json.Unmarshal(data, &policy); err != nil {
			issues = append(issues, "POLICY: permission_authority.json is invalid JSON")
		} else {
			// Check secrets_vault resource policy
			rp, _ := policy["resource_policies"].(map[string]interface{})
			if sv, ok := rp["secrets_vault"].(map[string]interface{}); ok {
				if sv["require_unlock"] != true {
					issues = append(issues, "POLICY: secrets_vault should require_unlock = true")
				}
			} else {
				issues = append(issues, "POLICY: secrets_vault resource policy missing")
			}

			// Check operator_profiles for credential governance
			op, _ := policy["operator_profiles"].(map[string]interface{})
			if len(op) < 2 {
				issues = append(issues, "POLICY: operator_profiles should define >= 2 operators")
			}

			// Validate scope isolation: thavren has product scope, others are restricted
			if thavren, ok := op["thavren"].(map[string]interface{}); ok {
				scopes, _ := thavren["scopes"].([]interface{})
				hasInstall := false
				for _, s := range scopes {
					if s == "install_sandbox" {
						hasInstall = true
					}
				}
				if !hasInstall {
					issues = append(issues, "POLICY: thavren should have install_sandbox scope")
				}
			} else {
				issues = append(issues, "POLICY: thavren operator profile missing")
			}

			if eidren, ok := op["eidren"].(map[string]interface{}); ok {
				mutate, _ := eidren["repo_local_mutate"].(string)
				if mutate != "deny_by_default" {
					issues = append(issues, "POLICY: eidren repo_local_mutate should be deny_by_default (scope isolation)")
				}
			} else {
				issues = append(issues, "POLICY: eidren operator profile missing")
			}

			// Check budget/session constraints
			cond, _ := policy["conditions"].(map[string]interface{})
			if sc, ok := cond["session_constraints"].(map[string]interface{}); ok {
				if sc["max_context_tokens"] == nil {
					issues = append(issues, "POLICY: session_constraints missing max_context_tokens")
				}
			} else {
				issues = append(issues, "POLICY: session_constraints section missing")
			}
		}
	} else {
		issues = append(issues, "POLICY: permission_authority.json not found")
	}

	// 3. Validate OVAV credentials config
	credsDir := filepath.Join(root, ".ovav", "credentials")
	if _, err := os.Stat(credsDir); os.IsNotExist(err) {
		// Not necessarily an error — credentials may use vault only
	}

	// 4. Check provider configuration references in codebase
	providerRefs := map[string]bool{}
	for _, prov := range expectedProviders {
		providerRefs[prov] = false
	}
	// Check opencode.json for provider references
	ocPath := filepath.Join(root, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		content := string(data)
		for _, prov := range expectedProviders {
			if strings.Contains(content, prov) {
				providerRefs[prov] = true
			}
		}
	}
	// Also check go-runtime for provider import references
	_ = filepath.Walk(filepath.Join(root, "go-runtime"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			data, _ := os.ReadFile(path)
			for _, prov := range expectedProviders {
				if !providerRefs[prov] && strings.Contains(string(data), prov) {
					providerRefs[prov] = true
				}
			}
		}
		return nil
	})

	missingProviders := []string{}
	for prov, found := range providerRefs {
		if !found {
			missingProviders = append(missingProviders, prov)
		}
	}
	if len(missingProviders) > 0 && len(missingProviders) == len(expectedProviders) {
		issues = append(issues, "PROVIDER: no provider references found in opencode.json or go-runtime — config may be stale")
	}

	// 5. Validate scope detection for OVAV system root
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")
	if data, err := os.ReadFile(capsPath); err == nil {
		content := string(data)
		if !strings.Contains(content, "product") {
			issues = append(issues, "SCOPE: caps.yaml missing product scope reference")
		}
		if strings.Contains(content, "personal") && strings.Contains(content, "scope") {
			// Check personal scope is correctly isolated
			if !strings.Contains(content, "product_surface_python: 0") {
				issues = append(issues, "SCOPE: personal scope configuration may be incorrectly placed in product system")
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: c.ID(), Name: c.Name(), Status: "fail", Weight: c.Weight(),
			Message:  fmt.Sprintf("FAIL credential governance — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: c.ID(), Name: c.Name(), Status: "pass", Weight: c.Weight(),
		Message:  "PASS credential governance — vault, policy, scope isolation verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*CredentialGovernance)(nil)
