package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/validators"
)

type integrityOptions struct {
	write bool
	help  bool
}

func cmdIntegrity(args []string) int {
	options, err := parseIntegrityArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity: %v\n", err)
		return 2
	}
	if options.help {
		printIntegrityHelp()
		return 0
	}
	root, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity: %v\n", err)
		return 1
	}

	var baseline validators.IntegrityBaseline
	if options.write {
		baseline, err = validators.WriteIntegrityBaseline(root)
	} else {
		baseline, err = validators.PlanIntegrityBaseline(root)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity baseline: %v\n", err)
		return 1
	}
	if options.write {
		fmt.Println("Wrote .ovav/integrity_backups/baseline.json")
		return 0
	}
	os.Stdout.Write(baseline.JSON)
	return 0
}

func parseIntegrityArgs(args []string) (integrityOptions, error) {
	options := integrityOptions{}
	if len(args) == 0 || args[0] != "baseline" {
		return options, fmt.Errorf("usage: ovav integrity baseline [--plan|--write]")
	}
	for _, arg := range args[1:] {
		switch arg {
		case "--plan":
		case "--write":
			options.write = true
		case "--help", "-h":
			options.help = true
		default:
			return integrityOptions{}, fmt.Errorf("unknown option %s", arg)
		}
	}
	return options, nil
}

func printIntegrityHelp() {
	data, _ := json.Marshal(map[string]string{
		"default": "dry-run plan",
		"plan":    "ovav integrity baseline --plan",
		"write":   "ovav integrity baseline --write (isolated feature branch; exact candidate staged or committed; no unstaged/untracked changes)",
	})
	fmt.Println(string(data))
}
