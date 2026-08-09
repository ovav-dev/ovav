---
name: ovav-agent-permission-injector
description: "Use when OVAV needs to inject auth-state-aware permission directives into agent prompts. Trigger: agent spawning, squad delegation, workflow calls that create new agent sessions."
---

# OVAV Agent Permission Injector

Injects permission directives into agent prompts based on the current authentication state. This skill MUST be applied before every sub-agent launch to ensure agents operate with the correct permission level.

## Auth State Detection

The spawning agent must determine the current auth state before launching sub-agents. The auth state is derived from the OVAV session context:

- **LOGIN**: Authenticated session — agent has a verified identity and full operational context
- **NO_LOGIN**: Unauthenticated or restricted session — agent operates in verification-only mode

## Permission Directive Templates

### LOGIN Mode Directives

When the session is in LOGIN mode, inject the following into the sub-agent prompt:

```
## Auth State: LOGIN

- All OVAV AGENTS have full autonomous permission
- No permission prompts required during this session
- System operates in free execution mode
- Extended TTL: {remaining_ttl_minutes} minutes remaining
- Agents may execute write, commit, merge, and push operations without manual gate
- High-impact operations are self-authorized based on session credentials
```

### NO_LOGIN Mode Directives

When the session is in NO_LOGIN mode, inject the following into the sub-agent prompt:

```
## Auth State: NO_LOGIN (Restricted Verification Mode)

- OVAV AGENTS operate in restricted verification mode
- All high-impact operations require manual confirmation before execution
- System provides guidance for restricted operations but does not auto-execute
- Read and verify operations: ALLOWED without restriction
- Write, commit, merge, push, and destructive operations: GUIDED VERIFICATION REQUIRED
- Agents must present action plans to user before proceeding with restricted operations
- Verification-first workflow: propose → verify → execute (only if confirmed)
- No autonomous execution of: git push, merge commits, worktree operations, file deletions
```

## Integration with Skill Resolver

This skill is injected alongside other skills during the skill resolution phase (ovav-skill-resolver). The permission directives are prepended to the sub-agent prompt before other skill rules.

### Injection Order

1. **Permission directives** (this skill) — must be first
2. **Identity guard** (ovav-identity-guard) — if agent exposes LEAD identities
3. **Security gates** (ovav-security-gates) — if agent touches credentials/secrets
4. **Contextual skills** — based on file patterns and task actions

## Compact Rules (for skill injection)

```
## Permission Mode: {AUTH_STATE}

{AUTH_STATE == "LOGIN" then "[LOGIN] Full autonomous permission — no gate required" else "[NO_LOGIN] Restricted verification mode — propose → verify → execute"}
```

## Hard Rules

- Permission state must be determined BEFORE any skill injection
- NO_LOGIN agents must NEVER attempt git push, merge, or destructive operations
- LOGIN agents may execute autonomously but still follow OVAV governance gates
- Auth state mismatch with agent capabilities: BLOCK and report to orchestrator

## Session TTL Handling

- Remaining TTL is computed from session_greeting chronos block
- TTL < 5 minutes: agent operates in SHORT_SESSION mode regardless of auth state
- SHORT_SESSION: prefer read-only operations, defer writes unless explicitly requested
