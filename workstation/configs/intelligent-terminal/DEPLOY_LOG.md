# IT Keybindings Deploy Log

This file tracks when the `workstation/scripts/deploy-it-keybindings.sh` script was run
and against which live settings.json. It is updated automatically by the deploy pipeline
(manual or automated) to provide an audit trail of fragment → live synchronization.

## Deployments

| Date (UTC) | Operator | Live Path | Fragment SHA (HEAD) | Result |
|------------|----------|-----------|---------------------|--------|
| 2026-08-14T12:54:28Z | thavren (initial pipeline setup) | `/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json` | `933f6cb` | success — 47 keybindings, 0 broken |

## How this file is updated

Each successful run of `deploy-it-keybindings.sh` appends a new row above. The deploy
script is idempotent — running it multiple times with the same fragment produces no
changes after the first run.

## Why this matters

Prior to this log, there was no audit trail of when (or whether) the fragment in
`workstation/configs/intelligent-terminal/settings-fragment.json` was actually applied
to the live IT settings.json. This led to a multi-day drift where the fragment was fixed
(commit `bc1fb2b`) but the live file still had 13 `id:null` + 4 wrong-action entries —
causing `shift+arrow → D + Windows beep` regression that CEO reported on 2026-08-14.

The new `it_live_keybindings` validator (validator #73) now catches this drift on every
`ovav validate` run, so future regressions will be caught before CEO notices.
