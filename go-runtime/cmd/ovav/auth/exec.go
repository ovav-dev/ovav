package auth

import "os/exec"

// execLookPathReal delegates to exec.LookPath so the real PATH lookup
// is used. Kept separate so the helper file doesn't import os/exec
// while keeping the public surface minimal.
func execLookPathReal(name string) (string, error) {
	return exec.LookPath(name)
}
