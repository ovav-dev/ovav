// Package data provides data loading for the Cockpit.
package data

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CapsData represents the canonical caps.yaml structure.
type CapsData struct {
	Version     string         `yaml:"version"`
	UpdatedAt   string         `yaml:"updated_at"`
	UpdatedBy   string         `yaml:"updated_by"`
	PlanVersion string         `yaml:"plan_version"`
	Strategy    string         `yaml:"strategy"`
	StackTarget StackTarget    `yaml:"stack_target"`
	Caps        map[string]Cap `yaml:"caps"`
	Pending     []PendingCap   `yaml:"pending"`
}

// StackTarget defines the Go+TS strategy.
type StackTarget struct {
	Go         string `yaml:"go"`
	TypeScript string `yaml:"typescript"`
	Python     string `yaml:"python"`
}

// Cap represents a completed capability.
type Cap struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Status   string `yaml:"status"`
	Pct      int    `yaml:"pct"`
	Items    int    `yaml:"items"`
	Merge    string `yaml:"merge"`
	MergedAt string `yaml:"merged_at"`
	Summary  string `yaml:"summary"`
}

// PendingCap represents a capability not yet merged.
type PendingCap struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Status   string   `yaml:"status"`
	Pct      int      `yaml:"pct"`
	Order    int      `yaml:"order"`
	Deps     []string `yaml:"deps"`
	Worktree string   `yaml:"worktree"`
	Commit   string   `yaml:"commit"`
	Summary  string   `yaml:"summary"`
	Stack    string   `yaml:"stack"`
	Tasks    []string `yaml:"tasks"`
}

// LoadCaps reads and parses caps.yaml from the OVAV root.
// root is the OVAV repo root (2 levels above go-runtime/).
func LoadCaps(root string) (*CapsData, error) {
	path := filepath.Join(root, ".ovav", "plan", "caps.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read caps.yaml: %w", err)
	}

	var caps CapsData
	if err := yaml.Unmarshal(b, &caps); err != nil {
		return nil, fmt.Errorf("parse caps.yaml: %w", err)
	}
	return &caps, nil
}
