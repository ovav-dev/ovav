# OVAV Documentation

OVAV es un **Professional AI Workstation Governor** — un sistema operativo de profesionales AI gobernados.

---

## Lectura Rápida

| Si querés entender... | Leé esto |
|---|---|
| Qué es OVAV realmente | [`system/00_IDENTITY.md`](system/00_IDENTITY.md) |
| Cómo está construido | [`system/01_ARCHITECTURE.md`](system/01_ARCHITECTURE.md) |
| El alma de OVAV (Identity Packet) | [`intelligence/02_ACTIVE_IDENTITY_PACKET.md`](intelligence/02_ACTIVE_IDENTITY_PACKET.md) |
| Cómo cambia de modelo sin perder identidad | [`intelligence/03_MODEL_BODY_ROUTER.md`](intelligence/03_MODEL_BODY_ROUTER.md) |
| Cómo se gobierna en runtime | [`runtime/04_RUNTIME_ENFORCEMENT.md`](runtime/04_RUNTIME_ENFORCEMENT.md) |
| Cómo aísla sesiones | [`runtime/05_SESSION_CAPSULE.md`](runtime/05_SESSION_CAPSULE.md) |
| Cómo se defiende | [`security/06_SECURITY_FRAMEWORK.md`](security/06_SECURITY_FRAMEWORK.md) |
| Qué se construye y en qué orden | [`implementation/07_IMPLEMENTATION_ROADMAP.md`](implementation/07_IMPLEMENTATION_ROADMAP.md) |
| Qué doc manda sobre cuál | [`reference/08_DOC_AUTHORITY_MATRIX.md`](reference/08_DOC_AUTHORITY_MATRIX.md) |
| Qué fuentes puede leer cada perfil | [`reference/09_SOURCE_REGISTRY.md`](reference/09_SOURCE_REGISTRY.md) |
| Contratos de uso seguro con AI | [`contracts/`](contracts/) |
| Contexto runtime bootstrap | [`26_RUNTIME_CONTEXT_BUDGET.md`](26_RUNTIME_CONTEXT_BUDGET.md) |

---

## Estructura

```text
docs/
├── system/          ← El QUÉ: identidad y arquitectura
├── intelligence/    ← El CÓMO: identity packet, model router
├── runtime/         ← El ENFORCEMENT: gateways, capsule, harnesses
├── security/        ← La DEFENSA: zero-trust, anti-poisoning
├── implementation/  ← El PLAN: roadmap por capas
├── reference/       ← El MAPA: authority matrix, source registry
├── contracts/       ← Contratos AI-safe
├── launch/          ← Evidencia de launch
└── workstation/     ← Configuración de workstation
```

---

## Los Dos Perfiles P0

| Perfil | Lead | Service Area |
|---|---|---|
| **Platform Engineering** | Thavren | Gobernanza de repo, workstation, OpenCode, runtime, validación, seguridad |
| **OVAV Research Intelligence** | Eidren | Verificación, benchmarking, evidencia, scoring, decisión |
