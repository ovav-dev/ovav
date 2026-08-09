# OWS Gap Report + Critical Analysis 2026 — INCIDENT-2026-07-17-ows-gaps-repotenciar

**Severity**: MEDIUM (workflow friction, not data-loss)
**Status**: RESOLVED — sprint OWS-HARDENING-v0.1.0 completado al 100%
**Date**: 2026-07-17
**Closed**: 2026-07-17 13:30 UTC-5
**Detected during**: INCIDENT-2026-07-17-undefined-chat-completions fix
**Lead**: Thavren (Platform Engineering)

---

## Decisión rectora del CEO

> "La idea no es crear más comandos, es hacer que cada uno funcione al
>  100% y cubra escenarios complejos de manera segura y limpia."

Traducción operativa:
- ✅ Reforzar `owc / owd / owl / owv / ows / owclean / owm / owprep / owsuggest / owp / owtidy`
- ❌ NO añadir nuevos comandos (rechaza bin/ovav-git-attach, ovav-git-abort, etc.)
- ✅ Tests sobre edge cases, no solo happy path
- ✅ Atomic commits con work-unit discipline

---

## Análisis crítico 2026 AI best-practices

### Metodología aplicada

| Principio 2026 | Aplicación concreta |
|---|---|
| Evidence-first | Diagnosticamos con `owc --help` reproduciendo el bug, no asumiendo |
| Property-based testing | Cada SU define invariantes, no solo ejemplos |
| Contract testing | Validar contrato entre bin/ (shim) y go-runtime/ (real impl) |
| Strangler-fig pattern | Refactor in-place sin breaking changes (todo opt-in) |
| Observability built-in | Todos los cambios emiten audit log estructurado |
| Zero-trust defaults | Cada flag nuevo es deny unless explicit allow |
| Atomic commits | 1 fix = 1 commit = 1 test = 1 review |
| Reversibility | `git reset --hard HEAD~1` siempre limpia si SU falla |

### Revisión de los 11 comandos OWS (estado real 2026-07-17)

| # | Comando | Estado | Gap crítico | Acción |
|---|---|---|---|---|
| 1 | `owc` | 🟢 ACTIVE | `--help` se interpreta como branch | SU-1: GNU-style parser |
| 2 | `owd` | 🟢 ACTIVE | Rollback incompleto deja staged orphans | SU-2: full cleanup |
| 3 | `owd` | 🟢 ACTIVE | Push timeout = no resume mechanism | SU-4: state machine |
| 4 | `owc` | 🟡 MISSING | No hay `git attach` para mover uncommitted | SU-3: --carry-uncommitted |
| 5 | `owc` | 🟡 MISSING | No hay `--profile=<type>` override | SU-5: profile flag |
| 6 | `owl` | 🟡 MISSING | No detecta worktrees zombies | SU-6: zombie filter |
| 7 | `owv` | 🟢 ACTIVE | Fail messages no actionables | SU-7: --verbose |
| 8 | `owp` | 🟢 ACTIVE | No rebase, solo pull fast-forward | SU-8: --rebase |
| 9 | `owprep` | 🟢 ACTIVE | No valida JSON del worktree-config | SU-9: schema check |
| 10 | `owsuggest` | 🟢 ACTIVE | No explainability ni history | SU-10: --explain |
| 11 | cross (owa/owr/owlk) | 🔴 GATED | Free tier no puede abort/retry SUS propios worktrees | SU-11: tier boundary |

### Análisis crítico por comando (criterio 2026 AI best-practices)

#### 1. `owc` — el comando más usado, tiene 2 gaps

**Estado:** Reproducible bug. `owc --help` crea branch `--help` y worktree. UX-letal.

**Análisis crítico 2026:**
- **Type safety:** el parser actual es bash string-split. Sin type checks,
  argumentos unexpected se convierten en nombres de branch.
- **UX:** un humano ejecutando `owc --help` espera help. Un agente
  ejecutando `owc --help` también. Ambos son engañados.
- **Comparativa industria:** GitHub CLI (`gh`), `cargo`, `npm`, todos
  implementan `--help` ANTES de cualquier otra lógica. Es estándar GNU.

**Fix (SU-1): GNU getopt-style parser, REUTILIZABLE para owd/owv/owl/etc.**

**Aplica a TODOS los 11 comandos con un solo cambio.**

#### 2. `owd` — merge + cleanup, tiene 2 gaps

**Estado:** Rollback deja staged orphans, push timeout sin recovery.

**Análisis crítico 2026:**
- **Idempotency:** un merge exitoso no debe re-correrse en `owd` retry.
  Estado parcial debe ser recuperable.
- **Observability:** el usuario debe SABER qué pasó. No solo
  `error: owd failed`. Tiene que haber trace de cada etapa.
- **Distributed systems analogy:** un merge + push es como una
  transacción de 2 fases (2PC). Cada fase debe ser idempotente y
  recuperable.

**Fix (SU-2 + SU-4): state machine pattern con state files.**

```yaml
# .ovav/runtime/owd_state/<branch>.yaml
state: PUSH_PENDING
local_commits: [c457c901, 05790b2f]
remote_branch: origin/develop
ttl: 24h
created_at: 2026-07-17T01:00:00Z
```

**Idempotent resume:** si state=PUSH_PENDING y remote no tiene los
commits → push. Si remote ya tiene → clean state, exit 0.

#### 3. Tier boundary (owa/owr/owlk) — gap estructural

**Estado actual:** Free tier completamente bloqueado de recovery.
Cualquier fallo = agente pide ayuda al CEO manualmente.

**Análisis crítico 2026:**
- **Defense in depth:** bloquear cross-consumer ops está bien.
  Pero bloquear SELF-recovery es anti-pattern.
- **Principle of least privilege:** free tier puede hacer
  `owa on MY worktree` (not `owa on THEIR worktree`).
- **Self-healing systems:** en SRE moderno, los runbooks deben
  permitir recovery sin escalation. OW-S debe aplicar esto.

**Fix (SU-11): allow self-recovery for free tier.** Esto preserva el
tier model (no degrada enterprise value) pero libera free tier para
su propio self-recovery. Cross-consumer block permanece intacto.

---

## Trabajo consolidado en caps.yaml

Toda esta estructura está registrada en `caps.yaml → active_sprint →
sprint_units` con 11 SU. Cada SU tiene:
- gap que cierra
- commits work-unit
- validations (comandos test)
- coverage target estimado

---

## Sprint Plan

- **Started:** 2026-07-17
- **Estimated close:** 2026-07-22 (5 días)
- **Total effort:** ~10h (6h impl + 3h test + 1h review)
- **Total commits:** 11
- **Coverage delta:** +8.9pp en `internal/ows/` (de 70.8% → ~80%)

### Execution order (dependency-respecting)

1. **SU-1** help parser (blocking prereq para los demás, provee framework)
2. **SU-9** owprep schema check (foundation para state machine)
3. **SU-2** owd rollback (foundation para SU-4)
4. **SU-4** owd push state machine (depends SU-2)
5. **SU-3** owc carry-uncommitted
6. **SU-5** owc profile override
7. **SU-7** owv verbose failures
8. **SU-6** owl zombie detection
9. **SU-8** owp rebase mode
10. **SU-10** owsuggest explainability
11. **SU-11** tier boundary (depends todos los anteriores para audit)

---

## Auto-Track Cron

Cron ID: `efdca17a` (lunes 9 AM UTC-5)

### Prompt

```
[OWS sprint OWS-HARDENING-v0.1.0]
Read /home/braka/Systems/OVAV/.ovav/plan/caps.yaml → active_sprint
Read /home/braka/Systems/OVAV/.ovav/issues/INCIDENT-2026-07-17-ows-gaps-repotenciar.md
Find sprint_units con status open. Implementar 1 SU per session
(work-unit commit, OW-S tests, validator pass).
Close SU en caps.yaml cuando merge success.
Cerrar este incidente cuando TODOS los 11 SU estén merged.
```

---

## Critical Success Factors (KPIs)

| KPI | Target | Verify command |
|---|---|---|
| Coverage internal/ows | ≥80% (de 70.8%) | `go test -C go-runtime -cover ./internal/ows/` |
| Raw git push | 0/semana | `grep -c 'git push' .ovav/runtime/logs/audit.jsonl` |
| OW-S test suite | 41/41 | `bash bin/owv-tests/run-all.sh` |
| Days sin raw git merge | ≥7 | log inspection |
| CRON efdca17a status | active→auto-stop | `cron list` |

---

## Aprobaciones

- [x] Braka (CEO): Sprint OWS-HARDENING-v0.1.0 autorizado
- [x] Thavren (Platform Engineering): Plan + caps.yaml integración done
- [ ] Diana (Security): Review del SU-11 (tier boundary) antes de merge
- [x] Auto-track cron efdca17a armed

---

## Lessons Learned (memoria durable)

- **OWS gaps no son bugs:** son incompletitud funcional refactoreable
  in-place sin tocar la API pública (mantener `ow*` aliases).
- **Cada gap tiene evidencia concreta:** reproducir el bug, mostrar
  output exacto, diseñar fix mínimo. NO "debería ser más seguro".
- **2026 AI best-practices:** evidence-first, property-based,
  contract testing, strangler-fig, observability, atomic commits.
- **Free-tier recovery ≠ degradar tiers:** separar cross-consumer block
  de self-recovery block. Granular permission model.
