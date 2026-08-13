package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ovav/ovav/internal/terminalconfig"
)

type windowsTerminalPlanOptions struct {
	settings string
	fragment string
}

func cmdTerminal(args []string) int {
	options, err := parseWindowsTerminalPlanArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV terminal: %v\n", err)
		return 2
	}
	current, err := os.ReadFile(options.settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV terminal: read installed settings: %v\n", err)
		return 1
	}
	fragment, err := os.ReadFile(options.fragment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV terminal: read fragment: %v\n", err)
		return 1
	}
	plan, err := terminalconfig.PlanWindowsTerminal(current, fragment, options.settings, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV terminal: %v\n", err)
		return 1
	}
	output := struct {
		Mode        string          `json:"mode"`
		Destination string          `json:"destination"`
		Backup      string          `json:"backup"`
		Merged      json.RawMessage `json:"merged"`
	}{
		Mode:        "dry-run",
		Destination: plan.Destination,
		Backup:      plan.Backup,
		Merged:      plan.Merged,
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV terminal: encode plan: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	return 0
}

func parseWindowsTerminalPlanArgs(args []string) (windowsTerminalPlanOptions, error) {
	options := windowsTerminalPlanOptions{}
	if len(args) < 2 || args[0] != "windows" || args[1] != "plan" {
		return options, fmt.Errorf("usage: ovav terminal windows plan --settings <settings.json> --fragment <ovav.fragment.json>")
	}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--settings":
			i++
			if i >= len(args) {
				return windowsTerminalPlanOptions{}, fmt.Errorf("--settings requires a path")
			}
			options.settings = args[i]
		case "--fragment":
			i++
			if i >= len(args) {
				return windowsTerminalPlanOptions{}, fmt.Errorf("--fragment requires a path")
			}
			options.fragment = args[i]
		case "--apply":
			return windowsTerminalPlanOptions{}, fmt.Errorf("global apply is unavailable; review the dry-run plan and use a separately approved installer")
		default:
			return windowsTerminalPlanOptions{}, fmt.Errorf("unknown option %s", args[i])
		}
	}
	if options.settings == "" || options.fragment == "" {
		return windowsTerminalPlanOptions{}, fmt.Errorf("--settings and --fragment are required")
	}
	return options, nil
}
