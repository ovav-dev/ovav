package ows

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProgressTracker manages animated progress reporting.

// collectIssueLines extracts up to maxLines trimmed, non-empty lines from command
// output. Guards against slice out-of-bounds when output is empty or shorter
// than maxLines (replaces the repeated buggy [:5] pattern).
func collectIssueLines(out string, maxLines int) []string {
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lines = append(lines, t)
			if len(lines) >= maxLines {
				break
			}
		}
	}
	return lines
}

type ProgressTracker struct {
	label   string
	total   int
	current int
	mu      sync.Mutex
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker(label string, total int) *ProgressTracker {
	return &ProgressTracker{label: label, total: total}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinnerIcon(n int) string {
	return spinnerFrames[n%len(spinnerFrames)]
}

// Increment advances the counter and prints updated progress.
func (p *ProgressTracker) Increment(stepLabel string) {
	p.mu.Lock()
	p.current++
	pct := 0
	if p.total > 0 {
		pct = p.current * 100 / p.total
	}
	cur := p.current
	p.mu.Unlock()

	icon := spinnerIcon(cur)
	if cur >= p.total {
		fmt.Printf("\r  ✅ %-42s %3d%%\n", stepLabel, pct)
	} else {
		fmt.Printf("\r  %s %-42s %3d%%", icon, stepLabel, pct)
	}
}

// PhaseResult holds the result of a single verification phase.
type PhaseResult struct {
	Name   string
	Pass   bool
	Issues []string
	DurMS  int64
}

// VerifyPhases runs multi-stack verification with animated progress.
// Detects all stacks (Go, Node.js, Python, Rust) and runs appropriate verifiers.
// changedFiles is used for scope-based validator filtering (targets only affected validators).
func VerifyPhases(repoRoot string, changedFiles []string) ([]PhaseResult, error) {
	// Phase 0: Stack detection
	fmt.Println()
	tracker := NewProgressTracker("verify", 1)
	tracker.Increment("detecting stacks")

	stacks := DetectStacks(repoRoot)
	stackSummary := stacks.Summary()
	fmt.Printf("  🏈 detected: %s\n", stackSummary)

	// Determine which verifiers to run based on detected stacks
	var results []PhaseResult

	// Run Go verification if Go stack detected
	if stacks.HasGo() {
		goRoot := repoRoot
		if _, err := os.Stat(repoRoot + "/go-runtime/go.mod"); err == nil {
			goRoot = repoRoot + "/go-runtime"
		}
		results = append(results, runGoVerification(goRoot, 3)...)
	}

	// Run Node.js verification if Node stack detected
	for _, s := range stacks.Stacks {
		if s.Type == StackTSReact || s.Type == StackTSNode || s.Type == StackTSVue {
			dir := repoRoot
			if s.Dir != "." {
				dir = repoRoot + "/" + s.Dir
			}
			results = append(results, runNodeJSVerification(dir, 2)...)
			break // only one Node.js project per repo
		}
	}

	// Run Python verification if Python stack detected
	if _, err := os.Stat(repoRoot + "/pyproject.toml"); err == nil {
		results = append(results, runPythonVerification(repoRoot, 3)...)
	} else if _, err := os.Stat(repoRoot + "/requirements.txt"); err == nil {
		results = append(results, runPythonVerification(repoRoot, 3)...)
	}

	// Run Rust verification if Rust stack detected
	if _, err := os.Stat(repoRoot + "/Cargo.toml"); err == nil {
		results = append(results, runRustVerification(repoRoot, 4)...)
	}

	// Phase: validators (if available)
	hasValidators := false
	if info, err := os.Stat(repoRoot + "/go-runtime/internal/validators/cmd/validate"); err == nil && info.IsDir() {
		hasValidators = true
	}
	if hasValidators {
		start := time.Now()
		fmt.Printf("\n  ⏳ validators running...\n")
		validateDir := repoRoot + "/go-runtime/internal/validators/cmd/validate"
		valArgs := []string{"run", "."}
		if len(changedFiles) > 0 {
			valArgs = append(valArgs, "--changed-files", strings.Join(changedFiles, ","))
			valArgs = append(valArgs, "--root", repoRoot)
		}
		valCmd := goCmd(validateDir, valArgs...)
		valOut, _ := valCmd.CombinedOutput()
		_, fail := parseValidateOutput(string(valOut))
		valIssues := []string{}
		if fail > 0 {
			valIssues = append(valIssues, fmt.Sprintf("%d validator(s) failed", fail))
		}
		results = append(results, PhaseResult{Name: "validators", Pass: fail == 0, Issues: valIssues, DurMS: time.Since(start).Milliseconds()})
	}

	// Final phase: hygiene
	start := time.Now()
	hygiene := WorkspaceHygieneScan(repoRoot)
	hygieneResult := hygienePhaseResult(hygiene)
	hygieneResult.DurMS = time.Since(start).Milliseconds()
	results = append(results, hygieneResult)

	return results, nil
}

// runGoVerification runs go vet, gofmt, and go test.
func runGoVerification(goRoot string, phaseCount int) []PhaseResult {
	var results []PhaseResult
	tracker := NewProgressTracker("go", phaseCount)

	var wg sync.WaitGroup
	var mu sync.Mutex

	addResult := func(pr PhaseResult) {
		mu.Lock()
		results = append(results, pr)
		mu.Unlock()
	}

	// Phase 1+2: go vet + gofmt in parallel
	wg.Add(2)
	go func() {
		defer wg.Done()
		start := time.Now()
		vetCmd := goCmd(goRoot, "vet", "./...")
		out, err := vetCmd.CombinedOutput()
		pass := true
		issues := []string{}
		if err != nil {
			pass = false
			issues = append(issues, fmt.Sprintf("go vet: %s", truncateOutput(string(out), 150)))
		}
		addResult(PhaseResult{Name: "go vet", Pass: pass, Issues: issues, DurMS: time.Since(start).Milliseconds()})
		tracker.Increment("go vet")
	}()

	go func() {
		defer wg.Done()
		start := time.Now()
		fmtCmd := exec.Command("gofmt", "-l", ".")
		fmtCmd.Dir = goRoot
		out, _ := fmtCmd.Output()
		pass := true
		issues := []string{}
		if strings.TrimSpace(string(out)) != "" {
			pass = false
			cnt := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
			issues = append(issues, fmt.Sprintf("%d unformatted file(s)", cnt))
		}
		addResult(PhaseResult{Name: "gofmt", Pass: pass, Issues: issues, DurMS: time.Since(start).Milliseconds()})
		tracker.Increment("gofmt")
	}()

	wg.Wait()

	// Phase 3: go test
	fmt.Printf("\n  ⏳ go test running (this may take a moment)...\n")
	startTest := time.Now()
	testCmd := goCmd(goRoot, "test", "-count=1", "./...")
	out, err := testCmd.CombinedOutput()
	testIssues := []string{}
	testPass := true
	if err != nil {
		testPass = false
		testIssues = append(testIssues, fmt.Sprintf("go test: %s", truncateOutput(string(out), 150)))
	}
	results = append(results, PhaseResult{Name: "go test", Pass: testPass, Issues: testIssues, DurMS: time.Since(startTest).Milliseconds()})
	tracker.Increment("go test")

	return results
}

// runNodeJSVerification runs only checks explicitly configured by the project.
func runNodeJSVerification(nodeRoot string, phaseCount int) []PhaseResult {
	return runNodeJSVerificationMode(nodeRoot, phaseCount, true)
}

func runNodeJSVerificationMode(nodeRoot string, phaseCount int, includeTests bool) []PhaseResult {
	tracker := NewProgressTracker("nodejs", phaseCount)
	manifest, err := loadNodePackageManifest(nodeRoot)
	if err != nil {
		return []PhaseResult{{Name: "Node manifest", Pass: false, Issues: []string{err.Error()}}}
	}
	manager, err := resolveNodePackageManager(nodeRoot, manifest.PackageManager)
	if err != nil {
		return []PhaseResult{{Name: "Node package manager", Pass: false, Issues: []string{err.Error()}}}
	}

	var results []PhaseResult
	runTool := func(name, tool string, args ...string) {
		start := time.Now()
		cmd, cmdErr := nodeToolCommand(nodeRoot, manager, tool, args...)
		var out []byte
		if cmdErr == nil {
			out, cmdErr = cmd.CombinedOutput()
		}
		issues := collectIssueLines(string(out), 5)
		if cmdErr != nil {
			issues = append(issues, cmdErr.Error())
		}
		results = append(results, PhaseResult{
			Name: name, Pass: cmdErr == nil, Issues: issues, DurMS: time.Since(start).Milliseconds(),
		})
		tracker.Increment(name)
	}

	if hasBiomeConfig(nodeRoot, manifest) {
		runTool("biome check", "biome", "check", ".")
	} else if hasTypeScriptConfig(nodeRoot) {
		runTool("tsc", "tsc", "--noEmit")
	} else {
		results = append(results, PhaseResult{Name: "tsc", Pass: true, Issues: []string{"skipped: no TypeScript or Biome configuration"}})
		tracker.Increment("tsc (skipped)")
	}

	if strings.TrimSpace(manifest.Scripts["test"]) == "" || !includeTests {
		reason := "skipped: package.json has no test script"
		if !includeTests && strings.TrimSpace(manifest.Scripts["test"]) != "" {
			reason = "skipped: quick verification mode"
		}
		results = append(results, PhaseResult{Name: "node test", Pass: true, Issues: []string{reason}})
		tracker.Increment("node test (skipped)")
		return results
	}

	start := time.Now()
	cmd, cmdErr := nodeScriptCommand(nodeRoot, manager, "test")
	var out []byte
	if cmdErr == nil {
		out, cmdErr = cmd.CombinedOutput()
	}
	issues := collectIssueLines(string(out), 5)
	if cmdErr != nil {
		issues = append(issues, cmdErr.Error())
	}
	results = append(results, PhaseResult{
		Name: "node test", Pass: cmdErr == nil, Issues: issues, DurMS: time.Since(start).Milliseconds(),
	})
	tracker.Increment("node test")

	return results
}

func hygienePhaseResult(hygiene *HygieneResult) PhaseResult {
	issues := make([]string, 0, 2)
	if hygiene.BlockingIssues > 0 {
		issues = append(issues, fmt.Sprintf("%d blocking hygiene issue(s)", hygiene.BlockingIssues))
	}
	if hygiene.WarningIssues > 0 {
		issues = append(issues, fmt.Sprintf("%d warning hygiene issue(s)", hygiene.WarningIssues))
	}
	return PhaseResult{Name: "hygiene", Pass: hygiene.BlockingIssues == 0, Issues: issues}
}

type nodePackageManifest struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PackageManager  string            `json:"packageManager"`
}

func loadNodePackageManifest(nodeRoot string) (nodePackageManifest, error) {
	data, err := os.ReadFile(filepath.Join(nodeRoot, "package.json"))
	if err != nil {
		return nodePackageManifest{}, fmt.Errorf("read package.json: %w", err)
	}
	var manifest nodePackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nodePackageManifest{}, fmt.Errorf("parse package.json: %w", err)
	}
	return manifest, nil
}

func hasTypeScriptConfig(nodeRoot string) bool {
	matches, err := filepath.Glob(filepath.Join(nodeRoot, "tsconfig*.json"))
	return err == nil && len(matches) > 0
}

func hasBiomeConfig(nodeRoot string, manifest nodePackageManifest) bool {
	for _, name := range []string{"biome.json", "biome.jsonc"} {
		if _, err := os.Stat(filepath.Join(nodeRoot, name)); err == nil {
			return true
		}
	}
	return manifest.Dependencies["@biomejs/biome"] != "" || manifest.DevDependencies["@biomejs/biome"] != ""
}

func resolveNodePackageManager(nodeRoot, declared string) (string, error) {
	if declared != "" {
		name := strings.SplitN(declared, "@", 2)[0]
		switch name {
		case "npm", "pnpm", "yarn", "bun":
			return name, nil
		default:
			return "", fmt.Errorf("unsupported packageManager %q", declared)
		}
	}

	for _, candidate := range []struct {
		lockfile string
		manager  string
	}{
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"bun.lock", "bun"},
		{"bun.lockb", "bun"},
		{"package-lock.json", "npm"},
	} {
		if _, err := os.Stat(filepath.Join(nodeRoot, candidate.lockfile)); err == nil {
			return candidate.manager, nil
		}
	}

	return "npm", nil
}

func nodeToolCommand(nodeRoot, manager, tool string, args ...string) (*exec.Cmd, error) {
	var command string
	var commandArgs []string
	switch manager {
	case "npm":
		command = "npx"
		commandArgs = append([]string{"--no-install", tool}, args...)
	case "pnpm", "yarn":
		command = manager
		commandArgs = append([]string{"exec", tool}, args...)
	case "bun":
		command = "bun"
		commandArgs = append([]string{"run", tool}, args...)
	default:
		return nil, fmt.Errorf("unsupported package manager %q", manager)
	}

	cmd := exec.Command(command, commandArgs...)
	cmd.Dir = nodeRoot
	cmd.Env = append(os.Environ(), "CI=true")
	return cmd, nil
}

func nodeScriptCommand(nodeRoot, manager, script string) (*exec.Cmd, error) {
	switch manager {
	case "npm", "pnpm", "yarn", "bun":
		cmd := exec.Command(manager, "run", script)
		cmd.Dir = nodeRoot
		cmd.Env = append(os.Environ(), "CI=true")
		return cmd, nil
	default:
		return nil, fmt.Errorf("unsupported package manager %q", manager)
	}
}

// runPythonVerification runs ruff and pytest.
func runPythonVerification(pythonRoot string, phaseCount int) []PhaseResult {
	var results []PhaseResult
	tracker := NewProgressTracker("python", phaseCount)

	// ruff check
	hasRuff := false
	if _, err := exec.LookPath("ruff"); err == nil {
		hasRuff = true
	}

	if hasRuff {
		start := time.Now()
		cmd := exec.Command("ruff", "check", ".")
		cmd.Dir = pythonRoot
		out, err := cmd.CombinedOutput()
		issues := []string{}
		if err != nil {
			for _, l := range collectIssueLines(string(out), 5) {
				if t := strings.TrimSpace(l); t != "" {
					issues = append(issues, t)
				}
			}
		}
		results = append(results, PhaseResult{Name: "ruff check", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
		tracker.Increment("ruff check")
	}

	// pytest
	start := time.Now()
	cmd := exec.Command("pytest")
	cmd.Dir = pythonRoot
	out, err := cmd.CombinedOutput()
	issues := []string{}
	if err != nil {
		for _, l := range collectIssueLines(string(out), 5) {
			if t := strings.TrimSpace(l); t != "" {
				issues = append(issues, t)
			}
		}
	}
	results = append(results, PhaseResult{Name: "pytest", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
	tracker.Increment("pytest")

	return results
}

// runRustVerification runs cargo check, fmt, test, clippy.
func runRustVerification(rustRoot string, phaseCount int) []PhaseResult {
	var results []PhaseResult
	tracker := NewProgressTracker("rust", phaseCount)

	// cargo check
	start := time.Now()
	cmd := exec.Command("cargo", "check")
	cmd.Dir = rustRoot
	out, err := cmd.CombinedOutput()
	issues := []string{}
	if err != nil {
		for _, l := range collectIssueLines(string(out), 5) {
			if t := strings.TrimSpace(l); t != "" {
				issues = append(issues, t)
			}
		}
	}
	results = append(results, PhaseResult{Name: "cargo check", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
	tracker.Increment("cargo check")

	// cargo fmt --check
	start = time.Now()
	cmd = exec.Command("cargo", "fmt", "--check")
	cmd.Dir = rustRoot
	out, err = cmd.CombinedOutput()
	issues = []string{}
	if err != nil {
		for _, l := range collectIssueLines(string(out), 5) {
			if t := strings.TrimSpace(l); t != "" {
				issues = append(issues, t)
			}
		}
	}
	results = append(results, PhaseResult{Name: "cargo fmt", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
	tracker.Increment("cargo fmt")

	// cargo test
	start = time.Now()
	cmd = exec.Command("cargo", "test")
	cmd.Dir = rustRoot
	out, err = cmd.CombinedOutput()
	issues = []string{}
	if err != nil {
		for _, l := range collectIssueLines(string(out), 5) {
			if t := strings.TrimSpace(l); t != "" {
				issues = append(issues, t)
			}
		}
	}
	results = append(results, PhaseResult{Name: "cargo test", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
	tracker.Increment("cargo test")

	// cargo clippy (optional)
	if _, err := exec.LookPath("cargo-clippy"); err == nil {
		start = time.Now()
		cmd = exec.Command("cargo", "clippy")
		cmd.Dir = rustRoot
		out, err = cmd.CombinedOutput()
		issues = []string{}
		if err != nil {
			for _, l := range collectIssueLines(string(out), 5) {
				if t := strings.TrimSpace(l); t != "" {
					issues = append(issues, t)
				}
			}
		}
		results = append(results, PhaseResult{Name: "cargo clippy", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
		tracker.Increment("cargo clippy")
	}

	return results
}

// detectChangedFiles auto-detects changed files using git diff vs parent branch.
func detectChangedFiles(repoRoot string) []string {
	branch, err := runGitOutput(repoRoot, "branch", "--show-current")
	if err != nil || strings.TrimSpace(branch) == "" {
		return nil
	}
	branch = strings.TrimSpace(branch)

	parent := "origin/develop"
	if strings.HasPrefix(branch, "hotfix/") || strings.HasPrefix(branch, "emergency/") {
		parent = "origin/main"
	}

	out, err := runGitOutput(repoRoot, "diff", "--name-only", parent+"...HEAD")
	if err != nil {
		return nil
	}

	var files []string
	for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
		f = strings.TrimSpace(f)
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}

// ScanPhase runs a quick pre-scan with animated progress.
// Shows % progress while detecting stack, branch, and changed files.
func ScanPhase(repoRoot string) (stackSummary, branch string, changedCount int) {
	fmt.Println("\n  🔍 Pre-scan started...")
	tracker := NewProgressTracker("scan", 4)

	// Step 1: Detect branch
	tracker.Increment("detecting branch")
	branch, _ = runGitOutput(repoRoot, "branch", "--show-current")
	branch = strings.TrimSpace(branch)

	// Step 2: Detect stack
	tracker.Increment("detecting stack")
	stacks := DetectStacks(repoRoot)
	stackSummary = stacks.Summary()

	// Step 3: Count changed files
	tracker.Increment("counting changes")
	files := detectChangedFiles(repoRoot)
	changedCount = len(files)

	// Step 4: Done
	tracker.Increment("scan complete")

	return
}
