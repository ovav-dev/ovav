# Security Backlog — Pendiente de implementación

> Actualizado: 2026-06-02 | Gate unificado: PR #56 | 11/11 vectores cubiertos en archivos

## Brechas activas (0) — TODAS CERRADAS ✅

### 1. Protección del gate consigo mismo — `priority: critical` ✅ CERRADA

**Riesgo:** Si un atacante modifica `check_host_config_drift.py` Y `.ovav/registry/core_hashes.yaml` simultáneamente, el gate validaría contra un baseline falso y daría PASS.

**Mitigación implementada:**
- `.ovav/gate_self_hash` — hash externo del gate (fuera del baseline)
- `tools/security/gate_self_protection.py` — verificador previo (237 líneas)
- Cableado: `check_host_config_drift.py` (línea 556), `validate_all.py`, `auto_triggers.yaml` (on_session_start + before_implementation)

---

### 2. Verificación de git HEAD — `priority: critical` ✅ CERRADA

**Riesgo:** El self-heal restaura archivos con `git checkout HEAD -- file`. Si un atacante logra hacer un commit malicioso, el self-heal restauraría desde ese HEAD comprometido.

**Mitigación implementada:**
- `.ovav/trusted_head_hash` — hash de confianza del último commit verificado
- `tools/security/head_integrity_verifier.py` — verificador (236 líneas)
- Cableado: `check_host_config_drift.py` (línea 441), `integrity_monitor.py` (pre-self-heal), `validate_all.py`, `auto_triggers.yaml` (on_session_start + before_apply)

---

### 3. Protección de contexto de sesión — `priority: high` ✅ CERRADA

**Riesgo:** El integrity seal se verifica al inicio de la sesión. Si alguien logra envenenar el contexto durante la sesión, OVAV podría procesar instrucciones contaminadas.

**Mitigación implementada:**
- `tools/security/session_context_guard.py` — guard de sesión (270 líneas)
- `.ovav/security/session_guard_policy.yaml` — política de guard
- Cableado: `session_greeting.py` (auto-acción _auto_session_context_guard), `validate_all.py`, `auto_triggers.yaml` (on_session_start + on_user_message)
- Detección de patrones de inyección: ignore_previous_instructions, role_override, gate_bypass, etc.

---

## Backlog general — Re-alineado 2026-06-02

> **La visión evolucionó.** OVAV como Collective Intelligence System.
> El Knowledge Compiler es ahora P0 — habilita todo lo demás.

### P0 — OVAV Knowledge Compiler
- Knowledge Compiler P0.2: implementado; consolidación documental y de autoridad en curso
- Pattern Detector + Alignment Engine: implementados; transición y criterio dinámico añadidos
- Sistema Nervioso Vivo: propuesta prioritaria, pendiente de diseño/gates antes de implementación

### Living Intelligence Evaluation Layer

| Fase | Estado | Nota |
|---|---|---|
| 10 — Capability Lifecycle | Re-planeada post-Knowledge Compiler | El KC informa readiness con evidencia histórica |
| 11 — Approval Risk Router | Re-planeada post-Knowledge Compiler | El KC provee evidencia y patrones de riesgo |
| 12 — Capability Market | Re-planeada post-Knowledge Compiler | El KC es fuente de verdad del scoring interno |

### Issues abiertos (re-priorizados)

| # | Título | Prioridad |
|---|---|---|
| P0 | Consolidación Knowledge Compiler P0.2 + diseño Sistema Nervioso Vivo | AHORA |
| #46 | Core Advance L5-L7 | Post-KC |
| #44 | CLI Living Experience RC10 | Post-KC |
