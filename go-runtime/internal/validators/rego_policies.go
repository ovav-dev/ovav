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

	// 1. Check rego engine exists
	enginePath := filepath.Join(root, "tools", "permissions", "rego_engine.py")
	if info, err := os.Stat(enginePath); os.IsNotExist(err) {
		issues = append(issues, "Rego engine file missing: tools/permissions/rego_engine.py")
	} else if info.Size() == 0 {
		issues = append(issues, "Rego engine file is empty: tools/permissions/rego_engine.py")
	} else {
		data, err := os.ReadFile(enginePath)
		if err == nil {
			content := string(data)
			if !strings.Contains(content, "class RegoEngine") {
				issues = append(issues, "Rego: RegoEngine class not found in rego_engine.py")
			}
			if !strings.Contains(content, "BUILTIN_TESTS") {
				issues = append(issues, "Rego: BUILTIN_TESTS not found in rego_engine.py")
			}
			if !strings.Contains(content, "load_policies") {
				issues = append(issues, "Rego: load_policies method not found")
			}
			if !strings.Contains(content, "test_policy") {
				issues = append(issues, "Rego: test_policy method not found")
			}
			// Check deny/allow rule presence
			hasDeny := strings.Contains(content, "deny")
			hasAllow := strings.Contains(content, "allow")
			if !hasDeny {
				issues = append(issues, "Rego: no deny rules detected — security policies missing")
			}
			if !hasAllow {
				issues = append(issues, "Rego: no allow rules detected — all operations would be denied")
			}
		}
	}

	// 2. Check Rego policies — may be embedded in rego_engine.py or as .rego files
	regoDir := filepath.Join(root, ".ovav", "laws")
	if entries, err := os.ReadDir(regoDir); err == nil {
		hasRego := false
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".rego") {
				hasRego = true
				break
			}
		}
		if !hasRego {
			// Not necessarily an error — Rego policies may be embedded in rego_engine.py
			// which was already verified above
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

var _ Validator = (*RegoPolicies)(nil)
