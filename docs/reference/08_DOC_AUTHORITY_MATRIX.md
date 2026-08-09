# Document Authority Matrix

Cuando fuentes entran en conflicto, esta matriz define cuál gobierna.

---

## Orden de Prevalencia

1. **`.ovav/plan/caps.yaml` + git HEAD** — fuente canónica de ruta estratégica, estado del sistema y next work. `IMPLEMENTATION_PLAN.md` es el dashboard renderizado derivado de caps.yaml. Para cualquier decisión de implementación, caps.yaml gobierna.
2. **Contratos YAML** en `.ovav/service_areas/shared/` — definen comportamiento runtime
3. **Permission authority** (`.ovav/policy/permission_authority.json`) — autoridad canónica de permisos
4. **Documentos del sistema** en `docs/system/` — arquitectura e identidad
5. **Documentos de inteligencia** en `docs/intelligence/` — diseño de packet, model router
6. **Documentos de runtime** en `docs/runtime/` — enforcement, capsule, gateways
7. **Documentos de implementación** en `docs/implementation/` — roadmap y segmentos
8. **Registry files** en `registry/` — estado operativo actual
9. **Runtime artifacts** en `.ovav/artifacts/` — evidencia histórica

---

## Conflict Resolution por Dominio

| Dominio | Fuente primaria | Fuente secundaria | Registry |
|---|---|---|---|
| **Identidad** | `docs/system/00_IDENTITY.md` | `.ovav/service_areas/areas/*.yaml` | `.ovav/registry/service_profiles.yaml` |
| **Arquitectura** | `docs/system/01_ARCHITECTURE.md` | `.ovav/plan/caps.yaml` | `.ovav/registry/phase_dag.yaml` |
| **Identity Packet** | `docs/intelligence/02_ACTIVE_IDENTITY_PACKET.md` | `tools/agent_runtime/identity_packet_compiler.py` | — |
| **Model Router** | `docs/intelligence/03_MODEL_BODY_ROUTER.md` | `tools/agent_runtime/model_body_router.py` | — |
| **Runtime Enforcement** | `docs/runtime/04_RUNTIME_ENFORCEMENT.md` | `tools/agent_runtime/*.py` | `.ovav/registry/harnesses.yaml` |
| **Seguridad** | `docs/security/06_SECURITY_FRAMEWORK.md` | `.ovav/policy/permission_authority.json` | `.ovav/registry/permissions.yaml` |
| **Implementación** | `.ovav/plan/caps.yaml` | `IMPLEMENTATION_PLAN.md` (dashboard) | — |
| **Contexto** | `.ovav/service_areas/shared/context_economy_contract.yaml` | `docs/26_RUNTIME_CONTEXT_BUDGET.md` | — |
| **Herramientas** | `.ovav/service_areas/shared/tool_access_policy.yaml` | `.ovav/policy/permission_authority.json` | — |
| **Memoria** | `.ovav/service_areas/shared/operational_memory_contract.yaml` | `.ovav/registry/memory_policy.yaml` |
| **Delegación** | `.ovav/registry/delegation_rules.yaml` | `.ovav/service_areas/shared/lead_work_method_contract.yaml` | — |
| **Delivery visual** | `.ovav/service_areas/shared/visual_delivery_contract.yaml` | `.ovav/service_areas/shared/lead_work_method_contract.yaml` | `.ovav/registry/result_contracts.yaml` |
| **Safe stop** | `.ovav/service_areas/shared/safe_stop_contract.yaml` | `docs/runtime/04_RUNTIME_ENFORCEMENT.md` | — |
| **Git** | `.ovav/policy/permission_authority.json` | `.ovav/service_areas/shared/tool_access_policy.yaml` | `.ovav/registry/harnesses.yaml` |
| **OpenCode** | `.opencode/agents/` + `.ovav/policy/permission_authority.json` | `docs/system/01_ARCHITECTURE.md` | — |

---

## Runtime Truth Rule

1. **Chat no es fuente de verdad.** Es solo un puntero a trabajo en progreso.
2. **Artifacts gobiernan el estado de implementación.** Los YAML en `.ovav/` definen la ley.
3. **Validators gobiernan decisiones de pass/fail.** Su veredicto es final.
4. **Registries reflejan estado operativo.** Una vez validados, son autoridad.
5. **Los docs en `docs/` explican el sistema.** Son la fuente de consulta humana.

---

## Docs Obsoletos (Eliminados)

| Archivo eliminado | Motivo |
|---|---|
| `docs/ovav/` (completo) | Histórico. El análisis fue válido en B23, reemplazado por L0-L4 + B23. |
| `docs/ARCHITECTURE.md` | 33 líneas. Reemplazado por `docs/system/01_ARCHITECTURE.md` (versión avanzada). |
| `docs/ROADMAP.md` | 23 líneas, solo mencionaba RC3. Reemplazado por `docs/implementation/07_IMPLEMENTATION_ROADMAP.md`. |
| `docs/GOVERNANCE.md` | 25 líneas. Reemplazado por `docs/runtime/04_RUNTIME_ENFORCEMENT.md`. |
| `docs/INTENDED_USAGE.md` | 27 líneas. Integrado en `docs/system/01_ARCHITECTURE.md`. |
| `docs/25_DOC_AUTHORITY_MATRIX.md` | Reemplazado por este archivo. |

---

## Documentación Vigente

```text
docs/
├── system/
│   ├── 00_IDENTITY.md                    ← Identidad, analogía humanista, perfiles P0
│   └── 01_ARCHITECTURE.md                ← Capas, runtime path, componentes
├── intelligence/
│   ├── 02_ACTIVE_IDENTITY_PACKET.md      ← Diseño del packet compilado portable
│   └── 03_MODEL_BODY_ROUTER.md           ← Cambio de cuerpo sin pérdida de identidad
├── runtime/
│   └── 04_RUNTIME_ENFORCEMENT.md         ← Gateways, harness router, defensas
├── security/
│   └── 06_SECURITY_FRAMEWORK.md          ← Zero-trust, poisoning defense, supply-chain
├── implementation/
│   └── 07_IMPLEMENTATION_ROADMAP.md      ← Roadmap por capas con done definitions
├── reference/
│   ├── 08_DOC_AUTHORITY_MATRIX.md        ← Este archivo
│   └── 09_SOURCE_REGISTRY.md             ← Clasificación L0-L4 de fuentes
├── contracts/                            ← Contratos AI-safe (conservados)
├── launch/                               ← Evidencia B18 (conservada)
├── workstation/                          ← WezTerm (conservado)
├── 26_RUNTIME_CONTEXT_BUDGET.md          ← Bootstrap activo (conservado)
└── README.md                             ← Entrada
```
