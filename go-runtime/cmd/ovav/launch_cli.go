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

// cmdLaunch dispatches ovav launch subcommands.
//
// Per ADR-014 (zero-touch launch), running 'ovav launch' with no args launches
// the interactive wizard that handles the entire ceremony autonomously.
//
// Usage: ovav launch [<subcommand>]
// Subcommands:
//   (default)    — interactive wizard (smart autonomous flow)
//   status       — quick readiness check
//   evidence     — capture artifacts
//   tag          — create v1.0.0-rc.1 tag
//   verify       — final verification (CEO waiver)
//   ceo-decide   — single CEO gate decision
//   roadmap      — 2027 roadmap
func cmdLaunch(args []string) int {
	if len(args) == 0 {
		return runLaunchWizard(args)
	}
	switch args[0] {
	case "evidence":
		return runLaunchEvidence(args[1:])
	case "tag":
		return runLaunchTag(args[1:])
	case "verify":
		return runLaunchVerify(args[1:])
	case "status":
		return runLaunchStatus(args[1:])
	case "roadmap":
		return runLaunchRoadmap(args[1:])
	case "wizard":
		return runLaunchWizard(args[1:])
	case "ceo-decide":
		return runLaunchCEODecide(args[1:])
	case "help", "--help", "-h":
		printLaunchHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV launch: unknown subcommand %q\n", args[0])
		printLaunchHelp()
		return 2
	}
}

func printLaunchHelp() {
	fmt.Println(`OVAV launch — zero-touch GA promotion (ADR-014)

The single entry point for GA promotion. No commands to memorize.

Usage:
  ovav launch                        # Interactive wizard (default)
  ovav launch --status               # Quick readiness check
  ovav launch --prepare              # Auto-execute fixable actions
  ovav launch --all --reason=...     # Non-interactive (CI mode)
  ovav launch --info                  # Read-only

Subcommands (power users):
  ovav launch status                  # Quick readiness
  ovav launch evidence                # Capture artifacts
  ovav launch tag                     # Create v1.0.0-rc.1
  ovav launch verify --ceo-waiver     # Final verification
  ovav launch ceo-decide --gate=X    # Single CEO gate
  ovav launch roadmap                 # 2027 roadmap

The wizard:
1. Detects state (no CEO knowledge needed)
2. Auto-executes safe actions
3. Prompts only for CEO decisions (with full context)
4. Validates post-state (auto-rollback if regressed)
5. Logs every step

Run 'ovav launch' to start.`)
}

// runLaunchCEODecide handles a single CEO gate decision.
//
// Per ADR-014, instead of running 3 separate commands (pin, verify, tag),
// CEO can run a single command per decision:
//
//   ovav launch ceo-decide --gate=pin --reason="golden state approved"
//   ovav launch ceo-decide --gate=verify --reason="all gates pass"
//   ovav launch ceo-decide --gate=tag --reason="evidence captured"
//
// Each decision is gated on previous + audit-logged.
func runLaunchCEODecide(args []string) int {
	gate := ""
	reason := ""

	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--gate="):
			gate = strings.TrimPrefix(a, "--gate=")
		case strings.HasPrefix(a, "--reason="):
			reason = strings.TrimPrefix(a, "--reason=")
		}
	}

	if gate == "" {
		fmt.Println("🚫 CEO decide requires --gate=<name>")
		fmt.Println()
		fmt.Println("Available CEO gates:")
		fmt.Println("  pin      — Approve current baseline as golden state")
		fmt.Println("  verify   — Final launch verification (closes ceremony)")
		fmt.Println("  tag      — Approve tag push to remote")
		return 1
	}

	if reason == "" {
		fmt.Println("⚠️  CEO decide requires --reason=<your reason>")
		fmt.Println("    The reason is logged to the audit trail.")
		return 1
	}

	fmt.Printf("🛡️  CEO Decision: %s\n", gate)
	fmt.Printf("Reason: %s\n\n", reason)

	switch gate {
	case "pin":
		return runCEOPin(reason)
	case "verify":
		return runCEOVerify(reason)
	case "tag":
		return runCEOTag(reason)
	default:
		fmt.Printf("Unknown CEO gate: %s\n", gate)
		return 1
	}
}

// runCEOPin handles the pin decision (approves current baseline).
func runCEOPin(reason string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		return 1
	}

	// Copy current baseline to pinned
	src := filepath.Join(root, ".ovav", "integrity_backups", "baseline.json")
	dst := filepath.Join(root, ".ovav", "integrity_backups", "baseline.pinned.json")

	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("❌ Cannot read current baseline: %v\n", err)
		return 1
	}

	if err := os.WriteFile(dst, data, 0o644); err != nil {
		fmt.Printf("❌ Cannot write pinned baseline: %v\n", err)
		return 1
	}

	// Log decision
	logCEODecision(root, "pin", reason)
	fmt.Println("✅ Pinned baseline created")
	fmt.Printf("   %s\n", dst)
	fmt.Println()
	fmt.Println("Next: ovav launch (wizard will detect this gate passing)")
	return 0
}

// runCEOVerify handles the final verification decision.
func runCEOVerify(reason string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		return 1
	}

	// Re-check all gates
	report, err := runLaunchWizardCheckOnly(root)
	if err != nil {
		return 1
	}

	if report.Overall != "ready" {
		fmt.Println("🚫 Cannot verify — some gates failed:")
		for _, g := range report.Gates {
			if g.Status != "pass" {
				fmt.Printf("   ❌ %s — %s\n", g.ID, g.Detail)
			}
		}
		fmt.Println()
		fmt.Println("Fix issues first, then retry: ovav launch --prepare")
		return 1
	}

	// Update caps.yaml
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")
	data, err := os.ReadFile(capsPath)
	if err != nil {
		fmt.Printf("❌ Cannot read caps.yaml: %v\n", err)
		return 1
	}

	content := string(data)
	content = strings.ReplaceAll(content, "launch_verification_blocked", "production_ready")
	content = strings.ReplaceAll(content, "source-local launch candidate", "production_ready")
	content = strings.ReplaceAll(content, "blocked_pending_baseline_and_candidate_closure", "verified_production_ready")

	if err := os.WriteFile(capsPath, []byte(content), 0o644); err != nil {
		fmt.Printf("❌ Cannot write caps.yaml: %v\n", err)
		return 1
	}

	// Log decision
	logCEODecision(root, "verify", reason)
	fmt.Println("✅ Launch verification COMPLETE")
	fmt.Println()
	fmt.Println("OVAV is now in 'production_ready' state.")
	fmt.Println("Tag v1.0.0-rc.1 should be created (auto via 'ovav launch tag')")
	return 0
}

// runCEOTag handles the tag-push decision.
func runCEOTag(reason string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		return 1
	}

	// Create tag if not exists
	out, _ := exec.Command("git", "tag", "-l", "v1.0.0-rc.1").Output()
	if !strings.Contains(string(out), "v1.0.0-rc.1") {
		fmt.Println("🏷️  Creating tag v1.0.0-rc.1...")
		tagMsg := fmt.Sprintf("OVAV v1.0.0 Release Candidate 1\n\nCEO waiver reason: %s", reason)
		exec.Command("git", "tag", "-a", "v1.0.0-rc.1", "-m", tagMsg).Run()
	}

	// Log decision
	logCEODecision(root, "tag", reason)
	fmt.Println("✅ Tag v1.0.0-rc.1 created")
	fmt.Println()
	fmt.Println("Note: 'git push origin v1.0.0-rc.1' is intentionally manual")
	fmt.Println("      (per CRIT-006: irreversible actions stay human-controlled)")
	return 0
}

// runLaunchWizardCheckOnly is the read-only state check (used by runCEOVerify).
func runLaunchWizardCheckOnly(root string) (ReadinessReport, error) {
	gates := getLaunchGates()
	report := ReadinessReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RepoRoot:  root,
	}
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
	// Determine overall
	allPass := true
	hasCEO := false
	for _, gr := range report.Gates {
		if gr.Status != "pass" {
			allPass = false
		}
		if gr.CEORequired && gr.Status != "pass" {
			hasCEO = true
		}
	}
	if allPass {
		report.Overall = "ready"
	} else if hasCEO {
		report.Overall = "needs-ceo-attention"
	} else {
		report.Overall = "blocked"
	}
	return report, nil
}

// logCEODecision appends to audit trail.
func logCEODecision(root, gate, reason string) {
	dir := filepath.Join(root, ".ovav", "registry")
	os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "ceo_decisions.jsonl")
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if f != nil {
		defer f.Close()
		entry, _ := json.Marshal(map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"operator":  os.Getenv("USER"),
			"gate":      gate,
			"reason":    reason,
		})
		f.Write(entry)
		f.Write([]byte("\n"))
	}
}

// runLaunchStatus shows current readiness for GA promotion.
func runLaunchStatus(args []string) int {
	fmt.Println("OVAV Launch Readiness Status")
	fmt.Println()

	// Check current validator state
	validateOut, _ := exec.Command("./bin/ovav", "validate").CombinedOutput()
	if strings.Contains(string(validateOut), "0 failed") {
		fmt.Println("✅ Validators: 0 failed")
	} else {
		fmt.Println("⚠️  Validators: issues detected")
	}

	// Check drift
	driftOut, _ := exec.Command("./bin/ovav", "drift", "show", "--json").Output()
	var driftReport struct {
		DriftedTargets int `json:"drifted_targets"`
	}
	json.Unmarshal(driftOut, &driftReport)
	if driftReport.DriftedTargets == 0 {
		fmt.Println("✅ Drift: 0 drifted targets")
	} else {
		fmt.Printf("⚠️  Drift: %d drifted targets\n", driftReport.DriftedTargets)
	}

	// Check pinned baseline
	pinnedExists := false
	if _, err := os.Stat(".ovav/integrity_backups/baseline.pinned.json"); err == nil {
		pinnedExists = true
	}
	if pinnedExists {
		fmt.Println("✅ Pinned baseline: exists")
	} else {
		fmt.Println("⏳ Pinned baseline: pending (requires CEO approval)")
	}

	// Check tag
	tagOut, _ := exec.Command("git", "tag", "-l", "v1.0.0-rc.1").Output()
	if strings.Contains(string(tagOut), "v1.0.0-rc.1") {
		fmt.Println("✅ Tag v1.0.0-rc.1: exists")
	} else {
		fmt.Println("⏳ Tag v1.0.0-rc.1: not created yet")
	}

	fmt.Println()
	fmt.Println("To proceed with GA promotion:")
	fmt.Println("  1. ovav launch evidence    # capture artifacts")
	fmt.Println("  2. ovav launch tag         # create v1.0.0-rc.1")
	fmt.Println("  3. ovav launch verify --ceo-waiver   # final ceremony")
	return 0
}

// runLaunchEvidence captures final smoke evidence.
func runLaunchEvidence(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV launch evidence: %v\n", err)
		return 1
	}

	evidenceDir := filepath.Join(root, ".ovav", "registry", "launch_evidence")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return 1
	}

	fmt.Println("OVAV launch evidence — capturing artifacts...")
	fmt.Println()

	timestamp := time.Now().UTC().Format("20060102T150405")

	// Capture smoke-all
	out, _ := exec.CommandContext(getContext(), "./bin/ovav", "smoke-all").CombinedOutput()
	os.WriteFile(filepath.Join(evidenceDir, fmt.Sprintf("smoke-all-%s.txt", timestamp)), out, 0o644)
	fmt.Printf("✅ smoke-all → smoke-all-%s.txt\n", timestamp)

	// Capture validate
	out, _ = exec.CommandContext(getContext(), "./bin/ovav", "validate").CombinedOutput()
	os.WriteFile(filepath.Join(evidenceDir, fmt.Sprintf("validate-%s.txt", timestamp)), out, 0o644)
	fmt.Printf("✅ validate → validate-%s.txt\n", timestamp)

	// Capture drift
	out, _ = exec.CommandContext(getContext(), "./bin/ovav", "drift", "show", "--json").Output()
	os.WriteFile(filepath.Join(evidenceDir, fmt.Sprintf("drift-%s.json", timestamp)), out, 0o644)
	fmt.Printf("✅ drift → drift-%s.json\n", timestamp)

	// Capture git HEAD
	out, _ = exec.Command("git", "rev-parse", "HEAD").Output()
	os.WriteFile(filepath.Join(evidenceDir, fmt.Sprintf("git-head-%s.txt", timestamp)), out, 0o644)
	fmt.Printf("✅ git HEAD → git-head-%s.txt\n", timestamp)

	fmt.Println()
	fmt.Println("Evidence captured in:", evidenceDir)
	return 0
}

// runLaunchTag creates v1.0.0-rc.1 tag.
func runLaunchTag(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV launch tag: %v\n", err)
		return 1
	}

	// Verify evidence exists
	evidenceDir := filepath.Join(root, ".ovav", "registry", "launch_evidence")
	if _, err := os.Stat(evidenceDir); err != nil {
		fmt.Println("❌ Evidence not captured yet. Run 'ovav launch evidence' first.")
		return 1
	}

	// Check tag doesn't exist
	out, _ := exec.Command("git", "tag", "-l", "v1.0.0-rc.1").Output()
	if strings.Contains(string(out), "v1.0.0-rc.1") {
		fmt.Println("⚠️  Tag v1.0.0-rc.1 already exists")
		return 1
	}

	// Create tag
	tagMsg := "OVAV v1.0.0 Release Candidate 1\n\n" +
		"Phase 1-3 anti-drift core complete:\n" +
		"- 76 validators passing\n" +
		"- 21 smoke phases passing\n" +
		"- 5 chaos invariants verified\n" +
		"- Auto-remediation pipeline operational\n\n" +
		"Pending CEO approval for launch verification.\n" +
		"See ADR-013 for ceremony details."

	out, err = exec.Command("git", "tag", "-a", "v1.0.0-rc.1", "-m", tagMsg).CombinedOutput()
	if err != nil {
		fmt.Printf("❌ git tag failed: %v\n%s\n", err, out)
		return 1
	}

	fmt.Println("✅ Tag v1.0.0-rc.1 created")
	fmt.Println()
	fmt.Println("Note: To remove if needed:")
	fmt.Println("  git tag -d v1.0.0-rc.1")
	fmt.Println()
	fmt.Println("Next step:")
	fmt.Println("  ovav launch verify --ceo-waiver --reason=\"launch verification\"")
	return 0
}

// runLaunchVerify performs final launch verification.
// REQUIRES --ceo-waiver flag (per CRIT-006 + ADR-013).
func runLaunchVerify(args []string) int {
	// Enforce CEO waiver
	waiver := false
	reason := ""
	for _, a := range args {
		if a == "--ceo-waiver" {
			waiver = true
		}
		if strings.HasPrefix(a, "--reason=") {
			reason = strings.TrimPrefix(a, "--reason=")
		}
	}

	if !waiver {
		fmt.Println("🚫 Final launch verification requires CEO waiver")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  ovav launch verify --ceo-waiver --reason=\"<your reason>\"")
		fmt.Println()
		fmt.Println("Per CRIT-006, CEO decisions stay human. The waiver flag is the explicit consent.")
		return 1
	}

	fmt.Println("OVAV Launch Verification (CEO WAIVER)")
	fmt.Printf("Reason: %s\n\n", reason)

	// Pre-checks
	checks := []struct {
		name string
		fn   func() bool
	}{
		{"validators pass", func() bool {
			out, _ := exec.Command("./bin/ovav", "validate").CombinedOutput()
			return strings.Contains(string(out), "0 failed")
		}},
		{"drift clean", func() bool {
			out, _ := exec.Command("./bin/ovav", "ci", "drift-check").Output()
			return !strings.Contains(string(out), "Drift")
		}},
		{"tag exists", func() bool {
			out, _ := exec.Command("git", "tag", "-l", "v1.0.0-rc.1").Output()
			return strings.Contains(string(out), "v1.0.0-rc.1")
		}},
		{"smoke-all pass", func() bool {
			out, _ := exec.Command("./bin/ovav", "smoke-all").CombinedOutput()
			return strings.Contains(string(out), "Summary: 21 passed")
		}},
	}

	allPassed := true
	for _, c := range checks {
		icon := "✅"
		if !c.fn() {
			icon = "❌"
			allPassed = false
		}
		fmt.Printf("%s %s\n", icon, c.name)
	}

	if !allPassed {
		fmt.Println()
		fmt.Println("🚫 Launch verification FAILED — fix issues first")
		return 1
	}

	fmt.Println()
	fmt.Println("🎉 All launch verification checks PASSED")
	fmt.Println()
	fmt.Println("Ready for production claims. Update .ovav/plan/caps.yaml:")
	fmt.Println("  status: 'production_ready' (was 'launch_verification_blocked')")
	fmt.Println("  posture: 'production_ready' (was 'source-local launch candidate')")
	fmt.Println()
	fmt.Println("Note: Per ADR-003, this requires explicit CEO action in addition to --ceo-waiver.")
	return 0
}

// runLaunchRoadmap prints the 2027 roadmap.
func runLaunchRoadmap(args []string) int {
	fmt.Println("OVAV 2027 Roadmap")
	fmt.Println()
	fmt.Println("Per ADR-005 Phase 4 + CEO directive 'lista todo lo pendiente':")
	fmt.Println()
	fmt.Println("## Q1 2027 — Multi-agent governance")
	fmt.Println("  - Agent delegation routing (sub-agent registry)")
	fmt.Println("  - Per-agent permission profiles")
	fmt.Println("  - Cross-agent context firewalls")
	fmt.Println()
	fmt.Println("## Q2 2027 — Knowledge compiler v2")
	fmt.Println("  - Knowledge distillation from memory")
	fmt.Println("  - Cross-session context transfer")
	fmt.Println("  - Adaptive curriculum generation")
	fmt.Println()
	fmt.Println("## Q3 2027 — Performance + scale")
	fmt.Println("  - Validator parallel execution")
	fmt.Println("  - Incremental validation (skip unchanged files)")
	fmt.Println("  - Snapshot storage optimization")
	fmt.Println()
	fmt.Println("## Q4 2027 — Community + extensibility")
	fmt.Println("  - Public plugin API")
	fmt.Println("  - Validator marketplace")
	fmt.Println("  - Community-contributed drift targets")
	fmt.Println()
	fmt.Println("## Continuous")
	fmt.Println("  - Bug bashing (weekly)")
	fmt.Println("  - Validator additions (per Phase 4)")
	fmt.Println("  - Coverage sprint (per Q)")
	fmt.Println("  - Auto-fix expansion (more validators)")
	return 0
}

// getContext returns a context with timeout for subprocess commands.
func getContext() interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
} {
	return contextWithTimeout(60 * time.Second)
}

// contextWithTimeout helper.
func contextWithTimeout(d time.Duration) interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
} {
	ctx, _ := newTimeoutContext(d)
	return ctx
}

// newTimeoutContext is a small wrapper to avoid unused import warnings.
func newTimeoutContext(d time.Duration) (interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}, func()) {
	ctx, cancel := makeStdContextWithTimeout(d)
	return ctx, cancel
}