package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ovav/ovav/internal/monitor"
	"github.com/ovav/ovav/internal/monitor/alerts"
)

// cmdMonitor handles `ovav monitor` — OMARS control interface
func cmdMonitor(args []string) int {
	if len(args) == 0 {
		return printMonitorHelp()
	}

	sub := args[0]
	ctx := context.Background()

	switch sub {
	case "run":
		return runMonitors(ctx, args[1:])
	case "status":
		return monitorStatus(ctx)
	case "start":
		return startMonitorLoop(ctx, args[1:])
	case "history":
		return monitorHistory(ctx)
	case "flush":
		return flushAlerts(ctx)
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", sub)
		return 1
	}
}

func printMonitorHelp() int {
	fmt.Print(`ovav monitor — OMARS (OVAV Monitoring & Auto-Remediation System)

Usage:
  ovav monitor run        Run all monitors once
  ovav monitor run <name> Run specific monitor (hygiene, agent_projection)
  ovav monitor status     Show pending alerts
  ovav monitor history     Show resolved alerts
  ovav monitor start      Start background monitor loop
  ovav monitor flush      Clear all pending alerts (requires CEO waiver)

Examples:
  ovav monitor run hygiene
  ovav monitor status
  ovav monitor run
`)
	return 0
}

func runMonitors(ctx context.Context, args []string) int {
	root, err := findOvavRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root\n")
		return 1
	}

	queue := alerts.NewQueue(root)
	dispatcher := alerts.NewDispatcher(queue)
	registry := monitor.NewRegistry(dispatcher)

	// Register monitors
	registry.Register(monitor.NewHygieneMonitor(root))
	registry.Register(monitor.NewAgentProjectionMonitor(root))

	// Register runbooks
	dispatcher.RegisterRunbook("hygiene", monitor.RunbookFixStaleLocks)
	dispatcher.RegisterRunbook("hygiene", monitor.RunbookFixGeneratedDrift)
	dispatcher.RegisterRunbook("agent_projection", monitor.RunbookFixAgentProjection)
	dispatcher.RegisterRunbook("agent_projection", monitor.RunbookFixSBOMBaseline)
	dispatcher.RegisterRunbook("agent_projection", monitor.RunbookFixRuntimeIntegrity)

	// Filter by specific monitor if requested
	if len(args) > 0 {
		monitors := []monitor.Monitor{}
		for _, m := range registry.GetMonitors() {
			if m.Name() == args[0] {
				monitors = append(monitors, m)
			}
		}
		if len(monitors) == 0 {
			fmt.Fprintf(os.Stderr, "❌ Unknown monitor: %s\n", args[0])
			return 1
		}
		// Run only specified monitor
		for _, m := range monitors {
			a, _ := m.Run(ctx)
			for _, alert := range a {
				dispatcher.Dispatch(ctx, alert)
				fmt.Printf("📢 %s | %s | %s\n", alert.Level, alert.Source, alert.Issue)
			}
		}
		return 0
	}

	// Run all monitors
	fmt.Println("🔍 Running OVAV Monitors...")
	fmt.Println()

	alertList, err := registry.RunAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Monitor run failed: %v\n", err)
		return 1
	}

	if len(alertList) == 0 {
		fmt.Println("✅ All monitors passed — no alerts")
		return 0
	}

	fmt.Printf("⚠️  %d alert(s) generated:\n\n", len(alertList))
	for _, a := range alertList {
		runbook := ""
		if a.Runbook != "" {
			runbook = fmt.Sprintf(" [auto-fix: %s]", a.Runbook)
		}
		fmt.Printf("  %s | %s | %s%s\n", a.Level, a.Source, a.Issue, runbook)
	}

	return 0
}

func monitorStatus(ctx context.Context) int {
	root, err := findOvavRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root\n")
		return 1
	}

	queue := alerts.NewQueue(root)
	pending, err := queue.GetPending()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read queue: %v\n", err)
		return 1
	}

	autoFixed, _ := queue.GetAutoFixed()

	fmt.Printf("📊 OMARS Status — %s\n", time.Now().Format(time.RFC3339))
	fmt.Println()
	fmt.Printf("Pending alerts:  %d\n", len(pending))
	fmt.Printf("Auto-fixed:     %d\n", len(autoFixed))
	fmt.Println()

	if len(pending) == 0 {
		fmt.Println("✅ No pending alerts")
		return 0
	}

	fmt.Println("Pending alerts:")
	for _, a := range pending {
		runbook := ""
		if a.Runbook != "" {
			runbook = fmt.Sprintf(" → %s", a.Runbook)
		}
		fmt.Printf("  %s | %-6s | %-20s | %s%s\n", a.ID, a.Level, a.Source, a.Issue, runbook)
	}

	return 0
}

func monitorHistory(ctx context.Context) int {
	root, err := findOvavRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root\n")
		return 1
	}

	queue := alerts.NewQueue(root)

	autoFixed, _ := queue.GetAutoFixed()
	acknowledged, _ := queue.GetAcknowledged()

	fmt.Printf("📜 OMARS History — %s\n", time.Now().Format(time.RFC3339))
	fmt.Println()

	fmt.Printf("Auto-fixed: %d\n", len(autoFixed))
	for _, a := range autoFixed {
		fmt.Printf("  ✅ %s | %s | %s → %s\n", a.ID, a.Source, a.Issue, a.Resolution)
	}

	fmt.Printf("\nHuman-acknowledged: %d\n", len(acknowledged))
	for _, a := range acknowledged {
		fmt.Printf("  👤 %s | %s | %s → %s\n", a.ID, a.Source, a.Issue, a.Resolution)
	}

	return 0
}

func startMonitorLoop(ctx context.Context, args []string) int {
	root, err := findOvavRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root\n")
		return 1
	}

	queue := alerts.NewQueue(root)
	dispatcher := alerts.NewDispatcher(queue)
	registry := monitor.NewRegistry(dispatcher)

	registry.Register(monitor.NewHygieneMonitor(root))
	registry.Register(monitor.NewAgentProjectionMonitor(root))

	dispatcher.RegisterRunbook("hygiene", monitor.RunbookFixStaleLocks)
	dispatcher.RegisterRunbook("hygiene", monitor.RunbookFixGeneratedDrift)
	dispatcher.RegisterRunbook("agent_projection", monitor.RunbookFixAgentProjection)

	fmt.Println("🚀 Starting OMARS monitor loop...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Run once immediately
	registry.RunAll(ctx)

	// Then run in background
	go registry.RunLoop(ctx)

	// Block until context cancelled
	<-ctx.Done()
	return 0
}

func flushAlerts(ctx context.Context) int {
	root, err := findOvavRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root\n")
		return 1
	}

	queue := alerts.NewQueue(root)
	pending, _ := queue.GetPending()

	for _, a := range pending {
		if err := queue.MarkAcknowledged(a.ID, "thavren", "flushed"); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to archive %s: %v\n", a.ID, err)
		}
	}

	fmt.Printf("✅ Flushed %d alerts\n", len(pending))
	return 0
}
