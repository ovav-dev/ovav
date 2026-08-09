// Package sync — OVAV Product Sync Engine
//
// GOV-009: Detects changes in agents, skills, configs, and CLI tools
// staged for OVAV Product distribution. Creates a sync manifest,
// manages the queue of pending items, and connects to cPanel for
// protected dispatch to end-user Product installations.
//
// Architecture:
//
//	OVAV Systems (Cockpit) → cPanel (protected) → Sync Queue
//	OVAV Product (installed) → check cPanel → pull queue
//
// The sync engine is the intermediary bridge — it never pushes directly
// to Product. Instead, it stages changes in cPanel's protected queue,
// and Product pulls from there when the user accepts the update.
package sync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── Types ───────────────────────────────────────────────────────────

// SyncItem represents a single change staged for distribution.
type SyncItem struct {
	ID       string    `json:"id"`
	Path     string    `json:"path"`     // relative to OVAV root
	Category string    `json:"category"` // "agent" | "skill" | "config" | "tool" | "cli"
	Action   string    `json:"action"`   // "add" | "update" | "remove"
	Checksum string    `json:"checksum"` // SHA-256 for integrity verification
	Size     int64     `json:"size"`     // bytes
	StagedAt time.Time `json:"staged_at"`
	Synced   bool      `json:"synced"`
	SyncedAt time.Time `json:"synced_at,omitempty"`
}

// SyncManifest records all changes staged and their sync status.
type SyncManifest struct {
	Version    string     `json:"version"` // OVAV Product target version
	Generated  time.Time  `json:"generated"`
	Source     string     `json:"source"` // OVAV root path
	Items      []SyncItem `json:"items"`
	TotalItems int        `json:"total_items"`
	Synced     int        `json:"synced"`
	Pending    int        `json:"pending"`
}

// SyncResult is returned after a sync operation completes.
type SyncResult struct {
	ItemsStaged int       `json:"items_staged"`
	ItemsSynced int       `json:"items_synced"`
	Errors      []string  `json:"errors,omitempty"`
	Manifest    string    `json:"manifest_path,omitempty"`
	CompletedAt time.Time `json:"completed_at"`
}

// ── Directories to scan for changes ─────────────────────────────────

// validSyncExtensions lists the file extensions considered sync-worthy.
// Anything else (test fixtures, vendor, backups, dotfiles) is skipped.
var validSyncExtensions = map[string]bool{
	".yaml": true,
	".yml":  true,
	".json": true,
	".md":   true,
	".go":   true,
	".sh":   true,
	".ps1":  true,
	".toml": true,
}

// syncDirectories lists the OVAV source paths tracked for distribution.
// Paths are relative to the OVAV repo root.
var syncDirectories = []struct {
	Path     string
	Category string
}{
	// Canonical agent sources — these are the source of truth for every runtime.
	{"ovav/agents/areas", "agent"},
	{"ovav/agents/leads", "agent"},
	{"ovav/agents/teams", "agent"},

	// Skills distributed per runtime (OpenCode).
	{".opencode/skills", "skill"},

	// OpenCode client source — themes, commands, plugins, agents
	// (separate from canonical ovav/agents/ which feeds the converter).
	{"clients/opencode", "config"},

	// Go runtime CLI surface (the canonical CLI tools).
	{"go-runtime/cmd", "cli"},

	// Top-level governance / policy / contract artifacts that drive runtime
	// behavior and must travel with every Product distribution.
	{"AGENTS.md", "config"},
	{".ovav/plan", "config"},
	{".ovav/laws", "config"},
	{".ovav/policy", "config"},
	{".ovav/service_areas", "config"},
	{".ovav/registry", "config"},

	// Visual identity — theme canónico y configuración de terminal.
	{".ovav/visual", "config"},

	// Generated runtimes — keep in sync when canonical sources change.
	{"runtimes", "runtime"},
}

// ── Public API ──────────────────────────────────────────────────────

// DetectChanges scans OVAV directories for files that have changed
// since the last sync manifest was recorded. It also detects files
// that were present in the previous manifest but no longer exist on
// disk (action = "remove"), so a sync to Product can prune stale files.
func DetectChanges(ovavRoot string) (*SyncManifest, error) {
	manifest := &SyncManifest{
		Version:   "1.0.0",
		Generated: time.Now().UTC(),
		Source:    ovavRoot,
	}

	// Load previous manifest for comparison (path = root/.ovav/sync/manifest.json).
	prev, _ := LoadManifest(filepath.Join(ovavRoot, ".ovav", "sync", "manifest.json"))

	currentPaths := make(map[string]bool, 256)

	for _, dir := range syncDirectories {
		fullPath := filepath.Join(ovavRoot, dir.Path)

		info, err := os.Stat(fullPath)
		if err != nil {
			continue // skip missing directories
		}

		if info.IsDir() {
			items, err := scanDir(fullPath, dir.Path, dir.Category, prev)
			if err == nil {
				for i := range items {
					currentPaths[items[i].Path] = true
				}
				manifest.Items = append(manifest.Items, items...)
			}
		} else {
			// Single file
			item := scanFile(fullPath, dir.Path, dir.Category, prev)
			if item != nil {
				currentPaths[item.Path] = true
				manifest.Items = append(manifest.Items, *item)
			}
		}
	}

	// Detect removals: any item from the previous manifest whose path is
	// no longer in the current scan is reported with action="remove" so
	// downstream Product sync can prune it. We only flag removals inside
	// directories we still scan (prevents false positives if a whole
	// directory disappeared — that's a separate concern handled by the
	// caller).
	if prev != nil {
		scannedPrefixes := make([]string, 0, len(syncDirectories))
		for _, d := range syncDirectories {
			scannedPrefixes = append(scannedPrefixes, d.Path)
		}
		for _, prevItem := range prev.Items {
			if currentPaths[prevItem.Path] {
				continue // still present
			}
			if !underAnyPrefix(prevItem.Path, scannedPrefixes) {
				continue // outside our scan surface
			}
			// File was previously tracked but is now gone.
			manifest.Items = append(manifest.Items, SyncItem{
				ID:       prevItem.ID,
				Path:     prevItem.Path,
				Category: prevItem.Category,
				Action:   "remove",
				Checksum: "",
				Size:     0,
				StagedAt: time.Time{},
				Synced:   false,
			})
		}
	}

	// Count statuses
	manifest.TotalItems = len(manifest.Items)
	for _, item := range manifest.Items {
		if item.Synced {
			manifest.Synced++
		}
	}
	manifest.Pending = manifest.TotalItems - manifest.Synced

	return manifest, nil
}

// underAnyPrefix returns true if path is inside any of the given
// directory prefixes. Both are forward-slash relative paths.
func underAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		// Normalize to forward slashes for comparison.
		np := filepath.ToSlash(p)
		if np == path || strings.HasPrefix(path, np+"/") || path == np {
			return true
		}
	}
	return false
}

// StageChanges marks selected items as staged (not yet synced).
// Items are identified by their ID.
func StageChanges(ovavRoot string, itemIDs []string) (*SyncResult, error) {
	manifest, err := DetectChanges(ovavRoot)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}

	result := &SyncResult{CompletedAt: time.Now().UTC()}

	idSet := make(map[string]bool, len(itemIDs))
	for _, id := range itemIDs {
		idSet[id] = true
	}

	for i, item := range manifest.Items {
		if idSet[item.ID] && !item.Synced {
			manifest.Items[i].StagedAt = time.Now().UTC()
			result.ItemsStaged++
		}
	}

	if result.ItemsStaged == 0 {
		return result, nil
	}

	// Persist staging
	manifestPath := filepath.Join(ovavRoot, ".ovav", "sync", "staged.json")
	if err := SaveManifest(manifest, manifestPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("save: %v", err))
		return result, nil
	}

	result.Manifest = manifestPath
	return result, nil
}

// ApplySync marks all staged items as synced and records the sync timestamp.
func ApplySync(ovavRoot string) (*SyncResult, error) {
	manifest, err := DetectChanges(ovavRoot)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}

	result := &SyncResult{CompletedAt: time.Now().UTC()}
	now := time.Now().UTC()

	staged := LoadStaged(ovavRoot)
	stagedIDs := make(map[string]bool)
	if staged != nil {
		for _, item := range staged.Items {
			if item.StagedAt.After(item.SyncedAt) {
				stagedIDs[item.ID] = true
			}
		}
	}

	for i, item := range manifest.Items {
		if stagedIDs[item.ID] && !item.Synced {
			manifest.Items[i].Synced = true
			manifest.Items[i].SyncedAt = now
			result.ItemsSynced++
		}
	}

	// Save updated manifest as the new baseline
	manifestPath := filepath.Join(ovavRoot, ".ovav", "sync", "manifest.json")
	if err := SaveManifest(manifest, manifestPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("save: %v", err))
	}

	// Clear staging
	os.Remove(filepath.Join(ovavRoot, ".ovav", "sync", "staged.json"))

	return result, nil
}

// QueueForProduct prepares a sync package for distribution to OVAV Product.
// This writes a queue file that cPanel reads when Product polls for updates.
// Uses the staged manifest (not fresh detection) to preserve StagedAt timestamps.
func QueueForProduct(ovavRoot string) (*SyncResult, error) {
	result := &SyncResult{CompletedAt: time.Now().UTC()}

	// Load the staged manifest — this has the StagedAt timestamps
	stagedPath := filepath.Join(ovavRoot, ".ovav", "sync", "staged.json")
	staged, err := LoadManifest(stagedPath)
	if err != nil || staged == nil {
		// Nothing staged — detect and stage everything pending
		manifest, err := DetectChanges(ovavRoot)
		if err != nil {
			return nil, fmt.Errorf("detect: %w", err)
		}
		var allIDs []string
		for _, item := range manifest.Items {
			if !item.Synced {
				allIDs = append(allIDs, item.ID)
			}
		}
		if len(allIDs) == 0 {
			return result, nil
		}
		// Stage all pending
		if _, err := StageChanges(ovavRoot, allIDs); err != nil {
			return nil, fmt.Errorf("stage: %w", err)
		}
		// Reload staged
		staged, err = LoadManifest(stagedPath)
		if err != nil || staged == nil {
			return result, nil
		}
	}

	// Filter to staged + unsynced items
	var queueItems []SyncItem
	for _, item := range staged.Items {
		if !item.Synced && !item.StagedAt.IsZero() {
			queueItems = append(queueItems, item)
		}
	}

	if len(queueItems) == 0 {
		return result, nil
	}

	queueManifest := &SyncManifest{
		Version:    staged.Version,
		Generated:  time.Now().UTC(),
		Source:     ovavRoot,
		Items:      queueItems,
		TotalItems: len(queueItems),
		Pending:    len(queueItems),
	}

	queueDir := filepath.Join(ovavRoot, ".ovav", "sync", "queue")
	if err := os.MkdirAll(queueDir, 0750); err != nil {
		return nil, fmt.Errorf("mkdir queue: %w", err)
	}

	// Clean old queue files to prevent accumulation
	entries, _ := os.ReadDir(queueDir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			os.Remove(filepath.Join(queueDir, entry.Name()))
		}
	}

	queuePath := filepath.Join(queueDir, fmt.Sprintf("queue_%s.json", time.Now().UTC().Format("20060102T150405Z")))
	if err := SaveManifest(queueManifest, queuePath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("save queue: %v", err))
		return result, nil
	}

	result.ItemsStaged = len(queueItems)
	result.Manifest = queuePath
	return result, nil
}

// GetQueueStatus returns the current sync queue status for cPanel to serve.
func GetQueueStatus(ovavRoot string) (*SyncManifest, error) {
	queueDir := filepath.Join(ovavRoot, ".ovav", "sync", "queue")
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		return &SyncManifest{
			Version:   "1.0.0",
			Generated: time.Now().UTC(),
			Source:    ovavRoot,
		}, nil
	}

	var allItems []SyncItem
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			m, err := LoadManifest(filepath.Join(queueDir, entry.Name()))
			if err != nil {
				continue
			}
			allItems = append(allItems, m.Items...)
		}
	}

	return &SyncManifest{
		Version:    "1.0.0",
		Generated:  time.Now().UTC(),
		Source:     ovavRoot,
		Items:      allItems,
		TotalItems: len(allItems),
		Pending:    len(allItems),
	}, nil
}

// ── Internal ────────────────────────────────────────────────────────

// skipDirName returns true if a directory name should be excluded from the
// sync scan (build artifacts, vendor, backups, hidden noise).
func skipDirName(name string) bool {
	switch name {
	case "vendor", "node_modules", ".git", "dist", "build",
		"__pycache__", ".cache", "tmp", "backups", "backups_old",
		// Self-exclusion: the sync engine must not ingest its own queue
		// manifests and trigger sync loops.
		"queue", "sync":
		return true
	}
	// Skip directories starting with underscore that are test fixtures or
	// scratch areas (e.g., `_test/`, `_fixtures/`).
	if strings.HasPrefix(name, "_") && name != "_ovav" {
		return true
	}
	return false
}

func scanDir(fullPath, relPath, category string, prev *SyncManifest) ([]SyncItem, error) {
	var items []SyncItem

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip on error
		}
		if info.IsDir() {
			if skipDirName(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only consider files with sync-worthy extensions.
		ext := strings.ToLower(filepath.Ext(path))
		if !validSyncExtensions[ext] {
			return nil
		}

		// Skip backup / swap / editor temp files.
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".bak") || strings.HasSuffix(base, ".swp") ||
			strings.HasPrefix(base, "~") || strings.HasPrefix(base, ".#") {
			return nil
		}

		// Skip Go test files (they are not distributed to end users).
		if ext == ".go" && strings.HasSuffix(base, "_test.go") {
			return nil
		}

		rel, _ := filepath.Rel(fullPath, path)
		itemRel := filepath.Join(relPath, rel)

		item := scanFile(path, itemRel, category, prev)
		if item != nil {
			items = append(items, *item)
		}

		return nil
	})

	return items, err
}

func scanFile(fullPath, relPath, category string, prev *SyncManifest) *SyncItem {
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil
	}

	checksum, err := fileChecksum(fullPath)
	if err != nil {
		return nil
	}

	item := SyncItem{
		ID:       makeID(relPath),
		Path:     relPath,
		Category: category,
		Checksum: checksum,
		Size:     info.Size(),
		StagedAt: time.Time{},
	}

	// Determine action: add, update, or remove
	if prev != nil {
		for _, p := range prev.Items {
			if p.Path == relPath {
				if p.Checksum != checksum {
					item.Action = "update"
				} else {
					item.Synced = true
					item.SyncedAt = p.SyncedAt
					item.Action = p.Action // preserve
				}
				return &item
			}
		}
	}
	item.Action = "add"
	return &item
}

func makeID(path string) string {
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", h[:8])
}

func fileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:16]), nil
}

// ── Persistence ─────────────────────────────────────────────────────

func LoadManifest(path string) (*SyncManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m SyncManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func SaveManifest(m *SyncManifest, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, data, 0640)
}

func LoadStaged(ovavRoot string) *SyncManifest {
	path := filepath.Join(ovavRoot, ".ovav", "sync", "staged.json")
	m, err := LoadManifest(path)
	if err != nil {
		return nil
	}
	return m
}
