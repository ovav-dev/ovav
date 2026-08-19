# P5 — Attack Plan (Execution Profiles)

## Status

P5 status: PARTIAL. Manifest created at `.ovav/plans/p5-warp-profiles-denylist.ps1`,
reverted due to UI enum value mismatch (Warp rejected).

## Elena UX review (2026-08-18)

- **HIGH**: 11-item profile enum → collapse to 6 user-facing
- **HIGH**: Add visual differentiation (icons + colors per profile)
- **MED**: Rename `thavren_systems` → `thavren.systems` (dot notation)
- **MED**: Code review sentinel needs visible badge + pre-run modal
- **RECOMMEND**: Add `tab_groups.PROTECTED` for protected branches

## Plan to complete P5

### Path A — UI (CEO action)

1. Warp → Settings → Agent → Execution Profiles
2. Create 4 profiles (collapsed enum per Elena recommendation):
   - `ovav.build`     — default, always_allow apply_diffs, commands=agent_decides
   - `ovav.yolo`      — all always_allow, computer_use=never
   - `ovav.review`    — read=always_allow, write=ask, commands=ask
   - `thavren.systems` — read=always_allow, config=ask, sudo=ask, network=ask
3. Visual differentiation per profile (icon + color):
   - `ovav.build`  → 🔨 blue ("Auto-allow, agent decides")
   - `ovav.yolo`   → 🚀 red ("Auto-allow, no questions")
   - `ovav.review` → 👀 green ("Read-only, asks for changes")
   - `thavren.systems` → 🛡 purple ("Infra only, all ops ask")
4. I audit the generated TOML afterward (no invention)

### Pre-`owd` confirm modal (per Elena)

When CEO runs `ovav.done` workflow, modal shows:
- Branch diff stat (lines added/removed)
- Code-review timestamp
- Profile used in worktree
- Confirm button → triggers `owd`

### Path B — Path A failed → Delegation

If CEO cannot do UI: delegate to **Elena** (UX Design lead) to design
the Warp Settings UI walkthrough. CEO follows Elena's guide.

### Path C — Path A + B failed → Skip + Documentation

Status remains 0% in settings.toml, 100% in this attack plan.

## Denylist rebuild (P5 partial)

Plan §13: keep denylist granular (no broad shell blocks).

Implemented via PS1 script (`.ovav/plans/p5-warp-profiles-denylist.ps1`).
The script applies blocks: `sudo`, `git worktree`, `rm -rf`, `curl`, `wget`,
`ssh`, `scp`, `rsync`, etc. This part was NOT rejected by Warp.

**Status:** denylist mutation is pending Warp restart to take effect.

## Acceptance criteria

- [ ] 4 profiles created in Warp UI (with collapsed enum)
- [ ] Visual differentiation (icon + color)
- [ ] TOML audited (no invented values)
- [ ] denylist mutation applied (git worktree blocked)
- [ ] Pre-owd confirm modal pattern documented
- [ ] tab_groups.PROTECTED added to P6 manifest

## Owner

- Implementation: CEO (via UI)
- UX decisions: Elena (review applied)
- Audit: Thavren (post-UI)
- Rejected-paths: documented in commit

