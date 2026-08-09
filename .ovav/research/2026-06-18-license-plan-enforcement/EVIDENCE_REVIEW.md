# EVIDENCE REVIEW — License & Plan Enforcement

## Topic 1: License Management in TUI Tools

### Pattern 1: OAuth Device Flow (RFC 8628)
**Users:** GitHub CLI, Terraform CLI, Supabase CLI, PlanetScale CLI
**Flow:** CLI prints device code + URL → user opens browser → user enters code → CLI polls for completion → receives access token
**Pros:** No secrets in CLI, short-lived tokens, standard protocol, supports 2FA
**Cons:** Requires OAuth server, internet required for initial auth, token refresh needed
**OVAV applicability:** Medium — good for web/TUI sync but adds server dependency

### Pattern 2: API Key from Dashboard
**Users:** Linear CLI, Vercel CLI (`--with-token`), most SaaS CLIs
**Flow:** User copies key from web dashboard → runs `tool auth --key <key>` → key stored locally
**Pros:** Simplest to implement, no OAuth server, works immediately
**Cons:** Long-lived secret on disk, no automatic rotation, key theft = full access
**OVAV applicability:** Low for license — API keys ≠ license keys

### Pattern 3: HMAC-Signed License Keys + Machine Binding
**Users:** JetBrains, IntelliJ, Sublime, OVAV (existing)
**Flow:** License key issued at purchase → entered in TUI → HMAC verified → PBKDF2 binds to machine ID → vault key derived
**Pros:** Offline-capable after bind, anti-forgery (HMAC), machine-specific
**Cons:** Machine re-imaging invalidates binding, key sharing via copy-paste
**OVAV applicability:** Already implemented — `internal/license/bind.go`. This is the right pattern.

### Pattern 4: Account Login (Email/SSO)
**Users:** Warp, Zed, Cursor
**Flow:** User logs in with email/Google/GitHub → TUI stores session token → periodic re-auth
**Pros:** User-friendly, ties to account, cross-device sync
**Cons:** Requires auth server, internet for login, session management
**OVAV applicability:** Complementary — could layer on top of license key for membership sync

### Anti-Tampering Mechanisms in Go
1. **Binary obfuscation** — `garble` tool: renames symbols, shuffles literals. Most practical.
2. **Stripped symbols** — `-ldflags="-s -w"`: removes DWARF debug info, 30% smaller binary, harder to reverse.
3. **Checksum self-verification** — Binary computes own SHA256 at startup, compares to embedded hash.
4. **`go:embed` license logic** — Embed critical verification code that can't be patched at runtime.
5. **Binary signing** — macOS notarization, Windows Authenticode. Prevents tampered binary execution.

## Topic 2: Plan/Document Enforcement Systems

### Pattern 1: Stacked Diffs (Graphite)
**Mechanism:** Branches form explicit parent→child chain. `gt stack submit` submits entire chain as PRs. Automatic rebasing keeps stack coherent.
**Enforcement:** Tool-level — Graphite CLI manages the stack. No git-level enforcement, but workflow is impossible without it.
**Key insight:** The tool *orchestrates* the plan rather than *blocking* deviance. More carrot than stick.

### Pattern 2: Branch Naming Convention + Hooks (Linear + GitHub)
**Mechanism:** Branch `TEAM-123-description` auto-links to Linear issue. Pre-push hook verifies branch matches open issue. GitHub branch protection can require linked issues.
**Enforcement:** Git hooks (pre-commit, commit-msg, pre-push). Reject commits not matching plan.
**Key insight:** Lightweight, git-native, no external tool needed.

### Pattern 3: caps.yaml as Canonical Plan (OVAV existing)
**Mechanism:** Single YAML file defines pending work with `id`, `deps`, `order`, `worktree`. Cockpit reads and displays. OWS (`owc`) creates worktrees from plan.
**Enforcement gap:** Cockpit *displays* the plan but doesn't *enforce* it. `owc` creates worktrees but doesn't check if the work matches plan order.

### Pattern 4: Conventional Commits
**Mechanism:** `type(scope): message` format enforced via commit-msg hook. Maps to plan items (e.g., `feat(cockpit): license activation screen`).
**Key insight:** Low-cost, high-value. Machine-parseable. Industry standard.

### Pattern 5: Git Hook Enforcement
**Mechanism:** Pre-commit, commit-msg, pre-push hooks in `.ovav/hooks/` checked into repo. Can validate:
- Branch matches an active plan item
- Worktree corresponds to declared work
- Commit message follows conventional format
- No work outside declared plan on protected branches
**Key insight:** The only true "git-level enforcement" mechanism available without custom git server.

## What to Avoid
1. ❌ Server-side enforcement for local dev — kills offline flow, adds latency
2. ❌ Complex DAG enforcement that blocks all forward progress — developer-hostile
3. ❌ Binary DRM beyond HMAC + machine binding — user-hostile, easy to crack
4. ❌ Phone-home on every operation — privacy issue, offline-breaking
5. ❌ Forcing exact task ordering when deps allow parallelism — reduces throughput
