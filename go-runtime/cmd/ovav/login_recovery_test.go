package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ovav/ovav/internal/identity"
)

func TestParseLoginOptions(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantRecover bool
		wantErr     bool
	}{
		{name: "recovery exact flag", args: []string{"--recover-ceo"}, wantRecover: true},
		{name: "unknown recovery alias", args: []string{"--recover"}, wantErr: true},
		{name: "unknown positional alias", args: []string{"recover-ceo"}, wantErr: true},
		{name: "unknown flag", args: []string{"--recover_ceo"}, wantErr: true},
		{name: "incompatible web", args: []string{"--recover-ceo", "--web"}, wantErr: true},
		{name: "incompatible force", args: []string{"--recover-ceo", "--force"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := parseLoginOptions(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLoginOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if options.RecoverCEO != tt.wantRecover {
				t.Fatalf("RecoverCEO = %v, want %v", options.RecoverCEO, tt.wantRecover)
			}
		})
	}
}

func TestRecoverySessionSnapshotRestoresExactBytesAndMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := sessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("{\"preexisting\":true}\n")
	if err := os.WriteFile(path, original, 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot, err := captureSessionSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := durableSessionReplace([]byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := restoreSessionSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("session bytes = %q, want %q", got, original)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("session mode = %o, want 640", info.Mode().Perm())
	}
}

func TestDurableSessionReplaceRejectsSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := sessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, "target")
	if err := os.WriteFile(target, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := durableSessionReplace([]byte("overwrite"), 0o600); err == nil {
		t.Fatal("durableSessionReplace accepted symlink")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "preserve" {
		t.Fatalf("symlink target changed: %q", got)
	}
}

func TestRecoverySessionRollbackPreservesConcurrentSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := sessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"identity_id":"other-user"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := restoreSessionSnapshotChecked(identity.RecoverySessionSnapshot{}); err == nil || !strings.Contains(err.Error(), "concurrently") {
		t.Fatalf("restoreSessionSnapshotChecked() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"identity_id":"other-user"}` {
		t.Fatalf("concurrent session overwritten: %q", got)
	}
}

func TestGitHubPinnedEnvironmentOverridesHost(t *testing.T) {
	environment := githubPinnedEnvironment([]string{"PATH=/bin", "GH_HOST=evil.example", "HOME=/tmp/home"})
	count := 0
	for _, variable := range environment {
		if strings.HasPrefix(variable, "GH_HOST=") {
			count++
			if variable != "GH_HOST=github.com" {
				t.Fatalf("unexpected GH_HOST: %q", variable)
			}
		}
	}
	if count != 1 {
		t.Fatalf("GH_HOST count = %d", count)
	}
}

func TestBuildWebSessionNilIdentityFailsClosed(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("buildWebSession panicked: %v", recovered)
		}
	}()
	_, err := buildWebSession("hash", "machine", "host", "user", "jwt", time.Now(), nil)
	if err == nil || !strings.Contains(err.Error(), "identity registry") {
		t.Fatalf("buildWebSession() error = %v", err)
	}
}

func TestBuildRecoverySessionUsesCanonicalRegistryIdentity(t *testing.T) {
	id := identity.Identity{
		ID: "ceo-alexander", Name: "Alexander Salvador", Email: "alexander@ovav.dev",
		Role: "ceo", Level: 10,
	}
	recovery := identity.RecoverySession{
		VaultKeyHash: "hash", MachineID: "machine", CreatedAt: "2026-08-13T12:30:00Z",
		IdentityID: id.ID, Name: id.Name, Email: id.Email, Role: id.Role, Level: id.Level,
	}
	session := sessionFromRecovery(recovery)
	if session.IdentityID != id.ID || session.Name != id.Name || session.Email != id.Email ||
		session.Role != id.Role || session.Level != id.Level {
		t.Fatalf("non-canonical session: %+v", session)
	}
}
