package validators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/permissions"
	"github.com/ovav/ovav/internal/security"
)

// F1Architecture validates F1 Architecture Integrity through the production
// permission and bootstrap APIs, plus canonical Rego policy behavior.
type F1Architecture struct{}

func NewF1Architecture() *F1Architecture { return &F1Architecture{} }

func (v *F1Architecture) ID() string   { return "f1_architecture" }
func (v *F1Architecture) Name() string { return "F1 Architecture Integrity" }
func (v *F1Architecture) Description() string {
	return "Validates F1 architecture behavior: permission authority, simulation, Rego, and bootstrap"
}
func (v *F1Architecture) Weight() int { return 7 }

func (v *F1Architecture) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	partialBootstrap := !security.BootstrapTrustAnchorsConfigured()

	if err := ctx.Err(); err != nil {
		issues = append(issues, "ERROR: validation context unavailable")
	}

	permissionAuthorityPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if !permissions.VerifyPermissionAuthorityV3(permissionAuthorityPath) {
		issues = append(issues, "CRITICAL: permission_authority.json schema_version v3 integrity verification failed")
	}

	if err := permissions.Simulate(); err != nil {
		issues = append(issues, fmt.Sprintf("ERROR: canonical permission simulation failed: %v", err))
	}

	regoDir, regoCount := canonicalRegoDirectory(root)
	if regoCount < 3 {
		issues = append(issues, fmt.Sprintf("ERROR: Only %d Rego policy files found (expected >= 3)", regoCount))
	} else if err := verifyRegoBehavior(regoDir); err != nil {
		issues = append(issues, "ERROR: Rego behavioral verification failed: "+err.Error())
	}

	if partialBootstrap {
		issues = append(issues, "INTENTIONALLY_GATED/PARTIAL: startup enforcement requires immutable permission-authority and runtime hashes injected at build time")
	} else if err := verifyBootstrapBehavior(root, permissionAuthorityPath); err != nil {
		issues = append(issues, "ERROR: Go F0 bootstrap behavioral verification failed: "+err.Error())
	}

	if _, err := os.Stat(filepath.Join(root, "docs", "research", "F1_EAL7_GUIDANCE.md")); err != nil {
		issues = append(issues, "WARNING: docs/research/F1_EAL7_GUIDANCE.md not found")
	}

	if len(issues) > 0 {
		hasCritical := false
		for _, issue := range issues {
			if strings.HasPrefix(issue, "CRITICAL:") {
				hasCritical = true
				break
			}
		}
		message := fmt.Sprintf("FAIL F1 architecture integrity — %d issue(s)", len(issues))
		if hasCritical {
			message += " including critical"
		}
		status := "fail"
		if partialBootstrap && len(issues) == 1 {
			status = "warn"
			message = "INTENTIONALLY_GATED/PARTIAL F1 architecture integrity — immutable startup trust anchors are not configured"
		}
		return Result{
			ID: v.ID(), Name: v.Name(), Status: status, Weight: v.Weight(),
			Message: message, Issues: issues, Duration: time.Since(start),
		}
	}

	return Result{
		ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
		Message:  fmt.Sprintf("PASS F1 architecture integrity — canonical APIs and %d Rego policies verified behaviorally", regoCount),
		Duration: time.Since(start),
	}
}

func canonicalRegoDirectory(root string) (string, int) {
	bestDir := ""
	bestCount := 0
	for _, rel := range []string{filepath.Join(".ovav", "policy", "rego"), filepath.Join(".ovav", "registry", "rego_policies")} {
		dir := filepath.Join(root, rel)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		count := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".rego") {
				count++
			}
		}
		if count > bestCount {
			bestDir, bestCount = dir, count
		}
	}
	return bestDir, bestCount
}

func verifyRegoBehavior(regoDir string) error {
	engine := permissions.NewRegoEngine(regoDir)
	if err := engine.LoadPolicies(); err != nil {
		return fmt.Errorf("load policies: %w", err)
	}
	allowed := engine.Evaluate("read", map[string]interface{}{
		"action":          "read",
		"operator":        "andres",
		"scope":           "repo_local",
		"bootstrap_valid": true,
		"explicit_grant":  true,
	})
	if !allowed.Allowed {
		return fmt.Errorf("explicit read grant was denied: %s", allowed.Reason)
	}
	denied := engine.Evaluate("bash", map[string]interface{}{
		"action":          "bash",
		"command":         "sudo id",
		"operator":        "andres",
		"scope":           "repo_local",
		"bootstrap_valid": true,
		"explicit_grant":  true,
	})
	if denied.Allowed {
		return fmt.Errorf("dangerous bash command was allowed")
	}
	return nil
}

func verifyBootstrapBehavior(root, authorityPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve validator executable: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("resolve validator executable links: %w", err)
	}
	checksum, err := fileSHA256(executable)
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "ovav-f1-bootstrap-")
	if err != nil {
		return fmt.Errorf("create bootstrap verification workspace: %w", err)
	}
	defer os.RemoveAll(tempDir)
	checksumPath := filepath.Join(tempDir, "runtime.sha256")
	if err := os.WriteFile(checksumPath, []byte(checksum+"\n"), 0o600); err != nil {
		return fmt.Errorf("write bootstrap checksum fixture: %w", err)
	}

	notRequired := false
	return security.VerifyBootstrapWithConfig(security.BootstrapConfig{
		Root:                      root,
		LicensePath:               filepath.Join(tempDir, "absent-license.json"),
		VaultKeyPath:              filepath.Join(tempDir, "absent-vault-key"),
		PermissionAuthorityPath:   authorityPath,
		RuntimePath:               executable,
		RuntimeChecksumPath:       checksumPath,
		PermissionAuthoritySHA256: security.BuildPermissionAuthoritySHA256,
		RuntimeSHA256:             security.BuildRuntimeSHA256,
		RequireLicense:            &notRequired,
		RequireVaultKey:           &notRequired,
	})
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open validator executable: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash validator executable: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

var _ Validator = (*F1Architecture)(nil)
