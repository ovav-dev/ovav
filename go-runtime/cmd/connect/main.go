// Package main implements the ovav connect command.
package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ovav/ovav/internal/connect/tracker"
)

func main() {
	// Get root directory
	root := os.Getenv("OVAV_ROOT")
	if root == "" {
		root = "."
	}

	tk := tracker.New(root + "/.ovav/connect")
	if err := tk.LoadProviders(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading providers: %v\n", err)
		os.Exit(1)
	}

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "status":
		status(tk)
	case "providers":
		providers(tk)
	case "add":
		add(tk, os.Args[2:])
	case "remove":
		remove(tk, os.Args[2:])
	case "history":
		history(tk, os.Args[2:])
	case "report":
		report(tk)
	case "optimize":
		optimize(tk)
	default:
		printUsage()
		os.Exit(1)
	}
}

func status(tk *tracker.Tracker) {
	providers := tk.ListProviders()

	fmt.Println("🔗 OVAV CONNECT Status")
	fmt.Println()

	if len(providers) == 0 {
		fmt.Println("No providers configured. Run 'ovav connect add' to add one.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Provider\tType\tEnabled\tLast Check\n")
	fmt.Fprintf(w, "--------\t----\t-------\t----------\n")
	for _, p := range providers {
		lastCheck := "Never"
		if !p.LastCheck.IsZero() {
			lastCheck = p.LastCheck.Format("2006-01-02 15:04")
		}
		enabled := "✓"
		if !p.Enabled {
			enabled = "✗"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Type, enabled, lastCheck)
	}
	w.Flush()

	// Show usage summary
	fmt.Println("\n📊 Today's Usage:")
	for _, p := range providers {
		usage, err := tk.GetTodayUsage(p.ID)
		if err != nil || usage.TotalTokens == 0 {
			continue
		}
		fmt.Printf("  %s: %d tokens ($%.4f)\n", p.ID, usage.TotalTokens, usage.TotalCostUSD)
	}
}

func providers(tk *tracker.Tracker) {
	providers := tk.ListProviders()

	fmt.Println("🎯 Configured Providers")
	fmt.Println()

	if len(providers) == 0 {
		fmt.Println("No providers configured.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tType\tAPI Key\tEnabled\n")
	fmt.Fprintf(w, "--\t----\t-------\t-------\n")
	for _, p := range providers {
		apiKey := p.APIKey
		if len(apiKey) > 8 {
			apiKey = "..." + apiKey[len(apiKey)-8:]
		}
		enabled := "✓"
		if !p.Enabled {
			enabled = "✗"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Type, apiKey, enabled)
	}
	w.Flush()
}

func add(tk *tracker.Tracker, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ovav connect add <type> <api_key>")
		fmt.Fprintln(os.Stderr, "  types: openai, anthropic, openrouter")
		os.Exit(1)
	}

	providerType := args[0]
	apiKey := args[1]

	provider := &tracker.TrackedProvider{
		ID:      providerType + "-" + time.Now().Format("20060102"),
		Type:    providerType,
		APIKey:  apiKey,
		Enabled: true,
	}

	if err := tk.AddProvider(provider); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Added provider: %s (%s)\n", provider.ID, provider.Type)
}

func remove(tk *tracker.Tracker, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ovav connect remove <provider_id>")
		os.Exit(1)
	}

	providerID := args[0]
	if err := tk.RemoveProvider(providerID); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Removed provider: %s\n", providerID)
}

func history(tk *tracker.Tracker, args []string) {
	providerID := ""
	days := 7

	for i := 0; i < len(args); i++ {
		if args[i] == "--provider" && i+1 < len(args) {
			providerID = args[i+1]
			i++
		} else if args[i] == "--days" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &days)
			i++
		}
	}

	providers := tk.ListProviders()
	if providerID == "" && len(providers) > 0 {
		providerID = providers[0].ID
	}

	if providerID == "" {
		fmt.Println("No providers configured.")
		return
	}

	since := time.Now().AddDate(0, 0, -days)
	records, err := tk.GetUsageHistory(providerID, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching history: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📜 Usage History for %s (last %d days)\n\n", providerID, days)

	if len(records) == 0 {
		fmt.Println("No usage records found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Time\tModel\tInput\tOutput\tTotal\tCost\n")
	fmt.Fprintf(w, "----\t-----\t-----\t------\t-----\t----\n")
	for _, r := range records {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t$%.6f\n",
			r.Timestamp.Format("2006-01-02 15:04"),
			r.Model,
			r.InputTokens,
			r.OutputTokens,
			r.TotalTokens,
			r.CostUSD,
		)
	}
	w.Flush()
}

func report(tk *tracker.Tracker) {
	providers := tk.ListProviders()

	fmt.Println("📊 Usage Report")
	fmt.Println()

	if len(providers) == 0 {
		fmt.Println("No providers configured.")
		return
	}

	for _, p := range providers {
		monthUsage, _ := tk.GetMonthUsage(p.ID)

		fmt.Printf("## %s\n\n", p.ID)
		if monthUsage == nil || monthUsage.TotalTokens == 0 {
			fmt.Println("  No usage this month.")
			continue
		}
		fmt.Printf("  Total Calls:  %d\n", monthUsage.TotalCalls)
		fmt.Printf("  Input Tokens: %d\n", monthUsage.TotalInput)
		fmt.Printf("  Output Tokens:%d\n", monthUsage.TotalOutput)
		fmt.Printf("  Total Tokens: %d\n", monthUsage.TotalTokens)
		fmt.Printf("  Total Cost:   $%.2f\n", monthUsage.TotalCostUSD)
		fmt.Println()
	}
}

func optimize(tk *tracker.Tracker) {
	fmt.Println("🧠 Running AI-Powered Optimization Analysis...")
	fmt.Println()

	optimizer := tracker.NewAutoOptimizer(tk)
	
	recommendations, err := optimizer.GenerateRecommendations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating recommendations: %v\n", err)
		os.Exit(1)
	}

	if len(recommendations) == 0 {
		fmt.Println("✅ No optimization opportunities found. Your setup looks good!")
		return
	}

	fmt.Printf("💡 Found %d optimization opportunities:\n\n", len(recommendations))
	
	for i, rec := range recommendations {
		priorityIcon := "🟡"
		if rec.Priority > 0.8 {
			priorityIcon = "🔴"
		} else if rec.Priority > 0.5 {
			priorityIcon = "🟠"
		}
		
		fmt.Printf("%d. %s [%s] %.0f%% priority\n", i+1, priorityIcon, rec.Type, rec.Priority*100)
		fmt.Printf("   Provider: %s\n", rec.ProviderID)
		if rec.Model != "" {
			fmt.Printf("   Model: %s\n", rec.Model)
		}
		fmt.Printf("   Potential Savings: $%.2f\n", rec.Savings)
		fmt.Printf("   Reason: %s\n", rec.Reason)
		fmt.Printf("   Action: %s\n", rec.Action)
		fmt.Println()
	}

	// Show provider analysis
	providers := tk.ListProviders()
	fmt.Println("📈 Provider Performance Analysis:")
	fmt.Println()
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Provider\tEfficiency\tTrend\tAvg Cost/1K\tBest Model\n")
	fmt.Fprintf(w, "--------\t----------\t-----\t-----------\t----------\n")
	
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		
		analysis, err := optimizer.AnalyzeProvider(p.ID)
		if err != nil {
			continue
		}
		
		trendIcon := "➡️"
		if analysis.TrendDirection == "increasing" {
			trendIcon = "📈"
		} else if analysis.TrendDirection == "decreasing" {
			trendIcon = "📉"
		}
		
		fmt.Fprintf(w, "%s\t%.0f%%\t%s\t$%.4f\t%s\n", 
			p.ID, 
			analysis.EfficiencyScore*100, 
			trendIcon,
			analysis.AverageCostPerK,
			analysis.BestModel)
	}
	w.Flush()
}

func printUsage() {
	fmt.Print(`OVAV Connect - Token Usage Tracking

Usage:
  ovav connect <command>

Commands:
  status      Show connection status and today's usage
  providers   List all configured providers
  add         Add a new provider
  remove      Remove a provider
  history     Show usage history
  report      Generate usage report
  optimize    AI-powered optimization recommendations

Examples:
  ovav connect status              # Check status
  ovav connect add openai sk-...   # Add OpenAI
  ovav connect add anthropic sk-.. # Add Anthropic
  ovav connect history --days 30    # Last 30 days
  ovav connect report              # Monthly report
  ovav connect optimize            # Get AI optimization tips
` + "\n")
}
