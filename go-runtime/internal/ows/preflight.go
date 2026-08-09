// Package ows implements the OVAV Worktree Orchestration System.
package ows

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ToolCheck represents a single tool and its detection result.
type ToolCheck struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Found       bool   `json:"found"`
	Version     string `json:"version,omitempty"`
	InstallHint string `json:"install_hint,omitempty"`
}

// PreflightResult holds the complete environment diagnostic.
type PreflightResult struct {
	OS          OSType      `json:"os"`
	Arch        string      `json:"arch"`
	GoVersion   string      `json:"go_version"`
	GitVersion  string      `json:"git_version,omitempty"`
	Tools       []ToolCheck `json:"tools"`
	AllRequired bool        `json:"all_required_ok"`
	AnyMissing  bool        `json:"any_missing"`
}

// OSType represents the detected operating system.
type OSType string

const (
	OSLinux   OSType = "linux"
	OSmacOS   OSType = "darwin"
	OSWindows OSType = "windows"
	OSUnknown OSType = "unknown"
)

// OSDisplayName returns a human-readable OS name.
func OSDisplayName(o OSType) string {
	switch o {
	case OSLinux:
		return "Linux"
	case OSmacOS:
		return "macOS"
	case OSWindows:
		return "Windows"
	default:
		return "Unknown OS"
	}
}

// DetectOS returns the current operating system.
func DetectOS() OSType {
	switch runtime.GOOS {
	case "linux":
		return OSLinux
	case "darwin":
		return OSmacOS
	case "windows":
		return OSWindows
	default:
		return OSUnknown
	}
}

// detectTool checks if a binary exists in PATH and returns its version string.
func detectTool(name string) (found bool, version string) {
	_, err := exec.LookPath(name)
	if err != nil {
		return false, ""
	}

	// Try --version first (works for git, go, gpg, gcc, gitleaks, semgrep)
	cmd := exec.Command(name, "--version")
	out, err := cmd.Output()
	if err != nil {
		// Try -v for gcc
		cmd = exec.Command(name, "-v")
		out, err = cmd.Output()
		if err != nil {
			return true, "" // exists but couldn't get version
		}
	}

	// Grab first line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	version = strings.TrimSpace(lines[0])

	// Normalize gitleaks output: "gitleaks version 8.x.x" → "8.x.x"
	if name == "gitleaks" && strings.Contains(version, "version") {
		parts := strings.Split(version, "version")
		if len(parts) == 2 {
			version = strings.TrimSpace(parts[1])
		}
	}

	return true, version
}

// RunPreflight executes the full environment diagnostic and returns a result.
func RunPreflight() *PreflightResult {
	osType := DetectOS()
	arch := runtime.GOARCH
	goVersion := runtime.Version()

	// Try to get git version via git command
	gitVersion := ""
	if out, err := exec.Command("git", "--version").Output(); err == nil {
		gitVersion = strings.TrimSpace(string(out))
	}

	result := &PreflightResult{
		OS:         osType,
		Arch:       arch,
		GoVersion:  goVersion,
		GitVersion: gitVersion,
		Tools:      []ToolCheck{},
	}

	// Tool definitions with OS-specific install hints
	type toolDef struct {
		name     string
		required bool
		hintLin  string
		hintMac  string
		hintWin  string
	}

	tools := []toolDef{
		{
			name:     "git",
			required: true,
			hintLin:  "sudo apt install git        # Debian/Ubuntu\nsudo dnf install git        # Fedora/RHEL\nsudo pacman -S git          # Arch/Manjaro",
			hintMac:  "brew install git",
			hintWin:  "winget install Git.Git     # Windows (winget)\nchoco install git           # Windows (Chocolatey)",
		},
		{
			name:     "go",
			required: true,
			hintLin:  "sudo apt install golang-go  # Debian/Ubuntu\nsudo dnf install golang      # Fedora/RHEL\nhttps://go.dev/dl/           # Official binaries",
			hintMac:  "brew install go",
			hintWin:  "winget install GoLang.Go   # Windows (winget)\nchoco install golang         # Windows (Chocolatey)",
		},
		{
			name:     "gpg",
			required: false,
			hintLin:  "sudo apt install gnupg       # Debian/Ubuntu\nsudo dnf install gnupg2       # Fedora/RHEL",
			hintMac:  "brew install gnupg",
			hintWin:  "winget install GnuPG.GnuPG # Windows (winget)",
		},
		{
			name:     "gcc",
			required: false,
			hintLin:  "sudo apt install build-essential  # Debian/Ubuntu\nsudo dnf install gcc               # Fedora/RHEL",
			hintMac:  "brew install gcc",
			hintWin:  "choco install mingw            # Windows (Chocolatey)",
		},
		{
			name:     "gitleaks",
			required: false,
			hintLin:  "brew install gitleaks       # macOS/Linux\ncurl -s https://git.io/JL4Gw | bash  # Linux",
			hintMac:  "brew install gitleaks",
			hintWin:  "winget install gitleaks     # Windows (winget)",
		},
		{
			name:     "semgrep",
			required: false,
			hintLin:  "pip install semgrep  # or: curl -L https://semgrep.dev/install | bash",
			hintMac:  "pip install semgrep",
			hintWin:  "pip install semgrep",
		},
	}

	for _, t := range tools {
		found, version := detectTool(t.name)
		var hint string
		switch osType {
		case OSLinux:
			hint = t.hintLin
		case OSmacOS:
			hint = t.hintMac
		case OSWindows:
			hint = t.hintWin
		default:
			hint = "Install " + t.name + " and ensure it is in your PATH"
		}
		result.Tools = append(result.Tools, ToolCheck{
			Name:        t.name,
			Required:    t.required,
			Found:       found,
			Version:     version,
			InstallHint: hint,
		})
		if !found && t.required {
			result.AnyMissing = true
		}
	}

	result.AllRequired = !result.AnyMissing
	return result
}

// DisplayPreflight prints the pre-flight report in human-readable format.
func DisplayPreflight(pr *PreflightResult) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  OWS Environment  ─  %-10s %-6s                        ║\n",
		OSDisplayName(pr.OS), pr.Arch)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  %-20s  %-36s ║\n", "Go", pr.GoVersion)
	fmt.Printf("║  %-20s  %-36s ║\n", "Git", pr.GitVersion)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	for _, tool := range pr.Tools {
		if tool.Required {
			if tool.Found {
				versionStr := tool.Version
				if versionStr == "" {
					versionStr = "✓"
				}
				fmt.Printf("║  \033[1m%-20s\033[0m  \033[32m✅ installed\033[0m  %-23s ║\n",
					tool.Name, versionStr)
			} else {
				fmt.Printf("║  \033[1m%-20s\033[0m  \033[31m❌ MISSING\033[0m  %-23s ║\n",
					tool.Name, "")
			}
		} else {
			if tool.Found {
				versionStr := tool.Version
				if versionStr == "" {
					versionStr = "✓"
				}
				fmt.Printf("║  %-20s  \033[33m⚐ optional\033[0m  %-23s ║\n",
					tool.Name, versionStr)
			} else {
				fmt.Printf("║  %-20s  \033[33m⚐ optional\033[0m  %-23s ║\n",
					tool.Name, "not found")
			}
		}
	}

	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	if pr.AnyMissing {
		fmt.Println("║  \033[31m⚠  Required tools missing — install before using OWS\033[0m   ║")
		for _, tool := range pr.Tools {
			if !tool.Found && tool.Required {
				fmt.Println("║")
				hints := strings.Split(tool.InstallHint, "\n")
				for _, h := range hints {
					if h != "" {
						fmt.Printf("║    $ %s\n", h)
					}
				}
			}
		}
	} else {
		fmt.Println("║  \033[32m✅ All required tools present — OWS ready\033[0m               ║")
	}

	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// PreflightJSON returns the pre-flight result as formatted JSON.
func PreflightJSON(pr *PreflightResult) string {
	out, _ := json.MarshalIndent(pr, "", "  ")
	return string(out)
}

// DisplayPreflightIfNeeded prints preflight only when tools are missing.
// Pass verbose=true to always show the environment box.
func DisplayPreflightIfNeeded(pr *PreflightResult, verbose bool) {
	if !verbose && !pr.AnyMissing {
		return // All good, silent
	}
	DisplayPreflight(pr)
}
