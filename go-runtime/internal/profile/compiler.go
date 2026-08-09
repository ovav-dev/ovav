// Package profile implements the professional profile compiler.
//
// C9.2: Reimplementa tools/cli/ovav_profile.py (670 loc → ~400 loc Go).
// Comandos: list, apply, remove. Lee .ovav/topology/ (canonical source of truth).
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ovav/ovav/internal/cli"
	"gopkg.in/yaml.v3"
)

// ── Profile metadata ────────────────────────────────────────────────────────

// Profile represents a professional specialisation.
type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Lead        string `json:"lead"`
	P0          bool   `json:"p0"`
	Description string `json:"description"`
}

// Profile descriptions (mirrors ovav_profile.py _profile_description).
// Keys are area IDs from .ovav/topology/ — used as fallback when topology
// doesn't provide a lead.description.
var profileDescriptions = map[string]string{
	"area_platform":              "Runtime, seguridad, terminal, CLI y DX de OpenCode",
	"area_research":              "Verificación de fuentes, benchmarks, evidencia y decisiones",
	"area_digital_product":       "Full-stack, arquitectura y desarrollo de producto digital",
	"area_commercial_growth":     "Negocio, monetización, crecimiento y estrategia comercial",
	"area_education_career":      "Aprendizaje personalizado, currículos y formación profesional",
	"area_health_performance":    "Nutrición, fitness, ciencia médica y planes personalizados",
	"area_devops_infrastructure": "CI/CD, cloud, SRE, monitoreo e infraestructura",
	"area_ux_design":             "Design system, UX research, accessibility y prototipado",
}

// p0Areas defines which areas are P0 (primary).
// P0 is a business designation, not topology data.
var p0Areas = map[string]bool{
	"area_platform": true,
	"area_research": true,
}

// ── Dynamic profile loading from topology ───────────────────────────────────

// topologyArea represents the YAML structure in .ovav/topology/area_*.yaml.
type topologyArea struct {
	Area struct {
		Name        string `yaml:"name"`
		ID          string `yaml:"id"`
		Description string `yaml:"description"`
	} `yaml:"area"`
	Lead struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"lead"`
}

// hardcodedFallbackProfiles is used when topology files can't be read.
// Mirrors area IDs from .ovav/topology/ (canonical source).
func hardcodedFallbackProfiles() []Profile {
	return []Profile{
		{ID: "area_platform", Name: "Platform Engineering", Lead: "thavren", P0: true, Description: profileDescriptions["area_platform"]},
		{ID: "area_research", Name: "Research Intelligence", Lead: "eidren", P0: true, Description: profileDescriptions["area_research"]},
		{ID: "area_digital_product", Name: "Web Development", Lead: "dante", P0: false, Description: profileDescriptions["area_digital_product"]},
		{ID: "area_commercial_growth", Name: "Commercial & Growth", Lead: "sofía", P0: false, Description: profileDescriptions["area_commercial_growth"]},
		{ID: "area_devops_infrastructure", Name: "DevOps & Infrastructure", Lead: "uriel", P0: false, Description: profileDescriptions["area_devops_infrastructure"]},
		{ID: "area_education_career", Name: "Education & Training", Lead: "valeria", P0: false, Description: profileDescriptions["area_education_career"]},
		{ID: "area_health_performance", Name: "Sports Science", Lead: "renata", P0: false, Description: profileDescriptions["area_health_performance"]},
		{ID: "area_ux_design", Name: "UI/UX Design", Lead: "elena", P0: false, Description: profileDescriptions["area_ux_design"]},
	}
}

var (
	profilesOnce  sync.Once
	profilesCache []Profile
)

// getProfiles returns the profile registry loaded from topology YAML files.
// Profiles are cached after first load (they don't change during a session).
// Falls back to hardcoded profiles if topology files can't be read.
func getProfiles() []Profile {
	profilesOnce.Do(func() {
		profilesCache = loadProfilesFromTopology()
	})
	return profilesCache
}

// loadProfilesFromTopology scans .ovav/topology/area_*.yaml and builds
// the profile registry dynamically. Falls back to hardcoded if anything fails.
func loadProfilesFromTopology() []Profile {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ ovav profile: repo root not found, using hardcoded profiles (%v)\n", err)
		return hardcodedFallbackProfiles()
	}

	topoDir := filepath.Join(repoRoot, ".ovav", "topology")
	pattern := filepath.Join(topoDir, "area_*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "⚠ ovav profile: no topology files found in %s, using hardcoded profiles\n", topoDir)
		return hardcodedFallbackProfiles()
	}

	profiles := make([]Profile, 0, len(matches))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ ovav profile: can't read %s, skipping (%v)\n", filepath.Base(path), err)
			continue
		}

		var ta topologyArea
		if err := yaml.Unmarshal(data, &ta); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ ovav profile: can't parse %s, skipping (%v)\n", filepath.Base(path), err)
			continue
		}

		if ta.Area.ID == "" || ta.Area.Name == "" {
			fmt.Fprintf(os.Stderr, "⚠ ovav profile: %s missing area.id or area.name, skipping\n", filepath.Base(path))
			continue
		}

		// Build description: lead.description from topology, or fallback to profileDescriptions.
		desc := ta.Lead.Description
		if desc == "" {
			if fallback, ok := profileDescriptions[ta.Area.ID]; ok {
				desc = fallback
			}
		}

		profiles = append(profiles, Profile{
			ID:          ta.Area.ID,
			Name:        ta.Area.Name,
			Lead:        strings.ToLower(ta.Lead.Name),
			P0:          p0Areas[ta.Area.ID],
			Description: desc,
		})
	}

	if len(profiles) == 0 {
		fmt.Fprintf(os.Stderr, "⚠ ovav profile: no valid profiles found in %s, using hardcoded fallback\n", topoDir)
		return hardcodedFallbackProfiles()
	}

	return profiles
}

// ── Command: list ────────────────────────────────────────────────────────────

// CmdList lists all available professional profiles.
func CmdList(args []string) int {
	profiles := getProfiles()
	jsonOutput := cli.HasJSONFlag(args)

	if jsonOutput {
		fmt.Println(`{"profiles": [`)
		for i, p := range profiles {
			comma := ","
			if i == len(profiles)-1 {
				comma = ""
			}
			fmt.Printf(`  {"id":"%s","name":"%s","lead":"%s","p0":%t,"description":"%s"}%s`,
				p.ID, p.Name, p.Lead, p.P0, p.Description, comma)
			fmt.Println()
		}
		fmt.Println(`]}`)
		return 0
	}

	// Human-readable table
	fmt.Println("PROFILE                  NAME                      DESCRIPTION")
	fmt.Println("------------------------------------------------------------------------")
	for _, p := range profiles {
		marker := "  "
		if p.P0 {
			marker = "⭐"
		}
		fmt.Printf("%s %-23s %-25s %s\n", marker, p.ID, p.Name, p.Description)
	}
	fmt.Printf("\n%d profiles available. Apply one with: ovav profile apply <profile>\n", len(profiles))
	return 0
}

// ── Command: apply ───────────────────────────────────────────────────────────

// CmdApply applies a professional profile to the target directory.
func CmdApply(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printApplyHelp()
		return 0
	}

	areaID := ""
	targetDir := "."
	skipConfirm := false
	dryRun := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target":
			if i+1 < len(args) {
				targetDir = args[i+1]
				i++
			}
		case "--yes", "-y":
			skipConfirm = true
		case "--dry-run":
			dryRun = true
		default:
			if areaID == "" {
				areaID = args[i]
			}
		}
	}

	if areaID == "" {
		fmt.Fprintln(os.Stderr, "Usage: ovav profile apply <area>")
		return 2
	}

	// Find profile
	profiles := getProfiles()
	var prof *Profile
	for _, p := range profiles {
		if p.ID == areaID {
			prof = &p
			break
		}
	}
	if prof == nil {
		fmt.Fprintf(os.Stderr, "Profile '%s' not found. Run 'ovav profile list'.\n", areaID)
		return 1
	}

	// Resolve target
	target, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving target: %v\n", err)
		return 1
	}

	// Preview
	fmt.Printf("\n  Profile: %s\n", prof.Name)
	fmt.Printf("  Area:    %s\n", prof.ID)
	fmt.Printf("  Lead:    %s\n", prof.Lead)
	fmt.Printf("  Target:  %s\n", target)
	fmt.Println()
	fmt.Println("  Files to generate:")
	fmt.Printf("    %s\n", filepath.Join(target, "AGENTS.md"))
	fmt.Printf("    %s\n", filepath.Join(target, "opencode.json"))
	fmt.Printf("    %s (squad agents)\n", filepath.Join(target, ".opencode", "agents"))
	fmt.Printf("    %s (area skills)\n", filepath.Join(target, ".opencode", "skills"))

	if dryRun {
		fmt.Println("\n  --dry-run: no files written.")
		return 0
	}

	// Confirm
	if !skipConfirm {
		fmt.Print("\n  Apply profile? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "yes" {
			fmt.Println("  Cancelled.")
			return 0
		}
	}

	// Generate files
	if err := generateProfile(target, prof); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying profile: %v\n", err)
		return 1
	}

	fmt.Printf("\n  ✅ Profile '%s' applied to %s\n", prof.Name, target)
	fmt.Println("  Next: run 'opencode' in this directory.")
	return 0
}

// ── Command: remove ──────────────────────────────────────────────────────────

// CmdRemove removes a professional profile from the target directory.
func CmdRemove(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printRemoveHelp()
		return 0
	}

	areaID := ""
	targetDir := "."
	skipConfirm := false
	dryRun := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target":
			if i+1 < len(args) {
				targetDir = args[i+1]
				i++
			}
		case "--yes", "-y":
			skipConfirm = true
		case "--dry-run":
			dryRun = true
		default:
			if areaID == "" {
				areaID = args[i]
			}
		}
	}

	if areaID == "" {
		fmt.Fprintln(os.Stderr, "Error: area argument required.")
		return 2
	}

	// Find profile
	profiles := getProfiles()
	var prof *Profile
	for _, p := range profiles {
		if p.ID == areaID {
			prof = &p
			break
		}
	}
	if prof == nil {
		fmt.Fprintf(os.Stderr, "Profile '%s' not found.\n", areaID)
		return 1
	}

	target, _ := filepath.Abs(targetDir)

	// Check what exists
	agentsMD := filepath.Join(target, "AGENTS.md")
	opencodeJSON := filepath.Join(target, "opencode.json")
	agentsDir := filepath.Join(target, ".opencode", "agents")
	skillsDir := filepath.Join(target, ".opencode", "skills")

	toRemove := []struct {
		label string
		path  string
		isDir bool
	}{}

	if _, err := os.Stat(agentsMD); err == nil {
		toRemove = append(toRemove, struct {
			label string
			path  string
			isDir bool
		}{"AGENTS.md", agentsMD, false})
	}
	if _, err := os.Stat(opencodeJSON); err == nil {
		toRemove = append(toRemove, struct {
			label string
			path  string
			isDir bool
		}{"opencode.json", opencodeJSON, false})
	}
	if info, err := os.Stat(agentsDir); err == nil && info.IsDir() {
		toRemove = append(toRemove, struct {
			label string
			path  string
			isDir bool
		}{".opencode/agents/", agentsDir, true})
	}
	if info, err := os.Stat(skillsDir); err == nil && info.IsDir() {
		toRemove = append(toRemove, struct {
			label string
			path  string
			isDir bool
		}{".opencode/skills/", skillsDir, true})
	}

	if len(toRemove) == 0 {
		fmt.Printf("No profile files found in %s.\n", target)
		return 0
	}

	fmt.Printf("\n  Profile: %s\n", prof.Name)
	fmt.Printf("  Target:  %s\n", target)
	fmt.Println("\n  Files to remove:")
	for _, item := range toRemove {
		fmt.Printf("    %s\n", item.label)
	}

	if dryRun {
		fmt.Println("\n  --dry-run: no files removed.")
		return 0
	}

	if !skipConfirm {
		fmt.Print("\n  Remove these files? [y/N] ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "yes" {
			fmt.Println("  Cancelled.")
			return 0
		}
	}

	removed := 0
	for _, item := range toRemove {
		var err error
		if item.isDir {
			err = os.RemoveAll(item.path)
		} else {
			err = os.Remove(item.path)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error removing %s: %v\n", item.label, err)
		} else {
			removed++
		}
	}

	fmt.Printf("\n  Profile '%s' removed. %d item(s) cleaned up.\n", prof.Name, removed)
	return 0
}

// ── File generation ──────────────────────────────────────────────────────────

func generateProfile(target string, prof *Profile) error {
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	// AGENTS.md
	agentsMD := fmt.Sprintf(`# %s — Professional Profile

<!-- Generated by OVAV profile apply %s -->
<!-- OVAV Go Runtime — CAPA 9 -->

This directory is configured for **%s** professional work.
Lead: **%s**. Area: **%s**.

## Quick Start
- Terminal: run `+"`opencode`"+` in this directory
- Desktop: open OpenCode Desktop and point to this directory

---
# %s
Lead: %s
Area: %s
`, prof.Name, prof.ID, prof.Name, prof.Lead, prof.ID, prof.Name, prof.Lead, prof.ID)

	if err := os.WriteFile(filepath.Join(target, "AGENTS.md"), []byte(agentsMD), 0644); err != nil {
		return err
	}
	fmt.Println("  ✓ AGENTS.md")

	// opencode.json
	opencodeJSON := fmt.Sprintf(`{
  "agents": {},
  "skills": {
    "paths": [".opencode/skills"]
  },
  "profile": {
    "area": "%s",
    "name": "%s",
    "lead": "%s",
    "applied_by": "ovav"
  }
}
`, prof.ID, prof.Name, prof.Lead)
	if err := os.WriteFile(filepath.Join(target, "opencode.json"), []byte(opencodeJSON), 0644); err != nil {
		return err
	}
	fmt.Println("  ✓ opencode.json")

	// Create .opencode directories
	agentsDir := filepath.Join(target, ".opencode", "agents")
	skillsDir := filepath.Join(target, ".opencode", "skills")
	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(skillsDir, 0755)

	fmt.Printf("  ✓ .opencode/agents/ (created)\n")
	fmt.Printf("  ✓ .opencode/skills/ (created)\n")

	return nil
}

// ── Help ─────────────────────────────────────────────────────────────────────

// PrintHelp prints usage for ovav profile.
func PrintHelp() {
	fmt.Println(`ovav profile — Professional Profile Manager

Commands:
  ovav profile list               List available profiles
  ovav profile apply <area>       Apply profile to current directory
  ovav profile remove <area>      Remove profile from current directory

Options:
  --target <dir>   Target directory (default: current)
  --yes, -y        Skip confirmation
  --dry-run        Preview without writing

Examples:
  ovav profile list
  ovav profile apply area_platform
  ovav profile apply area_health_performance --target ./my-project
  ovav profile remove area_digital_product --dry-run`)
}

func printApplyHelp() {
	fmt.Println(`Usage: ovav profile apply <area> [options]

  Apply a professional profile to the current directory.
  Generates AGENTS.md + opencode.json + .opencode/agents/ + .opencode/skills/

Options:
  --target <dir>   Target directory (default: current directory)
  --yes, -y        Skip confirmation prompt
  --dry-run        Preview files without writing

Examples:
  ovav profile apply area_platform
  ovav profile apply area_health_performance --target ./mi-proyecto
  ovav profile apply area_research --yes
  ovav profile apply area_digital_product --dry-run`)
}

func printRemoveHelp() {
	fmt.Println(`Usage: ovav profile remove <area> [options]

  Remove a previously applied profile. Cleans up:
    AGENTS.md, opencode.json, .opencode/agents/, .opencode/skills/

Options:
  --target <dir>   Target directory (default: current directory)
  --yes, -y        Skip confirmation prompt
  --dry-run        Preview files to remove without deleting

Examples:
  ovav profile remove area_health_performance
  ovav profile remove area_research --dry-run
  ovav profile remove area_digital_product --yes`)
}
