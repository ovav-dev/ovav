// Package truststore provides OVAV runtime trust primitives for gate integrity
// and per-worktree HEAD tracking. It is the security foundation for:
//   - GateSelfProtection validator: detects tampering with host defense gates
//   - HeadIntegrity validator: maintains per-worktree trusted HEAD references
//
// # Grace Period
//
// Gate hash changes within 5 minutes of a git operation are treated as legitimate
// developer workflow (rebase, merge, worktree operations), not compromise.
// The grace period is reset every time OWS records a git operation via RecordGitOp.
package truststore

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ovav/ovav/internal/identity"
)

const (
	// GracePeriod is the duration after a git operation during which
	// gate hash changes are considered legitimate.
	GracePeriod = 5 * time.Minute

	stateFile        = ".ovav/runtime/gate_state.json"
	worktreeHeadsKey = ".ovav/runtime/worktree_heads.json"

	// gateRelPath is the repo-relative path of the protected gate file.
	gateRelPath = "go-runtime/internal/validators/host_config_drift.go"
)

// GateState tracks the gate file hash and the last git operation timestamp.
// Stored at .ovav/runtime/gate_state.json.
type GateState struct {
	GateSHA256       string `json:"gate_sha256"`
	LastGitOpTime    int64  `json:"last_git_op_time"`    // Unix timestamp
	LastGitOpReflog  string `json:"last_git_op_reflog"`  // Reflog entry describing the op
}

// GateFileSHA256 computes the SHA-256 hash of the given file path.
// Used to detect whether a gate file has been modified.
func GateFileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// refreshMu serialises RefreshGateHash calls across goroutines so the
// stored hash and last_git_op_time never tear under concurrent refresh.
var refreshMu sync.Mutex

// statePath returns the path to the gate state file within repoRoot.
func statePath(repoRoot string) string {
	return filepath.Join(repoRoot, stateFile)
}

// worktreeHeadsPath returns the path to the worktree heads file within repoRoot.
func worktreeHeadsPath(repoRoot string) string {
	return filepath.Join(repoRoot, worktreeHeadsKey)
}

// RecordGitOp records the current time and git reflog entry as the last git operation.
// Called by OWS handlers after create, done, and merge operations to open the grace period.
func RecordGitOp(repoRoot string) error {
	// Get git reflog to capture what operation just happened
	reflog := gitReflog(repoRoot)

	state := ReadGateState(repoRoot)
	state.LastGitOpTime = time.Now().Unix()
	state.LastGitOpReflog = reflog

	return WriteGateState(repoRoot, state)
}

// gitReflog returns the most recent git reflog entry for HEAD, or "unknown" if unavailable.
func gitReflog(repoRoot string) string {
	// git reflog show -1 --format=%gd %gs  →  "HEAD@{0} commit: ..."
	out, err := runGit(repoRoot, "reflog", "show", "-1", "--format=%gs")
	if err != nil {
		return "git-op"
	}
	trimmed := trimLines(out)
	if trimmed == "" {
		return "git-op"
	}
	// Return just the operation description, not the reflog syntax
	return trimmed
}

// ReadGateState loads the gate state from .ovav/runtime/gate_state.json.
// Returns an empty GateState if the file does not exist.
func ReadGateState(repoRoot string) GateState {
	p := statePath(repoRoot)
	data, err := os.ReadFile(p)
	if err != nil {
		return GateState{}
	}
	var state GateState
	if err := json.Unmarshal(data, &state); err != nil {
		return GateState{}
	}
	return state
}

// WriteGateState persists the gate state to .ovav/runtime/gate_state.json.
func WriteGateState(repoRoot string, state GateState) error {
	dir := filepath.Dir(statePath(repoRoot))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(repoRoot), data, 0644)
}

// RefreshGateHash recomputes the SHA-256 of the protected gate file and
// publishes a new GateState atomically. It preserves LastGitOpTime when the
// current value still falls within GracePeriod; otherwise it resets the value
// to zero so a future validator sees an explicit "no recent op" baseline.
//
// Returns the previous and new hash prefixes (full SHA-256 each) so the caller
// can report the change. Refuses to operate when the gate file itself, or the
// parent directory of the state file, is a symlink — that would defeat the
// point of an atomic, directory-FD-checked publish.
//
// Concurrent callers are serialised by an internal mutex so the file is
// never observed in a half-written state by a parallel refresh.
func RefreshGateHash(repoRoot string) (prevHash, nextHash string, err error) {
	refreshMu.Lock()
	defer refreshMu.Unlock()

	gateAbs := filepath.Join(repoRoot, gateRelPath)

	// 1. Refuse symlink gate file — the trust target must be a real file.
	info, err := os.Lstat(gateAbs)
	if err != nil {
		return "", "", fmt.Errorf("stat gate file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("gate file is a symlink: %s", gateAbs)
	}

	// 2. Recompute hash from the on-disk file.
	nextHash = GateFileSHA256(gateAbs)
	if nextHash == "" {
		return "", "", fmt.Errorf("could not hash gate file: %s", gateAbs)
	}

	// 3. Parent-directory recheck — refuse if .ovav/runtime is a symlink.
	stateFileAbs := statePath(repoRoot)
	parentDir := filepath.Dir(stateFileAbs)
	pinfo, err := os.Lstat(parentDir)
	if err != nil {
		return "", "", fmt.Errorf("stat state parent dir: %w", err)
	}
	if pinfo.Mode()&os.ModeSymlink != 0 {
		return "", "", fmt.Errorf("state parent dir is a symlink: %s", parentDir)
	}

	// 4. Read existing state and decide grace preservation.
	state := ReadGateState(repoRoot)
	prevHash = state.GateSHA256
	state.GateSHA256 = nextHash

	if state.LastGitOpTime != 0 {
		elapsed := time.Since(time.Unix(state.LastGitOpTime, 0))
		if elapsed >= GracePeriod {
			// Outside grace — reset so the next validator reports
			// "no recent git op" rather than a stale timestamp.
			state.LastGitOpTime = 0
			state.LastGitOpReflog = ""
		}
		// else: preserve; refresh is happening inside the developer grace window.
	}

	// 5. Marshal and publish atomically.
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return prevHash, "", fmt.Errorf("marshal gate state: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return prevHash, "", fmt.Errorf("mkdir state dir: %w", err)
	}

	if err := identity.SecureAtomicReplace(stateFileAbs, data); err != nil {
		return prevHash, "", fmt.Errorf("atomic replace gate state: %w", err)
	}

	return prevHash, nextHash, nil
}

// GateRelPath returns the repo-relative path of the protected gate file.
// Useful for callers (CLI, validators) that need to reference it.
func GateRelPath() string {
	return gateRelPath
}

// GracePeriodOk returns true if the system is currently within the grace period
// after the last recorded git operation. During grace, gate hash changes are
// considered legitimate developer activity (rebase, merge, worktree ops).
func GracePeriodOk(repoRoot string) (bool, error) {
	state := ReadGateState(repoRoot)
	if state.LastGitOpTime == 0 {
		// No git op recorded — outside grace
		return false, nil
	}
	elapsed := time.Since(time.Unix(state.LastGitOpTime, 0))
	return elapsed < GracePeriod, nil
}

// ReadWorktreeHeads returns the map of worktree path → trusted HEAD SHA.
// Returns an empty map if the file does not exist.
func ReadWorktreeHeads(repoRoot string) (map[string]string, error) {
	p := worktreeHeadsPath(repoRoot)
	data, err := os.ReadFile(p)
	if err != nil {
		return map[string]string{}, nil // not an error — just no heads recorded yet
	}
	var heads map[string]string
	if err := json.Unmarshal(data, &heads); err != nil {
		return map[string]string{}, err
	}
	return heads, nil
}

// WriteWorktreeHead records the trusted HEAD for a specific worktree.
// Creates the worktree_heads.json if it does not exist.
func WriteWorktreeHead(repoRoot, worktreePath, headSHA string) error {
	heads, _ := ReadWorktreeHeads(repoRoot) // empty map on error — start fresh
	if heads == nil {
		heads = map[string]string{}
	}
	heads[worktreePath] = headSHA
	return writeWorktreeHeads(repoRoot, heads)
}

// RemoveWorktreeHead removes the trusted HEAD entry for a worktree.
// Idempotent — succeeds even if the worktree is not in the map.
func RemoveWorktreeHead(repoRoot, worktreePath string) error {
	heads, err := ReadWorktreeHeads(repoRoot)
	if err != nil {
		return err // propagate real errors
	}
	if heads == nil {
		return nil
	}
	delete(heads, worktreePath)
	return writeWorktreeHeads(repoRoot, heads)
}

// writeWorktreeHeads persists the worktree heads map to disk.
func writeWorktreeHeads(repoRoot string, heads map[string]string) error {
	dir := filepath.Dir(worktreeHeadsPath(repoRoot))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(heads, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(worktreeHeadsPath(repoRoot), data, 0644)
}

// trimLines removes leading/trailing whitespace and blank lines.
func trimLines(s string) string {
	for {
		if len(s) == 0 {
			return s
		}
		if s[0] == '\n' {
			s = s[1:]
		} else if s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		} else {
			break
		}
	}
	return s
}

// runGit executes a git command in repoRoot and returns stdout.
func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
