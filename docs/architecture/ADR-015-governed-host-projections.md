# ADR-015: Governed Host Projections

**Date:** 2026-08-29  
**Status:** Accepted  
**Related:** ADR-014, APP-3993  
**Decider:** Thavren + CEO

## Context and evidence

APP-3993 recorded repeated operation timeouts alongside
`Wsl/Service/E_UNEXPECTED`. The affected workstation has 8 GB of physical RAM,
while WSL was configured with `memory=8GB`. That allocation left no practical
headroom for Windows, Warp, or concurrent development processes and correlated
with the observed failure window.

Microsoft's 2026 WSL documentation identifies `.wslconfig` in Windows
`%UserProfile%` as the supported global WSL 2 VM configuration surface. It
officially supports `memory`, `processors`, `swap`, `networkingMode=mirrored`,
`dnsTunneling`, `autoProxy`, `firewall`, and experimental
`autoMemoryReclaim=dropCache`. The supported corrective profile therefore caps
WSL at 4 GB RAM and four processors, provisions 4 GB swap, enables supported
mirrored networking protections, and reclaims cache.

Microsoft also states that changes activate only after the subsystem fully
stops. OVAV waits for a **natural full WSL stop** after all sessions close. It
must never automatically invoke `wsl --shutdown` or `wsl --terminate`, because
either can interrupt unrelated distributions or active work.

Warp's official Tab Config schema stores Windows configs under
`%APPDATA%\warp\Warp\data\tab_configs\`. The OVAV profile contains one terminal
pane rooted at `~`, with no startup command and no `shell` override. It relies
on the Ubuntu-26.04 default shell selected by the user in Warp's UI.

## Decision

Host state is projected only through three exact source-controlled profiles:

| Profile | Canonical source | Exact destination authority |
|---|---|---|
| `opencode-bootstrap` | `ops/host-projections/opencode-bootstrap.json` | `$HOME/.config/opencode/opencode.json` below `$HOME/.config/opencode` |
| `wsl2-resource-policy` | `ops/host-projections/wsl2/.wslconfig` | `<windows-home>/.wslconfig` below exactly `<windows-home>` |
| `warp-wsl-tab` | `ops/host-projections/warp/ovav_wsl.toml` | `<windows-home>/AppData/Roaming/warp/Warp/data/tab_configs/ovav_wsl.toml` below exactly that `tab_configs` directory |

There is no arbitrary source, destination, or allowed-root interface. Sources
must resolve from the repository root and pass profile-specific strict content
validation. OpenCode bootstrap JSON is schema-only; it carries no agents,
providers, or permissions.

## Transaction gates

1. `ovav sync --host-profile <name>` is plan-only and performs no writes.
2. Apply requires both `--apply` and `--approve-host-write`.
3. Rollback requires an absolute governed journal path and
   `--approve-host-write`.
4. Apply authority includes the exact profile, source, destination, allowed
   root, backup root, and migration epoch where present. Recovery authority is
   journal-version-aware as described below.
5. `internal/hostprojection` enforces no-follow path traversal, locking,
   backup, journaling, atomic replacement, verification, and recovery.
6. Journals and backups live under
   `$HOME/.local/state/ovav/host-projection`, created privately as `0700` with
   private `0600` artifacts.

### Exact OpenCode symlink migration

The existing OpenCode destination may be a direct symlink to the canonical
main repository `opencode.json`. Keeping that mutable repository file as the
long-term host configuration authority would let unrelated source edits change
host behavior without a governed host-sync transaction. `hostprojection`
therefore exposes an explicit exact-symlink migration option; ordinary plans
continue to reject symlinks.

Only `opencode-bootstrap` enables this option. `hostsync` derives the canonical
main repository root from the supplied repo/worktree `.git` metadata through
no-follow descriptors and supplies exactly `<main-root>/opencode.json` as the
sole permitted absolute link target. Relative links, target mismatches,
symlinked target components, symlink targets, and directory targets fail
closed. WSL and Warp profiles never receive this option.

The profile owns one exact migration epoch,
`opencode-bootstrap-symlink-v1`. A successful symlink migration atomically
publishes a deterministic private `0600` consumed marker inside the governed
backup root. Its name, SHA-256, and inode identity are journaled. A valid marker
blocks every later plan or apply that encounters a symlink for that epoch;
regular-destination reapply remains allowed. Marker inspection and mutation are
no-follow, owner/link checked, and race-safe under the backup-root descriptor.

Planning records the destination symlink inode and exact link text but never
uses target content as original-content or backup authority. Apply revalidates
that identity and text under the destination parent descriptor, atomically
exchanges a staged regular schema-only bootstrap over the link, and validates
the exchanged link before discarding it. The canonical repository target is
never opened for mutation. The journal records the original destination kind,
link text, and expected target authority.

Rollback accepts only the journaled applied regular-file identity. It creates
a temporary symlink with the exact original text via `symlinkat`, validates it,
and atomically exchanges it with the applied file. The displaced file identity
is checked before removal; a concurrent replacement is exchanged back and the
rollback fails closed. Recovery recognizes both sides of this journaled
exchange and is idempotent. Before restoring the symlink, rollback atomically
quarantines, verifies, and removes only that transaction's exact consumed
marker. Successful rollback therefore deliberately reopens the same migration
epoch; a missing, replaced, linked, or raced transaction marker fails closed.

### Journal trust boundary

`internal/hostprojection` is the only journal inspection authority. It opens
the owner-private backup root and journal through no-follow descriptors,
requires a regular `0600` journal owned by the effective user with exactly one
hard link, bounds JSON size, and issues a digest/inode inspection token.
Recovery reacquires the journal lock, reopens the journal, and rejects any
digest or identity change before decoding or using its authority. `hostsync`
then matches that trusted authority against an exact registry profile and
supplies the expected roots back to inspected recovery; journal content never
selects recovery roots by itself.

New journals use schema v2 and record `profile_id` plus `migration_id`. V2
rollback matches the trusted journal against one exact registered profile by
destination, allowed root, backup root, profile ID, and migration ID; it does
not require the original absolute source worktree to survive. Existing v1
regular-file journals remain recoverable. Legacy matching requires a unique
exact destination/root match and a clean journaled source suffix equal to the
registered `SourceRelative`, without reading or requiring that source path.
V1 rejects v2 profile, migration, symlink, marker, and marker-state fields.

For OpenCode symlink migration, the expected canonical target is also part of
the journal authority matched back to the profile. Trust is limited to the
no-follow `.git` relationship, exact absolute link text, unchanged symlink
identity, and a regular target reached without symlink components. The target's
contents are intentionally outside transaction authority and remain untouched.

This boundary protects against path substitution and unauthorized filesystem
actors through owner-private roots and artifacts. It does **not** claim
protection from an attacker who has fully compromised the same UID and can
modify private files or ignore advisory locks.

Every operation returns structured JSON-ready evidence including mode,
resolved authority, durability, write status, and journal path.

## Rollback

Rollback accepts only a direct child journal of the governed backup root. The
journal must identify one current allowlisted profile exactly. Recovery then
restores the verified backup or removes a newly created destination. Concurrent
destination changes fail closed rather than being overwritten.

## Consequences

- Host writes are explicit, bounded, inspectable, reversible, and disabled by
  default.
- Existing repository sync behavior remains unchanged.
- Destination parent directories must already exist; this unit does not widen
  generic install or bootstrap surfaces.
- WSL changes remain dormant until the next natural full subsystem stop.

## References

- Microsoft Learn, **Advanced settings configuration in WSL**, updated
  2026-06-02: <https://learn.microsoft.com/en-us/windows/wsl/wsl-config>
- Warp, **Tab Configs**, updated 2026-08-27:
  <https://docs.warp.dev/terminal/windows/tab-configs/>
- OpenCode configuration schema: <https://opencode.ai/config.json>
