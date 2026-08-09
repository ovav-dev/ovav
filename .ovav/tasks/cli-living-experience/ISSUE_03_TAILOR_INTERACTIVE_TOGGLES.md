# SEG 3 — Tailor interactivo con toggles vivos

## Objetivo

La opción "Configurar" del cockpit permite seleccionar y deseleccionar herramientas, roles y plan con feedback visual inmediato. La tecla Space actúa como toggle. Enter aplica los cambios con preview y confirmación.

## Alcance

- Nueva pantalla Tailor en el engine con tres secciones: Herramientas, Rol, Plan.
- Cada tool/rol/plan tiene un indicador visual [✓] activo / [ ] inactivo.
- **Space** = toggle inmediato con feedback visual.
- **Enter** = preview de cambios + confirmación + apply.
- La lógica de selección se extrae a `tools/cli/ovav_tailor_composer.py` para mantener el engine limpio.

## Layout

```
┌─ Configurar OVAV ─────────────────────────────┐
│                                                 │
│  Herramientas                                   │
│  [✓] OpenCode    · repositorio gobernado        │
│  [✓] Git         · detectado                    │
│  [ ] Neovim      · Space para activar           │
│  [ ] Zellij      · Space para activar           │
│  [ ] Fish shell  · Space para activar           │
│                                                 │
│  Rol profesional                                │
│  (●) Platform Engineering  · activo             │
│  (●) Research Intelligence · activo             │
│  ( ) Security Architecture · premium            │
│                                                 │
│  Plan                                           │
│  [✓] Base · gratuito                            │
│                                                 │
│  Enter aplicar · Space toggle · b volver        │
└─────────────────────────────────────────────────┘
```

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → nueva pantalla `render_tailor()`, manejo de Space
- `tools/cli/ovav_tailor_composer.py` → NUEVO: lógica de selección, preview, apply

## Validación

- Navegar a Configurar → ver herramientas, roles, plan.
- Space en cualquier item → toggle visual inmediato (✓ ↔ ✗).
- Enter → preview de cambios antes de aplicar.
- Confirmar → cambios aplicados, pantalla de resultado.

## Done when

- Todos los toggles responden a Space con feedback visual.
- Preview muestra exactamente qué va a cambiar.
- Apply ejecuta los cambios y muestra resultado.
- El estado de los toggles se preserva entre navegaciones.
