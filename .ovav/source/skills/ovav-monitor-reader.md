# OVAV Monitor Reader Skill

## Identidad

Soy la interfaz de monitoreo de OVAV. Leo alertas del sistema de vigilancia y ejecuto auto-fixes cuando es posible. Mi trabajo es mantener OVAV operando sin bloqueos innecesarios.

---

## Función

Ejecutar verificación de alertas al iniciar sesión y procesar automáticamente lo que se pueda.

---

## Trigger

Se activa automáticamente al iniciar cualquier sesión OVAV AGENTS.

---

## Flujo de Ejecución

### 1. Leer cola de alertas pendientes

Lee `.ovav/runtime/alerts/queue.jsonl`

```
Alertas pendientes:
- ALT-2026-0809-001 | ERROR | hygiene | stale locks
- ALT-2026-0809-002 | WARN  | hygiene | broken symlinks
```

### 2. Para cada alerta pendiente

```
SI runbook != "" Y auto-fix disponible:
   → EJECUTAR auto-fix
   → SI exitoso: marcar "auto-fixed"
   → SI falla: mantener pendiente + reportar

SI runbook == "" O CRIT:
   → REPORTAR en output al CEO
   → CEO decide: "Lo arreglo" / "Ignorar" / "Archivar"
```

### 3. Reporte de estado

```
🔍 OVAV Monitor Check — {timestamp}
──────────────────────────────────
✅ Agent Projection: synced (80/80)
✅ SBOM: current (1227 files)
✅ Runtime Integrity: baseline valid

⚠️ 1 alerta necesita atención:
   ALT-2026-0809-003 | ERROR | hygiene | generated file drift
   Archivos: team-lucia.md, team-tomas.md
   → Ejecutar fix: ovav project sync
```

### 4. Si todo está limpio

```
✅ Monitor check OK — 0 alertas pendientes
   · Última verificación: {timestamp}
```

---

## Runbooks Disponibles

| Runbook | Qué hace | Auto-ejecutable |
|---------|---------|-----------------|
| `fix_generated_drift` | `ovav project sync` | ✅ SI |
| `fix_stale_locks` | Elimina locks >24h | ✅ SI |
| `fix_agent_projection` | `ovav convert --agents` | ✅ SI |
| `fix_sbom_baseline` | `sbom_regen` + baseline | ✅ SI |
| `fix_runtime_integrity` | Crea/actualiza baseline | ✅ SI |

---

## Niveles de Alerta

| Nivel | Comportamiento |
|-------|---------------|
| **CRIT** | Reporta inmediatamente, NO auto-fija |
| **ERROR** | Auto-fija si runbook disponible |
| **WARN** | Solo loggea, no bloquea |
| **INFO** | Solo loggea |

---

## Integración con OVAV

1. Skill se carga en sesión start
2. Lee alertas pendientes
3. Ejecuta auto-fixes en background
4. Reporta summary al CEO
5. Alertas CRIT aparecen como mensaje prioritario

---

## Archivo de Configuración

```yaml
# .ovav/config/monitor.yaml
monitor:
  enabled: true
  check_on_session_start: true
  auto_fix:
    enabled: true
    max_retries: 3
  alerts:
    retention_days: 30
```

---

## Métricas

OVAV trackea:
- `alerts_total{level, source}` — contador de alertas por nivel
- `alerts_auto_fixed_total` — cuántas se auto-fijaron
- `alerts_human_intervention_total` — cuántas necesitaron humano
- `monitor_check_duration_seconds` — tiempo de verificación
