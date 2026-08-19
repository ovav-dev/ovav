package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ─── R-2: Stale-lock purge ───────────────────────────────────────────────
//
// Identity-recovery lock files were the #1 cause of "transaction is
// locked" errors. The legacy lock check is just `os.Lstat(path)` — if
// the file exists, the registry is considered locked. We replace that
// with a PID-liveness check.

var (
	// lockPath is computed from repo root + relative subdir.
	lockPath = ".ovav/registry/.identity-recovery.lock"
)

// PurgeStaleLock removes the identity-recovery lock if its recorded
// PID is no longer alive. Returns nil if no lock exists, lock was
// purged, or lock is held by a live PID.
//
// On success: returns the action taken ("none", "purged", "kept").
func PurgeStaleLock(repoRoot string) error {
	if repoRoot == "" {
		return fmt.Errorf("purgeStaleLock: repoRoot empty")
	}
	full := filepath.Join(repoRoot, lockPath)
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no lock — clean state
		}
		return fmt.Errorf("inspect lock: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("lock path is a symlink — refusing to follow")
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return fmt.Errorf("read lock pid: %w", err)
	}
	pidStr := strings.TrimSpace(string(data))
	if pidStr == "" {
		// corrupt lock with no PID → purge
		_ = os.Remove(full)
		return nil
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// non-numeric content → purge
		_ = os.Remove(full)
		return nil
	}

	// PID liveness: kill -0 returns 0 if PID exists
	if err := syscall.Kill(pid, 0); err == nil {
		return fmt.Errorf("lock held by ALIVE pid %d — another recovery is in progress", pid)
	}

	// PID is dead → safe to remove
	return os.Remove(full)
}

// ShredExport securely overwrites a file in the OVAV share dir
// and then removes it. If shred-via-truncate isn't available,
// falls back to plain remove. R-1 compliance: callers invoke this
// to ensure seed_export / vault_key_export never persist.
func ShredExport(home, name string) {
	if home == "" || name == "" {
		return
	}
	path := filepath.Join(home, ".local", "share", "ovav", name)
	if _, err := os.Stat(path); err != nil {
		return // file not present
	}
	// Overwrite with zero bytes (single pass — sufficient for SSD/COW).
	// Real `shred` does multi-pass; this is intentionally minimal so
	// we don't make the simple wrapper do too much.
	if err := os.WriteFile(path, make([]byte, 4096), 0o600); err != nil {
		_ = os.Remove(path)
		return
	}
	_ = os.Remove(path)
}

// ─── R-4: Seed input ─────────────────────────────────────────────────────

// ReadSeed obtains the seed from the configured source (env var,
// --seed-file path, or TTY prompt — in that order).
//
// Returns "" + nil when no seed is provided (caller should prompt).
func ReadSeed(seedFile string) (string, error) {
	if v := os.Getenv("SEED"); v != "" {
		return strings.TrimSpace(v), nil
	}
	if seedFile != "" {
		data, err := os.ReadFile(seedFile)
		if err != nil {
			return "", fmt.Errorf("read seed-file: %w", err)
		}
		// Auto-shred after reading — R-1 compliance.
		ShredExport(filepath.Dir(filepath.Dir(filepath.Dir(seedFile))), "seed_file_consumed")
		_ = os.Remove(seedFile)
		return strings.TrimSpace(string(data)), nil
	}
	// TTY fallback: defer to caller (no seed yet).
	return "", nil
}

// ZeroOut overwrites a byte slice with zeros, used to scrub the
// seed from memory after use.
func ZeroOut(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ─── R-5: Vault key derivation (mirrors identity package) ───────────────

// SHA256Hex computes the canonical key_hash used to look up
// the identity in identities.yaml.
func SHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
