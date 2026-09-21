package hostsync

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/hostprojection"
)

// Run plans, applies, or rolls back one exact allowlisted host profile.
func Run(request Request) (Result, error) {
	resolvedRoots, err := validateRoots(request)
	if err != nil {
		return Result{}, err
	}
	if request.RollbackJournal != "" {
		if request.Profile != "" || request.Apply {
			return Result{}, errors.New("rollback journal conflicts with host profile apply/plan")
		}
		if !request.ApproveHostWrite {
			return Result{}, errors.New("rollback requires explicit host-write approval")
		}
		return rollback(request, resolvedRoots)
	}
	if request.Profile == "" {
		return Result{}, errors.New("an exact host profile is required")
	}
	if request.ApproveHostWrite && !request.Apply {
		return Result{}, errors.New("host-write approval is valid only with apply or rollback")
	}
	if request.Apply && !request.ApproveHostWrite {
		return Result{}, errors.New("apply requires explicit host-write approval")
	}
	definition, ok := profileByName(request.Profile)
	if !ok {
		return Result{}, fmt.Errorf("unknown host profile %q", request.Profile)
	}
	profile, err := resolveDefinition(definition, resolvedRoots)
	if err != nil {
		return Result{}, err
	}
	at := request.Now
	if at.IsZero() {
		at = time.Now()
	}
	options := hostprojection.PlanOptions{
		ProfileID: profile.definition.profile.Name, MigrationID: profile.definition.profile.MigrationID,
	}
	if profile.expectedSymlinkTarget != "" {
		options.ExactSymlinkMigration = &hostprojection.ExactSymlinkMigration{ExpectedTarget: profile.expectedSymlinkTarget}
	}
	transaction, err := hostprojection.PlanValidatedWithOptions(
		profile.source,
		profile.destination,
		profile.allowedRoot,
		profile.backupRoot,
		at,
		profile.definition.validate,
		options,
	)
	if err != nil {
		return Result{}, fmt.Errorf("plan host profile %q: %w", request.Profile, err)
	}
	result := resultFromPreview(profile, transaction.Preview())
	if !request.Apply {
		return result, nil
	}
	mutation, err := transaction.Apply()
	result.mergeMutation(mutation)
	result.Mode = "apply"
	result.Operation = "apply"
	result.Approved = true
	result.WritesPerformed = mutation.Applied || mutation.RolledBack
	if err != nil {
		return result, fmt.Errorf("apply host profile %q: %w", request.Profile, err)
	}
	return result, nil
}

func validateRoots(request Request) (roots, error) {
	repoRoot, err := exactAbsolute("repository root", request.RepoRoot)
	if err != nil {
		return roots{}, err
	}
	home, err := exactAbsolute("home", request.Home)
	if err != nil {
		return roots{}, err
	}
	var windowsHome string
	if request.WindowsHome != "" {
		windowsHome, err = exactAbsolute("Windows home", request.WindowsHome)
		if err != nil {
			return roots{}, err
		}
	}
	return roots{repoRoot: repoRoot, home: home, windowsHome: windowsHome}, nil
}

func exactAbsolute(label, path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be absolute", label)
	}
	clean := filepath.Clean(path)
	if clean != path {
		return "", fmt.Errorf("%s must be clean and traversal-free", label)
	}
	return clean, nil
}

func resultFromPreview(profile resolvedProfile, preview hostprojection.Preview) Result {
	return Result{
		SchemaVersion: resultSchema, Operation: "plan", Mode: "plan", Profile: profile.definition.profile.Name,
		Source: preview.Source, Destination: preview.Destination, AllowedRoot: preview.AllowedRoot,
		BackupRoot: preview.BackupRoot, BackupPath: preview.BackupPath, JournalPath: preview.JournalPath,
		SourceSHA256: preview.SourceSHA256, OriginalSHA256: preview.OriginalSHA256,
		DestinationExisted: preview.DestinationExisted, Durability: preview.Durability,
		DurabilityDetail: preview.DurabilityDetail,
	}
}

func (result *Result) mergeMutation(mutation hostprojection.Result) {
	result.Applied = mutation.Applied
	result.RolledBack = mutation.RolledBack
	result.Recovered = mutation.Recovered
	result.AlreadyComplete = mutation.AlreadyComplete
	result.JournalState = mutation.JournalState
	result.JournalPath = mutation.JournalPath
	result.BackupPath = mutation.BackupPath
	result.Durability = mutation.Durability
	result.DurabilityDetail = mutation.DurabilityDetail
}

func rollback(request Request, resolvedRoots roots) (Result, error) {
	journalPath, err := exactAbsolute("rollback journal", request.RollbackJournal)
	if err != nil {
		return Result{}, err
	}
	backupRoot := filepath.Join(resolvedRoots.home, ".local", "state", "ovav", "host-projection")
	if filepath.Dir(journalPath) != backupRoot {
		return Result{}, errors.New("rollback journal must be a direct child of the governed backup root")
	}
	inspection, err := hostprojection.InspectJournal(journalPath, backupRoot)
	if err != nil {
		return Result{}, err
	}
	profile, err := matchJournalProfile(inspection, resolvedRoots)
	if err != nil {
		return Result{}, err
	}
	expected := inspection.Authority()
	mutation, err := hostprojection.RecoverInspected(inspection, expected)
	result := Result{
		SchemaVersion: resultSchema, Operation: "rollback", Mode: "rollback", Profile: profile.definition.profile.Name,
		Source: profile.source, Destination: profile.destination, AllowedRoot: profile.allowedRoot,
		BackupRoot: profile.backupRoot, JournalPath: journalPath, Approved: true,
	}
	result.mergeMutation(mutation)
	result.WritesPerformed = mutation.RolledBack && !mutation.AlreadyComplete
	if err != nil {
		return result, fmt.Errorf("rollback host profile %q: %w", result.Profile, err)
	}
	return result, nil
}

func matchJournalProfile(inspection hostprojection.JournalInspection, resolvedRoots roots) (resolvedProfile, error) {
	authority := inspection.Authority()
	var matches []resolvedProfile
	for _, definition := range profileRegistry {
		if definition.profile.Windows && resolvedRoots.windowsHome == "" {
			continue
		}
		if !definition.profile.Windows && resolvedRoots.windowsHome != "" {
			continue
		}
		candidate, err := resolveRecoveryDefinition(definition, resolvedRoots)
		if err != nil {
			continue
		}
		if authority.Destination != candidate.destination || authority.AllowedRoot != candidate.allowedRoot || authority.BackupRoot != candidate.backupRoot {
			continue
		}
		if inspection.Version() == 2 {
			if authority.ProfileID != definition.profile.Name || authority.MigrationID != definition.profile.MigrationID {
				continue
			}
			if authority.ExpectedDestinationTarget != candidate.expectedSymlinkTarget {
				continue
			}
		} else if !sourceHasRegisteredSuffix(authority.Source, definition.profile.SourceRelative) {
			continue
		}
		candidate.source = authority.Source
		matches = append(matches, candidate)
	}
	if len(matches) != 1 {
		return resolvedProfile{}, errors.New("rollback journal does not uniquely match an exact allowlisted profile")
	}
	return matches[0], nil
}

func sourceHasRegisteredSuffix(source, sourceRelative string) bool {
	if !filepath.IsAbs(source) || filepath.Clean(source) != source {
		return false
	}
	suffix := filepath.FromSlash(sourceRelative)
	return source != suffix && strings.HasSuffix(source, string(filepath.Separator)+suffix)
}
