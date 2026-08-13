package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVerifyBootstrapWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(t *testing.T, fixture bootstrapFixture)
		wantErr   error
		forbidden string
	}{
		{name: "valid"},
		{
			name: "required license absent",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustRemove(t, f.config.LicensePath)
			},
			wantErr: ErrBootstrapLicense,
		},
		{
			name: "license containing raw key rejected",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustWrite(t, f.config.LicensePath, `{"schema_version":"ovav.license.v1","status":"active","key":"do-not-print","key_hash":"`+strings.Repeat("a", 64)+`","expires_at":"2099-01-01T00:00:00Z"}`, 0o600)
			},
			wantErr:   ErrBootstrapLicense,
			forbidden: "do-not-print",
		},
		{
			name: "vault absent",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustRemove(t, f.config.VaultKeyPath)
			},
			wantErr: ErrBootstrapVaultKey,
		},
		{
			name: "vault invalid format",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustWrite(t, f.config.VaultKeyPath, "short-secret", 0o600)
			},
			wantErr:   ErrBootstrapVaultKey,
			forbidden: "short-secret",
		},
		{
			name: "permission authority absent",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustRemove(t, f.config.PermissionAuthorityPath)
			},
			wantErr: ErrBootstrapPermissionAuth,
		},
		{
			name: "permission authority wrong schema",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustWrite(t, f.config.PermissionAuthorityPath, `{"schema_version":"ovav.permission_authority.v2"}`, 0o600)
			},
			wantErr: ErrBootstrapPermissionAuth,
		},
		{
			name: "permission authority hash mismatch",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustWrite(t, f.config.PermissionAuthorityPath, `{"schema_version":"ovav.permission_authority.v3","authority":"tampered"}`, 0o600)
			},
			wantErr: ErrBootstrapPermissionAuth,
		},
		{
			name: "runtime checksum mismatch",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustWrite(t, f.config.RuntimeChecksumPath, strings.Repeat("0", 64), 0o600)
			},
			wantErr: ErrBootstrapRuntime,
		},
		{
			name: "runtime absent",
			mutate: func(t *testing.T, f bootstrapFixture) {
				t.Helper()
				mustRemove(t, f.config.RuntimePath)
			},
			wantErr: ErrBootstrapRuntime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newBootstrapFixture(t)
			if tt.mutate != nil {
				tt.mutate(t, fixture)
			}

			err := VerifyBootstrapWithConfig(fixture.config)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("VerifyBootstrapWithConfig() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.forbidden != "" && strings.Contains(err.Error(), tt.forbidden) {
				t.Fatalf("error exposed secret material: %v", err)
			}
		})
	}
}

func TestVerifyBootstrapRequiresImmutableTrustAnchors(t *testing.T) {
	fixture := newBootstrapFixture(t)
	fixture.config.PermissionAuthoritySHA256 = ""
	if err := VerifyBootstrapWithConfig(fixture.config); !errors.Is(err, ErrBootstrapTrustAnchor) {
		t.Fatalf("error = %v, want ErrBootstrapTrustAnchor", err)
	}
}

func TestVerifyBootstrapOptionalArtifacts(t *testing.T) {
	fixture := newBootstrapFixture(t)
	optional := false
	fixture.config.RequireLicense = &optional
	fixture.config.RequireVaultKey = &optional
	mustRemove(t, fixture.config.LicensePath)
	mustRemove(t, fixture.config.VaultKeyPath)

	if err := VerifyBootstrapWithConfig(fixture.config); err != nil {
		t.Fatalf("optional artifacts should be accepted when absent: %v", err)
	}

	mustWrite(t, fixture.config.LicensePath, "invalid-present-artifact", 0o600)
	if err := VerifyBootstrapWithConfig(fixture.config); !errors.Is(err, ErrBootstrapLicense) {
		t.Fatalf("present invalid optional license error = %v, want ErrBootstrapLicense", err)
	}
}

type bootstrapFixture struct {
	config BootstrapConfig
}

func newBootstrapFixture(t *testing.T) bootstrapFixture {
	t.Helper()
	root := t.TempDir()
	licensePath := filepath.Join(root, "license.json")
	vaultPath := filepath.Join(root, "vault_key_export")
	policyPath := filepath.Join(root, "permission_authority.json")
	runtimePath := filepath.Join(root, "ovav")
	checksumPath := filepath.Join(root, "ovav.sha256")
	required := true

	mustWrite(t, licensePath, `{"schema_version":"ovav.license.v1","status":"active","key_hash":"`+strings.Repeat("a", 64)+`","expires_at":"2099-01-01T00:00:00Z"}`, 0o600)
	mustWrite(t, vaultPath, strings.Repeat("b", 64), 0o600)
	policyData := `{"schema_version":"ovav.permission_authority.v3","authority":"OVAV SYSTEMS","governor":"Thavren","bootstrap_required":true,"materialized_targets":["opencode.json"],"resource_policies":{"secrets_vault":{"require_unlock":true}}}`
	mustWrite(t, policyPath, policyData, 0o600)
	runtimeData := []byte("runtime-binary")
	mustWrite(t, runtimePath, string(runtimeData), 0o700)
	sum := sha256.Sum256(runtimeData)
	mustWrite(t, checksumPath, hex.EncodeToString(sum[:])+"  ovav\n", 0o600)
	policySum := sha256.Sum256([]byte(policyData))

	return bootstrapFixture{config: BootstrapConfig{
		Root:                      root,
		LicensePath:               licensePath,
		VaultKeyPath:              vaultPath,
		PermissionAuthorityPath:   policyPath,
		RuntimePath:               runtimePath,
		RuntimeChecksumPath:       checksumPath,
		PermissionAuthoritySHA256: hex.EncodeToString(policySum[:]),
		RuntimeSHA256:             hex.EncodeToString(sum[:]),
		RequireLicense:            &required,
		RequireVaultKey:           &required,
		Now:                       func() time.Time { return time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC) },
	}}
}

func mustWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func mustRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove fixture: %v", err)
	}
}
