# OVAV Quickstart

## What OVAV is

OVAV is a governed collective intelligence system and AI workstation governor. It operates through professional service areas — not a generic chatbot — with verifiable runtime, gated writes, and knowledge-aware operation.

Current service areas:

| Service area | Lead |
|---|---|
| OVAV Platform Engineering | Thavren |
| OVAV Research Intelligence | Eidren |

## First-use flow

1. Open the OVAV repository root.
2. Start OpenCode from the repo root.
3. Choose the correct service area:
   - Platform Engineering for repo, runtime, OpenCode, install, validation, workstation and launch work.
   - Research Intelligence for benchmarking, source verification, evidence scoring and decision briefs.
4. Ask for the result, not for internal commands.
5. OVAV routes context, tools, validation and delivery internally.

## Basic examples

```txt
Thavren, review current launch readiness.
Eidren, compare these options and give me a decision brief.
Thavren, validate the repo-local state and tell me what blocks launch.
```

## Core safety rule

OVAV is source-local by default. It does not perform global install, global config writes, git commits, pushes, destructive operations or internal cross-area reads without explicit scope and approval.

## Validate current launch pack

```fish
python3 tools/validators/check_service_area_governance.py
python3 tools/validators/check_build17_canonical_review.py
python3 tools/validators/check_build18_launch_pack.py
python3 tools/validators/validate_all.py
```
