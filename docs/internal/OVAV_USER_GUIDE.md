# Guía de Uso — OVAV

> Guía práctica. No teoría, no historia del proyecto.
> Para entender la arquitectura a fondo: `IMPLEMENTATION_PLAN.md`
> Para absorber criterio y decisiones: `.ovav/forge/design/SESSION_ABSORPTION_COMPLETE_2026-06-03.md`

---

## 1. ¿Qué es OVAV?

OVAV es un **sistema gobernador de workstation AI**. No es un agente, no es un chatbot, no es un framework. Es una capa de gobernanza que orquesta agentes, validadores, seguridad, memoria y runtime sobre una workstation profesional.

OVAV opera bajo el principio **OVAV-first**: toda la arquitectura, herramientas y contratos se diseñan para OVAV. Los clientes externos (OpenCode, Claude Code, VS Code, PI) son proyecciones downstream — leen de OVAV, escriben al CLI. Ningún cliente es "el principal".

OVAV tiene **dos áreas de servicio profesional** con leads humanos independientes:
- **Platform Engineering** — lead: Thavren (runtime, seguridad, CLI, terminal, workstation)
- **Research Intelligence** — lead: Eidren (evidencia, fuentes, benchmarks, investigación)

---

## 2. Estructura de directorios

```
OVAV/
├── .ovav/                          ← 🧠 Cerebro (todo interno)
│   ├── source/                     ← Desarrollo universal (agentes, skills, plugins)
│   ├── forge/                      ← Release engine multi-target
│   ├── connector_bus/connectors/   ← 🔌 9 archivos YAML — integración molecular
│   ├── registry/                   ← 35 YAMLs de configuración
│   ├── runtime/                    ← Sesiones, locks, estado
│   ├── memory/                     ← Ledger, contexto, handoffs
│   └── ...
├── clients/                        ← 🖥️ Proyecciones a clientes externos
│   ├── opencode/                   ← OpenCode CLI (symlink .opencode → clients/opencode)
│   ├── claude-code/
│   ├── vscode/
│   └── pi/
├── tools/                          ← ⚙️ Motor de ejecución (ver tools/INDEX.md)
├── docs/                           ← 📚 Documentación
├── bin/                            ← ⌨️ CLI (ovav, ovav-shell, ovav-logo)
├── IMPLEMENTATION_PLAN.md          ← 🧬 Ruta estratégica — línea temporal única
├── README.md                       ← Público
├── CHANGELOG.md                    ← Tracking histórico
└── AGENTS.md                       ← Bootstrap OpenCode
```

---

## 3. Comandos esenciales

| Comando | Qué hace |
|---------|----------|
| `python3 tools/validators/validate_all.py` | Validación completa del sistema (225+ configs) |
| `python3 tools/ovav_runtime.py validate` | Validación de runtime con gates de seguridad |
| `python3 tools/ovav_runtime.py context --next` | Diagnóstico rápido (alias: ovav doctor) |
| `python3 tools/agents/project_opencode.py` | Regenerar proyección de agentes a OpenCode |
| `python3 tools/visual/project_opencode_visual.py` | Regenerar proyección visual (tema, plugins) |
| `python3 .ovav/forge/pipeline.py` | Release pipeline completo |
| `python3 tools/security/head_integrity_verifier.py --update` | Actualizar hash de integridad post-commit |

---

## 4. Flujo de trabajo diario

```
1. INICIAR SESIÓN
   python3 tools/agent_runtime/session_greeting.py --json
   python3 tools/validators/validate_all.py

2. VERIFICAR ESTADO
   python3 tools/ovav_runtime.py context --next
   → Revisar rama, working tree, next work

3. TRABAJAR
   → Editar archivos según el plan (IMPLEMENTATION_PLAN.md)
   → Cada paso atómico

4. VALIDAR
   python3 tools/validators/validate_all.py
   OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate

5. COMMIT
   git add <archivos exactos>
   git commit -m "tipo(scope): descripción"

6. PUSH
   python3 -B tools/github/ovav_git_push_gate.py
   git push
   python3 tools/security/head_integrity_verifier.py --update
```

---

## 5. Cómo integrar un componente nuevo (ConnectorBus — 3 pasos)

OVAV usa arquitectura **molecular**: 9 archivos YAML en `.ovav/connector_bus/connectors/`, uno por tipo de slot.

| Archivo | Tipo | Para |
|---------|------|------|
| `validators.yaml` | validator | Nuevo validador |
| `harnesses.yaml` | harness | Nueva verificación de runtime |
| `tools.yaml` | tool | Nuevo comando CLI |
| `adapters.yaml` | adapter | Nuevo adaptador a cliente |
| `plugins.yaml` | plugin | Nuevo plugin de UI |
| `skills.yaml` | skill | Nueva skill de agente |
| `clients.yaml` | client | Nuevo cliente externo |
| `personnel.yaml` | personnel | Nuevo lead o team member |
| `watchers.yaml` | config_watcher | Nuevo monitor de drift |

**Ejemplo: agregar un validador nuevo**

```
Paso 1: Crear tools/validators/check_mi_validador.py
        con una función validate() que retorne un dict

Paso 2: Agregar UNA entrada en connectors/validators.yaml:
        validate_mi_validador:
          module: tools.validators.check_mi_validador
          function: validate
          tier: 9
          triggers: [before_implementation]
          blocking: true

Paso 3: Nada más.
        validate_all lo carga automáticamente.
        auto_triggers lo dispara en before_implementation.
```

**Para desconectar:** borrar la entry. Nada más.

---

## 6. Cómo agregar o remover un Lead

Los leads y team members se registran en **UN solo archivo**: `.ovav/connector_bus/connectors/personnel.yaml`

**Agregar un lead:**
```yaml
  nombre_lead:
    role: platform_engineering_lead
    area: platform_engineering
    type: lead
    artifacts:
      - .ovav/source/agents/leads/nombre_lead.md
      - clients/opencode/agents/lead-nombre_lead.md
    active: true
    labels: [lead, plataforma]
```

**Remover un lead:**
```bash
python3 tools/personnel/deregister_lead.py nombre_lead
```
El script archiva todos los artefactos del lead. OVAV sigue funcionando.

---

## 7. Troubleshooting común

| Problema | Solución |
|----------|----------|
| `head_integrity: head_mismatch` | Normal después de commits. Ejecutar: `python3 tools/security/head_integrity_verifier.py --update` |
| `validate_all` lento o timeout | Aumentar timeout en el comando. El sistema tiene 225+ validaciones. |
| `session_context_guard: FAIL` | Archivos de gobernanza modificados externamente. Ejecutar diagnóstico: `python3 tools/security/session_context_guard.py --check --json` |
| `workspace_safety_gate: BLOCKED` | Rama protegida. Solo lectura. Requiere waiver del CEO. |
| Skills no se cargan en OpenCode | Ejecutar: `python3 tools/agents/project_opencode.py` |
| Tema visual no se aplica | Ejecutar: `python3 tools/visual/project_opencode_visual.py` |

---

## Rutas de lectura

| Ruta | Para | Secuencia |
|------|------|-----------|
| **A — ENTENDER** | Nuevo en OVAV | Este archivo → `IMPLEMENTATION_PLAN.md` → `connectors/` → `tools/INDEX.md` |
| **B — IMPLEMENTAR** | Ejecutar trabajo | `MASTER_ACTION_PLAN` → `clients.yaml` → `personnel.yaml` → `ovav_laws.yaml` |
| **C — ABSORBER** | Recuperar contexto | `SESSION_ABSORPTION_COMPLETE` (leer = haber estado en la sesión) |

---

*Última actualización: 2026-06-03 · Arquitectura molecular activa · Fase A en ejecución*
