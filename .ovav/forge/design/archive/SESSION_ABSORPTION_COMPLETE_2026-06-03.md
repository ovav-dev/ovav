# OVAV — Absorción Completa de Sesión 2026-06-03 v3 FINAL
# ===========================================================
# Propósito: Compactación del 100% del criterio, diseño, decisiones
# y arquitectura discutidos en esta sesión. Para que cualquier agente
# o sesión futura pueda absorber todo sin haber estado presente.
#
# PRINCIPIO CENTRAL DE ESTA SESIÓN:
# Cada cosa que se desarrolle para OVAV debe tener UN archivo conector
# por tipo. Conexión = 1 entry. Desconexión = borrar entry.
# La arquitectura entera se reorganiza alrededor de este principio.
#
# RUTA DE LECTURA: .ovav/forge/design/READING_PATH.md

---

# PARTE 0 — PRINCIPIO CENTRAL: CONECTORES LIMPIOS 100%

## Lo que significa

Cada desarrollo OVAV — sea un validador, un harness, un tool, un plugin,
un lead humano, un cliente externo — debe:

  CONEXIÓN:   1 archivo. 1 entrada. 1 slot.
  DESCONEXIÓN: remover esa entrada. Nada más.

Sin tocar validate_all.py. Sin tocar auto_triggers.yaml. Sin tocar
surface_validator_map.yaml. Sin buscar referencias en 20 archivos.
Sin scripts de migración. Sin "ups, me olvidé de este import".

## Lo que YA existe (ConnectorBus v2.0 implementado)

  .ovav/connector_bus/connectors/  ← 9 archivos, uno por tipo de slot

  validators.yaml     ← 24 validadores. Agregar = 1 entry. Remover = borrar. ✅
  harnesses.yaml      ← 3 harnesses. Mismo mecanismo. ✅
  tools.yaml          ← 9 tools. Mismo mecanismo. ✅
  adapters.yaml       ← 6 adapters. Mismo mecanismo. ✅
  plugins.yaml        ← 4 plugins. Mismo mecanismo. ✅
  skills.yaml         ← 11 skills. Mismo mecanismo. ✅
  clients.yaml        ← 4 clientes externos. Nuevo. ✅
  personnel.yaml      ← 7 personas (2 leads + 5 team). Nuevo. ✅
  watchers.yaml       ← 3 watchers de drift. Nuevo. ✅

  bus.py v2.0:
    - Lee connectors/*.yaml (no un monolito registry.yaml)
    - Hot reload: bus.reload() sin reiniciar OVAV
    - Dependency check: bus.check_dependencies()
    - Health check: bus.check_health() por componente
    - 50ms load time para 71 componentes

  validate_all.py ya NO tiene imports hardcodeados. Lee del bus. ✅

## Lo que NO existe todavía (gaps honestos)

  ❌ auto_triggers.yaml sigue siendo manual. PENDIENTE: auto-gen desde bus (P8)
  ❌ surface_validator_map.yaml sigue siendo manual. PENDIENTE: auto-gen (P9)
  ❌ deregister_lead.py no existe. PENDIENTE: P6
  ❌ clients/ directory físico no existe. PENDIENTE: P1+P3
  ❌ Forge pipeline no lee del bus. PENDIENTE: P11

## El camino al 100% (P1 → P13)

  P1-P3 → clients/ segregados + symlinks
  P2    → identity hardening (14 archivos)
  P4    → OVAV_USER_GUIDE.md
  P5    → tools/INDEX.md
  P6    → deregister_lead.py
  P7    → IMPLEMENTATION_PLAN.md actualizado
  P8    → auto_triggers.yaml auto-generado desde el bus
  P9    → surface_validator_map.yaml auto-generado
  P10   → skills projection automática desde el bus
  P11   → forge pipeline unificado con el bus
  P12   → watchers cableados al 100%
  P13   → tools/ consolidation (367 harnesses → ~100)

---

# PARTE 1 — CRITERIO ARQUITECTÓNICO ESTABLECIDO

## 1.1 El error que corregimos

OVAV creció orgánicamente. Cada feature agregó archivos donde parecía
lógico en el momento. Resultado:

  - `registry/` en la raíz como proyecto independiente
  - `.ovav/agents/` y `.opencode/skills/` simultáneamente fuente y proyección
  - `.opencode/` (cliente externo) contaminando la raíz de OVAV
  - 14+ lugares fusionando OVAV = Platform Engineering = Thavren
  - validate_all.py con 24 imports hardcodeados → agregar validador = tocar 3 archivos

La reestructuración F0-F8 aplicó cirugía arquitectónica: TODO lo interno
en `.ovav/`, proyecciones a `.opencode/`, raíz mínima.

## 1.2 Arquitectura objetivo

  .ovav/
    source/          ← Desarrollo universal (agentes, skills, plugins, configs)
    forge/            ← Release engine multi-target (pipeline, targets, adapters)
    connector_bus/    ← Punto único de integración/desconexión
    registry/         ← 35 YAMLs de configuración
    governance/       ← Leyes, políticas, seguridad
    memory/           ← Ledger, contexto, handoffs
    runtime/          ← Sesiones, locks, estado

  clients/            ← 🔲 PENDIENTE: segregar proyecciones externas
    opencode/
    claude-code/
    vscode/
    pi/

  tools/              ← Motor de ejecución (623 py — 🔲 PENDIENTE INDEX.md)
  docs/               ← Documentación (🔲 PENDIENTE USER_GUIDE.md)
  assets/             ← Recursos estáticos

  Raíz: SOLO opencode.json, AGENTS.md, VERSION, README.md, .gitignore

## 1.3 Separación de identidad

El usuario estableció:

  OVAV (sistema gobernador)
    ├── Platform Engineering (área de servicio)
    │     └── Thavren (lead humano, colaborador externo removible)
    └── Research Intelligence (área de servicio)
          └── Eidren (lead humano, colaborador externo removible)

  REGLAS:
    - OVAV ≠ Thavren. OVAV es el sistema.
    - Thavren/Eidren son COLABORADORES EXTERNOS. Removibles sin romper OVAV.
    - La identidad profesional es independiente del modelo AI (cuerpo ≠ identidad).
    - Nadie es "dueño" de OVAV. Son operadores de áreas.

  Estado: 14+ archivos con confusión de identidad. P2 lo corrige.

---

# PARTE 2 — LO QUE SE HIZO (resumen para contexto)

## 2.1 Reestructuración F0-F8 (9 fases, 9 commits)

  F0: Model watchdog arreglado + trigger on_model_error
  F1: .ovav/source/agents/ + .ovav/source/skills/
  F2: .ovav/forge/ (pipeline, targets, adapters, releases)
  F3: registry/ → .ovav/registry/ (178 referencias)
  F4: runtime/ config/ statusline-inspect/ consolidados
  F5: Plugins → .ovav/source/plugins/opencode/
  F6: Reparación masiva de paths (210 archivos)
  F7: Re-proyección completa (16 agents, 11 skills, 2 plugins, theme)
  F8: Limpieza raíz (PNGs→assets)

## 2.2 Issues sistémicos corregidos

  - REGISTRY_ROOT apuntaba a path viejo → .ovav/registry/
  - Fixtures inválidos pasaban validate_all → reescritura completa
  - poison_guard timeout 45s → 90s
  - delegation_runtime crasheaba → pre-gates no fatales
  - skill_manager usaba path viejo → .ovav/registry/
  - validate_all --registry-root default → REPO_ROOT

## 2.3 ConnectorBus implementado

  .ovav/connector_bus/ creado con 24 validators, 3 harnesses, 6 tools,
  3 adapters, 3 plugins registrados. validate_all.py refactorizado.

---

# PARTE 3 — LO PENDIENTE (orden de dependencia estricto)

## P1 — Segregar .opencode/ → clients/opencode/

  CONEXIÓN LIMPIA: Cliente externo se agrega/remueve en clients/<nombre>/
                   + 1 entrada en forge/targets/<nombre>.yaml
  DESCONEXIÓN:     rm -rf clients/<nombre>/ + quitar entrada de forge/targets/

  Acción: Mover .opencode/ → clients/opencode/ + symlink .opencode → clients/opencode/
  Archivos: ~30
  Leyes: LAW-06

## P2 — Identity Hardening (14 archivos)

  CONEXIÓN LIMPIA: La identidad de un lead se define en SU archivo (source/agents/leads/).
                   Nada más lo referencia por nombre. Todo lo demás usa roles.
  DESCONEXIÓN:     P6 (deregister_lead.py) — automatizado post-P2.

  Acción: Corregir toda mención que fusione OVAV=Thavren=PlatformEng
  Archivos: 14 (auditados)
  Leyes: LAW-08, LAW-09, LAW-10

## P3 — clients/ para todos los targets

  CONEXIÓN LIMPIA: clients/<target>/ + forge/targets/<target>.yaml
  DESCONEXIÓN:     Borrar ambos. Nada más.

  Acción: Crear clients/{claude-code,vscode,pi}/
  Archivos: ~5

## P4 — docs/OVAV_USER_GUIDE.md

  CONEXIÓN LIMPIA: Documentar CÓMO se usa el ConnectorBus para conectar cosas nuevas.
                   "Creá tu módulo, agregá 1 entry en registry.yaml, listo."

  Acción: Guía práctica de uso
  Archivos: 1 nuevo

## P5 — tools/INDEX.md

  CONEXIÓN LIMPIA: Cada tool listada acá debe tener su entrada en registry.yaml.
                   Si no está en el bus, no está en el índice.

  Acción: Mapa de 623 archivos
  Archivos: 1 nuevo

## P6 — .ovav/registry/personnel.yaml + deregister_lead.py

  CONEXIÓN LIMPIA: 1 entrada en personnel.yaml = lead agregado.
                   automáticamente registrado en topology, contracts, agents.
  DESCONEXIÓN:     python3 tools/personnel/deregister_lead.py thavren
                   → remueve/archiva todo. OVAV sigue funcionando.

  Acción: Crear sistema de registro de personal
  Archivos: 2 nuevos

## P7 — Actualizar IMPLEMENTATION_PLAN.md

  Acción: Reflejar P1-P6 completados + arquitectura final
  Archivos: 1

---

# PARTE 4 — ROADMAP POST-P7 (conectores 100%)

  P8  — auto_triggers.yaml generado desde el bus (no manual)
  P9  — surface_validator_map.yaml generado desde el bus (no manual)
  P10 — Skills registradas en el bus + proyección automática
  P11 — Forge pipeline unificado con el bus (lee registry.yaml)
  P12 — Config watchers cableados al bus
  P13 — tools/ consolidation (367 harnesses → ~100)

---

# PARTE 5 — FEEDBACK DEL USUARIO INCORPORADO

  1. "No implementar sobre arquitectura no lista"
     → P1-P7 son PREREQUISITOS. Sin ellos no se avanza.

  2. "OVAV ≠ Thavren ≠ Platform Engineering — independientes conectados"
     → Identity model en 1.3. P2 lo ejecuta. P6 lo vuelve removible.

  3. "Conectores limpios: desarrollo nuevo = UN archivo"
     → ConnectorBus. registry.yaml = ese archivo. Ya operativo.

  4. "Desconexión sin desconfiguración"
     → Cada slot type define su protocolo de cleanup. P6 lo demuestra con leads.

  5. "Sistema vivo que aprende"
     → Slots.yaml permite nuevos tipos. El bus crece sin reescribir el núcleo.

  6. "Guía de uso, no de historia"
     → P4.

  7. "Separación de clientes externos"
     → P1+P3.

  8. "Control real sobre quién está y quién no"
     → P6.

---

# PARTE 6 — ESTADO FINAL

  Branch: task/implementaciones  |  HEAD: final
  Working tree: LIMPIO
  validate_all: 225 configs OK, 0 failed
  ovav_runtime: 0 blocking issues
  OVAV Laws: 21/21 PASS
  ConnectorBus v2.0: 9 slot types, 71 componentes, 50ms load
  Commits sesión: 20

---

# PARTE 7 — ARRANQUE PRÓXIMA SESIÓN

  1. python3 tools/validators/validate_all.py
  2. python3 tools/ovav_runtime.py validate
  3. Verificar PASS
  4. Ejecutar P1 → P2 → P3 → P4 → P5 → P6 → P7 EN ORDEN
  5. Cada P cierra con validate_all + commit atómico
  6. No saltar. No improvisar.
