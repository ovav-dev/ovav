package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/gitflow"
	"github.com/ovav/ovav/internal/validators"
)

// cmdPush handles `ovav push` — governed push with pre-flight validation,
// protected branch guardianship, audit trail, and backup ref.
//
// Value over raw `git push`:
//   - Pre-flight validation (protected branch, workspace safety, HTTPS gate)
//   - Dry-run preview (--dry-run flag)
//   - Protected branch waiver check
//   - Backup ref before push (refs/backups/<branch>/<timestamp>)
//   - Audit trail to .ovav/runtime/logs/push_audit.jsonl
//
// Usage:
//
//	ovav push [--dry-run] [--remote <name>]
func cmdPush(args []string) int {
	opts, err := parsePushArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ OVAV push: %v\n", err)
		return 2
	}
	if opts.help {
		printPushHelp()
		return 0
	}
	dryRun := opts.dryRun
	remote := opts.remote

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ OVAV push: %v\n", err)
		return 1
	}

	// ── 1. Determine current branch ────────────────────────────────────────
	branch := strings.TrimSpace(gitCmdOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch == "HEAD" {
		fmt.Fprintf(os.Stderr, "❌ OVAV push: detached HEAD — cannot push\n")
		return 1
	}

	// ── 2. Protected branch check ─────────────────────────────────────────
	protectedBranches := map[string]bool{
		"main":       true,
		"master":     true,
		"develop":    true,
		"production": true,
		"staging":    true,
	}

	if protectedBranches[branch] {
		waiverPath := fmt.Sprintf("%s/.ovav/runtime/protected_branch_waiver.yaml", repoRoot)
		if _, err := os.Stat(waiverPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "❌ OVAV push: push to protected branch %q requires CEO waiver.\n", branch)
			fmt.Fprintf(os.Stderr, "   Request waiver via: go run ./cmd/ovav/ waiver request --branch %s\n", branch)
			return 1
		}
		fmt.Printf("  ✅ Protected branch %q — waiver present\n", branch)
	}

	// ── 3. Pre-flight validation ──────────────────────────────────────────
	fmt.Println("🔍 Pre-flight validation...")

	results, passed := runPushPreflight(context.Background(), repoRoot, governedPushValidators())
	for _, r := range results {
		icon := "✅"
		if r.Status == "fail" || r.Status == "error" {
			icon = "❌"
		}
		fmt.Printf("  %s %s: %s\n", icon, r.Name, r.Message)
	}
	if !passed {
		fmt.Fprintf(os.Stderr, "\n❌ Pre-flight validation failed — fix issues before push.\n")
		fmt.Fprintf(os.Stderr, "   Run `go run ./cmd/ovav/ validate --gate` for full report.\n")
		return 1
	}
	fmt.Println("  ✅ Pre-flight passed")

	// ── 4. Fetch + divergence check ──────────────────────────────────────
	fmt.Println("🔍 Checking remote divergence...")
	if err := gitflow.Fetch(repoRoot, remote); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  Fetch failed (non-fatal): %v\n", err)
	}

	localCommit := strings.TrimSpace(gitCmdOutput(repoRoot, "rev-parse", "HEAD"))
	remoteCommit := strings.TrimSpace(gitCmdOutput(repoRoot, "rev-parse", fmt.Sprintf("%s/%s", remote, branch)))

	diverged := localCommit != remoteCommit
	behind := false
	ahead := false

	if diverged {
		// Use git merge-base --is-ancestor to correctly determine relationship
		// If remote/branch is ancestor of HEAD → we are ahead (remote is behind)
		// If HEAD is ancestor of remote/branch → we are behind (remote is ahead)
		// If neither is ancestor of the other → diverged
		remoteRef := fmt.Sprintf("%s/%s", remote, branch)
		ahead = isAncestor(repoRoot, remoteRef, "HEAD")
		behind = isAncestor(repoRoot, "HEAD", remoteRef)
	}

	if behind && !ahead {
		fmt.Fprintf(os.Stderr, "\n❌ Push rejected: your branch is behind %s/%s by %d commit(s).\n", remote, branch, countCommitsBetween(repoRoot, remoteCommit, localCommit))
		fmt.Fprintf(os.Stderr, "   Run `ovav git pull` or `owd` to integrate remote changes first.\n")
		fmt.Fprintf(os.Stderr, "   DO NOT use --force — that would overwrite remote changes.\n")
		return 1
	}

	if behind && ahead {
		fmt.Fprintf(os.Stderr, "\n❌ Push rejected: your branch has diverged from %s/%s.\n", remote, branch)
		fmt.Fprintf(os.Stderr, "   Remote has %d commit(s) you don't have.\n", countCommitsBetween(repoRoot, localCommit, remoteCommit))
		fmt.Fprintf(os.Stderr, "   Use `owd` to merge your branch intelligently, or `git pull --rebase` if confident.\n")
		return 1
	}

	if !behind && !ahead && diverged {
		// True diverged: both have commits the other doesn't
		behindCount := countCommitsBetween(repoRoot, localCommit, remoteCommit)
		aheadCount := countCommitsBetween(repoRoot, remoteCommit, localCommit)
		fmt.Fprintf(os.Stderr, "\n❌ Push rejected: your branch has diverged from %s/%s.\n", remote, branch)
		fmt.Fprintf(os.Stderr, "   You have %d commit(s) not on remote; remote has %d commit(s) not on your branch.\n", aheadCount, behindCount)
		fmt.Fprintf(os.Stderr, "   Use `owd` to merge your branch intelligently.\n")
		return 1
	}

	if !diverged {
		fmt.Printf("  ✅ Branch is up-to-date with %s/%s\n", remote, branch)
	} else if ahead && !behind {
		fmt.Printf("  ✅ Branch is ahead of %s/%s — %d commit(s) to push\n", remote, branch, countCommitsBetween(repoRoot, remoteCommit, localCommit))
	}

	// ── 5. Dry-run ────────────────────────────────────────────────────────
	if dryRun {
		fmt.Printf("\n🔎 Dry-run — what would be pushed:\n")
		cmd := exec.Command("git", "push", "--dry-run", remote, branch)
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Dry-run failed: %v\n", err)
			return 1
		}
		return 0
	}

	// ── 5. Backup ref before push ─────────────────────────────────────────
	backupRef := fmt.Sprintf("refs/backups/%s/%d", branch, time.Now().Unix())
	fmt.Printf("  📦 Creating backup ref: %s\n", backupRef)
	gitCmd(repoRoot, "update-ref", backupRef, "HEAD")

	// ── 6. Audit trail ────────────────────────────────────────────────────
	logPushAudit(repoRoot, branch, remote, false)

	// ── 7. Execute push via gitflow.Push (HTTPS-only, no force) ──────────
	fmt.Printf("\n🚀 Governed push %s → %s/%s (HTTPS)\n", branch, remote, branch)
	if err := gitflow.Push(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Push failed: %v\n", err)
		return 1
	}

	// ── 8. Post-push report ───────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("  ✅ Governed push complete\n")
	fmt.Printf("  📝 Audit log: .ovav/runtime/logs/push_audit.jsonl\n")
	fmt.Printf("  🔄 To verify: go run ./cmd/ovav/ validate\n")

	return 0
}

func governedPushValidators() []validators.Validator {
	return []validators.Validator{
		validators.NewProtectedBranch(),
		validators.NewGitPush(),
		validators.NewWorkspaceSafety(),
		validators.NewSupplyChain(validators.ValidationGate),
		validators.NewRuntimeIntegrity(validators.ValidationGate),
	}
}

func runPushPreflight(ctx context.Context, repoRoot string, validatorSet []validators.Validator) ([]validators.Result, bool) {
	results := validators.NewRegistry(validatorSet...).Run(ctx, repoRoot)
	for _, result := range results {
		if result.Status == "fail" || result.Status == "error" {
			return results, false
		}
	}
	return results, true
}

type pushOptions struct {
	dryRun bool
	remote string
	help   bool
}

func parsePushArgs(args []string) (pushOptions, error) {
	opts := pushOptions{remote: "origin"}
	for _, arg := range args {
		switch arg {
		case "--dry-run", "-n":
			opts.dryRun = true
		case "--help", "-h":
			opts.help = true
		case "--force", "-f", "--force-with-lease", "--skip-validate", "--no-validate":
			return pushOptions{}, fmt.Errorf("option %s is prohibited by push governance", arg)
		default:
			if strings.HasPrefix(arg, "--remote=") && strings.TrimPrefix(arg, "--remote=") != "" {
				opts.remote = strings.TrimPrefix(arg, "--remote=")
				continue
			}
			return pushOptions{}, fmt.Errorf("unknown option %s", arg)
		}
	}
	return opts, nil
}

// gitCmdOutput runs a git command and returns its stdout.
func gitCmdOutput(repoRoot string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// gitCmd runs a git command and panics on error (for non-critical git calls).
func gitCmd(repoRoot string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  git %v failed (non-fatal): %v\n", args, err)
	}
}

// logPushAudit writes a push event to the push audit log.
func logPushAudit(repoRoot, branch, remote string, force bool) {
	logDir := repoRoot + "/.ovav/runtime/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  Audit log dir failed: %v\n", err)
		return
	}

	logPath := logDir + "/push_audit.jsonl"
	entry := fmt.Sprintf(
		`{"event":"governed_push","branch":"%s","remote":"%s","force":%v,"timestamp":"%s","operator":"thavren"}`+"\n",
		branch, remote, force, time.Now().UTC().Format(time.RFC3339),
	)

	if err := os.WriteFile(logPath, []byte(entry), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠️  Audit log write failed: %v\n", err)
	}
}

func printPushHelp() {
	fmt.Print(`
OVAV Governed Push — safe git push with governance layer

Usage: ovav push [flags]

Flags:
  --dry-run, -n       Show what would be pushed (no changes)
  --remote=<name>    Push to specific remote (default: origin)
  --help, -h         Show this help

What ovav push does that raw git push doesn't:
  ✅ Pre-flight validation (protected branch, HTTPS, workspace safety)
  ✅ Remote divergence detection (behind/ahead/diverged — prevents overwriting remote)
  ✅ Protected branch waiver check
  ✅ Backup ref before push (refs/backups/<branch>/<timestamp>)
  ✅ Audit trail (.ovav/runtime/logs/push_audit.jsonl)
  ✅ Raw force options are rejected

Examples:
  ovav push                    # Normal governed push
  ovav push --dry-run          # Preview what would be pushed
`)
}

// countCommitsBetween returns the number of commits from fromCommit to toCommit.
// Both must be valid commit hashes.
func countCommitsBetween(repoRoot, fromCommit, toCommit string) int {
	if fromCommit == "" || toCommit == "" || fromCommit == toCommit {
		return 0
	}
	// Use git rev-list --count to count commits in the range
	cmd := exec.Command("git", "rev-list", "--count", fromCommit+".."+toCommit)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return -1 // unknown
	}
	var n int
	if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n); scanErr != nil {
		return -1
	}
	return n
}

// isAncestor checks if ancestorCommit is an ancestor of descendantCommit in the git DAG.
func isAncestor(repoRoot, ancestorCommit, descendantCommit string) bool {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestorCommit, descendantCommit)
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}
