# OVAV Consumer Bridge — Official Specification

**Version**: 1.0.0
**Date**: 2026-07-16
**Owner**: Thavren (Platform Engineering)
**Authority**: OVAV Permission Authority v2 + CEO waiver

---

## Purpose

The **OVAV Consumer Bridge** is the official mechanism for external consumer
projects (e.g. `bt-sys-react`, future consumer projects) to access OVAV
subsystems (OWS Worktree System, agents, memory, skills) safely.

**Replaces**: ad-hoc scripts that tried to inject grants directly into
`permission_authority.json` (see [INCIDENT-2026-07-16](../issues/INCIDENT-2026-07-16-bitel-agent.md)).

---

## Architecture

```
Consumer Project (e.g. bt-sys-react)
  │
  │  runs `bin/ovav-consumer register <id>`
  │  with CEO-signed waiver file
  │
  ▼
OVAV Consumer Bridge (`bin/ovav-consumer`)
  │
  │  validates:
  │  • CEO waiver signature
  │  • consumer ID format (^[a-z][a-z0-9-]{2,30}$)
  │  • root path exists and is git repo
  │  • URLs not typosquats
  │  • scope boundary explicit
  │
  ▼
.ovav/registry/consumers.yaml   (authoritative registry)
  │
  │  mirrors to:
  │  .ovav/runtime/logs/consumer_audit.jsonl (audit log)
  │
  ▼
OVAV permission_authority.json  (CEO-approved grant ONLY)
  ↑ mutated by CEO-approved `ovav grant` command, NOT by consumer bridge
```

**Critical invariant**: `bin/ovav-consumer` does **NOT** mutate
`permission_authority.json` directly. The CEO must explicitly run the grant
command after bridge registration.

---

## Commands

### `ovav-consumer register <consumer_id> --root <path> --waiver <file>`

Registers a new consumer project in `.ovav/registry/consumers.yaml`.

Required:
- `consumer_id`: kebab-case, lowercase, no special chars (`^[a-z][a-z0-9-]{2,30}$`)
- `--root <path>`: absolute path to consumer project root
- `--waiver <file>`: CEO waiver file with required signatures

**Waiver format** (`~/.config/ovav/consumer_waiver_<id>.yaml`):

```yaml
consumer_id: bt-sys-react
approved_by: Alexander Salvador
approved_at: 2026-07-16T20:00:00Z
bootstrap_version: 2.0.0
scope:
  subsystems: [ows]
  capabilities: [owc, owd, owl, owv, ows, owclean, owm, owx, owa, owr, owlk]
  flags:
    owc: [--carry, --profile]
    owd: [--resume, --compliance]
    owl: [--zombie-only, --json, --history]
    owv: [--verbose]
    ows: [--rebase, --full]
    owsuggest: [--explain]
  boundaries:
    git_operations: consumer_root_only
    audit_required: true
    isolation: .worktrees/
    self_recovery: own_worktrees_only
rationale: |
  Bitel-agent call center needs OWS worktree isolation for
  parallel feature work without interfering with main dev.
  v2.0.0: all SU-HARDENING-v0.1.0 flags available.
```

The bridge validates:
- waiver signatures (HMAC if available)
- consumer root exists and is git repo
- scope doesn't include destructive operations without audit

### `ovav-consumer list`

Prints all consumers in `.ovav/registry/consumers.yaml` with status.

### `ovav-consumer grant <consumer_id>`

The CEO-only command (gated by `~ /.config/ovav/ceo_keys/`). Generates
the `consumer_grants` entry for `permission_authority.json` and prints
a `git apply`-ready patch. CEO reviews, applies, commits.

### `ovav-consumer audit <consumer_id>`

Shows consumer audit log (command invocations, timestamps, scopes used).

### `ovav-consumer revoke <consumer_id>`

Marks consumer as `revoked` in registry. CEO must also remove
`consumer_grants` from `permission_authority.json`.

---

## Security Gates

1. **No auto-modification** of `permission_authority.json`. The bridge
   writes only to `.ovav/registry/consumers.yaml`. The CEO applies grants
   via `ovav-consumer grant` which generates a `git apply`-ready patch.

2. **URL typosquat validation** — the bridge uses
   `.ovav/security/url_allowlist.yaml` (allow-listed domains only):
   - `mimocode.ai` (official)
   - `xiaomi.com`, `xiaomimimo.com` ONLY if CEO approves
   - All other domains REJECTED with reason
   - Default DENY. Explicit ALLOW only.

3. **Consumer ID format** — strict regex, no special chars, no path
   traversal patterns (`..`, `/`, `\`).

4. **Audit log** — every command invocation writes to
   `.ovav/runtime/logs/consumer_audit.jsonl` with `{ts, actor, command,
   consumer_id, scope, decision, reason}`.

5. **No shell injection** — the bridge uses bash with strict quoting and
   no `eval`, no parameter expansion from user input. The Go core
   (forthcoming) will use `os/exec` with explicit argument arrays.

6. **Protected branch enforcement** — when CEO applies grant via
   `permission_authority.json`, the existing OVAV protected branch gate
   applies (waiver required).

---

## Replaces Old Patterns

| Old (insecure) | New (secure) |
|---|---|
| `bt-sys-react/.ovav/provider-changes/apply-ovs-provider-side.sh` writes directly to `permission_authority.json` | `bin/ovav-consumer register <id>` requires CEO waiver, writes only to `consumers.yaml` |
| `remediate-ows-slash-commands.sh` creates root-owned OWS scripts | `bin/ovav-consumer bootstrap <id>` creates user-owned scripts only after registry entry approved |
| `grant_by: thavren` without waiver | Waiver file with HMAC signature + timestamp + scope + rationale |
| Working-tree contamination spread via symlinks | Bridge writes only to `consumers.yaml` (single source of truth) |
| Direct config mutations of unrelated files (e.g. `.ovav/governance/auto_notification.yaml`) | All governance files protected by F0–F5 validators; mutations logged via `auto_notification.yaml` written only by Thavren with audit |

---

## Migration Path for Existing Consumers

For `bt-sys-react` (the first consumer that hit this incident):

1. **Old** scripts in `.ovav/provider-changes/` (deleted in FASE 1 cleanup)
2. **New** path: run `bin/ovav-consumer register bt-sys-react --root /home/braka/Work/web/products/bt-sys-react --waiver <file>`
3. CEO approves waiver → registry gets `bt-sys-react` entry
4. CEO runs `ovav-consumer grant bt-sys-react` → generates patch
5. CEO applies patch to `permission_authority.json` via normal commit flow
6. Consumer runs `bin/ovav-consumer bootstrap` to install OWS shims in their own bin/

---

## Files

| Path | Purpose |
|---|---|
| `bin/ovav-consumer` | Main CLI (bash, stdlib-only, Go core planned) |
| `.ovav/registry/consumers.yaml` | Authoritative consumer registry |
| `.ovav/security/consumer_bridge.md` | This document |
| `.ovav/security/url_allowlist.yaml` | Allow-listed domains |
| `.ovav/runtime/logs/consumer_audit.jsonl` | JSONL audit trail |
| `clients/ovav-consumer-bootstrap.sh` | Template for consumer-side bootstrap |

---

## Roadmap

- [ ] **v1.0.0** (NOW): Bash bridge with strict validation
- [ ] **v1.1.0**: Go core implementation (stdlib only, no 3rd party deps in critical path)
- [ ] **v1.2.0**: HMAC-signed consumer tokens for cross-machine consumer auth
- [ ] **v2.0.0**: Web-based consumer registry dashboard (cPanel integration)
