# tools/ — Índice de Herramientas OVAV

> Mapa navegable. Para el registro canónico de componentes activos, ver `.ovav/connector_bus/connectors/`

---

## Validadores (`tools/validators/`)

Verifican integridad, sintaxis, policy y seguridad. Registro canónico: `connectors/validators.yaml`

| Archivo | Qué verifica | Tier |
|---------|-------------|------|
| `validate_all.py` | Orquestador — carga dinámica desde ConnectorBus | 10 |

---

## Harnesses (`tools/harnesses/`)

Verificaciones de runtime por trigger o bajo demanda. Registro canónico: `connectors/harnesses.yaml`

| Archivo | Escenario | Trigger |
|---------|-----------|---------|
| `check_poison_guard_hardening.py` | Hardening del poison guard (timeout 90s) | on_session_start |
| `check_delegation_runtime.py` | Runtime de delegación entre agentes | before_implementation |
| `workspace_safety_gate.py` | Gate de seguridad del workspace | pre-write |

---

## Runtime (`tools/agent_runtime/`)

Motores de sesión, watchdog, triggers, inteligencia.

| Archivo | Función |
|---------|---------|
| `session_greeting.py` | Saludo de sesión — verifica integridad |
| `model_watchdog.py` | Monitoreo de modelos — auto-switch por credit exhaustion |
| `trigger_engine.py` | Motor de triggers automáticos (24 eventos) |
| `harness_router.py` | Router de harnesses por tipo de tarea |
| `context_gateway.py` | Gateway de contexto L5 — 5 checks |
| `injection_detector.py` | Detector de inyección en prompts |
| `token_budget_enforcer.py` | Control de presupuesto de tokens T0-T5 |
| `observability_engine.py` | Trazabilidad de operaciones |
| `identity_guard.py` | Guardián de identidad profesional |
| `session_handoff.py` | Transferencia de contexto entre sesiones |

### SNV — Sistema Nervioso Vivo

| Archivo | Función | Tamaño |
|---------|---------|--------|
| `nerve_bus.py` | Bus de eventos neuronales | 19 KB |
| `hebbian_weights.py` | Pesos hebbianos — aprendizaje asociativo | 18 KB |
| `temporal_cortex.py` | Corteza temporal — proyección de tendencias | 22 KB |
| `pattern_learner.py` | Aprendizaje de patrones entre sesiones | 29 KB |
| `snv_integration.py` | Puente de integración SNV ↔ OVAV | 27 KB |

---

## Seguridad (`tools/security/`)

Gates, integridad, defensa, credenciales.

| Archivo | Función |
|---------|---------|
| `intelligent_authorizer.py` | Autorizador inteligente (454 loc) |
| `secrets_vault.py` | Vault de secretos — encrypted at rest |
| `sbom.py` | Generador de SBOM |
| `network_guard.py` | Guardián de red — allowlist + TLS pinning |
| `implementation_guard.py` | Guardián de implementación |

---

## Permisos (`tools/permissions/`)

Gobernanza de permisos, sandbox, plugins y claims.

| Archivo | Función |
|---------|---------|
| `sandbox_governance.py` | Gobernanza de operaciones sandbox |
| `plugin_governance.py` | Gobernanza de plugins |
| `config_governance.py` | Gobernanza de configuración |
| `claims_governance.py` | Gobernanza de claims |
| `rego_engine.py` | Motor de políticas Rego |
| `ovav_permission_authority.py` | Autoridad de permisos canónica |

---

## CLI (`tools/cli/`)

| Archivo | Función |
|---------|---------|
| `ovav_surface_manager.py` | Gestor de superficies visibles |

---

## Proyección

| Directorio | Función |
|------------|---------|
| `tools/agents/project_opencode.py` | Adaptador OVAV → OpenCode (agentes) |
| `tools/visual/project_opencode_visual.py` | Adaptador OVAV → OpenCode (tema, plugins) |
| `tools/visual/theme_engine.py` | Motor de temas visuales |
| `tools/visual/release_pipeline.py` | Pipeline de release |
| `tools/visual/monitor_engine.py` | Motor de monitoreo |
| `tools/skills/skill_resource_index.py` | Índice de recursos de skills |

---

## Research Mesh (`tools/web/`)

| Archivo | Función |
|---------|---------|
| `search_gateway.py` | Gateway de búsqueda (Brave + Tavily + DDG + SearXNG) |
| `fetch_orchestrator.py` | Orquestador de fetch (10 workers) |
| `content_extractor.py` | Extractor de contenido HTML→texto |
| `research_cache.py` | Cache de investigación (24h búsquedas, 7d fetch) |

---

*623 archivos Python. Registro canónico de componentes activos: `.ovav/connector_bus/connectors/`*
