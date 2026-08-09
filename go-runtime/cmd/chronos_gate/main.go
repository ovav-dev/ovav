// OVAV Chronos Gate — CLI wrapper for internal/chronos Go package.
//
// Replaces tools/agent_runtime/chronos_gate.py (deleted Python, 841 LOC).
// Called by session_greeting.py via subprocess. Outputs chronos_gate.v1 JSON.
//
// USAGE:
//
//	chronos_gate                     # JSON output (default: 5 commits, 120min threshold)
//	chronos_gate --repo /path        # explicit repo root (default: auto-detect from cwd)
//	chronos_gate --timeline 10       # show 10 commits
//	chronos_gate --session-min 60    # 1hr session threshold
//	chronos_gate --compact           # compact JSON (single line)
//
// Stack: Go 1.25+, go-git v5, stdlib. Zero Python dependencies.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ovav/ovav/internal/chronos"
)

func main() {
	repoRoot := flag.String("repo", "", "Repository root path (default: auto-detect from cwd)")
	timelineCount := flag.Int("timeline", 5, "Number of timeline entries")
	sessionMin := flag.Int("session-min", 120, "Session continuation threshold in minutes")
	compact := flag.Bool("compact", false, "Compact JSON output (single line)")
	flag.Parse()

	root := *repoRoot
	if root == "" {
		root = findGitRoot()
	}
	if root == "" {
		cwd, _ := os.Getwd()
		root = cwd
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error":"chronos_gate: cannot resolve path: %v"}`+"\n", err)
		os.Exit(1)
	}

	output := chronos.BuildChronosOutput(absRoot, *timelineCount, *sessionMin)

	var jsonBytes []byte
	if *compact {
		jsonBytes, err = json.Marshal(output)
	} else {
		jsonBytes, err = json.MarshalIndent(output, "", "  ")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error":"chronos_gate: JSON marshal failed: %v"}`+"\n", err)
		os.Exit(2)
	}

	fmt.Println(string(jsonBytes))
}

// findGitRoot walks up from cwd until it finds a .git directory or file (worktree).
func findGitRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() || info.Mode().IsRegular() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
