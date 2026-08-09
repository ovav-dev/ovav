# CLI Living Experience RC9 — Feedback Evaluation and V4 Implementation Brief

Status: launch verification input. This is a source-local design/implementation brief, not a production/global-ready claim.

## 1. Evaluation

- The previous `Guided setup` label was misleading because Enter started the setup pipeline immediately. The correct posture is preview-first: show what OVAV will configure, then expose an explicit action.
- Static text panes for update, restore and uninstall reduce trust. These screens must become selectable, state-driven options.
- Left/right arrows must not behave as generic navigation. They are only valid for horizontal option rows; `b` is the only back action.
- The header logo should remain premium and stable. New logo explorations should be isolated from the CLI until approved.
- User-facing copy must describe OVAV-managed configurations, not imply installation of third-party tools like Zellij, WezTerm, OpenCode or editors.

## 2. Implemented RC9 Source-Local Corrections

- Restored the prior premium ASCII wordmark.
- Renamed first-run choices:
  - `Full OVAV setup` — complete OVAV config pack, backup first.
  - `Custom setup` — choose plan, config packs and roles.
- Added an explicit `Full OVAV setup` overview before setup starts.
- Reworked Update into dynamic categories: `System`, `Tools`, `Backup`, `Other`, `Update All`.
- Reworked Restore into selectable repair/backup paths with `Initial config` reserved as a reset posture.
- Reworked Uninstall into horizontal choices: `Uninstall OVAV`, `Restore backup`, `Other`.
- Restricted left/right arrows to horizontal choice rows; `b` is the back key.

## 3. Target UX Flow

### Full OVAV setup

1. Home opens `Full OVAV setup` overview.
2. User sees columns: plan, config surfaces, backup.
3. User chooses `Start setup` or `Back home` with left/right.
4. Setup sequence shows visible stages:
   - Read workstation
   - Backup configs
   - Confirm plan
   - Apply OVAV configs
   - Verify setup
5. Completion exits back to usable OVAV posture.

### Custom setup

1. User chooses plan.
2. User toggles config packs and roles.
3. User previews selection.
4. User confirms setup with the same backup/apply/verify sequence.

### Installed user home

If OVAV is already detected, `Full OVAV setup` is hidden. Home should prioritize:

- Update OVAV
- Custom setup
- Repair / Restore
- Uninstall OVAV

## 4. /connect and Paid Tier Direction

Recommended direction: do not expose a raw permanent API key as the main paid-user primitive.

Use a short-lived device-code or signed entitlement flow:

- `/connect` shows either login URL + one-time code, or paste-code option.
- Server validates payment tier and returns a scoped entitlement token.
- Local token is short-lived, revocable, tier-scoped and bound to workspace/user/device metadata.
- CLI never stores payment data.
- Offline grace can be policy-limited for free/core actions only.

Security rules:

- No hardcoded master API key.
- No unrestricted global write token.
- Token scopes map to plan capabilities: Free, paid tiers, advanced roles, config packs.
- Sensitive install/apply/restore actions remain gated locally even with a paid entitlement.

## 5. Advanced 2026 Improvements for V4 Pro

- Rust TUI core for high-framerate rendering, resize safety and deterministic key handling; Python remains orchestration/runtime bridge until migration is justified.
- Declarative screen schema: every CLI screen renders from state rows/actions, not custom static prose blocks.
- AI-assisted repair hints only after deterministic checks fail; no opaque AI auto-repair before backup.
- Event stream progress model for backup/apply/verify so users see live tool/config status without internal logs.
- License/entitlement gateway with device-code auth, signed scoped tokens and server-side revocation.
- Snapshot intelligence: compare current config hash against last known-good backup and offer precise restore instead of full reset when possible.
- Accessibility modes: compact, high contrast, ASCII-only, and non-interactive JSON parity.

## 6. V4 Pro Next Tasks

1. Convert all cockpit screens to a shared dynamic option renderer.
2. Add backup progress rows per managed config family: OpenCode, shell, terminal/session, editor, OVAV runtime.
3. Add a real entitlement model stub for `/connect` without external service behavior in source-local mode.
4. Add keyboard smoke tests for blocked left/right navigation.
5. Add screen contract tests so old static panes cannot regress.
