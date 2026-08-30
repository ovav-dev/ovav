//go:build linux

package hostprojection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const journalVersion = 1

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
	BackupSHA256       string          `json:"backup_sha256,omitempty"`
	SourceMode         uint32          `json:"source_mode"`
	OriginalMode       uint32          `json:"original_mode,omitempty"`
	DestinationExisted bool            `json:"destination_existed"`
	OriginalIdentity   fileIdentity    `json:"original_identity"`
	PendingIdentity    fileIdentity    `json:"pending_identity"`
	RestoreIdentity    fileIdentity    `json:"restore_identity"`
	RestoredIdentity   fileIdentity    `json:"restored_identity"`
	TempName           string          `json:"temp_name,omitempty"`
	Durability         DurabilityLevel `json:"durability"`
	DurabilityDetail   string          `json:"durability_detail,omitempty"`
}

// Transaction is a Linux-only, locked and journaled host projection.
type Transaction struct {
	mu             sync.Mutex
	preview        Preview
	source         snapshot
	original       snapshot
	destRel        string
	backupParentID fileIdentity
	backupMissing  []string
	journalHash    string
	afterRename    func(string) error
}

// Plan validates all path components with O_NOFOLLOW and performs no writes.
func Plan(source, destination, allowedRoot, backupRoot string, at time.Time) (*Transaction, error) {
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
	original, existed, err := readOptionalAt(destParent, destName)
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
	}

	durability := newDurability()
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
		SourceSHA256: sourceSnapshot.hash, DestinationExisted: existed, PlatformSupported: true,
		Durability: durability.level, DurabilityDetail: durability.detail,
	}
	if existed {
		preview.OriginalSHA256 = original.hash
	}
	return &Transaction{
		preview: preview, source: sourceSnapshot, original: original, destRel: destRel,
		backupParentID: backupParentID, backupMissing: append([]string(nil), backupMissing...),
	}, nil
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

	root, destParent, destName, err := openDestination(t.preview.AllowedRoot, t.destRel)
	if err != nil {
		return t.result("apply", ""), err
	}
	defer root.Close()
	defer destParent.Close()
	if err := verifyOptional(destParent, destName, t.original, t.preview.DestinationExisted); err != nil {
		return t.result("apply", ""), fmt.Errorf("apply precondition: %w", err)
	}
	if err := t.verifySource(); err != nil {
		return t.result("apply", ""), err
	}

	j := t.newJournal("prepared")
	if err := t.storeJournal(backupDir, &j, true, &durability); err != nil {
		return t.result("apply", j.State), err
	}
	if j.DestinationExisted {
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
	if j.DestinationExisted {
		mode = t.original.mode
	}
	temp, tempName, pendingID, err := createFileAt(destParent, ".hostprojection-", t.source.data, mode)
	if err != nil {
		return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("stage destination: %w", err)
	}
	temp.Close()
	defer func() { _ = syscall.Unlinkat(int(destParent.Fd()), tempName) }()
	j.State, j.TempName, j.PendingIdentity = "ready", tempName, pendingID
	if err := t.storeJournal(backupDir, &j, false, &durability); err != nil {
		return t.resultWithDurability("apply", j.State, durability), err
	}
	if err := verifyOptional(destParent, destName, t.original, j.DestinationExisted); err != nil {
		return t.resultWithDurability("apply", j.State, durability), fmt.Errorf("destination changed before rename: %w", err)
	}
	if err := syscall.Renameat(int(destParent.Fd()), tempName, int(destParent.Fd()), destName); err != nil {
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
	return recoverTransaction(t.preview.JournalPath, t.preview.AllowedRoot, t.preview.BackupRoot, false)
}

// Recover performs idempotent, lock-protected recovery from a durable journal.
func Recover(journalPath, allowedRoot, backupRoot string) (Result, error) {
	return recoverTransaction(journalPath, allowedRoot, backupRoot, true)
}

func recoverTransaction(journalPath, allowedRoot, backupRoot string, recovery bool) (Result, error) {
	journalPath, err := absolute(journalPath)
	if err != nil {
		return Result{Operation: "recover"}, fmt.Errorf("resolve journal: %w", err)
	}
	allowedRoot, err = absolute(allowedRoot)
	if err != nil {
		return Result{Operation: "recover"}, fmt.Errorf("resolve allowed root: %w", err)
	}
	backupRoot, err = absolute(backupRoot)
	if err != nil {
		return Result{Operation: "recover"}, fmt.Errorf("resolve backup root: %w", err)
	}
	if filepath.Dir(journalPath) != backupRoot {
		return Result{Operation: "recover"}, errors.New("journal is not a direct child of backup root")
	}
	backupDir, err := openBackupRoot(backupRoot)
	if err != nil {
		return Result{Operation: "recover"}, err
	}
	defer backupDir.Close()
	j, journalHash, err := loadJournal(backupDir, filepath.Base(journalPath))
	if err != nil {
		return Result{Operation: "recover"}, err
	}
	if err := validateJournal(j, journalPath, allowedRoot, backupRoot); err != nil {
		return resultFromJournal(j, "recover"), err
	}
	lock, err := openLockAt(backupDir, filepath.Base(j.LockPath))
	if err != nil {
		return resultFromJournal(j, "recover"), fmt.Errorf("recover: %w", err)
	}
	defer closeLock(lock)
	latest, latestHash, err := loadJournal(backupDir, filepath.Base(journalPath))
	if err != nil {
		return resultFromJournal(j, "recover"), fmt.Errorf("journal changed while acquiring lock: %w", err)
	}
	if latestHash != journalHash {
		return resultFromJournal(j, "recover"), fmt.Errorf("%w: journal changed while acquiring lock", ErrConcurrentChange)
	}
	j = latest
	durability := durabilityTracker{level: j.Durability, detail: j.DurabilityDetail}
	result, err := rollbackJournal(backupDir, &j, &journalHash, &durability)
	result.Recovered = recovery
	return result, err
}

func rollbackJournal(backupDir *os.File, j *journal, journalHash *string, durability *durabilityTracker) (Result, error) {
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
	current, exists, err := readOptionalAt(destParent, destName)
	if err != nil {
		return resultFromJournal(*j, "rollback"), fmt.Errorf("read rollback destination: %w", err)
	}

	if j.State == "rolled_back" {
		if err := verifyRolledBack(current, exists, *j); err != nil {
			return resultFromJournal(*j, "rollback"), err
		}
		result := resultFromJournal(*j, "rollback")
		result.RolledBack, result.AlreadyComplete = true, true
		return result, nil
	}
	if j.State == "rollback_ready" && exists && current.id == j.RestoreIdentity && current.hash == j.OriginalSHA256 {
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

	applied := exists && !j.PendingIdentity.zero() && current.id == j.PendingIdentity && current.hash == j.SourceSHA256
	original := j.DestinationExisted && exists && current.id == j.OriginalIdentity && current.hash == j.OriginalSHA256
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

	if j.DestinationExisted {
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
		temp, name, restoreID, err := createFileAt(destParent, ".hostprojection-restore-", backup.data, j.OriginalMode)
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

func (t *Transaction) rollbackAfterApplyFailure(backupDir *os.File, j *journal, durability durabilityTracker, cause error) (Result, error) {
	result, rollbackErr := rollbackJournal(backupDir, j, &t.journalHash, &durability)
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
	return journal{
		Version: journalVersion, State: state, Source: t.preview.Source, Destination: t.preview.Destination,
		AllowedRoot: t.preview.AllowedRoot, BackupRoot: t.preview.BackupRoot, BackupPath: t.preview.BackupPath,
		JournalPath: t.preview.JournalPath, LockPath: t.preview.LockPath, PlannedAt: t.preview.PlannedAt,
		SourceSHA256: t.preview.SourceSHA256, OriginalSHA256: t.preview.OriginalSHA256,
		SourceMode: t.source.mode, OriginalMode: t.original.mode, DestinationExisted: t.preview.DestinationExisted,
		OriginalIdentity: t.original.id, Durability: t.preview.Durability, DurabilityDetail: t.preview.DurabilityDetail,
	}
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
	file, err := readRegularAt(dir, name)
	if err != nil {
		return journal{}, "", fmt.Errorf("read transaction journal: %w", err)
	}
	if err := verifyPrivateArtifact(file, "transaction journal"); err != nil {
		return journal{}, "", err
	}
	var j journal
	decoder := json.NewDecoder(bytes.NewReader(file.data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&j); err != nil {
		return journal{}, "", fmt.Errorf("decode transaction journal: %w", err)
	}
	return j, file.hash, nil
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

func verifyRolledBack(current snapshot, exists bool, j journal) error {
	if !j.DestinationExisted {
		if exists {
			return fmt.Errorf("%w: destination recreated after rollback", ErrConcurrentChange)
		}
		return nil
	}
	if !exists || current.hash != j.OriginalSHA256 || current.id != j.RestoredIdentity {
		return fmt.Errorf("%w: restored destination changed", ErrConcurrentChange)
	}
	return nil
}

func verifyBackupIfPresent(dir *os.File, j journal) error {
	backup, exists, err := readOptionalAt(dir, filepath.Base(j.BackupPath))
	if err != nil {
		return fmt.Errorf("inspect recovery backup: %w", err)
	}
	if exists && backup.hash != j.OriginalSHA256 {
		return errors.New("recovery backup hash mismatch")
	}
	if (j.State == "backed_up" || j.State == "ready" || j.State == "applied" || j.State == "verified") && j.DestinationExisted && !exists {
		return errors.New("required recovery backup is missing")
	}
	return nil
}

func validateJournal(j journal, path, allowedRoot, backupRoot string) error {
	if j.Version != journalVersion {
		return fmt.Errorf("unsupported journal version %d", j.Version)
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
	if !validStates[j.State] {
		return fmt.Errorf("invalid journal state %q", j.State)
	}
	return nil
}

func verifyPrivateArtifact(file snapshot, label string) error {
	if file.mode != 0o600 || file.links != 1 || file.owner != uint32(os.Geteuid()) {
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
	return Result{Operation: operation, JournalState: state, JournalPath: t.preview.JournalPath, BackupPath: t.preview.BackupPath,
		Durability: t.preview.Durability, DurabilityDetail: t.preview.DurabilityDetail}
}

func (t *Transaction) resultWithDurability(operation, state string, d durabilityTracker) Result {
	result := t.result(operation, state)
	result.Durability, result.DurabilityDetail = d.level, d.detail
	return result
}

func resultFromJournal(j journal, operation string) Result {
	return Result{Operation: operation, JournalState: j.State, JournalPath: j.JournalPath, BackupPath: j.BackupPath,
		Durability: j.Durability, DurabilityDetail: j.DurabilityDetail}
}
