// Package config provides OVAV configuration display.
//
// C9.x: Show OVAV configuration sourced from authority contracts.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ovav/ovav/internal/cli"
)

// Show displays OVAV configuration.
func Show(args []string) int {
	jsonOutput := cli.HasJSONFlag(args)
	repoRoot := cli.MustFindRepoRoot()

	if jsonOutput {
		fmt.Printf(`{"status":"ok","command":"config","go_runtime":true,"go_version":"%s","os":"%s","arch":"%s"}`+"\n",
			runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return 0
	}

	version := "0.0.0-dev" // injected at build time
	fmt.Println("OVAV Config — Go Runtime")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("   Go version:   %s\n", runtime.Version())
	fmt.Printf("   OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("   Version:      %s\n", version)
	fmt.Println()

	if os.Getenv("OVAV_DEV") == "1" {
		fmt.Println("   Configuration sources:")
		ac := filepath.Join(repoRoot, ".ovav", "service_areas", "shared", "current_authority_contract.yaml")
		pa := filepath.Join(repoRoot, ".ovav", "policy", "permission_authority.json")

		if _, err := os.Stat(ac); err == nil {
			fmt.Println("   ✅ Authority contract")
		} else {
			fmt.Println("   ❌ Authority contract (missing)")
		}
		if _, err := os.Stat(pa); err == nil {
			fmt.Println("   ✅ Permission authority")
		} else {
			fmt.Println("   ❌ Permission authority (missing)")
		}
	}
	fmt.Println()
	fmt.Println("   Configuración gestionada por OVAV Go Runtime.")
	return 0
}
