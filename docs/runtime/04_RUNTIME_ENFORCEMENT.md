# Runtime Enforcement — El Sistema Nervioso de OVAV

## Distinción Crítica: Host Runtime vs OVAV Runtime

```text
HOST RUNTIME                    OVAV RUNTIME
────────────                    ────────────
OpenCode / agent execution      OVAV governance system
Puede limitar:                  Gobierna mediante:
  · steps                       · Service Area Router
  · tools                       · Context Gateway
  · time                        · Tool Gateway
  · actions                     · Delegation Router
                                · Handoff Protocol
                                · Validators y harnesses
                                · Policies y contratos

Cuando el Host Runtime interrumpe:
  → Safe Stop Report
  → Estado: PARTIAL / SAFE_STOP / READY_FOR_COMMIT

Cuando el OVAV Runtime deniega:
  → Decision trace
  → Razón documentada
  → El Host Runtime nunca puede sobreescribir OVAV
```

---

## Gateways — El Sistema de Atención y Acción

### Context Gateway (Atención / Foco)

```text
El contexto se trata como atención humana.
No todo lo que existe debe entrar a la mente activa.

REQUEST:
  · Service Area (platform_engineering | research_intelligence)
  · Source + Path
  · Reason

CLASSIFY:
  · L0_public_external → allow (todos)
  · L1_shared_governance → allow (todos)
  · L2_platform_internal → allow (platform) / deny (research)
  · L3_core_ovav_internal → allow (platform por tarea) / deny (research)
  · L4_sensitive_execution → deny (sin grant explícito)
  · unknown → fail_closed (requiere permiso)

ENFORCE:
  · Token budget T0-T5 por turno
  · Bloqueo si excede el tier
  · Context compression si es necesario
  · Deny-before-allow siempre
```

**Estado actual**: Clasifica correctamente (L0-L4). NO trackea tokens. NO enforcea budget. NO detecta context poisoning. NO comprime.

### Tool Gateway (Manos y Campo de Acción)

```text
Las herramientas no se conceden porque el agente las quiere.
Se conceden porque el estado, rol, riesgo y política lo permiten.

PLATFORM ENGINEERING:
  ✅ read_repo_files, edit_repo_files (por tarea)
  ✅ run_validators, git_status, sanitized_handoff
  ⚠️ git_commit (requiere workspace safety gate)
  ⚠️ git_push (requiere ovav_git_push_gate + confirmación)
  ⛔ install_apply, global_config_write (sin grant explícito)

RESEARCH INTELLIGENCE:
  ✅ public_source_research, benchmark_matrix, evidence_scoring
  ✅ read_shared_governance
  ⛔ edit_repo_files, git_write, install_apply (siempre denegado)
  ⚠️ read_repo_files (solo con permiso explícito o handoff)

REGLA: fail_closed. Herramienta desconocida = denegada.
```

**Estado actual**: Allowlist hardcodeado. NO scoring dinámico de riesgo. NO rollback tracking. NO auditoría de operaciones.

### Delegation Router (Toma de Decisiones)

```text
task_size + risk_level → delegation_mode

  micro / low     → lead_only
  simple / low    → skill_only
  medium / medium → focused_squad (1-3 roles)
  complex / high  → full_squad (3-6 roles)
  critical/ high  → critical_squad (validación estricta)

Triggers adicionales (desde delegation_rules.yaml):
  · >4 files para entender el flujo → delegar
  · >2 files no triviales tocados → delegar
  · Cambio de arquitectura → delegar
  · Cambio de seguridad/auth → delegar
  · >4 fuentes de research → delegar
```

**Estado actual**: If/elif por task_class. Ignora los triggers del YAML. No analiza file count ni tipo de cambio.

---

## Harness Router — Reflejos Condicionales

Los harnesses son reflejos, no tests manuales. Deben dispararse automáticamente según la superficie tocada.

```text
MAPPING superficie → validadores:

  .ovav/service_areas/ tocado
    → service governance validators
    → context firewall validators
    → delegation policy validators

  tools/agent_runtime/ tocado
    → runtime enforcement validators
    → context gateway cases
    → tool gateway cases

  .opencode/ tocado
    → OpenCode runtime wiring
    → visual delivery
    → permission drift

  registry/ tocado
    → registry validation
    → orphan agents/skills/harnesses check

  permissions/ tocado
    → permission authority
    → materialization
    → drift check

  memory/tool readiness tocado
    → advanced capability boundary check

REGLAS:
  · No correr todo siempre (lento, caro)
  · No correr nada (inseguro)
  · Disparo por git diff --name-only
  · Deduplicar validadores
  · Paralelizar donde sea seguro
```

**Estado actual**: ✅ Implementado. El Harness Router y los auto-triggers ya conectan superficies tocadas con validadores requeridos; el Knowledge Compiler ya usa esa evidencia como base para detectar patrones, drift de ruta, transiciones legítimas y criterios nuevos.

---

## Defensas Activas

### Context Poisoning Defense

```text
DETECTAR:
  · Stale docs (modificados hace >N días sin validación)
  · Wrong branch assumptions (path apunta a rama incorrecta)
  · Semantic overreach (L0 source pretende acceso a L3)
  · Prompt injection (instrucciones ocultas en contenido externo)

DEFENDER:
  · Source registry como allowlist
  · Deny-before-allow en Context Gateway
  · Fail-closed en fuentes desconocidas
  · Raw chat inheritance = prohibited
```

### Memory Poisoning Defense

```text
DETECTAR:
  · Inyección de creencias no confirmadas
  · Snapshot corrupto o manipulado

DEFENDER:
  · Solo crecer con hechos confirmados por el usuario
  · Sanitized YAML: sin raw chat, sin secrets, sin diffs sin resolver
```

---

## Safe Stop — Cuando el Host Falla

Cuando el Host Runtime (OpenCode) interrumpe por límite de steps/tools/tiempo:

```yaml
Safe Stop Report:
  State: PARTIAL | SAFE_STOP | READY_FOR_COMMIT
  Stop reason: (tool limit, step limit, timeout)
  Completed work: (qué se terminó)
  Pending work: (qué falta)
  Files changed: (lista exacta)
  Validators passed: (cuáles)
  Validators not run: (cuáles)
  Risk level: low | medium | high
  Exact next action: (comando o paso)
  Commit allowed: yes | no
```

**Estado actual**: Contrato definido en `safe_stop_contract.yaml`. No integrado en runtime Python.
