// Package warp implements the OWS → Warp presentation adapter.
//
// Plan §19: OWS owns git worktree lifecycle. This adapter is the
// presentation layer for Warp (tabs, Code Review, notifications).
// The adapter is READ-ONLY on git state — it does NOT create, merge,
// prune, or move worktrees. All lifecycle authority remains with OWS.
package warp

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// WarpURI is the documented Warp URI scheme per docs.warp.dev
// "Open a Tab Config from a URL" section.
const WarpURI = "warp://tab_config"

// Adapter wraps Warp CLI invocations. It is intentionally minimal —
// it must NEVER add git side effects.
type Adapter struct {
	// WarpPath is the path to the Warp CLI binary.
	// Defaults: "warp.exe" (Windows), "warp" (macOS/Linux).
	WarpPath string
}

// New returns an Adapter with platform-appropriate default binary.
func New() *Adapter {
	return &Adapter{WarpPath: defaultBinary()}
}

// defaultBinary returns the Warp CLI path for the current OS.
// Looks up standard install locations to avoid PATH ambiguity.
func defaultBinary() string {
	candidates := []string{
		`C:\Program Files\Warp\warp.exe`,
		`/Applications/Warp.app/Contents/MacOS/warp`,
		"/usr/local/bin/warp",
		"/opt/warp/warp",
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	// Fallback to platform-default binary name.
	return "warp"
}

// OpenTabConfig opens a saved Tab Config via Warp URI scheme.
// The Tab Config must already exist in `tab_configs/` directory.
//
// This method does NOT create, modify, or delete any git state.
// It only invokes Warp's GUI to render a tab.
func (a *Adapter) OpenTabConfig(ctx context.Context, configName string) error {
	if configName == "" {
		return fmt.Errorf("warp: config name required")
	}
	if strings.ContainsAny(configName, "/\\") {
		return fmt.Errorf("warp: invalid config name: %q", configName)
	}
	uri := fmt.Sprintf("%s/%s", WarpURI, configName)
	cmd := exec.CommandContext(ctx, a.WarpPath, "open", uri)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// OpenWorktree opens Warp pointing at a worktree directory using a
// named Tab Config. The worktree MUST already exist (created via OWS).
// This method does NOT create or modify the worktree.
func (a *Adapter) OpenWorktree(ctx context.Context, worktreePath, configName string) error {
	if !filepath.IsAbs(worktreePath) {
		return fmt.Errorf("warp: worktree path must be absolute: %s", worktreePath)
	}
	// Verification: the directory must exist. We don't enter it.
	if _, err := exec.CommandContext(ctx, "test", "-d", worktreePath).CombinedOutput(); err != nil {
		return fmt.Errorf("warp: worktree path not found: %s", worktreePath)
	}
	return a.OpenTabConfig(ctx, configName)
}
