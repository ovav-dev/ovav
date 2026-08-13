// Package security implements the read-only OVAV F0 bootstrap verifier.
package security

import (
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
)

var (
	ErrBootstrapLicense        = errors.New("bootstrap: license verification failed")
	ErrBootstrapVaultKey       = errors.New("bootstrap: vault_key verification failed")
	ErrBootstrapPermissionAuth = errors.New("bootstrap: permission_authority verification failed")
	ErrBootstrapRuntime        = errors.New("bootstrap: runtime verification failed")
	ErrBootstrapTrustAnchor    = errors.New("bootstrap: immutable trust anchor unavailable")
)

// These values must be injected with -ldflags for a startup-enforcing build.
// Empty development builds remain intentionally gated rather than self-trusting.
var (
	BuildPermissionAuthoritySHA256 string
	BuildRuntimeSHA256             string
)

// BootstrapConfig supplies source-local artifact paths. Nil requirement fields
// use the policy authority's bootstrap and vault posture.
type BootstrapConfig struct {
	Root                      string
	LicensePath               string
	VaultKeyPath              string
	PermissionAuthorityPath   string
	RuntimePath               string
	RuntimeChecksumPath       string
	PermissionAuthoritySHA256 string
	RuntimeSHA256             string
	RequireLicense            *bool
	RequireVaultKey           *bool
	Now                       func() time.Time
}

type permissionAuthorityV3 struct {
	SchemaVersion      string          `json:"schema_version"`
	Authority          json.RawMessage `json:"authority"`
	Governor           json.RawMessage `json:"governor"`
	BootstrapRequired  bool            `json:"bootstrap_required"`
	MaterializedTarget []string        `json:"materialized_targets"`
	ResourcePolicies   struct {
		SecretsVault struct {
			RequireUnlock bool `json:"require_unlock"`
		} `json:"secrets_vault"`
	} `json:"resource_policies"`
	Conditions struct {
		RequireBootstrap bool `json:"require_bootstrap"`
	} `json:"conditions"`
}

type licensePosture struct {
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	Key           json.RawMessage `json:"key"`
	LicenseKey    json.RawMessage `json:"license_key"`
	KeyHash       string          `json:"key_hash"`
	ExpiresAt     string          `json:"expires_at"`
}

// VerifyBootstrap verifies license, vault, permission authority, and runtime in
// order using source-local defaults. It never writes or performs network I/O.
func VerifyBootstrap() error {
	return VerifyBootstrapWithConfig(BootstrapConfig{})
}

// VerifyBootstrapWithConfig performs the same public chain with testable paths.
func VerifyBootstrapWithConfig(config BootstrapConfig) error {
	config = defaultBootstrapConfig(config)
	if !validSHA256(config.PermissionAuthoritySHA256) || !validSHA256(config.RuntimeSHA256) {
		return ErrBootstrapTrustAnchor
	}
	authority, err := readPermissionAuthority(config.PermissionAuthorityPath, config.PermissionAuthoritySHA256)
	if err != nil {
		return err
	}

	requireLicense := authority.BootstrapRequired || authority.Conditions.RequireBootstrap
	if config.RequireLicense != nil {
		requireLicense = *config.RequireLicense
	}
	if err := verifyLicense(config, requireLicense); err != nil {
		return err
	}

	requireVault := authority.ResourcePolicies.SecretsVault.RequireUnlock
	if config.RequireVaultKey != nil {
		requireVault = *config.RequireVaultKey
	}
	if err := verifyVaultKey(config, requireVault); err != nil {
		return err
	}
	if err := verifyPermissionAuthority(authority); err != nil {
		return err
	}
	return verifyRuntime(config)
}

func defaultBootstrapConfig(config BootstrapConfig) BootstrapConfig {
	if config.Root == "" {
		config.Root = repoRootFromWorkingDirectory()
	}
	if config.LicensePath == "" {
		config.LicensePath = filepath.Join(config.Root, ".ovav", "license.json")
	}
	if config.VaultKeyPath == "" {
		config.VaultKeyPath = filepath.Join(config.Root, ".ovav", "vault", "vault_key_export")
	}
	if config.PermissionAuthorityPath == "" {
		config.PermissionAuthorityPath = filepath.Join(config.Root, ".ovav", "policy", "permission_authority.json")
	}
	if config.RuntimePath == "" {
		config.RuntimePath = filepath.Join(config.Root, "go-runtime", "build", "ovav")
	}
	if config.RuntimeChecksumPath == "" {
		config.RuntimeChecksumPath = config.RuntimePath + ".sha256"
	}
	if config.PermissionAuthoritySHA256 == "" {
		config.PermissionAuthoritySHA256 = BuildPermissionAuthoritySHA256
	}
	if config.RuntimeSHA256 == "" {
		config.RuntimeSHA256 = BuildRuntimeSHA256
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return config
}

func repoRootFromWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if filepath.Base(cwd) == "go-runtime" {
		return filepath.Dir(cwd)
	}
	return cwd
}

func verifyLicense(config BootstrapConfig, required bool) error {
	data, err := os.ReadFile(config.LicensePath)
	if errors.Is(err, os.ErrNotExist) && !required {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: artifact unavailable", ErrBootstrapLicense)
	}

	var posture licensePosture
	if err := decodeStrictJSON(data, &posture); err != nil {
		return fmt.Errorf("%w: artifact is invalid", ErrBootstrapLicense)
	}
	if posture.SchemaVersion != "ovav.license.v1" || strings.ToLower(posture.Status) != "active" {
		return fmt.Errorf("%w: inactive or unsupported posture", ErrBootstrapLicense)
	}
	if rawPresent(posture.Key) || rawPresent(posture.LicenseKey) || !validSHA256(posture.KeyHash) {
		return fmt.Errorf("%w: key material posture is unsafe", ErrBootstrapLicense)
	}
	expiresAt, err := time.Parse(time.RFC3339, posture.ExpiresAt)
	if err != nil || !expiresAt.After(config.Now().UTC()) {
		return fmt.Errorf("%w: posture expired or malformed", ErrBootstrapLicense)
	}
	return nil
}

func verifyVaultKey(config BootstrapConfig, required bool) error {
	data, err := os.ReadFile(config.VaultKeyPath)
	if errors.Is(err, os.ErrNotExist) && !required {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: artifact unavailable", ErrBootstrapVaultKey)
	}
	info, err := os.Stat(config.VaultKeyPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: insecure artifact posture", ErrBootstrapVaultKey)
	}

	key := strings.TrimSpace(string(data))
	decoded, hexErr := hex.DecodeString(key)
	valid := hexErr == nil && len(decoded) == sha256.Size
	if !valid {
		valid = len(data) == sha256.Size
	}
	if !valid {
		return fmt.Errorf("%w: artifact format invalid", ErrBootstrapVaultKey)
	}
	return nil
}

func readPermissionAuthority(path, expectedHash string) (permissionAuthorityV3, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return permissionAuthorityV3{}, fmt.Errorf("%w: artifact unavailable", ErrBootstrapPermissionAuth)
	}
	if !strings.EqualFold(hashBytes(data), expectedHash) {
		return permissionAuthorityV3{}, fmt.Errorf("%w: immutable authority hash mismatch", ErrBootstrapPermissionAuth)
	}
	var authority permissionAuthorityV3
	if err := json.Unmarshal(data, &authority); err != nil {
		return permissionAuthorityV3{}, fmt.Errorf("%w: artifact is invalid", ErrBootstrapPermissionAuth)
	}
	return authority, nil
}

func verifyPermissionAuthority(authority permissionAuthorityV3) error {
	if authority.SchemaVersion != "ovav.permission_authority.v3" ||
		!rawNonEmpty(authority.Authority) ||
		!rawNonEmpty(authority.Governor) ||
		len(authority.MaterializedTarget) == 0 {
		return fmt.Errorf("%w: schema v3 integrity check failed", ErrBootstrapPermissionAuth)
	}
	seen := make(map[string]struct{}, len(authority.MaterializedTarget))
	for _, target := range authority.MaterializedTarget {
		clean := filepath.Clean(target)
		if target == "" || filepath.IsAbs(target) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%w: unsafe materialized target", ErrBootstrapPermissionAuth)
		}
		if _, exists := seen[clean]; exists {
			return fmt.Errorf("%w: duplicate materialized target", ErrBootstrapPermissionAuth)
		}
		seen[clean] = struct{}{}
	}
	return nil
}

func verifyRuntime(config BootstrapConfig) error {
	info, err := os.Stat(config.RuntimePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%w: executable unavailable or not executable", ErrBootstrapRuntime)
	}
	expectedData, err := os.ReadFile(config.RuntimeChecksumPath)
	if err != nil {
		return fmt.Errorf("%w: checksum unavailable", ErrBootstrapRuntime)
	}
	fields := strings.Fields(string(expectedData))
	if len(fields) == 0 || !validSHA256(fields[0]) {
		return fmt.Errorf("%w: checksum format invalid", ErrBootstrapRuntime)
	}

	file, err := os.Open(config.RuntimePath)
	if err != nil {
		return fmt.Errorf("%w: executable unreadable", ErrBootstrapRuntime)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("%w: checksum could not be computed", ErrBootstrapRuntime)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(fields[0], config.RuntimeSHA256) || !strings.EqualFold(actual, config.RuntimeSHA256) {
		return fmt.Errorf("%w: checksum mismatch", ErrBootstrapRuntime)
	}
	return nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// BootstrapTrustAnchorsConfigured reports whether this build can make a full
// startup-enforcement claim without trusting mutable source-local metadata.
func BootstrapTrustAnchorsConfigured() bool {
	return validSHA256(BuildPermissionAuthoritySHA256) && validSHA256(BuildRuntimeSHA256)
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func rawPresent(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func rawNonEmpty(raw json.RawMessage) bool {
	if !rawPresent(raw) {
		return false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case map[string]any:
		return len(typed) > 0
	default:
		return false
	}
}
