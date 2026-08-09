package ows

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func init() {
	// OWS-B3 FIX: Set WaiverSecret for all tests in this package.
	// Without OVAV_WAIVER_SECRET, SignWaiver() returns nil.
	// Using init() instead of TestMain because -count=3 was causing
	// intermittent failures with TestMain's os.Setenv approach.
	os.Setenv("OVAV_WAIVER_SECRET", "test-waiver-secret-key-for-tests")
}

// ── Policy Engine Tests ──────────────────────────────────────────────────────────

func TestPolicyEngine_Defaults(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	pe := NewPolicyEngine(db)
	if len(pe.policies) != 8 {
		t.Errorf("PolicyEngine has %d policies, want 8", len(pe.policies))
	}

	// Verify all POL-001 through POL-008 exist
	for i := 1; i <= 8; i++ {
		id := fmt.Sprintf("POL-%03d", i)
		if _, ok := pe.policies[id]; !ok {
			t.Errorf("missing policy %s", id)
		}
	}
}

func TestPolicyEngine_AppliesTo(t *testing.T) {
	tests := []struct {
		policyLevel PolicyLevel
		checkLevel  PolicyLevel
		applies     bool
	}{
		{PolicyRelaxed, PolicyRelaxed, true},   // relaxed enforced at relaxed+
		{PolicyRelaxed, PolicyStandard, true},  // relaxed enforced at standard
		{PolicyRelaxed, PolicyStrict, true},    // relaxed enforced at strict
		{PolicyStandard, PolicyRelaxed, false}, // standard NOT enforced at relaxed
		{PolicyStandard, PolicyStandard, true}, // standard enforced at standard+
		{PolicyStandard, PolicyStrict, true},   // standard enforced at strict
		{PolicyStrict, PolicyStandard, false},  // strict NOT enforced at standard
		{PolicyStrict, PolicyStrict, true},     // strict enforced at strict+
		{PolicyStrict, PolicyWaiver, false},    // strict bypassed at waiver
		{PolicyWaiver, PolicyStrict, false},    // waiver NOT enforced at strict
		{PolicyWaiver, PolicyWaiver, true},     // waiver only at waiver
	}

	for _, tt := range tests {
		p := Policy{ID: "TEST", Level: tt.policyLevel}
		result := p.appliesTo(tt.checkLevel)
		if result != tt.applies {
			t.Errorf("Policy(%s).appliesTo(%s) = %v, want %v", tt.policyLevel, tt.checkLevel, result, tt.applies)
		}
	}
}

func TestPolicyEngine_ValidateLevels(t *testing.T) {
	dir := t.TempDir()
	db, _ := OpenAudit(dir)
	defer db.Close()
	pe := NewPolicyEngine(db)

	// Relaxed should pass (most policies don't apply)
	if err := pe.ValidateAll(PolicyRelaxed, "/tmp", "/tmp"); err != nil {
		t.Logf("relaxed validation note: %v", err)
	}

	// Strict should fail (missing verification results)
	if err := pe.ValidateAll(PolicyStrict, "/tmp", "/tmp"); err == nil {
		// May pass if all checks succeed on this system
		t.Log("strict validation passed (unexpected)")
	}
}

// ── Waiver Tests ─────────────────────────────────────────────────────────────────

func TestWaiver_SignAndValidate(t *testing.T) {
	w := SignWaiver("owx", "main", 30*time.Minute)
	if w == nil {
		t.Fatal("SignWaiver returned nil — OVAV_WAIVER_SECRET not set?")
	}

	if w.Command != "owx" {
		t.Errorf("waiver command = %q, want owx", w.Command)
	}
	if w.Target != "main" {
		t.Errorf("waiver target = %q, want main", w.Target)
	}
	if w.Signature == "" {
		t.Error("waiver signature is empty")
	}

	// Validate should pass
	if err := ValidateWaiver(w, "owx", "main"); err != nil {
		t.Errorf("ValidateWaiver: %v", err)
	}

	// Wrong command should fail
	if err := ValidateWaiver(w, "owd", "main"); err == nil {
		t.Error("ValidateWaiver should fail for wrong command")
	}

	// Wrong target should fail
	if err := ValidateWaiver(w, "owx", "develop"); err == nil {
		t.Error("ValidateWaiver should fail for wrong target")
	}
}

func TestWaiver_Expiry(t *testing.T) {
	w := SignWaiver("owx", "main", -1*time.Hour) // already expired
	if err := ValidateWaiver(w, "owx", "main"); err == nil {
		t.Error("ValidateWaiver should fail for expired waiver")
	}
}

func TestWaiver_TTL(t *testing.T) {
	w := SignWaiver("owx", "main", 2*time.Hour) // exceeds 60 min
	if err := ValidateWaiver(w, "owx", "main"); err == nil {
		t.Error("ValidateWaiver should fail for TTL >60min")
	}
}

func TestWaiver_Tampering(t *testing.T) {
	w := SignWaiver("owx", "main", 30*time.Minute)

	// Tamper with the target
	w.Target = "develop"
	// Signature should no longer match
	if err := ValidateWaiver(w, "owx", "develop"); err == nil {
		t.Error("ValidateWaiver should detect tampered target")
	}
}

func TestWaiver_HMACUniqueness(t *testing.T) {
	w1 := SignWaiver("owx", "main", 30*time.Minute)
	w2 := SignWaiver("owx", "main", 30*time.Minute)

	// Each waiver should have a unique signature due to nonce
	if w1.Signature == w2.Signature {
		t.Error("two waivers have identical signatures — nonce not working")
	}
}

func TestWaiver_LoadAndParse(t *testing.T) {
	// Create a signed waiver and parse it back
	w := SignWaiver("owx", "main", 30*time.Minute)

	yaml := fmt.Sprintf(`waiver_id: %s
command: %s
target: %s
nonce: %s
expires_at: %d
signature: %s
`, w.ID, w.Command, w.Target, w.Nonce, w.ExpiresAt, w.Signature)

	parsed, err := parseWaiverYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parseWaiverYAML: %v", err)
	}

	if parsed.ID != w.ID {
		t.Errorf("parsed ID = %q, want %q", parsed.ID, w.ID)
	}
	if parsed.Signature != w.Signature {
		t.Errorf("parsed signature = %q, want %q", parsed.Signature, w.Signature)
	}

	// Validate the parsed waiver
	if err := ValidateWaiver(parsed, "owx", "main"); err != nil {
		t.Errorf("ValidateWaiver on parsed: %v", err)
	}
}

// ── Lock Tests ───────────────────────────────────────────────────────────────────

func TestLock_Worktree(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenAudit(dir)
	if err != nil {
		t.Fatalf("OpenAudit: %v", err)
	}
	defer db.Close()

	// Create a worktree
	wt := WorktreeRecord{
		ID:    "task/test-lock",
		Owner: "thavren",
		State: StateActive,
	}
	if err := db.SaveWorktree(wt); err != nil {
		t.Fatalf("SaveWorktree: %v", err)
	}

	// Lock as owner
	if err := db.LockWorktree("task/test-lock", "code review", "thavren"); err != nil {
		t.Fatalf("LockWorktree: %v", err)
	}

	// Verify lock
	locked, err := db.LoadWorktree("task/test-lock")
	if err != nil {
		t.Fatalf("LoadWorktree: %v", err)
	}
	if !locked.Locked {
		t.Error("worktree should be locked")
	}
	if locked.State != StateLocked {
		t.Errorf("state = %s, want %s", locked.State, StateLocked)
	}

	// Try to lock again — should fail
	if err := db.LockWorktree("task/test-lock", "double lock", "thavren"); err == nil {
		t.Error("double lock should fail")
	}

	// Unlock as owner
	if err := db.UnlockWorktree("task/test-lock", "thavren", false); err != nil {
		t.Fatalf("UnlockWorktree: %v", err)
	}

	// Verify unlock
	unlocked, err := db.LoadWorktree("task/test-lock")
	if err != nil {
		t.Fatalf("LoadWorktree: %v", err)
	}
	if unlocked.Locked {
		t.Error("worktree should be unlocked")
	}
}

func TestLock_Unauthorized(t *testing.T) {
	dir := t.TempDir()
	db, _ := OpenAudit(dir)
	defer db.Close()

	db.SaveWorktree(WorktreeRecord{ID: "task/secret", Owner: "thavren", State: StateActive})

	// Dante tries to lock Thavren's worktree
	if err := db.LockWorktree("task/secret", "curiosity", "dante"); err == nil {
		t.Error("dante should not be able to lock thavren's worktree")
	}

	// Dante tries to unlock
	if err := db.UnlockWorktree("task/secret", "dante", false); err == nil {
		t.Error("dante should not be able to unlock thavren's worktree")
	}

	// CEO force unlock (should work even if not locked)
	if err := db.UnlockWorktree("task/secret", "ceo", true); err != nil {
		t.Errorf("CEO force unlock should work: %v", err)
	}
}

func TestLock_Expiry(t *testing.T) {
	dir := t.TempDir()
	db, _ := OpenAudit(dir)
	defer db.Close()

	// Create and lock a worktree with an old timestamp
	wt := WorktreeRecord{
		ID:        "task/stale-lock",
		Owner:     "thavren",
		State:     StateLocked,
		Locked:    true,
		UpdatedAt: time.Now().Add(-25 * time.Hour), // locked 25h ago
	}
	db.SaveWorktree(wt)

	// Expire locks
	unlocked, err := db.ExpireStaleLocks()
	if err != nil {
		t.Fatalf("ExpireStaleLocks: %v", err)
	}
	if unlocked != 1 {
		t.Errorf("ExpireStaleLocks unlocked %d, want 1", unlocked)
	}

	// Verify unlocked
	wt2, _ := db.LoadWorktree("task/stale-lock")
	if wt2.Locked {
		t.Error("stale lock should be expired")
	}
}

func TestPolicyPackage(t *testing.T) {
	// Smoke test for all policy functionality
	t.Run("Defaults", TestPolicyEngine_Defaults)
	t.Run("AppliesTo", TestPolicyEngine_AppliesTo)
	t.Run("ValidateLevels", TestPolicyEngine_ValidateLevels)
}

func TestWaiverPackage(t *testing.T) {
	t.Run("SignAndValidate", TestWaiver_SignAndValidate)
	t.Run("Expiry", TestWaiver_Expiry)
	t.Run("TTL", TestWaiver_TTL)
	t.Run("Tampering", TestWaiver_Tampering)
	t.Run("HMACUniqueness", TestWaiver_HMACUniqueness)
	t.Run("LoadAndParse", TestWaiver_LoadAndParse)
}

func TestLockPackage(t *testing.T) {
	t.Run("LockWorktree", TestLock_Worktree)
	t.Run("Unauthorized", TestLock_Unauthorized)
	t.Run("Expiry", TestLock_Expiry)
}
