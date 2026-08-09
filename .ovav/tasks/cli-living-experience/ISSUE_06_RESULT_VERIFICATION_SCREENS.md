# SEG 6 — Pantallas de resultado y verificación

## Objetivo

Toda acción en el cockpit termina en una pantalla de resultado estructurada que muestra: qué pasó, qué cambió, verificación y próximo paso. Nunca hay una pantalla "muerta" tras una acción.

## Alcance

- `render_result()` unificado en el engine: recibe un payload estándar y lo muestra.
- Cada acción (install, tailor, update, recovery) devuelve un payload con:
  - `status`: ok / warning / error
  - `summary`: descripción humana de lo que pasó
  - `changes`: lista de archivos/superficies modificadas
  - `verification`: resultados de checks post-acción
  - `next_action`: sugerencia de siguiente paso
- Eliminar todo `print()` suelto en acciones: todo pasa por `render_result()`.

## Formato estándar de resultado

```
✓ Instalación completada

  Qué se instaló:
  · ovav command → ~/.local/bin/ovav
  · Source → ~/.local/share/ovav/source
  · OpenCode surfaces → sincronizadas

  Verificación:
  · repo-check ✓ · security ✓ · surfaces ✓

  Próximo paso: Enter para abrir Configurar
```

```
✗ Actualización fallida

  Error: conflicto en surfaces/opencode
  Detalle: el archivo agents/thavren.md fue modificado localmente

  Opciones:
  · r reintentar   · b volver   · d doctor
```

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → `render_result()`, adaptar todas las acciones para usar payload estándar

## Validación

- Install exitoso → pantalla de resultado con verificación.
- Install fallido → pantalla de error con opciones.
- Tailor apply → resultado con cambios aplicados.
- Toda acción termina en `render_result()`, nunca en `print()` suelto.

## Done when

- `render_result()` unificado existe y se usa en todas las acciones.
- Payload estándar definido y documentado.
- Sin `print()` sueltos en flujos de acción.
- Pantalla de resultado siempre incluye próximo paso sugerido.
