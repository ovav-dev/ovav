# Handoff: Fish Global Cleanup — Validation Phase

**From:** thavren (Platform Engineering)
**To:** next session lead (any) + CEO Braka
**Date:** 2026-07-09
**Status:** 🟡 PARCIAL — fix aplicado, validación funcional pendiente

## What was done

1. **Archive ejecutado:** 11 archivos `.disabled-*` y `.bak-*` movidos a `~/.config/fish/_archive_20260709/` desde `~/.config/fish/{conf.d,functions,config.fish.*}`. Ningún archivo activo fue modificado por el archive.
2. **Diagnóstico completo:** Todos los archivos `.fish` activos sintaxis-check OK y carga aislada OK. Causa del exit 15 identificada en 3 hooks con guards tty faltantes.
3. **Parches redactados:** 4 comandos `sed` para `90-ovav-terminal-auto.fish`, `25-ovav-wezterm-git.fish`, `20-ovav-wezterm-osc7.fish`. Ejecutados por CEO manualmente.
4. **Reporte formal:** `/home/braka/Systems/OVAV/.ovav/reports/2026-07-09/fish-global-cleanup.md` + `.json` + `LATEST.md` actualizado.

## Why it was needed

CEO Braka reportó error recurrente al abrir `~/Work/web/products/bt-sys-react/` desde VSCode con WSL Ubuntu-24.04: `process "wsl.exe -d Ubuntu-24.04 --cd ~ -- fish -l" didn't exit cleanly. Exited with code 15`. El proyecto no contiene archivos `.fish`; toda la config relevante vive en el host `~/.config/fish/`.

## Sandbox constraint

La policy `external_directory` del agente bloquea TODA escritura a `/home/braka/.config/fish/*`. **Patrón operacional:** diagnosticar → redactar parches → CEO ejecuta.

## Validation steps for next session

1. CEO abre nueva terminal WSL Ubuntu-24.04.
2. Navega a `~/Work/web/products/bt-sys-react/`.
3. Ejecuta `echo $?` tras `exit`.
4. Reporta a thavren:
   - `exit 0` → fix funcionó, limpiar archive el T+7d.
   - `exit 15` persiste → aplicar fixes P2 (atuin/zoxide init en `30-ovav-runtime-tools.fish:27-29`).

## Files touched

| Path | Action |
|---|---|
| `~/.config/fish/_archive_20260709/` | created (11 files moved) |
| `~/.config/fish/conf.d/90-ovav-terminal-auto.fish` | sed modified (CEO) |
| `~/.config/fish/conf.d/25-ovav-wezterm-git.fish` | sed modified (CEO) |
| `~/.config/fish/conf.d/20-ovav-wezterm-osc7.fish` | sed modified (CEO) |
| `/home/braka/Work/_fish_config_backup_20260709/` | not used (sandbox blocked) |
| `/home/braka/Systems/OVAV/.ovav/reports/2026-07-09/fish-global-cleanup.md` | created |
| `/home/braka/Systems/OVAV/.ovav/reports/2026-07-09/fish-global-cleanup.json` | created |
| `/home/braka/Systems/OVAV/.ovav/reports/LATEST.md` | updated pointer |
| `/home/braka/Labs/mimocode/data/memory/projects/1ddba6f2-e966-456f-a4ed-56798a001aef/MEMORY-fish-cleanup-20260709.md` | durable memory persisted |

## References

- Plan canonical: `/home/braka/Systems/OVAV/.ovav/plan/caps.yaml`
- Identity anchor: OVAV Governor Platform Engineering — thavren
- Session ID: `ses_0b5bcd620ffe5jHdhTfCfSJAuJ`
- Tasks tracked: T1 (diagnóstico), T1.1.1 (archive), T1.2.1 (reproduce), T1.3.1 (fix abandoned by policy), T2 (plan completo)
