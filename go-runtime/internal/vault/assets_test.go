package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestScanAssets_Profiles(t *testing.T) {
	repo := createOVAVRepo(t)
	defer os.RemoveAll(repo)

	bundles, err := ScanAssets(repo)
	if err != nil {
		t.Fatalf("ScanAssets: %v", err)
	}

	// Should find profiles (agents and skills are also created)
	found := map[AssetKind]bool{}
	for _, b := range bundles {
		found[b.Kind] = true
	}

	if !found[AssetProfiles] {
		t.Error("expected profiles bundle not found")
	}
	if !found[AssetAgents] {
		t.Error("expected agents bundle not found")
	}
	if !found[AssetSkills] {
		t.Error("expected skills bundle not found")
	}
}

func TestScanAssets_EmptyRepo(t *testing.T) {
	repo := t.TempDir()
	// No .ovav or .opencode dirs
	bundles, err := ScanAssets(repo)
	if err != nil {
		t.Fatalf("ScanAssets: %v", err)
	}
	if len(bundles) != 0 {
		t.Errorf("expected 0 bundles, got %d", len(bundles))
	}
}

func TestEncryptDecryptBundle(t *testing.T) {
	bundle := AssetBundle{
		Kind:    AssetProfiles,
		Version: 1,
		Files: map[string]string{
			"service_profiles.yaml": "# OVAV profiles\ndummy: true\n",
		},
	}

	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	ciphertext, err := EncryptBundle(bundle, key)
	if err != nil {
		t.Fatalf("EncryptBundle: %v", err)
	}

	decrypted, err := DecryptBundle(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptBundle: %v", err)
	}

	if decrypted.Kind != bundle.Kind {
		t.Errorf("kind: got %q, want %q", decrypted.Kind, bundle.Kind)
	}
	if decrypted.Version != bundle.Version {
		t.Errorf("version: got %d, want %d", decrypted.Version, bundle.Version)
	}
	if len(decrypted.Files) != len(bundle.Files) {
		t.Errorf("files count: got %d, want %d", len(decrypted.Files), len(bundle.Files))
	}
	for k, v := range bundle.Files {
		if decrypted.Files[k] != v {
			t.Errorf("file %s: got %q, want %q", k, decrypted.Files[k], v)
		}
	}
}

func TestEncryptBundleWrongKey(t *testing.T) {
	bundle := AssetBundle{
		Kind:  AssetAgents,
		Files: map[string]string{"test.md": "test"},
	}

	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	ciphertext, _ := EncryptBundle(bundle, key1)
	_, err := DecryptBundle(ciphertext, key2)
	if err == nil {
		t.Error("expected decryption with wrong key to fail")
	}
}

func TestEncryptDecryptAllAssets(t *testing.T) {
	repo := createOVAVRepo(t)
	defer os.RemoveAll(repo)

	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	// Encrypt all
	written, err := EncryptAllAssets(repo, key)
	if err != nil {
		t.Fatalf("EncryptAllAssets: %v", err)
	}

	if len(written) != 3 {
		t.Errorf("expected 3 output files, got %d", len(written))
	}

	for path, size := range written {
		if size == 0 {
			t.Errorf("encrypted file %s has zero size", path)
		}
	}

	// Read original content before deleting
	profilesPath := filepath.Join(repo, ".ovav", "registry", "service_profiles.yaml")
	origProfiles, _ := os.ReadFile(profilesPath)

	// Read original content from source paths before deleting
	leadPath := filepath.Join(repo, ".ovav", "source", "agents", "leads", "thavren.md")
	origLead, _ := os.ReadFile(leadPath)
	teamPath := filepath.Join(repo, ".ovav", "source", "agents", "teams", "platform-engineering", "kael.md")
	origTeam, _ := os.ReadFile(teamPath)
	skillPath := filepath.Join(repo, ".ovav", "source", "skills", "ovav-runtime-gates", "SKILL.md")
	origSkill, _ := os.ReadFile(skillPath)

	// Remove source files, then decrypt
	os.RemoveAll(filepath.Join(repo, ".ovav", "registry"))
	os.RemoveAll(filepath.Join(repo, ".ovav", "source"))

	// Decrypt all
	if err := DecryptAllAssets(repo, key); err != nil {
		t.Fatalf("DecryptAllAssets: %v", err)
	}

	// Verify restored profiles
	restoredProfiles, err := os.ReadFile(profilesPath)
	if err != nil {
		t.Fatalf("read restored profiles: %v", err)
	}
	if !bytes.Equal(origProfiles, restoredProfiles) {
		t.Error("restored profiles differ from original")
	}

	// Verify restored agents
	restoredLead, err := os.ReadFile(leadPath)
	if err != nil {
		t.Fatalf("read restored lead agent: %v", err)
	}
	if !bytes.Equal(origLead, restoredLead) {
		t.Error("restored lead agent differs from original")
	}

	restoredTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read restored team agent: %v", err)
	}
	if !bytes.Equal(origTeam, restoredTeam) {
		t.Error("restored team agent differs from original")
	}

	// Verify restored skills
	restoredSkill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read restored skill: %v", err)
	}
	if !bytes.Equal(origSkill, restoredSkill) {
		t.Error("restored skill differs from original")
	}
}

func TestDecryptAllAssets_NoFiles(t *testing.T) {
	repo := t.TempDir()
	key, _ := GenerateKey()
	err := DecryptAllAssets(repo, key)
	if err == nil {
		t.Error("expected error when no .enc files exist")
	}
}

func TestEncryptAllAssets_NoAssets(t *testing.T) {
	repo := t.TempDir()
	key, _ := GenerateKey()
	_, err := EncryptAllAssets(repo, key)
	if err == nil {
		t.Error("expected error when no assets found")
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// createOVAVRepo creates a minimal OVAV repo structure for testing.
// Uses canonical source paths (.ovav/source/), not projections (.opencode/).
func createOVAVRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	// Profiles (source — .ovav/registry/)
	registryDir := filepath.Join(repo, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(
		filepath.Join(registryDir, "service_profiles.yaml"),
		[]byte("# OVAV Service Profiles\nprofiles:\n  thavren:\n    area: platform_engineering\n"),
		0644,
	)

	// Agents (source — .ovav/source/agents/)
	leadsDir := filepath.Join(repo, ".ovav", "source", "agents", "leads")
	os.MkdirAll(leadsDir, 0755)
	os.WriteFile(filepath.Join(leadsDir, "thavren.md"), []byte("# Thavren\nLead of Platform Engineering\n"), 0644)
	os.WriteFile(filepath.Join(leadsDir, "eidren.md"), []byte("# Eidren\nLead of Research Intelligence\n"), 0644)
	os.WriteFile(filepath.Join(leadsDir, "andres.md"), []byte("# Andrés\nSenior Implementer\n"), 0644)

	// Teams
	teamDir := filepath.Join(repo, ".ovav", "source", "agents", "teams", "platform-engineering")
	os.MkdirAll(teamDir, 0755)
	os.WriteFile(filepath.Join(teamDir, "kael.md"), []byte("# Kael\nGo Runtime\n"), 0644)
	os.WriteFile(filepath.Join(teamDir, "zara.md"), []byte("# Zara\nSecurity\n"), 0644)

	// Skills (source — .ovav/source/skills/)
	skillsDir := filepath.Join(repo, ".ovav", "source", "skills")
	os.MkdirAll(filepath.Join(skillsDir, "ovav-runtime-gates"), 0755)
	os.MkdirAll(filepath.Join(skillsDir, "ovav-context-pack"), 0755)
	os.WriteFile(filepath.Join(skillsDir, "ovav-runtime-gates", "SKILL.md"), []byte("# Runtime Gates\nSkill for OVAV\n"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "ovav-context-pack", "SKILL.md"), []byte("# Context Pack\nCompact context\n"), 0644)

	return repo
}
