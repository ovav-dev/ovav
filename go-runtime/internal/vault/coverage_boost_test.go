package vault

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Encrypt/Decrypt error paths ─────────────────────────────────────────────

func TestDecryptInvalidKeySize(t *testing.T) {
	_, err := Decrypt([]byte("ciphertext"), []byte("short"))
	if err == nil {
		t.Error("expected error for short key on Decrypt")
	}
}

func TestDecryptCiphertextTooShort(t *testing.T) {
	key, _ := GenerateKey()
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

// ── EncryptFile / DecryptFile ────────────────────────────────────────────────

func TestEncryptDecryptFile(t *testing.T) {
	dir := t.TempDir()
	key, _ := GenerateKey()

	inputPath := filepath.Join(dir, "plain.txt")
	outputPath := filepath.Join(dir, "plain.txt.enc")
	restoredPath := filepath.Join(dir, "restored.txt")

	origContent := []byte("OVAV file encryption test — secrets here")
	if err := os.WriteFile(inputPath, origContent, 0644); err != nil {
		t.Fatal(err)
	}

	// EncryptFile
	if err := EncryptFile(inputPath, outputPath, key); err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	encStat, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("encrypted file missing: %v", err)
	}
	if encStat.Size() == 0 {
		t.Error("encrypted file is empty")
	}

	// DecryptFile
	if err := DecryptFile(outputPath, restoredPath, key); err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	restored, err := os.ReadFile(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(origContent) {
		t.Errorf("roundtrip mismatch: got %q, want %q", restored, origContent)
	}
}

func TestEncryptFile_MissingInput(t *testing.T) {
	key, _ := GenerateKey()
	err := EncryptFile("/nonexistent/file.txt", "/tmp/out.enc", key)
	if err == nil {
		t.Error("expected error for missing input file")
	}
}

func TestDecryptFile_MissingInput(t *testing.T) {
	key, _ := GenerateKey()
	err := DecryptFile("/nonexistent/file.enc", "/tmp/out.txt", key)
	if err == nil {
		t.Error("expected error for missing input file")
	}
}

func TestDecryptFile_WrongKey(t *testing.T) {
	dir := t.TempDir()
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	inputPath := filepath.Join(dir, "secret.txt")
	encPath := filepath.Join(dir, "secret.txt.enc")
	outPath := filepath.Join(dir, "out.txt")

	os.WriteFile(inputPath, []byte("confidential"), 0644)
	EncryptFile(inputPath, encPath, key1)

	err := DecryptFile(encPath, outPath, key2)
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

// ── EncryptFileAsset / DecryptFileAsset ──────────────────────────────────────

func TestEncryptDecryptFileAsset(t *testing.T) {
	dir := t.TempDir()
	key, _ := GenerateKey()

	inputPath := filepath.Join(dir, "asset.md")
	encPath := filepath.Join(dir, "asset.md.enc")
	restoredPath := filepath.Join(dir, "restored.md")

	orig := []byte("# Agent Asset\nSensitive configuration data")
	os.WriteFile(inputPath, orig, 0644)

	if err := EncryptFileAsset(inputPath, encPath, key); err != nil {
		t.Fatalf("EncryptFileAsset: %v", err)
	}

	if err := DecryptFileAsset(encPath, restoredPath, key); err != nil {
		t.Fatalf("DecryptFileAsset: %v", err)
	}

	got, _ := os.ReadFile(restoredPath)
	if string(got) != string(orig) {
		t.Errorf("FileAsset roundtrip mismatch: got %q, want %q", got, orig)
	}
}

// ── DecryptAllAssets error paths ─────────────────────────────────────────────

func TestDecryptAllAssets_ReadError(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, ".ovav", "vault")
	os.MkdirAll(vaultDir, 0755)

	// Create a directory named profiles.enc so ReadFile fails
	os.MkdirAll(filepath.Join(vaultDir, "profiles.enc"), 0755)

	key, _ := GenerateKey()
	err := DecryptAllAssets(dir, key)
	if err == nil {
		t.Error("expected error when .enc path is a directory")
	}
}

func TestDecryptAllAssets_DecryptError(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, ".ovav", "vault")
	os.MkdirAll(vaultDir, 0755)

	// Write garbage data as profiles.enc
	os.WriteFile(filepath.Join(vaultDir, "profiles.enc"), []byte("not-valid-ciphertext"), 0644)

	key, _ := GenerateKey()
	err := DecryptAllAssets(dir, key)
	if err == nil {
		t.Error("expected error when .enc file contains invalid ciphertext")
	}
}

// ── EncryptBundle / DecryptBundle error paths ────────────────────────────────

func TestDecryptBundle_InvalidJSON(t *testing.T) {
	key, _ := GenerateKey()

	// Manually create valid ciphertext that decrypts to invalid JSON
	fakeCiphertext, err := Encrypt([]byte("not-json"), key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptBundle(fakeCiphertext, key)
	if err == nil {
		t.Error("expected error when decrypted data is not valid JSON")
	}
}

// ── ScanAssets error paths ───────────────────────────────────────────────────

func TestScanAssets_PartialAssets(t *testing.T) {
	// Only create profiles, no agents or skills
	dir := t.TempDir()
	registryDir := filepath.Join(dir, ".ovav", "registry")
	os.MkdirAll(registryDir, 0755)
	os.WriteFile(filepath.Join(registryDir, "service_profiles.yaml"), []byte("profiles: {}"), 0644)

	bundles, err := ScanAssets(dir)
	if err != nil {
		t.Fatalf("ScanAssets: %v", err)
	}
	if len(bundles) != 1 {
		t.Errorf("expected 1 bundle (profiles only), got %d", len(bundles))
	}
	if bundles[0].Kind != AssetProfiles {
		t.Errorf("expected profiles bundle, got %v", bundles[0].Kind)
	}
}

func TestScanAssets_StripPrefix(t *testing.T) {
	dir := t.TempDir()

	// Create agents with specific prefix
	leadsDir := filepath.Join(dir, ".ovav", "source", "agents", "leads")
	os.MkdirAll(leadsDir, 0755)
	os.WriteFile(filepath.Join(leadsDir, "test-lead.md"), []byte("# Test Lead"), 0644)

	bundles, err := ScanAssets(dir)
	if err != nil {
		t.Fatalf("ScanAssets: %v", err)
	}

	for _, b := range bundles {
		if b.Kind == AssetAgents {
			for key := range b.Files {
				if filepath.IsAbs(key) {
					t.Errorf("expected stripped key, got absolute path: %s", key)
				}
				if key == ".ovav/source/agents/leads/test-lead.md" {
					t.Error("prefix was not stripped from key")
				}
			}
		}
	}
}

// ── DecryptAllAssets unknown kind ────────────────────────────────────────────

func TestDecryptAllAssets_UnknownKind(t *testing.T) {
	dir := t.TempDir()
	key, _ := GenerateKey()

	// Create a bundle with unknown kind, encrypt it, place as profiles.enc
	bundle := AssetBundle{
		Kind:    "unknown_kind",
		Version: 1,
		Files:   map[string]string{"file.txt": "content"},
	}
	ct, err := EncryptBundle(bundle, key)
	if err != nil {
		t.Fatal(err)
	}

	vaultDir := filepath.Join(dir, ".ovav", "vault")
	os.MkdirAll(vaultDir, 0755)
	os.WriteFile(filepath.Join(vaultDir, "profiles.enc"), ct, 0644)

	err = DecryptAllAssets(dir, key)
	if err == nil {
		t.Error("expected error for unknown bundle kind")
	}
}
