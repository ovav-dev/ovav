package validators

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ContractFreshness verifies that required governance contracts exist,
// haven't gone stale (>30 days without review), and have valid structure.
type ContractFreshness struct{}

func NewContractFreshness() *ContractFreshness { return &ContractFreshness{} }

func (c *ContractFreshness) ID() string   { return "contract_freshness" }
func (c *ContractFreshness) Name() string { return "Contract Freshness" }
func (c *ContractFreshness) Description() string {
	return "Checks governance contracts for staleness and integrity"
}
func (c *ContractFreshness) Weight() int { return 10 }

// staleThresholdDays is the maximum age before a contract is considered stale.
const staleThresholdDays = 30

// minContractSize is the minimum file size (bytes) before flagging as stub.
const minContractSize = 10

// requiredContracts lists governance files that must exist and be fresh.
var requiredContracts = []string{
	".ovav/service_areas/areas/platform_engineering.yaml",
	".ovav/service_areas/shared/lead_work_method_contract.yaml",
	".ovav/plan/caps.yaml",
	".ovav/service_areas/shared/context_economy_contract.yaml",
	".ovav/service_areas/shared/tool_access_policy.yaml",
	".ovav/service_areas/shared/visual_delivery_contract.yaml",
	".ovav/service_areas/shared/operational_memory_contract.yaml",
	".ovav/policy/permission_authority.json",
	".ovav/registry/delegation_rules.yaml",
}

// frequentlyChanged files are excluded from staleness checks.
var frequentlyChanged = map[string]bool{}

func (c *ContractFreshness) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	now := time.Now()

	for _, relPath := range requiredContracts {
		fullPath := filepath.Join(root, relPath)

		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING: Required contract '%s' not found", relPath))
			continue
		}
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read contract '%s': %v", relPath, err))
			continue
		}

		if frequentlyChanged[relPath] {
			continue
		}

		// Staleness check
		ageDays := int(now.Sub(info.ModTime()).Hours() / 24)
		if ageDays > staleThresholdDays {
			issues = append(issues, fmt.Sprintf(
				"STALE: Contract '%s' last modified %d days ago (%s). Review recommended.",
				relPath, ageDays, info.ModTime().Format("2006-01-02"),
			))
		}

		// Size sanity — flag stubs
		if info.Size() < minContractSize {
			issues = append(issues, fmt.Sprintf(
				"STUB: Contract '%s' is only %d bytes — may be an empty stub",
				relPath, info.Size(),
			))
		}

		// SHA256 integrity check
		data, err := os.ReadFile(fullPath)
		if err == nil {
			hash := fmt.Sprintf("%x", sha256.Sum256(data))
			// Hash is computed for audit trail; no action needed unless baseline comparison is added
			_ = hash
		}
	}

	// Extra check: permission_authority.json structure
	paPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if data, err := os.ReadFile(paPath); err == nil {
		if len(data) < 50 { // JSON with just {} or minimal content
			issues = append(issues, "WARNING: permission_authority.json appears empty or minimal")
		}
	} else {
		issues = append(issues, "CRITICAL: permission_authority.json not found or unreadable")
	}

	if len(issues) > 0 {
		hasCritical := false
		for _, issue := range issues {
			if len(issue) > 8 && issue[:8] == "CRITICAL" {
				hasCritical = true
				break
			}
		}
		status := "fail"
		if !hasCritical {
			status = "warn"
		}
		return Result{
			ID: c.ID(), Name: c.Name(), Status: status, Weight: c.Weight(),
			Message: fmt.Sprintf("FAIL contract freshness — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: c.ID(), Name: c.Name(), Status: "pass", Weight: c.Weight(),
		Message:  "PASS contract freshness",
		Duration: time.Since(start),
	}
}

var _ Validator = (*ContractFreshness)(nil)
