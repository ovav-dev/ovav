# OVAV Quickstart

OVAV is a commercial AI workstation governor with two professional service areas: **OVAV Platform Engineering** (Thavren, lead) and **OVAV Research Intelligence** (Eidren, lead).

## Quick Start

1. All work is **source-local** by default — operates within the repository working directory.
2. Start with `python3 tools/agent_runtime/session_greeting.py --json` to initialize a session.
3. Use `OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate` to verify system health.

## Key Commands

- `python3 tools/ovav_runtime.py context --next` — current work context
- `python3 tools/ovav_runtime.py next` — next work segment
- `python3 tools/ovav_runtime.py validate` — full system validation

## Architecture

OVAV follows OVAV-first architecture: canonical source in `.ovav/agents/`, projected to CLI clients via adapters.
