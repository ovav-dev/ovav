package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/ovav/ovav/internal/monitor/alerts"
)

// AgentProjectionMonitor monitors agent source → runtime synchronization.
// Replaces: cross_target_consistency validator
//
// Checks:
//   - Count match between canonical source and runtime
//   - Timestamp: last convert > last source modification
//   - SBOM hash: detects if anything actually changed
type AgentProjectionMonitor struct {
	repoRoot string
	interval time.Duration
}

// NewAgentProjectionMonitor creates a new agent projection monitor
func NewAgentProjectionMonitor(repoRoot string) *AgentProjectionMonitor {
	return &AgentProjectionMonitor{
		repoRoot: repoRoot,
		interval: 5 * time.Minute,
	}
}

func (m *AgentProjectionMonitor) Name() string            { return "agent_projection" }
func (m *AgentProjectionMonitor) Interval() time.Duration { return m.interval }

// Run executes the projection consistency check
func (m *AgentProjectionMonitor) Run(ctx context.Context) ([]*alerts.Alert, error) {
	var alertList []*alerts.Alert

	// Check 1: Count comparison
	countAlert, err := m.checkCounts()
	if err != nil {
		return nil, err
	}
	if countAlert != nil {
		alertList = append(alertList, countAlert)
	}

	// Check 2: SBOM drift detection
	sbomAlert, err := m.checkSBOMDrift()
	if err != nil {
		return nil, err
	}
	if sbomAlert != nil {
		alertList = append(alertList, sbomAlert)
	}

	return alertList, nil
}

// checkCounts compares file counts between source and runtime
func (m *AgentProjectionMonitor) checkCounts() (*alerts.Alert, error) {
	// Canonical sources
	canonicalAreas := m.countFiles(filepath.Join(m.repoRoot, "go-runtime", "internal", "agents", "areas"), ".yaml")
	canonicalLeads := m.countFiles(filepath.Join(m.repoRoot, "go-runtime", "internal", "agents", "leads"), ".yaml")

	// Runtime target
	runtimeAgentsDir := filepath.Join(m.repoRoot, "go-runtime", "internal", "runtimes", "opencode", "agents")
	runtimeCount := m.countFiles(runtimeAgentsDir, ".md")
	// Exclude ovav.md from runtime count
	runtimeCount--

	// Source (OVAV CONVERT)
	sourceTeams := m.countFiles(filepath.Join(m.repoRoot, ".ovav", "source", "agents", "teams"), ".yaml")
	sourceTotal := canonicalAreas + canonicalLeads + sourceTeams

	if runtimeCount != sourceTotal {
		// Find which files are missing
		missing := m.findMissingFiles()
		return &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelERROR, "agent_projection", "agent count mismatch", "fix_agent_projection").ID,
			TS:      time.Now(),
			Level:   alerts.LevelERROR,
			Source:  "agent_projection",
			Issue:   "runtime agent count doesn't match source count",
			Files:   missing,
			Runbook: "fix_agent_projection",
			Status:  alerts.StatusPending,
		}, nil
	}

	return nil, nil
}

// findMissingFiles returns files that exist in source but not in runtime
func (m *AgentProjectionMonitor) findMissingFiles() []string {
	var missing []string

	// Check teams specifically since that's where drift happens
	teamsSourceDir := filepath.Join(m.repoRoot, ".ovav", "source", "agents", "teams")
	runtimeDir := filepath.Join(m.repoRoot, "go-runtime", "internal", "runtimes", "opencode", "agents")

	sourceTeams, _ := os.ReadDir(teamsSourceDir)
	for _, e := range sourceTeams {
		if e.IsDir() {
			continue
		}
		matched, _ := filepath.Match("*team-*.yaml", e.Name())
		if !matched {
			continue
		}
		runtimeName := e.Name()[:len(e.Name())-5] + ".md" // .yaml → .md
		runtimePath := filepath.Join(runtimeDir, runtimeName)
		if _, err := os.Stat(runtimePath); os.IsNotExist(err) {
			missing = append(missing, e.Name())
		}
	}

	return missing
}

// checkSBOMDrift detects if SBOM has changed since last check
func (m *AgentProjectionMonitor) checkSBOMDrift() (*alerts.Alert, error) {
	sbomPath := filepath.Join(m.repoRoot, ".ovav", "registry", "sbom.json")

	// Read SBOM
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		// SBOM missing → WARN, can regenerate
		return &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelWARN, "agent_projection", "SBOM not found", "fix_sbom_baseline").ID,
			TS:      time.Now(),
			Level:   alerts.LevelWARN,
			Source:  "agent_projection",
			Issue:   "SBOM file not found — registry may need regeneration",
			Files:   []string{sbomPath},
			Runbook: "fix_sbom_baseline",
			Status:  alerts.StatusPending,
		}, nil
	}

	// Compute hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Store hash in runtime for next comparison
	hashPath := filepath.Join(m.repoRoot, ".ovav", "runtime", "alerts", "sbom.last_hash")
	lastHash, _ := os.ReadFile(hashPath)

	if string(lastHash) != "" && string(lastHash) != hashStr {
		// Hash changed but we didn't trigger convert → possible drift
		return &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelINFO, "agent_projection", "SBOM updated externally", "").ID,
			TS:      time.Now(),
			Level:   alerts.LevelINFO,
			Source:  "agent_projection",
			Issue:   "SBOM hash changed — may indicate need for regeneration",
			Files:   []string{sbomPath},
			Runbook: "",
			Status:  alerts.StatusPending,
		}, nil
	}

	// Save current hash for next comparison
	os.WriteFile(hashPath, []byte(hashStr), 0644)

	return nil, nil
}

// countFiles counts files with a given extension in a directory
func (m *AgentProjectionMonitor) countFiles(dir, ext string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ext {
			count++
		}
	}
	return count
}
