package monitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/monitor/alerts"
	"github.com/ovav/ovav/internal/sbom"
)

// RunbookFixGeneratedDrift auto-fixes generated file drift by re-running project sync
// Runbook: "fix_generated_drift"
func RunbookFixGeneratedDrift(ctx context.Context, a *alerts.Alert) error {
	root := filepath.Dir(filepath.Dir(filepath.Join(".ovav", "runtime", "alerts"))) // walk up to repo root

	// Run: ovav project sync
	cmd := exec.Command("go", "run", "-C", filepath.Join(root, "go-runtime"), "./cmd/ovav/", "project", "sync")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("project sync failed: %w, output: %s", err, string(out))
	}

	// Verify it worked
	cmd = exec.Command("git", "status", "--short")
	cmd.Dir = root
	statusOut, _ := cmd.Output()
	hasChanges := strings.Contains(string(statusOut), ".md") || strings.Contains(string(statusOut), ".json")

	if hasChanges {
		return nil // Sync happened, files were updated
	}
	return fmt.Errorf("project sync completed but no files were updated")
}

// RunbookFixStaleLocks expires locks older than 24h
// Runbook: "fix_stale_locks"
func RunbookFixStaleLocks(ctx context.Context, a *alerts.Alert) error {
	root := filepath.Dir(filepath.Dir(filepath.Join(".ovav", "runtime", "alerts")))

	locksDir := filepath.Join(root, ".ovav", "locks")
	entries, err := os.ReadDir(locksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No locks dir = nothing to do
		}
		return fmt.Errorf("read locks dir: %w", err)
	}

	now := time.Now()
	var expiredCount int

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		age := now.Sub(info.ModTime())
		if age > 24*time.Hour {
			// Delete expired lock
			lockPath := filepath.Join(locksDir, e.Name())
			if err := os.Remove(lockPath); err == nil {
				expiredCount++
			}
		}
	}

	if expiredCount == 0 {
		return fmt.Errorf("no expired locks found")
	}

	return nil
}

// RunbookFixAgentProjection runs ovav convert --agents to sync agents
// Runbook: "fix_agent_projection"
func RunbookFixAgentProjection(ctx context.Context, a *alerts.Alert) error {
	root := filepath.Dir(filepath.Dir(filepath.Join(".ovav", "runtime", "alerts")))

	// Run: ovav convert --agents
	cmd := exec.Command("go", "run", "-C", filepath.Join(root, "go-runtime"), "./cmd/ovav/", "convert", "--agents")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("convert --agents failed: %w, output: %s", err, string(out))
	}

	// Verify count matches
	cmd = exec.Command("bash", "-c", fmt.Sprintf(`
		source_count=$(find %s/go-runtime/internal/agents -name "*.yaml" | wc -l)
		runtime_count=$(find %s/go-runtime/internal/runtimes/opencode/agents -name "*.md" ! -name "ovav.md" | wc -l)
		if [ "$source_count" != "$runtime_count" ]; then
			echo "count mismatch: source=$source_count runtime=$runtime_count"
			exit 1
		fi
	`, root, root))
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("count verification failed")
	}

	return nil
}

// RunbookFixSBOMBaselines regenerates the SBOM and creates integrity baseline
// Runbook: "fix_sbom_baseline"
func RunbookFixSBOMBaseline(ctx context.Context, a *alerts.Alert) error {
	root := filepath.Dir(filepath.Dir(filepath.Join(".ovav", "runtime", "alerts")))

	// Generate SBOM using the sbom package directly (autonomous, no binary needed)
	s, err := sbom.Generate(root)
	if err != nil {
		return fmt.Errorf("sbom.Generate failed: %w", err)
	}

	// Write SBOM to .ovav/registry/sbom.yaml
	sbomDir := filepath.Join(root, ".ovav", "registry")
	os.MkdirAll(sbomDir, 0755)
	sbomPath := filepath.Join(sbomDir, "sbom.yaml")
	data, _ := json.MarshalIndent(s, "", "  ")
	if err := os.WriteFile(sbomPath, data, 0644); err != nil {
		return fmt.Errorf("write sbom: %w", err)
	}

	// Create/update integrity baseline
	baselineDir := filepath.Join(root, ".ovav", "integrity")
	os.MkdirAll(baselineDir, 0755)
	baselinePath := filepath.Join(baselineDir, "baseline.json")
	baseline := fmt.Sprintf(`{"version":"1.0","created":"%s","operator":"OVAV-AGENTS","sbom_hash":"%x"}`,
		time.Now().Format(time.RFC3339), sha256.Sum256(data))
	if err := os.WriteFile(baselinePath, []byte(baseline), 0644); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}

	return nil
}

// RunbookFixRuntimeIntegrity creates or refreshes the integrity baseline
// Runbook: "fix_runtime_integrity"
func RunbookFixRuntimeIntegrity(ctx context.Context, a *alerts.Alert) error {
	root := filepath.Dir(filepath.Dir(filepath.Join(".ovav", "runtime", "alerts")))

	baselineDir := filepath.Join(root, ".ovav", "integrity_backups")
	os.MkdirAll(baselineDir, 0755)
	baselinePath := filepath.Join(baselineDir, "baseline.json")

	baseline := fmt.Sprintf(`{
  "version": "1.0",
  "created": "%s",
  "operator": "OVAV-AGENTS",
  "core_files": {
    "AGENTS.md": "%s",
    "opencode.json": "%s",
    ".ovav/policy/permission_authority.json": "%s",
    ".ovav/plan/caps.yaml": "%s",
    "go-runtime/go.mod": "%s",
    "tools/validators/validate_all.py": "%s"
  }
}`, time.Now().Format(time.RFC3339),
		hashFile(filepath.Join(root, "AGENTS.md")),
		hashFile(filepath.Join(root, "opencode.json")),
		hashFile(filepath.Join(root, ".ovav", "policy", "permission_authority.json")),
		hashFile(filepath.Join(root, ".ovav", "plan", "caps.yaml")),
		hashFile(filepath.Join(root, "go-runtime", "go.mod")),
		hashFile(filepath.Join(root, "tools", "validators", "validate_all.py")))

	if err := os.WriteFile(baselinePath, []byte(baseline), 0644); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}

	return nil
}

// hashFile computes SHA256 hash of a file
func hashFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "file-not-found"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
