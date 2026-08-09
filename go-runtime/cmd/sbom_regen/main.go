package main

import (
	"fmt"
	"github.com/ovav/ovav/internal/sbom"
	"os"
	"path/filepath"
)

func main() {
	// Default to current working directory
	root := "."
	
	// If argument provided, use it
	if len(os.Args) > 1 {
		root = os.Args[1]
	} else {
		// Try to detect OVAV project root by looking for .ovav/ and go-runtime/
		if absRoot, err := detectOVAVRoot(); err == nil {
			root = absRoot
		}
	}
	
	// Convert to absolute path for reliable relative path resolution
	if absRoot, err := filepath.Abs(root); err == nil {
		root = absRoot
	}
	
	s, err := sbom.Generate(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating SBOM: %v\n", err)
		os.Exit(1)
	}
	if err := s.Save(root); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving SBOM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("SBOM regenerated: %d files tracked\n", len(s.CoreFiles))
}

// detectOVAVRoot finds the OVAV project root by looking for .ovav/ and go-runtime/
func detectOVAVRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	// Walk up looking for .ovav/ and go-runtime/
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go-runtime")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	
	return "", fmt.Errorf("OVAV project root not found")
}
