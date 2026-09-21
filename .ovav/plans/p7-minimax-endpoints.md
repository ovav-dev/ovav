# P6 + P7 — CEO Action Items (Warp UI)

CRIT-009 lesson applied: I no longer invent TOML enum values. The
following items **must be done via Warp UI**, then I audit the
generated TOML afterward.

## P6 — Warp Workflows (Warp Drive)

Create 9 workflows in Warp → Drive → Workflows:

| # | Name | Command |
|---|---|---|
| 1 | OVAV · Create Worktree | `ovav worktree owc {{task}} --profile {{profile}}` |
| 2 | OVAV · Status | `ovav worktree owl` |
| 3 | OVAV · Visual Status | `ovav worktree owlsv` |
| 4 | OVAV · Verify | `ovav worktree owv` |
| 5 | OVAV · Update | `ovav worktree owu` |
| 6 | OVAV · Done | `ovav worktree owd` |
| 7 | OVAV · Route Commit | `ovav worktree owx cherry-pick {{commit}}` |
| 8 | OVAV · Cleanup Preview | `ovav worktree owclean --dry-run` |
| 9 | OVAV · Sync | `ovav worktree ows` |

### Profile enum (insert as enum text input)

```
feature
refactor
docs
spike
research
migration
enterprise
hotfix
release
patch
emergency
```

### Rules

- Workflows call OWS — never `git worktree` directly
- Open Warp Drive → Workflows → New → paste command
- For dynamic enums, use shell-driven enums (Warp supports)

---

## P7 — Warp Agent → MiniMax-M3 custom endpoint

Open Warp → Settings → AI → Manage models → **+ Custom endpoint**

### Fields

| Field | Value |
|---|---|
| Base URL | `https://api.minimax.io/v1` |
| Schema | OpenAI Chat Completions |
| Model | `MiniMax-M3` |
| Key | Your MiniMax Subscription Key (in OVAV Vault: `minimax_api_key`) |

### After setup

- Set as default for new sessions
- Verify Warp AI credits NOT consumed (per plan §23)
- Disable Auto + Auto Genius as defaults

### Verification

```bash
# Test from OpenCode (already confirmed)
jq -r '."minimax-coding-plan".key | length' \
  /home/braka/.local/share/opencode/auth.json
# Should return ~120 (length of valid MiniMax key)
```

---

## P11 — Final acceptance run

After P6 + P7 UI work, run:

```bash
ovav worktree owc launch-final-acceptance
# Inside worktree:
ovav validate
ovav worktree owv
```

I will verify all 47 acceptance criteria from plan §42.

---

## Files in this commit

- `p7-minimax-endpoints.md` — this document (CEO action items)
- `opencode.json` — model swapped to `minimax-coding-plan/MiniMax-M3`,
  `@warp-dot-dev/opencode-warp` plugin added (verified on npm)

## Already completed (no CEO action needed)

- P7 OpenCode: `minimax-coding-plan` provider active since previous session
- P7 Crush: `MiniMax` provider configured, `MiniMax-M3` model loaded (1M ctx)
- P9 Cloud Conversations: ON (verified in settings.toml)
- P10 Privacy: telemetry + crash reporting ON (verified)

## After CEO completes the UI work

Reply with "P6+P7 UI done" and I will:
1. Audit the generated TOML
2. Commit any new profiles/workflow config
3. Run P11 final acceptance
4. Close the OVAV × WARP 2026 plan
