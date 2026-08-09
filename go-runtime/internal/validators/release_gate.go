package validators

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ReleaseGate validates version tag release readiness: clean workspace,
// version consistency, and valid source branch for releases.
// Replaces: check_release_gate.py
type ReleaseGate struct{}

func NewReleaseGate() *ReleaseGate { return &ReleaseGate{} }

func (r *ReleaseGate) ID() string   { return "release_gate" }
func (r *ReleaseGate) Name() string { return "Release Gate" }
func (r *ReleaseGate) Description() string {
	return "Validates version tag readiness for release to main"
}
func (r *ReleaseGate) Weight() int { return 14 }

func (r *ReleaseGate) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check working tree is clean
	statusCmd := exec.CommandContext(ctx, "git", "-C", root, "status", "--porcelain")
	statusOut, err := statusCmd.Output()
	if err != nil {
		issues = append(issues, fmt.Sprintf("GIT: Cannot check git status: %v", err))
	} else if len(strings.TrimSpace(string(statusOut))) > 0 {
		issues = append(issues, "DIRTY: Working tree has uncommitted changes — release requires clean workspace")
	}

	// 2. Check current branch is a valid release source
	branchCmd := exec.CommandContext(ctx, "git", "-C", root, "branch", "--show-current")
	branchOut, err := branchCmd.Output()
	if err != nil {
		issues = append(issues, fmt.Sprintf("GIT: Cannot determine branch: %v", err))
	} else {
		branch := strings.TrimSpace(string(branchOut))
		validSources := map[string]bool{"develop": true, "main": true, "master": true}
		if !validSources[branch] {
			issues = append(issues, fmt.Sprintf("BRANCH: '%s' is not a valid release source (must be develop or main)", branch))
		}
	}

	// 3. Check for version tag on HEAD
	tagCmd := exec.CommandContext(ctx, "git", "-C", root, "tag", "--points-at", "HEAD")
	tagOut, err := tagCmd.Output()
	if err != nil {
		issues = append(issues, fmt.Sprintf("GIT: Cannot check tags: %v", err))
	} else {
		tags := strings.TrimSpace(string(tagOut))
		if tags == "" {
			// No tag — not a release, this is OK (informational)
		} else {
			// Has tag — validate version consistency
			firstTag := strings.Split(tags, "\n")[0]
			// Check AGENTS.md contains the tag version
			agentsPath := filepath.Join(root, "AGENTS.md")
			if data, err := os.ReadFile(agentsPath); err == nil {
				version := strings.TrimPrefix(firstTag, "v")
				if !strings.Contains(string(data), version) {
					issues = append(issues, fmt.Sprintf("VERSION: Tag '%s' version '%s' not found in AGENTS.md", firstTag, version))
				}
			}
		}
	}

	// 4. Check VERSION file exists for release consistency
	versionPath := filepath.Join(root, "VERSION")
	if data, err := os.ReadFile(versionPath); err == nil {
		content := strings.TrimSpace(string(data))
		if content == "" {
			issues = append(issues, "EMPTY: VERSION file is empty")
		}
	} else {
		issues = append(issues, "MISSING: VERSION file not found")
	}

	// 5. Check CHANGELOG exists
	changelogPath := filepath.Join(root, "CHANGELOG.md")
	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: CHANGELOG.md not found")
	}

	if len(issues) > 0 {
		return Result{
			ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(),
			Message:  fmt.Sprintf("FAIL release gate — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: r.ID(), Name: r.Name(), Status: "pass", Weight: r.Weight(),
		Message:  "PASS release gate — clean workspace, valid source, version consistency verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*ReleaseGate)(nil)
