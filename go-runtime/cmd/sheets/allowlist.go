// Package main — multi-spreadsheet allowlist loader.
//
// Reads .ovav/vault/sheets_allowlist.yaml (gopkg.in/yaml.v3) so the
// bridge can address any spreadsheet the CEO pre-approves, not just
// the demo file. AssertAllowed rejects everything else.
//
// Schema is intentionally tiny — readability over abstraction.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// AllowEntry is one row of the allowlist YAML.
type AllowEntry struct {
	ID      string   `yaml:"id" json:"id"`
	Name    string   `yaml:"name" json:"name"`
	Tags    []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Default bool     `yaml:"default,omitempty" json:"default,omitempty"`
}

// AllowlistFile is the on-disk YAML shape.
type AllowlistFile struct {
	Spreadsheets []AllowEntry `yaml:"spreadsheets" json:"spreadsheets"`
}

type allowlistCache struct {
	mu      sync.RWMutex
	entries map[string]AllowEntry // by ID
	defaults []string              // IDs marked default
	path    string
}

var globalAllowlist = &allowlistCache{entries: map[string]AllowEntry{}}

// loadAllowlist reads + parses the YAML. Missing file is OK
// (returns empty allowlist + writes a seed template on first run).
func loadAllowlist(path string) error {
	globalAllowlist.mu.Lock()
	defer globalAllowlist.mu.Unlock()
	globalAllowlist.path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return seedAllowlist(path)
		}
		return fmt.Errorf("allowlist: read %s: %w", path, err)
	}

	var file AllowlistFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("allowlist: parse %s: %w", path, err)
	}

	entries := make(map[string]AllowEntry, len(file.Spreadsheets))
	defaults := []string{}
	for _, e := range file.Spreadsheets {
		if e.ID == "" {
			continue
		}
		entries[e.ID] = e
		if e.Default {
			defaults = append(defaults, e.ID)
		}
	}
	globalAllowlist.entries = entries
	globalAllowlist.defaults = defaults
	return nil
}

func seedAllowlist(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	seed := AllowlistFile{
		Spreadsheets: []AllowEntry{
			{
				ID:      "1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y",
				Name:    "OVAV demo workbook",
				Tags:    []string{"demo", "ovav"},
				Default: true,
			},
		},
	}
	data, err := yaml.Marshal(&seed)
	if err != nil {
		return err
	}
	header := "# OVAV Sheets Bridge — allowlist of approved spreadsheets.\n" +
		"# Add IDs you want to enable. The bridge REFUSES anything not here.\n\n"
	return os.WriteFile(path, []byte(header+string(data)), 0o600)
}

// assertAllowed is the gate used by every Sheets operation.
func assertAllowed(id string) error {
	globalAllowlist.mu.RLock()
	defer globalAllowlist.mu.RUnlock()
	if _, ok := globalAllowlist.entries[id]; ok {
		return nil
	}
	return fmt.Errorf("sheets: spreadsheet %q not in allowlist — refusing (run `ovav-sheets allowlist add --id ... --name ...`)", id)
}

// resolveSpreadsheet picks the spreadsheet ID from CLI flag,
// env var, or default fall-back. Empty result is allowed — the
// caller can pick the first default.
func resolveSpreadsheet(args []string) (id, name string) {
	if v := flagValue(args, "--spreadsheet", ""); v != "" {
		id = v
	}
	if id == "" {
		if v := os.Getenv("OVAV_SHEETS_SPREADSHEET"); v != "" {
			id = v
		}
	}
	globalAllowlist.mu.RLock()
	defer globalAllowlist.mu.RUnlock()
	if id == "" {
		for _, d := range globalAllowlist.defaults {
			id = d
			break
		}
	}
	if id == "" {
		// first entry as last resort
		for _, e := range globalAllowlist.entries {
			id = e.ID
			break
		}
	}
	if e, ok := globalAllowlist.entries[id]; ok {
		name = e.Name
	}
	return
}

// allowlistPath is the conventional location inside repoRoot.
func allowlistPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".ovav", "vault", "sheets_allowlist.yaml")
}

// cmdAllowlistList implements `sheets allowlist list`.
func cmdAllowlistList(args []string) error {
	globalAllowlist.mu.RLock()
	defer globalAllowlist.mu.RUnlock()
	fmt.Printf("OVAV Sheets allowlist (%d entries) — %s\n", len(globalAllowlist.entries), globalAllowlist.path)
	for _, e := range globalAllowlist.entries {
		tagStr := ""
		if len(e.Tags) > 0 {
			tagStr = "  tags=" + fmt.Sprint(e.Tags)
		}
		def := ""
		if e.Default {
			def = "  [DEFAULT]"
		}
		fmt.Printf("  • %s  — %s%s%s\n", e.ID, e.Name, tagStr, def)
	}
	return nil
}

// cmdAllowlistAdd implements `sheets allowlist add --id ... --name ... [--tag ...]`.
func cmdAllowlistAdd(args []string) error {
	id := flagValue(args, "--id", "")
	name := flagValue(args, "--name", "")
	if id == "" || name == "" {
		return fmt.Errorf("allowlist add: --id and --name required")
	}
	var tags []string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--tag" {
			tags = append(tags, args[i+1])
		}
	}
	makeDefault := false
	for _, a := range args {
		if a == "--default" {
			makeDefault = true
		}
	}

	data, err := os.ReadFile(globalAllowlist.path)
	if err != nil {
		return fmt.Errorf("allowlist add: %w", err)
	}
	var file AllowlistFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("allowlist add: parse: %w", err)
	}
	// upsert
	found := false
	for i, e := range file.Spreadsheets {
		if e.ID == id {
			file.Spreadsheets[i].Name = name
			if len(tags) > 0 {
				file.Spreadsheets[i].Tags = tags
			}
			if makeDefault {
				file.Spreadsheets[i].Default = true
			}
			found = true
			break
		}
	}
	if !found {
		file.Spreadsheets = append(file.Spreadsheets, AllowEntry{
			ID: id, Name: name, Tags: tags, Default: makeDefault,
		})
	}
	if makeDefault {
		for i := range file.Spreadsheets {
			if file.Spreadsheets[i].ID != id {
				file.Spreadsheets[i].Default = false
			}
		}
	}
	out, err := yaml.Marshal(&file)
	if err != nil {
		return err
	}
	header := "# OVAV Sheets Bridge — allowlist of approved spreadsheets.\n" +
		"# Add IDs you want to enable. The bridge REFUSES anything not here.\n\n"
	if err := os.WriteFile(globalAllowlist.path, []byte(header+string(out)), 0o600); err != nil {
		return err
	}
	// refresh cache
	return loadAllowlist(globalAllowlist.path)
}

// cmdAllowlistRemove implements `sheets allowlist remove --id ...`.
func cmdAllowlistRemove(args []string) error {
	id := flagValue(args, "--id", "")
	if id == "" {
		return fmt.Errorf("allowlist remove: --id required")
	}
	data, err := os.ReadFile(globalAllowlist.path)
	if err != nil {
		return fmt.Errorf("allowlist remove: %w", err)
	}
	var file AllowlistFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("allowlist remove: parse: %w", err)
	}
	kept := file.Spreadsheets[:0]
	for _, e := range file.Spreadsheets {
		if e.ID != id {
			kept = append(kept, e)
		}
	}
	file.Spreadsheets = kept
	out, _ := yaml.Marshal(&file)
	header := "# OVAV Sheets Bridge — allowlist of approved spreadsheets.\n" +
		"# Add IDs you want to enable. The bridge REFUSES anything not here.\n\n"
	if err := os.WriteFile(globalAllowlist.path, []byte(header+string(out)), 0o600); err != nil {
		return err
	}
	return loadAllowlist(globalAllowlist.path)
}
