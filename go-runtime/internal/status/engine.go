// Package status provides OVAV runtime status aggregation.
//
// Pure Go replacement for tools/ovav_status/status_engine.py (314 LOC Python).
// Generates .ovav/runtime/ovav_status.json consumed by the OpenCode JS plugin.
//
// Key design: lazy evaluation. No I/O until explicitly called.
// Zero Python dependencies. Zero subprocess calls.
package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ── Status Payload (mirrors Python schema for plugin compatibility) ───────────

// StatusPayload is the full OVAV status document written to ovav_status.json.
type StatusPayload struct {
	OVAV          OVAVStatus `json:"ovav"`
	GeneratedAt   string     `json:"generated_at"`
	ProjectRoot   string     `json:"project_root"`
	EngineVersion string     `json:"engine_version"`
}

// OVAVStatus holds all subsystem statuses.
type OVAVStatus struct {
	Overall   string          `json:"overall"`
	Icon      string          `json:"icon"`
	Governor  GovernorStatus  `json:"governor"`
	Memory    MemoryStatus    `json:"memory"`
	Integrity IntegrityStatus `json:"integrity"`
	Branch    BranchStatus    `json:"branch"`
	Capsule   CapsuleStatus   `json:"capsule"`
	Tokens    TokenStatus     `json:"tokens"`
}

// GovernorStatus indicates whether OVAV governor is active.
type GovernorStatus struct {
	Status        string  `json:"status"` // active | degraded | absent
	Label         string  `json:"label"`
	Icon          string  `json:"icon"`
	Active        bool    `json:"active"`
	SessionAgeMin float64 `json:"session_age_min,omitempty"`
}

// MemoryStatus indicates Memory Governor state.
type MemoryStatus struct {
	Status string `json:"status"` // active | inactive | degraded | unknown
	Label  string `json:"label"`
	Icon   string `json:"icon"`
}

// IntegrityStatus indicates Integrity Mesh check results.
type IntegrityStatus struct {
	Status      string   `json:"status"` // pass | fail | warn | unknown
	Label       string   `json:"label"`
	Icon        string   `json:"icon"`
	Score       float64  `json:"score,omitempty"`
	Passed      int      `json:"passed,omitempty"`
	Failed      int      `json:"failed,omitempty"`
	Errors      int      `json:"errors,omitempty"`
	Total       int      `json:"total,omitempty"`
	Intact      int      `json:"intact,omitempty"`
	Compromised []string `json:"compromised,omitempty"`
}

// BranchStatus indicates current git branch and protected status.
type BranchStatus struct {
	Branch    string `json:"branch"`
	Protected bool   `json:"protected"`
	Label     string `json:"label"`
	Icon      string `json:"icon"`
}

// CapsuleStatus — always inactive since v2.0 removal.
type CapsuleStatus struct {
	Active     bool   `json:"active"`
	CapsuleID  string `json:"capsule_id"`
	BudgetTier string `json:"budget_tier"`
	Label      string `json:"label"`
	Icon       string `json:"icon"`
	Source     string `json:"source"`
}

// TokenStatus holds cumulative token metrics.
type TokenStatus struct {
	TotalAll        int     `json:"total_all"`
	TotalInput      int     `json:"total_input"`
	TotalOutput     int     `json:"total_output"`
	Measurements    int     `json:"measurements"`
	CurrentUsagePct float64 `json:"current_usage_pct"`
	BPEVerified     bool    `json:"bpe_verified"`
}

// ── Protected branches (same as Python) ───────────────────────────────────────

var protectedBranches = map[string]bool{
	"main":        true,
	"master":      true,
	"develop":     true,
	"development": true,
	"prod":        true,
	"production":  true,
	"staging":     true,
}

// ── Engine ────────────────────────────────────────────────────────────────────

// Engine aggregates all OVAV runtime status.
type Engine struct {
	projectRoot string
	runtimeDir  string
}

// New creates a new StatusEngine for the given project root.
func New(projectRoot string) *Engine {
	return &Engine{
		projectRoot: projectRoot,
		runtimeDir:  filepath.Join(projectRoot, ".ovav", "runtime"),
	}
}

// Aggregate gathers all status checks into a single payload.
func (e *Engine) Aggregate() *StatusPayload {
	governor := e.checkGovernor()
	memory := e.checkMemory()
	integrity := e.checkIntegrity()
	branch := e.checkBranch()
	tokens := e.checkTokens()
	capsule := CapsuleStatus{
		Active:     false,
		CapsuleID:  "—",
		BudgetTier: "—",
		Label:      "Capsule: removed (v2.0)",
		Icon:       "📦",
		Source:     "removed_v2.0",
	}

	// Compute overall state
	overall, icon := e.computeOverall(governor, integrity)

	return &StatusPayload{
		OVAV: OVAVStatus{
			Overall:   overall,
			Icon:      icon,
			Governor:  governor,
			Memory:    memory,
			Integrity: integrity,
			Branch:    branch,
			Capsule:   capsule,
			Tokens:    tokens,
		},
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		ProjectRoot:   e.projectRoot,
		EngineVersion: "2.0.0-go",
	}
}

// ── Governor check ────────────────────────────────────────────────────────────

func (e *Engine) checkGovernor() GovernorStatus {
	ovavDir := filepath.Join(e.projectRoot, ".ovav")
	policyFile := filepath.Join(ovavDir, "policy", "permission_authority.json")
	serviceAreasDir := filepath.Join(ovavDir, "service_areas")
	sessionMarker := filepath.Join(e.runtimeDir, ".session_marker")

	if !dirExists(ovavDir) {
		return GovernorStatus{
			Status: "absent", Label: "No OVAV", Icon: "⚫", Active: false,
		}
	}

	hasPolicy := fileExists(policyFile)
	hasServiceAreas := dirExists(serviceAreasDir)

	if hasPolicy && hasServiceAreas {
		// Check session marker freshness
		if fi, err := os.Stat(sessionMarker); err == nil {
			ageMin := time.Since(fi.ModTime()).Minutes()
			if ageMin < 120 { // 2 hours
				return GovernorStatus{
					Status:        "active",
					Label:         "OVAV Governor activo",
					Icon:          "🟢",
					Active:        true,
					SessionAgeMin: round1(ageMin),
				}
			}
		}
		return GovernorStatus{
			Status: "active", Label: "OVAV configurado", Icon: "🟢", Active: true,
		}
	}

	return GovernorStatus{
		Status: "degraded", Label: "OVAV parcial", Icon: "🟡", Active: true,
	}
}

// ── Memory check ──────────────────────────────────────────────────────────────

func (e *Engine) checkMemory() MemoryStatus {
	// Check for memory governor marker files
	memoryActive := filepath.Join(e.runtimeDir, "memory_active")
	govActive := filepath.Join(e.runtimeDir, "memory_governor_active")

	if fileExists(memoryActive) || fileExists(govActive) {
		return MemoryStatus{Status: "active", Label: "Memory: active", Icon: "🟢"}
	}

	// Check if memory dir exists at all
	memoryDir := filepath.Join(e.projectRoot, ".ovav", "memory")
	if dirExists(memoryDir) {
		return MemoryStatus{Status: "inactive", Label: "Memory: inactive", Icon: "🟡"}
	}

	return MemoryStatus{Status: "absent", Label: "Memory: —", Icon: "⚫"}
}

// ── Integrity check ───────────────────────────────────────────────────────────

func (e *Engine) checkIntegrity() IntegrityStatus {
	// Read integrity status from marker file if it exists
	integrityFile := filepath.Join(e.runtimeDir, "integrity_status.json")
	data, err := os.ReadFile(integrityFile)
	if err != nil {
		// Try running a quick check on key files
		return e.quickIntegrityCheck()
	}

	var status IntegrityStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return IntegrityStatus{Status: "unknown", Label: "Integrity: —", Icon: "⚫"}
	}
	return status
}

// quickIntegrityCheck verifies critical governance files exist.
func (e *Engine) quickIntegrityCheck() IntegrityStatus {
	criticalFiles := []string{
		".ovav/policy/permission_authority.json",
		".ovav/laws/area_boundary_enforcement.yaml",
		".ovav/plan/caps.yaml",
		"AGENTS.md",
		"VERSION",
	}

	total := len(criticalFiles)
	intact := 0
	var compromised []string

	for _, f := range criticalFiles {
		if fileExists(filepath.Join(e.projectRoot, f)) {
			intact++
		} else {
			compromised = append(compromised, f)
		}
	}

	if intact == total {
		return IntegrityStatus{
			Status: "pass", Label: fmt.Sprintf("Integrity: %d/%d OK", intact, total),
			Icon: "🟢", Total: total, Intact: intact,
		}
	}
	if len(compromised) > 0 {
		return IntegrityStatus{
			Status: "fail", Label: fmt.Sprintf("Integrity: %d/%d", intact, total),
			Icon: "🔴", Total: total, Intact: intact, Compromised: compromised,
		}
	}
	return IntegrityStatus{Status: "warn", Label: "Integrity: checking", Icon: "🟡"}
}

// ── Branch check ──────────────────────────────────────────────────────────────

func (e *Engine) checkBranch() BranchStatus {
	branch := e.gitBranch()
	isProtected := protectedBranches[branch]

	label := branch
	icon := "⎇"
	if isProtected {
		label = "🔒 " + branch
		icon = "🔒"
	}

	return BranchStatus{
		Branch:    branch,
		Protected: isProtected,
		Label:     label,
		Icon:      icon,
	}
}

// gitBranch returns the current git branch using a quick filesystem read.
// Avoids shelling out to git when possible.
func (e *Engine) gitBranch() string {
	// Read .git/HEAD to get current branch
	headFile := filepath.Join(e.projectRoot, ".git", "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return "unknown"
	}

	content := strings.TrimSpace(string(data))
	// Format: "ref: refs/heads/<branch>"
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}

	// Detached HEAD — return short SHA
	if len(content) >= 7 {
		return content[:7]
	}
	return "detached"
}

// ── Token check ───────────────────────────────────────────────────────────────

func (e *Engine) checkTokens() TokenStatus {
	statsFile := filepath.Join(e.runtimeDir, "session_token_stats.json")
	data, err := os.ReadFile(statsFile)
	if err != nil {
		return TokenStatus{}
	}

	var ts TokenStatus
	if err := json.Unmarshal(data, &ts); err != nil {
		return TokenStatus{}
	}
	return ts
}

// ── Overall computation ───────────────────────────────────────────────────────

func (e *Engine) computeOverall(g GovernorStatus, i IntegrityStatus) (string, string) {
	switch {
	case g.Status == "absent":
		return "absent", "⚫"
	case i.Status == "fail":
		return "degraded", "🔴"
	case g.Status == "degraded":
		return "degraded", "🟡"
	default:
		return "active", "🟢"
	}
}

// ── Write markers ─────────────────────────────────────────────────────────────

// WriteMarkers writes all status marker files for the JS plugin to consume.
func (e *Engine) WriteMarkers() error {
	// Ensure runtime dir exists
	if err := os.MkdirAll(e.runtimeDir, 0755); err != nil {
		return fmt.Errorf("creating runtime dir: %w", err)
	}

	payload := e.Aggregate()

	// Write main status JSON
	statusPath := filepath.Join(e.runtimeDir, "ovav_status.json")
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling status: %w", err)
	}
	if err := os.WriteFile(statusPath, data, 0644); err != nil {
		return fmt.Errorf("writing status json: %w", err)
	}

	// Write governor active marker
	if payload.OVAV.Governor.Active {
		os.WriteFile(filepath.Join(e.runtimeDir, "governor_active"), []byte("1"), 0644)
	} else {
		os.Remove(filepath.Join(e.runtimeDir, "governor_active"))
	}

	// Write integrity status
	integrityPath := filepath.Join(e.runtimeDir, "integrity_status.json")
	integrityData, _ := json.MarshalIndent(payload.OVAV.Integrity, "", "  ")
	os.WriteFile(integrityPath, integrityData, 0644)

	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func round1(f float64) float64 {
	return float64(int(f*10)) / 10
}

// ── System info (for diagnostics) ─────────────────────────────────────────────

// SystemInfo returns basic system information.
func SystemInfo() map[string]string {
	return map[string]string{
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}
}
