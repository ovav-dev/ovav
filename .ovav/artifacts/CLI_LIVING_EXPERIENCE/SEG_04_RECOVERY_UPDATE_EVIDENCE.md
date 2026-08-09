# SEG 04 — Recovery + Update reales

Fecha: 2026-05-27  
Fase: Final Launch Verification / OpenCode smoke testing  
Estado: implementado en fuente local; sin claim production/global-ready.

## Cambios

  - Update ahora abre preview real desde estado Git/local upstream y plan `ovav update --plan`.
  - Update aplica solo si hay behind en upstream local, working tree limpio y confirmación; ejecuta backup pre-update, merge `ff-only` y verificación post-update.
  - Recovery lista backups reales desde `ovav rollback --list --json`.
  - Recovery muestra preview de entradas restaurables y requiere doble Enter antes de restaurar.
  - Restore usa `ovav rollback --apply --consent --accept-risk --backup-id ... --json` y verifica después.
- `tools/cli/ovav_backup_manager.py`
  - Añadido contrato JSON `ovav.backup_list.v1` con fecha, tamaño, descripción, scope y conteo.
  - Añadido flag `rollback --list --json` para alimentar el cockpit con backups reales.

## Límites preservados

- Source-local only.
- Sin writes a HOME/config global.
- Sin plugin install, MCP/A2A ni Engram live.
- Sin fetch externo desde cockpit update; compara contra upstream local conocido.
- Sin production/global-ready claim.

## Validación ejecutada

- `python3 tools/harnesses/workspace_safety_gate.py --mode mutate` → PASS.
- `python3 -B bin/ovav rollback --list --json` → PASS; contrato `ovav.backup_list.v1`.
- `python3 -B bin/ovav update --no-interactive` → PASS; preview estático con remoto/upstream/local dirty state.
- `python3 -B bin/ovav recovery --no-interactive` → PASS; lista backups reales.
- `python3 -B bin/ovav backup --plan --json` → PASS.
- `python3 -B bin/ovav rollback --plan --json` → PASS.
- Helper smoke Python (`build_update_preview`, `recovery_backup_list`, `recovery_plan_for`) → PASS.
- `python3 -B tools/cli/ovav_practical_smoke.py --json` → PASS.
- `python3 -B tools/cli/ovav_install_smoke.py --json` → PASS.
- `python3 -B tools/cli/ovav_fresh_clone_smoke.py --json` → PASS.
- `OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate` → PASS.
- `python3 tools/validators/check_final_launch_current_authority.py` → PASS.
- `python3 tools/validators/check_final_launch_runtime_authority.py` → PASS.

## Resultado

SEG 4 queda funcional en repo: Update y Recovery dejaron de ser pantallas nominales y ahora consumen estado/acciones reales gobernadas. El apply destructivo sigue protegido por confirmación explícita y scope source-local.

## Cierre `CIERRA TODO`

Revalidación ejecutada antes de commit atómico:

- workspace safety gate → PASS.
- py_compile cockpit/backup manager/router → PASS.
- rollback list JSON → PASS.
- update/recovery static screens → PASS.
- helper smoke SEG4 → PASS.
- practical/install/fresh-clone smokes → PASS.
- strict runtime validation → PASS.
- final launch current/runtime authority validators → PASS.
