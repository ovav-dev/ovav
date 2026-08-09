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
	"fmt"
	"os"
	"path/filepath"

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

	for _, p := range projector.AllProjectors() {
		sourcePath := filepath.Join(root, p.SourceDir())
		deployPath := filepath.Join(root, p.DeployDir())

		sourceInfo, srcErr := os.Stat(sourcePath)
		deployInfo, dstErr := os.Stat(deployPath)

		status := "⚠️  missing"
		if srcErr == nil {
			if dstErr == nil && deployInfo != nil {
				if sourceInfo.ModTime().After(deployInfo.ModTime()) {
					status = "🔄 outdated"
				} else {
					status = "✅ synced"
				}
			} else {
				status = "📝 source only"
			}
		}

		fmt.Printf("  %-12s %s\n", p.Name()+":", status)
	}
	return 0
}
