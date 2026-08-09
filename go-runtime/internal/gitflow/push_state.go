// Package gitflow provides OVAV git v3.0 workflow commands.
package gitflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PushState records the progress of an owd push operation so that it can be
// resumed after a timeout or network failure.
type PushState struct {
	SessionID string            `json:"session_id"` // unique ID for this owd run
	Branch    string            `json:"branch"`     // source branch being merged+pushed
	Targets   []PushTargetState `json:"targets"`    // per-target push state
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// PushTargetState records the push status for a single target branch.
type PushTargetState struct {
	Name        string `json:"name"`                   // branch name e.g. "develop"
	Status      string `json:"status"`                 // "pending", "pushed", "failed"
	Ref         string `json:"ref,omitempty"`          // SHA pushed (or last attempted)
	LastAttempt string `json:"last_attempt,omitempty"` // RFC3339 of last attempt
	Error       string `json:"error,omitempty"`        // error message from last attempt
}

// pushStatePath returns the path to the push state file for the given repo root.
func pushStatePath(repoRoot string) string {
	return filepath.Join(repoRoot, ".ovav", "runtime", "push_state.json")
}

// LoadPushState reads the current push state file, if it exists and is valid.
func LoadPushState(repoRoot string) (*PushState, error) {
	path := pushStatePath(repoRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state PushState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SavePushState atomically writes the push state to disk.
func SavePushState(repoRoot string, state *PushState) error {
	dir := filepath.Join(repoRoot, ".ovav", "runtime")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir push_state dir: %w", err)
	}
	state.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal push state: %w", err)
	}
	tmp := pushStatePath(repoRoot) + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write push_state tmp: %w", err)
	}
	if err := os.Rename(tmp, pushStatePath(repoRoot)); err != nil {
		return fmt.Errorf("rename push_state: %w", err)
	}
	return nil
}

// ClearPushState removes the push state file. Call on success.
func ClearPushState(repoRoot string) {
	os.Remove(pushStatePath(repoRoot))
}

// InitPushState creates a new PushState for a branch with all targets set to "pending".
func InitPushState(repoRoot, sessionID, branch string, targetNames []string) (*PushState, error) {
	targets := make([]PushTargetState, len(targetNames))
	for i, name := range targetNames {
		targets[i] = PushTargetState{Name: name, Status: "pending"}
	}
	state := &PushState{
		SessionID: sessionID,
		Branch:    branch,
		Targets:   targets,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := SavePushState(repoRoot, state); err != nil {
		return nil, err
	}
	return state, nil
}

// MarkTargetPushed updates a single target as successfully pushed.
func MarkTargetPushed(repoRoot string, state *PushState, targetName, ref string) error {
	for i := range state.Targets {
		if state.Targets[i].Name == targetName {
			state.Targets[i].Status = "pushed"
			state.Targets[i].Ref = ref
			state.Targets[i].LastAttempt = time.Now().UTC().Format(time.RFC3339)
			state.Targets[i].Error = ""
			return SavePushState(repoRoot, state)
		}
	}
	return fmt.Errorf("target %q not found in push state", targetName)
}

// MarkTargetFailed updates a single target as failed with an error message.
func MarkTargetFailed(repoRoot string, state *PushState, targetName, ref, errMsg string) error {
	for i := range state.Targets {
		if state.Targets[i].Name == targetName {
			state.Targets[i].Status = "failed"
			state.Targets[i].Ref = ref
			state.Targets[i].LastAttempt = time.Now().UTC().Format(time.RFC3339)
			state.Targets[i].Error = errMsg
			return SavePushState(repoRoot, state)
		}
	}
	return fmt.Errorf("target %q not found in push state", targetName)
}

// PendingTargets returns the names of targets that are still "pending".
func (s *PushState) PendingTargets() []string {
	var pending []string
	for _, t := range s.Targets {
		if t.Status == "pending" {
			pending = append(pending, t.Name)
		}
	}
	return pending
}

// FailedTargets returns the names of targets that are "failed".
func (s *PushState) FailedTargets() []string {
	var failed []string
	for _, t := range s.Targets {
		if t.Status == "failed" {
			failed = append(failed, t.Name)
		}
	}
	return failed
}

// HasFailures returns true if any target is in "failed" state.
func (s *PushState) HasFailures() bool {
	for _, t := range s.Targets {
		if t.Status == "failed" {
			return true
		}
	}
	return false
}

// IsComplete returns true if all targets are either "pushed" or "failed".
func (s *PushState) IsComplete() bool {
	for _, t := range s.Targets {
		if t.Status == "pending" {
			return false
		}
	}
	return true
}
