// Package shell provides OVAV-instrumented shell hooks for interactive sessions.
//
// Top-tier 2026 innovation: every shell command goes through an
// "intelligent shell" that learns from history, predicts mistakes,
// suggests safer alternatives, and tracks impact on OVAV state.
//
// Use cases:
//   - Pre-execution: predict if command will hit a gate
//   - Post-execution: record what changed, update ledger
//   - Continuously: learn patterns per-user
//
// Hooks fire on shell events: command_start, command_end, command_fail.
// They cannot alter the command itself — only observe + record.
//
// Data flow:
//
//	shell → execution capture → debouncer → async buffer → classifier
//	                          → alert bus (if drift) → memory bridge (if insight)
package shell

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ovav/ovav/internal/ceo"
	"github.com/ovav/ovav/internal/permissions"
)

// EventKind classifies shell events.
type EventKind string

const (
	EventCommandStart EventKind = "command.start"
	EventCommandEnd   EventKind = "command.end"
	EventCommandFail  EventKind = "command.fail"
)

// Event represents a single shell observation.
type Event struct {
	Kind     EventKind
	Command  string
	Args     []string
	StartAt  time.Time
	EndAt    time.Time
	ExitCode int
	Worktree string
	Actor    string
}

// Hook is the callback signature for shell observers.
type Hook func(Event)

// Observer is the OVAV-instrumented shell observer.
type Observer struct {
	hooks   []Hook
	mu      sync.Mutex
	wg      sync.WaitGroup
	running bool
	stopCh  chan struct{}
}

// NewObserver returns an Observer ready to receive events.
func NewObserver() *Observer {
	return &Observer{
		stopCh: make(chan struct{}),
	}
}

// Register adds a hook callback.
func (o *Observer) Register(h Hook) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.hooks = append(o.hooks, h)
}

// Wait blocks until all spawned hook goroutines complete.
func (o *Observer) Wait() {
	o.wg.Wait()
}

// Start begins event capture (no-op in stub — real impl uses readline hook).
func (o *Observer) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("observer already running")
	}
	o.running = true
	o.mu.Unlock()
	return nil
}

// Stop terminates the observer.
func (o *Observer) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.running {
		return
	}
	o.running = false
	close(o.stopCh)
}

// Emit fires an event to all registered hooks.
func (o *Observer) Emit(e Event) {
	o.mu.Lock()
	hooks := make([]Hook, len(o.hooks))
	copy(hooks, o.hooks)
	o.mu.Unlock()
	for _, h := range hooks {
		o.wg.Add(1)
		go func(hook Hook) {
			defer o.wg.Done()
			hook(e)
		}(h)
	}
}

// ── Run one command with capture ──────────────────────────────────────────

// Run executes a command and emits shell events around it.
func (o *Observer) Run(ctx context.Context, name string, args ...string) (int, error) {
	start := time.Now()
	fullArgs := append([]string{name}, args...)
	o.Emit(Event{
		Kind:     EventCommandStart,
		Command:  name,
		Args:     args,
		StartAt:  start,
		Actor:    os.Getenv("USER"),
		Worktree: os.Getenv("OVAV_WORKTREE"),
	})

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	end := time.Now()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		o.Emit(Event{
			Kind:     EventCommandFail,
			Command:  name,
			Args:     args,
			StartAt:  start,
			EndAt:    end,
			ExitCode: exitCode,
			Actor:    os.Getenv("USER"),
			Worktree: os.Getenv("OVAV_WORKTREE"),
		})
		return exitCode, err
	}
	o.Emit(Event{
		Kind:     EventCommandEnd,
		Command:  name,
		Args:     args,
		StartAt:  start,
		EndAt:    end,
		ExitCode: exitCode,
		Actor:    os.Getenv("USER"),
		Worktree: os.Getenv("OVAV_WORKTREE"),
	})
	_ = fullArgs
	return exitCode, nil
}

// ── Suggestion engine ──────────────────────────────────────────────────────

// bashGovernor is the package-level bash command governor, initialized once.
var bashGovernor *permissions.BashCommandGovernor

func getBashGovernor() *permissions.BashCommandGovernor {
	if bashGovernor == nil {
		bashGovernor = permissions.NewBashCommandGovernor()
	}
	repoRoot := os.Getenv("OVAV_REPO_ROOT")
	if repoRoot == "" {
		repoRoot = "."
	}
	bashGovernor.CEOActive = ceo.IsActive(repoRoot)
	return bashGovernor
}

// Suggest returns safer alternatives for risky commands.
// It uses BashCommandGovernor with CEO bypass to determine if a command is allowed.
// When CEO session is active, DENY rules are bypassed and no suggestions are returned.
func Suggest(command string) []string {
	suggestions := []string{}
	gov := getBashGovernor()

	decision := gov.CheckWithCEO(command, "shell")

	// CEO bypass active — DENY rules are overridden, no suggestions needed
	if decision.Allowed && strings.Contains(decision.Reason, "[CEO-BYPASS]") {
		return nil
	}

	// No match or ALLOW — nothing risky
	if decision.Allowed {
		return nil
	}

	// DENY matched — return contextual suggestions based on matched rule
	switch decision.MatchedRule {
	case "git_push_force":
		suggestions = append(suggestions,
			"usa `ovav git push` (gate enforced) en lugar de force push",
			"si requiere emergency: usa `owx <branch>` con waiver")
	case "sudo_root":
		suggestions = append(suggestions,
			"OVAV bloquea sudo por policy. usa `ovav admin <cmd>` si tienes permisos elevados")
	case "package_install":
		suggestions = append(suggestions,
			"OVAV bloquea package install en runtime. usa `ovav deps add <pkg>` que valida integridad")
	case "network_unbounded":
		// Preserve legacy localhost/127.0.0.1 exception from old Suggest()
		lower := strings.ToLower(command)
		if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
			return nil // localhost is safe
		}
		suggestions = append(suggestions,
			"network egress no permitido por network_guard. usa `ovav fetch <url>` (auditado)")
	case "git_branch_delete":
		suggestions = append(suggestions,
			"Branch deletion requiere acción del usuario. Usa `git branch -D <branch>` directamente si estás seguro.")
	case "git_checkout_new_branch":
		suggestions = append(suggestions,
			"Branch creation debe usar OVAV harness: `owc <branch-name>` para worktree dedicado.")
	case "gh_auth_token":
		suggestions = append(suggestions,
			"Reconfiguración de auth GitHub bloqueada. Contacta al admin del sistema.")
	default:
		// Generic fallback for any other DENY rule
		suggestions = append(suggestions,
			fmt.Sprintf("Comando bloqueado por gate: %s", decision.Reason))
	}

	return suggestions
}

// IsRisky returns true if the command warrants a suggestion.
func IsRisky(command string) bool {
	return len(Suggest(command)) > 0
}

// ── REPL wrapper ──────────────────────────────────────────────────────────

// RunREPL starts an interactive shell that hooks every command.
func (o *Observer) RunREPL(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-o.stopCh:
			return nil
		default:
		}
		fmt.Print("ovav> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		// Denies are mechanical. Suggestions explain the block but cannot be
		// confirmed through interactively, including during CEO sessions when a
		// permanent deny matched.
		decision := getBashGovernor().CheckWithCEO(line, "shell-repl")
		if !decision.Allowed {
			fmt.Printf("[ovav] bloqueado: %s\n", decision.Reason)
			for _, suggestion := range Suggest(line) {
				fmt.Println("  - " + suggestion)
			}
			continue
		}
		// Suggest if risky but still permitted.
		if sugg := Suggest(line); len(sugg) > 0 {
			fmt.Println("[ovav] sugerencia(s):")
			for _, s := range sugg {
				fmt.Println("  - " + s)
			}
		}
		// Execute
		code, _ := o.Run(ctx, parts[0], parts[1:]...)
		fmt.Printf("[ovav] exit code: %d\n", code)
	}
}

// ── JSON serialization ───────────────────────────────────────────────────

// Marshal serializes events for transport.
func MarshalEvent(e Event) ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEvent restores events.
func UnmarshalEvent(data []byte) (Event, error) {
	var e Event
	err := json.Unmarshal(data, &e)
	return e, err
}
