# OVAV Support Guide

## Support principle

Support should be diagnostic, source-local and reproducible.

## Before asking for help

Run:

```fish
python3 tools/validators/check_service_area_governance.py
python3 tools/validators/check_build17_canonical_review.py
python3 tools/validators/check_build18_launch_pack.py
python3 tools/validators/validate_all.py
git status --short
```

## Useful diagnostic information

Provide:

| Item | Command |
|---|---|
| Current branch | `git branch --show-current` |
| Last commit | `git log --oneline -1` |
| Working tree | `git status --short` |
| Validator state | commands above |
| Recent diff | `git diff --stat` |

## Support routing

| Problem | Service area |
|---|---|
| Repo/runtime/OpenCode/install/validation | OVAV Platform Engineering |
| Source verification/benchmark/research decision | OVAV Research Intelligence |

## Rule

Do not paste secrets, tokens or private keys into support requests.
