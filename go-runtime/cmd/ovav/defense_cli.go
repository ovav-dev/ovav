// OVAV Go Runtime — C9.1: Defense CLI commands
//
// ovav defend          Defense status dashboard
// ovav defend status   Show defense posture
// ovav defend scan     Run active defense scan
// ovav defend lockdown Toggle system lockdown
//
// Wires internal/security/defense/ into the OVAV CLI.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/security/defense"
)

// ── Defense command routing ──────────────────────────────────────────────────

func cmdDefend(args []string) int {
	sub := "status"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}

	switch sub {
	case "status":
		return defendStatus(args)
	case "scan":
		return defendScan(args)
	case "lockdown":
		return defendLockdown(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown defense subcommand: %s\n", sub)
		fmt.Println("Usage: ovav defend [status|scan|lockdown]")
		return 1
	}
}

// ── defend status — defense posture dashboard ────────────────────────────────

func defendStatus(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	cortex := defense.NewCortex()
	responder := defense.NewResponder(cortex)
	credMgr := defense.NewCredentialManager()

	hardening := cortex.HardeningState()
	lockdown := responder.IsLockdownActive()
	needsRotation := credMgr.NeedsRotation()
	credCount := credMgr.Count()

	result := map[string]interface{}{
		"command": "defend status",
		"status":  "ok",
		"posture": map[string]interface{}{
			"cortex_active":        true,
			"responder_active":     true,
			"lockdown":             lockdown,
			"hardening_components": len(hardening),
			"hardening_state":      hardening,
			"credentials_managed":  credCount,
			"needs_rotation":       needsRotation,
			"false_positives":      cortex.FalsePositiveCount(),
		},
	}

	if jsonOut {
		cli.Output(result, true)
		return 0
	}

	fmt.Println("═══ OVAV Defense — Security Posture ─────────────────────")
	fmt.Printf("  Cortex:       🟢 Active\n")
	fmt.Printf("  Responder:    🟢 Active\n")
	fmt.Printf("  Lockdown:     %s\n", lockdownBanner(lockdown))
	fmt.Printf("  Hardening:    %d components configured\n", len(hardening))
	for component, action := range hardening {
		fmt.Printf("    %s → %s\n", component, action)
	}
	fmt.Printf("  Credentials:  %d managed, rotation %s\n", credCount, rotationBanner(needsRotation))
	fmt.Println("══════════════════════════════════════════════════════════")

	if lockdown {
		return 2
	}
	return 0
}

// ── defend scan — run active defense scan ────────────────────────────────────

func defendScan(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	cortex := defense.NewCortex()
	responder := defense.NewResponder(cortex)

	// Resolve repo root for absolute path checks
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		repoRoot = "."
	}

	// Run real-time defense scan — live filesystem + git checks, no canned data
	scanResult, err := defense.RunActiveScan(repoRoot)
	if err != nil {
		cli.Output(map[string]interface{}{"command": "defend scan", "status": "error", "error": err.Error()}, jsonOut)
		return 1
	}
	if scanResult == nil {
		cli.Output(map[string]interface{}{"command": "defend scan", "status": "error", "error": "scan returned nil"}, jsonOut)
		return 1
	}

	var events []map[string]interface{}
	criticalCount := 0

	for _, ev := range scanResult.Events {
		// Check if known false positive
		if cortex.IsKnownFalsePositive(ev.Type, ev.Path) {
			continue
		}

		// Classify root cause
		rootCause := cortex.ClassifyRootCause(ev.Type, ev.Path)

		// Get response actions from responder
		actions := responder.Respond(ev.Type, ev.Severity, ev.Path)

		event := map[string]interface{}{
			"type":       ev.Type,
			"severity":   ev.Severity,
			"path":       ev.Path,
			"root_cause": rootCause,
			"actions":    actions,
			"message":    ev.Message,
		}
		events = append(events, event)

		if ev.Severity == defense.SevCritical || ev.Severity == defense.SevDeadly {
			criticalCount++
		}
	}

	out := map[string]interface{}{
		"command":  "defend scan",
		"status":   "ok",
		"scanned":  scanResult.SurfacesChecked,
		"events":   events,
		"critical": criticalCount,
	}

	if jsonOut {
		cli.Output(out, true)
		return 0
	}

	fmt.Printf("Defense Scan — %d surfaces checked, %d events, %d critical\n",
		scanResult.SurfacesChecked, len(events), criticalCount)
	fmt.Println()

	for _, e := range events {
		sev := e["severity"].(defense.Severity)
		fmt.Printf("  %s [%s] %s\n", severityIcon(sev), sev, e["type"])
		fmt.Printf("       Path: %s\n", e["path"])
		if msg, ok := e["message"].(string); ok && msg != "" {
			fmt.Printf("       Detail: %s\n", msg)
		}
		fmt.Printf("       Root: %s\n", e["root_cause"])
		actions := e["actions"].([]defense.ResponseAction)
		fmt.Printf("       Actions: %v\n", actions)
		fmt.Println()
	}

	if criticalCount > 0 {
		return 2
	}
	return 0
}

// ── defend lockdown — toggle system lockdown ─────────────────────────────────

func defendLockdown(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	cortex := defense.NewCortex()
	responder := defense.NewResponder(cortex)

	// Parse: ovav defend lockdown [on|off]
	activate := !responder.IsLockdownActive() // toggle default
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "on", "enable", "activate":
			activate = true
		case "off", "disable", "deactivate":
			activate = false
		}
	}

	responder.SetLockdown(activate)
	status := responder.IsLockdownActive()

	result := map[string]interface{}{
		"command":  "defend lockdown",
		"lockdown": status,
	}

	if jsonOut {
		cli.Output(result, true)
		return 0
	}

	if status {
		fmt.Println("🔒 System LOCKDOWN active — all non-immune writes blocked.")
	} else {
		fmt.Println("🔓 System lockdown released — normal operations restored.")
	}

	return 0
}

// ── Display helpers ──────────────────────────────────────────────────────────

func severityIcon(s defense.Severity) string {
	switch s {
	case defense.SevInfo:
		return "ℹ️"
	case defense.SevWarning:
		return "⚠️"
	case defense.SevCritical:
		return "🔴"
	case defense.SevDeadly:
		return "💀"
	default:
		return "•"
	}
}

func lockdownBanner(active bool) string {
	if active {
		return "🔒 LOCKDOWN"
	}
	return "🔓 OPEN"
}

func rotationBanner(needs bool) string {
	if needs {
		return "⚠️ REQUIRED"
	}
	return "✅ OK"
}
