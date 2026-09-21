//go:build unix

package identity

import (
	"bytes"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
)

func TestAppendRecoveryAuditDoesNotLeavePartialRecord(t *testing.T) {
	const helperEnv = "OVAV_TEST_PARTIAL_AUDIT_ROOT"
	if root := os.Getenv(helperEnv); root != "" {
		before, err := os.ReadFile(filepath.Join(root, AuditRelativePath))
		if err != nil {
			t.Fatal(err)
		}
		signal.Ignore(syscall.SIGXFSZ)
		limit := uint64(len(before) + 10)
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: limit, Max: limit}); err != nil {
			t.Fatal(err)
		}
		if err := appendRecoveryAudit(root, RecoveryAuditEntry{
			Action: "identity_recovered", IdentityID: CanonicalCEOID, Success: true,
		}); err == nil {
			t.Fatal("appendRecoveryAudit() succeeded after a short write")
		}
		return
	}

	root := t.TempDir()
	registryDir := filepath.Join(root, ".ovav", "registry")
	if err := os.MkdirAll(registryDir, 0o700); err != nil {
		t.Fatal(err)
	}
	before := []byte("{\"action\":\"existing\"}\n")
	auditPath := filepath.Join(root, AuditRelativePath)
	if err := os.WriteFile(auditPath, before, 0o600); err != nil {
		t.Fatal(err)
	}

	command := exec.Command(os.Args[0], "-test.run=^TestAppendRecoveryAuditDoesNotLeavePartialRecord$")
	command.Env = append(os.Environ(), helperEnv+"="+root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("partial audit helper failed: %v\n%s", err, output)
	}
	after, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("failed audit append left a partial JSONL record: got %d bytes, want %d", len(after), len(before))
	}
}
