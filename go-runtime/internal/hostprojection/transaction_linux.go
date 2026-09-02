//go:build linux

package hostprojection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const journalVersion = 2

type journal struct {
	Version            int             `json:"version"`
	State              string          `json:"state"`
	Source             string          `json:"source"`
	Destination        string          `json:"destination"`
	AllowedRoot        string          `json:"allowed_root"`
	BackupRoot         string          `json:"backup_root"`
	BackupPath         string          `json:"backup_path"`
	JournalPath        string          `json:"journal_path"`
	LockPath           string          `json:"lock_path"`
	PlannedAt          time.Time       `json:"planned_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	SourceSHA256       string          `json:"source_sha256"`
	OriginalSHA256     string          `json:"original_sha256,omitempty"`
	DestinationKind    DestinationKind `json:"destination_kind"`
	OriginalLinkText   string          `json:"original_link_text,omitempty"`
	ExpectedLinkTarget string          `json:"expected_link_target,omitempty"`
	ProfileID          string          `json:"profile_id,omitempty"`
	MigrationID        string          `json:"migration_id,omitempty"`
	MarkerName         string          `json:"marker_name,omitempty"`
	MarkerSHA256       string          `json:"marker_sha256,omitempty"`
	MarkerConsumed     bool            `json:"marker_consumed,omitempty"`
	BackupSHA256       string          `json:"backup_sha256,omitempty"`
	SourceMode         uint32          `json:"source_mode"`
	OriginalMode       uint32          `json:"original_mode,omitempty"`
	DestinationExisted bool            `json:"destination_existed"`
	OriginalIdentity   fileIdentity    `json:"original_identity"`
	PendingIdentity    fileIdentity    `json:"pending_identity"`
	RestoreIdentity    fileIdentity    `json:"restore_identity"`
	RestoredIdentity   fileIdentity    `json:"restored_identity"`
	MarkerIdentity     fileIdentity    `json:"marker_identity"`
	TempName           string          `json:"temp_name,omitempty"`
	RestoreTempName    string          `json:"restore_temp_name,omitempty"`
	MarkerTempName     string          `json:"marker_temp_name,omitempty"`
	MarkerRemoveName   string          `json:"marker_remove_name,omitempty"`
	Durability         DurabilityLevel `json:"durability"`
	DurabilityDetail   string          `json:"durability_detail,omitempty"`
}

// Transaction is a Linux-only, locked and journaled host projection.
type Transaction struct {
	mu                  sync.Mutex
	preview             Preview
	source              snapshot
	original            snapshot
	destRel             string
	backupParentID      fileIdentity
	backupMissing       []string
	journalHash         string
	afterRename         func(string) error
	beforeMarkerRemoval func(string) error
	afterMarkerRemoval  func(string) error
	markerName          string
	markerData          []byte
}

// Plan validates all path components with O_NOFOLLOW and performs no writes.
func Plan(source, destination, allowedRoot, backupRoot string, at time.Time) (*Transaction, error) {
	return PlanValidated(source, destination, allowedRoot, backupRoot, at, nil)
}

// PlanWithOptions plans with explicit destination migration options and no
// source-content validator.
func PlanWithOptions(source, destination, allowedRoot, backupRoot string, at time.Time, options PlanOptions) (*Transaction, error) {
	return PlanValidatedWithOptions(source, destination, allowedRoot, backupRoot, at, nil, options)
}

// PlanValidated plans a transaction and validates the exact source snapshot
// that Apply later revalidates before mutation.
func PlanValidated(source, destination, allowedRoot, backupRoot string, at time.Time, validate SourceValidator) (*Transaction, error) {
	return PlanValidatedWithOptions(source, destination, allowedRoot, backupRoot, at, validate, PlanOptions{})
}

// PlanValidatedWithOptions plans a validated transaction with explicit,
// fail-closed destination migration options.
func PlanValidatedWithOptions(source, destination, allowedRoot, backupRoot string, at time.Time, validate SourceValidator, options PlanOptions) (*Transaction, error) {
	var err error
	if source, err = absolute(source); err != nil {
		return nil, fmt.Errorf("resolve source: %w", err)
	}
	if destination, err = absolute(destination); err != nil {
		return nil, fmt.Errorf("resolve destination: %w", err)
	}
	if allowedRoot, err = absolute(allowedRoot); err != nil {
		return nil, fmt.Errorf("resolve allowed root: %w", err)
	}
	if backupRoot, err = absolute(backupRoot); err != nil {
		return nil, fmt.Errorf("resolve backup root: %w", err)
	}
	expectedLinkTarget, markerName, markerData, err := validatePlanOptions(options, destination)
	if err != nil {
		return nil, err
	}
	destRel, err := filepath.Rel(allowedRoot, destination)
	if err != nil {
		return nil, fmt.Errorf("relativize destination: %w", err)
	}
	if _, err := safeRelative(destRel); err != nil {
		return nil, fmt.Errorf("destination outside exact allowed root: %w", err)
	}
	if destRel == "." {
		return nil, errors.New("destination must be a file below allowed root")
	}

	sourceParent, sourceName, err := openParentAbsolute(source)
	if err != nil {
		return nil, fmt.Errorf("open source parent: %w", err)
	}
	sourceSnapshot, err := readRegularAt(sourceParent, sourceName)
	sourceParent.Close()
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	if validate != nil {
		if err := validate(append([]byte(nil), sourceSnapshot.data...)); err != nil {
			return nil, fmt.Errorf("validate source content: %w", err)
		}
	}
	root, err := openDirAbsolute(allowedRoot)
	if err != nil {
		return nil, fmt.Errorf("open allowed root: %w", err)
	}
	defer root.Close()
	destParent, destName, err := openParentRelative(root, destRel)
	if err != nil {
		return nil, fmt.Errorf("open destination parent: %w", err)
	}
	defer destParent.Close()
	original, existed, err := readDestinationOptionalAt(destParent, destName, expectedLinkTarget)
	if err != nil {
		return nil, fmt.Errorf("inspect destination: %w", err)
	}
	if existed && sourceSnapshot.id == original.id {
		return nil, errors.New("source and destination are the same file identity")
	}
	backupDir, backupMissing, err := openDirPrefix(backupRoot)
	if err != nil {
		return nil, fmt.Errorf("validate backup root parent: %w", err)
	}
	defer backupDir.Close()
	backupParentID, err := descriptorIdentity(backupDir)
	if err != nil {
		return nil, fmt.Errorf("identify backup root parent: %w", err)
	}
	if len(backupMissing) == 0 {
		if err := validateBackupRootDescriptor(backupDir); err != nil {
			return nil, err
		}
		if original.kind == DestinationSymlink {
			if err := rejectConsumedMarker(backupDir, markerName, markerData); err != nil {
				return nil, err
			}
		}
	}

	durability := newDurability()
	destinationFilesystem, err := descriptorFilesystemType(destParent)
	if err != nil {
		return nil, fmt.Errorf("identify destination filesystem: %w", err)
	}
	durability.noteDestinationFilesystem(destinationFilesystem)
	if err := durability.syncDir(destParent, destination); err != nil {
		return nil, fmt.Errorf("probe destination durability: %w", err)
	}
	if err := durability.syncDir(backupDir, backupRoot); err != nil {
		return nil, fmt.Errorf("probe backup durability: %w", err)
	}
	stamp := at.UTC()
	pathID := digest([]byte(destination))[:16]
	base := filepath.Base(destination) + "." + stamp.Format("20060102T150405.000000000Z") + "." + pathID
	preview := Preview{
		Source: source, Destination: destination, AllowedRoot: allowedRoot, BackupRoot: backupRoot,
		BackupPath: filepath.Join(backupRoot, base+".bak"), JournalPath: filepath.Join(backupRoot, base+".journal.json"),
		LockPath: filepath.Join(backupRoot, ".hostprojection-"+pathID+".lock"), PlannedAt: stamp,
		SourceSHA256: sourceSnapshot.hash, DestinationKind: original.kind, OriginalLinkText: original.linkText,
		ExpectedLinkTarget: expectedLinkTarget, DestinationExisted: existed, PlatformSupported: true,
		ProfileID: options.ProfileID, MigrationID: options.MigrationID,
		Durability: durability.level, DurabilityDetail: durability.detail,
	}
	if markerName != "" {
		preview.MigrationMarker = filepath.Join(backupRoot, markerName)
	}
	if existed && original.kind == DestinationRegular {
		preview.OriginalSHA256 = original.hash
	}
	return &Transaction{
		preview: preview, source: sourceSnapshot, original: original, destRel: destRel,
		backupParentID: backupParentID, backupMissing: append([]string(nil), backupMissing...),
		markerName: markerName, markerData: append([]byte(nil), markerData...),
	}, nil
}

func validatePlanOptions(options PlanOptions, destination string) (string, string, []byte, error) {
	if options.ProfileID != "" && !validAuthorityID(options.ProfileID) {
		return "", "", nil, errors.New("profile ID must contain only lowercase ASCII letters, digits, and hyphens")
	}
	if options.MigrationID != "" && !validAuthorityID(options.MigrationID) {
		return "", "", nil, errors.New("migration ID must contain only lowercase ASCII letters, digits, and hyphens")
	}
	if options.ExactSymlinkMigration == nil {
		if options.MigrationID != "" {
			return "", "", nil, errors.New("migration ID requires exact symlink migration")
		}
		return "", "", nil, nil
	}
	if options.ProfileID == "" || options.MigrationID == "" {
		return "", "", nil, errors.New("exact symlink migration requires profile ID and migration ID")
	}
	target := options.ExactSymlinkMigration.ExpectedTarget
	if target == "" || !filepath.IsAbs(target) || filepath.Clean(target) != target {
		return "", "", nil, errors.New("exact symlink migration target must be absolute and traversal-free")
	}
	if target == destination {
		return "", "", nil, errors.New("exact symlink migration target must differ from destination")
	}
	markerName := migrationMarkerName(options.MigrationID)
	markerData := migrationMarkerData(options.ProfileID, options.MigrationID, destination)
	return target, markerName, markerData, nil
}

func validAuthorityID(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || (character == '-' && index > 0) {
			continue
		}
		return false
	}
	return !strings.HasSuffix(value, "-")
}

func migrationMarkerName(migrationID string) string {
	return ".hostprojection-migration-" + digest([]byte(migrationID))[:16] + ".consumed"
}

// Preview returns immutable dry-run metadata.
func (t *Transaction) Preview() Preview {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.preview
}

// Apply acquires a process lock, journals every durable state transition, and
// atomically replaces the destination using renameat on Linux.
func (t *Transaction) Apply() (Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	durability := durabilityTracker{level: t.preview.Durability, detail: t.preview.DurabilityDetail}
	backupDir, err := ensureBackupRoot(t.preview.BackupRoot, t.backupParentID, t.backupMissing, &durability)
	if err != nil {
		return t.result("apply", ""), err
	}
	defer backupDir.Close()
	lock, err := openLockAt(backupDir, filepath.Base(t.preview.LockPath))
	if err != nil {
		return t.result("apply", ""), fmt.Errorf("apply: %w", err)
	}
	defer closeLock(lock)
	if _, exists, readErr := readOptionalAt(backupDir, filepath.Base(t.preview.JournalPath)); readErr != nil || exists {
		if readErr != nil {
			return t.result("apply", ""), fmt.Errorf("inspect journal: %w", readErr)
		}
		return t.result("apply", ""), errors.New("apply: journal already exists; recover or rollback first")
	}
	if t.original.kind == DestinationSymlink {
		if err := rejectConsumedMarker(backupDir, t.markerName, t.markerData); err != nil {
			return t.result("apply", ""), err
		}
	}

	root, destParent, destName, err := openDestination(t.preview.AllowedRoot, t.destRel)
	if err != nil {
		return t.result("apply", ""), err
	}
	defer root.Close()
	defer destParent.Close()
	if err := verifyDestinationOptional(destParent, destName, t.original, t.preview.DestinationExisted, t.preview.ExpectedLinkTarget); err != nil {
		return t.result("apply", ""), fmt.Errorf("apply precondition: %w", err)
	}
	if err := t.verifySource(); err != nil {
		return t.result("apply", ""), err
	}

	j := t.newJournal("prepared")
	if err := t.storeJournal(backupDir, &j, true, &durability); err != nil {
		return t.result("apply", j.State), err
	}
	if j.DestinationKind == DestinationRegular {
		if err := writeArtifact(backupDir, filepath.Base(j.BackupPath), t.original.data, 0o600, &durability); err != nil {
			return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("create backup: %w", err)
		}
		backup, err := readRegularAt(backupDir, filepath.Base(j.BackupPath))
		if err != nil {
			return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("verify backup: %w", err)
		}
		if err := verifyPrivateArtifact(backup, "backup"); err != nil || backup.hash != j.OriginalSHA256 || !bytes.Equal(backup.data, t.original.data) {
			if err != nil {
				return t.resultWithDurability("apply", j.State, durability), err
			}
			return t.resultWithDurability("apply", j.State, durability), errors.New("verify backup: SHA-256 or content mismatch")
		}
		j.BackupSHA256 = backup.hash
	}
	j.State = "backed_up"
	if err := t.storeJournal(backupDir, &j, false, &durability); err != nil {
		return t.resultWithDurability("apply", j.State, durability), err
	}

	mode := t.source.mode
	if j.DestinationKind == DestinationRegular {
		mode = t.original.mode
	}
	temp, tempName, pendingID, err := createDestinationFileAt(destParent, ".hostprojection-", t.source.data, mode, &durability)
	if err != nil {
		return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("stage destination: %w", err)
	}
	temp.Close()
	defer func() { _ = syscall.Unlinkat(int(destParent.Fd()), tempName) }()
	j.State, j.TempName, j.PendingIdentity = "ready", tempName, pendingID
	if err := t.storeJournal(backupDir, &j, false, &durability); err != nil {
		return t.resultWithDurability("apply", j.State, durability), err
	}
	if err := verifyDestinationOptional(destParent, destName, t.original, j.DestinationExisted, j.ExpectedLinkTarget); err != nil {
		return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("destination changed before rename: %w", err)
	}
	if j.DestinationKind == DestinationSymlink {
		if err := exchangeStagedDestination(destParent, tempName, destName, t.original, j.ExpectedLinkTarget); err != nil {
			return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("atomic replace destination symlink: %w", err)
		}
	} else if err := syscall.Renameat(int(destParent.Fd()), tempName, int(destParent.Fd()), destName); err != nil {
		return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("atomic rename destination: %w", err)
	}
	if err := durability.syncDir(destParent, t.preview.Destination); err != nil {
		return t.rollbackAfterApplyFailure(backupDir, &j, durability, fmt.Errorf("commit destination: %w", err))
	}
	if t.afterRename != nil {
		if err := t.afterRename(t.preview.Destination); err != nil {
			return t.rollbackAfterApplyFailure(backupDir, &j, durability, fmt.Errorf("post-rename hook: %w", err))
		}
	}
	current, err := readRegularAt(destParent, destName)
	if err != nil {
		return t.rollbackAfterApplyFailure(backupDir, &j, durability, fmt.Errorf("verify applied destination: %w", err))
	}
	if current.id != pendingID || current.hash != t.source.hash || !bytes.Equal(current.data, t.source.data) {
		return t.rollbackAfterApplyFailure(backupDir, &j, durability, errors.New("verify applied destination: identity, SHA-256, or content mismatch"))
	}
	j.State = "applied"
	if err := t.storeJournal(backupDir, &j, false, &durability); err != nil {
		return t.rollbackAfterApplyFailure(backupDir, &j, durability, err)
	}
	if j.DestinationKind == DestinationSymlink {
		if err := t.consumeMigrationMarker(backupDir, &j, &durability); err != nil {
			return t.rollbackAfterApplyFailure(backupDir, &j, durability, err)
		}
	}
	j.State = "verified"
	if err := t.storeJournal(backupDir, &j, false, &durability); err != nil {
		return t.rollbackAfterApplyFailure(backupDir, &j, durability, err)
	}
	result := t.resultWithDurability("apply", j.State, durability)
	result.Applied = true
	return result, nil
}

// Rollback safely restores this transaction's journaled original state.
func (t *Transaction) Rollback() (Result, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	inspection, err := InspectJournal(t.preview.JournalPath, t.preview.BackupRoot)
	if err != nil {
		return t.result("rollback", ""), err
	}
	result, err := recoverInspected(inspection, JournalAuthority{
		Source: t.preview.Source, Destination: t.preview.Destination,
		AllowedRoot: t.preview.AllowedRoot, BackupRoot: t.preview.BackupRoot,
		ExpectedDestinationTarget: t.preview.ExpectedLinkTarget,
		ProfileID:                 t.preview.ProfileID, MigrationID: t.preview.MigrationID,
	}, t.afterMarkerRemoval)
	result.Operation = "rollback"
	result.Recovered = false
	return result, err
}

// Recover performs idempotent, lock-protected recovery from a durable journal.
func Recover(journalPath, allowedRoot, backupRoot string) (Result, error) {
	allowedRoot, err := absolute(allowedRoot)
	if err != nil {
		return Result{Operation: "recover"}, fmt.Errorf("resolve allowed root: %w", err)
	}
	backupRoot, err = absolute(backupRoot)
	if err != nil {
		return Result{Operation: "recover"}, fmt.Errorf("resolve backup root: %w", err)
	}
	inspection, err := InspectJournal(journalPath, backupRoot)
	if err != nil {
		return Result{Operation: "recover"}, err
	}
	expected := inspection.Authority()
	if expected.AllowedRoot != allowedRoot || expected.BackupRoot != backupRoot {
		return Result{Operation: "recover"}, errors.New("journal roots do not match recovery authority")
	}
	return RecoverInspected(inspection, expected)
}

func rollbackJournal(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker, afterMarkerRemoval func(string) error) (Result, error) {
	if j.DestinationKind == "" {
		if j.DestinationExisted {
			j.DestinationKind = DestinationRegular
		} else {
			j.DestinationKind = DestinationAbsent
		}
	}
	destRel, err := filepath.Rel(j.AllowedRoot, j.Destination)
	if err != nil {
		return resultFromJournal(*j, "rollback"), fmt.Errorf("relativize journal destination: %w", err)
	}
	root, destParent, destName, err := openDestination(j.AllowedRoot, destRel)
	if err != nil {
		return resultFromJournal(*j, "rollback"), err
	}
	defer root.Close()
	defer destParent.Close()
	current, exists, err := readDestinationOptionalAt(destParent, destName, j.ExpectedLinkTarget)
	if err != nil {
		return resultFromJournal(*j, "rollback"), fmt.Errorf("read rollback destination: %w", err)
	}

	if j.State == "rolled_back" {
		if err := verifyOwnedMarkerGone(backupDir, *j); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		if err := verifyRolledBack(current, exists, *j); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		result := resultFromJournal(*j, "rollback")
		result.RolledBack, result.AlreadyComplete = true, true
		return result, nil
	}
	if j.DestinationKind == DestinationSymlink && (j.State == "rollback_ready" || j.State == "marker_remove_ready" || j.State == "marker_quarantined" || j.State == "marker_finalize_ready" || j.State == "marker_removed") {
		if err := rollbackSymlinkDestination(backupDir, destParent, destName, current, exists, j, journalHash, durability, afterMarkerRemoval); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		j.State = "rolled_back"
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		result := resultFromJournal(*j, "rollback")
		result.RolledBack = true
		return result, nil
	}
	if j.DestinationKind != DestinationSymlink && j.State == "rollback_ready" && restoredDestination(current, exists, *j) {
		if err := cleanupRestoreTemp(destParent, *j, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		j.State, j.RestoredIdentity = "rolled_back", current.id
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		result := resultFromJournal(*j, "rollback")
		result.RolledBack = true
		return result, nil
	}
	if j.State == "delete_ready" && !exists {
		j.State = "rolled_back"
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		result := resultFromJournal(*j, "rollback")
		result.RolledBack = true
		return result, nil
	}

	applied := exists && current.kind == DestinationRegular && !j.PendingIdentity.zero() && current.id == j.PendingIdentity && current.hash == j.SourceSHA256
	original := originalDestination(current, exists, *j)
	untouchedAbsent := !j.DestinationExisted && !exists
	if !applied {
		if !original && !untouchedAbsent {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("%w: rollback destination identity/content mismatch", ErrConcurrentChange)
		}
		if err := verifyBackupIfPresent(backupDir, *j); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		if original {
			j.RestoredIdentity = current.id
		}
		j.State = "rolled_back"
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		result := resultFromJournal(*j, "rollback")
		result.RolledBack = true
		return result, nil
	}

	if j.DestinationKind == DestinationRegular {
		backup, err := readRegularAt(backupDir, filepath.Base(j.BackupPath))
		if err != nil {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("verify rollback backup: %w", err)
		}
		if err := verifyPrivateArtifact(backup, "rollback backup"); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		if backup.hash != j.OriginalSHA256 {
			return resultFromJournal(*j, "rollback"), errors.New("verify rollback backup: SHA-256 mismatch")
		}
		temp, name, restoreID, err := createDestinationFileAt(destParent, ".hostprojection-restore-", backup.data, j.OriginalMode, durability)
		if err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		temp.Close()
		defer func() { _ = syscall.Unlinkat(int(destParent.Fd()), name) }()
		j.State, j.RestoreIdentity = "rollback_ready", restoreID
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		if _, err := verifyAt(destParent, destName, current); err != nil {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("destination changed before restore: %w", err)
		}
		if err := syscall.Renameat(int(destParent.Fd()), name, int(destParent.Fd()), destName); err != nil {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("restore destination: %w", err)
		}
		if err := durability.syncDir(destParent, j.Destination); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		restored, err := readRegularAt(destParent, destName)
		if err != nil {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("verify restored destination: %w", err)
		}
		if restored.id != restoreID || restored.hash != j.OriginalSHA256 {
			return resultFromJournal(*j, "rollback"), errors.New("verify restored destination: identity or SHA-256 mismatch")
		}
		j.RestoredIdentity = restored.id
	} else if j.DestinationKind == DestinationSymlink {
		if err := rollbackSymlinkDestination(backupDir, destParent, destName, current, exists, j, journalHash, durability, afterMarkerRemoval); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
	} else {
		j.State = "delete_ready"
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		if _, err := verifyAt(destParent, destName, current); err != nil {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("destination changed before delete: %w", err)
		}
		if err := syscall.Unlinkat(int(destParent.Fd()), destName); err != nil {
			return resultFromJournal(*j, "rollback"), fmt.Errorf("delete projected destination: %w", err)
		}
		if err := durability.syncDir(destParent, j.Destination); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
	}
	j.State = "rolled_back"
	if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
		return resultFromJournal(*j, "rollback"), err
	}
	result := resultFromJournal(*j, "rollback")
	result.RolledBack = true
	return result, nil
}

func originalDestination(current snapshot, exists bool, j journal) bool {
	if !j.DestinationExisted || !exists || current.id != j.OriginalIdentity {
		return false
	}
	if j.DestinationKind == DestinationSymlink {
		return current.kind == DestinationSymlink && current.linkText == j.OriginalLinkText
	}
	return current.kind == DestinationRegular && current.hash == j.OriginalSHA256
}

func restoredDestination(current snapshot, exists bool, j journal) bool {
	if !exists || current.id != j.RestoreIdentity {
		return false
	}
	if j.DestinationKind == DestinationSymlink {
		return current.kind == DestinationSymlink && current.linkText == j.OriginalLinkText
	}
	return current.kind == DestinationRegular && current.hash == j.OriginalSHA256
}

func rollbackSymlinkDestination(
	backupDir, destParent *os.File,
	destName string,
	current snapshot,
	exists bool,
	j *journal,
	journalHash *string,
	durability *durabilityTracker,
	afterMarkerRemoval func(string) error,
) error {
	if j.OriginalLinkText == "" || j.OriginalLinkText != j.ExpectedLinkTarget {
		return errors.New("rollback symlink authority is incomplete")
	}
	applied := exists && current.kind == DestinationRegular && current.id == j.PendingIdentity && current.hash == j.SourceSHA256
	restored := restoredDestination(current, exists, *j)
	if !applied && !restored {
		if err := restoreQuarantinedMarker(backupDir, j, journalHash, durability); err != nil {
			return err
		}
		return fmt.Errorf("%w: rollback destination changed before marker removal", ErrConcurrentChange)
	}
	if restored && j.State == "marker_finalize_ready" {
		return finalizeMarkerRemoval(backupDir, j, journalHash, durability)
	}
	if restored && j.State == "marker_removed" {
		return cleanupRestoreTemp(destParent, *j, durability)
	}
	if applied {
		if err := stageOriginalSymlink(backupDir, destParent, j, journalHash, durability); err != nil {
			return err
		}
		if _, err := verifyAt(destParent, destName, current); err != nil {
			return fmt.Errorf("destination changed after symlink staging: %w", err)
		}
	}
	if err := quarantineMarkerAfterStaging(backupDir, j, journalHash, durability); err != nil {
		return err
	}
	if afterMarkerRemoval != nil && !j.MarkerIdentity.zero() {
		if err := afterMarkerRemoval(j.MarkerRemoveName); err != nil {
			return fmt.Errorf("post-marker-removal hook: %w", err)
		}
	}
	if applied {
		if _, err := verifyAt(destParent, destName, current); err != nil {
			if restoreErr := restoreQuarantinedMarker(backupDir, j, journalHash, durability); restoreErr != nil {
				return fmt.Errorf("destination changed after marker quarantine: %w; marker restore failed: %v", err, restoreErr)
			}
			return fmt.Errorf("destination changed after marker quarantine: %w", err)
		}
		if err := exchangeStagedSymlink(destParent, destName, current, j, durability); err != nil {
			if restoreErr := restoreQuarantinedMarker(backupDir, j, journalHash, durability); restoreErr != nil {
				return fmt.Errorf("%w; marker restore failed: %v", err, restoreErr)
			}
			return err
		}
	} else {
		if err := cleanupRestoreTemp(destParent, *j, durability); err != nil {
			return err
		}
	}
	return finalizeMarkerRemoval(backupDir, j, journalHash, durability)
}

func stageOriginalSymlink(backupDir, destParent *os.File, j *journal, journalHash *string, durability *durabilityTracker) error {
	if j.RestoreTempName == "" || j.RestoreIdentity.zero() {
		name, restoreID, err := createSymlinkAt(destParent, ".hostprojection-restore-link-", j.OriginalLinkText)
		if err != nil {
			return err
		}
		if err := durability.syncDir(destParent, j.Destination); err != nil {
			_ = syscall.Unlinkat(int(destParent.Fd()), name)
			return err
		}
		j.State, j.RestoreTempName, j.RestoreIdentity = "rollback_ready", name, restoreID
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			_ = syscall.Unlinkat(int(destParent.Fd()), name)
			_ = durability.syncDir(destParent, j.Destination)
			return err
		}
	}
	staged, err := readExactSymlinkAt(destParent, j.RestoreTempName, j.OriginalLinkText)
	if err != nil || staged.id != j.RestoreIdentity || staged.linkText != j.OriginalLinkText {
		return fmt.Errorf("%w: rollback symlink staging changed", ErrConcurrentChange)
	}
	return nil
}

func exchangeStagedSymlink(destParent *os.File, destName string, current snapshot, j *journal, durability *durabilityTracker) error {
	if err := unix.Renameat2(int(destParent.Fd()), j.RestoreTempName, int(destParent.Fd()), destName, unix.RENAME_EXCHANGE); err != nil {
		return fmt.Errorf("exchange restored symlink: %w", err)
	}
	swapped, verifyErr := readRegularAt(destParent, j.RestoreTempName)
	if verifyErr == nil && (swapped.id != j.PendingIdentity || swapped.hash != j.SourceSHA256) {
		verifyErr = ErrConcurrentChange
	}
	if verifyErr != nil {
		if restoreErr := unix.Renameat2(int(destParent.Fd()), j.RestoreTempName, int(destParent.Fd()), destName, unix.RENAME_EXCHANGE); restoreErr != nil {
			return fmt.Errorf("%w; failed to restore raced rollback destination: %v", verifyErr, restoreErr)
		}
		return fmt.Errorf("%w: rollback destination changed during exchange", ErrConcurrentChange)
	}
	if err := durability.syncDir(destParent, j.Destination); err != nil {
		return err
	}
	restored, err := readExactSymlinkAt(destParent, destName, j.OriginalLinkText)
	if err != nil || restored.id != j.RestoreIdentity || restored.linkText != j.OriginalLinkText {
		return errors.New("verify restored destination symlink: identity or link text mismatch")
	}
	j.RestoredIdentity = restored.id
	return cleanupRestoreTemp(destParent, *j, durability)
}

func cleanupRestoreTemp(destParent *os.File, j journal, durability *durabilityTracker) error {
	if j.RestoreTempName == "" {
		return nil
	}
	temp, exists, err := readOptionalAt(destParent, j.RestoreTempName)
	if err != nil {
		return fmt.Errorf("inspect exchanged rollback file: %w", err)
	}
	if !exists {
		return nil
	}
	if temp.id != j.PendingIdentity || temp.hash != j.SourceSHA256 {
		return fmt.Errorf("%w: exchanged rollback file mismatch", ErrConcurrentChange)
	}
	if err := syscall.Unlinkat(int(destParent.Fd()), j.RestoreTempName); err != nil {
		return fmt.Errorf("remove exchanged rollback file: %w", err)
	}
	return durability.syncDir(destParent, j.Destination)
}

func quarantineMarkerAfterStaging(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker) error {
	if j.MarkerIdentity.zero() {
		return nil
	}
	quarantined, quarantineExists, err := exactMarkerAt(backupDir, j.MarkerRemoveName, *j)
	if err != nil {
		return err
	}
	if quarantineExists {
		if quarantined.id != j.MarkerIdentity {
			return fmt.Errorf("%w: quarantined migration marker identity changed", ErrConcurrentChange)
		}
		j.State = "marker_quarantined"
		return storeJournalStandalone(backupDir, j, journalHash, durability)
	}
	marker, markerExists, err := exactMarkerAt(backupDir, j.MarkerName, *j)
	if err != nil {
		return err
	}
	if markerExists && marker.id == j.MarkerIdentity {
		return beginMarkerQuarantine(backupDir, j, journalHash, durability)
	}
	temp, tempExists, err := exactMarkerAt(backupDir, j.MarkerTempName, *j)
	if err != nil {
		return err
	}
	if tempExists && temp.id == j.MarkerIdentity && !j.MarkerConsumed {
		if err := syscall.Unlinkat(int(backupDir.Fd()), j.MarkerTempName); err != nil {
			return fmt.Errorf("remove unconsumed staged migration marker: %w", err)
		}
		return durability.syncDir(backupDir, j.BackupRoot)
	}
	if j.MarkerConsumed {
		return fmt.Errorf("%w: consumed migration marker changed before rollback", ErrConcurrentChange)
	}
	return nil
}

func beginMarkerQuarantine(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker) error {
	for range 16 {
		name, err := randomEntryName(".hostprojection-marker-remove-")
		if err != nil {
			return err
		}
		if _, exists, err := readOptionalAt(backupDir, name); err != nil {
			return err
		} else if exists {
			continue
		}
		j.State, j.MarkerRemoveName = "marker_remove_ready", name
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return err
		}
		return finishMarkerQuarantine(backupDir, j, journalHash, durability)
	}
	return errors.New("allocate migration marker removal name: collision limit reached")
}

func finishMarkerQuarantine(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker) error {
	marker, markerExists, markerErr := exactMarkerAt(backupDir, j.MarkerName, *j)
	removed, removedExists, removedErr := exactMarkerAt(backupDir, j.MarkerRemoveName, *j)
	if markerErr != nil {
		return markerErr
	}
	if removedErr != nil {
		return removedErr
	}
	if markerExists && removedExists {
		return fmt.Errorf("%w: migration marker exists at both removal paths", ErrConcurrentChange)
	}
	if markerExists {
		if marker.id != j.MarkerIdentity {
			return fmt.Errorf("%w: consumed migration marker identity changed", ErrConcurrentChange)
		}
		if err := unix.Renameat2(int(backupDir.Fd()), j.MarkerName, int(backupDir.Fd()), j.MarkerRemoveName, unix.RENAME_NOREPLACE); err != nil {
			return fmt.Errorf("atomically quarantine consumed migration marker: %w", err)
		}
		if err := durability.syncDir(backupDir, j.BackupRoot); err != nil {
			return err
		}
		removed, removedExists, removedErr = exactMarkerAt(backupDir, j.MarkerRemoveName, *j)
		if removedErr != nil || !removedExists || removed.id != j.MarkerIdentity {
			if restoreErr := unix.Renameat2(int(backupDir.Fd()), j.MarkerRemoveName, int(backupDir.Fd()), j.MarkerName, unix.RENAME_NOREPLACE); restoreErr != nil {
				return fmt.Errorf("%w; failed to restore raced migration marker: %v", ErrConcurrentChange, restoreErr)
			}
			return fmt.Errorf("%w: quarantined migration marker identity changed", ErrConcurrentChange)
		}
	}
	if !removedExists || removed.id != j.MarkerIdentity {
		return fmt.Errorf("%w: consumed migration marker is not durably quarantined", ErrConcurrentChange)
	}
	j.State = "marker_quarantined"
	return storeJournalStandalone(backupDir, j, journalHash, durability)
}

func finalizeMarkerRemoval(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker) error {
	if j.MarkerIdentity.zero() {
		return nil
	}
	if j.State != "marker_finalize_ready" {
		j.State = "marker_finalize_ready"
		if err := storeJournalStandalone(backupDir, j, journalHash, durability); err != nil {
			return err
		}
	}
	removed, exists, err := exactMarkerAt(backupDir, j.MarkerRemoveName, *j)
	if err != nil {
		return err
	}
	if exists {
		if removed.id != j.MarkerIdentity {
			return fmt.Errorf("%w: quarantined migration marker identity changed", ErrConcurrentChange)
		}
		if err := syscall.Unlinkat(int(backupDir.Fd()), j.MarkerRemoveName); err != nil {
			return fmt.Errorf("remove quarantined migration marker: %w", err)
		}
		if err := durability.syncDir(backupDir, j.BackupRoot); err != nil {
			return err
		}
	}
	j.MarkerConsumed = false
	j.State = "marker_removed"
	return storeJournalStandalone(backupDir, j, journalHash, durability)
}

func restoreQuarantinedMarker(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker) error {
	if j.MarkerIdentity.zero() || j.MarkerRemoveName == "" {
		return nil
	}
	marker, markerExists, err := exactMarkerAt(backupDir, j.MarkerName, *j)
	if err != nil {
		return err
	}
	removed, removedExists, err := exactMarkerAt(backupDir, j.MarkerRemoveName, *j)
	if err != nil {
		return err
	}
	if markerExists {
		if marker.id != j.MarkerIdentity || removedExists {
			return fmt.Errorf("%w: cannot restore quarantined migration marker", ErrConcurrentChange)
		}
		return nil
	}
	if !removedExists || removed.id != j.MarkerIdentity {
		return fmt.Errorf("%w: quarantined migration marker is missing", ErrConcurrentChange)
	}
	if err := unix.Renameat2(int(backupDir.Fd()), j.MarkerRemoveName, int(backupDir.Fd()), j.MarkerName, unix.RENAME_NOREPLACE); err != nil {
		return fmt.Errorf("restore consumed migration marker: %w", err)
	}
	if err := durability.syncDir(backupDir, j.BackupRoot); err != nil {
		return err
	}
	j.State = "rollback_ready"
	j.MarkerConsumed = true
	return storeJournalStandalone(backupDir, j, journalHash, durability)
}

func exactMarkerAt(backupDir *os.File, name string, j journal) (snapshot, bool, error) {
	if name == "" {
		return snapshot{}, false, nil
	}
	marker, exists, err := readOptionalAt(backupDir, name)
	if err != nil || !exists {
		return marker, exists, err
	}
	if marker.id != j.MarkerIdentity {
		return marker, true, nil
	}
	if err := verifyPrivateArtifact(marker, "migration marker"); err != nil {
		return snapshot{}, false, err
	}
	expectedData := migrationMarkerData(j.ProfileID, j.MigrationID, j.Destination)
	if marker.hash != j.MarkerSHA256 || marker.hash != digest(expectedData) || !bytes.Equal(marker.data, expectedData) {
		return snapshot{}, false, fmt.Errorf("%w: migration marker content changed", ErrConcurrentChange)
	}
	return marker, true, nil
}

func verifyOwnedMarkerGone(backupDir *os.File, j journal) error {
	if j.Version != journalVersion || j.DestinationKind != DestinationSymlink || j.MarkerIdentity.zero() {
		return nil
	}
	for _, name := range []string{j.MarkerName, j.MarkerTempName, j.MarkerRemoveName} {
		marker, exists, err := exactMarkerAt(backupDir, name, j)
		if err != nil {
			return err
		}
		if exists && marker.id == j.MarkerIdentity {
			return fmt.Errorf("%w: transaction-owned migration marker still exists", ErrConcurrentChange)
		}
	}
	return nil
}

func migrationMarkerData(profileID, migrationID, destination string) []byte {
	return []byte("OVAV hostprojection migration consumed\nprofile=" + profileID + "\nmigration=" + migrationID + "\ndestination=" + destination + "\n")
}

func (t *Transaction) rollbackAfterApplyFailure(backupDir *os.File, j *journal, durability durabilityTracker, cause error) (Result, error) {
	result, rollbackErr := rollbackJournal(backupDir, j, &t.journalHash, &durability, t.afterMarkerRemoval)
	if rollbackErr != nil {
		return result, fmt.Errorf("%w; automatic rollback failed: %w", cause, rollbackErr)
	}
	return result, fmt.Errorf("%w; automatic rollback completed", cause)
}

func (t *Transaction) verifySource() error {
	parent, name, err := openParentAbsolute(t.preview.Source)
	if err != nil {
		return fmt.Errorf("reopen source: %w", err)
	}
	defer parent.Close()
	if _, err := verifyAt(parent, name, t.source); err != nil {
		return fmt.Errorf("source changed after planning: %w", err)
	}
	return nil
}

func (t *Transaction) newJournal(state string) journal {
	markerName := ""
	if t.preview.DestinationKind == DestinationSymlink {
		markerName = t.markerName
	}
	return journal{
		Version: journalVersion, State: state, Source: t.preview.Source, Destination: t.preview.Destination,
		AllowedRoot: t.preview.AllowedRoot, BackupRoot: t.preview.BackupRoot, BackupPath: t.preview.BackupPath,
		JournalPath: t.preview.JournalPath, LockPath: t.preview.LockPath, PlannedAt: t.preview.PlannedAt,
		SourceSHA256: t.preview.SourceSHA256, OriginalSHA256: t.preview.OriginalSHA256,
		DestinationKind: t.preview.DestinationKind, OriginalLinkText: t.preview.OriginalLinkText,
		ExpectedLinkTarget: t.preview.ExpectedLinkTarget, ProfileID: t.preview.ProfileID, MigrationID: t.preview.MigrationID,
		MarkerName: markerName,
		SourceMode: t.source.mode, OriginalMode: t.original.mode, DestinationExisted: t.preview.DestinationExisted,
		OriginalIdentity: t.original.id, Durability: t.preview.Durability, DurabilityDetail: t.preview.DurabilityDetail,
	}
}

func rejectConsumedMarker(backupDir *os.File, markerName string, expectedData []byte) error {
	marker, exists, err := readOptionalAt(backupDir, markerName)
	if err != nil {
		return fmt.Errorf("inspect consumed migration marker: %w", err)
	}
	if !exists {
		return nil
	}
	if err := verifyPrivateArtifact(marker, "consumed migration marker"); err != nil {
		return err
	}
	if marker.hash != digest(expectedData) || !bytes.Equal(marker.data, expectedData) {
		return errors.New("consumed migration marker content does not match exact migration authority")
	}
	return ErrMigrationConsumed
}

func (t *Transaction) consumeMigrationMarker(backupDir *os.File, j *journal, durability *durabilityTracker) error {
	if t.markerName == "" || len(t.markerData) == 0 {
		return errors.New("symlink migration marker authority is incomplete")
	}
	if err := rejectConsumedMarker(backupDir, t.markerName, t.markerData); err != nil {
		return err
	}
	temp, tempName, markerID, err := createFileAt(backupDir, ".hostprojection-marker-", t.markerData, 0o600)
	if err != nil {
		return fmt.Errorf("stage consumed migration marker: %w", err)
	}
	temp.Close()
	defer func() { _ = syscall.Unlinkat(int(backupDir.Fd()), tempName) }()
	staged, err := readRegularAt(backupDir, tempName)
	if err != nil {
		return err
	}
	if err := verifyPrivateArtifact(staged, "staged migration marker"); err != nil {
		return err
	}
	if err := durability.syncDir(backupDir, t.preview.BackupRoot); err != nil {
		return err
	}
	j.State, j.MarkerName, j.MarkerTempName = "marker_ready", t.markerName, tempName
	j.MarkerSHA256, j.MarkerIdentity = staged.hash, markerID
	if err := t.storeJournal(backupDir, j, false, durability); err != nil {
		_ = syscall.Unlinkat(int(backupDir.Fd()), tempName)
		_ = durability.syncDir(backupDir, t.preview.BackupRoot)
		return err
	}
	if err := unix.Renameat2(int(backupDir.Fd()), tempName, int(backupDir.Fd()), t.markerName, unix.RENAME_NOREPLACE); err != nil {
		if errors.Is(err, syscall.EEXIST) {
			return ErrMigrationConsumed
		}
		return fmt.Errorf("publish consumed migration marker: %w", err)
	}
	if err := durability.syncDir(backupDir, t.preview.BackupRoot); err != nil {
		return err
	}
	marker, exists, err := readOptionalAt(backupDir, t.markerName)
	if err != nil || !exists {
		return errors.New("verify consumed migration marker: marker is missing")
	}
	if err := verifyPrivateArtifact(marker, "consumed migration marker"); err != nil {
		return err
	}
	if marker.id != markerID || marker.hash != staged.hash || !bytes.Equal(marker.data, t.markerData) {
		return errors.New("verify consumed migration marker: identity or content mismatch")
	}
	j.State = "marker_consumed"
	j.MarkerConsumed = true
	return t.storeJournal(backupDir, j, false, durability)
}

func exchangeStagedDestination(parent *os.File, stagedName, destinationName string, expected snapshot, expectedTarget string) error {
	if err := unix.Renameat2(int(parent.Fd()), stagedName, int(parent.Fd()), destinationName, unix.RENAME_EXCHANGE); err != nil {
		return err
	}
	swapped, verifyErr := readExactSymlinkAt(parent, stagedName, expectedTarget)
	if verifyErr == nil && (swapped.id != expected.id || swapped.linkText != expected.linkText) {
		verifyErr = ErrConcurrentChange
	}
	if verifyErr != nil {
		if restoreErr := unix.Renameat2(int(parent.Fd()), stagedName, int(parent.Fd()), destinationName, unix.RENAME_EXCHANGE); restoreErr != nil {
			return fmt.Errorf("%w; failed to restore raced destination: %v", verifyErr, restoreErr)
		}
		return fmt.Errorf("%w: destination symlink changed during exchange", ErrConcurrentChange)
	}
	if err := syscall.Unlinkat(int(parent.Fd()), stagedName); err != nil {
		return fmt.Errorf("remove exchanged destination symlink: %w", err)
	}
	return nil
}

func (t *Transaction) storeJournal(dir *os.File, j *journal, initial bool, durability *durabilityTracker) error {
	hash, err := storeJournal(dir, filepath.Base(t.preview.JournalPath), j, t.journalHash, initial, durability)
	if hash != "" {
		t.journalHash = hash
	}
	return err
}

func storeJournalStandalone(dir *os.File, j *journal, expected *string, durability *durabilityTracker) error {
	hash, err := storeJournal(dir, filepath.Base(j.JournalPath), j, *expected, false, durability)
	if hash != "" {
		*expected = hash
	}
	return err
}

func storeJournal(dir *os.File, name string, j *journal, expectedHash string, initial bool, durability *durabilityTracker) (string, error) {
	j.UpdatedAt = time.Now().UTC()
	j.Durability, j.DurabilityDetail = durability.level, durability.detail
	data, err := json.Marshal(j)
	if err != nil {
		return "", fmt.Errorf("encode transaction journal: %w", err)
	}
	temp, tempName, _, err := createFileAt(dir, ".hostprojection-journal-", data, 0o600)
	if err != nil {
		return "", err
	}
	temp.Close()
	defer func() { _ = syscall.Unlinkat(int(dir.Fd()), tempName) }()
	current, exists, err := readOptionalAt(dir, name)
	if err != nil {
		return "", fmt.Errorf("read current journal: %w", err)
	}
	if initial && exists {
		return "", errors.New("journal already exists")
	}
	if !initial && (!exists || current.hash != expectedHash) {
		return "", fmt.Errorf("%w: journal changed", ErrConcurrentChange)
	}
	if err := syscall.Renameat(int(dir.Fd()), tempName, int(dir.Fd()), name); err != nil {
		return "", fmt.Errorf("replace transaction journal: %w", err)
	}
	syncErr := durability.syncDir(dir, dir.Name())
	written, err := readRegularAt(dir, name)
	if err != nil {
		return "", fmt.Errorf("verify transaction journal: %w", err)
	}
	if err := verifyPrivateArtifact(written, "transaction journal"); err != nil {
		return "", err
	}
	if !bytes.Equal(written.data, data) {
		return "", errors.New("verify transaction journal: content mismatch")
	}
	if syncErr != nil {
		return written.hash, syncErr
	}
	return written.hash, nil
}

func loadJournal(dir *os.File, name string) (journal, string, error) {
	j, file, err := loadJournalSnapshot(dir, name)
	return j, file.hash, err
}

func loadJournalSnapshot(dir *os.File, name string) (journal, snapshot, error) {
	file, err := readJournalSnapshot(dir, name)
	if err != nil {
		return journal{}, snapshot{}, err
	}
	j, err := decodeJournalSnapshot(file)
	return j, file, err
}

func readJournalSnapshot(dir *os.File, name string) (snapshot, error) {
	file, err := readRegularAtBounded(dir, name, maximumJournalBytes)
	if err != nil {
		return snapshot{}, fmt.Errorf("read transaction journal: %w", err)
	}
	if err := verifyPrivateArtifact(file, "transaction journal"); err != nil {
		return snapshot{}, err
	}
	return file, nil
}

func decodeJournalSnapshot(file snapshot) (journal, error) {
	var j journal
	decoder := json.NewDecoder(bytes.NewReader(file.data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&j); err != nil {
		return journal{}, fmt.Errorf("decode transaction journal: %w", err)
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return journal{}, errors.New("decode transaction journal: trailing JSON content")
	}
	if j.Version == 1 {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(file.data, &fields); err != nil {
			return journal{}, fmt.Errorf("inspect v1 journal fields: %w", err)
		}
		for _, field := range []string{
			"profile_id", "migration_id", "original_link_text", "expected_link_target",
			"marker_name", "marker_sha256", "marker_consumed", "marker_identity", "marker_temp_name", "marker_remove_name", "restore_temp_name",
		} {
			if _, exists := fields[field]; exists {
				return journal{}, fmt.Errorf("v1 journal contains v2 field %q", field)
			}
		}
	}
	return j, nil
}

func writeArtifact(dir *os.File, name string, data []byte, mode uint32, durability *durabilityTracker) error {
	temp, tempName, _, err := createFileAt(dir, ".hostprojection-backup-", data, mode)
	if err != nil {
		return err
	}
	temp.Close()
	defer func() { _ = syscall.Unlinkat(int(dir.Fd()), tempName) }()
	if _, exists, err := readOptionalAt(dir, name); err != nil || exists {
		if err != nil {
			return err
		}
		return errors.New("backup already exists")
	}
	if err := syscall.Renameat(int(dir.Fd()), tempName, int(dir.Fd()), name); err != nil {
		return fmt.Errorf("publish backup: %w", err)
	}
	return durability.syncDir(dir, dir.Name())
}

func openBackupRoot(path string) (*os.File, error) {
	dir, err := openDirAbsolute(path)
	if err != nil {
		return nil, fmt.Errorf("open backup root: %w", err)
	}
	if err := validateBackupRootDescriptor(dir); err != nil {
		dir.Close()
		return nil, err
	}
	return dir, nil
}

func ensureBackupRoot(path string, expectedParent fileIdentity, expectedMissing []string, durability *durabilityTracker) (*os.File, error) {
	parent, missing, err := openDirPrefix(path)
	if err != nil {
		return nil, fmt.Errorf("revalidate backup root path: %w", err)
	}
	parentID, err := descriptorIdentity(parent)
	if err != nil {
		parent.Close()
		return nil, err
	}
	if parentID != expectedParent || !equalStrings(missing, expectedMissing) {
		parent.Close()
		return nil, fmt.Errorf("%w: backup root path changed after planning", ErrConcurrentChange)
	}
	if len(missing) == 0 {
		if err := validateBackupRootDescriptor(parent); err != nil {
			parent.Close()
			return nil, err
		}
		return parent, nil
	}
	for _, component := range missing {
		if err := syscall.Mkdirat(int(parent.Fd()), component, 0o700); err != nil {
			parent.Close()
			if errors.Is(err, syscall.EEXIST) {
				return nil, fmt.Errorf("%w: backup component %q appeared after planning", ErrConcurrentChange, component)
			}
			return nil, fmt.Errorf("mkdirat backup component %q: %w", component, err)
		}
		fd, err := syscall.Openat(int(parent.Fd()), component, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if err != nil {
			parent.Close()
			return nil, fmt.Errorf("open created backup component %q: %w", component, err)
		}
		next := os.NewFile(uintptr(fd), filepath.Join(parent.Name(), component))
		if err := syscall.Fchmod(fd, 0o700); err != nil {
			next.Close()
			parent.Close()
			return nil, fmt.Errorf("chmod created backup component %q: %w", component, err)
		}
		if err := validateBackupRootDescriptor(next); err != nil {
			next.Close()
			parent.Close()
			return nil, fmt.Errorf("validate created backup component %q: %w", component, err)
		}
		if err := durability.syncDir(parent, parent.Name()); err != nil {
			next.Close()
			parent.Close()
			return nil, err
		}
		parent.Close()
		parent = next
	}
	createdID, err := descriptorIdentity(parent)
	if err != nil {
		parent.Close()
		return nil, err
	}
	reopened, err := openBackupRoot(path)
	if err != nil {
		parent.Close()
		return nil, fmt.Errorf("reopen created backup root: %w", err)
	}
	reopenedID, err := descriptorIdentity(reopened)
	parent.Close()
	if err != nil || reopenedID != createdID {
		reopened.Close()
		return nil, fmt.Errorf("%w: created backup root identity changed", ErrConcurrentChange)
	}
	return reopened, nil
}

func validateBackupRootDescriptor(dir *os.File) error {
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(dir.Fd()), &stat); err != nil {
		return fmt.Errorf("fstat backup root: %w", err)
	}
	if stat.Mode&syscall.S_IFMT != syscall.S_IFDIR || stat.Mode&0o777 != 0o700 {
		return fmt.Errorf("backup root mode is %04o; require directory 0700", stat.Mode&0o777)
	}
	if stat.Mode&(syscall.S_ISUID|syscall.S_ISGID|syscall.S_ISVTX) != 0 {
		return errors.New("backup root has unsafe special mode bits")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("backup root owner is uid %d; require effective uid %d", stat.Uid, os.Geteuid())
	}
	return nil
}

func descriptorIdentity(file *os.File) (fileIdentity, error) {
	var stat syscall.Stat_t
	if err := syscall.Fstat(int(file.Fd()), &stat); err != nil {
		return fileIdentity{}, fmt.Errorf("fstat %s: %w", file.Name(), err)
	}
	return identityOf(stat), nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func openDestination(rootPath, relative string) (*os.File, *os.File, string, error) {
	root, err := openDirAbsolute(rootPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("open allowed root: %w", err)
	}
	parent, name, err := openParentRelative(root, relative)
	if err != nil {
		root.Close()
		return nil, nil, "", fmt.Errorf("open destination parent: %w", err)
	}
	return root, parent, name, nil
}

func verifyOptional(parent *os.File, name string, expected snapshot, existed bool) error {
	current, actual, err := readOptionalAt(parent, name)
	if err != nil {
		return err
	}
	if actual != existed {
		return ErrConcurrentChange
	}
	if existed && (current.id != expected.id || current.hash != expected.hash || !bytes.Equal(current.data, expected.data)) {
		return ErrConcurrentChange
	}
	return nil
}

func verifyDestinationOptional(parent *os.File, name string, expected snapshot, existed bool, expectedTarget string) error {
	current, actual, err := readDestinationOptionalAt(parent, name, expectedTarget)
	if err != nil {
		return err
	}
	if actual != existed {
		return ErrConcurrentChange
	}
	if !existed {
		return nil
	}
	if current.kind != expected.kind || current.id != expected.id {
		return ErrConcurrentChange
	}
	if current.kind == DestinationSymlink {
		if current.linkText != expected.linkText {
			return ErrConcurrentChange
		}
		return nil
	}
	if current.hash != expected.hash || !bytes.Equal(current.data, expected.data) {
		return ErrConcurrentChange
	}
	return nil
}

func verifyRolledBack(current snapshot, exists bool, j journal) error {
	if !j.DestinationExisted {
		if exists {
			return fmt.Errorf("%w: destination recreated after rollback", ErrConcurrentChange)
		}
		return nil
	}
	if !exists || current.id != j.RestoredIdentity {
		return fmt.Errorf("%w: restored destination changed", ErrConcurrentChange)
	}
	if j.DestinationKind == DestinationSymlink {
		if current.kind != DestinationSymlink || current.linkText != j.OriginalLinkText {
			return fmt.Errorf("%w: restored destination symlink changed", ErrConcurrentChange)
		}
		return nil
	}
	if current.kind != DestinationRegular || current.hash != j.OriginalSHA256 {
		return fmt.Errorf("%w: restored destination changed", ErrConcurrentChange)
	}
	return nil
}

func verifyBackupIfPresent(dir *os.File, j journal) error {
	backup, exists, err := readOptionalAt(dir, filepath.Base(j.BackupPath))
	if err != nil {
		return fmt.Errorf("inspect recovery backup: %w", err)
	}
	if exists {
		if err := verifyPrivateArtifact(backup, "recovery backup"); err != nil {
			return err
		}
	}
	if exists && backup.hash != j.OriginalSHA256 {
		return errors.New("recovery backup hash mismatch")
	}
	if (j.State == "backed_up" || j.State == "ready" || j.State == "applied" || j.State == "verified") && j.DestinationKind == DestinationRegular && !exists {
		return errors.New("required recovery backup is missing")
	}
	return nil
}

func validateJournal(j journal, path, allowedRoot, backupRoot string) error {
	if j.Version != 1 && j.Version != journalVersion {
		return fmt.Errorf("unsupported journal version %d", j.Version)
	}
	kind := j.DestinationKind
	if kind == "" {
		if j.DestinationExisted {
			kind = DestinationRegular
		} else {
			kind = DestinationAbsent
		}
	}
	if kind != DestinationAbsent && kind != DestinationRegular && (kind != DestinationSymlink || j.Version == 1) {
		return fmt.Errorf("invalid destination kind %q", kind)
	}
	if kind == DestinationAbsent && j.DestinationExisted {
		return errors.New("absent destination kind conflicts with destination_existed")
	}
	if kind == DestinationRegular && (!j.DestinationExisted || j.OriginalSHA256 == "") {
		return errors.New("regular destination kind lacks original-file authority")
	}
	if j.Version == 1 {
		if err := validateV1JournalFields(j); err != nil {
			return err
		}
	} else if kind == DestinationSymlink {
		if !j.DestinationExisted || j.OriginalLinkText == "" || j.OriginalLinkText != j.ExpectedLinkTarget ||
			!filepath.IsAbs(j.OriginalLinkText) || filepath.Clean(j.OriginalLinkText) != j.OriginalLinkText {
			return errors.New("journal symlink migration authority is invalid")
		}
		if !validAuthorityID(j.ProfileID) || !validAuthorityID(j.MigrationID) {
			return errors.New("journal symlink migration profile or migration ID is invalid")
		}
		if j.MarkerName != migrationMarkerName(j.MigrationID) {
			return errors.New("journal migration marker name does not match migration authority")
		}
		if j.MarkerSHA256 != "" && j.MarkerSHA256 != digest(migrationMarkerData(j.ProfileID, j.MigrationID, j.Destination)) {
			return errors.New("journal migration marker digest does not match migration authority")
		}
	} else if j.OriginalLinkText != "" {
		return errors.New("non-symlink journal contains original symlink text")
	} else if j.ExpectedLinkTarget != "" && (!filepath.IsAbs(j.ExpectedLinkTarget) || filepath.Clean(j.ExpectedLinkTarget) != j.ExpectedLinkTarget) {
		return errors.New("journal expected symlink target is not absolute and traversal-free")
	}
	if j.Version == journalVersion && j.ProfileID != "" && !validAuthorityID(j.ProfileID) {
		return errors.New("journal profile ID is invalid")
	}
	if err := validateJournalTempName(j.TempName, ".hostprojection-"); err != nil {
		return fmt.Errorf("invalid apply temp name: %w", err)
	}
	if err := validateJournalTempName(j.RestoreTempName, ".hostprojection-restore-link-"); err != nil {
		return fmt.Errorf("invalid symlink restore temp name: %w", err)
	}
	if kind != DestinationSymlink && j.RestoreTempName != "" {
		return errors.New("non-symlink journal contains a symlink restore temp name")
	}
	if err := validateJournalTempName(j.MarkerTempName, ".hostprojection-marker-"); err != nil {
		return fmt.Errorf("invalid migration marker temp name: %w", err)
	}
	if err := validateJournalTempName(j.MarkerRemoveName, ".hostprojection-marker-remove-"); err != nil {
		return fmt.Errorf("invalid migration marker removal name: %w", err)
	}
	if kind != DestinationSymlink && (j.MarkerName != "" || j.MarkerSHA256 != "" || j.MarkerConsumed || !j.MarkerIdentity.zero() || j.MarkerTempName != "" || j.MarkerRemoveName != "") {
		return errors.New("non-symlink journal contains migration marker state")
	}
	if j.JournalPath != path || j.AllowedRoot != allowedRoot || j.BackupRoot != backupRoot {
		return errors.New("journal roots or path do not match recovery authority")
	}
	if filepath.Dir(j.BackupPath) != backupRoot || filepath.Dir(j.LockPath) != backupRoot || filepath.Dir(j.Destination) == "" {
		return errors.New("journal contains paths outside authorized roots")
	}
	rel, err := filepath.Rel(allowedRoot, j.Destination)
	if err != nil {
		return fmt.Errorf("validate journal destination: %w", err)
	}
	if _, err := safeRelative(rel); err != nil {
		return errors.New("journal destination escapes allowed root")
	}
	if rel == "." {
		return errors.New("journal destination is not below allowed root")
	}
	validStates := map[string]bool{"prepared": true, "backed_up": true, "ready": true, "applied": true, "verified": true, "rollback_ready": true, "delete_ready": true, "rolled_back": true}
	if j.Version == journalVersion {
		validStates["marker_ready"] = true
		validStates["marker_consumed"] = true
		validStates["marker_remove_ready"] = true
		validStates["marker_quarantined"] = true
		validStates["marker_finalize_ready"] = true
		validStates["marker_removed"] = true
	}
	if !validStates[j.State] {
		return fmt.Errorf("invalid journal state %q", j.State)
	}
	if j.Version == journalVersion && kind == DestinationSymlink {
		markerState := j.State == "marker_ready" || j.State == "marker_consumed" || j.State == "marker_remove_ready" || j.State == "marker_quarantined" || j.State == "marker_finalize_ready" || j.State == "marker_removed" || j.State == "verified"
		if markerState && (j.MarkerSHA256 == "" || j.MarkerIdentity.zero()) {
			return errors.New("journal migration marker state lacks digest or identity")
		}
		if (j.MarkerSHA256 == "") != j.MarkerIdentity.zero() {
			return errors.New("journal migration marker digest and identity must appear together")
		}
		if j.State == "marker_ready" && j.MarkerTempName == "" {
			return errors.New("marker_ready journal lacks marker temp name")
		}
		if j.State == "marker_remove_ready" && j.MarkerRemoveName == "" {
			return errors.New("marker_remove_ready journal lacks marker removal name")
		}
		if (j.State == "marker_consumed" || j.State == "verified" || j.State == "marker_remove_ready" || j.State == "marker_quarantined" || j.State == "marker_finalize_ready") && !j.MarkerConsumed {
			return errors.New("journal state requires a consumed migration marker")
		}
	}
	return nil
}

func validateV1JournalFields(j journal) error {
	if j.DestinationKind == DestinationSymlink || j.OriginalLinkText != "" || j.ExpectedLinkTarget != "" || j.ProfileID != "" || j.MigrationID != "" ||
		j.MarkerName != "" || j.MarkerSHA256 != "" || j.MarkerConsumed || !j.MarkerIdentity.zero() || j.RestoreTempName != "" || j.MarkerTempName != "" || j.MarkerRemoveName != "" {
		return errors.New("v1 journal contains v2 symlink migration fields")
	}
	return nil
}

func validateJournalTempName(name, prefix string) error {
	if name == "" {
		return nil
	}
	if filepath.Base(name) != name || name == "." || name == ".." || !strings.HasPrefix(name, prefix) {
		return errors.New("temporary name is not a direct governed child")
	}
	return nil
}

func verifyPrivateArtifact(file snapshot, label string) error {
	if file.mode != 0o600 || file.special != 0 || file.links != 1 || file.owner != uint32(os.Geteuid()) {
		return fmt.Errorf("%s must be mode 0600, have one hard link, and be owned by effective uid %d", label, os.Geteuid())
	}
	return nil
}

func absolute(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func (t *Transaction) result(operation, state string) Result {
	return Result{
		Operation: operation, JournalState: state, JournalPath: t.preview.JournalPath, BackupPath: t.preview.BackupPath,
		Durability: t.preview.Durability, DurabilityDetail: t.preview.DurabilityDetail,
	}
}

func (t *Transaction) resultWithDurability(operation, state string, d durabilityTracker) Result {
	result := t.result(operation, state)
	result.Durability, result.DurabilityDetail = d.level, d.detail
	return result
}

func resultFromJournal(j journal, operation string) Result {
	return Result{
		Operation: operation, JournalState: j.State, JournalPath: j.JournalPath, BackupPath: j.BackupPath,
		Durability: j.Durability, DurabilityDetail: j.DurabilityDetail,
	}
}
