package validators

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// safeFixRegistry is the whitelist of validators that can auto-fix.
// Per ADR-011 (auto-remediation architecture).
//
// Adding a new entry requires:
//  1. Implementing Fixable on the validator
//  2. Adding comment "// SAFE_FIX: <description>" above the struct
//  3. Adding the ID + description here with appropriate risk level
var safeFixRegistry = []SafeFixEntry{
	{
		ValidatorID: "bash_readline_bindings",
		Description: "Add 'deliberately UNBOUND' marker to ~/.inputrc if missing",
		RiskLevel:   "low",
	},
	{
		ValidatorID: "runtime_integrity_baseline_fresh",
		Description: "Regenerate baseline.json with current file hashes",
		RiskLevel:   "low",
	},
	{
		ValidatorID: "supply_chain",
		Description: "Regenerate sbom.json to match current tracked files",
		RiskLevel:   "low",
	},
}

// GetSafeFixRegistry returns the safe-fix whitelist (copy).
func GetSafeFixRegistry() []SafeFixEntry {
	out := make([]SafeFixEntry, len(safeFixRegistry))
	copy(out, safeFixRegistry)
	return out
}

// IsSafeFix returns true if the validator ID is whitelisted for auto-fix.
func IsSafeFix(validatorID string) bool {
	for _, entry := range safeFixRegistry {
		if entry.ValidatorID == validatorID {
			return true
		}
	}
	return false
}

// FixRegistrySnapshot creates a snapshot of the current registry for rollback.
// Files included: any file matching <root>/**/{files in safeFixRegistry entries}.
// Returns snapshot path.
func FixRegistrySnapshot(root string, entries []SafeFixEntry) (string, error) {
	deployID := fmt.Sprintf("fix-%d", time.Now().UnixNano())
	snapDir := filepath.Join(root, ".ovav", "registry", "snapshots", deployID)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return "", err
	}

	manifest := struct {
		DeployID  string            `json:"deploy_id"`
		Timestamp string            `json:"timestamp"`
		Files     map[string]string `json:"files"`
	}{
		DeployID:  deployID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Files:     map[string]string{},
	}

	// Snapshot inputrc (bash readline target)
	for _, entry := range entries {
		if entry.ValidatorID == "bash_readline_bindings" {
			inputrc := filepath.Join(os.Getenv("HOME"), ".inputrc")
			if data, err := os.ReadFile(inputrc); err == nil {
				manifest.Files[inputrc] = string(data)
			}
		}
		if entry.ValidatorID == "runtime_integrity_baseline_fresh" {
			baseline := filepath.Join(root, ".ovav", "integrity_backups", "baseline.json")
			if data, err := os.ReadFile(baseline); err == nil {
				manifest.Files[baseline] = string(data)
			}
		}
		if entry.ValidatorID == "supply_chain" {
			sbom := filepath.Join(root, ".ovav", "registry", "sbom.json")
			if data, err := os.ReadFile(sbom); err == nil {
				manifest.Files[sbom] = string(data)
			}
		}
	}

	manifestPath := filepath.Join(snapDir, "manifest.json")
	data, _ := jsonMarshalIndent(manifest)
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return "", err
	}
	return snapDir, nil
}

// FixRegistryRollback restores all files from a snapshot manifest.
func FixRegistryRollback(snapDir string) error {
	manifestPath := filepath.Join(snapDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest struct {
		Files map[string]string `json:"files"`
	}
	if err := jsonUnmarshal(data, &manifest); err != nil {
		return err
	}
	for path, content := range manifest.Files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// FixResultLog is one entry in the auto-fix history.
type FixResultLog struct {
	DeployID   string      `json:"deploy_id"`
	Operator   string      `json:"operator"`
	Results    []FixResult `json:"results"`
	Outcome    string      `json:"outcome"` // success | partial | failed | dry-run
	StartedAt  string      `json:"started_at"`
	DurationMs int64       `json:"duration_ms"`
}

// AppendFixHistory appends a fix result to .ovav/registry/auto_fix_history.jsonl.
func AppendFixHistory(root string, log FixResultLog) error {
	dir := filepath.Join(root, ".ovav", "registry")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "auto_fix_history.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := jsonMarshalIndent(log)
	f.Write(data)
	f.Write([]byte("\n"))
	return nil
}

// ReadFixHistory reads all fix history entries.
func ReadFixHistory(root string) ([]FixResultLog, error) {
	path := filepath.Join(root, ".ovav", "registry", "auto_fix_history.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var logs []FixResultLog
	for _, line := range splitLinesString(string(data)) {
		if line == "" {
			continue
		}
		var l FixResultLog
		if err := jsonUnmarshal([]byte(line), &l); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// JSON helpers (avoid importing encoding/json here to keep file small)
func jsonMarshalIndent(v interface{}) ([]byte, error) {
	return jsonMarshalHelper(v)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return jsonUnmarshalHelper(data, v)
}
