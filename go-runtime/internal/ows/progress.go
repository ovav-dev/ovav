package ows

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ProgressTracker manages animated progress reporting.
type ProgressTracker struct {
	label    string
	total   int
	current int
	mu       sync.Mutex
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
	Name    string
	Pass    bool
	Issues  []string
	DurMS   int64
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
			results = append(results, runNodeJSVerification(dir, 3)...)
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
		valCmd := exec.Command("go", valArgs...)
		valCmd.Dir = validateDir
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
	hygIssues := []string{}
	if !hygiene.Clean {
		hygIssues = append(hygIssues, fmt.Sprintf("%d hygiene issue(s)", hygiene.TotalIssues))
	}
	results = append(results, PhaseResult{Name: "hygiene", Pass: hygiene.Clean, Issues: hygIssues, DurMS: time.Since(start).Milliseconds()})

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
		vetCmd := exec.Command("go", "vet", "./...")
		vetCmd.Dir = goRoot
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
	testCmd := exec.Command("go", "test", "-count=1", "./...")
	testCmd.Dir = goRoot
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

// runNodeJSVerification runs biome/tsc and npm test.
func runNodeJSVerification(nodeRoot string, phaseCount int) []PhaseResult {
	var results []PhaseResult
	tracker := NewProgressTracker("nodejs", phaseCount)

	// Try biome (Rust-based, fast)
	hasBiome := false
	if _, err := exec.LookPath("biome"); err == nil {
		hasBiome = true
	}

	if hasBiome {
		start := time.Now()
		cmd := exec.Command("biome", "check", "--enabled", "lint", ".")
		cmd.Dir = nodeRoot
		out, err := cmd.CombinedOutput()
		pass := err == nil
		issues := []string{}
		if !pass {
			for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
				if t := strings.TrimSpace(l); t != "" {
					issues = append(issues, t)
				}
			}
		}
		results = append(results, PhaseResult{Name: "biome check", Pass: pass, Issues: issues, DurMS: time.Since(start).Milliseconds()})
		tracker.Increment("biome check")
	} else {
		// Fallback to tsc
		start := time.Now()
		cmd := exec.Command("npx", "tsc", "--noEmit")
		cmd.Dir = nodeRoot
		out, err := cmd.CombinedOutput()
		issues := []string{}
		if err != nil {
			for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
				if t := strings.TrimSpace(l); t != "" {
					issues = append(issues, t)
				}
			}
		}
		results = append(results, PhaseResult{Name: "tsc", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
		tracker.Increment("tsc")
	}

	// npm test
	start := time.Now()
	cmd := exec.Command("npm", "test")
	cmd.Dir = nodeRoot
	out, err := cmd.CombinedOutput()
	issues := []string{}
	if err != nil {
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
			if t := strings.TrimSpace(l); t != "" {
				issues = append(issues, t)
			}
		}
	}
	results = append(results, PhaseResult{Name: "npm test", Pass: err == nil, Issues: issues, DurMS: time.Since(start).Milliseconds()})
	tracker.Increment("npm test")

	return results
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
			for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
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
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
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
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
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
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
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
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
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
			for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n")[:5] {
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
