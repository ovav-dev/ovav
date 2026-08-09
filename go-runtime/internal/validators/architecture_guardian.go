package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArchitectureGuardian validates broader architecture structure compliance beyond F1-F3.
// Replaces: check_architecture_guardian.py
type ArchitectureGuardian struct{}

func NewArchitectureGuardian() *ArchitectureGuardian { return &ArchitectureGuardian{} }

func (a *ArchitectureGuardian) ID() string   { return "architecture_guardian" }
func (a *ArchitectureGuardian) Name() string { return "Architecture Guardian" }
func (a *ArchitectureGuardian) Description() string {
	return "Validates broader architecture structure: required dirs, files, and forbidden patterns"
}
func (a *ArchitectureGuardian) Weight() int { return 8 }

// Required canonical directories.
var requiredDirs = []string{
	".ovav/governor",
	".ovav/source/agents",
	".ovav/source/skills",
	".ovav/source/adapters",
	".ovav/connector_bus/connectors",
	".ovav/forge/adapters",
	".ovav/forge/targets",
	".ovav/forge/releases",
	".ovav/registry",
	".ovav/runtime",
	".ovav/logs",
	".ovav/knowledge",
	".ovav/policy",
	".ovav/laws",
	".ovav/visual",
	".ovav/schemas",
	".ovav/lab",
	".ovav/service_areas/shared",
	".ovav/service_areas/platform_engineering",
	".ovav/service_areas/research_intelligence",
	"tools/governor",
	"tools/validators",
	"tools/harnesses",
	"tools/agent_runtime",
	"tools/security",

	"tools/model_integrity",
	"tools/logging",
	"tools/economy",
	"tools/prompt",
	"tools/visual",
	"tools/permissions",
	"clients",
	"docs",
}

// Required foundational files.
var requiredFiles = []string{
	".ovav/governor/IDENTITY.md",
	".ovav/governor/PURPOSE.yaml",
	".ovav/governor/GENESIS.yaml",
	".ovav/governor/AUTHORITY.yaml",
	".ovav/governor/CREATOR.yaml",
	".ovav/source/agents/ovav.md",
	".ovav/laws/ovav_laws.yaml",
	".ovav/policy/permission_authority.json",
	".ovav/plan/caps.yaml",
	"AGENTS.md",
	"CHANGELOG.md",
}

// Known OVAV dirs (acceptable in .ovav/).
var knownOvavDirs = map[string]bool{
	"governor": true, "source": true, "connector_bus": true, "forge": true,
	"registry": true, "runtime": true, "logs": true, "knowledge": true,
	"policy": true, "laws": true, "visual": true, "schemas": true,
	"lab": true, "service_areas": true, "integrity_backups": true,
	"__pycache__": true, "artifacts": true, "context": true,
	"snv": true, "research": true, "economy": true, "config": true,
	"quarantine": true, "lockdown": true, "tasks": true, "integrity": true,
	"capability_lifecycle": true, "prompt": true, "vault": true,
	"evaluation": true, "topology": true, "cache": true, "reports": true,
	"cli": true, "docs": true, "worktrees": true, "governance": true,
	"security": true, "alerts": true, "thavren": true, "memory": true,
	"health": true, "plan": true, "eidren": true, "sofia": true,
	"renata": true, "valeria": true, "dante": true, "uriel": true,
	"elena": true, "projects": true, "company": true,
	"handoffs": true, "red_team": true, "sandbox": true,
	"audit": true, "issues": true, "scripts": true, "sync": true,
	"task": true, "templates": true, "verify": true, "plans": true,
}

// Forbidden patterns in repo root.
var forbiddenRootSuffixes = []string{
	".log", ".tmp", ".bak", ".zip", ".tar.gz", ".pyc",
}

func (a *ArchitectureGuardian) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	repoRoot := resolveRepoRoot(root)

	// 1. Check required directories
	dirsMissing := 0
	dirsFound := 0
	for _, d := range requiredDirs {
		fullPath := filepath.Join(repoRoot, d)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			dirsMissing++
			issues = append(issues, fmt.Sprintf("MISSING_DIR: %s", d))
		} else {
			dirsFound++
		}
	}

	// 2. Check required files
	filesMissing := 0
	filesFound := 0
	for _, f := range requiredFiles {
		fullPath := filepath.Join(repoRoot, f)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			filesMissing++
			issues = append(issues, fmt.Sprintf("MISSING_FILE: %s", f))
		} else {
			filesFound++
		}
	}

	// 3. Check forbidden root patterns
	rootEntries, _ := os.ReadDir(repoRoot)
	for _, e := range rootEntries {
		name := e.Name()
		for _, suffix := range forbiddenRootSuffixes {
			if strings.HasSuffix(name, suffix) {
				issues = append(issues, fmt.Sprintf("FORBIDDEN_ROOT: %s", name))
			}
		}
		if e.Name() == ".DS_Store" {
			issues = append(issues, "FORBIDDEN_ROOT: .DS_Store")
		}
	}

	// 4. Check .ovav/ root for loose files
	ovavRoot := filepath.Join(repoRoot, ".ovav")
	if entries, err := os.ReadDir(ovavRoot); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				if !knownOvavDirs[e.Name()] {
					issues = append(issues, fmt.Sprintf("UNKNOWN_OVAV_DIR: .ovav/%s", e.Name()))
				}
			} else if !e.IsDir() {
				name := e.Name()
				ext := filepath.Ext(name)
				if ext == ".py" || ext == ".json" || ext == ".yaml" || ext == ".yml" {
					if name != "README.md" {
						issues = append(issues, fmt.Sprintf("FORBIDDEN_OVAV_ROOT: .ovav/%s — loose files must be in subdirectories", name))
					}
				}
			}
		}
	}

	// 5. Check tools/ for loose .py files
	toolsDir := filepath.Join(repoRoot, "tools")
	if entries, err := os.ReadDir(toolsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
				if e.Name() != "__init__.py" && e.Name() != "ovav_runtime.py" && e.Name() != "ovav_dashboard.py" {
					issues = append(issues, fmt.Sprintf("LOOSE_TOOL: tools/%s — must be in subdirectory", e.Name()))
				}
			}
		}
	}

	if dirsMissing > 0 || filesMissing > 0 || len(issues) > 0 {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message:  fmt.Sprintf("FAIL architecture guardian — %d/%d dirs, %d/%d files, %d issue(s)", dirsFound, len(requiredDirs), filesFound, len(requiredFiles), len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message:  fmt.Sprintf("PASS architecture guardian — %d dirs, %d files verified", dirsFound, filesFound),
		Duration: time.Since(start),
	}
}

var _ Validator = (*ArchitectureGuardian)(nil)
