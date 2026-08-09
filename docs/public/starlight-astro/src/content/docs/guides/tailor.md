---
title: Tailor Composer
description: Create and manage development plans with the OVAV Tailor Composer — plan-gated workstation configuration.
---

The Tailor Composer is OVAV's workstation configuration engine. It models your setup as a **plan-gated state machine** — selecting a plan level unlocks compatible tools and roles.

## How It Works

Tailor uses three plan levels that form a hierarchy:

```
Nucleo (Core) → Studio → Command
```

Each plan unlocks specific tools and roles. Higher plans include everything from lower plans plus additional capabilities.

## Plan Levels

| Plan | Tagline | Unlocks |
|---|---|---|
| **Nucleo** | local base · free | OpenCode, Git, Platform Engineering |
| **Studio** | editor + sessions | Neovim, Zellij, Fish, Research Intelligence |
| **Command** | advanced operation | Security Architecture, full toolset |

## CLI Commands

### Show Current State

```bash
ovav tailor
# or
ovav tailor status
```

Output:

```
Plan: Core | 2 tools · 1 role
  [✓] OpenCode     governed AI workspace
  [✓] Git          safe versioning
  [ ] Neovim       technical editing
  [ ] Zellij       live sessions
  [ ] Fish         productive shell
  [✓] Platform Engineering    systems, CLI and runtime
```

### Select a Plan

```bash
ovav tailor select nucleo    # Base level
ovav tailor select studio    # Editor + sessions
ovav tailor select command   # Advanced operation
```

Selecting a plan automatically disables items not allowed at that level and unlocks compatible options.

### Toggle Tools and Roles

```bash
ovav tailor toggle opencode
ovav tailor toggle nvim
ovav tailor toggle platform_engineering
```

Items that require a higher plan will show a message:

```
Neovim requires plan Studio. Switch plan first.
```

### Preview Changes

```bash
ovav tailor preview
```

Shows pending changes before applying:

```
Pending changes:
  + Neovim: technical editing
  + Zellij: live sessions
  - OpenCode: governed AI workspace
```

### Apply Configuration

```bash
ovav tailor apply
```

Applies the current selection:

```
✓ Configuration applied:
  Plan:          Studio
  Tools:         OpenCode, Git, Neovim, Zellij
  Roles:         Platform Engineering
```

## Plan Gating

The gating system ensures compatibility:

```go
// A plan unlocks items whose min_plan rank <= the selected plan's rank
func (s *State) IsAllowed(minPlan string) bool {
    return planRank(s.SelectedPlan) >= planRank(minPlan)
}
```

Each tool and role declares a `min_plan` requirement:

| Item | Min Plan |
|---|---|
| OpenCode | Nucleo |
| Git | Nucleo |
| Platform Engineering | Nucleo |
| Neovim | Studio |
| Zellij | Studio |
| Fish | Studio |
| Research Intelligence | Studio |
| Security Architecture | Command |

## State Machine

Tailor maintains a snapshot of applied state for change detection:

```go
type Snapshot struct {
    SelectedPlan string
    Plans        map[string]bool
    Tools        map[string]bool
    Roles        map[string]bool
}
```

This enables:

- **Diff detection** — `PreviewChanges()` shows what's different
- **Rollback awareness** — previous state is preserved
- **Timestamp tracking** — `LastAppliedAt` records when changes were applied

## Integration with Cockpit

The Tailor Composer is integrated into the Cockpit TUI. From the Cockpit's Tailor view, you can:

1. Navigate plans with arrow keys
2. Toggle tools and roles with Enter
3. Preview changes before applying
4. Apply configuration with confirmation

## Programmatic Use

Tailor is a Go library (`go-runtime/internal/tailor/`) that can be used programmatically:

```go
import "github.com/ovav/ovav/internal/tailor"

// Create state with detected tools
detected := map[string]bool{"git": true, "opencode": true}
state := tailor.NewState(detected)

// Select a plan
state.SelectPlan("studio")

// Toggle an item
state.ToggleAt(2)

// Preview and apply
changes := state.PreviewChanges()
results := state.ApplySelection()
```

## Next Steps

- [First Profile Setup](/guides/first-profile) — Apply your first profile
- [CLI Reference](/reference/cli) — Full command documentation
- [Architecture](/core/architecture) — How OVAV works
