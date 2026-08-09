// Command: ovav delegation <task> — OVAV delegation runtime with PL-0 gates
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/runtime"
)

func main() {
	trigger := flag.Bool("trigger", false, "Run h_delegation_trigger")
	guard := flag.Bool("guard", false, "Run h_do_not_delegate_guard")
	route := flag.Bool("route", false, "Run delegation router")
	payload := flag.Bool("payload", false, "Build full delegation payload")
	newidea := flag.Bool("newidea", false, "Run NEW IDEA gate")
	multiArea := flag.Bool("multi-area", false, "Run multi-area detection")
	effort := flag.Bool("effort", false, "Run effort calibration")
	files := flag.Int("files", 0, "Number of files to touch")
	arch := flag.Bool("arch", false, "Architecture change")
	auth := flag.Bool("auth", false, "Auth/security change")
	sources := flag.Int("sources", 0, "Research sources")
	agent := flag.String("agent", "", "Specific agent ID")
	jsonOut := flag.Bool("json", false, "JSON output")
	auto := flag.Bool("auto", false, "Run all PL-0 gates: newidea + guard + route + payload")

	flag.Parse()
	task := flag.Arg(0)
	if task == "" {
		task = "implementar nueva feature en el sistema"
	}

	// Default: run full route + payload if no specific gate
	runAll := !(*trigger || *guard || *route || *payload || *newidea || *multiArea || *effort) || *auto

	// PL-0 [1] NEW IDEA GATE — runs first, always
	if *newidea || runAll {
		result := runtime.EvaluateNewIdea(task)
		if *jsonOut {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
		} else {
			if result.IsNewIdea {
				icon := "⚠️"
				if result.Severity == "hard" {
					icon = "🚨"
				}
				fmt.Printf("%s NEW IDEA: %s (%s)\n", icon, result.Category, result.Severity)
				fmt.Printf("   Action: %s\n", result.Recommendation)
				if len(result.MatchedPatterns) > 0 {
					fmt.Printf("   Patterns (%d):\n", len(result.MatchedPatterns))
					for _, p := range result.MatchedPatterns {
						parts := splitDotN(p, ":", 2)
						if len(parts) == 2 {
							fmt.Printf("     - [%s] %s\n", parts[0], parts[1])
						}
					}
				}
			} else {
				fmt.Printf("✅ CONTINUATION / EXISTING TASK\n")
				fmt.Printf("   No new idea patterns detected\n")
			}
		}
		// Exit codes for new idea gate
		if result.IsNewIdea && result.Severity == "hard" {
			os.Exit(2) // Hard new idea — require plan
		}
		if result.IsNewIdea {
			os.Exit(1) // Soft new idea — recommend plan
		}
	}

	// PL-0 [3] MULTI-AREA DETECTION
	if *multiArea || runAll {
		ma := runtime.DetectMultiArea(task)
		if *jsonOut {
			b, _ := json.MarshalIndent(ma, "", "  ")
			fmt.Println(string(b))
		} else {
			if ma.IsMultiArea {
				fmt.Printf("\n🌐 MULTI-AREA DETECTED\n")
				fmt.Printf("   Areas: %v\n", ma.AreasFound)
				fmt.Printf("   Primary: %s\n", ma.PrimaryArea)
			} else {
				fmt.Printf("\n🎯 SINGLE AREA: %s (%s)\n", ma.PrimaryArea, ma.LeadsFound[0])
			}
		}
	}

	// PL-0 [4] EFFORT CALIBRATION
	if *effort || runAll {
		ec := runtime.CalibrateEffort(task)
		if *jsonOut {
			b, _ := json.MarshalIndent(ec, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Printf("\n📊 EFFORT: %s\n", ec.Effort)
			for _, s := range ec.Signals {
				parts := splitDotN(s, ":", 2)
				if len(parts) == 2 {
					fmt.Printf("   - [%s] %s\n", parts[0], parts[1])
				}
			}
		}
	}

	// PL-0 [5] DELEGATION GATES
	if *trigger || runAll {
		result := runtime.EvaluateTaskComplexity(task, *files, *arch, *auth, false, *sources)
		if *jsonOut {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
		} else {
			verdict := "DELEGATE"
			if !result.ShouldDelegate {
				verdict = "LEAD_ONLY"
			}
			fmt.Printf("\n[h_delegation_trigger] score=%d threshold=%d → %s\n",
				result.Score, result.Threshold, verdict)
			for _, r := range result.Reasons {
				fmt.Printf("  - %s\n", r)
			}
		}
	}

	if *guard || runAll {
		result := runtime.EvaluateDoNotDelegate(task, *files)
		if *jsonOut {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
		} else {
			if result.Blocked {
				fmt.Printf("[h_do_not_delegate_guard] BLOCKED — %s\n", result.Reason)
			} else {
				fmt.Printf("[h_do_not_delegate_guard] ALLOWED — delegacion procede\n")
			}
		}
	}

	if *route || runAll {
		result := runtime.RouteDelegation(task, *agent)
		if *jsonOut {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Println("\n[delegation_router]")
			fmt.Printf("  Area: %s\n", result.ServiceArea)
			fmt.Printf("  Lead: %s\n", result.LeadResolved)
			fmt.Printf("  Agent: %s (%s)\n", result.AgentID, result.AgentKind)
			fmt.Printf("  Task: %.60s...\n", result.Task)
		}
	}

	if *payload || runAll {
		// Always resolve agent if empty — use routing to determine correct lead
		resolvedAgent := *agent
		if resolvedAgent == "" {
			routeResult := runtime.RouteDelegation(task, *agent)
			resolvedAgent = routeResult.AgentID
		}
		p, err := runtime.BuildDelegationPayload(resolvedAgent, task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if *jsonOut {
			b, _ := json.MarshalIndent(p, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Println("\n[delegation_payload]")
			fmt.Printf("  Agent: %s (%s)\n", p.AgentID, p.AgentKind)
			fmt.Printf("  Area: %s\n", p.AgentArea)
			fmt.Printf("  Git: %s @ %s\n", p.Workspace.Branch, p.Workspace.Head)
			fmt.Printf("  Status: %s\n", p.Workspace.Status)
			fmt.Printf("  Profile: %d chars loaded\n", len(p.Profile.SystemPrompt))
			fmt.Printf("  Generated: %s\n", p.Generated)
		}
	}
}

func splitDotN(s, sep string, n int) []string {
	parts := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := -1
		for j := len(parts); j < len(s); j++ {
			if s[j] == sep[0] {
				idx = j
				break
			}
		}
		if idx < 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}
