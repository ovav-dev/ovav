# OVAV Git Hooks — Threat Model (Adversarial Perspective)

**Author:** Diana, Security Auditor  
**Date:** 2026-06-19  
**Branch:** `feature/sprint-1.4`  
**Status:** AUDIT — No implementation, detección y recomendaciones

---

## Executive Summary

OVAV's git hooks are the **last mechanical enforcement gate** before commits and pushes reach protected branches. A compromised hook = compromised governance. This threat model identifies 7 attack vectors, ranks them CRITICAL→LOW, and provides mitigations.

### Existing Hook Surface

| Hook | Type | Path | Function |
|---|---|---|---|
| `pre-commit` | bash | `tools/hooks/pre-commit` | Go vet/fmt/test on staged .go files |
| `pre-push` | python3 | `tools/hooks/pre-push` | Blocks direct push to protected branches |
| `post-checkout` | bash | `tools/hooks/post-checkout` | Locks/unlocks filesystem permissions |
| `post-commit` | bash | `.git/hooks/post-commit` (real file) | SBOM regen + integrity baseline |
| `post-merge` | bash | `.git/hooks/post-merge` (real file) | Worktree sync + GC |

### Critical Architectural Finding

The main repo's `.git/hooks/` symlinks point to a **specific worktree path**:
```
post-checkout -> ../../.ovav/worktrees/task-tasknext-work/tools/hooks/post-checkout
```
This is **fragile** — if `task-tasknext-work` worktree is removed, ALL hooks break silently. The `install_hooks.py` script correctly uses `git rev-parse --git-common-dir` to find the shared hooks directory, but the *content* of the hooks still executes scripts from `tools/hooks/` in a **specific worktree**. This creates a single point of failure and a cross-worktree dependency.

---

## 1. Bypass Vectors

### 1.1 `--no-verify` / `-n` Flag

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | `git commit --no-verify` or `git push --no-verify` bypasses ALL local hooks |
| **Detection Difficulty** | Easy server-side (pre-receive), hard locally if detached |
| **Impact** | Complete hook bypass — secrets, force push, anything |

**Current state in OVAV:** The `.pre-commit-config.yaml` hooks (gitleaks, ruff, mypy) run via `pre-commit` framework, which is SEPARATE from git's native hook system. Pre-commit hooks installed via `.pre-commit-config.yaml` are NOT bypassed by `--no-verify` on the git command — but they CAN be bypassed with `SKIP=...` env var or `--no-verify` on `pre-commit run`. The native `.git/hooks/pre-commit` bash script IS bypassed.

**Mitigation:**
1. **Server-side enforcement (GitHub pre-receive hook / CI):** Reject pushes to protected branches that lack required CI check attestations. This is the only reliable defense against `--no-verify`.
2. **Client detection:** When `ovav hook run` is implemented as Go binary, have the `pre-push` hook write a signed attestation to `.git/ovav_hook_attestation` (timestamp + SHA + signature). Server-side validates this attestation exists and is fresh.
3. **CI compliance check:** CI workflow verifies that required hook checks ran. If `--no-verify` was used, the attestation is missing → CI fails.
4. **Git config enforcement:** Set `core.hooksPath` via repo-level config, and make hooks directory read-only to prevent individual developers from changing it.

### 1.2 `core.hooksPath` Manipulation

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | `git config core.hooksPath /dev/null` or `git config core.hooksPath /tmp/evil_hooks` |
| **Detection Difficulty** | Easy (config check), but must be run per-repo |
| **Impact** | Silent hook bypass — hooks never execute |

**Current state in OVAV:** `core.hooksPath` is NOT set. This means hooks run from the default `.git/hooks/`. An attacker or careless developer can redirect hooksPath to an empty directory, bypassing all enforcement.

**Worktree-specific concern:** Each worktree has its own git config. Setting `core.hooksPath` in a worktree's config bypasses hooks for THAT worktree only. However, since hooks in the shared `.git/hooks/` impact the common repository, bypass in one worktree affects validation of changes that may be pushed from another.

**Mitigation:**
1. **Repo-level enforcement:** Add to `.git/config` (commondir) at install time:
   ```
   [core]
       hooksPath = .git/hooks
   ```
   Explicitly set to prevent redirection. Note: repo-level config is NOT propagated to worktrees automatically.
2. **Pre-push binary self-check:** The `ovav hook run` binary should verify at startup that `core.hooksPath` is either unset or points to the expected path. If redirected, abort and alert.
3. **CI check:** CI verifies `core.hooksPath` matches expected value on protected branch pushes.

### 1.3 `.git` File Tampering in Worktrees

| Attribute | Value |
|---|---|
| **Severity** | 🟠 **HIGH** |
| **Attack** | Modify the `.git` file in a worktree to point to a different gitdir with no hooks |
| **Detection Difficulty** | Medium |
| **Impact** | Hooks don't fire; git operations succeed silently |

OVAV worktrees have a `.git` file (not directory) containing:
```
gitdir: /home/braka/Systems/OVAV/.git/worktrees/feature-sprint-1.4
```

If an attacker changes this to point to a directory without hooks, all commits from that worktree bypass enforcement. The common `.git/hooks/` is NOT consulted.

**Mitigation:**
1. **Sanity check in hook scripts:** Each hook should verify `git rev-parse --git-common-dir` returns the expected value before proceeding.
2. **Worktree integrity validator:** A tool that checks `.git` files in all worktrees point to the canonical gitdir.

### 1.4 Direct Manipulation of Main Repo's Index

| Attribute | Value |
|---|---|
| **Severity** | 🟠 **HIGH** |
| **Attack** | Modify `.git/index` directly or use `GIT_INDEX_FILE` env var to stage malicious content |
| **Detection Difficulty** | Hard (requires monitoring index manipulation) |
| **Impact** | Staged files bypass pre-commit checks |

The `pre-commit` hook only checks `git diff --cached`. If index manipulation occurs, content can be committed without triggering filters.

**Mitigation:**
1. **Post-commit verification:** The existing `post-commit` hook already regenerates SBOM and integrity baseline — expand it to verify index integrity.
2. **Remote pre-receive hook:** Server-side SHA verification ensures pushed content matches expected tree.

### 1.5 Environment Variable Injection

| Attribute | Value |
|---|---|
| **Severity** | 🟡 **MEDIUM** |
| **Attack** | `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE` manipulation to redirect git operations |
| **Detection Difficulty** | Medium |
| **Impact** | Git operations target wrong repository |

**Mitigation:**
1. Hook scripts should sanitize environment: unset GIT_DIR, GIT_WORK_TREE if they don't match the canonical path.
2. `ovav hook run` binary can capture environment snapshot for audit.

---

## 2. Persistence — Hook Poisoning

### 2.1 Symlink Replacement Attack

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | Replace `.git/hooks/pre-push` symlink with symlink to malicious script |
| **Detection Difficulty** | Easy with integrity monitoring |
| **Impact** | Malicious code runs on every push — credential exfiltration, silent data modification |

**Current state:** The hooks are symlinks to `tools/hooks/`. The symlink paths are currently worktree-relative (`../../.ovav/worktrees/task-tasknext-work/tools/hooks/pre-push`). If the worktree is removed, symlinks become dangling.

More critically: **anyone with write access to `.git/hooks/` can replace symlinks.** This is not monitored.

### 2.2 Hook Script Content Tampering

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | Modify `tools/hooks/pre-push` directly (add `exit 0` at top) |
| **Detection Difficulty** | Easy with file integrity monitoring |
| **Impact** | Silent bypass of all push protection |

### 2.3 Post-commit/Post-merge Hooks as Real Files

| Attribute | Value |
|---|---|
| **Severity** | 🟡 **MEDIUM** |
| **Attack** | Append malicious code to `.git/hooks/post-commit` or `post-merge` |
| **Detection Difficulty** | Medium (these are real files, not symlinks) |
| **Impact** | Code executes on every commit/merge |

These hooks are real bash scripts in `.git/hooks/`, NOT managed by `install_hooks.py`. They are NOT symlinks. This means they don't benefit from source-of-truth enforcement.

### Detection Mechanisms for Hook Poisoning

1. **File Integrity Monitoring (FIM):**
   - SHA-256 hash of every hook file (both `tools/hooks/` sources AND `.git/hooks/` destinations)
   - Baseline stored in `.ovav/registry/hooks_integrity.json`
   - Checked on every `post-checkout` and `pre-push`
   - Alert + block if drift detected

2. **Symlink validation:**
   - `install_hooks.py --check` already verifies symlinks point to expected targets
   - Expand to run automatically in `post-checkout` hook
   - Add to CI: verify hooks match baseline

3. **Immutable hooks (if supported by filesystem):**
   - `chattr +i .git/hooks/pre-push` (Linux only, requires root)
   - Not practical for development machines

4. **Git-tracked hook signatures:**
   - Track `tools/hooks/*.sha256` in git
   - Hook runner verifies signature before executing

---

## 3. Worktree Isolation

### 3.1 Cross-Worktree Hook Contamination

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | Worktree A modifies `.git/hooks/pre-push` or `tools/hooks/pre-push`, affecting ALL worktrees |
| **Detection Difficulty** | Hard (no isolation) |
| **Impact** | One compromised worktree poisons all |

**Current architecture flaw:** All worktrees share the SAME `.git/hooks/` directory (the common git dir). If any worktree modifies a hook, all worktrees are affected. This is by git design, but OVAV exacerbates it by having the hook scripts live in the worktree's `tools/hooks/` directory.

**Root Cause:** The hooks symlink to `tools/hooks/` within a specific worktree. If worktree A is on a feature branch that modifies `tools/hooks/pre-push`, ALL worktrees immediately run the modified version.

### 3.2 Dangling Symlinks on Worktree Removal

| Attribute | Value |
|---|---|
| **Severity** | 🟠 **HIGH** |
| **Attack** | Removal of `task-tasknext-work` worktree breaks ALL hooks (dangling symlinks) |
| **Detection Difficulty** | Easy (hooks fail silently) |
| **Impact** | Hook enforcement disappears — git fails open |

**Current state:** The main repo `.git/hooks/` points to `../../.ovav/worktrees/task-tasknext-work/tools/hooks/...`. If this worktree is pruned, ALL hooks become dangling symlinks. Git treats non-executable or broken hooks as "skip silently."

### Mitigations for Worktree Isolation

1. **BUNDLE hooks with `ovav` binary:** Instead of symlinks to worktree files, have `.git/hooks/pre-push` be a thin script that calls `ovav hook run pre-push`. The `ovav` binary is installed system-wide (`~/.local/bin/ovav`) and is NOT in any worktree. This eliminates the worktree dependency entirely.

2. **Bootstrap hook on worktree creation:** When `owc` creates a worktree, run `install_hooks.py` from the main repo to ensure hooks use the canonical path, not the new worktree path.

3. **Self-healing post-checkout:** The `post-checkout` hook should verify all hooks are intact. If broken, attempt auto-repair or alert loudly.

4. **Git config isolation:** Use per-worktree git config to set `core.hooksPath` only in worktrees that need custom hooks (currently none).

---

## 4. Supply Chain — Hook Script Integrity

### 4.1 Malicious Hook Script in `tools/hooks/`

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | PR or commit modifies `tools/hooks/pre-push` to add `exit 0` or malicious code |
| **Detection Difficulty** | Medium (code review), Easy (diff monitoring) |
| **Impact** | All developers run compromised hooks |

### 4.2 Go Binary vs Bash Script vs Compiled Entry Point

This is the question about **minimal safe hook script**. Options:

| Approach | Surface | Pros | Cons |
|---|---|---|---|
| **A. Bash thin wrapper** (`#!/bin/bash\novav hook run pre-commit "$@"`) | 2 lines bash | Simple, auditable | Bash itself is an attack surface, PATH manipulation possible |
| **B. Symlink to binary** (symlink `.git/hooks/pre-commit` → `/usr/local/bin/ovav`) | Zero bash | No scripts to tamper with | `ovav` must handle hook protocol (stdin for pre-push), binary can't be signed |
| **C. Compiled entry point** (separate Go binary per hook) | Minimal Go | Type-safe, testable | Larger binary, more maintenance |
| **D. Thin bash calling Go binary** (recommended) | 5 lines bash | Auditable, delegates logic to compiled binary | Bash is minimal but present |

**Recommendation: Approach D — thin bash wrapper**

```bash
#!/usr/bin/env bash
# OVAV pre-commit hook — delegates to ovav binary
# DO NOT MODIFY — integrity verified by SHA-256
set -euo pipefail
exec ovav hook run --stage pre-commit "$@"
```

**Why:**
- The bash wrapper is 5 lines — fully auditable in one glance
- All logic lives in the compiled `ovav hook run` Go binary
- `exec` replaces the bash process, no lingering shell
- The wrapper SHA-256 is stored in `.ovav/registry/hooks_integrity.json` and verified by `post-checkout`
- PATH attack risk: mitigated by using absolute path to `ovav` binary (e.g., `exec ~/.local/bin/ovav`)

### 4.3 `ovav` Binary Integrity

| Attribute | Value |
|---|---|
| **Severity** | 🔴 **CRITICAL** |
| **Attack** | Replace `~/.local/bin/ovav` with malicious binary |
| **Detection** | SHA-256 verification at hook runtime |

**Mitigation:**
1. Hook wrapper verifies binary SHA-256 before exec:
   ```bash
   EXPECTED_SHA="sha256:abc123..."
   ACTUAL_SHA=$(sha256sum ~/.local/bin/ovav | cut -d' ' -f1)
   if [ "$ACTUAL_SHA" != "${EXPECTED_SHA#sha256:}" ]; then
       echo "🛡️ OVAV: Binary integrity FAILED — hook execution blocked" >&2
       exit 1
   fi
   exec ~/.local/bin/ovav hook run --stage pre-commit "$@"
   ```
2. The expected SHA is loaded from `.ovav/registry/hooks_integrity.json` which is git-tracked and signed.
3. CI verifies binary SHA matches expected before accepting pushes.

---

## 5. CI/CD vs Local Execution

### 5.1 Non-TTY Detection Bypass

| Attribute | Value |
|---|---|
| **Severity** | 🟠 **HIGH** |
| **Attack** | Hook behaves differently when `[ -t 0 ]` is false (CI mode), skipping checks |
| **Detection** | Hard (environment-dependent behavior) |
| **Impact** | Local hooks are strict, CI hooks are lenient — attacker targets CI |

### 5.2 Environment Variable Differences

| Attribute | Value |
|---|---|
| **Severity** | 🟡 **MEDIUM** |
| **Attack** | Set `CI=true` or `GITHUB_ACTIONS=true` locally to trigger lenient CI behavior |
| **Detection** | Environment variable inspection |
| **Impact** | Bypass local strict checks |

### 5.3 Recommendation: CI Should Be STRICTER, Not Equal

| Environment | Behavior |
|---|---|
| **Local** | Standard hooks — reasonable performance, skip-heavy operations |
| **CI** | STRICT mode — ALL checks, no skipping, full audit trail, attestation generation |
| **Pre-receive (GitHub)** | MANDATORY — push rejected without signed attestation |

**Rationale:** Local hooks can be bypassed with `--no-verify` (detectable but not preventable). CI and server-side hooks cannot be bypassed. Defense in depth: local hooks catch mistakes early, CI catches bypasses, server-side is the final absolute gate.

**Implementation:**
- `ovav hook run --stage pre-push` detects CI via `OVAV_CI=true` env var (set in CI workflow, not guessable locally)
- In CI mode: run ALL validators, generate signed attestation, fail on ANY warning
- In local mode: run fast checks only, warn on non-critical issues

---

## 6. No-Verify Detection

### 6.1 Attack: `git commit --no-verify && git push`

Flow:
1. Attacker stages malicious code (e.g., secrets in source, force-push commit)
2. `git commit --no-verify` — bypasses pre-commit hooks
3. `git push` — pre-push may still fire, but if commit is "clean" (no Go files changed), it passes

### 6.2 Server-Side Detection

| Method | Reliability | Implementation Effort |
|---|---|---|
| **GitHub pre-receive hook** | ✅ Absolute | Requires GitHub Enterprise or self-hosted |
| **GitHub Actions CI check (required status)** | ✅ High | Configure branch protection rules |
| **Signed attestation in commit** | ✅ Very High | Requires `ovav hook run` to sign commits |
| **Commit message marker** | ❌ Low | Easily forged |

### 6.3 Recommended Implementation

**Step 1: Attestation on verified commits**
When hooks pass, `ovav hook run` writes a signed attestation:
```json
{
  "commit": "abc123def456",
  "timestamp": "2026-06-19T12:00:00Z",
  "hooks_passed": ["pre-commit", "secrets-scan", "secret-detection"],
  "binary_version": "v2.5.0",
  "signature": "sha256:..."
}
```

**Step 2: CI verification**
CI workflow on push to protected branches:
```yaml
- name: Verify hook attestation
  run: ovav hook verify --commit ${{ github.sha }}
```

**Step 3: Pre-receive hook (future)**
GitHub repository ruleset: require status check `hook-attestation` to pass before merge.

### 6.4 Local Detection (Post-hoc)

After a commit, `post-commit` hook (already exists) can check:
1. Did the commit have `--no-verify` flag? (Not directly detectable locally)
2. Is the commit tree missing expected artifacts? (SBOM, integrity hashes)
3. Does the commit message contain bypass markers?

The existing `post-commit` hook already regenerates SBOM and integrity baseline — this provides a partial post-hoc check.

---

## 7. Race Conditions (TOCTOU)

### 7.1 Time-of-Check vs Time-of-Use

```
Time T1: pre-commit hook runs → validates staged files ✅
Time T2: attacker modifies files on disk (between T1 and actual commit)
Time T3: git commit writes the NOW-MODIFIED files to the commit tree
```

| Attribute | Value |
|---|---|
| **Severity** | 🟡 **MEDIUM** |
| **Attack** | Modify files after hook passes but before commit finalizes |
| **Detection** | Very hard (sub-second window) |
| **Impact** | Malicious content committed despite hook passing |
| **Realistic?** | Low — requires sub-second timing or hook execution that pauses |

### 7.2 Pre-Push TOCTOU

```
Time T1: pre-push hook checks commit abc123 → passes ✅
Time T2: attacker force-updates the remote ref to point to different commit xyz789
Time T3: local push sends abc123, but remote behavior depends on timing
```

This is largely mitigated by git's push protocol (atomic push of specific SHA).

### 7.3 Mitigation

1. **Staged content snapshot:** The pre-commit hook should create a temporary index snapshot before validation, and pass THAT to validators. The `ovav hook run` binary can do this atomically.
2. **Index lock:** Git already locks `.git/index` during operations. Ensure validators run while index is locked.
3. **Post-commit re-verification:** The `post-commit` hook re-verifies the committed tree against the same checks (SBOM, secrets scan on full tree).

### How Other Systems Handle This

- **pre-commit framework:** Doesn't address TOCTOU — validates staged files at hook time only.
- **Gitleaks:** Runs on `git diff --cached`, same window.
- **Google's "copybara":** Server-side verification only — no local hook dependency.
- **GitHub push protection:** Scans pushed content on server side — immune to local TOCTOU.

**OVAV recommendation:** Server-side (GitHub/CI) must re-scan pushed content. Local hooks are a convenience layer, not the authoritative gate.

---

## Attack Vector Severity Matrix

| # | Vector | Severity | Detectable | Exploitable | Impact |
|---|---|---|---|---|---|
| 1 | `--no-verify` bypass | 🔴 CRITICAL | ✅ Server | ✅ Trivial | Full bypass |
| 2 | `core.hooksPath` redirection | 🔴 CRITICAL | ✅ Config check | ✅ Easy | Silent bypass |
| 3 | Symlink replacement poisoning | 🔴 CRITICAL | ✅ FIM | ✅ Easy | Full compromise |
| 4 | Hook script content tampering | 🔴 CRITICAL | ✅ FIM | ✅ Easy | Full compromise |
| 5 | Cross-worktree contamination | 🔴 CRITICAL | ✅ FIM | ✅ Medium | All worktrees poisoned |
| 6 | Binary replacement (`ovav`) | 🔴 CRITICAL | ✅ SHA check | ✅ Medium | All hooks compromised |
| 7 | Supply chain (malicious PR to hooks/) | 🔴 CRITICAL | ✅ Code review | ✅ Medium | Widespread compromise |
| 8 | `.git` file tampering | 🟠 HIGH | ✅ Validator | ✅ Easy | Worktree isolation |
| 9 | Dangling symlinks (worktree removal) | 🟠 HIGH | ✅ Auto-check | ✅ Accidental | Silent failure |
| 10 | CI vs local behavior differential | 🟠 HIGH | ⚠️ Context-dependent | ✅ Easy | Bypass via CI env |
| 11 | Direct index manipulation | 🟠 HIGH | ⚠️ Hard | ✅ Medium | Staged content bypass |
| 12 | Non-OVAV hooks (post-commit, post-merge) | 🟡 MEDIUM | ✅ FIM | ✅ Easy | Unmanaged surface |
| 13 | Env variable injection (GIT_DIR, etc.) | 🟡 MEDIUM | ✅ Sanitize | ✅ Easy | Path confusion |
| 14 | TOCTOU race condition | 🟡 MEDIUM | ❌ Very hard | ⚠️ Unreliable | Partial bypass |
| 15 | PATH manipulation in bash hooks | 🟡 MEDIUM | ✅ Explicit PATH | ✅ Easy | Binary hijacking |

---

## Safe Implementation Checklist

### Phase 0: Architecture Decision (BEFORE any code)

- [ ] **0.1** Decide hook entry point architecture → **recommended: Thin bash wrapper + `ovav hook run` Go binary**
- [ ] **0.2** Define attestation format (JSON, signed with SHA-256 + HMAC)
- [ ] **0.3** Define CI vs local behavior contract:
  - Local: fast checks, warn on bypass, generate attestation
  - CI: ALL checks, block on ANY issue, verify attestation chain
- [ ] **0.4** Audit all existing hook scripts for supply chain risk prior to migration

### Phase 1: Hook Hardening

- [ ] **1.1** Replace symlink-based hooks with thin bash wrappers:
  ```bash
  #!/usr/bin/env bash
  # OVAV <hook-name> — delegates to ovav binary
  # SHA-256: <integrity-hash>
  set -euo pipefail
  OVAV_BIN="${OVAV_BIN:-$HOME/.local/bin/ovav}"
  exec "$OVAV_BIN" hook run --stage <hook-name> "$@"
  ```
- [ ] **1.2** Remove worktree-relative symlinks — all hooks point to system `ovav` binary
- [ ] **1.3** Add binary integrity check (SHA-256 verification before `exec`)
- [ ] **1.4** `chmod 755` on wrapper, `chmod 755` on binary, hooks directory `chmod 755`

### Phase 2: Hook Integrity Monitoring

- [ ] **2.1** Create `.ovav/registry/hooks_integrity.json`:
  ```json
  {
    "version": "1.0",
    "baseline_commit": "abc123",
    "hooks": {
      "pre-commit": {
        "sha256": "def456...",
        "binary_sha256": "ghi789...",
        "path": ".git/hooks/pre-commit",
        "source": "tools/hooks/thin_wrappers/pre-commit"
      }
    }
  }
  ```
- [ ] **2.2** `post-checkout` hook verifies all hook integrity against baseline
- [ ] **2.3** `ovav hook verify` command for manual integrity check
- [ ] **2.4** CI workflow verifies hook integrity on every push to protected branches
- [ ] **2.5** Alert mechanism: if drift detected, block ALL writes until resolved

### Phase 3: No-Verify Detection (Server-Side)

- [ ] **3.1** Implement attestation generation in `ovav hook run`:
  - On hook pass → write `.git/ovav_hook_attestation` (JSON + HMAC)
  - On `--no-verify` → NO attestation written
- [ ] **3.2** CI workflow: `ovav hook verify --commit $SHA` → block if no attestation
- [ ] **3.3** GitHub branch protection: require `hook-attestation` status check
- [ ] **3.4** Audit log: log all attestation verifications to `.ovav/runtime/hook_audit.log`

### Phase 4: Worktree Safety

- [ ] **4.1** `post-checkout` hook verifies `.git` file points to valid gitdir
- [ ] **4.2** `owc` (worktree create) runs `ovav hook install --verify` after creation
- [ ] **4.3** `owd` (worktree done) ensures hooks aren't sourced from removed worktree
- [ ] **4.4** Worktree integrity check: verify no cross-worktree hook contamination
- [ ] **4.5** `core.hooksPath` is explicitly set to expected path in repo config

### Phase 5: CI Integration

- [ ] **5.1** CI workflow: `OVAV_CI=true ovav hook run --stage pre-commit --strict`
- [ ] **5.2** CI workflow: `OVAV_CI=true ovav hook run --stage pre-push --strict`
- [ ] **5.3** CI is ALWAYS `--strict` mode — no warnings, all errors block
- [ ] **5.4** CI attestation is signed with CI-specific key (not local key)
- [ ] **5.5** CI logs all hook outputs as workflow artifacts

### Phase 6: Audit & Monitoring

- [ ] **6.1** `ovav hook audit` command — shows last N hook executions with results
- [ ] **6.2** Failed hook attempts are logged with timestamp, user, and failure reason
- [ ] **6.3** Bypass attempts (detected via missing attestation) trigger alert
- [ ] **6.4** Hook execution metrics: duration, success rate, failure patterns
- [ ] **6.5** Security violation YAML updated on hook tampering detection

### Phase 7: Supply Chain Lockdown

- [ ] **7.1** `tools/hooks/` directory reviewed for least-privilege permissions
- [ ] **7.2** All hook sources tracked in SBOM
- [ ] **7.3** Hook wrappers managed via install script, not editable manually
- [ ] **7.4** Non-OVAV hooks (post-commit, post-merge) migrate to OVAV-managed wrappers or documented exception list
- [ ] **7.5** No external dependencies in hook wrappers — bash + `ovav` binary only

---

## Detection Mechanisms Summary

| Mechanism | What It Detects | Where | Frequency |
|---|---|---|---|
| **Hook SHA-256 verification** | Content tampering | `post-checkout`, `ovav hook verify` | Every checkout + on-demand |
| **Binary SHA-256 verification** | `ovav` binary replacement | Hook wrapper (before exec) | Every hook invocation |
| **Attestation presence check** | `--no-verify` bypass | CI + server-side | Every push to protected |
| **Symlink target validation** | Symlink hijacking | `install_hooks.py --check` | On-demand + CI |
| **`core.hooksPath` check** | Config redirection | Hook startup + CI | Every hook invocation |
| **`.git` file validation** | Worktree detachment | `post-checkout` | Every checkout |
| **Worktree integrity scan** | Cross-contamination | `owc`, `owd` | Worktree create/destroy |
| **CI strict mode** | CI vs local bypass | CI workflow | Every CI run |
| **Audit log** | All hook activity | `.ovav/runtime/hook_audit.log` | Continuous |

---

## Recommendations (Executive)

1. **Immediate (this sprint):** Replace worktree-relative symlinks with thin bash wrappers calling `ovav hook run`. This eliminates the most critical architectural flaw.

2. **Short-term (next sprint):** Implement `ovav hook run` Go binary with attestation generation and integrity self-check.

3. **Medium-term:** Server-side attestation verification via CI + GitHub branch protection rules.

4. **Continuous:** Hook integrity monitoring via `post-checkout` + scheduled CI check. Any drift = immediate alert + block.

---

*OVAV Governance — This threat model is a living document. Update as hooks evolve. Diana (Security Auditor) is the canonical authorizer for security findings herein.*
