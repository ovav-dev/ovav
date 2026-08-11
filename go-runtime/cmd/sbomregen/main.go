package main

import (
	"fmt"
	"log"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/sbom"
)

func main() {
	root, err := cli.FindRepoRoot()
	if err != nil {
		log.Fatalf("Failed to find repo root: %v", err)
	}
	sb, err := sbom.Generate(root)
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}
	if err := sb.Save(root); err != nil {
		log.Fatalf("Save failed: %v", err)
	}
	fmt.Printf("SBOM regenerated: %d files, %d Go deps\n", len(sb.CoreFiles), len(sb.Dependencies.Go))
}
