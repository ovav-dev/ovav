// [OVAV-FIX] Use parameterized queries (sql.Stmt) instead of string concatenation
// [OVAV-FIX] Use parameterized queries (sql.Stmt) instead of string concatenation
// [OVAV-FIX] Use parameterized queries (sql.Stmt) instead of string concatenation
// [OVAV-FIX] Use parameterized queries (sql.Stmt) instead of string concatenation
// Package ows implements the OVAV Worktree Orchestration System.
// Architecture: Command Registry → State Machine → Git Adapter → SQLite Audit.
package ows

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ── Tier levels ────────────────────────────────────────────────────────────────

// TierLevel represents the access tier for a command.
// Lower numbers = more permissive. Free = 1 (accessible to all).
type TierLevel int

const (
	TierFree       TierLevel = 1 // accessible to all tiers (including free)
	TierPro        TierLevel = 2 // pro tier and above
	TierBusiness   TierLevel = 3 // business tier and above
	TierEnterprise TierLevel = 4 // enterprise tier only
)

// TierFromString converts a tier name to TierLevel.
func TierFromString(name string) TierLevel {
	switch strings.ToLower(name) {
	case "free":
		return TierFree
	case "pro":
		return TierPro
	case "business":
		return TierBusiness
	case "enterprise":
		return TierEnterprise
	default:
		return TierFree
	}
}

// String returns the tier name.
// Tier 0 (unset) returns "free" for consistency with CanRun zero-value behavior.
func (t TierLevel) String() string {
	if t == 0 {
		return "free"
	}
	switch t {
	case TierFree:
		return "free"
	case TierPro:
		return "pro"
	case TierBusiness:
		return "business"
	case TierEnterprise:
		return "enterprise"
	default:
		return "unknown"
	}
}

// CanRun returns true if the given effective tier can run a command at the required tier.
// Tier 0 (unset/default) is treated as TierFree for backward compatibility.
func (t TierLevel) CanRun(effectiveTier string) bool {
	if t == 0 {
		t = TierFree // unset tier defaults to free (backward compatible)
	}
	effective := TierFromString(effectiveTier)
	return int(effective) >= int(t)
}

// ── Command Registry ────────────────────────────────────────────────────────────

// Command defines a single OWS command — the single source of truth for CLI
// dispatch, Cockpit integration, help text, and shell completions.
type Command struct {
	Name        string // "ovav worktree create"
	ShortName   string // "owc"
	Short       string // one-line description for --help
	Long        string // detailed description for --help <command>
	Args        []Arg  // positional arguments
	Interactive bool   // true → opens Cockpit for visual confirmation
	OfflineOK   bool   // true → works without internet
	Profile     string // required profile ("" = no profile needed)
	Tier        TierLevel
	Handler     func(ctx context.Context, args map[string]string) error
}

// Arg defines a positional argument for a command.
type Arg struct {
	Name     string
	Required bool
	Default  string
}

// ProfileConfig defines rules for worktree creation and merge.
type ProfileConfig struct {
	BaseBranch    string // "develop" | "main"
	MergeTo       string // "develop" | "main" | "main+develop" | "none"
	PolicyLevel   string // "relaxed" | "standard" | "strict" | "waiver"
	AutoCleanup   bool   // delete worktree automatically
	RequireReview bool   // require lead approval before merge
}

// ── Registry ────────────────────────────────────────────────────────────────────

// CommandRegistry is the canonical source of all OWS commands.
var CommandRegistry = map[string]Command{
	"ovav worktree": {
		Name:      "ovav worktree",
		ShortName: "ow",
		Short:     "Worktree lifecycle: create/verify/merge/sync/lock/clean/nuke",
		Long: `OVAV Worktree System — Full lifecycle management.

Usage: ovav worktree <command> [flags]

Commands:
  owc   create      Create a new feature worktree from develop
  owd   done        Verify → integrate → push → cleanup worktree
  owl   list        List all worktrees with conflict predictions
  owv   verify      Run full validation pipeline (tests + lint + security)
  ows   sync        Sync remotes + maintenance + prune stale worktrees
  owu   update     Fetch + rebase worktree onto base branch
  owp   prepare     Sync current branch with origin (fast-forward or rebase)
  owlk  lock        Lock worktree for multi-agent coordination
  owm   move        Move worktree to new location
  owclean  clean    Remove orphaned/stale/abandoned worktrees
  owx   route       Cherry-pick / patch / hotfix / emergency routing
  owa   abort       Rollback current in-progress operation
  owr   rescue      Recover lost branches, worktrees, or commits
  owprep prep       Verify or regenerate worktree configuration
  owsuggest suggest Suggest next OWS command based on repo state
  own   nuke        Nuclear delete: worktree + branch + remote (NO merge)

Run 'ovav worktree <command> --help' for detailed help on a specific command.`,
		OfflineOK: true,
		Tier:      TierFree,
		Handler: func(ctx context.Context, args map[string]string) error {
			// Handled by cmdWorktree in main.go — this is for programmatic Dispatch calls
			fmt.Print("OVAV Worktree System — run 'ovav worktree --help' or 'ovav worktree <command> --help'\n")
			return nil
		},
	},
	"ovav worktree create": {
		Name:        "ovav worktree create",
		ShortName:   "owc",
		Short:       "Create a new feature worktree from develop",
		Long:        "Creates a branch + worktree with sparse checkout. Registers in state machine (CREATED→ACTIVE). Emits WORKTREE_CREATED event. Without arguments, opens Cockpit for interactive profile selection.",
		Args:        []Arg{{Name: "name", Required: false}},
		Interactive: true,
		OfflineOK:   true,
		Profile:     "feature",
		Handler:     nil, // wired at init or via SetHandler
	},
	"ovav worktree update": {
		Name:      "ovav worktree update",
		ShortName: "owu",
		Short:     "Fetch + rebase worktree onto base branch",
		Long:      "Updates current worktree against origin/base. Detects conflicts before rebase. If conflict → state DIRTY + notifies owner with file list.",
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree prepare": {
		Name:      "ovav worktree prepare",
		ShortName: "owp",
		Short:     "Sync current branch with origin (fast-forward pull or rebase)",
		Long:      "Prepares the current worktree branch by fetching from origin and updating the local branch. Without --rebase: fast-forward pull only (git pull --ff-only). With --rebase: git rebase onto tracking branch.\n\nFlags:\n  --rebase    Use git rebase instead of fast-forward pull",
		Args:      []Arg{{Name: "rebase", Default: "false"}},
		OfflineOK: false,
		Handler:   nil,
	},
	"ovav worktree sync": {
		Name:      "ovav worktree sync",
		ShortName: "ows",
		Short:     "Sync all remotes + maintenance + prune stale worktrees",
		Long:      "Fetches all remotes. Runs git maintenance (gc, commit-graph, prefetch). Detects orphaned worktrees and prunes them. Refreshes state machine. Worktrees inactive >7d → STALE.\n\nFlags:\n  --rebase    Fetch + rebase onto origin/develop (skip maintenance/prune)\n  --full      Full sync: fetch + rebase + maintenance + prune",
		Args:      []Arg{{Name: "rebase", Default: "false"}, {Name: "full", Default: "false"}},
		OfflineOK: false,
		Handler:   nil,
	},
	"ovav worktree verify": {
		Name:      "ovav worktree verify",
		ShortName: "owv",
		Short:     "Run full validation pipeline (tests + lint + security + policy)",
		Long:      "Executes in order: go test -race, go vet, gofmt, SBOM regen, validate CLI (77 validators), gitleaks, semgrep, policy engine. All must pass to reach VERIFIED state.",
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree done": {
		Name:        "ovav worktree done",
		ShortName:   "owd",
		Short:       "Verify → integrate → push → cleanup worktree",
		Long:        "Location-agnostic: works from anywhere — inside worktree, main repo, or any directory.\n\nModes:\n  owd                   → auto-detect from current HEAD\n  owd feature/sprint-1   → find worktree by branch name\n  owd .ovav/worktrees/.. → explicit worktree path\n\nRuns full compliance pipeline (secrets, forbidden, owv, conflict, hygiene, GPG, reviewer).\nIf all gates pass: merge locally → cleanup worktree+branch → push to origin.\nReturns HEAD to the origin branch (develop/main) on success.",
		Interactive: true,
		OfflineOK:   false,
		Args:        []Arg{{Name: "branch", Required: false}},
		Handler:     nil,
	},
	"ovav worktree route": {
		Name:      "ovav worktree route",
		ShortName: "owx",
		Short:     "Cherry-pick / patch / hotfix / emergency routing",
		Long:      "Exports commits to another branch. Modes: cherry-pick (selective), patch (all commits), hotfix (main+develop simultaneously), emergency (bypass policies with waiver).",
		Args:      []Arg{{Name: "target", Required: true}, {Name: "mode", Default: "cherry-pick"}},
		OfflineOK: false,
		Tier:      TierFree,
		Handler:   nil,
	},
	"ovav worktree abort": {
		Name:      "ovav worktree abort",
		ShortName: "owa",
		Short:     "Rollback current operation, preserve workspace",
		Long:      "Aborts in-progress merge/rebase. Restores pre-operation state. Worktree → RESCUED. Nothing is lost.",
		OfflineOK: true,
		Tier:      TierFree,
		Handler:   nil,
	},
	"ovav worktree rescue": {
		Name:        "ovav worktree rescue",
		ShortName:   "owr",
		Short:       "Recover lost branches, worktrees, or commits from reflog",
		Long:        "Scans reflog + deleted branches + orphaned worktrees. Interactive selection of what to recover. State → RESCUED.",
		Interactive: true,
		OfflineOK:   true,
		Tier:        TierFree,
		Handler:     nil,
	},
	"ovav worktree list": {
		Name:      "ovav worktree list",
		ShortName: "owl",
		Short:     "Inventory of all worktrees with ownership, state, and health",
		Long:      "Lists all worktrees with: state, owner, age, ahead/behind, policy version, health, conflict predictions (⚠️). Supports --mine, --all, --stale, --json.",
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree lock": {
		Name:      "ovav worktree lock",
		ShortName: "owlk",
		Short:     "Lock a worktree to prevent modifications",
		Long:      "Locks worktree for multi-agent coordination. Only owner can unlock. State → LOCKED. Emits WORKTREE_LOCKED event notifying area leads.",
		Args:      []Arg{{Name: "target", Required: true}, {Name: "reason", Default: "manual"}},
		OfflineOK: true,
		Tier:      TierFree,
		Handler:   nil,
	},
	"ovav worktree move": {
		Name:      "ovav worktree move",
		ShortName: "owm",
		Short:     "Move a worktree to a new location without breaking git link",
		Long:      "Relocates a worktree to a new path. Useful for reorganizing disk layout, moving to SSD/NAS, or CI/CD node migration. Preserves git worktree link integrity. Uses git worktree move (Git 2.33+).",
		Args:      []Arg{{Name: "target", Required: true}, {Name: "to", Required: true}},
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree clean": {
		Name:      "ovav worktree clean",
		ShortName: "owclean",
		Short:     "Remove orphaned, stale, or abandoned worktrees",
		Long:      "Scans for orphaned worktrees (no matching branch), stale worktrees (inactive >7d), and abandoned worktrees (>30d). Prunes git worktree metadata. Safe: never removes the main repo worktree. Run `owclean --dry-run` to preview.",
		Args:      []Arg{{Name: "dry-run", Default: "false"}},
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree prep": {
		Name:      "ovav worktree prep",
		ShortName: "owprep",
		Short:     "Verify or regenerate worktree configuration",
		Long:      "Verifies .ovav/worktree-config.json. Returns an explicit error if the file is missing or corrupt (invalid JSON), with an actionable hint. Use --repair to regenerate a valid default config (default_profile: feature, compliance: standard, auto_cleanup: true).",
		Args:      []Arg{{Name: "repair", Default: "false"}},
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree suggest": {
		Name:      "ovav worktree suggest",
		ShortName: "owsuggest",
		Short:     "Suggest next OWS command based on repo state",
		Long:      "Analyzes working tree state and suggests the most appropriate OWS command. With --explain, shows WHY each command is recommended and writes a rich audit entry to .ovav/runtime/owsuggest_audit.jsonl (git history, branch state, worktree status). All runs append a minimal entry to .ovav/runtime/owsuggest_history.jsonl.",
		Args:      []Arg{{Name: "explain", Default: "false"}},
		OfflineOK: true,
		Handler:   nil,
	},
	"ovav worktree nuke": {
		Name:      "ovav worktree nuke",
		ShortName: "own",
		Short:     "Aggressive delete: worktree + local branch + remote branch",
		Long:      "Nuclear delete: removes worktree directory, deletes local branch, deletes remote branch. NO merge, NO validation, NO safety gates. Use --force to skip confirmation. Use --local-only to skip remote branch deletion.\n\nUsage:\n  own my-feature          # interactive confirm\n  own my-feature --force # no confirm\n  own my-feature --force --local-only  # local only\n\nWARNING: This is irreversible. All commits on the branch are lost.",
		Args:      []Arg{{Name: "name", Required: true}, {Name: "force", Default: "false"}, {Name: "local-only", Default: "false"}},
		OfflineOK: true,
		Tier:      TierFree,
		Handler:   nil,
	},
}

// ProfileRegistry is the canonical source of all worktree profiles.
var ProfileRegistry = map[string]ProfileConfig{
	"feature":    {BaseBranch: "develop", MergeTo: "develop", PolicyLevel: "standard"},
	"refactor":   {BaseBranch: "develop", MergeTo: "develop", PolicyLevel: "standard"},
	"docs":       {BaseBranch: "develop", MergeTo: "develop", PolicyLevel: "relaxed"},
	"spike":      {BaseBranch: "develop", MergeTo: "none", PolicyLevel: "relaxed", AutoCleanup: true},
	"hotfix":     {BaseBranch: "main", MergeTo: "main+develop", PolicyLevel: "strict"},
	"release":    {BaseBranch: "develop", MergeTo: "main", PolicyLevel: "strict", RequireReview: true},
	"patch":      {BaseBranch: "main", MergeTo: "main+develop", PolicyLevel: "strict"},
	"research":   {BaseBranch: "develop", MergeTo: "none", PolicyLevel: "relaxed", AutoCleanup: true},
	"emergency":  {BaseBranch: "main", MergeTo: "main+develop", PolicyLevel: "waiver"},
	"migration":  {BaseBranch: "develop", MergeTo: "develop", PolicyLevel: "standard"},
	"enterprise": {BaseBranch: "develop", MergeTo: "develop", PolicyLevel: "strict", RequireReview: true},
	"fix":        {BaseBranch: "develop", MergeTo: "develop", PolicyLevel: "standard"},
}

// ── Dispatch ─────────────────────────────────────────────────────────────────────

// Dispatch resolves a CLI argument list to a registered command and executes it.
// It tries longest prefix match first ("ovav worktree create" before "ovav worktree").
// Tier access is checked against OVAV_CONSUMER_TIER env var (defaults to "free").
func Dispatch(ctx context.Context, repoRoot string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided. Run 'ovav help' for usage.")
	}

	// Longest prefix match
	for i := len(args); i > 0; i-- {
		key := strings.Join(args[:i], " ")
		if cmd, ok := CommandRegistry[key]; ok {
			if cmd.Handler == nil {
				return fmt.Errorf("handler not wired for %s — call WireHandlers before Dispatch", key)
			}

			// Tier access check — abort/retry/cherry-pick/lock are now free tier (OWS-GAP-11)
			effectiveTier := os.Getenv("OVAV_CONSUMER_TIER")
			if effectiveTier == "" {
				effectiveTier = "free"
			}
			if !cmd.Tier.CanRun(effectiveTier) {
				return fmt.Errorf("%s: tier %q is required — your current tier is %q (upgrade to access this command)",
					cmd.ShortName, cmd.Tier.String(), effectiveTier)
			}

			parsed := parseArgs(args[i:], cmd.Args)

			// --help flag → show command help
			if parsed["help"] == "true" {
				fmt.Printf("%s — %s\n\n%s\n", cmd.Name, cmd.Short, cmd.Long)
				if len(cmd.Args) > 0 {
					fmt.Println("Arguments:")
					for _, a := range cmd.Args {
						req := ""
						if a.Required {
							req = " (required)"
						}
						fmt.Printf("  %s%s", a.Name, req)
						if a.Default != "" {
							fmt.Printf(" [default: %s]", a.Default)
						}
						fmt.Println()
					}
				}
				return nil
			}

			// Validate required arguments
			for _, a := range cmd.Args {
				if a.Required && parsed[a.Name] == "" {
					return fmt.Errorf("%s: %s is required\nUsage: ovav worktree %s <%s>%s\nRun 'ovav worktree %s --help' for details",
						cmd.ShortName, a.Name, cmd.ShortName, a.Name,
						func() string {
							if len(cmd.Args) > 1 {
								s := ""
								for _, oa := range cmd.Args {
									if !oa.Required {
										s += fmt.Sprintf(" [%s]", oa.Name)
									}
								}
								return s
							}
							return ""
						}(),
						cmd.ShortName)
				}
			}

			return cmd.Handler(ctx, parsed)
		}
	}

	return fmt.Errorf("unknown command: %s — use owc, owd, owl, owv, ows, owx, owa, owr, owu, owlk, owm, or owclean", args[0])
}

// parseArgs parses positional arguments and --key value flags against the command's Arg definitions.
// Positional args fill in order; flags like --profile <value> are extracted separately.
func parseArgs(rest []string, defs []Arg) map[string]string {
	result := make(map[string]string)

	// Extract --key value flags first (consume flag + value)
	// Handles both "--flag value" and "--flag=value" syntax.
	positional := make([]string, 0, len(rest))
	for i := 0; i < len(rest); i++ {
		if strings.HasPrefix(rest[i], "--") {
			raw := strings.TrimPrefix(rest[i], "--")
			// "--flag=value" syntax: split on first '='
			if idx := strings.Index(raw, "="); idx >= 0 {
				key := raw[:idx]
				val := raw[idx+1:]
				if val != "" {
					result[key] = val
				} else {
					result[key] = "true"
				}
				continue
			}
			// "--flag value" syntax
			key := raw
			if i+1 < len(rest) && !strings.HasPrefix(rest[i+1], "--") {
				result[key] = rest[i+1]
				i++ // skip value
			} else {
				result[key] = "true" // boolean flag
			}
		} else {
			positional = append(positional, rest[i])
		}
	}

	// Fill positional args — only if not already set by explicit --flag
	for i, def := range defs {
		if i < len(positional) {
			if _, exists := result[def.Name]; !exists {
				result[def.Name] = positional[i]
			}
		} else if def.Default != "" {
			if _, exists := result[def.Name]; !exists {
				result[def.Name] = def.Default
			}
		}
	}
	return result
}

// AllCommandNames returns all registered command names for help generation.
func AllCommandNames() []string {
	names := make([]string, 0, len(CommandRegistry))
	for name := range CommandRegistry {
		names = append(names, name)
	}
	return names
}

// FindByShortName finds a command by its abbreviated name (e.g., "owc").
func FindByShortName(short string) (Command, bool) {
	for _, cmd := range CommandRegistry {
		if cmd.ShortName == short {
			return cmd, true
		}
	}
	return Command{}, false
}
