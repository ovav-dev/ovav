# OVAV Security

## Research Intelligence Boundaries

**Research Intelligence** (Eidren) operates with governed access. It does not operate from the **repo root** by default. All repo edits, git writes, install/apply, global config writes, and raw snapshot reads require **explicit** Platform Engineering authorization.

## Platform Engineering Authority

Platform Engineering (Thavren) retains exclusive authority over:
- Plugin installation and evaluation
- Global configuration writes
- Install/apply operations
- Force push and branch deletion

## Security Gates

- `workspace_safety_gate` — runs before all writes
- `session_context_guard` — verifies governance file integrity
- `git_push_gate` — governs all push operations via HTTPS only
