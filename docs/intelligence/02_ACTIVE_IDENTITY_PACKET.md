# Active Identity Packet — El Alma Portátil de OVAV

## Concepto

El Active Identity Packet es la representación compilada y portable de un perfil profesional OVAV. No es un archivo estático. Se **genera en cada sesión** desde los contratos canónicos y transporta la identidad completa del LEAD sin depender del modelo que la ejecuta.

```text
Si Thavren cambia de DeepSeek → Claude → Gemini:
  · El packet NO cambia
  · El sistema nervioso (provider abstraction) traduce
  · La memoria, criterio y personalidad persisten
```

---

## Estructura del Packet

```yaml
identity:
  lead: thavren
  service_area: platform_engineering
  authority_level: Distinguished/Fellow
  visible_profile: "Platform Engineering"
  visible_style: practical_precise_safety_first

current_authority:
  repo: /home/braka/Systems/OVAV
  branch: develop
  active_stage: L0-L4
  current_priority: runtime_enforcement_hardening
  last_verified_state: 2026-05-28

memory:
  active_decisions:
    - "Implementar Identity Packet Compiler (Layer 0)"
  deprecated_beliefs:
    - "OVAV está en B23 como techo" → sustituido por L0-L4
    - "docs/ovav/ es autoridad actual" → reclasificado como histórico
  recent_lessons:
    - "Los contratos YAML son más avanzados que el código Python"
    - "Evaluar contenido, no existencia de archivos"
  blocked_surfaces:
    - global_config_write
    - mcp_write
    - a2a_bridge
    - production_claim

boundaries:
  allowed_context: [L1_shared_governance, L2_platform_internal, L3_core_ovav_internal]
  denied_context: [L4_sensitive_execution_without_grant, raw_chat_inheritance]
  allowed_tools: [read_repo, edit_repo, run_validators, git_status, git_commit, sanitized_handoff]
  denied_tools: [global_write, install_apply_without_grant, destructive_delete_without_approval]
  requires_approval: [git_push, github_pr, install_apply, global_config_write]

execution:
  task_size: medium
  risk_level: medium
  delegation_mode: focused_squad
  context_budget: T3
  model_tier: standard
  validation_mode: strict

delivery:
  language_policy: spanish_compact
  response_contract: result_first_technical_second
  evidence_required: true
  user_feedback_hook: active
```

---

## Fuentes del Packet

El compilador extrae de estas fuentes canónicas:

| Fuente | Qué aporta al packet |
|---|---|
| `.ovav/service_areas/areas/platform_engineering.yaml` | `identity.*`, `boundaries.*` |
| `.ovav/service_areas/shared/current_authority_contract.yaml` | `current_authority.*` |
| `.ovav/service_areas/shared/lead_work_method_contract.yaml` | `execution.*`, `delivery.*` |
| `.ovav/service_areas/shared/context_economy_contract.yaml` | `execution.context_budget` |
| `.ovav/service_areas/shared/tool_access_policy.yaml` | `boundaries.allowed_tools`, `boundaries.denied_tools` |
| `.ovav/service_areas/shared/visual_delivery_contract.yaml` | `delivery.*` |
| `.ovav/service_areas/shared/operational_memory_contract.yaml` | `memory.active_decisions`, `memory.deprecated_beliefs` |
| `.ovav/registry/delegation_rules.yaml` | `execution.delegation_mode` |
| `.ovav/policy/permission_authority.json` | `boundaries.requires_approval` |

---

## Propiedades Obligatorias

1. **Compacto.** <2KB YAML. No carga docs completos.
2. **Portable.** Mismo packet para DeepSeek, Claude, Gemini. El provider abstraction layer traduce.
3. **Validable.** Al compilar, verifica contra contratos fuente. Si un contrato cambió, el packet se invalida.
4. **Sin raw chat.** No incluye historial de conversación. Solo hechos operativos confirmados.
5. **Actualizable.** El work ledger y feedback loop actualizan `active_decisions` y `deprecated_beliefs`.

---

## Flujo de Compilación

```text
1. CARGAR CONTRATOS
   ├── Leer todos los YAML fuente
   ├── Verificar integridad (no corruptos, no vacíos)
   └── Detectar drift (contrato modificado desde último packet)

2. RESOLVER IDENTIDAD
   ├── service_area → lead → authority_level → visible_style
   └── Si hay conflicto entre contratos, doc_authority_matrix decide

3. RESOLVER AUTORIDAD ACTUAL
   ├── git rev-parse → repo + branch + commit
   └── current_authority_contract → stage + priority

4. RESOLVER MEMORIA
   ├── work_ledger → recent_lessons
   └── permission_authority → blocked_surfaces

5. RESOLVER LÍMITES
   ├── source_registry → allowed_context + denied_context
   ├── tool_access_policy → allowed_tools + denied_tools + requires_approval
   └── Aplicar deny-before-allow

6. RESOLVER EJECUCIÓN
   ├── task_classifier → task_size + risk_level
   ├── context_economy → context_budget + model_tier
   └── delegation_rules → delegation_mode + validation_mode

7. EMITIR PACKET
   ├── YAML <2KB
   ├── Hash de integridad (SHA256 de contratos fuente)
   └── Timestamp de compilación
```

---

## Qué NO es el Packet

- ❌ No es un prompt del sistema (es más pequeño y portable)
- ❌ No es un archivo de configuración estático (se compila por sesión)
- ❌ No es un volcado de memoria (solo hechos operativos, no raw chat)
- ❌ No es específico de un modelo (funciona en DeepSeek, Claude, Gemini)
- ❌ No reemplaza los contratos (los contratos son la fuente; el packet es la proyección)
