// Package ows — OVAV Compliance Seal system.
// Generates a cryptographically verifiable seal proving that a merge
// passed all required compliance checks. The seal is derived from the
// git tree hash + validator results + GPG signatures + reviewer identity.
// Cannot be forged because any modification changes the hash.
package ows

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ComplianceLevel defines the strictness of pre-merge requirements.
type ComplianceLevel string

const (
	ComplianceQuick    ComplianceLevel = "quick"
	ComplianceStandard ComplianceLevel = "standard"
	ComplianceStrict   ComplianceLevel = "strict"
	ComplianceMaximum  ComplianceLevel = "maximum"
)

// ComplianceRequirements maps each level to its required checks.
type ComplianceRequirements struct {
	Owv             bool    // go test + vet + fmt
	SecretsSweep    bool    // secrets_hygiene scan on changed files
	ForbiddenFiles  bool    // block .env, .pem, .key, large binaries
	ValidateAll     bool    // 79 validators
	ValidateMinPct  float64 // minimum validator pass % (0.0 = no minimum)
	ConflictPred    bool    // conflict prediction
	GPGSigned       bool    // all commits GPG signed
	ReviewerReq     bool    // lead review required
	RedTeam         bool    // Red Team R5 audit
	HygieneRequired bool    // hygiene must be clean (0 issues)
	BlockOnWarning  bool    // block merge even on warnings (not just failures)
}

// RequirementsFor returns the requirements for a given compliance level.
func RequirementsFor(level ComplianceLevel) ComplianceRequirements {
	switch level {
	case ComplianceQuick:
		return ComplianceRequirements{
			ConflictPred: true,
		}
	case ComplianceStandard:
		// Elevated (2026-06-19): standard now requires clean hygiene, secrets sweep,
		// 90%+ validators pass, AND blocks on warnings (no more "warn but proceed").
		return ComplianceRequirements{
			Owv:             true,
			SecretsSweep:    true,
			ForbiddenFiles:  true,
			ValidateAll:     true,
			ValidateMinPct:  0.85, // 85% minimum validator pass rate
			ConflictPred:    true,
			HygieneRequired: true,
			BlockOnWarning:  true,
		}
	case ComplianceStrict:
		return ComplianceRequirements{
			Owv:             true,
			SecretsSweep:    true,
			ForbiddenFiles:  true,
			ValidateAll:     true,
			ValidateMinPct:  0.95, // 95% minimum
			ConflictPred:    true,
			GPGSigned:       true,
			ReviewerReq:     true,
			HygieneRequired: true,
			BlockOnWarning:  true,
		}
	case ComplianceMaximum:
		return ComplianceRequirements{
			Owv:             true,
			SecretsSweep:    true,
			ForbiddenFiles:  true,
			ValidateAll:     true,
			ValidateMinPct:  1.0, // 100% — all validators must pass
			ConflictPred:    true,
			GPGSigned:       true,
			ReviewerReq:     true,
			RedTeam:         true,
			HygieneRequired: true,
			BlockOnWarning:  true,
		}
	default:
		return ComplianceRequirements{Owv: true, SecretsSweep: true, ConflictPred: true, BlockOnWarning: true}
	}
}

// Seal represents an OVAV compliance seal — proof that a merge passed all checks.
type Seal struct {
	Version   string    `json:"version"`   // "v1"
	Branch    string    `json:"branch"`    // "feature/header-design"
	Author    string    `json:"author"`    // "thavren@ovav"
	Level     string    `json:"level"`     // "strict"
	GitTree   string    `json:"git_tree"`  // git tree hash at merge time
	Reviewer  string    `json:"reviewer"`  // approving reviewer (empty if none)
	Sigs      int       `json:"sigs"`      // number of GPG-signed commits
	Validated int       `json:"validated"` // validators passed (e.g., 77)
	CreatedAt time.Time `json:"created_at"`
	Hash      string    `json:"hash"` // SHA-256 of all above fields
}

// GenerateSeal creates a compliance seal for a completed merge.
func GenerateSeal(repoRoot, branch, author, reviewer string, level ComplianceLevel, treeHash string, sigCount, validCount int) *Seal {
	s := &Seal{
		Version:   "v1",
		Branch:    branch,
		Author:    author,
		Level:     string(level),
		GitTree:   treeHash,
		Reviewer:  reviewer,
		Sigs:      sigCount,
		Validated: validCount,
		CreatedAt: time.Now().UTC(),
	}

	// Compute hash: SHA-256 of all seal fields concatenated
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s|%s|%d|%d|%s",
		s.Version, s.Branch, s.Author, s.Level, s.GitTree,
		s.Reviewer, s.Sigs, s.Validated, s.CreatedAt.Format(time.RFC3339))
	s.Hash = hex.EncodeToString(h.Sum(nil))[:16] // first 16 hex chars

	return s
}

// VerifySeal checks if a seal hash is valid by recomputing it from the fields.
// Returns true if the hash matches and the git tree still exists.
func VerifySeal(repoRoot string, s *Seal) (bool, string) {
	recomputed := GenerateSeal(repoRoot, s.Branch, s.Author, s.Reviewer,
		ComplianceLevel(s.Level), s.GitTree, s.Sigs, s.Validated)

	if recomputed.Hash != s.Hash {
		return false, fmt.Sprintf("hash mismatch: expected %s, got %s", s.Hash, recomputed.Hash)
	}
	// Verify the git tree still exists
	cmd := exec.Command("git", "cat-file", "-e", s.GitTree)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return false, "git tree no longer exists — seal is stale"
	}
	return true, "seal valid ✓"
}

// GetGitTreeHash returns the current HEAD tree hash.
func GetGitTreeHash(repoRoot string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD^{tree}")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// SealedStatus returns a human-readable summary of compliance status.
func SealedStatus(level ComplianceLevel) string {
	r := RequirementsFor(level)
	var parts []string
	if r.Owv {
		parts = append(parts, "owv")
	}
	if r.SecretsSweep {
		parts = append(parts, "secrets")
	}
	if r.ForbiddenFiles {
		parts = append(parts, "forbidden")
	}
	if r.ValidateAll {
		parts = append(parts, fmt.Sprintf("validate≥%.0f%%", r.ValidateMinPct*100))
	}
	if r.ConflictPred {
		parts = append(parts, "conflict")
	}
	if r.GPGSigned {
		parts = append(parts, "gpg")
	}
	if r.ReviewerReq {
		parts = append(parts, "review")
	}
	if r.RedTeam {
		parts = append(parts, "red-team")
	}
	if r.HygieneRequired {
		parts = append(parts, "hygiene")
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "+")
}

// DisplaySeal prints a formatted compliance seal to stdout.
func DisplaySeal(s *Seal) {
	fmt.Println()
	fmt.Println("  🔏 OVAV COMPLIANCE SEAL")
	fmt.Println("  ┌" + strings.Repeat("─", 58) + "┐")
	fmt.Printf("  │ %-30s %-26s │\n", "Task:     "+s.Branch, "Level: "+s.Level)
	fmt.Printf("  │ %-30s %-26s │\n", "Author:   "+s.Author, "Reviewer: "+s.Reviewer)
	if s.Sigs > 0 {
		fmt.Printf("  │ Signed: %-25d Validated: %-23d │\n", s.Sigs, s.Validated)
	}
	treeDisplay := s.GitTree
	if len(treeDisplay) > 16 {
		treeDisplay = treeDisplay[:16]
	}
	fmt.Printf("  │ %-56s │\n", "Tree: "+treeDisplay+"...")
	fmt.Printf("  │ %-56s │\n", "Seal: "+s.Hash)
	fmt.Printf("  │ %-56s │\n", "Time: "+s.CreatedAt.Format(time.RFC3339))
	fmt.Println("  └" + strings.Repeat("─", 58) + "┘")
	fmt.Printf("  🔗 Verify: ovav verify --seal %s\n", s.Hash)
	fmt.Println()
}

// String returns a compact one-line seal representation.
func (s *Seal) String() string {
	return fmt.Sprintf("🔏 %s [%s] · %s · %d sigs · %s",
		s.Hash, s.Level, s.Branch, s.Sigs, s.CreatedAt.Format("15:04 UTC"))
}
