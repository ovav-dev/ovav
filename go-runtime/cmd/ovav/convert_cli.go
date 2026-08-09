// convert_cli.go — OVAV Convert Command
//
// ovav convert — Project canonical sources to deploy targets
//
// Canonical source structure:
//   .ovav/source/  — Agent, skill, config, harness, program definitions
//   .ovav/visual/  — Theme, assets (NOT configs)
//
// Usage:
//   ovav convert              Run all projections
//   ovav convert --agents     Project agents only
//   ovav convert --skills     Project skills only
//   ovav convert --configs    Project configs only
//   ovav convert --harnesses Project harnesses only
//   ovav convert --programs   Project programs only
//   ovav convert --visual    Project visual assets only
//   ovav convert --status     Show conversion status
//   ovav convert --list      List all projectors
//   ovav convert -v           Verbose output

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/projector"
)

// cmdConvert handles `ovav convert` — projection from canonical source to deploy.
func cmdConvert(args []string) int {
	verbose := false
	target := ""

	for _, arg := range args {
		switch arg {
		case "-v", "--verbose":
			verbose = true
		case "--agents":
			target = "agents"
		case "--skills":
			target = "skills"
		case "--configs":
			target = "configs"
		case "--harnesses":
			target = "harnesses"
		case "--programs":
			target = "programs"
		case "--visual":
			target = "visual"
		case "--status":
			return printConvertStatus()
		case "--list":
			return listProjectors()
		case "--help", "-h":
			printConvertHelp()
			return 0
		}
	}

	root, err := findOvavRoot()
	if err != nil || root == "" {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root directory\n")
		return 1
	}

	fmt.Println("🔄 OVAV Convert Projection")
	fmt.Println()

	totalFailed := 0
	totalProjected := 0

	projectors := projector.AllProjectors()
	if target != "" {
		// Run specific projector
		p := projector.FindProjector(target)
		if p == nil {
			fmt.Fprintf(os.Stderr, "❌ Unknown projector: %s\n", target)
			return 1
		}
		projectors = []projector.Projector{p}
	}

	for _, p := range projectors {
		count, err := p.Project(root, verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", p.Name(), err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ %s: %d artifacts\n", p.Name(), count)
			totalProjected += count
		}
	}

	fmt.Println()
	if totalFailed > 0 {
		fmt.Printf("❌ Convert completed with %d errors\n", totalFailed)
		return 1
	}
	fmt.Printf("✅ Convert complete — %d artifacts projected\n", totalProjected)
	return 0
}

func printConvertHelp() {
	fmt.Println(`ovav convert — Project canonical sources to deploy targets

Usage:
  ovav convert              Run all projections
  ovav convert --agents     Project agents only
  ovav convert --skills     Project skills only
  ovav convert --configs    Project tool configs (wezterm, fish, git)
  ovav convert --harnesses Project test harnesses
  ovav convert --programs   Project CI/CD programs
  ovav convert --visual    Project visual assets (theme, plugins)
  ovav convert --status     Show conversion status
  ovav convert --list      List all projectors
  ovav convert -v           Verbose output

Canonical Source → Deploy Target:
  .ovav/source/configs/   → config/
  .ovav/source/harnesses/ → go-runtime/internal/testing/harnesses/
  .ovav/source/programs/  → .github/workflows/
  .ovav/source/skills/    → .opencode/skills/
  .ovav/visual/           → .opencode/themes/, .opencode/plugins/`)
}

func listProjectors() int {
	fmt.Println("Available projectors:")
	for _, p := range projector.AllProjectors() {
		fmt.Printf("  %-12s  %s → %s\n", p.Name(), p.SourceDir(), p.DeployDir())
	}
	return 0
}

func printConvertStatus() int {
	root, err := findOvavRoot()
	if err != nil || root == "" {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root directory\n")
		return 1
	}

	fmt.Println("📊 OVAV Convert Status")
	fmt.Println()

	hasPending := false

	for _, p := range projector.AllProjectors() {
		sourcePath := filepath.Join(root, p.SourceDir())
		deployPath := filepath.Join(root, p.DeployDir())

		srcErr := checkDir(sourcePath)

		if srcErr != nil {
			fmt.Printf("  %-12s ⚠️  source missing\n", p.Name()+":")
			continue
		}

		if p.Name() == "configs" {
			// configs: direct copy - use hash comparison to detect exact differences
			diffCount, staleFiles := compareSourceVsDeploy(sourcePath, deployPath)
			if diffCount > 0 {
				hasPending = true
				fmt.Printf("  %-12s 🔄 pending update (%d files)\n", p.Name()+":", diffCount)
				for _, f := range staleFiles {
					fmt.Printf("         · %s\n", f)
				}
			} else {
				fmt.Printf("  %-12s ✅ synced\n", p.Name()+":")
			}
		} else {
			// generators (visual, agents, skills, programs, harnesses):
			// just check if deploy exists and has content
			deployErr := checkDir(deployPath)
			if deployErr != nil || isDirEmpty(deployPath) {
				hasPending = true
				fmt.Printf("  %-12s 📝 needs projection\n", p.Name()+":")
			} else {
				fmt.Printf("  %-12s ✅ projected\n", p.Name()+":")
			}
		}
	}

	fmt.Println()
	if hasPending {
		fmt.Println("Run 'ovav convert' to project pending changes")
	}
	return 0
}

func checkDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func compareSourceVsDeploy(sourceRoot, deployRoot string) (int, []string) {
	var diffCount int
	var staleFiles []string

	filepath.Walk(sourceRoot, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(sourceRoot, srcPath)
		if err != nil {
			return nil
		}

		// Skip top-level files (like model_routing.json) - they are system configs, not tool configs
		if !strings.Contains(rel, string(filepath.Separator)) {
			return nil
		}

		depPath := filepath.Join(deployRoot, rel)

		srcHash, _ := fileHash(srcPath)
		depHash, depErr := fileHash(depPath)

		if depErr != nil || srcHash != depHash {
			diffCount++
			staleFiles = append(staleFiles, rel)
		}
		return nil
	})

	return diffCount, staleFiles
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// isDirEmpty returns true if the directory has no entries (files or subdirs).
func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return true
	}
	return len(entries) == 0
}
