//go:build linux

package hostprojection

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

type fixture struct {
	base, root, backup, source, destination string
}

func newFixture(t *testing.T, existing bool) fixture {
	t.Helper()
	base := t.TempDir()
	f := fixture{
		base: base, root: filepath.Join(base, "allowed"), backup: filepath.Join(base, "backups"),
		source: filepath.Join(base, "source"), destination: filepath.Join(base, "allowed", "config"),
	}
	mustMkdir(t, f.root, 0o755)
	mustMkdir(t, f.backup, 0o700)
	mustWrite(t, f.source, "new content", 0o600)
	if existing {
		mustWrite(t, f.destination, "old content", 0o640)
	}
	return f
}

func newFixtureWithoutBackup(t *testing.T, existing bool) fixture {
	t.Helper()
	f := newFixture(t, existing)
	if err := os.Remove(f.backup); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f fixture) plan(t *testing.T) *Transaction {
	t.Helper()
	tx, err := Plan(f.source, f.destination, f.root, f.backup, time.Unix(1, 2))
	if err != nil {
		t.Fatalf("Plan(): %v", err)
	}
	return tx
}

func TestPlanRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, f fixture) (string, string, string)
	}{
		{"source symlink", func(t *testing.T, f fixture) (string, string, string) {
			path := filepath.Join(f.base, "source-link")
			mustSymlink(t, f.source, path)
			return path, f.destination, f.backup
		}},
		{"source symlink parent", func(t *testing.T, f fixture) (string, string, string) {
			parent := filepath.Join(f.base, "source-parent-link")
			mustSymlink(t, f.base, parent)
			return filepath.Join(parent, "source"), f.destination, f.backup
		}},
		{"destination outside root", func(t *testing.T, f fixture) (string, string, string) {
			return f.source, filepath.Join(f.base, "outside"), f.backup
		}},
		{"destination symlink", func(t *testing.T, f fixture) (string, string, string) {
			mustSymlink(t, f.source, f.destination)
			return f.source, f.destination, f.backup
		}},
		{"destination symlink parent", func(t *testing.T, f fixture) (string, string, string) {
			realParent := filepath.Join(f.base, "real-parent")
			mustMkdir(t, realParent, 0o755)
			linked := filepath.Join(f.root, "linked")
			mustSymlink(t, realParent, linked)
			return f.source, filepath.Join(linked, "config"), f.backup
		}},
		{"backup mode", func(t *testing.T, f fixture) (string, string, string) {
			if err := os.Chmod(f.backup, 0o755); err != nil {
				t.Fatal(err)
			}
			return f.source, f.destination, f.backup
		}},
		{"backup symlink", func(t *testing.T, f fixture) (string, string, string) {
			linked := filepath.Join(f.base, "backup-link")
			mustSymlink(t, f.backup, linked)
			return f.source, f.destination, linked
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, false)
			source, destination, backup := tt.setup(t, f)
			if _, err := Plan(source, destination, f.root, backup, time.Unix(1, 2)); err == nil {
				t.Fatal("Plan() error = nil, want fail-closed rejection")
			}
		})
	}
}

func TestPlanDryRunAndDurabilityPreview(t *testing.T) {
	t.Parallel()
	f := newFixture(t, false)
	tx := f.plan(t)
	p := tx.Preview()
	if !p.PlatformSupported || runtime.GOOS != "linux" {
		t.Fatalf("platform support = %v on %s", p.PlatformSupported, runtime.GOOS)
	}
	if p.Durability != DurabilityFull && p.Durability != DurabilityDegraded {
		t.Fatalf("Durability = %q", p.Durability)
	}
	if !strings.Contains(p.BackupPath, "19700101T000001.000000002Z") {
		t.Fatalf("BackupPath = %q, want deterministic UTC timestamp", p.BackupPath)
	}
	for _, path := range []string{p.BackupPath, p.JournalPath, p.LockPath, p.Destination} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("dry-run created %s: %v", path, err)
		}
	}
	if !isUnsupportedDirSync(syscall.EINVAL) || !isUnsupportedDirSync(syscall.EOPNOTSUPP) {
		t.Fatal("DrvFs-style directory fsync errors must classify as degraded")
	}
}

func TestMissingBackupRootDryRunAndSecureCreation(t *testing.T) {
	f := newFixtureWithoutBackup(t, true)
	tx := f.plan(t)
	if _, err := os.Lstat(f.backup); !os.IsNotExist(err) {
		t.Fatalf("Plan created backup root: %v", err)
	}
	result, err := tx.Apply()
	if err != nil || !result.Applied {
		t.Fatalf("Apply() = %+v, %v", result, err)
	}
	assertMode(t, f.backup, 0o700)
	assertOwner(t, f.backup)
	for _, path := range []string{tx.Preview().LockPath, tx.Preview().JournalPath, tx.Preview().BackupPath} {
		assertMode(t, path, 0o600)
		assertOwner(t, path)
	}
}

func TestMissingBackupRootRejectsPostPlanRaces(t *testing.T) {
	tests := []struct {
		name string
		race func(t *testing.T, f fixture)
	}{
		{"symlink", func(t *testing.T, f fixture) { mustSymlink(t, f.base, f.backup) }},
		{"wrong mode directory", func(t *testing.T, f fixture) { mustMkdir(t, f.backup, 0o755) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixtureWithoutBackup(t, true)
			tx := f.plan(t)
			tt.race(t, f)
			if _, err := tx.Apply(); err == nil {
				t.Fatal("Apply() error = nil, want backup-root race rejection")
			}
			assertContent(t, f.destination, "old content")
			if _, err := os.Lstat(tx.Preview().JournalPath); !os.IsNotExist(err) {
				t.Fatalf("race created journal: %v", err)
			}
		})
	}
}

func TestPrivateArtifactOwnershipHelper(t *testing.T) {
	valid := snapshot{mode: 0o600, links: 1, owner: uint32(os.Geteuid())}
	if err := verifyPrivateArtifact(valid, "test artifact"); err != nil {
		t.Fatalf("valid artifact rejected: %v", err)
	}
	invalid := valid
	invalid.owner++
	if err := verifyPrivateArtifact(invalid, "test artifact"); err == nil {
		t.Fatal("foreign-owned artifact accepted")
	}
}

func TestApplyRollbackAndIdempotentRecovery(t *testing.T) {
	tests := []struct {
		name     string
		existing bool
	}{
		{"replace existing", true},
		{"create new", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.existing)
			tx := f.plan(t)
			applied, err := tx.Apply()
			if err != nil {
				t.Fatalf("Apply(): %v", err)
			}
			if !applied.Applied || applied.JournalState != "verified" {
				t.Fatalf("Apply() result = %+v", applied)
			}
			assertContent(t, f.destination, "new content")
			assertMode(t, tx.Preview().JournalPath, 0o600)
			assertOwner(t, tx.Preview().JournalPath)
			assertOwner(t, tx.Preview().LockPath)
			assertMode(t, f.backup, 0o700)
			assertOwner(t, f.backup)
			if tt.existing {
				assertContent(t, tx.Preview().BackupPath, "old content")
				assertOwner(t, tx.Preview().BackupPath)
				assertMode(t, f.destination, 0o640)
			}

			rolled, err := Recover(tx.Preview().JournalPath, f.root, f.backup)
			if err != nil || !rolled.RolledBack || !rolled.Recovered {
				t.Fatalf("Recover() = %+v, %v", rolled, err)
			}
			if tt.existing {
				assertContent(t, f.destination, "old content")
			} else if _, err := os.Lstat(f.destination); !os.IsNotExist(err) {
				t.Fatalf("created destination survived recovery: %v", err)
			}
			again, err := tx.Rollback()
			if err != nil || !again.AlreadyComplete || !again.RolledBack {
				t.Fatalf("idempotent Rollback() = %+v, %v", again, err)
			}
		})
	}
}

func TestRollbackRejectsConcurrentDestinationMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, path string)
	}{
		{"in-place content", func(t *testing.T, path string) { mustWrite(t, path, "external", 0o640) }},
		{"same content new inode", func(t *testing.T, path string) {
			replacement := path + ".external"
			mustWrite(t, replacement, "new content", 0o640)
			if err := os.Rename(replacement, path); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, true)
			tx := f.plan(t)
			if _, err := tx.Apply(); err != nil {
				t.Fatalf("Apply(): %v", err)
			}
			tt.mutate(t, f.destination)
			if _, err := tx.Rollback(); !errors.Is(err, ErrConcurrentChange) {
				t.Fatalf("Rollback() error = %v, want ErrConcurrentChange", err)
			}
			if tt.name == "in-place content" {
				assertContent(t, f.destination, "external")
			} else {
				assertContent(t, f.destination, "new content")
			}
		})
	}
}

func TestApplyLockContention(t *testing.T) {
	f := newFixture(t, true)
	tx := f.plan(t)
	backupDir, err := openBackupRoot(f.backup)
	if err != nil {
		t.Fatal(err)
	}
	defer backupDir.Close()
	lock, err := openLockAt(backupDir, filepath.Base(tx.Preview().LockPath))
	if err != nil {
		t.Fatal(err)
	}
	defer closeLock(lock)
	if _, err := tx.Apply(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Apply() error = %v, want ErrLocked", err)
	}
	assertContent(t, f.destination, "old content")
}

func TestApplyFailureAutomaticallyRollsBack(t *testing.T) {
	f := newFixture(t, true)
	tx := f.plan(t)
	tx.afterRename = func(string) error { return errors.New("injected verification failure") }
	result, err := tx.Apply()
	if err == nil || !result.RolledBack {
		t.Fatalf("Apply() = %+v, %v; want automatic rollback", result, err)
	}
	assertContent(t, f.destination, "old content")
	again, err := Recover(tx.Preview().JournalPath, f.root, f.backup)
	if err != nil || !again.AlreadyComplete {
		t.Fatalf("Recover() after automatic rollback = %+v, %v", again, err)
	}
}

func TestInspectJournalRejectsSymlinkAndOversize(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		f := newFixture(t, true)
		tx := f.plan(t)
		if _, err := tx.Apply(); err != nil {
			t.Fatal(err)
		}
		journalPath := tx.Preview().JournalPath
		realPath := journalPath + ".real"
		if err := os.Rename(journalPath, realPath); err != nil {
			t.Fatal(err)
		}
		mustSymlink(t, realPath, journalPath)
		if _, err := InspectJournal(journalPath, f.backup); err == nil {
			t.Fatal("InspectJournal accepted a symlink journal")
		}
		assertContent(t, f.destination, "new content")
	})

	t.Run("hard link", func(t *testing.T) {
		f := newFixture(t, true)
		tx := f.plan(t)
		if _, err := tx.Apply(); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(tx.Preview().JournalPath, tx.Preview().JournalPath+".link"); err != nil {
			t.Fatal(err)
		}
		if _, err := InspectJournal(tx.Preview().JournalPath, f.backup); err == nil {
			t.Fatal("InspectJournal accepted a multiply-linked journal")
		}
		assertContent(t, f.destination, "new content")
	})

	t.Run("oversize", func(t *testing.T) {
		f := newFixture(t, true)
		tx := f.plan(t)
		if _, err := tx.Apply(); err != nil {
			t.Fatal(err)
		}
		oversize := strings.Repeat("x", int(maximumJournalBytes)+1)
		mustWrite(t, tx.Preview().JournalPath, oversize, 0o600)
		if _, err := InspectJournal(tx.Preview().JournalPath, f.backup); err == nil {
			t.Fatal("InspectJournal accepted an oversized journal")
		}
		assertContent(t, f.destination, "new content")
	})
}

func TestRecoverInspectedRejectsJournalTamperAndSwap(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, path string)
	}{
		{name: "content tamper", mutate: func(t *testing.T, path string) {
			mustWrite(t, path, "{}", 0o600)
		}},
		{name: "identity swap", mutate: func(t *testing.T, path string) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(path, path+".old"); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t, true)
			tx := f.plan(t)
			if _, err := tx.Apply(); err != nil {
				t.Fatal(err)
			}
			inspection, err := InspectJournal(tx.Preview().JournalPath, f.backup)
			if err != nil || inspection.Digest() == "" || inspection.Identity() == (JournalIdentity{}) {
				t.Fatalf("InspectJournal() = %+v, %v", inspection, err)
			}
			test.mutate(t, tx.Preview().JournalPath)
			if _, err := RecoverInspected(inspection, inspection.Authority()); !errors.Is(err, ErrConcurrentChange) {
				t.Fatalf("RecoverInspected() error = %v, want ErrConcurrentChange", err)
			}
			assertContent(t, f.destination, "new content")
		})
	}
}

func TestRecoverInspectedRejectsSymlinkAfterInspection(t *testing.T) {
	f := newFixture(t, true)
	tx := f.plan(t)
	if _, err := tx.Apply(); err != nil {
		t.Fatal(err)
	}
	inspection, err := InspectJournal(tx.Preview().JournalPath, f.backup)
	if err != nil {
		t.Fatal(err)
	}
	realPath := tx.Preview().JournalPath + ".old"
	if err := os.Rename(tx.Preview().JournalPath, realPath); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, realPath, tx.Preview().JournalPath)
	if _, err := RecoverInspected(inspection, inspection.Authority()); err == nil {
		t.Fatal("RecoverInspected accepted a journal symlink after inspection")
	}
	assertContent(t, f.destination, "new content")
}

func mustMkdir(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Mkdir(path, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatal(err)
	}
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("%s content = %q, %v; want %q", path, got, err, want)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != want {
		t.Fatalf("%s mode = %v, %v; want %04o", path, info, err, want)
	}
}

func assertOwner(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		t.Fatalf("%s owner = %v; want effective uid %d", path, info.Sys(), os.Geteuid())
	}
}
