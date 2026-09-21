package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ovav/ovav/internal/product"
)

func cmdProduct(args []string) int {
	if len(args) == 0 {
		printProductUsage()
		return 0
	}

	switch args[0] {
	case "install":
		return productInstall(args[1:])
	case "launch":
		return productLaunch()
	case "bootstrap":
		return productBootstrap()
	case "cockpit":
		return productCockpit(args[1:])
	case "verify":
		return productVerify()
	case "uninstall":
		return productUninstall()
	case "status":
		return productStatus()
	case "--help", "-h", "help":
		printProductUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown: %s\nRun 'ovav product help'\n", args[0])
		return 2
	}
}

func productInstall(args []string) int {
	ovavRoot, err := findOvavRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	mode := "install"
	for _, a := range args {
		if a == "--dry-run" || a == "-n" {
			mode = "dry-run"
		}
	}

	result, err := product.ProductInstall(ovavRoot, mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if result.Preview != "" {
		fmt.Print(result.Preview)
	}

	if mode == "install" {
		if len(result.Errors) > 0 {
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "  ⚠ %s\n", e)
			}
		}
		fmt.Printf("✅ OVAV Product installed: %d files + %d symlinks\n", result.FilesCopied, result.LinksCreated)
		fmt.Printf("   %s\n", result.ProductDir)
	} else {
		fmt.Printf("\n📊 %d files + %d symlinks would be installed\n", result.FilesCopied, result.LinksCreated)
	}
	return 0
}

func productLaunch() int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	productDir, _ := product.ProductDir()
	if _, err := os.Stat(filepath.Join(productDir, ".ovav-manifest.json")); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV Product not installed. Run: ovav product install\n")
		return 1
	}

	fmt.Printf("🔧 Preparing %s...\n", cwd)
	result, err := product.Bootstrap(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	parts := []string{}
	if result.AgentsLinked {
		parts = append(parts, "agents")
	}
	if result.SkillsLinked {
		parts = append(parts, "skills")
	}
	if result.IdentityCopied {
		parts = append(parts, "identity")
	}
	fmt.Printf("   ✅ %s ready\n", strings.Join(parts, ", "))

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  ⚠ %s\n", e)
		}
	}

	bin := findMimo()
	if bin == "" {
		fmt.Println("\n⚠️  mimo not found. Install: npm i -g @anthropic-ai/mimocode")
		return 0
	}

	fmt.Printf("🚀 %s\n", bin)
	cmd := exec.Command(bin)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return 0
}

func productBootstrap() int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	result, err := product.Bootstrap(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("✅ %s ready for OVAV Product\n", cwd)
	fmt.Printf("   agents=%v skills=%v identity=%v\n", result.AgentsLinked, result.SkillsLinked, result.IdentityCopied)
	return 0
}

func productCockpit(args []string) int {
	url := ""
	for i, a := range args {
		if a == "--url" && i+1 < len(args) {
			url = args[i+1]
		}
	}

	// Try installed binary first (~/.local/share/ovav/product-cockpit)
	home, _ := os.UserHomeDir()
	cockpitBin := filepath.Join(home, ".local", "share", "ovav", "product-cockpit")
	if _, err := os.Stat(cockpitBin); os.IsNotExist(err) {
		// Fallback: look in same directory as ovav binary
		if exe, err := os.Executable(); err == nil {
			cockpitBin = filepath.Join(filepath.Dir(exe), "product-cockpit")
		}
	}

	cmdArgs := []string{}
	if url != "" {
		cmdArgs = append(cmdArgs, "--url", url)
	}

	cmd := exec.Command(cockpitBin, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "product cockpit: %v\n", err)
		return 1
	}
	return 0
}

func productVerify() int {
	result, err := product.ProductInstall(".", "verify")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(result.Errors) == 0 {
		fmt.Printf("✅ %d files + %d symlinks verified\n", result.FilesCopied, result.LinksCreated)
	} else {
		fmt.Fprintf(os.Stderr, "❌ %d issues:\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  • %s\n", e)
		}
		return 1
	}
	return 0
}

func productUninstall() int {
	result, err := product.ProductInstall(".", "uninstall")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("🗑️  Removed %d files + %d symlinks\n", result.FilesCopied, result.LinksCreated)
	return 0
}

func productStatus() int {
	manifest, err := product.LoadManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	productDir, _ := product.ProductDir()

	if manifest == nil {
		fmt.Printf("❌ Not installed\n   Run: ovav product install\n")
		return 0
	}

	fmt.Printf("✅ OVAV Product v%s\n", manifest.Version)
	fmt.Printf("   Installed: %s\n", manifest.Installed)
	fmt.Printf("   Source:    %s\n", manifest.OvavRoot)
	fmt.Printf("   Location:  %s\n", productDir)
	fmt.Printf("   Files:     %d\n", len(manifest.Entries))
	fmt.Printf("   Symlinks:  %d\n", len(manifest.Symlinks))
	return 0
}

func printProductUsage() {
	fmt.Println(`OVAV Product — AI Workstation for Developers

Usage: ovav product <command>

Commands:
  install [--dry-run]   Install OVAV Product to ~/.local/share/ovav/
  launch                Bootstrap CWD + launch mimo
  bootstrap             Prepare CWD for OVAV (no launch)
  cockpit [--url URL]   Product update cockpit — check & apply updates
  verify                Check installation integrity
  uninstall             Remove OVAV Product
  status                Show installation info

Quick start:
  ovav product install       # one-time install
  cd ~/any-project
  ovav product cockpit       # check for updates + apply
  ovav product launch        # bootstrap + open mimo with OVAV agents`)
}

func findOvavRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		// Must have .ovav/ directory
		hasGov := false
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			hasGov = true
		}
		// Must have go-runtime/go.mod (OVAV mono-repo structure)
		hasMod := false
		if _, err := os.Stat(filepath.Join(dir, "go-runtime", "go.mod")); err == nil {
			hasMod = true
		}
		// Must have .ovav/service_areas/ to distinguish OVAV root from go-runtime/ (which has .ovav/vault/)
		hasServiceAreas := false
		if _, err := os.Stat(filepath.Join(dir, ".ovav", "service_areas")); err == nil {
			hasServiceAreas = true
		}
		if hasGov && hasMod && hasServiceAreas {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("not in an OVAV Systems directory")
}

func findMimo() string {
	// Allow env override for testing or CI environments
	if p := os.Getenv("OVAV_MIMO_BIN"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	home, _ := os.UserHomeDir()
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(home, "AppData", "Roaming", "npm", "mimo.cmd"),
		}
	} else {
		candidates = []string{
			filepath.Join(home, ".mimocode", "bin", "mimo"),
			"/usr/local/bin/mimo",
			"/usr/bin/mimo",
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("mimo"); err == nil {
		return p
	}
	return ""
}
