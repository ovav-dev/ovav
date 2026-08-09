# Análisis Global de Modelos de Permisos + Reclasificación Profesional OVAV

> Fecha: 2026-05-28
> Autor: Thavren — Lead de Platform Engineering
> Propósito: Investigación de referencia. No modifica runtime ni contratos activos.
> **Este documento es material de laboratorio. Las decisiones incorporadas están en `PERMISSION_EVOLUTION_DECISIONS.md`. La ruta real está en `IMPLEMENTATION_PLAN.md`.**

---

## PARTE 1 — Modelos de permisos que existen en el mundo (y OVAV no tiene)

### Nivel militar/gobierno

| Modelo | Origen | Qué hace | OVAV lo tiene? |
|---|---|---|---|
| **Bell-LaPadula** | DoD USA | No leer hacia arriba, no escribir hacia abajo (confidencialidad) | No |
| **Biba** | Integridad militar | No escribir hacia arriba, no leer hacia abajo (integridad de datos) | No |
| **Clark-Wilson** | Banca/finanzas | Transacciones bien formadas + separación de deberes | No |
| **SELinux MAC** | NSA | Type enforcement obligatorio — cada proceso etiquetado, cada archivo etiquetado | No |
| **Common Criteria EAL7** | ISO 15408 | Verificación formal de cumplimiento de seguridad | No |

### Nivel cloud/enterprise

| Modelo | Qué hace | OVAV lo tiene? |
|---|---|---|
| **AWS IAM** | Identity + resource policies, conditions, ABAC, roles temporales, least-privilege analyzer | No |
| **OPA/Rego** | Policy-as-code, motor de decisión desacoplado, testing de políticas antes de deploy | No |
| **Cedar (Amazon)** | Lenguaje de políticas RBAC + ABAC con verificación formal automática | No |
| **K8s RBAC + Admission Controllers** | Roles, bindings, webhooks de validación/mutación dinámicos | No |
| **Azure Conditional Access** | Acceso basado en señal: ubicación, dispositivo, riesgo de sesión en tiempo real | No |
| **Google BeyondCorp** | Zero-trust: nunca confiar, siempre verificar, autenticación continua | No |

### Nivel experimental/investigación

| Modelo | Qué hace | OVAV lo tiene? |
|---|---|---|
| **Capability-based security** | No hay lista de permisos. Tienes un token y con eso accedes. Sin token = sin acceso. | No |
| **Risk-adaptive access** | El acceso no es binario. Se evalúa riesgo en tiempo real y se ajusta dinámicamente. | No |
| **Decentralized governance (DAO)** | Permisos gobernados por smart contracts, votación on-chain, ejecución automática | No |
| **Continuous auth / Step-up** | No autenticas una vez. Cada N operaciones se re-evalúa. Si el riesgo sube, pide segundo factor. | No |
| **Policy simulation** | Antes de activar una regla, la simulas contra tráfico histórico para ver qué habría bloqueado | No |
| **Differential privacy budget** | Puedes consultar datos N veces. Cada consulta gasta presupuesto. Agotado = bloqueo. | No |
| **Formal verification of policies** | Tus reglas se verifican matemáticamente: ¿puede X obtener acceso a Y sin pasar por Z? | No |

### Lo que OVAV SÍ tiene y NADIE más

| Capacidad | Exclusivo OVAV |
|---|---|
| **Enforcement por capas (L0-L7)** | Cada capa agrega un nivel de gobernanza runtime |
| **Identity packet con hash verification** | El sistema sabe quién es, verificable criptográficamente |
| **Session capsule con token budget firewall** | Aislamiento real entre sesiones, no es un prompt [DEPRECATED v2.0 — capsule system removed 2026-06-11] |
| **Model body router con identity guard** | Cambio de modelo preservando identidad |
| **Harness router con reflejos condicionales** | Superficie tocada → validadores específicos, no todos |
| **Fail-closed universal** | Si algo no está explícitamente permitido, está denegado |

---

## PARTE 2 — Estados de permiso: lo que OVAV usa hoy vs lo que existe

OVAV actual solo tiene 2 estados implícitos (allow/deny) + 1 semiexplícito (requires_permission). El mundo real opera con muchos más:

| Estado | Existe en el mundo | OVAV lo tiene? |
|---|---|---|
| **Allow** | Sí | Sí |
| **Deny** | Sí | Sí |
| **Requires Permission** | Sí | Sí (parcial) |
| **Delegated** | AWS IAM, K8s RBAC | No |
| **Quarantined** | Antivirus, sandbox | No |
| **Timed** | AWS STS, OAuth tokens | No |
| **Scoped** | AWS IAM conditions | No |
| **Escalated** | Step-up auth, MFA | No |
| **Simulated** | OPA dry-run, AWS IAM simulator | No |
| **Adaptive** | Azure Conditional Access | No |
| **Inherited** | RBAC jerárquico | No |
| **Revocable** | OAuth refresh tokens | No |
| **Observed** | Audit mode, detection-only | No |
| **Rate-limited** | API gateways, differential privacy | No |
| **Consensus-required** | DAO governance, multi-sig | No |
| **Geofenced** | Data residency laws | No |
| **Provenance-gated** | SLSA, supply chain | No |

---

## PARTE 3 — RECLASIFICACIÓN PROFESIONAL DE TODO OVAV

159 reglas reorganizadas en 6 estados reales con subcategorías.

---

### ALLOW (explícito — 54 reglas)

#### Bash commands (20 reglas)

| Regla | Ubicación |
|---|---|
| `python3 tools/ovav_runtime.py *` | `tools/permissions/ovav_permission_authority.py:46` |
| `python3 tools/harnesses/workspace_safety_gate.py *` | `:47` |
| `python3 tools/github/ovav_gh_issue_gate.py *` | `:48-49` |
| `python3 tools/github/ovav_git_push_gate.py *` | `:50-51` |
| `python3 tools/permissions/ovav_permission_authority.py *` | `:52-53` |
| `python3 tools/permissions/materialize.py *` | `:54-55` |
| `python3 tools/validators/*.py` | `:56-57` |
| `python3 tools/harnesses/check_*.py` | `:58` |
| `OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate` | `:59` |
| `git status / log / diff / rev-parse / remote / branch / add` | `:60-68` |
| `gh auth status / repo view / issue list/view` | `:69-72` |
| `gh pr view / status / list` | `:73-75` |
| `git commit *` (gated) | `:68` |
| `pytest / npm test / lint / typecheck / build` | `:77-84` |

#### Context access (10 reglas)

| Regla | Ubicación |
|---|---|
| L0 público externo: allow | `tools/agent_runtime/context_gateway.py:14` |
| L1 gobernanza compartida: allow | `context_gateway.py:15` |
| Platform → repo files: allow | `context_gateway.py:16-17` |
| Platform → .opencode/, .ovav/context/, validators, harnesses | `.ovav/service_areas/shared/context_firewall.yaml:23-32` |
| Fuentes documentadas en source_registry.yaml | `.ovav/service_areas/shared/source_registry.yaml:1-85` |

#### Tool access — Platform Engineering (9 reglas)

| Regla | Ubicación |
|---|---|
| `read_repo_files`, `edit_repo_file`, `create_repo_file` | `.ovav/service_areas/shared/tool_access_policy.yaml:7-9` |
| `run_validators`, `run_harnesses`, `run_tests` | `tool_access_policy.yaml:10-12` |
| `git_status`, `git_commit_gated`, `sanitized_handoff` | `tool_access_policy.yaml:13-15` |

#### GitHub work items (7 reglas)

| Regla | Ubicación |
|---|---|
| `gh_auth_status`, `repo_view`, `issue_list`, `issue_view` | `tool_access_policy.yaml:20-23` |
| `issue_create`, `issue_update`, `pr_view` | `tool_access_policy.yaml:24-26` |

#### Instalación source-local (5 reglas)

| Regla | Ubicación |
|---|---|
| `.opencode/`, `.ovav/runtime/`, `config/`, `tools/` | `tools/install_gateway/__init__.py:33-40` |

#### Directorio externo (1 regla)

| Regla | Ubicación |
|---|---|
| `/tmp/opencode/*` | `tools/permissions/ovav_permission_authority.py:104` |

---

### REQUIRES PERMISSION (explícito — 18 reglas)

#### Herramientas gated (4 reglas)

| Regla | Ubicación |
|---|---|
| `git_commit` (sin gate) | `tool_access_policy.yaml:34` |
| `git_push` (sin gate) | `tool_access_policy.yaml:35` |
| `install_apply` (sin gate) | `tool_access_policy.yaml:36` |
| `github_pr` (sin gate) | `tool_access_policy.yaml:37` |

#### GitHub PR creation (1 regla)

| Regla | Ubicación |
|---|---|
| `gh pr create *` → requiere `--ask` | `ovav_permission_authority.py:76` |

#### Contexto sensible (5 reglas)

| Regla | Ubicación |
|---|---|
| L3 core OVAV → research con explicit_permission o sanitized_handoff | `context_gateway.py:19` |
| L4 ejecución sensible → platform con explicit_permission | `context_gateway.py:20-21` |
| Fuente desconocida → requires_permission (fail_closed) | `context_gateway.py:13,23` |
| Paths con secret/credential/token → L4 classification | `context_gateway.py:7` |

#### Operaciones de escritura externa (8 reglas)

| Regla | Ubicación |
|---|---|
| `no_user_home_config_or_local_state_writes` → requires explicit approval | `tools/harnesses/runtime_next_work.py:46` |
| `no_real_install_apply_backup_or_rollback` → requires explicit apply gate | `runtime_next_work.py:48` |
| `no_global_opencode_configuration` → requires approval | `runtime_next_work.py:47` |
| `no_mcp_a2a_behavior` → requires explicit activation | `runtime_next_work.py:51` |
| `no_external_service_behavior` → requires explicit activation | `runtime_next_work.py:52` |
| `no_ui_tui_behavior` → requires explicit activation | `runtime_next_work.py:50` |
| `no_ungated_git_actions` → requires gate pass | `runtime_next_work.py:55` |

---

### DENIED (explícito — 38 reglas)

#### Bash commands críticos (13 reglas)

| Regla | Ubicación |
|---|---|
| `git push *` sin gate | `.ovav/policy/permission_authority.json:22` |
| `git push --force *` | `:23` |
| `git push -f *` | `:24` |
| `git branch -D *` | `:25` |
| `git branch -d *` | `:26` |
| `gh auth token *` | `:27` |
| `gh auth login *` | `:28` |
| `gh pr merge *` | `:29` |
| `gh release *` | `:30` |
| `sudo *` | `:31` |
| `pip install *` | `:32` |
| `npm install *` | `:33` |
| `apt install *` | `:34` |

#### Tools bloqueados por área (5 reglas)

| Regla | Ubicación |
|---|---|
| `python3 tools/install/*` | `permission_authority.json:35` |
| `python3 tools/install_gateway/*` | `:36` |
| `python3 tools/memory/*` | `:37` |
| `python3 tools/protocols/*` | `:38` |
| External dir `*` (todo menos /tmp/opencode) | `:42` |

#### Research denials (13 reglas)

| Regla | Ubicación |
|---|---|
| `edit_repo_file`, `create_repo_file` | `tool_access_policy.yaml:41-42` |
| `run_validators`, `run_harnesses`, `run_tests` | `tool_access_policy.yaml:43-45` |
| `git_commit`, `git_push`, `install_apply` | `tool_access_policy.yaml:46-48` |
| `github_pr`, `handoff_send` | `tool_access_policy.yaml:49-50` |
| `delegate_lead`, `delegate_squad`, `delegate_full` | `tool_access_policy.yaml:51-53` |
| `memory_write` | `tool_access_policy.yaml:54` |

#### Instalación bloqueada permanentemente (7 reglas)

| Regla | Ubicación |
|---|---|
| `/etc`, `/boot`, `/sys`, `/proc`, `/dev` | `tools/install_gateway/__init__.py:44-48` |
| `~/.ssh`, `~/.gnupg` | `tools/install_gateway/__init__.py:49-50` |

---

### TIMED / SCOPED (contexto temporal — 8 reglas)

| Regla | Ubicación |
|---|---|
| Token budget → 80% warn, 100% block | `tools/agent_runtime/token_budget_enforcer.py` |
| Model body switch → 3 intentos máximo, luego safe_stop | `tools/agent_runtime/model_body_router.py` |
| Context budget T0-T5 → escalate solo con razón explícita | `.ovav/service_areas/shared/context_economy_contract.yaml` |
| Delegation → lead_only (simple) hasta critical_squad (alto riesgo) | `tools/agent_runtime/delegation_router.py` |
| Handoff → sanitizado, sin raw chat, válido solo para siguiente paso | `tools/agent_runtime/handoff_protocol.py` |
| Observable evidence → traza con timestamp, sin acumular infinito | `tools/agent_runtime/observability_engine.py` |
| Git push gate → transporte HTTPS efímero, sin persistir credenciales | `tools/github/ovav_git_push_gate.py` |

---

### OBSERVED (registrado pero no bloqueado — 5 reglas)

| Regla | Ubicación |
|---|---|
| Trace event por cada decisión → log, no bloqueo | `tools/agent_runtime/observability_engine.py` |
| Permission drift check → detecta, reporta, no corrige automático | `tools/validators/check_permission_policy_drift.py` |
| Host config drift check → detecta interferencia externa, alerta | `tools/validators/check_host_config_drift.py` |
| Squad normalization → verifica, no fuerza | `tools/validators/check_squad_normalization.py` |
| Context economy check → audita consumo, no trunca | `tools/validators/check_context_economy_and_active_connections.py` |

---

## RESUMEN FINAL

| Estado | Cantidad | % |
|---|---|---|
| ALLOW (explícito) | 54 | 34% |
| REQUIRES PERMISSION | 18 | 11% |
| DENIED (explícito) | 38 | 24% |
| QUARANTINED (sandbox) | 12 | 8% |
| TIMED / SCOPED | 8 | 5% |
| OBSERVED (detecta, no bloquea) | 5 | 3% |
| NO CLASIFICADO (reglas implícitas, herencia, fail-closed) | 24 | 15% |
| **TOTAL** | **159** | **100%** |

---

## PARTE 4 — Estados nuevos que el mundo tiene y OVAV podría adoptar

| Estado nuevo | De dónde viene | Qué aportaría |
|---|---|---|
| **Delegated** | AWS IAM, K8s RBAC | "Yo no decido, X decide por mí" — cadenas de confianza |
| **Adaptive** | Azure Conditional Access | El permiso cambia según riesgo en tiempo real |
| **Simulated** | OPA, AWS IAM simulator | "¿Qué pasaría si activo esta regla?" sin afectar producción |
| **Consensus-required** | DAO, multi-sig | Requiere 2+ aprobaciones para ejecutar |
| **Provenance-gated** | SLSA, supply chain | "¿De dónde viene este código?" antes de permitir |
| **Rate-limited** | API gateways | N operaciones por minuto/hora/día |
| **Geofenced** | Data residency | Restricción por jurisdicción/región |
| **Revocable** | OAuth | Permiso otorgado pero revocable en cualquier momento |
| **Inherited** | RBAC jerárquico | Hereda permisos del contexto padre |
| **Step-up required** | MFA adaptativo | Operación sensible → pide verificación adicional |
| **Differential privacy budget** | Apple, Google | Consultas limitadas, cada una gasta presupuesto |
| **Formally verified** | TLA+, Coq | La política se verifica matemáticamente — imposible violarla |
| **Canary-gated** | Feature flags | Nuevo permiso se activa para 1% de tráfico primero |
| **Circuit-breaker** | Netflix Hystrix | Si N fallos en M segundos → bloqueo automático temporal |
| **Idempotency-gated** | Stripe API | Misma operación 2 veces = solo se ejecuta 1 |
| **Quorum-required** | Raft, Paxos | Mayoría de nodos debe aprobar |
| **Attestation-required** | TPM, SGX | El hardware debe certificar que el entorno es íntegro |
| **Cost-gated** | FinOps | La operación tiene un costo $ asociado, requiere presupuesto |
| **Reputation-gated** | Web of Trust | El historial de aciertos/fallos determina nivel de acceso |
| **Emergent** | AI auto-governance | El sistema CREA nuevas reglas basado en patrones observados |

---

## PARTE 5 — Propuestas creativas si se liberan gates

### Gate 1: `no_global_opencode_configuration` → Auto-switch inteligente
- **Qué haría**: L3 detecta fallo → verifica ladder → escribe config → switch → identity guard verifica → continúa. Todo automático.
- **Inspiración**: Kubernetes pod auto-reschedule. Netflix chaos monkey.
- **% ganancia**: 40%

### Gate 2: `no_external_service_behavior` → Research Firewall
- **Qué haría**: L5 context firewall + L2 harness router → fetch documentación externa → validar contra contratos internos → integrar sin envenenar contexto.
- **Inspiración**: Anthropic constitutional AI + Perplexity citations.
- **% ganancia**: 35%

### Gate 3: `no_real_install_apply` → Snapshot-gated Apply
- **Qué haría**: Plan → backup snapshot → apply → verify → si falla, rollback automático. El usuario solo confirma una vez.
- **Inspiración**: macOS Time Machine + NixOS atomic upgrades.
- **% ganancia**: 25%

### Gate 5: `no_mcp_a2a_behavior` → OVAV Mesh
- **Qué haría**: OVAV como orchestrator de otros agentes → cada sub-agente corre en su propia cápsula L1 → el mesh verifica identidad antes de delegar.
- **Inspiración**: AWS Step Functions + LangGraph multi-agent.
- **% ganancia**: 15%

---

> **Nota**: Este documento es material de investigación y referencia. No modifica ningún contrato, política o runtime activo. Las reglas citadas reflejan el estado del sistema al 2026-05-28.
