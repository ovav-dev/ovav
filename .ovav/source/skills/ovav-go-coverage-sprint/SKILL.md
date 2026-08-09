---
name: ovav-go-coverage-sprint
description: "OVAV Go coverage sprint — systematic coverage boost for Go packages. Trigger: coverage sprint, cobertura, coverage boost, go test.*cover, hitting below 80% on a package. Runs inside a feature branch worktree via owc."
---

# OVAV Go Coverage Sprint

Systematic coverage boost for OVAV Go packages. 3-phase iterative pattern.

## Phase 0 — Prepare worktree

```bash
owc sprint{N}-{topic}
# e.g.: owc coverage-fase3-ows
```

## Phase 1 — Baseline analysis

```bash
go test -timeout 180s ./... -coverprofile=/tmp/c.cover
go tool cover -func=/tmp/c.cover | sort -t% -k3 -r | head -40
```

Target: every package ≥80%.

## Phase 2 — Add coverage tests

Create `coverage_boost_test.go` in the target package directory.

```go
func Test<UncoveredFunc>(t *testing.T) {
    repoDir := t.TempDir()
    // ... set up real git repo, run function, assert result
}
```

Key patterns from past sprints:
- `httptest.NewServer` for HTTP handlers (watch: Go ServeMux redirects /path → /path/ when exact + prefix patterns both exist → 301/405 errors)
- `os.Chdir + os.Pipe()` for `main()` coverage (subprocess invisible to coverprofile)
- `os.Chmod(0555)` for error-path coverage
- `t.TempDir()` for fake repos — `setupTestRepoWithRemote` creates "master" default; use `git checkout -b main` for main-branch tests
- Real git worktrees via `git init && git commit --allow-empty` for OWS handler coverage

**Rule**: test-only commits (`coverage_boost_test.go` only) keep blast radius contained and simplify rollback.

**Rule**: `go test` + `go tool cover -func` after every edit — compilation errors are common with OWS type signatures. Iterate until tests pass.

## Phase 3 — gofmt + commit + owd

After tests pass, format all modified Go files:

```bash
gofmt -w <modified_dirs>
```

Then commit and merge:

```bash
git add go-runtime/internal/<pkg>/coverage_boost_test.go
git commit -m "test(<pkg>): coverage boost — <N> tests, <X>%→<Y>%"
# NOT 'git add .' — stage only the coverage file
owd
```

## Structural ceiling rule

Some packages cannot reach 80% via unit tests alone:
- Handlers needing real git worktree state (makeDoneHandler, makeMoveHandler)
- Functions requiring profile configs or signed commits
- If ceiling hit: document the gap and move to next package.

## Coverage targets (verified 2026-07-25)

| Package | Target | Actual |
|---|---|---|
| hooks | ≥80% | 94.0% |
| validators | ≥80% | 81.5% |
| doctor | ≥80% | 97.0% |
| economy | ≥80% | 95.3% |
| project | ≥80% | 84.8% |
| chronos | ≥80% | 85.3% |
| vault | ≥80% | 87.9% |
| status | ≥80% | 95.3% |
| sbom | ≥80% | 91.1% |
| permissions | ≥80% | 94.7% |
| license | ≥80% | 84.1% |
| cli | ≥80% | 81.7% |
| ows | ≥80% | 79.2% ⚠️ |
| cmd/ovav | ≥80% | 62.6% ⚠️ |
