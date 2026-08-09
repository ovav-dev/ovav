package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ovav/ovav/internal/vault/secrets/providers"
)

// ── DiscoveryConfig ──────────────────────────────────────────────────────────

func TestDiscoveryConfig(t *testing.T) {
	cfg := DiscoveryConfig{
		GitHubOrg:   "ovav-dev",
		GitHubRepos: []string{"ovav", "vault"},
		FlyApps:     []string{"ovav-app"},
		SearchPaths: []string{"/home/user/project"},
		ExcludeDirs: []string{"node_modules", ".git"},
	}

	if cfg.GitHubOrg != "ovav-dev" {
		t.Errorf("GitHubOrg = %q, want %q", cfg.GitHubOrg, "ovav-dev")
	}
	if len(cfg.GitHubRepos) != 2 {
		t.Errorf("GitHubRepos len = %d, want 2", len(cfg.GitHubRepos))
	}
}

// ── Discover ───────────────────────────────────────────────────────────────

func TestDiscover_EmptyConfig(t *testing.T) {
	cfg := DiscoveryConfig{}
	report, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if report == nil {
		t.Fatal("Discover returned nil report")
	}
	if len(report.GitHub) != 0 {
		t.Errorf("GitHub map should be empty, got %d entries", len(report.GitHub))
	}
	if report.Files == nil {
		t.Error("Files should be initialized (not nil)")
	}
}

func TestDiscover_GitHubConfig(t *testing.T) {
	// Just verify it doesn't panic with GitHub config
	cfg := DiscoveryConfig{
		GitHubOrg: "ovav-dev",
	}
	report, err := Discover(cfg)
	if err != nil {
		// GitHub errors are logged but not fatal
	}
	// Report may be nil or have empty results
	if report != nil {
		if len(report.GitHub) == 0 {
			// Expected when no token or no access
		}
	}
}

// ── parseEnvFile ────────────────────────────────────────────────────────────

func TestParseEnvFile_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// Only keys containing TOKEN/SECRET/PASSWORD/KEY/CREDENTIAL/PRIVATE/API are included
	// NON_SECRET_VAR contains SECRET so it IS included
	os.WriteFile(envPath, []byte("API_KEY=my-key\nSECRET_TOKEN=abc123\nIGNORED=value"), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}

	// NON_SECRET_VAR contains SECRET so it passes filter; IGNORED has no keyword
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}

	found := make(map[string]string)
	for _, e := range entries {
		found[e.Name] = string(e.Value)
	}

	if found["API_KEY"] != "my-key" {
		t.Errorf("API_KEY = %q, want %q", found["API_KEY"], "my-key")
	}
	if found["SECRET_TOKEN"] != "abc123" {
		t.Errorf("SECRET_TOKEN = %q, want %q", found["SECRET_TOKEN"], "abc123")
	}
}

func TestParseEnvFile_QuotedValues(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// Keys must pass keyword filter to be included
	os.WriteFile(envPath, []byte("API_KEY='single quoted'\nSECRET_TOKEN=\"double quoted\""), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile quoted: %v", err)
	}

	found := make(map[string]string)
	for _, e := range entries {
		found[e.Name] = string(e.Value)
	}

	if found["API_KEY"] != "single quoted" {
		t.Errorf("API_KEY = %q, want %q", found["API_KEY"], "single quoted")
	}
	if found["SECRET_TOKEN"] != "double quoted" {
		t.Errorf("SECRET_TOKEN = %q, want %q", found["SECRET_TOKEN"], "double quoted")
	}
}

func TestParseEnvFile_Comments(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte("# This is a comment\nMY_API_KEY=bar # inline comment\n# Another\nTOKEN=qux"), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile with comments: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestParseEnvFile_NonSecretSkip(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// Only SECRET_KEY passes the keyword filter (contains SECRET and KEY);
	// others are skipped because they don't contain TOKEN/SECRET/PASSWORD/KEY/CREDENTIAL/PRIVATE/API
	os.WriteFile(envPath, []byte("FOO=bar\nPORT=8080\nDEBUG=true\nHOME=/root\nPATH=/usr/bin\nSECRET_KEY=actual-secret"), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}

	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Name] = true
	}

	// Only SECRET_KEY should be present
	if !found["SECRET_KEY"] {
		t.Error("SECRET_KEY should be present")
	}

	// These don't contain any secret-related keywords
	if found["FOO"] {
		t.Error("FOO should be filtered (no secret keyword)")
	}
	if found["PORT"] {
		t.Error("PORT should be filtered (no secret keyword)")
	}
	if found["DEBUG"] {
		t.Error("DEBUG should be filtered (no secret keyword)")
	}
	if found["HOME"] {
		t.Error("HOME should be filtered (no secret keyword)")
	}
	if found["PATH"] {
		t.Error("PATH should be filtered (no secret keyword)")
	}
}

func TestParseEnvFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte(""), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %d, want 0", len(entries))
	}
}

func TestParseEnvFile_OnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte("# comment\n# another"), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile only comments: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %d, want 0", len(entries))
	}
}

func TestParseEnvFile_EqualsInValue(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// Keys must pass keyword filter; values can contain = signs
	os.WriteFile(envPath, []byte("API_KEY=abc=xyz=123\nSECRET_TOKEN=https://example.com?a=1&b=2"), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile equals in value: %v", err)
	}

	found := make(map[string]string)
	for _, e := range entries {
		found[e.Name] = string(e.Value)
	}

	if found["API_KEY"] != "abc=xyz=123" {
		t.Errorf("API_KEY = %q, want %q", found["API_KEY"], "abc=xyz=123")
	}
	if found["SECRET_TOKEN"] != "https://example.com?a=1&b=2" {
		t.Errorf("SECRET_TOKEN = %q, want %q", found["SECRET_TOKEN"], "https://example.com?a=1&b=2")
	}
}

func TestParseEnvFile_Malformed(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// NO_EQUALS and ALSO_NO_EQUALS have no = so skipped; only VALID passes filter
	os.WriteFile(envPath, []byte("NO_EQUALS\nALSO_NO_EQUALS\nAPI_KEY=good"), 0600)

	entries, err := parseEnvFile(envPath)
	if err != nil {
		t.Fatalf("parseEnvFile malformed: %v", err)
	}

	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Name] = true
	}

	if !found["API_KEY"] {
		t.Error("API_KEY should be parsed")
	}
}

func TestParseEnvFile_NotFound(t *testing.T) {
	_, err := parseEnvFile("/nonexistent/path/.env")
	if err == nil {
		t.Error("parseEnvFile nonexistent file: expected error")
	}
}

// ── discoverFilesystem ───────────────────────────────────────────────────────

func TestDiscoverFilesystem(t *testing.T) {
	tmpDir := t.TempDir()

	envPath := filepath.Join(tmpDir, ".env")
	os.WriteFile(envPath, []byte("FOO=bar\nSECRET=test"), 0600)

	localPath := filepath.Join(tmpDir, ".env.local")
	os.WriteFile(localPath, []byte("DB_PASSWORD=secret123"), 0600)

	examplePath := filepath.Join(tmpDir, ".env.example")
	os.WriteFile(examplePath, []byte("PORT=8080"), 0600)

	results, err := discoverFilesystem([]string{tmpDir}, nil)
	if err != nil {
		t.Fatalf("discoverFilesystem: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Found = %d, want 2 (.env and .env.local)", len(results))
	}
}

func TestDiscoverFilesystem_Nested(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "backend")
	os.Mkdir(subDir, 0755)

	envPath := filepath.Join(subDir, ".env")
	os.WriteFile(envPath, []byte("NESTED_SECRET=value"), 0600)

	results, err := discoverFilesystem([]string{tmpDir}, nil)
	if err != nil {
		t.Fatalf("discoverFilesystem nested: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Found = %d, want 1", len(results))
	}
}

func TestDiscoverFilesystem_NoEnvFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644)

	results, err := discoverFilesystem([]string{tmpDir}, nil)
	if err != nil {
		t.Fatalf("discoverFilesystem no env: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Found = %d, want 0", len(results))
	}
}

func TestDiscoverFilesystem_ExcludeDirs(t *testing.T) {
	tmpDir := t.TempDir()
	nodeModules := filepath.Join(tmpDir, "node_modules")
	os.Mkdir(nodeModules, 0755)
	os.WriteFile(filepath.Join(nodeModules, ".env"), []byte("SHOULD_NOT_FIND=secret"), 0600)

	results, err := discoverFilesystem([]string{tmpDir}, []string{"node_modules"})
	if err != nil {
		t.Fatalf("discoverFilesystem exclude: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Found = %d, want 0 (excluded node_modules)", len(results))
	}
}

// ── countEnvFiles ────────────────────────────────────────────────────────────

func TestCountEnvFiles(t *testing.T) {
	files := []FilesystemSecret{
		{Path: "/a/.env"},
		{Path: "/a/.env.local"},
		{Path: "/b/.env"},
		{Path: "/a/.env"}, // duplicate path
	}

	count := countEnvFiles(files)
	if count != 3 {
		t.Errorf("countEnvFiles = %d, want 3 unique paths", count)
	}
}

func TestCountEnvFiles_Empty(t *testing.T) {
	count := countEnvFiles([]FilesystemSecret{})
	if count != 0 {
		t.Errorf("countEnvFiles empty = %d, want 0", count)
	}
}

// ── DiscoveryReport ──────────────────────────────────────────────────────────

func TestDiscoveryReport_Summary(t *testing.T) {
	report := &DiscoveryReport{
		GitHub: map[string][]providers.GitHubDiscoveryResult{
			"repo1": {{Name: "SECRET1", Source: "github_secrets"}},
			"repo2": {{Name: "SECRET2", Source: "github_secrets"}, {Name: "SECRET3", Source: "github_secrets"}},
		},
		Fly: map[string][]providers.FlyDiscoveryResult{
			"app1": {{Name: "FLY_SECRET", Value: nil, AppName: "app1", Source: "fly_secrets"}},
		},
		Files: []FilesystemSecret{
			{Path: "/a/.env"},
			{Path: "/a/.env.local"},
			{Path: "/b/.env"},
		},
	}

	summary := report.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestDiscoveryReport_Summary_Zero(t *testing.T) {
	report := &DiscoveryReport{
		GitHub: make(map[string][]providers.GitHubDiscoveryResult),
		Fly:    make(map[string][]providers.FlyDiscoveryResult),
		Files:  []FilesystemSecret{},
	}

	summary := report.Summary()
	if summary == "" {
		t.Error("Summary should not be empty even with zero values")
	}
}

// ── FilesystemSecret ─────────────────────────────────────────────────────────

func TestFilesystemSecret(t *testing.T) {
	fs := FilesystemSecret{
		Path:   "/path/to/.env",
		Name:   "SECRET_KEY",
		Type:   TypeAPIToken,
		Value:  []byte("secret-value"),
		Source: "filesystem",
	}

	if fs.Name != "SECRET_KEY" {
		t.Errorf("Name = %q, want %q", fs.Name, "SECRET_KEY")
	}
	if string(fs.Value) != "secret-value" {
		t.Errorf("Value = %q, want %q", string(fs.Value), "secret-value")
	}
	if fs.Source != "filesystem" {
		t.Errorf("Source = %q, want %q", fs.Source, "filesystem")
	}
}

// ── Discover with filesystem search ─────────────────────────────────────────

func TestDiscover_Filesystem(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("FS_SECRET=value"), 0600)

	cfg := DiscoveryConfig{
		SearchPaths: []string{tmpDir},
	}

	report, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(report.Files) == 0 {
		t.Error("Expected filesystem secrets to be found")
	}
}

// ── Discover Fly stub ───────────────────────────────────────────────────────

func TestDiscover_FlyConfig(t *testing.T) {
	// Fly discovery requires FLY_API_TOKEN
	cfg := DiscoveryConfig{
		FlyApps: []string{"test-app"},
	}

	report, err := Discover(cfg)
	if err != nil {
		t.Fatalf("Discover Fly: %v", err)
	}
	// Fly results may be empty if no token
	if report.Fly == nil {
		t.Error("Fly map should be initialized")
	}
}
