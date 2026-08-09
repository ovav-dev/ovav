# OVAV Installation — Go Runtime v58.0

## Preflight

Before installing OVAV, verify system integrity:

```bash
# Go-native health check (replaces python3 tools/ovav_runtime.py validate)
ovav doctor --quick
ovav govern health
```

For a full diagnostic:
```bash
ovav doctor
```

## Installation

OVAV Go Runtime installs via the native Go binary:

```bash
# Build from source
cd go-runtime && go build -o /usr/local/bin/ovav ./cmd/ovav/

# Verify installation
ovav version
ovav status
```

### Install packs (governed apply)

```bash
# Plan and review changes
ovav install --plan --dry-run

# Apply with safety gates
ovav install --apply
```

### Component verification

```bash
# Verify governor wiring
ovav govern status

# Verify defense posture
ovav defend status

# Run defense scan
ovav defend scan

# Full system verify
ovav verify
```

## cPanel (Web Dashboard)

```bash
# Build and run
cd go-runtime && go build -o bin/cpanel ./cmd/cpanel/
./bin/cpanel

# Default port: 5858
# API: http://localhost:5858/api/v1/health
```

## Cockpit (Terminal UI)

```bash
# Build and launch
cd go-runtime && make build-cockpit
ovav cockpit
```

## Rollback

If installation fails, OVAV's governed apply creates a backup automatically:

```bash
# List available backups
ovav backup --list

# Restore from backup
ovav restore <backup-id>
```

## Fish Shell Integration

Fish functions are located in `config/fish/`. Source them:

```fish
# In ~/.config/fish/config.fish
source /path/to/ovav/config/fish/30-ovav-runtime-tools.fish
source /path/to/ovav/config/fish/35-ovav-wt-tsk.fish
source /path/to/ovav/config/fish/90-ovav-terminal-auto.fish
```

## Requirements

- Go 1.22+
- Git 2.40+
- Linux/macOS/Windows (amd64)
- No Python required for runtime operations

## Architecture

OVAV is now a Go-native governor system. All operational tools (`cmd/ovav/`, `cmd/cpanel/`, `cmd/cockpit/`, `cmd/tailor/`, `internal/`) are compiled Go binaries. Python tools are being migrated or wrapped as thin bridges (≤80 LOC).

- **CLI**: `cmd/ovav/` → `ovav` binary
- **API**: `cmd/cpanel/` → HTTP server on :5858
- **TUI**: `cmd/cockpit/` → Bubble Tea terminal UI
- **Worktrees**: `internal/ows/` → `owc`, `owd` worktree operations
- **Governor**: `internal/governor/` → Health, decisions, trust gate
- **Defense**: `internal/security/defense/` → Cortex, responder, authorizer
