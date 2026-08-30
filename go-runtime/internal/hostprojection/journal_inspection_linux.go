//go:build linux

package hostprojection

import (
	"errors"
	"fmt"
	"path/filepath"
)

const maximumJournalBytes int64 = 1 << 20

// InspectJournal securely reads owner-private journal authority through the
// governed backup-root descriptor. It performs no writes.
func InspectJournal(journalPath, backupRoot string) (JournalInspection, error) {
	if !filepath.IsAbs(journalPath) || filepath.Clean(journalPath) != journalPath {
		return JournalInspection{}, errors.New("journal path must be absolute and traversal-free")
	}
	if !filepath.IsAbs(backupRoot) || filepath.Clean(backupRoot) != backupRoot {
		return JournalInspection{}, errors.New("backup root must be absolute and traversal-free")
	}
	journalPath, err := absolute(journalPath)
	if err != nil {
		return JournalInspection{}, fmt.Errorf("resolve journal: %w", err)
	}
	backupRoot, err = absolute(backupRoot)
	if err != nil {
		return JournalInspection{}, fmt.Errorf("resolve backup root: %w", err)
	}
	if filepath.Dir(journalPath) != backupRoot {
		return JournalInspection{}, errors.New("journal is not a direct child of backup root")
	}
	backupDir, err := openBackupRoot(backupRoot)
	if err != nil {
		return JournalInspection{}, err
	}
	defer backupDir.Close()
	j, file, err := loadJournalSnapshot(backupDir, filepath.Base(journalPath))
	if err != nil {
		return JournalInspection{}, err
	}
	if err := validateJournal(j, journalPath, j.AllowedRoot, backupRoot); err != nil {
		return JournalInspection{}, err
	}
	authority := authorityFromJournal(j)
	if authority.Source == "" || authority.Destination == "" || authority.AllowedRoot == "" {
		return JournalInspection{}, errors.New("journal has incomplete authority")
	}
	return JournalInspection{
		authority:   authority,
		identity:    JournalIdentity{Device: file.id.Device, Inode: file.id.Inode},
		journalPath: journalPath,
		backupRoot:  backupRoot,
		lockPath:    j.LockPath,
		digest:      file.hash,
	}, nil
}

// RecoverInspected recovers only when the trusted inspection token and caller-
// supplied exact authority still match the journal reopened under its lock.
func RecoverInspected(inspection JournalInspection, expected JournalAuthority) (Result, error) {
	if err := validateInspectionToken(inspection, expected); err != nil {
		return Result{Operation: "recover"}, err
	}
	backupDir, err := openBackupRoot(expected.BackupRoot)
	if err != nil {
		return Result{Operation: "recover"}, err
	}
	defer backupDir.Close()
	lock, err := openLockAt(backupDir, filepath.Base(inspection.lockPath))
	if err != nil {
		return Result{Operation: "recover"}, fmt.Errorf("recover: %w", err)
	}
	defer closeLock(lock)

	file, err := readJournalSnapshot(backupDir, filepath.Base(inspection.journalPath))
	if err != nil {
		return Result{Operation: "recover"}, err
	}
	identity := JournalIdentity{Device: file.id.Device, Inode: file.id.Inode}
	if file.hash != inspection.digest || identity != inspection.identity {
		return Result{Operation: "recover", JournalPath: inspection.journalPath}, fmt.Errorf("%w: journal changed after inspection", ErrConcurrentChange)
	}
	j, err := decodeJournalSnapshot(file)
	if err != nil {
		return Result{Operation: "recover", JournalPath: inspection.journalPath}, err
	}
	if err := validateJournal(j, inspection.journalPath, expected.AllowedRoot, expected.BackupRoot); err != nil {
		return resultFromJournal(j, "recover"), err
	}
	if authorityFromJournal(j) != expected {
		return resultFromJournal(j, "recover"), errors.New("journal authority does not match recovery authority")
	}
	durability := durabilityTracker{level: j.Durability, detail: j.DurabilityDetail}
	journalHash := file.hash
	result, err := rollbackJournal(backupDir, &j, &journalHash, &durability)
	result.Recovered = true
	return result, err
}

func validateInspectionToken(inspection JournalInspection, expected JournalAuthority) error {
	if inspection.journalPath == "" || inspection.backupRoot == "" || inspection.lockPath == "" ||
		inspection.digest == "" || inspection.identity == (JournalIdentity{}) {
		return errors.New("invalid or zero journal inspection token")
	}
	if expected != inspection.authority || expected.BackupRoot != inspection.backupRoot {
		return errors.New("expected authority does not match inspected journal authority")
	}
	if filepath.Dir(inspection.journalPath) != inspection.backupRoot || filepath.Dir(inspection.lockPath) != inspection.backupRoot {
		return errors.New("inspection token paths escape backup root")
	}
	return nil
}

func authorityFromJournal(j journal) JournalAuthority {
	return JournalAuthority{
		Source: j.Source, Destination: j.Destination,
		AllowedRoot: j.AllowedRoot, BackupRoot: j.BackupRoot,
	}
}
