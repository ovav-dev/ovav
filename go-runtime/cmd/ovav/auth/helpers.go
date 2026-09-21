package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

// HomeDirOrDefault returns $HOME or the supplied default.
func HomeDirOrDefault(def string) string {
	if v := os.Getenv("HOME"); v != "" {
		return v
	}
	return def
}

// RepoRootFromCwd finds the OVAV repo root by walking up until it
// finds the .ovav directory. Falls back to cwd if not found.
func RepoRootFromCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

// PrintOK / PrintWarn / PrintErr — single-line status with optional prefix.
func PrintOK(msg string)   { fmt.Printf("  \033[32m✅ %s\033[0m\n", msg) }
func PrintWarn(msg string) { fmt.Printf("  \033[33m⚠️  %s\033[0m\n", msg) }
func PrintErr(msg string)  { fmt.Fprintf(os.Stderr, "  \033[31m❌ %s\033[0m\n", msg) }

func Die(code int, format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "  \033[31m❌ "+format+"\033[0m\n", args...)
	return code
}

// unused re-exports for tidy build
var _ = filepath.Join
