package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/truststore"
)

// HeadIntegrity validates git HEAD integrity against a trusted hash.
// Per-worktree: stores trusted HEADs in .ovav/runtime/worktree_heads.json.
// Falls back to global .ovav/runtime/trusted_head_hash.json for main-repo.
type HeadIntegrity struct{}

func NewHeadIntegrity() *HeadIntegrity { return &HeadIntegrity{} }

func (h *HeadIntegrity) ID() string   { return "head_integrity" }
func (h *HeadIntegrity) Name() string { return "HEAD Integrity Verifier" }
func (h *HeadIntegrity) Description() string {
	return "Validates git HEAD integrity against trusted hash for self-heal safety"
}
func (h *HeadIntegrity) Weight() int { return 15 }

func (h *HeadIntegrity) getCurrentHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectWorktreeRoot finds the git worktree that contains `root`.
// Returns "" if root is not inside a worktree.
func (h *HeadIntegrity) detectWorktreeRoot(root string) string {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var worktreeRoots []string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			worktreeRoots = append(worktreeRoots, strings.TrimPrefix(line, "worktree "))
		}
	}
	// Find the worktree whose path is a prefix of root (excluding main repo itself)
	for _, wt := range worktreeRoots {
		if wt != root && strings.HasPrefix(root, wt) {
			return wt
		}
	}
	return ""
}

func (h *HeadIntegrity) readTrustedHead(root string) (worktreeRoot, trusted string) {
	// Try per-worktree first
	wtRoot := h.detectWorktreeRoot(root)
	if wtRoot != "" {
		heads, err := truststore.ReadWorktreeHeads(root)
		if err == nil {
			if trusted, ok := heads[wtRoot]; ok && trusted != "" {
				return wtRoot, trusted
			}
		}
	}
	// Fallback to global
	globalPath := filepath.Join(root, ".ovav", "runtime", "trusted_head_hash.json")
	data, err := os.ReadFile(globalPath)
	if err != nil {
		return "", ""
	}
	var rec struct{ TrustedHeadSHA string `json:"trusted_head_sha"` }
	if err := json.Unmarshal(data, &rec); err == nil && rec.TrustedHeadSHA != "" {
		return "", rec.TrustedHeadSHA
	}
	sha := strings.TrimSpace(string(data))
	if len(sha) == 40 {
		return "", sha
	}
	return "", ""
}

// json is needed for parsing the trusted_head_sha field
func (h *HeadIntegrity) unmarshalTrusted(data []byte) (string, error) {
	var rec struct{ TrustedHeadSHA string }
	if err := json.Unmarshal(data, &rec); err != nil {
		return "", err
	}
	return rec.TrustedHeadSHA, nil
}

func (h *HeadIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	current := h.getCurrentHead(root)
	if current == "" {
		issues = append(issues, "HEAD INTEGRITY: cannot read git HEAD — repository may be corrupted")
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  "FAIL head integrity — cannot read git HEAD",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	wtRoot, trusted := h.readTrustedHead(root)

	if trusted == "" {
		// First run — initialize
		if wtRoot != "" {
			_ = truststore.WriteWorktreeHead(root, wtRoot, current)
			return Result{
				ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
				Message:  fmt.Sprintf("PASS head integrity — trusted hash initialized (worktree: %s, HEAD: %s)", wtRoot, current[:8]),
				Duration: time.Since(start),
			}
		}
		// Global init (main repo first run)
		p := filepath.Join(root, ".ovav", "runtime", "trusted_head_hash.json")
		dir := filepath.Dir(p)
		_ = os.MkdirAll(dir, 0755)
		_ = os.WriteFile(p, []byte(fmt.Sprintf(`{"trusted_head_sha":"%s"}`, current)), 0644)
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
			Message:  fmt.Sprintf("PASS head integrity — trusted hash initialized. Current HEAD: %s", current[:8]),
			Duration: time.Since(start),
		}
	}

	if current != trusted {
		ctxInfo := ""
		if wtRoot != "" {
			ctxInfo = fmt.Sprintf(" (worktree: %s, expected: %s...)", wtRoot, trusted[:8])
		}
		issues = append(issues, fmt.Sprintf(
			"HEAD INTEGRITY MISMATCH: current HEAD (%s) != trusted (%s).%s",
			current[:8], trusted[:8], ctxInfo,
		))
		if wtRoot != "" {
			issues = append(issues, "Worktree just created via owc? This is expected. Run: truststore.WriteWorktreeHead(repo, wt, head)")
		}
		issues = append(issues, "Verify: git log --oneline -10")
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  "FAIL head integrity — HEAD does not match trusted hash",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	// AGENTS.md integrity seal check
	agentsPath := filepath.Join(root, "AGENTS.md")
	if data, err := os.ReadFile(agentsPath); err == nil {
		if !strings.Contains(string(data), "OVAV_INTEGRITY_SEAL") {
			issues = append(issues, "AGENTS.md integrity seal missing — possible tampering")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  fmt.Sprintf("FAIL head integrity — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
		Message:  fmt.Sprintf("PASS head integrity — HEAD matches trusted hash (%s)", current[:8]),
		Duration: time.Since(start),
	}
}

var _ Validator = (*HeadIntegrity)(nil)
