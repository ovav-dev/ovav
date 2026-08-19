# P5 — Attack Plan (Execution Profiles)

## Status

P5 status: PARTIAL. Manifest created at `.ovav/plans/p5-warp-profiles-denylist.ps1`,
reverted due to UI enum value mismatch (Warp rejected).

## Root cause of failure

I invented `execution_profiles.<name>` block structure without verifying
Warp's actual schema. The values `name`, `read_files`, `apply_code_diffs`,
etc. within a profile block are not all valid in Warp's internal schema.

## Plan to complete P5

### Path A — UI (CEO action)

1. Warp → Settings → Agent → Execution Profiles
2. Create 4 profiles manually following Warp's UI fields:
   - `ovav_build` — apply_diffs=always_allow, read=always_allow, commands=agent_decides, ...
   - `ovav_yolo`  — everything always_allow except computer_use=never
   - `ovav_review` — read=always_allow, write=ask, commands=ask
   - `thavren_systems` — read=always_allow, config=ask, sudo=ask, network=ask
3. I audit the generated TOML afterward (no invention)

### Path B — Path A failed → Delegation

If CEO cannot do UI: delegate to **Elena** (UX Design lead) to design
a Warp Settings UI walkthrough document. CEO follows Elena's guide.

### Path C — Path A + B failed → Skip + Documentation

Document P5 as deferred to CEO with all info in `.ovav/plans/p5-attack-plan.md`.
Status: 0% in settings.toml, 100% in this attack plan.

## Denylist rebuild (P5 partial)

Plan §13: keep denylist granular (no broad shell blocks).

Implemented via PS1 script (`.ovav/plans/p5-warp-profiles-denylist.ps1`).
The script applies Blocks: `sudo`, `git worktree`, `rm -rf`, `curl`, `wget`,
`ssh`, `scp`, `rsync`, etc. This part was NOT rejected by Warp.

**Status:** denylist mutation is pending Warp restart to take effect.

## Acceptance criteria

- [ ] 4 profiles created in Warp UI
- [ ] TOML audited (no invented values)
- [ ] denylist mutation applied (git worktree blocked)
- [ ] commit + merge audit

## Owner

- Implementation: CEO (via UI)
- Audit: Thavren (post-UI)
- Rejected-paths: documented in commit
