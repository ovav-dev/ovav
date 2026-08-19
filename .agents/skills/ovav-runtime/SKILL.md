---
name: ovav-runtime
description: OVAV Go runtime tasks, build, install, validate.
trigger: ovav runtime, build, install, go-runtime
---

# ovav-runtime

OVAV Go runtime operations: build, install, test, validate.

## Workflow

```bash
# Build current platform
cd go-runtime && make build-current

# Build all platforms
cd go-runtime && make build

# Install to ~/.local/bin
cd go-runtime && make install

# Tests
cd go-runtime && make test

# Coverage
cd go-runtime && make test-cover

# Lint
cd go-runtime && make lint

# Format
cd go-runtime && make fmt
```

## Tools built

| Binary | Purpose |
|---|---|
| `ovav` | Main CLI (worktree, validators, governance) |
| `ovav-cockpit` | TUI (Bubble Tea) |
| `ovav-vault-secrets` | Secrets subsystem |
| `ovav-vault-tui` | Vault TUI |

## Key packages

- `internal/validators/` — 50+ validation gates
- `internal/governor/` — decision engine, delegation
- `internal/memory/` — agent memory, vector store
- `internal/gitflow/` — worktree management (OWS)
- `internal/vault/` — secrets management, encryption
- `internal/connect/` — provider abstraction (Anthropic, OpenAI, MiniMax)

## Rules

- Format with `gofumpt` (NOT gofmt)
- Tests: `go test -v -count=1 -race`
- Never use `panic()` in library code — return `error`
- Zero-value initialization preferred
- Interface names: `-er` suffix (Reader, Writer, Validator)

## Failure modes

- "build failure" → `go vet`, check imports
- "test race" → `go test -race -count=1`
- "permission denied" on install → check `~/.local/bin` ownership
