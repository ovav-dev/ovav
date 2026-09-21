package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CrossTargetConsistency validates consistency between canonical agents
// (.ovav/service_areas/) and generated runtime agents (go-runtime/internal/runtimes/opencode/agents/).
// Replaces: tools/validators/check_cross_target_consistency.py
type CrossTargetConsistency struct{}

func NewCrossTargetConsistency() *CrossTargetConsistency { return &CrossTargetConsistency{} }

func (v *CrossTargetConsistency) ID() string   { return "cross_target_consistency" }
func (v *CrossTargetConsistency) Name() string { return "Cross-Target Consistency" }
func (v *CrossTargetConsistency) Description() string {
	return "Validates consistency between canonical YAML agents and generated runtime agents"
}
func (v *CrossTargetConsistency) Weight() int { return 6 }

// configFiles are key files that should be consistent across surfaces.
var configFiles = []string{
	"opencode.json",
	"opencode.jsonc",
	"AGENTS.md",
}

func (v *CrossTargetConsistency) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// ── 1. Agent count consistency ───────────────────────────────────────────
	canonicalAreasDir := filepath.Join(root, "go-runtime", "internal", "agents", "areas")
	canonicalLeadsDir := filepath.Join(root, "go-runtime", "internal", "agents", "leads")
	canonicalTeamsDir := filepath.Join(root, "go-runtime", "internal", "agents", "teams")
	runtimeAgentsDir := filepath.Join(root, ".ovav", "service_areas")

	canonicalAreas, errAreas := countYAMLFiles(canonicalAreasDir)
	canonicalLeads, errLeads := countYAMLFiles(canonicalLeadsDir)
	canonicalTeams, errTeams := countYAMLFiles(canonicalTeamsDir)
	canonicalTotal := canonicalAreas + canonicalLeads + canonicalTeams

	runtimeEntries, errRuntime := os.ReadDir(runtimeAgentsDir)

	if errAreas != nil {
		issues = append(issues, fmt.Sprintf("ERROR: Cannot read go-runtime/internal/agents/areas/: %v", errAreas))
	}
	if errLeads != nil {
		issues = append(issues, fmt.Sprintf("ERROR: Cannot read go-runtime/internal/agents/leads/: %v", errLeads))
	}
	if errTeams != nil {
		issues = append(issues, fmt.Sprintf("ERROR: Cannot read go-runtime/internal/agents/teams/: %v", errTeams))
	}
	if errRuntime != nil {
		issues = append(issues, fmt.Sprintf("ERROR: Cannot read .ovav/service_areas/: %v", errRuntime))
	}

	if errAreas == nil && errLeads == nil && errTeams == nil && errRuntime == nil {
		runtimeCount := countMDFiles(runtimeEntries)
		// Exclude ovav.md — central governor, not generated from canonical YAML
		for _, e := range runtimeEntries {
			if e.Name() == "ovav.md" {
				runtimeCount--
				break
			}
		}
		if canonicalTotal != runtimeCount {
			issues = append(issues, fmt.Sprintf("DRIFT: Agent count mismatch — canonical(YAML)=%d runtime(MD)=%d", canonicalTotal, runtimeCount))
		}
	}

	// ── 2. Key config file consistency ───────────────────────────────────────
	for _, cf := range configFiles {
		rootPath := filepath.Join(root, cf)
		dotPath := filepath.Join(root, ".opencode", cf)
		clientsPath := filepath.Join(root, "clients", "opencode", cf)

		// Check if root config exists
		rootData, errRoot := os.ReadFile(rootPath)
		if errRoot != nil {
			continue // not applicable for all configs
		}

		// Compare .opencode/ version if it exists
		if dotData, err := os.ReadFile(dotPath); err == nil {
			if string(rootData) != string(dotData) {
				issues = append(issues, fmt.Sprintf("DRIFT: %s differs from .opencode/%s", cf, cf))
			}
		}

		// Compare clients/opencode/ version if it exists
		if clientsData, err := os.ReadFile(clientsPath); err == nil {
			if string(rootData) != string(clientsData) {
				issues = append(issues, fmt.Sprintf("DRIFT: %s differs from clients/opencode/%s", cf, cf))
			}
		}
	}

	// ── 3. Runtime agents directory accessibility ────────────────────────────
	if entries, err := os.ReadDir(runtimeAgentsDir); err == nil {
		if countMDFiles(entries) == 0 {
			issues = append(issues, "INFO: .ovav/service_areas/ exists but has no area directories — canonical agents not yet deployed")
		}
	} else {
		issues = append(issues, "INFO: .ovav/service_areas/ not found — canonical agents not yet deployed")
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL cross-target consistency — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
		Message:  "PASS cross-target consistency — all surfaces aligned",
		Duration: time.Since(start),
	}
}

// countMDFiles counts .md files in a directory listing.
func countMDFiles(entries []os.DirEntry) int {
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			count++
		}
	}
	return count
}

// countYAMLFiles counts .yaml files in a directory, returning 0 if the dir doesn't exist.
func countYAMLFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".yaml") {
			count++
		}
	}
	return count, nil
}

var _ Validator = (*CrossTargetConsistency)(nil)
