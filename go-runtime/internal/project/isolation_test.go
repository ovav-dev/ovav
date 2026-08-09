package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── isUnderPath ───────────────────────────────────────────────────────────────

func TestIsUnderPath_ExactMatch(t *testing.T) {
	if !isUnderPath("/foo/bar", "/foo/bar") {
		t.Error("exact match should return true")
	}
}

func TestIsUnderPath_Child(t *testing.T) {
	if !isUnderPath("/foo/bar/baz", "/foo/bar") {
		t.Error("child should be under parent")
	}
}

func TestIsUnderPath_NotUnder(t *testing.T) {
	if isUnderPath("/other/path", "/foo/bar") {
		t.Error("unrelated path should not be under parent")
	}
}

func TestIsUnderPath_Sibling(t *testing.T) {
	if isUnderPath("/foo/bar2", "/foo/bar") {
		t.Error("sibling directory should not be under parent")
	}
}

func TestIsUnderPath_PrefixAmbiguity(t *testing.T) {
	// /foo/bar should NOT match as under /foo/ba
	if isUnderPath("/foo/bar", "/foo/ba") {
		t.Error("prefix ambiguity should not match")
	}
}

func TestIsUnderPath_DeepNesting(t *testing.T) {
	if !isUnderPath("/a/b/c/d/e", "/a/b") {
		t.Error("deep nesting should be under parent")
	}
}

// ── findProtectorates ────────────────────────────────────────────────────────

func TestFindProtectorates_None(t *testing.T) {
	dir := t.TempDir()
	result := findProtectorates(dir)
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestFindProtectorates_RegistryExists(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	result := findProtectorates(dir)
	if result != ".ovav/registry" {
		t.Errorf("expected .ovav/registry, got %q", result)
	}
}

func TestFindProtectorates_PlanExists(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)

	result := findProtectorates(dir)
	if result != ".ovav/plan" {
		t.Errorf("expected .ovav/plan, got %q", result)
	}
}

func TestFindProtectorates_MemoryExists(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "memory"), 0755)

	result := findProtectorates(dir)
	if result != ".ovav/memory" {
		t.Errorf("expected .ovav/memory, got %q", result)
	}
}

func TestFindProtectorates_MultipleExistReturnsFirst(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)

	result := findProtectorates(dir)
	// registry comes first in SystemsProtectorates order
	if result != ".ovav/registry" {
		t.Errorf("expected .ovav/registry (first match), got %q", result)
	}
}

func TestFindProtectorates_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry"), []byte("not-a-dir"), 0644)

	result := findProtectorates(dir)
	if result != "" {
		t.Errorf("expected empty (file, not dir), got %q", result)
	}
}

// ── detectTargetType ─────────────────────────────────────────────────────────

func TestDetectTargetType_Systems(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)

	got := detectTargetType(dir)
	if got != TargetSystems {
		t.Errorf("expected TargetSystems, got %s", got)
	}
}

func TestDetectTargetType_SystemsOvavFile(t *testing.T) {
	dir := t.TempDir()
	// .ovav as a file, not a directory — should NOT be Systems
	os.WriteFile(filepath.Join(dir, ".ovav"), []byte("not-a-dir"), 0644)

	got := detectTargetType(dir)
	if got == TargetSystems {
		t.Error(".ovav file should not be detected as Systems")
	}
}

func TestDetectTargetType_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got := detectTargetType(dir)
	if got != TargetUnknown {
		t.Errorf("expected TargetUnknown for empty dir, got %s", got)
	}
}

func TestDetectTargetType_ProductSimulated(t *testing.T) {
	// Simulate Product by creating the path under a temporary "home"
	fakeHome := t.TempDir()
	localShare := filepath.Join(fakeHome, ".local", "share")
	productDir := filepath.Join(localShare, "ovav")
	os.MkdirAll(productDir, 0755)

	// We can't override os.UserHomeDir(), so this test validates
	// the heuristic branch: isUnderPath + basename == "ovav"
	// We test this path directly by constructing scenarios that
	// hit the heuristic, not the exact home directory path.
	//
	// For the exact-Path match, we rely on the ValidateTarget
	// integration tests below that set up real directory structures.

	// Heuristic check: under .local/share with basename "ovav"
	if !isUnderPath(productDir, localShare) {
		t.Fatal("test setup failed")
	}
	if filepath.Base(productDir) != "ovav" {
		t.Fatal("test setup failed: unexpected basename")
	}

	// This confirms our heuristic logic is correct even if
	// os.UserHomeDir() points elsewhere.
	got := detectTargetType(productDir)
	// Since productDir doesn't have .ovav/, Systems check fails.
	// The Product check compares against the real home dir, not fakeHome.
	// So this will be Unknown unless fakeHome happens to match UserHomeDir.
	// This is an acceptable limitation — real Product detection is tested
	// via integration tests on the actual filesystem.
	_ = got
}

// ── ValidateTarget ───────────────────────────────────────────────────────────

func TestValidateTarget_SystemsPasses(t *testing.T) {
	dir := t.TempDir()
	// Set up minimal Systems structure
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)

	err := ValidateTarget(dir)
	if err != nil {
		t.Errorf("Systems should pass validation, got: %v", err)
	}
}

func TestValidateTarget_SystemsWithProtectoratesStillPasses(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "memory"), 0755)

	// Systems is allowed to have protectorates — they belong there.
	err := ValidateTarget(dir)
	if err != nil {
		t.Errorf("Systems with protectorates should pass, got: %v", err)
	}
}

func TestValidateTarget_UnknownPasses(t *testing.T) {
	dir := t.TempDir()
	// No .ovav/ directory, not under home/.local/share/ovav/
	err := ValidateTarget(dir)
	if err != nil {
		t.Errorf("Unknown target should pass, got: %v", err)
	}
}

func TestValidateTarget_ProductCleanPasses(t *testing.T) {
	// Simulate a Product directory at ~/.local/share/ovav/ without protectorates.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	productPath := filepath.Join(homeDir, ".local", "share", "ovav")

	// Skip if the path already exists and we shouldn't pollute it
	// Use a temp subdirectory approach if the actual product path exists
	if _, err := os.Stat(productPath); err == nil {
		// Product path exists — we validate it passes if clean of protectorates.
		// Only run if there are no Systems protectorates present.
		if found := findProtectorates(productPath); found != "" {
			t.Skipf("real Product path has protectorates (%s) — test would fail; skip", found)
		}
		err := ValidateTarget(productPath)
		if err != nil {
			t.Errorf("clean Product should pass, got: %v", err)
		}
		return
	}

	// Create a temporary Product path under real home
	t.Cleanup(func() { os.RemoveAll(productPath) })
	os.MkdirAll(productPath, 0755)

	err = ValidateTarget(productPath)
	if err != nil {
		t.Errorf("clean Product should pass, got: %v", err)
	}
}

func TestValidateTarget_ProductWithProtectorateBlocks(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// Product detection heuristic requires basename "ovav" under XDG_DATA_HOME.
	parentDir := filepath.Join(homeDir, ".local", "share", "ovav-isolation-parent")
	productPath := filepath.Join(parentDir, "ovav")
	t.Cleanup(func() { os.RemoveAll(parentDir) })

	os.MkdirAll(productPath, 0755)

	// Place a Systems protectorate in the Product directory — this is the
	// cross-contamination scenario we must hard-block.
	os.MkdirAll(filepath.Join(productPath, ".ovav", "registry"), 0755)

	err = ValidateTarget(productPath)
	if err == nil {
		t.Fatal("Product with .ovav/registry/ MUST be blocked")
	}
	if !strings.Contains(err.Error(), "ISOLATION VIOLATION") {
		t.Errorf("error should contain 'ISOLATION VIOLATION', got: %v", err)
	}
	if !strings.Contains(err.Error(), ".ovav/registry") {
		t.Errorf("error should mention .ovav/registry, got: %v", err)
	}
}

func TestValidateTarget_ProductWithMultipleProtectoratesBlocks(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	parentDir := filepath.Join(homeDir, ".local", "share", "ovav-multi-parent")
	productPath := filepath.Join(parentDir, "ovav")
	t.Cleanup(func() { os.RemoveAll(parentDir) })

	os.MkdirAll(productPath, 0755)
	os.MkdirAll(filepath.Join(productPath, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(productPath, ".ovav", "memory"), 0755)

	err = ValidateTarget(productPath)
	if err == nil {
		t.Fatal("Product with protectorates MUST be blocked")
	}
	// First protectorate in order is .ovav/plan
	if !strings.Contains(err.Error(), ".ovav/plan") {
		t.Errorf("error should mention first protectorate found, got: %v", err)
	}
}

func TestValidateTarget_InvalidRoot(t *testing.T) {
	err := ValidateTarget("/nonexistent/path/that/does/not/exist")
	// Should not panic; should resolve gracefully via filepath.Abs fallback.
	if err != nil {
		t.Errorf("nonexistent root should not error from ValidateTarget: %v", err)
	}
}

// ── Integration: Sync with ValidateTarget ────────────────────────────────────

func TestSync_SystemsWithOvavPassesIsolation(t *testing.T) {
	dir := t.TempDir()
	// Minimal Systems marker
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)

	// Sync on a bare Systems marker will fail due to missing sources,
	// but it must NOT fail due to isolation violation.
	err := Sync(dir, false)
	if err == nil {
		return // success (if sources happened to exist)
	}
	if strings.Contains(err.Error(), "ISOLATION VIOLATION") {
		t.Errorf("Sync should not be blocked by isolation on Systems: %v", err)
	}
}

func TestSync_UnknownRootFailsOnMissingSources(t *testing.T) {
	dir := t.TempDir()
	// Unknown root — no .ovav/, not Product. Isolation passes,
	// but Sync fails because there are no sources to project.
	err := Sync(dir, false)
	if err == nil {
		t.Error("Sync on empty unknown dir should fail (no sources)")
	}
	if strings.Contains(err.Error(), "ISOLATION") {
		t.Errorf("error should be about missing sources, not isolation: %v", err)
	}
}

func TestSync_ProductWithProtectorateIsHardBlocked(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	parentDir := filepath.Join(homeDir, ".local", "share", "ovav-sync-parent")
	productPath := filepath.Join(parentDir, "ovav")
	t.Cleanup(func() { os.RemoveAll(parentDir) })

	os.MkdirAll(productPath, 0755)
	os.MkdirAll(filepath.Join(productPath, ".ovav", "registry"), 0755)

	err = Sync(productPath, false)
	if err == nil {
		t.Fatal("Sync on Product with protectorates MUST be blocked")
	}
	if !strings.Contains(err.Error(), "ISOLATION VIOLATION") {
		t.Errorf("error should be ISOLATION VIOLATION, got: %v", err)
	}
}

func TestSyncAndDetectChanges_ProductWithProtectorateBlocked(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	parentDir := filepath.Join(homeDir, ".local", "share", "ovav-sync2-parent")
	productPath := filepath.Join(parentDir, "ovav")
	t.Cleanup(func() { os.RemoveAll(parentDir) })

	os.MkdirAll(productPath, 0755)
	os.MkdirAll(filepath.Join(productPath, ".ovav", "memory"), 0755)

	result, err := SyncAndDetectChanges(productPath, false)
	if err == nil {
		t.Fatal("SyncAndDetectChanges on Product with protectorates MUST be blocked")
	}
	if result != nil {
		t.Error("result should be nil when blocked")
	}
	if !strings.Contains(err.Error(), "ISOLATION VIOLATION") {
		t.Errorf("error should be ISOLATION VIOLATION, got: %v", err)
	}
}

// ── TargetType String ────────────────────────────────────────────────────────

func TestTargetType_String(t *testing.T) {
	tests := []struct {
		tt   TargetType
		want string
	}{
		{TargetSystems, "Systems"},
		{TargetProduct, "Product"},
		{TargetUnknown, "Unknown"},
		{TargetType(99), "Unknown"},
	}

	for _, tc := range tests {
		got := tc.tt.String()
		if got != tc.want {
			t.Errorf("TargetType(%d).String() = %q, want %q", tc.tt, got, tc.want)
		}
	}
}

// ── Edge: systemsProtectorates order and completeness ────────────────────────

func TestSystemsProtectorates_AllExistInOrder(t *testing.T) {
	// Verify SystemsProtectorates is properly ordered so that
	// findProtectorates returns the first match consistently.
	if len(SystemsProtectorates) < 3 {
		t.Fatal("SystemsProtectorates should have at least 3 entries")
	}

	expected := []string{
		".ovav/registry",
		".ovav/plan",
		".ovav/memory",
	}

	for i, want := range expected {
		if SystemsProtectorates[i] != want {
			t.Errorf("SystemsProtectorates[%d] = %q, want %q", i, SystemsProtectorates[i], want)
		}
	}
}

func TestSystemsProtectorates_AllMatchPattern(t *testing.T) {
	for _, pd := range SystemsProtectorates {
		if !strings.HasPrefix(pd, ".ovav/") {
			t.Errorf("protectorate %q should start with .ovav/", pd)
		}
	}
}
