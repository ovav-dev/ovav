package identity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const recoveryRegistryYAML = `version: 1
canonical: true
updated_by: ceo
identities:
  - id: ceo-alexander
    name: Alexander Salvador
    email: alexander@ovav.dev
    role: ceo
    level: 10
    key_hash: "` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `"
    permissions:
      - full_system
      - manage_identities
    status: active
  - id: developer-one
    name: Developer One
    email: developer@ovav.dev
    role: developer
    level: 5
    key_hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
    permissions:
      - vault_read
    status: active
roles:
  ceo:
    level: 10
    description: CEO
  developer:
    level: 5
    description: Developer
signature:
  algorithm: HMAC-SHA256
  signed_by: ceo
  signed_at: "2026-01-01T00:00:00Z"
  value: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
`

type recoveryFixture struct {
	root           string
	original       []byte
	key            []byte
	saved          *RecoverySession
	sessionRemoved bool
	deps           RecoveryDependencies
	ghPath         string
	gitPath        string
}

func newRecoveryFixture(t *testing.T) *recoveryFixture {
	t.Helper()
	root := t.TempDir()
	registryDir := filepath.Join(root, ".ovav", "registry")
	if err := os.MkdirAll(registryDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(recoveryRegistryYAML)
	if err := os.WriteFile(filepath.Join(registryDir, "identities.yaml"), original, 0o600); err != nil {
		t.Fatal(err)
	}
	f := &recoveryFixture{
		root:     root,
		original: original,
		key:      bytes.Repeat([]byte{0x42}, 32),
	}
	f.ghPath, _ = exec.LookPath("gh")
	f.gitPath, _ = exec.LookPath("git")
	if f.ghPath == "" || f.gitPath == "" {
		t.Skip("git and gh executables required")
	}
	if absolute, err := filepath.Abs(f.ghPath); err == nil {
		f.ghPath = absolute
	}
	if absolute, err := filepath.Abs(f.gitPath); err == nil {
		f.gitPath = absolute
	}
	if _, err := os.Stat(f.ghPath); err != nil {
		t.Fatal(err)
	}
	f.deps = RecoveryDependencies{
		IsTTY:     func() bool { return true },
		MachineID: func() (string, error) { return "machine-immutable-id", nil },
		ReadSeed:  func() (string, error) { return "correct horse battery staple", nil },
		Confirm:   func(RecoverySummary) (string, error) { return RecoveryConfirmation, nil },
		Now:       func() time.Time { return time.Date(2026, 8, 13, 12, 30, 0, 123, time.UTC) },
		DeriveKey: func(_, _ string) ([]byte, error) { return append([]byte(nil), f.key...), nil },
		LookPath: func(name string) (string, error) {
			switch name {
			case "gh":
				return f.ghPath, nil
			case "git":
				return f.gitPath, nil
			default:
				return "", errors.New("unexpected executable")
			}
		},
		Run: func(name string, args ...string) ([]byte, error) {
			command := name + " " + strings.Join(args, " ")
			switch command {
			case f.gitPath + " -C " + root + " remote get-url origin":
				return []byte(CanonicalOrigin + "\n"), nil
			case f.gitPath + " -C " + root + " show HEAD:" + RegistryRelativePath:
				return append([]byte(nil), original...), nil
			case f.ghPath + " api graphql --hostname github.com -f query=" + githubAuthorizationQuery:
				return []byte(`{"data":{"viewer":{"login":"Alexander-Salvador","databaseId":97975177},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":1328456440,"viewerPermission":"ADMIN"}}}`), nil
			default:
				return nil, errors.New("unexpected command: " + command)
			}
		},
		SaveSession: func(session RecoverySession) error {
			copy := session
			f.saved = &copy
			return nil
		},
		RemoveSession: func() error {
			f.saved = nil
			f.sessionRemoved = true
			return nil
		},
		CaptureSession: func() (RecoverySessionSnapshot, error) {
			if f.saved == nil {
				return RecoverySessionSnapshot{}, nil
			}
			copy := *f.saved
			return RecoverySessionSnapshot{Exists: true, Session: copy, Mode: 0o600}, nil
		},
		RestoreSession: func(snapshot RecoverySessionSnapshot) error {
			if !snapshot.Exists {
				f.saved = nil
				f.sessionRemoved = true
				return nil
			}
			copy := snapshot.Session
			f.saved = &copy
			return nil
		},
	}
	return f
}

func TestCanonicalCEOSelection(t *testing.T) {
	valid := func() Registry {
		var reg Registry
		if err := yaml.Unmarshal([]byte(recoveryRegistryYAML), &reg); err != nil {
			t.Fatal(err)
		}
		return reg
	}
	tests := []struct {
		name    string
		mutate  func(*Registry)
		wantErr bool
	}{
		{name: "valid unique CEO"},
		{name: "wrong registry version", mutate: func(reg *Registry) { reg.Version = 2 }, wantErr: true},
		{name: "non canonical registry", mutate: func(reg *Registry) { reg.Canonical = false }, wantErr: true},
		{name: "non exact CEO ID", mutate: func(reg *Registry) { reg.Identities[0].ID = " CEO-ALEXANDER " }, wantErr: true},
		{name: "normalized duplicate rejected", mutate: func(reg *Registry) {
			duplicate := reg.Identities[0]
			duplicate.ID = " CEO-ALEXANDER "
			duplicate.Status = " ACTIVE "
			reg.Identities = append(reg.Identities, duplicate)
		}, wantErr: true},
		{name: "wrong role", mutate: func(reg *Registry) { reg.Identities[0].Role = "lead" }, wantErr: true},
		{name: "wrong level", mutate: func(reg *Registry) { reg.Identities[0].Level = 9 }, wantErr: true},
		{name: "missing manage identities", mutate: func(reg *Registry) {
			reg.Identities[0].Permissions = []string{"full_system"}
		}, wantErr: true},
		{name: "missing full system", mutate: func(reg *Registry) {
			reg.Identities[0].Permissions = []string{"manage_identities"}
		}, wantErr: true},
		{name: "inactive", mutate: func(reg *Registry) { reg.Identities[0].Status = "suspended" }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := valid()
			if tt.mutate != nil {
				tt.mutate(&reg)
			}
			id, err := CanonicalCEO(&reg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CanonicalCEO() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id.ID != CanonicalCEOID {
				t.Fatalf("CanonicalCEO() ID = %q", id.ID)
			}
		})
	}
}

func TestRecoverCEOFailClosedPreconditions(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, *recoveryFixture)
		want    string
	}{
		{name: "non TTY", prepare: func(_ *testing.T, f *recoveryFixture) { f.deps.IsTTY = func() bool { return false } }, want: "TTY"},
		{name: "wrong origin", prepare: func(_ *testing.T, f *recoveryFixture) {
			base := f.deps.Run
			f.deps.Run = func(name string, args ...string) ([]byte, error) {
				if strings.Contains(strings.Join(args, " "), "remote get-url origin") {
					return []byte("git@github.com:ovav-dev/ovav.git\n"), nil
				}
				return base(name, args...)
			}
		}, want: "origin"},
		{name: "dirty registry", prepare: func(_ *testing.T, f *recoveryFixture) {
			path := filepath.Join(f.root, RegistryRelativePath)
			_ = os.WriteFile(path, append(f.original, []byte("# dirty\n")...), 0o600)
		}, want: "byte-identical"},
		{name: "registry symlink", prepare: func(t *testing.T, f *recoveryFixture) {
			path := filepath.Join(f.root, RegistryRelativePath)
			target := filepath.Join(f.root, "substitute.yaml")
			if err := os.WriteFile(target, f.original, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		}, want: "symlink"},
		{name: "short seed", prepare: func(_ *testing.T, f *recoveryFixture) {
			f.deps.ReadSeed = func() (string, error) { return "too-short", nil }
		}, want: "16"},
		{name: "bad confirmation", prepare: func(_ *testing.T, f *recoveryFixture) {
			f.deps.Confirm = func(RecoverySummary) (string, error) { return "recover ceo", nil }
		}, want: RecoveryConfirmation},
		{name: "invalid canonical CEO", prepare: func(t *testing.T, f *recoveryFixture) {
			bad := bytes.Replace(f.original, []byte("level: 10"), []byte("level: 9"), 1)
			if err := os.WriteFile(filepath.Join(f.root, RegistryRelativePath), bad, 0o600); err != nil {
				t.Fatal(err)
			}
			f.original = bad
			base := f.deps.Run
			f.deps.Run = func(name string, args ...string) ([]byte, error) {
				if strings.Contains(strings.Join(args, " "), "show HEAD:") {
					return bad, nil
				}
				return base(name, args...)
			}
		}, want: "level 10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecoveryFixture(t)
			tt.prepare(t, f)
			_, err := RecoverCEO(f.root, f.deps)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RecoverCEO() error = %v, want containing %q", err, tt.want)
			}
			if f.saved != nil {
				t.Fatal("session saved after failed precondition")
			}
		})
	}
}

func TestRecoverCEOGitHubAuthorization(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		err      error
		want     string
	}{
		{name: "gh absent", err: errors.New("executable file not found"), want: "GitHub CLI"},
		{name: "not authenticated", err: errors.New("authentication required"), want: "GitHub CLI"},
		{name: "malformed response", response: []byte(`{`), want: "GitHub authorization"},
		{name: "missing immutable user ID", response: []byte(`{"data":{"viewer":{"login":"braka","databaseId":0},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":1328456440,"viewerPermission":"ADMIN"}}}`), want: "GitHub authorization"},
		{name: "non admin", response: []byte(`{"data":{"viewer":{"login":"braka","databaseId":123},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":1328456440,"viewerPermission":"WRITE"}}}`), want: "ADMIN"},
		{name: "wrong repo response", response: []byte(`{"data":{"viewer":{"login":"braka","databaseId":123},"repository":{"nameWithOwner":"fork/ovav","databaseId":1328456440,"viewerPermission":"ADMIN"}}}`), want: "canonical repository"},
		{name: "wrong immutable repo ID", response: []byte(`{"data":{"viewer":{"login":"braka","databaseId":123},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":7,"viewerPermission":"ADMIN"}}}`), want: "repository ID"},
		{name: "wrong CEO login", response: []byte(`{"data":{"viewer":{"login":"attacker","databaseId":97975177},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":1328456440,"viewerPermission":"ADMIN"}}}`), want: "canonical CEO"},
		{name: "wrong CEO ID", response: []byte(`{"data":{"viewer":{"login":"Alexander-Salvador","databaseId":123},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":1328456440,"viewerPermission":"ADMIN"}}}`), want: "canonical CEO"},
		{name: "CEO login case normalized", response: []byte(`{"data":{"viewer":{"login":"alexander-salvador","databaseId":97975177},"repository":{"nameWithOwner":"ovav-dev/ovav","databaseId":1328456440,"viewerPermission":"ADMIN"}}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecoveryFixture(t)
			if tt.err != nil && tt.name == "gh absent" {
				f.deps.LookPath = func(name string) (string, error) {
					if name == "gh" {
						return "", tt.err
					}
					return f.gitPath, nil
				}
			} else {
				base := f.deps.Run
				f.deps.Run = func(name string, args ...string) ([]byte, error) {
					if name == f.ghPath {
						if tt.err != nil {
							return nil, tt.err
						}
						return tt.response, nil
					}
					return base(name, args...)
				}
			}
			_, err := RecoverCEO(f.root, f.deps)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("RecoverCEO() unexpected error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RecoverCEO() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRecoverCEOSuccess(t *testing.T) {
	f := newRecoveryFixture(t)
	result, err := RecoverCEO(f.root, f.deps)
	if err != nil {
		t.Fatalf("RecoverCEO(): %v", err)
	}

	expectedHashBytes := sha256.Sum256(f.key)
	expectedHash := hex.EncodeToString(expectedHashBytes[:])
	if result.NewHashPrefix != expectedHash[:12] || result.OldHashPrefix != strings.Repeat("a", 12) {
		t.Fatalf("hash prefixes = %q/%q", result.OldHashPrefix, result.NewHashPrefix)
	}
	if f.saved == nil {
		t.Fatal("canonical session was not saved")
	}
	if f.saved.IdentityID != CanonicalCEOID || f.saved.Role != "ceo" || f.saved.Level != 10 ||
		f.saved.Name != "Alexander Salvador" || f.saved.Email != "alexander@ovav.dev" {
		t.Fatalf("non-canonical session: %+v", *f.saved)
	}

	backupPath := filepath.Join(f.root, filepath.FromSlash(result.BackupRelativePath))
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, f.original) {
		t.Fatal("backup is not byte-identical")
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup mode = %o, want 600", info.Mode().Perm())
	}

	valid, err := VerifySignature(f.root, f.key)
	if err != nil || !valid {
		t.Fatalf("rotated signature valid = %v, error = %v", valid, err)
	}
	reg, err := LoadRegistry(f.root)
	if err != nil {
		t.Fatal(err)
	}
	ceo, err := CanonicalCEO(reg)
	if err != nil {
		t.Fatal(err)
	}
	if ceo.KeyHash != expectedHash {
		t.Fatalf("CEO key hash = %q", ceo.KeyHash)
	}

	var before Registry
	if err := yaml.Unmarshal(f.original, &before); err != nil {
		t.Fatal(err)
	}
	beforeCEO, _ := CanonicalCEO(&before)
	beforeCEO.KeyHash = expectedHash
	before.Signature = reg.Signature
	if !reflect.DeepEqual(before, *reg) {
		t.Fatalf("unrelated registry semantics changed\nbefore: %#v\nafter: %#v", before, *reg)
	}

	auditData, err := os.ReadFile(filepath.Join(f.root, AuditRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(auditData, []byte("correct horse")) || bytes.Contains(auditData, f.key) ||
		bytes.Contains(auditData, []byte(expectedHash)) || bytes.Contains(auditData, []byte(strings.Repeat("a", 64))) {
		t.Fatalf("audit leaked secret or full hash: %s", auditData)
	}
	var audit RecoveryAuditEntry
	if err := json.Unmarshal(bytes.TrimSpace(auditData), &audit); err != nil {
		t.Fatal(err)
	}
	if audit.Action != "identity_recovered" || audit.GitHubLogin != CanonicalCEOLogin || audit.GitHubUserID != CanonicalCEOUserID {
		t.Fatalf("unexpected audit: %+v", audit)
	}
	if len(audit.OldHashPrefix) > 12 || len(audit.NewHashPrefix) > 12 {
		t.Fatalf("audit hash prefixes are too long: %+v", audit)
	}
}

func TestRecoverCEOLockContention(t *testing.T) {
	f := newRecoveryFixture(t)
	lockPath := filepath.Join(f.root, RecoveryLockRelativePath)
	if err := os.WriteFile(lockPath, []byte("held"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := RecoverCEO(f.root, f.deps)
	if err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
}

func TestRecoverCEORollback(t *testing.T) {
	tests := []struct {
		name   string
		inject func(*recoveryFixture)
	}{
		{name: "atomic write failure", inject: func(f *recoveryFixture) {
			f.deps.WriteRegistry = func(string, []byte) (bool, error) {
				return false, errors.New("injected pre-rename atomic write failure")
			}
		}},
		{name: "post verify failure", inject: func(f *recoveryFixture) {
			f.deps.PostVerify = func() error { return errors.New("injected post-verify failure") }
		}},
		{name: "session save failure", inject: func(f *recoveryFixture) {
			f.deps.SaveSession = func(RecoverySession) error { return errors.New("injected session failure") }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecoveryFixture(t)
			tt.inject(f)
			_, err := RecoverCEO(f.root, f.deps)
			if err == nil {
				t.Fatal("RecoverCEO() succeeded with injected failure")
			}
			got, readErr := os.ReadFile(filepath.Join(f.root, RegistryRelativePath))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, f.original) {
				t.Fatalf("registry was not atomically rolled back: recovery error=%v", err)
			}
			if f.saved != nil {
				t.Fatal("session remains after rollback")
			}
		})
	}
}

func TestRecoverCEORequiresExactCanonicalOrigin(t *testing.T) {
	tests := []string{
		"https://github.com/ovav-dev/ovav",
		"https://github.com/ovav-dev/ovav.git/",
		"https://github.com/OVAV-DEV/ovav.git",
		"https://github.com/ovav-dev/ovav.git?ref=main",
		"git@github.com:ovav-dev/ovav.git",
	}
	for _, origin := range tests {
		t.Run(origin, func(t *testing.T) {
			f := newRecoveryFixture(t)
			base := f.deps.Run
			f.deps.Run = func(name string, args ...string) ([]byte, error) {
				if name == f.gitPath && len(args) == 5 && args[2] == "remote" {
					return []byte(origin + "\n"), nil
				}
				return base(name, args...)
			}
			_, err := RecoverCEO(f.root, f.deps)
			if err == nil || !strings.Contains(err.Error(), "exactly") {
				t.Fatalf("RecoverCEO() origin %q error = %v", origin, err)
			}
		})
	}
}

func TestRecoverCEORejectsRegistryParentSymlink(t *testing.T) {
	f := newRecoveryFixture(t)
	registryDir := filepath.Join(f.root, ".ovav", "registry")
	targetDir := filepath.Join(f.root, "substitute-registry")
	if err := os.Rename(registryDir, targetDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, registryDir); err != nil {
		t.Fatal(err)
	}

	_, err := RecoverCEO(f.root, f.deps)
	if err == nil || !strings.Contains(err.Error(), "parent rejected") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
}

func TestRecoverCEOInputFailuresDoNotReachRegistryWrite(t *testing.T) {
	tests := []struct {
		name   string
		inject func(*recoveryFixture)
		want   string
	}{
		{name: "confirmation read", inject: func(f *recoveryFixture) {
			f.deps.Confirm = func(RecoverySummary) (string, error) { return "", errors.New("injected confirm read") }
		}, want: "confirmation failed"},
		{name: "hidden seed read", inject: func(f *recoveryFixture) {
			f.deps.ReadSeed = func() (string, error) { return "", errors.New("injected hidden read") }
		}, want: "hidden seed input failed"},
		{name: "key derivation", inject: func(f *recoveryFixture) {
			f.deps.DeriveKey = func(_, _ string) ([]byte, error) { return nil, errors.New("injected derivation") }
		}, want: "key derivation failed"},
		{name: "empty derived key", inject: func(f *recoveryFixture) {
			f.deps.DeriveKey = func(_, _ string) ([]byte, error) { return nil, nil }
		}, want: "empty key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecoveryFixture(t)
			wrote := false
			f.deps.WriteRegistry = func(string, []byte) (bool, error) {
				wrote = true
				return true, nil
			}
			tt.inject(f)
			_, err := RecoverCEO(f.root, f.deps)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("RecoverCEO() error = %v, want containing %q", err, tt.want)
			}
			if wrote {
				t.Fatal("registry write reached after rejected interactive input")
			}
		})
	}
}

func TestVerifySignatureRejectsTamperedMetadata(t *testing.T) {
	f := newRecoveryFixture(t)
	if _, err := RecoverCEO(f.root, f.deps); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.root, RegistryRelativePath)
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ old, replacement string }{
		{"algorithm: HMAC-SHA256-V2", "algorithm: hmac-sha256-v2"},
		{"signed_by: ceo-alexander", "signed_by: attacker"},
		{"signed_at: \"2026-08-13T12:30:00.000000123Z\"", "signed_at: \"2026-08-13T12:31:00Z\""},
	} {
		tampered := bytes.Replace(valid, []byte(test.old), []byte(test.replacement), 1)
		if bytes.Equal(tampered, valid) {
			t.Fatalf("metadata fixture did not contain %q", test.old)
		}
		if err := os.WriteFile(path, tampered, 0o600); err != nil {
			t.Fatal(err)
		}
		if ok, verifyErr := VerifySignature(f.root, f.key); verifyErr == nil && ok {
			t.Fatalf("VerifySignature accepted metadata tamper %q", test.old)
		}
	}
}

func TestRecoverCEORejectsUnresolvedJournal(t *testing.T) {
	f := newRecoveryFixture(t)
	if err := os.WriteFile(filepath.Join(f.root, RecoveryJournalRelativePath), []byte(`{"stage":"registry_written"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverCEO(f.root, f.deps); err == nil || !strings.Contains(err.Error(), "unresolved recovery journal") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
}

func TestRecoverCEORejectsUntrustedGitHubExecutable(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*testing.T, *recoveryFixture)
	}{
		{name: "symlink", prepare: func(t *testing.T, f *recoveryFixture) {
			target := f.ghPath
			link := filepath.Join(f.root, "gh-link")
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			f.deps.LookPath = func(name string) (string, error) {
				if name == "gh" {
					return link, nil
				}
				return f.gitPath, nil
			}
		}},
		{name: "non regular", prepare: func(_ *testing.T, f *recoveryFixture) {
			f.deps.LookPath = func(name string) (string, error) {
				if name == "gh" {
					return f.root, nil
				}
				return f.gitPath, nil
			}
		}},
		{name: "world writable", prepare: func(t *testing.T, f *recoveryFixture) {
			path := filepath.Join(f.root, "world-writable-gh")
			if err := os.WriteFile(path, []byte("binary"), 0o757); err != nil {
				t.Fatal(err)
			}
			f.deps.LookPath = func(name string) (string, error) {
				if name == "gh" {
					return path, nil
				}
				return f.gitPath, nil
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecoveryFixture(t)
			tt.prepare(t, f)
			if _, err := RecoverCEO(f.root, f.deps); err == nil || !strings.Contains(err.Error(), "GitHub CLI") {
				t.Fatalf("RecoverCEO() error = %v", err)
			}
		})
	}
}

func TestRecoverCEOGitHubCommandPinsHostname(t *testing.T) {
	f := newRecoveryFixture(t)
	called := false
	base := f.deps.Run
	f.deps.Run = func(name string, args ...string) ([]byte, error) {
		if name == f.ghPath {
			called = true
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "api graphql --hostname github.com") {
				t.Fatalf("unpinned gh command: %s %s", name, joined)
			}
		}
		return base(name, args...)
	}
	if _, err := RecoverCEO(f.root, f.deps); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("gh was not called")
	}
}

func TestRecoverCEORollbackPreservesConcurrentPostWriteChange(t *testing.T) {
	f := newRecoveryFixture(t)
	registryPath := filepath.Join(f.root, RegistryRelativePath)
	concurrent := []byte("concurrent post-write registry\n")
	f.deps.PostVerify = func() error {
		if err := os.WriteFile(registryPath, concurrent, 0o600); err != nil {
			return err
		}
		return errors.New("trigger rollback")
	}
	_, err := RecoverCEO(f.root, f.deps)
	if err == nil || !strings.Contains(err.Error(), "manual recovery required") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
	got, readErr := os.ReadFile(registryPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, concurrent) {
		t.Fatal("rollback overwrote concurrent post-write registry")
	}
}

func TestRecoverCEORejectsRegistryParentSwapBeforeBackup(t *testing.T) {
	f := newRecoveryFixture(t)
	originalDir := filepath.Join(f.root, ".ovav", "registry")
	movedDir := filepath.Join(f.root, ".ovav", "registry-original")
	base := f.deps.DeriveKey
	f.deps.DeriveKey = func(seed, machineID string) ([]byte, error) {
		if err := os.Rename(originalDir, movedDir); err != nil {
			return nil, err
		}
		if err := os.Mkdir(originalDir, 0o700); err != nil {
			return nil, err
		}
		return base(seed, machineID)
	}
	if _, err := RecoverCEO(f.root, f.deps); err == nil || !strings.Contains(err.Error(), "directory identity changed") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(originalDir, "backups")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup created in swapped parent: %v", err)
	}
}

func TestRecoverCEOAuditFailurePreservesExistingAudit(t *testing.T) {
	f := newRecoveryFixture(t)
	auditPath := filepath.Join(f.root, AuditRelativePath)
	originalAudit := []byte("{\"action\":\"existing\"}\n")
	if err := os.WriteFile(auditPath, originalAudit, 0o600); err != nil {
		t.Fatal(err)
	}
	// Force the copy-on-write publication to fail without touching the existing audit.
	if err := os.Chmod(filepath.Dir(auditPath), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(auditPath), 0o700) })
	_, _ = RecoverCEO(f.root, f.deps)
	got, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, originalAudit) {
		t.Fatalf("existing audit changed after failure: %q", got)
	}
}

func TestAtomicReplaceReportsPublishedOnDirectorySyncFailure(t *testing.T) {
	f := newRecoveryFixture(t)
	path := filepath.Join(f.root, RegistryRelativePath)
	_, info, err := readRegularNoSymlink(path)
	if err != nil {
		t.Fatal(err)
	}
	rotated := []byte("rotated registry\n")
	published, err := atomicReplaceCheckedWithSync(path, rotated, f.original, info, func(string) error {
		return errors.New("injected directory sync failure")
	})
	if err == nil || !published {
		t.Fatalf("atomicReplaceCheckedWithSync() published=%v error=%v", published, err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, rotated) {
		t.Fatalf("rename was not committed: %q", got)
	}
}

func TestRecoverCEORegistrySyncFailureRetainsPublishedRegistryAndJournal(t *testing.T) {
	f := newRecoveryFixture(t)
	f.deps.WriteRegistry = func(path string, data []byte) (bool, error) {
		_, info, err := readRegularNoSymlink(path)
		if err != nil {
			return false, err
		}
		return atomicReplaceCheckedWithSync(path, data, f.original, info, func(string) error {
			return errors.New("injected registry directory sync failure")
		})
	}
	_, err := RecoverCEO(f.root, f.deps)
	if err == nil || !strings.Contains(err.Error(), "published") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
	registry, readErr := os.ReadFile(filepath.Join(f.root, RegistryRelativePath))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if bytes.Equal(registry, f.original) {
		t.Fatal("published registry was unsafely rolled back")
	}
	if _, statErr := os.Stat(filepath.Join(f.root, RecoveryJournalRelativePath)); statErr != nil {
		t.Fatalf("journal not retained: %v", statErr)
	}
}

func TestRecoveryAuditCASConflictPreservesConcurrentRecord(t *testing.T) {
	f := newRecoveryFixture(t)
	auditPath := filepath.Join(f.root, AuditRelativePath)
	before := []byte("{\"action\":\"existing\"}\n")
	if err := os.WriteFile(auditPath, before, 0o600); err != nil {
		t.Fatal(err)
	}
	_, staleInfo, err := readRegularNoSymlink(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	concurrent := append(append([]byte(nil), before...), []byte("{\"action\":\"concurrent\"}\n")...)
	if err := os.WriteFile(auditPath, concurrent, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate := append(append([]byte(nil), before...), []byte("{\"action\":\"identity_recovered\"}\n")...)
	if err := atomicReplaceCAS(auditPath, candidate, before, staleInfo, false); err == nil || !strings.Contains(err.Error(), "changed during recovery") {
		t.Fatalf("atomicReplaceCAS() error = %v, want audit CAS conflict", err)
	}
	got, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, concurrent) {
		t.Fatalf("audit CAS conflict overwrote concurrent record: got %q, want %q", got, concurrent)
	}
}

func TestRecoverCEOBackupDirectoryIsPrivate(t *testing.T) {
	f := newRecoveryFixture(t)
	backupDir := filepath.Join(f.root, ".ovav", "registry", "backups", "identity-recovery")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := RecoverCEO(f.root, f.deps); err != nil {
		t.Fatalf("RecoverCEO(): %v", err)
	}
	info, err := os.Stat(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("backup directory mode = %o, want 700", info.Mode().Perm())
	}
}

func TestRecoverCEODoesNotOverwriteConcurrentRegistryChange(t *testing.T) {
	f := newRecoveryFixture(t)
	registryPath := filepath.Join(f.root, RegistryRelativePath)
	concurrent := append(append([]byte(nil), f.original...), []byte("# concurrent operator update\n")...)
	baseDerive := f.deps.DeriveKey
	f.deps.DeriveKey = func(seed, machineID string) ([]byte, error) {
		if err := os.WriteFile(registryPath, concurrent, 0o600); err != nil {
			return nil, err
		}
		return baseDerive(seed, machineID)
	}

	_, err := RecoverCEO(f.root, f.deps)
	if err == nil || !strings.Contains(err.Error(), "changed during recovery") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
	got, readErr := os.ReadFile(registryPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, concurrent) {
		t.Fatalf("concurrent registry update was overwritten\ngot:\n%s\nwant:\n%s", got, concurrent)
	}
}

func TestRecoverCEOFailurePreservesPreexistingSession(t *testing.T) {
	tests := []struct {
		name   string
		inject func(*testing.T, *recoveryFixture)
	}{
		{name: "post verify failure", inject: func(_ *testing.T, f *recoveryFixture) {
			f.deps.PostVerify = func() error { return errors.New("injected post-verify failure") }
		}},
		{name: "partial session save failure", inject: func(_ *testing.T, f *recoveryFixture) {
			f.deps.SaveSession = func(RecoverySession) error {
				f.saved = &RecoverySession{IdentityID: "partially-written-recovery-session"}
				return errors.New("injected partial session save failure")
			}
		}},
		{name: "audit failure", inject: func(t *testing.T, f *recoveryFixture) {
			if err := os.Mkdir(filepath.Join(f.root, AuditRelativePath), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecoveryFixture(t)
			preexisting := &RecoverySession{IdentityID: "preexisting-session", VaultKeyHash: "preexisting-hash"}
			f.saved = preexisting
			f.deps.RemoveSession = func() error {
				f.saved = nil
				f.sessionRemoved = true
				return nil
			}
			tt.inject(t, f)

			if _, err := RecoverCEO(f.root, f.deps); err == nil {
				t.Fatal("RecoverCEO() succeeded with injected failure")
			}
			if f.saved == nil || !reflect.DeepEqual(*f.saved, *preexisting) {
				t.Fatalf("preexisting session was not preserved: got %+v, want %+v", f.saved, preexisting)
			}
		})
	}
}

func TestRecoverCEOLockIsCleanedAndSymlinkRejected(t *testing.T) {
	t.Run("cleaned after failure", func(t *testing.T) {
		f := newRecoveryFixture(t)
		f.deps.Confirm = func(RecoverySummary) (string, error) { return "no", nil }
		if _, err := RecoverCEO(f.root, f.deps); err == nil {
			t.Fatal("RecoverCEO() succeeded without exact confirmation")
		}
		if _, err := os.Lstat(filepath.Join(f.root, RecoveryLockRelativePath)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("recovery lock remains after failure: %v", err)
		}
	})

	t.Run("symlink rejected without target modification", func(t *testing.T) {
		f := newRecoveryFixture(t)
		target := filepath.Join(f.root, "lock-target")
		if err := os.WriteFile(target, []byte("do not modify"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(f.root, RecoveryLockRelativePath)); err != nil {
			t.Fatal(err)
		}
		if _, err := RecoverCEO(f.root, f.deps); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("RecoverCEO() error = %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "do not modify" {
			t.Fatalf("lock symlink target changed: %q", got)
		}
	})
}

func TestRecoverCEOAuditFailureRollsBackWithoutAppendingSuccess(t *testing.T) {
	f := newRecoveryFixture(t)
	auditPath := filepath.Join(f.root, AuditRelativePath)
	if err := os.Mkdir(auditPath, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := RecoverCEO(f.root, f.deps)
	if err == nil || !strings.Contains(err.Error(), "durable audit") {
		t.Fatalf("RecoverCEO() error = %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(f.root, RegistryRelativePath))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, f.original) {
		t.Fatal("registry was not restored after audit failure")
	}
	if !f.sessionRemoved || f.saved != nil {
		t.Fatal("new recovery session remains after audit failure")
	}
	entries, readDirErr := os.ReadDir(auditPath)
	if readDirErr != nil {
		t.Fatal(readDirErr)
	}
	if len(entries) != 0 {
		t.Fatalf("audit failure left partial entries: %v", entries)
	}
}
