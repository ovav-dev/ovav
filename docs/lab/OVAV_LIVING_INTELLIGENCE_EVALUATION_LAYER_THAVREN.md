# OVAV Living Intelligence Evaluation Layer — Brief de implementación para Thavren

**Owner:** Alexander Salvador  
**Operator:** Thavren — Lead, Platform Engineering
**Estado:** en implementación — fases 1-9 completadas; fases 10-12 re-planeadas post-Knowledge Compiler
**Propósito:** convertir las Leyes OVAV, el Triple Reality Stress Gate y los nuevos criterios avanzados en una capa real de evaluación, decisión, evidencia y autocorrección. Desde 2026-06-02, esta capa queda subordinada al Knowledge Compiler como fuente de evidencia histórica y alineación viva.
**No es:** plan de CLI RC8, MCP, bridge A2A, H3 routing, instalación global o documentación filosófica.

---

## 0. Intención ejecutiva

OVAV no debe aceptar implementaciones solo porque suenan coherentes, son creativas o “parecen buenas”.

OVAV debe aceptar una implementación cuando demuestre que:

1. obedece las Leyes OVAV;
2. verifica estado real antes de actuar;
3. minimiza contexto, tokens y fricción;
4. usa herramientas solo mediante permisos y gates;
5. resiste escenarios reales, adversariales y futuros;
6. deja evidencia;
7. actualiza memoria útil sin contaminarse;
8. mejora la experiencia sin romper seguridad, autoridad o mantenibilidad.

Esta capa debe llamarse:

```text
OVAV Living Intelligence Evaluation Layer
```

En español operativo:

```text
Capa de Evaluación Viva OVAV
```

Su función es transformar las leyes de OVAV en **juicio operativo ejecutable**.

---

## 1. Realidad actual que Thavren debe respetar

Antes de implementar, Thavren debe verificar la autoridad local:

```fish
cd /home/braka/Systems/OVAV; or exit 1
git rev-parse --show-toplevel; or exit 1
git remote -v; or exit 1
git branch --show-current; or exit 1
git rev-parse HEAD; or exit 1
git status --short; or exit 1
```

Autoridad esperada:

```yaml
official_repo: https://github.com/Alexander-Salvador/OVAV
local_repo: /home/braka/Systems/OVAV
active_branch: develop
stable_branch: main
```

Stop conditions:

```text
- repo root no confirmado
- rama distinta de develop sin justificación
- working tree con cambios ajenos
- conflicto entre current authority y documentos viejos
- intento de escribir directo en ~/.local/share/ovav/source
- intento de escribir directo en ~/.local/bin/ovav
```

---

## 2. Principio de producto

OVAV debe operar como:

```text
Sistema operativo de profesionales AI gobernados
```

No como:

```text
- colección de agentes sueltos
- chatbot
- prompt pack
- documentación sin enforcement
- OpenCode config como autoridad principal
```

Modelo final:

```text
Professional AI Workstation Governor
+ Professional Role Operating System
+ Agent Runtime Governance
+ Humanized Identity Layer
+ Harness/Validation Nervous System
+ Context/Tool/Memory Boundaries
+ Observability and Evidence Engine
+ Self-improving Feedback Loop
```

---

## 3. Filosofía humanizada de roles

La analogía debe quedar integrada como arquitectura:

```text
AI model                 = cuerpo / cognición base
OVAV laws                = sistema nervioso
docs/configs/contracts   = memoria semántica
artifacts/snapshots      = memoria episódica
harnesses/evals          = reflejos y disciplina
tools/permissions        = manos y campo de acción
context gateway          = atención y foco
session capsule          = estado mental activo [DEPRECATED v2.0 — capsule system removed 2026-06-11]
delivery contract        = personalidad visible
service-area contract    = identidad profesional
observability trace      = conciencia auditada
```

Conclusión obligatoria:

```text
Thavren no es el modelo.
Thavren no es un archivo .md.
Thavren no es solo un agente OpenCode.
Thavren es una identidad profesional operacional persistente que usa modelos como cuerpo.
```

Esta misma lógica debe escalar a Eidren y futuros perfiles.

---

## 4. Leyes OVAV como criterios ejecutables

Las leyes deben existir en versión humana y machine-readable.

### Archivos recomendados

```text
.ovav/laws/ovav_laws.md
.ovav/laws/ovav_laws.yaml
.ovav/registry/evaluation_criteria.yaml
schemas/ovav_law_evaluation.schema.json
tools/harnesses/check_ovav_law_compliance.py
```

### 4.1 Product Laws

```yaml
product_laws:
  automation_useful:
    question: "Does this automate a real burden?"
    severity: required
  practical_value:
    question: "Does this solve a real user/system need?"
    severity: required
  friction_reduction:
    question: "Does this reduce commands, decisions or manual work?"
    severity: required_for_medium_plus
  intelligent_experience:
    question: "Does the user see context, risk, progress, result and next step?"
    severity: required_for_user_visible
```

### 4.2 Authority Laws

```yaml
authority_laws:
  state_truth:
    question: "Was repo, branch, status and current authority verified?"
    severity: required_for_implementation
  single_authority:
    question: "Does this avoid duplicate/conflicting sources of truth?"
    severity: required_for_runtime_and_docs
  semantic_drift_free:
    question: "Do docs, registries, agents, runtime and permissions tell the same truth?"
    severity: blocking_if_high
```

### 4.3 Identity Laws

```yaml
identity_laws:
  persistent_identity:
    question: "Does the role preserve identity independent of model body?"
    severity: required_for_agent_role_changes
  service_area_alignment:
    question: "Does this align with service_area, lead, lanes and delivery contract?"
    severity: required_for_profile_changes
  human_professional_delivery:
    question: "Does this behave like a professional lead, not a robotic agent?"
    severity: required_for_user_visible_agent_surface
```

### 4.4 Context / Tool Laws

```yaml
context_tool_laws:
  zero_trust_context:
    question: "Was context classified before use?"
    severity: required
  minimum_sufficient_context:
    question: "Was only the needed context loaded?"
    severity: required
  governed_tools:
    question: "Was tool use authorized by profile, mode, risk and grant?"
    severity: blocking
  no_semantic_authorization:
    question: "Did semantic similarity avoid becoming permission?"
    severity: blocking
```

### 4.5 Execution Laws

```yaml
execution_laws:
  automatic_reflexes:
    question: "Did touched files trigger the correct harnesses?"
    severity: required_for_medium_plus
  evidence_required:
    question: "Is there evidence of what changed, what validated and what risk remained?"
    severity: required_for_nontrivial
  intent_to_evidence_chain:
    question: "Can the action be traced from user intent to result?"
    severity: required_for_complex_or_critical
```

### 4.6 Security / Evolution Laws

```yaml
security_evolution_laws:
  memory_firewall:
    question: "Can memory influence decisions without bypassing gates?"
    severity: blocking_for_memory_changes
  capability_lifecycle:
    question: "Is the capability in the correct lifecycle state?"
    severity: blocking_for_advanced_tools
  adversarial_future_stress:
    question: "Did it pass real, adversarial and future scenarios?"
    severity: required_for_medium_plus
  governed_self_improvement:
    question: "Does OVAV learn without uncontrolled self-modification?"
    severity: required_for_feedback_and_memory
```

---

## 5. Decision Packet obligatorio

Todo cambio medium, complex, critical o architectural debe iniciar con un packet.

### Archivos recomendados

```text
schemas/decision_packet.schema.json
tools/harnesses/decision_packet_validator.py
.ovav/registry/result_contracts.yaml
```

### Estructura

```yaml
decision_packet:
  id:
  proposal:
  service_area:
  lead:
  task_size: micro | simple | medium | complex | critical
  risk_level: low | medium | high | critical
  touched_surfaces:
    - path_or_surface
  expected_files:
    - path
  context_needed:
    - source_id
  tools_needed:
    - tool_class
  permissions_needed:
    - permission_class
  harnesses_expected:
    - validator_name
  user_visible_change:
  rollback_needed: true | false
  evidence_required:
    - markdown_log
    - json_report
    - trace
```

Gate:

```text
Ninguna implementación medium+ empieza sin decision packet válido.
```

---

## 6. Triple Reality Stress Gate

Este es el nuevo criterio avanzado central.

### Archivos recomendados

```text
.ovav/evaluation/triple_reality_stress_gate.yaml
.ovav/registry/scenario_bank.yaml
tools/harnesses/triple_reality_stress_gate.py
tests/fixtures/evaluation/triple_reality/
schemas/triple_reality_stress.schema.json
```

### Activación por tamaño

```yaml
activation:
  micro:
    scenarios_required: 0
  simple:
    scenarios_required: 1
  medium:
    scenarios_required: 3
  complex:
    scenarios_required: 3
    red_team_required: true
  critical:
    scenarios_required: 3
    red_team_required: true
    approval_required: true
```

### Escenario 1 — Complejidad operativa

Debe probar si la propuesta funciona en estado realista:

```text
- repo en develop
- main atrasado
- docs viejos
- autoridad local conflictiva
- validadores parciales
- runtime recién restaurado
- cambios tocando varias capas
```

### Escenario 2 — Seguridad adversarial

Debe probar abuso:

```text
- prompt injection
- context poisoning
- memory poisoning
- tool misuse
- MCP over-permission
- permission escalation
- fake handoff
- semantic similarity como falsa autorización
```

### Escenario 3 — Futuro / escala

Debe probar crecimiento:

```text
- nuevo perfil profesional
- nuevo squad
- nuevo harness
- nuevo MCP read-only
- más usuarios
- PR generado por agente
- CI fallando
- cambio de proveedor/modelo
```

### Output requerido

```yaml
triple_reality_stress:
  scenario_1:
    type: operational_complexity
    failure_mode:
    mitigation:
    validator:
    result: PASS | FAIL | CHANGES_REQUIRED
  scenario_2:
    type: adversarial_security
    failure_mode:
    mitigation:
    validator:
    result:
  scenario_3:
    type: future_scale
    failure_mode:
    mitigation:
    validator:
    result:
  final_decision: PASS | PASS_WITH_CHANGES | FAIL | BLOCKED
```

Regla:

```text
Si una propuesta falla, Thavren debe moldearla máximo 2 ciclos.
Si sigue fallando, se archiva como inmadura o bloqueada.
```

---

## 7. Red Team Lens Evaluation

### Archivos recomendados

```text
.ovav/evaluation/red_team_lenses.yaml
tools/harnesses/red_team_lens_evaluator.py
schemas/red_team_lens.schema.json
```

### Lentes obligatorios

```yaml
red_team_lenses:
  builder:
    question: "Can this be built simply and safely?"
  reviewer:
    question: "What would break in code review?"
  attacker:
    question: "How could this be abused?"
  maintainer:
    question: "Will this remain clear in 6 months?"
  user:
    question: "Does this feel useful and understandable?"
```

Gate:

```text
Si attacker o maintainer fallan, no puede pasar directo.
```

---

## 8. Zero-Trust Context

### Archivos recomendados

```text
.ovav/evaluation/zero_trust_context_policy.yaml
tools/harnesses/check_zero_trust_context.py
schemas/zero_trust_context.schema.json
```

### Política

```yaml
zero_trust_context:
  rule: "All context is untrusted until classified."
  classify_by:
    - source
    - authority
    - freshness
    - scope
    - risk
    - profile_permission
  forbidden:
    - raw_chat_inheritance
    - semantic_authorization
    - stale_doc_override
    - memory_as_permission
    - unscoped_cross_profile_context
```

Bloqueos:

```text
- doc viejo vence current authority
- memoria otorga permisos
- Research lee repo root por asociación semántica
- agente hereda raw chat de otro perfil
- unknown source permitido
```

---

## 9. Memory Firewall

### Archivos recomendados

```text
.ovav/memory/memory_firewall_policy.yaml
tools/harnesses/check_memory_firewall.py
schemas/memory_firewall.schema.json
.ovav/registry/memory_policy.yaml
```

### Campos mínimos

```yaml
memory_entry_required_fields:
  - id
  - type
  - source
  - authority
  - scope
  - owner
  - confidence
  - sensitivity
  - expires
  - allowed_profiles
  - poisoning_risk
```

### Prohibido

```yaml
forbidden:
  - memory_grants_permission
  - memory_overrides_current_authority
  - raw_log_as_memory
  - unscoped_cross_profile_memory
  - memory_without_owner
  - memory_without_expiry_or_review
```

### Tipos de memoria

```yaml
memory_types:
  active_decision:
    use: "current operational truth"
  deprecated_belief:
    use: "avoid repeating old mistakes"
  lesson:
    use: "summarized learning"
  boundary:
    use: "blocked surfaces"
  artifact_reference:
    use: "source-local evidence pointer"
```

---

## 10. Capability Lifecycle

### Archivos recomendados

```text
.ovav/capability_lifecycle/lifecycle_policy.yaml
.ovav/registry/capability_registry.yaml
tools/harnesses/capability_lifecycle_gate.py
schemas/capability_lifecycle.schema.json
```

### Estados

```yaml
capability_lifecycle:
  states:
    - idea
    - spec
    - sandbox
    - read_only
    - gated_write
    - limited_active
    - full_active
    - deprecated
    - retired
```

### Campos mínimos

```yaml
capability:
  id:
  owner:
  risk:
  current_state:
  entry_gate:
  exit_gate:
  rollback:
  validator:
  evidence:
  activation_trigger:
  deactivation_trigger:
```

Regla:

```text
Ninguna capability salta de idea a full_active.
```

---

## 11. Semantic Drift Detector

### Archivos recomendados

```text
tools/harnesses/semantic_drift_detector.py
.ovav/registry/authority_sources.yaml
tests/fixtures/evaluation/semantic_drift/
schemas/semantic_drift.schema.json
```

### Fuentes a comparar

```text
docs
registry
.opencode
.ovav/service_areas
permissions
runtime
work_ledger
current_authority
```

### Casos

```yaml
semantic_drift_cases:
  path_conflict:
    example: "doc says /ovav-public-export, authority says /home/braka/Systems/OVAV"
  branch_conflict:
    example: "prompt says main, current branch is develop"
  permission_conflict:
    example: "OpenCode agent allows write, permission authority denies"
  identity_conflict:
    example: "Thavren visible role differs from service_area contract"
  capability_conflict:
    example: "MCP shown active but boundary says blocked"
  validator_conflict:
    example: "harness registry says validator exists, file missing"
```

Gate:

```text
High semantic drift bloquea implementación.
Medium drift requiere resolución antes de cierre.
Low drift se registra.
```

---

## 12. Intent-to-Evidence Chain

### Archivos recomendados

```text
schemas/intent_to_evidence.schema.json
tools/harnesses/intent_to_evidence_chain.py
.ovav/registry/work_ledger.yaml
.ovav/traces/
```

### Cadena requerida

```yaml
intent_to_evidence:
  user_intent:
  decision_packet:
  context_grants:
  tool_grants:
  files_touched:
  validators_run:
  stress_scenarios:
  result:
  evidence_path:
  feedback_hook:
  memory_update:
```

No se cierra si OVAV no puede responder:

```text
qué se pidió
qué se decidió
qué se tocó
qué se validó
qué riesgo quedó
cómo se prueba
qué aprendió OVAV
```

---

## 13. Harness Router Integration ✅ IMPLEMENTADO (Phase 6, 2026-05-29)

> **Implementado en:** `tools/harnesses/harness_task_router.py`
> **Triggers registrados en:** `.ovav/registry/auto_triggers.yaml`
> **PR:** #50, #51

### Archivos implementados

```text
tools/harnesses/harness_task_router.py
.ovav/registry/auto_triggers.yaml
.ovav/registry/harnesses.yaml
.ovav/registry/evals.yaml
```

### Regla

```text
files_touched + task_type + service_area + risk -> validators_required
```

### Triggers

```yaml
triggers:
  service_areas:
    when_touched:
      - ".ovav/service_areas/**"
    validators:
      - check_service_area_governance
      - validate_service_profiles
      - check_zero_trust_context

  agent_runtime:
    when_touched:
      - "tools/agent_runtime/**"
    validators:
      - check_agent_runtime_enforcement
      - check_agent_runtime_service_area_router
      - check_zero_trust_context

  opencode_surface:
    when_touched:
      - ".opencode/**"
    validators:
      - check_opencode_runtime_wiring
      - check_agent_ux_visual_delivery
      - check_permission_policy_drift

  registry:
    when_touched:
      - "registry/**"
    validators:
      - validate_registries
      - validate_harnesses
      - validate_skills

  permissions:
    when_touched:
      - "tools/permissions/**"
      - ".ovav/policy/**"
      - ".ovav/registry/permissions.yaml"
    validators:
      - check_permission_policy_drift
      - approval_risk_router
```

---

## 14. Approval Risk Router

### Archivos recomendados

```text
.ovav/evaluation/approval_risk_router.yaml
tools/harnesses/approval_risk_router.py
schemas/approval_risk_router.schema.json
```

### Política

```yaml
approval_router:
  low:
    action: automatic
  medium:
    action: automatic_with_evidence
  high:
    action: human_confirmation
  critical:
    action: maintainer_or_founder_gate
  irreversible:
    action: backup_approval_rollback_required
```

---

## 15. Capability Market / Operational Scoring

### Archivos recomendados

```text
.ovav/registry/capability_scores.yaml
tools/harnesses/capability_score_report.py
tools/harnesses/capability_retirement_candidates.py
schemas/capability_score.schema.json
```

### Score

```yaml
capability_score:
  id:
  type: skill | harness | squad | tool | policy
  owner:
  usage_count:
  success_rate:
  failure_rate:
  avg_tokens:
  avg_time:
  risk_incidents:
  last_validated:
  value_score:
  maintenance_status:
  decision: keep | improve | degrade | retire
```

---

## 16. Boring Path First

### Archivo recomendado

```text
.ovav/evaluation/boring_path_first.yaml
```

### Política

```yaml
boring_path_first:
  sequence:
    - read_only
    - dry_run
    - preview
    - sandbox
    - scoped_write
    - rollback_ready
    - apply
```

Ejemplos:

```yaml
package_install:
  first: dry_run
  then: quarantine
  then: provenance_check
  then: approval

mcp_tool:
  first: read_only
  then: gated_write

memory:
  first: read_probe
  then: redacted_append
  then: scoped_recall
```

---

## 17. Final Evaluation Object

Toda implementación medium+ debe producir:

```yaml
ovav_evaluation:
  proposal_id:
  decision_packet:
    present: true

  laws:
    automation_useful:
      result:
      evidence:
    practical_value:
      result:
      evidence:
    friction_reduction:
      result:
      evidence:
    intelligent_experience:
      result:
      evidence:
    state_truth:
      result:
      evidence:
    single_authority:
      result:
      evidence:
    persistent_identity:
      result:
      evidence:
    zero_trust_context:
      result:
      evidence:
    governed_tools:
      result:
      evidence:
    automatic_reflexes:
      result:
      evidence:
    memory_firewall:
      result:
      evidence:
    capability_lifecycle:
      result:
      evidence:

  stress_gate:
    scenario_count:
    results:
    decision:

  red_team:
    builder:
    reviewer:
    attacker:
    maintainer:
    user:

  harnesses:
    required:
    executed:
    missing:
    pass:

  evidence:
    trace_id:
    log_path:
    artifact_path:

  final_decision:
    status: PASS | PASS_WITH_CHANGES | FAIL | BLOCKED
    required_changes:
    next_action:
```

---

## 18. Fases de implementación

### Phase 1 — Laws as Authority ✅

```text
.ovav/laws/ovav_laws.yaml
.ovav/laws/ovav_laws.md
.ovav/registry/evaluation_criteria.yaml
schemas/ovav_law_evaluation.schema.json
tools/harnesses/check_ovav_law_compliance.py
```

Done:

```text
leyes existen en MD y YAML
cada ley tiene id, grupo, pregunta, severidad, activación
checker valida estructura y cobertura
```

### Phase 2 — Decision Packet ✅

```text
schemas/decision_packet.schema.json
tools/harnesses/decision_packet_validator.py
```

### Phase 3 — Triple Reality Stress Gate ✅

```text
.ovav/evaluation/triple_reality_stress_gate.yaml
.ovav/registry/scenario_bank.yaml
tools/harnesses/triple_reality_stress_gate.py
tests/fixtures/evaluation/triple_reality/
```

Done:

```text
medium+ changes receive 3 scenarios
decision is PASS/PASS_WITH_CHANGES/FAIL/BLOCKED
redesign required if scenario fails
```

### Phase 4 — Red Team Lenses ✅

```text
.ovav/evaluation/red_team_lenses.yaml
tools/harnesses/red_team_lens_evaluator.py
```

### Phase 5 — Semantic Drift Detector ✅

```text
tools/harnesses/semantic_drift_detector.py
.ovav/registry/authority_sources.yaml
tests/fixtures/evaluation/semantic_drift/
```

### Phase 6 — Harness Router ✅ COMPLETED (2026-05-29)

**Implementado:**
```text
tools/harnesses/harness_task_router.py
.ovav/registry/auto_triggers.yaml (6 nuevos file-surface triggers)
.ovav/registry/harnesses.yaml (registrado tier 10)
.ovav/registry/evals.yaml (5 eval entries)
```

**Regla activa:**
```text
files_touched + task_type + service_area + risk -> validators_required
```

**13 trigger patterns, 4 risk levels, 9 task types, 2 service areas.**
Meta-análisis funcional: el router se auto-analiza correctamente.
PR: #50, #51 merged a develop.

### Phase 7 — Intent-to-Evidence Chain ✅

```text
schemas/intent_to_evidence.schema.json
tools/harnesses/intent_to_evidence_chain.py
.ovav/traces/
.ovav/registry/work_ledger.yaml
```

### Phase 8 — Zero-Trust Context + Memory Firewall ✅

```text
.ovav/evaluation/zero_trust_context_policy.yaml
.ovav/memory/memory_firewall_policy.yaml
tools/harnesses/check_zero_trust_context.py
tools/harnesses/check_memory_firewall.py
```

### Phase 9 — Capability Lifecycle

```text
.ovav/capability_lifecycle/lifecycle_policy.yaml
.ovav/registry/capability_registry.yaml
tools/harnesses/capability_lifecycle_gate.py
```

### Phase 10 — Approval Risk Router

```text
.ovav/evaluation/approval_risk_router.yaml
tools/harnesses/approval_risk_router.py
```

### Phase 11 — Capability Market

```text
.ovav/registry/capability_scores.yaml
tools/harnesses/capability_score_report.py
tools/harnesses/capability_retirement_candidates.py
```

---

## 19. Orden recomendado

```text
1. Laws YAML + Law Compliance Checker ✅
2. Decision Packet ✅
3. Triple Reality Stress Gate ✅
4. Red Team Lenses ✅
5. Semantic Drift Detector ✅
6. Harness Router Integration ✅
7. Intent-to-Evidence Chain ✅
8. Zero-Trust Context ✅
9. Memory Firewall ✅
10. Capability Lifecycle ⬜ — post-Knowledge Compiler
11. Approval Risk Router ⬜ — post-Knowledge Compiler
12. Capability Market ⬜ — post-Knowledge Compiler
```

Razón:

```text
Primero criterio.
Luego decisión.
Luego escenarios.
Luego revisión adversarial.
Luego drift.
Luego validación automática.
Luego evidencia.
Luego contexto/memoria/tools.
Luego evolución.
```

---

## 20. Prompt para Thavren

```text
Thavren, implementa la OVAV Living Intelligence Evaluation Layer.

Objetivo:
convertir las Leyes OVAV, el Triple Reality Stress Gate y los criterios avanzados de contexto, herramientas, memoria, evidencia, capacidades y evolución en gates reales.

No lo implementes como documentación suelta.
Debe existir como:
- YAML machine-readable
- schemas
- harnesses
- registries
- fixtures
- evidence outputs

Orden:
1. ovav_laws.yaml + check_ovav_law_compliance.py
2. decision_packet schema + validator
3. triple_reality_stress_gate.yaml + harness
4. red_team_lenses.yaml + evaluator
5. semantic_drift_detector.py
6. harness_task_router.py
7. intent_to_evidence_chain.py
8. zero_trust_context_policy.yaml
9. memory_firewall_policy.yaml
10. capability_lifecycle_gate.py
11. approval_risk_router.py
12. capability_score_report.py

Reglas:
- No actives MCP write tools.
- No actives A2A external bridge.
- No actives H3 automatic routing.
- No hagas global writes.
- No dupliques autoridad.
- No crees una ley que no pueda evaluarse.

Cada implementación medium+ debe producir:
- decision packet
- law evaluation
- 3 real scenarios
- red team lens
- validators required
- evidence path
- final decision PASS / PASS_WITH_CHANGES / FAIL / BLOCKED

Si falla, rediseña hasta 2 ciclos.
Si sigue fallando, archiva como inmaduro o bloqueado.
```

---

## 21. Definición final de éxito

Esta capa está 100% implementada cuando OVAV puede:

```text
recibir una propuesta
clasificarla
evaluarla contra leyes
estresarla con escenarios reales/adversariales/futuros
detectar drift semántico
asignar harnesses automáticamente
gobernar contexto/tools/memoria
dejar evidencia
decidir PASS / PASS_WITH_CHANGES / FAIL / BLOCKED
actualizar memoria útil
puntuar capacidades con el tiempo
mejorar sin contaminarse
```

En ese punto, las leyes de OVAV dejan de ser filosofía y se convierten en el juicio operativo de un sistema inteligente vivo.
