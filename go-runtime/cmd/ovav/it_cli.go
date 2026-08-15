package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// cmdIT dispatches ovav it subcommands.
//
// Usage: ovav it <subcommand>
// Subcommands:
//
//	reload            — trigger IT reload via Win32 broadcast (ADR-010)
//	reload --force    — restart IT process if broadcast fails
//	status            — check if IT process is running
//	pid               — print IT process ID (for scripting)
//	logs              — tail IT logs (if accessible)
func cmdIT(args []string) int {
	if len(args) == 0 {
		return runITReload(args) // default to reload
	}
	switch args[0] {
	case "reload":
		return runITReload(args[1:])
	case "status":
		return runITStatus(args[1:])
	case "pid":
		return runITPid(args[1:])
	case "logs":
		return runITLogs(args[1:])
	case "help", "--help", "-h":
		printITHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV it: unknown subcommand %q\n", args[0])
		printITHelp()
		return 2
	}
}

func printITHelp() {
	fmt.Println(`OVAV it — Intelligent Terminal (v0.1.4+) operations

Usage:
  ovav it reload [--force]   # trigger IT reload (ADR-010)
  ovav it status             # check if IT is running
  ovav it pid                # print IT process ID
  ovav it logs               # tail IT logs

Reload methods (in order):
1. WM_SETTINGCHANGE broadcast (preferred — silent)
2. WMI process restart (fallback)
3. Operator notification (final fallback)

Bypass IT restart: --no-reload flag on 'ovav deploy run'`)
}

// runITReload triggers an IT reload via PowerShell + Win32 API.
// Per ADR-010: 3-method approach.
func runITReload(args []string) int {
	force := false
	noReload := false
	for _, a := range args {
		switch a {
		case "--force":
			force = true
		case "--no-reload":
			noReload = true
		case "--help", "-h":
			printITHelp()
			return 0
		}
	}
	if noReload {
		fmt.Println("OVAV it reload: skipped (--no-reload)")
		return 0
	}

	// Step 0: Check if IT is running
	if !isITRunning() {
		fmt.Println("⚠️  IT not running — no reload needed")
		return 0
	}

	// Method 1: WM_SETTINGCHANGE broadcast
	fmt.Println("→ Method 1: WM_SETTINGCHANGE broadcast")
	scriptPath, err := findITReloadScript()
	if err != nil {
		fmt.Printf("   ⚠️  Script not found: %v\n", err)
	} else {
		if err := runPowerShell(scriptPath); err == nil {
			fmt.Println("   ✅ Broadcast sent successfully")
			// Verify with health check
			if err := verifyITHealth(); err != nil {
				fmt.Printf("   ⚠️  Health check: %v\n", err)
			} else {
				fmt.Println("   ✅ Health check passed")
			}
			return 0
		} else {
			fmt.Printf("   ⚠️  Broadcast failed: %v\n", err)
			if !force {
				fmt.Println("   Use --force to escalate to process restart")
				return 1
			}
		}
	}

	// Method 2: WMI process restart
	if force {
		fmt.Println("→ Method 2: Process restart (--force)")
		if err := restartITProcess(); err != nil {
			fmt.Printf("   ❌ Restart failed: %v\n", err)
			// Method 3: operator notification
			fmt.Println("→ Method 3: Operator notification (fallback)")
			fmt.Println("   Please restart Intelligent Terminal manually:")
			fmt.Println("   1. Close all IT windows")
			fmt.Println("   2. Reopen IT")
			fmt.Println("   3. Verify keybindings work")
			return 2
		}
		fmt.Println("   ✅ IT restarted")
		return 0
	}

	return 1
}

// findITReloadScript locates the PowerShell reload script.
func findITReloadScript() (string, error) {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		return "", err
	}
	// Look in workstation/scripts/
	candidates := []string{
		filepath.Join(root, "workstation", "scripts", "it-reload.ps1"),
		filepath.Join(root, "workstation", "scripts", "it_reload.ps1"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("it-reload.ps1 not found")
}

// runPowerShell executes a PowerShell script via WSL bridge.
func runPowerShell(scriptPath string) error {
	// Convert WSL path to Windows path
	winPath, err := wslToWindows(scriptPath)
	if err != nil {
		return err
	}
	// Run powershell.exe
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-ExecutionPolicy", "Bypass", "-File", winPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// wslToWindows converts a WSL path to Windows path (for PowerShell).
func wslToWindows(wslPath string) (string, error) {
	// If already a Windows path, return as-is
	if len(wslPath) >= 3 && wslPath[1] == ':' {
		return wslPath, nil
	}
	// Convert /home/braka/... → \\wsl$\Ubuntu\home\braka\...
	// Or use wslpath command for proper conversion
	cmd := exec.Command("wslpath", "-w", wslPath)
	out, err := cmd.Output()
	if err != nil {
		// Fallback: manual conversion for /mnt/c paths
		if strings.HasPrefix(wslPath, "/mnt/c/") {
			return "C:\\" + strings.ReplaceAll(wslPath[6:], "/", "\\"), nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// isITRunning checks if the IntelligentTerminal process is running.
func isITRunning() bool {
	// Use tasklist.exe (Windows) to check
	cmd := exec.Command("tasklist.exe", "/FI", "IMAGENAME eq Microsoft.IntelligentTerminal.exe")
	out, err := cmd.Output()
	if err != nil {
		// Fallback: check via wmic or PowerShell
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "intelligentterminal")
}

// verifyITHealth checks that IT picked up new settings.
// Compares the live settings.json hash to the fragment hash.
func verifyITHealth() error {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		return err
	}
	// Build drift report — should show no drift after reload
	report, err := buildDriftReport(root, "it-keybindings")
	if err != nil {
		return err
	}
	if report.DriftedTargets > 0 {
		return fmt.Errorf("%d drift items remain", report.TotalItems)
	}
	return nil
}

// restartITProcess stops and restarts the IT process.
func restartITProcess() error {
	// Stop
	stop := exec.Command("taskkill.exe", "/IM", "Microsoft.IntelligentTerminal.exe", "/F")
	if err := stop.Run(); err != nil {
		// Continue anyway — process may not be running
		fmt.Printf("   (taskkill: %v — continuing)\n", err)
	}
	time.Sleep(1 * time.Second)
	// Start
	launch := exec.Command("explorer.exe", "shell:AppsFolder\\Microsoft.IntelligentTerminal_8wekyb3d8bbwe!App")
	if err := launch.Start(); err != nil {
		return fmt.Errorf("launch: %w", err)
	}
	return nil
}

// runITStatus checks IT process status.
func runITStatus(args []string) int {
	running := isITRunning()
	if running {
		fmt.Println("✅ Intelligent Terminal is running")
	} else {
		fmt.Println("❌ Intelligent Terminal is NOT running")
		return 1
	}
	return 0
}

// runITPid prints the IT process ID.
func runITPid(args []string) int {
	cmd := exec.Command("powershell.exe", "-Command",
		"Get-Process Microsoft.IntelligentTerminal -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Id")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	fmt.Print(strings.TrimSpace(string(out)))
	return 0
}

// runITLogs tails IT logs.
func runITLogs(args []string) int {
	// IT logs are in AppData/Local/Packages/.../LocalCache/Local/IntelligentTerminal/
	logPath := "/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalCache/Local/IntelligentTerminal/"
	if _, err := os.Stat(logPath); err != nil {
		fmt.Printf("IT logs not found at %s\n", logPath)
		return 1
	}
	// Find newest log file
	entries, err := os.ReadDir(logPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	var newest string
	var newestTime time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = filepath.Join(logPath, e.Name())
		}
	}
	if newest == "" {
		fmt.Println("No log files found")
		return 0
	}
	fmt.Printf("Tailing %s (Ctrl+C to exit):\n\n", newest)
	// tail -f equivalent (simplified — just dump current content)
	data, err := os.ReadFile(newest)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	return 0
}
