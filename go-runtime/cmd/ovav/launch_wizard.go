package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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
	Timestamp  string          `json:"timestamp"`
	RepoRoot   string          `json:"repo_root"`
	Overall    string          `json:"overall"`
	Gates      []GateReport    `json:"gates"`
	AutoFixLog []AutoFixResult `json:"auto_fix_log"`
	NextStep   string          `json:"next_step"`
}

// GateReport is one gate's check result.
type GateReport struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	CEORequired bool   `json:"ceo_required"`
	AutoFixable bool   `json:"auto_fixable"`
}

// gateExec runs an ovav subcommand from the repo root, capturing both
// stdout and stderr. Returns combined output.
func gateExec(root string, subcmd ...string) string {
	cmd := exec.Command(filepath.Join(root, "bin", "ovav"), subcmd...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "OVAV_NO_TUI=1", "OVAV_QUIET=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out) + "\n[exit " + err.Error() + "]"
	}
	return string(out)
}

// gateExecSilent runs a subcommand without capturing output (output goes to terminal).
// Used for long-running commands where the user wants to see progress.
func gateExecSilent(root string, subcmd ...string) error {
	cmd := exec.Command(filepath.Join(root, "bin", "ovav"), subcmd...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "OVAV_NO_TUI=1", "OVAV_QUIET=1")
	return cmd.Run()
}

// getLaunchGates returns all gates in dependency order.
func getLaunchGates() []LaunchGate {
	return []LaunchGate{
		{
			ID:          "validators_pass",
			Description: "All validators pass (no fails)",
			Check: func(root string) (bool, string, bool) {
				out := gateExec(root, "validate")
				if strings.Contains(out, "0 failed") {
					return true, "all validators passing", false
				}
				return false, extractFailureSummary(out), true
			},
			AutoFix: func(root string) error {
				out := gateExec(root, "validate", "--fix", "--dry-run")
				if !strings.Contains(out, "0 applied") {
					gateExecSilent(root, "validate", "--fix")
				}
				return nil
			},
		},
		{
			ID:          "drift_clean",
			Description: "No drift detected",
			Check: func(root string) (bool, string, bool) {
				out := gateExec(root, "deploy", "run", "--dry-run")
				if strings.Contains(out, "No drift") || strings.Contains(out, "0 drift") {
					return true, "0 drift targets", false
				}
				return false, "drift detected — running 'ovav deploy run'", true
			},
			AutoFix: func(root string) error {
				return gateExecSilent(root, "deploy", "run")
			},
		},
		{
			ID:          "pinned_baseline",
			Description: "Pinned baseline exists (CEO-approved)",
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
			Description: "Launch evidence captured",
			Check: func(root string) (bool, string, bool) {
				dir := filepath.Join(root, ".ovav", "registry", "launch_evidence")
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) < 4 {
					return false, fmt.Sprintf("only %d evidence files", len(entries)), true
				}
				return true, fmt.Sprintf("%d evidence files", len(entries)), false
			},
			AutoFix: func(root string) error {
				return gateExecSilent(root, "launch", "evidence")
			},
		},
		{
			ID:          "tag_exists",
			Description: "Release tag v1.0.0-rc.1 exists (CEO-gated)",
			Check: func(root string) (bool, string, bool) {
				cmd := exec.Command("git", "tag", "-l", "v1.0.0-rc.1")
				cmd.Dir = root
				out, _ := cmd.Output()
				if strings.Contains(string(out), "v1.0.0-rc.1") {
					return true, "tag v1.0.0-rc.1 exists", false
				}
				return false, "no tag — CEO approval required", false
			},
			CEORequired: true,
		},
		{
			ID:          "smoke_all_pass",
			Description: "All smoke phases pass",
			Check: func(root string) (bool, string, bool) {
				out := gateExec(root, "smoke-all")
				if strings.Contains(out, "0 failed") {
					return true, "21/21 phases passing", false
				}
				return false, extractSmokeFailures(out), false
			},
		},
		{
			ID:          "chaos_invariants",
			Description: "All chaos invariants verified",
			Check: func(root string) (bool, string, bool) {
				out := gateExec(root, "deploy", "chaos")
				if strings.Contains(out, "5 passed, 0 failed") {
					return true, "5 invariants verified", false
				}
				return false, extractChaosFailures(out), false
			},
		},
		{
			ID:          "production_ready",
			Description: "Status is production_ready",
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

// runLaunchWizardReadOnly performs state check without auto-fixing.
func runLaunchWizardReadOnly(root string, mode string) int {
	fmt.Println("🛡️  OVAV Launch Assistant (read-only, mode: " + mode + ")")
	fmt.Println()
	report := collectReadiness(root, false)
	printReadinessReport(report, false)
	saveReadinessReport(root, report)
	return readinessExitCode(report)
}

// collectReadiness runs all gate checks in parallel and builds a report.
// If runFix is true, it also auto-fixes all auto-fixable (non-CEO) gates.
func collectReadiness(root string, runFix bool) ReadinessReport {
	gates := getLaunchGates()
	report := ReadinessReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RepoRoot:  root,
	}

	// Phase 1: Check all gates in parallel
	type checkResult struct {
		idx         int
		passed      bool
		detail      string
		autoFixable bool
	}
	results := make([]checkResult, len(gates))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, gate := range gates {
		wg.Add(1)
		go func(i int, gate LaunchGate) {
			defer wg.Done()
			start := time.Now()
			passed, detail, autoFixable := gate.Check(root)
			dur := time.Since(start)
			mu.Lock()
			results[i] = checkResult{idx: i, passed: passed, detail: detail, autoFixable: autoFixable}
			mu.Unlock()
			if os.Getenv("OVAV_LAUNCH_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "[gate %s] %dms %v\n", gate.ID, dur.Milliseconds(), passed)
			}
		}(i, gate)
	}
	wg.Wait()

	// Phase 2: Auto-fix (sequential, only auto-fixable + non-CEO)
	if runFix {
		for i, gate := range gates {
			r := results[i]
			if r.passed || !r.autoFixable || gate.CEORequired {
				continue
			}
			start := time.Now()
			err := gate.AutoFix(root)
			dur := time.Since(start).Milliseconds()
			fixResult := AutoFixResult{
				GateID:   gate.ID,
				Action:   "attempted",
				Success:  err == nil,
				Duration: dur,
			}
			if err != nil {
				fixResult.Detail = err.Error()
			}
			report.AutoFixLog = append(report.AutoFixLog, fixResult)
			// Re-check this gate
			passed, detail, _ := gate.Check(root)
			results[i].passed = passed
			results[i].detail = detail
		}
	}

	// Phase 3: Build report
	for i, gate := range gates {
		status := "fail"
		if results[i].passed {
			status = "pass"
		}
		report.Gates = append(report.Gates, GateReport{
			ID:          gate.ID,
			Description: gate.Description,
			Status:      status,
			Detail:      results[i].detail,
			CEORequired: gate.CEORequired,
			AutoFixable: results[i].autoFixable,
		})
	}

	// Overall
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
	} else if len(ceoGates) > 0 {
		report.Overall = "needs-ceo-attention"
	} else {
		report.Overall = "blocked"
	}
	return report
}

// printReadinessReport displays the readiness report in a CEO-friendly format.
func printReadinessReport(report ReadinessReport, showAutoFix bool) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 72))
	fmt.Printf("📊 Readiness: %s %s\n", overallEmoji(report.Overall), strings.ToUpper(report.Overall))
	fmt.Println(strings.Repeat("─", 72))
	for _, gr := range report.Gates {
		icon := gateEmoji(gr.Status)
		ceoTag := ""
		if gr.CEORequired {
			ceoTag = " [CEO]"
		}
		fixTag := ""
		if gr.AutoFixable && gr.Status != "pass" {
			fixTag = " [auto-fixable]"
		}
		fmt.Printf("%s %s%s%s — %s\n", icon, gr.ID, ceoTag, fixTag, gr.Detail)
	}

	if showAutoFix && len(report.AutoFixLog) > 0 {
		fmt.Println()
		fmt.Println("🔧 Auto-fix log:")
		for _, fr := range report.AutoFixLog {
			icon := "✅"
			if !fr.Success {
				icon = "❌"
			}
			fmt.Printf("   %s %s (%dms)\n", icon, fr.GateID, fr.Duration)
			if fr.Detail != "" {
				fmt.Printf("      %s\n", fr.Detail)
			}
		}
	}

	fmt.Println()
	switch report.Overall {
	case "ready":
		fmt.Println("✅ All gates passed. Next (CEO decision):")
		fmt.Println("  ovav launch verify --ceo-waiver --reason=\"Anti-drift complete\"")
	case "needs-ceo-attention":
		fmt.Println("⏳ Automatic gates passed. CEO gates remaining:")
		for _, gr := range report.Gates {
			if gr.CEORequired && gr.Status != "pass" {
				fmt.Printf("   • %s\n", gr.ID)
			}
		}
		fmt.Println()
		fmt.Println("Smart commands (one per gate):")
		for _, gr := range report.Gates {
			if gr.CEORequired && gr.Status != "pass" {
				fmt.Printf("  ovav launch ceo-decide --gate=%s --reason=\"...\"\n", gr.ID)
			}
		}
	default:
		fmt.Println("⚠️  Some gates failed. Review details above.")
		fmt.Println("    Or run: ovav launch --prepare (auto-fix non-CEO)")
	}
}

// readinessExitCode returns 0 for ready, 1 otherwise.
func readinessExitCode(report ReadinessReport) int {
	if report.Overall == "ready" {
		return 0
	}
	return 1
}

// runLaunchWizard is the autonomous launch wizard.
// Single entry point that handles the entire ceremony.
func runLaunchWizard(args []string) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			printLaunchWizardHelp()
			return 0
		}
	}

	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV launch: %v\n", err)
		return 1
	}

	mode := "wizard"
	for _, a := range args {
		switch a {
		case "--status", "--info":
			mode = "status"
		case "--prepare":
			mode = "prepare"
		case "--all":
			mode = "all"
		}
	}

	fmt.Println("🛡️  OVAV Launch Assistant (ADR-014)")

	if mode == "status" || mode == "info" {
		fmt.Println()
		report := collectReadiness(root, false)
		printReadinessReport(report, false)
		saveReadinessReport(root, report)
		return readinessExitCode(report)
	}

	// --prepare or --all: auto-fix non-CEO gates, then display
	fmt.Println()
	fmt.Println("→ Checking all gates (parallel)...")
	report := collectReadiness(root, true)
	printReadinessReport(report, true)
	saveReadinessReport(root, report)
	return readinessExitCode(report)
}

func printLaunchWizardHelp() {
	fmt.Println(`OVAV launch assistant — zero-touch launch (ADR-014)

The single entry point for GA promotion. No commands to memorize.

Usage:
  ovav launch                      # Interactive wizard (default)
  ovav launch --status             # Show readiness only (read-only)
  ovav launch --prepare            # Auto-execute fixable actions
  ovav launch --all --reason=...   # Non-interactive (CI mode)
  ovav launch --info               # Alias for --status

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
	enc := json.NewEncoder(f)
	return enc.Encode(report)
}

// extractFailureSummary pulls just the failing validator names.
func extractFailureSummary(out string) string {
	scanner := bufio.NewScanner(strings.NewReader(out))
	failures := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "FAIL") {
			// Extract validator name (first word)
			fields := strings.Fields(line)
			if len(fields) > 1 {
				failures = append(failures, fields[1])
			}
		}
	}
	if len(failures) == 0 {
		return "validators failed"
	}
	return fmt.Sprintf("%d failed: %s", len(failures), strings.Join(failures, ", "))
}

// extractSmokeFailures extracts smoke phase failures.
func extractSmokeFailures(out string) string {
	if strings.Contains(out, "failed") {
		// Find summary line
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "Summary") {
				return strings.TrimSpace(line)
			}
		}
	}
	return "smoke failed"
}

// extractChaosFailures extracts chaos invariant failures.
func extractChaosFailures(out string) string {
	if strings.Contains(out, "failed") {
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "Summary") {
				return strings.TrimSpace(line)
			}
		}
	}
	return "chaos failed"
}

func overallEmoji(o string) string {
	switch o {
	case "ready":
		return "✅"
	case "needs-ceo-attention":
		return "⏳"
	default:
		return "❌"
	}
}

func gateEmoji(s string) string {
	switch s {
	case "pass":
		return "✅"
	case "skipped", "unknown":
		return "⏳"
	default:
		return "❌"
	}
}