// truststore_test.go — Tests for the truststore package.
//
// Mirrors the style of internal/validators/coverage_boost_test.go: table-driven
// sub-tests where it adds clarity, t.TempDir() for sandboxed filesystem
// isolation, and direct OS calls (no mocking) because every code path here is
// about how state lands on disk.

package truststore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// setupGateRepo creates a minimal repo skeleton in a temp dir:
//   <tmp>/<gateRelPath>             — real file with deterministic content
//   <tmp>/.ovav/runtime/            — gate state parent directory
//
// The temp directory is owned by the test user, so identity.validateSecureDirectoryFD
// accepts it (matches euid and has world-writable bits stripped).
func setupGateRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	gateDir := filepath.Join(dir, filepath.Dir(gateRelPath))
	if err := os.MkdirAll(gateDir, 0755); err != nil {
		t.Fatalf("mkdir gate dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, gateRelPath), []byte("package validators\n"), 0644); err != nil {
		t.Fatalf("write gate file: %v", err)
	}

	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}

	return dir
}

// writeState is a test helper that publishes a GateState without going
// through atomic-replace. Useful for seeding baseline / grace scenarios.
func writeState(t *testing.T, dir string, st GateState) {
	t.Helper()
	runtimeDir := filepath.Join(dir, ".ovav", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "gate_state.json"), data, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

// ── RefreshGateHash — happy path ────────────────────────────────────────────

func TestRefreshGateHash_UpdatesStoredHash(t *testing.T) {
	dir := setupGateRepo(t)

	// Seed a wrong stored hash so we can prove refresh actually wrote the new one.
	seed := GateState{GateSHA256: "stale-baseline-0000000000000000000000000000000000000000000000000000"}
	writeState(t, dir, seed)

	prev, next, err := RefreshGateHash(dir)
	if err != nil {
		t.Fatalf("RefreshGateHash: %v", err)
	}
	if prev != seed.GateSHA256 {
		t.Errorf("prev = %q, want %q", prev, seed.GateSHA256)
	}
	if next == "" {
		t.Fatal("next hash is empty")
	}
	if next == prev {
		t.Errorf("next hash should differ from prev: both = %q", next)
	}

	got := ReadGateState(dir).GateSHA256
	if got != next {
		t.Errorf("stored hash after refresh = %q, want %q", got, next)
	}
}

func TestRefreshGateHash_EmptyBaselineReturnsCorrectNext(t *testing.T) {
	dir := setupGateRepo(t)
	// No prior state file at all → ReadGateState returns zero value,
	// prev should be empty and next should be the recomputed hash.
	prev, next, err := RefreshGateHash(dir)
	if err != nil {
		t.Fatalf("RefreshGateHash: %v", err)
	}
	if prev != "" {
		t.Errorf("prev = %q, want empty when no prior state", prev)
	}
	if len(next) != 64 {
		t.Errorf("next = %q, want 64-char SHA-256", next)
	}
}

// ── RefreshGateHash — grace preservation ────────────────────────────────────

func TestRefreshGateHash_PreservesLastGitOpTimeWithinGrace(t *testing.T) {
	dir := setupGateRepo(t)

	within := time.Now().Add(-30 * time.Second).Unix() // 30 s ago — well inside 5 min
	seed := GateState{
		GateSHA256:      "old-hash",
		LastGitOpTime:   within,
		LastGitOpReflog: "commit: feature work",
	}
	writeState(t, dir, seed)

	if _, _, err := RefreshGateHash(dir); err != nil {
		t.Fatalf("RefreshGateHash: %v", err)
	}

	got := ReadGateState(dir)
	if got.LastGitOpTime != within {
		t.Errorf("LastGitOpTime = %d, want %d (preserved within grace)", got.LastGitOpTime, within)
	}
	if got.LastGitOpReflog != "commit: feature work" {
		t.Errorf("LastGitOpReflog = %q, want %q", got.LastGitOpReflog, "commit: feature work")
	}
	if got.GateSHA256 == "old-hash" {
		t.Errorf("GateSHA256 should have been updated, still = %q", got.GateSHA256)
	}
}

func TestRefreshGateHash_ResetsLastGitOpTimeOutsideGrace(t *testing.T) {
	dir := setupGateRepo(t)

	outside := time.Now().Add(-30 * time.Minute).Unix() // 30 min ago — well past 5 min
	seed := GateState{
		GateSHA256:      "old-hash",
		LastGitOpTime:   outside,
		LastGitOpReflog: "commit: long ago",
	}
	writeState(t, dir, seed)

	if _, _, err := RefreshGateHash(dir); err != nil {
		t.Fatalf("RefreshGateHash: %v", err)
	}

	got := ReadGateState(dir)
	if got.LastGitOpTime != 0 {
		t.Errorf("LastGitOpTime = %d, want 0 (reset outside grace)", got.LastGitOpTime)
	}
	if got.LastGitOpReflog != "" {
		t.Errorf("LastGitOpReflog = %q, want empty after reset", got.LastGitOpReflog)
	}
}

func TestRefreshGateHash_ZeroLastGitOpTimeStaysZero(t *testing.T) {
	dir := setupGateRepo(t)
	seed := GateState{GateSHA256: "old-hash", LastGitOpTime: 0}
	writeState(t, dir, seed)

	if _, _, err := RefreshGateHash(dir); err != nil {
		t.Fatalf("RefreshGateHash: %v", err)
	}

	got := ReadGateState(dir)
	if got.LastGitOpTime != 0 {
		t.Errorf("LastGitOpTime = %d, want 0 (already zero)", got.LastGitOpTime)
	}
}

// ── RefreshGateHash — symlink refusals ──────────────────────────────────────

func TestRefreshGateHash_RefusesSymlinkGateFile(t *testing.T) {
	dir := setupGateRepo(t)

	// Replace the real gate file with a symlink.
	gatePath := filepath.Join(dir, gateRelPath)
	if err := os.Remove(gatePath); err != nil {
		t.Fatalf("remove gate: %v", err)
	}
	target := filepath.Join(dir, "real-gate-target.go")
	if err := os.WriteFile(target, []byte("package validators\n"), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, gatePath); err != nil {
		t.Fatalf("symlink gate: %v", err)
	}

	_, _, err := RefreshGateHash(dir)
	if err == nil {
		t.Fatal("expected error when gate file is a symlink, got nil")
	}
	if !contains(err.Error(), "gate file is a symlink") {
		t.Errorf("error = %v, want symlink refusal", err)
	}
}

func TestRefreshGateHash_RefusesSymlinkParentDir(t *testing.T) {
	dir := setupGateRepo(t)

	// Replace .ovav/runtime with a symlink to a different directory.
	runtimePath := filepath.Join(dir, ".ovav", "runtime")
	if err := os.RemoveAll(runtimePath); err != nil {
		t.Fatalf("remove runtime: %v", err)
	}
	altDir := filepath.Join(dir, "alt-runtime")
	if err := os.MkdirAll(altDir, 0755); err != nil {
		t.Fatalf("mkdir alt: %v", err)
	}
	if err := os.Symlink(altDir, runtimePath); err != nil {
		t.Fatalf("symlink runtime: %v", err)
	}

	_, _, err := RefreshGateHash(dir)
	if err == nil {
		t.Fatal("expected error when parent dir is a symlink, got nil")
	}
	if !contains(err.Error(), "parent dir is a symlink") && !contains(err.Error(), "state parent dir is a symlink") {
		t.Errorf("error = %v, want parent-dir symlink refusal", err)
	}
}

// ── RefreshGateHash — concurrency ───────────────────────────────────────────

func TestRefreshGateHash_ConcurrentCallersSerialized(t *testing.T) {
	dir := setupGateRepo(t)

	// 16 parallel callers — mutex must keep the file from tearing.
	const n = 16
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, _, err := RefreshGateHash(dir); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent RefreshGateHash: %v", err)
	}

	// Final file must be parseable and hold a single coherent hash.
	state := ReadGateState(dir)
	if state.GateSHA256 == "" {
		t.Fatal("GateSHA256 empty after concurrent refresh")
	}
	if len(state.GateSHA256) != 64 {
		t.Errorf("GateSHA256 = %q, want 64-char SHA-256", state.GateSHA256)
	}
}

// ── small contains helper, mirrors validators.contains ─────────────────────

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ── guards: refreshMu is held correctly even on error ──────────────────────

func TestRefreshGateHash_LeavesStateUntouchedOnGateStatError(t *testing.T) {
	dir := t.TempDir()
	// No gate file, no runtime dir — first Lstat should fail and we should
	// not have created an empty state file as a side-effect.
	_, _, err := RefreshGateHash(dir)
	if err == nil {
		t.Fatal("expected error when gate file is missing")
	}
	statePath := filepath.Join(dir, ".ovav", "runtime", "gate_state.json")
	if _, statErr := os.Stat(statePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("state file should not exist after failed refresh, stat err = %v", statErr)
	}
}
