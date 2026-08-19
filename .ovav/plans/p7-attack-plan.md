# P7 — Attack Plan (MiniMax-M3 Configuration)

## Status

P7 status: PARTIAL.

| Component | Status |
|---|---|
| OpenCode M3 | ✅ `minimax-coding-plan` provider active |
| Crush M3 | ✅ `MiniMax` provider configured, M3 model loaded |
| Warp Agent → M3 | ⏸️ CEO UI (custom inference endpoint) |

## Plan to complete P7

### Warp Agent custom endpoint (UI)

1. Open Warp → Settings → AI → Manage models
2. Click **+ Add Custom Endpoint**
3. Fill fields (per plan §23):
   - **Base URL**: `https://api.minimax.io/v1`
   - **Schema**: OpenAI Chat Completions
   - **Model**: `MiniMax-M3`
   - **Key**: MiniMax Subscription Key (already in OVAV Vault: `minimax_api_key`)
4. Set as default for new sessions
5. Disable "Auto" and "Auto Genius" routing

### Risk assessment

- Warp Free tier: custom endpoint may still consume AI credits (per plan §23)
- Mitigation: monitor credit usage; switch to paid tier if needed
- Verification: send test query, check credit delta

### Alternative: OpenCode as primary M3 path

If Warp Agent + M3 has friction, OpenCode is already 100% M3.
Warp can be used for non-AI terminal tasks; OpenCode for AI work.

This is the documented **path of greater independence** per plan §24.

## Files in commits

- `.ovav/plans/p7-minimax-endpoints.md` — CEO UI instructions
- `opencode.json` — model set to `minimax-coding-plan/MiniMax-M3`

## Already verified

- OpenCode: `minimax-coding-plan` key in `~/.local/share/opencode/auth.json`
- Crush: `MiniMax` provider in `~/.local/share/crush/providers.json`
- Vault: `minimax_api_key` entry, encrypted with `~/.config/ovav/vault.key`

## Owner

- Configuration: CEO (via Warp UI)
- Audit: Thavren (post-UI)
- OpenCode/Crush: operational already
