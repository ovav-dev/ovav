# OVAV Code Quality Enforcement — Cross-Model Guarantee Architecture

**Author**: Platform Engineering (Thavren's squad)  
**Specialists**: Marco (Systems Architect) + Andrés (Senior Implementer)  
**Date**: 2026-07-28  
**Status**: proposal/v1  

---

## Section 1: 10 Code Quality Dimensions

### D1 — Architecture Coherence
Single-responsibility enforcement, proper layering (presentation/domain/data), cyclic dependency detection, correct import direction (no inward violations).

### D2 — Data Type Rigor
TypeScript/Go interface coverage, schema validation at boundaries (Zod, Valibot, JSON Schema), no `any` / `interface{}` escapes without explicit suppressed type-ignore.

### D3 — Error Handling Completeness
Every external call (I/O, network, DB, FS) has typed error handling; no silent `catch (e) {}`; error propagation chains preserve context.

### D4 — Naming Consistency
 snake_case / camelCase / PascalCase enforced per language; no abbreviations that break dictionary lookup; file names match exported symbols.

### D5 — Test Coverage Minimums
Per-module coverage floor (default 80% lines, 70% branches); critical paths (auth, payment, data mutation) require 90%+.

### D6 — Security Hygiene
Input sanitization, SQL injection prevention, secrets not in source, dependency vulnerability scanning (OSV), no `eval()`, correct CORS/CSRF handling.

### D7 — Code Organization
Module boundaries respected; no circular requires; barrel-file hygiene; files under 300 LOC; directories under 10 files unless aggregation is intentional.

### D8 — Dependency Management
Locked versions; no transitive dev-dependencies in production bundle; no duplicate top-level deps; automated update cadence with changelog audit.

### D9 — Documentation Completeness
Every exported symbol has a doc comment; README per package; CHANGELOG.md; inline justification comments for non-obvious business logic.

### D10 — Performance Considerations
No N+1 query patterns; O(n²) or worse flagged; bundle size budget per entry point; memory leak patterns (unclosed timers, listeners) detected.

---

## Section 2: Validators F6–F10 Proposal

### F6 — Architecture Coherence Validator

**F6_arch_coherence**

| Field | Value |
|---|---|
| **What it checks** | Layering violations (domain imports presentation), cyclic deps, single-responsibility score per module |
| **How it runs** | Pre-commit hook + CI stage `ovav validate --gate arch` |
| **Pass criteria** | Zero layering violations, zero cycles (import graph acyclic), no module > 5 responsibilities |
| **Auto-fix** | No — requires architect review for restructuring |

**Implementation notes**:
- Build import-graph via AST (TypeScript: `ts-morph`, Go: `go/parser`+`go list -f '{{.Imports}}'`)
- Layer rules encoded in `.ovav/rules/layering.yaml`
- Cycles reported as: `ARCH_CYCLE: pkg/a → pkg/b → pkg/c`

---

### F7 — Data Type Rigor Validator

**F7_type_rigor**

| Field | Value |
|---|---|
| **What it checks** | `% any` / `interface{}` escape count, schema validation presence at module boundaries, discriminated union coverage |
| **How it runs** | Pre-commit hook + CI `ovav validate --gate types` |
| **Pass criteria** | `any` count = 0 in production TS; `interface{}` count = 0 in production Go; all API boundaries have schema |
| **Auto-fix** | Partial — `any` → `unknown` + type-guard generation; schema skeleton generation |

**Implementation notes**:
- ESLint: `@typescript-eslint/no-explicit-any` set to `error`; `@typescript-eslint/explicit-function-return-type` on public APIs
- Go: `go vet` + custom rule via `staticcheck` for `interface{}`
- Schema validation: require `zod`, `valibot`, or `ajv` at every HTTP boundary
- TypedStrict mode: generated code from `.ovav/schemas/` must pass

---

### F8 — Error Handling + Naming Coherence Validator

**F8_err_naming**

| Field | Value |
|---|---|
| **What it checks** | Unhandled promise rejections, catch blocks without re-throw or typed handling, naming-rule violations |
| **How it runs** | Pre-commit + CI `ovav validate --gate err-naming` |
| **Pass criteria** | Zero unhandled rejections, zero bare `catch {}`, naming rules: no `tmp`, `data`, `info`, `obj` variables |
| **Auto-fix** | Yes — `catch (e) {}` → `catch (e: unknown) { logger.error(...); throw e; }`; rename variables via `cspell`/`gofmt -r` |

**Implementation notes**:
- ESLint: `no-emptyCatch`, `@typescript-eslint/unbound-method` for error callbacks
- Go: `errcheck` tool enforcing `_ = fn()` pattern is banned
- Naming: project-specific `dictionary.txt` + `.github/labeler.yml` for name enforcement

---

### F9 — Test Coverage + Security Hygiene Validator

**F9_cov_sec**

| Field | Value |
|---|---|
| **What it checks** | Line/branch coverage per module; OSV vulnerabilities; secrets in source; OWASP Top 10 patterns |
| **How it runs** | CI-only (too heavy for pre-commit) `ovav validate --gate coverage --threshold 80` |
| **Pass criteria** | Coverage ≥ 80% lines, ≥ 70% branches; zero OSV HIGH/CRITICAL; zero detected secrets; zero OWASP flags |
| **Auto-fix** | Coverage: no; Secrets: yes — auto-redact pattern matches into `.env.example`; OSV: auto-create PR with patch if available |

**Implementation notes**:
- Coverage: `c8` for JS/TS, `go test -coverprofile` + `covplot` for Go
- OSV: `osv-scanner` in CI; threshold configurable per `.ovav/rules/security.yaml`
- Secrets: `gitrob` / `ggshield` as pre-commit OR `ovav scan --secrets`
- OWASP: ESLint plugin + Bandit (Python), staticcheck (Go)

---

### F10 — Code Organization + Dependency + Documentation + Performance Validator

**F10_org_dep_doc_perf**

| Field | Value |
|---|---|
| **What it checks** | File LOC, module size, circular requires, dep version drift, doc comment coverage, N+1, O(n²) |
| **How it runs** | CI `ovav validate --gate full` (bundles F6–F9 + these extras) |
| **Pass criteria** | No file > 300 LOC; no dir > 10 files unless aggregated; deps locked + up-to-date within 30 days; exported fns documented; zero N+1/quadratic patterns |
| **Auto-fix** | LOC/org: no; Deps: yes — `npm audit fix` / `go mod tidy`; Docs: partial (stub generation); Perf: no (flag for human review) |

**Implementation notes**:
- LOC: `cloc` or `tokei` in CI
- Deps: `npm outdated` + `Dependabot` / `Renovate` config validation
- Docs: `typedoc` / `godoc` validation; missing docs → `WARN` not `FAIL` unless public API
- Perf: `eslint-plugin-no-expensive-tests`; Go: `pprof` diff on performance-critical paths

---

## Section 3: Quality Gate Pipeline Design

```
[CODE GENERATED by Any Model]
         │
         ▼
┌─────────────────────────────────┐
│  PRE-COMMIT GATE (local)        │
│  ovav pre-commit --fast         │
│  ├─ F7_type_rigor (fast AST)    │
│  ├─ F8_err_naming (fast lint)   │
│  └─ F6_arch_coherence (import   │
│      graph, no full parse)      │
│  BLOCK: F6/F7/F8 failures       │
│  AUTO-FIX: F8 naming+catch      │
└─────────────────────────────────┘
         │ (pass)
         ▼
┌─────────────────────────────────┐
│  STAGED → CI PIPELINE           │
│  ovav validate --gate full       │
│  ├─ F6  arch_coherence          │
│  ├─ F7  type_rigor              │
│  ├─ F8  err_naming              │
│  ├─ F9  cov_sec                 │
│  └─ F10 org_dep_doc_perf        │
│                                 │
│  Results → .ovav/quality_report  │
│  ├─ score per dimension (0-100) │
│  ├─ violations list             │
│  └─ model fingerprint (which    │
│      model generated this)       │
└─────────────────────────────────┘
         │
    ┌────┴────┐
    │  PASS   │  FAIL
    ▼         ▼
┌────────┐  ┌──────────────────┐
│MERGE   │  │BLOCK MERGE      │
│ALLOWED │  │ovav quality --  │
│        │  │block-reason     │
│        │  │auto-assign to   │
│        │  │next free model  │
└────────┘  │(self-healing if  │
           │fixable, else     │
           │human review)      │
           └──────────────────┘
```

### Gate Execution Order

| Step | Gate | Runtime | Blocks Commit? | Auto-fix? |
|---|---|---|---|---|
| 1 | F7_type_rigor | < 5s | Yes | Partial |
| 2 | F8_err_naming | < 10s | Yes | Yes |
| 3 | F6_arch_coherence | < 15s | Yes | No |
| 4 | F9_cov_sec | 30–120s | No (CI only) | Partial |
| 5 | F10_org_dep_doc_perf | 60–180s | No (CI only) | Partial |

### Merge Requirements

- All F6–F8 must pass locally (pre-commit)
- F9–F10 must pass in CI before merge is allowed
- Quality score per dimension must be ≥ 70/100
- Any dimension < 50 triggers automatic model re-generation request

---

## Section 4: Cross-Model Quality Guarantee Architecture

### The Core Problem

Top-tier models (Fable 5, Opus 5) produce superior architecture **by default**.
Mid-tier models (DeepSeek, MiniMax) can match that quality **if forced to**.
OVAV's job: make the output indistinguishable.

### How OVAV Becomes the Intelligent Layer

```
┌─────────────────────────────────────────────────────────────┐
│                    OVAV QUALITY LAYER                       │
│                                                             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────────┐   │
│  │ Fable 5     │   │ Opus 5      │   │ DeepSeek/MiniMax│   │
│  │ (high-base) │   │ (high-base) │   │ (mid-base)      │   │
│  └──────┬──────┘   └──────┬──────┘   └────────┬────────┘   │
│         │                 │                   │             │
│         └─────────────────┼───────────────────┘             │
│                           ▼                                 │
│              ┌────────────────────────┐                     │
│              │  QUALITY GATE (F6-F10) │                     │
│              │  Same for ALL models    │                     │
│              │  No model-specific rules │                    │
│              └────────────┬─────────────┘                    │
│                           │                                  │
│         ┌─────────────────┼─────────────────┐              │
│         ▼                 ▼                 ▼              │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐       │
│  │ PASS       │    │ AUTO-FIX   │    │ REJECT +   │       │
│  │ (proceed)  │    │ + RESUBMIT │    │ RE-GEN     │       │
│  └────────────┘    └────────────┘    └────────────┘       │
│                                                             │
│  RE-GEN: same model OR fallback model                      │
│  Decision: ovav quality --route-retry                      │
└─────────────────────────────────────────────────────────────┘
```

### Key Architectural Principles

1. **Model-Agnostic Gates** — F6–F10 never inspect which model generated the code. Quality is binary.

2. **Feedback Loop** — Quality failures are logged with model fingerprint. Over time: which model fails which dimension most → used to route to better-suited model for given task type.

3. **Self-Healing Pipeline** — Auto-fixable violations (F8 naming, F9 secrets, F10 dep updates) are fixed and re-validated without human involvement. Non-auto-fixable violations trigger re-generation.

4. **Quality Budget per Model** — If DeepSeek consistently fails F6 (arch coherence) on complex tasks, OVAV routes those tasks to Fable/Opus. The quality gate enforces the standard; routing optimizes cost.

5. **Unified Output Contract** — All models must produce artifacts that pass F6–F10. The contract is the only thing that matters, not the model's internal reasoning quality.

### Cross-Model Equivalence Guarantee

| Concern | Without OVAV | With OVAV |
|---|---|---|
| Architecture | Fable 5 ≫ DeepSeek | All models → same standard via F6 |
| Type Safety | Fable 5 ≫ DeepSeek | All models → same standard via F7 |
| Error Handling | Opus 5 > MiniMax | All models → same standard via F8 |
| Coverage | Model-dependent | Enforced via F9 |
| Organization | Fable 5 ≫ others | All models → same standard via F10 |

### Routing Decision Matrix (Future)

```
Task Type          │ DeepSeek  │ MiniMax  │ Fable 5  │ Opus 5
───────────────────┼───────────┼──────────┼──────────┼──────
Simple CRUD        │ ✓ (F6-F10)│ ✓ (F6-F10│ ✓ (F6-F10│ ✓
Complex domain      │ ✗ →Fable  │ ✗ →Opus  │ ✓        │ ✓
Real-time perf     │ ✗ →Opus   │ ✗ →Fable │ ✓        │ ✓
Security-critical  │ ✓ (F9)    │ ✓ (F9)   │ ✓        │ ✓
```

OVAV routes based on failure patterns tracked in `.ovav/quality_history.db`.

---

## Appendix: Validator Registry Entry

```yaml
validators:
  - id: F6
    name: arch_coherence
    description: Layering + cyclic dependency check
    gate: pre-commit + CI
    blocking: true
    autofix: false
    weight: 20

  - id: F7
    name: type_rigor
    description: TypeScript/Go type enforcement
    gate: pre-commit + CI
    blocking: true
    autofix: partial
    weight: 20

  - id: F8
    name: err_naming
    description: Error handling + naming consistency
    gate: pre-commit + CI
    blocking: true
    autofix: true
    weight: 15

  - id: F9
    name: cov_sec
    description: Coverage minimums + security hygiene
    gate: CI only
    blocking: true
    autofix: partial
    weight: 25

  - id: F10
    name: org_dep_doc_perf
    description: Organization, dependencies, docs, performance
    gate: CI only
    blocking: true
    autofix: partial
    weight: 20
```

**Total enforcement weight**: 100 — every artifact must score ≥ 70 to pass.
