package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// cmdHooks dispatches the ovav hooks subcommands.
//
// Usage: ovav hooks <subcommand>
// Subcommands:
//
//	install-pre-commit    — install pre-commit hook enforcing baseline freshness
//	uninstall-pre-commit  — remove OVAV pre-commit hook
//	install-pre-push      — install pre-push hook for drift gate (ADR-009)
//	uninstall-pre-push    — remove OVAV pre-push hook
//	install-all           — install all OVAV hooks
//	status                — show all hook installation states
func cmdHooks(args []string) int {
	if len(args) == 0 {
		printHooksHelp()
		return 0
	}
	switch args[0] {
	case "install-pre-commit":
		return cmdHooksInstallPreCommit(args[1:])
	case "uninstall-pre-commit":
		return cmdHooksUninstallPreCommit(args[1:])
	case "install-pre-push":
		return cmdHooksInstallPrePush(args[1:])
	case "uninstall-pre-push":
		return cmdHooksUninstallPrePush(args[1:])
	case "install-all":
		return cmdHooksInstallAll(args[1:])
	case "status":
		return cmdHooksStatusAll(args[1:])
	case "help", "--help", "-h":
		printHooksHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV hooks: unknown subcommand %q\n", args[0])
		printHooksHelp()
		return 2
	}
}

func printHooksHelp() {
	fmt.Println(`OVAV hooks — install/manage pre-commit hooks

Usage:
  ovav hooks install-pre-commit     # install pre-commit baseline freshness hook
  ovav hooks uninstall-pre-commit   # remove OVAV pre-commit hook
  ovav hooks install-pre-push       # install pre-push drift gate hook
  ovav hooks uninstall-pre-push     # remove OVAV pre-push hook
  ovav hooks install-all            # install all OVAV hooks
  ovav hooks status                 # show all hook states

Installed hooks enforce ADR-006 (baseline versioning) + ADR-009 (drift gate):
- pre-commit: protected surface changes require baseline.json update
- pre-push: blocks push to develop if drift detected
- Bypass pre-commit: OVAV_BYPASS_BASELINE_CHECK=1 git commit ...
- Bypass pre-push: OVAV_BYPASS_DRIFT_CHECK=1 git push`)
}

func cmdHooksInstallPreCommit(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: %v\n", err)
		return 1
	}

	source := filepath.Join(root, ".ovav", "hooks", "pre-commit")

	gitPath, err := resolveGitDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: %v\n", err)
		return 1
	}
	hooksDir := filepath.Join(gitPath, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: mkdir hooks dir: %v\n", err)
		return 1
	}

	dest := filepath.Join(hooksDir, "pre-commit")

	// Check if pre-commit exists and is not OVAV's
	if data, err := os.ReadFile(dest); err == nil && len(data) > 0 {
		const header = "# OVAV pre-commit hook"
		if len(data) < len(header) || string(data[:len(header)]) != header {
			fmt.Fprintf(os.Stderr, "OVAV hooks install: .git/hooks/pre-commit exists and is not OVAV's.\n")
			fmt.Fprintf(os.Stderr, "  Back it up: mv .git/hooks/pre-commit .git/hooks/pre-commit.backup\n")
			fmt.Fprintf(os.Stderr, "  Then retry.\n")
			return 2
		}
	}

	// Copy source to dest
	data, err := os.ReadFile(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: read source: %v\n", err)
		return 1
	}
	if err := os.WriteFile(dest, data, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: write dest: %v\n", err)
		return 1
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		// Non-fatal on Windows where chmod is no-op
		_ = err
	}

	fmt.Printf("✅ OVAV pre-commit hook installed → %s\n", dest)
	fmt.Println("   Enforcement:")
	fmt.Println("   - Protected surface changes require baseline.json update")
	fmt.Println("   - Bypass: OVAV_BYPASS_BASELINE_CHECK=1 git commit ...")
	fmt.Println("   - Uninstall: rm", dest)

	return 0
}

func cmdHooksUninstallPreCommit(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks uninstall: %v\n", err)
		return 1
	}
	gitPath, err := resolveGitDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks uninstall: %v\n", err)
		return 1
	}
	dest := filepath.Join(gitPath, "hooks", "pre-commit")
	if err := os.Remove(dest); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks uninstall: %v\n", err)
		return 1
	}
	fmt.Printf("✅ OVAV pre-commit hook removed ← %s\n", dest)
	return 0
}

func cmdHooksStatus(args []string) int {
	return cmdHooksStatusAll(args)
}

func cmdHooksStatusAll(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks status: %v\n", err)
		return 1
	}
	gitPath, err := resolveGitDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks status: %v\n", err)
		return 1
	}
	hooks := []string{"pre-commit", "pre-push"}
	for _, h := range hooks {
		dest := filepath.Join(gitPath, "hooks", h)
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("✅ %s: installed at %s\n", h, dest)
		} else {
			fmt.Printf("❌ %s: NOT installed (install: ovav hooks install-%s)\n", h, h)
		}
	}
	return 0
}

func cmdHooksInstallPrePush(args []string) int {
	return installHook("pre-push")
}

func cmdHooksUninstallPrePush(args []string) int {
	return uninstallHook("pre-push")
}

func cmdHooksInstallAll(args []string) int {
	rc := 0
	for _, h := range []string{"pre-commit", "pre-push"} {
		if installHook(h) != 0 {
			rc = 1
		}
	}
	return rc
}

func installHook(name string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: %v\n", err)
		return 1
	}
	source := filepath.Join(root, ".ovav", "hooks", name)
	if _, err := os.Stat(source); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: source not found: %s\n", source)
		return 1
	}
	gitPath, err := resolveGitDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: %v\n", err)
		return 1
	}
	hooksDir := filepath.Join(gitPath, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: mkdir hooks dir: %v\n", err)
		return 1
	}
	dest := filepath.Join(hooksDir, name)

	if data, err := os.ReadFile(dest); err == nil && len(data) > 0 {
		isOVAV := false
		for _, marker := range []string{"# OVAV pre-commit hook", "# OVAV pre-push hook"} {
			if len(data) >= len(marker) && string(data[:len(marker)]) == marker {
				isOVAV = true
				break
			}
		}
		if !isOVAV {
			fmt.Fprintf(os.Stderr, "OVAV hooks install: .git/hooks/%s exists and is not OVAV.\n", name)
			return 2
		}
	}

	data, err := os.ReadFile(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: read source: %v\n", err)
		return 1
	}
	if err := os.WriteFile(dest, data, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks install: write dest: %v\n", err)
		return 1
	}
	_ = os.Chmod(dest, 0o755)
	fmt.Printf("✅ OVAV %s hook installed → %s\n", name, dest)
	return 0
}

func uninstallHook(name string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks uninstall: %v\n", err)
		return 1
	}
	gitPath, err := resolveGitDir(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks uninstall: %v\n", err)
		return 1
	}
	dest := filepath.Join(gitPath, "hooks", name)
	if err := os.Remove(dest); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV hooks uninstall: %v\n", err)
		return 1
	}
	fmt.Printf("✅ OVAV %s hook removed ← %s\n", name, dest)
	return 0
}

// resolveGitDir returns the actual .git directory path (handles worktrees
// where .git is a file pointing to the shared dir).
func resolveGitDir(root string) (string, error) {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("not a git repo: %w", err)
	}
	if info.IsDir() {
		return gitPath, nil
	}
	// It's a file (worktree) — read the gitdir pointer
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("read .git file: %w", err)
	}
	line := string(data)
	if len(line) < 8 || line[:8] != "gitdir: " {
		return "", fmt.Errorf("malformed .git file: %q", line)
	}
	dir := line[8:]
	if len(dir) > 0 && dir[len(dir)-1] == '\n' {
		dir = dir[:len(dir)-1]
	}
	// /path/to/.git/worktrees/<name> → /path/to/.git
	return filepath.Dir(filepath.Dir(dir)), nil
}

// cliFindRepoRootSafe is a copy of cli.FindRepoRoot that we can call
// without an extra import (this package already imports cli via the
// sibling files). Keeping it here for clarity.
func cliFindRepoRootSafe() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".ovav", "plan", "caps.yaml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("not in OVAV repo (cwd=%s)", cwd)
}
