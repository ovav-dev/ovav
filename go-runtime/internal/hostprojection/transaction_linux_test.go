//go:build linux

package hostprojection

import (
	"encoding/json"
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

func TestDestinationModePolicy(t *testing.T) {
	t.Parallel()
	valid := syscall.Stat_t{
		Mode:  syscall.S_IFREG | 0o777,
		Nlink: 1,
		Uid:   uint32(os.Geteuid()),
	}
	tests := []struct {
		name           string
		stat           syscall.Stat_t
		purpose        createdFilePurpose
		filesystemType int64
		wantDegraded   bool
		wantErr        bool
	}{
		{name: "strict ext4 match", stat: withMode(valid, 0o640), purpose: destinationArtifact, filesystemType: 0xEF53},
		{name: "strict ext4 mismatch", stat: valid, purpose: destinationArtifact, filesystemType: 0xEF53, wantErr: true},
		{name: "v9fs destination mismatch", stat: valid, purpose: destinationArtifact, filesystemType: v9fsMagic, wantDegraded: true},
		{name: "v9fs private artifact mismatch", stat: valid, purpose: privateArtifact, filesystemType: v9fsMagic, wantErr: true},
		{name: "v9fs non-regular", stat: withType(valid, syscall.S_IFDIR), purpose: destinationArtifact, filesystemType: v9fsMagic, wantErr: true},
		{name: "v9fs multiple links", stat: withLinks(valid, 2), purpose: destinationArtifact, filesystemType: v9fsMagic, wantErr: true},
		{name: "v9fs foreign owner", stat: withOwner(valid, valid.Uid+1), purpose: destinationArtifact, filesystemType: v9fsMagic, wantErr: true},
		{name: "v9fs special mode", stat: withSpecial(valid, syscall.S_ISUID), purpose: destinationArtifact, filesystemType: v9fsMagic, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			degraded, err := validateCreatedFile(test.stat, 0o640, test.purpose, test.filesystemType)
			if (err != nil) != test.wantErr || degraded != test.wantDegraded {
				t.Fatalf("validateCreatedFile() = (%v, %v), want degraded=%v err=%v", degraded, err, test.wantDegraded, test.wantErr)
			}
		})
	}
}

func TestV9FSClassifierAndDegradedResultPropagation(t *testing.T) {
	t.Parallel()
	if !isV9FS(v9fsMagic) || isV9FS(0xEF53) {
		t.Fatal("v9fs statfs classifier returned an unsafe result")
	}
	durability := newDurability()
	durability.noteDestinationFilesystem(v9fsMagic)
	if durability.level != DurabilityDegraded || durability.detail != destinationModeEnforcementUnsupported {
		t.Fatalf("v9fs durability = %+v", durability)
	}
	tx := &Transaction{preview: Preview{Durability: DurabilityFull}}
	result := tx.resultWithDurability("apply", "ready", durability)
	if result.Durability != DurabilityDegraded || result.DurabilityDetail != destinationModeEnforcementUnsupported {
		t.Fatalf("degraded result = %+v", result)
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

func TestV1RegularJournalRecoveryCompatibility(t *testing.T) {
	f := newFixture(t, true)
	tx := f.plan(t)
	if _, err := tx.Apply(); err != nil {
		t.Fatal(err)
	}
	rewriteJournalVersionOne(t, tx.preview.JournalPath, nil)
	recovered, err := Recover(tx.preview.JournalPath, f.root, f.backup)
	if err != nil || !recovered.RolledBack || !recovered.Recovered {
		t.Fatalf("Recover(v1) = %+v, %v", recovered, err)
	}
	assertContent(t, f.destination, "old content")

	t.Run("rejects v2 symlink field", func(t *testing.T) {
		other := newFixture(t, true)
		otherTx := other.plan(t)
		if _, err := otherTx.Apply(); err != nil {
			t.Fatal(err)
		}
		rewriteJournalVersionOne(t, otherTx.preview.JournalPath, map[string]any{"original_link_text": "/tmp/forbidden"})
		if _, err := InspectJournal(otherTx.preview.JournalPath, other.backup); err == nil {
			t.Fatal("InspectJournal() accepted a v1 journal with a v2 symlink field")
		}
	})
}

func TestExactSymlinkMigrationApplyRollbackAndRecovery(t *testing.T) {
	f := newFixture(t, false)
	targetDir := filepath.Join(f.base, "main")
	mustMkdir(t, targetDir, 0o755)
	target := filepath.Join(targetDir, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)

	tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(4, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatalf("PlanWithOptions(): %v", err)
	}
	preview := tx.Preview()
	if preview.DestinationKind != DestinationSymlink || preview.OriginalLinkText != target || preview.OriginalSHA256 != "" {
		t.Fatalf("symlink preview = %+v", preview)
	}
	applied, err := tx.Apply()
	if err != nil || !applied.Applied {
		t.Fatalf("Apply() = %+v, %v", applied, err)
	}
	info, err := os.Lstat(f.destination)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("applied destination = %v, %v; want regular file", info, err)
	}
	assertContent(t, f.destination, "new content")
	assertContent(t, target, "main repository config")
	assertMode(t, preview.MigrationMarker, 0o600)
	assertOwner(t, preview.MigrationMarker)
	journalData, err := os.ReadFile(preview.JournalPath)
	if err != nil {
		t.Fatal(err)
	}
	journalText := string(journalData)
	if !strings.Contains(journalText, `"version":2`) ||
		!strings.Contains(journalText, `"destination_kind":"symlink"`) ||
		!strings.Contains(journalText, `"original_link_text":"`+target+`"`) ||
		!strings.Contains(journalText, `"profile_id":"opencode-bootstrap"`) ||
		!strings.Contains(journalText, `"migration_id":"opencode-bootstrap-symlink-v1"`) ||
		!strings.Contains(journalText, `"marker_sha256":"`) {
		t.Fatalf("journal does not preserve symlink authority: %s", journalText)
	}
	appliedPath := f.destination + ".applied"
	if err := os.Rename(f.destination, appliedPath); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, target, f.destination)
	if _, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(4, 1), exactMigrationOptions(target)); !errors.Is(err, ErrMigrationConsumed) {
		t.Fatalf("consumed migration PlanWithOptions() error = %v, want ErrMigrationConsumed", err)
	}
	if err := os.Remove(f.destination); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(appliedPath, f.destination); err != nil {
		t.Fatal(err)
	}

	rolled, err := tx.Rollback()
	if err != nil || !rolled.RolledBack {
		t.Fatalf("Rollback() = %+v, %v", rolled, err)
	}
	linkText, err := os.Readlink(f.destination)
	if err != nil || linkText != target {
		t.Fatalf("restored symlink = %q, %v; want %q", linkText, err, target)
	}
	assertContent(t, target, "main repository config")
	if _, err := os.Lstat(preview.MigrationMarker); !os.IsNotExist(err) {
		t.Fatalf("rollback retained consumed migration marker: %v", err)
	}
	if _, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(4, 2), exactMigrationOptions(target)); err != nil {
		t.Fatalf("rollback did not reopen migration epoch: %v", err)
	}
	again, err := Recover(preview.JournalPath, f.root, f.backup)
	if err != nil || !again.RolledBack || !again.AlreadyComplete {
		t.Fatalf("idempotent Recover() = %+v, %v", again, err)
	}
}

func TestExactSymlinkMigrationRejectsUntrustedTargets(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, f fixture) string
	}{
		{name: "relative link", setup: func(t *testing.T, f fixture) string {
			target := filepath.Join(f.base, "target")
			mustWrite(t, target, "target", 0o600)
			mustSymlink(t, "../target", f.destination)
			return target
		}},
		{name: "target mismatch", setup: func(t *testing.T, f fixture) string {
			target := filepath.Join(f.base, "target")
			other := filepath.Join(f.base, "other")
			mustWrite(t, target, "target", 0o600)
			mustWrite(t, other, "other", 0o600)
			mustSymlink(t, other, f.destination)
			return target
		}},
		{name: "target is symlink", setup: func(t *testing.T, f fixture) string {
			realTarget := filepath.Join(f.base, "real-target")
			target := filepath.Join(f.base, "target")
			mustWrite(t, realTarget, "target", 0o600)
			mustSymlink(t, realTarget, target)
			mustSymlink(t, target, f.destination)
			return target
		}},
		{name: "nested target symlink", setup: func(t *testing.T, f fixture) string {
			realDir := filepath.Join(f.base, "real-main")
			linkedDir := filepath.Join(f.base, "linked-main")
			mustMkdir(t, realDir, 0o755)
			mustWrite(t, filepath.Join(realDir, "opencode.json"), "target", 0o600)
			mustSymlink(t, realDir, linkedDir)
			target := filepath.Join(linkedDir, "opencode.json")
			mustSymlink(t, target, f.destination)
			return target
		}},
		{name: "directory target", setup: func(t *testing.T, f fixture) string {
			target := filepath.Join(f.base, "target-dir")
			mustMkdir(t, target, 0o755)
			mustSymlink(t, target, f.destination)
			return target
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t, false)
			target := test.setup(t, f)
			if _, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(5, 0), exactMigrationOptions(target)); err == nil {
				t.Fatal("PlanWithOptions() accepted an untrusted symlink target")
			}
		})
	}
}

func TestExactSymlinkMigrationRejectsRacedSymlink(t *testing.T) {
	f := newFixture(t, false)
	target := filepath.Join(f.base, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)
	tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(6, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.destination); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, target, f.destination)
	if _, err := tx.Apply(); !errors.Is(err, ErrConcurrentChange) {
		t.Fatalf("Apply() error = %v, want ErrConcurrentChange", err)
	}
	linkText, err := os.Readlink(f.destination)
	if err != nil || linkText != target {
		t.Fatalf("raced symlink was overwritten: %q, %v", linkText, err)
	}
	assertContent(t, target, "main repository config")
}

func TestMigrationMarkerRacesFailClosed(t *testing.T) {
	t.Run("marker path becomes symlink", func(t *testing.T) {
		f := newFixture(t, false)
		target := filepath.Join(f.base, "opencode.json")
		outside := filepath.Join(f.base, "outside-marker")
		mustWrite(t, target, "main repository config", 0o600)
		mustWrite(t, outside, "outside", 0o600)
		mustSymlink(t, target, f.destination)
		tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(10, 1), exactMigrationOptions(target))
		if err != nil {
			t.Fatal(err)
		}
		mustSymlink(t, outside, tx.preview.MigrationMarker)
		if _, err := tx.Apply(); err == nil {
			t.Fatal("Apply() accepted a symlink migration marker")
		}
		if linkText, err := os.Readlink(f.destination); err != nil || linkText != target {
			t.Fatalf("destination changed after marker symlink race: %q, %v", linkText, err)
		}
		assertContent(t, outside, "outside")
	})

	t.Run("marker appears before publish", func(t *testing.T) {
		f := newFixture(t, false)
		target := filepath.Join(f.base, "opencode.json")
		mustWrite(t, target, "main repository config", 0o600)
		mustSymlink(t, target, f.destination)
		tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(11, 0), exactMigrationOptions(target))
		if err != nil {
			t.Fatal(err)
		}
		tx.afterRename = func(string) error {
			return os.WriteFile(tx.preview.MigrationMarker, tx.markerData, 0o600)
		}
		if _, err := tx.Apply(); !errors.Is(err, ErrMigrationConsumed) {
			t.Fatalf("Apply() error = %v, want ErrMigrationConsumed", err)
		}
		if linkText, err := os.Readlink(f.destination); err != nil || linkText != target {
			t.Fatalf("automatic rollback symlink = %q, %v", linkText, err)
		}
		assertContent(t, tx.preview.MigrationMarker, string(tx.markerData))
	})

	t.Run("marker identity changes before rollback", func(t *testing.T) {
		f := newFixture(t, false)
		target := filepath.Join(f.base, "opencode.json")
		mustWrite(t, target, "main repository config", 0o600)
		mustSymlink(t, target, f.destination)
		tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(12, 0), exactMigrationOptions(target))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Apply(); err != nil {
			t.Fatal(err)
		}
		replacement := tx.preview.MigrationMarker + ".replacement"
		mustWrite(t, replacement, string(tx.markerData), 0o600)
		if err := os.Rename(replacement, tx.preview.MigrationMarker); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Rollback(); !errors.Is(err, ErrConcurrentChange) {
			t.Fatalf("Rollback() error = %v, want ErrConcurrentChange", err)
		}
		assertContent(t, f.destination, "new content")
	})

	t.Run("marker gains hard link before rollback", func(t *testing.T) {
		f := newFixture(t, false)
		target := filepath.Join(f.base, "opencode.json")
		mustWrite(t, target, "main repository config", 0o600)
		mustSymlink(t, target, f.destination)
		tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(13, 0), exactMigrationOptions(target))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Apply(); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(tx.preview.MigrationMarker, tx.preview.MigrationMarker+".link"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Rollback(); err == nil {
			t.Fatal("Rollback() accepted a multiply-linked migration marker")
		}
		assertContent(t, f.destination, "new content")
	})
}

func TestRegularDestinationReapplyIgnoresConsumedMarker(t *testing.T) {
	f := newFixture(t, false)
	target := filepath.Join(f.base, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)
	first, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(14, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Apply(); err != nil {
		t.Fatal(err)
	}
	markerBefore, err := os.Lstat(first.preview.MigrationMarker)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(15, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatalf("regular destination reapply plan: %v", err)
	}
	if _, err := second.Apply(); err != nil {
		t.Fatalf("regular destination reapply: %v", err)
	}
	markerAfter, err := os.Lstat(first.preview.MigrationMarker)
	if err != nil || !os.SameFile(markerBefore, markerAfter) {
		t.Fatalf("regular reapply changed consumed marker: %v, %v", markerAfter, err)
	}
}

func TestExactSymlinkRollbackRejectsConcurrentRegularReplacement(t *testing.T) {
	f := newFixture(t, false)
	target := filepath.Join(f.base, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)
	tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(8, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Apply(); err != nil {
		t.Fatal(err)
	}
	markerBefore, err := os.Lstat(tx.preview.MigrationMarker)
	if err != nil {
		t.Fatal(err)
	}
	replacement := f.destination + ".external"
	mustWrite(t, replacement, "new content", 0o600)
	if err := os.Rename(replacement, f.destination); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Rollback(); !errors.Is(err, ErrConcurrentChange) {
		t.Fatalf("Rollback() error = %v, want ErrConcurrentChange", err)
	}
	info, err := os.Lstat(f.destination)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("concurrent destination changed: %v, %v", info, err)
	}
	assertContent(t, f.destination, "new content")
	assertContent(t, target, "main repository config")
	markerAfter, err := os.Lstat(tx.preview.MigrationMarker)
	if err != nil || !os.SameFile(markerBefore, markerAfter) {
		t.Fatalf("concurrent destination mutation consumed rollback marker: %v, %v", markerAfter, err)
	}
}

func TestRecoveryCompletesCrashAfterMarkerQuarantine(t *testing.T) {
	f := newFixture(t, false)
	target := filepath.Join(f.base, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)
	tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(18, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Apply(); err != nil {
		t.Fatal(err)
	}
	tx.afterMarkerRemoval = func(string) error { return errors.New("injected crash before destination rename") }
	if _, err := tx.Rollback(); err == nil || !strings.Contains(err.Error(), "injected crash") {
		t.Fatalf("Rollback() error = %v, want injected crash", err)
	}
	assertContent(t, f.destination, "new content")
	if _, err := os.Lstat(tx.preview.MigrationMarker); !os.IsNotExist(err) {
		t.Fatalf("consumed marker was not quarantined: %v", err)
	}
	inspection, err := InspectJournal(tx.preview.JournalPath, f.backup)
	if err != nil {
		t.Fatal(err)
	}
	backupDir, err := openBackupRoot(f.backup)
	if err != nil {
		t.Fatal(err)
	}
	j, _, err := loadJournal(backupDir, filepath.Base(tx.preview.JournalPath))
	backupDir.Close()
	if err != nil || j.State != "marker_quarantined" {
		t.Fatalf("crash journal = %+v, %v", j, err)
	}
	if linkText, err := os.Readlink(filepath.Join(filepath.Dir(f.destination), j.RestoreTempName)); err != nil || linkText != target {
		t.Fatalf("staged rollback symlink = %q, %v; want %q", linkText, err, target)
	}
	if inspection.Version() != 2 {
		t.Fatalf("journal version = %d", inspection.Version())
	}
	recovered, err := Recover(tx.preview.JournalPath, f.root, f.backup)
	if err != nil || !recovered.RolledBack || !recovered.Recovered {
		t.Fatalf("Recover(marker_quarantined) = %+v, %v", recovered, err)
	}
	if linkText, err := os.Readlink(f.destination); err != nil || linkText != target {
		t.Fatalf("recovered destination symlink = %q, %v", linkText, err)
	}
	if _, err := os.Lstat(filepath.Join(f.backup, j.MarkerRemoveName)); !os.IsNotExist(err) {
		t.Fatalf("recovery retained quarantined marker: %v", err)
	}
}

func TestSymlinkJournalRejectsEscapingRestoreTempName(t *testing.T) {
	f := newFixture(t, false)
	target := filepath.Join(f.base, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)
	tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(9, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatal(err)
	}
	j := tx.newJournal("rollback_ready")
	j.RestoreTempName = "../outside"
	if err := validateJournal(j, tx.preview.JournalPath, f.root, f.backup); err == nil {
		t.Fatal("validateJournal() accepted an escaping restore temp name")
	}
}

func TestExactSymlinkRecoveryResumesRollbackReady(t *testing.T) {
	f := newFixture(t, false)
	target := filepath.Join(f.base, "opencode.json")
	mustWrite(t, target, "main repository config", 0o600)
	mustSymlink(t, target, f.destination)
	tx, err := PlanWithOptions(f.source, f.destination, f.root, f.backup, time.Unix(10, 0), exactMigrationOptions(target))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Apply(); err != nil {
		t.Fatal(err)
	}

	backupDir, err := openBackupRoot(f.backup)
	if err != nil {
		t.Fatal(err)
	}
	j, journalHash, err := loadJournal(backupDir, filepath.Base(tx.preview.JournalPath))
	if err != nil {
		backupDir.Close()
		t.Fatal(err)
	}
	root, destParent, _, err := openDestination(f.root, tx.destRel)
	if err != nil {
		backupDir.Close()
		t.Fatal(err)
	}
	restoreName, restoreID, err := createSymlinkAt(destParent, ".hostprojection-restore-link-", target)
	if err != nil {
		root.Close()
		destParent.Close()
		backupDir.Close()
		t.Fatal(err)
	}
	durability := durabilityTracker{level: j.Durability, detail: j.DurabilityDetail}
	if err := durability.syncDir(destParent, f.destination); err != nil {
		t.Fatal(err)
	}
	j.State, j.RestoreTempName, j.RestoreIdentity = "rollback_ready", restoreName, restoreID
	if err := storeJournalStandalone(backupDir, &j, &journalHash, &durability); err != nil {
		t.Fatal(err)
	}
	root.Close()
	destParent.Close()
	backupDir.Close()

	recovered, err := Recover(tx.preview.JournalPath, f.root, f.backup)
	if err != nil || !recovered.RolledBack || !recovered.Recovered {
		t.Fatalf("Recover(rollback_ready) = %+v, %v", recovered, err)
	}
	linkText, err := os.Readlink(f.destination)
	if err != nil || linkText != target {
		t.Fatalf("recovered symlink = %q, %v; want %q", linkText, err, target)
	}
	again, err := Recover(tx.preview.JournalPath, f.root, f.backup)
	if err != nil || !again.AlreadyComplete {
		t.Fatalf("idempotent Recover() = %+v, %v", again, err)
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

func TestPreexistingJournalIsRetainedForExplicitRecovery(t *testing.T) {
	f := newFixture(t, true)
	first := f.plan(t)
	if _, err := first.Apply(); err != nil {
		t.Fatalf("first Apply(): %v", err)
	}
	second, err := Plan(f.source, f.destination, f.root, f.backup, time.Unix(1, 2))
	if err != nil {
		t.Fatalf("second Plan(): %v", err)
	}
	if _, err := second.Apply(); err == nil || !strings.Contains(err.Error(), "journal already exists") {
		t.Fatalf("second Apply() error = %v, want retained-journal rejection", err)
	}
	if _, err := os.Lstat(first.Preview().JournalPath); err != nil {
		t.Fatalf("failed apply removed recoverable journal: %v", err)
	}
	recovered, err := Recover(first.Preview().JournalPath, f.root, f.backup)
	if err != nil || !recovered.RolledBack || !recovered.Recovered {
		t.Fatalf("Recover() = %+v, %v", recovered, err)
	}
	assertContent(t, f.destination, "old content")
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

func exactMigrationOptions(target string) PlanOptions {
	return PlanOptions{
		ProfileID: "opencode-bootstrap", MigrationID: "opencode-bootstrap-symlink-v1",
		ExactSymlinkMigration: &ExactSymlinkMigration{ExpectedTarget: target},
	}
}

func rewriteJournalVersionOne(t *testing.T, path string, additions map[string]any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	fields["version"] = float64(1)
	for _, field := range []string{
		"destination_kind", "profile_id", "migration_id", "original_link_text", "expected_link_target",
		"marker_name", "marker_sha256", "marker_consumed", "marker_identity", "marker_temp_name", "marker_remove_name", "restore_temp_name",
	} {
		delete(fields, field)
	}
	for key, value := range additions {
		fields[key] = value
	}
	rewritten, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, rewritten, 0o600); err != nil {
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

func withMode(stat syscall.Stat_t, mode uint32) syscall.Stat_t {
	stat.Mode = stat.Mode&^0o777 | mode
	return stat
}

func withType(stat syscall.Stat_t, fileType uint32) syscall.Stat_t {
	stat.Mode = stat.Mode&^syscall.S_IFMT | fileType
	return stat
}

func withLinks(stat syscall.Stat_t, links uint64) syscall.Stat_t {
	stat.Nlink = links
	return stat
}

func withOwner(stat syscall.Stat_t, owner uint32) syscall.Stat_t {
	stat.Uid = owner
	return stat
}

func withSpecial(stat syscall.Stat_t, special uint32) syscall.Stat_t {
	stat.Mode |= special
	return stat
}
