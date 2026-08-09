# OVAV SYSTEM CATALOG — Documentación Canónica

> Auditado: 2026-06-10 · Commit: b7bc370 · v1.0.0 RELEASED
> Este documento es autoridad. Cualquier doc que lo contradiga está obsoleto.

---

## 📊 MÉTRICAS DEL SISTEMA

| Dimensión | Cantidad |
|-----------|----------|
| **Total herramientas Python** | 722 |
| **Validadores** | 73 (73 auto-triggered, 0 sin trigger — 100% cobertura) |
| **Harnesses** | 385 |
| **Agentes** | 70 (.opencode/agents/) |
| **Skills** | 11 (.opencode/skills/) |
| **Comandos OpenCode** | 7 |
| **Plugins OpenCode** | 2 (ovav-monitor.js, ovav-status.js) |
| **Modelos configurados** | 8 en 5 proveedores |
| **Commit total** | 597 |
| **Release candidates** | 8 |

---

## 🏗️ ARQUITECTURA DE CAPAS (L0–L7)

| Capa | Sistema | Estado | Trigger |
|------|---------|--------|---------|
| **L0** | Active Identity Packet Compiler | ✅ | on_session_start |
| **L1** | Session Capsule v2 | ✅ | on_session_start |
| **L2** | Harness Router | ✅ | pre/post ops |
| **L3** | Model Body Router (switch real) | ✅ | credit exhaustion |
| **L4** | Observability Engine | ✅ | on_session_start |
| **L5** | Context Firewall v2 (5-check pipeline + hash) | ✅ | on_user_message |
| **L6** | Risk Scoring + Quarantine + Lockdown + Budget | ✅ | on_session_start, continuous |
| **L7** | Feedback Loop (governed operational memory) | ✅ | before_close, continuous |

---

## 🧩 SUBSISTEMAS PRINCIPALES

### 🧠 SNV — Sistema Nervioso Vivo
| Módulo | Estado | Función |
|--------|--------|---------|
| NerveBus | ✅ | Bus de comunicación entre módulos |
| KnowledgeGraph | ✅ | Grafo de conocimiento con pesos hebbianos |
| HebbianWeights | ✅ | Fortalecimiento/debilitamiento de conexiones |
| TemporalCortex | ✅ | Memoria temporal con decaimiento |
| PatternLearner | ✅ | Detección de patrones en interacciones |
| SNV Bridge | ✅ | Conexión SNV ↔ OVAV runtime |
| PainScorer D1 | ✅ | Clasificador de impacto multi-dimensional |
| Dashboard↔Lockdown | ✅ | Integración con sistema de bloqueo |
| **Total módulos** | **7** | **State: activo (state.kc)** |

### 💰 Economy Engine
| Herramienta | LOC | Función | Auto-trigger |
|-------------|-----|---------|-------------|
| `budget_governor.py` | 364 | Presupuesto $200/mes, $10/sesión, alertas 70/85/95%, auto-stop | ✅ on_session_start |
| `cost_tracker.py` | 463 | Tracking de tokens y costos por sesión | ✅ before_close |
| `cost_tracker_auto.py` | — | Auto-tracking continuo | ✅ |
| `cost_estimator.py` | — | Estimación pre-ejecución | ✅ |
| `provider_prices.yaml` | — | 8 modelos, 5 proveedores, cache discounts | ✅ on_session_start |
| `budget_status.json` | — | Estado actual de presupuesto | ✅ continuous |

**Proveedores y modelos:**
| Proveedor | Modelos | Tier | Cache |
|-----------|---------|------|-------|
| DeepSeek | V4 Pro, V4 Flash | premium/budget | 50% input |
| OpenAI | GPT-5.5, GPT-4o | ultra/premium | 50% |
| Anthropic | Claude Sonnet 4, Haiku 3.5 | premium/budget | 90% |
| Qwen | 3.5 Plus | budget | 0% |

### 🛡️ Defense Stack
| Herramienta | LOC | Función | Auto-trigger |
|-------------|-----|---------|-------------|
| `defense_cortex.py` | 550 | Cortex central de defensa | ✅ on_session_start |
| `lockdown_authority.py` | 432 | Bloqueo global por dominios | ✅ continuous |
| `quarantine.py` | 442 | Cuarentena de archivos | ✅ on_detect |
| `root_immune_list.py` | 289 | Whitelist inmutable (AGENTS.md, policy, vault) | ✅ continuous |
| `host_defense_responder.py` | — | Respuesta automática a intrusiones | ✅ |
| `host_defense_logger.py` | — | Log de eventos de defensa | ✅ |
| `exfil_detector.py` | — | Detección de exfiltración de datos | ✅ |
| `integrity_monitor.py` | — | Monitoreo de integridad continua | ✅ periodic_hourly |
| `session_context_guard.py` | — | Guardián de contexto de sesión | ✅ on_session_start |
| `gate_self_protection.py` | — | Auto-protección de gates | ✅ on_session_start |
| `living_integrity.py` | — | Modelo de integridad viva 5-capas | ✅ |
| `network_guard.py` | — | Rate-limiting, TLS, dominios | ✅ |
| `head_integrity_verifier.py` | — | Verificación de HEAD | ✅ on_session_start |
| `provenance_checker.py` | — | Verificación de procedencia | ✅ |
| `canonical_source_resolver.py` | — | Resolución de fuente canónica | ✅ |
| `implementation_guard.py` | — | Guardián de implementación | ✅ |
| `intelligent_authorizer.py` | — | Autorización inteligente | ✅ |
| `secrets_vault.py` | — | Bóveda de secretos | ✅ |
| `credential_governor.py` | — | Gobierno de credenciales | ✅ on_session_start |
| `credential_health.py` | — | Salud de credenciales | ✅ periodic_hourly |
| `credential_vault.py` | — | Vault de credenciales | ✅ |
| `sbom.py` | — | Software Bill of Materials | ✅ |
| `memory_zero.py` | — | Zeroización de memoria | ✅ |
| **Total** | **~2000+** | **33 herramientas** | **24/33 auto-triggered** |

### 🧠 Memory System
| Herramienta | LOC | Función | Auto-trigger |
|-------------|-----|---------|-------------|
| `memory_governor.py` | 742 | Gobernador de memoria OVAV | ✅ capsule-bound F5-gated |
| `pipeline.py` | 157 | Pipeline de consolidación de memoria | ✅ |
| `context_cut_detector.py` | 342 | Detección de cambio de tarea/contexto | ✅ on_session_start |
| `harness_bridge.py` | — | Puente Memory ↔ Harnesses | ✅ |
| `governor/` (dir) | — | Lógica de gobierno de memoria | ✅ |
| `signals/` (dir) | — | Señales de memoria | ✅ |

### 🔧 CLI Tools (18 herramientas)
| Herramienta | Función |
|-------------|---------|
| `ovav_install.py` | Instalación OVAV |
| `ovav_install_smoke.py` | Smoke test post-instalación |
| `ovav_first_run_cockpit.py` | ~~Cockpit de primer arranque~~ → Migrado a Go (`go-runtime/cmd/cockpit/`) |
| `ovav_fresh_clone_smoke.py` | Smoke test de clone fresco |
| `ovav_practical_smoke.py` | Smoke test práctico |
| `ovav_official_update.py` | Update oficial OVAV |
| `ovav_release_package.py` | Empaquetado de release |
| `ovav_public_export_gate.py` | Gate de exportación pública |
| `ovav_repo_presentation_gate.py` | Gate de presentación de repo |
| `ovav_plan_artifacts.py` | Planificación de artefactos |
| `ovav_surface_manager.py` | Gestión de superficies |
| `ovav_backup_manager.py` | Gestión de backups |
| `ovav_execution_gateway.py` | Gateway de ejecución |
| `ovav_tool_configs.py` | Configuración de herramientas |
| `ovav_tailor_composer.py` | ~~Compositor tailor~~ → Migrado a Go (`go-runtime/cmd/tailor/`) |
| `ovav_tui.py` | Terminal UI |
| `ovav_visual_theme.py` | Tema visual |
| `router.py` | Router CLI |

### 🎨 Visual System
| Artefacto | Estado |
|-----------|--------|
| `.ovav/visual/theme/theme.yaml` | ✅ |
| `.ovav/visual/monitoring/monitoring.yaml` | ✅ |
| `tools/visual/theme_engine.py` | ✅ |
| `tools/visual/release_pipeline.py` | ✅ (internal→external) |
| `tools/visual/project_opencode_visual.py` | ✅ |
| `tools/visual/monitor_engine.py` | ✅ |
| `.opencode/themes/ovav.json` | ✅ |
| `.opencode/plugins/ovav-monitor.js` | ✅ |
| `.opencode/plugins/ovav-status.js` | ✅ |
| `tui.json` | ✅ |

### 📋 OpenCode Commands (7)
| Comando | Función |
|---------|---------|
| `ovav-status` | Estado del sistema |
| `ovav-context` | Contexto actual |
| `ovav-validate` | Validación del sistema |
| `ovav-verify` | Verificación |
| `ovav-work` | Iniciar trabajo |
| `ovav-close` | Cerrar sesión |
| `ovav-refresh-skills` | Refrescar skills |

### 🧭 Knowledge Compiler P0
| Módulo | Estado |
|--------|--------|
| Pattern Detector | ✅ |
| Alignment Engine | ✅ |
| Transition Detector | ✅ |
| Criterion Compiler | ✅ |
| `compiler.py` (607 loc) | ✅ |
| `feedback_bridge.py` | ✅ |
| `seed_knowledge.py` | ✅ |

### 🌐 Research Mesh
| Motor | Estado |
|-------|--------|
| Brave Search | ✅ |
| Tavily | ✅ |
| DuckDuckGo | ✅ |
| SearXNG | ✅ |
| Capacidad | 2,000 búsquedas/mes |

### 🔌 ConnectorBus v2.0
| Conector | Entradas | Estado |
|----------|----------|--------|
| validators.yaml | 24 | ✅ |
| harnesses.yaml | 3 | ✅ |
| tools.yaml | 9 | ✅ |
| adapters.yaml | 6 | ✅ |
| plugins.yaml | 4 | ✅ |
| skills.yaml | 11 | ✅ |
| clients.yaml | 4 | ✅ |
| personnel.yaml | 7+ | ✅ |
| watchers.yaml | 3 | ✅ |

---

## ⚡ AUTO-TRIGGERS — Estado Actual

### Eventos del ciclo de vida
| Evento | Triggers activos | Cobertura |
|--------|-----------------|-----------|
| `on_session_start` | 25 | ✅ Completo |
| `before_implementation` | 20 | ✅ Completo |
| `before_close` | 8 | ⚠️ Algunos gaps |
| `on_user_message` | 2 | ⚠️ Bajo |
| `before_git_push` | 3 | ✅ |
| `before_git_stage` | 4 | ✅ |
| `after_git_stage` | 2 | ✅ |
| `periodic_hourly` | 5 | ✅ |
| `periodic_daily` | 2 | ✅ |
| `on_demand` | 6 | — |

### ❌ Validadores SIN auto-trigger — CORREGIDO (0 gaps)
> **Bloque A — 2026-06-10:** 22 validadores sin trigger → 100% cobertura.
> Todos los 73 validadores ahora tienen al menos un auto-trigger asignado.
> Eventos con cobertura mínima verificada: on_session_start(51), before_implementation(51),
> before_close(19), on_user_message(5), before_git_push(5), periodic_hourly(9), periodic_daily(5).

---

## 🔒 ESTADO DE DEFENSA ACTUAL

```
Lockdown: global_lockdown: false (CORREGIDO 2026-06-10)
Dominios activos: ninguno
SBOM: requirements.hash verificado — WARNING-only, no dispara bloqueo global
AGENTS.md: INTACTO (root immune list activo)
Fix aplicado: LockdownAuthority._load_state() + SBOM scope quirúrgico
```

---

## 📈 LO QUE FALTA — Mapa de gaps

### 🔴 Críticos (bloquean operación)
1. **SBOM requirements mismatch** → lockdown activo → I-034
2. **22 validadores sin auto-trigger** → brechas de seguridad
3. **Sin panel visual de uso** → control monetario ciego → I-016

### 🟡 Importantes (bloquean v2.0.0)
4. **CLI pública no compactada** → I-012
5. **Credenciales globales, no por proyecto** → I-019
6. **Gateway OVAV no existe** → routing manual → I-021
7. **Sin KPIs por LEAD** → I-031

### 🟢 Mejoras (v2.1+)
8. Memory inteligente sin integrar → I-017
9. Sin prompt caching → I-015
10. Sin capacidades multimedia → I-013, I-014

---

## 🗺️ RUTA RECOMENDADA

```
AHORA (P0):
  ├── I-034: Fix SBOM mismatch + scope quirúrgico de lockdown
  └── I-016: Panel visual de uso en tiempo real

v2.0.0 (P1):
  ├── I-012: CLI compacta para usuarios
  ├── I-019: Credenciales por proyecto
  └── Cerrar 22 gaps de auto-triggers

v2.1 (P2):
  ├── I-002: ovav update
  ├── I-021: Gateway OVAV
  └── I-017: Memoria inteligente integrada

v2.2+ (P3):
  └── Resto de exploración
```

---

_Última actualización: 2026-06-10 · Canonical source: `.ovav/service_areas/shared/current_authority_contract.yaml`_
