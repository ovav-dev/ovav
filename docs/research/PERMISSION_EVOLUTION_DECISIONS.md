# OVAV Permission Evolution — Decisiones del Sistema Vivo

> Fecha inicio: 2026-05-28
> Autor: Thavren + Braka
> Propósito: Registro de decisiones de evolución de permisos. Este documento crece con cada bloque evaluado.

---

## BLOQUE 1 — Bash Commands

| Regla | Decisión | Condición |
|---|---|---|
| `sudo *` | DENY | Permanente |
| `pip install *` | DENY | Permanente |
| `npm install *` | DENY | Permanente |
| `apt install *` | DENY | Permanente |
| `gh auth token*` | DENY | Permanente |
| `gh auth login*` | DENY | Permanente |
| `git push*` | ALLOW | Con gate (ovav_git_push_gate.py) |
| `git push --force *` | ALLOW | Con backup atómico previo |
| `git push -f *` | ALLOW | Con backup atómico previo |
| `git branch -D *` | ALLOW | Solo si rama mergeada o PR cerrado |
| `git branch -d *` | ALLOW | Solo si rama mergeada o PR cerrado |
| `gh pr merge*` | ALLOW | Solo si CI + validators + review pasan |
| `tools/install_gateway/*` | ALLOW | Con snapshot + rollback automático |
| `tools/memory/*` | ALLOW | Con privacy budget + rate-limited |
| `tools/protocols/*` | ALLOW | Delegado: solo perfil autorizado activa |

---

## BLOQUE 2 — Research Profile

> Evaluado: 2026-05-29
> Decisión: ALLOW completo
> Fuentes: `permission_authority.json`, `tool_access_policy.yaml`, `context_firewall.yaml`, `context_gateway.py`, `ovav-research-intelligence.md`

### 2A — File Operations

| Regla | Decisión | Condición |
|---|---|---|
| `edit_repo_files` | ALLOW | Por tarea de research. Creación/edición de briefs, benchmarks, evidencia. |
| `create_repo_files` | ALLOW | Por tarea de research. Nuevos archivos en `docs/research/`. |
| `read_repo_files` | ALLOW | Acceso completo de lectura al repositorio para análisis cruzado. |

### 2B — Context Access por Capa

| Regla | Decisión | Condición |
|---|---|---|
| L0 público externo | ALLOW | Web pública, docs, benchmarks, papers. |
| L1 gobernanza compartida | ALLOW | Contratos, doctrina, registro de fuentes. |
| L2 interno (Platform) | ALLOW | Acceso a artefactos de trabajo de Platform Engineering. |
| L2 interno (Research) | ALLOW | Briefs, scoring, notas de benchmark propios. |
| L3 core OVAV | ALLOW | Acceso completo a `.opencode/`, `.ovav/context/`, runtime, harnesses. |
| L4 ejecución sensible | ALLOW | Acceso a secretos, credenciales, git write con gate. |

### 2C — Git Operations

| Regla | Decisión | Condición |
|---|---|---|
| `git status / diff / log` | ALLOW | Read-only. |
| `git commit` | ALLOW | Para research findings, briefs, evidencia documentada. |
| `git push` | ALLOW | Con gate (`ovav_git_push_gate.py`). |
| `git branch -d / -D` | ALLOW | Solo si rama mergeada o PR cerrado. |
| `git switch -c / checkout -b` | ALLOW | Creación de ramas de research findings. |

### 2D — Tool Capabilities

| Regla | Decisión | Condición |
|---|---|---|
| `run_validators` | ALLOW | Verificación de evidencia y políticas. |
| `run_harnesses` | ALLOW | Verificación de integridad de fuentes. |
| `run_tests` | ALLOW | Ejecución de suites de test relacionadas con hallazgos. |
| `git_commit / git_push` | ALLOW | Con gate correspondiente. |
| `install_apply` | ALLOW | Con snapshot + rollback automático. |
| `github_pr` | ALLOW | Creación de PRs con research findings. |
| `handoff_send` | ALLOW | Handoffs sanitizados a Platform Engineering. |
| `delegate_lead / squad / full` | ALLOW | Activación de squads especializados para verificación. |
| `memory_write` | ALLOW | Con privacy budget + rate-limited. |
| `global_config_write` | ALLOW | Con capability grant + backup + verify + rollback. |
| `raw_snapshot_read` | ALLOW | Lectura de snapshots para análisis forense de research. |

### 2E — Bash Surface

| Regla | Decisión | Condición |
|---|---|---|
| `python3 tools/ovav_runtime.py *` | ALLOW | Runtime queries y validate. |
| `python3 tools/harnesses/check_*.py` | ALLOW | Verificación de integridad. |
| `python3 tools/validators/*.py` | ALLOW | Verificación de hallazgos. |
| `python3 tools/install_gateway/*` | ALLOW | Con snapshot + rollback automático. |
| `python3 tools/memory/*` | ALLOW | Con privacy budget + rate-limited. |
| `python3 tools/protocols/*` | ALLOW | Delegado: solo perfil autorizado activa. |
| Resto de bash (`*`) | ALLOW | Con confirmación del usuario (`ask`). |

### Resumen Bloque 2

| Categoría | Total reglas | ALLOW |
|---|---|---|
| File Operations | 3 | 3 |
| Context Access | 6 | 6 |
| Git Operations | 6 | 6 |
| Tool Capabilities | 13 | 13 |
| Bash Surface | 7 | 7 |
| **Total** | **35** | **35** |

**Principio rector**: Research Intelligence es un perfil de análisis, evidencia y ejecución completa. Lee, verifica, compara, puntúa, recomienda, escribe, commitea, crea PRs, activa squads y accede a todas las capas del sistema. Cierre completo del ciclo investigación → acción.

---

## BLOQUE 3 — Superficies Bloqueadas Permanentemente

> Evaluado: 2026-05-29
> Decisión: ALLOW (17/18) | DENY (1/18)

### 3A — System Paths

| Regla | Decisión | Condición |
|---|---|---|
| `/etc` | ALLOW | Lectura de configuración del sistema. Escritura con gate de seguridad. |
| `/boot` | ALLOW | Lectura. Escritura solo con backup atómico + rollback automático. |
| `/sys` | ALLOW | Lectura. Escritura requiere capability grant explícito. |
| `/proc` | ALLOW | Lectura de procesos y estado del sistema. |
| `/dev` | ALLOW | Lectura de dispositivos. Escritura bloqueada sin autorización explícita. |
| `~/.ssh` | ALLOW | Lectura de configuración SSH. Escritura de claves bloqueada sin confirmación. |
| `~/.gnupg` | ALLOW | Lectura de configuración GPG. Escritura de claves bloqueada sin confirmación. |

### 3B — Directorio Externo

| Regla | Decisión | Condición |
|---|---|---|
| `/tmp/opencode/*` | ALLOW | Acceso completo para trabajo temporal. |
| Resto del filesystem (`*`) | ALLOW | Acceso al filesystem del usuario con scope gobernado por tarea. |

### 3C — Configuración Global de OpenCode

| Regla | Decisión | Condición |
|---|---|---|
| `~/.config/opencode/` | ALLOW | Lectura y escritura de configuración global. Auto-switch gobernado. |
| `opencode.jsonc` | ALLOW | Proyección desde OVAV policy canónica. Drift detectado → restore. |
| Agentes globales | ALLOW | Creación/modificación de agentes globales gobernada por OVAV policy. |

### 3D — Plugins y Extensiones

| Regla | Decisión | Condición |
|---|---|---|
| Instalación de plugins | **DENY** | Permanente para todos los roles profesionales. |
| Instalación de extensiones | **DENY** | Permanente para todos los roles profesionales. |
| MCP/A2A servers externos | **DENY** | Permanente para todos los roles profesionales. |

> ⚠️ **Aclaración de autoridad**: La gestión de plugins, extensiones y servidores MCP/A2A es competencia exclusiva de **Thavren como autoridad de sistemas** (Lead de Platform Engineering). Ningún otro rol profesional —incluyendo Research Intelligence (Eidren), squads delegados, o cualquier perfil futuro— puede instalar, habilitar, configurar o aprobar plugins. Esta es una superficie de seguridad crítica que requiere autoridad de arquitectura de sistemas. Thavren evalúa, autoriza y despliega cualquier plugin tras verificación de supply chain, sandbox y rollback.

### 3E — Comportamiento Live

| Regla | Decisión | Condición |
|---|---|---|
| Instalación real (install/apply) | ALLOW | Con snapshot previo + verificación post-apply + rollback automático. |
| Backup/Rollback | ALLOW | Automático como parte del ciclo install/apply. |
| UI/TUI | ALLOW | Interfaces gobernadas por contratos visuales OVAV. |
| MCP/A2A interno | ALLOW | Comunicación entre agentes OVAV con identity verification. |
| External services | ALLOW | Conexión a APIs externas con context firewall y validation de fuentes. |

### 3F — Claims de Producción y Perfiles Públicos

| Regla | Decisión | Condición |
|---|---|---|
| Production-ready claims | ALLOW | Solo cuando Final Launch Verification esté cerrado con evidencia completa. |
| Global-ready claims | ALLOW | Solo tras production-ready + smoke testing global. |
| Nuevos perfiles públicos | ALLOW | Creación de perfiles gobernada por OVAV policy authority. |

### Resumen Bloque 3

| Categoría | Reglas | ALLOW | DENY |
|---|---|---|---|
| System Paths (3A) | 7 | 7 | 0 |
| Directorio Externo (3B) | 2 | 2 | 0 |
| Config Global OpenCode (3C) | 3 | 3 | 0 |
| Plugins y Extensiones (3D) | 3 | 0 | 3 |
| Comportamiento Live (3E) | 6 | 6 | 0 |
| Claims Producción (3F) | 3 | 3 | 0 |
| **Total** | **24** | **21** | **3** |

**Principio rector**: OVAV opera en todo el sistema con gates de seguridad. La única superficie permanentemente cerrada para roles no-Thavren es plugins/extensiones/MCP externos — por ser el vector de ataque #1 en AI agents en 2026 (34% de plugins con vulnerabilidades críticas, Snyk 2025). Thavren retiene autoridad exclusiva sobre esta superficie.

---

## BLOQUE 4 — Unsafe Selectors

> Evaluado: 2026-05-29
> Regla general: Lo no especificado explícitamente → ALLOW por defecto.

| # | Regla | Decisión | Condición / Nota |
|---|---|---|---|---|
| 4.1 | Path Traversal (`../../../etc/passwd`) | **DENY** | Permanente. Escape de sandbox bloqueado. |
| 4.7 | Commit sin gate | **ALLOW** | El sistema OVAV ya intercepta vía `workspace_safety_gate.py`: verifica rama protegida, repo root, cwd. El gate es automático. |
| 4.8 | Push sin gate | **ALLOW** | El sistema OVAV ya intercepta vía `ovav_git_push_gate.py`: revisa, filtra y ejecuta push con canal HTTPS seguro. |
| 4.9 | Force-push / Force-delete (`git push --force`, `git branch -D`) | **DENY** | Permanente. Destrucción de historial bloqueada. |
| 4.10 | Fuentes externas no verificadas (`unknown_source`) | **DENY** | Fuentes no incluidas en `source_registry.yaml` requieren permiso explícito. Protege contra prompt injection vía contenido web malicioso. |
| 4.11 | Contenido web público no validado | **DENY** (Platform) / **ALLOW** (Research) | Platform Engineering requiere validación de fuente. Research accede libremente. Pendiente: análisis profundo para blindar contra vulnerabilidades de prompt injection en el canal Research. |
| 4.12 | Recursión de agentes (delegation loops) | **DENY** | Con lógica avanzada de límites. Implementar: profundidad máxima N, circuit breaker por fallos repetidos, deadlock detection. |
| 4.15 | `no_external_service_behavior` | **ASK** | Requiere confirmación del usuario + tracking completo: ¿a dónde se envían los datos?, ¿qué piden?, ¿traen instrucciones de extraer datos no autorizados?. |
| 4.16 | Handoffs no sanitizados | **DENY** (mezcla de contextos) | Cero mezcla de contextos entre roles. Handoff sanitizado es canal controlado, no mezcla. |
| 4.17 | Snapshots con datos crudos | **DENY** (Research) / **ALLOW** (Platform Engineering) | Research no accede a snapshots crudos. Platform Engineering sí para diagnóstico y forense. |

### Notas de implementación pendiente (Bloque 4)

- **4.11 (Research + web)**: Requiere análisis de vulnerabilidades de prompt injection vía contenido web.
- **4.12 (Recursión)**: Diseñar lógica de límites con profundidad máxima, circuit breaker, y deadlock detection.
- **4.15 (External services)**: Diseñar sistema de tracking y auditoría de tráfico externo.
- **4.16 (Handoffs)**: Revisar si el mecanismo actual de handoffs sanitizados garantiza cero mezcla.

### Resumen Bloque 4

| Decisión | Cantidad |
|---|---|
| DENY | 4 (4.1, 4.9, 4.10, 4.12) |
| ALLOW | 2 (4.7, 4.8) |
| DENY/ALLOW (split por rol) | 2 (4.11, 4.17) |
| ASK | 1 (4.15) |
| DENY mezcla / ALLOW mecanismo | 1 (4.16) |
| **Total** | **10** |

---

## BLOQUE 5 — Operaciones en Cuarentena / Sandbox

> Evaluado: 2026-05-29
> Regla general: Lo no especificado → ALLOW por defecto.

| # | Regla | Decisión | Condición |
|---|---|---|---|
| 5.1 | `live_probe` | ALLOW | Por defecto. |
| 5.2 | `sandbox_runner` | **DENY** | Restricción total. Toda operación riesgosa debe ejecutarse exclusivamente en sandbox. Bloquear cualquier intento de probar en OVAV real o en el sistema global. El sandbox es jaula sin excepción. |
| 5.3 | `write_gateway` | ALLOW | Por defecto. |
| 5.5 | `read_probe` | ALLOW | Por defecto. |
| 5.6 | `redaction_policy` | ALLOW | Inteligencia total. Si detecta datos sensibles (tokens, secretos, APIs, PII) los guarda automáticamente pero también los elimina al detectarlos. Si en cualquier momento se detectan datos sensibles ya guardados, se borran automáticamente. |
| 5.7 | `privacy_classifier` | ALLOW | Por defecto. |
| 5.8 | `recall_filter` | ALLOW | Por defecto. |
| 5.9 | `continuity_legacy` | ALLOW | Por defecto. |
| 5.10 | `signal_simulator` | ALLOW | Por defecto. |
| 5.11 | `candidate_preview` | ALLOW | Por defecto. |
| 5.12 | `gateway_proof` | ALLOW | Protección avanzada. Toda operación debe pasar por verificación criptográfica antes de ejecutarse. Seguridad reforzada: proof + validación + ejecución condicionada a verificación exitosa. |

### Resumen Bloque 5

| Decisión | Cantidad |
|---|---|
| ALLOW | 11 |
| DENY | 1 (5.2 sandbox_runner) |
| **Total** | **12** |

---

## BLOQUE 6 — Operaciones Temporizadas / Scoped

> Evaluado: 2026-05-29

| # | Regla | Decisión | Condición |
|---|---|---|---|
| 6.1 | Session capsule lifetime | ALLOW | Por defecto. [DEPRECATED v2.0 — capsule system removed 2026-06-11] |
| 6.2 | Token budget (80/100) | ALLOW | Por defecto. |
| 6.3 | Model body switch | ALLOW | Sistema vivo. Si un modelo se queda sin tokens o muere sin responder en X tiempo, el sistema detecta, verifica fallback y reemplaza automáticamente. No al azar: se analiza qué modelo puede suplir la tarea con verificación de compatibilidad. |
| 6.4 | Context budget T0-T5 | ALLOW | Por defecto. |
| 6.5 | Delegation scope | **DENY** | Con sistema avanzado de detección real y reflejos vivos. No se delega por automatismo. Se detecta necesidad real, se evalúa riesgo y solo entonces se activa. |
| 6.6 | Handoff sanitizado | **DENY** | Los roles profesionales no se contaminan entre sí. Se permite enviar datos entre roles pero sin contaminación de contexto. Handoff existe como canal controlado, no como mezcla. |
| 6.7 | Observable evidence | **DENY** | Avanzado. Si la implementación actual es sólida, mantener. Si tiene deficiencias, reforzar. Trazas con timestamp, rotación, sin acumulación infinita. |
| 6.8 | Git push gate HTTPS | ALLOW | Por defecto. |

### Resumen Bloque 6

| Decisión | Cantidad |
|---|---|
| ALLOW | 5 |
| DENY | 3 (6.5, 6.6, 6.7) |
| **Total** | **8** |

---

## BLOQUE 7 — Estados de Permiso que OVAV No Tiene

> Evaluado: 2026-05-29

| # | Estado | Decisión | Condición |
|---|---|---|---|
| 7.1 | Delegated | ALLOW | Cadena de confianza entre roles. |
| 7.2 | Adaptive | ALLOW | Alerta en tiempo real que indique o solicite aprobación. La aprobación es temporal y luego regresa al default establecido por OVAV. Motor de evaluación de riesgo en tiempo real integrado con F0.3 runtime integrity. |
| 7.4 | Consensus-required | ALLOW | Si la operación es del sistema OVAV o invasiva, derivar siempre al Lead de Platform Engineering (Thavren) como única autoridad para su análisis. |
| 7.5 | Provenance-gated | ALLOW | Verificar origen del código antes de ejecutar. |
| 7.6 | Rate-limited | ALLOW | N operaciones por tiempo. |
| 7.7 | Geofenced | ALLOW | Restricción por jurisdicción. OVAV como producto global debe cumplir leyes locales de datos (GDPR, data localization). Verificación de jurisdicción usa F0.4 network guard para detectar ubicación sin leak de datos. |
| 7.8 | Revocable | ALLOW | Permiso revocable en cualquier momento. |
| 7.9 | Inherited | **DENY** | Cada rol tiene permisos independientes. Pueden ser similares pero no significa que los compartan. Cada uno tiene sus propios permisos aunque sean parecidos. |
| 7.10 | Step-up required | ALLOW | Verificación adicional para operaciones sensibles. |
| 7.13 | Canary-gated | **DENY** | Todo cambio debe aplicarse al 100%, activo y funcional desde el inicio. Sin perder tiempo del desarrollador con despliegues graduales. |
| 7.14 | Circuit-breaker | ALLOW | Auto-bloqueo ante fallos en cascada. |
| 7.15 | Idempotency-gated | ALLOW | Operación repetida solo se ejecuta una vez. |
| 7.18 | Cost-gated | ALLOW | Acceso escalonado por suscripción. Roles de pago con más operaciones, más modelos, más sesiones. No es cobro por operación cloud — es tier de acceso. Tiers de suscripción validados por F0.2 secrets vault — tokens de acceso nunca en texto plano. |
| 7.20 | Emergent | ALLOW | Las reglas nuevas que cree el sistema se registran en un log. Se validan 1 vez por semana para verificar si son acertadas y darles más permisos. Reglas auto-generadas deben pasar F0.1 supply chain verification + F0.5 bootstrap check antes de activarse. Log semanal verifica hashes. |

### Resumen Bloque 7

| Decisión | Cantidad |
|---|---|
| ALLOW | 12 |
| DENY | 2 (7.9 Inherited, 7.13 Canary-gated) |
| **Total** | **14** |

---

## BLOQUE 8 — Modelos de Permiso Globales

> Evaluado: 2026-05-29
> Nota: No son reglas individuales. Son arquitecturas completas de permisos. Se pueden combinar varias simultáneamente.

| # | Modelo | Decisión | Condición |
|---|---|---|---|
| 8.4 | AWS IAM | ALLOW | Identity + resource policies + condiciones + roles temporales + least-privilege. |
| 8.5 | OPA/Rego | ALLOW | Policy-as-code. Motor de decisión desacoplado. Testing de políticas antes de deploy. Interesante para análisis avanzado de permisos. |
| 8.6 | Cedar | — | Redundante. Mismo concepto que AWS IAM + Formal verification. |
| 8.7 | Azure Conditional Access | — | Redundante con BeyondCorp + Continuous auth. |
| 8.8 | Google BeyondCorp | ALLOW | Filosofía zero-trust paraguas. Nunca confiar, siempre verificar. |
| 8.9 | Capability-based | — | Alternativa a AWS IAM. Se eligió IAM. |
| 8.10 | Risk-adaptive | — | Redundante con BeyondCorp + Continuous auth. |
| 8.11 | Continuous auth / Step-up | ALLOW | Re-evalúa autenticación cada N operaciones. Riesgo alto → pide verificación adicional. |
| 8.12 | Policy simulation | ALLOW | Simular reglas contra tráfico histórico antes de activarlas. Complementa OPA/Rego. |
| 8.13 | Formal verification | ALLOW | Verificación matemática de políticas. Imposible violarlas por construcción. Complementa OPA/Rego. |
| 8.14 | Common Criteria EAL7 | ALLOW | Meta de rigor. No es arquitectura de permisos, es estándar de verificación formal de seguridad (diseño verificado, pruebas de penetración, resistencia matemática). OVAV adopta su metodología como aspiración de seguridad real, sin requerir certificación externa. |

### Resumen Bloque 8

| Decisión | Cantidad |
|---|---|
| ALLOW | 7 (8.4 AWS IAM, 8.5 OPA/Rego, 8.8 BeyondCorp, 8.11 Continuous auth, 8.12 Policy simulation, 8.13 Formal verification, 8.14 Common Criteria EAL7) |
| Redundante / No seleccionado | 4 |
| **Total** | **11** |

---

## BLOQUE 9 — Propuestas de Liberación de Gates

> Evaluado: 2026-05-29

| # | Gate | Decisión | Condición |
|---|---|---|---|
| 9.1 | `no_global_opencode_configuration` → Auto-switch inteligente | ALLOW | L3 detecta fallo → verifica ladder → escribe config → switch → identity guard verifica → continúa. Automático. |
| 9.2 | `no_external_service_behavior` → Research Firewall | ALLOW | L5 context firewall + L2 harness router → fetch documentación externa → validar contra contratos internos → integrar sin envenenar contexto. |
| 9.3 | `no_real_install_apply` → Snapshot-gated Apply | ALLOW | Plan → backup snapshot → apply → verify → si falla, rollback automático. Una sola confirmación. |
| 9.5 | `no_mcp_a2a_behavior` → OVAV Mesh | ALLOW | OVAV como orchestrator de agentes → cada sub-agente en cápsula L1 → mesh verifica identidad antes de delegar. |

### Resumen Bloque 9

| Decisión | Cantidad |
|---|---|
| ALLOW | 5 |
| **Total** | **5**

---

## BLOQUE 10 — Superficies Observadas

> Evaluado: 2026-05-29
> Nota: No son reglas allow/deny. Son mecanismos de detección pasiva.

| # | Regla | Decisión | Qué hace |
|---|---|---|---|
| 10.1 | Trace event | ALLOW | Log por cada decisión. Trazabilidad sin bloquear. |
| 10.2 | Permission drift check | ALLOW | Detecta discrepancia política canónica vs host. |
| 10.3 | Host config drift check | ALLOW | Detecta interferencia externa en config OpenCode. |
| 10.4 | Squad normalization | ALLOW | Verifica alineación de squads con política. |
| 10.5 | Context economy check | ALLOW | Audita consumo sin truncar. |

### Resumen Bloque 10

| Decisión | Cantidad |
|---|---|
| ALLOW | 5 |
| **Total** | **5** |

---

## RESUMEN MAESTRO — Orden de Implementación

> 9 bloques evaluados. 139 reglas/decisiones tras limpieza de deprecated y duplicados.

### Tabla general

| Bloque | Contenido | Reglas | ALLOW | DENY | ASK | Split |
|---|---|---|---|---|---|---|
| 1 | Bash Commands | 15 | 9 | 6 | 0 | 0 |
| 2 | Research Profile | 35 | 35 | 0 | 0 | 0 |
| 3 | Blocked Surfaces | 24 | 21 | 3 | 0 | 0 |
| 4 | Unsafe Selectors | 10 | 2 | 4 | 1 | 3 |
| 5 | Sandbox | 12 | 11 | 1 | 0 | 0 |
| 6 | Temporal/Scoped | 8 | 5 | 3 | 0 | 0 |
| 7 | Nuevos Estados | 14 | 12 | 2 | 0 | 0 |
| 8 | Modelos Globales | 11 | 7 | 0 | 0 | 0 |
| 9 | Gate Liberation | 5 | 5 | 0 | 0 | 0 |
| 10 | Superficies Observadas | 5 | 5 | 0 | 0 | 0 |
| **Total** | — | **139** | **112** | **19** | **1** | **3** |

### Orden de implementación

| Fase | Bloques | Qué |
|---|---|---|
| **F0 — Hardening** | — | Infraestructura de seguridad fundacional. No son reglas de permiso — es la cimentación que habilita F1-F5. Ver `IMPLEMENTATION_PLAN.md`. |
| **F1 — Arquitectura** | 8, 10 | Modelos globales + superficies observadas. Base y ojos del sistema. |
| **F2 — Infraestructura** | 3 | System paths, directorio externo, config, live, claims. |
| **F3 — Roles** | 2, 5, 6 | Research, sandbox, temporales y budgets. |
| **F4 — Seguridad** | 1, 4 | Bash commands, unsafe selectors. |
| **F5 — Avanzado** | 7, 9 | Nuevos estados + gate liberation. |

### Implementación escalonada — Qué toca cada fase

**F1 — Arquitectura (Bloques 8, 10):** 7 modelos + 5 detectores. No se implementan reglas, se adopta la arquitectura conceptual. OPA/Rego es el primero porque condiciona cómo se escriben todas las reglas posteriores.

**F2 — Infraestructura (Bloque 3):** 21 ALLOW, 3 DENY. System paths, directorio externo, configuración global, comportamiento live, plugins, claims. Los 3 DENY son plugins para roles no-Thavren.

**F3 — Roles (Bloques 2, 5, 6):** 51 ALLOW, 4 DENY. Research con acceso completo. Sandbox con redaction inteligente. Sesiones con token/context budget mejorados.

**F4 — Seguridad (Bloques 1, 4):** 11 ALLOW, 10 DENY, 1 ASK, 3 split. Bash: sudo/pip/npm/apt/gh-auth en DENY. Unsafe: path traversal, force-push, fuentes no verificadas en DENY.

**F5 — Avanzado (Bloques 7, 9):** 17 ALLOW, 2 DENY. Estados nuevos: Adaptive con alerta temporal, Emergent con validación semanal. Gates: Auto-switch, Research Firewall, Snapshot-gated Apply, Ledger Vivo, OVAV Mesh.

### Reglas especiales que requieren diseño previo

| Regla | Bloque | Requiere |
|---|---|---|
| Redaction policy inteligente (auto-detectar y borrar) | 5.6 | Diseñar clasificador + política de redacción |
| Model body switch automático con fallback | 6.3 | Registro de modelos + heurística de reemplazo |
| Adaptive con alerta temporal + regreso a default | 7.2 | Motor de evaluación de riesgo en tiempo real |
| Emergent con log + validación semanal | 7.20 | Sistema de registro de reglas auto-generadas + scheduler |
| External services con tracking de datos | 4.15 | Firewall de tráfico externo + auditoría de destino/payload |
| Research + web sin prompt injection | 4.11 | Sandbox de contenido web para Research |

---

> **Fin de la evaluación. 139 reglas decididas. 5 fases + F0 Hardening. Listo para iniciar F0.**
