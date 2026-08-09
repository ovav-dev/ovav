package defense

import (
	"strings"
	"testing"
	"time"
)

// ── Cortex Tests ─────────────────────────────────────────────────────────

func TestCortex_ClassifyRootCause(t *testing.T) {
	c := NewCortex()

	tests := []struct {
		intrusionType string
		path          string
		want          string
	}{
		{"user_creation", "/etc/passwd", "user_management"},
		{"git_operation", ".git/config", "source_control"},
		{"file_sync", "/data/sync", "synchronization"},
		{"unknown_type", "/tmp/test", "unknown"},
	}
	for _, tt := range tests {
		got := c.ClassifyRootCause(tt.intrusionType, tt.path)
		if got != tt.want {
			t.Errorf("ClassifyRootCause(%q, %q) = %q, want %q", tt.intrusionType, tt.path, got, tt.want)
		}
	}
}

func TestCortex_LearnFalsePositive(t *testing.T) {
	c := NewCortex()

	// First two occurrences → not whitelisted yet
	if c.LearnFalsePositive("git_operation", ".git/config", true) {
		t.Error("should not whitelist on first occurrence")
	}
	if c.LearnFalsePositive("git_operation", ".git/config", true) {
		t.Error("should not whitelist on second occurrence")
	}

	// Third occurrence → whitelisted
	if !c.LearnFalsePositive("git_operation", ".git/config", true) {
		t.Error("should whitelist on third occurrence")
	}

	// Verify it's now known
	if !c.IsKnownFalsePositive("git_operation", ".git/config") {
		t.Error("should be known false positive after 3+ occurrences")
	}
}

func TestCortex_IsKnownFalsePositive_Unknown(t *testing.T) {
	c := NewCortex()

	if c.IsKnownFalsePositive("unknown", "/tmp/test") {
		t.Error("should not be known for unseen pattern")
	}
}

func TestCortex_ApplyHardening(t *testing.T) {
	c := NewCortex()

	c.ApplyHardening("file_sync", "reduced_sensitivity")
	c.ApplyHardening("network_access", "increased_monitoring")

	state := c.HardeningState()
	if state["file_sync"] != "reduced_sensitivity" {
		t.Errorf("hardening state = %q", state["file_sync"])
	}
	if len(state) != 2 {
		t.Errorf("want 2 hardening entries, got %d", len(state))
	}
}

// ── Responder Tests ──────────────────────────────────────────────────────

func TestResponder_RespondInfo(t *testing.T) {
	r := NewResponder(nil)
	actions := r.Respond("test", SevInfo, "/tmp/test")

	if len(actions) != 1 || actions[0] != ActLog {
		t.Errorf("info should log only, got %v", actions)
	}
}

func TestResponder_RespondWarning(t *testing.T) {
	r := NewResponder(nil)
	actions := r.Respond("test", SevWarning, "/tmp/test")

	if len(actions) != 2 {
		t.Errorf("warning should log+alert, got %v", actions)
	}
	if !containsAction(actions, ActAlert) {
		t.Error("warning should include alert")
	}
}

func TestResponder_RespondCritical(t *testing.T) {
	r := NewResponder(nil)
	actions := r.Respond("test", SevCritical, "/tmp/test")

	if !containsAction(actions, ActQuarantine) {
		t.Error("critical should include quarantine for non-immune files")
	}
	if !containsAction(actions, ActAlert) {
		t.Error("critical should include alert")
	}
}

func TestResponder_RespondDeadly(t *testing.T) {
	r := NewResponder(nil)
	actions := r.Respond("test", SevDeadly, "/tmp/test")

	if !containsAction(actions, ActQuarantine) {
		t.Error("deadly should include quarantine")
	}
	if !containsAction(actions, ActLockdown) {
		t.Error("deadly should include lockdown")
	}
}

func TestResponder_ImmuneFile(t *testing.T) {
	r := NewResponder(nil)

	// Attempt critical action on immune file
	actions := r.Respond("test", SevCritical, ".ovav/plan/caps.yaml")

	if containsAction(actions, ActQuarantine) {
		t.Error("should NOT quarantine immune file")
	}
	if !containsAction(actions, ActBlock) {
		t.Error("should block on immune file")
	}
}

func TestResponder_FalsePositive(t *testing.T) {
	c := NewCortex()
	// Learn a false positive
	c.LearnFalsePositive("git_op", "/safe/path", true)
	c.LearnFalsePositive("git_op", "/safe/path", true)
	c.LearnFalsePositive("git_op", "/safe/path", true)

	r := NewResponder(c)
	actions := r.Respond("git_op", SevDeadly, "/safe/path")

	// Should only log (false positive)
	if len(actions) != 1 || actions[0] != ActLog {
		t.Errorf("false positive should log only, got %v", actions)
	}
}

func TestResponder_Lockdown(t *testing.T) {
	r := NewResponder(nil)

	if r.IsLockdownActive() {
		t.Error("lockdown should start inactive")
	}

	r.SetLockdown(true)
	if !r.IsLockdownActive() {
		t.Error("lockdown should be active after set")
	}

	r.SetLockdown(false)
	if r.IsLockdownActive() {
		t.Error("lockdown should be inactive after clear")
	}
}

func TestResponder_AddImmuneFile(t *testing.T) {
	r := NewResponder(nil)
	r.AddImmuneFile("/custom/immune")

	actions := r.Respond("test", SevCritical, "/custom/immune/file.txt")
	if !containsAction(actions, ActBlock) {
		t.Error("custom immune path should block quarantine")
	}
}

// ── Authorizer Tests ─────────────────────────────────────────────────────

func TestAuthorizer_ProtectedBranch(t *testing.T) {
	a := NewAuthorizer()
	entry := DriftEntry{File: "go.mod", Domain: "governed_config", Branch: "main"}

	result := a.EvaluateDrift(entry, true, true)
	if result.Authorized {
		t.Error("protected branch should NOT be authorized")
	}
	if !strings.Contains(result.Reason, "CEO") {
		t.Errorf("reason should mention CEO waiver: %q", result.Reason)
	}
}

func TestAuthorizer_MutableRuntime(t *testing.T) {
	a := NewAuthorizer()
	entry := DriftEntry{File: "runtime/cache", Domain: "mutable_runtime", Branch: "feature/test"}

	result := a.EvaluateDrift(entry, false, false)
	if !result.Authorized {
		t.Error("mutable runtime should be auto-authorized")
	}
}

func TestAuthorizer_NoSession(t *testing.T) {
	a := NewAuthorizer()
	entry := DriftEntry{File: "caps.yaml", Domain: "governed_config", Branch: "feature/test"}

	result := a.EvaluateDrift(entry, false, false)
	if result.Authorized {
		t.Error("no session should require manual authorization")
	}
}

func TestAuthorizer_KnownPattern(t *testing.T) {
	a := NewAuthorizer()

	// Record 3 prior authorizations
	a.RecordAuthorization("go.mod")
	a.RecordAuthorization("go.mod")
	a.RecordAuthorization("go.mod")

	entry := DriftEntry{File: "go.mod", Domain: "governed_config", Branch: "feature/test"}
	result := a.EvaluateDrift(entry, false, true)

	if !result.Authorized {
		t.Error("known pattern on task branch should be auto-authorized")
	}
	if result.Rule != "R4_KNOWN_PATTERN" {
		t.Errorf("rule = %s, want R4_KNOWN_PATTERN", result.Rule)
	}
}

func TestAuthorizer_GovernedConfig(t *testing.T) {
	a := NewAuthorizer()
	entry := DriftEntry{File: "caps.yaml", Domain: "governed_config", Branch: "feature/test"}

	result := a.EvaluateDrift(entry, false, true)
	if !result.Authorized {
		t.Error("governed config on task branch with session should be auto")
	}
}

func TestAuthorizer_Unclassified(t *testing.T) {
	a := NewAuthorizer()
	entry := DriftEntry{File: "new_file.txt", Domain: "unclassified", Branch: "feature/test"}

	result := a.EvaluateDrift(entry, false, true)
	if !result.Authorized {
		t.Error("unclassified on task branch with session should be auto")
	}
}

// ── Credentials Tests ────────────────────────────────────────────────────

func TestCredentialManager_Register(t *testing.T) {
	cm := NewCredentialManager()

	cred := cm.RegisterCredential("cloudflare", "secret-token-123", 24*time.Hour)

	if cred.Service != "cloudflare" {
		t.Errorf("service = %s, want cloudflare", cred.Service)
	}
	if !cred.Healthy {
		t.Error("new credential should be healthy")
	}
	if cm.Count() != 1 {
		t.Errorf("count = %d, want 1", cm.Count())
	}
}

func TestCredentialManager_Rotate(t *testing.T) {
	cm := NewCredentialManager()

	cm.RegisterCredential("github", "old-token", 1*time.Hour)
	newCred, err := cm.RotateCredential("github", "new-token", 24*time.Hour)

	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}
	if newCred.Hash == "" {
		t.Error("new credential should have hash")
	}
	if cm.Count() != 1 {
		t.Errorf("count after rotate = %d, want 1", cm.Count())
	}
}

func TestCredentialManager_CheckHealth(t *testing.T) {
	cm := NewCredentialManager()

	cm.RegisterCredential("svc1", "t1", 1*time.Hour)  // healthy
	cm.RegisterCredential("svc2", "t2", -1*time.Hour) // expired

	unhealthy := cm.CheckHealth()
	if len(unhealthy) != 1 {
		t.Errorf("want 1 unhealthy, got %d", len(unhealthy))
	}
	if unhealthy[0].Service != "svc2" {
		t.Errorf("unhealthy service = %s, want svc2", unhealthy[0].Service)
	}
}

func TestCredentialManager_NeedsRotation(t *testing.T) {
	cm := NewCredentialManager()

	if cm.NeedsRotation() {
		t.Error("empty manager should not need rotation")
	}

	cm.RegisterCredential("svc", "token", -1*time.Hour) // expired
	if !cm.NeedsRotation() {
		t.Error("expired credential should need rotation")
	}
}

func TestCredentialManager_MultipleServices(t *testing.T) {
	cm := NewCredentialManager()

	cm.RegisterCredential("cloudflare", "cf-token", 24*time.Hour)
	cm.RegisterCredential("github", "gh-token", 24*time.Hour)
	cm.RegisterCredential("codeberg", "cb-token", 24*time.Hour)

	if cm.Count() != 3 {
		t.Errorf("count = %d, want 3", cm.Count())
	}
}

func TestHashSecret(t *testing.T) {
	h1 := hashSecret("test-secret")
	h2 := hashSecret("test-secret")
	h3 := hashSecret("different")

	if h1 != h2 {
		t.Error("same secret should produce same hash")
	}
	if h1 == h3 {
		t.Error("different secrets should produce different hashes")
	}
	if len(h1) != 64 {
		t.Errorf("SHA256 hash length = %d, want 64", len(h1))
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────

func containsAction(actions []ResponseAction, want ResponseAction) bool {
	for _, a := range actions {
		if a == want {
			return true
		}
	}
	return false
}
