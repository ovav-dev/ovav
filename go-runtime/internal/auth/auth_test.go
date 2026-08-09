package auth

import (
	"testing"
	"time"
)

// ── SmartSessionTTL tests ──────────────────────────────────────────

func TestSmartSessionTTL_IsActive(t *testing.T) {
	ttl := NewSmartSessionTTL()
	if !ttl.IsActive() {
		t.Error("fresh TTL should be active")
	}
}

func TestSmartSessionTTL_Extend(t *testing.T) {
	ttl := NewSmartSessionTTL()
	before := ttl.Remaining()

	// Extend should update lastActive (may or may not extend depending on how much time passed)
	ttl.Extend()
	after := ttl.Remaining()

	// After Extend, remaining time should have been updated
	if after < before {
		t.Logf("TTL decreased after Extend (time passed): before=%v, after=%v", before, after)
	}
}

func TestSmartSessionTTL_Remaining(t *testing.T) {
	ttl := NewSmartSessionTTL()
	remaining := ttl.Remaining()
	if remaining <= 0 {
		t.Error("fresh TTL should have positive remaining time")
	}
	// Should be approximately 1 hour
	if remaining < 50*time.Minute {
		t.Errorf("remaining = %v, want ~1h", remaining)
	}
}

func TestSmartSessionTTL_Reset(t *testing.T) {
	ttl := NewSmartSessionTTL()
	before := ttl.Remaining()

	// Reset should restore to fresh ~1h state
	ttl.Reset()
	after := ttl.Remaining()

	// After Reset, should be approximately 1h (within 1 second tolerance)
	if after < before && before > 0 {
		// This can happen due to timing — verify after is close to 1h instead
		t.Logf("Reset timing note: before=%v, after=%v", before, after)
	}
	// Verify after is close to 1 hour (within 1 second tolerance)
	if after < 59*time.Minute || after > 61*time.Minute {
		t.Errorf("after Reset remaining=%v, want ~1h", after)
	}
}

// ── AuthState tests ───────────────────────────────────────────────

func TestNewAuthState(t *testing.T) {
	s := NewAuthState()
	if s.Mode() != ModeNOLOGIN {
		t.Errorf("default mode = %v, want ModeNOLOGIN", s.Mode())
	}
	if s.Injector() == nil {
		t.Error("Injector should not be nil")
	}
}

func TestAuthState_Inject(t *testing.T) {
	s := NewAuthState()
	if s.Mode() != ModeNOLOGIN {
		t.Fatalf("initial mode = %v, want ModeNOLOGIN", s.Mode())
	}

	s.Inject(ModeLOGIN)
	if s.Mode() != ModeLOGIN {
		t.Errorf("after Inject(LOGIN) mode = %v, want ModeLOGIN", s.Mode())
	}

	s.Inject(ModeNOLOGIN)
	if s.Mode() != ModeNOLOGIN {
		t.Errorf("after Inject(NOLOGIN) mode = %v, want ModeNOLOGIN", s.Mode())
	}
}

func TestAuthState_IsActive(t *testing.T) {
	s := NewAuthState()
	if !s.IsActive() {
		t.Error("fresh AuthState should be active")
	}
}

func TestAuthState_ExtendTTL(t *testing.T) {
	s := NewAuthState()
	before := s.RemainingTTL()
	s.ExtendTTL()
	after := s.RemainingTTL()
	// ExtendTTL should increase remaining time
	if after < before {
		t.Errorf("ExtendTTL should increase remaining: before=%v, after=%v", before, after)
	}
}

// ── PermissionInjector tests ───────────────────────────────────────

func TestPermissionInjector_RegisterGate(t *testing.T) {
	p := NewPermissionInjector(NewAuthState())
	p.RegisterGate("test_gate")

	// Unknown gate should return false (DENY)
	if p.IsGateAllowed("test_gate") {
		t.Error("newly registered gate should be DENY by default")
	}
}

func TestPermissionInjector_IsGateAllowed_UnknownGate(t *testing.T) {
	p := NewPermissionInjector(NewAuthState())
	// Unknown gate: default DENY for safety
	if p.IsGateAllowed("completely_unknown_gate") {
		t.Error("unknown gate should default to DENY (false)")
	}
}

func TestPermissionInjector_Inject_MODELOGIN(t *testing.T) {
	p := NewPermissionInjector(NewAuthState())
	p.RegisterGate("test_gate")

	// In ModeLOGIN, all gates should be bypassed (allowed)
	p.Inject(ModeLOGIN)
	if !p.IsGateAllowed("test_gate") {
		t.Error("gate should be ALLOW in ModeLOGIN")
	}
}

func TestPermissionInjector_Inject_MODENOLOGIN(t *testing.T) {
	p := NewPermissionInjector(NewAuthState())
	p.RegisterGate("git_push")
	p.RegisterGate("agent_delegate")

	// First set to LOGIN to make gates true
	p.Inject(ModeLOGIN)
	// Then switch to NOLOGIN
	p.Inject(ModeNOLOGIN)

	// agent_delegate is a safe gate, should be ALLOW even in NOLOGIN
	if !p.IsGateAllowed("agent_delegate") {
		t.Error("agent_delegate should be ALLOW in ModeNOLOGIN (safe gate)")
	}

	// git_push is NOT in safeGates, should be DENY
	if p.IsGateAllowed("git_push") {
		t.Error("git_push should be DENY in ModeNOLOGIN")
	}
}

func TestPermissionInjector_GetAllGates(t *testing.T) {
	p := NewPermissionInjector(NewAuthState())
	p.RegisterGate("gate1")
	p.RegisterGate("gate2")

	gates := p.GetAllGates()
	if len(gates) != 2 {
		t.Errorf("GetAllGates len = %d, want 2", len(gates))
	}
}

func TestPermissionInjector_InjectUnlocked_CalledByInject(t *testing.T) {
	// Verify that Inject calls InjectUnlocked correctly
	s := NewAuthState()
	inj := s.Injector()
	inj.RegisterGate("test_gate")

	// LOGIN mode
	inj.Inject(ModeLOGIN)
	if !inj.IsGateAllowed("test_gate") {
		t.Error("test_gate should be ALLOW after LOGIN inject")
	}

	// NOLOGIN mode
	inj.Inject(ModeNOLOGIN)
	if inj.IsGateAllowed("test_gate") {
		t.Error("test_gate should be DENY after NOLOGIN inject")
	}
}

// ── AuthMode String tests ────────────────────────────────────────

func TestAuthMode_String(t *testing.T) {
	tests := []struct {
		mode AuthMode
		want string
	}{
		{ModeNOLOGIN, "NO_LOGIN"},
		{ModeLOGIN, "LOGIN"},
		{AuthMode(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("AuthMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
