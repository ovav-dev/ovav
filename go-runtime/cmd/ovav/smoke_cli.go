package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/cli"
)

const (
	defaultSourceSmokeTimeout = 30 * time.Second
	defaultFreshSmokeTimeout  = 5 * time.Minute
	maximumSmokeTimeout       = 10 * time.Minute
)

type smokeOptions struct {
	timeout   time.Duration
	json      bool
	keepClone bool
}

func parseSmokeArgs(args []string, fresh bool) (smokeOptions, error) {
	opts := smokeOptions{timeout: defaultSourceSmokeTimeout}
	if fresh {
		opts.timeout = defaultFreshSmokeTimeout
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			opts.json = true
		case arg == "--keep-clone" && fresh:
			opts.keepClone = true
		case arg == "--timeout":
			if i+1 >= len(args) {
				return smokeOptions{}, fmt.Errorf("--timeout requires a duration")
			}
			i++
			duration, err := time.ParseDuration(args[i])
			if err != nil {
				return smokeOptions{}, fmt.Errorf("invalid --timeout: %w", err)
			}
			opts.timeout = duration
		case strings.HasPrefix(arg, "--timeout="):
			duration, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return smokeOptions{}, fmt.Errorf("invalid --timeout: %w", err)
			}
			opts.timeout = duration
		default:
			return smokeOptions{}, fmt.Errorf("unknown smoke argument %q", arg)
		}
	}
	if opts.timeout <= 0 || opts.timeout > maximumSmokeTimeout {
		return smokeOptions{}, fmt.Errorf("--timeout must be greater than zero and at most %s", maximumSmokeTimeout)
	}
	return opts, nil
}

type sourceSmokeResult struct {
	SchemaVersion string                   `json:"schema_version"`
	Command       string                   `json:"command"`
	Timeout       string                   `json:"timeout"`
	TimedOut      bool                     `json:"timed_out"`
	Passed        bool                     `json:"passed"`
	Phases        []map[string]interface{} `json:"phases"`
}

func cmdSmoke(args []string) int {
	opts, err := parseSmokeArgs(args, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	repoRoot := cli.MustFindRepoRoot()
	result := runSourceSmoke(repoRoot, opts.timeout)

	if opts.json {
		if jsonOutput(result) != 0 {
			return 1
		}
	} else {
		fmt.Printf("OVAV Source Smoke (timeout %s)\n", opts.timeout)
		for _, phase := range result.Phases {
			icon := "✅"
			if phase["passed"] != true {
				icon = "❌"
			}
			fmt.Printf("  %s %s: %v\n", icon, phase["name"], phase["detail"])
		}
	}
	if !result.Passed {
		return 1
	}
	return 0
}

func runSourceSmoke(repoRoot string, timeout time.Duration) sourceSmokeResult {
	result := sourceSmokeResult{
		SchemaVersion: "ovav.source_smoke.v1",
		Command:       "smoke",
		Timeout:       timeout.String(),
		Phases:        []map[string]interface{}{},
	}
	runtimeRoot := filepath.Join(repoRoot, "go-runtime")
	if info, err := os.Stat(runtimeRoot); err != nil || !info.IsDir() {
		result.Phases = append(result.Phases, map[string]interface{}{
			"name": "locate_go_runtime", "passed": false, "detail": "go-runtime directory missing",
		})
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./cmd/ovav", "./internal/cli", "./internal/install", "-run", "^$", "-count=1")
	cmd.Dir = runtimeRoot
	cmd.Env = append(os.Environ(), "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	detail := "compile-only package smoke passed"
	if err != nil {
		detail = strings.TrimSpace(string(out))
		if len(detail) > 500 {
			detail = detail[:500]
		}
		if detail == "" {
			detail = err.Error()
		}
	}
	result.TimedOut = ctx.Err() == context.DeadlineExceeded
	if result.TimedOut {
		detail = "phase exceeded timeout " + timeout.String()
	}
	result.Passed = err == nil && !result.TimedOut
	result.Phases = append(result.Phases, map[string]interface{}{
		"name": "compile_cli_packages", "passed": result.Passed, "detail": detail,
	})
	return result
}
