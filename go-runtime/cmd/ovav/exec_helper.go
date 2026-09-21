package main

import (
	"os/exec"
)

// goExec is a thin wrapper around os/exec to keep imports clean.
func goExec(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
