# Investigación 2026 — Gobernanza de Agentes AI: Estado del Arte

> **Fecha:** 2026-06-03
> **Investigación:** Thavren + Research Mesh (Tavily + Brave Search)
> **Propósito:** Mapear sistemas top mundiales alineados con OVAV y extraer criterios aplicables.
> **Documento:** Laboratorio — no es plan de implementación.
> **Fuentes:** 20+ resultados de búsqueda, 5 fuentes fetcheadas para análisis profundo.

---

## 1. Microsoft Agent Governance Toolkit ⭐⭐⭐⭐⭐

**Fuente:** [GitHub](https://github.com/microsoft/agent-governance-toolkit) | [Blog](https://opensource.microsoft.com/blog/2026/04/02/introducing-the-agent-governance-toolkit-open-source-runtime-security-for-ai-agents/)
**Fecha:** Abril 2026
**Licencia:** Open source (MIT)

### Arquitectura

```
Agent ──► Policy Engine ──► Identity ──► Audit Log
              (YAML/OPA/Cedar)   (SPIFFE/DID/mTLS)   (Tamper-evident)
                │                    │
                ├── Allowed ──► Tool executes
                └── Denied ───► Block + log
```

### Características clave

| Componente | Detalle |
|---|---|
| **Policy Engine** | Intercepta cada acción del agente antes de ejecución. Soporta YAML, OPA/Rego y Cedar. |
| **Latencia** | <0.1ms p99 — sub-milisegundo |
| **Identidad** | Zero-trust con SPIFFE, DID, mTLS |
| **Audit Log** | Tamper-evident (a prueba de manipulación) |
| **Estado** | Stateless — sin dependencia de memoria compartida |

### Relevancia para OVAV

Microsoft está construyendo exactamente lo que OVAV ya tiene: un governor runtime que intercepta acciones, aplica políticas, verifica identidad y registra auditoría. La diferencia: OVAV es un **governor de origen** (diseñado como sistema gobernador desde el concepto), mientras que AGT es un **toolkit de seguridad** (añadido a agentes existentes).

**Qué podemos aprender:**
- AGT usa **Cedar + OPA/Rego** como lenguajes de política — OVAV ya decidió adoptar OPA/Rego (Bloque 8.5, PERMISSION_EVOLUTION_DECISIONS.md)
- El patrón **stateless policy engine** con interceptación sub-ms es el estándar 2026
- La identidad zero-trust (SPIFFE/DID/mTLS) valida nuestra dirección con Identity Packet Compiler (L0)

---

## 2. Ecosistema de Protocolos Agente-Agente 2026 ⭐⭐⭐⭐

**Fuentes:** [Zylos Research](https://zylos.ai/research/2026-03-26-agent-interoperability-protocols-mcp-a2a-acp-convergence) | [Ecosystem Map](https://www.digitalapplied.com/blog/ai-agent-protocol-ecosystem-map-2026-mcp-a2a-acp-ucp) | [Ruh.ai Guide](https://www.ruh.ai/blogs/ai-agent-protocols-2026-complete-guide)
**Fecha:** Marzo-Mayo 2026

### Protocolos activos

| Protocolo | Propósito | Estado 2026 |
|---|---|---|
| **MCP** (Model Context Protocol) | Agente ↔ Herramienta | Estándar de facto (Anthropic) |
| **A2A** (Agent-to-Agent) | Multi-agente coordinado | Google, especificación abierta |
| **ACP** (Agent Communication Protocol) | Comunicación ligera entre agentes | Emergente |
| **UCP** (Universal Connector Protocol) | Integración universal | Experimental |

### Hito clave: Q3 2026

Se espera la **primera especificación conjunta MCP/A2A** — el primer estándar formal de interoperabilidad entre agentes. Esto define cómo los agentes se descubren, autentican y coordinan.

### Relevancia para OVAV

OVAV ya tiene un **modelo de mesh interno** (OVAV Mesh, propuesto en Gate 5 de PERMISSION_EVOLUTION_DECISIONS.md). La dirección de la industria confirma que:
- El protocolo A2A valida nuestro diseño de comunicación entre áreas (Platform Engineering ↔ Research Intelligence)
- La especificación conjunta Q3 2026 es nuestro deadline natural para tener OVAV Mesh diseñado
- MCP es el estándar para tool access — nuestro Tool Gateway ya sigue este patrón

---

## 3. Verificación en Runtime para Agentes AI 2026 ⭐⭐⭐⭐

**Fuente:** [The Backend Developers](https://thebackenddevelopers.substack.com/p/runtime-verification-for-ai-agents)
**Fecha:** 2026

### Principios

| Principio | Implementación 2026 | OVAV equivalente |
|---|---|---|
| **Event tracing** | Telemetría y audit logs en tiempo real | L4 Observability Engine + trace events |
| **Invariant checks** | Reglas que nunca deben violarse | F0.3 Runtime Integrity Monitor |
| **Policy enforcement** | Motor de políticas que bloquea antes de ejecutar | L5 Context Firewall + permission_authority |
| **Sandbox execution** | Entorno aislado para código no verificado | Session Capsule (L1) |
| **Tamper-evident logs** | Logs que detectan manipulación | Hash chain en bootstrap_verifier.py |

### Relevancia para OVAV

La industria está convergiendo exactamente en el modelo de capas que OVAV diseñó:
- **L0**: Identity verification (AGT usa SPIFFE, OVAV usa Identity Packet Compiler)
- **L1**: Isolation (AGT usa sandboxes, OVAV usa Session Capsule)
- **L2-L5**: Policy enforcement en capas (AGT es monolítico, OVAV tiene 5 capas diferenciadas)

**Ventaja OVAV:** La separación en capas L0-L7 es más granular y gobernable que el modelo monolítico de AGT.

---

## 4. Herramientas Open Source para Gobernanza de Agentes 2026 ⭐⭐⭐

**Fuente:** [DEV Community](https://dev.to/jagmarques/5-open-source-tools-for-ai-agent-governance-in-2026-54le) | [Galileo](https://galileo.ai/blog/best-ai-agent-guardrails-solutions)
**Fecha:** Abril 2026

### Panorama competitivo

| Herramienta | Creador | Enfoque | Estrellas |
|---|---|---|---|
| **Agent Governance Toolkit** | Microsoft | Policy engine + identity + audit | Nuevo (Abr 2026) |
| **Guardrails Hub** | Community | Validadores de output para LLMs | 6.6K |
| **NeMo Guardrails** | NVIDIA | Guardrails conversacionales | 4K+ |
| **Superagent** | Open source | Framework con seguridad integrada | ~3K |
| **LlamaFirewall** | Meta AI | Firewall de seguridad para agentes | Nuevo |

### Qué hace cada uno

| Herramienta | Validación de output | Policy engine | Identity | Audit | Sandbox |
|---|---|---|---|---|---|
| **AGT (Microsoft)** | ✅ | ✅ OPA/Cedar/YAML | ✅ SPIFFE/DID | ✅ Tamper-evident | ✅ |
| **Guardrails Hub** | ✅ RAIL spec | ❌ | ❌ | ❌ | ❌ |
| **NeMo Guardrails** | ✅ Colang | ❌ | ❌ | ❌ | ❌ |
| **Superagent** | ✅ | ✅ básico | ❌ | ❌ | ✅ |
| **LlamaFirewall** | ✅ | ✅ | ❌ | ❌ | ✅ |

### Relevancia para OVAV

OVAV es el único sistema que integra **las 5 capacidades** (validación, policy engine, identity, audit, sandbox) en una arquitectura unificada de capas. Las herramientas listadas son especializadas en 1-3 capacidades.

**Riesgo competitivo:** Microsoft AGT está más cerca de OVAV que cualquier otra herramienta. Si AGT evoluciona hacia un governor completo, podría solaparse con el espacio de OVAV. La ventaja de OVAV es ser un **governor de origen** (no un add-on) y tener separación de áreas (Platform + Research).

---

## 5. Superagent y LlamaFirewall ⭐⭐⭐

**Fuentes:** [Help Net Security](https://www.helpnetsecurity.com/2025/12/29/superagent-framework-guardrails-agentic-ai/) | [Meta AI Research](https://ai.meta.com/research/publications/llamafirewall-an-open-source-guardrail-system-for-building-secure-ai-agents/)
**Fecha:** Diciembre 2025 / 2026

### Superagent

- Framework para construir, ejecutar y controlar agentes AI
- Seguridad integrada en el workflow (no añadida después)
- Enfoque: "construir con guardrails desde el inicio"

### LlamaFirewall (Meta)

- Guardrail system open-source enfocado en seguridad
- Diseñado para operaciones de agentes de alto riesgo
- Capa de firewall que intercepta acciones del agente

### Relevancia para OVAV

Ambos validan el principio fundacional de OVAV: **la seguridad no es un add-on, es la arquitectura**. La diferencia: ellos son frameworks para construir agentes seguros. OVAV es un governor para cualquier agente CLI.

---

## 6. Criterios extraídos — Aplicables a OVAV

### 6.1 Lo que OVAV ya tiene y la industria está adoptando

| Capacidad OVAV | Equivalente industria 2026 | Estado |
|---|---|---|
| L0 Identity Packet Compiler | SPIFFE/DID/mTLS (AGT) | ✅ OVAV pionero |
| L1 Session Capsule | Sandbox execution (AGT, Superagent) | ✅ OVAV pionero |
| L2 Harness Router | Event tracing + conditional routing | ✅ OVAV pionero |
| L3 Model Body Router | Sin equivalente directo | ✅ Exclusivo OVAV |
| L4 Observability Engine | Telemetry + audit logs | ✅ Estándar |
| L5 Context Firewall | Policy engine (AGT) | ✅ OVAV más granular |
| L6 Risk Scoring | Sin equivalente integrado | ✅ Exclusivo OVAV |
| L7 Feedback Loop | Sin equivalente | ✅ Exclusivo OVAV |

### 6.2 Lo que la industria tiene y OVAV puede adoptar

| Capacidad | Fuente | Aplicación OVAV |
|---|---|---|
| **Cedar policy language** | AWS / AGT | Alternativa a OPA/Rego para reglas simples. Evaluar adopción parcial. |
| **MCP protocol** | Anthropic | Estandarizar Tool Gateway como MCP server |
| **A2A protocol** | Google | Diseñar OVAV Mesh sobre spec A2A (Q3 2026) |
| **Tamper-evident audit logs** | AGT | Extender bootstrap_verifier hash chain |
| **SPIFFE identity** | Cloud Native | Evaluar para L0 v2 (Identity Packet Compiler avanzado) |
| **Sub-ms policy evaluation** | AGT (<0.1ms) | Benchmark OVAV policy engine latency |

### 6.3 Lo que NADIE tiene (ventaja OVAV)

1. **Gobernador de origen** — no es un add-on de seguridad, es la arquitectura misma
2. **Separación de áreas** — Platform Engineering + Research Intelligence como roles profesionales
3. **Model Body Router** — cambio de modelo preservando identidad (sin equivalente)
4. **Session Capsule con token budget** — aislamiento real, no prompt engineering
5. **Fail-closed universal** — lo no explícitamente permitido está denegado
6. **Criterion Compiler** — preserva criterio de ingeniería entre sesiones

---

## 7. Recomendaciones estratégicas

### Inmediato (esta sesión)
- [x] Documentar estado del arte 2026 en este brief
- [ ] Incorporar hallazgos al ARCHITECTURE_LAB.md (sección de materializador)

### Corto plazo (próximo segmento)
- [ ] Evaluar Cedar vs OPA/Rego para el materializador de superficies
- [ ] Diseñar OVAV Mesh sobre spec A2A (preparar para Q3 2026)
- [ ] Benchmark de latencia del policy engine OVAV vs AGT (<0.1ms)

### Mediano plazo
- [ ] SPIFFE/DID para L0 v2
- [ ] Tamper-evident audit logs con hash chain extendida
- [ ] MCP server wrapper para Tool Gateway OVAV

---

> **Principio:** OVAV no compite con estos sistemas. OVAV es un governor de origen — ellos son toolkits de seguridad. La investigación confirma que nuestra arquitectura de capas L0-L7 está adelantada. El riesgo no es quedarse atrás, es no ejecutar la ventaja que ya tenemos.
