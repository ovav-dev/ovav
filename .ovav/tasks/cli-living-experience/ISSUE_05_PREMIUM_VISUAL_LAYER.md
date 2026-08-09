# SEG 5 — Capa visual premium

## Objetivo

Identidad visual Catppuccin Mocha completa aplicada al cockpit. Transiciones suaves entre pantallas. Motion donde suma. Consistencia cromática en toda la experiencia.

## Alcance

- Extraer constantes de tema a `tools/cli/ovav_visual_theme.py`.
- Aplicar paleta Catppuccin Mocha completa: Lavender, Blue, Green, Red, Mauve, Flamingo, Surface, Base.
- Transición fade al cambiar de pantalla (clear + breve delay render).
- Barra de progreso con caracteres Unicode suaves (no ASCII crudo).
- Status chips con color semántico (✓ verde, ✗ rojo, ◌ gris, ● activo).
- Logo OVAV refinado con gradiente de color.
- Footer contextual sin flicker entre pantallas.

## Paleta Catppuccin Mocha

| Token | Hex | Uso |
|---|---|---|
| Lavender | #B4BEFE | Títulos, logo, encabezados |
| Blue | #89B4FA | Acciones, links, progreso |
| Green | #A6E3A1 | Éxito, verificación, chips OK |
| Red | #F38BA8 | Error, alertas, chips FAIL |
| Mauve | #CBA6F7 | Destacados, gradiente logo |
| Flamingo | #F2CDCD | Acentos suaves |
| Surface | #313244 | Fondos de panel |
| Base | #1E1E2E | Fondo principal |

## Archivos

- `tools/cli/ovav_visual_theme.py` → NUEVO: constantes de color, función `transition()`, `progress_bar()`
- `tools/cli/ovav_first_run_cockpit.py` → reemplazar colores hardcodeados por imports del tema

## Validación

- Todos los colores en el cockpit vienen del tema, no hay hardcodeados.
- Transición entre pantallas es suave (sin flicker).
- Barra de progreso usa caracteres Unicode suaves.
- Status chips usan color semántico correcto.

## Done when

- Archivo de tema creado y usado por el engine.
- Paleta Catppuccin Mocha aplicada consistentemente.
- Sin colores hardcodeados fuera del tema.
- Transiciones sin flicker visible.
