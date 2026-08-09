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

// F1Architecture validates F1 Architecture Integrity — permission authority,
// Rego policies, F1 tools, and guidance documents.
// Replaces: tools/validators/check_f1_architecture.py
type F1Architecture struct{}

func NewF1Architecture() *F1Architecture { return &F1Architecture{} }

func (v *F1Architecture) ID() string   { return "f1_architecture" }
func (v *F1Architecture) Name() string { return "F1 Architecture Integrity" }
func (v *F1Architecture) Description() string {
	return "Validates F1 architecture: permission authority, Rego policies, F1 tools, and guidance"
}
func (v *F1Architecture) Weight() int { return 7 }

func (v *F1Architecture) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Validate permission_authority.json exists and has v2 schema
	paPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	paData, err := os.ReadFile(paPath)
	if err != nil {
		issues = append(issues, "CRITICAL: .ovav/policy/permission_authority.json not found")
	} else {
		var pa map[string]interface{}
		if err := json.Unmarshal(paData, &pa); err != nil {
			issues = append(issues, fmt.Sprintf("CRITICAL: permission_authority.json invalid JSON: %v", err))
		} else {
			schemaVer, _ := pa["schema_version"].(string)
			if schemaVer != "ovav.permission_authority.v2" && schemaVer != "ovav.permission_authority.v3" {
				issues = append(issues, fmt.Sprintf("ERROR: permission_authority.json schema_version is '%s' (expected 'v2' or 'v3')", schemaVer))
			}
			// v2 fields
			if schemaVer == "ovav.permission_authority.v2" {
				if _, ok := pa["architecture"]; !ok {
					issues = append(issues, "ERROR: permission_authority.json missing 'architecture' section")
				}
				if _, ok := pa["resource_policies"]; !ok {
					issues = append(issues, "ERROR: permission_authority.json missing 'resource_policies' section")
				}
				if _, ok := pa["hardening_baseline"]; !ok {
					issues = append(issues, "ERROR: permission_authority.json missing 'hardening_baseline' section")
				}
			}
			// v3 fields
			if schemaVer == "ovav.permission_authority.v3" {
				if _, ok := pa["authority"]; !ok {
					issues = append(issues, "ERROR: permission_authority.json v3 missing 'authority' section")
				}
				if _, ok := pa["governor"]; !ok {
					issues = append(issues, "ERROR: permission_authority.json v3 missing 'governor' section")
				}
			}
		}
	}

	// 2. Check Rego policies exist (at least 3 .rego files)
	// Check both .ovav/policy/rego/ and .ovav/registry/rego_policies/
	regoDirs := []string{
		filepath.Join(root, ".ovav", "policy", "rego"),
		filepath.Join(root, ".ovav", "registry", "rego_policies"),
	}
	regoFound := 0
	var regoLocations []string
	for _, dir := range regoDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".rego") {
				regoFound++
				regoLocations = append(regoLocations, filepath.Join(dir, e.Name()))
			}
		}
	}
	if regoFound == 0 {
		issues = append(issues, "ERROR: No Rego policy files found in .ovav/policy/rego/ or .ovav/registry/rego_policies/")
	} else if regoFound < 3 {
		issues = append(issues, fmt.Sprintf("ERROR: Only %d Rego policy files found (expected >= 3)", regoFound))
	}

	// 3. Check F1 tools exist
	f1Tools := []string{
		"tools/permissions/rego_engine.py",
		"tools/permissions/simulate.py",
		"tools/permissions/verify.py",
	}
	for _, tool := range f1Tools {
		toolPath := filepath.Join(root, tool)
		if _, err := os.Stat(toolPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("ERROR: Missing F1 tool: %s", tool))
		}
	}

	// 4. Check bootstrap_verifier (F0.5 → F1 dependency)
	bootstrapPath := filepath.Join(root, "tools", "security", "bootstrap_verifier.py")
	if _, err := os.Stat(bootstrapPath); os.IsNotExist(err) {
		issues = append(issues, "WARNING: tools/security/bootstrap_verifier.py not found — F1 requires F0 bootstrap")
	}

	// 5. Check F1 EAL7 guidance exists
	eal7Path := filepath.Join(root, "docs", "research", "F1_EAL7_GUIDANCE.md")
	if _, err := os.Stat(eal7Path); os.IsNotExist(err) {
		issues = append(issues, "WARNING: docs/research/F1_EAL7_GUIDANCE.md not found")
	}

	// Determine result
	hasCritical := false
	for _, issue := range issues {
		if strings.HasPrefix(issue, "CRITICAL:") {
			hasCritical = true
			break
		}
	}
	if hasCritical || len(issues) > 0 {
		status := "fail"
		msg := fmt.Sprintf("FAIL F1 architecture integrity — %d issue(s)", len(issues))
		if hasCritical {
			msg = fmt.Sprintf("FAIL F1 architecture integrity — %d issue(s) including critical", len(issues))
		}
		return Result{
			ID:       v.ID(),
			Name:     v.Name(),
			Status:   status,
			Weight:   v.Weight(),
			Message:  msg,
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID:       v.ID(),
		Name:     v.Name(),
		Status:   "pass",
		Weight:   v.Weight(),
		Message:  fmt.Sprintf("PASS F1 architecture integrity — permission authority v2, %d Rego policies, F1 tools verified", regoFound),
		Duration: time.Since(start),
	}
}

var _ Validator = (*F1Architecture)(nil)
