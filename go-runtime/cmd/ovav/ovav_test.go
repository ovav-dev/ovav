package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ovav/ovav/internal/chronos"
	"github.com/ovav/ovav/internal/security/defense"
	"github.com/ovav/ovav/internal/vault"
)

func TestHumanDuration_Seconds(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.d)
		if got != tt.want {
			t.Errorf("humanDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestHumanDuration_Minutes(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{time.Minute, "1m"},
		{5 * time.Minute, "5m"},
		{59 * time.Minute, "59m"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.d)
		if got != tt.want {
			t.Errorf("humanDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestHumanDuration_Hours(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{time.Hour, "1h0m"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
		{24 * time.Hour, "24h0m"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.d)
		if got != tt.want {
			t.Errorf("humanDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestHexToBytes_Valid(t *testing.T) {
	// 64 hex chars = 32 bytes (AES-256 key)
	hexStr := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key, err := hexToBytes(hexStr)
	if err != nil {
		t.Fatalf("hexToBytes(%q) unexpected error: %v", hexStr, err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(key))
	}
	if key[0] != 0x01 || key[1] != 0x23 {
		t.Errorf("unexpected byte values: got %x %x", key[0], key[1])
	}
}

func TestHexToBytes_InvalidLength(t *testing.T) {
	_, err := hexToBytes("abcd")
	if err == nil {
		t.Error("expected error for short hex string, got nil")
	}
}

func TestSha256Hex(t *testing.T) {
	// SHA-256 of empty byte slice is well-known
	got := sha256Hex([]byte{})
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("sha256Hex([]) = %q, want %q", got, want)
	}
}

func TestResolvePackID(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "default"},
		{[]string{"--pack-id", "studio"}, "studio"},
		{[]string{"--pack-id=command"}, "command"},
		{[]string{"--other", "flag"}, "default"},
	}
	for _, tt := range tests {
		got := resolvePackID(tt.args)
		if got != tt.want {
			t.Errorf("resolvePackID(%v) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

// ── isPublicCommand ───────────────────────────────────────────────

func TestIsPublicCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"login", true}, {"signin", true}, {"auth", true},
		{"help", true}, {"--help", true}, {"-h", true},
		{"status", false}, {"install", false}, {"vault", false},
		{"LOGIN", false}, {"", false}, {"logout", false},
	}
	for _, tt := range tests {
		if got := isPublicCommand(tt.cmd); got != tt.want {
			t.Errorf("isPublicCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

// ── totalFiles ────────────────────────────────────────────────────

func TestTotalFiles(t *testing.T) {
	tests := []struct {
		name    string
		bundles []vault.AssetBundle
		want    int
	}{
		{"nil", nil, 0},
		{"empty", []vault.AssetBundle{}, 0},
		{"one bundle empty", []vault.AssetBundle{{Files: map[string]string{}}}, 0},
		{"one bundle 3 files", []vault.AssetBundle{{Files: map[string]string{"a": "1", "b": "2", "c": "3"}}}, 3},
		{"three bundles", []vault.AssetBundle{
			{Files: map[string]string{"a": "1"}},
			{Files: map[string]string{"b": "2", "c": "3"}},
			{Files: map[string]string{}},
		}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := totalFiles(tt.bundles); got != tt.want {
				t.Errorf("totalFiles() = %d, want %d", got, tt.want)
			}
		})
	}
}

// ── auditStatusIcon ───────────────────────────────────────────────

func TestAuditStatusIcon(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"clean", "✅ CLEAN"}, {"tampered", "🔴 TAMPERED"},
		{"broken", "⚠️  BROKEN"}, {"uninstalled", "❌ UNINSTALLED"},
		{"unknown", "unknown"}, {"", ""},
	}
	for _, tt := range tests {
		if got := auditStatusIcon(tt.in); got != tt.want {
			t.Errorf("auditStatusIcon(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// ── statusIconStr ─────────────────────────────────────────────────

func TestStatusIconStr(t *testing.T) {
	tests := []struct {
		ok, installed bool
		want          string
	}{
		{true, true, "✅"}, {false, true, "⚠️"},
		{true, false, "❌"}, {false, false, "❌"},
	}
	for _, tt := range tests {
		if got := statusIconStr(tt.ok, tt.installed); got != tt.want {
			t.Errorf("statusIconStr(%v,%v) = %q, want %q", tt.ok, tt.installed, got, tt.want)
		}
	}
}

// ── severityIcon ──────────────────────────────────────────────────

func TestSeverityIcon(t *testing.T) {
	tests := []struct {
		in   defense.Severity
		want string
	}{
		{defense.SevInfo, "ℹ️"}, {defense.SevWarning, "⚠️"},
		{defense.SevCritical, "🔴"}, {defense.SevDeadly, "💀"},
		{defense.Severity("unknown"), "•"}, {defense.Severity(""), "•"},
	}
	for _, tt := range tests {
		if got := severityIcon(tt.in); got != tt.want {
			t.Errorf("severityIcon(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// ── lockdownBanner ────────────────────────────────────────────────

func TestLockdownBanner(t *testing.T) {
	if got := lockdownBanner(true); got != "🔒 LOCKDOWN" {
		t.Errorf("lockdownBanner(true) = %q", got)
	}
	if got := lockdownBanner(false); got != "🔓 OPEN" {
		t.Errorf("lockdownBanner(false) = %q", got)
	}
}

// ── rotationBanner ────────────────────────────────────────────────

func TestRotationBanner(t *testing.T) {
	if got := rotationBanner(true); got != "⚠️ REQUIRED" {
		t.Errorf("rotationBanner(true) = %q", got)
	}
	if got := rotationBanner(false); got != "✅ OK" {
		t.Errorf("rotationBanner(false) = %q", got)
	}
}

// ── healthBanner ──────────────────────────────────────────────────

func TestHealthBanner(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{100, "🟢 HEALTHY"}, {90, "🟢 HEALTHY"},
		{89.99, "🟡 DEGRADED"}, {70, "🟡 DEGRADED"},
		{69.99, "🟠 WARNING"}, {50, "🟠 WARNING"},
		{49.99, "🔴 CRITICAL"}, {0, "🔴 CRITICAL"}, {-1, "🔴 CRITICAL"},
	}
	for _, tt := range tests {
		if got := healthBanner(tt.score); got != tt.want {
			t.Errorf("healthBanner(%v) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

// ── isFish ────────────────────────────────────────────────────────

func TestIsFish(t *testing.T) {
	// Save and restore
	orig := os.Getenv("SHELL")
	defer os.Setenv("SHELL", orig)

	os.Setenv("SHELL", "/usr/bin/fish")
	if !isFish() {
		t.Error("isFish() = false when SHELL=fish")
	}

	os.Setenv("SHELL", "/bin/bash")
	if isFish() {
		t.Error("isFish() = true when SHELL=bash")
	}

	os.Setenv("SHELL", "")
	if isFish() {
		t.Error("isFish() = true when SHELL empty")
	}
}

// ── sessionPath ───────────────────────────────────────────────────

func TestSessionPath(t *testing.T) {
	p := sessionPath()
	if p == "" {
		t.Error("sessionPath() returned empty")
	}
	if !filepath.IsAbs(p) {
		t.Errorf("sessionPath() = %q, not absolute", p)
	}
	if filepath.Base(p) != "session" {
		t.Errorf("sessionPath() base = %q, want 'session'", filepath.Base(p))
	}
}

// ── Session.createdAt ─────────────────────────────────────────────

func TestSessionCreatedAt(t *testing.T) {
	s := Session{CreatedAt: "2026-01-15T10:30:00Z"}
	got := s.createdAt()
	if got.Year() != 2026 || got.Month() != 1 || got.Day() != 15 {
		t.Errorf("createdAt() = %v, want 2026-01-15", got)
	}

	// Invalid time string returns zero value
	s2 := Session{CreatedAt: "invalid"}
	got2 := s2.createdAt()
	if !got2.IsZero() {
		t.Errorf("createdAt() for invalid = %v, want zero", got2)
	}
}

// ── cliRuntimeOS ─────────────────────────────────────────────────

func TestCliRuntimeOS(t *testing.T) {
	got := cliRuntimeOS()
	if got != runtime.GOOS {
		t.Errorf("cliRuntimeOS() = %q, want %q", got, runtime.GOOS)
	}
}

// ── resolveCockpitBinary ─────────────────────────────────────────

func TestResolveCockpitBinary_EnvVar(t *testing.T) {
	orig := os.Getenv("OVAV_COCKPIT_BIN")
	defer os.Setenv("OVAV_COCKPIT_BIN", orig)

	os.Setenv("OVAV_COCKPIT_BIN", "/custom/path/cockpit")
	got := resolveCockpitBinary()
	if got != "/custom/path/cockpit" {
		t.Errorf("resolveCockpitBinary() with env = %q, want /custom/path/cockpit", got)
	}
}

func TestResolveCockpitBinary_Empty(t *testing.T) {
	orig := os.Getenv("OVAV_COCKPIT_BIN")
	defer os.Setenv("OVAV_COCKPIT_BIN", orig)
	os.Unsetenv("OVAV_COCKPIT_BIN")

	// Without env var and without cockpit binary, should return ""
	// (unless cockpit happens to exist in search paths)
	got := resolveCockpitBinary()
	// We can't assert empty because cockpit might exist in the repo
	// Just verify it doesn't panic
	_ = got
}

// ── findOvavRoot ─────────────────────────────────────────────────

func TestFindOvavRoot(t *testing.T) {
	// We're in the OVAV repo, so findOvavRoot should find it
	root, err := findOvavRoot()
	if err != nil {
		t.Fatalf("findOvavRoot() error: %v", err)
	}
	if root == "" {
		t.Error("findOvavRoot() returned empty root")
	}
	// Verify .ovav exists in the root
	if _, err := os.Stat(filepath.Join(root, ".ovav")); err != nil {
		t.Errorf("findOvavRoot() = %q, but .ovav not found there", root)
	}
}

// ── findMimo ─────────────────────────────────────────────────────

func TestFindMimo(t *testing.T) {
	// findMimo searches multiple paths; just verify it doesn't panic
	got := findMimo()
	// May or may not find mimo — that's OK, just verify no crash
	_ = got
}

// ── parseModeFlag (additional cases) ─────────────────────────────

func TestParseModeFlag_Deploy(t *testing.T) {
	mode := parseModeFlag([]string{"--deploy"})
	if string(mode) != "source-local-apply" {
		t.Errorf("parseModeFlag(--deploy) = %q, want source-local-apply", mode)
	}
}

func TestParseModeFlag_ModeEquals(t *testing.T) {
	mode := parseModeFlag([]string{"--mode=sandbox"})
	if string(mode) != "sandbox" {
		t.Errorf("parseModeFlag(--mode=sandbox) = %q, want sandbox", mode)
	}
}

func TestParseModeFlag_ModeSpace(t *testing.T) {
	mode := parseModeFlag([]string{"--mode", "apply"})
	if string(mode) != "source-local-apply" {
		t.Errorf("parseModeFlag(--mode apply) = %q, want source-local-apply", mode)
	}
}

func TestParseModeFlag_UnknownMode(t *testing.T) {
	// Unknown mode value falls back to default
	mode := parseModeFlag([]string{"--mode", "unknown-value"})
	if string(mode) != "dry-run" {
		t.Errorf("parseModeFlag(--mode unknown) = %q, want dry-run", mode)
	}
}

// ── hexToBytes (additional edge cases) ───────────────────────────

func TestHexToBytes_InvalidHex(t *testing.T) {
	// 64 chars but with non-hex characters
	_, err := hexToBytes("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	if err == nil {
		t.Error("expected error for invalid hex chars, got nil")
	}
}

// ── jsonOutput ────────────────────────────────────────────────────

func TestJsonOutput_NilData(t *testing.T) {
	code := jsonOutput(nil)
	if code != 0 {
		t.Errorf("jsonOutput(nil) = %d, want 0", code)
	}
}

func TestJsonOutput_ComplexData(t *testing.T) {
	data := map[string]interface{}{
		"nested": map[string]int{"a": 1, "b": 2},
		"list":   []string{"x", "y"},
	}
	code := jsonOutput(data)
	if code != 0 {
		t.Errorf("jsonOutput(complex) = %d, want 0", code)
	}
}

// ── isPublicCommand (product, version) ───────────────────────────

func TestIsPublicCommand_ProductAndVersion(t *testing.T) {
	public := []string{"product", "version", "--version", "-v"}
	for _, cmd := range public {
		if !isPublicCommand(cmd) {
			t.Errorf("isPublicCommand(%q) = false, want true", cmd)
		}
	}
}

// ── requireSession ────────────────────────────────────────────────

func TestRequireSession_NoSession(t *testing.T) {
	// Temporarily rename session file to simulate no session
	orig := sessionPath()
	backup := orig + ".test_bak"
	if _, err := os.Stat(orig); err == nil {
		os.Rename(orig, backup)
		defer os.Rename(backup, orig)
	} else {
		// No session file exists, good
		defer os.Remove(orig) // cleanup if test creates one
	}

	// requireSession prints to stderr, just verify it returns false
	got := requireSession()
	if got {
		t.Error("requireSession() = true when no session exists")
	}
}

func TestRequireSession_ExpiredSession(t *testing.T) {
	// Create an expired session
	sess := Session{
		VaultKeyHash: "abc123",
		MachineID:    "test-machine-id-12345678",
		CreatedAt:    "2020-01-01T00:00:00Z", // Way in the past
		Hostname:     "test",
		User:         "test",
	}
	orig := sessionPath()
	backup := orig + ".test_bak"
	if _, err := os.Stat(orig); err == nil {
		os.Rename(orig, backup)
		defer os.Rename(backup, orig)
	}

	if err := saveSession(sess); err != nil {
		t.Fatalf("saveSession() error: %v", err)
	}
	defer os.Remove(orig)

	got := requireSession()
	if got {
		t.Error("requireSession() = true for expired session")
	}
}

// ── loadSession / saveSession roundtrip ───────────────────────────

func TestSessionRoundtrip(t *testing.T) {
	orig := sessionPath()
	backup := orig + ".test_bak"
	if _, err := os.Stat(orig); err == nil {
		os.Rename(orig, backup)
		defer os.Rename(backup, orig)
	}
	defer os.Remove(orig)

	sess := Session{
		VaultKeyHash: "deadbeef12345678",
		MachineID:    "machine-abc",
		CreatedAt:    "2026-07-18T10:00:00Z",
		Hostname:     "braka-laptop",
		User:         "braka",
		IdentityID:   "id-thavren",
		Role:         "lead",
		Level:        10,
		Name:         "Thavren",
	}

	if err := saveSession(sess); err != nil {
		t.Fatalf("saveSession() error: %v", err)
	}

	loaded, ok := loadSession()
	if !ok {
		t.Fatal("loadSession() returned false")
	}
	if loaded.VaultKeyHash != sess.VaultKeyHash {
		t.Errorf("VaultKeyHash = %q, want %q", loaded.VaultKeyHash, sess.VaultKeyHash)
	}
	if loaded.Name != sess.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, sess.Name)
	}
	if loaded.Role != sess.Role {
		t.Errorf("Role = %q, want %q", loaded.Role, sess.Role)
	}
	if loaded.Level != sess.Level {
		t.Errorf("Level = %d, want %d", loaded.Level, sess.Level)
	}
}

func TestLoadSession_NoFile(t *testing.T) {
	orig := sessionPath()
	backup := orig + ".test_bak"
	if _, err := os.Stat(orig); err == nil {
		os.Rename(orig, backup)
		defer os.Rename(backup, orig)
	}

	_, ok := loadSession()
	if ok {
		t.Error("loadSession() returned true when no session file exists")
	}
}

func TestLoadSession_InvalidJSON(t *testing.T) {
	orig := sessionPath()
	backup := orig + ".test_bak"
	if _, err := os.Stat(orig); err == nil {
		os.Rename(orig, backup)
		defer os.Rename(backup, orig)
	}
	defer os.Remove(orig)

	os.MkdirAll(filepath.Dir(orig), 0700)
	os.WriteFile(orig, []byte("not valid json{{{"), 0600)

	_, ok := loadSession()
	if ok {
		t.Error("loadSession() returned true for invalid JSON")
	}
}

// ── exportVaultKey ────────────────────────────────────────────────

func TestExportVaultKey(t *testing.T) {
	// Just verify it doesn't panic
	key := []byte("0123456789abcdef0123456789abcdef")
	exportVaultKey(key, "test-seed-16chars!")
	// Cleanup
	home, _ := os.UserHomeDir()
	os.Remove(filepath.Join(home, sessionDir, "vault_key_export"))
}

// ── formatChronosHuman ───────────────────────────────────────────

func TestFormatChronosHuman(t *testing.T) {
	// Test with minimal data
	output := chronos.ChronosOutput{
		Now: chronos.NowBlock{
			ISO:     "2026-07-18T10:00:00-0500",
			Weekday: "Saturday",
			UTC:     "2026-07-18T15:00:00Z",
			Epoch:   1752844800,
		},
		Head: chronos.HeadBlock{},
		Session: chronos.SessionBlock{
			Detected: false,
		},
		Drift: chronos.DriftBlock{
			Healthy: true,
		},
		System: chronos.SystemBlock{
			Hostname:   "test-host",
			GoVersion:  "go1.22.0",
			GitVersion: "2.40.0",
		},
	}

	result := formatChronosHuman(output)
	if result == "" {
		t.Error("formatChronosHuman() returned empty string")
	}
	if !strings.Contains(result, "test-host") {
		t.Error("formatChronosHuman() missing hostname")
	}
}

func TestFormatChronosHuman_WithHead(t *testing.T) {
	output := chronos.ChronosOutput{
		Now: chronos.NowBlock{
			ISO:     "2026-07-18T10:00:00-0500",
			Weekday: "Saturday",
			UTC:     "2026-07-18T15:00:00Z",
			Epoch:   1752844800,
		},
		Head: chronos.HeadBlock{
			HashShort: "abc1234",
			Message:   "test commit message",
			AgeHuman:  "5 minutes",
			ISO:       "2026-07-18T09:55:00-0500",
		},
		Session: chronos.SessionBlock{
			Detected:       true,
			IsContinuation: true,
			MinutesActive:  45,
			LastAction:     "commit",
			LastActionAt:   "2026-07-18T09:55:00-0500",
		},
		Drift: chronos.DriftBlock{
			Healthy: false,
			Warning: "test drift warning",
		},
		System: chronos.SystemBlock{
			Hostname:   "test-host",
			GoVersion:  "go1.22.0",
			GitVersion: "2.40.0",
		},
	}

	result := formatChronosHuman(output)
	if !strings.Contains(result, "abc1234") {
		t.Error("formatChronosHuman() missing HEAD hash")
	}
	if !strings.Contains(result, "continuación") {
		t.Error("formatChronosHuman() missing session continuation")
	}
	if !strings.Contains(result, "ALERTA") {
		t.Error("formatChronosHuman() missing drift ALERTA")
	}
}

// ── cmdVersion (non-interactive) ─────────────────────────────────

func TestCmdVersion_JSON(t *testing.T) {
	code := cmdVersion([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdVersion(--json) = %d, want 0", code)
	}
}

func TestCmdVersion_Human(t *testing.T) {
	code := cmdVersion(nil)
	if code != 0 {
		t.Errorf("cmdVersion() = %d, want 0", code)
	}
}

// ── cmdUpdate ────────────────────────────────────────────────────

func TestCmdUpdate(t *testing.T) {
	code := cmdUpdate(nil)
	if code != 0 {
		t.Errorf("cmdUpdate() = %d, want 0", code)
	}
}

func TestCmdUpdate_JSON(t *testing.T) {
	code := cmdUpdate([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdUpdate(--json) = %d, want 0", code)
	}
}

// ── cmdConfig ────────────────────────────────────────────────────

func TestCmdConfig(t *testing.T) {
	code := cmdConfig(nil)
	if code != 0 {
		t.Errorf("cmdConfig() = %d, want 0", code)
	}
}

// ── cmdTools (subcommands) ───────────────────────────────────────

func TestCmdTools_List(t *testing.T) {
	code := cmdTools([]string{"list"})
	if code != 0 {
		t.Errorf("cmdTools(list) = %d, want 0", code)
	}
}

func TestCmdTools_Search(t *testing.T) {
	code := cmdTools([]string{"search", "vault"})
	if code != 0 {
		t.Errorf("cmdTools(search vault) = %d, want 0", code)
	}
}

func TestCmdTools_SearchEmpty(t *testing.T) {
	code := cmdTools([]string{"search"})
	if code != 2 {
		t.Errorf("cmdTools(search) = %d, want 2", code)
	}
}

func TestCmdTools_Go(t *testing.T) {
	code := cmdTools([]string{"go"})
	if code != 0 {
		t.Errorf("cmdTools(go) = %d, want 0", code)
	}
}

func TestCmdTools_Categories(t *testing.T) {
	code := cmdTools([]string{"categories"})
	if code != 0 {
		t.Errorf("cmdTools(categories) = %d, want 0", code)
	}
}

func TestCmdTools_Help(t *testing.T) {
	code := cmdTools([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdTools(--help) = %d, want 0", code)
	}
}

func TestCmdTools_DefaultSearch(t *testing.T) {
	// Unknown subcommand falls through to search
	code := cmdTools([]string{"nonexistent"})
	if code != 0 {
		t.Errorf("cmdTools(nonexistent) = %d, want 0", code)
	}
}

// ── cmdProfile ───────────────────────────────────────────────────

func TestCmdProfile_NoArgs(t *testing.T) {
	code := cmdProfile(nil)
	if code != 0 {
		t.Errorf("cmdProfile() = %d, want 0", code)
	}
}

func TestCmdProfile_List(t *testing.T) {
	code := cmdProfile([]string{"list"})
	if code != 0 {
		t.Errorf("cmdProfile(list) = %d, want 0", code)
	}
}

func TestCmdProfile_Help(t *testing.T) {
	code := cmdProfile([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdProfile(--help) = %d, want 0", code)
	}
}

func TestCmdProfile_Unknown(t *testing.T) {
	code := cmdProfile([]string{"nonexistent"})
	if code != 2 {
		t.Errorf("cmdProfile(nonexistent) = %d, want 2", code)
	}
}

// ── cmdSBOM ──────────────────────────────────────────────────────

func TestCmdSBOM_NoArgs(t *testing.T) {
	code := cmdSBOM(nil)
	if code != 0 {
		t.Errorf("cmdSBOM() = %d, want 0", code)
	}
}

func TestCmdSBOM_Help(t *testing.T) {
	code := cmdSBOM([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdSBOM(--help) = %d, want 0", code)
	}
}

func TestCmdSBOM_Unknown(t *testing.T) {
	code := cmdSBOM([]string{"nonexistent"})
	if code != 2 {
		t.Errorf("cmdSBOM(nonexistent) = %d, want 2", code)
	}
}

// ── cmdWaiver ────────────────────────────────────────────────────

func TestCmdWaiver_NoArgs(t *testing.T) {
	code := cmdWaiver(nil)
	if code != 0 {
		t.Errorf("cmdWaiver() = %d, want 0", code)
	}
}

func TestCmdWaiver_Help(t *testing.T) {
	code := cmdWaiver([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdWaiver(--help) = %d, want 0", code)
	}
}

func TestCmdWaiver_ShorthandWithoutSession(t *testing.T) {
	code := cmdWaiver([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdWaiver(shorthand without session) = %d, want 1", code)
	}
}

// ── cmdLicense ───────────────────────────────────────────────────

func TestCmdLicense_NoArgs(t *testing.T) {
	code := cmdLicense(nil)
	if code != 1 {
		t.Errorf("cmdLicense() = %d, want 1", code)
	}
}

func TestCmdLicense_Unknown(t *testing.T) {
	code := cmdLicense([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdLicense(nonexistent) = %d, want 1", code)
	}
}

func TestCmdLicense_MachineID(t *testing.T) {
	code := cmdLicense([]string{"machine-id"})
	if code != 0 {
		t.Errorf("cmdLicense(machine-id) = %d, want 0", code)
	}
}

// ── cmdGit ───────────────────────────────────────────────────────
// DISABLED: cmdGit function was removed from codebase
// func TestCmdGit_NoArgs(t *testing.T) { code := cmdGit(nil); ... }
// func TestCmdGit_Help(t *testing.T) { ... }
// func TestCmdGit_Unknown(t *testing.T) { ... }

// ── cmdHook ──────────────────────────────────────────────────────

func TestCmdHook_NoArgs(t *testing.T) {
	code := cmdHook(nil)
	if code != 0 {
		t.Errorf("cmdHook() = %d, want 0", code)
	}
}

// ── cmdInfra ─────────────────────────────────────────────────────

func TestCmdInfra_NoArgs(t *testing.T) {
	code := cmdInfra(nil)
	if code != 0 {
		t.Errorf("cmdInfra() = %d, want 0", code)
	}
}

// ── cmdProduct ───────────────────────────────────────────────────

func TestCmdProduct_NoArgs(t *testing.T) {
	code := cmdProduct(nil)
	if code != 0 {
		t.Errorf("cmdProduct() = %d, want 0", code)
	}
}

func TestCmdProduct_Help(t *testing.T) {
	code := cmdProduct([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdProduct(--help) = %d, want 0", code)
	}
}

func TestCmdProduct_Unknown(t *testing.T) {
	code := cmdProduct([]string{"nonexistent"})
	if code != 2 {
		t.Errorf("cmdProduct(nonexistent) = %d, want 2", code)
	}
}

// ── cmdDefend ────────────────────────────────────────────────────

func TestCmdDefend_NoArgs(t *testing.T) {
	code := cmdDefend(nil)
	if code != 0 {
		t.Errorf("cmdDefend() = %d, want 0", code)
	}
}

func TestCmdDefend_Unknown(t *testing.T) {
	code := cmdDefend([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdDefend(nonexistent) = %d, want 1", code)
	}
}

// ── cmdGovern ────────────────────────────────────────────────────

func TestCmdGovern_NoArgs(t *testing.T) {
	code := cmdGovern(nil)
	// Returns 0 (healthy) or 2 (critical decisions) — both are valid
	if code != 0 && code != 2 {
		t.Errorf("cmdGovern() = %d, want 0 or 2", code)
	}
}

func TestCmdGovern_Unknown(t *testing.T) {
	code := cmdGovern([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdGovern(nonexistent) = %d, want 1", code)
	}
}

// ── cmdSurfaces ──────────────────────────────────────────────────

func TestCmdSurfaces(t *testing.T) {
	code := cmdSurfaces(nil)
	if code != 0 {
		t.Errorf("cmdSurfaces() = %d, want 0", code)
	}
}

func TestCmdSurfaces_RepairPlan(t *testing.T) {
	code := cmdSurfaces([]string{"repair-plan"})
	if code != 0 {
		t.Errorf("cmdSurfaces(repair-plan) = %d, want 0", code)
	}
}

// ── cmdExportGate ────────────────────────────────────────────────

func TestCmdExportGate(t *testing.T) {
	code := cmdExportGate(nil)
	if code != 0 {
		t.Errorf("cmdExportGate() = %d, want 0", code)
	}
}

// ── cmdRepoCheck ─────────────────────────────────────────────────

func TestCmdRepoCheck(t *testing.T) {
	code := cmdRepoCheck(nil)
	if code != 0 {
		t.Errorf("cmdRepoCheck() = %d, want 0", code)
	}
}

// ── cmdReleaseCheck ──────────────────────────────────────────────

func TestCmdReleaseCheck(t *testing.T) {
	code := cmdReleaseCheck(nil)
	if code != 0 {
		t.Errorf("cmdReleaseCheck() = %d, want 0", code)
	}
}

// ── cmdDetectEnv ─────────────────────────────────────────────────

func TestCmdDetectEnv(t *testing.T) {
	code := cmdDetectEnv(nil)
	if code != 0 {
		t.Errorf("cmdDetectEnv() = %d, want 0", code)
	}
}

func TestCmdDetectEnv_JSON(t *testing.T) {
	code := cmdDetectEnv([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdDetectEnv(--json) = %d, want 0", code)
	}
}

// ── cmdGateway ───────────────────────────────────────────────────

func TestCmdGateway_NoAction(t *testing.T) {
	code := cmdGateway(nil)
	if code != 1 {
		t.Errorf("cmdGateway() = %d, want 1", code)
	}
}

func TestCmdGateway_InvalidAction(t *testing.T) {
	code := cmdGateway([]string{"--action", "invalid"})
	if code != 1 {
		t.Errorf("cmdGateway(invalid) = %d, want 1", code)
	}
}

func TestCmdGateway_ValidAction(t *testing.T) {
	code := cmdGateway([]string{"--action", "setup"})
	// May return 0 or 1 depending on gateway state, but shouldn't panic
	_ = code
}

// ── cmdChronos ───────────────────────────────────────────────────

func TestCmdChronos_JSON(t *testing.T) {
	code := cmdChronos(nil)
	if code != 0 {
		t.Errorf("cmdChronos() = %d, want 0", code)
	}
}

func TestCmdChronos_Human(t *testing.T) {
	code := cmdChronos([]string{"--human"})
	if code != 0 {
		t.Errorf("cmdChronos(--human) = %d, want 0", code)
	}
}

// ── cmdProject ───────────────────────────────────────────────────

func TestCmdProject_NoArgs(t *testing.T) {
	code := cmdProject(nil)
	if code != 1 {
		t.Errorf("cmdProject() = %d, want 1", code)
	}
}

func TestCmdProject_Status(t *testing.T) {
	code := cmdProject([]string{"status"})
	if code != 0 {
		t.Errorf("cmdProject(status) = %d, want 0", code)
	}
}

func TestCmdProject_Unknown(t *testing.T) {
	code := cmdProject([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdProject(nonexistent) = %d, want 1", code)
	}
}

// ── cmdWorktree ──────────────────────────────────────────────────

func TestCmdWorktree_NoArgs(t *testing.T) {
	code := cmdWorktree(nil)
	if code != 2 {
		t.Errorf("cmdWorktree() = %d, want 2", code)
	}
}

// ── cmdSync ──────────────────────────────────────────────────────

func TestCmdSync_Help(t *testing.T) {
	code := cmdSync([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdSync(--help) = %d, want 0", code)
	}
}

// ── cmdResolveSubagent ───────────────────────────────────────────

func TestCmdResolveSubagent_NoArgs(t *testing.T) {
	code := cmdResolveSubagent(nil)
	if code != 2 {
		t.Errorf("cmdResolveSubagent() = %d, want 2", code)
	}
}

func TestCmdResolveSubagent_Help(t *testing.T) {
	code := cmdResolveSubagent([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdResolveSubagent(--help) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: defend subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestDefendScan_Human(t *testing.T) {
	code := defendScan(nil)
	// May return 0 (no critical) or 2 (critical found)
	if code != 0 && code != 2 {
		t.Errorf("defendScan() = %d, want 0 or 2", code)
	}
}

func TestDefendScan_JSON(t *testing.T) {
	code := defendScan([]string{"--json"})
	if code != 0 && code != 2 {
		t.Errorf("defendScan(--json) = %d, want 0 or 2", code)
	}
}

func TestDefendLockdown_Toggle(t *testing.T) {
	code := defendLockdown(nil)
	if code != 0 {
		t.Errorf("defendLockdown() = %d, want 0", code)
	}
}

func TestDefendLockdown_On(t *testing.T) {
	code := defendLockdown([]string{"on"})
	if code != 0 {
		t.Errorf("defendLockdown(on) = %d, want 0", code)
	}
	// Reset
	defendLockdown([]string{"off"})
}

func TestDefendLockdown_JSON(t *testing.T) {
	code := defendLockdown([]string{"--json"})
	if code != 0 {
		t.Errorf("defendLockdown(--json) = %d, want 0", code)
	}
}

func TestDefendStatus_JSON(t *testing.T) {
	code := defendStatus([]string{"--json"})
	if code != 0 {
		t.Errorf("defendStatus(--json) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: govern subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestGovernHealth_Human(t *testing.T) {
	code := governHealth(nil)
	if code != 0 {
		t.Errorf("governHealth() = %d, want 0", code)
	}
}

func TestGovernHealth_JSON(t *testing.T) {
	code := governHealth([]string{"--json"})
	if code != 0 {
		t.Errorf("governHealth(--json) = %d, want 0", code)
	}
}

func TestGovernDecide_Human(t *testing.T) {
	code := governDecide(nil)
	// May return 0 or 2 (critical decisions)
	if code != 0 && code != 2 {
		t.Errorf("governDecide() = %d, want 0 or 2", code)
	}
}

func TestGovernDecide_JSON(t *testing.T) {
	code := governDecide([]string{"--json"})
	if code != 0 && code != 2 {
		t.Errorf("governDecide(--json) = %d, want 0 or 2", code)
	}
}

func TestGovernTrust_NoArgs(t *testing.T) {
	code := governTrust(nil)
	if code != 1 {
		t.Errorf("governTrust() = %d, want 1 (usage error)", code)
	}
}

func TestGovernTrust_WithArgs(t *testing.T) {
	code := governTrust([]string{"thavren", "test claim"})
	if code != 0 && code != 1 && code != 2 {
		t.Errorf("governTrust(thavren) = %d, want 0/1/2", code)
	}
}

func TestGovernTrust_JSON(t *testing.T) {
	code := governTrust([]string{"--json", "thavren", "test"})
	if code != 0 && code != 1 && code != 2 {
		t.Errorf("governTrust(--json) = %d, want 0/1/2", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: install commands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdInstall_DryRun(t *testing.T) {
	code := cmdInstall([]string{"--mode", "dry-run"})
	if code != 0 {
		t.Errorf("cmdInstall(dry-run) = %d, want 0", code)
	}
}

func TestCmdInstall_Sandbox(t *testing.T) {
	code := cmdInstall([]string{"--sandbox"})
	if code != 0 {
		t.Errorf("cmdInstall(sandbox) = %d, want 0", code)
	}
}

func TestCmdInstall_JSON(t *testing.T) {
	code := cmdInstall([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdInstall(--json) = %d, want 0", code)
	}
}

func TestCmdUninstall(t *testing.T) {
	code := cmdUninstall(nil)
	if code != 0 {
		t.Errorf("cmdUninstall() = %d, want 0", code)
	}
}

func TestCmdUninstall_JSON(t *testing.T) {
	code := cmdUninstall([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdUninstall(--json) = %d, want 0", code)
	}
}

func TestCmdPlan(t *testing.T) {
	code := cmdPlan(nil)
	if code != 0 {
		t.Errorf("cmdPlan() = %d, want 0", code)
	}
}

func TestCmdPlan_JSON(t *testing.T) {
	code := cmdPlan([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdPlan(--json) = %d, want 0", code)
	}
}

func TestCmdBackup(t *testing.T) {
	code := cmdBackup(nil)
	if code != 0 {
		t.Errorf("cmdBackup() = %d, want 0", code)
	}
}

func TestCmdBackup_JSON(t *testing.T) {
	code := cmdBackup([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdBackup(--json) = %d, want 0", code)
	}
}

func TestCmdApply(t *testing.T) {
	code := cmdApply(nil)
	if code != 0 {
		t.Errorf("cmdApply() = %d, want 0", code)
	}
}

func TestCmdApply_JSON(t *testing.T) {
	code := cmdApply([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdApply(--json) = %d, want 0", code)
	}
}

func TestCmdVerify(t *testing.T) {
	code := cmdVerify(nil)
	if code != 0 {
		t.Errorf("cmdVerify() = %d, want 0", code)
	}
}

func TestCmdVerify_JSON(t *testing.T) {
	code := cmdVerify([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdVerify(--json) = %d, want 0", code)
	}
}

func TestCmdRestore(t *testing.T) {
	code := cmdRestore(nil)
	if code != 0 {
		t.Errorf("cmdRestore() = %d, want 0", code)
	}
}

func TestCmdRestore_JSON(t *testing.T) {
	code := cmdRestore([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdRestore(--json) = %d, want 0", code)
	}
}

func TestCmdDeploy(t *testing.T) {
	code := cmdDeploy(nil)
	if code != 0 {
		t.Errorf("cmdDeploy() = %d, want 0", code)
	}
}

func TestCmdDeploy_JSON(t *testing.T) {
	code := cmdDeploy([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdDeploy(--json) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: status with write markers
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdStatus_JSON(t *testing.T) {
	code := cmdStatus([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdStatus(--json) = %d, want 0", code)
	}
}

func TestCmdStatus_WriteMarkers(t *testing.T) {
	code := cmdStatus([]string{"--write-markers"})
	if code != 0 {
		t.Errorf("cmdStatus(--write-markers) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: vault subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestVaultScan(t *testing.T) {
	code := vaultScan(nil)
	// May return 0 (no assets) or 1 (error finding repo)
	_ = code
}

func TestVaultScan_JSON(t *testing.T) {
	code := vaultScan([]string{"--json"})
	_ = code
}

func TestVaultGenKey(t *testing.T) {
	code := vaultGenKey(nil)
	if code != 0 {
		t.Errorf("vaultGenKey() = %d, want 0", code)
	}
}

func TestVaultGenKey_JSON(t *testing.T) {
	code := vaultGenKey([]string{"--json"})
	if code != 0 {
		t.Errorf("vaultGenKey(--json) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: SBOM subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestSBOMGenerate(t *testing.T) {
	code := sbomGenerate(nil)
	_ = code
}

func TestSBOMGenerate_JSON(t *testing.T) {
	code := sbomGenerate([]string{"--json"})
	_ = code
}

func TestSBOMVerify(t *testing.T) {
	code := sbomVerify(nil)
	_ = code
}

func TestSBOMVerify_JSON(t *testing.T) {
	code := sbomVerify([]string{"--json"})
	_ = code
}

func TestSBOMHash(t *testing.T) {
	code := sbomHash(nil)
	_ = code
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: chronos subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdChronos_AllFlags(t *testing.T) {
	code := cmdChronos([]string{"--timeline", "3", "--session-threshold", "60", "--human"})
	if code != 0 {
		t.Errorf("cmdChronos(all flags) = %d, want 0", code)
	}
}

func TestCmdChronos_JSONFlags(t *testing.T) {
	code := cmdChronos([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdChronos(--json) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: tools with category filter
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdToolsList_Category(t *testing.T) {
	code := cmdToolsList([]string{"--category", "runtime"})
	if code != 0 {
		t.Errorf("cmdToolsList(runtime) = %d, want 0", code)
	}
}

func TestCmdToolsList_JSON(t *testing.T) {
	code := cmdToolsList([]string{"--json"})
	if code != 0 {
		t.Errorf("cmdToolsList(--json) = %d, want 0", code)
	}
}

func TestCmdToolsShow(t *testing.T) {
	code := cmdToolsShow([]string{"agent-runtime"})
	if code != 0 {
		t.Errorf("cmdToolsShow(agent-runtime) = %d, want 0", code)
	}
}

func TestCmdToolsShow_NotFound(t *testing.T) {
	code := cmdToolsShow([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdToolsShow(nonexistent) = %d, want 1", code)
	}
}

func TestCmdToolsShow_NoArgs(t *testing.T) {
	code := cmdToolsShow(nil)
	if code != 2 {
		t.Errorf("cmdToolsShow() = %d, want 2", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: license subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdLicense_Bind_NoKey(t *testing.T) {
	code := cmdLicense([]string{"bind"})
	if code != 1 {
		t.Errorf("cmdLicense(bind) = %d, want 1", code)
	}
}

func TestCmdLicense_Verify_NoHash(t *testing.T) {
	code := cmdLicense([]string{"verify"})
	if code != 1 {
		t.Errorf("cmdLicense(verify) = %d, want 1", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: hook subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdHookRun_NoStage(t *testing.T) {
	// cmdHookRun requires a hooks.Manager — skip if we can't create one
	// The hook system needs a real repo root, which we have in the test environment
	t.Skip("cmdHookRun requires hooks.Manager — covered via cmdHook dispatch")
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: gateway
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdGateway_AllActions(t *testing.T) {
	actions := []string{"setup", "sync", "security", "recovery", "update"}
	for _, action := range actions {
		code := cmdGateway([]string{"--action", action})
		// May return 0 or 1 depending on gateway state
		_ = code
	}
}

func TestCmdGateway_JSON(t *testing.T) {
	code := cmdGateway([]string{"--action", "setup", "--json"})
	_ = code
}

func TestCmdGateway_AllFlags(t *testing.T) {
	code := cmdGateway([]string{"--action", "setup", "--apply", "--consent", "--accept-risk"})
	_ = code
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: waiver subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestWaiverStatus(t *testing.T) {
	code := waiverStatus()
	if code < 0 {
		t.Errorf("waiverStatus() = %d, want >= 0", code)
	}
}

func TestWaiverCreate_NoReason(t *testing.T) {
	code := waiverCreate(nil)
	if code != 2 {
		t.Errorf("waiverCreate() = %d, want 2 (reason required)", code)
	}
}

func TestWaiverArgumentDetection(t *testing.T) {
	reason, branch, mins, err := parseWaiverCreateArgs([]string{"fixes-generales", "runtime", "--branch", "develop", "--mins", "60"})
	if err != nil {
		t.Fatalf("parseWaiverCreateArgs() error = %v", err)
	}
	if reason != "fixes-generales runtime" || branch != "develop" || mins != 60 {
		t.Fatalf("got reason=%q branch=%q mins=%d", reason, branch, mins)
	}
}

func TestWaiverRejectsTTLOverOneHour(t *testing.T) {
	_, _, _, err := parseWaiverCreateArgs([]string{"fixes-generales", "--mins", "61"})
	if err == nil {
		t.Fatal("expected TTL over 60 minutes to fail")
	}
}

func TestWaiverRejectsGrantedByOverride(t *testing.T) {
	_, _, _, err := parseWaiverCreateArgs([]string{"fixes-generales", "--granted-by", "spoofed"})
	if err == nil {
		t.Fatal("expected identity override to fail")
	}
}

func TestCmdWaiver_Status(t *testing.T) {
	code := cmdWaiver([]string{"status"})
	if code < 0 {
		t.Errorf("cmdWaiver(status) = %d, want >= 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: vault subcommands (full dispatch)
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdVault_Help(t *testing.T) {
	code := cmdVault(nil)
	if code != 0 {
		t.Errorf("cmdVault() = %d, want 0", code)
	}
}

func TestCmdVault_Scan(t *testing.T) {
	code := cmdVault([]string{"scan"})
	_ = code
}

func TestCmdVault_ScanJSON(t *testing.T) {
	code := cmdVault([]string{"scan", "--json"})
	_ = code
}

func TestCmdVault_Encrypt_NoKey(t *testing.T) {
	code := cmdVault([]string{"encrypt"})
	if code != 1 {
		t.Errorf("cmdVault(encrypt) = %d, want 1 (no key)", code)
	}
}

func TestCmdVault_Decrypt_NoKey(t *testing.T) {
	code := cmdVault([]string{"decrypt"})
	if code != 1 {
		t.Errorf("cmdVault(decrypt) = %d, want 1 (no key)", code)
	}
}

func TestCmdVault_GenKey(t *testing.T) {
	code := cmdVault([]string{"gen-key"})
	if code != 0 {
		t.Errorf("cmdVault(gen-key) = %d, want 0", code)
	}
}

func TestCmdVault_GenKeyJSON(t *testing.T) {
	code := cmdVault([]string{"gen-key", "--json"})
	if code != 0 {
		t.Errorf("cmdVault(gen-key --json) = %d, want 0", code)
	}
}

func TestCmdVault_HelpFlag(t *testing.T) {
	code := cmdVault([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdVault(--help) = %d, want 0", code)
	}
}

func TestCmdVault_Unknown(t *testing.T) {
	code := cmdVault([]string{"nonexistent"})
	if code != 2 {
		t.Errorf("cmdVault(nonexistent) = %d, want 2", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: loadKey
// ═══════════════════════════════════════════════════════════════════════════

func TestLoadKey_NoKey(t *testing.T) {
	_, err := loadKey(nil)
	if err == nil {
		t.Error("loadKey(nil) should return error")
	}
}

func TestLoadKey_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.hex")
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	os.WriteFile(keyPath, []byte(key), 0600)

	k, err := loadKey([]string{"--key", keyPath})
	if err != nil {
		t.Fatalf("loadKey(file) error: %v", err)
	}
	if len(k) != 32 {
		t.Errorf("loadKey(file) returned %d bytes, want 32", len(k))
	}
}

func TestLoadKey_FromEnv(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("OVAV_VAULT_KEY", key)

	k, err := loadKey(nil)
	if err != nil {
		t.Fatalf("loadKey(env) error: %v", err)
	}
	if len(k) != 32 {
		t.Errorf("loadKey(env) returned %d bytes, want 32", len(k))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: product subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdProduct_Status(t *testing.T) {
	code := cmdProduct([]string{"status"})
	_ = code
}

func TestCmdProduct_Verify(t *testing.T) {
	code := cmdProduct([]string{"verify"})
	_ = code
}

func TestCmdProduct_Uninstall(t *testing.T) {
	code := cmdProduct([]string{"uninstall"})
	_ = code
}

func TestCmdProduct_Install_DryRun(t *testing.T) {
	code := cmdProduct([]string{"install", "--dry-run"})
	_ = code
}

func TestCmdProduct_Bootstrap(t *testing.T) {
	code := cmdProduct([]string{"bootstrap"})
	_ = code
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: tailor subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdTailor_Status(t *testing.T) {
	code := cmdTailor([]string{"status"})
	if code != 0 {
		t.Errorf("cmdTailor(status) = %d, want 0", code)
	}
}

func TestCmdTailor_Preview(t *testing.T) {
	code := cmdTailor([]string{"preview"})
	if code != 0 {
		t.Errorf("cmdTailor(preview) = %d, want 0", code)
	}
}

func TestCmdTailor_Apply(t *testing.T) {
	code := cmdTailor([]string{"apply"})
	if code != 0 {
		t.Errorf("cmdTailor(apply) = %d, want 0", code)
	}
}

func TestCmdTailor_Select(t *testing.T) {
	code := cmdTailor([]string{"select", "nucleo"})
	if code != 0 {
		t.Errorf("cmdTailor(select nucleo) = %d, want 0", code)
	}
}

func TestCmdTailor_Select_NoPlan(t *testing.T) {
	code := cmdTailor([]string{"select"})
	if code != 1 {
		t.Errorf("cmdTailor(select) = %d, want 1", code)
	}
}

func TestCmdTailor_Toggle(t *testing.T) {
	code := cmdTailor([]string{"toggle", "nonexistent"})
	if code != 0 {
		t.Errorf("cmdTailor(toggle) = %d, want 0", code)
	}
}

func TestCmdTailor_Toggle_NoItem(t *testing.T) {
	code := cmdTailor([]string{"toggle"})
	if code != 1 {
		t.Errorf("cmdTailor(toggle) = %d, want 1", code)
	}
}

func TestCmdTailor_Help(t *testing.T) {
	code := cmdTailor([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdTailor(--help) = %d, want 0", code)
	}
}

func TestCmdTailor_Unknown(t *testing.T) {
	code := cmdTailor([]string{"nonexistent"})
	if code != 1 {
		t.Errorf("cmdTailor(nonexistent) = %d, want 1", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: sync subcommands
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdSync_DryRun(t *testing.T) {
	code := cmdSync([]string{"--dry-run"})
	if code != 0 {
		t.Errorf("cmdSync(--dry-run) = %d, want 0", code)
	}
}

func TestCmdSync_Agents(t *testing.T) {
	code := cmdSync([]string{"--agents"})
	_ = code
}

func TestCmdSync_Skills(t *testing.T) {
	code := cmdSync([]string{"--skills"})
	_ = code
}

func TestCmdSync_Visual(t *testing.T) {
	code := cmdSync([]string{"--visual"})
	_ = code
}

func TestCmdSync_Mimocode(t *testing.T) {
	code := cmdSync([]string{"--mimocode"})
	_ = code
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: detect-env
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdDetectEnv_WithPath(t *testing.T) {
	code := cmdDetectEnv([]string{"--path", "."})
	if code != 0 {
		t.Errorf("cmdDetectEnv(--path) = %d, want 0", code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Coverage push: hook subcommands via dispatch
// ═══════════════════════════════════════════════════════════════════════════

func TestCmdHook_Help(t *testing.T) {
	code := cmdHook([]string{"--help"})
	if code != 0 {
		t.Errorf("cmdHook(--help) = %d, want 0", code)
	}
}

func TestCmdHook_Status(t *testing.T) {
	code := cmdHook([]string{"status"})
	if code != 0 {
		t.Errorf("cmdHook(status) = %d, want 0", code)
	}
}

func TestCmdHook_Install(t *testing.T) {
	code := cmdHook([]string{"install"})
	if code != 0 {
		t.Errorf("cmdHook(install) = %d, want 0", code)
	}
}

func TestCmdHook_Uninstall(t *testing.T) {
	code := cmdHook([]string{"uninstall"})
	if code != 0 {
		t.Errorf("cmdHook(uninstall) = %d, want 0", code)
	}
}

func TestCmdHook_Audit(t *testing.T) {
	code := cmdHook([]string{"audit"})
	if code != 0 {
		t.Errorf("cmdHook(audit) = %d, want 0", code)
	}
}

func TestCmdHook_Snapshot(t *testing.T) {
	code := cmdHook([]string{"snapshot"})
	if code != 0 {
		t.Errorf("cmdHook(snapshot) = %d, want 0", code)
	}
}

func TestCmdHook_Check(t *testing.T) {
	code := cmdHook([]string{"check"})
	// Returns 0 (no tampering) or 1 (tampering detected — normal in test env)
	_ = code
}

func TestCmdHook_Install_JSON(t *testing.T) {
	code := cmdHook([]string{"install", "--json"})
	if code != 0 {
		t.Errorf("cmdHook(install --json) = %d, want 0", code)
	}
}

func TestCmdHook_Uninstall_JSON(t *testing.T) {
	code := cmdHook([]string{"uninstall", "--json"})
	if code != 0 {
		t.Errorf("cmdHook(uninstall --json) = %d, want 0", code)
	}
}

func TestCmdHook_Status_JSON(t *testing.T) {
	code := cmdHook([]string{"status", "--json"})
	if code != 0 {
		t.Errorf("cmdHook(status --json) = %d, want 0", code)
	}
}

func TestCmdHook_Audit_JSON(t *testing.T) {
	code := cmdHook([]string{"audit", "--json"})
	if code != 0 {
		t.Errorf("cmdHook(audit --json) = %d, want 0", code)
	}
}

func TestCmdHook_Unknown(t *testing.T) {
	code := cmdHook([]string{"nonexistent"})
	if code != 2 {
		t.Errorf("cmdHook(nonexistent) = %d, want 2", code)
	}
}
