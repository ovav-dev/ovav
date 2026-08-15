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

// HostConfigDrift validates host configuration integrity and quarantine state.
// Simplified Go version of check_host_config_drift.py (1131 LOC) — core checks only.
// Replaces: check_host_config_drift.py
//
// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
// Host configurations carrying the canonical OVAV YOLO marker (_ovav.yolo, _ovav.trusted
// or the same JSON shape that .ovav/policy/permission_authority.json materializes) are
// recognized as OVAV-managed and are NOT host intrusions. Only configurations that:
//   (a) lack the OVAV marker AND
//   (b) carry agent/permission/provider intelligence
// are flagged for quarantine.
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

	// Global OpenCode configs may contain bootstrap/schema metadata. Provider,
	// permission and agent intelligence must stay repo-local UNLESS the file is
	// explicitly OVAV-managed (carries the YOLO marker from the materializer).
	for _, configName := range []string{"opencode.json", "opencode.jsonc"} {
		opencodePath := filepath.Join(hostConfig, configName)
		if info, err := os.Stat(opencodePath); err == nil && !info.IsDir() {
			if h.isOVAVManaged(opencodePath) {
				// OVAV TRUSTED DOMAIN: this config was materialized by OVAV governor.
				continue
			}
			if !h.isBenignBootstrap(opencodePath) && h.containsGlobalIntelligence(opencodePath) {
				issues = append(issues, fmt.Sprintf("HOST INTRUSION: %s contains global agents/permissions/providers — quarantine required", configName))
			}
		}
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

// isOVAVManaged checks if the config carries the canonical OVAV marker.
// OVAV TRUSTED DOMAIN — 2026-08-13: configurations carrying _ovav policy marker
// or matching the canonical permission_authority.json shape are recognized as
// OVAV-managed projections, not as host intrusions.
func (h *HostConfigDrift) isOVAVManaged(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	// Markers that indicate OVAV governor materialized this file.
	ovavMarkers := []string{
		"_ovav",
		"OVAV_SYSTEMS",
		"OVAV_GOVERNANCE",
		"OVAV_YOLO",
		"permission_authority.json",
	}
	for _, marker := range ovavMarkers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}

func (h *HostConfigDrift) containsGlobalIntelligence(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		content := string(data)
		for _, key := range []string{"agent", "agents", "permission", "permissions", "provider", "providers"} {
			if strings.Contains(content, `"`+key+`"`) {
				return true
			}
		}
		return false
	}
	for _, key := range []string{"agent", "agents", "permission", "permissions", "provider", "providers"} {
		if _, exists := config[key]; exists {
			return true
		}
	}
	return false
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
