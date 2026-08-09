# Migration Handoff: WezTerm → PowerShell + Windows Terminal

**From:** thavren (Platform Engineering)
**To:** next session + CEO Braka
**Date:** 2026-07-09
**Status:** 📋 PLAN COMPLETO — pendiente CEO ejecutar

## What was designed

Plan completo de migración desde **WezTerm** (con WSL2 Ubuntu-24.04 + fish) a **PowerShell 7 + Windows Terminal + Ubuntu WSL2 + fish**.

### Detection findings

| Componente | Estado detectado |
|---|---|
| PowerShell 7.6.3 LTS | ❌ No instalado |
| Windows Terminal 1.24+ | ❌ No instalado |
| JetBrainsMono Nerd Font | ❌ No instalado |
| PowerShell 5.1 (built-in) | ✅ Disponible |
| WezTerm | ✅ Instalado (mantener como ssh) |
| WSL2 Ubuntu-24.04 | ✅ Running |
| fish 4.7.1 | ✅ Instalado en Ubuntu |

### Recommended PowerShell version

**PowerShell 7.6.3 LTS** — publicada 2026-06-16 por Microsoft, base .NET SDK 10.0.301, soporte LTS hasta 2028+. Es el sweet spot entre stability y features modernos.

### Plan documentado en 4 fases

- **Fase 0** — Instalación: PowerShell 7, Windows Terminal, Nerd Font (~10 min)
- **Fase 1** — PowerShell `$PROFILE` con PSReadLine, oh-my-posh, posh-git, Terminal-Icons + bridge aliases OVAV (`owc`, `owd`, etc.) que llaman `wsl -d Ubuntu-24.04 -- fish -c "owc $args"` (~15 min)
- **Fase 2** — Cleanup fish: archivar 2 hooks wezterm + desactivar `OVAV_TERMINAL=wezterm` (~10 min)
- **Fase 3** — Windows Terminal: settings.json con Nerd Font, scheme Tango Dark, keybindings panes/quake mode (~15 min)
- **Fase 4** — Validación con tests de regresión funcional (~5 min)

### UI Personalization spec

- **Font:** JetBrainsMono Nerd Font 12pt (ligaduras + glyphs para lsd/eza)
- **Theme:** Tango Dark sobre negro (#000000)
- **Cursor:** filledBox blanco
- **Acrylic:** 85% opacity
- **Keybindings custom:** Ctrl+Shift+T/W (tab/close), Alt+Shift+/- (split H/V), Alt+Arrows (navigate panes), Ctrl+~ (quake mode)
- **Profile icons:** icons MS oficiales para Ubuntu/pwsh/PowerShell 5.1

### Atajos consolidados (vs WezTerm)

| Atajo | Acción | Equivalente antes |
|---|---|---|
| `Ctrl+Shift+T` | Nueva tab en WT | WezTerm `Ctrl+Shift+Enter` |
| `Alt+Shift++` | Split horizontal | WezTerm `Ctrl+Shift+H` |
| `Alt+Shift+-` | Split vertical | WezTerm `Ctrl+Shift+V` |
| `Ctrl+~` | Quake dropdown | WezTerm tenía quake nativo ya |
| `owc <feat>` | Crear worktree (en pwsh o fish) | igual |
| `owd` | Done worktree | igual |

### Files persisted

| Path | |
|---|---|
| `/home/braka/Systems/OVAV/.ovav/reports/2026-07-09/migration-wezterm-to-powershell.md` | Plan completo (4 fases + comandos) |
| `/home/braka/Systems/OVAV/.ovav/reports/2026-07-09/MIGRATION_PLAN_SUMMARY.txt` | Resumen ejecutivo CEO-readable |
| `/home/braka/Systems/OVAV/.ovav/handoffs/2026-07-09-powershell-migration.md` | Este handoff |

## What is pending

1. **CEO ejecuta Fase 0** (instalación): 3 comandos `winget install`. Tiempo: 10 min.
2. **CEO notifica a thavren** cuando termine.
3. **CEO o thavren** continúa Fase 1-4 (configuración) — se puede automatizar con scripts idempotentes.

## Sandbox limitation

El agente mimo **no puede escribir** a `~/Documents/PowerShell/` ni `%LOCALAPPDATA%\Microsoft\WindowsTerminal\` por la policy `external_directory`. La Fase 1 (crear `$PROFILE`) requiere CEO ejecutar desde PowerShell local, o un script firmado por CEO que el agente prepare.

## Restore path

Si la migración falla o CEO decide volver a WezTerm:
```bash
mv ~/.config/fish/_archive_20260709_wezterm/*.fish ~/.config/fish/conf.d/
```
Y los hooks wezterm vuelven activos sin más cambios. **No-destructivo por diseño.**

## References

- PowerShell releases: https://github.com/PowerShell/PowerShell/releases/tag/v7.6.3
- Windows Terminal: https://github.com/microsoft/terminal
- Windows Terminal docs: https://learn.microsoft.com/en-us/windows/terminal/
- Nerd Fonts: https://www.nerdfonts.com/
- oh-my-posh themes: https://ohmyposh.dev/docs/themes
- OVAV Systems canonical: `/home/braka/Systems/OVAV/.ovav/plan/caps.yaml`
