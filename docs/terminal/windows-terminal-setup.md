# Windows Terminal Setup — Braka Dev Environment

> Session: 2026-07-13 | Author: thavren (Platform Engineering)
> Archivos modificados fuera del repo OVAV. Este documento es la referencia canónica.

---

## 1. PowerShell Profile — PSReadLine Version-Agnostic

**Archivo:** `C:\Users\Alexa\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`

**Problema:** `-PredictionSource` y `-PredictionViewStyle` son parámetros de PSReadLine ≥2.2. PowerShell 5.1 usa PSReadLine 2.0.0 y tiraba error al cargar el perfil canónico.

**Fix:** Guard de versión + eliminación de duplicados de parches anteriores:

```powershell
Import-Module PSReadLine -MinimumVersion 2.0 -ErrorAction SilentlyContinue
$psrl = Get-Module PSReadLine
if ($psrl -and $psrl.Version -ge [Version]"2.2") {
    Set-PSReadLineOption -PredictionSource History
    Set-PSReadLineOption -PredictionViewStyle ListView
}
```

También se corrigió `$home` → `$UserHome` para evitar conflicto con la variable automática `$HOME` de PowerShell (case-insensitive).

---

## 2. Default Profile — powershell.exe → pwsh.exe

**Archivo:** `Microsoft.WindowsTerminal_*/LocalState/settings.json`

**Problema:** El perfil "PowerShell (Local)" ejecutaba `powershell.exe` (siempre resuelve a PS 5.1), aunque el GUID y `defaultProfile` apuntaban correctamente.

**Fix:** `commandline` cambiado a `pwsh.exe`:

```json
"commandline": "pwsh.exe -NoLogo -NoExit -Command \"cd $env:USERPROFILE\""
```

---

## 3. Transparencia — Eliminada

**Archivo:** `settings.json` de Windows Terminal

**Cambio:** `profiles.defaults.opacity: 92 → 100`

Sin transparencia. Fondo sólido `#242424`.

---

## 4. Perfiles — Nombres, Colores y Visibilidad

**Archivo:** `settings.json` de Windows Terminal

3 perfiles WSL estaban ocultos (`hidden: true`). Se des-ocultaron y se asignaron colores distintivos. PowerShell renombrado a BRAKA.

| Perfil | GUID | Color | Tab Title | Visible |
|---|---|---|---|---|
| WSL — Ubuntu (OVAV) | `a5a97cb0...` | `#2563eb` | 🏠 Home | ✅ |
| WSL — OVAV Systems | `b6b87dc1...` | `#7c3aed` | ⚙️ OVAV | ✅ |
| WSL — Development | `c7c98ed2...` | `#059669` | 💻 Dev | ✅ |
| BRAKA (PowerShell) | `61c54bbd...` | `#f59e0b` | BRAKA | — |
| CMD (Local) | `e2e4b2d0...` | `#6b7280` | ▸ CMD | — |

```json
// Ejemplo — WSL OVAV Systems
{
  "tabTitle": "⚙️ OVAV",
  "tabColor": "#7c3aed",
  "hidden": false,
  "suppressApplicationTitle": true
}
```

---

## 5. Keybindings — Gestión de Tabs

**Archivo:** `settings.json` de Windows Terminal

| Atajo | Acción |
|---|---|
| `Ctrl+Shift+N` | Nueva ventana |
| `Ctrl+Shift+,` | Abrir settings UI |
| `Ctrl+Shift+Space` | Buscador de tabs |
| `Ctrl+Shift+R` | Renombrar tab |
| `Ctrl+Shift+E` | Colorear tab |
| `Ctrl+Alt+←/→` | Mover tab izquierda/derecha |
| `Ctrl+0` | Ir a tab 10 |
| `Ctrl+9` | Ir a tab 9 |

Atajos clásicos conservados: `Ctrl+T` nueva tab, `Ctrl+Tab`/`Ctrl+Shift+Tab` navegar tabs, `Ctrl+1-8` tabs directos, `Ctrl+F4` cerrar tab.

---

## 6. Keybindings — Toggle Pane Zoom

**Archivo:** `settings.json` de Windows Terminal

**Nuevo:** `Ctrl+Shift+Z` → `Terminal.TogglePaneZoom`

```json
{"id": "Terminal.TogglePaneZoom", "keys": "ctrl+shift+z"}
```

Flujo: 2 panes → posicionarse en pane 2 → `Ctrl+Shift+Z` → pane 2 fullscreen → `Ctrl+Shift+Z` otra vez → ambos panes visibles. Misma función que `Alt+Enter`.

---

## 7. Keybindings — Redimensionar Panes (Reorganización)

**Archivo:** `settings.json` de Windows Terminal

**Problema:** Redimensionar panes requería `Alt+Shift+←/→/↑/↓`, poco intuitivo. `Ctrl+Shift+←/→` estaba asignado a MoveTab (operación poco frecuente).

**Reorganización:**

| Atajo | Antes | Ahora |
|---|---|---|
| `Ctrl+Shift+←/→/↑/↓` | MoveTab / Scroll | **Redimensionar pane** |
| `Ctrl+Alt+←/→` | — | Mover tab |
| `Alt+Shift+←/→/↑/↓` | Redimensionar pane | Sigue funcionando (fallback) |

Scroll con `Ctrl+Shift+↑/↓` se mantiene vía `Ctrl+Shift+PgUp/PgDn`.

---

## 8. PowerShell Welcome — Prompt Dinámico

**Archivo:** `C:\Users\Alexa\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`

Reemplaza el welcome estático por uno que muestra fecha real, directorio actual, y git status:

```powershell
$now = Get-Date -Format "ddd dd MMM yyyy  HH:mm"
$UserHome = $env:USERPROFILE
# detección automática de git branch, dirty, staged
```

Salida de ejemplo:
```
  Braka-Dev v6.0  dom 13 jul 2026  14:32
  ~/Systems/OVAV  | main ! +
  PS7 | Starship | Delta | LazyGit | Neovim | WSL2
```

Nótese `$UserHome` — PowerShell es case-insensitive, `$home` choca con `$HOME` automática (read-only).

---

## 9. Starship Config — Sin Warnings

**Archivos:**
- `/home/braka/.config/starship.toml` (WSL)
- `C:\Users\Alexa\.config\starship.toml` (Windows)

**Cambios:**
- `scan_timeout = 50` — elimina `[WARN] Scanning current directory timed out`
- `[time]` module: `format = "[$time]($style) "` + `time_format = "%H:%M"` (el formato `[%H:%M]` directo no es válido en starship)
- Módulos activados con íconos: git, nodejs, python, golang, rust, docker, kubernetes

---

## 10. Fish Shell — bind -k Fix

**Archivo:** `/home/braka/.config/fish/config.fish`

Fish 4.x eliminó el flag `-k`. Sintaxis migrada:

```fish
# Antes (roto en fish ≥4.0)
bind -k ctrl-l 'ls -lah'

# Ahora
bind \cl 'ls -lah; commandline -f repaint'
```

---

## 11. BITEL Agent — Perfil WSL

**Archivo:** `settings.json` de Windows Terminal

Nuevo perfil que abre directo en el proyecto BITEL Agent (`bt-sys-react`):

```json
{
  "name": "WSL — BITEL Agent",
  "guid": "{5c8323a7-f061-4f82-a3d0-638a332a6199}",
  "commandline": "C:\\Windows\\System32\\wsl.exe -d Ubuntu-24.04 --cd /home/braka/Work/web/products/bt-sys-react",
  "startingDirectory": "//wsl.localhost/Ubuntu-24.04/home/braka/Work/web/products/bt-sys-react",
  "tabTitle": "🔴 BITEL",
  "tabColor": "#dc2626",
  "hidden": false,
  "suppressApplicationTitle": true
}
```

---

## Resumen de Archivos Modificados

| # | Archivo | Sistema |
|---|---|---|
| 1 | `Microsoft.PowerShell_profile.ps1` | Windows (PowerShell) |
| 2 | `settings.json` | Windows (Terminal) |
| 3 | `/home/braka/.config/starship.toml` | WSL (Linux) |
| 4 | `C:\Users\Alexa\.config\starship.toml` | Windows |
| 5 | `/home/braka/.config/fish/config.fish` | WSL (Linux) |
| 6 | `docs/terminal/windows-terminal-setup.md` | OVAV repo (este archivo) |
