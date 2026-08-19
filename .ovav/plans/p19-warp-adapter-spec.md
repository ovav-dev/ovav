# P19 — OWS Warp Adapter (preliminary spec)

Per plan §19, future OWS adapter for Warp Drive.

## Scope (presentation only — NOT git authority)

```
OWS
└── adapter/
    └── warp/
```

The adapter **may**:
- Open worktree path in Warp
- Set tab name, color, group
- Show task/profile in tab title
- Open Warp Code Review panel

The adapter **may NOT**:
- Create git worktree
- Delete worktree
- Merge branches
- Prune
- Move branches

## Interface (proposed)

```go
package adapter

type WarpAdapter struct {
    WarpBinaryPath string
    WorkflowName   string
}

func (w *WarpAdapter) OpenWorktree(worktreePath string, opts OpenOpts) error {
    // 1. Validate worktree exists (canonical via OWS)
    // 2. Resolve Warp workflow identifier (ovav.worktree)
    // 3. Invoke Warp with params: --worktree <path> --name <task>
    // 4. NO git worktree commands here
}

type OpenOpts struct {
    TabName    string
    Color      string  // blue, red, green, purple
    Group      string  // OVAV CORE, ACTIVE AGENTS, DEV
    Profile    string  // wt.feature, wt.refactor, ...
    Icon       string
}

func (w *WarpAdapter) ShowCodeReview(worktreePath string) error {
    // 1. Open Warp Code Review panel
    // 2. Attach to current branch differential
}

func (w *WarpAdapter) MarkCodeReviewRequired(worktreePath string) error {
    // 1. Set sentinel flag in worktree metadata
    // 2. Visible in Warp UI badge
}
```

## Security boundary

- Adapter validates OWS authority before ANY Warp action
- Rejects calls if worktree not in OWS registry
- Read-only on git state; writes only via Warp UI

## Implementation status

**DEFERRED.** Plan §19 says "crear posteriormente". Current focus is P0-P11.

## Files

- `go-runtime/internal/adapter/warp/adapter.go` (future)
- `go-runtime/cmd/ovav-worktree-warp/main.go` (future CLI wrapper)

## Acceptance criteria

- [ ] Adapter respects OWS authority (read-only on git)
- [ ] Warp tab config matches OWS workflow manifest
- [ ] No git worktree commands in adapter code path
- [ ] Tests verify rejection of direct git worktree creation
