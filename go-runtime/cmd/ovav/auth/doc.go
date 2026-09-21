// Package auth implements OVAV identity authentication with two
// distinct, non-overlapping flows:
//
//	local.go — offline seed-based login, machine-bound
//	web.go   — browser-based OAuth login via https://ovav.dev
//
// Design rules (see .ovav/plan/auth-reconstruction.md for the full
// source of truth):
//
//	R-1  Auto-export of seed to disk is OPT-IN only (--persist flag).
//	     Default behavior: never write seed_export or vault_key_export.
//	R-2  Lock files include a PID and TTL. Stale locks (dead PID)
//	     are purged automatically before any auth attempt.
//	R-3  Web login has a mandatory preflight HTTP probe; the
//	     interactive flow refuses to launch on broken backend.
//	R-4  Seed input modes: TTY prompt (default) | --seed-file <path>
//	     | SEED env var. All validated for entropy.
//	R-5  whoami reads from SINGLE canonical source: vault.key +
//	     identities.yaml; mismatch is reconciled at session start.
//
// Nothing in this package writes a plaintext seed to disk by default.
// The legacy login.go's `exportVaultKey` is preserved for back-compat
// (called via `ovav login` only), but the new commands intentionally
// do NOT call it.
package auth
