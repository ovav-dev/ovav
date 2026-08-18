package ows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// goBinaryCache memoizes the resolved go binary path.
// Populated by resolveGoBinary() on first call; nil if unresolved.
var goBinaryCache string

// resolveGoBinary returns the absolute path to the `go` binary, searching
// common installation locations when `go` is not on the inherited PATH.
//
// Why this exists: ovav is often launched from environments (cron, sudo,
// IDE integrations, shell scripts) that don't have mise/Go activated in
// PATH. Without this resolver, `exec.Command("go", ...)` silently fails
// with "executable file not found in $PATH" and validators report bogus
// "go vet FAILED" / "go test FAILED" with empty stderr.
//
// Search order:
//  1. exec.LookPath("go")               — inherited PATH (fast path)
//  2. ~/.local/share/mise/shims/go     — mise (user-mode)
//  3. ~/.local/share/mise/installs/go/*/bin/go  — mise pinned version
//  4. /usr/local/go/bin/go             — system install
//  5. /opt/go/bin/go                   — Linux distro
//  6. /opt/homebrew/bin/go             — macOS Homebrew (Apple Silicon)
//  7. /usr/local/bin/go                — macOS Intel Homebrew
//  8. /snap/bin/go                     — Ubuntu snap
//  9. /usr/lib/go-*/bin/go             — Debian/Ubuntu packages
//
// Returns "" if go cannot be found. Callers should treat "" as a hard error.
func resolveGoBinary() string {
	if goBinaryCache != "" {
		return goBinaryCache
	}

	// 1. Inherited PATH
	if p, err := exec.LookPath("go"); err == nil {
		goBinaryCache = p
		return p
	}

	home, _ := os.UserHomeDir()
	candidates := []string{}

	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "mise", "shims", "go"),
		)
		// mise installs: walk ~/.local/share/mise/installs/go/*/bin/go
		matches, _ := filepath.Glob(filepath.Join(home, ".local", "share", "mise", "installs", "go", "*", "bin", "go"))
		candidates = append(candidates, matches...)
	}

	candidates = append(candidates,
		"/usr/local/go/bin/go",
		"/opt/go/bin/go",
		"/opt/homebrew/bin/go",
		"/usr/local/bin/go",
		"/snap/bin/go",
	)
	// Debian/Ubuntu: /usr/lib/go-1.21/bin/go etc.
	if matches, err := filepath.Glob("/usr/lib/go-*/bin/go"); err == nil {
		candidates = append(candidates, matches...)
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			goBinaryCache = c
			return c
		}
	}

	return ""
}

// goCmd returns an *exec.Cmd for the `go` binary with the given args,
// automatically resolving the binary path via resolveGoBinary().
//
// If go cannot be found, returns a Cmd that will fail with a clear error
// message rather than the ambiguous "executable file not found".
func goCmd(dir string, args ...string) *exec.Cmd {
	bin := resolveGoBinary()
	if bin == "" {
		// Return a cmd that will fail loudly. We use the literal "go"
		// so exec surfaces its own diagnostic; the caller already wraps
		// the error in a validator-style message.
		return exec.Command("go", args...)
	}
	return exec.Command(bin, args...)
}

// GoBinaryPath returns the resolved go binary path, or empty string if not found.
// Exposed for diagnostics (e.g., `ovav doctor`, `ovav status`).
func GoBinaryPath() string {
	return resolveGoBinary()
}

// ErrGoNotFound is returned when no `go` binary can be located in any of the
// searched paths. Callers should surface this as actionable guidance.
var ErrGoNotFound = fmt.Errorf("go binary not found in PATH or common install locations (mise shims, /usr/local/go/bin, /opt/go/bin, Homebrew, snap, /usr/lib/go-*)")

// PathDirs returns the directories from the current PATH, for diagnostics.
func PathDirs() []string {
	return strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
}
