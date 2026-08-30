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
4. Profile, source, destination, allowed root, and backup root must match exact
   registry authority.
5. `internal/hostprojection` enforces no-follow path traversal, locking,
   backup, journaling, atomic replacement, verification, and recovery.
6. Journals and backups live under
   `$HOME/.local/state/ovav/host-projection`, created privately as `0700` with
   private `0600` artifacts.

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
