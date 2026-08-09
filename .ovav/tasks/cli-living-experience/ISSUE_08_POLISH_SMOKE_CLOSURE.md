# SEG 8 — Pulido final, smoke tests y cierre RC9.0

## Objetivo

Eliminar todo rastro de placeholders ("soon", "RC8.4", "gated", "real apply path remains gated"). Todo cableado, todo verificado. El cockpit está vivo y listo para release.

## Alcance

**Limpieza de placeholders:**
- Eliminar "RC8.4 wires full apply" del cockpit.
- Eliminar "Space select soon" del footer de Tailor.
- Eliminar "real apply path remains gated" del cockpit.
- Eliminar "repair apply is not active from RC5 cockpit".
- Revisar todo `tools/cli/ovav_first_run_cockpit.py` en busca de "soon", "gated", "RC8", "TODO", "FIXME".

**Nota:** El cockpit curses (`ovav_first_run_cockpit.py`) es la base viva, NO se archiva. El experience engine y smokes obsoletos ya fueron movidos a `tools/cli/_legacy/` durante el refactor pre-segmentos.

**Smoke tests:**
1. Navegación: ↑↓/jk mueven foco, 1-4 saltan, Enter abre, b/q regresan/salen.
2. Install: pipeline completo desde cockpit, progreso real, resultado con verificación.
3. Tailor: Space togglea, Enter aplica, cambios persisten.
4. Update: verifica remoto, preview, aplica si hay cambios.
5. Recovery: lista backups, preview, restore con confirmación.
6. First-run: detección, bienvenida, guía, salto a experto.

**Cierre:**
- `VERSION` → `v1.0.0-rc9`
- Validadores OVAV: `python3 tools/ovav_runtime.py validate` → OK
- Evidencia en `.ovav/artifacts/RC9_0/`
- Checklist de cierre documentado.

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → limpieza de placeholders y pulido final
- `bin/ovav` → verificar que el router apunta correctamente al cockpit
- `tools/cli/router.py` → verificar rutas finales
- `VERSION` → bump a rc9
- `.ovav/artifacts/RC9_0/` → evidencia de cierre

## Validación

- `grep -r "soon\|gated\|RC8.4\|FIXME\|TODO" tools/cli/ovav_first_run_cockpit.py` → cero resultados.
- 6 smoke tests pasan.
- `python3 tools/ovav_runtime.py validate` → OK.
- `ovav` sin argumentos abre el cockpit curses vivo, sin errores.

## Done when

- Cero placeholders en código de cockpit.
- 6/6 smoke tests pasan.
- Validadores OVAV pasan.
- VERSION actualizado.
- Evidencia de cierre generada.
- Rama lista para merge a develop.
