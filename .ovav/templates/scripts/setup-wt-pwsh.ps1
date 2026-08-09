# setup-wt-pwsh.ps1 — OVAV Windows Terminal + PowerShell 7 — One-Shot Setup
# =============================================================================
# Ejecutar DESDE PowerShell 7 COMO ADMINISTRADOR (click derecho → Run as Admin)
# Idempotente: podés re-ejecutarlo sin daño.
# CEO Braka / OVAV Platform Engineering — 2026-07-10
# =============================================================================

#Requires -RunAsAdministrator
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'
$OVAV_REPO = "\\wsl$\Ubuntu-24.04\home\braka\Systems\OVAV"

Write-Host @"
╔══════════════════════════════════════════════════════════╗
║  OVAV — Windows Terminal + PowerShell 7 Setup v2.0      ║
║  Idempotent installer. Re-runnable without damage.      ║
╚══════════════════════════════════════════════════════════╝
"@ -ForegroundColor Cyan

# ═══════════════════════════════════════════════════════════════════════
# FASE 0 — Install dependencies (winget)
# ═══════════════════════════════════════════════════════════════════════
Write-Host "`n[FASE 0] Installing dependencies via winget..." -ForegroundColor Yellow

# 0.1 — Windows Terminal
$wt = Get-Command wt -ErrorAction SilentlyContinue
if ($wt) {
    Write-Host "  ✔ Windows Terminal: $($wt.Source)" -ForegroundColor Green
} else {
    Write-Host "  ⬇ Installing Windows Terminal..." -ForegroundColor Cyan
    winget install --id Microsoft.WindowsTerminal --source winget --accept-source-agreements --accept-package-agreements
}

# 0.2 — PowerShell 7.6 LTS
$pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
if ($pwsh) {
    Write-Host "  ✔ PowerShell 7: $($pwsh.Source) — v$($PSVersionTable.PSVersion)" -ForegroundColor Green
} else {
    Write-Host "  ⬇ Installing PowerShell 7.6 LTS..." -ForegroundColor Cyan
    winget install --id Microsoft.PowerShell --source winget --accept-source-agreements --accept-package-agreements
}

# 0.3 — JetBrainsMono Nerd Font
$font = Get-Item "C:\Windows\Fonts\JetBrainsMono*" -ErrorAction SilentlyContinue
if ($font) {
    Write-Host "  ✔ JetBrainsMono Nerd Font: installed" -ForegroundColor Green
} else {
    Write-Host "  ⬇ Installing JetBrainsMono Nerd Font..." -ForegroundColor Cyan
    winget install --id NerdFonts.JetBrainsMono --source winget --accept-source-agreements --accept-package-agreements
}

# ═══════════════════════════════════════════════════════════════════════
# FASE 1 — PowerShell 7 $PROFILE
# ═══════════════════════════════════════════════════════════════════════
Write-Host "`n[FASE 1] Configuring PowerShell 7 profile..." -ForegroundColor Yellow

# 1.1 — Install oh-my-posh (winget, NOT PS module — module is deprecated 2026)
$omp = Get-Command oh-my-posh -ErrorAction SilentlyContinue
if ($omp) {
    Write-Host "  ✔ oh-my-posh: $($omp.Source)" -ForegroundColor Green
} else {
    Write-Host "  ⬇ Installing oh-my-posh (winget)..." -ForegroundColor Cyan
    winget install --id JanDeDobbeleer.OhMyPosh --source winget --accept-source-agreements --accept-package-agreements
    # Refresh PATH so oh-my-posh.exe is found immediately
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# 1.2 — Install PS modules (Terminal-Icons + posh-git only)
$modules = @('Terminal-Icons', 'posh-git')
foreach ($m in $modules) {
    if (Get-Module -ListAvailable -Name $m) {
        Write-Host "  ✔ Module $m" -ForegroundColor Green
    } else {
        Write-Host "  ⬇ Installing $m..." -ForegroundColor Cyan
        Install-Module -Name $m -Force -SkipPublisherCheck -Scope CurrentUser -AllowClobber
    }
}

# PSReadLine (may be in use — catch gracefully)
try {
    Install-Module -Name PSReadLine -Force -SkipPublisherCheck -Scope CurrentUser -AllowClobber
    Write-Host "  ✔ PSReadLine updated" -ForegroundColor Green
} catch {
    Write-Host "  ⚠ PSReadLine in use — will apply next session" -ForegroundColor DarkYellow
}

# 1.2 — Write $PROFILE
$profileDir = Split-Path -Path $PROFILE -Parent
if (-not (Test-Path $profileDir)) {
    New-Item -Path $profileDir -ItemType Directory -Force | Out-Null
}

$profileContent = @'
# ══════════════════════════════════════════════════════════════════════════
# PowerShell 7 Profile — Braka-Dev v2.0
# Plataforma: Windows Terminal + WSL2 Ubuntu-24.04 + fish 4.7.1
# Generado por: setup-wt-pwsh.ps1 (OVAV Platform Engineering)
# ══════════════════════════════════════════════════════════════════════════

# ── PSReadLine (history, predictions, keybindings) ─────────────────────────
Import-Module PSReadLine
Set-PSReadLineOption -PredictionSource History
Set-PSReadLineOption -EditMode Windows
Set-PSReadLineOption -BellStyle None
Set-PSReadLineOption -HistorySearchCursorMovesToEnd
Set-PSReadLineKeyHandler -Key UpArrow   -Function HistorySearchBackward
Set-PSReadLineKeyHandler -Key DownArrow -Function HistorySearchForward
Set-PSReadLineKeyHandler -Key Tab       -Function MenuComplete
Set-PSReadLineKeyHandler -Key Ctrl+d    -Function DeleteChar

# ── Terminal-Icons (vscode-style icons in ls) ──────────────────────────────
if (Get-Module -ListAvailable -Name Terminal-Icons) { Import-Module Terminal-Icons }

# ── posh-git (git status in prompt) ───────────────────────────────────────
if (Get-Module -ListAvailable -Name posh-git) { Import-Module posh-git }

# ── oh-my-posh (standalone .exe, NOT PS module — module deprecated 2026) ──
$omp = Get-Command oh-my-posh -ErrorAction SilentlyContinue
if ($omp) {
    # Check if paradox theme exists; fallback to default if not
    $paradoxTheme = "$env:POSH_THEMES_PATH\paradox.omp.json"
    if (Test-Path $paradoxTheme) {
        oh-my-posh init pwsh --config $paradoxTheme | Invoke-Expression
    } else {
        oh-my-posh init pwsh | Invoke-Expression
    }
}

# ── OVAV Bridge aliases (PowerShell → WSL2 fish) ─────────────────────────
function wsl-fish { wsl -d Ubuntu-24.04 -- fish @args }
function owc  { wsl -d Ubuntu-24.04 -- fish -c "owc $args" }
function owd  { wsl -d Ubuntu-24.04 -- fish -c "owd $args" }
function owl  { wsl -d Ubuntu-24.04 -- fish -c "owl $args" }
function owv  { wsl -d Ubuntu-24.04 -- fish -c "owv $args" }
function ows  { wsl -d Ubuntu-24.04 -- fish -c "ows $args" }
function owr  { wsl -d Ubuntu-24.04 -- fish -c "owr $args" }
function owx  { wsl -d Ubuntu-24.04 -- fish -c "owx $args" }
function owa  { wsl -d Ubuntu-24.04 -- fish -c "owa $args" }
function obc  { wsl -d Ubuntu-24.04 -- fish -c "obc $args" }
function ovls { wsl -d Ubuntu-24.04 -- fish -c "ovls $args" }
function ovs  { wsl -d Ubuntu-24.04 -- fish -c "ovs $args" }

# ── Git shortcuts ─────────────────────────────────────────────────────────
function gs  { git status --short --branch @args }
function gd  { git diff @args }
function gds { git diff --staged @args }
function ga  { git add @args }
function gc  { git commit @args }
function gp  { git push @args }
function g   { git @args }

# ── Quick aliases ─────────────────────────────────────────────────────────
Set-Alias -Name ll -Value ls -Force -Option AllScope
Set-Alias -Name la -Value ls -Force -Option AllScope

# ── Execution policy ──────────────────────────────────────────────────────
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force

Write-Host "✓ PS7 — Braka-Dev v2.0" -ForegroundColor Cyan
'@

Set-Content -Path $PROFILE -Value $profileContent -Encoding UTF8 -Force
Write-Host "  ✔ Profile written: $PROFILE" -ForegroundColor Green

# 1.3 — Reload profile
try {
    & $PROFILE
} catch {
    Write-Host "  ⚠ Reload had warnings (expected on first run)" -ForegroundColor DarkYellow
}

# ═══════════════════════════════════════════════════════════════════════
# FASE 2 — Windows Terminal settings.json
# ═══════════════════════════════════════════════════════════════════════
Write-Host "`n[FASE 2] Configuring Windows Terminal..." -ForegroundColor Yellow

$wtSettingsDir = "$env:LOCALAPPDATA\Microsoft\Windows Terminal"
$wtSettingsFile = "$wtSettingsDir\settings.json"

Write-Host "  Target: $wtSettingsFile" -ForegroundColor Gray

# Check if WSL repo is reachable
$templatePath = "$OVAV_REPO\.ovav\templates\windows-terminal\settings.json"
$templateLocal = "$env:TEMP\ovav-wt-settings-template.json"

if (Test-Path $templatePath) {
    Write-Host "  ✔ Template found in OVAV repo" -ForegroundColor Green
    Write-Host ""
    Write-Host "  ╔══════════════════════════════════════════════════════╗" -ForegroundColor Yellow
    Write-Host "  ║  MANUAL STEP REQUIRED                               ║" -ForegroundColor Yellow
    Write-Host "  ╠══════════════════════════════════════════════════════╣" -ForegroundColor Yellow
    Write-Host "  ║  The settings.json template needs GUIDs replaced.   ║" -ForegroundColor Yellow
    Write-Host "  ║  1. Open Windows Terminal → Settings (Ctrl+,)       ║" -ForegroundColor Yellow
    Write-Host "  ║  2. Copy the profile GUIDs for:                     ║" -ForegroundColor Yellow
    Write-Host "  ║     - Ubuntu-24.04                                  ║" -ForegroundColor Yellow
    Write-Host "  ║     - PowerShell 7                                  ║" -ForegroundColor Yellow
    Write-Host "  ║  3. Replace {YOUR-*-GUID} in the template file      ║" -ForegroundColor Yellow
    Write-Host "  ║     Template: $templatePath" -ForegroundColor Yellow
    Write-Host "  ║  4. Paste into Windows Terminal settings.json       ║" -ForegroundColor Yellow
    Write-Host "  ╚══════════════════════════════════════════════════════╝" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  OR: Copy the template from WSL with:" -ForegroundColor Gray
    Write-Host "    cp '$templatePath' '$wtSettingsFile'" -ForegroundColor Gray
    Write-Host "    (then edit GUIDs manually)" -ForegroundColor Gray
} else {
    Write-Host "  ⚠ Template not found at $templatePath" -ForegroundColor DarkYellow
    Write-Host "    Ensure OVAV repo is at \\wsl$\Ubuntu-24.04\home\braka\Systems\OVAV" -ForegroundColor Gray
}

# ═══════════════════════════════════════════════════════════════════════
# FASE 3 — Verification
# ═══════════════════════════════════════════════════════════════════════
Write-Host "`n[FASE 3] Verification..." -ForegroundColor Yellow

# Test PS7
Write-Host "  PowerShell version: $($PSVersionTable.PSVersion)" -ForegroundColor White

# Test WSL
$wslTest = wsl -d Ubuntu-24.04 -- fish -c 'echo OK_FISH; fish --version' 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  ✔ WSL2 Ubuntu + fish: OK" -ForegroundColor Green
} else {
    Write-Host "  ⚠ WSL2 test failed — is Ubuntu-24.04 running?" -ForegroundColor DarkYellow
}

# Test OVAV aliases
$ovavTest = wsl -d Ubuntu-24.04 -- fish -c 'type owc' 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  ✔ OVAV aliases (owc): loaded" -ForegroundColor Green
} else {
    Write-Host "  ⚠ owc not found — run fish session and verify ~/.config/fish/" -ForegroundColor DarkYellow
}

# Test exit code (regression: exit 15)
$exitTest = wsl -d Ubuntu-24.04 -- fish -l -c 'exit 0' 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "  ✔ Exit code test: rc=0 (exit 15 fixed)" -ForegroundColor Green
} else {
    Write-Host "  ⚠ Exit code: $LASTEXITCODE — possible regression" -ForegroundColor DarkYellow
}

# ═══════════════════════════════════════════════════════════════════════
# DONE
# ═══════════════════════════════════════════════════════════════════════
Write-Host @"

╔══════════════════════════════════════════════════════════╗
║  SETUP COMPLETE                                          ║
╠══════════════════════════════════════════════════════════╣
║  Next steps:                                             ║
║  1. Open Windows Terminal (Win key → 'wt')               ║
║  2. Set Ubuntu-24.04 as default profile (Ctrl+,)         ║
║  3. Run FISH cleanup script (in Ubuntu tab):             ║
║     cd ~/Systems/OVAV                                    ║
║     bash .ovav/templates/scripts/fish-phase2-cleanup.sh  ║
║  4. Reopen Windows Terminal                              ║
║  5. Profit: tabs, splits, quake, acrylic, Nerd Fonts     ║
╚══════════════════════════════════════════════════════════╝

"@ -ForegroundColor Cyan
