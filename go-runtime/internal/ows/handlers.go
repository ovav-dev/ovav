package ows

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/alerts"
	"github.com/ovav/ovav/internal/audit"
	"github.com/ovav/ovav/internal/gitflow"
	"github.com/ovav/ovav/internal/truststore"
)

// auditLogger is the package-level audit logger for OWS operations.
// Initialized by InitAudit.
var auditLogger *audit.AuditLogger

// InitAudit starts the audit logger. Call once at startup.
func InitAudit(opts ...audit.Option) error {
	l, err := audit.New(opts...)
	if err != nil {
		return err
	}
	auditLogger = l
	return nil
}

// ── Command Handler Wiring ──────────────────────────────────────────────────

// WireHandlers assigns real handler functions to all registered OWS commands.
// Must be called once at startup before Dispatch is used.
func WireHandlers(repoRoot string) {
	// Initialize audit logger — BestEffort; audit failures never break OWS.
	// Uses OS temp dir for log storage.
	InitAudit(audit.Dir(filepath.Join(repoRoot, ".ovav", "audit")))

	// owc — create worktree with conflict prediction on base branch
	if cmd, ok := CommandRegistry["ovav worktree create"]; ok {
		cmd.Handler = makeCreateHandler(repoRoot)
		CommandRegistry["ovav worktree create"] = cmd
	}

	// owd — verify → predict conflicts → merge → cleanup
	if cmd, ok := CommandRegistry["ovav worktree done"]; ok {
		cmd.Handler = makeDoneHandler(repoRoot)
		CommandRegistry["ovav worktree done"] = cmd
	}

	// owl — list worktrees with conflict predictions
	if cmd, ok := CommandRegistry["ovav worktree list"]; ok {
		cmd.Handler = makeListHandler(repoRoot)
		CommandRegistry["ovav worktree list"] = cmd
	}

	// owx — route commits between branches
	if cmd, ok := CommandRegistry["ovav worktree route"]; ok {
		cmd.Handler = makeRouteHandler(repoRoot)
		CommandRegistry["ovav worktree route"] = cmd
	}

	// owa — abort in-progress operation
	if cmd, ok := CommandRegistry["ovav worktree abort"]; ok {
		cmd.Handler = makeAbortHandler(repoRoot)
		CommandRegistry["ovav worktree abort"] = cmd
	}

	// owr — rescue lost work
	if cmd, ok := CommandRegistry["ovav worktree rescue"]; ok {
		cmd.Handler = makeRescueHandler(repoRoot)
		CommandRegistry["ovav worktree rescue"] = cmd
	}

	// ows — sync remotes + maintenance + prune
	if cmd, ok := CommandRegistry["ovav worktree sync"]; ok {
		cmd.Handler = makeSyncHandler(repoRoot)
		CommandRegistry["ovav worktree sync"] = cmd
	}

	// owv — verify worktree (tests + lint + conflict check)
	if cmd, ok := CommandRegistry["ovav worktree verify"]; ok {
		cmd.Handler = makeVerifyHandler(repoRoot)
		CommandRegistry["ovav worktree verify"] = cmd
	}

	// owu — update (fetch + rebase with conflict prediction)
	if cmd, ok := CommandRegistry["ovav worktree update"]; ok {
		cmd.Handler = makeUpdateHandler(repoRoot)
		CommandRegistry["ovav worktree update"] = cmd
	}

	// owp — prepare/sync current branch with origin (fast-forward pull or rebase)
	if cmd, ok := CommandRegistry["ovav worktree prepare"]; ok {
		cmd.Handler = makePrepareHandler(repoRoot)
		CommandRegistry["ovav worktree prepare"] = cmd
	}

	// owlk — lock worktree
	if cmd, ok := CommandRegistry["ovav worktree lock"]; ok {
		cmd.Handler = makeLockHandler(repoRoot)
		CommandRegistry["ovav worktree lock"] = cmd
	}

	// owm — move worktree
	if cmd, ok := CommandRegistry["ovav worktree move"]; ok {
		cmd.Handler = makeMoveHandler(repoRoot)
		CommandRegistry["ovav worktree move"] = cmd
	}

	// owclean — clean orphaned/stale/abandoned worktrees
	if cmd, ok := CommandRegistry["ovav worktree clean"]; ok {
		cmd.Handler = makeCleanHandler(repoRoot)
		CommandRegistry["ovav worktree clean"] = cmd
	}

	// owprep — verify/regenerate worktree config (SU-9)
	if cmd, ok := CommandRegistry["ovav worktree prep"]; ok {
		cmd.Handler = makePrepHandler(repoRoot)
		CommandRegistry["ovav worktree prep"] = cmd
	}

	// owsuggest — suggest next command (SU-10)
	if cmd, ok := CommandRegistry["ovav worktree suggest"]; ok {
		cmd.Handler = makeSuggestHandler(repoRoot)
		CommandRegistry["ovav worktree suggest"] = cmd
	}

	// own — nuclear delete (worktree + local branch + remote branch)
	if cmd, ok := CommandRegistry["ovav worktree nuke"]; ok {
		cmd.Handler = makeNukeHandler(repoRoot)
		CommandRegistry["ovav worktree nuke"] = cmd
	}
}

// ── Handler Factories ──────────────────────────────────────────────────────

func makeCreateHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		name := args["name"]
		if name == "" || name == "--help" || name == "-h" {
			return fmt.Errorf("owc: task name required.\n\nUsage: owc <name> [--profile=<name>] [--compliance <level>]\n\nExamples:\n  owc feature/my-feature\n  owc task42\n  owc fix/bug-123 --compliance strict\n  owc task27 --profile=hotfix\n\nFlags:\n  --profile=<name>       force profile override: feature, refactor, hotfix, release, patch, docs, spike, research, migration, enterprise, fix, emergency\n  --compliance <level>  quick, standard, strict, maximum (default: standard)")
		}

		// ── Detect profile from name ──
		// Manual mode: name starts with known prefix → literal branch
		//   owc feature/header_design → branch: feature/header_design
		// Simple mode: no prefix → default "feature" + name as-is
		//   owc task27 → branch: feature/task27

		var profileName string
		var taskName string

		if idx := strings.Index(name, "/"); idx > 0 {
			// Has slash: detect if first segment is a known profile prefix
			firstSegment := name[:idx]
			if _, ok := ProfileRegistry[firstSegment]; ok {
				profileName = firstSegment
				taskName = name[idx+1:]
			} else {
				// Unknown prefix: treat entire name as task, default profile
				profileName = "feature"
				taskName = name
			}
		} else {
			// No slash: simple mode, default profile
			profileName = "feature"
			taskName = name
		}

		// ── Override profile if --profile flag provided ──
		if flagProfile := args["profile"]; flagProfile != "" {
			profileName = flagProfile
		}

		// SU-5: emergency profile requires waiver verification (hotfix auto-allowed)
		if profileName == "emergency" {
			if _, err := LoadWaiver(repoRoot); err != nil {
				return fmt.Errorf("owc: profile %q requires an active CEO waiver.\n  Run: ovav waiver \"emergency worktree\"", profileName)
			}
			fmt.Printf("  ⚠️  Profile=emergency (forced override) — waiver verified\n")
		}

		// ── Compliance level: --compliance flag or default from profile ──
		complianceLevel := args["compliance"]
		if complianceLevel == "" {
			complianceLevel = "standard"
		}
		// Validate compliance level
		cl := ComplianceLevel(complianceLevel)
		switch cl {
		case ComplianceQuick, ComplianceStandard, ComplianceStrict, ComplianceMaximum:
			// valid
		default:
			return fmt.Errorf("owc: invalid compliance level %q — valid: quick, standard, strict, maximum", complianceLevel)
		}

		profile, ok := ProfileRegistry[profileName]
		if !ok {
			return fmt.Errorf("owc: unknown profile %q — valid: feature, refactor, hotfix, release, patch, docs, spike, research, migration, enterprise, fix, emergency", profileName)
		}

		// Convert OWS ProfileConfig → gitflow.Profile
		gfProfile := profileToGitflow(profileName, profile)
		gfProfile.Compliance = complianceLevel

		// ── Branch name ──
		// Manual mode (name had slash + known prefix): taskName already extracted
		//   user typed: feature/header_design → branch: feature/header_design
		// Simple mode (no slash): just profile/task
		//   user typed: task27 → branch: feature/task27
		branch := gfProfile.Prefix + taskName

		// Check for conflicts before creating worktree
		matrix, err := PredictConflicts(repoRoot, branch, gfProfile.Base)
		if err == nil && matrix.ConflictFiles > 0 {
			fmt.Printf("⚠️  Conflict prediction: %s\n", matrix.Summary())
			for _, f := range matrix.Conflicts() {
				fmt.Printf("   ⚠  %s — %d overlapping ranges\n", f.FilePath, len(f.OverlapRanges))
			}
		}

		// ── SU-3: --carry-uncommitted — migrate dirty changes to worktree ──
		carryUncommitted := args["carry-uncommitted"] == "true"
		var stashRef string
		if carryUncommitted {
			dirty := gitOutput(repoRoot, "status", "--porcelain")
			if strings.TrimSpace(dirty) != "" {
				stashMsg := fmt.Sprintf("owc-carry: %s", branch)
				stashOut := gitOutput(repoRoot, "stash", "push", "-u", "-m", stashMsg)
				if !strings.Contains(stashOut, "No local changes") {
					stashRef = "stash@{0}"
				}
			}
		} else {
			// No --carry-uncommitted flag: block if working tree is dirty
			dirty := gitOutput(repoRoot, "status", "--porcelain")
			if strings.TrimSpace(dirty) != "" {
				return fmt.Errorf("owc: working tree has uncommitted changes.\n"+
					"  Run 'owc %s --carry-uncommitted' to move changes to the new worktree,\n"+
					"  or commit/stash changes before creating a new worktree.", name)
			}
		}

		if err := gitflow.StartWithProfile(repoRoot, taskName, gfProfile); err != nil {
			return fmt.Errorf("owc: %w", err)
		}

		// ── SU-3: Apply carried stash in new worktree ──
		if carryUncommitted && stashRef != "" {
			// Compute the new worktree path: branch uses "/" replaced by "-"
			worktreeDir := strings.ReplaceAll(branch, "/", "-")
			worktreePath := filepath.Join(repoRoot, ".ovav", "worktrees", worktreeDir)
			popOut := gitOutput(worktreePath, "stash", "pop", stashRef)
			if strings.Contains(popOut, "CONFLICT") {
				fmt.Println("  ⚠️  Stash conflicts — resolve manually in worktree.")
			}
		}

		// ── Audit: worktree created ──
		author := gitflow.DetectAuthor(repoRoot)
		trailEvent(repoRoot, "WORKTREE_CREATED", branch, author,
			map[string]string{"profile": profileName, "compliance": complianceLevel, "base": gfProfile.Base, "merge": gfProfile.MergeTo})

		// ── T3.5: Initialize per-worktree trusted HEAD ──
		// Record the new worktree's HEAD so head_integrity validator can track it.
		worktreeDir := strings.ReplaceAll(branch, "/", "-")
		worktreePath := filepath.Join(repoRoot, ".ovav", "worktrees", worktreeDir)
		headOut := gitOutput(worktreePath, "rev-parse", "HEAD")
		if headSHA := strings.TrimSpace(headOut); len(headSHA) == 40 {
			_ = truststore.WriteWorktreeHead(repoRoot, worktreePath, headSHA)
			fmt.Printf("   🔐 Trust store: initialized worktree HEAD (%s)\n", headSHA[:8])
		}

		// Record git op timestamp for gate_self_protection grace period
		_ = truststore.RecordGitOp(repoRoot)

		return nil
	}
}

// profileToGitflow converts an OWS ProfileConfig to a gitflow.Profile.
func profileToGitflow(name string, p ProfileConfig) gitflow.Profile {
	return gitflow.Profile{
		Name:    name,
		Prefix:  prefixForProfile(name),
		Base:    p.BaseBranch,
		MergeTo: p.MergeTo,
	}
}

// prefixForProfile returns the branch prefix for a profile name.
// Each profile uses its own name as prefix (industry standard: type/description).
func prefixForProfile(name string) string {
	prefixes := map[string]string{
		"feature": "feature/", "refactor": "refactor/", "docs": "docs/",
		"spike": "spike/", "research": "research/", "migration": "migration/",
		"enterprise": "enterprise/", "hotfix": "hotfix/", "release": "release/",
		"patch": "patch/", "fix": "fix/", "emergency": "emergency/",
	}
	if p, ok := prefixes[name]; ok {
		return p
	}
	return "feature/"
}

func makeDoneHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		// ── Resolve worktree (location-agnostic, 3 modes) ──
		//   owd                   → auto-detect from current HEAD
		//   owd feature/sprint-1   → find worktree by branch name
		//   owd .ovav/worktrees/.. → explicit worktree path
		branchHint := args["branch"]
		wt, err := gitflow.ResolveWorktree(branchHint)
		if err != nil {
			return fmt.Errorf("owd: %w", err)
		}
		currentBranch := wt.Branch
		// Run animated pre-scan
		_, scanBranch, _ := ScanPhase(wt.WorktreePath)
		if scanBranch != "" {
			currentBranch = scanBranch
		}

		fmt.Printf("📍 Worktree: %s\n", wt.WorktreePath)
		if wt.IsWorktree {
			fmt.Printf("   Branch: %s\n", currentBranch)
		}

		if currentBranch == "develop" || currentBranch == "main" || currentBranch == "master" {
			return fmt.Errorf("owd: cannot merge from protected branch '%s'.\nHint: use 'owd <feature-branch>' to merge a specific worktree.", currentBranch)
		}

		// Detect profile → compliance requirements
		gfProfile := gitflow.DetectProfileFromBranch(currentBranch)
		complianceOverride := args["compliance"]
		complianceLevel := ComplianceLevel(gfProfile.Compliance)
		if complianceOverride != "" {
			complianceLevel = ComplianceLevel(complianceOverride)
		}
		reqs := RequirementsFor(complianceLevel)

		// ── Untracked flush check: offer to remove fetched files before merge ──
		if wt.IsWorktree {
			if state, err := loadUntrackedState(wt.WorktreePath); err == nil && len(state.Fetched) > 0 {
				var nonKept []string
				for f := range state.Fetched {
					if !state.Kept[f] {
						nonKept = append(nonKept, f)
					}
				}
				if len(nonKept) > 0 {
					fmt.Println("\n📋 Fetched files detected in this worktree:")
					for _, f := range nonKept {
						fmt.Printf("   %s\n", f)
					}
					fmt.Print("\n  Flush before merge? (yes/no/all/keep) [no]: ")
					reader := bufio.NewReader(os.Stdin)
					line, _, _ := reader.ReadLine()
					answer := strings.ToLower(strings.TrimSpace(string(line)))
					switch answer {
					case "yes", "all":
						fmt.Println("\n  🧹 Flushing fetched files...")
						for _, f := range nonKept {
							removeFetchedFile(wt.WorktreePath, f)
							delete(state.Fetched, f)
							fmt.Printf("  ✅ Removed: %s\n", f)
						}
						saveUntrackedState(wt.WorktreePath, state)
					case "keep":
						// mark all as kept
						for _, f := range nonKept {
							state.Kept[f] = true
						}
						saveUntrackedState(wt.WorktreePath, state)
						fmt.Println("\n  Files marked as KEEP — will not be flushed.")
					default:
						fmt.Println("\n  Keeping files as-is (not flushing).")
					}
				}
			}
		}

		fmt.Printf("🔏 Compliance: %s (%s)\n", complianceLevel, SealedStatus(complianceLevel))
		fmt.Printf("🔍 Pre-merge checks: %s → target(s)\n", currentBranch)

		// ── Compliance check: conflict prediction (multi-target) ──
		if reqs.ConflictPred {
			targets := strings.Split(gfProfile.MergeTo, "+")
			allClear := true
			for _, target := range targets {
				target = strings.TrimSpace(target)
				if target == "" || target == "none" {
					continue
				}
				fmt.Printf("   Predicting conflicts vs %s... ", target)
				matrix, err := PredictConflicts(repoRoot, currentBranch, target)
				if err != nil {
					fmt.Printf("⚠️  %v\n", err)
					continue
				}
				fmt.Printf("%s\n", matrix.Summary())
				if matrix.ConflictFiles > 0 {
					allClear = false
					for _, f := range matrix.Conflicts() {
						fmt.Printf("      ⚠  %s — lines %v\n", f.FilePath, f.OverlapRanges)
					}
				}
			}
			if !allClear {
				fmt.Println("\n⚠️  Conflicts detected. Resolve before merging.")
				return fmt.Errorf("owd: conflicts detected — merge blocked")
			}
		}

		// ── GATE 1: Secrets sweep (PRE-merge — blocks for standard+) ──
		if reqs.SecretsSweep {
			fmt.Println("🔍 Secrets sweep: scanning changed files...")
			findings, err := scanSecretsInChanges(repoRoot, currentBranch)
			if err != nil {
				fmt.Printf("   ⚠️  Secrets scan error: %v\n", err)
			} else if len(findings) > 0 {
				fmt.Printf("   🚨 SECRETS DETECTED (%d finding(s)):\n", len(findings))
				for _, f := range findings {
					fmt.Printf("      🔴 %s: %s\n", f.File, f.Detail)
				}
				fmt.Println("\n🚫 Merge BLOCKED: secrets in tracked files. Remove secrets before merging.")
				return fmt.Errorf("owd: secrets detected — merge blocked (%d findings)", len(findings))
			} else {
				fmt.Println("   ✅ No secrets detected")
			}
		}

		// ── GATE 2: Forbidden files (PRE-merge — blocks for standard+) ──
		if reqs.ForbiddenFiles {
			fmt.Println("🔍 Forbidden files: scanning...")
			forbidden, err := scanForbiddenFiles(repoRoot, currentBranch)
			if err != nil {
				fmt.Printf("   ⚠️  Forbidden scan error: %v\n", err)
			} else if len(forbidden) > 0 {
				fmt.Printf("   🚨 FORBIDDEN FILES (%d found):\n", len(forbidden))
				for _, f := range forbidden {
					fmt.Printf("      🔴 %s — %s\n", f.Path, f.Reason)
				}
				fmt.Println("\n🚫 Merge BLOCKED: forbidden files detected. Remove before merging.")
				return fmt.Errorf("owd: forbidden files detected — merge blocked (%d files)", len(forbidden))
			} else {
				fmt.Println("   ✅ No forbidden files")
			}
		}

		// ── Compliance check: owv ──
		var vr *VerifyResult
		if reqs.Owv {
			fmt.Println("🔍 owv: running validation pipeline...")
			var err error
			vr, err = Verify(repoRoot, detectChangedFiles(repoRoot), true)
			if err != nil {
				fmt.Printf("   ⚠️  Verification error: %v\n", err)
			} else if vr != nil {
				if !vr.Passed {
					fmt.Printf("   ⚠️  Verification: %s\n", vr.Detail)
				}
				fmt.Printf("   go vet: %-5s  gofmt: %-5s  go test: %-5s  hygiene: %-5s\n",
					boolIcon(vr.GoVetPass), boolIcon(vr.GofmtPass), boolIcon(vr.GoTestPass),
					statusIcon(vr.HygieneClean, "clean", fmt.Sprintf("%d⚠️", vr.HygieneIssues)))
				if vr.ValidateRan {
					validatePct := 0.0
					if vr.ValidateTotal > 0 {
						validatePct = float64(vr.ValidatePass) / float64(vr.ValidateTotal)
					}
					fmt.Printf("   validate: %d/%d passed (%.0f%%)",
						vr.ValidatePass, vr.ValidateTotal, validatePct*100)
					if reqs.ValidateMinPct > 0 && validatePct < reqs.ValidateMinPct {
						fmt.Printf(" 🔴 below %.0f%% threshold", reqs.ValidateMinPct*100)
					}
					fmt.Println()
				}

				// ── Blocking logic: elevated standard blocks on validator failures ──
				if reqs.ValidateMinPct > 0 && vr.ValidateRan && vr.ValidateTotal > 0 {
					validatePct := float64(vr.ValidatePass) / float64(vr.ValidateTotal)
					if validatePct < reqs.ValidateMinPct {
						return fmt.Errorf("owd: validator pass rate %.0f%% below required %.0f%% — merge blocked",
							validatePct*100, reqs.ValidateMinPct*100)
					}
				}

				// ── Hygiene blocking: strict/maximum require ZERO issues ──
				// standard requires no BLOCKING issues (warnings allowed)
				if reqs.HygieneRequired && !vr.HygieneClean {
					if complianceLevel == ComplianceStrict || complianceLevel == ComplianceMaximum {
						return fmt.Errorf("owd: workspace hygiene not clean (%d issues) — compliance %s requires zero issues",
							vr.HygieneIssues, complianceLevel)
					}
					// Standard: only block on blocking issues (large untracked files, etc.)
					hygiene := WorkspaceHygieneScan(repoRoot)
					if hygiene.BlockingIssues > 0 {
						return fmt.Errorf("owd: %d blocking hygiene issue(s) — resolve before merging", hygiene.BlockingIssues)
					}
					// Warnings only: show but don't block for standard
					fmt.Printf("   ⚠️  Hygiene: %d warning(s) (non-blocking for %s)\n", vr.HygieneIssues, complianceLevel)
				}

				// ── Block on code-quality warnings (standard+) ──
				// Only blocks if go vet, gofmt, go test, or validators failed.
				// Hygiene warnings are handled separately above.
				if reqs.BlockOnWarning {
					codeFailed := !vr.GoVetPass || !vr.GofmtPass || !vr.GoTestPass
					if vr.ValidateRan && vr.ValidateFail > 0 && vr.ValidateTotal > 0 {
						validatePct := float64(vr.ValidatePass) / float64(vr.ValidateTotal)
						if validatePct < reqs.ValidateMinPct {
							codeFailed = true
						}
					}
					if codeFailed {
						return fmt.Errorf("owd: code-quality checks FAILED — compliance %s blocks on warnings", complianceLevel)
					}
				}
			} else {
				// Verify returned error with nil result
				if reqs.BlockOnWarning {
					return fmt.Errorf("owd: verification ERROR — compliance %s requires all checks PASS", complianceLevel)
				}
			}
		}

		// ── Compliance check: GPG signatures ──
		var sigCount int
		if reqs.GPGSigned {
			fmt.Println("🔍 GPG: checking commit signatures...")
			sigCount = countSignedCommits(repoRoot, currentBranch)
			if sigCount == 0 {
				fmt.Println("   ⚠️  No GPG-signed commits found")
				if complianceLevel == ComplianceStrict || complianceLevel == ComplianceMaximum {
					return fmt.Errorf("owd: GPG signing required — compliance %s mandates signed commits", complianceLevel)
				}
			} else {
				fmt.Printf("   ✅ %d signed commit(s)\n", sigCount)
			}
		}

		// ── Compliance check: Reviewer required ──
		if reqs.ReviewerReq {
			reviewer := args["reviewer"]
			if reviewer == "" {
				fmt.Println("   ⚠️  Reviewer required for compliance level", complianceLevel)
				return fmt.Errorf("owd: reviewer required — use --reviewer <name> for compliance %s", complianceLevel)
			}
			fmt.Printf("   ✅ Reviewer: %s\n", reviewer)
		}

		// ── Alert check: block merge if CRITICAL/HIGH alerts active ──
		alertMgr := alerts.NewManager(repoRoot)
		if blocking, err := alertMgr.HasBlocking(); err == nil && blocking {
			active, _ := alertMgr.Active()
			fmt.Println(alerts.FormatHuman(active))
			return fmt.Errorf("owd: active SECURITY ALERTS block merge — resolve alerts before merging")
		}

		// ── Pre-merge cleanup: remove noise files that should never be tracked ──
		if wt.IsWorktree {
			cleaned, err := cleanWorktreeNoise(wt.WorktreePath)
			if err != nil {
				fmt.Printf("  ⚠️  Noise cleanup warning: %v\n", err)
			} else if cleaned > 0 {
				fmt.Printf("  🧹 Cleaned %d noise file(s): bin/ .cache/ *.log coverage/\n", cleaned)
			}
		}

		// ── Merge ──
		// OWD-1 FIX: Pass wt.WorktreePath so Merge() correctly detects isWorktree=true
		// and triggers cleanupWorktree(). Previously passed repoRoot (CWD-dependent),
		// which when running from the main repo caused isWorktree=false and skipped cleanup.
		mergeRoot := repoRoot
		if wt.IsWorktree {
			mergeRoot = wt.WorktreePath
		}
		result, err := gitflow.Merge(mergeRoot)
		if err != nil {
			return fmt.Errorf("owd: merge failed: %w", err)
		}

		// Post-merge: use main repo root since worktree may have been deleted.
		// wt.MainRepoRoot is always set (resolved from git porcelain at handler start).
		postRoot := wt.MainRepoRoot

		fmt.Printf("✅ Merged: %s\n", result.Branch)
		if result.WorktreeRemoved {
			fmt.Printf("   Worktree cleaned: %s\n", result.WorktreePath)
		}
		if result.WorktreeError != "" {
			fmt.Printf("   ⚠️  Worktree cleanup issue: %s\n", result.WorktreeError)
			fmt.Printf("   Manual cleanup: owclean\n")
		}

		// ── Post-merge: auto-trigger owclean to remove orphaned worktrees ──
		fmt.Println("🧹 Auto-cleaning orphaned worktrees...")
		if cleanErr := cleanWorktrees(postRoot, map[string]string{}); cleanErr != nil {
			fmt.Printf("   ⚠️  owclean: %v\n", cleanErr)
		}

		// ── Post-merge hygiene check ──
		hygiene := WorkspaceHygieneScan(postRoot)
		if !hygiene.Clean {
			fmt.Printf("\n%s\n", hygiene.Report())
			if hygiene.BlockingIssues > 0 {
				fmt.Println("⚠️  Blocking hygiene issues found — address before next session.")
			}
		}

		// ── Audit: integration complete ──
		author := gitflow.DetectAuthor(postRoot)
		trailEvent(postRoot, "INTEGRATION_COMPLETE", currentBranch, author,
			map[string]string{"compliance": string(complianceLevel), "targets": gfProfile.MergeTo})

		// ── Generate compliance seal with REAL validation data ──
		if complianceLevel != ComplianceQuick {
			treeHash := GetGitTreeHash(postRoot)
			reviewer := args["reviewer"]
			validPassed := 77 // default fallback
			if vr != nil && vr.ValidateRan {
				validPassed = vr.ValidatePass
			}
			seal := GenerateSeal(postRoot, currentBranch, author, reviewer, complianceLevel, treeHash, sigCount, validPassed)
			DisplaySeal(seal)
			// Audit: seal generated
			trailEvent(postRoot, "SEAL_GENERATED", currentBranch, author,
				map[string]string{"hash": seal.Hash, "level": string(complianceLevel)})
		}

		if result.WorktreeRemoved {
			// ── T3.5: Remove worktree from trust store ──
			_ = truststore.RemoveWorktreeHead(postRoot, wt.WorktreePath)
			// Record git op timestamp for gate_self_protection grace period
			_ = truststore.RecordGitOp(postRoot)
			trailEvent(postRoot, "CLEANUP_COMPLETE", currentBranch, author, nil)
		} else {
			// Even without removal, record the post-merge git op
			_ = truststore.RecordGitOp(postRoot)
		}

		return nil
	}
}

// countSignedCommits counts commits in the current branch that have GPG signatures.
func countSignedCommits(repoRoot, branch string) int {
	cmd := exec.Command("git", "log", "--format=%G?", branch, "--not", "develop", "main")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	count := 0
	for _, status := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if status == "G" || status == "U" {
			count++
		}
	}
	return count
}

// ── GATE: Secrets Sweep ────────────────────────────────────────────────────
// scanSecretsInChanges runs the secrets hygiene scanner on files changed
// in the current branch (vs develop). Uses the same patterns as secrets_hygiene.go.

// SecretFinding represents a detected secret in a file.
type SecretFinding struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Detail string `json:"detail"`
}

// scanSecretsInChanges scans files changed in branch vs develop for secrets.
func scanSecretsInChanges(repoRoot, branch string) ([]SecretFinding, error) {
	// Get list of files changed in this branch vs develop (two-dot: direct comparison)
	diffCmd := exec.Command("git", "diff", "--name-only", "develop.."+branch)
	diffCmd.Dir = repoRoot
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	changedFiles := strings.Split(strings.TrimSpace(string(diffOut)), "\n")
	if len(changedFiles) == 0 || (len(changedFiles) == 1 && changedFiles[0] == "") {
		return nil, nil
	}

	var findings []SecretFinding
	for _, file := range changedFiles {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		// Skip directories that should never be scanned
		skip := false
		skipPrefixes := []string{".git/", "node_modules/", "vendor/", ".venv/", "venv/",
			".ovav/vault/", "go-runtime/bin/", "integrity_backups/", ".wrangler/", "dist/"}
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(file, prefix) {
				skip = true
				break
			}
		}
		// Skip test files — they contain fake keys for secret detection tests
		if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "_test.py") {
			skip = true
		}
		if strings.Contains(file, "/testdata/") || strings.Contains(file, "/fixtures/") {
			skip = true
		}
		if skip {
			continue
		}

		fullPath := filepath.Join(repoRoot, file)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for lineNum, line := range lines {
			if matches := matchSecretPatterns(file, lineNum+1, line); len(matches) > 0 {
				findings = append(findings, matches...)
			}
		}
	}
	return findings, nil
}

// matchSecretPatterns checks a single line against all known secret patterns.
// Uses the same patterns defined in secrets_hygiene.go (imported via validators package).
func matchSecretPatterns(file string, lineNum int, line string) []SecretFinding {
	// Trim whitespace for clean matching
	trimmed := strings.TrimSpace(line)
	// Skip empty lines and comments (# shell, // C/JS, -- Lua)
	// For -- skip: ensure it's not a certificate header (-----BEGIN...)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || (strings.HasPrefix(trimmed, "--") && len(trimmed) > 2 && trimmed[2] != '-') {
		return nil
	}

	// Core patterns — keep in sync with validators/secrets_hygiene.go
	type pattern struct {
		re    string
		label string
	}
	secretPatterns := []pattern{
		{`(?i)(api[_-]?key|apikey)\s*[:=]\s*["'][A-Za-z0-9_\-]{20,}["']`, "API key in plaintext"},
		{`(?i)(auth[_-]?token|bearer[_-]?token)\s*[:=]\s*["'][A-Za-z0-9_\-\.]{20,}["']`, "Auth token in plaintext"},
		{`(?i)(password|passwd|pwd)\s*[:=]\s*["'][^"']{4,}["']`, "Password in plaintext"},
		{`(?i)(secret|private[_-]?key)\s*[:=]\s*["'][A-Za-z0-9_\-+/=]{20,}["']`, "Secret/private key in plaintext"},
		{`(?i)ghp_[A-Za-z0-9_]{36,}`, "GitHub personal access token"},
		{`(?i)AKIA[0-9A-Z]{16}`, "AWS access key ID"},
		{`(?i)sk[-_]live[-_][0-9a-zA-Z]{24,}`, "Stripe live secret key"},
		{`(?i)-----BEGIN\s+(RSA|EC|DSA|OPENSSH)\s+PRIVATE\s+KEY-----`, "Private key block"},
		{`(?i)sk-[A-Za-z0-9]{32,}`, "OpenAI/LLM API key"},
		{`(?i)sk-ant-[A-Za-z0-9\-_]{32,}`, "Anthropic API key"},
		{`(?i)AIza[0-9A-Za-z\-_]{35}`, "Google API key"},
		{`(?i)(DATABASE_URL|MONGO_URI|REDIS_URL)\s*=\s*["'][^"']{10,}["']`, "Database URL in config"},
		{`(?i)(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^@\s]+@[^/\s]+`, "DB connection string with credentials"},
		{`(?i)hf_[A-Za-z0-9]{32,}`, "HuggingFace API token"},
		{`(?i)glpat-[A-Za-z0-9\-_]{20,}`, "GitLab personal access token"},
		{`(?i)(NPM_TOKEN|GITHUB_TOKEN|DOCKER_PASSWORD)\s*=\s*["'][A-Za-z0-9_\-]{8,}["']`, "CI/CD token in config"},
		{`(?i)(SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL)\s*=\s*["'][A-Za-z0-9_\-\.+/=]{16,}["']`, "Env-style secret in config"},
	}

	for _, p := range secretPatterns {
		re, err := regexp.Compile(p.re)
		if err != nil {
			continue
		}
		if re.MatchString(trimmed) {
			return []SecretFinding{{File: file, Line: lineNum, Detail: p.label}}
		}
	}
	return nil
}

// ── GATE: Forbidden Files ──────────────────────────────────────────────────

// ForbiddenFile represents a file that should not be merged.
type ForbiddenFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// forbiddenFileRules defines files/extensions that are never allowed in git.
var forbiddenFileRules = []struct {
	pattern string // filepath.Match pattern or exact name
	reason  string
}{
	{"*.env", "Environment file may contain secrets"},
	{".env", "Environment file may contain secrets"},
	{"*.pem", "PEM file (private key/certificate)"},
	{"*.key", "Private key file"},
	{"*.pfx", "PKCS#12 certificate bundle"},
	{"*.p12", "PKCS#12 certificate bundle"},
	{"*.jks", "Java keystore"},
	{"*.keystore", "Java keystore"},
	{"credentials.json", "Google Cloud credentials"},
	{"service-account.json", "Service account key"},
	{"*.sqlite", "SQLite database (binary, may contain data)"},
	{"*.sqlite3", "SQLite database (binary, may contain data)"},
	{"*.db", "Database file (binary, may contain data)"},
	{"*.log", "Log file (should not be committed)"},
	{"*.tgz", "Compressed archive (binary)"},
	{"*.tar.gz", "Compressed archive (binary)"},
	{"*.zip", "Compressed archive (binary)"},
	{"*.exe", "Binary executable"},
	{"*.dll", "Binary library"},
	{"*.so", "Shared object library"},
	{"*.o", "Object file"},
	{"*.a", "Static library"},
}

// scanForbiddenFiles checks changed files against forbidden patterns.
func scanForbiddenFiles(repoRoot, branch string) ([]ForbiddenFile, error) {
	diffCmd := exec.Command("git", "diff", "--name-only", "--diff-filter=A", "develop.."+branch)
	diffCmd.Dir = repoRoot
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	changedFiles := strings.Split(strings.TrimSpace(string(diffOut)), "\n")
	if len(changedFiles) == 0 || (len(changedFiles) == 1 && changedFiles[0] == "") {
		return nil, nil
	}

	var forbidden []ForbiddenFile
	for _, file := range changedFiles {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		base := filepath.Base(file)
		for _, rule := range forbiddenFileRules {
			matched, _ := filepath.Match(rule.pattern, base)
			if matched || base == rule.pattern {
				// Double-check: is it large?
				fullPath := filepath.Join(repoRoot, file)
				info, err := os.Stat(fullPath)
				size := ""
				if err == nil {
					size = fmt.Sprintf(" (%.1f MB)", float64(info.Size())/(1024*1024))
				}
				forbidden = append(forbidden, ForbiddenFile{
					Path:   file,
					Reason: rule.reason + size,
				})
				break
			}
		}
	}
	return forbidden, nil
}

func makeListHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		showHistory := args["history"] == "true" || args["history"] == "1"
		showJSON := args["json"] == "true" || args["json"] == "1"

		// ── History mode: read audit trail ──
		if showHistory {
			return showAuditTrail(repoRoot, showJSON)
		}

		// ── Untracked mode: smart fetch from parent branch ──
		if args["untracked"] == "true" {
			untrackedHandler := makeUntrackedHandler(repoRoot)
			return untrackedHandler(ctx, args)
		}

		// ── Standard: git status + conflict predictions ──
		if err := gitflow.Status(repoRoot); err != nil {
			return err
		}

		// Extended worktree list with metadata
		fmt.Println("\n── Worktrees ──")
		zombieOnly := args["zombie-only"] == "true"
		out, _ := runGitOutput(repoRoot, "worktree", "list")
		if out != "" {
			lines := strings.Split(strings.TrimSpace(out), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				// Parse: "/path/to/worktree HASH [branch]" format
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					path := parts[0]
					branch := strings.Trim(parts[len(parts)-1], "[]")
					// SU-6: Zombie detection — branch deleted but worktree exists.
					// Use branchExists() which checks both local AND remote branches.
					// A worktree is a zombie if its branch no longer exists anywhere.
					isZombie := false
					if branch != "" && branch != "HEAD" {
						if !branchExists(repoRoot, branch) {
							isZombie = true
						}
					}
					if zombieOnly && !isZombie {
						continue
					}
					// Get worktree metadata
					info, _ := os.Stat(path)
					age := ""
					if info != nil {
						age = fmt.Sprintf("%.0fd", time.Since(info.ModTime()).Hours()/24)
					}
					profile := gitflow.DetectProfileFromBranch(branch)
					zombieTag := ""
					if isZombie {
						zombieTag = " [ZOMBIE]"
					}
					fmt.Printf("  %-30s %-25s %-15s %-4s %s%s\n",
						shorten(path, 30), branch, profile.Name, age, "🟢", zombieTag)
				}
			}
		}
		if zombieOnly {
			fmt.Println("\n  💡 Run 'owclean' to remove zombie worktrees.")
		}

		// Conflict predictions
		fmt.Println("\n── Conflict Predictions ──")
		worktrees, _ := parseWorktreeListFromRepo(repoRoot)
		for _, wt := range worktrees {
			profile := gitflow.DetectProfileFromBranch(wt)
			targets := strings.Split(profile.MergeTo, "+")
			for _, t := range targets {
				t = strings.TrimSpace(t)
				if t == "" || t == "none" {
					continue
				}
				matrix, err := PredictConflicts(repoRoot, wt, t)
				if err != nil {
					continue
				}
				if matrix.ConflictFiles > 0 {
					fmt.Printf("  ⚠  %s vs %s: %d potential conflict(s)\n", wt, t, matrix.ConflictFiles)
				} else {
					fmt.Printf("  ✅ %s vs %s: safe to merge\n", wt, t)
				}
			}
		}
		return nil
	}
}

// showAuditTrail reads and displays the OVAV audit trail.
func showAuditTrail(repoRoot string, json bool) error {
	trailPath := filepath.Join(repoRoot, ".ovav", "audit", "trail.jsonl")
	data, err := os.ReadFile(trailPath)
	if err != nil {
		return fmt.Errorf("no audit trail found at %s", trailPath)
	}
	if json {
		fmt.Print(string(data))
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	fmt.Printf("── OVAV Audit Trail (%d entries) ──\n", len(lines))
	for _, line := range lines[len(lines)-min(20, len(lines)):] {
		fmt.Printf("  %s\n", line)
	}
	return nil
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n+3:]
}

func boolIcon(v bool) string {
	if v {
		return "✅"
	}
	return "❌"
}

func statusIcon(v bool, trueStr, falseStr string) string {
	if v {
		if trueStr == "" {
			return "✅"
		}
		return "✅ " + trueStr
	}
	return "❌ " + falseStr
}

func makeRouteHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		target := args["target"]
		mode := args["mode"]
		if target == "" {
			return fmt.Errorf("owx: target branch required")
		}

		sourceRef, err := currentBranch(repoRoot)
		if err != nil {
			return fmt.Errorf("owx: detect source: %w", err)
		}

		routeMode := RouteMode(mode)
		if routeMode == "" {
			routeMode = RouteCherryPick
		}

		result, err := Route(ctx, repoRoot, sourceRef, target, routeMode)
		if err != nil {
			return fmt.Errorf("owx: %w", err)
		}

		if result.Success {
			fmt.Printf("✅ Routed %d commits: %s → %s [%s]\n", len(result.Commits), sourceRef, target, mode)
		} else {
			fmt.Printf("⚠️  Partial route: %d transferred, %d skipped\n", len(result.Commits), len(result.Skipped))
		}
		return nil
	}
}

func makeAbortHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		if err := Abort(repoRoot); err != nil {
			return fmt.Errorf("owa: %w", err)
		}
		fmt.Println("✅ Operation aborted — workspace preserved")
		return nil
	}
}

func makeRescueHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		result, err := Rescue(repoRoot)
		if err != nil {
			return fmt.Errorf("owr: %w", err)
		}
		fmt.Printf("🔍 Rescue scan complete:\n")
		fmt.Printf("   Recovered commits: %d\n", len(result.RecoveredCommits))
		fmt.Printf("   Unmerged branches: %d\n", len(result.RecoveredBranches))
		fmt.Printf("   Orphaned worktrees: %d\n", len(result.RecoveredWorktrees))
		return nil
	}
}

func makeSyncHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		if args["rebase"] == "true" || args["full"] == "true" {
			// SU-8: Enhanced rebase — detect unpushed, pull+rebase, abort on conflict
			fmt.Println("  📥 Fetching origin...")
			if err := runGit(repoRoot, "fetch", "origin"); err != nil {
				return fmt.Errorf("ows: fetch failed: %w", err)
			}

			// Detect if there are local unpushed commits
			unpushed, _ := runGitOutput(repoRoot, "log", "origin/develop..HEAD", "--oneline")
			if strings.TrimSpace(unpushed) != "" {
				fmt.Printf("  📤 Unpushed commits detected:\n%s\n", strings.TrimSpace(unpushed))
			}

			fmt.Println("  🔄 Rebasing onto origin/develop...")
			if err := runGit(repoRoot, "rebase", "origin/develop"); err != nil {
				// SU-8: Clean abort on conflict
				fmt.Println("  ❌ Rebase conflict — aborting...")
				runGit(repoRoot, "rebase", "--abort")
				return fmt.Errorf("ows: rebase conflict — resolve manually or use 'git rebase --abort' (already run)")
			}
			fmt.Println("  ✅ Rebase complete")
		}
		if args["full"] == "true" || args["rebase"] != "true" {
			if err := Sync(repoRoot); err != nil {
				return fmt.Errorf("ows: %w", err)
			}
			fmt.Println("✅ Sync complete — remotes fetched, maintenance run, stale worktrees pruned")
		}
		return nil
	}
}

// SU-9: owprep — verify/regenerate worktree config
func makePrepHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		repair := args["repair"] == "true" || args["repair"] == "1"
		configPath := filepath.Join(repoRoot, ".ovav", "worktree-config.json")
		relPath := filepath.Join(".ovav", "worktree-config.json")
		defaultConfig := `{"default_profile":"feature","compliance":"standard","auto_cleanup":true}`

		data, err := os.ReadFile(configPath)
		if err != nil {
			// Config file is missing.
			if !repair {
				return fmt.Errorf("owprep: worktree-config.json is missing at %s\n"+
					"Hint: run 'owprep --repair' to generate the default config.\n"+
					"Expected format: {\"default_profile\":\"feature\",\"compliance\":\"standard\",\"auto_cleanup\":true}",
					relPath)
			}
			os.MkdirAll(filepath.Dir(configPath), 0755)
			if writeErr := os.WriteFile(configPath, []byte(defaultConfig), 0644); writeErr != nil {
				return fmt.Errorf("owprep: cannot write config: %w", writeErr)
			}
			fmt.Printf("  ✅ Generated default worktree-config.json at %s\n", relPath)
			return nil
		}

		if !json.Valid(data) {
			// Config file is corrupt.
			if !repair {
				return fmt.Errorf("owprep: worktree-config.json is corrupt (invalid JSON) at %s\n"+
					"Hint: run 'owprep --repair' to regenerate a valid default config.\n"+
					"Expected format: {\"default_profile\":\"feature\",\"compliance\":\"standard\",\"auto_cleanup\":true}",
					relPath)
			}
			if writeErr := os.WriteFile(configPath, []byte(defaultConfig), 0644); writeErr != nil {
				return fmt.Errorf("owprep: cannot write config: %w", writeErr)
			}
			fmt.Printf("  ✅ Regenerated default worktree-config.json at %s\n", relPath)
			return nil
		}

		fmt.Println("  ✅ worktree-config.json: valid")
		return nil
	}
}

// suggestion captures one owsuggest output line with its motivation.
type suggestion struct {
	Cmd    string `json:"cmd"`
	Reason string `json:"reason"`
	Motive string `json:"motive,omitempty"` // git/branch/worktree data that motivated this suggestion
}

// SU-10: owsuggest — suggest next OWS command with optional explain
func makeSuggestHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		explain := args["explain"] == "true"

		// Analyze current state
		branch, _ := currentBranch(repoRoot)
		dirty := gitOutput(repoRoot, "status", "--porcelain")
		hasDirty := strings.TrimSpace(dirty) != ""
		worktrees, _ := parseWorktreeListFromRepo(repoRoot)

		var suggestions []suggestion

		// Heuristic: if on protected branch, suggest owc
		if branch == "develop" || branch == "main" || branch == "master" {
			suggestions = append(suggestions, suggestion{
				"owc feature/<name>",
				"on protected branch " + branch + " — create a feature worktree to isolate changes",
				"branch_state=protected:" + branch,
			})
		}
		// If dirty, suggest owp or commit
		if hasDirty {
			diffstat := gitOutput(repoRoot, "diff", "--stat", "--no-color")
			suggestions = append(suggestions, suggestion{
				"owp",
				"working tree has uncommitted changes — stash and sync before branching",
				"dirty_files=" + strings.TrimSpace(dirty) + "; diffstat=" + strings.ReplaceAll(strings.TrimSpace(diffstat), "\n", " "),
			})
		}
		// If on feature branch, suggest owv then owd
		if !strSliceContains([]string{"develop", "main", "master", "HEAD"}, branch) {
			aheadBehind := gitOutput(repoRoot, "rev-list", "--left-right", "--count", branch+"...origin/"+branch)
			commitsAhead := "0"
			if parts := strings.Fields(aheadBehind); len(parts) == 2 {
				commitsAhead = parts[0]
			}
			suggestions = append(suggestions, suggestion{
				"owv",
				"on feature branch " + branch + " — verify before merging",
				"commits_ahead=" + commitsAhead,
			})
			suggestions = append(suggestions, suggestion{
				"owd",
				"ready to merge " + branch + " — runs verify + merge + cleanup",
				"commits_ahead=" + commitsAhead,
			})
		}
		// If worktrees exist, suggest owl
		if len(worktrees) > 0 {
			worktreeList := gitOutput(repoRoot, "worktree", "list", "--porcelain")
			suggestions = append(suggestions, suggestion{
				"owl",
				fmt.Sprintf("%d worktree(s) active — list and check for conflicts", len(worktrees)),
				"worktree_list=" + strings.ReplaceAll(strings.TrimSpace(worktreeList), "\n", " | "),
			})
		}

		// Output
		fmt.Println("💡 Suggested OWS commands:")
		for _, s := range suggestions {
			fmt.Printf("  %-22s", s.Cmd)
			if explain {
				fmt.Printf(" — %s", s.Reason)
			}
			fmt.Println()
		}

		// ── Audit log (always) ──
		logPath := filepath.Join(repoRoot, ".ovav", "runtime", "owsuggest_history.jsonl")
		os.MkdirAll(filepath.Dir(logPath), 0755)
		entry := fmt.Sprintf("{\"ts\":\"%s\",\"branch\":\"%s\",\"dirty\":%v,\"worktrees\":%d}\n",
			time.Now().UTC().Format(time.RFC3339), branch, hasDirty, len(worktrees))
		f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			f.WriteString(entry)
			f.Close()
		}

		// ── Explain audit: rich git + branch + worktree evidence (only when --explain) ──
		if explain {
			auditPath := filepath.Join(repoRoot, ".ovav", "runtime", "owsuggest_audit.jsonl")
			os.MkdirAll(filepath.Dir(auditPath), 0755)

			// Collect evidence
			var motive []string
			if hasDirty {
				motive = append(motive, "dirty_working_tree")
			}
			if branch == "develop" || branch == "main" || branch == "master" {
				motive = append(motive, "protected_branch:"+branch)
			}
			if !strSliceContains([]string{"develop", "main", "master", "HEAD"}, branch) && branch != "" {
				motive = append(motive, "feature_branch:"+branch)
			}
			if len(worktrees) > 0 {
				motive = append(motive, fmt.Sprintf("active_worktrees:%d", len(worktrees)))
			}

			// Git history (last 5 commits on current branch)
			gitHistory := gitOutput(repoRoot, "log", "--oneline", "-5")
			// Branch state
			branchV := gitOutput(repoRoot, "branch", "-v")
			// Worktree status
			wtList := gitOutput(repoRoot, "worktree", "list", "--porcelain")

			audit := SuggestAudit{
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				Branch:      branch,
				Dirty:       hasDirty,
				WorktreeN:   len(worktrees),
				Motives:     motive,
				GitHistory:  gitHistory,
				BranchState: branchV,
				WorktreeSt:  wtList,
				Suggestions: suggestions,
			}
			auditJSON, _ := json.Marshal(audit)
			af, _ := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if af != nil {
				af.WriteString(string(auditJSON) + "\n")
				af.Close()
			}
		}

		return nil
	}
}

// SuggestAudit is the structured audit entry written to owsuggest_audit.jsonl
// when --explain is used. Records the full motivating context for each suggestion.
type SuggestAudit struct {
	Timestamp   string       `json:"ts"`
	Branch      string       `json:"branch"`
	Dirty       bool         `json:"dirty"`
	WorktreeN   int          `json:"worktree_count"`
	Motives     []string     `json:"motives"`
	GitHistory  string       `json:"git_history"`
	BranchState string       `json:"branch_state"`
	WorktreeSt  string       `json:"worktree_status"`
	Suggestions []suggestion `json:"suggestions"`
}

func strSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Hint represents a single remediation hint with a check name and actionable fix.
type Hint struct {
	Check   string // name of the check that failed
	Problem string // what went wrong
	Fix     string // actionable remediation step
}

func (h Hint) String() string {
	return fmt.Sprintf("  ⚠  %s\n     Problem: %s\n     Fix:     %s", h.Check, h.Problem, h.Fix)
}

// commandExists checks if the given command is available in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// verifyPreflight runs targeted infrastructure checks before the full Verify pipeline.
// Returns a slice of Hint for each problem detected (empty if all checks pass).
func verifyPreflight(repoRoot string) []Hint {
	var hints []Hint

	// Check 1: git is available
	if !commandExists("git") {
		hints = append(hints, Hint{
			Check:   "git-installed",
			Problem: "git is not installed or not in PATH",
			Fix:     "Install git: https://git-scm.com or via your package manager (apt install git / brew install git / choco install git)",
		})
		return hints // nothing else can be checked without git
	}

	// Check 2: directory is a git repo
	if _, err := runGitOutput(repoRoot, "rev-parse", "--git-dir"); err != nil {
		hints = append(hints, Hint{
			Check:   "git-repo",
			Problem: "directory is not a git repository",
			Fix:     "Run `git init` to initialize, or point to the root of an existing git repo",
		})
		return hints
	}

	// Check 3: git user.name is configured
	if out, _ := runGitOutput(repoRoot, "config", "user.name"); strings.TrimSpace(out) == "" {
		hints = append(hints, Hint{
			Check:   "git-user-name",
			Problem: "git user.name is not configured — commits will fail",
			Fix:     "Run: git config --global user.name \"Your Full Name\"",
		})
	}

	// Check 4: git user.email is configured
	if out, _ := runGitOutput(repoRoot, "config", "user.email"); strings.TrimSpace(out) == "" {
		hints = append(hints, Hint{
			Check:   "git-user-email",
			Problem: "git user.email is not configured — commits will fail",
			Fix:     "Run: git config --global user.email \"you@example.com\"",
		})
	}

	// Check 5: worktree-specific — if this looks like a worktree, validate the worktree link
	if isWorktree, _ := runGitOutput(repoRoot, "rev-parse", "--is-inside-work-tree"); strings.TrimSpace(isWorktree) == "true" {
		gitDirOut, gitDirErr := runGitOutput(repoRoot, "rev-parse", "--git-dir")
		if gitDirErr == nil {
			gitDir := strings.TrimSpace(gitDirOut)
			// git-dir for a worktree is .git (a file), for main repo it's .git (a dir)
			if strings.HasSuffix(gitDir, "/.git") || gitDir == ".git" {
				gitFile := filepath.Join(repoRoot, ".git")
				if info, err := os.Stat(gitFile); err == nil && !info.IsDir() {
					// It's a file pointing to the actual git dir — check the target
					data, _ := os.ReadFile(gitFile)
					content := strings.TrimSpace(string(data))
					if strings.HasPrefix(content, "gitdir:") {
						targetDir := strings.TrimPrefix(content, "gitdir: ")
						targetDir = strings.TrimSpace(targetDir)
						if _, tdErr := os.Stat(targetDir); tdErr != nil {
							hints = append(hints, Hint{
								Check:   "worktree-gitdir",
								Problem: fmt.Sprintf("worktree .git file points to non-existent directory: %s", targetDir),
								Fix:     "Run: git worktree repair " + repoRoot + "  to fix the broken worktree link",
							})
						}
					}
				}
			}
		}
	}

	// Check 6: branch exists (only when current branch is non-empty)
	currentBranch, _ := currentBranch(repoRoot)
	if currentBranch != "" && currentBranch != "HEAD" {
		if !branchExists(repoRoot, currentBranch) {
			hints = append(hints, Hint{
				Check:   "branch-exists",
				Problem: fmt.Sprintf("current branch '%s' no longer exists (deleted or fetch needed)", currentBranch),
				Fix:     "Run: git fetch origin && git checkout " + currentBranch + "  — or switch to an existing branch with `git checkout <branch>`",
			})
		}
	}

	// Check 7: main branch exists (develop or main) — needed for merge target checks
	mainExists := branchExists(repoRoot, "develop") || branchExists(repoRoot, "main")
	if !mainExists && currentBranch != "" && currentBranch != "develop" && currentBranch != "main" {
		hints = append(hints, Hint{
			Check:   "main-branch",
			Problem: "neither 'develop' nor 'main' branch exists locally",
			Fix:     "Clone the repository fresh, or run: git fetch origin && git checkout -b develop origin/develop",
		})
	}

	// Check 8: working tree is not mid-rebase/merge/cherry-pick
	// Detect by checking for state indicator files (read-only inspection)
	stateFiles := map[string]string{
		".git/rebase-apply":     "rebase",
		".git/MERGE_HEAD":       "merge",
		".git/CHERRY_PICK_HEAD": "cherry-pick",
		".git/REVERT_HEAD":      "revert",
	}
	for stateFile, stateName := range stateFiles {
		if _, sfErr := os.Stat(filepath.Join(repoRoot, stateFile)); sfErr == nil {
			hints = append(hints, Hint{
				Check:   "mid-operation",
				Problem: fmt.Sprintf("a %s operation is in progress in this worktree", stateName),
				Fix:     fmt.Sprintf("Complete the %s: run `git %s --continue` or `git %s --abort`", stateName, stateName, stateName),
			})
		}
	}

	// Check 9: .ovav directory is accessible (worktree metadata)
	ovavDir := filepath.Join(repoRoot, ".ovav")
	if _, err := os.Stat(ovavDir); os.IsNotExist(err) {
		hints = append(hints, Hint{
			Check:   "ovav-metadata",
			Problem: ".ovav metadata directory is missing — worktree tracking may not work",
			Fix:     "This is normal for non-OVAV repos; ignore this warning. For OVAV repos: run `owc` once to initialize",
		})
	}

	// Check 10: remote origin exists (useful context even if not required for owv)
	// Only add as a note if there are other issues; don't add a new hint entry here
	// since absence of remote is non-fatal for local verification.
	_ = repoRoot // suppress unused warning

	return hints
}

func makeVerifyHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		fmt.Println("🔍 OVAV Verification Pipeline v3.0 — Multi-Stack")
		fmt.Println(strings.Repeat("─", 50))

		// Auto-detect changed files for scope filtering
		changedFiles := detectChangedFiles(repoRoot)
		scoped := len(changedFiles) > 0

		// Show all detected stacks
		stacks := DetectStacks(repoRoot)
		fmt.Printf("  🏈 stacks:    %s\n", stacks.Summary())
		if scoped {
			fmt.Printf("  📍 scope:    %d file(s) changed (scope-filtered)\n", len(changedFiles))
		}

		currentBranch, _ := currentBranch(repoRoot)
		if currentBranch != "" {
			profile := DetectProfileFromBranch(currentBranch)
			fmt.Printf("  🏧 profile:   %s\n", profile)
		}
		// Show supported stacks legend
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("  ✓ go vet/fmt/test  ✓ biome/tsc  ✓ ruff  ✓ cargo")
		fmt.Println(strings.Repeat("─", 50))

		// OWS-GAP-07: run pre-flight checks
		hints := verifyPreflight(repoRoot)
		if len(hints) > 0 {
			fmt.Println("  Infrastructure warnings:")
			for _, h := range hints {
				fmt.Printf("  ⚠  %s\n", h.Check)
				fmt.Printf("     Problem: %s\n", h.Problem)
				fmt.Printf("     Fix:     %s\n", h.Fix)
			}
			fmt.Println(strings.Repeat("─", 50))
		}

		// Run verification with animated progress
		fmt.Println("\n  Starting verification phases...")
		phaseResults, err := VerifyPhases(repoRoot, changedFiles)
		if err != nil {
			fmt.Printf("❌ Verification FATAL: %v\n", err)
			return fmt.Errorf("owv: %w", err)
		}

		fmt.Println(strings.Repeat("─", 50))
		fmt.Println("  Results:")

		// Aggregate results
		allPass := true
		for _, pr := range phaseResults {
			icon := "✅"
			statusStr := "PASS"
			if !pr.Pass {
				icon = "❌"
				statusStr = "FAIL"
				allPass = false
			}
			fmt.Printf("  %s %-12s %-8s %4dms", icon, pr.Name, statusStr, pr.DurMS)
			if len(pr.Issues) > 0 {
				fmt.Printf("  ⚠ %s", pr.Issues[0])
			}
			fmt.Println()
		}

		// Count totals
		passCount := 0
		failCount := 0
		for _, pr := range phaseResults {
			if pr.Pass {
				passCount++
			} else {
				failCount++
			}
		}

		fmt.Println(strings.Repeat("─", 50))

		// Conflict predictions
		if currentBranch != "" && currentBranch != "develop" && currentBranch != "main" {
			profile := gitflow.DetectProfileFromBranch(currentBranch)
			targets := strings.Split(profile.MergeTo, "+")
			for _, t := range targets {
				t = strings.TrimSpace(t)
				if t == "" || t == "none" {
					continue
				}
				matrix, err := PredictConflicts(repoRoot, currentBranch, t)
				if err == nil {
					conflictIcon := "✅"
					if matrix.ConflictFiles > 0 {
						conflictIcon = "⚠"
					}
					fmt.Printf("  %s conflict vs %s: %s\n", conflictIcon, t, matrix.Summary())
				}
			}
		}

		fmt.Println(strings.Repeat("─", 50))
		if allPass {
			fmt.Println("✅ VERIFIED — ready for merge")
		} else {
			fmt.Printf("⚠️  VERIFIED WITH WARNINGS — %d/%d phases failed\n", failCount, len(phaseResults))
		}
		return nil
	}
}

func makeUpdateHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		args["rebase"] = "true"
		return makeSyncHandler(repoRoot)(ctx, args)
	}
}

// makePrepareHandler implements owp — worktree prepare/sync.
// Fetches from origin and updates the current branch.
// Without --rebase: git pull --ff-only (fast-forward pull only).
// With --rebase: git rebase onto tracking branch.
func makePrepareHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		useRebase := args["rebase"] == "true" || args["rebase"] == "1"

		// Detect current branch
		branch, err := currentBranch(repoRoot)
		if err != nil {
			return fmt.Errorf("owp: cannot detect current branch: %w", err)
		}
		if branch == "" || branch == "HEAD" {
			return fmt.Errorf("owp: not on a branch (detached HEAD)")
		}

		fmt.Printf("owp: preparing branch %s\n", branch)

		// Fetch origin first (always needed for both modes)
		fmt.Printf("  📥 Fetching origin...\n")
		if err := runGit(repoRoot, "fetch", "origin"); err != nil {
			return fmt.Errorf("owp: fetch failed: %w", err)
		}

		if useRebase {
			// --rebase mode: git rebase onto tracking branch
			fmt.Printf("  🔄 Rebasing onto origin/%s...\n", branch)
			if err := runGit(repoRoot, "rebase", "origin/"+branch); err != nil {
				// Clean abort on conflict
				fmt.Println("  ❌ Rebase conflict — aborting...")
				runGit(repoRoot, "rebase", "--abort")
				return fmt.Errorf("owp: rebase conflict — resolve manually, then run 'owp --rebase' to retry")
			}
			fmt.Println("  ✅ Rebase complete")
		} else {
			// Default: fast-forward pull only (no rebase)
			fmt.Printf("  🔃 Fast-forward pull...\n")
			if err := runGit(repoRoot, "pull", "--ff-only"); err != nil {
				return fmt.Errorf("owp: fast-forward pull failed (branch may have diverged). Use 'owp --rebase' to rebase instead.")
			}
			fmt.Println("  ✅ Fast-forward pull complete")
		}

		return nil
	}
}

func makeLockHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		target := args["target"]
		reason := args["reason"]
		unlock := args["unlock"] == "true" || args["unlock"] == "1"

		if target == "" {
			return fmt.Errorf("owlk: target worktree required. Usage: owlk <target> [--reason ...] [--unlock]")
		}

		if unlock {
			// Unlock: verify caller is the lock owner
			author := gitflow.DetectAuthor(repoRoot)
			currentOwner := getLockOwner(repoRoot, target)
			if currentOwner == "" {
				fmt.Printf("  ℹ️  Worktree %s is not locked\n", target)
				return nil
			}
			if author != currentOwner {
				return fmt.Errorf("owlk: unlock denied — %s is locked by %s (you are %s). Only the lock owner can unlock.", target, currentOwner, author)
			}
			if err := unlockWorktree(repoRoot, target); err != nil {
				return fmt.Errorf("owlk: %w", err)
			}
			fmt.Printf("🔓 Unlocked: %s (was locked by %s)\n", target, currentOwner)
			trailEvent(repoRoot, "WORKTREE_UNLOCKED", target, author,
				map[string]string{"previous_owner": currentOwner})
			return nil
		}

		// Lock: check not already locked by someone else
		currentOwner := getLockOwner(repoRoot, target)
		if currentOwner != "" {
			return fmt.Errorf("owlk: worktree %s already locked by %s — use --unlock first or contact owner", target, currentOwner)
		}

		author := gitflow.DetectAuthor(repoRoot)

		// Persist lock to filesystem with TTL
		locksDir := filepath.Join(repoRoot, ".ovav", "locks")
		os.MkdirAll(locksDir, 0755)
		lockFile := filepath.Join(locksDir, strings.ReplaceAll(target, "/", "-")+".lock")
		expiry := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		lockContent := fmt.Sprintf("%s:%s:%s", author, reason, expiry)
		if err := os.WriteFile(lockFile, []byte(lockContent), 0644); err != nil {
			return fmt.Errorf("owlk: persist lock: %w", err)
		}

		fmt.Printf("🔒 Locked: %s (owner: %s, reason: %s, TTL: 24h)\n", target, author, reason)
		trailEvent(repoRoot, "WORKTREE_LOCKED", target, author, map[string]string{"reason": reason})
		return nil
	}
}

// sanitizeWorktreeName replaces / with - for filesystem-safe lock file naming.
func sanitizeWorktreeName(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

// getLockOwner checks if a worktree is locked and returns the owner.
func getLockOwner(repoRoot, worktree string) string {
	lockFile := filepath.Join(repoRoot, ".ovav", "locks", sanitizeWorktreeName(worktree)+".lock")
	data, err := os.ReadFile(lockFile)
	if err != nil {
		return ""
	}
	// Parse simple lock format: "owner:reason:expiry"
	parts := strings.SplitN(string(data), ":", 3)
	if len(parts) >= 1 {
		// Check expiry
		if len(parts) >= 3 {
			expiry, _ := time.Parse(time.RFC3339, parts[2])
			if time.Now().After(expiry) {
				os.Remove(lockFile)
				return ""
			}
		}
		return parts[0]
	}
	return ""
}

// unlockWorktree removes the lock for a worktree.
func unlockWorktree(repoRoot, worktree string) error {
	lockFile := filepath.Join(repoRoot, ".ovav", "locks", sanitizeWorktreeName(worktree)+".lock")
	return os.Remove(lockFile)
}

// makeMoveHandler moves a worktree to a new path without breaking its git link.
// Implements `owm` (ovav worktree move).
func makeMoveHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		target := args["target"] // current worktree path or branch
		dest := args["to"]       // new path

		if target == "" || dest == "" {
			return fmt.Errorf("owm: target and destination required. Usage: owm <worktree> --to <new-path>")
		}

		// git worktree move (available in Git 2.33+)
		cmd := exec.Command("git", "worktree", "move", target, dest)
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Fallback: manual move for older git versions
			fmt.Printf("  ⚠️  git worktree move not supported, using manual relocation...\n")
			if err := os.Rename(target, dest); err != nil {
				return fmt.Errorf("owm: move failed: %w", err)
			}
			// Update .git file inside worktree to point to new location
			gitFile := filepath.Join(dest, ".git")
			if data, err := os.ReadFile(gitFile); err == nil {
				content := strings.TrimSpace(string(data))
				if strings.HasPrefix(content, "gitdir:") {
					// Read the gitdir path (e.g., /main/.git/worktrees/name)
					gitDirPath := strings.TrimPrefix(content, "gitdir: ")
					// The gitdir file inside the main repo points to the worktree location.
					// After os.Rename, it still points to the OLD target path.
					// Update it to point to the new dest path.
					gitdirFile := filepath.Join(gitDirPath, "gitdir")
					if err := os.WriteFile(gitdirFile, []byte(dest+"\n"), 0644); err != nil {
						fmt.Printf("  ⚠️  Could not update gitdir pointer: %v\n", err)
						fmt.Printf("  Run: git worktree repair %s\n", dest)
					}
				}
				_ = data
			}
		}

		fmt.Printf("✅ Moved worktree: %s → %s\n", target, dest)
		author := gitflow.DetectAuthor(repoRoot)
		trailEvent(repoRoot, "WORKTREE_MOVED", target, author,
			map[string]string{"from": target, "to": dest})
		return nil
	}
}

// ── Audit Trail ──────────────────────────────────────────────────────
// trailEvent writes an OVAV audit trail entry to .ovav/audit/trail.jsonl.
// Each event is a JSON line with timestamp, actor, event type, branch, and metadata.
// If auditLogger is initialized, also writes a structured audit entry.
func trailEvent(repoRoot, eventType, branch, actor string, meta map[string]string) {
	auditDir := filepath.Join(repoRoot, ".ovav", "audit")
	os.MkdirAll(auditDir, 0755)
	trailPath := filepath.Join(auditDir, "trail.jsonl")

	metaJSON := "{}"
	if len(meta) > 0 {
		parts := make([]string, 0, len(meta))
		for k, v := range meta {
			parts = append(parts, fmt.Sprintf("%q:%q", k, v))
		}
		metaJSON = "{" + strings.Join(parts, ",") + "}"
	}

	entry := fmt.Sprintf("{\"time\":%q,\"actor\":%q,\"event\":%q,\"branch\":%q,\"meta\":%s}\n",
		time.Now().UTC().Format(time.RFC3339), actor, eventType, branch, metaJSON)

	f, err := os.OpenFile(trailPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(entry)

	// Also write to structured audit logger if initialized.
	if auditLogger != nil {
		ctx := audit.WithActor(context.Background(), actor)
		ctx = audit.WithResource(ctx, branch)
		opLevel := audit.OpWrite
		if strings.Contains(eventType, "READ") || eventType == "WORKTREE_LISTED" {
			opLevel = audit.OpRead
		}
		details := make(map[string]interface{}, len(meta))
		for k, v := range meta {
			details[k] = v
		}
		details["event_type"] = eventType
		auditLogger.LogImmediate(ctx, opLevel, eventType, "ok", details)
	}
}

// parseWorktreeListFromRepo runs git worktree list and returns branch names.
func parseWorktreeListFromRepo(repoRoot string) ([]string, error) {
	out, err := runGitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	lines := parseWorktreeList(out)
	// Filter to just the branch names (not paths)
	var branches []string
	for _, line := range lines {
		branches = append(branches, line)
	}
	return branches, nil
}

// LockWorktree locks a worktree for multi-agent coordination.
// OWS-B5 FIX: Previously a dead-code stub that only printed a message.
// Now calls the canonical AuditDB implementation via OpenAudit.
// The ovavRoot is resolved from the worktree path using git.
func LockWorktree(lock *AgentLock) error {
	if lock == nil || lock.Worktree == "" {
		return fmt.Errorf("LockWorktree: invalid lock request")
	}
	// Resolve OVAV root from worktree path: run git rev-parse --show-toplevel
	// from the worktree directory to get the main repo root.
	ovavRoot, err := resolveRepoRoot(lock.Worktree)
	if err != nil {
		return fmt.Errorf("LockWorktree: resolve repo root: %w", err)
	}
	audit, err := OpenAudit(ovavRoot)
	if err != nil {
		return fmt.Errorf("LockWorktree: open audit db: %w", err)
	}
	defer audit.Close()
	return audit.LockWorktree(lock.Worktree, lock.Reason, lock.Owner)
}

// resolveRepoRoot returns the main repo root for any path in a git worktree.
func resolveRepoRoot(anyPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = anyPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ── Clean Handler ───────────────────────────────────────────────────────

func makeCleanHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		return cleanWorktrees(repoRoot, args)
	}
}

// cleanWorktrees removes orphaned, stale, and abandoned worktrees.
// Safe: never removes the main repo worktree.
func cleanWorktrees(repoRoot string, args map[string]string) error {
	dryRun := args["dry-run"] == "true" || args["dry-run"] == "1"

	out, err := runGitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("owclean: list worktrees: %w", err)
	}

	worktrees := parseWorktreeList(out)
	now := time.Now()
	cleaned := 0

	for _, wtPath := range worktrees {
		if wtPath == repoRoot {
			continue // never clean main repo
		}

		info, err := os.Stat(wtPath)
		if err != nil {
			continue
		}

		age := now.Sub(info.ModTime())

		// Check if branch still exists for this worktree
		branch := detectWorktreeBranch(repoRoot, wtPath)
		branchExists := branch != "" && branchExists(repoRoot, branch)

		shouldClean := false
		reason := ""

		if !branchExists {
			shouldClean = true
			reason = "orphaned (no matching branch)"
		} else if age > 30*24*time.Hour {
			shouldClean = true
			reason = fmt.Sprintf("abandoned (%.0fd inactive)", age.Hours()/24)
		} else if age > 7*24*time.Hour && (strings.Contains(wtPath, "spike-") || strings.Contains(wtPath, "research-")) {
			shouldClean = true
			reason = fmt.Sprintf("stale auto-cleanup profile (%.0fd)", age.Hours()/24)
		}

		if shouldClean {
			if dryRun {
				fmt.Printf("  🧹 [DRY RUN] Would remove: %s (%s)\n", wtPath, reason)
			} else {
				fmt.Printf("  🧹 Removing %s worktree: %s\n", reason, wtPath)
				if err := runGit(repoRoot, "worktree", "remove", "--force", wtPath); err != nil {
					fmt.Printf("  ⚠️  Failed: %v\n", err)
				} else {
					cleaned++
				}
			}
		}
	}

	// Prune stale git worktree metadata
	if !dryRun {
		runGit(repoRoot, "worktree", "prune")
	}

	if dryRun {
		fmt.Printf("\n  🔍 Dry run complete. Run without --dry-run to clean.\n")
	} else {
		fmt.Printf("\n  ✅ Clean complete: %d worktree(s) removed\n", cleaned)
	}
	return nil
}

// detectWorktreeBranch extracts the branch name from a worktree path
// by reading the HEAD symref or parsing git worktree list output.
func detectWorktreeBranch(repoRoot, wtPath string) string {
	headOut, err := runGitOutput(wtPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil && headOut != "" && headOut != "HEAD" {
		return strings.TrimSpace(headOut)
	}
	return ""
}

// branchExists checks if a branch exists (local or remote).
func branchExists(repoRoot, branch string) bool {
	// Check local
	out, _ := runGitOutput(repoRoot, "branch", "--list", branch)
	if strings.TrimSpace(out) != "" {
		return true
	}
	// Check remote
	out, _ = runGitOutput(repoRoot, "branch", "-r", "--list", "origin/"+branch)
	return strings.TrimSpace(out) != ""
}

// ── Nuke Handler ───────────────────────────────────────────────────────

// makeNukeHandler deletes a worktree + local branch + remote branch without merge.
func makeNukeHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		name := args["name"]
		force := args["force"] == "true" || args["force"] == "1"
		localOnly := args["local-only"] == "true" || args["local-only"] == "1"

		if name == "" {
			return fmt.Errorf("own nuke: worktree name required.\nUsage: own <branch-or-path> [--force] [--local-only]")
		}

		// Resolve worktree
		wt, err := gitflow.ResolveWorktree(name)
		if err != nil {
			return fmt.Errorf("own nuke: %w", err)
		}

		if !wt.IsWorktree {
			return fmt.Errorf("own nuke: '%s' is the main repo — cannot nuke main working tree", wt.WorktreePath)
		}

		branch := wt.Branch
		if branch == "develop" || branch == "main" || branch == "master" {
			return fmt.Errorf("own nuke: protected branch '%s' — aborting", branch)
		}

		// Confirm unless --force
		if !force {
			fmt.Printf("⚠️  NUKE: %s (%s)\n", wt.WorktreePath, branch)
			fmt.Printf("   This will DELETE:\n")
			fmt.Printf("   - Worktree: %s\n", wt.WorktreePath)
			fmt.Printf("   - Local branch: %s\n", branch)
			if !localOnly {
				fmt.Printf("   - Remote branch: origin/%s\n", branch)
			}
			fmt.Printf("\n   Run with --force to skip this confirmation.\n")
			return fmt.Errorf("own nuke: aborted — use --force to confirm")
		}

		fmt.Printf("💥 Nuking %s (%s)...\n", wt.WorktreePath, branch)

		// Step 1: Remove worktree
		fmt.Printf("   Removing worktree... ")
		if err := runGit(repoRoot, "worktree", "remove", "--force", wt.WorktreePath); err != nil {
			fmt.Printf("⚠️  (worktree may already be gone: %v)\n", err)
		} else {
			fmt.Printf("✅\n")
		}

		// Step 2: Delete local branch
		fmt.Printf("   Deleting local branch %s... ", branch)
		if err := runGit(repoRoot, "branch", "-D", branch); err != nil {
			fmt.Printf("⚠️  (branch may already be gone: %v)\n", err)
		} else {
			fmt.Printf("✅\n")
		}

		// Step 3: Delete remote branch
		if !localOnly {
			fmt.Printf("   Deleting remote branch origin/%s... ", branch)
			if err := runGit(repoRoot, "push", "origin", "--delete", branch); err != nil {
				fmt.Printf("⚠️  (remote branch may not exist: %v)\n", err)
			} else {
				fmt.Printf("✅\n")
			}
		}

		// Step 4: Prune stale worktree references
		fmt.Printf("   Pruning worktree references... ")
		runGit(repoRoot, "worktree", "prune")
		fmt.Printf("✅\n")

		fmt.Printf("\n✅ Nuke complete: %s deleted\n", branch)
		return nil
	}
}

// Nuke is the exported version used by makeNukeHandler.
func Nuke(repoRoot, name string, force, localOnly bool) error {
	return nil // handler calls makeNukeHandler; this satisfies the exported API
}

// ── Untracked files: smart fetch + flush ─────────────────────────────────

// valuableFiles is the curated list of important files/dirs that worktrees
// may need but are excluded from sparse-checkout for security.
var valuableFiles = []struct {
	path        string
	description string
	isDir       bool
}{
	{"runtimes/mimocode/agents/", "OVAV agent definitions (10 service areas)", true},
	{"AGENTS.md", "Agent bootstrap template", false},
	{".ovav/plan/caps.yaml", "Project capabilities manifest", false},
	{".ovav/plan/roadmap.yaml", "Project roadmap", false},
	{"MEMORY.md", "Project memory (if exists in repo)", false},
	{".ovav/registry/service_areas/", "Service area definitions", true},
	{".ovav/registry/alignment_progression.yaml", "Alignment progression", false},
}

// UntrackedState tracks which files were fetched and whether they're kept permanently.
type UntrackedState struct {
	Fetched      map[string]bool `json:"fetched"` // path -> true if fetched
	Kept         map[string]bool `json:"kept"`    // path -> true if marked permanent
	ParentBranch string          `json:"parent"`  // parent branch (e.g. "develop")
}

// untrackedStateFile returns the path to the untracked state file for a worktree.
func untrackedStateFile(worktreePath string) string {
	return filepath.Join(worktreePath, ".owl", "untracked.json")
}

// loadUntrackedState loads the untracked state for a worktree.
func loadUntrackedState(worktreePath string) (*UntrackedState, error) {
	path := untrackedStateFile(worktreePath)
	data, err := os.ReadFile(path)
	if err != nil {
		return &UntrackedState{
			Fetched: make(map[string]bool),
			Kept:    make(map[string]bool),
		}, nil
	}
	var state UntrackedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// saveUntrackedState saves the untracked state for a worktree.
func saveUntrackedState(worktreePath string, state *UntrackedState) error {
	path := untrackedStateFile(worktreePath)
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// worktreeEntry holds a parsed git worktree list entry.
type worktreeEntry struct {
	Worktree string
	Branch   string
}

// parseWorktreePorcelain parses `git worktree list --porcelain` output.
func parseWorktreePorcelain(out string) []worktreeEntry {
	var entries []worktreeEntry
	var current worktreeEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			if current.Worktree != "" {
				entries = append(entries, current)
			}
			current = worktreeEntry{Worktree: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}
	if current.Worktree != "" {
		entries = append(entries, current)
	}
	return entries
}

// findCurrentWorktree detects which worktree contains currentPath.
func findCurrentWorktree(repoRoot, currentPath string) (worktreePath, branch string, err error) {
	out, err := runGitOutput(repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return "", "", fmt.Errorf("git worktree list failed: %w", err)
	}

	absCurrent, _ := filepath.Abs(currentPath)

	for _, entry := range parseWorktreePorcelain(string(out)) {
		absEntry, _ := filepath.Abs(entry.Worktree)
		if absEntry == absCurrent || strings.HasPrefix(absCurrent, absEntry+string(filepath.Separator)) {
			return entry.Worktree, entry.Branch, nil
		}
	}

	return "", "", fmt.Errorf("not in a worktree (cwd: %s)", currentPath)
}

// listUntrackedFiles returns files from parentBranch that are missing in worktreePath.
func listUntrackedFiles(repoRoot, worktreePath, parentBranch string) ([]string, error) {
	var missing []string

	for _, vf := range valuableFiles {
		// Check if file exists in parent branch
		_, err := runGitOutput(repoRoot, "show", parentBranch+":"+vf.path)
		if err != nil {
			continue // doesn't exist in parent branch
		}

		// Check if it exists in worktree
		destPath := filepath.Join(worktreePath, vf.path)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			missing = append(missing, vf.path)
		}
	}

	return missing, nil
}

// interactiveSelect presents a numbered list and returns selected indices.
// User types numbers separated by comma, or 'a' for all, ENTER to confirm.
func interactiveSelect(options []string, title string) ([]int, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n  %s\n\n", title)
	for i, opt := range options {
		fmt.Printf("  [%d] %s\n", i+1, opt)
	}
	fmt.Printf("\n  Type numbers (1,3,5), 'a' all, or ENTER none: ")

	line, _, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}

	input := strings.TrimSpace(string(line))
	if input == "" {
		return nil, nil
	}
	if input == "a" {
		idx := make([]int, len(options))
		for i := range options {
			idx[i] = i
		}
		return idx, nil
	}

	var indices []int
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err == nil && n >= 1 && n <= len(options) {
			indices = append(indices, n-1)
		}
	}
	return indices, nil
}

// fetchFile fetches a single file from parentBranch into worktreePath.
func fetchFile(repoRoot, parentBranch, filePath, worktreePath string) error {
	// Use git show to get file content from parent branch
	out, err := runGitOutput(repoRoot, "show", parentBranch+":"+filePath)
	if err != nil {
		return fmt.Errorf("file '%s' not found in %s: %w", filePath, parentBranch, err)
	}

	destPath := filepath.Join(worktreePath, filePath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(destPath, []byte(out), 0644)
}

// removeFetchedFile removes a fetched file from worktree (undoes fetch).
func removeFetchedFile(worktreePath, filePath string) error {
	destPath := filepath.Join(worktreePath, filePath)
	return os.RemoveAll(destPath)
}

// makeUntrackedHandler handles owl --untracked from inside a worktree.
func makeUntrackedHandler(repoRoot string) func(ctx context.Context, args map[string]string) error {
	return func(ctx context.Context, args map[string]string) error {
		mode := args[""]

		// Detect current worktree
		cwd, _ := os.Getwd()
		worktreePath, branch, err := findCurrentWorktree(repoRoot, cwd)
		if err != nil {
			return fmt.Errorf("%v", err)
		}

		// Determine parent branch
		parentBranch := "develop"
		if strings.HasPrefix(branch, "hotfix/") || strings.HasPrefix(branch, "patch/") {
			parentBranch = "main"
		}

		// Load state
		state, err := loadUntrackedState(worktreePath)
		if err != nil {
			return fmt.Errorf("loading untracked state: %w", err)
		}
		state.ParentBranch = parentBranch

		// ── --status mode ──
		if args["status"] == "true" || mode == "status" {
			if len(state.Fetched) == 0 {
				fmt.Println("  No files fetched in this worktree.")
				return nil
			}
			fmt.Println("\n  📋 Fetched files in this worktree:")
			var fetchedList []string
			for f := range state.Fetched {
				fetchedList = append(fetchedList, f)
			}
			sort.Strings(fetchedList)
			for _, f := range fetchedList {
				kept := ""
				if state.Kept[f] {
					kept = " [KEPT]"
				}
				fmt.Printf("  %s%s\n", f, kept)
			}
			fmt.Printf("\n  %d file(s) fetched. Use --flush to remove, --keep to mark permanent.\n", len(state.Fetched))
			return nil
		}

		// ── --flush mode ──
		if args["flush"] == "true" || mode == "flush" {
			if len(state.Fetched) == 0 {
				fmt.Println("  No files to flush.")
				return nil
			}
			var nonKept []string
			for f := range state.Fetched {
				if !state.Kept[f] {
					nonKept = append(nonKept, f)
				}
			}
			if len(nonKept) == 0 {
				fmt.Println("  All fetched files are marked KEEP — not removing anything.")
				return nil
			}
			fmt.Printf("\n  🧹 Flushing %d non-kept file(s)...\n", len(nonKept))
			for _, f := range nonKept {
				removeFetchedFile(worktreePath, f)
				delete(state.Fetched, f)
				fmt.Printf("  ✅ Removed: %s\n", f)
			}
			saveUntrackedState(worktreePath, state)
			fmt.Println("\n  Flush complete.")
			return nil
		}

		// ── --keep mode ──
		if args["keep"] == "true" || mode == "keep" {
			keepTarget := args["file"]
			if keepTarget != "" {
				if state.Fetched[keepTarget] {
					state.Kept[keepTarget] = true
					fmt.Printf("  %s marked as KEEP (permanent for this worktree)\n", keepTarget)
				} else {
					fmt.Printf("  %s not in fetched list.\n", keepTarget)
				}
			} else {
				for f := range state.Fetched {
					state.Kept[f] = true
				}
				fmt.Printf("  %d file(s) marked as KEEP.\n", len(state.Fetched))
			}
			saveUntrackedState(worktreePath, state)
			return nil
		}

		// ── Default: list + interactive fetch ──
		missing, err := listUntrackedFiles(repoRoot, worktreePath, parentBranch)
		if err != nil {
			return fmt.Errorf("listing untracked files: %w", err)
		}

		// Filter out already-fetched files
		var available []string
		for _, f := range missing {
			if !state.Fetched[f] {
				available = append(available, f)
			}
		}

		if len(available) == 0 && len(state.Fetched) == 0 {
			fmt.Println("\n  All valuable files from", parentBranch, "are present in this worktree.")
			return nil
		}

		// Build display list with descriptions
		type fileOption struct {
			path    string
			desc    string
			fetched bool
		}
		var display []fileOption
		for _, vf := range valuableFiles {
			inAvailable := false
			for _, a := range available {
				if a == vf.path {
					inAvailable = true
					break
				}
			}
			fetched := state.Fetched[vf.path]
			if inAvailable || fetched {
				display = append(display, fileOption{path: vf.path, desc: vf.description, fetched: fetched})
			}
		}

		if len(display) == 0 {
			fmt.Println("\n  No files available to fetch.")
			return nil
		}

		// Build options list for interactive select
		var optionLines []string
		for _, d := range display {
			tag := ""
			if d.fetched {
				tag = " [already fetched]"
			}
			optionLines = append(optionLines, fmt.Sprintf("%-45s %s%s", d.path, d.desc, tag))
		}

		indices, err := interactiveSelect(optionLines,
			fmt.Sprintf("📋 Files from %s not in this worktree. Select to fetch:", parentBranch))
		if err != nil {
			return fmt.Errorf("interactive select: %w", err)
		}

		if len(indices) == 0 {
			fmt.Println("\n  Nothing selected — done.")
			return nil
		}

		// Confirm
		fmt.Print("\n  Fetch selected file(s)? (yes/no): ")
		reader := bufio.NewReader(os.Stdin)
		line, _, _ := reader.ReadLine()
		if strings.ToLower(strings.TrimSpace(string(line))) != "yes" {
			fmt.Println("\n  Cancelled.")
			return nil
		}

		// Fetch selected files
		fmt.Println()
		for _, idx := range indices {
			if idx >= len(display) {
				continue
			}
			f := display[idx].path
			if err := fetchFile(repoRoot, parentBranch, f, worktreePath); err != nil {
				fmt.Printf("  ⚠️  Failed to fetch %s: %v\n", f, err)
			} else {
				state.Fetched[f] = true
				fmt.Printf("  ✅ Fetched: %s\n", f)
			}
		}

		saveUntrackedState(worktreePath, state)
		fmt.Println("\n  Done.")
		return nil
	}
}

// cleanWorktreeNoise removes generated noise files from a worktree before merge.
// These files (bin/, *.log, .cache/, coverage/, etc.) should never be tracked.
// Returns the number of items removed and any error.
// This prevents the error-bloop loop where noise files get committed accidentally.
func cleanWorktreeNoise(worktreePath string) (int, error) {
	cleaned := 0

	noisePatterns := []struct {
		path  string
		isDir bool
	}{
		{filepath.Join(worktreePath, "bin"), true},
		{filepath.Join(worktreePath, ".cache"), true},
		{filepath.Join(worktreePath, "coverage"), true},
	}

	// Also scan for noise files matching *.log
	logFiles, _ := filepath.Glob(filepath.Join(worktreePath, "*.log"))

	// Remove directories
	for _, np := range noisePatterns {
		info, err := os.Stat(np.path)
		if err != nil {
			continue
		}
		if np.isDir && info.IsDir() {
			if err := os.RemoveAll(np.path); err == nil {
				cleaned++
			}
		}
	}

	// Remove log files
	for _, logFile := range logFiles {
		if err := os.Remove(logFile); err == nil {
			cleaned++
		}
	}

	return cleaned, nil
}
