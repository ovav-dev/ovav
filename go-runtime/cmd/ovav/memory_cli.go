// memory_cli.go — OVAV Agent Memory CLI
//
// Usage:
//
//	ovav memory store <topic> <summary> [--rule "operational rule"] [--agent thavren] [--tags foo,bar] [--priority HIGH]
//	ovav memory recall [--query "text"] [--tags foo,bar] [--agent eidren] [--limit 10]
//	ovav memory recent [--limit 5]
//	ovav memory stats
//	ovav memory verify [--card-id <id>]
//	ovav memory dump [--format yaml|json]
//
// Examples:
//
//	ovav memory store "Merge Readiness threshold" "Changed from 7d to 60d to avoid false positives on session files" --rule "Never block merge for files older than 60 days without explicit review" --agent thavren --tags governance,validator --priority HIGH
//
//	ovav memory recall --query "MergeReadiness" --limit 5
//
//	ovav memory recall --tags security --limit 10
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/memory"
)

func cmdMemory(args []string) int {
	if len(args) == 0 {
		printMemoryHelp()
		return 0
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "store", "add", "write", "save":
		return memoryStore(subArgs)
	case "recall", "query", "search", "find":
		return memoryRecall(subArgs)
	case "recent", "latest", "last":
		return memoryRecent(subArgs)
	case "stats", "stat", "status":
		return memoryStats(subArgs)
	case "verify", "check", "auth":
		return memoryVerify(subArgs)
	case "dump", "export":
		return memoryDump(subArgs)
	case "flush":
		return memoryFlush(subArgs)
	case "session", "autonomous":
		return memorySession(subArgs)
	case "install-hooks", "setup-hooks":
		return memoryInstallHooks(subArgs)
	case "help", "--help", "-h":
		printMemoryHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "ovav memory: unknown subcommand %q\n", sub)
		fmt.Fprintf(os.Stderr, "Run 'ovav memory help' for usage.\n")
		return 2
	}
}

func printMemoryHelp() {
	fmt.Print(`OVAV Agent Memory — Persistent memory for OVAV agents

Usage:
  ovav memory <subcommand> [flags]

Subcommands:
  store <topic> <summary>   Store a new memory card
  recall [--query text]      Recall memories matching query or tags
  recent [--limit N]         Show most recent memory cards
  stats                      Show memory statistics
  verify [--card-id ID]      Verify authenticity of memory cards
  dump [--format yaml|json]  Export all memory as YAML or JSON
  flush                      Flush the session write buffer to persistent memory
  session [--summary TEXT]   Show/propose session summary cards (autonomous)
  install-hooks             Install autonomous git hooks for post-commit memory

Examples:
  ovav memory store "MergeReadiness" "Changed threshold 7d→60d" \
    --rule "Never block merge for stale runtime files" \
    --agent thavren --tags governance --priority HIGH

  ovav memory recall --query "MergeReadiness" --limit 5

  ovav memory recall --tags security --limit 10

  ovav memory stats

  ovav memory flush                 Flush session buffer to persistent memory
  ovav memory session --summary X   Propose session summary card (autonomous)
  ovav memory session --commit HASH,MSG,AUTHOR,BRANCH   Propose commit card
  ovav memory session --error MSG,CTX  Propose error card
  ovav memory session --decision DEC,RAT,BY  Propose governance decision

Flags for store:
  --rule "operational rule"   The operational rule/decision (required)
  --agent ID                 Agent ID (default: thavren)
  --tags foo,bar             Comma-separated tags
  --priority LEVEL           CRITICAL|HIGH|NORMAL|LOW (default: NORMAL)
  --topic is now positional  Topic for contradiction detection (positional arg 1)

Flags for recall:
  --query TEXT               Free-text search
  --tags foo,bar             Filter by tags (AND)
  --agent ID                 Filter by agent
  --limit N                  Max results (default: 10)
  --min-relevance F          Minimum relevance 0.0-1.0 (default: 0)
`)
}

// ── Store ──────────────────────────────────────────────────────────────────

func memoryStore(args []string) int {
	fs := flag.NewFlagSet("ovav memory store", flag.ContinueOnError)
	fs.Usage = func() {}
	rule := fs.String("rule", "", "Operational rule/decision (required)")
	agent := fs.String("agent", "thavren", "Agent ID")
	tags := fs.String("tags", "", "Comma-separated tags")
	priority := fs.String("priority", "NORMAL", "Priority: CRITICAL|HIGH|NORMAL|LOW")
	format := fs.String("format", "text", "Output format: text|json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	positional := fs.Args()
	if len(positional) < 2 {
		fmt.Fprintf(os.Stderr, "ovav memory store: requires <topic> <summary>\n")
		return 2
	}
	cardTopic := strings.TrimSpace(positional[0])
	summary := positional[1]

	if *rule == "" {
		fmt.Fprintf(os.Stderr, "ovav memory store: --rule is required\n")
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory store: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory store: %v\n", err)
		return 2
	}

	// Parse tags
	var tagList []string
	if *tags != "" {
		for _, t := range strings.Split(*tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	card := memory.Card{
		Topic:           cardTopic,
		Summary:         summary,
		OperationalRule: *rule,
	}

	gitHead := guessGitHead(repoRoot)

	result, err := am.Store(card, memory.StoreOptions{
		AgentID:  *agent,
		Priority: *priority,
		Tags:     tagList,
		Commit:   gitHead,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory store: %v\n", err)
		return 1
	}

	if *format == "json" {
		j, _ := json.Marshal(map[string]interface{}{
			"id":     result.Card.ID,
			"topic":  result.Card.Topic,
			"tag":    result.Tag,
			"reason": result.Reason,
			"commit": result.Card.Commit,
		})
		fmt.Println(string(j))
	} else {
		fmt.Printf("✅ Memory stored\n")
		fmt.Printf("   ID:      %s\n", result.Card.ID)
		fmt.Printf("   Topic:   %s\n", result.Card.Topic)
		fmt.Printf("   Tag:     %s\n", result.Tag)
		fmt.Printf("   Agent:   %s\n", result.Card.ProposedBy)
		fmt.Printf("   Commit:  %s\n", shortHash(result.Card.Commit))
		if len(tagList) > 0 {
			fmt.Printf("   Tags:    %s\n", strings.Join(tagList, ", "))
		}
	}

	return 0
}

// ── Recall ─────────────────────────────────────────────────────────────────

func memoryRecall(args []string) int {
	fs := flag.NewFlagSet("ovav memory recall", flag.ContinueOnError)
	fs.Usage = func() {}
	query := fs.String("query", "", "Free-text search query")
	tags := fs.String("tags", "", "Comma-separated tags to filter by")
	agent := fs.String("agent", "", "Filter by agent ID")
	limit := fs.Int("limit", 10, "Maximum number of results")
	minRel := fs.Float64("min-relevance", 0, "Minimum relevance score 0.0-1.0")
	format := fs.String("format", "text", "Output format: text|json|compact")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory recall: %v\n", err)
		return 2
	}

	*query = strings.TrimSpace(*query)
	if *query == "" && *tags == "" {
		fmt.Fprintf(os.Stderr, "ovav memory recall: --query or --tags required\n")
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory recall: %v\n", err)
		return 2
	}

	var tagList []string
	if *tags != "" {
		for _, t := range strings.Split(*tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	results := am.Recall(memory.RecallOptions{
		Query:        *query,
		Tags:         tagList,
		AgentID:      *agent,
		Limit:        *limit,
		MinRelevance: *minRel,
	})

	if *format == "json" {
		j, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(j))
		return 0
	}

	if len(results.Cards) == 0 {
		fmt.Println("No memories found matching criteria.")
		return 0
	}

	fmt.Printf("Found %d memory(s) (total: %d, verified: %d, conflicts: %d)\n\n",
		len(results.Cards), results.Authenticity.Total,
		results.Authenticity.Verified, results.Authenticity.Conflicts)

	if results.Authenticity.Conflicts > 0 {
		fmt.Println("⚠️  Contradiction warnings:")
		for _, issue := range results.Authenticity.Issues {
			if strings.HasPrefix(issue, "contradiction") {
				fmt.Printf("   - %s\n", issue)
			}
		}
		fmt.Println()
	}

	for i, card := range results.Cards {
		prio := card.Priority
		if prio == "" {
			prio = "NORMAL"
		}
		fmt.Printf("[%d] %s | %s | %s\n", i+1, card.ID, prio, card.ProposedBy)
		fmt.Printf("    Topic: %s\n", card.Topic)
		fmt.Printf("    %s\n", card.Summary)
		if card.Commit != "" {
			fmt.Printf("    Commit: %s\n", shortHash(card.Commit))
		}
		if len(card.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(card.Tags, ", "))
		}
		fmt.Printf("    Status: %s | Confirmed: %s\n", card.Status, card.LastConfirmed)
		if *format == "compact" {
			continue
		}
		fmt.Println()
	}

	return 0
}

// ── Recent ──────────────────────────────────────────────────────────────────

func memoryRecent(args []string) int {
	fs := flag.NewFlagSet("ovav memory recent", flag.ContinueOnError)
	fs.Usage = func() {}
	limit := fs.Int("limit", 5, "Number of recent cards to show")
	format := fs.String("format", "text", "Output format: text|json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory recent: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory recent: %v\n", err)
		return 2
	}

	cards := am.Recent(*limit)

	if *format == "json" {
		j, _ := json.MarshalIndent(cards, "", "  ")
		fmt.Println(string(j))
		return 0
	}

	if len(cards) == 0 {
		fmt.Println("No memory cards yet. Run 'ovav memory store' to create one.")
		return 0
	}

	fmt.Printf("Recent %d memory card(s):\n\n", len(cards))
	for i, card := range cards {
		fmt.Printf("[%d] %s | %s | %s\n", i+1, card.ID, card.ProposedBy, card.Priority)
		fmt.Printf("    %s\n", card.Summary)
		fmt.Printf("    %s\n", card.OperationalRule)
		if card.Commit != "" {
			fmt.Printf("    @ %s | %s\n", shortHash(card.Commit), card.LastConfirmed)
		}
		if len(card.Tags) > 0 {
			fmt.Printf("    Tags: %s\n", strings.Join(card.Tags, ", "))
		}
		fmt.Println()
	}

	return 0
}

// ── Stats ──────────────────────────────────────────────────────────────────

func memoryStats(args []string) int {
	fs := flag.NewFlagSet("ovav memory stats", flag.ContinueOnError)
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory stats: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory stats: %v\n", err)
		return 2
	}

	total, active, byAgent, byTag := am.Stats()

	fmt.Printf("OVAV Agent Memory — Statistics\n")
	fmt.Printf("════════════════════════════════\n")
	fmt.Printf("  Total cards:   %d\n", total["total"])
	fmt.Printf("  Active cards:  %d\n", active["active"])
	fmt.Println()
	if len(byAgent) > 0 {
		fmt.Printf("  By agent:\n")
		for agent, count := range byAgent {
			fmt.Printf("    %-12s  %d\n", agent+":", count)
		}
		fmt.Println()
	}
	if len(byTag) > 0 {
		fmt.Printf("  By tag:\n")
		for tag, count := range byTag {
			fmt.Printf("    %-20s  %d\n", tag+":", count)
		}
	}

	return 0
}

// ── Verify ─────────────────────────────────────────────────────────────────

func memoryVerify(args []string) int {
	fs := flag.NewFlagSet("ovav memory verify", flag.ContinueOnError)
	fs.Usage = func() {}
	cardID := fs.String("card-id", "", "Verify specific card by ID")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory verify: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory verify: %v\n", err)
		return 2
	}

	var cards []memory.Card
	if *cardID != "" {
		found := am.Recall(memory.RecallOptions{Limit: 1})
		for _, c := range found.Cards {
			if c.ID == *cardID {
				cards = append(cards, c)
				break
			}
		}
		if len(cards) == 0 {
			fmt.Fprintf(os.Stderr, "ovav memory verify: card %q not found\n", *cardID)
			return 1
		}
	} else {
		cards = am.Recent(0)
	}

	report := am.Verify(cards)

	fmt.Printf("Authenticity Report — %d card(s) checked\n", report.Total)
	fmt.Printf("  ✅ Verified (valid hash):  %d\n", report.Verified)
	fmt.Printf("  ⚠️  Stale (commit gone):    %d\n", report.Stale)
	fmt.Printf("  ❌ No source chain:        %d\n", report.NoSource)
	fmt.Printf("  ⚠️  Contradictions:         %d\n", report.Conflicts)
	if len(report.Issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range report.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	if report.Conflicts > 0 || report.NoSource > 0 {
		return 1
	}
	return 0
}

// ── Dump ───────────────────────────────────────────────────────────────────

func memoryDump(args []string) int {
	fs := flag.NewFlagSet("ovav memory dump", flag.ContinueOnError)
	fs.Usage = func() {}
	format := fs.String("format", "yaml", "Format: yaml|json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory dump: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory dump: %v\n", err)
		return 2
	}

	cards := am.Recent(0)

	if *format == "json" {
		j, _ := json.MarshalIndent(cards, "", "  ")
		fmt.Println(string(j))
	} else {
		for _, card := range cards {
			fmt.Printf("- id: %s\n", card.ID)
			fmt.Printf("  topic: %s\n", card.Topic)
			fmt.Printf("  summary: %s\n", card.Summary)
			fmt.Printf("  operational_rule: %s\n", card.OperationalRule)
			fmt.Printf("  status: %s\n", card.Status)
			fmt.Printf("  priority: %s\n", card.Priority)
			fmt.Printf("  proposed_by: %s\n", card.ProposedBy)
			fmt.Printf("  commit: %s\n", card.Commit)
			fmt.Printf("  tags: [%s]\n", strings.Join(card.Tags, ", "))
			fmt.Printf("  last_confirmed: %s\n\n", card.LastConfirmed)
		}
	}

	return 0
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func guessGitHead(root string) string {
	headPath := filepath.Join(root, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if len(content) > len(prefix) && content[:len(prefix)] == prefix {
		branch := content[len(prefix):]
		refPath := filepath.Join(root, ".git", "refs", "heads", branch)
		refData, err := os.ReadFile(refPath)
		if err == nil {
			return strings.TrimSpace(string(refData))
		}
	}
	if !strings.Contains(content, "ref:") {
		return content
	}
	return ""
}

func shortHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

// ── Flush ───────────────────────────────────────────────────────────────────

func memoryFlush(args []string) int {
	fs := flag.NewFlagSet("ovav memory flush", flag.ContinueOnError)
	fs.Usage = func() {}
	stage := fs.Bool("stage", false, "Stage to disk without flushing (crash recovery)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory flush: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory flush: %v\n", err)
		return 2
	}

	sw := memory.NewSessionWriter(am, repoRoot)

	if *stage {
		if err := sw.Stage(); err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory flush: stage: %v\n", err)
			return 1
		}
		fmt.Println("Session buffer staged to disk.")
		return 0
	}

	n, err := sw.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory flush: %v\n", err)
		fmt.Printf("Flushed %d card(s) before error.\n", n)
		return 1
	}

	if n == 0 {
		fmt.Println("No cards in session buffer to flush.")
		return 0
	}

	fmt.Printf("✅ Flushed %d card(s) to persistent memory.\n", n)
	return 0
}

// ── Session (autonomous) ───────────────────────────────────────────────────────

func memorySession(args []string) int {
	fs := flag.NewFlagSet("ovav memory session", flag.ContinueOnError)
	fs.Usage = func() {}
	summary := fs.String("summary", "", "Session summary text to propose as memory card")
	tasks := fs.String("tasks", "", "Comma-separated list of tasks completed")
	sessionID := fs.String("session-id", "", "Session ID (default: read from session_marker)")
	proposeCommit := fs.String("commit", "", "Propose a git commit event: hash,message,author,branch")
	proposeError := fs.String("error", "", "Propose an error card: error_msg,context")
	proposeDecision := fs.String("decision", "", "Propose a decision card: decision,rationale,by")
	proposeValidator := fs.String("validator", "", "Propose validator result: name,passed,issues")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory session: %v\n", err)
		return 2
	}

	am, err := memory.NewAgentMemory(repoRoot, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory session: %v\n", err)
		return 2
	}

	sw := memory.NewSessionWriter(am, repoRoot)

	switch {
	case *sessionID != "" && *summary != "":
		// Propose session summary
		var taskList []string
		if *tasks != "" {
			for _, t := range strings.Split(*tasks, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					taskList = append(taskList, t)
				}
			}
		}
		if err := sw.ProposeSessionSummary(*sessionID, *summary, taskList); err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory session: propose: %v\n", err)
			return 1
		}
		n, err := sw.Flush()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory session: flush: %v\n", err)
			return 1
		}
		fmt.Printf("✅ Session card stored (%d card(s)).\n", n)

	case *proposeCommit != "":
		// Format: hash,message,author,branch
		parts := strings.SplitN(*proposeCommit, ",", 4)
		if len(parts) < 4 {
			fmt.Fprintf(os.Stderr, "ovav memory session: --commit needs hash,message,author,branch\n")
			return 2
		}
		if err := sw.ProposeCommit(parts[0], parts[1], parts[2], parts[3]); err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory session: %v\n", err)
			return 1
		}
		n, _ := sw.Flush()
		fmt.Printf("✅ Commit card stored (%d card(s)).\n", n)

	case *proposeError != "":
		// Format: error_msg,context
		parts := strings.SplitN(*proposeError, ",", 2)
		if len(parts) < 2 {
			fmt.Fprintf(os.Stderr, "ovav memory session: --error needs error_msg,context\n")
			return 2
		}
		if err := sw.ProposeError(parts[0], parts[1]); err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory session: %v\n", err)
			return 1
		}
		n, _ := sw.Flush()
		fmt.Printf("✅ Error card stored (%d card(s)).\n", n)

	case *proposeDecision != "":
		// Format: decision,rationale,by
		parts := strings.SplitN(*proposeDecision, ",", 3)
		if len(parts) < 3 {
			fmt.Fprintf(os.Stderr, "ovav memory session: --decision needs decision,rationale,by\n")
			return 2
		}
		if err := sw.ProposeDecision(parts[0], parts[1], parts[2]); err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory session: %v\n", err)
			return 1
		}
		n, _ := sw.Flush()
		fmt.Printf("✅ Decision card stored (%d card(s)).\n", n)

	case *proposeValidator != "":
		// Format: name,passed,issues (issues is comma-separated)
		parts := strings.SplitN(*proposeValidator, ",", 3)
		if len(parts) < 2 {
			fmt.Fprintf(os.Stderr, "ovav memory session: --validator needs name,passed,issues\n")
			return 2
		}
		var issueList []string
		if len(parts) > 2 && parts[2] != "" {
			for _, i := range strings.Split(parts[2], ";") {
				i = strings.TrimSpace(i)
				if i != "" {
					issueList = append(issueList, i)
				}
			}
		}
		passed := strings.TrimSpace(parts[1]) == "true" || parts[1] == "PASS"
		if err := sw.ProposeValidatorResult(parts[0], passed, issueList); err != nil {
			fmt.Fprintf(os.Stderr, "ovav memory session: %v\n", err)
			return 1
		}
		n, _ := sw.Flush()
		fmt.Printf("✅ Validator card stored (%d card(s)).\n", n)

	default:
		// Show session buffer status
		bufLen := sw.BufferLen()
		flushCount := sw.FlushCount()
		if bufLen == 0 && flushCount == 0 {
			fmt.Println("Session writer: no active session buffer.")
			fmt.Println("Run 'ovav memory session --summary TEXT' to propose a session card.")
			fmt.Println("Or use --commit, --error, --decision, --validator for autonomous events.")
			return 0
		}
		fmt.Printf("Session writer state:\n")
		fmt.Printf("  Buffer:   %d card(s) pending\n", bufLen)
		fmt.Printf("  Flushes:  %d\n", flushCount)
		if sw.LastErr() != nil {
			fmt.Printf("  Last err: %v\n", sw.LastErr())
		}
	}
	return 0
}

// postCommitHook is the content of the .githooks/post-commit script.
// This script auto-stores every commit as a memory card — true autonomous memory.
// Installed via: git config core.hooksPath <repo>/.githooks
const postCommitHook = `#!/bin/bash
# OVAV Agent Memory — Autonomous post-commit hook
# Auto-stores every git commit as a memory card.
# Generated by 'ovav memory install-hooks'. Do not edit manually.

REPO_ROOT="$(git rev-parse --show-toplevel)"
COMMIT_HASH="$(git rev-parse HEAD)"
COMMIT_MSG="$(git log -1 --format=%s)"
COMMIT_AUTHOR="$(git log -1 --format=%an)"
CURRENT_BRANCH="$(git branch --show-current)"

# Skip if no OVAV binary available
if ! command -v ovav &>/dev/null; then
    exit 0
fi

# Call OVAV memory to store the commit card (non-blocking, silent on failure)
ovav memory session \
    --commit "$COMMIT_HASH,$COMMIT_MSG,$COMMIT_AUTHOR,$CURRENT_BRANCH" \
    2>/dev/null &

exit 0
`

func memoryInstallHooks(args []string) int {
	fs := flag.NewFlagSet("ovav memory install-hooks", flag.ContinueOnError)
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory install-hooks: %v\n", err)
		return 2
	}

	hooksDir := filepath.Join(repoRoot, ".githooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory install-hooks: mkdir %s: %v\n", hooksDir, err)
		return 1
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte(postCommitHook), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory install-hooks: write %s: %v\n", hookPath, err)
		return 1
	}

	cmd := exec.Command("git", "config", "core.hooksPath", hooksDir)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "ovav memory install-hooks: git config: %v\n%s\n", err, out)
		return 1
	}

	fmt.Printf("✅ OVAV autonomous memory hooks installed.\n")
	fmt.Printf("   Hooks dir: %s\n", hooksDir)
	fmt.Printf("   Git config: core.hooksPath = %s\n", hooksDir)
	fmt.Printf("\n")
	fmt.Printf("Every 'git commit' will now automatically store a memory card.\n")
	fmt.Printf("Run 'git commit' to trigger the first autonomous memory entry.\n")

	return 0
}
