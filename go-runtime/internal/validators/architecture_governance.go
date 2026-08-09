// F3 — Architecture Governance validator.
//
// Validates OVAV architectural governance:
//   - Stack purity: no new Python product code, TS only in frontend
//   - Migration progress: Go % of total codebase tracked
//   - Architecture decision records presence
//   - Tech stack compliance with declared strategy
package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArchitectureGovernance validates architectural governance rules (F3).
type ArchitectureGovernance struct{}

func NewArchitectureGovernance() *ArchitectureGovernance { return &ArchitectureGovernance{} }

func (a *ArchitectureGovernance) ID() string   { return "architecture_governance" }
func (a *ArchitectureGovernance) Name() string { return "F3 — Architecture Governance" }
func (a *ArchitectureGovernance) Description() string {
	return "Validates stack purity, migration progress, and architectural governance compliance"
}
func (a *ArchitectureGovernance) Weight() int { return 5 }

func (a *ArchitectureGovernance) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	repoRoot := resolveRepoRoot(root)

	// ── 1. Stack purity — count Go vs Python vs TS LOC ─────────────────────
	goLOC := countLOC(filepath.Join(repoRoot, "go-runtime"), ".go")
	pyLOC := countLOC(filepath.Join(repoRoot, "tools"), ".py")
	tsLOC := countLOC(filepath.Join(repoRoot, "tools", "cpanel", "src"), ".ts") +
		countLOC(filepath.Join(repoRoot, "tools", "cpanel", "src"), ".tsx")

	totalProduct := goLOC + tsLOC
	if totalProduct > 0 {
		goPct := float64(goLOC) / float64(totalProduct) * 100
		if goPct < 80 {
			issues = append(issues, fmt.Sprintf(
				"F3: Go product share below 80%%: %.1f%% (%d Go LOC / %d total product LOC)",
				goPct, goLOC, totalProduct))
		}
	}

	// Warn if Python governance overwhelms product code
	governanceRatio := float64(pyLOC) / float64(totalProduct+1)
	if governanceRatio > 10 {
		issues = append(issues, fmt.Sprintf(
			"F3: Python governance is %.0fx larger than product code (%d Python vs %d product LOC) — migration candidate",
			governanceRatio, pyLOC, totalProduct))
	}

	// ── 2. Stack compliance — no .go files in tools/ ──────────────────────
	var goInTools []string
	filepath.Walk(filepath.Join(repoRoot, "tools"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			rel, _ := filepath.Rel(repoRoot, path)
			goInTools = append(goInTools, rel)
		}
		return nil
	})
	for _, v := range goInTools {
		issues = append(issues, fmt.Sprintf("F3: Go file in tools/ governance directory: %s", v))
	}

	// ── 3. Architecture decision records ───────────────────────────────────
	adrDir := filepath.Join(repoRoot, "docs", "architecture")
	if _, err := os.Stat(adrDir); os.IsNotExist(err) {
		issues = append(issues, "F3: architecture decision records directory missing: docs/architecture/")
	} else {
		entries, _ := os.ReadDir(adrDir)
		adrCount := 0
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "ADR-") || strings.HasPrefix(e.Name(), "adr-") {
				adrCount++
			}
		}
		if adrCount == 0 {
			issues = append(issues, "F3: no architecture decision records found in docs/architecture/")
		}
	}

	// ── 4. Caps.yaml plan freshness ────────────────────────────────────────
	capsYAML := filepath.Join(repoRoot, ".ovav", "plan", "caps.yaml")
	if info, err := os.Stat(capsYAML); err == nil {
		age := time.Since(info.ModTime())
		if age > 72*time.Hour {
			issues = append(issues, fmt.Sprintf(
				"F3: caps.yaml plan is stale (%.0f hours since last update)", age.Hours()))
		}
	} else {
		issues = append(issues, "F3: caps.yaml plan not found — canonical plan required")
	}

	// ── 5. Deprecated doc references ───────────────────────────────────────
	deprecatedDocs := []string{
		"IMPLEMENTATION_PLAN.md",
		"docs/implementation/07_IMPLEMENTATION_ROADMAP.md",
		"active_context_ledger.yaml",
	}
	for _, dd := range deprecatedDocs {
		path := filepath.Join(repoRoot, dd)
		if _, err := os.Stat(path); err == nil {
			issues = append(issues, fmt.Sprintf("F3: deprecated document found: %s — remove it", dd))
		}
		// Also check inside .ovav/
		altPath := filepath.Join(repoRoot, ".ovav", "runtime", dd)
		if _, err := os.Stat(altPath); err == nil {
			issues = append(issues, fmt.Sprintf("F3: deprecated document found: .ovav/runtime/%s — remove it", dd))
		}
	}

	// ── Result ─────────────────────────────────────────────────────────────
	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}
	return Result{
		ID:          a.ID(),
		Name:        a.Name(),
		Status:      status,
		Issues:      issues,
		Weight:      a.Weight(),
		Duration:    time.Since(start),
		Description: a.Description(),
	}
}

// countLOC counts lines in files with the given extension under dir.
func countLOC(dir, ext string) int {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0
	}
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ext) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		count += len(strings.Split(string(data), "\n"))
		return nil
	})
	return count
}
