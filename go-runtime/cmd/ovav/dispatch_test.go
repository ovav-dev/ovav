package main

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// dispatch_test.go — Tests for command routing (dispatch.go)
// ═══════════════════════════════════════════════════════════════════════════

// ── routeCommand: known commands ──────────────────────────────────────────

func TestRouteCommand_VersionReturnsZero(t *testing.T) {
	code := routeCommand("version", nil)
	if code != 0 {
		t.Errorf("routeCommand(version) = %d, want 0", code)
	}
}

func TestRouteCommand_VersionAlias(t *testing.T) {
	code := routeCommand("--version", nil)
	if code != 0 {
		t.Errorf("routeCommand(--version) = %d, want 0", code)
	}
	code = routeCommand("-v", nil)
	if code != 0 {
		t.Errorf("routeCommand(-v) = %d, want 0", code)
	}
}

func TestRouteCommand_HelpReturnsZero(t *testing.T) {
	code := routeCommand("help", nil)
	if code != 0 {
		t.Errorf("routeCommand(help) = %d, want 0", code)
	}
}

func TestRouteCommand_HelpAlias(t *testing.T) {
	code := routeCommand("--help", nil)
	if code != 0 {
		t.Errorf("routeCommand(--help) = %d, want 0", code)
	}
	code = routeCommand("-h", nil)
	if code != 0 {
		t.Errorf("routeCommand(-h) = %d, want 0", code)
	}
}

func TestRouteCommand_UnknownReturnsTwo(t *testing.T) {
	code := routeCommand("nonexistent-command", nil)
	if code != 2 {
		t.Errorf("routeCommand(unknown) = %d, want 2", code)
	}
}

func TestRouteCommand_UnknownCommandVariants(t *testing.T) {
	unknowns := []string{"foo", "bar", "VERSION", "HELP", "", "statusx"}
	for _, cmd := range unknowns {
		code := routeCommand(cmd, nil)
		if code != 2 {
			t.Errorf("routeCommand(%q) = %d, want 2", cmd, code)
		}
	}
}

// ── isKnownCommand ────────────────────────────────────────────────────────

func TestIsKnownCommand_Known(t *testing.T) {
	known := []string{
		"status", "version", "help", "login", "vault", "doctor",
		"install", "deploy", "git", "worktree", "chronos",
		"--version", "-v", "--help", "-h",
	}
	for _, cmd := range known {
		if !isKnownCommand(cmd) {
			t.Errorf("isKnownCommand(%q) = false, want true", cmd)
		}
	}
}

func TestIsKnownCommand_Unknown(t *testing.T) {
	unknown := []string{"foo", "bar", "", "STATUS", "Version", "xyz"}
	for _, cmd := range unknown {
		if isKnownCommand(cmd) {
			t.Errorf("isKnownCommand(%q) = true, want false", cmd)
		}
	}
}

// ── knownCommands ─────────────────────────────────────────────────────────

func TestKnownCommands_NotEmpty(t *testing.T) {
	cmds := knownCommands()
	if len(cmds) == 0 {
		t.Fatal("knownCommands() returned empty slice")
	}
	// Should have at least 30 commands
	if len(cmds) < 30 {
		t.Errorf("knownCommands() has %d entries, want >= 30", len(cmds))
	}
}

func TestKnownCommands_NoDuplicates(t *testing.T) {
	cmds := knownCommands()
	seen := make(map[string]bool)
	for _, cmd := range cmds {
		if seen[cmd] {
			t.Errorf("duplicate command in knownCommands(): %q", cmd)
		}
		seen[cmd] = true
	}
}

func TestKnownCommands_ContainsCoreCommands(t *testing.T) {
	cmds := knownCommands()
	core := []string{"status", "version", "help", "login", "vault", "doctor"}
	cmdSet := make(map[string]bool)
	for _, c := range cmds {
		cmdSet[c] = true
	}
	for _, c := range core {
		if !cmdSet[c] {
			t.Errorf("knownCommands() missing core command %q", c)
		}
	}
}

// ── routeCommand: aliases route to same handler ───────────────────────────

func TestRouteCommand_DoctorAndHealthAreAliases(t *testing.T) {
	// Both "doctor" and "health" should route to cmdDoctor.
	// We can't easily test the return value (depends on repo state),
	// but we verify they don't return 2 (unknown command).
	code1 := routeCommand("doctor", []string{"--quick"})
	code2 := routeCommand("health", []string{"--quick"})
	if code1 == 2 {
		t.Error("doctor should be a known command")
	}
	if code2 == 2 {
		t.Error("health should be a known command")
	}
}

func TestRouteCommand_RestoreAndRollbackAreAliases(t *testing.T) {
	code1 := routeCommand("restore", nil)
	code2 := routeCommand("rollback", nil)
	if code1 == 2 {
		t.Error("restore should be a known command")
	}
	if code2 == 2 {
		t.Error("rollback should be a known command")
	}
}

func TestRouteCommand_WorktreeAndWtAreAliases(t *testing.T) {
	// routeCommand returns code 2 for "needs subcommand" (not "unknown")
	// isKnownCommand is the right function to test command recognition
	if !isKnownCommand("worktree") {
		t.Error("worktree should be a known command")
	}
	if !isKnownCommand("wt") {
		t.Error("wt should be a known command")
	}
}

// ── routeCommand: version with JSON flag ──────────────────────────────────

func TestRouteCommand_VersionJSON(t *testing.T) {
	code := routeCommand("version", []string{"--json"})
	if code != 0 {
		t.Errorf("routeCommand(version --json) = %d, want 0", code)
	}
}

// ── parseModeFlag (additional coverage) ───────────────────────────────────

func TestParseModeFlag_Default(t *testing.T) {
	mode := parseModeFlag(nil)
	if string(mode) != "dry-run" {
		t.Errorf("parseModeFlag(nil) = %q, want dry-run", mode)
	}
}

func TestParseModeFlag_Sandbox(t *testing.T) {
	mode := parseModeFlag([]string{"--sandbox"})
	if string(mode) != "sandbox" {
		t.Errorf("parseModeFlag(--sandbox) = %q, want sandbox", mode)
	}
}

func TestParseModeFlag_Apply(t *testing.T) {
	mode := parseModeFlag([]string{"--apply"})
	if string(mode) != "source-local-apply" {
		t.Errorf("parseModeFlag(--apply) = %q, want source-local-apply", mode)
	}
}

// ── jsonOutput ────────────────────────────────────────────────────────────

func TestJsonOutput_ValidData(t *testing.T) {
	code := jsonOutput(map[string]string{"key": "value"})
	if code != 0 {
		t.Errorf("jsonOutput(valid) = %d, want 0", code)
	}
}

// ── isPublicCommand (additional edge cases) ──────────────────────────────

func TestIsPublicCommand_AllPublicCommands(t *testing.T) {
	public := []string{"login", "signin", "auth", "help", "--help", "-h"}
	for _, cmd := range public {
		if !isPublicCommand(cmd) {
			t.Errorf("isPublicCommand(%q) = false, want true", cmd)
		}
	}
}

func TestIsPublicCommand_PrivateCommands(t *testing.T) {
	private := []string{"status", "vault", "doctor", "install", "deploy", "logout", "whoami"}
	for _, cmd := range private {
		if isPublicCommand(cmd) {
			t.Errorf("isPublicCommand(%q) = true, want false", cmd)
		}
	}
}
