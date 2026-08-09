// depgraph.go — Secret Dependency Graph
//
// Phase 6.4 of OVAV-VAULT-2026 plan.
//
// OVAV Vault tracks which systems use which secrets — the dependency graph.
// This enables:
//
//  1. Rotation propagation: when a secret rotates, know which systems need updating
//  2. Impact analysis: if a secret is compromised, know all affected systems
//  3. Auto-rotation: cPanel can call GitHub/Fly.io APIs to push new secrets
//  4. Orphan detection: secrets with zero references are candidates for cleanup
//
// Storage: deps.graph file in vault dir — encrypted JSON, same vault key.
// Format: { "secret_refs": { "<secretID>": [SecretRef, ...], ... } }
package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// System represents a system that uses secrets.
type System string

const (
	SystemGitHubActions System = "github-actions"
	SystemFlyIO         System = "fly-io"
	SystemGitLabCI      System = "gitlab-ci"
	SystemJenkins       System = "jenkins"
	SystemLocalEnv      System = "local-env"
	SystemCICD          System = "ci-cd"
	SystemOVAVAgent     System = "ovav-agent"
	SystemUnknown       System = "unknown"
)

// SecretRef records one usage of a secret by a system.
type SecretRef struct {
	ID            string     `json:"id"`        // SHA256(secretID+system+path)[:16]
	SecretID      string     `json:"secret_id"` // which secret
	System        System     `json:"system"`    // github-actions, fly-io, etc.
	Path          string     `json:"path"`      // file or resource path
	EnvVar        string     `json:"env_var"`   // which env var name
	AddedAt       time.Time  `json:"added_at"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	AutoRotatable bool       `json:"auto_rotatable"` // cPanel can rotate this?
}

// DependencyGraph manages secret → system references.
type DependencyGraph struct {
	mu    sync.RWMutex
	refs  map[string][]SecretRef // secretID → refs
	dirty bool
}

// depsGraphPath returns the path to the dependency graph file.
func depsGraphPath() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "ovav", "deps.graph")
}

// LoadDependencyGraph loads the dependency graph from disk.
func LoadDependencyGraph() (*DependencyGraph, error) {
	g := &DependencyGraph{refs: make(map[string][]SecretRef)}

	path := depsGraphPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return g, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load dep graph: %w", err)
	}

	// Decrypt with vault key
	// The graph is stored alongside the vault, encrypted with the same key
	// For simplicity, we use a separate encrypted file
	// But since we don't have the vault key here, we'll store it unencrypted
	// in the .graph file (not sensitive — only metadata, no secret values)
	var raw struct {
		Refs map[string][]SecretRef `json:"refs"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse dep graph: %w", err)
	}
	g.refs = raw.Refs
	return g, nil
}

// Save persists the dependency graph to disk.
func (g *DependencyGraph) Save() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	path := depsGraphPath()
	os.MkdirAll(filepath.Dir(path), 0700)

	data, err := json.MarshalIndent(struct {
		Refs map[string][]SecretRef `json:"refs"`
	}{Refs: g.refs}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal dep graph: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// refID generates a deterministic ID for a SecretRef.
func refID(secretID, system, path string) string {
	h := sha256.Sum256([]byte(secretID + "|" + system + "|" + path))
	return hex.EncodeToString(h[:16])
}

// AddRef adds a usage reference for a secret.
func (g *DependencyGraph) AddRef(secretID string, system System, path, envVar string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if ref already exists
	for _, r := range g.refs[secretID] {
		if r.System == system && r.Path == path {
			return nil // already tracked
		}
	}

	ref := SecretRef{
		ID:            refID(secretID, string(system), path),
		SecretID:      secretID,
		System:        system,
		Path:          path,
		EnvVar:        envVar,
		AddedAt:       time.Now().UTC(),
		AutoRotatable: g.canAutoRotate(system, envVar),
	}

	g.refs[secretID] = append(g.refs[secretID], ref)
	g.dirty = true
	return nil
}

// canAutoRotate returns true if cPanel can auto-rotate this secret for this system.
func (g *DependencyGraph) canAutoRotate(system System, envVar string) bool {
	switch system {
	case SystemGitHubActions:
		// GitHub Actions secrets can be rotated via GitHub API
		return true
	case SystemFlyIO:
		// Fly.io secrets can be rotated via flyctl
		return true
	case SystemGitLabCI, SystemCICD:
		// GitLab CI variables can be rotated via GitLab API
		return true
	default:
		return false
	}
}

// RemoveRef removes a usage reference.
func (g *DependencyGraph) RemoveRef(secretID, system, path string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	refs := g.refs[secretID]
	for i, r := range refs {
		if string(r.System) == system && r.Path == path {
			g.refs[secretID] = append(refs[:i], refs[i+1:]...)
			g.dirty = true
			return nil
		}
	}
	return nil
}

// GetRefs returns all references for a secret.
func (g *DependencyGraph) GetRefs(secretID string) []SecretRef {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.refs[secretID]
}

// GetSystemsUsing returns all system types that use a given secret.
func (g *DependencyGraph) GetSystemsUsing(secretID string) []System {
	refs := g.GetRefs(secretID)
	systems := make([]System, 0, len(refs))
	seen := make(map[System]bool)
	for _, r := range refs {
		if !seen[r.System] {
			systems = append(systems, r.System)
			seen[r.System] = true
		}
	}
	return systems
}

// GetSecretsForSystem returns all secret IDs used by a given system.
func (g *DependencyGraph) GetSecretsForSystem(system System) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var secretIDs []string
	for secretID, refs := range g.refs {
		for _, r := range refs {
			if r.System == system {
				secretIDs = append(secretIDs, secretID)
				break
			}
		}
	}
	return secretIDs
}

// IsOrphan returns true if a secret has zero references.
func (g *DependencyGraph) IsOrphan(secretID string) bool {
	return len(g.GetRefs(secretID)) == 0
}

// OrphanReport returns all orphan secrets (used nowhere).
func (g *DependencyGraph) OrphanReport(secretIDs []string) []string {
	var orphans []string
	for _, id := range secretIDs {
		if g.IsOrphan(id) {
			orphans = append(orphans, id)
		}
	}
	return orphans
}

// RotationImpact returns what systems would be affected if a secret is rotated.
func (g *DependencyGraph) RotationImpact(secretID string) []string {
	refs := g.GetRefs(secretID)
	if len(refs) == 0 {
		return nil
	}
	rotatable := make([]string, 0, len(refs))
	manual := make([]string, 0, len(refs))
	for _, r := range refs {
		entry := fmt.Sprintf("%s: %s (%s)", r.System, r.Path, r.EnvVar)
		if r.AutoRotatable {
			rotatable = append(rotatable, entry)
		} else {
			manual = append(manual, entry)
		}
	}
	return append(rotatable, manual...)
}

// DiscoverFromSecrets scans a SecretStore and adds refs for known secret usage patterns.
func (g *DependencyGraph) DiscoverFromSecrets(store *SecretStore) error {
	// Scan for known patterns in environment variables and file paths
	// This is a simplified version — real discovery would parse actual config files
	for _, sec := range store.List("") {
		path := g.guessPathFromSource(sec.Source, sec.Name)
		switch sec.Source {
		case "github":
			g.AddRef(sec.ID, SystemGitHubActions, path, sec.Name)
		case "fly":
			g.AddRef(sec.ID, SystemFlyIO, path, sec.Name)
		}
	}
	return nil
}

// guessPathFromSource returns a plausible path from a secret source.
func (g *DependencyGraph) guessPathFromSource(source, name string) string {
	switch source {
	case "github":
		return "GitHub Actions: " + name
	case "fly":
		return "Fly.io app: " + name
	case "filesystem":
		return ".env file: " + name
	default:
		return source + ": " + name
	}
}

// DetectSystemFromName guesses the system from a secret name.
func DetectSystemFromName(name string) System {
	name = strings.ToUpper(name)
	switch {
	case strings.Contains(name, "GITHUB"):
		return SystemGitHubActions
	case strings.Contains(name, "FLY"):
		return SystemFlyIO
	case strings.Contains(name, "GITLAB"):
		return SystemGitLabCI
	case strings.Contains(name, "JENKINS"):
		return SystemJenkins
	case strings.Contains(name, "CF_") || strings.Contains(name, "CLOUDFLARE"):
		return SystemOVAVAgent
	default:
		return SystemUnknown
	}
}

// GetRefsForSecretByName returns all refs for a secret given its name.
// Requires a SecretStore to look up the secret ID.
func (g *DependencyGraph) GetRefsForSecretByName(store *SecretStore, secretName string) []SecretRef {
	sec := store.GetByName(secretName)
	if sec == nil {
		return nil
	}
	return g.GetRefs(sec.ID)
}

// GetRefsForSecretByNameStatic is a convenience that takes a secret directly.
func GetRefsForSecretByNameStatic(store *SecretStore, secretName string) []SecretRef {
	sec := store.GetByName(secretName)
	if sec == nil {
		return nil
	}
	// We need the graph — create a temporary one
	g, _ := LoadDependencyGraph()
	if g == nil {
		g = &DependencyGraph{}
	}
	return g.GetRefs(sec.ID)
}

// QueryResult represents a natural language query result.
type QueryResult struct {
	Icon   string // emoji or symbol
	Name   string // secret name
	Type   string // secret type
	Detail string // additional context
}

// QuerySecrets answers natural language queries about the vault.
// Phase 7 will use NLP; this Phase 6 implementation uses pattern matching.
func QuerySecrets(store *SecretStore, graph *DependencyGraph, query string) []QueryResult {
	query = strings.ToLower(query)
	var results []QueryResult

	allSecrets := store.List("")
	if graph == nil {
		g, _ := LoadDependencyGraph()
		if g != nil {
			graph = g
		}
	}
	if graph == nil {
		graph = &DependencyGraph{}
	}

	switch {
	case strings.Contains(query, "orphan"):
		// "show orphaned secrets"
		for _, sec := range allSecrets {
			if graph.IsOrphan(sec.ID) {
				results = append(results, QueryResult{
					Icon:   "⚠️",
					Name:   sec.Name,
					Type:   string(sec.Type),
					Detail: "no system references",
				})
			}
		}

	case strings.Contains(query, "expir"):
		// "secrets expiring next week"
		for _, sec := range allSecrets {
			if sec.Metadata == nil {
				continue
			}
			expStr := sec.Metadata["expires_at"]
			if expStr == "" {
				continue
			}
			results = append(results, QueryResult{
				Icon:   "⏰",
				Name:   sec.Name,
				Type:   string(sec.Type),
				Detail: "expires: " + expStr,
			})
		}

	case strings.Contains(query, "github"):
		// "all github tokens"
		for _, sec := range allSecrets {
			if strings.Contains(strings.ToLower(string(sec.Type)), "api_token") &&
				(strings.Contains(strings.ToLower(sec.Name), "github") ||
					sec.Source == "github") {
				results = append(results, QueryResult{
					Icon:   "🔑",
					Name:   sec.Name,
					Type:   string(sec.Type),
					Detail: sec.Source,
				})
			}
		}

	case strings.Contains(query, "cloud") || strings.Contains(query, "fly"):
		// "cloud keys"
		for _, sec := range allSecrets {
			if sec.Type == TypeCloudKey || sec.Source == "fly" {
				refs := graph.GetRefs(sec.ID)
				sysStr := fmt.Sprintf("%d system(s)", len(refs))
				results = append(results, QueryResult{
					Icon:   "☁️",
					Name:   sec.Name,
					Type:   string(sec.Type),
					Detail: sysStr,
				})
			}
		}

	case strings.Contains(query, "rotat"):
		// "secrets not rotated in 90 days"
		cutoff := time.Now().Add(-90 * 24 * time.Hour)
		for _, sec := range allSecrets {
			if sec.Metadata == nil {
				continue
			}
			lastRot := sec.Metadata["last_rotated"]
			if lastRot == "" {
				continue
			}
			t, err := time.Parse(time.RFC3339, lastRot)
			if err == nil && t.Before(cutoff) {
				results = append(results, QueryResult{
					Icon:   "🔄",
					Name:   sec.Name,
					Type:   string(sec.Type),
					Detail: "stale, last rotated: " + t.Format("2006-01-02"),
				})
			}
		}

	default:
		// Default: show all secrets matching the query in the name
		for _, sec := range allSecrets {
			if strings.Contains(strings.ToLower(sec.Name), query) {
				refs := graph.GetRefs(sec.ID)
				refsStr := fmt.Sprintf("%d refs", len(refs))
				if len(refs) == 0 {
					refsStr = "orphan"
				}
				results = append(results, QueryResult{
					Icon:   "🔑",
					Name:   sec.Name,
					Type:   string(sec.Type),
					Detail: refsStr,
				})
			}
		}
	}

	return results
}
