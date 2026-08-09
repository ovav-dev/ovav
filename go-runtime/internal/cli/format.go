// Package cli provides shared CLI utilities for the OVAV Go Runtime.
//
// Visual output, git helpers, YAML reading, and repo root detection.
// All stdlib-only. No external dependencies.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── Output formatting ────────────────────────────────────────────────────────

// InitOutput ensures consistent output encoding for the terminal.
func InitOutput() {
	// Go handles UTF-8 natively. No special setup needed.
}

// Output prints a result map as JSON or human-readable format.
func Output(result map[string]interface{}, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		return
	}

	status, _ := result["status"].(string)
	summary, _ := result["summary"].(string)

	switch status {
	case "ok", "pass":
		fmt.Printf("✓ %s\n", summary)
	case "blocked":
		fmt.Printf("✗ BLOCKED: %s\n", summary)
	case "error":
		fmt.Printf("✗ ERROR: %s\n", summary)
	default:
		fmt.Printf("⚠ %s\n", summary)
	}

	// Print extra sections
	for _, key := range []string{"plan", "manifest", "report"} {
		if val, ok := result[key]; ok {
			fmt.Printf("\n--- %s ---\n", key)
			if m, ok := val.(map[string]interface{}); ok {
				data, _ := json.MarshalIndent(m, "", "  ")
				if len(data) > 500 {
					data = data[:500]
				}
				fmt.Println(string(data))
			} else {
				fmt.Printf("%v\n", val)
			}
		}
	}
}

// HasJSONFlag checks if --json flag is present in args.
func HasJSONFlag(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

// ── Git helpers ──────────────────────────────────────────────────────────────

// GitInfo returns branch, SHA, and dirty status using git commands.
func GitInfo() (branch, sha, dirty string) {
	branch = runGitCmd("branch", "--show-current")
	if branch == "" {
		branch = "unknown"
	}

	sha = runGitCmd("rev-parse", "--short", "HEAD")
	if sha == "" {
		sha = "unknown"
	}

	dirtyOutput := runGitCmd("status", "--short")
	if dirtyOutput != "" {
		dirty = "dirty"
	} else {
		dirty = "clean"
	}
	return
}

// HasGitRemote checks if a git remote is configured.
func HasGitRemote() bool {
	return runGitCmd("remote") != ""
}

// GitRemoteURL returns the origin remote URL.
func GitRemoteURL() string {
	return runGitCmd("remote", "get-url", "origin")
}

func runGitCmd(args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ── Repo root detection ─────────────────────────────────────────────────────

// FindRepoRoot finds the repo root by walking up from cwd.
// Checks for .git/ first, falls back to .ovav/ (for container environments).
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Primary: .git directory or worktree file
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() || !info.IsDir() {
				return dir, nil // regular repo OR worktree
			}
		}

		// Secondary: .ovav/ + go-runtime/go.mod (ensures true OVAV root)
		ovavPath := filepath.Join(dir, ".ovav")
		goModPath := filepath.Join(dir, "go-runtime", "go.mod")
		if info, err := os.Stat(ovavPath); err == nil && info.IsDir() {
			if _, err := os.Stat(goModPath); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in an OVAV repository (no .git/ or .ovav/ found)")
		}
		dir = parent
	}
}

// MustFindRepoRoot returns the repo root or exits.
func MustFindRepoRoot() string {
	root, err := FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not in an OVAV repository.\n")
		os.Exit(1)
	}
	return root
}

// ── YAML reading (basic, stdlib-only) ────────────────────────────────────────

// ReadYAML reads a simple YAML file and returns a map.
// This is a minimal parser for OVAV waiver/config files.
// Full YAML compliance requires an external library (not needed for waivers).
func ReadYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseSimpleYAML(string(data)), nil
}

// parseSimpleYAML handles the subset of YAML used in OVAV config files.
// Supports: key: value, nested maps (2-space indent), strings (quoted or unquoted), integers, bools.
func parseSimpleYAML(content string) map[string]interface{} {
	result := make(map[string]interface{})
	currentMap := result
	parentStack := []map[string]interface{}{}
	prevIndent := -1

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Count indent
		indent := 0
		for _, ch := range line {
			if ch == ' ' {
				indent++
			} else {
				break
			}
		}

		// Pop stack if indent decreased
		for prevIndent >= 0 && indent <= prevIndent && len(parentStack) > 0 {
			currentMap = parentStack[len(parentStack)-1]
			parentStack = parentStack[:len(parentStack)-1]
			if len(parentStack) > 0 {
				prevIndent = indent
			} else {
				break
			}
		}

		// Parse key: value
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx < 0 {
			continue
		}

		key := strings.TrimSpace(trimmed[:colonIdx])
		valueStr := strings.TrimSpace(trimmed[colonIdx+1:])

		if valueStr == "" {
			// Nested map
			newMap := make(map[string]interface{})
			currentMap[key] = newMap
			parentStack = append(parentStack, currentMap)
			currentMap = newMap
			prevIndent = indent
		} else {
			// Scalar value
			val := parseYAMLValue(valueStr)
			currentMap[key] = val
		}
	}

	return result
}

func parseYAMLValue(s string) interface{} {
	s = strings.TrimSpace(s)
	// Remove surrounding quotes
	if (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) ||
		(strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) {
		s = s[1 : len(s)-1]
		return s
	}
	// Bool
	switch s {
	case "true", "True", "TRUE":
		return true
	case "false", "False", "FALSE":
		return false
	}
	// Int
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
		return i
	}
	return s
}

// ── Timestamp ────────────────────────────────────────────────────────────────

// TimestampUTC returns current UTC time in ISO 8601 format.
func TimestampUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// ── Logo ─────────────────────────────────────────────────────────────────────

// Logo returns the OVAV ASCII logo.
func Logo() string {
	return "OVAV"
}
