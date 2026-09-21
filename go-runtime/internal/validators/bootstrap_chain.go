package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/security"
)

// BootstrapChain validates the integrity seal chain from AGENTS.md.
// Replaces: check_bootstrap_chain.py
type BootstrapChain struct{}

func NewBootstrapChain() *BootstrapChain { return &BootstrapChain{} }

func (b *BootstrapChain) ID() string   { return "bootstrap_chain" }
func (b *BootstrapChain) Name() string { return "Bootstrap Chain Validator" }
func (b *BootstrapChain) Description() string {
	return "Validates the integrity seal chain from AGENTS.md through OVAV_INTEGRITY_SEAL"
}
func (b *BootstrapChain) Weight() int { return 9 }

// Required integrity markers that must exist in the bootstrap chain.
type sealCheck struct {
	file    string
	markers []string
}

var bootstrapSeals = []sealCheck{
	{
		file:    "AGENTS.md",
		markers: []string{"OVAV_INTEGRITY_SEAL", "OVAV is a sealed governor system", "OVAV GOVERNOR ALERT"},
	},
	{
		file:    ".ovav/plan/caps.yaml",
		markers: []string{"version:", "updated_at:", "plan_version:"},
	},
	{
		file:    ".ovav/laws/ovav_laws.yaml",
		markers: []string{"LAW-001", "Non-Invasion", "Area Boundary"},
	},
	{
		file:    ".ovav/policy/permission_authority.json",
		markers: []string{"permission_authority", "Thavren", "Platform Engineering"},
	},
}

func (b *BootstrapChain) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	var chainLinks []string

	// Verify integrity seal is present and unmodified
	agentsMD := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(agentsMD)
	if err != nil {
		return Result{
			ID: b.ID(), Name: b.Name(), Status: "fail", Weight: b.Weight(),
			Message:  "FAIL — AGENTS.md not found (bootstrap root missing)",
			Issues:   []string{"MISSING: AGENTS.md (bootstrap root)"},
			Duration: time.Since(start),
		}
	}
	content := string(data)
	if !security.BootstrapTrustAnchorsConfigured() {
		issues = append(issues, "INTENTIONALLY_GATED/PARTIAL: immutable bootstrap trust anchors are not configured in build metadata")
	}

	// Check integrity seal block
	const sealStart = "OVAV_INTEGRITY_SEAL"
	const sealEnd = "/OVAV_INTEGRITY_SEAL"
	if !strings.Contains(content, sealStart) {
		issues = append(issues, "MISSING: OVAV_INTEGRITY_SEAL block in AGENTS.md")
	}
	if strings.Contains(content, "DO NOT MODIFY THIS BLOCK") && !strings.Contains(content, sealStart) {
		issues = append(issues, "TAMPERED: integrity seal comment present but seal missing")
	}

	// Check each bootstrap seal file
	for _, seal := range bootstrapSeals {
		fullPath := filepath.Join(root, seal.file)
		fdata, ferr := os.ReadFile(fullPath)
		if ferr != nil {
			issues = append(issues, fmt.Sprintf("MISSING: %s (bootstrap chain broken)", seal.file))
			continue
		}
		fcontent := string(fdata)
		for _, marker := range seal.markers {
			if !strings.Contains(fcontent, marker) {
				issues = append(issues, fmt.Sprintf("BROKEN_LINK: %s missing marker '%s'", seal.file, marker))
			}
		}
		chainLinks = append(chainLinks, seal.file)
	}

	if len(issues) > 0 {
		if len(issues) == 1 && strings.HasPrefix(issues[0], "INTENTIONALLY_GATED/PARTIAL:") {
			return Result{
				ID: b.ID(), Name: b.Name(), Status: "warn", Weight: b.Weight(),
				Message:  "INTENTIONALLY_GATED/PARTIAL — bootstrap files are intact but immutable build trust anchors are not configured",
				Issues:   issues,
				Duration: time.Since(start),
			}
		}
		return Result{
			ID: b.ID(), Name: b.Name(), Status: "fail", Weight: b.Weight(),
			Message:  fmt.Sprintf("FAIL — bootstrap chain broken: %d issue(s) in %d links checked", len(issues), len(chainLinks)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: b.ID(), Name: b.Name(), Status: "pass", Weight: b.Weight(),
		Message:  fmt.Sprintf("PASS — bootstrap chain intact (%d links verified)", len(chainLinks)),
		Duration: time.Since(start),
	}
}

var _ Validator = (*BootstrapChain)(nil)
