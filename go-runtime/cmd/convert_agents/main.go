package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/convert"
)

// findRepoRoot walks up from wd looking for the OVAV repo root.
// The OVAV repo root has .ovav/ with service_areas/ AND go-runtime/go.mod.
// This distinguishes it from go-runtime/ which only has .ovav/vault/.
func findRepoRoot(wd string) (string, error) {
	repoRoot := wd
	for {
		// Must have .ovav/ at repo root (distinguishes OVAV repo from subdirs)
		hasGov := false
		if _, err := os.Stat(filepath.Join(repoRoot, ".ovav")); err == nil {
			hasGov = true
		}
		// Must have go-runtime/go.mod (OVAV mono-repo structure)
		hasMod := false
		if _, err := os.Stat(filepath.Join(repoRoot, "go-runtime", "go.mod")); err == nil {
			hasMod = true
		}
		// Must have .ovav/service_areas/ to distinguish OVAV root from go-runtime/ (which has .ovav/vault/)
		hasServiceAreas := false
		if _, err := os.Stat(filepath.Join(repoRoot, ".ovav", "service_areas")); err == nil {
			hasServiceAreas = true
		}
		if hasGov && hasMod && hasServiceAreas {
			return repoRoot, nil
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			return "", fmt.Errorf("could not find repo root")
		}
		repoRoot = parent
	}
}

// cleanOldOutput removes generated .md files from agentDir, preserving ovav.md.
func cleanOldOutput(agentDir string) error {
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "ovav.md" {
			if err := os.Remove(filepath.Join(agentDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// countGenerated counts area-, lead-, and team- prefixed entries.
func countGenerated(entries []fs.DirEntry) (areas, leads, teams int) {
	for _, e := range entries {
		switch {
		case len(e.Name()) > 5 && e.Name()[:5] == "area-":
			areas++
		case len(e.Name()) > 5 && e.Name()[:5] == "lead-":
			leads++
		case len(e.Name()) > 5 && e.Name()[:5] == "team-":
			teams++
		}
	}
	return
}

// generateTarget generates agents for a single CLI target.
func generateTarget(canonicalRoot, outputRoot string, target convert.Target, levelsOverride string) (int, int, int, error) {
	c, err := convert.GetConverter(target)
	if err != nil {
		return 0, 0, 0, err
	}

	agentDir := filepath.Join(outputRoot, c.OutputDir())
	if err := cleanOldOutput(agentDir); err != nil {
		return 0, 0, 0, fmt.Errorf("cleaning old output: %w", err)
	}

	if err := convert.GenerateAllWithFilter(canonicalRoot, target, outputRoot, levelsOverride); err != nil {
		return 0, 0, 0, err
	}

	entries, _ := os.ReadDir(agentDir)
	areas, leads, teams := countGenerated(entries)
	return areas, leads, teams, nil
}

// generateConfigJSON generates a config.json for mimocode
func generateConfigJSON(canonicalRoot, outputRoot string, levelsOverride string) error {
	cc := &convert.MimocodeConfigConverter{
		AreasOnly: levelsOverride == "areas" || levelsOverride == "",
	}

	areas, leads, teams, err := convert.LoadCanonicalAgents(canonicalRoot)
	if err != nil {
		return fmt.Errorf("loading canonical agents: %w", err)
	}

	data, err := cc.GenerateConfigJSON(areas, leads, teams)
	if err != nil {
		return fmt.Errorf("generating config JSON: %w", err)
	}

	// Output to config/mimocode/mimocode.jsonc so the projector can include it
	// in the inject pipeline (config/mimocode/ → ~/.config/mimocode/).
	// Root fix: previous hardcoded output was .mimocode/global_config/config.json
	// which was NOT in the inject path, causing stale JSONC to persist in user home.
	configDir := filepath.Join(outputRoot, "config", "mimocode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config/mimocode dir: %w", err)
	}
	configPath := filepath.Join(configDir, "mimocode.jsonc")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing mimocode.jsonc: %w", err)
	}

	fmt.Printf("  ✅ mimocode.jsonc: %d areas, %d leads, %d teams → config/mimocode/mimocode.jsonc\n",
		len(areas), len(leads), len(teams))
	return nil
}

func main() {
	wd, _ := os.Getwd()
	repoRoot, err := findRepoRoot(wd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}

	canonicalRoot := filepath.Join(repoRoot, "go-runtime", "internal", "agents")
	outputRoot := repoRoot

	fmt.Printf("Converting canonical agents from %s\n", canonicalRoot)
	fmt.Printf("Output root: %s\n", outputRoot)

	// Parse flags: --target <name>, --levels <all|areas>, --format <md|json>
	targetFilter := ""
	levelsOverride := ""
	format := "md"
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target":
			if i+1 < len(args) {
				targetFilter = args[i+1]
				i++
			}
		case "--levels":
			if i+1 < len(args) {
				levelsOverride = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	// Special --format json mode for mimocode config.json generation
	if format == "json" {
		if err := generateConfigJSON(canonicalRoot, outputRoot, levelsOverride); err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
		return
	}

	targets := convert.AvailableTargets()

	for _, target := range targets {
		if targetFilter != "" && string(target) != targetFilter {
			continue
		}

		areas, leads, teams, err := generateTarget(canonicalRoot, outputRoot, target, levelsOverride)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ❌ %s: %v\n", target, err)
			continue
		}

		c, _ := convert.GetConverter(target)
		label := string(target)
		if levelsOverride == "all" {
			label += " (forced all)"
		} else if levelsOverride == "areas" {
			label += " (forced areas)"
		}
		fmt.Printf("  ✅ %s: %d areas, %d leads, %d teams → %s\n",
			label, areas, leads, teams, c.OutputDir())
	}

	fmt.Println("Done.")
}
