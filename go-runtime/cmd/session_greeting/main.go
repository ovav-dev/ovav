// OVAV Session Greeting — Go-native replacement for session_greeting.py (456 LOC).
//
// Uses internal/chronos (in-process, zero subprocess) for temporal data.
// Uses internal/economy for budget data. Zero Python dependencies.
//
// Replaces: tools/agent_runtime/session_greeting.py (deleted v61.0).
// Called by AGENTS.md at session start: go run ./cmd/session_greeting --json
//
// Architecture:
//   - chronos data: internal/chronos.BuildChronosOutput() — in-process, no subprocess
//   - economy data: internal/economy.Load() — reads budget_status.json directly
//   - integrity: inline host config intrusion detection
//   - session marker: inline protected branch check
//   - git status: exec.Command git (lightweight — only 2 calls)
//   - handoff: generates .ovav/context/CURRENT_HANDOFF.md from git HEAD + caps.yaml
//
// Stack: Go 1.25+, go-git v5 (via internal/chronos), stdlib.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/ceo"
	"github.com/ovav/ovav/internal/chronos"
	"github.com/ovav/ovav/internal/economy"
	"github.com/ovav/ovav/internal/fde"
	"github.com/ovav/ovav/internal/loopguard"
	"github.com/ovav/ovav/internal/memory"
)

// ═══════════════════════════════════════════════════════════════════════════
// CONSTANTS
// ═══════════════════════════════════════════════════════════════════════════

const (
	schemaVersion = "ovav.session_greeting.v2"
	timelineCount = 5
	sessionMin    = 120 // minutes
)

var protectedBranches = map[string]bool{
	"main": true, "master": true, "develop": true, "development": true,
	"prod": true, "production": true, "staging": true,
}

// ═══════════════════════════════════════════════════════════════════════════
// OUTPUT SCHEMA (exact match with session_greeting.py v2)
// ═══════════════════════════════════════════════════════════════════════════

type GreetingOutput struct {
	Schema         string          `json:"schema"`
	GreetingType   string          `json:"greeting_type"`
	GreetingText   string          `json:"greeting_text"`
	ContextSummary string          `json:"context_summary"`
	Branch         string          `json:"branch"`
	Chronos        *ChronosBlock   `json:"chronos"`
	Integrity      IntegrityBlock  `json:"integrity"`
	Auth           AuthBlock       `json:"auth"`
	Session        SessionMark     `json:"session"`
	Capsule        CapsuleInfo     `json:"capsule"`
	GitMetrics     GitMetricsBlock `json:"git_metrics"`
	GitStatus      GitStatusBlock  `json:"git_status"`
	LatestWork     LatestWorkBlock `json:"latest_work"`
	NextWork       string          `json:"next_work"`
	Economy        json.RawMessage `json:"economy"`
	SelfDiagnosis  SelfDiagnosis   `json:"self_diagnosis"`
	AutoActions    []AutoAction    `json:"auto_actions"`
	FDEBrain       interface{}     `json:"fde_brain,omitempty"`
	Memory         MemoryBlock     `json:"memory"`
}

// MemoryBlock carries OVAV SYSTEM's persistent memory context into each session.
// Loaded at session start from .ovav/runtime/agent_memory.yaml.
// Provides: recent cards, critical decisions, active rules, authenticity report.
type MemoryBlock struct {
	TotalCards   int                       `json:"total_cards"`
	ActiveCards  int                       `json:"active_cards"`
	RecentCards  []MemoryCard              `json:"recent_cards,omitempty"`
	TopRules     []string                  `json:"top_rules,omitempty"` // operational rules from critical cards
	Authenticity memory.AuthenticityReport `json:"authenticity,omitempty"`
}

// MemoryCard is a compact card representation for session greeting.
type MemoryCard struct {
	ID         string   `json:"id"`
	Topic      string   `json:"topic"`
	Summary    string   `json:"summary"`
	Priority   string   `json:"priority,omitempty"`
	ProposedBy string   `json:"proposed_by,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	LastSeen   string   `json:"last_confirmed,omitempty"`
}

type ChronosBlock struct {
	Now                 NowCompact      `json:"now"`
	HeadAge             string          `json:"head_age"`
	HeadISO             string          `json:"head_iso"`
	SessionContinuation bool            `json:"session_continuation"`
	Timeline            []TimelineEntry `json:"timeline"`
}

type NowCompact struct {
	ISO     string `json:"iso"`
	Weekday string `json:"weekday"`
	Hour    int    `json:"hour"`
	Minute  int    `json:"minute"`
}

type TimelineEntry struct {
	Hash     string `json:"hash"`
	AgeHuman string `json:"age_human"`
	Message  string `json:"message"`
}

type IntegrityBlock struct {
	Status string `json:"status"`
	Clean  bool   `json:"clean"`
}

type SessionMark struct {
	Marked        bool `json:"marked"`
	CeoActive     bool `json:"ceo_active,omitempty"`
	OwnerLoggedIn bool `json:"owner_logged_in,omitempty"` // true when auth session is cryptographically valid (HMAC + expiry pass)
}

type CapsuleInfo struct {
	Active bool   `json:"active"`
	Note   string `json:"note"`
}

type GitMetricsBlock struct {
	HeadHash    string `json:"head_hash"`
	HeadShort   string `json:"head_short"`
	CommitCount int    `json:"commit_count"`
}

type GitStatusBlock struct {
	TotalChanges  int      `json:"total_changes"`
	ModifiedFiles []string `json:"modified_files"`
}

type LatestWorkBlock struct {
	Source  string `json:"source"`
	Ref     string `json:"ref"`
	Summary string `json:"summary"`
}

type SelfDiagnosis struct {
	Status        string   `json:"status"`
	Checks        int      `json:"checks"`
	FailingChecks []string `json:"failing_checks"`
	Reason        string   `json:"reason"`
}

type AutoAction struct {
	Action string `json:"action"`
	Status string `json:"status"`
}

// AuthBlock carries Login System v2 authentication mode information.
type AuthBlock struct {
	Mode         string `json:"mode"`                  // "LOGIN" | "NO_LOGIN"
	Level        string `json:"level"`                 // "CEO" | "ANONYMOUS"
	TTLRemaining string `json:"ttl_remaining"`         // e.g. "47m", "1h23m"
	ExtendedBy   string `json:"extended_by,omitempty"` // "session_activity" or empty
}

// ═══════════════════════════════════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════════════════════════════════

func main() {
	human := flag.Bool("human", false, "Human-readable output")
	jsonFlag := flag.Bool("json", false, "JSON output (default, kept for CLI compatibility)")
	repo := flag.String("repo", "", "Repository root (default: auto-detect)")
	checkProtected := flag.Bool("check-protected", false, "Exit 0 if writes allowed, exit 1 if blocked (pre-write gate)")
	ceoLogin := flag.Bool("ceo", false, "Create CEO session marker (bypasses protected branch + waiver)")
	ceoRevoke := flag.Bool("ceo-revoke", false, "Revoke CEO session marker")
	leadID := flag.String("lead", "", "Load FDE Brain Pack for this lead (e.g., thavren)")
	flag.Parse()

	repoRoot := *repo
	if repoRoot == "" {
		var err error
		repoRoot, err = detectRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(2)
		}
	}

	// CEO session management — creates or revokes CEO auth marker
	if *ceoRevoke {
		if err := ceo.Revoke(repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("🔓 CEO session revoked — security gates restored.")
		os.Exit(0)
	}
	if *ceoLogin {
		if err := ceo.Create(repoRoot, 8); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(2)
		}
		fmt.Println("🔐 CEO session active (8h) — protected branch gates bypassed.")
		os.Exit(0)
	}

	// Pre-write gate mode: check protected branch, exit code-based
	if *checkProtected {
		mark := markSession(repoRoot)
		if !mark.Marked {
			fmt.Fprintf(os.Stderr, "BLOCKED: Protected branch with no waiver. Read-only mode.\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	// LOOPGUARD GATE — session-greeting hook (ISS-2026-0728-005).
	// Runs DetectFromLog on the on-disk session events file. If a loop is
	// in progress from a previous session, hard-stop with actionable advice.
	// Use ABSOLUTE path (filepath.Join with repoRoot) — CWD when running
	// via `go run ./cmd/session_greeting` is the cmd subdir, not the repo.
	loopguard.SetSessionEventsPath(filepath.Join(repoRoot, ".ovav/runtime/session_events.jsonl"))
	loopguard.SetSessionID("session_greeting")
	if found, det, err := loopguard.DetectFromLog(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: loopguard startup check failed: %v\n", err)
	} else if found {
		fmt.Fprintf(os.Stderr, "\n🚨 OVAV LOOP DETECTED on session start\n")
		fmt.Fprintf(os.Stderr, "   tool:     %s\n", det.Tool)
		fmt.Fprintf(os.Stderr, "   hash:     %s\n", det.OutputHash)
		fmt.Fprintf(os.Stderr, "   occurs:   %d in %.2fs window\n", det.Occurrences, det.WindowSeconds)
		fmt.Fprintf(os.Stderr, "   action:   %s\n", det.SuggestedAction)
		fmt.Fprintf(os.Stderr, "\n   Proceeding with greeting, but a loop is likely active.\n")
		fmt.Fprintf(os.Stderr, "   Resolve: review recent work, end turn, or change strategy.\n\n")
	}

	// Suppress unused warning
	_ = jsonFlag

	output := buildGreeting(repoRoot)

	// Inject FDE Brain if --lead specified
	if *leadID != "" {
		brain := injectBrain(repoRoot, *leadID)
		if brain != nil {
			output.FDEBrain = brain
		}
	}

	if *human {
		printHuman(output)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		enc.Encode(output)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ORCHESTRATOR
// ═══════════════════════════════════════════════════════════════════════════

func buildGreeting(repoRoot string) GreetingOutput {
	// Phase 1: security-critical checks
	sessionMark := markSession(repoRoot)
	integrity := checkIntegrity()
	gitState := getGitState(repoRoot)
	authBlock := buildAuthBlock(repoRoot)
	// OwnerLoggedIn: true when auth session is cryptographically valid (HMAC + expiry)
	sessionMark.OwnerLoggedIn = authBlock.Mode == "LOGIN"

	// Phase 2: temporal data from chronos (in-process, zero subprocess)
	chronosOut := chronos.BuildChronosOutput(repoRoot, timelineCount, sessionMin)

	// Phase 2b: OVAV SYSTEM persistent memory — loaded at every session start
	memoryBlock := loadMemoryBlock(repoRoot)

	// Phase 3: economy snapshot
	economyData := loadEconomy(repoRoot)

	// Classification
	greetingType := classifyGreeting(chronosOut)
	greetingText := buildGreetingText(chronosOut, greetingType)
	contextSummary := buildContextSummary(chronosOut, gitState)

	// Build chronos compact block
	chronosBlock := buildChronosBlock(chronosOut)

	// Auto-actions status
	autoActions := []AutoAction{
		{Action: "session_marked", Status: statusBool(sessionMark.Marked)},
		{Action: "integrity_check", Status: statusBoolWarn(integrity.Clean)},
		{Action: "git_state", Status: statusBool(gitState.Branch != "unknown")},
		{Action: "economy_update", Status: statusBool(economyData != nil)},
	}

	// Raw economy JSON
	var econRaw json.RawMessage
	if economyData != nil {
		econRaw, _ = json.Marshal(economyData)
	} else {
		econRaw = json.RawMessage(`{"alert_level":"unknown","error":"budget_status.json not found"}`)
	}

	output := GreetingOutput{
		Schema:         schemaVersion,
		GreetingType:   greetingType,
		GreetingText:   greetingText,
		ContextSummary: contextSummary,
		Branch:         gitState.Branch,
		Chronos:        chronosBlock,
		Integrity:      integrity,
		Auth:           authBlock,
		Session:        sessionMark,
		Capsule: CapsuleInfo{
			Active: false,
			Note:   "Capsule system removed in v2.0 simplification (2026-06-11).",
		},
		GitMetrics: GitMetricsBlock{
			HeadHash:    chronosOut.Head.HashShort,
			HeadShort:   chronosOut.Head.HashShort,
			CommitCount: gitState.CommitCount,
		},
		GitStatus: GitStatusBlock{
			TotalChanges:  gitState.Changes,
			ModifiedFiles: gitState.ModifiedFiles,
		},
		LatestWork: LatestWorkBlock{
			Source:  "git_head",
			Ref:     chronosOut.Head.HashShort,
			Summary: chronosOut.Head.Message,
		},
		NextWork: "Determinar según caps.yaml + git HEAD (fuentes canónicas).",
		Economy:  econRaw,
		SelfDiagnosis: SelfDiagnosis{
			Status:        "simplified_v2",
			Checks:        4,
			FailingChecks: []string{},
			Reason:        "Minimal session start: session marker + integrity + git state + economy.",
		},
		AutoActions: autoActions,
		Memory:      memoryBlock,
	}

	// Generate CURRENT_HANDOFF.md (non-blocking)
	writeHandoff(repoRoot, output)

	return output
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION MARKER — reads HEAD, checks protected branch, looks for waiver
// ═══════════════════════════════════════════════════════════════════════════

func markSession(repoRoot string) SessionMark {
	headPath := filepath.Join(repoRoot, ".git", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		// Worktree: .git is a file pointing to the real git dir
		gitFile := filepath.Join(repoRoot, ".git")
		if data, gitErr := os.ReadFile(gitFile); gitErr == nil {
			line := strings.TrimSpace(string(data))
			if strings.HasPrefix(line, "gitdir: ") {
				realGit := strings.TrimSpace(line[8:])
				headPath = filepath.Join(realGit, "HEAD")
				content, err = os.ReadFile(headPath)
			}
		}
	}
	if err != nil {
		return SessionMark{Marked: false}
	}

	line := strings.TrimSpace(string(content))
	const prefix = "ref: refs/heads/"
	if !strings.HasPrefix(line, prefix) {
		return SessionMark{Marked: false}
	}

	branch := line[len(prefix):]
	if protectedBranches[branch] {
		// CEO session bypass — if CEO is authenticated, skip waiver requirement
		if ceo.IsActive(repoRoot) {
			return SessionMark{Marked: true, CeoActive: true}
		}
		waiver := filepath.Join(repoRoot, ".ovav", "runtime", "protected_branch_waiver.yaml")
		if _, err := os.Stat(waiver); os.IsNotExist(err) {
			return SessionMark{Marked: false}
		}
	}
	// Always report CEO session status regardless of branch protection
	return SessionMark{Marked: true, CeoActive: ceo.IsActive(repoRoot)}
}

// buildAuthBlock returns the Login System v2 auth block.
func buildAuthBlock(repoRoot string) AuthBlock {
	sess, err := ceo.Load(repoRoot)
	if err != nil || sess == nil {
		return AuthBlock{Mode: "NO_LOGIN", Level: "ANONYMOUS", TTLRemaining: "0m", ExtendedBy: ""}
	}
	if !sess.Valid() {
		return AuthBlock{Mode: "NO_LOGIN", Level: "ANONYMOUS", TTLRemaining: "0m", ExtendedBy: ""}
	}

	expiry, err := time.Parse(time.RFC3339, sess.ExpiresAt)
	if err != nil {
		return AuthBlock{Mode: "LOGIN", Level: "CEO", TTLRemaining: "0m", ExtendedBy: "session_activity"}
	}
	remaining := time.Until(expiry)
	if remaining < 0 {
		remaining = 0
	}

	level := "CEO"
	if sess.Operator != "" && sess.Operator != "ceo-alexander" {
		level = "USER"
	}

	return AuthBlock{
		Mode:         "LOGIN",
		Level:        level,
		TTLRemaining: formatDuration(remaining),
		ExtendedBy:   "session_activity",
	}
}

// formatDuration returns a human-readable duration string (e.g. "47m", "1h23m").
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "0m"
	}
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

// ═══════════════════════════════════════════════════════════════════════════
// INTEGRITY CHECK — host config intrusion detection
// ═══════════════════════════════════════════════════════════════════════════

func checkIntegrity() IntegrityBlock {
	home, err := os.UserHomeDir()
	if err != nil {
		return IntegrityBlock{Status: "ERROR", Clean: false}
	}

	hostConfig := filepath.Join(home, ".config", "opencode")
	intrusionFiles := []string{"AGENTS.md", "opencode.json", "opencode.jsonc"}
	var intrusions []string

	for _, fname := range intrusionFiles {
		fpath := filepath.Join(hostConfig, fname)
		data, err := os.ReadFile(fpath)
		if err != nil {
			continue // file doesn't exist = no intrusion
		}
		content := strings.TrimSpace(string(data))
		// Benign bootstrap: only $schema key in opencode config files
		if (fname == "opencode.json" || fname == "opencode.jsonc") &&
			strings.Contains(content, "\"$schema\"") && len(content) < 200 {
			continue
		}
		intrusions = append(intrusions, fpath)
	}

	if len(intrusions) > 0 {
		return IntegrityBlock{
			Status: "INTRUSION",
			Clean:  false,
		}
	}
	return IntegrityBlock{Status: "clean", Clean: true}
}

// ═══════════════════════════════════════════════════════════════════════════
// GIT STATE — branch, status, commit count
// ═══════════════════════════════════════════════════════════════════════════

type gitState struct {
	Branch        string
	Changes       int
	ModifiedFiles []string
	CommitCount   int
}

func getGitState(repoRoot string) gitState {
	gs := gitState{Branch: "unknown"}

	// Branch
	if out, err := runGit(repoRoot, "branch", "--show-current"); err == nil {
		gs.Branch = strings.TrimSpace(out)
	}

	// Commit count
	if out, err := runGit(repoRoot, "rev-list", "--count", "HEAD"); err == nil {
		var n int
		if _, scanErr := fmt.Sscanf(strings.TrimSpace(out), "%d", &n); scanErr == nil {
			gs.CommitCount = n
		}
	}

	// Status
	if out, err := runGit(repoRoot, "status", "--short"); err == nil {
		lines := strings.Split(strings.TrimSpace(out), "\n")
		gs.Changes = 0
		gs.ModifiedFiles = nil
		for _, l := range lines {
			if l = strings.TrimSpace(l); l != "" {
				gs.Changes++
				if len(gs.ModifiedFiles) < 15 {
					gs.ModifiedFiles = append(gs.ModifiedFiles, l)
				}
			}
		}
	}

	return gs
}

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ECONOMY — load budget_status.json via internal/economy
// ═══════════════════════════════════════════════════════════════════════════

func loadEconomy(repoRoot string) *economy.BudgetStatus {
	bs, err := economy.Load(repoRoot)
	if err != nil || bs == nil {
		return nil
	}
	return bs
}

// loadMemoryBlock loads OVAV SYSTEM's persistent memory at session start.
// This is the PRIMARY integration point for Agent Memory Persistence —
// every session start receives the full memory context injected into GreetingOutput.
// Falls gracefully if memory file doesn't exist yet (first use).
func loadMemoryBlock(repoRoot string) MemoryBlock {
	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		// Memory system not yet initialized — return empty block
		return MemoryBlock{}
	}

	// Load stats
	total, active, _, _ := am.Stats()

	// Load recent cards (top 5 for compact session context)
	recentCards := am.Recent(5)

	// Build compact card representations
	var compact []MemoryCard
	for _, c := range recentCards {
		compact = append(compact, MemoryCard{
			ID:         c.ID,
			Topic:      c.Topic,
			Summary:    c.Summary,
			Priority:   c.Priority,
			ProposedBy: c.ProposedBy,
			Tags:       c.Tags,
			LastSeen:   c.LastConfirmed,
		})
	}

	// Extract top operational rules from critical/high priority cards
	var topRules []string
	criticalCards := am.Recall(memory.RecallOptions{
		Tags:  []string{"governance", "rule", "decision"},
		Limit: 5,
	})
	for _, r := range criticalCards.Cards {
		if r.OperationalRule != "" {
			// Truncate for compactness
			rule := r.OperationalRule
			if len(rule) > 120 {
				rule = rule[:120] + "..."
			}
			topRules = append(topRules, rule)
		}
	}

	// Verify authenticity of loaded cards
	var authReport memory.AuthenticityReport
	if len(recentCards) > 0 {
		authReport = am.Verify(recentCards)
	}

	return MemoryBlock{
		TotalCards:   total["total"],
		ActiveCards:  active["active"],
		RecentCards:  compact,
		TopRules:     topRules,
		Authenticity: authReport,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GREETING CLASSIFICATION
// ═══════════════════════════════════════════════════════════════════════════

func classifyGreeting(co chronos.ChronosOutput) string {
	if !co.Session.Detected {
		return "first_time"
	}
	if co.Session.IsContinuation {
		return "recent_return"
	}
	return "stale_return"
}

func buildGreetingText(co chronos.ChronosOutput, greetingType string) string {
	if co.Session.Detected {
		if co.Session.IsContinuation {
			return fmt.Sprintf("Sesión retomada (%d min activos).", co.Session.MinutesActive)
		}
		if co.Head.AgeHuman != "" {
			return fmt.Sprintf("Sesión nueva. Último commit: %s.", co.Head.AgeHuman)
		}
	}
	return "Sesión nueva. Sistema OVAV verificado."
}

func buildContextSummary(co chronos.ChronosOutput, gs gitState) string {
	var parts []string
	if co.Head.HashShort != "" {
		parts = append(parts, fmt.Sprintf("HEAD: %s — %s",
			co.Head.HashShort, co.Head.Message))
	}
	if gs.Changes > 0 {
		parts = append(parts, fmt.Sprintf("Working tree: %d archivos modificados", gs.Changes))
	}
	if gs.Branch != "" && gs.Branch != "unknown" {
		parts = append(parts, fmt.Sprintf("Branch: %s", gs.Branch))
	}
	if len(parts) == 0 {
		return "Sin contexto previo."
	}
	return strings.Join(parts, "\n")
}

// ═══════════════════════════════════════════════════════════════════════════
// CHRONOS COMPACT BLOCK
// ═══════════════════════════════════════════════════════════════════════════

func buildChronosBlock(co chronos.ChronosOutput) *ChronosBlock {
	if !co.Session.Detected && co.Head.HashShort == "" {
		return nil
	}

	block := &ChronosBlock{
		Now: NowCompact{
			ISO:     co.Now.ISO,
			Weekday: co.Now.Weekday,
			Hour:    co.Now.Hour,
			Minute:  co.Now.Minute,
		},
		HeadAge:             co.Head.AgeHuman,
		HeadISO:             co.Head.ISO,
		SessionContinuation: co.Session.IsContinuation,
		Timeline:            nil,
	}

	for i, t := range co.Timeline {
		if i >= timelineCount {
			break
		}
		block.Timeline = append(block.Timeline, TimelineEntry{
			Hash:     t.HashShort,
			AgeHuman: t.AgeHuman,
			Message:  t.Message,
		})
	}

	return block
}

// ═══════════════════════════════════════════════════════════════════════════
// CURRENT_HANDOFF.md GENERATOR
// ═══════════════════════════════════════════════════════════════════════════

func writeHandoff(repoRoot string, output GreetingOutput) {
	defer func() { _ = recover() }() // non-blocking

	handoffDir := filepath.Join(repoRoot, ".ovav", "context")
	os.MkdirAll(handoffDir, 0755)

	var lines []string
	lines = append(lines,
		"# CURRENT HANDOFF — GENERADO DESDE git HEAD + .ovav/plan/caps.yaml",
		"> SIN AUTORIDAD. Output derivado de git. Se regenera en cada sesión.",
		"",
		fmt.Sprintf("**Branch:** %s", output.Branch),
		fmt.Sprintf("**HEAD:** %s — %s",
			truncate(output.LatestWork.Ref, 12), output.LatestWork.Summary),
	)

	if output.Chronos != nil {
		lines = append(lines, fmt.Sprintf("**Hora:** %s", output.Chronos.Now.ISO))
	}
	lines = append(lines, fmt.Sprintf("**Cambios sin commit:** %d", output.GitStatus.TotalChanges))
	lines = append(lines, "", "## Últimos 5 commits (fuente: git log)")

	if output.Chronos != nil {
		for _, t := range output.Chronos.Timeline {
			lines = append(lines, fmt.Sprintf("- `%s` — %s (%s)",
				t.Hash, t.Message, t.AgeHuman))
		}
	}

	// Read plan priority (first 35 lines of caps.yaml)
	capsPath := filepath.Join(repoRoot, ".ovav", "plan", "caps.yaml")
	if data, err := os.ReadFile(capsPath); err == nil {
		capsLines := strings.Split(string(data), "\n")
		for i, line := range capsLines {
			if i >= 35 {
				break
			}
			if strings.Contains(line, "owc ") || strings.Contains(line, "MAXIMA") {
				lines = append(lines, "", "## Prioridad del plan")
				lines = append(lines, line)
				break
			}
		}
	}

	lines = append(lines, "", "*Archivo generado automáticamente. No editar manualmente.*")

	handoffPath := filepath.Join(handoffDir, "CURRENT_HANDOFF.md")
	os.WriteFile(handoffPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// ═══════════════════════════════════════════════════════════════════════════
// UTILITIES
// ═══════════════════════════════════════════════════════════════════════════

func detectRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot get working directory: %w", err)
	}
	// Walk up from cwd until we find .git
	for dir := cwd; ; dir = filepath.Dir(dir) {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return dir, nil
			}
			// .git file (worktree): read gitdir pointer
			if data, err := os.ReadFile(gitPath); err == nil {
				line := strings.TrimSpace(string(data))
				if strings.HasPrefix(line, "gitdir: ") {
					return dir, nil
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return cwd, fmt.Errorf("not in a git repository (no .git found)")
}

func statusBool(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

func statusBoolWarn(ok bool) string {
	if ok {
		return "ok"
	}
	return "warning"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// ═══════════════════════════════════════════════════════════════════════════
// HUMAN-READABLE OUTPUT
// ═══════════════════════════════════════════════════════════════════════════

func printHuman(output GreetingOutput) {
	fmt.Println("── OVAV Session Greeting v2.0 ──")
	fmt.Printf("  %s\n", output.GreetingText)

	// ── Auth block ────────────────────────────────────────────────
	auth := output.Auth
	if auth.Mode == "LOGIN" {
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════════════════════════╗")
		fmt.Println("║  OVAV Systems — AI Workstation Governor                 ║")
		fmt.Println("╠══════════════════════════════════════════════════════════╣")
		fmt.Printf("║  🔓 MODO: LOGIN          %s ACTIVE              ║\n", padRight(auth.Level, 12))
		ttl := auth.TTLRemaining
		if ttl == "" || ttl == "0m" {
			ttl = "< 1m"
		}
		fmt.Printf("║  ⏱ TTL: %s remaining   Extendida por actividad      ║\n", padRight(ttl, 12))
		fmt.Println("╚══════════════════════════════════════════════════════════╝")
	} else {
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════════════════════════╗")
		fmt.Println("║  OVAV Systems — AI Workstation Governor                 ║")
		fmt.Println("╠══════════════════════════════════════════════════════════╣")
		fmt.Println("║  🔒 MODO: NO_LOGIN      Seguridad Máxima              ║")
		fmt.Println("║  ℹ️ Operaciones requieren verificación guiada           ║")
		fmt.Println("╚══════════════════════════════════════════════════════════╝")
	}

	integrityIcon := "✅ LIMPIO"
	if !output.Integrity.Clean {
		integrityIcon = "⚠️  " + output.Integrity.Status
	}
	fmt.Printf("  Integridad: %s\n", integrityIcon)
	fmt.Printf("  Branch: %s\n", output.Branch)

	sessionIcon := "✅ marcada"
	if !output.Session.Marked {
		sessionIcon = "❌ error"
	}
	fmt.Printf("  Sesión:   %s\n", sessionIcon)

	if output.Chronos != nil {
		c := output.Chronos
		fmt.Printf("  Hora:     %s (%s)\n", c.Now.ISO, c.Now.Weekday)
		fmt.Printf("  HEAD:     %s (%s)\n", c.HeadAge, c.HeadISO)
		if c.SessionContinuation {
			fmt.Println("  Tipo:     continuación de sesión")
		} else {
			fmt.Println("  Tipo:     sesión nueva")
		}
		if len(c.Timeline) > 0 {
			fmt.Println("  Timeline:")
			for _, t := range c.Timeline {
				msg := t.Message
				if len(msg) > 55 {
					msg = msg[:55]
				}
				fmt.Printf("    %s  %25s  %s\n", t.Hash, t.AgeHuman, msg)
			}
		}
	}

	if output.GitMetrics.CommitCount > 0 {
		fmt.Printf("  Git:      %d commits, HEAD %s\n",
			output.GitMetrics.CommitCount, output.GitMetrics.HeadShort)
	}
	if output.GitStatus.TotalChanges > 0 {
		fmt.Printf("  Working tree: %d cambios\n", output.GitStatus.TotalChanges)
	}

	// Economy summary
	econ := output.Economy
	if len(econ) > 4 {
		var bs economy.BudgetStatus
		if json.Unmarshal(econ, &bs) == nil {
			fmt.Printf("  Budget:   $%.2f / $%.2f sesión (%.0f%%) | $%.2f / $%.2f mes (%.0f%%) [%s]\n",
				bs.SessionCostUSD, bs.SessionBudgetUSD, bs.SessionPct,
				bs.MonthlyCostUSD, bs.MonthlyBudgetUSD, bs.MonthlyPct,
				bs.AlertLevel)
		}
	}

	shouldRefresh := time.Now().Hour()%6 == 0
	if shouldRefresh {
		fmt.Println("  ℹ️  Recomendación: ejecutar monitoreo de presupuesto con 'ovav budget-alert'")
	}
}

// ── FDE Brain Injection ──────────────────────────────────────────────────

var leadToArea = map[string]string{
	"thavren": "platform_engineering",
	"eidren":  "research_intelligence",
	"valeria": "education_career",
	"dante":   "digital_product",
	"renata":  "health_performance",
	"sofia":   "commercial_growth",
	"elena":   "ux_design",
	"uriel":   "devops_infrastructure",
	"kenji":   "adversarial_intelligence",
	"camila":  "legal_compliance",
}

// injectBrain loads the FDE Brain Pack for a lead and returns it.
// Returns nil if the brain cannot be loaded (lead unknown, files missing).
func injectBrain(repoRoot, leadID string) *fde.BrainPack {
	areaID, ok := leadToArea[leadID]
	if !ok {
		return nil
	}
	brain, err := fde.LoadBrainPack(repoRoot, areaID, leadID)
	if err != nil {
		return nil
	}
	return brain
}
