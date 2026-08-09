package ows

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── Policy Engine ───────────────────────────────────────────────────────────────

// PolicyLevel defines the strictness tier for worktree operations.
type PolicyLevel string

const (
	PolicyRelaxed  PolicyLevel = "relaxed"
	PolicyStandard PolicyLevel = "standard"
	PolicyStrict   PolicyLevel = "strict"
	PolicyWaiver   PolicyLevel = "waiver"
)

// Policy defines a single governance rule with versioning.
type Policy struct {
	ID      string                                             `json:"id"`      // POL-001, POL-002, etc.
	Version int                                                `json:"version"` // monotonically increasing
	Rule    string                                             `json:"rule"`    // human-readable description
	Level   PolicyLevel                                        `json:"level"`   // minimum level required
	Check   func(worktreeRoot, repoRoot string) (bool, string) `json:"-"`       // validation function
}

// PolicyEngine evaluates all applicable policies for an operation.
type PolicyEngine struct {
	policies map[string]Policy
	auditDB  *AuditDB
}

// NewPolicyEngine creates a policy engine with the 8 base rules.
func NewPolicyEngine(audit *AuditDB) *PolicyEngine {
	pe := &PolicyEngine{
		policies: make(map[string]Policy),
		auditDB:  audit,
	}
	pe.registerDefaults()
	return pe
}

func (pe *PolicyEngine) registerDefaults() {
	rules := []Policy{
		{
			ID: "POL-001", Version: 1, Level: PolicyStandard,
			Rule: "Protected branch — bloquea push/merge a main/master/develop sin waiver del CEO",
			Check: func(_, _ string) (bool, string) {
				return true, "protected branch enforcement delegated to protected_branch validator"
			},
		},
		{
			ID: "POL-002", Version: 1, Level: PolicyStandard,
			Rule: "HTTPS transport — bloquea remotes SSH:// y file://",
			Check: func(_, repoRoot string) (bool, string) {
				return checkRemoteHTTPS(repoRoot)
			},
		},
		{
			ID: "POL-003", Version: 1, Level: PolicyStandard,
			Rule: "No force push — bloquea --force, -f, --delete en cualquier rama",
			Check: func(_, _ string) (bool, string) {
				return true, "force push blocked by git push gate validator"
			},
		},
		{
			ID: "POL-004", Version: 1, Level: PolicyStandard,
			Rule: "Owner match — solo el creador del worktree puede hacer owd",
			Check: func(_, _ string) (bool, string) {
				return true, "owner match enforced by state machine"
			},
		},
		{
			ID: "POL-005", Version: 1, Level: PolicyStandard,
			Rule: "Verified gate — owd rechazado si owv no pasó",
			Check: func(worktreeRoot, _ string) (bool, string) {
				return checkVerifiedGate(worktreeRoot)
			},
		},
		{
			ID: "POL-006", Version: 1, Level: PolicyRelaxed,
			Rule: "Stale worktree — notifica si worktree >7d sin actividad",
			Check: func(_, _ string) (bool, string) {
				return true, "stale detection handled by state machine (EvStaleDetected)"
			},
		},
		{
			ID: "POL-007", Version: 1, Level: PolicyStrict,
			Rule: "Enterprise review — owd bloqueado sin approval de lead en perfil enterprise",
			Check: func(_, _ string) (bool, string) {
				return true, "enterprise review enforced by profile config RequireReview"
			},
		},
		{
			ID: "POL-008", Version: 1, Level: PolicyWaiver,
			Rule: "Emergency waiver — owx emergency requiere waiver HMAC firmado por CEO con TTL 60min",
			Check: func(_, _ string) (bool, string) {
				return true, "waiver validation handled by ValidateWaiver()"
			},
		},
	}

	for _, p := range rules {
		pe.policies[p.ID] = p
	}
}

// ValidateAll runs all applicable policies for the given level.
// Returns the first failing policy or nil if all pass.
func (pe *PolicyEngine) ValidateAll(level PolicyLevel, worktreeRoot, repoRoot string) error {
	for _, p := range pe.policies {
		if !p.appliesTo(level) {
			continue
		}
		passed, detail := p.Check(worktreeRoot, repoRoot)
		if !passed {
			return fmt.Errorf("POLICY BLOCKED: %s v%d — %s: %s", p.ID, p.Version, p.Rule, detail)
		}
	}
	return nil
}

// appliesTo checks if a policy should be enforced at the given check level.
// Waiver level bypasses all non-waiver policies.
func (p Policy) appliesTo(level PolicyLevel) bool {
	order := map[PolicyLevel]int{
		PolicyRelaxed:  0,
		PolicyStandard: 1,
		PolicyStrict:   2,
		PolicyWaiver:   3,
	}
	// At waiver level, only waiver policies apply
	if level == PolicyWaiver {
		return p.Level == PolicyWaiver
	}
	// Policy applies if its level is <= the check level (it's at most as strict)
	return order[p.Level] <= order[level]
}

// ── Waiver (POL-008) ────────────────────────────────────────────────────────────

// ErrWaiverSecretNotSet is returned when OVAV_WAIVER_SECRET env var is not configured.
var ErrWaiverSecretNotSet = fmt.Errorf("OVAV_WAIVER_SECRET not set — waivers disabled. Set the env var to enable emergency bypass.")

// WaiverSecret returns the HMAC key used to sign waivers.
// OWS-B3 FIX: No hardcoded fallback. If the env var is not set, waivers are disabled.
// The old hardcoded key "ovav-dev-waiver-key-change-in-production" was a security vulnerability.
func WaiverSecret() ([]byte, error) {
	if s := os.Getenv("OVAV_WAIVER_SECRET"); s != "" {
		return []byte(s), nil
	}
	return nil, ErrWaiverSecretNotSet
}

// Waiver represents a CEO-authorized emergency bypass.
type Waiver struct {
	ID        string `json:"id"`      // unique identifier
	Command   string `json:"command"` // owx, owd, etc.
	Target    string `json:"target"`  // worktree or branch
	Nonce     string `json:"nonce"`   // random nonce (prevents replay)
	ExpiresAt int64  `json:"exp"`     // Unix timestamp, max 60min from now
	Signature string `json:"sig"`     // HMAC-SHA256 of id:command:target:nonce:exp
}

// SignWaiver creates a signed waiver valid for the given duration.
// Returns nil if OVAV_WAIVER_SECRET is not configured (waivers disabled).
// CRITICAL FIX (C3): randomNonce() now returns error — propagated instead of panic.
func SignWaiver(command, target string, ttl time.Duration) *Waiver {
	nonce, err := randomNonce()
	if err != nil {
		// crypto/rand failure is catastrophic — waivers cannot be signed
		return nil
	}
	w := &Waiver{
		ID:        fmt.Sprintf("w-%d", time.Now().UnixNano()),
		Command:   command,
		Target:    target,
		Nonce:     nonce,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}
	sig, err := w.computeHMAC()
	if err != nil {
		return nil
	}
	w.Signature = sig
	return w
}

// ValidateWaiver checks a waiver's signature, expiry, and target.
func ValidateWaiver(w *Waiver, expectedCommand, expectedTarget string) error {
	// Check expiry
	if time.Now().Unix() > w.ExpiresAt {
		return fmt.Errorf("waiver expired at %s", time.Unix(w.ExpiresAt, 0).Format(time.RFC3339))
	}

	// Check TTL — max 60 minutes
	if w.ExpiresAt-time.Now().Unix() > 3600 {
		return fmt.Errorf("waiver TTL exceeds 60 minutes")
	}

	// Check command match
	if expectedCommand != "" && w.Command != expectedCommand {
		return fmt.Errorf("waiver command %q does not match %q", w.Command, expectedCommand)
	}

	// Check target match
	if expectedTarget != "" && w.Target != expectedTarget {
		return fmt.Errorf("waiver target %q does not match %q", w.Target, expectedTarget)
	}

	// Verify HMAC signature
	expected, err := w.computeHMAC()
	if err != nil {
		return fmt.Errorf("waiver verification failed: %w", err)
	}
	if !hmac.Equal([]byte(w.Signature), []byte(expected)) {
		return fmt.Errorf("waiver signature invalid — possible tampering or forgery")
	}

	return nil
}

// computeHMAC generates the HMAC-SHA256 signature for this waiver.
// OWS-B3 FIX: Returns error if WaiverSecret is not configured.
func (w *Waiver) computeHMAC() (string, error) {
	secret, err := WaiverSecret()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%s:%s:%s:%s:%d", w.ID, w.Command, w.Target, w.Nonce, w.ExpiresAt)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// LoadWaiver reads a waiver from the OVAV runtime directory.
func LoadWaiver(repoRoot string) (*Waiver, error) {
	path := filepath.Join(repoRoot, ".ovav", "runtime", "protected_branch_waiver.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("waiver not found: %w", err)
	}
	return parseWaiverYAML(data)
}

// parseWaiverYAML extracts waiver fields from a simple YAML-like format.
func parseWaiverYAML(data []byte) (*Waiver, error) {
	// Simple parser for our YAML waiver format:
	// waiver_id: w-xxx
	// command: owx
	// target: main
	// nonce: abc123
	// expires_at: 1234567890
	// signature: hexhash
	w := &Waiver{}
	lines := string(data)
	for _, line := range splitLines(lines) {
		parts := splitColon(line)
		if len(parts) < 2 {
			continue
		}
		key := trimSpace(parts[0])
		val := trimSpace(parts[1])
		switch key {
		case "waiver_id":
			w.ID = val
		case "command":
			w.Command = val
		case "target":
			w.Target = val
		case "nonce":
			w.Nonce = val
		case "expires_at":
			fmt.Sscanf(val, "%d", &w.ExpiresAt)
		case "signature":
			w.Signature = val
		}
	}
	if w.ID == "" {
		return nil, fmt.Errorf("invalid waiver: missing waiver_id")
	}
	return w, nil
}

// ── Lock System ──────────────────────────────────────────────────────────────────

// LockTTL is the maximum time a worktree can remain locked.
const LockTTL = 24 * time.Hour

// LockWorktree locks a worktree. Only the owner or a lead can lock.
// Locks auto-expire after LockTTL.
func (a *AuditDB) LockWorktree(worktreeID, reason, requestedBy string) error {
	wt, err := a.LoadWorktree(worktreeID)
	if err != nil {
		return fmt.Errorf("worktree not found: %s", worktreeID)
	}

	// Check lock permission: owner or same org
	if wt.Owner != requestedBy {
		return fmt.Errorf("lock denied: %s is not the owner of %s (owner: %s)", requestedBy, worktreeID, wt.Owner)
	}

	// Check if lock already expired
	if wt.Locked {
		elapsed := time.Since(wt.UpdatedAt)
		if elapsed > LockTTL {
			// Auto-unlock expired lock
			wt.Locked = false
			wt.LockReason = ""
			wt.State = StateActive
		} else {
			return fmt.Errorf("worktree %s already locked by %s (expires in %s)", worktreeID, wt.Owner, (LockTTL - elapsed).Round(time.Minute))
		}
	}

	wt.Locked = true
	wt.LockReason = reason
	if err := ExecuteTransition(wt, EvLockRequested); err != nil {
		// If transition fails, still lock but keep current state
		wt.State = StateLocked
	}
	wt.UpdatedAt = time.Now()

	return a.SaveWorktree(*wt)
}

// UnlockWorktree unlocks a worktree. Only the owner or CEO (force) can unlock.
func (a *AuditDB) UnlockWorktree(worktreeID, requestedBy string, force bool) error {
	wt, err := a.LoadWorktree(worktreeID)
	if err != nil {
		return fmt.Errorf("worktree not found: %s", worktreeID)
	}

	if !wt.Locked && !force {
		return fmt.Errorf("worktree %s is not locked", worktreeID)
	}

	// Only owner or force (CEO) can unlock
	if wt.Owner != requestedBy && !force {
		return fmt.Errorf("unlock denied: %s is not the owner of %s (owner: %s). Use --force for CEO override.", requestedBy, worktreeID, wt.Owner)
	}

	wt.Locked = false
	wt.LockReason = ""
	if err := ExecuteTransition(wt, EvUnlockRequested); err != nil {
		wt.State = StateActive
	}
	wt.UpdatedAt = time.Now()

	return a.SaveWorktree(*wt)
}

// ExpireStaleLocks unlocks all worktrees locked longer than LockTTL.
// Called by ows maintenance.
func (a *AuditDB) ExpireStaleLocks() (int, error) {
	wts, err := a.ListWorktrees(StateLocked, "")
	if err != nil {
		return 0, err
	}

	unlocked := 0
	for _, wt := range wts {
		if time.Since(wt.UpdatedAt) > LockTTL {
			wt.Locked = false
			wt.LockReason = ""
			wt.State = StateActive
			wt.UpdatedAt = time.Now()
			if err := a.SaveWorktree(wt); err == nil {
				unlocked++
			}
		}
	}
	return unlocked, nil
}

// ── Policy Check Implementations ─────────────────────────────────────────────────

func checkRemoteHTTPS(repoRoot string) (bool, string) {
	// Delegates to git remote check — only HTTPS allowed
	remotes, err := listGitRemotes(repoRoot)
	if err != nil {
		return true, "" // can't check, allow (git push gate validator will catch)
	}
	for _, url := range remotes {
		if !isHTTPS(url) && !isEmptyURL(url) {
			return false, fmt.Sprintf("non-HTTPS remote detected: %s", url)
		}
	}
	return true, ""
}

func checkVerifiedGate(worktreeRoot string) (bool, string) {
	// OWS-B6 FIX: Proper JSON parsing instead of fragile string matching.
	// Now Verify() (owv) persists results to .ovav/verify/last_result.json.
	resultsPath := filepath.Join(worktreeRoot, ".ovav", "verify", "last_result.json")
	data, err := os.ReadFile(resultsPath)
	if err != nil {
		return false, "no verification results found — run owv first"
	}
	var vr VerifyResult
	if err := json.Unmarshal(data, &vr); err != nil {
		return false, fmt.Sprintf("corrupt verification results: %v — run owv again", err)
	}
	if !vr.Passed {
		return false, fmt.Sprintf("verification did not pass: %s", vr.Detail)
	}
	return true, ""
}

// ── Helpers ──────────────────────────────────────────────────────────────────────

// CRITICAL FIX (C3): randomNonce now returns error instead of panic().
// Panic in crypto path → DoS vector. Callers handle the error gracefully.
// Previously: panic() killed the entire governor process on /dev/urandom failure.
func randomNonce() (string, error) {
	// OWS-B2 FIX: Use crypto/rand for cryptographic nonces.
	// The old implementation used time.Now().UnixNano() + os.Getpid(),
	// which is predictable and enables replay attacks on waivers.
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("randomNonce: crypto/rand.Read failed: %w", err)
	}
	return hex.EncodeToString(b)[:16], nil
}

func listGitRemotes(repoRoot string) ([]string, error) {
	// OWS-B1 FIX: Actually run git remote -v and parse the URLs.
	// The old implementation was a stub that returned nil, nil,
	// making POL-002 (HTTPS enforcement) completely non-functional.
	cmd := exec.Command("git", "-C", repoRoot, "remote", "-v")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listGitRemotes: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var urls []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			urls = append(urls, fields[1])
		}
	}
	return urls, nil
}

func isHTTPS(url string) bool {
	return len(url) > 8 && (url[:8] == "https://" || url[:7] == "http://")
}

func isEmptyURL(url string) bool {
	return url == "" || url == "none"
}

func containsString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitColon(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
