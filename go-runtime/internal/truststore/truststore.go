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
	"time"
)

const (
	// GracePeriod is the duration after a git operation during which
	// gate hash changes are considered legitimate.
	GracePeriod = 5 * time.Minute

	stateFile        = ".ovav/runtime/gate_state.json"
	worktreeHeadsKey = ".ovav/runtime/worktree_heads.json"
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
