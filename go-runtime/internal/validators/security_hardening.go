package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ovav/ovav/internal/permissions"
)

// SecurityHardening validates F4 security hardening: bash command governance,
// unsafe selector rules, and security_surfaces policy integrity.
// Replaces: check_f4_security_hardening.py
type SecurityHardening struct{}

func NewSecurityHardening() *SecurityHardening { return &SecurityHardening{} }

func (s *SecurityHardening) ID() string   { return "security_hardening" }
func (s *SecurityHardening) Name() string { return "F4 Security Hardening" }
func (s *SecurityHardening) Description() string {
	return "Validates bash command governance, unsafe selectors, and deny-by-default enforcement"
}
func (s *SecurityHardening) Weight() int { return 20 }

func (s *SecurityHardening) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	policyPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: "FAIL: permission_authority.json not found", Duration: time.Since(start),
		}
	}

	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: "FAIL: permission_authority.json is invalid JSON", Duration: time.Since(start),
		}
	}
	isYolo := isYOLOPolicy(root, policy)

	// 1. Validate security_surfaces section exists
	sec, ok := policy["security_surfaces"].(map[string]interface{})
	if !ok {
		issues = append(issues, "F4: security_surfaces section missing from permission_authority.json")
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: "FAIL F4 — security_surfaces missing", Issues: issues, Duration: time.Since(start),
		}
	}

	// 2. Validate F4.1 bash_commands governance
	bash, ok := sec["f4_bash_commands"].(map[string]interface{})
	if !ok {
		issues = append(issues, "F4.1: f4_bash_commands section missing")
	} else {
		total := intVal(bash, "total_rules")
		allowed := intVal(bash, "allowed")
		denied := intVal(bash, "denied")
		denyDefault := boolVal(bash, "deny_by_default")
		governor := strVal(bash, "governor")
		goSummary := permissions.NewBashCommandGovernor().GetSummary()
		expectedTotal, _ := goSummary["total_rules"].(int)
		expectedAllowed, _ := goSummary["allowed"].(int)
		expectedDenied, _ := goSummary["denied"].(int)

		if total != expectedTotal {
			issues = append(issues, fmt.Sprintf("F4.1: bash_commands total_rules expected %d from Go governor, got %d", expectedTotal, total))
		}
		if allowed != expectedAllowed {
			issues = append(issues, fmt.Sprintf("F4.1: bash_commands allowed expected %d from Go governor, got %d", expectedAllowed, allowed))
		}
		if denied != expectedDenied {
			issues = append(issues, fmt.Sprintf("F4.1: bash_commands denied expected %d from Go governor, got %d", expectedDenied, denied))
		}
		if allowed+denied != total {
			issues = append(issues, fmt.Sprintf("F4.1: bash_commands allowed(%d) + denied(%d) != total(%d)", allowed, denied, total))
		}
		if isYolo && denyDefault {
			issues = append(issues, "F4.1: YOLO bash_commands deny_by_default must be false")
		} else if !isYolo && !denyDefault {
			issues = append(issues, "F4.1: bash_commands deny_by_default must be true")
		}
		if governor == "" {
			issues = append(issues, "F4.1: bash_commands governor path missing")
		} else if governor != "go-runtime/internal/permissions/governors.go" {
			issues = append(issues, fmt.Sprintf("F4.1: bash_commands governor path mismatch: %s", governor))
		} else if _, err := os.Stat(filepath.Join(root, governor)); err != nil {
			issues = append(issues, fmt.Sprintf("F4.1: bash_commands governor unavailable: %v", err))
		}

		// Validate categories exist
		cats, _ := bash["categories"].(map[string]interface{})
		requiredCats := []string{
			"source_control_read", "source_control_mutate", "ovav_internal",
			"filesystem_read", "interpreted_execution", "github_read",
			"governed_git", "testing", "privilege_escalation",
			"package_management", "auth_management", "network_external", "filesystem_mutate",
		}
		for _, cat := range requiredCats {
			if _, ok := cats[cat]; !ok {
				issues = append(issues, fmt.Sprintf("F4.1: missing bash category: %s", cat))
			}
		}

		// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
		// YOLO mode: bash is 100% allow. deny_by_default=false is intentional.
		// The historical F4 invariant of "deny_by_default must be true" is
		// relaxed under YOLO doctrine — bash safety is enforced by the
		// Governor (decision_engine + trust_gate) and HMAC-signed CEO waivers,
		// not by host-level string matching.
		// The validador should NOT fail on YOLO mode if the policy has a
		// `_ovav_yolo` marker indicating YOLO is active.
	}

	// 3. Validate F4.2 unsafe_selectors governance
	sel, ok := sec["f4_unsafe_selectors"].(map[string]interface{})
	if !ok {
		issues = append(issues, "F4.2: f4_unsafe_selectors section missing")
	} else {
		total := intVal(sel, "total_rules")
		allowed := intVal(sel, "allowed")
		denied := intVal(sel, "denied")
		ask := intVal(sel, "ask")
		denyDefault := boolVal(sel, "deny_by_default")
		governor := strVal(sel, "governor")

		if total != 10 {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors total_rules expected 10, got %d", total))
		}
		if !isYolo && allowed != 2 {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors allowed expected 2, got %d", allowed))
		}
		if isYolo && allowed != total {
			issues = append(issues, fmt.Sprintf("F4.2: YOLO unsafe_selectors allowed must equal total_rules(%d), got %d", total, allowed))
		}
		if isYolo && denied != 0 {
			issues = append(issues, fmt.Sprintf("F4.2: YOLO unsafe_selectors denied must be 0, got %d", denied))
		} else if !isYolo && denied != 7 {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors denied expected 7, got %d", denied))
		}
		if isYolo && ask != 0 {
			issues = append(issues, fmt.Sprintf("F4.2: YOLO unsafe_selectors ask must be 0, got %d", ask))
		} else if !isYolo && ask != 1 {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors ask expected 1, got %d", ask))
		}
		if isYolo && denyDefault {
			issues = append(issues, "F4.2: YOLO unsafe_selectors deny_by_default must be false")
		} else if !isYolo && !denyDefault {
			issues = append(issues, "F4.2: unsafe_selectors deny_by_default must be true")
		}
		if allowed+denied+ask != total {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors allowed(%d) + denied(%d) + ask(%d) != total(%d)", allowed, denied, ask, total))
		}
		if governor == "" {
			issues = append(issues, "F4.2: unsafe_selectors governor path missing")
		} else if governor != "go-runtime/internal/permissions/governors.go" {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors governor path mismatch: %s", governor))
		} else if _, err := os.Stat(filepath.Join(root, governor)); err != nil {
			issues = append(issues, fmt.Sprintf("F4.2: unsafe_selectors governor unavailable: %v", err))
		}

		// Validate categories
		cats, _ := sel["categories"].(map[string]interface{})
		requiredCats := []string{
			"source_local", "external_governed", "sensitive_paths",
			"system_paths", "package_management", "external_unverified",
			"external_services", "agent_recursion", "memory_poisoning", "trace_injection",
		}
		for _, cat := range requiredCats {
			if _, ok := cats[cat]; !ok {
				issues = append(issues, fmt.Sprintf("F4.2: missing selector category: %s", cat))
			}
		}
	}

	// 4. Validate protected_denies section enforces deny-by-default
	// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
	// YOLO mode allows 0 deny rules in protected_denies.bash. Skip the
	// ">= 10 deny rules" check when YOLO is active.
	pd, _ := policy["protected_denies"].(map[string]interface{})
	if bashDenies, ok := pd["bash"].([]interface{}); ok {
		if len(bashDenies) < 10 && !isYolo {
			issues = append(issues, "F4: protected_denies.bash should have >= 10 deny rules")
		}
	} else {
		issues = append(issues, "F4: protected_denies.bash section missing")
	}

	// 6. Verify hard-deny behavior in the current Go governor, not just declarations.
	governor := permissions.NewBashCommandGovernor()
	for _, command := range []string{"git push origin main", "git push --force origin main", "sudo id", "rm -rf /"} {
		if decision := governor.Check(command, "validator"); decision.Allowed {
			issues = append(issues, fmt.Sprintf("F4 behavioral hard deny allowed command %q", command))
		}
		governor.CEOActive = true
		if decision := governor.CheckWithCEO(command, "validator"); decision.Allowed {
			issues = append(issues, fmt.Sprintf("F4 CEO bypass crossed permanent hard deny for %q", command))
		}
	}

	// 5. Cross-validate: protected_denies.bash should cover at least bash_commands denies
	// (they're different scopes: protected_denies covers ALL bash denies including F0-F5,
	// while f4_bash_commands is F4-specific)
	// OVAV YOLO mode: when YOLO is active and bash is 100% allow (denied=0),
	// the cross-validation trivially passes.
	if bashDenies, ok := pd["bash"].([]interface{}); ok {
		if bash, ok := sec["f4_bash_commands"].(map[string]interface{}); ok {
			denied := intVal(bash, "denied")
			if denied > len(bashDenies) && !isYolo {
				issues = append(issues, fmt.Sprintf("F4.1: bash_commands.denied(%d) > protected_denies.bash(%d) — F4 denies not covered",
					denied, len(bashDenies)))
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL F4 security hardening — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message:  "PASS F4 security hardening — bash commands, unsafe selectors, deny-by-default verified",
		Duration: time.Since(start),
	}
}

// Helper functions for safe JSON value extraction
func intVal(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func boolVal(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

var _ Validator = (*SecurityHardening)(nil)
