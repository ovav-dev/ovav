package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RuntimeWiring validates harness runtime wiring: surface files exist,
// required governance terms are present, and stale patterns are absent.
// It is HARNESS-AWARE: MiMoCode harness (areas-only) only validates shared
// surfaces; full-hierarchy harnesses also validate harness-specific agents.
// Replaces: check_opencode_runtime_wiring.py
type RuntimeWiring struct{}

func NewRuntimeWiring() *RuntimeWiring { return &RuntimeWiring{} }

func (r *RuntimeWiring) ID() string   { return "runtime_wiring" }
func (r *RuntimeWiring) Name() string { return "Runtime Wiring" }
func (r *RuntimeWiring) Description() string {
	return "Validates harness surface files, governance terms, and stale pattern detection (harness-aware)"
}
func (r *RuntimeWiring) Weight() int { return 14 }

// sharedSurfaces are .opencode command surfaces shared across all harnesses.
var sharedSurfaces = []struct {
	path  string
	label string
}{
	{".opencode/commands/ovav-work.md", "ovav-work command"},
	{".opencode/commands/ovav-context.md", "ovav-context command"},
	{".opencode/commands/ovav-validate.md", "ovav-validate command"},
	{".opencode/commands/ovav-close.md", "ovav-close command"},
	{".opencode/commands/ovav-status.md", "ovav-status command"},
}

// fullHierarchySurfaces are agent surfaces only present in full-hierarchy harnesses
// (opencode, claude-code, cursor). NOT present in MiMoCode harness.
var fullHierarchySurfaces = []struct {
	path  string
	label string
}{
	{".ovav/service_areas/platform_engineering/area_boundaries.yaml", "Platform Engineering agent"},
	{".ovav/service_areas/platform_engineering/lead_contract.yaml", "Thavren lead agent"},
	{".ovav/service_areas/research_intelligence/lead_contract.yaml", "Eidren lead agent"},
}

// Required governance terms for platform engineering agent.
var platformTerms = []string{
	"workspace_safety_gate",
	"ovav_git_push_gate",
	"protected_branch_gate",
	"check_living_integrity",
	"permission_authority.json",
	"Governance Wiring",
}

// Required governance terms for ovav-work command.
var ovavWorkTerms = []string{
	"Service Area Router",
	"Context Gateway",
	"Tool Gateway",
	"Handoff Protocol",
}

// Required governance terms for ovav-context command.
var ovavContextTerms = []string{
	"Context Gateway",
	"context_gateway.py",
	"sanitized handoff",
}

// Required governance terms for ovav-close command.
var ovavCloseTerms = []string{
	"runtime enforcement validation before closure",
	"Closure is blocked",
}

// Stale patterns that must NOT appear in active surfaces.
var stalePatterns = []string{
	"OVAV is post-BUILD 16",
	"BUILD 17 uses it for canonical review",
	"Inside the OVAV repository root, Eidren may read, edit, create files",
}

// surfacesForHarness returns the full surface list for the given harness.
func surfacesForHarness(h Harness) []struct {
	path  string
	label string
} {
	combined := append([]struct {
		path  string
		label string
	}{}, sharedSurfaces...)
	if h.isFullHierarchy() {
		combined = append(combined, fullHierarchySurfaces...)
	}
	return combined
}

func (r *RuntimeWiring) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	harness := DetectHarness(root)
	requiredSurfaces := surfacesForHarness(harness)
	agentsDir := harness.agentsDir(root)

	// 1. Verify all required surfaces exist and have content
	for _, sf := range requiredSurfaces {
		fullPath := filepath.Join(root, sf.path)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING: %s (%s)", sf.path, sf.label))
			continue
		}
		if info.Size() == 0 {
			issues = append(issues, fmt.Sprintf("EMPTY: %s (%s)", sf.path, sf.label))
		}
	}

	// 2. Validate platform engineering agent governance terms (full-hierarchy only)
	if harness.isFullHierarchy() {
		platformPath := filepath.Join(agentsDir, "area-platform-engineering.md")
		if data, err := os.ReadFile(platformPath); err == nil {
			content := string(data)
			for _, term := range platformTerms {
				if !strings.Contains(strings.ToLower(content), strings.ToLower(term)) {
					issues = append(issues, fmt.Sprintf("PLATFORM: area-platform-engineering.md missing term: %s", term))
				}
			}
			if !strings.Contains(content, "Governance Wiring") {
				issues = append(issues, "PLATFORM: area-platform-engineering.md missing Governance Wiring section")
			}
		}
	}

	// 3. Validate ovav-work command terms (shared across harnesses)
	workPath := filepath.Join(root, ".opencode", "commands", "ovav-work.md")
	if data, err := os.ReadFile(workPath); err == nil {
		content := string(data)
		for _, term := range ovavWorkTerms {
			if !strings.Contains(strings.ToLower(content), strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("WORK: ovav-work.md missing term: %s", term))
			}
		}
	}

	// 4. Validate ovav-context command terms (shared across harnesses)
	ctxPath := filepath.Join(root, ".opencode", "commands", "ovav-context.md")
	if data, err := os.ReadFile(ctxPath); err == nil {
		content := string(data)
		for _, term := range ovavContextTerms {
			if !strings.Contains(strings.ToLower(content), strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("CONTEXT: ovav-context.md missing term: %s", term))
			}
		}
	}

	// 5. Validate ovav-close command terms (shared across harnesses)
	closePath := filepath.Join(root, ".opencode", "commands", "ovav-close.md")
	if data, err := os.ReadFile(closePath); err == nil {
		content := string(data)
		for _, term := range ovavCloseTerms {
			if !strings.Contains(strings.ToLower(content), strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("CLOSE: ovav-close.md missing term: %s", term))
			}
		}
	}

	// 6. Scan ALL surfaces for stale patterns
	staleFound := 0
	for _, sf := range requiredSurfaces {
		fullPath := filepath.Join(root, sf.path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)
		for _, pattern := range stalePatterns {
			if strings.Contains(content, pattern) {
				issues = append(issues, fmt.Sprintf("STALE: %s contains deprecated pattern: %s", sf.path, pattern))
				staleFound++
			}
		}
	}

	// 7. Check for AGENTS.md integrity seal freshness
	agentsPath := filepath.Join(root, "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err == nil {
		content := string(data)
		if !strings.Contains(content, "OVAV_INTEGRITY_SEAL") {
			issues = append(issues, "WIRING: AGENTS.md missing OVAV_INTEGRITY_SEAL block")
		}
		if strings.Contains(content, "BUILD 16") || strings.Contains(content, "BUILD 17") {
			issues = append(issues, "STALE: AGENTS.md contains deprecated BUILD references")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(),
			Message:  fmt.Sprintf("FAIL runtime wiring — %d issue(s) [%s harness]", len(issues), harness),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: r.ID(), Name: r.Name(), Status: "pass", Weight: r.Weight(),
		Message:  fmt.Sprintf("PASS runtime wiring — %s harness: %d surfaces, %d stale patterns checked", harness, len(requiredSurfaces), staleFound),
		Duration: time.Since(start),
	}
}

var _ Validator = (*RuntimeWiring)(nil)
