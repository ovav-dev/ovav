# Plan Maestro: Migración WezTerm → PowerShell + Windows Terminal + Ubuntu WSL2

**Fecha:** 2026-07-09
**Para:** CEO Braka
**Autor:** thavren (Platform Engineering)
**Estado:** 📋 DISEÑO — pendiente ejecución del CEO

---

## 📋 Resumen Ejecutivo

Migración total de la sesión de terminal desde **WezTerm + WSL2 + fish** a **PowerShell 7 + Windows Terminal + Ubuntu-24.04 WSL2 + fish**. Preserva toda la productividad actual, elimina el exit 15, mejora rendimiento y desacopla la dependencia de WezTerm.

### Verificación previa del entorno actual
| Componente | Estado actual |
|---|---|
| PowerShell instalado | ⚠️ Solo Windows PowerShell 5.1 (built-in) |
| PowerShell 7 (pwsh) | ❌ NO instalado |
| Windows Terminal | ❌ NO instalado (no se encontró en paths estándar) |
| WezTerm | ✅ Instalado (mantener como herramienta ssh) |
| WSL2 Ubuntu-24.04 | ✅ Running |
| fish 4.7.1 | ✅ Instalado en Ubuntu |

### Stack final propuesto
```
Windows 11/10
  ├── Windows Terminal 1.24+ (UI unificada, panes nativos)
  │     └── Tab default: Ubuntu-24.04 (WSL2)
  │           └── fish 4.7.1 (shell interactivo)
  │                 └── bash on logout → PowerShell 7 (pwsh)
  │
  └── PowerShell 7.6.3 LTS (scripts admin Windows + acceso a binarios .exe)
```

---

## 🎯 Objetivos

| # | Objetivo | Métrica de éxito |
|---|---|---|
| 1 | Eliminar exit 15 definitivamente | `wsl -d Ubuntu-24.04 -- fish -l` retorna 0 incluso al cerrar panel |
| 2 | Mejorar latencia de keystroke | ~200ms → ~50ms en filesystem /home/braka |
| 3 | Preservar 100% funcionalidad OVAV | Todos los aliases `owc/owd/owl/owv/owr/owx/...` siguen disponibles |
| 4 | Mantener wezterm disponible como herramienta ssh especializada | `wezterm --help` sigue funcionando |
| 5 | Tener PowerShell 7 LTS como host de scripts Windows | `pwsh -Command "$PSVersionTable.PSVersion"` retorna 7.6.3 |

---

## 📦 FASE 0 — Instalación de dependencias (10 min, CEO ejecuta)

### 0.1 Instalar PowerShell 7.6.3 LTS (latest stable)

**Opción A — winget (recomendado, ya incluido en Windows):**
```powershell
# Desde PowerShell 5.1 (built-in) como admin:
winget install --id Microsoft.PowerShell --version 7.6.3 --source winget
```

**Opción B — MSI directo:**
```
https://github.com/PowerShell/PowerShell/releases/download/v7.6.3/PowerShell-7.6.3-win-x64.msi
```
Descargar, ejecutar MSI, "Install for all users" (para que quede en `C:\Program Files\PowerShell\7\`).

**Verificación:**
```powershell
pwsh -Command "$PSVersionTable.PSVersion"
# Debe retornar: Major=7, Minor=6, Build=3
```

### 0.2 Instalar Microsoft Windows Terminal

```powershell
# Desde PowerShell admin:
winget install --id Microsoft.WindowsTerminal --source winget
```

O desde **Microsoft Store** (recomendado para updates automáticos):
- Abrir Store → buscar "Windows Terminal" → Install.

**Verificación:**
```powershell
# Lanzar Windows Terminal:
wt
# Debe abrir terminal unificada con tabs.
```

### 0.3 Instalar Cascadia Code / JetBrainsMono Nerd Font (para ligaduras)

```powershell
# Opción A — Nerd Font (mejor para glyphs/icons):
winget install --id NerdFonts.JetBrainsMono --source winget

# Opción B — Microsoft official:
winget install --id Microsoft.CascadiaCode --source winget
```

### 0.4 Configurar Windows Terminal como terminal default

1. Abrir Windows Terminal.
2. Settings (Ctrl+,).
3. "Startup" → Default profile: **Ubuntu-24.04**.
4. "Default terminal application": **Windows Terminal** (sobrescribe conhost).

---

## ⚙️ FASE 1 — Configuración PowerShell 7 (15 min)

### 1.1 Crear perfil PowerShell

`$PROFILE` apunta a `~\Documents\PowerShell\Microsoft.PowerShell_profile.ps1`. Si no existe:

```powershell
# Crear directorio y archivo:
New-Item -Path $PROFILE -ItemType File -Force
```

### 1.2 Configuración `$PROFILE` recomendada (Plantilla Braka-Dev v1)

```powershell
# ══════════════════════════════════════════════════════════════════════════
# PowerShell 7 Profile — Braka-Dev v1.0
# Plataforma: Windows Terminal + WSL2 Ubuntu-24.04 + fish 4.7.1
# ══════════════════════════════════════════════════════════════════════════

# ── PSReadLine (history, predictions, key bindings) ───────────────────────
Import-Module PSReadLine
Set-PSReadLineOption -PredictionSource History
Set-PSReadLineOption -EditMode Windows
Set-PSReadLineOption -BellStyle None
Set-PSReadLineOption -HistorySearchCursorMovesToEnd
Set-PSReadLineKeyHandler -Key UpArrow -Function HistorySearchBackward
Set-PSReadLineKeyHandler -Key DownArrow -Function HistorySearchForward
Set-PSReadLineKeyHandler -Key Tab -Function MenuComplete
Set-PSReadLineKeyHandler -Key Ctrl+d -Function DeleteChar
Set-PSReadLineKeyHandler -Key Ctrl+Shift+c -Function Copy
Set-PSReadLineKeyHandler -Key Ctrl+Shift+v -Function Paste

# ── Terminal-Icons (vscode-style icons en Get-ChildItem) ──────────────────
Import-Module Terminal-Icons

# ── Prompt powerlevel10k-style ───────────────────────────────────────────
Import-Module posh-git
Import-Module oh-my-posh
Set-PoshPrompt -Theme paradox

# ── OWS aliases — bridge hacia fish/WSL ─────────────────────────────────
function wsl-fish { wsl -d Ubuntu-24.04 -- fish }
function owc { wsl -d Ubuntu-24.04 -- fish -c "owc $args" }
function owd { wsl -d Ubuntu-24.04 -- fish -c "owd $args" }
function owl { wsl -d Ubuntu-24.04 -- fish -c "owl $args" }
function owv { wsl -d Ubuntu-24.04 -- fish -c "owv $args" }
function ows { wsl -d Ubuntu-24.04 -- fish -c "ows $args" }
function owr { wsl -d Ubuntu-24.04 -- fish -c "owr $args" }

# ── Atajos locales (git, navegación) ─────────────────────────────────────
Set-Alias ll ls
Set-Alias g git
Set-Alias ga 'git add'
Set-Alias gc 'git commit'
Set-Alias gp 'git push'
Set-Alias gd 'git diff'
Set-Alias gs 'git status'

# ── Activación de script execution policy para este user ──────────────────
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force

Write-Host "✓ PowerShell 7 — Braka-Dev v1.0 loaded" -ForegroundColor Cyan
```

### 1.3 Instalar módulos PS

```powershell
Install-Module -Name PSReadLine -AllowPrerelease -Force -SkipPublisherCheck
Install-Module -Name Terminal-Icons -Repository PSGallery -Force
Install-Module -Name posh-git -Repository PSGallery -Force
Install-Module -Name oh-my-posh -Repository PSGallery -Force
```

---

## 🐚 FASE 2 — Configuración fish sobre WSL2 (10 min)

### 2.1 Quitar hooks wezterm de la config fish

**Backup antes:**
```bash
mkdir -p ~/.config/fish/_archive_20260709_wezterm
mv ~/.config/fish/conf.d/20-ovav-wezterm-osc7.fish ~/.config/fish/_archive_20260709_wezterm/
mv ~/.config/fish/conf.d/25-ovav-wezterm-git.fish   ~/.config/fish/_archive_20260709_wezterm/
```

**Desactivar WEZTERM_TERMINAL en runtime tools:**
```bash
# En ~/.config/fish/conf.d/30-ovav-runtime-tools.fish línea 4:
# ANTES: set -gx OVAV_TERMINAL wezterm
# DESPUÉS: set -gx OVAV_TERMINAL native
sed -i 's|set -gx OVAV_TERMINAL wezterm|# set -gx OVAV_TERMINAL wezterm  # disabled 2026-07-09 — PowerShell migration|' \
  ~/.config/fish/conf.d/30-ovav-runtime-tools.fish
```

### 2.2 Limpiar config.fish (líneas 53-57 wezterm-only)

Edit manual: eliminar bloque `__wezterm_report_cwd` que ya no aplica.

```bash
sed -i '/^# OVAV: report CWD to wezterm via OSC 0 title sequence$/,/^__wezterm_report_cwd$/d' ~/.config/fish/config.fish
sed -i '/^printf "\\e\]7;file/d' ~/.config/fish/config.fish
```

### 2.3 Atajos nativos de fish (sin cambios, ya están)

| Atajo | Función |
|---|---|
| `owc <feature>` | Crear worktree + auto-cd |
| `owd` | Merge → develop + cleanup + auto-cd |
| `owl` | List worktrees |
| `owv` | Full verification pipeline |
| `owr` | Rescue lost branches |
| `ows` | Sync + maintenance |
| `owx` | Route commits |
| `ova` | Abort operation |

(Fish preserva el autocompletado nativo para todos estos.)

---

## 🪟 FASE 3 — Configuración Windows Terminal (15 min)

### 3.1 settings.json (template)

Archivo: `%LOCALAPPDATA%\Microsoft\WindowsTerminal\settings.json`

```json
{
  "$schema": "https://aka.ms/terminal-profiles-schema",
  "defaultProfile": "{YOUR-UBUNTU-GUID}",
  "profiles": {
    "defaults": {
      "font": { "face": "JetBrainsMono Nerd Font", "size": 12 },
      "padding": "8, 8, 8, 8",
      "useAcrylic": true,
      "acrylicOpacity": 0.85,
      "cursorShape": "filledBox",
      "cursorColor": "#FFFFFF"
    },
    "list": [
      {
        "name": "Ubuntu-24.04",
        "source": "Windows.Terminal.Wsl",
        "guid": "{YOUR-UBUNTU-GUID}",
        "colorScheme": "Tango Dark",
        "commandline": "wsl.exe -d Ubuntu-24.04 -- fish",
        "icon": "ms-appx:///ProfileIcons/{9acb9455-ca41-5af1-895f-720416cf4c81}.png",
        "startingDirectory": "~"
      },
      {
        "name": "PowerShell 7",
        "commandline": "pwsh.exe -NoLogo",
        "icon": "ms-appx:///ProfileIcons/{61c54bbd-c2c6-5271-96e7-009a571024ae}.png"
      },
      {
        "name": "Windows PowerShell 5.1",
        "commandline": "powershell.exe",
        "icon": "ms-appx:///ProfileIcons/{61c54bbd-c2c6-5271-96e7-009a571024ae}.png"
      }
    ]
  },
  "schemes": [
    {
      "name": "Tango Dark",
      "background": "#000000",
      "foreground": "#FFFFFF",
      "black": "#000000",
      "red": "#CC0000",
      "green": "#4E9A06",
      "yellow": "#C4A000",
      "blue": "#3465A4",
      "purple": "#75507B",
      "cyan": "#06989A",
      "white": "#D3D7CF"
    }
  ],
  "keybindings": [
    { "command": { "action": "splitPane", "split": "horizontal" }, "keys": "alt+shift+plus" },
    { "command": { "action": "splitPane", "split": "vertical" },   "keys": "alt+shift+-" },
    { "command": "closePane", "keys": "ctrl+shift+w" },
    { "command": "newTab",  "keys": "ctrl+shift+t" },
    { "command": "nextTab", "keys": "ctrl+pgdn" },
    { "command": "prevTab", "keys": "ctrl+pgup" }
  ]
}
```

### 3.2 Atajos globales (settings.json → actions)

| Acción | Atajo |
|---|---|
| Nueva tab | `Ctrl+Shift+T` |
| Cerrar tab | `Ctrl+Shift+W` |
| Split horizontal | `Alt+Shift++` |
| Split vertical | `Alt+Shift+-` |
| Siguiente pane | `Alt+Arrow` |
| Tab siguiente | `Ctrl+PageDown` |
| Tab anterior | `Ctrl+PageUp` |
| Quake mode (dropdown) | `Ctrl+~` |

### 3.3 Quake mode (terminal dropdown)

Habilitar en settings.json:
```json
"windowingBehavior": "useExisting",
"minimized": false,
"automaticLayout": true
```

---

## 🧪 FASE 4 — Validación final (5 min)

### 4.1 Test de regresión funcional

```powershell
# 1. PowerShell 7:
pwsh -Command "Write-Host 'PS7 OK'; $PSVersionTable.PSVersion"

# 2. WSL Ubuntu:
wsl -d Ubuntu-24.04 -- fish -c "echo OK_FISH; fish --version"

# 3. OVAV aliases (cargar fresh):
wsl -d Ubuntu-24.04 -- fish -c "type owc"

# 4. Exit code test (regresión exit 15):
wsl -d Ubuntu-24.04 -- fish -l -c 'exit 0'; echo "rc=$?"
# ESPERADO: rc=0

# 5. Path tests críticos:
wsl -d Ubuntu-24.04 -- bash -c 'echo $PATH | tr ":" "\n" | grep -E "(ovav|local/bin|cargo)"'
```

### 4.2 Checklist de aceptación

- [ ] `pwsh --version` retorna 7.6.3
- [ ] Windows Terminal abre con default Ubuntu-24.04
- [ ] Split panes funcionan en Ubuntu
- [ ] `owc`, `owd`, `owl`, `owv` ejecutan correctamente
- [ ] `exit` en Ubuntu retorna código 0
- [ ] No aparece más "Exited with code 15"
- [ ] git, pnpm, node, go, python, rust siguen disponibles en Ubuntu
- [ ] Nerd Font renderiza iconos en `ls`

---

## 📂 Archivos a persistir en OVAV (post-validación)

| Path | Contenido |
|---|---|
| `Systems/OVAV/.ovav/templates/powershell/Microsoft.PowerShell_profile.ps1` | Template Braka-Dev v1.0 |
| `Systems/OVAV/.ovav/templates/windows-terminal/settings.json` | Template Windows Terminal v1.0 |
| `Systems/OVAV/.ovav/templates/scripts/install-dev-stack.ps1` | Idempotent installer |
| `Systems/OVAV/.ovav/docs/migration-wezterm-to-powershell.md` | Migration guide ejecutable |

---

## 📅 Timeline

| Fase | Tiempo | Bloqueante para siguiente |
|---|---|---|
| 0 — Instalar | 10 min | sí |
| 1 — PS profile | 15 min | no (puede hacerse en paralelo) |
| 2 — Fish cleanup | 10 min | sí (validar exit 0) |
| 3 — Windows Terminal | 15 min | no |
| 4 — Validación | 5 min | fin |

**Total: ~55 minutos secuenciales, ~30 minutos en paralelo.**

---

## 🎁 Bonus — Idempotent installer script

```powershell
# install-dev-stack.ps1 — Idempotent
# Ejecutar UNA VEZ como admin. Re-ejecutable sin daño.

# 1. PowerShell 7 LTS
$pwshInstalled = (Get-Command pwsh -ErrorAction SilentlyContinue)
if (-not $pwshInstalled) {
    winget install --id Microsoft.PowerShell --version 7.6.3 --source winget
} else {
    Write-Host "✓ PowerShell 7 ya instalado: $($pwshInstalled.Source)" -ForegroundColor Green
}

# 2. Windows Terminal
$wtInstalled = (Get-Command wt -ErrorAction SilentlyContinue)
if (-not $wtInstalled) {
    winget install --id Microsoft.WindowsTerminal --source winget
} else {
    Write-Host "✓ Windows Terminal ya instalado" -ForegroundColor Green
}

# 3. Nerd Font
$nerdFont = Get-Item "C:\Windows\Fonts\JetBrainsMono*" -ErrorAction SilentlyContinue
if (-not $nerdFont) {
    winget install --id NerdFonts.JetBrainsMono --source winget
} else {
    Write-Host "✓ JetBrainsMono Nerd Font ya instalada" -ForegroundColor Green
}

# 4. Crear profile dir
$profileDir = Split-Path $PROFILE -Parent
if (-not (Test-Path $profileDir)) {
    New-Item -Path $profileDir -ItemType Directory -Force
}

Write-Host "`n✓ Setup base completo. Reiniciar Windows Terminal para aplicar cambios." -ForegroundColor Cyan
```

---

## 🔧 Troubleshooting anticipado

| Síntoma | Fix |
|---|---|
| `pwsh` no encontrado después de instalar | Reiniciar PowerShell / reiniciar Windows Terminal |
| Windows Terminal no muestra Ubuntu | Agregar profile desde settings.json; reiniciar WT |
| Ligaduras no se ven | Confirmar font установлена + restart WT; usar JetBrainsMono Nerd Font Mono |
| `wsl` retorna `0x800701bc` | `wsl --update` desde PowerShell admin |
| Ubuntu startup lento | Verificar `.wslconfig` tiene `vmIdleTimeout=-1` |
| Cursor invisible | Settings → cursor shape = "filledBox" |

---

## 📌 Comandos críticos para ejecutar en orden

```bash
# === DESDE POWERSHELL 5.1 COMO ADMIN ===
# Fase 0
winget install --id Microsoft.PowerShell --version 7.6.3 --source winget
winget install --id Microsoft.WindowsTerminal --source winget
winget install --id NerdFonts.JetBrainsMono --source winget

# === DESDE UBUNTU-24.04 (wsl -d Ubuntu-24.04) ===
# Fase 1 (instalar módulos via pwsh, no via bash)
pwsh -NoProfile -Command 'Install-Module PSReadLine,Terminal-Icons,posh-git,oh-my-posh -Force -SkipPublisherCheck'

# Fase 2 (limpiar hooks wezterm)
mv ~/.config/fish/conf.d/20-ovav-wezterm-osc7.fish ~/.config/fish/_archive_20260709_wezterm/
mv ~/.config/fish/conf.d/25-ovav-wezterm-git.fish ~/.config/fish/_archive_20260709_wezterm/
sed -i 's|set -gx OVAV_TERMINAL wezterm|# OVAV_TERMINAL deshabilitado — PowerShell migration 2026-07-09|' \
    ~/.config/fish/conf.d/30-ovav-runtime-tools.fish

# Validación final
wsl -d Ubuntu-24.04 -- fish -l -c 'echo MIGRATION_OK; exit 0'
```

---

## 🎯 Cierre

Cuando CEO ejecute y valide exit 0 estable, este plan se mueve a `Systems/OVAV/.ovav/registry/migration-log.md` como entrada fechada y se cierra el ciclo.

CEO: ¿ejecuto Fase 0 ahora si me das poder para escribir a PowerShell profile directory, o preferís correrlo vos paso a paso en PowerShell nativo?
