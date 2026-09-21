package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RegoPolicies validates Rego engine configuration and deny/allow rule presence.
// Replaces: check_rego_policies.py
type RegoPolicies struct{}

func NewRegoPolicies() *RegoPolicies { return &RegoPolicies{} }

func (r *RegoPolicies) ID() string   { return "rego_policies" }
func (r *RegoPolicies) Name() string { return "Rego Policy Integrity" }
func (r *RegoPolicies) Description() string {
	return "Validates Rego policy engine and deny/allow rule presence"
}
func (r *RegoPolicies) Weight() int { return 5 }

func (r *RegoPolicies) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check the current Go-native engine and its public policy surface.
	engineRel := "go-runtime/internal/permissions/rego_engine.go"
	enginePath := filepath.Join(root, engineRel)
	if info, err := os.Stat(enginePath); os.IsNotExist(err) {
		issues = append(issues, "Rego engine file missing: "+engineRel)
	} else if info.Size() == 0 {
		issues = append(issues, "Rego engine file is empty: "+engineRel)
	} else {
		data, err := os.ReadFile(enginePath)
		if err == nil {
			content := string(data)
			for _, signature := range []string{"type RegoEngine", "NewRegoEngine", "LoadPolicies", "Evaluate", "TestPolicy", "BuiltinTests"} {
				if !strings.Contains(content, signature) {
					issues = append(issues, fmt.Sprintf("Rego engine missing Go signature %q", signature))
				}
			}
		}
	}

	// 2. Preserve actual policy checks: rules belong in .rego authority files,
	// not in implementation source.
	policies, err := readRegoPolicies(root)
	if err != nil {
		issues = append(issues, err.Error())
	} else {
		if !strings.Contains(policies, "deny") {
			issues = append(issues, "Rego: no deny rules detected — security policies missing")
		}
		if !strings.Contains(policies, "allow") {
			issues = append(issues, "Rego: no allow rules detected — all operations would be denied")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(),
			Message:  fmt.Sprintf("FAIL rego policies — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: r.ID(), Name: r.Name(), Status: "pass", Weight: r.Weight(),
		Message:  "PASS rego policies — engine and rules verified",
		Duration: time.Since(start),
	}
}

func readRegoPolicies(root string) (string, error) {
	var policies strings.Builder
	count := 0
	for _, rel := range []string{".ovav/policy/rego", ".ovav/registry/rego_policies"} {
		entries, err := os.ReadDir(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".rego") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root, rel, entry.Name()))
			if err != nil {
				return "", fmt.Errorf("Rego: cannot read policy %s: %w", filepath.Join(rel, entry.Name()), err)
			}
			count++
			policies.Write(data)
			policies.WriteByte('\n')
		}
	}
	if count == 0 {
		return "", fmt.Errorf("Rego: no .rego policy files found")
	}
	return policies.String(), nil
}

var _ Validator = (*RegoPolicies)(nil)
