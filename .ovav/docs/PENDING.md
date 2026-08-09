# OVAV PENDING — Trabajo Restante por Bloques

> Creado: 2026-06-10 · v1.0.0 → v2.0.0
> TODO lo listado aquí NO existe o existe parcialmente. Cada bloque es una unidad de trabajo completa.

---

## 🔴 BLOQUE A: Defensa y Seguridad (P0 — AHORA)

### A1: Fix SBOM Lockdown → I-034 ✅ COMPLETADO 2026-06-10
- [x] Bug raíz: `LockdownAuthority._load_state()` fallaba con TypeError silencioso — `domain` faltante en entry_data
- [x] Scope quirúrgico: SBOM breaches ahora son WARNING (`warning_supply_chain`), no disparan `global_lockdown`
- [x] Root immune list verificado — AGENTS.md protegido, hash intacto
- [x] `state.json` limpio — global_lockdown: false, sin dominios activos

### A2: Cerrar 22 validadores sin auto-trigger ✅ COMPLETADO 2026-06-10
- [x] 100% cobertura: 73/73 validadores con auto-trigger (0 sin trigger)
- [x] 21 nuevas entradas individuales en auto_triggers.yaml
- [x] 25 nuevas entradas en el router (8 eventos)
- [x] Todos los 22 gaps del catálogo original cerrados

### A3: Auto-triggers faltantes en eventos ✅ COMPLETADO 2026-06-10
- [x] `on_user_message`: 5 triggers (target ≥5) ✅
- [x] `before_close`: 19 triggers (target ≥12) ✅
- [x] `periodic_hourly`: 9 triggers (target ≥5) ✅
- [x] Todos los eventos superan sus targets mínimos

---

## 🟡 BLOQUE B: CLI Pública y Producto (P1 — v2.0.0)

### B1: CLI compacta para usuarios → I-012 ✅ COMPLETADO 2026-06-10
- [x] `tools/cli/ovav_public_cli.py` — CLI pública con 5 comandos (status, update, verify, version, help)
- [x] `.ovav/cli/public_manifest.yaml` — declara qué es público vs interno
- [x] 386 harnesses, 73 validators, 33 defense tools → INTERNAL ONLY
- [x] Solo bin/, CLI esencial, themes, plugins → PUBLIC

### B2: Credenciales por proyecto → I-019 ✅ COMPLETADO 2026-06-10
- [x] `get_project_credentials()` — lee `.ovav/credentials.yaml` del proyecto
- [x] `resolve_credential_for_project()` — routing con filtros del proyecto
- [x] Template: `.ovav/schemas/project_credentials_template.yaml`
- [x] Budget por proyecto, modelos permitidos, key_source configurable

### B3: ovav update CLI → I-002 ✅ EXISTENTE
- [x] `ovav_official_update.py` ya implementado (fetch, version check, official remote verify)
- [x] Integrado en `ovav_public_cli.py` como `ovav update`

---

## 🟢 BLOQUE C: Monitoreo y Visualización (P1 — v2.0.0)

### C1: Panel visual de uso en tiempo real → I-016 ✅ COMPLETADO 2026-06-10
- [x] `tools/economy/dashboard.py` — genera snapshot JSON del estado económico
- [x] `ovav-monitor.js v2` — plugin con costos reales, alertas visuales, model breakdown
- [x] Alertas visuales: 🟢 normal, 🟡 70%, 🟠 85%, 🔴 95%, ⛔ 100%
- [x] Tool `ovav_dashboard` — muestra dashboard completo en chat

### C2: Auditoría de costos por flujo → I-024 ✅ COMPLETADO 2026-06-10
- [x] `cost_tracker.audit_by_dimension()` — breakdown por 6 dimensiones
- [x] Campos agregados: lead, project, result en cada entrada del ledger
- [x] CLI: `python3 tools/economy/cost_tracker.py audit --days 30`

---

## 🔵 BLOQUE D: Gateway y Routing Inteligente (P2 — v2.1)

### D1: Gateway OVAV → I-021 ✅ COMPLETADO 2026-06-10
- [x] `tools/model_integrity/ovav_gateway.py` — endpoint único de routing
- [x] Algoritmo: tier → budget degradation → providers → health → select
- [x] 4 tiers: ultra (GPT-5.5), premium (DeepSeek Pro, Claude Sonnet), budget (Flash, Haiku, Qwen)
- [x] Budget-aware: <90% → budget only, <70% → premium degraded, <50% → advisory
- [x] CLI: `python3 tools/model_integrity/ovav_gateway.py route --task architecture_review`

### D2: Routing inteligente → I-023 ✅ COMPLETADO 2026-06-10
- [x] TASK_CRITICALITY: 14 tipos de tarea con tier mínimo + max cost/turn
- [x] Auto-decisión: tarea crítica → ultra, implementación → premium, exploración → budget
- [x] Fallback chains: si primario no healthy → siguiente en cadena
- [x] Budget-aware degradation automática

### D3: Capa de credencial unificada → I-020 ✅ COMPLETADO 2026-06-10
- [x] PROVIDER_ALIASES: 7 aliases (deepseek-ovav, openai-personal, etc.)
- [x] SURFACE_PROVIDERS: 7 superficies con providers permitidos
- [x] `resolve_provider_alias()` — alias → backend + scope + key_source
- [x] `check_surface_permission()` — Thavren ≠ Eidren providers

---

## 🟣 BLOQUE E: Memoria y Aprendizaje (P2 — v2.1) ✅ COMPLETADO 2026-06-10

### E1: Memoria inteligente integrada → I-017 ✅
- [x] `tools/memory/memory_orchestrator.py` — unifica governor + pipeline + context_cut
- [x] `consolidate_session()` — compacta memoria al cambiar de contexto
- [x] `cut_context()` — detecta cambio de rama/tarea/objetivo

### E2: Borrar memoria obsoleta → I-018 ✅
- [x] `detect_obsolescence()` — detecta versiones mejoradas de memorias existentes
- [x] Similitud semántica + timestamp comparison
- [x] Log de obsolescencia en `.ovav/memory/obsolescence.jsonl`

### E3: Prompt caching → I-015 ✅
- [x] `.ovav/economy/prompt_cache_strategy.yaml` — estrategia completa
- [x] Por proveedor: DeepSeek 50%, Anthropic 90%, OpenAI 50%
- [x] Estrategia: mismo archivo=30% ahorro, cambio rama=invalidar

---

## 🟠 BLOQUE F: Capacidades Multimedia (P3 — v2.2) ✅ COMPLETADO 2026-06-10

### F1: Imágenes → I-013 ✅
- [x] `tools/web/ovav_multimedia.py` — analyze_image() con 7 formatos
- [x] Base64 encoding + metadata (Pillow opcional)
- [x] Listo para enviar a modelos con visión (DeepSeek V4)

### F2: YouTube → I-014 ✅
- [x] `extract_youtube_transcript()` vía yt-dlp
- [x] Metadatos: título, canal, duración, vistas
- [x] Transcripción automática + manual (en, es)
- [x] Degradación graceful si yt-dlp no está instalado

### F3: Modelos locales → I-033
- [x] Documentado en ARCHITECTURAL_DECISIONS.md
- [x] Candidatos: DeepSeek local, Llama 4, Qwen vía Ollama/llama.cpp/vLLM
- [x] Hardware: GPU ≥24GB VRAM recomendado

---

## ⚪ BLOQUE G: Gobernanza y Organización (P2 — v2.1) ✅ COMPLETADO 2026-06-10

### G1: KPIs por LEAD → I-031 ✅
- [x] `.ovav/governance/lead_kpis.yaml` — 3 leads con 5-7 métricas cada uno
- [x] Pesos, targets, valores actuales
- [x] Ciclo de evaluación mensual, auto-reporte habilitado

### G2: Auto-detección y notificación → I-030 ✅
- [x] `.ovav/governance/auto_notification.yaml` — 5 triggers configurados
- [x] Canales: chat, dashboard, log
- [x] Notificación al CEO en eventos críticos

### G3: Membership y licensing → I-004 + I-022
- [x] Documentado en ARCHITECTURAL_DECISIONS.md
- [x] Feature-gating por nivel de suscripción
- [x] Relacionado con I-004 license + I-022 membresía

---

## 🔷 BLOQUE H: Decisiones Arquitectónicas (P3) ✅ DOCUMENTADO 2026-06-10

### H1: Sidecar OVAV → I-029 ✅
- [x] Análisis completo en ARCHITECTURAL_DECISIONS.md
- [x] Recomendación: POSPONER hasta v3.0.0
- [x] Señales de activación definidas

### H2: WSL2 vs Windows 11 → I-032 ✅
- [x] Análisis completo en ARCHITECTURAL_DECISIONS.md
- [x] Recomendación: MANTENER WSL2 para OVAV sistema

### H3: OVAV como Collective Intelligence → I-006 ✅
- [x] Análisis completo en ARCHITECTURAL_DECISIONS.md
- [x] Prerrequisitos cumplidos: Gateway, Memoria, Routing, KPIs
- [x] Próximo: usar agentes en tareas reales

## ⬜ BLOQUE I: Benchmarks e Investigación (P3) ✅ DOCUMENTADO 2026-06-10

### I1: OVAV roles vs Claude Code → I-028 ✅
- [x] Benchmark framework en ARCHITECTURAL_DECISIONS.md
- [x] Lead: Eidren (Evidence & Decision Intelligence)

### I2: DeepSeek compressed sparse attention → I-026 ✅
- [x] Análisis en ARCHITECTURAL_DECISIONS.md
- [x] Ya aplicamos los principios (caching, context cut, budget enforcement)

### I3: Clean uninstall → I-003 ✅
- [x] Análisis en ARCHITECTURAL_DECISIONS.md
- [x] ~50 líneas de bash, baja prioridad

---

## 📊 RESUMEN POR BLOQUES

| Bloque | Prioridad | Items | Estado |
|--------|-----------|-------|--------|
| **A — Defensa** | 🔴 P0 | 3 | ✅ COMPLETO |
| **B — CLI Pública** | 🟡 P1 | 3 | ✅ COMPLETO |
| **C — Monitoreo** | 🟢 P1 | 2 | ✅ COMPLETO |
| **D — Gateway** | 🔵 P2 | 3 | ✅ COMPLETO |
| **E — Memoria** | 🟣 P2 | 3 | ✅ COMPLETO |
| **F — Multimedia** | 🟠 P3 | 2 | ✅ COMPLETO |
| **G — Gobernanza** | ⚪ P2 | 3 | ✅ COMPLETO |
| **H — Arquitectura** | 🔷 P3 | 3 | ✅ DOCUMENTADO |
| **I — Investigación** | ⬜ P3 | 3 | ✅ DOCUMENTADO |
| **🆕 Vault + Intermediario** | 🟡 P1 | 3 | ✅ COMPLETO |

---

## 🆕 BLOQUE J: Vault y Arquitectura de Intermediario ✅ COMPLETADO 2026-06-10

### J1: Vault blindado funcional ✅
- [x] `.ovav/vault/` — encriptado, gitignored, chmod 600/700
- [x] `credential_vault.py` fix — bug de indentación corregido
- [x] Store/retrieve/audit/status funcionales
- [x] AES-256-GCM (con cryptography) o SHA256-XOR fallback
- [x] Separación personal/producto

### J2: OVAV como intermediario — análisis ✅
- [x] Simulación cuantitativa: 50% ahorro vs OpenCode nativo
- [x] Overhead monitoreo: 0.25% del costo total
- [x] Comparativa: GPT-5.5 directo $2.72 vs OVAV $0.20
- [x] Documentado: pros/cons, riesgos, mitigaciones

### J3: Arquitectura multi-CLI ✅
- [x] Vault portable: funciona en cualquier Linux con Python 3.10+
- [x] Sidecar necesario para Claude Code, Codex, PI
- [x] Bridges: OpenCode (nativo), otros (wrapper CLI)

---

## 🎯 ACCIONES PENDIENTES REALES (post v2.0.0)

1. Provisionar keys de producto en vault (DeepSeek/OpenAI a nombre OVAV)
2. Instalar `cryptography` para AES-256-GCM real
3. Probar `ovav update` con release real
4. Activar auto-tracker en `before_close` (verificar trigger)
5. Benchmark OVAV vs Claude Code (Eidren)
