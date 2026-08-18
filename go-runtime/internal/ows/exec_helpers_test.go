package ows

import (
	"os/exec"
	"strings"
	"testing"
)

func TestResolveGoBinary_FindsGo(t *testing.T) {
	// Either inherited PATH or mise shims must be locatable in the test env.
	bin := resolveGoBinary()
	if bin == "" {
		t.Skip("go not in PATH or common install locations — skipping")
	}
	if !strings.HasSuffix(bin, "go") {
		t.Errorf("expected binary to end with 'go', got %q", bin)
	}
	// Verify it's actually executable
	if _, err := exec.LookPath(bin); err != nil && !strings.HasPrefix(bin, "/") {
		t.Errorf("resolved path %q not executable: %v", bin, err)
	}
}

func TestGoCmd_RunsGo(t *testing.T) {
	bin := resolveGoBinary()
	if bin == "" {
		t.Skip("go not in PATH or common install locations — skipping")
	}
	cmd := goCmd("/tmp", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go version failed: %v\nout: %s", err, out)
	}
	if !strings.Contains(string(out), "go version") {
		t.Errorf("expected 'go version' in output, got: %s", out)
	}
}

func TestGoBinaryPath_Exposed(t *testing.T) {
	// GoBinaryPath is the public accessor for diagnostics.
	// Should never panic, should match resolveGoBinary().
	if got := GoBinaryPath(); got != resolveGoBinary() {
		t.Errorf("GoBinaryPath() = %q, resolveGoBinary() = %q", got, resolveGoBinary())
	}
}

func TestErrGoNotFound_Defined(t *testing.T) {
	if ErrGoNotFound == nil {
		t.Fatal("ErrGoNotFound must be defined for callers to surface actionable diagnostics")
	}
	if !strings.Contains(ErrGoNotFound.Error(), "go binary not found") {
		t.Errorf("ErrGoNotFound message should explain the search, got: %v", ErrGoNotFound)
	}
}
