# SEG 7 — Guía contextual y detección first-run

## Objetivo

La primera vez que un usuario abre OVAV, ve una guía amigable que lo orienta. En uso normal, el footer muestra ayuda contextual específica de cada pantalla. El usuario nunca se siente perdido.

## Alcance

**Detección first-run:**
- Si `~/.local/share/ovav/source` no existe → first-run detectado.
- Mostrar pantalla de bienvenida: "¡Bienvenido a OVAV! Te guío en 3 pasos."
- Opción de saltar guía para usuarios avanzados.

**Footers contextuales vivos:**

| Pantalla | Footer |
|---|---|
| Menú principal | `↑↓ mover · 1-4 focus · Enter abrir · q salir` |
| Instalar (en progreso) | `Instalando... no cierres la terminal` |
| Instalar (completado) | `Enter continuar · b volver · q salir` |
| Configurar | `Space toggle · Enter aplicar · b volver` |
| Actualizar | `Enter verificar · b volver · q salir` |
| Recuperar | `↑↓ seleccionar backup · Enter preview · r restaurar` |
| Resultado | `Enter siguiente · b volver · q salir` |

**Guía first-run (3 pasos):**
1. "Instalar OVAV" → prepara tu estación de trabajo
2. "Configurar" → elige herramientas y roles
3. "Actualizar" → mantén todo al día

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → detección first-run, `render_welcome()`, footers contextuales
- `bin/ovav` → pasar flag first-run al engine

## Validación

- First-run detectado → pantalla de bienvenida con guía.
- Segundo uso → va directo al menú principal.
- Cada pantalla tiene footer contextual correcto.
- Footer cambia durante progreso (Instalando...).
- Footer cambia al completar (Enter continuar).

## Done when

- First-run muestra bienvenida guiada.
- Usuario recurrente ve menú directo.
- Todos los footers son contextuales y precisos.
- Sin footers genéricos ni placeholders.
