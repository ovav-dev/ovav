package validators

import (
	"os"
	"path/filepath"
)

// IsDev returns true when the working tree has uncommitted changes,
// indicating a development environment where strict validators should
// report warnings instead of failures.
func IsDev(root string) bool {
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return true // no .git = likely dev/testing
	}
	// Check for dirty working tree via git status porcelain
	headFile := filepath.Join(gitDir, "HEAD")
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return true
	}
	return false
}

// DevPass returns a pass result with a warning message for dev environments.
// Use this when a validator would normally fail but the failure is expected
// in a development (uncommitted) state.
func DevPass(id, name, msg string, weight int) Result {
	return Result{
		ID:      id,
		Name:    name,
		Status:  "pass",
		Weight:  weight,
		Message: msg + " (dev mode: warning only)",
	}
}
