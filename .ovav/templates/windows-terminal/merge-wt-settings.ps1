# merge-wt-settings.ps1 — Merge OVAV customizations into Windows Terminal settings.json
# =============================================================================
# Ejecutar DESDE PowerShell 7 COMO ADMIN (Run as Administrator)
# Lee el settings.json existente de WT, inyecta OVAV theme + keybindings.
# CEO Braka / OVAV Platform Engineering — 2026-07-10
# =============================================================================

$ErrorActionPreference = 'Stop'

# Find WT settings.json — winget installs to Packages\*, Store to Microsoft\
$WT_PACKAGE = Get-ChildItem "$env:LOCALAPPDATA\Packages\Microsoft.WindowsTerminal_*\LocalState\settings.json" -ErrorAction SilentlyContinue | Select-Object -First 1
$WT_DIRECT  = "$env:LOCALAPPDATA\Microsoft\Windows Terminal\settings.json"

if ($WT_PACKAGE) {
    $WT_SETTINGS = $WT_PACKAGE.FullName
} elseif (Test-Path $WT_DIRECT) {
    $WT_SETTINGS = $WT_DIRECT
} else {
    Write-Host "  ⚠ settings.json not found. Open Windows Terminal once (Win → wt) then re-run." -ForegroundColor Yellow
    exit 1
}

Write-Host "╔══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  OVAV — Windows Terminal Settings Merge                 ║" -ForegroundColor Cyan
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ── Step 0: Backup existing settings ──────────────────────────────────
if (Test-Path $WT_SETTINGS) {
    $backup = "$WT_SETTINGS.backup-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    Copy-Item $WT_SETTINGS $backup
    Write-Host "  ✔ Backup: $backup" -ForegroundColor Green
} else {
    Write-Host "  ⚠ No existing settings.json — WT may not have been opened yet." -ForegroundColor Yellow
    Write-Host "    Open Windows Terminal once (Win → wt) then re-run this script." -ForegroundColor Yellow
    exit 1
}

# ── Step 1: Read current settings ─────────────────────────────────────
Write-Host ""
Write-Host "[1/3] Reading current Windows Terminal settings..." -ForegroundColor Yellow
$settings = Get-Content $WT_SETTINGS -Raw | ConvertFrom-Json -Depth 10

# Show current profiles
Write-Host "  Current profiles:"
foreach ($p in $settings.profiles.list) {
    $name = if ($p.name) { $p.name } else { "(auto-detected)" }
    $guid = if ($p.guid) { $p.guid.Substring(0,8) + "..." } else { "(no guid)" }
    Write-Host "    - $name ($guid)" -ForegroundColor Gray
}

# ── Step 2: Inject OVAV customizations ────────────────────────────────
Write-Host ""
Write-Host "[2/3] Injecting OVAV customizations..." -ForegroundColor Yellow

# 2a — Set Ubuntu as default (find its GUID)
$ubuntuProfile = $settings.profiles.list | Where-Object { 
    $_.name -like "*Ubuntu*" -or $_.source -eq "Windows.Terminal.Wsl" 
} | Select-Object -First 1

if ($ubuntuProfile -and $ubuntuProfile.guid) {
    $settings | Add-Member -NotePropertyName "defaultProfile" -NotePropertyValue $ubuntuProfile.guid -Force
    Write-Host "  ✔ Default profile: Ubuntu ($($ubuntuProfile.name))" -ForegroundColor Green
} else {
    Write-Host "  ⚠ Ubuntu profile not found — Skipping defaultProfile" -ForegroundColor Yellow
}

# 2b — Startup size
$settings | Add-Member -NotePropertyName "startupSize" -NotePropertyValue @{rows=40; columns=120} -Force
Write-Host "  ✔ Startup size: 120x40" -ForegroundColor Green

# 2c — Inject Tango Dark theme (if not present)
if (-not $settings.schemes) {
    $settings | Add-Member -NotePropertyName "schemes" -NotePropertyValue @() -Force
}
$hasTango = $settings.schemes | Where-Object { $_.name -eq "Tango Dark" }
if (-not $hasTango) {
    $tango = @{
        name = "Tango Dark"
        background = "#000000"
        foreground = "#FFFFFF"
        black = "#000000"
        red = "#CC0000"
        green = "#4E9A06"
        yellow = "#C4A000"
        blue = "#3465A4"
        purple = "#75507B"
        cyan = "#06989A"
        white = "#D3D7CF"
        brightBlack = "#555753"
        brightRed = "#EF2929"
        brightGreen = "#8AE234"
        brightYellow = "#FCE94F"
        brightBlue = "#729FCF"
        brightPurple = "#AD7FA8"
        brightCyan = "#34E2E2"
        brightWhite = "#EEEEEC"
    }
    $settings.schemes += $tango
    Write-Host "  ✔ Tango Dark theme added" -ForegroundColor Green
} else {
    Write-Host "  - Tango Dark already present" -ForegroundColor DarkGray
}

# 2d — Profile defaults (font, acrylic, cursor, theme)
$defaults = @{
    font = @{ face = "JetBrainsMono Nerd Font"; size = 12 }
    padding = "8, 8, 8, 8"
    useAcrylic = $true
    acrylicOpacity = 0.85
    cursorShape = "filledBox"
    cursorColor = "#FFFFFF"
    colorScheme = "Tango Dark"
}

if (-not $settings.profiles.defaults) {
    $settings.profiles | Add-Member -NotePropertyName "defaults" -NotePropertyValue $defaults -Force
} else {
    $settings.profiles.defaults = $defaults
}
Write-Host "  ✔ Profile defaults: JetBrainsMono 12pt + Acrylic 85% + Tango Dark" -ForegroundColor Green

# 2e — Keybindings (merge, don't overwrite)
$ovavKeys = @(
    @{ command = @{ action = "splitPane"; split = "horizontal" }; keys = "alt+shift+plus" },
    @{ command = @{ action = "splitPane"; split = "vertical" }; keys = "alt+shift+-" },
    @{ command = "closePane"; keys = "ctrl+shift+w" },
    @{ command = "newTab"; keys = "ctrl+shift+t" },
    @{ command = "nextTab"; keys = "ctrl+pgdn" },
    @{ command = "prevTab"; keys = "ctrl+pgup" },
    @{ command = "closeTab"; keys = "ctrl+w" },
    @{ command = @{ action = "moveFocus"; direction = "up" }; keys = "alt+up" },
    @{ command = @{ action = "moveFocus"; direction = "down" }; keys = "alt+down" },
    @{ command = @{ action = "moveFocus"; direction = "left" }; keys = "alt+left" },
    @{ command = @{ action = "moveFocus"; direction = "right" }; keys = "alt+right" },
    @{ command = @{ action = "resizePane"; direction = "up" }; keys = "alt+shift+up" },
    @{ command = @{ action = "resizePane"; direction = "down" }; keys = "alt+shift+down" },
    @{ command = @{ action = "resizePane"; direction = "left" }; keys = "alt+shift+left" },
    @{ command = @{ action = "resizePane"; direction = "right" }; keys = "alt+shift+right" },
    @{ command = @{ action = "quakeMode" }; keys = "ctrl+``" },
    @{ command = "toggleFullscreen"; keys = "f11" },
    @{ command = "find"; keys = "ctrl+shift+f" }
)

if (-not $settings.keybindings) {
    $settings | Add-Member -NotePropertyName "keybindings" -NotePropertyValue @() -Force
}

# Only add keys that don't conflict with existing
$existingKeys = $settings.keybindings | ForEach-Object { $_.keys }
foreach ($k in $ovavKeys) {
    if ($k.keys -notin $existingKeys) {
        $settings.keybindings += $k
    }
}
Write-Host "  ✔ Keybindings merged" -ForegroundColor Green

# ── Step 3: Write back ────────────────────────────────────────────────
Write-Host ""
Write-Host "[3/3] Writing settings.json..." -ForegroundColor Yellow

$json = $settings | ConvertTo-Json -Depth 10
# Fix PowerShell's escaping of the backtick for quakeMode
$json = $json -replace 'ctrl\+`', 'ctrl+`'
Set-Content -Path $WT_SETTINGS -Value $json -Encoding UTF8 -Force

Write-Host "  ✔ Written: $WT_SETTINGS" -ForegroundColor Green

# ── Done ──────────────────────────────────────────────────────────────
Write-Host @"

╔══════════════════════════════════════════════════════════╗
║  WINDOWS TERMINAL CONFIGURED                              ║
╠══════════════════════════════════════════════════════════╣
║  ✓ Tango Dark theme (black bg)                          ║
║  ✓ JetBrainsMono Nerd Font 12pt                         ║
║  ✓ Acrylic blur 85%                                     ║
║  ✓ filledBox cursor white                               ║
║  ✓ Keybindings: splits, quake, tabs                     ║
║  ✓ Ubuntu as default profile                             ║
║                                                          ║
║  Next: Close & reopen Windows Terminal                   ║
║  Default tab should be Ubuntu-24.04 with Tango Dark      ║
╚══════════════════════════════════════════════════════════╝

"@ -ForegroundColor Cyan
