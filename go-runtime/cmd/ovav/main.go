// OVAV Go Runtime — C9.1: CLI Entry Point
//
// Reimplementa bin/ovav (1642 loc Python → ~500 loc Go).
// Superficie pública distribuida como binario nativo.
// Stack: Go 1.22+, solo stdlib. Sin dependencias externas.

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/auth"
	"github.com/ovav/ovav/internal/ceo"
	"github.com/ovav/ovav/internal/chronos"
	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/config"
	"github.com/ovav/ovav/internal/doctor"
	"github.com/ovav/ovav/internal/economy"
	"github.com/ovav/ovav/internal/gitflow"
	"github.com/ovav/ovav/internal/hooks"
	"github.com/ovav/ovav/internal/infra"
	"github.com/ovav/ovav/internal/install"
	"github.com/ovav/ovav/internal/license"
	"github.com/ovav/ovav/internal/ows"
	"github.com/ovav/ovav/internal/profile"
	"github.com/ovav/ovav/internal/project"
	"github.com/ovav/ovav/internal/sbom"
	"github.com/ovav/ovav/internal/status"
	"github.com/ovav/ovav/internal/tailor"
	"github.com/ovav/ovav/internal/tools"
	"github.com/ovav/ovav/internal/validators"
	"github.com/ovav/ovav/internal/vault"
	"github.com/ovav/ovav/internal/vault/secrets"
)

// ── Build metadata (injected at compile time) ────────────────────────────────
var (
	Version   = readVersion()
	Build     = "CAPA 9 — Go Runtime"
	GitSHA    = "unknown"
	BuildTime = "unknown"
)

func readVersion() string {
	// Try root VERSION first (source truth)
	if data, err := os.ReadFile("VERSION"); err == nil {
		return strings.TrimSpace(string(data))
	}
	// Fall back to version.txt in go-runtime
	if data, err := os.ReadFile("go-runtime/VERSION"); err == nil {
		return strings.TrimSpace(string(data))
	}
	return "2.2.0" // safe fallback — matches root VERSION
}

// ── Root command routing ─────────────────────────────────────────────────────

const usage = `OVAV — AI Workstation Governor
  %s

Commands:
  ovav status          System posture check
  ovav profile         Professional profile management
  ovav profile list    List available profiles
  ovav profile apply   Apply a profile to current directory
  ovav config          Show OVAV configuration
  ovav tools           Tool catalog — discover OVAV tools
  ovav tools list      List all tools with status
  ovav tools search    Find tools by keyword
  ovav tools go        Show Go-native tools only
  ovav doctor          Full system diagnostic
  ovav doctor --quick  Quick check (git + branch only)
  ovav vault           Asset encryption (AES-256-GCM)
  ovav vault scan      Discover encryptable assets
  ovav vault encrypt   Encrypt all assets → .ovav/vault/*.enc
  ovav vault decrypt   Decrypt .enc files → restore assets
  ovav vault gen-key   Generate AES-256 key
  ovav tailor          Workstation composer (interactive)
  ovav tailor status   Show current selection
  ovav tailor select   Select plan (nucleo|studio|command)
  ovav tailor preview  Preview pending changes
  ovav tailor apply    Apply workstation configuration
  ovav update          Check for updates
  ovav project         Source → OpenCode projection
  ovav project sync    Sync all projections (agents, skills, visual)
  ovav git             Git workflow v3.0 (start, status, save, push, merge, release)
  ovav chronos         Temporal orientation (Git-based, pure Go)
  ovav login           Unlock OVAV with seed → vault + web identity
  ovav whoami          Show current OVAV identity and session
  ovav logout          Close session and clear vault key
  ovav waiver          Waiver inteligente (identidad, firma, TTL máximo 60 min)
  ovav ceo              CEO session management (bypasses security gates)
  ovav govern          Governor dashboard (health, decisions, trust)
  ovav govern health   Unified health scores
  ovav govern decide   Run decision engine
  ovav govern trust    Verify trust gate on claims
  ovav validate        Run security validators (all | list | <id>)
  ovav defend          Defense status dashboard
  ovav defend scan     Run active defense scan
  ovav defend lockdown Toggle system lockdown
  ovav memory          Agent memory — store, recall, verify, stats
  ovav version         Show version information
  ovav help            This help

Safe by default. Writes require explicit consent.
Go Runtime — native binary, no Python required.`

// isPublicCommand returns true for commands that do not require login.
// OVAV Systems is fully locked: only login and help bypass the session gate.
func isPublicCommand(cmd string) bool {
	switch cmd {
	case "login", "signin", "auth", "help", "--help", "-h",
		"ceo",     // CEO session management (must be accessible without session to login)
		"product", // OVAV Product: sellable product surface, no Systems session required
		"version", "--version", "-v",
		"memory", "mem": // Agent memory is readable without session
		return true
	default:
		return false
	}
}

// requireSession checks for an active, unexpired OVAV session.
// If no session exists or it's expired, prints the lock message and returns false.
func requireSession() bool {
	sess, ok := loadSession()
	if !ok {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "🔒  OVAV Systems — Sealed Governor")
		fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintln(os.Stderr, "  All operations are blocked.")
		fmt.Fprintln(os.Stderr, "  Authenticate to unlock the system:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "    ovav login")
		fmt.Fprintln(os.Stderr, "")
		return false
	}

	age := time.Since(sess.createdAt())
	if age > sessionTTL {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "⏰  OVAV Systems — Session Expired")
		fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintf(os.Stderr, "  Session age: %s (max %s)\n", humanDuration(age), humanDuration(sessionTTL))
		fmt.Fprintln(os.Stderr, "  Re-authenticate:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "    ovav login")
		fmt.Fprintln(os.Stderr, "")
		return false
	}

	return true
}

func main() {
	// Ensure consistent output encoding
	cli.InitOutput()

	// Initialize auth state
	authState := auth.NewAuthState()
	repoRoot, _ := cli.FindRepoRoot()
	if ceo.IsActive(repoRoot) {
		authState.Inject(auth.ModeLOGIN)
	} else {
		authState.Inject(auth.ModeNOLOGIN)
	}

	if len(os.Args) < 2 {
		// Default (no args): launch Cockpit TUI if available
		// Fallback to flat help if cockpit not built or not in repo
		code := launchCockpitDefault()
		if code == 0 {
			os.Exit(0)
		}
		// Cockpit unavailable — show help with build hint
		printUsage()
		fmt.Fprintf(os.Stderr, "\n💡 Run 'make build-cockpit' in go-runtime/ to build the TUI.\n")
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// CEO bypass: if ceo session is cryptographically valid (HMAC-verified), allow even if
	// authState TTL is not active. CEO bypass requires HMAC validation, not just TTL check.
	if !isPublicCommand(cmd) && !authState.IsActive() && !ceo.IsActive(repoRoot) {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "🔒  OVAV Systems — Session Inactive")
		fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintln(os.Stderr, "  Session expired or inactive.")
		fmt.Fprintln(os.Stderr, "  Re-authenticate with: ovav login")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}

	// Route to subcommand (dispatched via dispatch.go)
	if cmd == "--cli" {
		// Force flat CLI mode (skip cockpit even if available)
		if len(args) == 0 {
			printUsage()
			os.Exit(0)
		}
		// Re-route: ovav --cli <subcommand> <args...>
		os.Args = append([]string{os.Args[0]}, args...)
		main()
		return
	}
	if cmd == "--tui" || cmd == "--cockpit" {
		// Force cockpit TUI even from script/pipe
		os.Exit(cmdCockpit(args))
	}

	// Extend TTL on activity (called after each successful command)
	authState.ExtendTTL()
	os.Exit(routeCommand(cmd, args))
}

func printUsage() {
	logo := cli.Logo()
	fmt.Printf(usage, logo)
	fmt.Println()
}

// ── Cockpit default launcher ──────────────────────────────────────────────────

// launchCockpitDefault attempts to launch the Cockpit TUI.
// Returns 0 on success, non-zero if cockpit is unavailable.
// Searches: 1) go-runtime/build/cockpit  2) ~/.local/bin/ovav-cockpit
//  3. same directory as the ovav binary (ovav-cockpit)
func launchCockpitDefault() int {
	// Resolve cockpit binary path
	cockpitPath := resolveCockpitBinary()
	if cockpitPath == "" {
		return 1
	}

	// Verify binary exists and is executable
	if _, err := os.Stat(cockpitPath); err != nil {
		return 1
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		// Not in a repo — still try to launch cockpit without repo context
		repoRoot, _ = os.Getwd()
	}

	cmd := exec.Command(cockpitPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoRoot

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Cockpit error: %v\n", err)
		return 1
	}
	return 0
}

// resolveCockpitBinary finds the cockpit binary using multiple search paths.
func resolveCockpitBinary() string {
	// 1) OVAV_COCKPIT_BIN env var (explicit override)
	if env := os.Getenv("OVAV_COCKPIT_BIN"); env != "" {
		return env
	}

	// 2) go-runtime/build/cockpit relative to repo root
	if repoRoot, err := cli.FindRepoRoot(); err == nil {
		repoCockpit := filepath.Join(repoRoot, "go-runtime", "build", "cockpit")
		if _, err := os.Stat(repoCockpit); err == nil {
			return repoCockpit
		}
	}

	// 3) ~/.local/bin/ovav-cockpit (installed alongside ovav)
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		localCockpit := filepath.Join(homeDir, ".local", "bin", "ovav-cockpit")
		if _, err := os.Stat(localCockpit); err == nil {
			return localCockpit
		}
	}

	// 4) Same directory as the ovav binary (ovav-cockpit)
	if execPath, err := os.Executable(); err == nil {
		siblingCockpit := filepath.Join(filepath.Dir(execPath), "ovav-cockpit")
		if _, err := os.Stat(siblingCockpit); err == nil {
			return siblingCockpit
		}
	}

	return ""
}

// ── Install command helpers ───────────────────────────────────────────────────

func parseModeFlag(args []string) install.Mode {
	for i, a := range args {
		if a == "--mode" && i+1 < len(args) {
			mode, err := install.ResolveMode(args[i+1])
			if err == nil {
				return mode
			}
		}
		if strings.HasPrefix(a, "--mode=") {
			mode, err := install.ResolveMode(strings.TrimPrefix(a, "--mode="))
			if err == nil {
				return mode
			}
		}
	}
	// Default based on flags
	for _, a := range args {
		switch a {
		case "--apply", "--deploy":
			return install.ModeSourceLocalApply
		case "--sandbox":
			return install.ModeSandbox
		}
	}
	return install.ModeDryRun
}

func resolvePackID(args []string) string {
	// Two-pass: explicit --pack-id always wins; --full is the fallback.
	// This way, --pack-id wins regardless of argv order.
	for i, a := range args {
		if a == "--pack-id" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--pack-id=") {
			return strings.TrimPrefix(a, "--pack-id=")
		}
	}
	for _, a := range args {
		if a == "--full" {
			// Map --full to the canonical Build 8 graduated pack.
			// This is the user-friendly "do everything" path.
			return "build8_source_local_apply_pack"
		}
	}
	return "default"
}

func jsonOutput(v interface{}) int {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Println(string(b))
	return 0
}

// ── Subcommand: install ──────────────────────────────────────────────────────

func cmdInstall(args []string) int {
	mode := parseModeFlag(args)
	packID := resolvePackID(args)
	repoRoot := cli.MustFindRepoRoot()
	plan := install.BuildPlan(packID, mode, repoRoot)
	return jsonOutput(plan)
}

// ── Subcommand: uninstall ────────────────────────────────────────────────────

func cmdUninstall(args []string) int {
	result := map[string]interface{}{
		"command":                      "uninstall",
		"status":                       "ok",
		"mode":                         string(parseModeFlag(args)),
		"summary":                      "Safe uninstall guidance. OVAV does not remove unmanaged user files.",
		"recommended_recovery_command": "ovav rollback --dry-run",
	}
	return jsonOutput(result)
}

// ── Subcommand: plan ─────────────────────────────────────────────────────────

func cmdPlan(args []string) int {
	mode := parseModeFlag(args)
	packID := resolvePackID(args)
	repoRoot := cli.MustFindRepoRoot()
	plan := install.BuildPlan(packID, mode, repoRoot)
	result := map[string]interface{}{
		"command": "plan",
		"status":  "ok",
		"plan":    plan,
		"summary": fmt.Sprintf("Plan generated for pack '%s' in %s mode.", packID, mode),
	}
	if plan.Status != "pass" {
		result["status"] = "blocked"
	}
	return jsonOutput(result)
}

// ── Subcommand: backup ───────────────────────────────────────────────────────

func cmdBackup(args []string) int {
	mode := parseModeFlag(args)
	packID := resolvePackID(args)
	repoRoot := cli.MustFindRepoRoot()
	plan := install.BuildPlan(packID, mode, repoRoot)
	manifest := install.BuildManifest(plan)
	backupReport := install.ExecuteBackup(manifest, mode, repoRoot)
	result := map[string]interface{}{
		"command": "backup",
		"status":  "ok",
		"backup":  backupReport,
		"summary": fmt.Sprintf("Backup for pack '%s' in %s mode.", packID, mode),
	}
	return jsonOutput(result)
}

// ── Subcommand: apply ────────────────────────────────────────────────────────

func cmdApply(args []string) int {
	mode := parseModeFlag(args)
	packID := resolvePackID(args)
	repoRoot := cli.MustFindRepoRoot()
	report := install.ExecuteApply(packID, mode, repoRoot)
	result := map[string]interface{}{
		"command": "apply",
		"status":  "ok",
		"pack_id": report.PackID,
		"mode":    string(report.Mode),
		"stages":  report.Stages,
		"summary": fmt.Sprintf("Apply for pack '%s' in %s mode. Status: %s.", packID, mode, report.Status),
	}
	if report.Status != "ok" && report.Status != "pass" {
		result["status"] = "blocked"
	}
	return jsonOutput(result)
}

// ── Subcommand: verify ───────────────────────────────────────────────────────

func cmdVerify(args []string) int {
	mode := parseModeFlag(args)
	packID := resolvePackID(args)
	repoRoot := cli.MustFindRepoRoot()
	plan := install.BuildPlan(packID, mode, repoRoot)
	_ = install.BuildManifest(plan)
	// Use strict validation for comprehensive verification
	validated := install.RunStrictValidation(repoRoot)
	result := map[string]interface{}{
		"command": "verify",
		"status":  "ok",
		"verify":  validated,
		"summary": fmt.Sprintf("Verification for pack '%s' in %s mode.", packID, mode),
	}
	return jsonOutput(result)
}

// ── Subcommand: restore/rollback ─────────────────────────────────────────────

func cmdRestore(args []string) int {
	mode := parseModeFlag(args)
	packID := resolvePackID(args)
	repoRoot := cli.MustFindRepoRoot()
	plan := install.BuildPlan(packID, mode, repoRoot)
	manifest := install.BuildManifest(plan)
	backupReport := install.ExecuteBackup(manifest, mode, repoRoot)
	rollbackReport := install.ExecuteRollback(backupReport, manifest, mode, repoRoot)
	result := map[string]interface{}{
		"command":  "restore",
		"status":   "ok",
		"rollback": rollbackReport,
		"summary":  fmt.Sprintf("Rollback for pack '%s' in %s mode.", packID, mode),
	}
	if !rollbackReport.RollbackPerformed && mode != install.ModeDryRun {
		result["status"] = "blocked"
	}
	return jsonOutput(result)
}

// ── Subcommand: deploy ───────────────────────────────────────────────────────

func cmdDeploy(args []string) int {
	mode := parseModeFlag(args)
	repoRoot := cli.MustFindRepoRoot()
	result := install.GovernedDeploy(mode, repoRoot)
	output := map[string]interface{}{
		"command": "deploy",
		"status":  result.Status,
		"mode":    string(result.Mode),
		"summary": fmt.Sprintf("Governed deploy: %d entries in %s mode.", result.DeployEntries, mode),
	}
	return jsonOutput(output)
}

// ── Subcommand: status ──────────────────────────────────────────────────────

func cmdStatus(args []string) int {
	jsonOut := cli.HasJSONFlag(args)
	writeMarkers := false
	for _, a := range args {
		if a == "--write-markers" || a == "--write" {
			writeMarkers = true
		}
	}

	repoRoot, repoRootErr := cli.FindRepoRoot()
	if repoRootErr != nil {
		repoRoot, _ = os.Getwd()
	}

	result := map[string]interface{}{
		"command": "status",
		"status":  "ok",
	}

	// Go runtime info
	result["runtime"] = map[string]string{
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"version":    Version,
		"build":      Build,
	}

	// Git info (best-effort)
	gitBranch, gitSHA, gitDirty := cli.GitInfo()
	result["git"] = map[string]string{
		"branch": gitBranch,
		"sha":    gitSHA,
		"dirty":  gitDirty,
	}

	// Repo root detection
	if repoRootErr == nil {
		result["repo"] = map[string]string{
			"root": repoRoot,
		}
	}

	// ── Go-native status engine (replaces Python status_engine.py) ──
	engine := status.New(repoRoot)
	statusPayload := engine.Aggregate()

	// Write markers if requested
	if writeMarkers {
		if err := engine.WriteMarkers(); err != nil {
			result["status"] = "error"
			result["markers_error"] = err.Error()
		} else {
			result["markers"] = map[string]string{
				"status_json": filepath.Join(repoRoot, ".ovav", "runtime", "ovav_status.json"),
				"status":      "written",
			}
		}
	}

	// Embed status payload
	result["ovav_status"] = statusPayload

	// Economy / budget data
	if repoRootErr == nil {
		bs, err := economy.Load(repoRoot)
		if err != nil {
			result["economy"] = map[string]string{
				"status": "error",
				"error":  err.Error(),
			}
		} else if bs == nil {
			result["economy"] = map[string]string{
				"status": "no_data",
			}
		} else {
			result["economy"] = bs
		}
	} else {
		result["economy"] = map[string]string{
			"status": "no_data",
		}
	}

	summary := fmt.Sprintf("OVAV %s — %s. Go %s %s/%s. Branch: %s, %s. Status: %s.",
		Version, Build, runtime.Version(), runtime.GOOS, runtime.GOARCH,
		gitBranch, gitDirty, statusPayload.OVAV.Overall)
	result["summary"] = summary

	// ── Human output ──
	if !jsonOut {
		// Economy section
		if bs, ok := result["economy"].(*economy.BudgetStatus); ok && bs != nil {
			fmt.Println("── Economy ───────────────────────────────────────")
			fmt.Println(economy.FormatHuman(bs))
			fmt.Println()
		} else if em, ok := result["economy"].(map[string]string); ok && em["status"] == "no_data" {
			fmt.Println("── Economy ───────────────────────────────────────")
			fmt.Println("  economy: no data")
			fmt.Println()
		}

		// Status section (Go-native, no Python)
		fmt.Println("── System Status ─────────────────────────────────")
		fmt.Printf("  Governor:   %s %s\n", statusPayload.OVAV.Governor.Icon, statusPayload.OVAV.Governor.Label)
		fmt.Printf("  Integrity:  %s %s\n", statusPayload.OVAV.Integrity.Icon, statusPayload.OVAV.Integrity.Label)
		fmt.Printf("  Branch:     %s %s\n", statusPayload.OVAV.Branch.Icon, statusPayload.OVAV.Branch.Label)
		fmt.Printf("  Memory:     %s %s\n", statusPayload.OVAV.Memory.Icon, statusPayload.OVAV.Memory.Label)
		fmt.Printf("  Tokens:     %d total (%d in / %d out)\n",
			statusPayload.OVAV.Tokens.TotalAll,
			statusPayload.OVAV.Tokens.TotalInput,
			statusPayload.OVAV.Tokens.TotalOutput)
		fmt.Printf("  Engine:     Go %s (zero Python)\n", statusPayload.EngineVersion)

		if writeMarkers {
			fmt.Println()
			fmt.Println("── Markers ───────────────────────────────────────")
			fmt.Printf("  ✅ ovav_status.json written\n")
			fmt.Printf("  📍 %s\n", filepath.Join(repoRoot, ".ovav", "runtime", "ovav_status.json"))
		}
		fmt.Println()
	}

	cli.Output(result, jsonOut)
	if result["status"] != "ok" {
		return 1
	}
	return 0
}

// ── Subcommand: profile ─────────────────────────────────────────────────────

func cmdProfile(args []string) int {
	if len(args) == 0 {
		// Default: list profiles
		return profile.CmdList([]string{})
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list", "show":
		return profile.CmdList(rest)
	case "apply":
		return profile.CmdApply(rest)
	case "remove":
		return profile.CmdRemove(rest)
	case "--help", "-h", "help":
		profile.PrintHelp()
		return 0
	default:
		// Treat as profile name for apply shorthand
		fmt.Fprintf(os.Stderr, "Usage: ovav profile <list|apply|remove> [args]\n")
		return 2
	}
}

// ── Subcommand: config ──────────────────────────────────────────────────────

func cmdConfig(args []string) int {
	return config.Show(args)
}

// ── Subcommand: tools ──────────────────────────────────────────────────────

func cmdTools(args []string) int {
	if len(args) == 0 {
		// Default: list all tools
		fmt.Print(tools.FormatList(tools.Catalog(), false))
		return 0
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		return cmdToolsList(rest)
	case "search", "find":
		return cmdToolsSearch(rest)
	case "show", "info":
		return cmdToolsShow(rest)
	case "go", "golang":
		return cmdToolsGo(rest)
	case "categories", "cats":
		return cmdToolsCategories(rest)
	case "--help", "-h", "help":
		printToolsHelp()
		return 0
	default:
		// Treat as search term
		return cmdToolsSearch(args)
	}
}

func cmdToolsList(args []string) int {
	jsonOutput := cli.HasJSONFlag(args)
	cat := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--category", "-c":
			if i+1 < len(args) {
				cat = args[i+1]
				i++
			}
		}
	}

	var result []tools.Tool
	if cat != "" {
		result = tools.ByCategory(cat)
	} else {
		result = tools.Catalog()
	}

	if jsonOutput {
		cli.Output(map[string]interface{}{
			"status": "ok",
			"count":  len(result),
			"tools":  result,
		}, true)
		return 0
	}

	fmt.Print(tools.FormatList(result, false))
	return 0
}

func cmdToolsSearch(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: ovav tools search <keyword>")
		return 2
	}

	keyword := args[0]
	results := tools.Search(keyword)

	if len(results) == 0 {
		fmt.Printf("No tools found matching '%s'.\n", keyword)
		return 0
	}

	fmt.Print(tools.FormatList(results, true))
	return 0
}

func cmdToolsShow(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: ovav tools show <tool-id>")
		return 2
	}

	t := tools.ByID(args[0])
	if t == nil {
		fmt.Printf("Tool '%s' not found. Use 'ovav tools list' to see all tools.\n", args[0])
		return 1
	}

	fmt.Print(tools.FormatTool(t))
	return 0
}

func cmdToolsGo(args []string) int {
	goTools := tools.ByLanguage(tools.LangGo)
	fmt.Print(tools.FormatList(goTools, true))
	return 0
}

func cmdToolsCategories(args []string) int {
	_ = args
	fmt.Print(tools.FormatCategories())
	return 0
}

func printToolsHelp() {
	fmt.Println(`ovav tools — OVAV Tool Catalog

  Discover OVAV tools by name, category, or keyword.
  Every tool has a unique ID and description.

Commands:
  ovav tools list              List all tools (default)
  ovav tools list -c runtime   List tools by category
  ovav tools search <word>     Find tools by keyword
  ovav tools show <id>         Show tool details
  ovav tools go                Show Go-native tools only
  ovav tools categories        List all categories

Categories: runtime, cli, web, security, quality, devops, dev

Examples:
  ovav tools list
  ovav tools search vault
  ovav tools show pipeline
  ovav tools go`)
}

// ── Subcommand: doctor ──────────────────────────────────────────────────────

func cmdDoctor(args []string) int {
	quick := false
	jsonOutput := false
	for _, a := range args {
		switch a {
		case "--quick", "-q":
			quick = true
		case "--json":
			jsonOutput = true
		}
	}

	results := doctor.Run(quick)

	if jsonOutput {
		cli.Output(map[string]interface{}{
			"status":  "ok",
			"command": "doctor",
			"results": results,
		}, true)
		return 0
	}

	fmt.Print(doctor.FormatResults(results))

	// Exit code based on failures
	for _, r := range results {
		if r.Status == "fail" {
			return 1
		}
	}
	return 0
}

// ── Subcommand: validate ───────────────────────────────────────────────────

func cmdValidate(args []string) int {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot find repo root: %v\n", err)
		return 1
	}

	// Build registry with all available validators (aligned with DefaultRegistry)
	registry := validators.DefaultRegistry()

	if len(args) == 0 || args[0] == "all" {
		// Run all validators
		ctx := context.Background()
		results := registry.Run(ctx, repoRoot)
		failed := 0
		for _, r := range results {
			icon := "✅"
			if r.Status == "fail" {
				icon = "❌"
				failed++
			} else if r.Status == "error" {
				icon = "⚠️"
				failed++
			}
			fmt.Printf("%s %-30s %s\n", icon, r.Name, r.Message)
			if len(r.Issues) > 0 && r.Status != "pass" {
				for _, iss := range r.Issues {
					fmt.Printf("   └─ %s\n", iss)
				}
			}
		}
		fmt.Printf("\n%d/%d validators passed", len(results)-failed, len(results))
		if failed > 0 {
			fmt.Println(" ❌")
			return 1
		}
		fmt.Println(" ✅")
		return 0
	}

	if args[0] == "list" {
		// List all validators
		fmt.Println("Available validators:")
		fmt.Println()
		all := registry.All()
		for _, v := range all {
			fmt.Printf("  %-30s %s\n", v.ID(), v.Description())
		}
		fmt.Printf("\n%d validators total\n", len(all))
		return 0
	}

	// Run specific validator by ID
	targetID := args[0]
	ctx := context.Background()
	all := registry.All()
	for _, v := range all {
		if v.ID() == targetID {
			result := v.Validate(ctx, repoRoot)
			icon := "✅"
			if result.Status == "fail" {
				icon = "❌"
			} else if result.Status == "error" {
				icon = "⚠️"
			}
			fmt.Printf("%s %s — %s\n", icon, result.Name, result.Message)
			if len(result.Issues) > 0 {
				for _, iss := range result.Issues {
					fmt.Printf("   └─ %s\n", iss)
				}
			}
			if result.Status == "fail" || result.Status == "error" {
				return 1
			}
			return 0
		}
	}
	fmt.Fprintf(os.Stderr, "Unknown validator: %s\nRun 'ovav validate list' to see available validators.\n", targetID)
	return 1
}

// ── Subcommand: update ──────────────────────────────────────────────────────

func cmdUpdate(args []string) int {
	// Check for update availability via git remote
	hasRemote := cli.HasGitRemote()
	summary := "OVAV Go Runtime — no update mechanism yet (CAPA 9 in progress)."

	if !hasRemote {
		summary = "No git remote configured. Update check unavailable."
	}

	result := map[string]interface{}{
		"command":    "update",
		"status":     "ok",
		"available":  false,
		"version":    Version,
		"go_runtime": true,
		"summary":    summary,
	}

	if hasRemote {
		result["remote"] = cli.GitRemoteURL()
	}

	cli.Output(result, cli.HasJSONFlag(args))
	return 0
}

// ── Subcommand: ceo ─────────────────────────────────────────────────────────

func cmdCeo(args []string) int {
	if len(args) == 0 {
		fmt.Println("ovav ceo — CEO Session Management")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  ovav ceo login     Authenticate as CEO (bypasses security gates for 8h)")
		fmt.Println("  ovav ceo logout    Revoke CEO session (restores security gates)")
		fmt.Println("  ovav ceo status    Show current CEO session status")
		fmt.Println()
		fmt.Println("When CEO session is active, protected branch gates and waiver")
		fmt.Println("requirements are automatically bypassed across the entire repo.")
		return 0
	}

	sub := args[0]
	repoRoot := cli.MustFindRepoRoot()

	switch sub {
	case "login":
		if err := ceo.Create(repoRoot, 8); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println("🔐 CEO session active (8h).")
		fmt.Println("   Protected branch gates bypassed across the repo.")
		fmt.Println("   Run 'ovav ceo logout' to revoke.")
		return 0
	case "logout", "revoke":
		if err := ceo.Revoke(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println("🔓 CEO session revoked — security gates restored.")
		return 0
	case "status":
		if ceo.IsActive(repoRoot) {
			s, err := ceo.Load(repoRoot)
			if err == nil {
				fmt.Printf("🔐 CEO session ACTIVE\n")
				fmt.Printf("   Operator: %s\n", s.Operator)
				fmt.Printf("   Expires:  %s\n", s.ExpiresAt)
				return 0
			}
		}
		fmt.Println("🔓 No active CEO session.")
		fmt.Println("   Run 'ovav ceo login' to authenticate.")
		return 0
	case "--help", "-h", "help":
		fmt.Println("ovav ceo login     Authenticate as CEO (8h, bypasses security gates)")
		fmt.Println("ovav ceo logout    Revoke CEO session")
		fmt.Println("ovav ceo status    Show session status")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown ceo command: %s\n", sub)
		return 2
	}
}

// ── Subcommand: vault ───────────────────────────────────────────────────────

func cmdVault(args []string) int {
	if len(args) == 0 {
		printVaultHelp()
		return 0
	}

	sub := args[0]
	switch sub {
	case "scan":
		return vaultScan(args[1:])
	case "encrypt":
		return vaultEncrypt(args[1:])
	case "decrypt":
		return vaultDecrypt(args[1:])
	case "gen-key", "genkey", "generate-key":
		return vaultGenKey(args[1:])
	case "secrets", "secret":
		// OVAV VAULT secrets subsystem — intelligent credential manager
		return vaultSecretsMain(args[1:])
	case "--help", "-h", "help":
		printVaultHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown vault command: %s\nRun 'ovav vault help' for usage.\n", sub)
		return 2
	}
}

func vaultScan(args []string) int {
	jsonOutput := cli.HasJSONFlag(args)
	repoRoot := cli.MustFindRepoRoot()

	bundles, err := vault.ScanAssets(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning assets: %v\n", err)
		return 1
	}

	if len(bundles) == 0 {
		fmt.Println("No encryptable assets found.")
		return 0
	}

	if jsonOutput {
		cli.Output(map[string]interface{}{
			"status":      "ok",
			"command":     "vault scan",
			"bundles":     len(bundles),
			"total_files": totalFiles(bundles),
			"assets":      bundles,
		}, true)
		return 0
	}

	fmt.Println("OVAV Vault — Asset Discovery")
	fmt.Println()
	for _, b := range bundles {
		fmt.Printf("  📦 %s (%d files)\n", b.Kind, len(b.Files))
		for path := range b.Files {
			fmt.Printf("     └─ %s\n", path)
		}
	}
	fmt.Println()
	fmt.Printf("Total: %d bundles, %d files ready to encrypt.\n", len(bundles), totalFiles(bundles))
	return 0
}

func vaultEncrypt(args []string) int {
	key, err := loadKey(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	repoRoot := cli.MustFindRepoRoot()
	written, err := vault.EncryptAllAssets(repoRoot, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encrypting assets: %v\n", err)
		return 1
	}

	fmt.Println("OVAV Vault — Encryption Complete")
	fmt.Println()
	totalBytes := 0
	for path, size := range written {
		fmt.Printf("  ✅ %s (%d bytes)\n", path, size)
		totalBytes += size
	}
	fmt.Printf("\nEncrypted %d files → %d total bytes.\n", len(written), totalBytes)
	fmt.Println("Key required for decryption. Store it securely.")
	return 0
}

func vaultDecrypt(args []string) int {
	key, err := loadKey(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	repoRoot := cli.MustFindRepoRoot()
	if err := vault.DecryptAllAssets(repoRoot, key); err != nil {
		fmt.Fprintf(os.Stderr, "Error decrypting assets: %v\n", err)
		return 1
	}

	fmt.Println("OVAV Vault — Decryption Complete")
	fmt.Println("All assets restored to their original locations.")
	return 0
}

func vaultGenKey(args []string) int {
	jsonOutput := cli.HasJSONFlag(args)
	outPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out", "-o":
			if i+1 < len(args) {
				outPath = args[i+1]
				i++
			}
		}
	}

	key, err := vault.GenerateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		return 1
	}

	hexKey := fmt.Sprintf("%x", key)

	if outPath != "" {
		if err := os.WriteFile(outPath, []byte(hexKey+"\n"), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing key: %v\n", err)
			return 1
		}
		fmt.Printf("AES-256 key written to: %s\n", outPath)
		fmt.Println("⚠️  Permissions set to 0600. Guard this file securely.")
		return 0
	}

	if jsonOutput {
		cli.Output(map[string]interface{}{
			"status":   "ok",
			"command":  "vault gen-key",
			"key_hex":  hexKey,
			"key_size": len(key),
		}, true)
		return 0
	}

	fmt.Println(hexKey)
	fmt.Fprintln(os.Stderr, "⚠️  Copy this key and store it securely. It CANNOT be recovered.")
	return 0
}

// ── OVAV VAULT Secrets Subsystem ──────────────────────────────────────────────

// vaultSecretsMain routes OVAV VAULT secrets commands.
// Requires OVAV session (vault key from login or OVAV_VAULT_KEY env).
func vaultSecretsMain(args []string) int {
	if len(args) == 0 {
		printVaultSecretsHelp()
		return 0
	}

	sub := args[0]
	switch sub {
	case "add":
		return vaultSecretsAdd(args[1:])
	case "list", "ls":
		return vaultSecretsList(args[1:])
	case "get":
		return vaultSecretsGet(args[1:])
	case "remove", "rm", "delete":
		return vaultSecretsRemove(args[1:])
	case "revoke":
		return vaultSecretsRevoke(args[1:])
	case "rotate":
		return vaultSecretsRotate(args[1:])
	case "query":
		return vaultSecretsQuery(args[1:])
	case "export":
		return vaultSecretsExport(args[1:])
	case "import":
		return vaultSecretsImport(args[1:])
	case "sync":
		return vaultSecretsSync(args[1:])
	case "health", "status":
		return vaultSecretsHealth(args[1:])
	case "audit":
		return vaultSecretsAudit(args[1:])
	case "connect":
		return vaultSecretsConnect(args[1:])
	case "disconnect":
		return vaultSecretsDisconnect(args[1:])
	case "--help", "-h", "help":
		printVaultSecretsHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown secrets command: %s\nRun 'ovav vault secrets help' for usage.\n", sub)
		return 2
	}
}

func loadVaultKeyForSecrets(args []string) ([]byte, error) {
	// Try loadKey first (checks --key flag and OVAV_VAULT_KEY env)
	key, err := loadKey(args)
	if err == nil {
		return key, nil
	}
	// Fallback: try vault_key_export file written by login
	home, _ := os.UserHomeDir()
	exportPath := filepath.Join(home, ".local/share/ovav/vault_key_export")
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, fmt.Errorf("vault key not found: run 'ovav login' first or set OVAV_VAULT_KEY env var")
	}
	return hexToBytes(strings.TrimSpace(string(data)))
}

func openSecretsStore(key []byte) (*secrets.SecretStore, *secrets.DependencyGraph, error) {
	store, err := secrets.Load(key)
	if err != nil {
		return nil, nil, fmt.Errorf("open vault: %w", err)
	}
	graph, _ := secrets.LoadDependencyGraph()
	if graph == nil {
		graph = &secrets.DependencyGraph{}
	}
	return store, graph, nil
}

// vaultSecretsList — list all secrets
func vaultSecretsList(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	filter := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			filter = a
		}
	}

	all := store.List(secrets.SecretType(filter))
	if len(all) == 0 {
		if filter != "" {
			fmt.Printf("No secrets matching %q\n", filter)
		} else {
			fmt.Println("Vault is empty — no secrets stored.")
		}
		return 0
	}

	fmt.Printf("OVAV VAULT — %d secret(s)\n\n", len(all))
	for _, sec := range all {
		refs := []secrets.SecretRef{} // stub: would need graph
		tagStr := ""
		if len(sec.Tags) > 0 {
			tagStr = " [" + strings.Join(sec.Tags, ", ") + "]"
		}
		rotatable := ""
		if sec.Rotatable {
			rotatable = " 🔄"
		}
		orphan := ""
		if len(refs) == 0 {
			orphan = " ⚠️ orphan"
		}
		expStr := ""
		if sec.ExpiresAt != nil && !sec.ExpiresAt.IsZero() {
			expStr = " ⏱ expires " + sec.ExpiresAt.Format("2006-01-02")
		}
		fmt.Printf("  🔑 %-30s %s%s%s%s\n", sec.Name, sec.Type, tagStr, rotatable, orphan)
		if expStr != "" {
			fmt.Printf("     %s\n", expStr)
		}
	}
	return 0
}

// vaultSecretsGet — get secret detail
func vaultSecretsGet(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ovav vault secrets get <name>")
		return 1
	}
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	name := args[0]
	sec := store.GetByName(name)
	if sec == nil {
		fmt.Fprintf(os.Stderr, "❌ Secret %q not found\n", name)
		return 1
	}

	age := time.Since(sec.CreatedAt)
	hash := sec.Hash
	if len(hash) > 16 {
		hash = hash[:16] + "..."
	}

	fmt.Printf("OVAV VAULT — Secret: %s\n\n", sec.Name)
	fmt.Printf("  %-12s %s\n", "Type:", sec.Type)
	fmt.Printf("  %-12s %s\n", "Provider:", sec.Provider)
	fmt.Printf("  %-12s %s\n", "Source:", sec.Source)
	fmt.Printf("  %-12s %s\n", "Hash:", hash)
	fmt.Printf("  %-12s %s\n", "Created:", sec.CreatedAt.Format("2006-01-02 15:04 MST"))
	fmt.Printf("  %-12s %s\n", "Age:", humanDuration(age))
	if sec.ExpiresAt != nil && !sec.ExpiresAt.IsZero() {
		fmt.Printf("  %-12s %s\n", "Expires:", sec.ExpiresAt.Format("2006-01-02 15:04 MST"))
	}
	if sec.LastUsed != nil && !sec.LastUsed.IsZero() {
		fmt.Printf("  %-12s %s\n", "Last used:", sec.LastUsed.Format("2006-01-02 15:04 MST"))
	}
	if sec.Rotatable {
		fmt.Printf("  %-12s %s\n", "Rotatable:", "✅ yes")
	}
	if len(sec.Tags) > 0 {
		fmt.Printf("  %-12s %v\n", "Tags:", sec.Tags)
	}
	fmt.Printf("  %-12s [REDACTED — use 'ovav vault secrets reveal %s' in TUI]\n", "Value:", sec.Name)
	return 0
}

// vaultSecretsAdd — add a new secret
func vaultSecretsAdd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ovav vault secrets add <name> [--type <type>] [--value <value>]")
		return 1
	}
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	name := args[0]
	secType := secrets.TypeAPIToken
	value := ""

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--type", "-t":
			if i+1 < len(args) {
				secType = secrets.SecretType(args[i+1])
				i++
			}
		case "--value", "-v":
			if i+1 < len(args) {
				value = args[i+1]
				i++
			}
		}
	}

	if value == "" {
		fmt.Fprintln(os.Stderr, "❌ --value required (or use 'ovav vault secrets add --value <secret>'")
		return 1
	}

	sec := secrets.NewSecret(name, secType, "", "manual", []byte(value))
	if err := store.Add(sec); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Add failed: %v\n", err)
		return 1
	}
	if err := store.Save(key); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Save failed: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Secret %q added to vault\n", name)
	return 0
}

// vaultSecretsRemove — remove a secret
func vaultSecretsRemove(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ovav vault secrets remove <name>")
		return 1
	}
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	name := args[0]
	sec := store.GetByName(name)
	if sec == nil {
		fmt.Fprintf(os.Stderr, "❌ Secret %q not found\n", name)
		return 1
	}
	if err := store.Remove(sec.ID); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Remove failed: %v\n", err)
		return 1
	}
	if err := store.Save(key); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Save failed: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Secret %q removed from vault\n", name)
	return 0
}

// vaultSecretsRevoke — revoke a secret from all providers
func vaultSecretsRevoke(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, graph, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	// Preview or confirm based on args
	confirm := false
	name := ""
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			confirm = true
		}
		if !strings.HasPrefix(a, "-") && name == "" {
			name = a
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "Usage: ovav vault secrets revoke <name> [--yes]")
		return 1
	}

	sec := store.GetByName(name)
	if sec == nil {
		fmt.Fprintf(os.Stderr, "❌ Secret %q not found\n", name)
		return 1
	}

	refs := graph.GetRefs(sec.ID)
	if len(refs) == 0 {
		fmt.Printf("No providers registered for %q — removing from vault only.\n", name)
		store.Remove(sec.ID)
		store.Save(key)
		return 0
	}

	fmt.Printf("Will revoke %q from %d provider(s):\n", name, len(refs))
	for _, ref := range refs {
		fmt.Printf("  • %s: %s (%s)\n", ref.System, ref.Path, ref.EnvVar)
	}
	if !confirm {
		fmt.Println("\nRun with --yes to confirm.")
		return 1
	}

	report, err := secrets.RevokeSecret(store, graph, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Revoke failed: %v\n", err)
		return 1
	}
	var successCount int
	for _, r := range report.Results {
		status := "✅"
		if r.Status == "failed" || r.Status == "failed_vault_delete" {
			status = "❌"
		} else if r.Status == "manual" || r.Status == "source_not_in_depgraph_manual_action_required" {
			status = "⚠️ "
		}
		detail := r.Error
		if detail == "" {
			detail = r.Status
		}
		fmt.Printf("  %s %s: %s (%s)\n", status, r.Provider, r.Path, detail)
		if r.Status == "revoked" || r.Status == "not_found" {
			successCount++
		}
	}
	fmt.Printf("\n✅ Revocation complete — %d system(s) updated, %d failed\n",
		successCount, len(report.Results)-successCount)
	return 0
}

// vaultSecretsRotate — rotate a secret
func vaultSecretsRotate(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, graph, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	name := ""
	confirm := false
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			confirm = true
		}
		if !strings.HasPrefix(a, "-") && name == "" {
			name = a
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "Usage: ovav vault secrets rotate <name> [--yes]")
		return 1
	}

	sec := store.GetByName(name)
	if sec == nil {
		fmt.Fprintf(os.Stderr, "❌ Secret %q not found\n", name)
		return 1
	}
	if !sec.Rotatable {
		fmt.Fprintf(os.Stderr, "❌ Secret %q is not rotatable — track dependencies first with 'ovav vault deps track'\n", name)
		return 1
	}

	if !confirm {
		fmt.Printf("Rotate %q? This will generate a new value and push to all %d provider(s).\n",
			name, len(graph.GetRefs(sec.ID)))
		fmt.Println("Run with --yes to confirm.")
		return 1
	}

	report, err := secrets.RotateSecret(store, graph, name, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Rotate failed: %v\n", err)
		return 1
	}
	for _, r := range report.Results {
		status := "✅"
		if r.Status == "failed" {
			status = "❌"
		}
		detail := r.Error
		if detail == "" {
			detail = r.Status
		}
		fmt.Printf("  %s %s: %s (%s)\n", status, r.Provider, r.Path, detail)
	}
	if report.VaultUpdated {
		fmt.Println("  💾 Vault updated with new secret value")
	}
	if report.RollbackOccurred {
		fmt.Println("  ⚠️  Rollback occurred — some providers may still have old value")
	}
	return 0
}

// vaultSecretsQuery — natural language query (Phase 7 stub: pattern-matching)
func vaultSecretsQuery(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, graph, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	if len(args) == 0 || args[0] == "--help" {
		fmt.Println(`OVAV VAULT — Natural Language Query (Phase 7 — NLP stub)

Usage: ovav vault secrets query "<question>"

Examples:
  "what secrets expire next week"
  "show me all github tokens"
  "which secrets are used in production"
  "list all orphaned secrets"
  "which cloud keys are older than 90 days"`)
		return 0
	}

	query := strings.Join(args, " ")
	q := strings.ToLower(query)
	fmt.Printf("🔍 Query: %s\n\n", query)

	all := store.List("")
	var results []*secrets.Secret

	for _, sec := range all {
		if strings.Contains(strings.ToLower(sec.Name), q) {
			results = append(results, sec)
			continue
		}
		if strings.Contains(strings.ToLower(string(sec.Type)), q) {
			results = append(results, sec)
			continue
		}
		if strings.Contains(strings.ToLower(sec.Provider), q) {
			results = append(results, sec)
			continue
		}
		if strings.Contains(strings.ToLower(sec.Source), q) {
			results = append(results, sec)
			continue
		}
		// Orphan check: no dependency graph refs
		if strings.Contains(q, "orphan") && len(graph.GetRefs(sec.ID)) == 0 {
			results = append(results, sec)
			continue
		}
		// Expiring check
		if strings.Contains(q, "expir") && sec.ExpiresAt != nil && !sec.ExpiresAt.IsZero() && sec.ExpiresAt.Before(time.Now().Add(7*24*time.Hour)) {
			results = append(results, sec)
			continue
		}
		// Rotatable
		if strings.Contains(q, "rotat") && sec.Rotatable {
			results = append(results, sec)
			continue
		}
		// Cloud keys
		if strings.Contains(q, "cloud") && (sec.Type == secrets.TypeCloudKey || strings.Contains(strings.ToLower(sec.Provider), "cloud")) {
			results = append(results, sec)
			continue
		}
		// GitHub tokens
		if strings.Contains(q, "github") && (sec.Type == secrets.TypeAPIToken || strings.Contains(strings.ToLower(sec.Source), "github")) {
			results = append(results, sec)
			continue
		}
	}

	if len(results) == 0 {
		fmt.Println("  No results found.")
		return 0
	}
	for _, r := range results {
		icon := "🔑"
		if r.Type == secrets.TypeCloudKey {
			icon = "☁️"
		} else if r.Type == secrets.TypeOAuthCreds {
			icon = "🔐"
		} else if r.Type == secrets.TypeDBCredential {
			icon = "🗄️"
		}
		refs := graph.GetRefs(r.ID)
		orphan := ""
		if len(refs) == 0 {
			orphan = " ⚠️ orphan"
		}
		rotatable := ""
		if r.Rotatable {
			rotatable = " 🔄"
		}
		fmt.Printf("  %s %-30s %s%s%s\n", icon, r.Name, r.Type, orphan, rotatable)
	}
	fmt.Printf("\n%d result(s)\n", len(results))
	return 0
}

// vaultSecretsExport — export vault to encrypted .airgap file
func vaultSecretsExport(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, graph, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	outputPath := ""
	password := ""
	expStr := ""
	for i, a := range args {
		if (a == "--output" || a == "-o") && i+1 < len(args) {
			outputPath = args[i+1]
			i++
		} else if a == "--password" && i+1 < len(args) {
			password = args[i+1]
			i++
		} else if a == "--expires" && i+1 < len(args) {
			expStr = args[i+1]
			i++
		} else if !strings.HasPrefix(a, "-") && outputPath == "" {
			outputPath = a
		}
	}

	if outputPath == "" {
		outputPath = fmt.Sprintf("ovav-vault-%s.airgap", time.Now().Format("20060102"))
	}

	seed := os.Getenv("OVAV_SEED")
	if seed == "" {
		// Fall back to machine-ID based derivation
		machineID, _ := license.MachineID()
		seedKey, _ := secrets.DeriveVaultKey("ovav-vault-seed", machineID)
		seed = hex.EncodeToString(seedKey)
	}

	var expiration time.Time
	if expStr != "" {
		if d, err := time.ParseDuration(expStr); err == nil {
			expiration = time.Now().UTC().Add(d)
		} else if t, err := time.Parse(time.RFC3339, expStr); err == nil {
			expiration = t
		}
	}

	opts := &secrets.ExportOptions{
		Password:   password,
		Expiration: expiration,
	}
	data, err := secrets.ExportToAirgap(store, graph, seed, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Export failed: %v\n", err)
		return 1
	}
	if err := secrets.WriteAirgapFile(outputPath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Write failed: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Exported to %s (%d bytes)\n", outputPath, len(data))
	if !expiration.IsZero() {
		fmt.Printf("   Expires: %s\n", expiration.Format(time.RFC3339))
	}
	return 0
}

// vaultSecretsImport — import from .airgap file
func vaultSecretsImport(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	inputPath := ""
	password := ""
	for i, a := range args {
		if a == "--password" && i+1 < len(args) {
			password = args[i+1]
			i++
		} else if !strings.HasPrefix(a, "-") && inputPath == "" {
			inputPath = a
		}
	}

	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: ovav vault secrets import <file.airgap> [--password <pw>]")
		return 1
	}

	seed := os.Getenv("OVAV_SEED")
	if seed == "" {
		machineID, _ := license.MachineID()
		seedKey, _ := secrets.DeriveVaultKey("ovav-vault-seed", machineID)
		seed = hex.EncodeToString(seedKey)
	}

	h := secrets.NewAirgapHandle(inputPath)
	result, err := h.Import(store, seed, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Import failed: %v\n", err)
		return 1
	}
	if result.Expired {
		fmt.Printf("⚠️  Package expired\n")
	}
	fmt.Printf("✅ Imported: %d secrets (%d skipped, %d errors)\n",
		result.SecretsImported, result.SecretsSkipped, len(result.Errors))
	fmt.Printf("   Origin: %s\n", result.OriginDevice)
	if result.HadExpiration {
		fmt.Printf("   Expired: %v\n", result.Expired)
	}
	for _, e := range result.Errors {
		fmt.Printf("   ❌ %s\n", e)
	}
	return 0
}

// vaultSecretsSync — sync vault with cPanel tunnel
func vaultSecretsSync(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	machineID, _ := license.MachineID()
	result, err := secrets.FullSync(store, hex.EncodeToString(key), machineID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Sync failed: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Synced — uploaded %v, downloaded %v, merged %d, conflicts %d\n",
		result.Uploaded, result.Downloaded, result.Merged, result.Conflicts)
	return 0
}

// vaultSecretsHealth — check vault health
func vaultSecretsHealth(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}
	store, _, err := openSecretsStore(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	reports := secrets.CheckStoreHealth(store)
	secrets.PrintHealthReport(reports)
	return 0
}

// vaultSecretsAudit — show audit log
func vaultSecretsAudit(args []string) int {
	key, err := loadVaultKeyForSecrets(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		return 1
	}

	log, err := secrets.NewAuditLog(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Audit log unavailable: %v\n", err)
		return 1
	}

	entries, err := log.ReadAll()
	if err != nil || len(entries) == 0 {
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Read audit log: %v\n", err)
		}
		fmt.Println("Audit log is empty.")
		return 0
	}
	fmt.Printf("OVAV VAULT — Audit Log (%d entries)\n\n", len(entries))
	for _, e := range entries {
		icon := "✅"
		if e.Action == secrets.AuditRemove || e.Action == secrets.AuditRotate {
			icon = "⚠️ "
		}
		detail := ""
		if e.SecretName != "" {
			detail = e.SecretName
		}
		if e.Count > 0 {
			detail = fmt.Sprintf("%s (%d secrets)", detail, e.Count)
		}
		fmt.Printf("  %s %s  %-10s  %-15s  %s\n",
			icon, e.Timestamp.Format("01-02 15:04"), e.Action, e.Source, detail)
	}
	return 0
}

// vaultSecretsConnect — validate vault web session JWT
func vaultSecretsConnect(args []string) int {
	sess, ok := loadSessionForVault()
	if !ok {
		fmt.Println("🔒 Not logged in. Run 'ovav login' first.")
		return 1
	}
	if sess.VaultJWT == "" {
		fmt.Println("⚠️  No vault web session.")
		fmt.Println("   Run 'ovav login' to authenticate with d678beea.ovav.dev")
		return 1
	}

	// Validate JWT against cPanel session endpoint
	valid, err := validateVaultJWT(sess.VaultJWT)
	if err != nil || !valid {
		fmt.Println("⚠️  Vault web session expired or invalid.")
		fmt.Println("   Run 'ovav login' to re-authenticate.")
		return 1
	}

	fmt.Println("🟢 Connected to OVAV VAULT web session")
	fmt.Printf("   Identity: %s\n", sess.IdentityID)
	if sess.Name != "" {
		fmt.Printf("   Name:    %s\n", sess.Name)
	}
	fmt.Println("   Sync:    ready (use 'ovav vault secrets sync')")
	return 0
}

// vaultSecretsDisconnect — clear vault web session JWT
func vaultSecretsDisconnect(args []string) int {
	sess, ok := loadSessionForVault()
	if !ok {
		fmt.Println("🔒 Not logged in.")
		return 1
	}
	if sess.VaultJWT == "" {
		fmt.Println("⚠️  No active vault web session.")
		return 0
	}

	sess.VaultJWT = ""
	if err := saveVaultSession(sess); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Could not update session: %v\n", err)
		return 1
	}

	fmt.Println("👋 Vault web session ended.")
	fmt.Println("   Local vault remains unlocked.")
	fmt.Println("   Run 'ovav login' to reconnect.")
	return 0
}

// loadSessionForVault reads the current session for vault operations.
func loadSessionForVault() (Session, bool) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return Session{}, false
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, false
	}
	return s, true
}

// saveVaultSession saves the session after vault web auth changes.
func saveVaultSession(s Session) error {
	dir := filepath.Dir(sessionPath())
	os.MkdirAll(dir, 0700)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := sessionPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, sessionPath())
}

// validateVaultJWT calls cPanel /api/1/auth/session to verify the JWT is still valid.
func validateVaultJWT(jwt string) (bool, error) {
	const cpanelURL = "https://d678beea.ovav.dev/api/1/auth/session"

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, cpanelURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	return true, nil
}

func printVaultSecretsHelp() {
	fmt.Print(`OVAV VAULT — Intelligent Secrets Manager
  Decrypted vault key derived from seed at login (never stored).

Commands:
  ovav vault secrets add <name>      Add a new secret
  ovav vault secrets list [filter]   List secrets (filter by name substring)
  ovav vault secrets get <name>     Show secret details
  ovav vault secrets remove <name>   Remove a secret from vault
  ovav vault secrets revoke <name>   Revoke from all providers + delete
  ovav vault secrets rotate <name>  Generate new value + push to providers
  ovav vault secrets query "<q>"     Natural language query
  ovav vault secrets export <f>      Export to encrypted .airgap file
  ovav vault secrets import <f>      Import from .airgap file
  ovav vault secrets sync            Sync with cPanel tunnel
  ovav vault secrets health          Vault health check
  ovav vault secrets audit          Show audit log
  ovav vault secrets connect         Validate vault web session
  ovav vault secrets disconnect     End vault web session

Flags:
  --yes, -y            Skip confirmation prompt
  --output, -o <f>    Output file (export)
  --password <pw>      Additional password protection
  --expires <dur>      Expiration (export): 24h, 7d, 30d, or RFC3339

Key:
  Vault key loaded from: OVAV_VAULT_KEY env, or vault_key_export (from login)

Examples:
  ovav vault secrets list
  ovav vault secrets add GITHUB_TOKEN --value ghp_xxx --type api_token
  ovav vault secrets revoke GITHUB_TOKEN --yes
  ovav vault secrets rotate CLOUD_API_KEY --yes
  ovav vault secrets query "github tokens"
  ovav vault secrets export backup.airgap --expires 30d
  ovav vault secrets sync`)
}

// ── OVAV Asset Vault Help ──────────────────────────────────────────────────────

func printVaultHelp() {
	fmt.Println(`ovav vault — OVAV Asset Encryption
  AES-256-GCM encryption for OVAV source assets.

Commands:
  ovav vault scan              Discover encryptable assets
  ovav vault encrypt           Encrypt all assets → .ovav/vault/*.enc
  ovav vault decrypt           Decrypt .enc files → restore source files
  ovav vault gen-key           Generate AES-256 key

Options (encrypt/decrypt):
  --key <path>                 Read key from file
  OVAV_VAULT_KEY=<hex>         Or set key via environment variable
  
Options (gen-key):
  --out, -o <path>             Write key to file (default: stdout)

Asset bundles (canonical source — irreplaceable):
  profiles.enc                 .ovav/registry/service_profiles.yaml
  agents.enc                   .ovav/source/agents/leads/*.md + teams/*/*.md
  skills.enc                   .ovav/source/skills/*/SKILL.md

Output directory: .ovav/vault/

Examples:
  ovav vault scan
  ovav vault gen-key --out .ovav/vault/master.key
  ovav vault encrypt --key .ovav/vault/master.key
  ovav vault decrypt --key .ovav/vault/master.key`)
}

// ── Vault helpers ────────────────────────────────────────────────────────────

func loadKey(args []string) ([]byte, error) {
	// 1. Check --key flag
	for i := 0; i < len(args); i++ {
		if args[i] == "--key" && i+1 < len(args) {
			keyPath := args[i+1]
			hexData, err := os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("reading key file %s: %w", keyPath, err)
			}
			hexStr := strings.TrimSpace(string(hexData))
			return hexToBytes(hexStr)
		}
	}

	// 2. Check OVAV_VAULT_KEY env var
	if envKey := os.Getenv("OVAV_VAULT_KEY"); envKey != "" {
		return hexToBytes(strings.TrimSpace(envKey))
	}

	return nil, fmt.Errorf("no key provided — use --key <path> or set OVAV_VAULT_KEY environment variable")
}

func hexToBytes(hexStr string) ([]byte, error) {
	hexStr = strings.TrimSpace(hexStr)
	if len(hexStr) != 64 {
		return nil, fmt.Errorf("invalid key length: %d hex chars (expected 64 for AES-256)", len(hexStr))
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		var b byte
		if _, err := fmt.Sscanf(hexStr[i*2:i*2+2], "%02x", &b); err != nil {
			return nil, fmt.Errorf("invalid hex key at position %d: %w", i*2, err)
		}
		key[i] = b
	}
	return key, nil
}

func totalFiles(bundles []vault.AssetBundle) int {
	n := 0
	for _, b := range bundles {
		n += len(b.Files)
	}
	return n
}

// ── Subcommand: cockpit ─────────────────────────────────────────────────────

func cmdCockpit(args []string) int {
	_ = args
	// Launch the cockpit TUI binary
	repoRoot := cli.MustFindRepoRoot()
	cockpitBin := filepath.Join(repoRoot, "go-runtime", "build", "cockpit")

	cmd := exec.Command(cockpitBin)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoRoot

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Cockpit error: %v\n", err)
		return 1
	}
	return 0
}

// ── Subcommand: version ─────────────────────────────────────────────────────

func cmdVersion(args []string) int {
	if cli.HasJSONFlag(args) {
		cli.Output(map[string]interface{}{
			"status":      "ok",
			"name":        "OVAV",
			"version":     Version,
			"build":       Build,
			"git_sha":     GitSHA,
			"go_runtime":  true,
			"go_version":  runtime.Version(),
			"os":          runtime.GOOS,
			"arch":        runtime.GOARCH,
			"description": "AI Workstation Governor — Go Runtime",
		}, true)
	} else {
		fmt.Printf("OVAV %s — %s\n", Version, Build)
		fmt.Printf("Go Runtime · %s · %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Build: %s · SHA: %s\n", BuildTime, GitSHA)
		fmt.Println("Posture: source-local launch candidate")
		fmt.Println("Default: dry-run, gated writes, rollback-aware operation")
	}
	return 0
}

// ── Subcommand: tailor ──────────────────────────────────────────────────────

func cmdTailor(args []string) int {
	if len(args) == 0 {
		// Interactive mode
		s := tailor.NewState(nil)
		printTailorStatus(s)
		fmt.Println("\nUse 'ovav tailor select|toggle|preview|apply' for one-shot commands.")
		return 0
	}

	s := tailor.NewState(nil)

	switch args[0] {
	case "status":
		printTailorStatus(s)
	case "select", "plan":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: ovav tailor select <nucleo|studio|command>")
			return 1
		}
		if err := s.SelectPlan(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Printf("✓ Plan %s selected.\n", tailor.PlanLabel(s.SelectedPlan))
		printTailorStatus(s)
	case "toggle":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: ovav tailor toggle <item>")
			return 1
		}
		toggleTailorByID(s, args[1])
		printTailorStatus(s)
	case "preview":
		changes := s.PreviewChanges()
		if len(changes) == 0 {
			fmt.Println("No pending changes.")
		} else {
			fmt.Println("Pending changes:")
			for _, c := range changes {
				mark := "+"
				if !c.After {
					mark = "-"
				}
				fmt.Printf("  %s %s: %s\n", mark, c.Label, c.Summary)
			}
		}
	case "apply":
		results := s.ApplySelection()
		fmt.Println("✓ Configuration applied:")
		for _, r := range results {
			fmt.Printf("  %-14s %s\n", r.Label+":", r.Value)
		}
	case "help", "-h", "--help":
		printTailorHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown tailor command: %s\n", args[0])
		printTailorHelp()
		return 1
	}
	return 0
}

func printTailorStatus(s *tailor.State) {
	fmt.Printf("Plan: %s | %d tools · %d roles\n",
		tailor.PlanLabel(s.SelectedPlan),
		s.ActiveToolCount(), s.ActiveRoleCount())
	for _, t := range s.Tools {
		mark := " "
		if t.Active {
			mark = "✓"
		}
		extra := ""
		if t.Detected {
			extra = " [detected]"
		}
		fmt.Printf("  [%s] %-12s %s%s\n", mark, t.Label, t.Note, extra)
	}
	for _, r := range s.Roles {
		mark := " "
		if r.Active {
			mark = "✓"
		}
		fmt.Printf("  [%s] %-24s %s\n", mark, r.Label, r.Note)
	}
}

func toggleTailorByID(s *tailor.State, id string) {
	rows := s.SelectableRows()
	for i, r := range rows {
		if r.ID == id && r.Type == "item" {
			fmt.Println(s.ToggleAt(i))
			return
		}
	}
	// Search all items for hidden ones
	for _, t := range s.Tools {
		if t.ID == id {
			if !s.IsAllowed(t.MinPlan) {
				fmt.Printf("%s requires plan %s. Switch plan first.\n", t.Label, tailor.PlanLabel(t.MinPlan))
				return
			}
		}
	}
	for _, r := range s.Roles {
		if r.ID == id {
			if !s.IsAllowed(r.MinPlan) {
				fmt.Printf("%s requires plan %s. Switch plan first.\n", r.Label, tailor.PlanLabel(r.MinPlan))
				return
			}
		}
	}
	fmt.Printf("Item not found: %s\n", id)
}

func printTailorHelp() {
	fmt.Println(`ovav tailor — Workstation Composer

Usage:
  ovav tailor                  Show current state
  ovav tailor status           Show current state
  ovav tailor select <plan>    Select nucloe|studio|command
  ovav tailor toggle <item>    Toggle a tool or role
  ovav tailor preview          Show pending changes
  ovav tailor apply            Apply current selection`)
}

// ── Subcommand: sbom ────────────────────────────────────────────────────────

func cmdSBOM(args []string) int {
	if len(args) == 0 {
		printSBOMHelp()
		return 0
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "generate", "gen":
		return sbomGenerate(rest)
	case "verify", "check":
		return sbomVerify(rest)
	case "hash":
		return sbomHash(rest)
	case "--help", "-h", "help":
		printSBOMHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown sbom command: %s\nRun 'ovav sbom help' for usage.\n", sub)
		return 2
	}
}

func sbomGenerate(args []string) int {
	jsonOutput := cli.HasJSONFlag(args)
	repoRoot := cli.MustFindRepoRoot()

	s, err := sbom.Generate(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating SBOM: %v\n", err)
		return 1
	}

	if err := s.Save(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving SBOM: %v\n", err)
		return 1
	}

	if jsonOutput {
		cli.Output(map[string]interface{}{
			"status":      "ok",
			"command":     "sbom generate",
			"files":       len(s.CoreFiles),
			"go_deps":     len(s.Dependencies.Go),
			"python_deps": len(s.Dependencies.Python),
			"registry":    sbom.SBOMRegistry,
		}, true)
		return 0
	}

	fmt.Printf("[OVAV SBOM] Supply Chain Bill of Materials generated.\n")
	fmt.Printf("  Schema:  %s\n", s.SchemaVersion)
	fmt.Printf("  Commit:  %s\n", s.Metadata.GitCommit)
	fmt.Printf("  Files:   %d core files tracked\n", len(s.CoreFiles))
	fmt.Printf("  Go deps: %d dependencies\n", len(s.Dependencies.Go))
	fmt.Printf("  Registry: %s\n", sbom.SBOMRegistry)
	return 0
}

func sbomVerify(args []string) int {
	jsonOutput := cli.HasJSONFlag(args)
	repoRoot := cli.MustFindRepoRoot()

	result, err := sbom.Verify(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SBOM verification error: %v\n", err)
		return 1
	}

	if jsonOutput {
		cli.Output(map[string]interface{}{
			"status":     "ok",
			"command":    "sbom verify",
			"valid":      result.Valid,
			"total":      result.TotalFiles,
			"mismatches": result.Mismatches,
		}, true)
		return 0
	}

	if result.Valid {
		fmt.Printf("[OVAV SBOM Verify] ✅ PASS — %d files verified, 0 mismatches.\n", result.TotalFiles)
		return 0
	}

	fmt.Printf("[OVAV SBOM Verify] ❌ FAIL — %d mismatch(es):\n", len(result.Mismatches))
	for _, m := range result.Mismatches {
		fmt.Printf("  %s\n", m)
	}
	return 1
}

func sbomHash(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	hash := sbom.ComputeRequirementsHash(repoRoot)
	fmt.Printf("Requirements hash (SHA-256): %s\n", hash)
	return 0
}

func printSBOMHelp() {
	fmt.Print(`ovav sbom — Supply Chain Bill of Materials

  Generate and verify SHA-256 hashes for all core OVAV files.
  Common Criteria EAL7-guided: every dependency must be verifiable.

Commands:
  ovav sbom generate        Generate SBOM → .ovav/registry/sbom.json
  ovav sbom verify          Verify current files against stored baseline
  ovav sbom hash            Compute combined requirements hash

Go-native. Replaces tools/security/sbom.py (Python).
`)
}

// ── Project command ───────────────────────────────────────────────────────────

func cmdProject(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, projectHelp)
		return 1
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "sync":
		verbose := true
		for _, a := range subArgs {
			if a == "--quiet" || a == "-q" {
				verbose = false
			}
		}
		fmt.Println("OVAV → OpenCode Projection Sync")
		fmt.Println("──────────────────────────────────")
		if err := project.Sync(repoRoot, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "\n✗ Sync failed: %v\n", err)
			return 1
		}
		fmt.Println("\n✓ All projections synced successfully.")
		return 0
	case "status":
		fmt.Println("Projection status: check connectors/skills.yaml and connectors/personnel.yaml")
		fmt.Println("Run 'ovav project sync' to sync all projections.")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown project subcommand: %s\n", sub)
		fmt.Fprint(os.Stderr, projectHelp)
		return 1
	}
}

const projectHelp = `ovav project — Source → OpenCode Projection

  OVAV works natively with well-organized configs, then converts
  and projects for the OpenCode CLI surface.

Commands:
  ovav project sync         Sync all projections (agents, skills, visual)
  ovav project status       Show projection status

Source (CANONICAL):         Target (PROJECTION):
  .ovav/source/agents/   →  clients/opencode/agents/
  .ovav/source/skills/   →  .opencode/skills/
  .ovav/visual/          →  .opencode/themes|plugins|tui
`

// ── Chronos command ──────────────────────────────────────────────────────────

func cmdChronos(args []string) int {
	repoRoot := cli.MustFindRepoRoot()

	// Parse flags
	timelineCount := 5
	sessionThreshold := 120
	humanOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--timeline":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &timelineCount)
				i++
			}
		case "--session-threshold":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &sessionThreshold)
				i++
			}
		case "--human":
			humanOutput = true
		}
	}

	output := chronos.BuildChronosOutput(repoRoot, timelineCount, sessionThreshold)

	if humanOutput {
		fmt.Println(formatChronosHuman(output))
		return 0
	}

	// Default: JSON output
	jsonBytes, err := output.ToJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling chronos output: %v\n", err)
		return 1
	}
	fmt.Println(string(jsonBytes))
	return 0
}

// formatChronosHuman renders a human-readable Spanish summary.
func formatChronosHuman(output chronos.ChronosOutput) string {
	var b strings.Builder

	now := output.Now
	head := output.Head
	session := output.Session
	drift := output.Drift
	sys := output.System

	fmt.Fprintf(&b, "── OVAV ChronosGate ──\n\n")
	fmt.Fprintf(&b, "  Ahora:    %s (%s)\n", now.ISO, now.Weekday)
	fmt.Fprintf(&b, "  UTC:      %s\n", now.UTC)
	fmt.Fprintf(&b, "  Epoch:    %d\n\n", now.Epoch)

	if head.HashShort != "" {
		msg := head.Message
		if len(msg) > 60 {
			msg = msg[:60]
		}
		fmt.Fprintf(&b, "  HEAD:     %s — %s\n", head.HashShort, msg)
		fmt.Fprintf(&b, "  Edad:     %s (%s)\n", head.AgeHuman, head.ISO)
	} else {
		fmt.Fprintf(&b, "  HEAD:     no disponible\n")
	}
	fmt.Fprintf(&b, "\n")

	if session.Detected {
		if session.IsContinuation {
			fmt.Fprintf(&b, "  Sesión:   continuación (%d min activos)\n", session.MinutesActive)
		} else {
			fmt.Fprintf(&b, "  Sesión:   nueva\n")
		}
		fmt.Fprintf(&b, "  Última acción: %s\n", session.LastAction)
		fmt.Fprintf(&b, "  Cuándo:        %s\n", session.LastActionAt)
	} else {
		fmt.Fprintf(&b, "  Sesión:   no detectada (reflog no disponible)\n")
	}
	fmt.Fprintf(&b, "\n")

	driftStatus := "saludable"
	if !drift.Healthy {
		driftStatus = "ALERTA"
	}
	fmt.Fprintf(&b, "  Drift:    %s\n", driftStatus)
	if drift.Warning != "" {
		fmt.Fprintf(&b, "  ⚠ %s\n", drift.Warning)
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "  Sistema:  %s | Go %s | Git %s\n",
		sys.Hostname, sys.GoVersion, sys.GitVersion)

	return b.String()
}

func init() {
	// Ensure binary name in usage
	execName := filepath.Base(os.Args[0])
	if execName != "ovav" && !strings.HasPrefix(execName, "ovav") {
		// Running from go run or test binary
		execName = "ovav"
	}
}

// ── Worktree Commands (OWS) ──────────────────────────────────────────────

func cmdWorktree(args []string) int {
	// ── --preflight flag: show environment diagnostic and exit ──────────────
	wantPreflight := false
	cleanArgs := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--preflight" || a == "-p" {
			wantPreflight = true
		} else {
			cleanArgs = append(cleanArgs, a)
		}
	}

	if wantPreflight {
		pr := ows.RunPreflight()
		ows.DisplayPreflight(pr)
		return 0
	}

	if len(cleanArgs) < 1 {
		// No subcommand: compact OWS usage
		pr := ows.RunPreflight()
		ows.DisplayPreflightIfNeeded(pr, false)
		fmt.Fprintf(os.Stderr, "OVAV Worktree System — usage:\n")
		fmt.Fprintf(os.Stderr, "  owc <name>    Create worktree from develop\n")
		fmt.Fprintf(os.Stderr, "  owd           Merge → verify → push → cleanup\n")
		fmt.Fprintf(os.Stderr, "  owl           List worktrees\n")
		fmt.Fprintf(os.Stderr, "  owv           Verify (tests + lint + security)\n")
		fmt.Fprintf(os.Stderr, "  ows           Sync remotes\n")
		fmt.Fprintf(os.Stderr, "  owu           Fetch + rebase\n")
		fmt.Fprintf(os.Stderr, "  owa           Abort in-progress operation\n")
		fmt.Fprintf(os.Stderr, "  owr           Rescue lost work\n")
		fmt.Fprintf(os.Stderr, "\nRun 'ovav worktree --preflight' for environment diagnostic.\n")
		fmt.Fprintf(os.Stderr, "Run 'ovav worktree <cmd> --help' for command help.\n")
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not in an OVAV repository\n")
		return 1
	}

	// Wire OWS handlers at first use
	ows.WireHandlers(repoRoot)

	// Dispatch expects full command prefix ("ovav worktree create", not just "create")
	fullArgs := append([]string{"ovav", "worktree"}, cleanArgs...)

	ctx := context.Background()
	if err := ows.Dispatch(ctx, repoRoot, fullArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

// ── Own (Nuke) command — Nuclear worktree delete ───────────────────────

func cmdOwn(args []string) int {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not in an OVAV repository\n")
		return 1
	}

	// Parse flags
	var name string
	force := false
	localOnly := false
	for _, a := range args {
		switch a {
		case "--force", "-f", "force":
			force = true
		case "--local-only", "-l", "local-only":
			localOnly = true
		case "--help", "-h", "help":
			fmt.Print(`own — Nuclear worktree delete

Usage:
  own <branch> [--force] [--local-only]

  own feature/my-branch        # interactive Y/N (type "yes")
  own feature/my-branch --force  # skip confirmation
  own feature/my-branch --force --local-only  # local only

Deletes: worktree + local branch + remote branch (no merge, no validation)
WARNING: IRREVERSIBLE — all commits on the branch are lost.
`)
			return 0
		default:
			if !strings.HasPrefix(a, "-") {
				name = a
			}
		}
	}

	// Wire OWS handlers
	ows.WireHandlers(repoRoot)

	// Detect parent branch
	parentBranch := "develop"
	if strings.HasPrefix(name, "hotfix/") || strings.HasPrefix(name, "patch/") {
		parentBranch = "main"
	}

	// Try to resolve as worktree first
	wt, err := gitflow.ResolveWorktree(name)

	// Branch-only mode: no worktree but branch exists locally
	branchOnly := false
	branchName := name

	if err != nil {
		// Worktree not found — check if it's an orphaned branch
		out, _ := runGit(repoRoot, "branch", "--list", name)
		branchName = strings.TrimSpace(string(out))
		if branchName == "" {
			fmt.Fprintf(os.Stderr, "Error: own: no worktree and no branch found for '%s'\n", name)
			return 1
		}
		branchOnly = true
	} else {
		if !wt.IsWorktree {
			fmt.Fprintf(os.Stderr, "Error: own: cannot nuke the main repository\n")
			return 1
		}
		branchName = wt.Branch
	}

	// Protected branch check
	if branchName == "develop" || branchName == "main" || branchName == "master" {
		fmt.Fprintf(os.Stderr, "Error: own: protected branch '%s' — aborting\n", branchName)
		return 1
	}

	// Step 1: If inside this worktree → git worktree remove handles HEAD migration
	if !branchOnly {
		currentPath, _ := os.Getwd()
		isInsideWorktree := strings.HasPrefix(currentPath, wt.WorktreePath)
		if isInsideWorktree {
			fmt.Printf("🔀 You are inside this worktree — git will move HEAD to %s automatically\n", parentBranch)
		}
	}

	// Confirmation
	if !force {
		var prompt string
		if branchOnly {
			prompt = fmt.Sprintf("  💥 Delete orphaned branch '%s'?  (no worktree)  Type 'yes': ", branchName)
		} else {
			remoteInfo := "origin/" + branchName
			if localOnly {
				remoteInfo = "skipped"
			}
			prompt = fmt.Sprintf("  💥 Delete %s [%s]?  Type 'yes': ", branchName, remoteInfo)
		}
		fmt.Print(prompt)

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Fprintf(os.Stderr, "\n  Own: aborted — no changes made\n")
			return 1
		}
		if strings.ToLower(response) != "yes" {
			fmt.Fprintf(os.Stderr, "\n  Own: aborted — no changes made\n")
			return 1
		}
		fmt.Println()
	}

	// Execute nuclear delete — single-line status
	fmt.Printf("💥 Deleting %s: ", branchName)

	// Remove worktree (skip if branch-only)
	if branchOnly {
		fmt.Printf("worktree --  ")
	} else if _, err := runGit(repoRoot, "worktree", "remove", "--force", wt.WorktreePath); err != nil {
		fmt.Printf("worktree ⚠️  ")
	} else {
		fmt.Printf("worktree ✅  ")
	}

	// Delete local branch
	if _, err := runGit(repoRoot, "branch", "-D", branchName); err != nil {
		fmt.Printf("local ⚠️  ")
	} else {
		fmt.Printf("local ✅  ")
	}

	// Delete remote branch
	if !localOnly {
		if _, err := runGit(repoRoot, "push", "origin", "--delete", branchName); err != nil {
			fmt.Printf("remote ⚠️")
		} else {
			fmt.Printf("remote ✅")
		}
	}
	fmt.Println()
	fmt.Printf("✅ Own: %s deleted\n", branchName)
	return 0
}

// ── Hook command — Git Hook Management ────────────────────────────────────

func cmdHook(args []string) int {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not in an OVAV repository\n")
		return 1
	}

	manager := hooks.NewManager(repoRoot)

	if len(args) == 0 {
		printHookHelp()
		return 0
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "install":
		return cmdHookInstall(manager, rest)
	case "uninstall":
		return cmdHookUninstall(manager, rest)
	case "status":
		return cmdHookStatus(manager, rest)
	case "run":
		return cmdHookRun(manager, rest)
	case "audit":
		return cmdHookAudit(manager, rest)
	case "snapshot":
		return cmdHookSnapshot(manager, rest)
	case "check":
		return cmdHookCheck(manager, rest)
	case "--help", "-h", "help":
		printHookHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown hook command: %s\nRun 'ovav hook help' for usage.\n", sub)
		return 2
	}
}

func cmdHookInstall(m *hooks.Manager, args []string) int {
	jsonOut := cli.HasJSONFlag(args)
	results, err := m.Install()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing hooks: %v\n", err)
		return 1
	}

	if jsonOut {
		return jsonOutput(map[string]interface{}{
			"command": "hook install",
			"status":  "ok",
			"results": results,
		})
	}

	installed := 0
	updated := 0
	skipped := 0
	failed := 0
	for _, r := range results {
		switch r.Status {
		case "installed":
			installed++
			fmt.Printf("  ✅ %s — %s\n", r.Stage.Label(), r.Message)
		case "updated":
			updated++
			fmt.Printf("  🔄 %s — %s\n", r.Stage.Label(), r.Message)
		case "skip":
			skipped++
			fmt.Printf("  ⏭️  %s — %s\n", r.Stage.Label(), r.Message)
		case "fail":
			failed++
			fmt.Printf("  ❌ %s — %s\n", r.Stage.Label(), r.Message)
		}
	}
	fmt.Printf("\n  %d installed · %d updated · %d skipped · %d failed\n", installed, updated, skipped, failed)
	return 0
}

func cmdHookUninstall(m *hooks.Manager, args []string) int {
	jsonOut := cli.HasJSONFlag(args)
	results, err := m.Uninstall()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error uninstalling hooks: %v\n", err)
		return 1
	}

	if jsonOut {
		return jsonOutput(map[string]interface{}{
			"command": "hook uninstall",
			"status":  "ok",
			"results": results,
		})
	}

	for _, r := range results {
		switch r.Status {
		case "removed":
			fmt.Printf("  🗑️  %s — removed\n", r.Stage.Label())
		case "skip":
			fmt.Printf("  ⏭️  %s — %s\n", r.Stage.Label(), r.Message)
		}
	}
	fmt.Println("\n  ✅ Hooks removed — validation now runs via OWS (owv) only.")
	return 0
}

func cmdHookStatus(m *hooks.Manager, args []string) int {
	jsonOut := cli.HasJSONFlag(args)
	report, err := m.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking hooks: %v\n", err)
		return 1
	}

	if jsonOut {
		return jsonOutput(report)
	}

	fmt.Printf("── OVAV Hook Status ──\n")
	fmt.Printf("  Repo:       %s\n", report.RepoRoot)
	fmt.Printf("  Binary:     %s\n", report.OVAVBinary)
	fmt.Printf("  Platform:   %s\n", cliRuntimeOS())
	fmt.Println()

	for _, h := range report.Hooks {
		icon := "❌"
		status := "NOT INSTALLED"
		if h.Installed && h.OVAV && h.Executable && h.SHA256OK {
			icon = "✅"
			status = "HEALTHY"
		} else if h.Installed && h.OVAV && !h.SHA256OK {
			icon = "⚠️"
			status = "TAMPERED"
		} else if h.Installed && !h.OVAV {
			icon = "🔶"
			status = "FOREIGN"
		} else if h.Installed && !h.Executable {
			icon = "⚠️"
			status = "NOT EXECUTABLE"
		}
		fmt.Printf("  %s %-20s %s\n", icon, h.Stage.Label(), status)
		if h.Message != "" {
			fmt.Printf("     └─ %s\n", h.Message)
		}
	}

	if report.AllHealthy {
		fmt.Println("\n  ✅ All hooks healthy — automatic validation active.")
	} else {
		fmt.Println("\n  ⚠️  Hooks not healthy — run 'ovav hook install' to repair.")
	}
	return 0
}

func cmdHookRun(m *hooks.Manager, args []string) int {
	stageFlag := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--stage" && i+1 < len(args) {
			stageFlag = args[i+1]
			i++
		}
	}

	if stageFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --stage <pre-commit|pre-push> required\n")
		return 1
	}

	stage := hooks.Stage(stageFlag)

	// ── Tampering check before execution (real-time defense) ──
	if events := m.CheckTampering(); len(events) > 0 {
		fmt.Fprintf(os.Stderr, "🚫 OVAV HOOK TAMPERING DETECTED\n")
		for _, e := range events {
			fmt.Fprintf(os.Stderr, "   %s %s: %s\n", e.Timestamp, e.Type, e.Detail)
		}
		fmt.Fprintf(os.Stderr, "\n   Run 'ovav hook install' to restore hooks.\n")
		return 1
	}

	// ── Run validators ──
	result := m.RunStage(stage)
	jsonOut := cli.HasJSONFlag(args)

	if jsonOut {
		return jsonOutput(result)
	}

	// Human output — compact, scan-friendly
	fmt.Printf("🔍 ovav hook run — %s\n", stage.Label())

	if len(result.Results) == 0 {
		fmt.Println("   (no validators configured for this stage)")
		if result.Error != "" {
			fmt.Printf("   ⚠️  %s\n", result.Error)
		}
		return 0
	}

	passed := 0
	failed := 0
	for _, r := range result.Results {
		icon := "✅"
		if r.Status == "fail" || r.Status == "error" {
			icon = "❌"
			failed++
		} else {
			passed++
		}
		fmt.Printf("  %s %-30s %s (%dms)\n", icon, r.Name, r.Status, r.Duration.Milliseconds())
		if len(r.Issues) > 0 {
			for _, issue := range r.Issues {
				fmt.Printf("     └─ %s\n", issue)
			}
		}
	}

	fmt.Printf("\n  %d passed · %d failed · %dms total\n", passed, failed, result.Duration.Milliseconds())

	if !result.Passed {
		fmt.Println("\n🚫 Commit blocked — fix failures above and try again.")
		return 1
	}
	return 0
}

func cmdHookAudit(m *hooks.Manager, args []string) int {
	jsonOut := cli.HasJSONFlag(args)
	report, err := m.Audit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error auditing hooks: %v\n", err)
		return 1
	}

	if jsonOut {
		return jsonOutput(report)
	}

	fmt.Printf("── OVAV Hook Security Audit ──\n")
	fmt.Printf("  Status: %s\n", auditStatusIcon(report.Status))
	fmt.Printf("  Last:   %s\n", report.LastAudit)
	fmt.Println()

	for _, h := range report.Hooks {
		fmt.Printf("  %s %s\n", statusIconStr(h.SHA256OK, h.Installed && h.OVAV), h.Stage.Label())
	}

	if len(report.Threats) > 0 {
		fmt.Printf("\n⚠️  Threats detected (%d):\n", len(report.Threats))
		for _, t := range report.Threats {
			fmt.Printf("  🔴 %s\n", t)
		}
		fmt.Println("\n  Recommended: ovav hook install")
	} else {
		fmt.Println("\n✅ No threats detected — hooks secure.")
	}

	// No-verify check
	nvReport, _ := m.NoVerifyCheck()
	fmt.Printf("\n── No-Verify Check ──\n")
	fmt.Print(hooks.FormatNoVerifyHuman(nvReport))

	return 0
}

func cmdHookSnapshot(m *hooks.Manager, args []string) int {
	snap, err := m.GenerateIntegritySnapshot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating snapshot: %v\n", err)
		return 1
	}
	return jsonOutput(snap)
}

func cmdHookCheck(m *hooks.Manager, args []string) int {
	events := m.CheckTampering()
	if len(events) == 0 {
		fmt.Println("✅ Hooks intact — no tampering detected.")
		return 0
	}
	fmt.Printf("🚫 TAMPERING DETECTED (%d events):\n", len(events))
	for _, e := range events {
		fmt.Printf("  %s %s: %s\n", e.Timestamp, e.Type, e.Detail)
	}
	return 1
}

func printHookHelp() {
	fmt.Print(`ovav hook — OVAV Git Hook Management
  Native Go hook system — zero external dependencies.
  Integrates with OWS validator engine (owv).

Commands:
  ovav hook install          Install hook shims in .git/hooks/
  ovav hook uninstall        Remove OVAV hook shims
  ovav hook status           Show hook installation status + SHA-256 integrity
  ovav hook run --stage <s>  Run validators for a stage (called by git hooks)
  ovav hook audit            Full security audit + tampering detection + no-verify check
  ovav hook snapshot         Generate SHA-256 integrity snapshot
  ovav hook check            Quick tampering check (runs on every hook invocation)

Stages:
  pre-commit          Runs before git commit (secrets, branch, workspace safety)
  pre-push            Runs before git push (secrets, branch shield, release gate)
  post-checkout       Runs after git checkout
  commit-msg          Runs after commit message is written

Auto-install:
  owc (worktree create) auto-installs hooks on every new worktree.
  ovav infra bootstrap installs hooks system-wide.

Examples:
  ovav hook install
  ovav hook status
  ovav hook run --stage pre-commit
  ovav hook audit
`)
}

func auditStatusIcon(status string) string {
	switch status {
	case "clean":
		return "✅ CLEAN"
	case "tampered":
		return "🔴 TAMPERED"
	case "broken":
		return "⚠️  BROKEN"
	case "uninstalled":
		return "❌ UNINSTALLED"
	default:
		return status
	}
}

func statusIconStr(ok, installed bool) string {
	if !installed {
		return "❌"
	}
	if ok {
		return "✅"
	}
	return "⚠️"
}

func cliRuntimeOS() string {
	return runtime.GOOS
}

// ── infra command ──────────────────────────────────────────────────────────

func cmdInfra(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("OVAV Infrastructure Manager")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  ovav infra bootstrap    Install dependencies + load credentials")
		fmt.Println("  ovav infra check        Verify connectivity to all services")
		fmt.Println("  ovav infra tokens       Show token status")
		fmt.Println("  ovav infra dns list     List DNS records for ovav.dev")
		fmt.Println("  ovav infra dns check    Check if cpanel.ovav.dev exists")
		fmt.Println("  ovav infra dns delete   Delete cpanel.ovav.dev DNS record")
		fmt.Println("  ovav infra tunnel list  List Cloudflare tunnels and hostnames")
		fmt.Println("  ovav infra tunnel check Verify d678beea.ovav.dev tunnel routing")
		return 0
	}

	root, err := infra.ResolveRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "bootstrap":
		fmt.Println("OVAV Infrastructure Bootstrap")
		fmt.Println("═══════════════════════════════")
		results, err := infra.Bootstrap(root)
		for _, r := range results {
			icon := "✅"
			if r.Status == "fail" {
				icon = "❌"
			} else if r.Status == "skip" {
				icon = "⏭️ "
			}
			fmt.Printf("  %s %-15s %s\n", icon, r.Step, r.Detail)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n⚠️  Bootstrap completed with issues: %v\n", err)
			return 1
		}
		fmt.Println("\n✅ Bootstrap complete. Run 'ovav infra check' to verify.")

	case "check":
		results := infra.CheckAll(root)
		infra.PrintStatusReport(results)

		fmt.Println("Tokens:")
		tokens := infra.CheckTokens(root)
		for _, t := range tokens {
			icon := "✅"
			if !t.Found && !strings.Contains(t.Details, "optional") {
				icon = "❌"
			} else if !t.Found {
				icon = "⚪"
			}
			fmt.Printf("  %s %-20s %-12s %s\n", icon, t.Name, t.Source, t.Details)
		}

	case "tokens":
		tokens := infra.CheckTokens(root)
		for _, t := range tokens {
			icon := "✅"
			if !t.Found {
				icon = "❌"
			}
			fmt.Printf("  %s %-20s %s\n", icon, t.Name, t.Details)
		}

	case "dns":
		if len(subArgs) == 0 {
			fmt.Println("Usage: ovav infra dns <list|check|delete>")
			return 1
		}
		switch subArgs[0] {
		case "list":
			records, err := infra.ListDNSRecords("ovav.dev", infra.VaultPath(root))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			for _, r := range records {
				fmt.Printf("  %-40s %-5s %s\n", r.Name, r.Type, r.Content)
			}
		case "check":
			result, _ := infra.CheckDNSRecord("ovav.dev", "cpanel.ovav.dev", infra.VaultPath(root))
			fmt.Printf("  %s: %s\n", result.Action, result.Error)
		case "delete":
			fmt.Println("Deleting cpanel.ovav.dev from Cloudflare DNS...")
			result, err := infra.DeleteDNSRecord("ovav.dev", "cpanel.ovav.dev", infra.VaultPath(root))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			fmt.Printf("  ✅ %s: %s\n", result.Action, result.Record.Name)
		default:
			fmt.Printf("Unknown dns subcommand: %s\n", subArgs[0])
			return 1
		}

	case "tunnel":
		if len(subArgs) == 0 {
			fmt.Println("Usage: ovav infra tunnel <list|check>")
			return 1
		}
		switch subArgs[0] {
		case "list":
			tunnels, err := infra.ListTunnels(infra.VaultPath(root))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			for _, t := range tunnels {
				fmt.Printf("  %s (%s):\n", t.Name, t.ID)
				for _, h := range t.Hostnames {
					fmt.Printf("    → %s\n", h)
				}
			}
		case "check":
			msg, err := infra.VerifyTunnelHostname("d678beea.ovav.dev", infra.VaultPath(root))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			fmt.Println(msg)
		default:
			fmt.Printf("Unknown tunnel subcommand: %s\n", subArgs[0])
			return 1
		}

	default:
		fmt.Printf("Unknown infra subcommand: %s\nRun 'ovav infra' for help.\n", sub)
		return 1
	}

	return 0
}

// ── License command ─────────────────────────────────────────────────────────

func cmdLicense(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: ovav license <bind|verify|machine-id>")
		return 1
	}

	sub := args[0]
	switch sub {
	case "machine-id":
		mid, err := license.MachineID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(mid)
		return 0
	case "bind":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: ovav license bind <license-key>\n")
			return 1
		}
		lic := &license.License{Key: args[1]}
		result, err := license.Bind(lic)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Bind failed: %v\n", err)
			return 1
		}
		if result.Bound {
			fmt.Println("✅ License bound successfully")
			fmt.Printf("   Machine ID: %s\n", result.MachineID)
			fmt.Printf("   Vault hash: %s\n", result.VaultHash)
			fmt.Printf("   Expires:    %s\n", result.ExpiresAt)
		} else {
			fmt.Fprintf(os.Stderr, "Bind failed\n")
			return 1
		}
		return 0
	case "verify":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: ovav license verify <vault-hash>\n")
			return 1
		}
		lic := &license.License{}
		result := license.Verify(lic, args[1])
		if result.Valid {
			fmt.Println("✅ License valid")
			if result.DaysLeft > 0 {
				fmt.Printf("   Days left: %d\n", result.DaysLeft)
			}
		} else {
			fmt.Fprintf(os.Stderr, "License verification failed: %s\n", result.Message)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown license subcommand: %s\n", sub)
		fmt.Println("Usage: ovav license <bind|verify|machine-id>")
		return 1
	}
}

// ── Gate/Check commands (migrated from tools/cli/*.py) ───────────────────────

func cmdSurfaces(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	jsonOut := cli.HasJSONFlag(args)

	var result map[string]interface{}
	if len(args) > 0 && args[0] == "repair-plan" {
		result = cli.SurfacesRepairPlan(repoRoot)
	} else {
		result = cli.SurfacesCheck(repoRoot)
	}

	if jsonOut {
		return jsonOutput(result)
	}

	surfaces, _ := result["surfaces"].([]map[string]interface{})
	fmt.Println("OVAV Surface Manager")
	for _, s := range surfaces {
		icon := "✅"
		if s["status"] == "missing" {
			icon = "❌"
		}
		req := ""
		if s["required"] == true {
			req = " (required)"
		}
		fmt.Printf("  %s %s%s — %s\n", icon, s["path"], req, s["desc"])
	}
	return 0
}

func cmdExportGate(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	jsonOut := cli.HasJSONFlag(args)
	result := cli.ExportGateCheck(repoRoot)

	if jsonOut {
		return jsonOutput(result)
	}

	passed := result["passed"].(bool)
	checks, _ := result["checks"].([]map[string]interface{})
	fmt.Println("OVAV Public Export Gate")
	for _, c := range checks {
		icon := "✅"
		if !c["passed"].(bool) {
			icon = "❌"
		}
		fmt.Printf("  %s %s: %v\n", icon, c["name"], c["detail"])
	}
	if passed {
		fmt.Println("\n✅ All export checks passed.")
	} else {
		fmt.Println("\n⚠️  Some checks failed — review before publishing.")
	}
	return 0
}

func cmdRepoCheck(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	jsonOut := cli.HasJSONFlag(args)
	result := cli.RepoPresentationGate(repoRoot)

	if jsonOut {
		return jsonOutput(result)
	}

	passed := result["passed"].(bool)
	checks, _ := result["checks"].([]map[string]interface{})
	fmt.Println("OVAV Repo Presentation Gate")
	for _, c := range checks {
		icon := "✅"
		if !c["passed"].(bool) {
			icon = "❌"
		}
		fmt.Printf("  %s %s: %v\n", icon, c["name"], c["detail"])
	}
	if passed {
		fmt.Println("\n✅ Repo presentation passed.")
	}
	return 0
}

func cmdReleaseCheck(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	jsonOut := cli.HasJSONFlag(args)
	result := cli.ReleasePackageCheck(repoRoot)

	if jsonOut {
		return jsonOutput(result)
	}

	ready := result["ready"].(bool)
	checks, _ := result["checks"].([]map[string]interface{})
	fmt.Println("OVAV Release Package")
	for _, c := range checks {
		icon := "✅"
		if !c["passed"].(bool) {
			icon = "❌"
		}
		fmt.Printf("  %s %s: %v\n", icon, c["name"], c["detail"])
	}
	if ready {
		fmt.Println("\n✅ Release ready.")
	} else {
		fmt.Println("\n⚠️  Release not ready — fix issues above.")
	}
	return 0
}

func cmdFreshSmoke(args []string) int {
	repoRoot := cli.MustFindRepoRoot()
	jsonOut := cli.HasJSONFlag(args)
	keepClone := false
	for _, a := range args {
		if a == "--keep-clone" {
			keepClone = true
		}
	}

	fmt.Println("OVAV Fresh Clone Smoke — cloning repo...")
	result := cli.FreshCloneSmoke(repoRoot, keepClone)

	if jsonOut {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		if result.OverallOK {
			return 0
		}
		return 1
	}

	for _, c := range result.Checks {
		icon := "✅"
		if !c["passed"].(bool) {
			icon = "❌"
		}
		fmt.Printf("  %s %s: %v\n", icon, c["name"], c["detail"])
	}
	if result.OverallOK {
		fmt.Println("\n✅ Fresh clone smoke passed.")
		return 0
	}
	fmt.Println("\n❌ Fresh clone smoke failed.")
	return 1
}

func cmdDetectEnv(args []string) int {
	jsonOut := cli.HasJSONFlag(args)
	startPath := ""
	for i, a := range args {
		if a == "--path" && i+1 < len(args) {
			startPath = args[i+1]
		}
	}

	result := cli.DetectEnv(startPath)

	if jsonOut {
		return jsonOutput(map[string]interface{}{
			"env":                string(result.Env),
			"root":               result.Root,
			"has_ovav":           result.HasOVAV,
			"has_dev_tools":      result.HasDevTools,
			"commands_available": result.CommandsAvailable,
			"ovav_dir":           result.OVAVDir,
			"suggestion":         result.Suggestion,
		})
	}

	fmt.Printf("OVAV Environment: %s\n", result.Env)
	if result.Root != "" {
		fmt.Printf("  Root: %s\n", result.Root)
	}
	fmt.Printf("  OVAV: %v  Dev tools: %v\n", result.HasOVAV, result.HasDevTools)
	if result.Suggestion != "" {
		fmt.Printf("  💡 %s\n", result.Suggestion)
	}
	return 0
}

func cmdGateway(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	action := ""
	mode := "dry_run"
	consent := false
	riskAccepted := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--action":
			if i+1 < len(args) {
				action = args[i+1]
				i++
			}
		case "--apply":
			mode = "apply"
		case "--dry-run":
			mode = "dry_run"
		case "--consent":
			consent = true
		case "--accept-risk":
			riskAccepted = true
		}
	}

	if action == "" {
		fmt.Fprintf(os.Stderr, "Usage: ovav gateway --action <setup|sync|security|recovery|update> [--dry-run|--apply] [--consent] [--accept-risk]\n")
		return 1
	}

	valid := false
	for _, a := range cli.ValidGatewayActions() {
		if a == action {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Fprintf(os.Stderr, "Error: invalid action '%s'. Choose from: setup, sync, security, recovery, update\n", action)
		return 1
	}

	result := cli.ExecutionGateReport(action, mode, consent, riskAccepted)

	if jsonOut {
		return jsonOutput(result)
	}

	allowed := result["allowed"].(bool)
	icon := "✅"
	if !allowed {
		icon = "❌"
	}
	fmt.Printf("OVAV Execution Gateway\n")
	fmt.Printf("  %s Action: %s  Mode: %s  Allowed: %v\n", icon, action, mode, allowed)
	if reqs, ok := result["requires"].([]string); ok && len(reqs) > 0 {
		fmt.Printf("  Requires: %s\n", strings.Join(reqs, ", "))
	}
	if allowed {
		return 0
	}
	return 1
}
