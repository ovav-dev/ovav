package identity

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	CanonicalCEOID      = "ceo-alexander"
	CanonicalOrigin     = "https://github.com/ovav-dev/ovav.git"
	CanonicalGitHubRepo = "ovav-dev/ovav"
	// CanonicalGitHubRepoID was independently verified from https://api.github.com/repos/ovav-dev/ovav on 2026-08-13.
	CanonicalGitHubRepoID = int64(1328456440)
	// Canonical CEO principal independently verified at https://api.github.com/users/Alexander-Salvador on 2026-08-13.
	CanonicalCEOLogin           = "Alexander-Salvador"
	CanonicalCEOUserID          = int64(97975177)
	RecoveryConfirmation        = "RECOVER CEO"
	RegistryRelativePath        = ".ovav/registry/identities.yaml"
	RecoveryLockRelativePath    = ".ovav/registry/.identity-recovery.lock"
	RecoveryJournalRelativePath = ".ovav/registry/.identity-recovery.journal"
	AuditRelativePath           = ".ovav/registry/audit.jsonl"
)

const maxHashPrefix = 12

// RecoverySummary is the non-secret authorization summary shown before confirmation.
type RecoverySummary struct {
	RepoRoot     string
	Origin       string
	IdentityID   string
	IdentityName string
	MachineID    string
	GitHubLogin  string
	GitHubUserID int64
	GitHubRepoID int64
}

// RecoverySession contains only the canonical identity data needed by the CLI session.
type RecoverySession struct {
	VaultKeyHash string
	MachineID    string
	CreatedAt    string
	IdentityID   string
	Role         string
	Level        int
	Name         string
	Email        string
}

// RecoverySessionSnapshot preserves the exact pre-recovery session representation.
// Session is retained for dependency-injected tests; production uses Data and Mode.
type RecoverySessionSnapshot struct {
	Exists          bool
	Data            []byte
	Mode            os.FileMode
	Session         RecoverySession
	ExpectedCurrent []byte
	ExpectedMode    os.FileMode
	ExpectedSession RecoverySession
}

// RecoveryResult reports only non-secret recovery metadata.
type RecoveryResult struct {
	Identity           Identity
	GitHubLogin        string
	GitHubUserID       int64
	BackupRelativePath string
	OldHashPrefix      string
	NewHashPrefix      string
}

// RecoveryAuditEntry is the durable, redacted identity recovery audit record.
type RecoveryAuditEntry struct {
	Timestamp          string `json:"timestamp"`
	Action             string `json:"action"`
	IdentityID         string `json:"identity_id"`
	MachineID          string `json:"machine_id"`
	GitHubLogin        string `json:"github_login"`
	GitHubUserID       int64  `json:"github_user_id"`
	GitHubRepoID       int64  `json:"github_repo_id"`
	OldHashPrefix      string `json:"old_hash_prefix"`
	NewHashPrefix      string `json:"new_hash_prefix"`
	BackupRelativePath string `json:"backup_relative_path"`
	Success            bool   `json:"success"`
}

// RecoveryDependencies isolates all host, terminal, process, and session effects.
type RecoveryDependencies struct {
	IsTTY          func() bool
	MachineID      func() (string, error)
	ReadSeed       func() (string, error)
	Confirm        func(RecoverySummary) (string, error)
	Now            func() time.Time
	DeriveKey      func(seed, machineID string) ([]byte, error)
	LookPath       func(name string) (string, error)
	Run            func(name string, args ...string) ([]byte, error)
	SaveSession    func(RecoverySession) error
	RemoveSession  func() error
	CaptureSession func() (RecoverySessionSnapshot, error)
	RestoreSession func(RecoverySessionSnapshot) error
	WriteRegistry  func(path string, data []byte) (bool, error)
	PostVerify     func() error
	SyncDirectory  func(path string) error
}

const githubAuthorizationQuery = `query{viewer{login databaseId} repository(owner:"ovav-dev",name:"ovav"){nameWithOwner databaseId viewerPermission}}`

type githubAuthorization struct {
	Data struct {
		Viewer struct {
			Login      string `json:"login"`
			DatabaseID int64  `json:"databaseId"`
		} `json:"viewer"`
		Repository struct {
			NameWithOwner    string `json:"nameWithOwner"`
			DatabaseID       int64  `json:"databaseId"`
			ViewerPermission string `json:"viewerPermission"`
		} `json:"repository"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

type githubUser struct {
	ID    int64
	Login string
}
type githubRepository struct {
	ID       int64
	FullName string
}

type recoveryJournal struct {
	Version        int    `json:"version"`
	Stage          string `json:"stage"`
	OriginalDigest string `json:"original_registry_digest"`
	RotatedDigest  string `json:"rotated_registry_digest"`
	BackupPath     string `json:"backup_relative_path"`
	StartedAt      string `json:"started_at"`
}

// CanonicalCEO returns the one normalized, active canonical CEO identity.
func CanonicalCEO(reg *Registry) (*Identity, error) {
	if reg == nil {
		return nil, fmt.Errorf("identity recovery: registry is nil")
	}
	if reg.Version != 1 || !reg.Canonical {
		return nil, fmt.Errorf("identity recovery: registry must be version 1 and canonical")
	}

	var matches []*Identity
	for i := range reg.Identities {
		id := &reg.Identities[i]
		if normalize(id.ID) == CanonicalCEOID && normalize(id.Status) == "active" {
			matches = append(matches, id)
		}
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("identity recovery: expected exactly one active %s identity, found %d", CanonicalCEOID, len(matches))
	}

	ceo := matches[0]
	if ceo.ID != CanonicalCEOID {
		return nil, fmt.Errorf("identity recovery: canonical CEO id must be exactly %q", CanonicalCEOID)
	}
	if normalize(ceo.Role) != "ceo" {
		return nil, fmt.Errorf("identity recovery: canonical CEO role must be ceo")
	}
	if ceo.Level != 10 {
		return nil, fmt.Errorf("identity recovery: canonical CEO must be level 10")
	}
	if !hasNormalizedPermission(ceo.Permissions, "manage_identities") ||
		!hasNormalizedPermission(ceo.Permissions, "full_system") {
		return nil, fmt.Errorf("identity recovery: canonical CEO requires manage_identities and full_system")
	}
	return ceo, nil
}

// RecoverCEO performs fail-closed, machine-bound CEO identity recovery.
func RecoverCEO(repoRoot string, deps RecoveryDependencies) (RecoveryResult, error) {
	if err := validateRecoveryDependencies(deps); err != nil {
		return RecoveryResult{}, err
	}
	if !deps.IsTTY() {
		return RecoveryResult{}, fmt.Errorf("identity recovery: an interactive TTY is required")
	}
	if repoRoot == "" {
		return RecoveryResult{}, fmt.Errorf("identity recovery: repository root is required")
	}
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: resolve repository root: %w", err)
	}

	registryPath := filepath.Join(repoRoot, filepath.FromSlash(RegistryRelativePath))
	if err := rejectSymlinkParents(repoRoot, filepath.Dir(filepath.FromSlash(RegistryRelativePath))); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: registry parent rejected: %w", err)
	}
	registryDir := filepath.Dir(registryPath)
	registryDirInfo, err := os.Lstat(registryDir)
	if err != nil || !registryDirInfo.IsDir() || registryDirInfo.Mode()&os.ModeSymlink != 0 {
		return RecoveryResult{}, fmt.Errorf("identity recovery: registry directory is unavailable")
	}

	lock, err := acquireRecoveryLock(repoRoot)
	if err != nil {
		return RecoveryResult{}, err
	}
	defer releaseRecoveryLock(lock)
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return RecoveryResult{}, err
	}
	if _, err := os.Lstat(filepath.Join(repoRoot, RecoveryJournalRelativePath)); err == nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: unresolved recovery journal exists at %s; inspect backup and registry, then remove it manually", RecoveryJournalRelativePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return RecoveryResult{}, fmt.Errorf("identity recovery: inspect recovery journal: %w", err)
	}

	gitPath, err := trustedExecutable(deps.LookPath, "git")
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: trusted git executable is required")
	}
	origin, err := deps.Run(gitPath, "-C", repoRoot, "remote", "get-url", "origin")
	if err != nil || strings.TrimSpace(string(origin)) != CanonicalOrigin {
		return RecoveryResult{}, fmt.Errorf("identity recovery: fetch origin must be exactly %s", CanonicalOrigin)
	}

	original, originalInfo, err := readRegularNoSymlink(registryPath)
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: registry rejected: %w", err)
	}
	head, err := deps.Run(gitPath, "-C", repoRoot, "show", "HEAD:"+RegistryRelativePath)
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: tracked HEAD registry is unavailable")
	}
	if !bytes.Equal(original, head) {
		return RecoveryResult{}, fmt.Errorf("identity recovery: registry must be byte-identical to HEAD")
	}

	var reg Registry
	if err := yaml.Unmarshal(original, &reg); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: invalid registry YAML: %w", err)
	}
	ceo, err := CanonicalCEO(&reg)
	if err != nil {
		return RecoveryResult{}, err
	}

	principal, repository, err := authorizeGitHub(deps.LookPath, deps.Run)
	if err != nil {
		return RecoveryResult{}, err
	}
	machineID, err := deps.MachineID()
	if err != nil || strings.TrimSpace(machineID) == "" {
		return RecoveryResult{}, fmt.Errorf("identity recovery: machine identity is unavailable")
	}

	summary := RecoverySummary{
		RepoRoot: repoRoot, Origin: CanonicalOrigin, IdentityID: ceo.ID, IdentityName: ceo.Name,
		MachineID: machineID, GitHubLogin: principal.Login, GitHubUserID: principal.ID,
		GitHubRepoID: repository.ID,
	}
	confirmation, err := deps.Confirm(summary)
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: confirmation failed: %w", err)
	}
	if confirmation != RecoveryConfirmation {
		return RecoveryResult{}, fmt.Errorf("identity recovery: confirmation must be exactly %q", RecoveryConfirmation)
	}

	seed, err := deps.ReadSeed()
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: hidden seed input failed: %w", err)
	}
	if utf8.RuneCountInString(seed) < 16 {
		return RecoveryResult{}, fmt.Errorf("identity recovery: seed must contain at least 16 characters")
	}
	key, err := deps.DeriveKey(seed, machineID)
	seed = ""
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: key derivation failed: %w", err)
	}
	defer clearBytes(key)
	if len(key) == 0 {
		return RecoveryResult{}, fmt.Errorf("identity recovery: key derivation returned an empty key")
	}

	keyHashSum := sha256.Sum256(key)
	newHash := hex.EncodeToString(keyHashSum[:])
	oldHash := ceo.KeyHash
	now := deps.Now().UTC()
	rotated, rotatedIdentity, err := rotateRegistry(original, newHash, key, now)
	if err != nil {
		return RecoveryResult{}, err
	}
	var sessionSnapshot RecoverySessionSnapshot
	if deps.CaptureSession != nil {
		sessionSnapshot, err = deps.CaptureSession()
		if err != nil {
			return RecoveryResult{}, fmt.Errorf("identity recovery: capture existing session: %w", err)
		}
	}

	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return RecoveryResult{}, err
	}
	backupRelative, err := createRecoveryBackup(repoRoot, registryDirInfo, original, now)
	if err != nil {
		return RecoveryResult{}, err
	}
	journal := recoveryJournal{
		Version: 1, Stage: "prepared", OriginalDigest: digestHex(original), RotatedDigest: digestHex(rotated),
		BackupPath: backupRelative, StartedAt: now.Format(time.RFC3339Nano),
	}
	if err := writeRecoveryJournal(repoRoot, registryDirInfo, journal, nil); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: durable intent journal: %w", err)
	}
	journalBytes, _, _ := readRegularNoSymlink(filepath.Join(repoRoot, RecoveryJournalRelativePath))
	journalActive := true
	removeJournal := func() error {
		if !journalActive {
			return nil
		}
		if err := removeRecoveryJournal(repoRoot, registryDirInfo, journalBytes); err != nil {
			return err
		}
		journalActive = false
		return nil
	}
	cleanupPrepared := func() error {
		if err := removeJournal(); err != nil {
			return err
		}
		return removeRecoveryBackup(repoRoot, registryDirInfo, backupRelative, original)
	}

	writeRegistry := deps.WriteRegistry
	if writeRegistry == nil {
		writeRegistry = func(path string, data []byte) (bool, error) {
			if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
				return false, err
			}
			return atomicReplaceCheckedWithSync(path, data, original, originalInfo, deps.SyncDirectory)
		}
	}
	written, writeErr := writeRegistry(registryPath, rotated)
	if writeErr != nil {
		if !written {
			journalErr := cleanupPrepared()
			return RecoveryResult{}, recoveryFailureNoWrite("write registry", writeErr, journalErr)
		}
		return RecoveryResult{}, fmt.Errorf("identity recovery: registry was published but directory sync failed; journal retained for manual reconciliation: %w", writeErr)
	}
	journal.Stage = "registry_written"
	newJournalBytes, err := marshalJournal(journal)
	if err != nil {
		return RecoveryResult{}, err
	}
	if err := writeRecoveryJournal(repoRoot, registryDirInfo, journal, journalBytes); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: journal update may be published; manual reconciliation required: %w", err)
	}
	journalBytes = newJournalBytes

	sessionTouched := false
	rollback := func(cause error) (RecoveryResult, error) {
		var sessionErr error
		if sessionTouched {
			if deps.RestoreSession != nil {
				sessionErr = deps.RestoreSession(sessionSnapshot)
			} else {
				sessionErr = deps.RemoveSession()
			}
		}
		registryErr := restoreRegistryChecked(repoRoot, registryDirInfo, registryPath, rotated, original)
		if registryErr == nil && sessionErr == nil {
			registryErr = removeRecoveryJournal(repoRoot, registryDirInfo, journalBytes)
			if registryErr == nil {
				journalActive = false
				registryErr = removeRecoveryBackup(repoRoot, registryDirInfo, backupRelative, original)
			}
		}
		return RecoveryResult{}, combinedRecoveryFailure(cause, registryErr, sessionErr)
	}

	loaded, err := LoadRegistry(repoRoot)
	if err != nil {
		return rollback(err)
	}
	loadedCEO, err := CanonicalCEO(loaded)
	if err != nil {
		return rollback(err)
	}
	if loadedCEO.KeyHash != newHash || !sameCanonicalIdentity(rotatedIdentity, *loadedCEO) {
		return rollback(fmt.Errorf("reloaded canonical identity does not match rotation"))
	}
	valid, err := VerifySignature(repoRoot, key)
	if err != nil || !valid {
		if err == nil {
			err = fmt.Errorf("signature verification returned false")
		}
		return rollback(err)
	}
	if deps.PostVerify != nil {
		if err := deps.PostVerify(); err != nil {
			return rollback(err)
		}
	}

	session := RecoverySession{
		VaultKeyHash: newHash, MachineID: machineID, CreatedAt: now.Format(time.RFC3339),
		IdentityID: loadedCEO.ID, Role: loadedCEO.Role, Level: loadedCEO.Level,
		Name: loadedCEO.Name, Email: loadedCEO.Email,
	}
	sessionTouched = true
	if err := deps.SaveSession(session); err != nil {
		return rollback(fmt.Errorf("save session: %w", err))
	}
	if deps.CaptureSession != nil {
		recoverySessionSnapshot, captureErr := deps.CaptureSession()
		if captureErr != nil {
			return rollback(fmt.Errorf("capture recovery session for CAS rollback: %w", captureErr))
		}
		sessionSnapshot.ExpectedCurrent = append([]byte(nil), recoverySessionSnapshot.Data...)
		sessionSnapshot.ExpectedMode = recoverySessionSnapshot.Mode
		sessionSnapshot.ExpectedSession = recoverySessionSnapshot.Session
	}
	journal.Stage = "session_written"
	if err := writeRecoveryJournal(repoRoot, registryDirInfo, journal, journalBytes); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery: session saved and journal update may be published; manual reconciliation required: %w", err)
	}
	journalBytes, _ = marshalJournal(journal)

	audit := RecoveryAuditEntry{
		Timestamp: now.Format(time.RFC3339Nano), Action: "identity_recovered",
		IdentityID: loadedCEO.ID, MachineID: machineID, GitHubLogin: principal.Login,
		GitHubUserID: principal.ID, GitHubRepoID: repository.ID,
		OldHashPrefix: hashPrefix(oldHash), NewHashPrefix: hashPrefix(newHash),
		BackupRelativePath: backupRelative, Success: true,
	}
	auditPublished, err := appendRecoveryAuditChecked(repoRoot, registryDirInfo, audit, deps.SyncDirectory)
	if err != nil && !auditPublished {
		return rollback(fmt.Errorf("durable audit: %w", err))
	}
	if err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery completed and audit was published, but directory sync failed; journal retained for manual finalization: %w", err)
	}
	journal.Stage = "audit_written"
	if err := writeRecoveryJournal(repoRoot, registryDirInfo, journal, journalBytes); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery completed but journal finalization failed; manual recovery required: %w", err)
	}
	journalBytes, _ = marshalJournal(journal)
	if err := removeJournal(); err != nil {
		return RecoveryResult{}, fmt.Errorf("identity recovery completed but journal removal failed; manual recovery required: %w", err)
	}

	return RecoveryResult{
		Identity: *loadedCEO, GitHubLogin: principal.Login, GitHubUserID: principal.ID,
		BackupRelativePath: backupRelative, OldHashPrefix: audit.OldHashPrefix,
		NewHashPrefix: audit.NewHashPrefix,
	}, nil
}

func validateRecoveryDependencies(deps RecoveryDependencies) error {
	if deps.IsTTY == nil || deps.MachineID == nil || deps.ReadSeed == nil || deps.Confirm == nil ||
		deps.Now == nil || deps.DeriveKey == nil || deps.LookPath == nil || deps.Run == nil || deps.SaveSession == nil ||
		deps.RemoveSession == nil {
		return fmt.Errorf("identity recovery: incomplete dependencies")
	}
	return nil
}

func authorizeGitHub(lookPath func(string) (string, error), run func(string, ...string) ([]byte, error)) (githubUser, githubRepository, error) {
	ghPath, err := trustedExecutable(lookPath, "gh")
	if err != nil {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: authenticated GitHub CLI principal is required")
	}
	response, err := run(ghPath, "api", "graphql", "--hostname", "github.com", "-f", "query="+githubAuthorizationQuery)
	if err != nil {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: authenticated GitHub CLI principal is required")
	}
	var authorization githubAuthorization
	if err := json.Unmarshal(response, &authorization); err != nil || len(authorization.Errors) != 0 ||
		authorization.Data.Viewer.DatabaseID <= 0 || strings.TrimSpace(authorization.Data.Viewer.Login) == "" {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: malformed GitHub authorization response")
	}
	repository := authorization.Data.Repository
	if repository.NameWithOwner != CanonicalGitHubRepo {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: GitHub response is not the canonical repository")
	}
	if repository.DatabaseID != CanonicalGitHubRepoID {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: canonical repository ID mismatch")
	}
	if repository.ViewerPermission != "ADMIN" {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: GitHub principal requires ADMIN permission")
	}
	if !strings.EqualFold(authorization.Data.Viewer.Login, CanonicalCEOLogin) ||
		authorization.Data.Viewer.DatabaseID != CanonicalCEOUserID {
		return githubUser{}, githubRepository{}, fmt.Errorf("identity recovery: GitHub principal is not the canonical CEO")
	}
	return githubUser{ID: authorization.Data.Viewer.DatabaseID, Login: authorization.Data.Viewer.Login},
		githubRepository{ID: repository.DatabaseID, FullName: repository.NameWithOwner}, nil
}

func rotateRegistry(original []byte, newHash string, key []byte, now time.Time) ([]byte, Identity, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(original, &node); err != nil {
		return nil, Identity{}, fmt.Errorf("identity recovery: parse registry document: %w", err)
	}
	if len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return nil, Identity{}, fmt.Errorf("identity recovery: registry document must be a mapping")
	}
	root := node.Content[0]
	identities := mappingValue(root, "identities")
	if identities == nil || identities.Kind != yaml.SequenceNode {
		return nil, Identity{}, fmt.Errorf("identity recovery: identities sequence is missing")
	}
	var selected *yaml.Node
	for _, item := range identities.Content {
		idNode := mappingValue(item, "id")
		statusNode := mappingValue(item, "status")
		if idNode != nil && statusNode != nil && normalize(idNode.Value) == CanonicalCEOID && normalize(statusNode.Value) == "active" {
			if selected != nil {
				return nil, Identity{}, fmt.Errorf("identity recovery: duplicate canonical CEO mapping")
			}
			selected = item
		}
	}
	if selected == nil {
		return nil, Identity{}, fmt.Errorf("identity recovery: canonical CEO mapping is missing")
	}
	keyHash := mappingValue(selected, "key_hash")
	if keyHash == nil {
		return nil, Identity{}, fmt.Errorf("identity recovery: canonical CEO key_hash is missing")
	}
	keyHash.Value = newHash
	keyHash.Tag = "!!str"

	signature := mappingValue(root, "signature")
	if signature == nil || signature.Kind != yaml.MappingNode {
		return nil, Identity{}, fmt.Errorf("identity recovery: signature mapping is missing")
	}
	for _, field := range []string{"algorithm", "signed_by", "signed_at", "value"} {
		if mappingValue(signature, field) == nil {
			return nil, Identity{}, fmt.Errorf("identity recovery: signature.%s is missing", field)
		}
	}
	if !strings.EqualFold(mappingValue(signature, "algorithm").Value, "HMAC-SHA256") &&
		!strings.EqualFold(mappingValue(signature, "algorithm").Value, "HMAC-SHA256-V2") {
		return nil, Identity{}, fmt.Errorf("identity recovery: signature algorithm is unsupported")
	}
	setExistingScalar(signature, "algorithm", "HMAC-SHA256-V2")
	setExistingScalar(signature, "signed_by", CanonicalCEOID)
	setExistingScalar(signature, "signed_at", now.Format(time.RFC3339Nano))
	setExistingScalar(signature, "value", strings.Repeat("0", sha256.Size*2))

	unsigned, err := yaml.Marshal(&node)
	if err != nil {
		return nil, Identity{}, fmt.Errorf("identity recovery: marshal unsigned registry: %w", err)
	}
	signedContent, err := preSignatureContent(unsigned)
	if err != nil {
		return nil, Identity{}, err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(signaturePayloadV2(signedContent, "HMAC-SHA256-V2", CanonicalCEOID, now.Format(time.RFC3339Nano)))
	setExistingScalar(signature, "value", hex.EncodeToString(mac.Sum(nil)))
	rotated, err := yaml.Marshal(&node)
	if err != nil {
		return nil, Identity{}, fmt.Errorf("identity recovery: marshal signed registry: %w", err)
	}

	var reg Registry
	if err := yaml.Unmarshal(rotated, &reg); err != nil {
		return nil, Identity{}, fmt.Errorf("identity recovery: validate rotated registry: %w", err)
	}
	ceo, err := CanonicalCEO(&reg)
	if err != nil {
		return nil, Identity{}, err
	}
	return rotated, *ceo, nil
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func setExistingScalar(mapping *yaml.Node, key, value string) {
	node := mappingValue(mapping, key)
	if node != nil {
		node.Value = value
		node.Tag = "!!str"
	}
}

func preSignatureContent(data []byte) ([]byte, error) {
	marker := []byte("\nsignature:")
	index := bytes.Index(data, marker)
	if index < 0 {
		return nil, fmt.Errorf("identity recovery: signature block not found")
	}
	return data[:index], nil
}

func createRecoveryBackup(repoRoot string, registryDirInfo os.FileInfo, original []byte, now time.Time) (string, error) {
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return "", err
	}
	dirRelative := ".ovav/registry/backups/identity-recovery"
	dir, err := ensureSecureDirectory(repoRoot, dirRelative)
	if err != nil {
		return "", fmt.Errorf("identity recovery: create backup directory: %w", err)
	}
	if err := verifyOwnedDirectory(dir); err != nil {
		return "", fmt.Errorf("identity recovery: backup directory ownership: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("identity recovery: secure backup directory: %w", err)
	}
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return "", err
	}
	digest := sha256.Sum256(original)
	name := fmt.Sprintf("%s-%s.yaml", now.UTC().Format("20060102T150405.000000000Z"), hex.EncodeToString(digest[:]))
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("identity recovery: create backup: %w", err)
	}
	writeErr := writeSyncClose(f, original)
	if writeErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("identity recovery: persist backup: %w", writeErr)
	}
	backup, _, err := readRegularNoSymlink(path)
	if err != nil || sha256.Sum256(backup) != digest {
		_ = os.Remove(path)
		return "", fmt.Errorf("identity recovery: backup digest verification failed")
	}
	if err := syncDirectory(dir); err != nil {
		return "", fmt.Errorf("identity recovery: sync backup directory: %w", err)
	}
	relative, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "", fmt.Errorf("identity recovery: resolve backup path: %w", err)
	}
	return filepath.ToSlash(relative), nil
}

func atomicReplaceChecked(path string, data, expected []byte, expectedInfo os.FileInfo) (bool, error) {
	return atomicReplaceCheckedWithSync(path, data, expected, expectedInfo, nil)
}

func atomicReplaceCheckedWithSync(path string, data, expected []byte, expectedInfo os.FileInfo, syncFn func(string) error) (bool, error) {
	return secureAtomicReplace(path, data, expected, expectedInfo, false, syncFn)
}

func atomicReplaceCAS(path string, data, expected []byte, expectedInfo os.FileInfo, unconditional bool) error {
	_, err := secureAtomicReplace(path, data, expected, expectedInfo, unconditional, nil)
	return err
}

func restoreRegistryChecked(repoRoot string, registryDirInfo os.FileInfo, path string, rotated, original []byte) error {
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return err
	}
	current, _, err := readRegularNoSymlink(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, rotated) {
		return fmt.Errorf("manual recovery required: registry changed after recovery write; current bytes preserved")
	}
	_, currentInfo, err := readRegularNoSymlink(path)
	if err != nil {
		return err
	}
	_, err = secureAtomicReplace(path, original, rotated, currentInfo, false, nil)
	return err
}

func appendRecoveryAudit(repoRoot string, entry RecoveryAuditEntry) error {
	registryDir := filepath.Join(repoRoot, ".ovav", "registry")
	info, err := os.Lstat(registryDir)
	if err != nil {
		return err
	}
	_, err = appendRecoveryAuditChecked(repoRoot, info, entry, nil)
	return err
}

func appendRecoveryAuditChecked(repoRoot string, registryDirInfo os.FileInfo, entry RecoveryAuditEntry, syncFn func(string) error) (bool, error) {
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return false, err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	path := filepath.Join(repoRoot, filepath.FromSlash(AuditRelativePath))
	var before []byte
	var info os.FileInfo
	info, statErr := os.Lstat(path)
	if errors.Is(statErr, os.ErrNotExist) {
		before = nil
	} else if statErr != nil {
		return false, statErr
	} else if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("audit path is not a regular file")
	} else {
		before, info, err = readRegularNoSymlink(path)
		if err != nil {
			return false, err
		}
	}
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return false, err
	}
	currentInfo, currentErr := os.Lstat(path)
	if info == nil {
		if currentErr == nil {
			return false, fmt.Errorf("audit changed during recovery")
		}
		if !errors.Is(currentErr, os.ErrNotExist) {
			return false, currentErr
		}
	} else if currentErr != nil || !os.SameFile(info, currentInfo) {
		return false, fmt.Errorf("audit changed during recovery")
	}
	combined := make([]byte, 0, len(before)+len(data))
	combined = append(combined, before...)
	combined = append(combined, data...)
	if info == nil {
		return secureAtomicCreate(path, combined, syncFn)
	}
	return secureAtomicReplace(path, combined, before, info, false, syncFn)
}

func acquireRecoveryLock(repoRoot string) (*os.File, error) {
	registryDir := filepath.Join(repoRoot, ".ovav", "registry")
	registryInfo, err := os.Lstat(registryDir)
	if err != nil || !registryInfo.IsDir() || registryInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("identity recovery: lock directory is not trusted")
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(RecoveryLockRelativePath))
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("identity recovery: lock path is a symlink")
		}
		return nil, fmt.Errorf("identity recovery: transaction is locked")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("identity recovery: inspect lock: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("identity recovery: transaction is locked")
		}
		return nil, fmt.Errorf("identity recovery: acquire lock: %w", err)
	}
	if _, err := fmt.Fprintf(f, "%d\n", os.Getpid()); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("identity recovery: initialize lock: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("identity recovery: sync lock: %w", err)
	}
	if err := verifyRegistryDirectory(repoRoot, registryInfo); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func releaseRecoveryLock(lock *os.File) {
	if lock == nil {
		return
	}
	path := lock.Name()
	openedInfo, statErr := lock.Stat()
	_ = lock.Close()
	pathInfo, lstatErr := os.Lstat(path)
	if statErr == nil && lstatErr == nil && pathInfo.Mode()&os.ModeSymlink == 0 && os.SameFile(openedInfo, pathInfo) {
		_ = os.Remove(path)
	}
}

func readRegularNoSymlink(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("%s is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	openedInfo, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		return nil, nil, fmt.Errorf("%s changed while opening", path)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	afterInfo, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, afterInfo) {
		return nil, nil, fmt.Errorf("%s changed while reading", path)
	}
	return data, info, nil
}

func ensureSecureDirectory(repoRoot, relative string) (string, error) {
	current := repoRoot
	createdBoundary := filepath.Join(repoRoot, ".ovav", "registry", "backups")
	for _, component := range strings.Split(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", fmt.Errorf("%s is not a real directory", current)
		}
		if current == createdBoundary || strings.HasPrefix(current, createdBoundary+string(filepath.Separator)) {
			if err := verifyOwnedDirectory(current); err != nil {
				return "", err
			}
			if err := os.Chmod(current, 0o700); err != nil {
				return "", err
			}
		}
	}
	return current, nil
}

func verifyRegistryDirectory(repoRoot string, expected os.FileInfo) error {
	if err := rejectSymlinkParents(repoRoot, filepath.Join(".ovav", "registry")); err != nil {
		return fmt.Errorf("identity recovery: registry directory changed: %w", err)
	}
	current, err := os.Lstat(filepath.Join(repoRoot, ".ovav", "registry"))
	if err != nil || !os.SameFile(expected, current) {
		return fmt.Errorf("identity recovery: registry directory identity changed; operation stopped")
	}
	if err := verifyTrustedDirectoryInfo(current, false); err != nil {
		return fmt.Errorf("identity recovery: insecure registry directory: %w", err)
	}
	return nil
}

func trustedExecutable(lookPath func(string) (string, error), name string) (string, error) {
	path, err := lookPath(name)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 || verifyCurrentUserOwnership(info) != nil {
		return "", fmt.Errorf("untrusted executable")
	}
	for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
		parentInfo, statErr := os.Lstat(parent)
		if statErr != nil || verifyTrustedDirectoryInfo(parentInfo, parent == string(filepath.Separator)) != nil {
			return "", fmt.Errorf("untrusted executable parent %s", parent)
		}
		if parent == string(filepath.Separator) {
			break
		}
	}
	return path, nil
}

func verifyTrustedDirectoryInfo(info os.FileInfo, root bool) error {
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || verifyCurrentUserOwnership(info) != nil {
		return fmt.Errorf("directory is not trusted")
	}
	if info.Mode().Perm()&0o022 != 0 {
		if !(root && info.Mode()&os.ModeSticky != 0) {
			return fmt.Errorf("directory is group/world writable")
		}
	}
	return nil
}

func verifyOwnedDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s is not a real directory", path)
	}
	return verifyCurrentUserOwnership(info)
}

func marshalJournal(journal recoveryJournal) ([]byte, error) {
	data, err := json.Marshal(journal)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeRecoveryJournal(repoRoot string, registryDirInfo os.FileInfo, journal recoveryJournal, expected []byte) error {
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return err
	}
	path := filepath.Join(repoRoot, RecoveryJournalRelativePath)
	if expected == nil {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("recovery journal already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else {
		current, _, err := readRegularNoSymlink(path)
		if err != nil || !bytes.Equal(current, expected) {
			return fmt.Errorf("recovery journal changed concurrently")
		}
	}
	data, err := marshalJournal(journal)
	if err != nil {
		return err
	}
	if expected == nil {
		_, err = secureAtomicCreate(path, data, nil)
		return err
	}
	_, info, readErr := readRegularNoSymlink(path)
	if readErr != nil {
		return readErr
	}
	_, err = secureAtomicReplace(path, data, expected, info, false, nil)
	return err
}

func removeRecoveryJournal(repoRoot string, registryDirInfo os.FileInfo, expected []byte) error {
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return err
	}
	path := filepath.Join(repoRoot, RecoveryJournalRelativePath)
	current, info, err := readRegularNoSymlink(path)
	if err != nil || !bytes.Equal(current, expected) {
		return fmt.Errorf("manual recovery required: recovery journal changed concurrently")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if after, err := os.Lstat(path); err == nil && os.SameFile(info, after) {
		return fmt.Errorf("recovery journal removal did not complete")
	}
	return syncDirectory(filepath.Dir(path))
}

func removeRecoveryBackup(repoRoot string, registryDirInfo os.FileInfo, relative string, expected []byte) error {
	if err := verifyRegistryDirectory(repoRoot, registryDirInfo); err != nil {
		return err
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(relative))
	cleanRoot := filepath.Join(repoRoot, ".ovav", "registry", "backups", "identity-recovery") + string(filepath.Separator)
	if !strings.HasPrefix(path, cleanRoot) {
		return fmt.Errorf("identity recovery: backup path escaped recovery directory")
	}
	current, info, err := readRegularNoSymlink(path)
	if err != nil || !bytes.Equal(current, expected) {
		return fmt.Errorf("identity recovery: backup changed; preserving it for manual review")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if after, err := os.Lstat(path); err == nil && os.SameFile(info, after) {
		return fmt.Errorf("identity recovery: backup removal did not complete")
	}
	return syncDirectory(filepath.Dir(path))
}

func digestHex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func rollbackRegistryAndJournal(repoRoot string, registryDirInfo os.FileInfo, registryPath string, rotated, original, journal []byte, cause error) (RecoveryResult, error) {
	rollbackErr := restoreRegistryChecked(repoRoot, registryDirInfo, registryPath, rotated, original)
	if rollbackErr == nil {
		rollbackErr = removeRecoveryJournal(repoRoot, registryDirInfo, journal)
	}
	return RecoveryResult{}, recoveryFailure("post-write validation", cause, rollbackErr)
}

func recoveryFailureNoWrite(stage string, cause, cleanupErr error) error {
	if cleanupErr != nil {
		return fmt.Errorf("identity recovery: %s failed before write: %v; journal cleanup failed: %w", stage, cause, cleanupErr)
	}
	return fmt.Errorf("identity recovery: %s failed before write: %w", stage, cause)
}

func combinedRecoveryFailure(cause, registryErr, sessionErr error) error {
	if registryErr != nil || sessionErr != nil {
		return fmt.Errorf("identity recovery: post-write failure: %v; registry rollback: %v; session rollback: %v", cause, registryErr, sessionErr)
	}
	return fmt.Errorf("identity recovery: post-write failure and state restored: %w", cause)
}

func rejectSymlinkParents(repoRoot, relativeDir string) error {
	rootInfo, err := os.Lstat(repoRoot)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("repository root is not a real directory")
	}
	current := repoRoot
	for _, component := range strings.Split(relativeDir, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("%s is not a real directory", current)
		}
	}
	return nil
}

func writeSyncClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func recoveryFailure(stage string, cause, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("identity recovery: %s failed: %v; rollback failed: %w", stage, cause, rollbackErr)
	}
	return fmt.Errorf("identity recovery: %s failed and registry was restored: %w", stage, cause)
}

func sameCanonicalIdentity(left, right Identity) bool {
	left.KeyHash = ""
	right.KeyHash = ""
	return left.ID == right.ID && left.Name == right.Name && left.Email == right.Email &&
		left.Role == right.Role && left.Level == right.Level && left.Status == right.Status &&
		strings.Join(left.Permissions, "\x00") == strings.Join(right.Permissions, "\x00")
}

func hasNormalizedPermission(permissions []string, wanted string) bool {
	for _, permission := range permissions {
		if normalize(permission) == wanted {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func hashPrefix(value string) string {
	if len(value) <= maxHashPrefix {
		return value
	}
	return value[:maxHashPrefix]
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
