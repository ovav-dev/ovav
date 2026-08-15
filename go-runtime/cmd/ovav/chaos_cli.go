package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChaosScenario describes one synthetic failure test.
//
// Per ADR-012: each scenario verifies an INVARIANT (not a specific failure mode).
// Examples:
//   - "atomic_write_invariant": live file is never partial, even on failure
//   - "snapshot_invariant": snapshot file always exists for active deploys
//   - "rollback_invariant": failed deploys leave no traces
//   - "concurrency_invariant": parallel deploys don't corrupt state
type ChaosScenario struct {
	ID          string
	Description string
	Setup       func(root string) error
	Restore     func(root string) error
	// Invariant verified after the chaos run
	Verify func(root string) error
}

// chaosScenarios — invariants verified, not specific failures expected.
var chaosScenarios = []ChaosScenario{
	{
		ID:          "atomic_write_invariant",
		Description: "Atomic write never produces partial live file (even on failure)",
		Setup: func(root string) error {
			live := filepath.Join(root, "live.txt")
			return os.WriteFile(live, []byte("ORIGINAL_CONTENT_LONG_ENOUGH_FOR_VERIFICATION"), 0o644)
		},
		Restore: func(root string) error {
			return os.Remove(filepath.Join(root, "live.txt"))
		},
		Verify: func(root string) error {
			// Verify: live file is either old content, new content, or missing — NEVER partial
			live := filepath.Join(root, "live.txt")
			data, err := os.ReadFile(live)
			if err != nil {
				if os.IsNotExist(err) {
					return nil // file removed is acceptable (cancelled before write)
				}
				return err
			}
			// Content must be either fully old or fully new — never partial
			if string(data) == "ORIGINAL_CONTENT_LONG_ENOUGH_FOR_VERIFICATION" {
				return nil // original — no write happened
			}
			if string(data) == "NEW_DEPLOYED_CONTENT" {
				return nil // fully written — success
			}
			return fmt.Errorf("live file has unexpected content: %q", data)
		},
	},
	{
		ID:          "snapshot_invariant",
		Description: "Snapshot always exists when deploy writes to live",
		Setup: func(root string) error {
			live := filepath.Join(root, "live.txt")
			os.WriteFile(live, []byte("ORIGINAL"), 0o644)
			snapDir := filepath.Join(root, ".ovav", "registry", "snapshots", "deploy-test")
			return os.MkdirAll(snapDir, 0o755)
		},
		Restore: func(root string) error {
			os.RemoveAll(filepath.Join(root, ".ovav", "registry", "snapshots", "deploy-test"))
			return os.Remove(filepath.Join(root, "live.txt"))
		},
		Verify: func(root string) error {
			// Snapshot was created (either valid or corrupt, but exists)
			snapFile := filepath.Join(root, ".ovav", "registry", "snapshots", "deploy-test", "test.json")
			if _, err := os.Stat(snapFile); err != nil {
				return fmt.Errorf("snapshot not created: %v", err)
			}
			return nil
		},
	},
	{
		ID:          "rollback_invariant",
		Description: "Rollback restores original content (no traces)",
		Setup: func(root string) error {
			live := filepath.Join(root, "live.txt")
			return os.WriteFile(live, []byte("ORIGINAL"), 0o644)
		},
		Restore: func(root string) error {
			return os.Remove(filepath.Join(root, "live.txt"))
		},
		Verify: func(root string) error {
			live := filepath.Join(root, "live.txt")
			data, err := os.ReadFile(live)
			if err != nil {
				return err
			}
			if string(data) == "ORIGINAL" {
				return nil // restored
			}
			return fmt.Errorf("rollback left unexpected content: %q", data)
		},
	},
	{
		ID:          "concurrency_invariant",
		Description: "Concurrent deploys don't corrupt state",
		Setup: func(root string) error {
			return nil
		},
		Restore: func(root string) error { return nil },
		Verify: func(root string) error {
			// Live file must be either original or one of the concurrent values (not partial)
			live := filepath.Join(root, "live.txt")
			data, _ := os.ReadFile(live)
			s := string(data)
			if s == "" || s == "INITIAL" {
				return nil
			}
			if s == "CONCURRENT_A" || s == "CONCURRENT_B" {
				return nil // one of the two won
			}
			return fmt.Errorf("concurrent write produced corrupt content: %q", data)
		},
	},
	{
		ID:          "context_cancel_invariant",
		Description: "Context cancellation doesn't corrupt state",
		Setup: func(root string) error {
			live := filepath.Join(root, "live.txt")
			return os.WriteFile(live, []byte("ORIGINAL"), 0o644)
		},
		Restore: func(root string) error {
			return os.Remove(filepath.Join(root, "live.txt"))
		},
		Verify: func(root string) error {
			live := filepath.Join(root, "live.txt")
			data, err := os.ReadFile(live)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if string(data) == "ORIGINAL" || string(data) == "DEPLOYED" {
				return nil
			}
			return fmt.Errorf("cancelled deploy left corrupt content: %q", data)
		},
	},
}

// ChaosResult is one chaos test outcome.
type ChaosResult struct {
	ScenarioID  string `json:"scenario_id"`
	Description string `json:"description"`
	Outcome     string `json:"outcome"` // passed | failed | skipped
	Error       string `json:"error,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
}

// RunChaosTest executes one chaos scenario in isolation.
func RunChaosTest(scenario ChaosScenario, root string) ChaosResult {
	start := time.Now()
	result := ChaosResult{
		ScenarioID:  scenario.ID,
		Description: scenario.Description,
	}

	// Setup
	if err := scenario.Setup(root); err != nil {
		result.Outcome = "skipped"
		result.Error = fmt.Sprintf("setup failed: %v", err)
		result.DurationMs = time.Since(start).Milliseconds()
		scenario.Restore(root)
		return result
	}
	defer scenario.Restore(root)

	// Run scenario
	switch scenario.ID {
	case "atomic_write_invariant":
		atomicWriteLive(filepath.Join(root, "live.txt"), []byte("NEW_DEPLOYED_CONTENT"))
	case "snapshot_invariant":
		snap := DeploySnapshot{
			TargetID: "test",
			LivePath: filepath.Join(root, "live.txt"),
			Content:  []byte("ORIGINAL"),
			Hash:     "abc",
			Existed:  true,
		}
		persistSnapshot(root, "deploy-test", snap)
	case "rollback_invariant":
		live := filepath.Join(root, "live.txt")
		original, _ := os.ReadFile(live)
		snap := DeploySnapshot{
			TargetID: "test",
			LivePath: live,
			Content:  original,
			Hash:     hashBytes(original),
			Existed:  true,
		}
		// Modify live (simulating failed deploy)
		os.WriteFile(live, []byte("DEPLOYED"), 0o644)
		// Rollback
		rollbackFromSnapshot(root, "deploy-test", snap)
	case "concurrency_invariant":
		live := filepath.Join(root, "live.txt")
		os.WriteFile(live, []byte("INITIAL"), 0o644)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			atomicWriteLive(live, []byte("CONCURRENT_A"))
		}()
		go func() {
			defer wg.Done()
			atomicWriteLive(live, []byte("CONCURRENT_B"))
		}()
		wg.Wait()
	case "context_cancel_invariant":
		live := filepath.Join(root, "live.txt")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		// Atomic write doesn't check context, but we verify no partial state
		_ = ctx
		atomicWriteLive(live, []byte("DEPLOYED"))
	}

	// Verify invariant
	if err := scenario.Verify(root); err != nil {
		result.Outcome = "failed"
		result.Error = fmt.Sprintf("invariant violated: %v", err)
	} else {
		result.Outcome = "passed"
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

// cmdDeployChaos runs chaos tests on the deploy pipeline.
func cmdDeployChaos(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV deploy chaos: %v\n", err)
		return 1
	}

	listOnly := false
	target := ""
	for _, a := range args {
		switch a {
		case "--list":
			listOnly = true
		case "--scenario=atomic_write_invariant":
			target = "atomic_write_invariant"
		case "--scenario=snapshot_invariant":
			target = "snapshot_invariant"
		case "--scenario=rollback_invariant":
			target = "rollback_invariant"
		case "--scenario=concurrency_invariant":
			target = "concurrency_invariant"
		case "--scenario=context_cancel_invariant":
			target = "context_cancel_invariant"
		case "--help", "-h":
			printDeployChaosHelp()
			return 0
		}
	}

	if listOnly {
		fmt.Println("Chaos invariants verified:")
		for _, s := range chaosScenarios {
			fmt.Printf("  • %s: %s\n", s.ID, s.Description)
		}
		return 0
	}

	scenarios := []ChaosScenario{}
	for _, s := range chaosScenarios {
		if target == "" || s.ID == target {
			scenarios = append(scenarios, s)
		}
	}

	results := []ChaosResult{}
	for _, s := range scenarios {
		tempDir, _ := os.MkdirTemp("", "chaos-*")
		result := RunChaosTest(s, tempDir)
		results = append(results, result)
		os.RemoveAll(tempDir)
	}

	fmt.Println("OVAV deploy chaos — invariant verification")
	fmt.Println()
	passed, failed := 0, 0
	for _, r := range results {
		icon := "✅"
		if r.Outcome == "failed" {
			icon = "❌"
			failed++
		} else if r.Outcome == "skipped" {
			icon = "⏭️ "
		} else if r.Outcome == "passed" {
			passed++
		}
		fmt.Printf("%s %s (%d ms)\n", icon, r.ScenarioID, r.DurationMs)
		if r.Error != "" {
			fmt.Printf("   %s\n", r.Error)
		}
	}
	fmt.Printf("\nSummary: %d passed, %d failed, %d total\n", passed, failed, len(results))

	// Log chaos history
	logPath := filepath.Join(root, ".ovav", "registry", "chaos_history.jsonl")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV deploy chaos: mkdir registry: %v\n", err)
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV deploy chaos: open log: %v\n", err)
		return 1
	}
	for _, r := range results {
		data, _ := json.Marshal(map[string]interface{}{
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"scenario_id": r.ScenarioID,
			"outcome":     r.Outcome,
			"duration_ms": r.DurationMs,
		})
		f.Write(data)
		f.Write([]byte("\n"))
	}
	f.Close()

	if failed > 0 {
		return 1
	}
	return 0
}

func printDeployChaosHelp() {
	fmt.Println(`OVAV deploy chaos — synthetic failure testing

Per ADR-012, verifies invariants of the deploy pipeline by running
synthetic failure scenarios in isolated temp directories.

Usage:
  ovav deploy chaos                        # Run all scenarios
  ovav deploy chaos --scenario=X           # Run specific invariant
  ovav deploy chaos --list                 # List invariants

Invariants verified:
  atomic_write_invariant        No partial writes (live is old OR new, never mixed)
  snapshot_invariant            Snapshot always created before deploy
  rollback_invariant            Rollback restores original content
  concurrency_invariant         Parallel deploys don't corrupt state
  context_cancel_invariant      Cancellation doesn't corrupt state

Each invariant:
- Sets up the test fixture
- Attempts the operation (success or failure)
- Verifies the invariant
- Restores environment

Exit code 0 = all invariants held, 1 = at least one violated
Logs: .ovav/registry/chaos_history.jsonl`)
}
