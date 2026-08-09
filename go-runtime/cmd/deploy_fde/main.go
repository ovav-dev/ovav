// deploy_fde loads a lead's FDE Brain Pack from service area YAMLs
// and prints a summary. Used to verify brain integrity.
//
// Usage:
//
//	go run -C go-runtime ./cmd/deploy_fde <lead_id>
//
// Example:
//
//	go run -C go-runtime ./cmd/deploy_fde thavren
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/fde"
)

// areaForLead maps a lead ID to its service area directory name.
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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: deploy_fde <lead_id>")
		fmt.Fprintln(os.Stderr, "  lead_id: thavren | eidren | valeria | dante | renata | sofia | elena | uriel | kenji | camila")
		os.Exit(1)
	}

	leadID := strings.ToLower(os.Args[1])
	areaID, ok := areaForLead[leadID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown lead: %s\n", leadID)
		os.Exit(1)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	pack, err := fde.LoadBrainPack(repoRoot, areaID, leadID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading brain: %v\n", err)
		os.Exit(1)
	}

	// Print JSON summary for verification
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(pack)
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
		_, agentsErr := os.Stat(filepath.Join(root, "ovav", "agents", "areas"))
		if ovavErr == nil && modErr == nil && agentsErr == nil {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", fmt.Errorf("could not find repo root from %s", wd)
		}
		root = parent
	}
}
