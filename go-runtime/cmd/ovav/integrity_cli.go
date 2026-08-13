package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/truststore"
	"github.com/ovav/ovav/internal/validators"
)

// integrityOptions captures CLI flags for the baseline subcommand.
type integrityOptions struct {
	write bool
	help  bool
}

// integrityGateOptions captures CLI flags for the `integrity gate refresh`
// subcommand. We intentionally keep the option set minimal — auth is gated
// on session_marker OR --ceowaiver, both checked at dispatch time.
type integrityGateOptions struct {
	ceoWaiver bool
	help      bool
}

func cmdIntegrity(args []string) int {
	if len(args) == 0 {
		printIntegrityHelp()
		return 0
	}
	switch args[0] {
	case "baseline":
		return cmdIntegrityBaseline(args[1:])
	case "gate":
		return cmdIntegrityGate(args[1:])
	case "help", "--help", "-h":
		printIntegrityHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV integrity: unknown subcommand %q\n", args[0])
		printIntegrityHelp()
		return 2
	}
}

// ── baseline ────────────────────────────────────────────────────────────────

func cmdIntegrityBaseline(args []string) int {
	options, err := parseIntegrityArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity baseline: %v\n", err)
		return 2
	}
	if options.help {
		printIntegrityHelp()
		return 0
	}
	root, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity baseline: %v\n", err)
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
	for _, arg := range args {
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

// ── gate refresh ────────────────────────────────────────────────────────────

func cmdIntegrityGate(args []string) int {
	if len(args) == 0 {
		printIntegrityGateHelp()
		return 0
	}
	switch args[0] {
	case "refresh":
		return cmdIntegrityGateRefresh(args[1:])
	case "help", "--help", "-h":
		printIntegrityGateHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV integrity gate: unknown subcommand %q\n", args[0])
		printIntegrityGateHelp()
		return 2
	}
}

func cmdIntegrityGateRefresh(args []string) int {
	options, err := parseIntegrityGateArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity gate refresh: %v\n", err)
		return 2
	}
	if options.help {
		printIntegrityGateHelp()
		return 0
	}

	root, err := cli.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity gate refresh: %v\n", err)
		return 1
	}

	// Auth: session_marker must exist OR caller must pass --ceowaiver.
	authReason := ""
	switch {
	case integritySessionMarkerPresent(root):
		authReason = "session_marker"
	case options.ceoWaiver:
		authReason = "ceowaiver"
	default:
		fmt.Fprintln(os.Stderr, "OVAV integrity gate refresh: refused — no .ovav/runtime/.session_marker present and --ceowaiver not set")
		return 1
	}

	prev, next, err := truststore.RefreshGateHash(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV integrity gate refresh: %v\n", err)
		return 1
	}

	// Pretty-print: previous vs new 16-char prefix for human eyes.
	prevPrefix := prev
	if len(prevPrefix) > 16 {
		prevPrefix = prevPrefix[:16]
	}
	nextPrefix := next
	if len(nextPrefix) > 16 {
		nextPrefix = nextPrefix[:16]
	}
	fmt.Printf("🟢 gate hash refreshed: %s... → %s...\n", prevPrefix, nextPrefix)

	// Audit append — best-effort; never fail the refresh because the audit log
	// itself is unwritable. We log to stderr so the operator sees the warning.
	if err := appendGateRefreshAudit(root, authReason, prev, next); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  audit append failed: %v\n", err)
	}

	return 0
}

func parseIntegrityGateArgs(args []string) (integrityGateOptions, error) {
	options := integrityGateOptions{}
	for _, arg := range args {
		switch arg {
		case "--ceowaiver":
			options.ceoWaiver = true
		case "--help", "-h":
			options.help = true
		default:
			return integrityGateOptions{}, fmt.Errorf("unknown option %s", arg)
		}
	}
	return options, nil
}

// integritySessionMarkerPresent returns true when .ovav/runtime/.session_marker
// exists and is non-empty. Mirrors the auth gate used by GateSelfProtection.
func integritySessionMarkerPresent(root string) bool {
	marker := filepath.Join(root, ".ovav", "runtime", ".session_marker")
	data, err := os.ReadFile(marker)
	if err != nil {
		return false
	}
	for _, b := range data {
		if b == ' ' || b == '\n' || b == '\t' || b == '\r' {
			continue
		}
		return true
	}
	return false
}

// appendGateRefreshAudit writes one JSONL line to .ovav/runtime/logs/gate_refresh.jsonl.
// Failure here is non-fatal but logged by the caller.
func appendGateRefreshAudit(root, authReason, prev, next string) error {
	logDir := filepath.Join(root, ".ovav", "runtime", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	logPath := filepath.Join(logDir, "gate_refresh.jsonl")
	entry := map[string]interface{}{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"action":        "gate_refresh",
		"auth":          authReason,
		"previous_hash": prev,
		"new_hash":      next,
		"gate_file":     truststore.GateRelPath(),
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// ── help ────────────────────────────────────────────────────────────────────

func printIntegrityHelp() {
	data, _ := json.Marshal(map[string]string{
		"baseline":    "ovav integrity baseline [--plan|--write]",
		"gate_refresh": "ovav integrity gate refresh [--ceowaiver]",
		"gate_help":    "ovav integrity gate refresh needs .ovav/runtime/.session_marker OR --ceowaiver",
	})
	fmt.Println(string(data))
}

func printIntegrityGateHelp() {
	data, _ := json.Marshal(map[string]string{
		"refresh": "ovav integrity gate refresh [--ceowaiver]",
		"auth":    "requires .ovav/runtime/.session_marker OR --ceowaiver",
		"effect":  "recomputes SHA-256 of host_config_drift.go and writes .ovav/runtime/gate_state.json atomically",
	})
	fmt.Println(string(data))
}
