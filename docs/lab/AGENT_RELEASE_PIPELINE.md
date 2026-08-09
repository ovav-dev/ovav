# OVAV Agent Release Pipeline — Diseño Completo

> **Fecha:** 2026-06-03
> **Estado:** Diseño arquitectónico. No implementado.

---

## Visión general

OVAV no "copia archivos a OpenCode". OVAV gobierna un ciclo de vida completo para cada mejora: desde la investigación que la origina hasta el release versionado que reciben los usuarios. Entremedio, una cadena de verificación ineludible garantiza que nada roto, nada inseguro y nada no aprobado llegue a producción.

```
CREACIÓN          STAGING         VERIFICACIÓN      REVISIÓN         RELEASE           MONITOR
(OVAV Brain)   (OVAV CLI test)   (gate ineludible)  (humano crítico)  (a CLI externo)   (L7 feedback)

   │               │                  │                  │                │                │
   ▼               ▼                  ▼                  ▼                ▼                ▼
.ovav/source/agents/  .ovav/cli/agents/  7 capas de tests   Thavren/Eidren   .opencode/agents/  L7 aprende
Investigación  Solo OVAV interno  Miles de checks    aprueban crítico  Versionado        Alertas
Arquitectura   Prueba manual      Si falla 1 → STOP  Sin firma → STOP  Rollback ready    Mejora continua
```

---

## CAPA 0 — CREACIÓN (OVAV Brain)

**Dónde:** `.ovav/source/agents/areas/`, `leads/`, `teams/`
**Quién:** Research Intelligence (Eidren) descubre. Platform Engineering (Thavren) implementa.
**Qué produce:** mejoras en agentes — nueva capacidad, mejor recepción de lectura, idiomas extra, más salidas, refuerzos de criterio.

```
Eidren investiga → brief de decisión
         │
         ▼
Thavren evalúa → diseña → implementa en .ovav/source/agents/
         │
         ▼
Knowledge Compiler → detecta impacto, propaga cambios, actualiza ledger
```

**Gates en esta capa:**
- Living Integrity: la fuente canónica no fue manipulada
- Permission Authority: solo leads pueden modificar agentes
- Session Capsule: cada cambio está aislado y trazado

---

## CAPA 1 — STAGING (OVAV CLI interno)

**Dónde:** `.ovav/cli/agents/`
**Comando:** `ovav stage --agents thavren,eidren`
**Qué hace:** proyecta agentes desde `.ovav/source/agents/` al entorno interno de OVAV. Este entorno ES OVAV — no es un CLI externo. Es donde OVAV se prueba a sí mismo.

```
.ovav/source/agents/areas/platform-engineering.md
.ovav/source/agents/leads/thavren.md
         │
         │  ovav stage
         ▼
.ovav/cli/agents/
├── platform-engineering.md    ← "Soy el área. Cargo a Thavren."
└── thavren.md                 ← "Soy Thavren. Identidad completa."
```

**Reglas del staging:**
- Solo visible para leads (Thavren, Eidren). Squads no acceden.
- Solo escritura vía `ovav stage`. No se edita a mano.
- Aquí se prueba: ¿el agente responde? ¿la nueva feature funciona? ¿rompe algo?
- Si no pasa prueba manual → vuelve a desarrollo. No avanza.

---

## CAPA 2 — VERIFICACIÓN AUTOMÁTICA (gate ineludible)

**Comando:** `ovav verify --staged`
**Regla:** **NO SE PUEDE SALTAR. NO EXISTE --skip. NO EXISTE --force.**

Si un solo check falla, el pipeline se detiene. No hay release sin verificación completa.

### Las 7 capas de verificación

```
┌─────────────────────────────────────────────────────────────┐
│                 ABRASIVE VERIFICATION SYSTEM                 │
│                                                             │
│  V1  SCHEMA        ──  ¿El archivo cumple el formato        │
│                        requerido por el CLI target?          │
│                        Frontmatter válido, campos presentes, │
│                        sin typos en mode/hidden/name.        │
│                                                             │
│  V2  STRUCTURE     ──  ¿La jerarquía es correcta?           │
│                        Áreas → mode:all, hidden:false.       │
│                        Leads → mode:subagent, hidden:false.  │
│                        Teams → mode:subagent, hidden:true.   │
│                        Sin duplicados. Sin huérfanos.        │
│                                                             │
│  V3  PERMISSIONS   ──  ¿Los permisos son compatibles        │
│                        con el CLI target?                    │
│                        OpenCode permite X, Claude permite Y. │
│                        Sin denegaciones contradictorias.     │
│                        Sin escalaciones no autorizadas.      │
│                                                             │
│  V4  CROSS-REF     ──  ¿Las referencias entre archivos      │
│                        son válidas?                          │
│                        Área → lead: ¿el lead existe?         │
│                        Routing lines: ¿apuntan a archivo     │
│                        que existe en el target?              │
│                                                             │
│  V5  CONTENT       ──  ¿El contenido es íntegro?            │
│                        Sin archivos vacíos.                  │
│                        Sin markdown roto.                    │
│                        Sin patrones de inyección.            │
│                        Sin secrets en texto plano.           │
│                        Sin referencias a paths internos.     │
│                                                             │
│  V6  REGRESSION    ──  ¿Este cambio rompe algo que ya       │
│                        funcionaba?                           │
│                        Compara con snapshot anterior.        │
│                        Detecta eliminaciones no intencionales│
│                        Detecta cambios de comportamiento.    │
│                                                             │
│  V7  BEHAVIORAL    ──  ¿El agente responde correctamente?   │
│                        Simulación de carga del agente.        │
│                        ¿Respeta su identidad?                │
│                        ¿Respeta sus permisos?                │
│                        ¿Produce output en el idioma correcto?│
│                                                             │
│  RESULTADO:  ████████ 7/7 PASS → avanza a revisión          │
│              ███████░ X/7 FAIL → BLOQUEADO. No avanza.      │
└─────────────────────────────────────────────────────────────┘
```

**Qué hace cada capa en detalle:**

| Capa | Checks | Ejemplos de fallo |
|---|---|---|
| V1 Schema | ~50 checks | `mode: alll` (typo), falta `name:`, `hidden:` no es bool |
| V2 Structure | ~30 checks | Área duplicada, lead con mode:all, team con hidden:false |
| V3 Permissions | ~80 checks | `sudo: allow` en target, permiso incompatible con CLI |
| V4 Cross-ref | ~20 checks | Routing a lead que no existe, área sin lead asignado |
| V5 Content | ~100 checks | Archivo vacío, secret en texto, path interno expuesto |
| V6 Regression | ~200 checks | Campo eliminado, comportamiento cambiado sin intención |
| V7 Behavioral | ~50 checks | Agente responde en inglés cuando debe español, ignora permisos |

**Total: ~530 verificaciones automáticas. Si 1 falla → STOP.**

---

## CAPA 3 — REVISIÓN HUMANA (gate crítico)

**Comando:** `ovav review --staged`
**Regla:** Para superficies críticas, se requiere aprobación explícita del lead.

**Qué es crítico:**
- Cambios en leads (Thavren, Eidren) → requiere aprobación de Thavren
- Cambios en bloques de permisos → requiere aprobación de Thavren
- Nuevos agentes → requiere aprobación de Thavren
- Cambios en routing de áreas → requiere aprobación de Thavren

**Qué NO requiere revisión manual (automático si V1-V7 pasa):**
- Cambios en teams (no tocan superficie pública)
- Correcciones de typo sin cambio semántico
- Actualizaciones de descripciones sin cambio de comportamiento

**Flujo de revisión:**
```
ovav review --staged

  Cambios detectados:
  ✦ lead-thavren.md — MODIFICADO (superficie crítica)
  ◆ area-platform-engineering.md — MODIFICADO (routing)
  ◇ team-soren.md — MODIFICADO (no crítico, auto-aprobado)

  Esperando aprobación para:
  [1] lead-thavren.md
      Diff: +12 líneas en "Professional criteria"
      Riesgo: MEDIO

  Aprobar? (y/n/diff) █

  > y
  ✓ Aprobado. Hash de aprobación: a3f2b1...
  Registrado en ledger.

  [2] area-platform-engineering.md
      Diff: routing line actualizada
      Riesgo: BAJO — referencia verificada V4

  Aprobar? (y/n/diff) █

  > y
  ✓ Aprobado.

  Revisión completa. 2/2 aprobados. Listo para release.
```

**Reglas de la aprobación:**
- Expira en 24 horas. Si no se hace release en ese tiempo, hay que re-aprobar.
- Queda registrada en `.ovav/registry/release_ledger.yaml` con hash, timestamp y lead.
- No se puede delegar. Solo el lead del área aprueba sus agentes.

---

## CAPA 4 — RELEASE (a CLI externo)

**Comando:** `ovav release --target opencode --version 2.2.0`

**Precondiciones ineludibles:**
- V1-V7: ALL PASS
- Revisión humana: APROBADA para todas las superficies críticas
- Staging: sin cambios pendientes sin stage

### Pipeline S1-S7

```
┌─────────────────────────────────────────────────────────────┐
│                    RELEASE PIPELINE                          │
│                                                             │
│  S1  IDENTITY    ──  L0 Identity Packet Compiler            │
│                      Hash SHA256 de fuente canónica          │
│                      El release sabe exactamente qué versión │
│                      de .ovav/source/agents/ está entregando        │
│                                                             │
│  S2  INTEGRITY   ──  F0.4 Living Integrity (5-layer scan)   │
│                      1,602 archivos. 0 drifts = OK.         │
│                      Si hay drift → BLOQUEADO.              │
│                                                             │
│  S3  COMPAT      ──  Knowledge Compiler + Manifest          │
│                      ¿OpenCode v2.2 acepta este formato?    │
│                      ¿El adapter está actualizado?          │
│                      ¿Hay breaking changes detectados?      │
│                                                             │
│  S4  ADAPT       ──  Format Adapter (opencode.py)           │
│                      Traduce formato canónico → OpenCode.   │
│                      Reglas verificables, no copy ciego.    │
│                                                             │
│  S5  VERIFY      ──  Evaluation Pipeline completo           │
│                      OVAV Laws → Stress (3) → Red Team (5)  │
│                      → Drift → Decisión final               │
│                      El release debe pasar lo mismo que     │
│                      cualquier implementación OVAV.         │
│                                                             │
│  S6  SNAPSHOT    ──  Backup atómico del target actual       │
│                      .opencode/agents/ → snapshot           │
│                      Si S7 falla → rollback automático      │
│                                                             │
│  S7  RELEASE     ──  Escritura gobernada                    │
│                      Escribe .opencode/agents/              │
│                      Trace event: release-20260603-01        │
│                      CHANGELOG + Release Notes              │
│                      L7 Feedback: registra para aprendizaje │
│                                                             │
│  RESULTADO:  ████████ Release 2.2.0 completado              │
│              16 agentes. Rollback listo.                    │
└─────────────────────────────────────────────────────────────┘
```

### Versionado

```
OVAV Release 2.2.0 → OpenCode agents
  ├── Build: .ovav/source/agents/ commit: 9fc9f28
  ├── Pipeline: S1-S7 all PASS
  ├── Verification: V1-V7 530/530 PASS
  ├── Review: Thavren approved (hash: a3f2b1...)
  ├── Rollback: snapshot disponible
  └── Released: 2026-06-03 14:22 UTC
```

---

## CAPA 5 — MONITOREO POST-RELEASE

**Sistema:** L7 Feedback Loop + Observability Engine
**Qué hace:** después del release, OVAV no se desentiende.

```
Release 2.2.0 → OpenCode
         │
         ▼
L7 monitorea:
  → ¿Errores reportados en los agentes?
  → ¿Incompatibilidad detectada con nueva versión de OpenCode?
  → ¿Usuarios reportan fallos?

Si detecta problemas:
  → Alerta a Thavren
  → Propone rollback al snapshot anterior
  → KC registra el patrón de fallo
  → Próximo release incluye checks específicos para este fallo

Si todo OK:
  → KC refuerza el patrón de éxito
  → Próximo release reutiliza configuración validada
```

---

## CUBRIENDO HUECOS

### H1 — ¿Qué pasa si OpenCode cambia su formato?

```
KC monitorea Research Mesh → detecta anuncio de OpenCode v3.0
  → Evalúa impacto: ¿nuestros agentes son compatibles?
  → Si no → alerta: "OpenCode v3.0 requiere adapter update"
  → Propone cambios al adapter opencode.py
  → Thavren revisa, aprueba, testea en staging
  → Nuevo release con adapter actualizado
```

### H2 — ¿Qué pasa si un team member modifica staging accidentalmente?

```
.ovav/cli/agents/ es READ-ONLY excepto vía ovav stage.
Cualquier write directo → bloqueado por Permission Authority.
Cualquier modificación manual → detectada por Living Integrity.
```

### H3 — ¿Qué pasa si V1-V7 pasa pero producción falla?

```
Rollback inmediato al snapshot S6.
L7 registra: "Release 2.2.0 falló en producción. Motivo: X."
KC aprende: "Agregar check V8 para detectar X en futuros releases."
Release 2.2.1 incluye el nuevo check.
```

### H4 — ¿Qué pasa si necesito downgradear?

```
ovav release rollback --target opencode --to 2.1.0
  → Restaura snapshot de release 2.1.0
  → Trace event: rollback registrado
  → Usuarios vuelven a versión estable
```

### H5 — ¿Qué pasa con el OVAV CLI?

```
OVAV CLI (.ovav/cli/agents/) es siempre el primer target.
Recibe updates antes que cualquier CLI externo.
Es el reference implementation — si funciona en OVAV CLI,
tiene altas probabilidades de funcionar en el resto.
```

### H6 — ¿Qué evita que alguien saltee el gate?

```
El comando ovav release verifica:
  → ¿V1-V7 pasaron? Si no → ERROR: "Verification required. Run ovav verify."
  → ¿Revisión humana aprobada? Si no → ERROR: "Review required. Run ovav review."
  → ¿Staging actualizado? Si no → ERROR: "Stage required. Run ovav stage."

No existe --skip. No existe --force.
El Governor OVAV (no el script) bloquea la operación.
```

---

## ARQUITECTURA DE ARCHIVOS

```
.ovav/
├── agents/                          ← CAPA 0: fuente canónica
│   ├── areas/
│   ├── leads/
│   └── teams/
│
├── cli/agents/                      ← CAPA 1: staging (OVAV interno)
│
├── releases/                        ← CAPA 4: snapshots y versionado
│   ├── snapshots/
│   │   ├── opencode_2.1.0.tar.gz
│   │   └── opencode_2.2.0.tar.gz
│   └── manifest.yaml                ← versiones, tiers, compatibilidad
│
tools/agents/
├── manifest.yaml                    ← qué agente → qué CLI → qué tier
├── adapters/
│   ├── opencode.py
│   ├── ovav_cli.py
│   └── claude_code.py (futuro)
├── verify.py                        ← CAPA 2: V1-V7 (530+ checks)
├── review.py                        ← CAPA 3: gate humano
├── release.py                       ← CAPA 4: S1-S7 pipeline
└── monitor.py                       ← CAPA 5: post-release L7

registry/
├── agent_manifest.yaml              ← declara agentes y sus targets
├── release_ledger.yaml              ← historial de releases + aprobaciones
└── compatibility_matrix.yaml        ← versiones de CLI compatibles por adapter
```

---

## COMANDOS OVAV RUNTIME

```bash
# CAPA 1 — Staging
ovav stage                          # proyecta todo a .ovav/cli/agents/
ovav stage --agents thavren,eidren  # solo agentes específicos
ovav stage --diff                   # muestra qué cambió vs staging actual

# CAPA 2 — Verificación (ineludible)
ovav verify --staged                # V1-V7. SIEMPRE requerido antes de release.
ovav verify --staged --target opencode  # verifica compatibilidad con CLI específico
ovav verify --staged --verbose      # muestra los 530+ checks individuales

# CAPA 3 — Revisión humana
ovav review --staged                # presenta cambios críticos para aprobación
ovav review --staged --approve-all  # solo si ya se revisaron manualmente

# CAPA 4 — Release
ovav release --target opencode --version 2.2.0   # S1-S7 pipeline completo
ovav release --target opencode --dry-run          # preview sin escribir
ovav release rollback --target opencode --to 2.1.0

# CAPA 5 — Monitoreo
ovav release status                 # estado de todos los releases activos
ovav release history                # historial completo
ovav release check                  # ¿hay problemas detectados por L7?
```

---

## LEYES DE OVAV QUE GOBIERNAN ESTE SISTEMA

1. **Fail-closed universal.** Si un gate no se puede verificar → bloqueo. Nunca se asume "debe estar bien".
2. **No bypass.** No existe `--skip`, `--force`, ni `--ignore`. El Governor bloquea, no el script.
3. **Single source of truth.** `.ovav/source/agents/` es la única fuente. Todo lo demás es proyección.
4. **Traceability.** Cada release, cada verificación, cada aprobación deja trace y hash.
5. **Rollback siempre.** Todo release tiene snapshot. Todo fallo tiene vuelta atrás.
6. **OVAV first.** OVAV CLI recibe updates antes que cualquier externo. Es el reference.
7. **Aprender del fallo.** KC + L7 detectan patrones y mejoran el pipeline.
