# OVAV Install Guide

## Scope

This guide covers source-local launch usage. It does not authorize global install by default.

## Supported launch posture

| Mode | Status |
|---|---|
| Source-local repo usage | Supported |
| OpenCode repo-local surfaces | Supported |
| Global install/apply | Governed; requires explicit approval, backup, verify and rollback |
| Global config writes | Governed; not default |
| Destructive cleanup | Governed; requires approval and backup |

## Preflight

This preflight must pass before any launch, install or governed apply action.

From the OVAV root:

```fish
python3 tools/validators/check_service_area_governance.py
python3 tools/validators/check_build17_canonical_review.py
python3 tools/validators/check_build18_launch_pack.py
python3 tools/validators/validate_all.py
```

All must pass before calling the repository launch-ready.

## Source-local use

```fish
cd /home/braka/dev/bab/ventures/ovav
opencode
```

Use the visible service areas from OpenCode:

- OVAV Platform Engineering
- OVAV Research Intelligence

## Global install policy

Global install is not a default launch action.

Before any global operation, OVAV must provide:

1. scope,
2. affected paths,
3. backup plan,
4. apply plan,
5. verify command,
6. rollback plan,
7. explicit user approval.

If any item is missing, the operation must fail closed.
