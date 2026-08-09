// Package vault — Asset encryption for OVAV runtime.
//
// Encrypts canonical SOURCE assets (irreplaceable), not projections (regenerable).
//   - profiles: .ovav/registry/service_profiles.yaml                    (source)
//   - agents:   .ovav/source/agents/leads/*.md + teams/*/*.md          (source)
//   - skills:   .ovav/source/skills/*/SKILL.md                          (source)
//
// OVAV works natively with well-organized configs, then converts and projects
// for CLI. The source is irreplaceable; the projection is regenerable.
//
// Bundle format: JSON map { "kind": "...", "version": 1, "files": { "path": "content" } }
// Encrypted with AES-256-GCM (encrypt.go).
package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── Asset kinds ──────────────────────────────────────────────────────────────

// AssetKind classifies OVAV asset types for encryption.
type AssetKind string

const (
	AssetProfiles AssetKind = "profiles"
	AssetAgents   AssetKind = "agents"
	AssetSkills   AssetKind = "skills"
)

// OutputDir is the directory where encrypted assets are written.
const OutputDir = ".ovav/vault"

// ── Bundle types ─────────────────────────────────────────────────────────────

// AssetBundle holds a set of source files to encrypt as one unit.
type AssetBundle struct {
	Kind    AssetKind         `json:"kind"`
	Version int               `json:"version"`
	Files   map[string]string `json:"files"` // relative_path → content
}

// assetDef defines what to scan for a given asset kind.
type assetDef struct {
	Kind        AssetKind
	Glob        string   // single glob relative to repo root (legacy; prefer Globs)
	Globs       []string // multiple glob patterns relative to repo root
	SingleFile  string   // exact file path, if not a glob
	StripPrefix string   // remove this prefix from keys in bundle
}

var assetDefs = []assetDef{
	{
		Kind:        AssetProfiles,
		SingleFile:  ".ovav/registry/service_profiles.yaml",
		StripPrefix: ".ovav/registry/",
	},
	{
		Kind: AssetAgents,
		Globs: []string{
			".ovav/source/agents/leads/*.md",
			".ovav/source/agents/teams/*/*.md",
		},
		StripPrefix: ".ovav/source/agents/",
	},
	{
		Kind:        AssetSkills,
		Glob:        ".ovav/source/skills/*/SKILL.md",
		StripPrefix: ".ovav/source/skills/",
	},
}

// ── Public API ───────────────────────────────────────────────────────────────

// ScanAssets discovers all encryptable assets in the OVAV repo.
// repoRoot must be the root of an OVAV repository.
func ScanAssets(repoRoot string) ([]AssetBundle, error) {
	var bundles []AssetBundle

	for _, def := range assetDefs {
		bundle := AssetBundle{
			Kind:    def.Kind,
			Version: 1,
			Files:   make(map[string]string),
		}

		var paths []string

		if def.SingleFile != "" {
			fullPath := filepath.Join(repoRoot, def.SingleFile)
			if _, statErr := os.Stat(fullPath); statErr == nil {
				paths = append(paths, def.SingleFile)
			} else {
				// Single file missing — skip but don't error (non-fatal)
				continue
			}
		} else {
			patterns := def.Globs
			if len(patterns) == 0 {
				patterns = []string{def.Glob}
			}
			for _, pattern := range patterns {
				fullPattern := filepath.Join(repoRoot, pattern)
				matches, globErr := filepath.Glob(fullPattern)
				if globErr != nil {
					return nil, fmt.Errorf("vault: glob %s: %w", pattern, globErr)
				}
				for _, p := range matches {
					rel, relErr := filepath.Rel(repoRoot, p)
					if relErr != nil {
						return nil, fmt.Errorf("vault: relative path for %s: %w", p, relErr)
					}
					paths = append(paths, rel)
				}
			}
		}

		for _, relPath := range paths {
			fullPath := filepath.Join(repoRoot, relPath)
			content, readErr := os.ReadFile(fullPath)
			if readErr != nil {
				return nil, fmt.Errorf("vault: read %s: %w", relPath, readErr)
			}

			// Strip prefix to get clean key
			key := relPath
			if def.StripPrefix != "" && strings.HasPrefix(key, def.StripPrefix) {
				key = key[len(def.StripPrefix):]
			}
			bundle.Files[key] = string(content)
		}

		if len(bundle.Files) == 0 {
			continue // nothing to bundle
		}

		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// EncryptBundle serializes and encrypts an AssetBundle.
// Returns AES-256-GCM ciphertext: nonce(12) || tag(16) || encrypted_json.
func EncryptBundle(bundle AssetBundle, key []byte) ([]byte, error) {
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("vault: marshal bundle: %w", err)
	}
	return Encrypt(data, key)
}

// DecryptBundle decrypts and deserializes an AssetBundle.
func DecryptBundle(ciphertext []byte, key []byte) (*AssetBundle, error) {
	plaintext, err := Decrypt(ciphertext, key)
	if err != nil {
		return nil, err
	}

	var bundle AssetBundle
	if err := json.Unmarshal(plaintext, &bundle); err != nil {
		return nil, fmt.Errorf("vault: unmarshal bundle: %w", err)
	}

	return &bundle, nil
}

// EncryptAllAssets discovers all assets and writes .enc files to OutputDir.
// Returns a map of output paths → byte count for reporting.
func EncryptAllAssets(repoRoot string, key []byte) (map[string]int, error) {
	bundles, err := ScanAssets(repoRoot)
	if err != nil {
		return nil, err
	}

	if len(bundles) == 0 {
		return nil, errors.New("vault: no assets found to encrypt")
	}

	vaultDir := filepath.Join(repoRoot, OutputDir)
	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		return nil, fmt.Errorf("vault: mkdir %s: %w", vaultDir, err)
	}

	written := make(map[string]int)
	for _, bundle := range bundles {
		ciphertext, err := EncryptBundle(bundle, key)
		if err != nil {
			return nil, fmt.Errorf("vault: encrypt %s: %w", bundle.Kind, err)
		}

		outPath := filepath.Join(vaultDir, string(bundle.Kind)+".enc")
		if err := os.WriteFile(outPath, ciphertext, 0644); err != nil {
			return nil, fmt.Errorf("vault: write %s: %w", outPath, err)
		}

		written[outPath] = len(ciphertext)
	}

	return written, nil
}

// DecryptAllAssets reads all .enc files from OutputDir and restores source files.
func DecryptAllAssets(repoRoot string, key []byte) error {
	vaultDir := filepath.Join(repoRoot, OutputDir)

	patterns := []string{"profiles.enc", "agents.enc", "skills.enc"}
	found := false

	for _, name := range patterns {
		encPath := filepath.Join(vaultDir, name)
		ciphertext, err := os.ReadFile(encPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("vault: read %s: %w", encPath, err)
		}

		bundle, err := DecryptBundle(ciphertext, key)
		if err != nil {
			return fmt.Errorf("vault: decrypt %s: %w", name, err)
		}

		// Restore files to their original source locations
		for relKey, content := range bundle.Files {
			var fullPath string
			switch bundle.Kind {
			case AssetProfiles:
				fullPath = filepath.Join(repoRoot, ".ovav", "registry", relKey)
			case AssetAgents:
				fullPath = filepath.Join(repoRoot, ".ovav", "source", "agents", relKey)
			case AssetSkills:
				fullPath = filepath.Join(repoRoot, ".ovav", "source", "skills", relKey)
			default:
				return fmt.Errorf("vault: unknown bundle kind: %s", bundle.Kind)
			}

			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("vault: mkdir %s: %w", dir, err)
			}

			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("vault: write %s: %w", fullPath, err)
			}
		}

		found = true
	}

	if !found {
		return errors.New("vault: no .enc files found in " + vaultDir)
	}

	return nil
}

// EncryptFileAsset encrypts a single file and writes the .enc output.
func EncryptFileAsset(inputPath, outputPath string, key []byte) error {
	return EncryptFile(inputPath, outputPath, key)
}

// DecryptFileAsset decrypts a single .enc file back to plaintext.
func DecryptFileAsset(inputPath, outputPath string, key []byte) error {
	return DecryptFile(inputPath, outputPath, key)
}
