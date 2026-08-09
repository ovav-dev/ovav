// Command resolve-subagent resolves an intent or alias to a canonical subagent_type.
//
// Usage:
//
//	go run ./go-runtime/cmd/resolve_subagent <intent_or_alias>
//	go run ./go-runtime/cmd/resolve_subagent --list
//	go run ./go-runtime/cmd/resolve_subagent --check <id>
//
// Examples:
//
//	go run ./go-runtime/cmd/resolve_subagent elena
//	  → AMBIGUOUS: lead-elena + team-elena-frontend
//
//	go run ./go-runtime/cmd/resolve_subagent frontend-elena
//	  → team-elena-frontend
//
//	go run ./go-runtime/cmd/resolve_subagent team-elena-frontend
//	  → team-elena-frontend (exact)
//
// Exit codes:
//
//	0  Resolved uniquely
//	1  Ambiguous (multiple matches)
//	2  Not found
//	3  Catalog error
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/subagent"
)

func main() {
	jsonOut := cli.HasJSONFlag(os.Args[1:])
	args := stripJSONFlag(os.Args[1:])

	if len(args) == 0 {
		printUsage()
		os.Exit(2)
	}

	// Subcommands
	switch args[0] {
	case "--list", "-l":
		runList(jsonOut)
		return
	case "--check":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "ERROR: --check requires an id")
			os.Exit(2)
		}
		runCheck(args[1], jsonOut)
		return
	case "--help", "-h":
		printUsage()
		return
	}

	// Default: resolve the input
	input := args[0]
	runResolve(input, jsonOut)
}

// stripJSONFlag removes --json / -json from args (HasJSONFlag already detected it).
func stripJSONFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" || a == "-json" {
			continue
		}
		out = append(out, a)
	}
	return out
}

func runResolve(input string, jsonOut bool) {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fail("not in OVAV repo: "+err.Error(), jsonOut, 3)
		return
	}

	catalog, err := subagent.LoadCatalog(repoRoot)
	if err != nil {
		fail(err.Error(), jsonOut, 3)
		return
	}

	res := catalog.Resolve(input)

	if jsonOut {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		printResolutionHuman(res)
	}

	switch {
	case res.Error != "":
		os.Exit(2)
	case res.Ambiguous:
		os.Exit(1)
	}
}

func runList(jsonOut bool) {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fail(err.Error(), jsonOut, 3)
		return
	}

	catalog, err := subagent.LoadCatalog(repoRoot)
	if err != nil {
		fail(err.Error(), jsonOut, 3)
		return
	}

	if jsonOut {
		out, _ := json.MarshalIndent(catalog.Agents, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Printf("OVAV Subagent Catalog v%s\n\n", catalog.Version)
	fmt.Printf("%-30s %-6s %-25s %s\n", "ID", "KIND", "AREA", "FUNCTION")
	fmt.Println(strings.Repeat("─", 100))
	for _, a := range catalog.Agents {
		fn := a.Function
		if len(fn) > 50 {
			fn = fn[:47] + "..."
		}
		fmt.Printf("%-30s %-6s %-25s %s\n", a.ID, a.Kind, a.Area, fn)
	}
	fmt.Printf("\nTotal: %d agents\n", len(catalog.Agents))
}

func runCheck(id string, jsonOut bool) {
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fail(err.Error(), jsonOut, 3)
		return
	}

	catalog, err := subagent.LoadCatalog(repoRoot)
	if err != nil {
		fail(err.Error(), jsonOut, 3)
		return
	}

	r := catalog.Resolve(id)
	if jsonOut {
		out, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(out))
		return
	}

	if r.Error != "" {
		fmt.Printf("❌ %s\n", r.Error)
		os.Exit(2)
	}
	if r.Ambiguous {
		fmt.Printf("⚠️  Ambiguous: %s\n", strings.Join(r.AmbiguousIDs, ", "))
		if r.Suggestion != "" {
			fmt.Println(r.Suggestion)
		}
		os.Exit(1)
	}

	// Single match
	var a subagent.Agent
	if len(r.ExactMatches) > 0 {
		a = r.ExactMatches[0]
	} else if len(r.AliasMatches) > 0 {
		a = r.AliasMatches[0]
	}

	fmt.Printf("✅ %s\n", a.ID)
	fmt.Printf("   Name:     %s\n", a.Name)
	fmt.Printf("   Kind:     %s\n", a.Kind)
	fmt.Printf("   Area:     %s\n", a.Area)
	if a.Lead != nil {
		fmt.Printf("   Lead:     %s\n", *a.Lead)
	}
	fmt.Printf("   Function: %s\n", a.Function)
	if len(a.Aliases) > 0 {
		fmt.Printf("   Aliases:  %s\n", strings.Join(a.Aliases, ", "))
	}
	if a.Note != "" {
		fmt.Printf("   Note:     %s\n", a.Note)
	}
}

func printResolutionHuman(r subagent.Resolution) {
	if r.Error != "" {
		fmt.Printf("❌ %s\n", r.Error)
		return
	}

	if r.Ambiguous {
		fmt.Printf("⚠️  AMBIGUOUS: '%s' matches %d agents:\n", r.Input, len(r.AmbiguousIDs))
		for _, id := range r.AmbiguousIDs {
			fmt.Printf("   - %s\n", id)
		}
		if r.Suggestion != "" {
			fmt.Println()
			fmt.Println(r.Suggestion)
		}
		return
	}

	var a subagent.Agent
	if len(r.ExactMatches) > 0 {
		a = r.ExactMatches[0]
		fmt.Printf("✅ EXACT MATCH: %s\n", a.ID)
	} else if len(r.AliasMatches) > 0 {
		a = r.AliasMatches[0]
		fmt.Printf("✅ ALIAS MATCH: %s (via alias '%s')\n", a.ID, r.Input)
	}
	fmt.Printf("   Name:     %s\n", a.Name)
	fmt.Printf("   Area:     %s\n", a.Area)
	if a.Lead != nil {
		fmt.Printf("   Lead:     %s\n", *a.Lead)
	}
	fmt.Printf("   Function: %s\n", a.Function)
	if len(a.DisambiguatesFrom) > 0 {
		fmt.Printf("   ⚠️  Disambiguates from: %s\n", strings.Join(a.DisambiguatesFrom, "; "))
	}
}

func printUsage() {
	fmt.Println(`ovav resolve-subagent — Resolve subagent intent to canonical id

Usage:
  ovav resolve-subagent <intent_or_alias>    Resolve a single input
  ovav resolve-subagent --list               List all catalog agents
  ovav resolve-subagent --check <id>         Check a specific id
  ovav resolve-subagent --json               Output JSON

Examples:
  ovav resolve-subagent elena                  → AMBIGUOUS (lead-elena, team-elena-frontend)
  ovav resolve-subagent team-elena-frontend    → exact match
  ovav resolve-subagent frontend-elena         → alias match
  ovav resolve-subagent --check team-sergio    → detail view

Exit codes:
  0  Resolved uniquely
  1  Ambiguous (multiple matches — disambiguate)
  2  Not found / invalid input
  3  Catalog error`)
}

func fail(msg string, jsonOut bool, code int) {
	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{"error": msg}, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
	}
	os.Exit(code)
}
