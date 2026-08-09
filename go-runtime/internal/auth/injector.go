package auth

import (
	"sync"
	"time"
)

// AuthMode represents the current authentication/authorization mode.
type AuthMode int

const (
	// ModeNOLOGIN is the default mode: full OVAV gates active, guided
	// verification instead of blocking. All gates return gate-specific
	// defaults (DENY by default, ALLOW for safe read operations).
	ModeNOLOGIN AuthMode = iota
	// ModeLOGIN bypasses all gates — all permissions are allowed.
	// Use only for fully authenticated CEO sessions.
	ModeLOGIN
)

// String returns a human-readable name for the auth mode.
func (m AuthMode) String() string {
	switch m {
	case ModeLOGIN:
		return "LOGIN"
	case ModeNOLOGIN:
		return "NO_LOGIN"
	default:
		return "UNKNOWN"
	}
}

// AuthState is the global authentication state singleton.
// It coordinates between the PermissionInjector, SmartSessionTTL, and the rest of the system.
type AuthState struct {
	mu       sync.RWMutex
	mode     AuthMode
	injector *PermissionInjector
	ttl      *SmartSessionTTL
}

// NewAuthState creates a new AuthState with default ModeNOLOGIN and SmartSessionTTL.
func NewAuthState() *AuthState {
	s := &AuthState{
		mode: ModeNOLOGIN,
		ttl:  NewSmartSessionTTL(),
	}
	s.injector = &PermissionInjector{
		state: s,
		gates: make(map[string]bool),
	}
	// Register canonical OVAV gates
	s.injector.RegisterGate("git_push")
	s.injector.RegisterGate("git_merge")
	s.injector.RegisterGate("protected_branch")
	s.injector.RegisterGate("worktree_write")
	s.injector.RegisterGate("agent_delegate")
	s.injector.RegisterGate("skill_exec_high_impact")
	return s
}

// Mode returns the current auth mode.
func (s *AuthState) Mode() AuthMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// Inject applies the given auth mode, updating all gate permissions.
// ModeLOGIN: all gates → ALLOW (bypass all checks)
// ModeNOLOGIN: gates active with safe operations allowed
func (s *AuthState) Inject(mode AuthMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
	s.injector.InjectUnlocked(mode)
}

// Injector returns the PermissionInjector for this AuthState.
func (s *AuthState) Injector() *PermissionInjector {
	return s.injector
}

// ExtendTTL extends the session TTL by 1 hour (capped at 8h max).
// Safe to call from multiple goroutines.
func (s *AuthState) ExtendTTL() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ttl.Extend()
}

// RemainingTTL returns the time left in the current TTL window.
// Returns 0 if the session has expired.
func (s *AuthState) RemainingTTL() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ttl.Remaining()
}

// IsActive returns true if the session TTL is still valid.
func (s *AuthState) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ttl.IsActive()
}

// ResetTTL resets the session TTL to a fresh 1 hour window.
func (s *AuthState) ResetTTL() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ttl.Reset()
}

// PermissionInjector manages gate-level permission bypass.
type PermissionInjector struct {
	state *AuthState      // reference to parent AuthState
	gates map[string]bool // gate name → allowed (true = bypassed/allowed)
	mu    sync.RWMutex
}

// NewPermissionInjector creates a new injector linked to the given AuthState.
func NewPermissionInjector(state *AuthState) *PermissionInjector {
	return &PermissionInjector{
		state: state,
		gates: make(map[string]bool),
	}
}

// RegisterGate registers a new gate with default DENY (false).
// Safe operations should be explicitly allowed after registration.
func (p *PermissionInjector) RegisterGate(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.gates[name]; !ok {
		p.gates[name] = false
	}
}

// InjectUnlocked injects permissions based on mode (must hold lock).
// Visible for testing and use within AuthState.
func (p *PermissionInjector) InjectUnlocked(mode AuthMode) {
	switch mode {
	case ModeLOGIN:
		// All gates are bypassed in LOGIN mode
		for k := range p.gates {
			p.gates[k] = true
		}
	case ModeNOLOGIN:
		// Reset all gates to default: DENY for high-impact, ALLOW for safe reads
		safeGates := map[string]bool{
			"agent_delegate": true, // delegation itself is safe; content may not be
		}
		for k := range p.gates {
			if safeGates[k] {
				p.gates[k] = true
			} else {
				p.gates[k] = false
			}
		}
	}
}

// Inject injects permissions globally based on the given mode.
// ModeLOGIN: all gates → ALLOW (bypass all checks)
// ModeNOLOGIN: gates active, safe ops allowed, high-impact ops DENY
func (p *PermissionInjector) Inject(mode AuthMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.InjectUnlocked(mode)
}

// IsGateAllowed returns true if the gate is bypassed (allowed) in the current mode.
// Returns false for unknown gates (gates must be registered before use).
func (p *PermissionInjector) IsGateAllowed(gateName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	allowed, ok := p.gates[gateName]
	if !ok {
		// Unknown gate: default to DENY (false) for safety
		return false
	}
	return allowed
}

// GetAllGates returns a list of all registered gate names.
func (p *PermissionInjector) GetAllGates() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	gates := make([]string, 0, len(p.gates))
	for k := range p.gates {
		gates = append(gates, k)
	}
	return gates
}
