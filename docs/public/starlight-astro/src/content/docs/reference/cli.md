---
title: CLI Commands
description: Complete reference for the OVAV Go CLI — all commands and flags.
---

The OVAV CLI (`ovav`) is a Go binary that provides the main interface to OVAV functionality.

## Installation

```bash
# Build from source
cd go-runtime && go build -o bin/ovav ./cmd/ovav/

# Or use the pre-built binary
./bin/ovav --help
```

## Commands

### `ovav status`
Display system status including git branch, commit, integrity mesh score, and active validators.
```bash
ovav status
ovav status --json
```

### `ovav validate`
Run the validator pipeline. Executes all registered validators and reports results.
```bash
ovav validate
ovav validate --quick     # Skip slow validators
ovav validate --json      # JSON output
```

### `ovav install`
Governed installation with safety gates, backup, and rollback.
```bash
ovav install plan         # Preview installation plan
ovav install apply        # Apply installation (with safety checks)
ovav install verify       # Verify installation integrity
ovav install rollback     # Rollback to previous state
```

### `ovav doctor`
Run system diagnostics — 13 checks covering git, Go, config, and runtime health.
```bash
ovav doctor
ovav doctor --json
```

### `ovav vault`
Manage the AES-256-GCM encrypted secrets vault.
```bash
ovav vault init           # Initialize vault
ovav vault set KEY VALUE  # Store a secret
ovav vault get KEY        # Retrieve a secret
ovav vault list           # List stored keys
```

### `ovav profile`
Profile compiler — manage OVAV professional profiles.
```bash
ovav profile list         # List available profiles
ovav profile apply NAME   # Apply a profile
ovav profile remove NAME  # Remove a profile
```

### `ovav cockpit`
Launch the Bubble Tea TUI dashboard (interactive terminal UI).
```bash
ovav cockpit
```

### `ovav tailor`
Launch the workstation composer for guided profile setup.
```bash
ovav tailor
```

### `ovav cpanel`
Manage the cPanel API server.
```bash
ovav cpanel start         # Start cPanel server
ovav cpanel status        # Check cPanel status
```

### `ovav deploy`
Governed deployment with safety gates.
```bash
ovav deploy --mode dry-run
ovav deploy --mode sandbox
ovav deploy --mode apply
```

### `ovav diagnose`
Full system diagnosis with theme verification.
```bash
ovav diagnose
ovav diagnose --json
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--help` | Show help for any command |
| `--version` | Show OVAV version |
| `--json` | Output results as JSON |
