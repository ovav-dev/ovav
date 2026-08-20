# OVAV Auth Reconstruction — 2026-08-19

**Status:** partial. Local + Web wrapper scripts shipped under `bin/auth/`. Full source refactor deferred until Go toolchain is available in the environment.

## TL;DR — what the user runs now

```bash
# One-time install (from repo root)
bash bin/auth/ovav-auth-install           # copies to ~/.local/bin
bash bin/auth/ovav-auth-install --symlink # or symlinks

# Daily use
ovav-local                                 # local login (offline, machine-bound)
ovav-web                                   # web login (browser-based OAuth)
ovav-web --check                           # preflight probe — fails fast if backend broken
ovav-stale-lock-purge                      # clear dead locks (run before --recover-ceo)
```

## Why the original design accumulated bugs

### Bug surface in current `cmd/ovav/login.go`

| Bug | Code reference | Root cause |
|---|---|---|
| **Auto-export of seed to disk** | `cmdLogin()` lines ~200, 306 → `exportVaultKey(vaultKey, seed)` | Called unconditionally on every successful login. No env-var gate. No opt-out. |
| **Stale lock with no auto-cleanup** | `internal/identity/recovery.go:770-777` | Lock check is just `os.Lstat(path)` — if file exists, "locked". No PID liveness check. |
| **Web backend returns HTML on 302** | `cmdLoginWeb()` line 445 → `https://d678beea.ovav.dev/api/v1/auth/login-challenge-web` | Backend misconfigured (Cloudflare 302 redirect). JSON parser then fails with `invalid character '<'`. |
| **Three flows merged into one dispatcher** | `cmdLogin(args)` lines 107-130 | `--force`, `--web`, `--recover-ceo`, default all branch from one function. Hidden coupling. |
| **TTY required for recovery** | `cmdRecoverCEO()` line 341 → `term.ReadPassword(int(syscall.Stdin))` | No env-var fallback. No `--seed-file` flag. |
| **Identity binding drift** | `vault.key` (disk) ↔ `identities.yaml` (registry) ↔ `whoami` (cache) | Multiple recoveries left 3 different hashes in different layers. No canonical-version-of-truth logic. |

### Architectural anti-patterns

1. **No separation of concerns**: identity binding + vault unlock + web auth + session creation in one binary
2. **Secrets persisted by default** (mode 600 ≠ "no persistence")
3. **Lock files without liveness detection**
4. **No preflight checks** for web auth dependency
5. **No env-var opt-outs** for anything

## Proposed rebuild (Go source)

### New package layout: `cmd/ovav/auth/`

```
cmd/ovav/auth/
  local.go        # offline seed-based login
  web.go          # browser-based OAuth login
  status.go       # show both states
  signout.go      # clear both
  common.go       # shared identity-resolution logic
  recovery.go     # CEO machine-bound recovery (replaces cmdRecoverCEO)
  preflight.go    # network probes, lock cleanup, secret shredders
```

### Target CLI surface

```
ovav auth local                 # offline seed-based
ovav auth local --seed-file PATH
ovav auth local --rotate        # generate new seed, re-bind identity
ovav auth local --bind NAME     # bind this device with friendly name
ovav auth web                   # browser-based via ovav.dev
ovav auth web --check           # preflight only (HTTP 200 + JSON schema)
ovav auth web --device NAME     # register device
ovav auth status                # show local + web session
ovav auth signout               # clear both (vault, registry, cache)
```

### Design rules (must hold after refactor)

- **R-1**: Auto-export of seed is **opt-in only** (`--persist` flag). Default OFF.
- **R-2**: All locks have TTL + PID liveness. Stale detection is automatic.
- **R-3**: `auth web` has mandatory preflight probe (`HEAD /api/v1/health`). Fails fast if backend unreachable.
- **R-4**: Seed input modes: TTY prompt (default) / `--seed-file <path>` / `SEED` env var. All validated for entropy.
- **R-5**: `whoami` reads from SINGLE canonical source — `vault.key` + `identities.yaml` are diffed and reconciled at session start; mismatch → user prompted.
- **R-6**: Recovery binding is HMAC-signed with new key_hash AND invalidated old key_hash in same atomic commit.
- **R-7**: No command paths share state machines. Failure of `auth web` cannot affect `auth local`.

### Sample target: `auth/local.go` (sketch)

```go
//go:build !windows

package auth

func CmdLocal(args []string) int {
    opts, err := parseLocalOptions(args)
    if err != nil { return help(localUsage) }

    // Preflight: stale-lock purge
    if err := purgeStaleLocks(opts.RepoRoot); err != nil {
        fmt.Fprintf(os.Stderr, "❌ Lock blocked: %v\n", err)
        return 1
    }

    // Acquire seed (TTY | --seed-file | $SEED)
    seed, err := readSeed(opts)
    if err != nil { return die("cannot read seed: %v", err) }
    defer zeroOut(seed)

    // Derive vault_key
    machineID, _ := license.MachineID()
    vaultKey := pbkdf2.Key(append([]byte(seed), []byte(machineID)...), salt, 600_000, 32, sha256.New)

    // Resolve identity
    identity, err := identity.FindByKeyHash(sha256Hex(vaultKey))
    if err != nil { return die("identity resolution: %v", err) }

    // Persist session (NOT the seed)
    writeSession(vaultKey, identity, machineID)

    // R-1: never export seed unless --persist
    if !opts.Persist {
        _ = secureRemove(filepath.Join(opts.ShareDir, "seed_export"))
        _ = secureRemove(filepath.Join(opts.ShareDir, "vault_key_export"))
    }

    fmt.Println("🟢", identity.Welcome())
    return 0
}
```

### Sample target: `auth/web.go` (sketch)

```go
func CmdWeb(args []string) int {
    backendURL := os.Getenv("OVAV_WEB_URL")
    if backendURL == "" { backendURL = "https://ovav.dev" }

    // R-3: mandatory preflight
    if err := preflightProbe(backendURL); err != nil {
        return die("web backend unreachable: %v (refuse to launch interactive flow)", err)
    }

    challenge := getChallenge(backendURL)
    openBrowser(fmt.Sprintf("%s/oauth/start?code=%s", backendURL, challenge))

    jwt := pollForJWT(backendURL, challenge, 90*time.Second)
    if jwt == "" { return die("login timed out") }

    storeJWTInVault(jwt)
    fmt.Println("🟢 web session ready:", maskJWT(jwt))
    return 0
}

func preflightProbe(u string) error {
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(u + "/api/v1/health")
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 { return fmt.Errorf("HTTP %d", resp.StatusCode) }
    // verify JSON contract
    var h struct{ Status string ` + "`json:\"status\"`" + ` }
    return json.NewDecoder(resp.Body).Decode(&h)
}
```

## Why the wrapper scripts exist (interim)

The Go binary at `~/.local/bin/ovav` cannot be re-built in this environment (no Go toolchain in WSL — only the prebuilt bin from May). The wrapper scripts in `bin/auth/` provide:

1. **Local login** with auto-shred of the seed_export and vault_key_export (the security gap)
2. **Web login** with mandatory preflight that fails fast (no more "invalid character")
3. **Stale-lock cleanup** that runs before any recovery
4. **One-shot install** that lands them in `~/.local/bin`

The wrappers do NOT change the binary's behavior beyond what the binary already does. They intercept the side-effects.

## Files created this session

```
bin/auth/ovav-local                # local login wrapper (auto-shred)
bin/auth/ovav-web                  # web login wrapper (preflight + auto-shred)
bin/auth/ovav-stale-lock-purge     # dead-PID lock cleanup
bin/auth/ovav-auth-install         # one-shot installer (copy or symlink)
```

## Follow-up work (deferred until Go toolchain available)

1. Implement `cmd/ovav/auth/{local,web,status,signout}.go`
2. Add `preflight.go` with shared probe helpers
3. Add env-var gating for seed export (R-1)
4. Replace lock-`os.Lstat` with PID-liveness check (R-2)
5. Deprecate `cmd/ovav/login.go` → re-route to `auth.CmdLocal` / `auth.CmdWeb`
6. Rebuild ovav binary, run `make install`
7. Add tests: `auth/local_test.go`, `auth/web_test.go`, `auth/preflight_test.go`

## Acceptance criteria for the rebuild

- [ ] All current `ovav login` invocations continue to work (back-compat)
- [ ] New `ovav auth local|web` commands available
- [ ] No path that writes seed without `--persist`
- [ ] Stale locks auto-clear on next login
- [ ] Web preflight fails fast with clear error
- [ ] Smoke tests for all subcommands pass
- [ ] Coverage > 80% on `cmd/ovav/auth/`

---

## Update 2026-08-19 — YOLO 2026 login deactivation

### Status: applied (PR-equivalent: commit pending)

Three login entry points were disabled by default in the YOLO 2026 baseline:

| Command | Default behavior | Bypass |
|---|---|---|
| `ovav login` | exits 78 (EX_CONFIG), prints banner, redirects to `ovav waiver` | `--force` or `OVAV_AUTH_LOGIN_ENABLED=1` |
| `ovav auth local` | exits 78, banner, redirect | `--force` / env |
| `ovav auth web` | exits 78, banner, redirect | `--force` / env |

### Why

1. **R-1 risk**: every successful `ovav login` writes `seed_export` and `vault_key_export` to `~/.local/share/ovav/` before the auth package can shred them (CRIT-014 — plaintext window).
2. **R-3 broken**: `ovav auth web` always fails at preflight because Cloudflare Access blocks `/api/v1/auth/*` with a 302 to `ovav.cloudflareaccess.com`. The web backend would need a server-side fix to function.
3. **Canonical alternative**: `ovav waiver permission-ceo` is already the operational auth surface — login was dead code in this environment.

### Implementation

- New gate helpers in `go-runtime/cmd/ovav/auth/preflight.go`:
  - `LoginDisabled()` — default true (YOLO 2026)
  - `HasForceArg(args)` — detects `--force` / `--enable-login`
  - `CheckLoginAllowed(args)` — single guard, prints banner, returns decision
  - `ExitConfigDisabled = 78` — sysexits `EX_CONFIG`
- All three command entry points call `CheckLoginAllowed` as their first action.
- Help text in each command documents the new default and bypass.
- Imports: `cmd/ovav/login.go` gained `"github.com/ovav/ovav/cmd/ovav/auth"` for the gate symbol.

### Future work (deferred)

- [ ] Move seed-export logic OUT of `cmdLogin` entirely (R-1 permanent fix).
- [ ] Replace `ovav auth web` server-side block (R-3).
- [ ] Add `caps.yaml` `auth.login_command_enabled` capability flag (currently env-based; canonical source-of-truth still missing).
- [ ] Coverage test for `CheckLoginAllowed` (env=false, env=true, --force, --enable-login combinations).
