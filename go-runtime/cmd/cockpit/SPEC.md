# OVAV Cockpit — SPEC.md
## Advanced 2026 TUI Experience

**Version:** 1.0.0
**Date:** 2026-08-11
**Lead:** Thavren (Platform Engineering)
**Vision:** Premium terminal experience — dense but digestible, alive with data, zero friction navigation.

---

## 1. Vision & Principles

### Core Experience
OVAV Cockpit es el **centro de comando** de un cerebro vivo. Cada view debe transmitir:
- **Vitalidad** — datos reales, actualizados, con contexto histórico
- **Profundidad** — multi-nivel: overview → detail → action
- **Velocidad** — información crítica visible sin scroll, actions en 1-2 keys
- **Confianza** — status siempre claro, errores impossibles de ignorar

### Design Principles (heredados + extendidos)
| Principle | Implementation |
|-----------|---------------|
| **Información densa** | Multiple data points per viewport, no wasted space |
| **Visual hierarchy** | Color-coded severity, size-coded importance, position-coded context |
| **Zero-latency feedback** | Every keypress produces immediate visual response |
| **Graceful degradation** | Narrow terminals (60+) get compact layouts; wide (120+) get full cards |
| **Accessibility** | High contrast ratios (WCAG AA), color never sole indicator |

### Terminal Responsive Strategy
```
Width < 60   → Single column, minimal chrome, essential info only
Width 60-90  → Two column, compact cards, truncated labels
Width 90-120 → Full two/three column, complete cards, full labels
Width 120+   → Maximum density, sidebar available, full detail panels
```

---

## 2. View Architecture

### 2.1 Complete View Map

```
ROOT (Menu)
├── 🚀 Work Done & Updates        → ViewUpdates (EXISTING — needs enhancement)
├── 📊 Plan Dashboard             → ViewDashboard (EXISTING — needs enhancement)
├── 💚 Health & Status           → ViewHealth (EXISTING — needs enhancement)
├── 🔐 OVAV Vault                → ViewVault (EXISTING — enhancement optional)
├── 🔄 Sync Projection           → ViewSync (EXISTING — enhancement optional)
├── ⚙️ Configuration             → ViewConfig (EXISTING — enhancement optional)
├── 📦 Install Pipeline          → ViewInstall (EXISTING — needs enhancement)
├── 🧩 Tailor Composer           → ViewTailor (EXISTING — enhancement optional)
├── ⚡ CLI Runtimes              → ViewCLI (EXISTING — enhancement optional)
├── 🧪 Testing & Coverage        → ViewTesting (NEW)
├── 🎯 Delegation Runtime        → ViewDelegation (NEW)
├── 🔬 Research & Evidence       → ViewResearch (NEW)
├── 🛡️ Adversarial View          → ViewAdversarial (NEW)
├── 📈 Performance Monitor        → ViewPerformance (NEW)
└── ❓ Help                       → ViewHelp (EXISTING — minor enhancement)
```

### 2.2 View Specifications

---

#### VIEW: Work Done & Updates (`updates_view.go`)
**Purpose:** Timeline of completed work ready to push / already synced

**Layout (wide terminal):**
```
┌─────────────────────────────────────────────────────────────────────┐
│  🚀 Work Done & Updates                              [c] config  [?]│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ PENDING ────────────────┐  ┌─ SYNCED ─────────────────────────┐│
│  │ 🔵 P1-KC-compile        │  │ ✅ P1-delegation-system          ││
│  │    2026-08-11 · HEAD~3  │  │    2026-08-10 · abc123f         ││
│  │                         │  │                                  ││
│  │ 🔵 P1-memory-ovav      │  │ ✅ P1-context-pack               ││
│  │    2026-08-11 · HEAD~1  │  │    2026-08-09 · def456a         ││
│  └─────────────────────────┘  └──────────────────────────────────┘│
│                                                                     │
│  ┌─ COMMIT DIFF ───────────────────────────────────────────────────┐│
│  │ diff --git a/go-runtime/... b/go-runtime/...                   ││
│  │ +func (m Model) renderUpdates() string {                        ││
│  │ ...                                                            ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  [↑↓] Navigate  [Enter] Detail  [s] Sync All  [c] Config  [Esc] Back│
└─────────────────────────────────────────────────────────────────────┘
```

**Data sources:**
- Git log --oneline (local pending commits)
- `.ovav/registry/sync_ledger.yaml` (synced items)
- Diff preview via `git show --stat HEAD`

**Keyboard:**
- `↑↓` — navigate items
- `Enter` — show commit diff in detail panel
- `s` — trigger sync to Product
- `c` — open sync config
- `Esc` — back to menu

**States:**
- Empty: "No pending commits. Run `owd` to merge completed worktrees."
- Syncing: spinner + "Syncing with Product..."
- Sync error: red banner with error message
- All synced: green banner "Fully synchronized"

---

#### VIEW: Plan Dashboard (`dashboard.go`)
**Purpose:** caps.yaml visual with milestone tracking

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  📊 Plan Dashboard                              [v 2.1.0]  [?]      │
├─────────────────────────────────────────────────────────────────────┤
│  Strategy: Full-Stack Go+TS    Stack: go1.22 / node20              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ MILESTONES ────────────────────────────────────────────────────┐│
│  │ ● P1: Knowledge Compiler      ████████████░░░░░░░░  75%  (3/4) ││
│  │ ○ P2: Memory Bridge          ░░░░░░░░░░░░░░░░░░░░   0%  (0/4) ││
│  │ ○ P3: Governance Mesh        ░░░░░░░░░░░░░░░░░░░░   0%  (0/3) ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ COMPLETED ────────┐  ┌─ PENDING ──────────────────────────────┐│
│  │ ✅ P1-KC-compile  │  │ 🔵 P1-memory-ovav (23%)                ││
│  │ ✅ P1-context-pack│  │ 🔵 P1-delegation-system (67%)          ││
│  │ ✅ P1-validate    │  │ 🔵 P1-brainstorm (10%)                 ││
│  └───────────────────┘  └────────────────────────────────────────┘│
│                                                                     │
│  [↑↓] Navigate  [Enter] Detail  [/] Filter  [d] Dependencies  [?]  │
└─────────────────────────────────────────────────────────────────────┘
```

**Data sources:**
- `.ovav/plan/caps.yaml` (CapsData struct)
- Milestone grouping via `cap.milestone` or `cap.order`

**Enhancement over current:**
- Add milestone grouping with progress bars
- Add dependency visualization (`d` key shows DAG)
- Search filters by milestone, status, stack
- Click on pending item → inline detail without nav push

---

#### VIEW: Health & Status (`health.go`)
**Purpose:** Real-time system diagnostics

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  💚 Health & Status                              [r] refresh  [?]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ IDENTITY ────────┐  ┌─ GIT ──────────────┐  ┌─ RUNTIME ────────┐│
│  │ Plan   v2.1.0    │  │ Branch  develop   │  │ Go      1.22.4   ││
│  │ Root   /ovav    │  │ HEAD    abc123f   │  │ Goroutines  12   ││
│  │ Strategy Go+TS  │  │ Status  ● clean   │  │ Memory    45MB    ││
│  └─────────────────┘  └───────────────────┘  └───────────────────┘│
│                                                                     │
│  ┌─ DOCTOR ────────────────────────────────────────────────────────┐│
│  │ 🟢 secrets-hygiene        🟢 git-push-gate      🟢 protected-br ││
│  │ 🟢 workspace-safety       🟡 coverage (78%)     🔴 F0-validate  ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ CAPS ──────────────┐  ┌─ WORKTREES ───────────────────────────┐│
│  │ Done    12         │  │ ● feat-kc-compile (active)            ││
│  │ Pending  5         │  │ ○ feat-delegation                    ││
│  │ Progress ████░░ 70%│  │ ○ fix-vault-encrypt                   ││
│  └─────────────────────┘  └──────────────────────────────────────┘│
│                                                                     │
│  [r] Refresh  [d] Doctor Detail  [w] Worktree List  [Esc] Back     │
└─────────────────────────────────────────────────────────────────────┘
```

**Enhancement over current:**
- Runtime metrics: goroutine count, memory usage, GC stats
- Worktree list with active indicator
- Doctor check icons with color coding (not just pass/fail count)
- `d` key → drill into doctor failures

---

#### VIEW: Testing & Coverage (NEW — `testing_view.go`)
**Purpose:** Coverage sprint runner, test suite status, loop detection

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  🧪 Testing & Coverage                         [s] sprint  [r] refresh│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ COVERAGE OVERVIEW ────────────────────────────────────────────┐│
│  │                                                                        │
│  │  validators      ████████████████████░░░░░░░  82%  (+3% today)  ││
│  │  internal/*     ██████████░░░░░░░░░░░░░░░░  51%               ││
│  │  cmd/cockpit    ████████████████████████░░  89%               ││
│  │  cmd/ovav       ██████████████████████████████████████  96%  ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ TEST SUITES ────────┐  ┌─ LOOP DETECTION ─────────────────────┐│
│  │ ✓ unit         142  │  │ No circular dependencies detected     ││
│  │ ✓ integration    38  │  │                                    ││
│  │ ✓ coverage       12  │  │ DAG depth: 7 (healthy)             ││
│  │ ○ e2e            0  │  │                                    ││
│  └─────────────────────┘  └────────────────────────────────────┘│
│                                                                     │
│  [s] Start Coverage Sprint  [t] Run All Tests  [l] Loop Detect  [Esc]│
└─────────────────────────────────────────────────────────────────────┘
```

**Data sources:**
- `go test -cover` output parsed per package
- `go list ./...` for package inventory
- Dependency graph via `go mod graph`

**Keyboard:**
- `s` — start coverage sprint (background, with progress)
- `t` — run all tests (with streaming output)
- `l` — run loop detection on dependencies
- `p` — toggle coverage percent vs absolute numbers

---

#### VIEW: Delegation Runtime (NEW — `delegation_view.go`)
**Purpose:** Visualize active agents, delegation chains, subagent status

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  🎯 Delegation Runtime                         [r] refresh  [d] diag │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ ACTIVE SESSIONS ───────────────────────────────────────────────┐│
│  │                                                                        │
│  │  thavren (you)    ──► andres        [running]  12m  refactor-validators│
│  │       │           ──► clara         [running]   8m  test-coverage       ││
│  │       │                                                                    ││
│  │  eidren            ──► carmen        [idle]     2h  research-cache       ││
│  │       │           ──► fatima        [running]  1h  benchmark-runner    ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ DELEGATION CHAIN ──────────────────────────────────────────────┐│
│  │                                                                        │
│  │  thavren → andres → team-andres                                  ││
│  │          → clara → team-clara                                     ││
│  │                                                                        ││
│  │  eidren  → carmen → team-celia (context-pack)                    ││
│  │          → fatima → team-fatima (benchmark)                      ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  [r] Refresh  [d] Full Diag  [k] Kill Session  [Esc] Back          │
└─────────────────────────────────────────────────────────────────────┘
```

**Data sources:**
- `.ovav/registry/session_ledger.yaml` (active sessions)
- `.ovav/registry/delegation_traces.yaml` (chains)
- Process list for running agent PIDs

---

#### VIEW: Research & Evidence (NEW — `research_view.go`)
**Purpose:** Benchmark results, evidence scores, source verification

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  🔬 Research & Evidence                        [r] refresh  [b] +benchmark│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ BENCHMARKS ────────────────────────────────────────────────────┐│
│  │                                                                        │
│  │  KC-compile (v2)     12ms   ████████████████████  FAST          ││
│  │  Memory-bridge      89ms   ██████████████░░░░░░  MEDIUM         ││
│  │  Context-pack        4ms   █████████████████████  FASTEST        ││
│  │  Ovav-validate      234ms  ██████░░░░░░░░░░░░░░  SLOW           ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ EVIDENCE SCORES ────────┐  ┌─ SOURCE STATUS ──────────────────┐│
│  │ KB-eidren        0.94   │  │ Verified   12 sources             ││
│  │ KB-health        0.87   │  │ Pending     3 sources             ││
│  │ KB-platform      0.91   │  │ Failed      1 source (lab/)       ││
│  └─────────────────────────┘  └───────────────────────────────────┘│
│                                                                     │
│  [b] New Benchmark  [v] Verify Sources  [s] Score Summary  [Esc] Back│
└─────────────────────────────────────────────────────────────────────┘
```

---

#### VIEW: Adversarial View (NEW — `adversarial_view.go`)
**Purpose:** Security gates status, threat model, anomaly detection

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  🛡️ Adversarial & Security                       [r] refresh  [a] audit│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ SECURITY GATES ─────────────────────────────────────────────────┐│
│  │                                                                        │
│  │  🟢 workspace-safety      PASS   Auto-trigger on write           ││
│  │  🟢 git-push-gate         PASS   HTTPS-only, no force push        ││
│  │  🟢 protected-branch      PASS   main/develop/staging protected   ││
│  │  🟢 secrets-hygiene       PASS   No plaintext in tracked files   ││
│  │  🟡 coverage-gate         WARN   78% < 80% threshold              ││
│  │  🟢 F0-validate           PASS   All F0 validators green           ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ THREAT MODEL ────────────┐  ┌─ RECENT AUDITS ──────────────────┐│
│  │ Attack Surface   12 pts   │  │ 2026-08-11  secrets-scan  CLEAN  ││
│  │ Critical          0      │  │ 2026-08-10  dependency  CLEAN    ││
│  │ High             2      │  │ 2026-08-09  push-audit   CLEAN    ││
│  │ Medium           4      │  │ 2026-08-08  scope-risk   2 WARN   ││
│  │ Low              6      │  │                                    ││
│  └─────────────────────────┘  └────────────────────────────────────┘│
│                                                                     │
│  [a] Run Audit  [t] Threat Model  [s] Scan Secrets  [Esc] Back     │
└─────────────────────────────────────────────────────────────────────┘
```

---

#### VIEW: Performance Monitor (NEW — `performance_view.go`)
**Purpose:** Runtime metrics, memory profiling, latency charts

**Layout:**
```
┌─────────────────────────────────────────────────────────────────────┐
│  📈 Performance Monitor                         [r] refresh  [p] profile│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─ RUNTIME METRICS ───────────────────────────────────────────────┐│
│  │                                                                        │
│  │  Goroutines    ████░░░░░░░░░░░░░░░░░  12 / 100 limit             ││
│  │  Heap Alloc    ███░░░░░░░░░░░░░░░░░░  45 MB / 200 MB            ││
│  │  GC Cycles     3 in last hour                                           ││
│  │  CPU Load      ██░░░░░░░░░░░░░░░░░░░  2% avg                     ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  ┌─ LATENCY (last 24h) ────────────────────────────────────────────┐│
│  │  ovav validate   ▁▂▃▂▁▁▂▄▆▇█▇▆▄▂▁▁▂▃▂▁                       ││
│  │  kc-compile      ▁▁▂▁▁▁▂▂▃▂▂▂▃▂▂▁▁▁▂▂▁                        ││
│  │  memory-recall  ▁▂▄▆█▇▆▄▂▁▁▁▂▃▂▁▁▁▁▂▃                        ││
│  │                                                                        ││
│  └────────────────────────────────────────────────────────────────┘│
│                                                                     │
│  [p] Profile Heap  [c] CPU Profile  [g] GC Tune  [Esc] Back      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. UI Component System

### 3.1 Cards

**Standard Card:**
```
┌─ Title ──────────────────────────────────┐
│  Key1   Value1
│  Key2   Value2
└──────────────────────────────────────────┘
```
Style: `PrimaryBorder`, 1px padding, title via `CardHeader`

**Metric Card:**
```
┌─ Metric ─────────────────────────────────┐
│         12
│     items done
└──────────────────────────────────────────┘
```
Large centered number, small label below

**Progress Card:**
```
┌─ Feature ────────────────────────────────┐
│  ████████████░░░░░░░░░  67%   (8/12)    │
│  func_x.go  ████████████████░░  85%     │
│  func_y.go  ██████░░░░░░░░░░░░  35%     │
└──────────────────────────────────────────┘
```

### 3.2 Status Indicators

| Status | Visual | Use case |
|--------|--------|----------|
| `active` | 🟢 green dot + bold | Running, healthy |
| `idle` | ⚪ gray dot | Waiting, no work |
| `warning` | 🟡 yellow dot + bold | Attention needed |
| `error` | 🔴 red dot + bold | Failed, blocked |
| `synced` | ✓ green check | Confirmed |
| `pending` | ◐ yellow half-circle | In progress |

### 3.3 Data Visualization

**Progress Bar (horizontal):**
```
████████████░░░░░░░░░░░░  50%
```
- Fill: `Green` (0-50%), `Yellow` (51-80%), `Red` (81-100%)
- Empty: `Dark` background
- Width: proportional to terminal width (min 10 chars)

**Sparkline (inline):**
```
▁▂▃▄▅▆▇█▇▆▄▂▁▂▃▄▅▆▇
```
- 20 chars wide
- Min/max dots highlighted

**Vertical Bar:**
```
 ┌──┐
 │██│ 89%
 │██│
 │▓▓│
 │▓▓│
 │░░│
 └──┘
```

### 3.4 Navigation Components

**Breadcrumb:**
```
 ← Root / Health & Status / Doctor Detail
```
- Clickable segments
- `←` = back

**Tab Bar:**
```
[ Overview ]  Details  [ Audit Log ]
```
- Active tab: underlined + bold
- Inactive: muted

**Menu Item (root):**
```
▸ Dashboard   📊  Plan Dashboard, caps & milestones    [d]
```
- Selected: primary background
- Icon + label + description + hotkey

### 3.5 Feedback Components

**Toast (bottom):**
```
┌─ ✓ Sync complete ─────────────────────────────────┐
│   3 items pushed to Product                        │
└────────────────────────────────────────────────────┘
```
- Auto-dismiss after 3s
- Types: success (green), error (red), warning (yellow), info (blue)

**Confirm Dialog:**
```
┌─ Confirm ─────────────────────────────────────────┐
│   This will push 3 commits to origin/develop       │
│   Continue?                                        │
│        [Cancel]            [Confirm]               │
└────────────────────────────────────────────────────┘
```

---

## 4. States Reference

### Per-View States

| View | Empty | Loading | Error | Success |
|------|-------|---------|-------|---------|
| Updates | "No pending commits" | Spinner + "Fetching..." | Red banner + error | Green banner |
| Dashboard | "No caps.yaml found" | Skeleton lines | Red banner | (always has data) |
| Health | (always has data) | Spinner | Red border on failed checks | Green border |
| Testing | "No test results" | Progress bar | Red summary | Green summary |
| Delegation | "No active sessions" | Dot animation | Red if agent crashed | Green dot |
| Research | "No benchmarks" | Spinner | Red if source failed | Green score |
| Adversarial | (always has data) | Spinner | Red for critical threats | Green if clean |
| Performance | "No metrics yet" | Real-time updating | Red if OOM | Green metrics |

---

## 5. Color System (extends theme.go)

### Semantic Colors
| Role | Hex | Usage |
|------|-----|-------|
| Primary | `#6366F1` | Brand, active selections, links |
| Success | `#22C55E` | Passed, synced, healthy |
| Warning | `#EAB308` | Attention needed, in-progress |
| Error | `#EF4444` | Failed, blocked, critical |
| Info | `#3B82F6` | Informational, neutral |
| Muted | `#64748B` | Secondary text, labels |

### Terminal Dark Theme (default)
Background: `#0F172A` (Dark)
Surface: `#1E293B` (Darker)
Text: `#E2E8F0` (Bright)
Border: `#334155`

### Contrast Requirements
- All text: minimum 4.5:1 against background
- Large text (18px+): minimum 3:1
- Never use color alone to convey information (always pair with icon or text)

---

## 6. Animations & Transitions

### Principles
- **Instant feedback** — button presses, selections: < 50ms
- **Smooth transitions** — view changes: 150ms fade
- **Progress animations** — loading states: continuous
- **No decorative animation** — every animation serves function

### Animation Specs

**View Transition:**
```go
// Fade out old view (50ms) → clear → fade in new view (100ms)
```

**Progress Bar:**
```go
// Continuous fill animation for indeterminate progress
// Stepped fill for determinate progress (every 100ms)
```

**Toast:**
```go
// Slide up from bottom (100ms ease-out)
// Hold (3s)
// Slide down (100ms ease-in)
```

**Loading Spinner:**
```go
// 8-frame spinner, 50ms per frame
// Characters: "⠋⠙⠹⠸⠼⠴⠦⠧"
```

---

## 7. Keyboard Navigation

### Global Keys (all views)
| Key | Action |
|-----|--------|
| `↑↓` or `k/j` | Navigate list |
| `←→` | Navigate tabs/panels |
| `Enter` | Select / confirm |
| `Esc` | Back / cancel |
| `?` | Toggle help overlay |
| `q` | Quit (from root) |
| `ctrl+c` | Force quit (anywhere) |

### View-Specific Keys

| View | Keys | Action |
|------|------|--------|
| Updates | `s` | Sync all |
| Dashboard | `/` | Toggle search |
| Health | `r` | Refresh |
| Testing | `s` | Sprint, `t` | Tests, `l` | Loop detect |
| Delegation | `k` | Kill session |
| Research | `b` | New benchmark |
| Adversarial | `a` | Run audit |
| Performance | `p` | Profile heap |

---

## 8. Implementation Priority

### Phase 1: Core Enhancement (Priority 1)
1. **Health** — Add goroutine/memory metrics, worktree list, doctor detail drill-down
2. **Dashboard** — Add milestone grouping, dependency view (`d` key)
3. **Updates** — Add commit diff preview panel, sync status
4. **Install** — Add step visualization, progress bars

### Phase 2: Developer Views (Priority 2)
5. **Testing** — Coverage sprint runner with per-package breakdown
6. **Delegation** — Active session visualization, chain display
7. **Research** — Benchmark runner, evidence scores
8. **Adversarial** — Gate status panel, threat model

### Phase 3: Polish (Priority 3)
9. **Performance** — Real-time metrics, latency sparklines
10. **Toast notifications** — Async event feedback
11. **Tab bar** — Multi-panel views (e.g., Updates with diff)
12. **Confirm dialogs** — Destructive action protection

---

## 9. File Structure

```
cmd/cockpit/
├── main.go              # Entry point
├── model.go             # Model struct + constructor
├── update.go            # Update function + message types
├── view.go              # View router
├── nav.go               # NavStack
├── root.go              # Root menu
├── welcome.go           # Welcome screen
├── help.go              # Help overlay
├── quit.go              # Quit confirmation
│
├── dashboard.go         # Plan Dashboard (enhance)
├── health.go            # Health & Status (enhance)
├── updates_view.go      # Work Done & Updates (enhance)
├── vault_view.go        # Vault (existing)
├── install.go           # Install Pipeline (enhance)
├── tailor.go            # Tailor Composer (existing)
├── cli_selector.go      # CLI Runtimes (existing)
├── sync_view.go         # Sync Projection (existing)
├── config_view.go       # Configuration (existing)
├── detail.go            # Plan detail (existing)
│
├── testing_view.go      # NEW: Testing & Coverage
├── delegation_view.go   # NEW: Delegation Runtime
├── research_view.go     # NEW: Research & Evidence
├── adversarial_view.go  # NEW: Adversarial & Security
├── performance_view.go # NEW: Performance Monitor
│
├── progress.go          # Progress components
├── button.go            # Button components
├── toast.go             # Toast notification system
├── dialog.go            # Confirm dialog system
│
├── data/
│   ├── caps.go          # Caps loading
│   └── system.go        # System info gathering
│
└── styles/
    ├── theme.go         # Existing styles
    └── components.go    # NEW: Reusable component styles
```

---

## 10. Open Questions

1. **Real-time data** — Should cockpit poll for health/metrics, or use subscriptions?
2. **External commands** — Coverage sprint, benchmark runner — blocking or async?
3. **Large diffs** — Updates view diff panel: limit lines? Virtual scroll?
4. **Delegation data** — How to get active session data? Via file ledger + polling?
5. **Benchmark storage** — Where are benchmark results persisted? `.ovav/registry/benchmarks/`
