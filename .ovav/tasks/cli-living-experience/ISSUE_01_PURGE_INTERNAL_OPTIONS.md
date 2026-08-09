# SEG 1 — Purga de opciones internas del menú principal

## Objetivo

El menú principal del cockpit (`ovav` sin argumentos) solo muestra las opciones que un usuario normal necesita. Las herramientas de diagnóstico interno se mueven a `ovav doctor` y `ovav advanced`.

## Alcance

- Redefinir `OPTIONS` en `tools/cli/ovav_first_run_cockpit.py`: de 6 opciones a 4.
- Redefinir `SCREEN_COPY`: eliminar pantallas de Preview Plan y Control Room del engine.
- `bin/ovav`: asegurar que `ovav doctor` y `ovav advanced` sigan accesibles por CLI.
- El footer, la navegación y los atajos de teclado se actualizan al nuevo menú.

## Menú resultante

| # | Opción | Grupo | Modo |
|---|---|---|---|
| 1 | Instalar OVAV | SETUP | guided |
| 2 | Configurar | SETUP | custom |
| 3 | Actualizar | GOVERN | controlled |
| 4 | Recuperar | GOVERN | guarded |

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → `OPTIONS`, `SCREEN_COPY`, `menu_lines()`, `render_main()`
- `bin/ovav` → sin cambios si `ovav doctor` ya funciona

## Validación

- `ovav` → 4 opciones visibles, navegables con ↑↓/jk y 1-4.
- `ovav doctor` → diagnóstico completo accesible.
- Ninguna opción es decorativa.

## Done when

- Menú principal muestra exactamente 4 opciones.
- Navegación y footers reflejan el nuevo menú.
- Smoke test de navegación pasa.
- Preview Plan y Control Room siguen accesibles por CLI (`ovav doctor`, `ovav setup --plan`).
