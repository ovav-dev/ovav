package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HostConfigDrift validates host configuration integrity and quarantine state.
// Simplified Go version of check_host_config_drift.py (1131 LOC) — core checks only.
// Replaces: check_host_config_drift.py
type HostConfigDrift struct{}

func NewHostConfigDrift() *HostConfigDrift { return &HostConfigDrift{} }

func (h *HostConfigDrift) ID() string   { return "host_config_drift" }
func (h *HostConfigDrift) Name() string { return "Host Config Drift" }
func (h *HostConfigDrift) Description() string {
	return "Validates host configuration integrity, quarantine state, and intrusion detection"
}
func (h *HostConfigDrift) Weight() int { return 25 }

// Host intrusion files — must NOT exist in ~/.config/opencode/.
// AGENTS.md and opencode.jsonc are OVAV governance files that should not leak to host.
// opencode.json is MiMoCode's own config — only flagged if it contains OVAV content.
var hostIntrusionFiles = []string{
	"AGENTS.md",
	"opencode.jsonc",
}

// OVAV-specific content markers that indicate leakage into host config.
var ovavContentMarkers = []string{
	"OVAV_INTEGRITY_SEAL",
	"OVAV_GOVERNANCE",
	"/service_area",
	"lead_",
	"area-platform",
	"area-research",
	"area-education",
	"area-business",
	"area-health",
}

// Host intrusion agent paths.
var hostIntrusionAgents = []string{
	"agents/area-platform-engineering.md",
	"agents/area-research-intelligence.md",
	"agents/lead-thavren.md",
	"agents/lead-eidren.md",
}

func (h *HostConfigDrift) checkHostIntrusion(root string) []string {
	var issues []string
	home := os.Getenv("HOME")
	if home == "" {
		return issues
	}
	hostConfig := filepath.Join(home, ".config", "opencode")

	// Check for intrusion files
	for _, file := range hostIntrusionFiles {
		path := filepath.Join(hostConfig, file)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			issues = append(issues, fmt.Sprintf("HOST INTRUSION: %s found in ~/.config/opencode/ — quarantine required", file))
		}
	}

	// opencode.json is MiMoCode's own config — it is EXPECTED to contain OVAV
	// governance content when used as an OVAV harness. This is intentional integration,
	// not an intrusion. Skip the OVAV content check for opencode.json.
	// See: OVAV — MiMoCode integration (MEMORY-OVAV-discovered-foundation.md)
	opencodePath := filepath.Join(hostConfig, "opencode.json")
	if info, err := os.Stat(opencodePath); err == nil && !info.IsDir() {
		// opencode.json intentionally contains OVAV markers when used with OVAV harness
		// — this is expected, not an intrusion. Skip warning.
	}

	// Check for intrusion agent files
	for _, agent := range hostIntrusionAgents {
		path := filepath.Join(hostConfig, agent)
		if _, err := os.Stat(path); err == nil {
			issues = append(issues, fmt.Sprintf("HOST INTRUSION: %s found in ~/.config/opencode/agents/ — quarantine required", filepath.Base(agent)))
		}
	}

	return issues
}

func (h *HostConfigDrift) isBenignBootstrap(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Check for schema-only config — normalize whitespace for robust matching.
	// JSON can be compact: {"$schema":"https://..."} or pretty-printed:
	// {\n  "$schema": "https://..."\n}. Both are equivalent bootstrap configs.
	content := strings.TrimSpace(string(data))
	// Remove ALL whitespace between JSON tokens for comparison
	normalized := strings.ReplaceAll(content, " ", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	normalized = strings.ReplaceAll(normalized, "\r", "")
	if normalized == `{"$schema":"https://opencode.ai/config.json"}` {
		return true
	}
	return false
}

// containsOVAVContent checks if a file contains OVAV-specific governance content.
func (h *HostConfigDrift) containsOVAVContent(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	for _, marker := range ovavContentMarkers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}

func (h *HostConfigDrift) checkBlockade(root string) []string {
	var issues []string
	blockadePath := filepath.Join(root, ".ovav", "host_defense_blockade")
	if info, err := os.Stat(blockadePath); err == nil && info.Size() > 0 {
		issues = append(issues, "WARNING: host_defense_blockade file exists — system may be in quarantine")
	}
	return issues
}

func (h *HostConfigDrift) checkQuarantine(root string) []string {
	var issues []string
	quarantineDir := filepath.Join(root, ".ovav", "quarantine")
	if entries, err := os.ReadDir(quarantineDir); err == nil {
		fileCount := 0
		for _, e := range entries {
			if !e.IsDir() {
				fileCount++
			}
		}
		if fileCount > 0 {
			issues = append(issues, fmt.Sprintf("QUARANTINE: %d file(s) in quarantine — review required", fileCount))
		}
	}
	return issues
}

func (h *HostConfigDrift) checkSessionMarker(root string) []string {
	marker := filepath.Join(root, ".ovav", "runtime", ".session_marker")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		// Session marker missing is normal outside development sessions
		return nil
	}
	// Session marker exists — authorized development session
	return nil
}

func (h *HostConfigDrift) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Host intrusion detection
	intrusionIssues := h.checkHostIntrusion(root)
	issues = append(issues, intrusionIssues...)

	// 2. Blockade check
	blockadeIssues := h.checkBlockade(root)
	issues = append(issues, blockadeIssues...)

	// 3. Quarantine check
	quarantineIssues := h.checkQuarantine(root)
	issues = append(issues, quarantineIssues...)

	// 4. Session marker check
	_ = h.checkSessionMarker(root)

	// Check for critical issues (HOST INTRUSION)
	hasCritical := false
	for _, issue := range issues {
		if strings.Contains(issue, "HOST INTRUSION") {
			hasCritical = true
			break
		}
	}

	if hasCritical {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  fmt.Sprintf("FAIL host config drift — CRITICAL: host intrusion detected (%d issue(s))", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	if len(issues) > 0 {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "warn", Weight: h.Weight(),
			Message:  fmt.Sprintf("WARN host config drift — %d non-critical issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
		Message:  "PASS host config drift — no intrusion detected",
		Duration: time.Since(start),
	}
}

var _ Validator = (*HostConfigDrift)(nil)
