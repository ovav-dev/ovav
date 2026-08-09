package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/truststore"
)

// GateSelfProtection validates host defense gate integrity.
// Uses git reflog grace period: gate hash changes within 5 minutes of a git
// operation are treated as legitimate (developer workflow), not compromise.
type GateSelfProtection struct{}

func NewGateSelfProtection() *GateSelfProtection { return &GateSelfProtection{} }

func (g *GateSelfProtection) ID() string   { return "gate_self_protection" }
func (g *GateSelfProtection) Name() string { return "Gate Self-Protection" }
func (g *GateSelfProtection) Description() string {
	return "Validates host defense gate integrity via hash verification"
}
func (g *GateSelfProtection) Weight() int { return 18 }

const gateFile = "go-runtime/internal/validators/host_config_drift.go"

func (g *GateSelfProtection) fileSHA256(path string) string {
	return truststore.GateFileSHA256(path)
}

func (g *GateSelfProtection) isAuthorizedSession(root string) bool {
	marker := filepath.Join(root, ".ovav", "runtime", ".session_marker")
	data, err := os.ReadFile(marker)
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(data))) > 0
}

func (g *GateSelfProtection) checkBlockade(root string) []string {
	var issues []string
	blockadePath := filepath.Join(root, ".ovav", "host_defense_blockade")
	if info, err := os.Stat(blockadePath); err == nil && info.Size() > 0 {
		data, err := os.ReadFile(blockadePath)
		if err == nil {
			var rec struct {
				Blockade string `json:"blockade"`
				Reason   string `json:"reason"`
			}
			if json.Unmarshal(data, &rec) == nil && rec.Blockade == "active" {
				issues = append(issues, fmt.Sprintf("BLOCKADE ACTIVE: %s", rec.Reason))
			}
		}
	}
	return issues
}

// RecordGitOp is called by OWS after git operations to enable the grace period.
func (g *GateSelfProtection) RecordGitOp(repoRoot string) error {
	return truststore.RecordGitOp(repoRoot)
}

func (g *GateSelfProtection) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	gateFullPath := filepath.Join(root, gateFile)

	// 1. Gate file must exist
	if _, err := os.Stat(gateFullPath); os.IsNotExist(err) {
		issues = append(issues, "GATE SELF-PROTECTION: host_config_drift.go MISSING")
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "fail", Weight: g.Weight(),
			Message:  "FAIL gate self-protection — gate file missing",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	// 2. Check blockade
	blockadeIssues := g.checkBlockade(root)
	issues = append(issues, blockadeIssues...)

	// 3. Compute current gate hash
	currentHash := g.fileSHA256(gateFullPath)
	state := truststore.ReadGateState(root)
	storedHash := state.GateSHA256

	// 4. Authorized session — bypass hash check
	if g.isAuthorizedSession(root) {
		if currentHash != storedHash && storedHash != "" {
			state.GateSHA256 = currentHash
			truststore.WriteGateState(root, state)
		}
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "pass", Weight: g.Weight(),
			Message:  fmt.Sprintf("PASS gate self-protection — authorized session, hash updated (%s...)", currentHash[:8]),
			Duration: time.Since(start),
		}
	}

	// 5. First run — initialize
	if storedHash == "" {
		state.GateSHA256 = currentHash
		truststore.WriteGateState(root, state)
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "pass", Weight: g.Weight(),
			Message:  "PASS gate self-protection — gate hash initialized",
			Duration: time.Since(start),
		}
	}

	// 6. Hash matches
	if currentHash == storedHash {
		return Result{
			ID: g.ID(), Name: g.Name(), Status: "pass", Weight: g.Weight(),
			Message:  fmt.Sprintf("PASS gate self-protection — gate integrity verified (%s...)", currentHash[:8]),
			Duration: time.Since(start),
		}
	}

	// 7. Hash mismatch — check grace period
	if state.LastGitOpTime > 0 {
		ok, _ := truststore.GracePeriodOk(root)
		if ok {
			elapsed := time.Since(time.Unix(state.LastGitOpTime, 0))
			state.GateSHA256 = currentHash
			truststore.WriteGateState(root, state)
			return Result{
				ID: g.ID(), Name: g.Name(), Status: "pass", Weight: g.Weight(),
				Message: fmt.Sprintf(
					"PASS gate self-protection — hash updated post-git-op (%.0f min grace, reflog: %s)",
					elapsed.Minutes(), state.LastGitOpReflog,
				),
				Duration: time.Since(start),
			}
		}
		// Outside grace period
		elapsed := time.Since(time.Unix(state.LastGitOpTime, 0))
		issues = append(issues, fmt.Sprintf(
			"GATE HASH MISMATCH: current %s... != stored %s... — possible compromise",
			currentHash[:16], storedHash[:16],
		))
		issues = append(issues, fmt.Sprintf(
			"Last git op: %s (%.0f min ago) — outside 5-min grace period",
			state.LastGitOpReflog, elapsed.Minutes(),
		))
	} else {
		issues = append(issues, fmt.Sprintf(
			"GATE HASH MISMATCH: current %s... != stored %s... — possible compromise (no git-op record)",
			currentHash[:16], storedHash[:16],
		))
	}

	return Result{
		ID: g.ID(), Name: g.Name(), Status: "fail", Weight: g.Weight(),
		Message:  fmt.Sprintf("FAIL gate self-protection — %d issue(s)", len(issues)),
		Issues:   issues,
		Duration: time.Since(start),
	}
}

var _ Validator = (*GateSelfProtection)(nil)
