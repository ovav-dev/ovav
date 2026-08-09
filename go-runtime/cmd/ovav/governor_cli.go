// OVAV Go Runtime — C9.1: Governor CLI commands
//
// ovav govern         Governor health + decisions + trust status
// ovav govern health  Unified health scores
// ovav govern decide  Run decision engine
// ovav govern trust   Verify trust gate on claims
//
// Wires internal/governor/ into the OVAV CLI.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ovav/ovav/internal/cli"
	"github.com/ovav/ovav/internal/governor"
)

// ── Governor command routing ─────────────────────────────────────────────────

func cmdGovern(args []string) int {
	sub := "status"
	if len(args) > 0 {
		sub = args[0]
		args = args[1:]
	}

	switch sub {
	case "health":
		return governHealth(args)
	case "decide":
		return governDecide(args)
	case "trust":
		return governTrust(args)
	case "status", "":
		return governStatus(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown governor subcommand: %s\n", sub)
		fmt.Println("Usage: ovav govern [health|decide|trust|status]")
		return 1
	}
}

// ── govern status — full governor dashboard ──────────────────────────────────

func governStatus(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	// Unified health via existing governor API
	p, f, t, _ := governor.QuickIntegrityMesh()
	integrity := governor.IntegrityMeshHealth(p, f, t)
	o, w, c, _ := governor.QuickSelfDiagnosis()
	sd := governor.SelfDiagnosisHealth(o, w, c)
	pain := governor.PainScorerHealth(
		governor.QuickPainScorer(),
	)
	health := governor.ComputeUnifiedHealth(integrity, sd, pain)

	// Decision engine
	state := governor.GatherSystemState()
	decisions := governor.Decide(state)

	// Trust gate
	trustResult := governor.VerifyTrust(governor.TrustInput{
		LeadName:     "thavren",
		OutputClaims: []string{"34/34 go tests passing", "0 data races"},
	})

	result := map[string]interface{}{
		"command":   "govern",
		"status":    "ok",
		"health":    health,
		"decisions": decisions,
		"trust":     trustResult,
	}

	if jsonOut {
		cli.Output(result, true)
		return 0
	}

	// Human-readable output
	imScore := 0.0
	sdScore := 0.0
	painScore := 0.0
	if s, ok := health.Subsystems["integrity_mesh"]; ok {
		imScore = s.Score
	}
	if s, ok := health.Subsystems["self_diagnosis"]; ok {
		sdScore = s.Score
	}
	if s, ok := health.Subsystems["pain_scorer"]; ok {
		painScore = s.Score
	}

	fmt.Println("═══ OVAV Governor ──────────────────────────────────────")
	fmt.Printf("  Health Score:  %.1f%%  %s\n", health.CompositeScore, healthBanner(health.CompositeScore))
	fmt.Printf("  │  Integrity Mesh:  %.1f%%\n", imScore)
	fmt.Printf("  │  Self Diagnosis:  %.1f%%\n", sdScore)
	fmt.Printf("  │  Pain Scorer:     %.1f%%\n", painScore)
	fmt.Printf("  │  Status:          %s\n", health.Overall)

	fmt.Printf("\n  Decisions:         %d pending\n", len(decisions))
	for _, d := range decisions {
		fmt.Printf("  │  [%s] %s → %s (%s)\n", d.Priority, d.Domain, d.Action, d.Reason)
	}

	fmt.Printf("\n  Trust Gate:        %s\n", trustResult.Action)
	fmt.Println("══════════════════════════════════════════════════════════")

	if governor.HasCriticalDecisions(decisions) {
		return 2
	}
	return 0
}

// ── govern health — unified health scores ────────────────────────────────────

func governHealth(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	p1, f1, t1, _ := governor.QuickIntegrityMesh()
	integrity := governor.IntegrityMeshHealth(p1, f1, t1)
	o1, w1, c1, _ := governor.QuickSelfDiagnosis()
	sd := governor.SelfDiagnosisHealth(o1, w1, c1)
	pain := governor.PainScorerHealth(
		governor.QuickPainScorer(),
	)
	health := governor.ComputeUnifiedHealth(integrity, sd, pain)

	if jsonOut {
		cli.Output(map[string]interface{}{
			"command": "govern health",
			"health":  health,
		}, true)
		return 0
	}

	fmt.Printf("Governor Health: %.1f%% [%s]\n", health.CompositeScore, health.Overall)
	for name, sub := range health.Subsystems {
		fmt.Printf("  %s: %.1f%% [%s]\n", name, sub.Score, sub.Status)
	}
	return 0
}

// ── govern decide — run decision engine ──────────────────────────────────────

func governDecide(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	state := governor.GatherSystemState()
	decisions := governor.Decide(state)

	if jsonOut {
		cli.Output(map[string]interface{}{
			"command":   "govern decide",
			"state":     state,
			"decisions": decisions,
			"critical":  governor.HasCriticalDecisions(decisions),
		}, true)
		return 0
	}

	if len(decisions) == 0 {
		fmt.Println("✓ No decisions required — system is healthy.")
		return 0
	}

	fmt.Printf("%d decisions emitted:\n", len(decisions))
	byPriority := governor.CountByPriority(decisions)
	for _, p := range []governor.Priority{governor.PriorityCritical, governor.PriorityHigh, governor.PriorityMedium, governor.PriorityLow} {
		if count, ok := byPriority[p]; ok {
			fmt.Printf("  %s: %d\n", p, count)
		}
	}
	fmt.Println()
	for _, d := range decisions {
		fmt.Printf("  [%s] %s → %s\n", d.Priority, d.Domain, d.Action)
		fmt.Printf("       %s\n", d.Reason)
	}

	if governor.HasCriticalDecisions(decisions) {
		return 2
	}
	return 0
}

// ── govern trust — verify trust gate on claims ───────────────────────────────

func governTrust(args []string) int {
	jsonOut := cli.HasJSONFlag(args)

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: ovav govern trust <lead> <claim1,claim2,...>\n")
		fmt.Fprintf(os.Stderr, "Example: ovav govern trust thavren \"34/34 tests OK,0 data races\"\n")
		return 1
	}

	leadName := args[0]
	claims := []string{}
	if len(args) > 1 {
		claims = strings.Split(args[1], ",")
		for i := range claims {
			claims[i] = strings.TrimSpace(claims[i])
		}
	}

	result := governor.VerifyTrust(governor.TrustInput{
		LeadName:     leadName,
		OutputClaims: claims,
	})

	if jsonOut {
		cli.Output(map[string]interface{}{
			"command": "govern trust",
			"input": map[string]interface{}{
				"lead":   leadName,
				"claims": claims,
			},
			"result": result,
		}, true)
		return 0
	}

	fmt.Printf("Trust Gate — %s\n", leadName)
	fmt.Printf("  Action:  %s\n", result.Action)
	fmt.Printf("  Score:   %.1f%%\n", result.TrustScore)
	fmt.Printf("  Summary: %s\n", result.Summary)

	switch result.Action {
	case governor.TrustDeliver:
		return 0
	case governor.TrustDisclaimer:
		return 0
	case governor.TrustBlock:
		return 1
	case governor.TrustReject:
		return 2
	}
	return 0
}

// ── Helper ───────────────────────────────────────────────────────────────────

func healthBanner(score float64) string {
	switch {
	case score >= 90:
		return "🟢 HEALTHY"
	case score >= 70:
		return "🟡 DEGRADED"
	case score >= 50:
		return "🟠 WARNING"
	default:
		return "🔴 CRITICAL"
	}
}
