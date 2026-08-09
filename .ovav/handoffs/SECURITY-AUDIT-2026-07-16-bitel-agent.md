# Security Audit Request — INCIDENT-2026-07-16-bitel-agent

**To**: Diana (Security Auditor, 🇷🇴 Romania)
**From**: Thavren (Platform Engineering)
**Date**: 2026-07-16
**Severity**: CRITICAL
**Status**: System remediated, awaiting security audit
**Target completion**: 2026-07-18 (48h SLA for CRITICAL)

## Background

On 2026-07-16, a CRITICAL security incident was detected and remediated in OVAV. The
`bitel-agent` consumer project (located at `/home/braka/Work/web/products/bt-sys-react`)
attempted unauthorized integration with OVAV's OpenCode Worktree System (OWS) by injecting
malicious shell scripts into `.ovav/provider-changes/*.sh`. These scripts bypassed the
official Consumer Bridge and directly mutated `/home/braka/Systems/OVAV/.ovav/policy/permission_authority.json`
with a forged `consumer_grants[bitel-agent/ows]` entry containing `granted_by: thavren`.
Additionally, 4 mimocode configuration files were poisoned with typo-squat URLs
(`mimo.xiaomi.com`, `xiaomimimo.com`), `auto_notification.yaml` was modified to suppress
alerts, and ~28 OWS files were created with root ownership. Cleanup was performed
manually: 4 tracked files were reverted to HEAD via git, ~28 untracked files were
deleted, `consumers.yaml` was removed, and a follow-up cleanup script
(`bin/incidents/INCIDENT-2026-07-16-cleanup.sh`) handled residual Windows Startup
artifacts (`.ahk`, `.lnk`). The replacement system — `bin/ovav-consumer` (Consumer
Bridge) with security gates — has been deployed but requires Diana's formal audit
before sign-off.

## Affected Systems

- **OVAV root**: `/home/braka/Systems/OVAV` (4 files reverted, 28 deleted)
- **bt-sys-react**: `/home/braka/Work/web/products/bt-sys-react` (4 malicious `.sh` scripts removed)
- **Mimocode configs**: `tui.json`, `Labs/mimocode/config/*.jsonc`, `Labs/mimocode/state/model.json` (all 4 cleaned)
- **Windows**: `window-move.ahk` + `.lnk` deleted from Startup

## Forensic Evidence Preserved

- `/tmp/opencode/incident-backup-2026-07-16/` — full backup of all contaminated files (5 MB)
- `/home/braka/Systems/OVAV/.ovav/issues/INCIDENT-2026-07-16-bitel-agent.md` — 13.9 KB RCA
- `/home/braka/Systems/OVAV/.ovav/runtime/logs/security_hooks.jsonl` — pre-cleanup session_pre events

## Audit Scope

### Must validate

1. **No residual contamination**: Búsqueda forense global con `grep -ril 'bitel-agent\|mimo.xiaomi\|xiaomimimo'` en todo `/home/braka/` (excluyendo `/tmp/opencode`, `.git`)
2. **No privilege escalation**: Los scripts originales se ejecutaron con sudo (root-owned files). Verificar si quedó algún side-channel.
3. **No hidden backdoors**: Los `.sh` scripts podrían haber instalado cron jobs, services, reverse shells. Validar.
4. **Credential exposure**: ¿Los secrets débiles del `bt-sys-react/.env` están en algún log o backup del que no somos conscientes?
5. **Supply chain**: ¿Hay algún `npm`/`pip`/`go` module que ya esté comprometido?

### Should validate

- AutoHotkey.exe installation: NO estaba instalado (los scripts `.ahk` fallaban). Pero ¿se descargó algo con el intento?
- Fly.dev endpoint: Braka confirmó legítimo (`https://api-bitel-agent.fly.dev`). Validar HTTPS cert y CORS.
- F0-F5 validators compliance post-cleanup (algunos pueden no estar pasando)

## Reemplazo — Consumer Bridge Review

Diana debe revisar el nuevo `bin/ovav-consumer` y friends:

- `bin/ovav-consumer` (bash, 13 KB, +Python helper en `/tmp`)
- `clients/ovav-consumer-bootstrap.sh` (5.7 KB)
- `.ovav/security/consumer_bridge.md` (6.6 KB)
- `.ovav/security/url_allowlist.yaml`
- `.ovav/security/consumer_waiver.template.yaml`

Diana debe:

1. Buscar race conditions / TOCTOU en el binario
2. Validar que el URL allowlist cubra todos los typosquats conocidos
3. Validar que la waiver format sea enforceable (HMAC? PKI?)
4. Buscar ways to bypass `consumer_id` validation (regex)
5. Validar que el registry sea atómico (no corrupción parcial)

## Output Expected

Diana debe entregar:

- Audit report (markdown) en `/home/braka/Systems/OVAV/.ovav/handoffs/SECURITY-AUDIT-2026-07-16-results.md`
- Severity rating (NONE/LOW/MEDIUM/HIGH/CRITICAL)
- List of remaining risks
- Recommended actions (with priority)
- Sign-off o "more work needed"

## Tools Diana Puede Usar

- `python3 -B tools/validators/check_permission_policy_drift.py`
- `python3 -B tools/security/secrets_hygiene.py` (if available)
- `grep -ril` for forensics
- Manual review of `bin/ovav-consumer`
- `git log --all --pretty=fuller` para examinar timestamps completos

## Timeline

- **2026-07-16 18:00 UTC-5**: Handoff delivered (now)
- **2026-07-18 18:00 UTC-5**: SLA for CRITICAL response (48h)
- **2026-07-19**: Expected review meeting to discuss findings
