# Final Launch Smoke Evidence

## Result

PASS for source-local launch smoke evidence. Final launch closure remains blocked until the final commit and tag are created.

## Scope

- Issue: `#5 Final launch verification smoke evidence`
- Branch: `task/rc8.3-experience-engine-foundation`
- Base checked locally: `origin/develop@6ca89ce`
- Authority: Final Launch Verification on top of B23.

## Evidence

- Strict runtime validation: PASS
- OpenCode runtime wiring: PASS
- Permission policy drift: PASS
- Agent runtime enforcement: PASS
- Final launch current/runtime authority: PASS
- Aggregate validator: PASS
- CLI practical smoke: PASS, 21/21
- CLI fresh source smoke: PASS, 15/15
- Clean install smoke: PASS
- Publish/export gate: PASS
- Repo presentation gate: PASS
- Close-layer dry-run: complete; close target still `not_closed`.

## Fixes made during smoke

- Updated CLI smoke expectations from the old `Home` marker to the RC8.3 cockpit `OVAV Launch` marker.
- Updated fresh-smoke and local installer payload export to validate the current working tree, not stale `HEAD` only.
- Updated public export gate to scan git-visible files and avoid ignored local payload noise.
- Updated permission drift validation to run read-only in source archives without `.git`.
- Split secret-marker fixture strings so export scanning does not report validator fixtures as real secrets.

## Remaining closure blockers

- Final commit is not created yet.
- Final tag is not created yet.
- Issue #5 should remain open until final close/tag approval.
