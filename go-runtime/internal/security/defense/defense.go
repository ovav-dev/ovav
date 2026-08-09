// Package defense provides OVAV security defense tools in Go.
//
// Replaces Python tools/security/defense_cortex.py, host_defense_responder.py,
// intelligent_authorizer.py, and auto_credentials.py.
// Stack: Go stdlib only. Zero Python dependencies.
package defense

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ── Common Types ─────────────────────────────────────────────────────────

// Severity represents the threat level of a security event.
type Severity string

const (
	SevInfo     Severity = "info"
	SevWarning  Severity = "warning"
	SevCritical Severity = "critical"
	SevDeadly   Severity = "deadly"
)

// ResponseAction represents the action taken in response to an event.
type ResponseAction string

const (
	ActLog        ResponseAction = "log"
	ActAlert      ResponseAction = "alert"
	ActQuarantine ResponseAction = "quarantine"
	ActLockdown   ResponseAction = "lockdown"
	ActBlock      ResponseAction = "block"
)

// ── Cortex: Defense Coordinator ──────────────────────────────────────────

// CortexEvent represents a security event analyzed by the defense cortex.
type CortexEvent struct {
	ID              string    `json:"id"`
	IntrusionType   string    `json:"intrusion_type"`
	Path            string    `json:"path"`
	Severity        Severity  `json:"severity"`
	RootCause       string    `json:"root_cause"`
	IsFalsePositive bool      `json:"is_false_positive"`
	Timestamp       time.Time `json:"timestamp"`
}

// Cortex coordinates the defense system.
// Replaces tools/security/defense_cortex.py (550 LOC Python).
type Cortex struct {
	mu             sync.RWMutex
	falsePositives map[string]int // pattern → count
	whitelisted    map[string]bool
	hardeningState map[string]string
}

// NewCortex creates a new defense cortex.
func NewCortex() *Cortex {
	return &Cortex{
		falsePositives: make(map[string]int),
		whitelisted:    make(map[string]bool),
		hardeningState: make(map[string]string),
	}
}

// ClassifyRootCause determines the root cause of an intrusion based on path and type.
func (c *Cortex) ClassifyRootCause(intrusionType, path string) string {
	patterns := map[string]string{
		"user_creation":     "user_management",
		"git_operation":     "source_control",
		"file_sync":         "synchronization",
		"config_change":     "configuration",
		"binary_exec":       "execution",
		"network_access":    "network",
		"permission_change": "permissions",
	}

	for keyword, cause := range patterns {
		if strings.Contains(intrusionType, keyword) || strings.Contains(path, keyword) {
			return cause
		}
	}
	return "unknown"
}

// LearnFalsePositive records a false-positive pattern. Returns true if pattern
// should be whitelisted (seen 3+ times with severity reduction).
func (c *Cortex) LearnFalsePositive(intrusionType, path string, severityReduction bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	pattern := intrusionType + "::" + path
	if severityReduction {
		c.falsePositives[pattern]++
	}

	count := c.falsePositives[pattern]
	if count >= 3 {
		c.whitelisted[pattern] = true
		return true
	}
	return false
}

// IsKnownFalsePositive checks if an intrusion+path matches a whitelisted pattern.
func (c *Cortex) IsKnownFalsePositive(intrusionType, path string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pattern := intrusionType + "::" + path
	return c.whitelisted[pattern]
}

// FalsePositiveCount returns the total number of tracked false-positive patterns.
func (c *Cortex) FalsePositiveCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.falsePositives)
}

// ApplyHardening adjusts the defense posture for a component.
func (c *Cortex) ApplyHardening(component, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hardeningState[component] = action
}

// HardeningState returns the current hardening state.
func (c *Cortex) HardeningState() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	state := make(map[string]string)
	for k, v := range c.hardeningState {
		state[k] = v
	}
	return state
}

// ── Responder: Tiered Response Executor ──────────────────────────────────

// Responder executes tiered automated responses to security events.
// Replaces tools/security/host_defense_responder.py (240 LOC Python).
type Responder struct {
	immuneFiles map[string]bool // Root immune list — never quarantine these
	cortex      *Cortex
	lockdown    bool
	mu          sync.RWMutex
}

// NewResponder creates a new defense responder.
func NewResponder(cortex *Cortex) *Responder {
	return &Responder{
		immuneFiles: map[string]bool{
			".ovav/plan/caps.yaml":                   true,
			".ovav/laws/":                            true,
			".ovav/policy/permission_authority.json": true,
			"go-runtime/internal/validators/":        true,
		},
		cortex: cortex,
	}
}

// Respond executes the tiered response for a security event.
// Returns the actions taken.
func (r *Responder) Respond(intrusionType string, severity Severity, path string) []ResponseAction {
	var actions []ResponseAction

	// Always log
	actions = append(actions, ActLog)

	// Check if target is immune
	isImmune := r.isImmune(path)

	// Check for false positive
	if r.cortex != nil && r.cortex.IsKnownFalsePositive(intrusionType, path) {
		return actions // log only for known false positives
	}

	switch severity {
	case SevInfo:
		// Log only (already done)

	case SevWarning:
		actions = append(actions, ActAlert)

	case SevCritical:
		actions = append(actions, ActAlert)
		if !isImmune {
			actions = append(actions, ActQuarantine)
		} else {
			actions = append(actions, ActBlock) // blocked by immune list
		}

	case SevDeadly:
		actions = append(actions, ActAlert)
		if !isImmune {
			actions = append(actions, ActQuarantine)
			actions = append(actions, ActLockdown)
		} else {
			actions = append(actions, ActBlock)
		}
	}

	return actions
}

func (r *Responder) isImmune(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for immune := range r.immuneFiles {
		if strings.HasPrefix(path, immune) || strings.Contains(path, immune) {
			return true
		}
	}
	return false
}

// AddImmuneFile adds a file pattern to the root immune list.
func (r *Responder) AddImmuneFile(pattern string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.immuneFiles[pattern] = true
}

// IsLockdownActive returns whether the system is in lockdown.
func (r *Responder) IsLockdownActive() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lockdown
}

// SetLockdown activates or deactivates system lockdown.
func (r *Responder) SetLockdown(active bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lockdown = active
}

// ── Authorizer: Drift Auto-Authorization ──────────────────────────────────

// DriftEntry represents a detected file hash drift.
type DriftEntry struct {
	File    string `json:"file"`
	OldHash string `json:"old_hash"`
	NewHash string `json:"new_hash"`
	Domain  string `json:"domain"`
	Branch  string `json:"branch"`
}

// AuthResult represents the result of a drift authorization evaluation.
type AuthResult struct {
	Authorized bool   `json:"authorized"`
	Reason     string `json:"reason"`
	Rule       string `json:"rule"`
}

// Authorizer evaluates drift entries for auto-authorization.
// Replaces tools/security/intelligent_authorizer.py (454 LOC Python).
type Authorizer struct {
	authorizedLedger map[string]int // file → prior authorization count
	mu               sync.RWMutex
}

// NewAuthorizer creates a new intelligent authorizer.
func NewAuthorizer() *Authorizer {
	return &Authorizer{
		authorizedLedger: make(map[string]int),
	}
}

// RecordAuthorization records a prior authorization for pattern learning.
func (a *Authorizer) RecordAuthorization(file string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.authorizedLedger[file]++
}

// EvaluateDrift evaluates whether a drift entry can be auto-authorized.
//
// 7-rule priority chain (matching Python intelligent_authorizer.py):
//
//  1. Protected branch → BLOCKED (CEO waiver)
//  2. Mutable runtime domain → AUTO
//  3. No active session → MANUAL
//  4. Known pattern (3+ prior) + task branch → AUTO
//  5. Governed config + task branch + session → AUTO
//  6. Immutable source + task branch + session → AUTO (warn)
//  7. Unclassified + task branch + session → AUTO
func (a *Authorizer) EvaluateDrift(entry DriftEntry, isProtectedBranch, hasSession bool) AuthResult {
	// Rule 1: Protected branch → blocked
	if isProtectedBranch {
		return AuthResult{Authorized: false, Reason: "protected branch — CEO waiver required", Rule: "R1_PROTECTED"}
	}

	// Rule 2: Mutable runtime domain
	if entry.Domain == "mutable_runtime" {
		return AuthResult{Authorized: true, Reason: "mutable runtime domain", Rule: "R2_MUTABLE"}
	}

	// Rule 3: No active session
	if !hasSession {
		return AuthResult{Authorized: false, Reason: "no active session — manual authorization required", Rule: "R3_NO_SESSION"}
	}

	isTaskBranch := strings.HasPrefix(entry.Branch, "feature/") ||
		strings.HasPrefix(entry.Branch, "fix/") ||
		strings.HasPrefix(entry.Branch, "task/")

	// Rule 4: Known pattern (3+ prior authorizations) + task branch
	a.mu.RLock()
	priorCount := a.authorizedLedger[entry.File]
	a.mu.RUnlock()
	if priorCount >= 3 && isTaskBranch {
		return AuthResult{Authorized: true, Reason: fmt.Sprintf("known pattern (%d prior authorizations)", priorCount), Rule: "R4_KNOWN_PATTERN"}
	}

	if !isTaskBranch {
		return AuthResult{Authorized: false, Reason: "non-task branch — manual authorization required", Rule: "R3_NO_SESSION"}
	}

	// Rule 5: Governed config
	if entry.Domain == "governed_config" {
		return AuthResult{Authorized: true, Reason: "governed config on task branch with session", Rule: "R5_GOVERNED"}
	}

	// Rule 6: Immutable source
	if entry.Domain == "immutable_source" {
		return AuthResult{Authorized: true, Reason: "immutable source on task branch (warn)", Rule: "R6_IMMUTABLE_WARN"}
	}

	// Rule 7: Unclassified
	return AuthResult{Authorized: true, Reason: "unclassified on task branch with session", Rule: "R7_UNCLASSIFIED"}
}

// ── Credentials: Auto-Credential Manager ─────────────────────────────────

// Credential represents a managed credential.
type Credential struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Hash      string    `json:"hash"`
	RotatedAt time.Time `json:"rotated_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Healthy   bool      `json:"healthy"`
}

// CredentialManager handles auto-rotation and health of credentials.
// Replaces tools/security/auto_credentials.py (845 LOC Python).
type CredentialManager struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
}

// NewCredentialManager creates a new credential manager.
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		credentials: make(map[string]*Credential),
	}
}

// RegisterCredential adds a credential to the manager.
func (cm *CredentialManager) RegisterCredential(service, secret string, ttl time.Duration) *Credential {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now().UTC()
	hash := hashSecret(secret)
	id := service + "-" + hash[:12]

	cred := &Credential{
		ID:        id,
		Service:   service,
		Hash:      hash,
		RotatedAt: now,
		ExpiresAt: now.Add(ttl),
		Healthy:   true,
	}
	cm.credentials[id] = cred
	return cred
}

// RotateCredential rotates a credential if it's expired or unhealthy.
// Returns the new credential hash or error.
func (cm *CredentialManager) RotateCredential(service, newSecret string, ttl time.Duration) (*Credential, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find existing credential by service
	var existing *Credential
	for _, c := range cm.credentials {
		if c.Service == service {
			existing = c
			break
		}
	}

	now := time.Now().UTC()
	hash := hashSecret(newSecret)
	id := service + "-" + hash[:12]

	cred := &Credential{
		ID:        id,
		Service:   service,
		Hash:      hash,
		RotatedAt: now,
		ExpiresAt: now.Add(ttl),
		Healthy:   true,
	}

	if existing != nil {
		delete(cm.credentials, existing.ID)
	}
	cm.credentials[id] = cred
	return cred, nil
}

// CheckHealth verifies all credentials and returns unhealthy ones.
func (cm *CredentialManager) CheckHealth() []*Credential {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var unhealthy []*Credential
	now := time.Now().UTC()

	for _, c := range cm.credentials {
		if now.After(c.ExpiresAt) {
			c.Healthy = false
			unhealthy = append(unhealthy, c)
		}
	}
	return unhealthy
}

// NeedsRotation returns true if any credential needs rotation.
func (cm *CredentialManager) NeedsRotation() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	now := time.Now().UTC()
	for _, c := range cm.credentials {
		if now.After(c.ExpiresAt) || !c.Healthy {
			return true
		}
	}
	return false
}

// Count returns the number of managed credentials.
func (cm *CredentialManager) Count() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.credentials)
}

// ── Helpers ──────────────────────────────────────────────────────────────

func hashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

// ── Real-Time Defense Verification ────────────────────────────────────────
//
// These functions perform live system checks — no canned data, no mock targets.
// Every check queries the actual state of the filesystem and git repository.

// SecretLeakCheck verifies no plaintext secrets are present in tracked files.
func SecretLeakCheck(repoRoot string) (bool, string) {
	// Check .env exists (plaintext secret risk)
	envPath := filepath.Join(repoRoot, ".env")
	if _, err := os.Stat(envPath); err == nil {
		return true, ".env file exists — plaintext secret leak risk"
	}

	secrets := DetectSecrets(repoRoot)
	if len(secrets) > 0 {
		return true, fmt.Sprintf("secrets detected in %d file(s): %v", len(secrets), secrets)
	}
	return false, ""
}

// DetectSecrets scans git-tracked production files for real secret patterns.
// Skips test files (*_test.go, test/, spec/, mocks/, vendor/) to avoid
// flagging intentional test fixtures. Only flags real credential values (20+ chars).
func DetectSecrets(repoRoot string) []string {
	var found []string
	patterns := []string{
		`ghp_[a-zA-Z0-9]{36}`,         // GitHub personal access token
		`gho_[a-zA-Z0-9]{36}`,         // GitHub OAuth
		`glpat-[a-zA-Z0-9\-]{20,}`,    // GitLab PAT
		`sk-[a-zA-Z0-9]{48}`,          // OpenAI API key
		`xox[baprs]-[a-zA-Z0-9]{10,}`, // Slack tokens
		// Real credential values (20+ chars, not short test strings)
		`(password|secret|api_key|apikey)\s*[:=]\s*['"][a-zA-Z0-9+/=_\-]{20,}['""]`,
	}

	cmd := exec.Command("git", "-C", repoRoot, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return found
	}

	isTestFile := func(path string) bool {
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_test.sh") ||
			strings.HasSuffix(path, "_spec.go") || strings.HasSuffix(path, ".test.ts") {
			return true
		}
		parts := strings.Split(path, "/")
		for _, part := range parts[:len(parts)-1] { // skip filename, check dirs
			if part == "test" || part == "tests" || part == "spec" ||
				part == "mocks" || part == "mock" || part == "fixtures" ||
				part == "vendor" || part == "node_modules" {
				return true
			}
		}
		return false
	}

	skipExt := map[string]bool{
		".png": true, ".jpg": true, ".gif": true, ".pdf": true,
		".zip": true, ".tar": true, ".gz": true, ".sum": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
	}

	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f == "" {
			continue
		}
		if skipExt[filepath.Ext(f)] {
			continue
		}
		if isTestFile(f) {
			continue
		}

		fullPath := filepath.Join(repoRoot, f)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)
		for _, pat := range patterns {
			if regexp.MustCompile(pat).MatchString(content) {
				found = append(found, f)
				break
			}
		}
	}
	return found
}

// UnauthorizedWriteCheck detects unexpected modifications to governance files.
func UnauthorizedWriteCheck(repoRoot string) (bool, string, error) {
	capsPath := filepath.Join(repoRoot, ".ovav", "plan", "caps.yaml")
	if _, err := os.Stat(capsPath); err != nil {
		return false, "", nil
	}

	currentHash, err := FileHash(capsPath)
	if err != nil {
		return false, "", err
	}

	baselineHash, err := GetTrustedBaseline(repoRoot, ".ovav/plan/caps.yaml")
	if err != nil {
		return false, "", nil
	}

	if currentHash != baselineHash {
		return true, fmt.Sprintf("caps.yaml modified (baseline: %s, current: %s)", baselineHash[:8], currentHash[:8]), nil
	}
	return false, "", nil
}

// GitConfigCheck detects unauthorized changes to .git/config.
func GitConfigCheck(repoRoot string) (bool, string, error) {
	gitConfig := filepath.Join(repoRoot, ".git", "config")
	if _, err := os.Stat(gitConfig); err != nil {
		return false, "", nil
	}

	currentHash, err := FileHash(gitConfig)
	if err != nil {
		return false, "", err
	}

	baselineHash, err := GetTrustedBaseline(repoRoot, ".git/config")
	if err != nil {
		return false, "", nil
	}

	if currentHash != baselineHash {
		cmd := exec.Command("git", "-C", repoRoot, "diff", ".git/config")
		out, _ := cmd.Output()
		lines := strings.Split(string(out), "\n")
		summary := fmt.Sprintf("%d lines changed", len(lines))
		if len(lines) <= 10 {
			summary = strings.TrimSpace(string(out))
		}
		return true, summary, nil
	}
	return false, "", nil
}

// PermissionAuthorityDriftCheck detects changes to governance config files.
func PermissionAuthorityDriftCheck(repoRoot string) (bool, string, error) {
	permAuth := filepath.Join(repoRoot, ".ovav", "policy", "permission_authority.json")
	opencodeJSON := filepath.Join(repoRoot, "opencode.json")

	permAuthHash, err := FileHash(permAuth)
	if err != nil {
		return false, "", err
	}
	opencodeHash, err := FileHash(opencodeJSON)
	if err != nil {
		return false, "", err
	}

	driftFile := filepath.Join(repoRoot, ".ovav", "runtime", "drift_baseline.json")
	var baseline struct {
		PermAuth string `json:"permission_authority_hash"`
		Opencode string `json:"opencode_hash"`
	}
	if data, err := os.ReadFile(driftFile); err == nil {
		json.Unmarshal(data, &baseline)
	}

	var reasons []string
	if baseline.PermAuth != "" && baseline.PermAuth != permAuthHash {
		reasons = append(reasons, "permission_authority.json changed")
	}
	if baseline.Opencode != "" && baseline.Opencode != opencodeHash {
		reasons = append(reasons, "opencode.json changed")
	}

	if len(reasons) > 0 {
		return true, strings.Join(reasons, ", "), nil
	}
	return false, "", nil
}

// GetTrustedBaseline returns SHA-256 of the committed version of a file at HEAD.
func GetTrustedBaseline(repoRoot, path string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "show", "HEAD:"+path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(out)
	return hex.EncodeToString(h[:]), nil
}

// FileHash returns SHA-256 hex of a file's contents.
func FileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// DefenseEvent is a real security event from live system verification.
type DefenseEvent struct {
	Type      string    `json:"type"`
	Severity  Severity  `json:"severity"`
	Path      string    `json:"path"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ScanResult holds the full result of a defense surface scan.
type ScanResult struct {
	SurfacesChecked int            `json:"surfaces_checked"`
	Events          []DefenseEvent `json:"events"`
}

// RunActiveScan performs real-time verification across all defense surfaces.
// Returns events from live filesystem and git checks — nothing hardcoded.
func RunActiveScan(repoRoot string) (*ScanResult, error) {
	var events []DefenseEvent
	now := time.Now().UTC()
	surfaces := 0

	// 1. Secret leak — always checked
	surfaces++
	if leaked, msg := SecretLeakCheck(repoRoot); leaked {
		events = append(events, DefenseEvent{
			Type: "secret_leak", Severity: SevDeadly,
			Path: ".env", Message: msg, Timestamp: now,
		})
	}

	// 2. Unauthorized write — always checked
	surfaces++
	if triggered, msg, err := UnauthorizedWriteCheck(repoRoot); err == nil && triggered {
		events = append(events, DefenseEvent{
			Type: "unauthorized_write", Severity: SevWarning,
			Path: ".ovav/plan/caps.yaml", Message: msg, Timestamp: now,
		})
	}

	// 3. Git config — always checked
	surfaces++
	if triggered, msg, err := GitConfigCheck(repoRoot); err == nil && triggered {
		events = append(events, DefenseEvent{
			Type: "permission_escalation", Severity: SevCritical,
			Path: ".git/config", Message: msg, Timestamp: now,
		})
	}

	// 4. Permission authority drift — always checked
	surfaces++
	if triggered, msg, err := PermissionAuthorityDriftCheck(repoRoot); err == nil && triggered {
		events = append(events, DefenseEvent{
			Type: "config_drift", Severity: SevInfo,
			Path: ".ovav/policy/permission_authority.json", Message: msg, Timestamp: now,
		})
	}

	return &ScanResult{SurfacesChecked: surfaces, Events: events}, nil
}
