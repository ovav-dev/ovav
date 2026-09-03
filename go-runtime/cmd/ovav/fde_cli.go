// fde_cli.go — OVAV FDE Brain Pack Verification
// Loads and verifies FDE (Flattened DELite) Brain Packs from service areas.
// Usage: ovav fde <lead_id>

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/fde"
)

// areaForLead maps lead IDs to their service area directory names.
var areaForLead = map[string]string{
	"thavren": "platform_engineering",
	"eidren":  "research_intelligence",
	"valeria": "education_career",
	"dante":   "digital_product",
	"renata":  "health_performance",
	"sofia":   "commercial_growth",
	"elena":   "ux_design",
	"uriel":   "devops_infrastructure",
	"kenji":   "adversarial_intelligence",
	"camila":  "legal_compliance",
}

func cmdFDE(args []string) int {
	if len(args) < 1 {
		return printFDEHelp()
	}

	leadID := strings.ToLower(args[0])
	jsonOut := false
	for _, a := range args[1:] {
		if a == "--json" || a == "-j" {
			jsonOut = true
		}
	}

	areaID, ok := areaForLead[leadID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown lead %q\n", leadID)
		fmt.Fprintln(os.Stderr, "Valid leads: thavren, eidren, valeria, dante, renata, sofia, elena, uriel, kenji, camila")
		return 1
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	pack, err := fde.LoadBrainPack(repoRoot, areaID, leadID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading brain pack: %v\n", err)
		return 1
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(pack)
		return 0
	}

	// Human-readable output
	fmt.Printf("🧠 FDE Brain Pack: %s (%s)\n", leadID, areaID)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("   Area: %s\n", pack.Area)
	fmt.Printf("   Loaded from: %s\n", pack.LoadedFrom)
	if pack.SelfModel != nil {
		fmt.Printf("   SelfModel: present\n")
	}
	if pack.Criteria != nil {
		fmt.Printf("   Criteria: present\n")
	}
	if pack.OpLevel != nil {
		fmt.Printf("   OperatingLevel: present\n")
	}
	fmt.Println()
	fmt.Printf("✅ Brain pack verified for %s\n", leadID)
	return 0
}

func printFDEHelp() int {
	fmt.Println(`ovav fde — FDE Brain Pack Verification

Load and verify a lead's Flattened DELite (FDE) Brain Pack.

Usage:
  ovav fde <lead_id>         Show brain pack summary
  ovav fde <lead_id> --json  Output JSON format

Valid leads:
  thavren  eidren  valeria  dante  renata
  sofia    elena   uriel    kenji  camila

Examples:
  ovav fde thavren
  ovav fde eidren --json`)
	return 2
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := wd
	for {
		_, ovavErr := os.Stat(filepath.Join(root, ".ovav"))
		_, modErr := os.Stat(filepath.Join(root, "go-runtime", "go.mod"))
		if ovavErr == nil && modErr == nil {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", fmt.Errorf("could not find repo root from %s", wd)
		}
		root = parent
	}
}
