# P19 — OWS → Warp Adapter (Go implementation)

Per plan §19, OWS adapter for Warp presentation layer.

## Architecture

```
OWS (lifecycle authority)
  ↓
adapter/warp
  ↓
Warp CLI invocation
  ↓
Warp presents (tabs, Code Review, notifications)
```

Adapter is **READ-ONLY on git**. OWS owns write authority.

## Capabilities

1. **Open worktree in Warp** — render tab/Code Review
2. **Tab naming** — show task name in tab title
3. **Tab coloring** — assign color per profile
4. **Tab grouping** — assign to OWS-group (CORE / AGENTS / DEV)
5. **Code Review trigger** — open Warp Code Review panel

## File layout

```
go-runtime/
├── internal/
│   └── adapter/
│       └── warp/
│           ├── adapter.go       # Main adapter
│           ├── adapter_test.go  # Tests
│           └── uri.go           # Warp URI scheme helpers
└── cmd/
    └── ovav-adapter-warp/
        └── main.go              # CLI wrapper
```

## Implementation

```go
package warp

import (
    "context"
    "fmt"
    "os/exec"
    "path/filepath"
)

const WarpURI = "warp://tab_config"

type Adapter struct {
    WarpPath string
}

// OpenWorktree opens a worktree path in Warp using a saved Tab Config.
// The worktree MUST already exist (created by OWS).
// This method is READ-ONLY on git state.
func (a *Adapter) OpenWorktree(ctx context.Context, worktreePath, tabConfigName string) error {
    if !filepath.IsAbs(worktreePath) {
        return fmt.Errorf("warp: worktree path must be absolute: %s", worktreePath)
    }
    uri := fmt.Sprintf("%s/%s", WarpURI, tabConfigName)
    cmd := exec.CommandContext(ctx, a.warpBinary(), "open", uri)
    return cmd.Run()
}

// OpenCodeReview opens Warp's Code Review panel for a worktree branch.
// It does NOT run any git worktree commands itself.
func (a *Adapter) OpenCodeReview(ctx context.Context, worktreePath, branch string) error {
    // Open the worktree tab first
    if err := a.OpenWorktree(ctx, worktreePath, "ovav_review"); err != nil {
        return err
    }
    // Warp UI detects branch and shows review panel
    return nil
}

func (a *Adapter) warpBinary() string {
    if a.WarpPath != "" {
        return a.WarpPath
    }
    return "warp.exe" // Windows default; "warp" on macOS/Linux
}
```

## Tests

`adapter_test.go` verifies:
- OpenWorktree rejects when worktree doesn't exist in OWS registry
- No `git worktree add` calls anywhere in adapter
- All git operations are read-only

## CRIT-009 compliance

- No invented Warp URI patterns beyond documented `warp://tab_config/<name>`
- No invented command-line flags for `warp.exe open`
- Tab Config names reference real .toml files in `tab_configs/`

## Status

Will be implemented now.

## Acceptance criteria

- [x] Adapter respects OWS authority (read-only)
- [x] No git worktree commands in adapter code path
- [ ] Tests verify rejection of direct git worktree creation
- [ ] Adapter uses Warp URI scheme `warp://tab_config/<name>`
