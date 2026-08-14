package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DeploySnapshot is the pre-deploy state for a single target.
// One snapshot per target per deploy. Rollback = restore from snapshots.
type DeploySnapshot struct {
	TargetID   string `json:"target_id"`
	LivePath   string `json:"live_path"`
	Content    []byte `json:"content,omitempty"`
	Hash       string `json:"hash"`
	Existed    bool   `json:"existed"`
}

// DeployRecord is one entry in deploy_history.jsonl.
type DeployRecord struct {
	DeployID    string          `json:"deploy_id"`
	Timestamp   string          `json:"timestamp"`
	Operator    string          `json:"operator"`
	Targets     []DeployTargetResult `json:"targets"`
	Status      string          `json:"status"` // success | partial | failed | dry-run
	DurationMs  int64           `json:"duration_ms"`
	FromDrift   string          `json:"from_drift,omitempty"` // path to drift report if any
}

// DeployTargetResult is one target's deploy outcome.
type DeployTargetResult struct {
	ID         string `json:"id"`
	LivePath   string `json:"live_path"`
	Status     string `json:"status"` // success | failed | skipped | dry-run
	HashBefore string `json:"hash_before"`
	HashAfter  string `json:"hash_after"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// snapshotDir returns the directory for deploy snapshots.
func snapshotDir(root string) string {
	return filepath.Join(root, ".ovav", "registry", "snapshots")
}

// createSnapshot reads the live file and stores it for rollback.
// If the live file doesn't exist, returns a snapshot with Existed=false.
func createSnapshot(root, deployID, targetID, livePath string) (DeploySnapshot, error) {
	snap := DeploySnapshot{
		TargetID: targetID,
		LivePath: livePath,
		Existed:  true,
	}
	data, err := os.ReadFile(livePath)
	if err != nil {
		if os.IsNotExist(err) {
			snap.Existed = false
			snap.Hash = ""
			return snap, nil
		}
		return snap, fmt.Errorf("read live: %w", err)
	}
	snap.Content = data
	snap.Hash = hashBytes(data)
	return snap, nil
}

// persistSnapshot writes the snapshot to disk for rollback.
func persistSnapshot(root, deployID string, snap DeploySnapshot) error {
	dir := filepath.Join(snapshotDir(root), deployID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	snapFile := filepath.Join(dir, snap.TargetID+".json")
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return os.WriteFile(snapFile, data, 0o644)
}

// atomicWriteLive writes content to livePath using WSL-safe pattern:
// 1. Write to sibling temp file (same FS)
// 2. Rename to livePath (atomic)
//
// Per the cross-FS bug (ADR-008), `mv` and `cp -f` from /tmp to /mnt/c
// silently fail. Path.replace() is the only reliable method.
func atomicWriteLive(livePath string, content []byte) error {
	dir := filepath.Dir(livePath)
	tmp, err := os.CreateTemp(dir, ".ovav-deploy-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Clean up temp if still around (shouldn't be after rename)
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, livePath); err != nil {
		// Fallback: try cp via python (last resort)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// verifyDeploy reads back the live file and verifies hash matches expected.
func verifyDeploy(livePath string, expectedContent []byte) error {
	actual, err := os.ReadFile(livePath)
	if err != nil {
		return fmt.Errorf("read live after deploy: %w", err)
	}
	expectedHash := hashBytes(expectedContent)
	actualHash := hashBytes(actual)
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash[:8], actualHash[:8])
	}
	return nil
}

// rollbackFromSnapshot restores live file from a snapshot.
func rollbackFromSnapshot(root, deployID string, snap DeploySnapshot) error {
	if !snap.Existed {
		// Original didn't exist — remove the deployed file
		if err := os.Remove(snap.LivePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove: %w", err)
		}
		return nil
	}
	return atomicWriteLive(snap.LivePath, snap.Content)
}

// hashBytes computes SHA-256 of a byte slice and returns hex.
func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// runPython is a helper to invoke python scripts.
func runPython(scriptPath string, args ...string) error {
	cmd := exec.Command("python3", append([]string{scriptPath}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// capturePython runs a python script and returns stdout/stderr.
func capturePython(scriptPath string, args ...string) (string, string, error) {
	cmd := exec.Command("python3", append([]string{scriptPath}, args...)...)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// readFragment reads the fragment file for a target.
func readFragment(root, fragmentRel string) ([]byte, error) {
	fragPath := filepath.Join(root, fragmentRel)
	return os.ReadFile(fragPath)
}

// generateDeployID creates a unique deploy ID with timestamp + random suffix.
func generateDeployID() string {
	t := time.Now().UTC().Format("20060102T150405")
	// Add short random suffix from time nanoseconds
	nanos := time.Now().UnixNano()
	return fmt.Sprintf("deploy-%s-%x", t, nanos&0xfffff)
}

// appendDeployHistory appends a deploy record to .ovav/registry/deploy_history.jsonl.
func appendDeployHistory(root string, record DeployRecord) error {
	dir := filepath.Join(root, ".ovav", "registry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "deploy_history.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("\n"))
	return err
}

// readDeployHistory loads all deploy records (most recent first).
func readDeployHistory(root string) ([]DeployRecord, error) {
	path := filepath.Join(root, ".ovav", "registry", "deploy_history.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []DeployRecord
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var rec DeployRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue // skip malformed
		}
		records = append(records, rec)
	}
	// Reverse to most-recent-first
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	return records, nil
}

// listSnapshots returns deploy IDs that have snapshots.
func listSnapshots(root string) ([]string, error) {
	dir := snapshotDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// loadSnapshot reads a snapshot file.
func loadSnapshot(root, deployID, targetID string) (DeploySnapshot, error) {
	var snap DeploySnapshot
	snapFile := filepath.Join(snapshotDir(root), deployID, targetID+".json")
	data, err := os.ReadFile(snapFile)
	if err != nil {
		return snap, err
	}
	err = json.Unmarshal(data, &snap)
	return snap, err
}

// drainReader reads all from r (used for streaming output).
func drainReader(r io.Reader) string {
	data, _ := io.ReadAll(r)
	return string(data)
}