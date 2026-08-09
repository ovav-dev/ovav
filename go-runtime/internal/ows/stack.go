// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
// [OVAV-FIX] Use filepath.Clean() and validate path against allowlist
package ows

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ── Stack Detection Engine ──────────────────────────────────────────────────
// Detects project stack (Go/TypeScript/Python/Rust/monorepo) for stack-aware
// validation. Used by Verify() to run the correct validators from the correct
// directories. Consumer-grade: works for any project structure.

// StackType represents a detected technology stack.
type StackType string

const (
	StackGo       StackType = "go"
	StackTSReact  StackType = "ts:react"
	StackTSNode   StackType = "ts:node"
	StackTSVue    StackType = "ts:vue"
	StackPython   StackType = "python"
	StackRust     StackType = "rust"
	StackMonorepo StackType = "monorepo"
	StackUnknown  StackType = "unknown"
)

// StackInfo holds detected stack information for a project.
type StackInfo struct {
	Stacks     []DetectedStack `json:"stacks"`
	IsMonorepo bool            `json:"is_monorepo"`
	Root       string          `json:"root"`
}

// DetectedStack is a single detected stack with its working directory.
type DetectedStack struct {
	Type StackType `json:"type"`
	Dir  string    `json:"dir"` // relative to repoRoot, "." if root
}

// ValidatorsFor returns the recommended validator names for a stack type.
func ValidatorsFor(s StackType) []string {
	switch s {
	case StackGo:
		return []string{"go_vet", "go_test", "gofmt"}
	case StackTSReact, StackTSNode, StackTSVue:
		return []string{"typecheck", "lint", "test"}
	case StackPython:
		return []string{"ruff_check", "pytest"}
	case StackRust:
		return []string{"cargo_check", "cargo_test", "cargo_fmt"}
	case StackMonorepo:
		return []string{"typecheck", "test"}
	default:
		return nil
	}
}

// DetectStacks detects all technology stacks in a project.
// Scans for go.mod, package.json, pyproject.toml, Cargo.toml, and monorepo markers.
func DetectStacks(repoRoot string) *StackInfo {
	info := &StackInfo{Root: repoRoot}
	info.detectGoStacks(repoRoot)
	info.detectJSStacks(repoRoot)
	info.detectPythonStacks(repoRoot)
	info.detectRustStacks(repoRoot)
	info.IsMonorepo = info.detectMonorepo(repoRoot)
	if len(info.Stacks) == 0 {
		info.Stacks = append(info.Stacks, DetectedStack{Type: StackUnknown, Dir: "."})
	}
	return info
}

func (si *StackInfo) detectGoStacks(repoRoot string) {
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		si.Stacks = append(si.Stacks, DetectedStack{Type: StackGo, Dir: "."})
	}
	filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "social-citas" || name == ".ovav" {
			if path != repoRoot {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		if rel != "." && strings.Count(rel, string(filepath.Separator)) >= 3 {
			return filepath.SkipDir
		}
		if rel != "." {
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				si.Stacks = append(si.Stacks, DetectedStack{Type: StackGo, Dir: rel})
			}
		}
		return nil
	})
}

func (si *StackInfo) detectJSStacks(repoRoot string) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "package.json"))
	if err != nil {
		return
	}
	content := string(data)
	stack := StackTSNode
	if strings.Contains(content, `"react"`) || strings.Contains(content, `"next"`) {
		stack = StackTSReact
	} else if strings.Contains(content, `"vue"`) || strings.Contains(content, `"nuxt"`) {
		stack = StackTSVue
	}
	si.Stacks = append(si.Stacks, DetectedStack{Type: stack, Dir: "."})
}

func (si *StackInfo) detectPythonStacks(repoRoot string) {
	// OVAV SYSTEM is Go-native. Skip Python detection if go-runtime exists.
	if _, err := os.Stat(filepath.Join(repoRoot, "go-runtime", "go.mod")); err == nil {
		return // OVAV is Go — skip Python detection entirely
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "pyproject.toml")); err == nil {
		si.Stacks = append(si.Stacks, DetectedStack{Type: StackPython, Dir: "."})
		return
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "requirements.txt")); err == nil {
		si.Stacks = append(si.Stacks, DetectedStack{Type: StackPython, Dir: "."})
	}
}

func (si *StackInfo) detectRustStacks(repoRoot string) {
	if _, err := os.Stat(filepath.Join(repoRoot, "Cargo.toml")); err == nil {
		si.Stacks = append(si.Stacks, DetectedStack{Type: StackRust, Dir: "."})
	}
}

func (si *StackInfo) detectMonorepo(repoRoot string) bool {
	for _, marker := range []string{"pnpm-workspace.yaml", "lerna.json", "turbo.json", "nx.json"} {
		if _, err := os.Stat(filepath.Join(repoRoot, marker)); err == nil {
			return true
		}
	}
	return false
}

// HasGo returns true if any Go stack was detected.
func (si *StackInfo) HasGo() bool {
	for _, s := range si.Stacks {
		if s.Type == StackGo {
			return true
		}
	}
	return false
}

// GoDirs returns all directories containing Go code (relative to repoRoot).
func (si *StackInfo) GoDirs() []string {
	var dirs []string
	for _, s := range si.Stacks {
		if s.Type == StackGo {
			dirs = append(dirs, s.Dir)
		}
	}
	return dirs
}

// PrimaryStack returns the first detected stack (the dominant one).
func (si *StackInfo) PrimaryStack() DetectedStack {
	if len(si.Stacks) > 0 {
		return si.Stacks[0]
	}
	return DetectedStack{Type: StackUnknown, Dir: "."}
}

// Summary returns a human-readable summary of detected stacks.
func (si *StackInfo) Summary() string {
	if len(si.Stacks) == 0 {
		return "no stacks detected"
	}
	parts := make([]string, len(si.Stacks))
	for i, s := range si.Stacks {
		if s.Dir == "." {
			parts[i] = string(s.Type)
		} else {
			parts[i] = string(s.Type) + " @" + s.Dir
		}
	}
	mono := ""
	if si.IsMonorepo {
		mono = " (monorepo)"
	}
	return strings.Join(parts, ", ") + mono
}

// DetectProfileFromBranch detects the worktree profile from a branch name.
func DetectProfileFromBranch(branch string) string {
	if strings.HasPrefix(branch, "hotfix/") || strings.HasPrefix(branch, "hotfix-") {
		return "hotfix"
	}
	if strings.HasPrefix(branch, "release/") || strings.HasPrefix(branch, "release-") {
		return "release"
	}
	if strings.HasPrefix(branch, "emergency") {
		return "emergency"
	}
	if strings.HasPrefix(branch, "enterprise/") || strings.HasPrefix(branch, "enterprise-") {
		return "enterprise"
	}
	if strings.HasPrefix(branch, "spike/") || strings.HasPrefix(branch, "spike-") {
		return "spike"
	}
	if strings.HasPrefix(branch, "research/") || strings.HasPrefix(branch, "research-") {
		return "research"
	}
	if strings.HasPrefix(branch, "docs/") || strings.HasPrefix(branch, "docs-") {
		return "docs"
	}
	return "feature"
}

// VerificationLevel defines how strict verification should be.
type VerificationLevel int

const (
	VerifyBasic    VerificationLevel = iota // go vet + test + fmt + hygiene
	VerifyStandard                          // + coverage + secrets
	VerifyStrict                            // + validate CLI + SBOM
	VerifyMaximum                           // + compliance gate + audit
)

// LevelForProfile returns the verification level for a profile.
func LevelForProfile(profile string) VerificationLevel {
	switch profile {
	case "hotfix":
		return VerifyStandard
	case "release":
		return VerifyStrict
	case "enterprise":
		return VerifyMaximum
	case "emergency":
		return VerifyStrict
	case "spike", "research", "docs":
		return VerifyBasic
	default:
		return VerifyStandard
	}
}

func parseGoModModule(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
