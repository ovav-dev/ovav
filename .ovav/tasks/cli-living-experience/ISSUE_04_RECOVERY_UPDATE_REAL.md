# SEG 4 — Recovery y Update con acciones reales

## Objetivo

Las opciones "Actualizar" y "Recuperar" del cockpit ejecutan flujos reales con progreso, confirmación y verificación. Nada es simulado.

## Alcance

**Update:**
- Verificar remoto → descargar → backup → aplicar → verificar.
- Cada paso mueve la barra de progreso real.
- Si no hay cambios: "OVAV ya está actualizado."
- Si hay cambios: preview de qué cambia antes de aplicar.

**Recovery:**
- Listar backups disponibles (fecha, tamaño, descripción).
- Seleccionar backup → preview de contenido.
- "Restaurar" → confirmación explícita (doble ENTER) → progreso → verificación.

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → pantallas Update y Recovery con acciones reales
- Comandos existentes: `ovav update`, `ovav backup`, `ovav rollback` → consumir output real

## Pipelines

```
Update:
[1] Verificar remoto      ██████░░░░░░  30%
[2] Descargar cambios      ██████████░░  50%
[3] Backup pre-update      ████████████  70%
[4] Aplicar actualización  ████████████  90%
[5] Verificar post-update  ████████████ 100%

Recovery:
[1] Listar backups      → seleccionar con ↑↓
[2] Preview contenido    → Enter para ver
[3] Confirmar restore    → "Escribe RESTAURAR para confirmar"
[4] Restaurar           → barra de progreso real
[5] Verificar           → checks post-restore
```

## Validación

- Update con cambios disponibles → preview + progreso + resultado.
- Update sin cambios → mensaje claro, sin acción innecesaria.
- Recovery → lista backups reales, preview, restore con confirmación.
- Ambos flujos manejan errores con mensajes claros.

## Done when

- Update pipeline funciona de extremo a extremo desde el cockpit.
- Recovery lista backups reales y restaura con confirmación.
- Ambos flujos muestran progreso real y verificación post-acción.
