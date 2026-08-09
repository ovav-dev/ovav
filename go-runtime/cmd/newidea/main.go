// Command: ovav newidea <task> — Test the NEW IDEA detector
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ovav/ovav/internal/runtime"
)

func main() {
	jsonOut := flag.Bool("json", false, "JSON output")
	multiArea := flag.Bool("multi-area", false, "Run multi-area detection")
	effort := flag.Bool("effort", false, "Run effort calibration")
	flag.Parse()
	task := flag.Arg(0)
	if task == "" {
		task = "quiero hacer un proyecto nuevo de app de citas"
	}

	// NEW IDEA detection
	result := runtime.EvaluateNewIdea(task)
	if *jsonOut {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
	} else {
		if result.IsNewIdea {
			fmt.Printf("🚨 NEW IDEA DETECTED\n")
			fmt.Printf("   Category: %s\n", result.Category)
			fmt.Printf("   Severity: %s\n", result.Severity)
			fmt.Printf("   Action: %s\n", result.Recommendation)
			fmt.Printf("   Patterns matched (%d):\n", len(result.MatchedPatterns))
			for _, p := range result.MatchedPatterns {
				parts := strings.SplitN(p, ":", 2)
				if len(parts) == 2 {
					fmt.Printf("     - [%s] %s\n", parts[0], parts[1])
				}
			}
		} else {
			fmt.Printf("✅ CONTINUATION / EXISTING TASK\n")
			fmt.Printf("   No new idea patterns detected\n")
		}
	}

	// Multi-area detection
	if *multiArea {
		ma := runtime.DetectMultiArea(task)
		if *jsonOut {
			b, _ := json.MarshalIndent(ma, "", "  ")
			fmt.Println(string(b))
		} else {
			if ma.IsMultiArea {
				fmt.Printf("\n🌐 MULTI-AREA DETECTED\n")
				fmt.Printf("   Areas: %v\n", ma.AreasFound)
				fmt.Printf("   Leads: %v\n", ma.LeadsFound)
				fmt.Printf("   Primary: %s\n", ma.PrimaryArea)
			} else {
				fmt.Printf("\n🎯 SINGLE AREA: %s (%s)\n", ma.PrimaryArea, ma.LeadsFound[0])
			}
		}
	}

	// Effort calibration
	if *effort {
		ec := runtime.CalibrateEffort(task)
		if *jsonOut {
			b, _ := json.MarshalIndent(ec, "", "  ")
			fmt.Println(string(b))
		} else {
			fmt.Printf("\n📊 EFFORT: %s\n", ec.Effort)
			for _, s := range ec.Signals {
				fmt.Printf("   - %s\n", s)
			}
		}
	}

	// Exit code
	if result.IsNewIdea && result.Severity == "hard" {
		os.Exit(2) // Hard new idea — require plan
	}
	if result.IsNewIdea {
		os.Exit(1) // Soft new idea — recommend plan
	}
}
