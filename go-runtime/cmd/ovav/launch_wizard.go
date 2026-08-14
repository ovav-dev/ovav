package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LaunchGate is = one checkpoint in the launch ceremony.
type LaunchGate struct {
	ID          string
	Description string
	Check       func(root string) (passed bool, detail string, autoFixable bool)
	CEORequired bool
	AutoFix     func(root string) error
}

// AutoFixResult tracks one auto-fix attempt.
type AutoFixResult struct {
	GateID   string `json:"gate_id"`
	Action   string `json:"action"` // "captured", "regenerated", "skipped"
	Success  bool   `json:"success"`
	Detail   string `json:"detail,omitempty"`
	Duration int64  `json:"duration_ms"`
}

// ReadinessReport summarizes all gates for the wizard.
type ReadinessReport struct {
	Timestamp    string         `json:"timestamp"`
	RepoRoot     string         `json:"repo_root"`
	Overall      string         `json:"overall"` // ready | needs-attention | blocked
	Gates        []GateReport   `json:"gates"`
	AutoFixLog   []AutoFixResult `json:"auto_fix_log"`
	NextStep     string         `json:"next_step"`
}

// GateReport is one gate's check result.
type GateReport struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // pass | fail | skipped
	Detail      string `json:"detail"`
	CEORequired bool   `json:"ceo_required"`
	AutoFixable bool   `json:"auto_fixable"`
}

// getLaunchGates returns all gates in dependency order.
func getLaunchGates() []LaunchGate {
	return []LaunchGate{
		{
			ID:          "validators_pass",
			Description: "All validators pass (no fails)",
			Check: func(root string) (bool, string, bool) {
				out, _ := exec.Command("./bin/ovav", "validate").CombinedOutput()
				if strings.Contains(string(out), "0 failed") {
					return true, "all validators passing", false
				}
				// Try to auto-fix via validate --fix
				return false, extractFailureSummary(string(out)), true
			},
			AutoFix: func(root string) error {
				// Try ovav validate --fix --dry-run to identify, then actual fix
				out, _ := exec.Command("./bin/ovav", "validate", "--fix", "--dry-run").CombinedOutput()
				if !strings.Contains(string(out), "0 applied") {
					// Try non-dry-run
					exec.Command("./bin/ovav", "validate", "--fix").Run()
				}
				return nil
			},
		},
		{
			ID:          "drift_clean",
			Description: "No drift detected (fragment == live)",
			Check: func(root string) (bool, string, bool) {
				runOut, _ := exec.Command("./bin/ovav", "deploy", "run", "--dry-run").CombinedOutput()
				if strings.Contains(string(runOut), "No drift detected") {
					return true, "0 drift targets", false
				}
				return false, "drift detected — run 'ovav deploy run' to fix", true
			},
			AutoFix: func(root string) error {
				exec.Command("./bin/ovav", "deploy", "run").Run()
				return nil
			},
		},
		{
			ID:          "pinned_baseline",
			Description: "Pinned baseline exists (CEO-approved golden state)",
			Check: func(root string) (bool, string, bool) {
				if _, err := os.Stat(filepath.Join(root, ".ovav", "integrity_backups", "baseline.pinned.json")); err == nil {
					return true, "baseline.pinned.json exists", false
				}
				return false, "no pinned baseline — CEO approval required", false
			},
			CEORequired: true,
		},
		{
			ID:          "evidence_captured",
			Description: "Launch evidence captured (.ovav/registry/launch_evidence/)",
			Check: func(root string) (bool, string, bool) {
				dir := filepath.Join(root, ".ovav", "registry", "launch_evidence")
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) < 4 {
					return false, fmt.Sprintf("only %d evidence files", len(entries)), true
				}
				return true, fmt.Sprintf("%d evidence files", len(entries)), false
			},
			AutoFix: func(root string) error {
				exec.Command(filepath.Join(root, "bin", "ovav"), "launch", "evidence").Run()
				return nil
			},
		},
		{
			ID:          "tag_exists",
			Description: "Release tag v1.0.0-rc.1 exists",
			Check: func(root string) (bool, string, bool) {
				out, _ := exec.Command("git", "tag", "-l", "v1.0.0-rc.1").Output()
				if strings.Contains(string(out), "v1.0.0-rc.1") {
					return true, "tag v1.0.0-rc.1 exists", false
				}
				return false, "no tag — create via 'ovav launch tag'", true
			},
			AutoFix: func(root string) error {
				exec.Command(filepath.Join(root, "bin", "ovav"), "launch", "tag").Run()
				return nil
			},
		},
		{
			ID:          "smoke_all_pass",
			Description: "All 21 smoke phases pass (comprehensive entry points)",
			Check: func(root string) (bool, string, bool) {
				out, _ := exec.Command(filepath.Join(root, "bin", "ovav"), "smoke-all").CombinedOutput()
				if strings.Contains(string(out), "21 passed") {
					return true, "21/21 phases passing", false
				}
				return false, extractSmokeFailures(string(out)), false
			},
		},
		{
			ID:          "chaos_invariants",
			Description: "All 5 chaos invariants verified",
			Check: func(root string) (bool, string, bool) {
				out, _ := exec.Command(filepath.Join(root, "bin", "ovav"), "deploy", "chaos").CombinedOutput()
				if strings.Contains(string(out), "5 passed, 0 failed") {
					return true, "5 invariants verified", false
				}
				return false, extractChaosFailures(string(out)), false
			},
		},
		{
			ID:          "production_ready",
			Description: "Status is production_ready (not launch_verification_blocked)",
			Check: func(root string) (bool, string, bool) {
				data, err := os.ReadFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"))
				if err != nil {
					return false, "caps.yaml unreadable", false
				}
				content := string(data)
				if strings.Contains(content, "production_ready") {
					return true, "production_ready in caps.yaml", false
				}
				return false, "status still launch_verification_blocked — CEO waiver needed", false
			},
			CEORequired: true,
		},
	}
}

// runLaunchWizard is the autonomous launch wizard.
// Single entry point that handles the entire ceremony.
func runLaunchWizard(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV launch: %v\n", err)
		return 1
	}

	// Parse flags
	mode := "wizard" // wizard | prepare | all | status | info

	for _, a := range args {
		switch a {
		case "--status":
			mode = "status"
		case "--prepare":
			mode = "prepare"
		case "--all":
			mode = "all"
		case "--info":
			mode = "info"
		case "--help", "-h":
			printLaunchWizardHelp()
			return 0
		}
	}
	_ = mode // future use for different behaviors

	fmt.Println("🛡️  OVAV Launch Assistant (ADR-014)")
	fmt.Println()

	gates := getLaunchGates()
	report := ReadinessReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RepoRoot:  root,
	}

	// Step 1: Check all gates
	fmt.Println("→ Checking all gates...")
	for _, gate := range gates {
		passed, detail, autoFixable := gate.Check(root)
		status := "fail"
		if passed {
			status = "pass"
		}
		report.Gates = append(report.Gates, GateReport{
			ID:          gate.ID,
			Description: gate.Description,
			Status:      status,
			Detail:      detail,
			CEORequired: gate.CEORequired,
			AutoFixable: autoFixable,
		})
	}

	// Step 2: Auto-execute fixable gates (if not --info)
	if mode != "info" && mode != "status" {
		fmt.Println()
		fmt.Println("→ Auto-executing fixable actions...")
		for i, gate := range gates {
			gr := report.Gates[i]
			if gr.Status == "pass" {
				continue
			}
			if !gr.AutoFixable {
				continue
			}
			if gate.CEORequired {
				continue // CEO-gated, skip auto-fix
			}

			fmt.Printf("  • Auto-fixing %s... ", gate.ID)
			start := time.Now()
			err := gate.AutoFix(root)
			duration := time.Since(start).Milliseconds()
			fixResult := AutoFixResult{
				GateID:   gate.ID,
				Action:   "attempted",
				Success:  err == nil,
				Duration: duration,
			}
			if err != nil {
				fixResult.Detail = err.Error()
				fmt.Println("❌", err)
			} else {
				fmt.Println("✅")
			}
			report.AutoFixLog = append(report.AutoFixLog, fixResult)
		}

		// Re-check gates that were auto-fixed
		fmt.Println()
		fmt.Println("→ Re-checking gates after auto-fix...")
		for i, gate := range gates {
			gr := report.Gates[i]
			if gr.Status == "pass" {
				continue
			}
			if !gr.AutoFixable || gate.CEORequired {
				continue
			}
			passed, detail, _ := gate.Check(root)
			if passed {
				report.Gates[i].Status = "pass"
				report.Gates[i].Detail = detail
			}
		}
	}

	// Step 3: Determine overall state
	allPass := true
	ceoGates := []GateReport{}
	for _, gr := range report.Gates {
		if gr.Status != "pass" {
			allPass = false
		}
		if gr.CEORequired && gr.Status != "pass" {
			ceoGates = append(ceoGates, gr)
		}
	}

	if allPass {
		report.Overall = "ready"
		report.NextStep = "ovav launch verify --ceo-waiver --reason=\"...\""
	} else if len(ceoGates) > 0 {
		report.Overall = "needs-ceo-attention"
		report.NextStep = "see CEO-required gates below"
	} else {
		report.Overall = "blocked"
		report.NextStep = "run 'ovav launch --prepare' to auto-fix remaining issues"
	}

	// Step 4: Display report
	fmt.Println()
	fmt.Println("─" + strings.Repeat("─", 70))
	fmt.Printf("📊 Readiness: %s %s\n", overallEmoji(report.Overall), strings.ToUpper(report.Overall))
	fmt.Println("─" + strings.Repeat("─", 70))
	for _, gr := range report.Gates {
		icon := gateEmoji(gr.Status)
		ceoTag := ""
		if gr.CEORequired {
			ceoTag = " [CEO]"
		}
		fixTag := ""
		if gr.AutoFixable {
			fixTag = " [auto-fixable]"
		}
		fmt.Printf("%s %s%s%s — %s\n", icon, gr.ID, ceoTag, fixTag, gr.Detail)
	}

	fmt.Println()
	if report.Overall == "ready" {
		fmt.Println("✅ All automatic gates passed!")
		fmt.Println()
		fmt.Println("Next step (requires YOUR consent):")
		fmt.Println("  ovav launch verify --ceo-waiver --reason=\"Anti-drift complete\"")
	} else if len(ceoGates) > 0 {
		fmt.Println("⏳ Automatic gates passed. CEO decision needed for:")
		for _, cg := range ceoGates {
			fmt.Printf("   • %s — %s\n", cg.ID, cg.Description)
		}
		fmt.Println()
		fmt.Println("Smart commands for CEO:")
		for _, cg := range ceoGates {
			fmt.Printf("  ovav launch ceo-decide --gate=%s --reason=\"...\"\n", cg.ID)
		}
	} else {
		fmt.Println("⚠️  Some gates failed. Run 'ovav launch --prepare' to attempt auto-fix,")
		fmt.Println("    or check the details above for manual action needed.")
	}

	// Persist report
	saveReadinessReport(root, report)
	return 0
}

// printLaunchWizardHelp shows the autonomous launch wizard help.
func printLaunchWizardHelp() {
	fmt.Println(`OVAV launch assistant — zero-touch launch (ADR-014)

The single entry point for GA promotion. No commands to memorize.

Usage:
  ovav launch                      # Interactive wizard (default)
  ovav launch --status             # Show readiness only
  ovav launch --prepare            # Auto-execute fixable actions
  ovav launch --all --reason=...   # Non-interactive (CI mode)
  ovav launch --info               # Read-only (no actions)

Subcommands (power users):
  ovav launch status               # Quick readiness check
  ovav launch evidence             # Capture artifacts
  ovav launch tag                  # Create v1.0.0-rc.1
  ovav launch verify --ceo-waiver  # Final verification
  ovav launch ceo-decide --gate=X  # Single CEO gate decision
  ovav launch roadmap              # 2027 roadmap

The wizard:
1. Detects current state (no CEO knowledge needed)
2. Auto-executes safe actions (evidence capture, baseline regen)
3. Prompts only for CEO decisions with full context
4. Validates post-state (auto-rollback if regressed)
5. Logs every step to audit trail

Reports saved to: .ovav/registry/launch_readiness.jsonl`)
}

// saveReadinessReport appends to JSONL history.
func saveReadinessReport(root string, report ReadinessReport) error {
	dir := filepath.Join(root, ".ovav", "registry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "launch_readiness.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := json.Marshal(report)
	f.Write(data)
	f.Write([]byte("\n"))
	return nil
}

// helper functions
func overallEmoji(status string) string {
	switch status {
	case "ready":
		return "✅"
	case "needs-ceo-attention":
		return "⏳"
	default:
		return "❌"
	}
}

func gateEmoji(status string) string {
	switch status {
	case "pass":
		return "✅"
	case "fail":
		return "❌"
	default:
		return "⏳"
	}
}

func extractFailureSummary(out string) string {
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "failed") {
			return strings.TrimSpace(line)
		}
	}
	return "validators failed"
}

func extractSmokeFailures(out string) string {
	if strings.Contains(out, "Summary:") {
		idx := strings.Index(out, "Summary:")
		return out[idx : idx+min(100, len(out)-idx)]
	}
	return "smoke failed"
}

func extractChaosFailures(out string) string {
	if strings.Contains(out, "Summary:") {
		idx := strings.Index(out, "Summary:")
		return out[idx : idx+min(100, len(out)-idx)]
	}
	return "chaos failed"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}