# OVAV Security Policy

## Security model

OVAV uses service-area governance, context isolation and tool gating.

Visible service areas:

- OVAV Platform Engineering
- OVAV Research Intelligence

## Context boundary

Research Intelligence does not read repo root, `.opencode`, `.ovav/context`, raw snapshots, install artifacts or git history by default when the user mentions OVAV.

Internal review requires one of:

- explicit scoped user permission,
- attached specific files,
- sanitized Platform Engineering handoff.

## Tool boundary

High-risk operations require explicit approval and traceable scope:

- git commit,
- git push,
- install/apply,
- global config write,
- destructive delete,
- secrets handling,
- external adapter action.

## Secrets

Do not commit secrets, tokens, credentials or private keys.

Suspicious files or strings must trigger review before commit.

## Reporting security issues

For source-local development, report security issues through the active Platform Engineering workflow. Include:

1. affected surface,
2. reproduction steps,
3. expected behavior,
4. actual behavior,
5. risk,
6. proposed fix if known.
