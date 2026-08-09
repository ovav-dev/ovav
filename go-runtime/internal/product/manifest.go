package product

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProductManifestVersion is the schema version for the global install manifest.
const ProductManifestVersion = "1.0"

// Manifest tracks what was installed to ~/.local/share/ovav/.
// Used for verify, uninstall, and incremental updates.
type Manifest struct {
	Version   string            `json:"version"`   // manifest schema version
	Product   string            `json:"product"`   // OVAV Product release version (v1.1.0)
	Installed string            `json:"installed"` // ISO 8601
	OvavRoot  string            `json:"ovav_root"` // absolute path to OVAV Systems repo
	Platform  string            `json:"platform"`  // mimocode, opencode, etc.
	Entries   []ManifestEntry   `json:"entries"`
	Symlinks  []ManifestSymlink `json:"symlinks,omitempty"`
}

// ManifestEntry represents a single installed file.
type ManifestEntry struct {
	Source   string `json:"source"`   // absolute source path
	Target   string `json:"target"`   // absolute target path
	RelPath  string `json:"rel_path"` // relative to global dir
	Category string `json:"category"` // agents, skills, identity, config
	Hash     string `json:"hash"`     // SHA-256 of source file
	Size     int64  `json:"size"`
	IsCopy   bool   `json:"is_copy"` // true=copied, false=symlinked
}

// ManifestSymlink represents a created symlink.
type ManifestSymlink struct {
	Link   string `json:"link"`   // absolute symlink path
	Target string `json:"target"` // absolute target (what it points to)
	Valid  bool   `json:"valid"`  // was valid at install time
}

// ManifestPath returns the path to the manifest file in the global dir.
func ManifestPath() (string, error) {
	productDir, err := ProductDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(productDir, ".ovav-manifest.json"), nil
}

// LoadManifest reads the manifest from disk. Returns nil if not found.
func LoadManifest() (*Manifest, error) {
	path, err := ManifestPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// SaveManifest writes the manifest to disk.
func SaveManifest(m *Manifest) error {
	productDir, err := ProductDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(productDir, 0755); err != nil {
		return fmt.Errorf("mkdir global dir: %w", err)
	}
	path := filepath.Join(productDir, ".ovav-manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// NewManifest creates a fresh manifest for the current install.
func NewManifest(ovavRoot, platform string) *Manifest {
	return &Manifest{
		Version:   ProductManifestVersion,
		Product:   ProductVersion,
		Installed: time.Now().UTC().Format(time.RFC3339),
		OvavRoot:  ovavRoot,
		Platform:  platform,
		Entries:   make([]ManifestEntry, 0),
		Symlinks:  make([]ManifestSymlink, 0),
	}
}

// AddEntry adds a file entry to the manifest.
func (m *Manifest) AddEntry(source, target, relPath, category string, isCopy bool) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat source %s: %w", source, err)
	}

	hash, err := fileHash(source)
	if err != nil {
		hash = "" // non-fatal
	}

	m.Entries = append(m.Entries, ManifestEntry{
		Source:   source,
		Target:   target,
		RelPath:  relPath,
		Category: category,
		Hash:     hash,
		Size:     info.Size(),
		IsCopy:   isCopy,
	})
	return nil
}

// AddSymlink adds a symlink entry to the manifest.
func (m *Manifest) AddSymlink(link, target string, valid bool) {
	m.Symlinks = append(m.Symlinks, ManifestSymlink{
		Link:   link,
		Target: target,
		Valid:  valid,
	})
}

// fileHash returns the SHA-256 hex digest of a file.
func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h), nil
}
