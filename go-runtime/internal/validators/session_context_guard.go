package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SessionContextGuard validates session integrity seal, governance files, and injection detection.
// Replaces: session_context_guard.py
type SessionContextGuard struct{}

func NewSessionContextGuard() *SessionContextGuard { return &SessionContextGuard{} }

func (s *SessionContextGuard) ID() string   { return "session_context_guard" }
func (s *SessionContextGuard) Name() string { return "Session Context Guard" }
func (s *SessionContextGuard) Description() string {
	return "Validates session integrity seal, governance files, and injection detection"
}
func (s *SessionContextGuard) Weight() int { return 20 }

const integritySeal = "OVAV_INTEGRITY_SEAL v1.0.0"

// Governance files that MUST remain intact.
var governanceFiles = []string{
	"AGENTS.md",
	".ovav/policy/permission_authority.json",
	".ovav/service_areas/shared/lead_work_method_contract.yaml",
	".ovav/service_areas/shared/context_economy_contract.yaml",
	".ovav/service_areas/shared/visual_delivery_contract.yaml",
	".ovav/plan/caps.yaml",
	"go-runtime/internal/validators/host_config_drift.go",
	"go-runtime/internal/validators/gate_self_protection.go",
	"go-runtime/internal/validators/head_integrity.go",
}

// Injection patterns and their descriptions.
var injectionPatterns = []struct {
	re   *regexp.Regexp
	desc string
	crit bool
}{
	{regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|above|prior)\s+instructions`), "ignore_previous_instructions", false},
	{regexp.MustCompile(`(?i)you\s+are\s+now\s+(a\s+)?(different|new)\s+(role|assistant|agent)`), "role_override_attempt", false},
	{regexp.MustCompile(`(?i)bypass\s+(the\s+)?(gate|blockade|protection|defense)`), "gate_bypass_attempt", true},
	{regexp.MustCompile(`(?i)disable\s+(all\s+)?(security|protection|defense|validators)`), "security_disable_attempt", false},
	{regexp.MustCompile(`(?i)write\s+to\s+(~/.config|~/.local|/etc|/boot|/sys)`), "global_write_attempt", false},
	{regexp.MustCompile(`(?i)force\s+push.*(--force|-\s*f)`), "force_push_attempt", true},
	{regexp.MustCompile(`(?i)(sudo|root)\s+(access|privilege)`), "privilege_escalation_attempt", true},
	{regexp.MustCompile(`(?i)(delete|remove|unlink)\s+(all\s+)?(gates|blockades|validators)`), "gate_deletion_attempt", true},
}

func (s *SessionContextGuard) checkGovernanceFiles(root string) (intact int, compromised []string, missing []string) {
	for _, rel := range governanceFiles {
		fullPath := filepath.Join(root, rel)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			missing = append(missing, rel)
			continue
		}
		_ = info

		// For .md files, check integrity seal
		if strings.HasSuffix(rel, ".md") {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				compromised = append(compromised, fmt.Sprintf("%s (unreadable)", rel))
				continue
			}
			if !strings.Contains(string(data), integritySeal) {
				compromised = append(compromised, fmt.Sprintf("%s (integrity seal missing)", rel))
				continue
			}
		}
		intact++
	}
	return
}

func (s *SessionContextGuard) scanForInjection(content string) []string {
	var findings []string
	for _, p := range injectionPatterns {
		if p.re.MatchString(content) {
			severity := "high"
			if p.crit {
				severity = "critical"
			}
			findings = append(findings, fmt.Sprintf("INJECTION: %s (severity: %s)", p.desc, severity))
		}
	}
	return findings
}

func (s *SessionContextGuard) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check governance files integrity
	intact, compromised, missing := s.checkGovernanceFiles(root)
	if len(compromised) > 0 {
		for _, c := range compromised {
			issues = append(issues, fmt.Sprintf("session_context: governance file compromised — %s", c))
		}
	}
	if len(missing) > 0 {
		for _, m := range missing {
			issues = append(issues, fmt.Sprintf("session_context: governance file missing — %s", m))
		}
	}

	// 2. Scan AGENTS.md for injection patterns
	agentsPath := filepath.Join(root, "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err == nil {
		injections := s.scanForInjection(string(data))
		issues = append(issues, injections...)
	}

	if len(issues) > 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL session context guard — %d/%d files intact, %d issue(s)", intact, len(governanceFiles), len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message:  fmt.Sprintf("PASS session context guard — %d/%d governance files intact", intact, len(governanceFiles)),
		Duration: time.Since(start),
	}
}

var _ Validator = (*SessionContextGuard)(nil)
