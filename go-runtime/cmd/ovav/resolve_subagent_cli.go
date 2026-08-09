// resolve_subagent_cli.go — Thin wrapper to invoke the resolve_subagent tool
// from the main ovav binary (cmd/ovav resolve-subagent <input>).
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/subagent"
)

func cmdResolveSubagent(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	// Strip --json
	cleaned := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" || a == "-json" {
			continue
		}
		cleaned = append(cleaned, a)
	}
	args = cleaned

	if len(args) == 0 {
		printResolveHelp()
		return 2
	}

	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: not in OVAV repo: %v\n", err)
		return 3
	}

	catalog, err := subagent.LoadCatalog(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 3
	}

	switch args[0] {
	case "--list", "-l":
		if jsonOut {
			out, _ := json.MarshalIndent(catalog.Agents, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("OVAV Subagent Catalog v%s — %d agents\n", catalog.Version, len(catalog.Agents))
			for _, a := range catalog.Agents {
				fmt.Printf("  %-30s %-6s %s\n", a.ID, a.Kind, a.Area)
			}
		}
		return 0

	case "--help", "-h":
		printResolveHelp()
		return 0
	}

	// Resolve
	input := args[0]
	res := catalog.Resolve(input)

	if jsonOut {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
	} else {
		if res.Error != "" {
			fmt.Printf("❌ %s\n", res.Error)
		} else if res.Ambiguous {
			fmt.Printf("⚠️  AMBIGUOUS: '%s' matches %d agents:\n", res.Input, len(res.AmbiguousIDs))
			for _, id := range res.AmbiguousIDs {
				fmt.Printf("   - %s\n", id)
			}
			if res.Suggestion != "" {
				fmt.Println()
				fmt.Println(res.Suggestion)
			}
		} else {
			var a subagent.Agent
			if len(res.ExactMatches) > 0 {
				a = res.ExactMatches[0]
				fmt.Printf("✅ EXACT: %s — %s\n", a.ID, a.Function)
			} else if len(res.AliasMatches) > 0 {
				a = res.AliasMatches[0]
				fmt.Printf("✅ ALIAS: %s — %s\n", a.ID, a.Function)
			}
		}
	}

	if res.Error != "" {
		return 2
	}
	if res.Ambiguous {
		return 1
	}
	return 0
}

func printResolveHelp() {
	fmt.Println(`ovav resolve-subagent — Resolve subagent intent to canonical id

Usage:
  ovav resolve-subagent <intent>    Resolve input to subagent_type
  ovav resolve-subagent --list      List all catalog agents
  ovav resolve-subagent --help      This help

Examples:
  ovav resolve-subagent elena                  → AMBIGUOUS (lead-elena + team-elena-frontend)
  ovav resolve-subagent team-elena-frontend    → EXACT MATCH
  ovav resolve-subagent frontend-elena         → ALIAS MATCH
  ovav resolve-subagent uriel                  → AMBIGUOUS (lead-uriel + team-uriel-devops)

Exit codes:
  0  Resolved uniquely
  1  Ambiguous
  2  Not found
  3  Catalog error`)
}
