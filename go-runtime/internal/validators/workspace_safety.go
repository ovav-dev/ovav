// workspace_safety.go - OVAV Pre-Write Safety Check
// Migrated from workspace_safety_gate.py
package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ExitSafe    = 0
	ExitUnsafe  = 1
	ExitWarning = 2
)

// SafetyResult holds the check result
type SafetyResult struct {
	Safe  bool
	Level string // "block", "warn", "ok"
	Msg   string
}

// SafetyReport is the JSON output structure
type SafetyReport struct {
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// WorkspaceSafetyValidator implements the workspace safety gate
type WorkspaceSafetyValidator struct {
	repoRoot string
}

// NewWorkspaceSafety creates a new validator instance
func NewWorkspaceSafety() *WorkspaceSafetyValidator {
	return &WorkspaceSafetyValidator{
		repoRoot: findRepoRoot(),
	}
}

// ID returns the unique identifier
func (v *WorkspaceSafetyValidator) ID() string { return "workspace_safety" }

// Name returns the human-readable name
func (v *WorkspaceSafetyValidator) Name() string { return "Workspace Safety Gate" }

// Description returns a one-line description
func (v *WorkspaceSafetyValidator) Description() string {
	return "Checks workspace safety before write operations"
}

// Weight returns the importance weight
func (v *WorkspaceSafetyValidator) Weight() int { return 50 }

// Validate executes the validation
func (v *WorkspaceSafetyValidator) Validate(ctx context.Context, root string) Result {
	// Run in check mode (passive safety status)
	v.RunCheck(false)

	var issues []string

	// Check .git exists
	result := v.checkGitStatus()
	if !result.Safe {
		issues = append(issues, result.Msg)
	}

	// Check workspace_safety_gate reference
	agentsDir := filepath.Join(root, "go-runtime", "internal", "runtimes", "opencode", "agents")
	hasGateRef := false
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, entry := range entries {
			if data, err := os.ReadFile(filepath.Join(agentsDir, entry.Name())); err == nil {
				if strings.Contains(string(data), "workspace_safety_gate") {
					hasGateRef = true
					break
				}
			}
		}
	}

	autoTriggersPath := filepath.Join(root, ".ovav", "registry", "auto_triggers.yaml")
	_, autoTriggersErr := os.Stat(autoTriggersPath)

	if hasGateRef && autoTriggersErr != nil {
		issues = append(issues, "cannot read auto_triggers")
	}

	broken := v.checkBrokenSymlinks()
	for _, b := range broken {
		issues = append(issues, fmt.Sprintf("broken symlink: %s", b))
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}

	return Result{
		ID:          v.ID(),
		Name:        v.Name(),
		Status:      status,
		Message:     fmt.Sprintf("Workspace safety check: %s", status),
		Issues:      issues,
		Weight:      v.Weight(),
		Duration:    0,
		Description: v.Description(),
	}
}

func findRepoRoot() string {
	// Walk up from current working directory to find repo root
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func (v *WorkspaceSafetyValidator) checkGitStatus() SafetyResult {
	// First check if .git exists
	gitPath := filepath.Join(v.repoRoot, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return SafetyResult{
			Safe:  false,
			Level: "block",
			Msg:   ".git not found",
		}
	}

	dangerousDirs := []string{
		"go-runtime/cmd",
		"tools/",
		".ovav/",
	}

	for _, d := range dangerousDirs {
		dirPath := filepath.Join(v.repoRoot, d)
		if _, err := os.Stat(dirPath); err == nil {
			cmd := exec.Command("git", "status", "--porcelain", d)
			cmd.Dir = v.repoRoot
			out, _ := cmd.Output()
			if len(strings.TrimSpace(string(out))) > 0 {
				return SafetyResult{
					Safe:  false,
					Level: "warn",
					Msg:   fmt.Sprintf("Uncommitted changes in %s", d),
				}
			}
		}
	}
	return SafetyResult{Safe: true, Level: "ok", Msg: "No uncommitted changes in critical dirs"}
}

func (v *WorkspaceSafetyValidator) checkProtectedPaths(path string) SafetyResult {
	protected := []string{
		".git/objects",
		".git/refs",
		"go-runtime/internal/vault/secrets",
		".ovav/vault/tokens",
	}

	for _, p := range protected {
		if strings.Contains(path, p) {
			return SafetyResult{
				Safe:  false,
				Level: "block",
				Msg:   fmt.Sprintf("Protected path: %s", p),
			}
		}
	}
	return SafetyResult{Safe: true, Level: "ok", Msg: "Path not protected"}
}

func (v *WorkspaceSafetyValidator) checkFileSize(path string) SafetyResult {
	info, err := os.Stat(path)
	if err != nil {
		return SafetyResult{Safe: true, Level: "ok", Msg: "File does not exist"}
	}

	const maxSize = 10 * 1024 * 1024 // 10MB
	if info.Size() > maxSize {
		return SafetyResult{
			Safe:  false,
			Level: "warn",
			Msg:   fmt.Sprintf("File too large (%d bytes)", info.Size()),
		}
	}
	return SafetyResult{Safe: true, Level: "ok", Msg: "File size OK"}
}

func (v *WorkspaceSafetyValidator) checkBrokenSymlinks() []string {
	var broken []string
	cmd := exec.Command("find", v.repoRoot, "-type", "l", "!", "-exec", "test", "-e", "{}", ";", "-print")
	cmd.Dir = v.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return broken
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line != "" && len(broken) < 10 {
			broken = append(broken, line)
		}
	}
	return broken
}

// RunMutate runs pre-write safety check
func (v *WorkspaceSafetyValidator) RunMutate(path string, jsonOutput bool) int {
	var issues []string
	hasBlock := false

	// Check git status
	result := v.checkGitStatus()
	if !result.Safe {
		issues = append(issues, fmt.Sprintf("WARN: %s", result.Msg))
	}

	// Check specific path if provided
	if path != "" {
		result = v.checkProtectedPaths(path)
		if !result.Safe {
			if result.Level == "block" {
				issues = append(issues, fmt.Sprintf("BLOCK: %s", result.Msg))
				hasBlock = true
			} else {
				issues = append(issues, fmt.Sprintf("WARN: %s", result.Msg))
			}
		}

		if _, err := os.Stat(path); err == nil {
			result = v.checkFileSize(path)
			if !result.Safe {
				issues = append(issues, fmt.Sprintf("WARN: %s", result.Msg))
			}
		}
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Println(issue)
		}
		if hasBlock {
			return ExitUnsafe
		}
		return ExitWarning
	}

	fmt.Println("✅ Workspace safety check passed")
	return ExitSafe
}

// RunCheck runs passive safety status check
func (v *WorkspaceSafetyValidator) RunCheck(jsonOutput bool) int {
	fmt.Println("🔍 OVAV Workspace Safety Check")
	fmt.Println(strings.Repeat("=", 40))

	result := v.checkGitStatus()
	status := "✅"
	if !result.Safe {
		status = "⚠️"
	}
	fmt.Printf("   %s %s\n", status, result.Msg)

	// Check for broken symlinks
	broken := v.checkBrokenSymlinks()
	if len(broken) > 0 {
		fmt.Println("   ⚠️  Found broken symlinks")
		for _, link := range broken {
			if len(link) > 0 {
				fmt.Printf("      - %s\n", link)
			}
		}
	}

	if jsonOutput {
		report := SafetyReport{Status: "ok", Code: 0}
		out, _ := json.Marshal(report)
		fmt.Println(string(out))
	}
	return ExitSafe
}

// WorkspaceSafetyMain is the main entry point for the validator CLI
func WorkspaceSafetyMain(args []string) int {
	var mode, path string
	var jsonOutput bool

	// Simple flag parsing (mimics pflag behavior)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			if i+1 < len(args) {
				mode = args[i+1]
				i++
			}
		case "--path":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
		case "--json":
			jsonOutput = true
		}
	}

	if mode == "" {
		mode = "check"
	}

	validator := NewWorkspaceSafety()

	switch mode {
	case "mutate":
		return validator.RunMutate(path, jsonOutput)
	default:
		return validator.RunCheck(jsonOutput)
	}
}
