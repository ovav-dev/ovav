# install-powershell-ovav-stack.ps1
# Idempotent. Run once from PowerShell 7 (pwsh) admin.
# CEO Braka / OVAV Platform Engineering - 2026-07-09

#Requires -RunAsAdministrator
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Write-Host "=== OVAV PowerShell 7 Stack Installer ===" -ForegroundColor Cyan
Write-Host "User: $env:USERNAME"
Write-Host "Profile: $PROFILE"
Write-Host ""

# ── 0. Execution policy ─────────────────────────────────────────────────────
Write-Host "[0/5] Set execution policy..." -ForegroundColor Yellow
Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned -Force

# ── 1. Install PSReadLine, Terminal-Icons, posh-git, oh-my-posh ─────────────
Write-Host "[1/5] Install PSReadLine, Terminal-Icons, posh-git, oh-my-posh..." -ForegroundColor Yellow
Write-Host "  (If PSReadLine shows 'currently in use' error, close and reopen this terminal, then re-run.)" -ForegroundColor Gray

$modules = @('Terminal-Icons','posh-git','oh-my-posh')
foreach ($m in $modules) {
    if (-not (Get-Module -ListAvailable -Name $m)) {
        Install-Module -Name $m -Force -SkipPublisherCheck -Scope CurrentUser -AllowClobber
        Write-Host "  + installed $m" -ForegroundColor Green
    } else {
        Write-Host "  - $m already installed" -ForegroundColor DarkGray
    }
}

try {
    Install-Module -Name PSReadLine -Force -SkipPublisherCheck -Scope CurrentUser -AllowClobber
    Write-Host "  + PSReadLine updated" -ForegroundColor Green
} catch {
    Write-Host "  ! PSReadLine in use; will pick up on next session start" -ForegroundColor DarkYellow
}

# ── 2. Verify Nerd Font ─────────────────────────────────────────────────────
Write-Host ""
Write-Host "[2/5] Verify JetBrainsMono Nerd Font..." -ForegroundColor Yellow
$font = Get-Item "C:\Windows\Fonts\JetBrainsMono*" -ErrorAction SilentlyContinue
if ($font) {
    Write-Host "  + JetBrainsMono Nerd Font installed" -ForegroundColor Green
} else {
    Write-Host "  ! Not found. Run: winget install --id NerdFonts.JetBrainsMono" -ForegroundColor Yellow
}

# ── 3. Write $PROFILE ───────────────────────────────────────────────────────
Write-Host ""
Write-Host "[3/5] Write $PROFILE..." -ForegroundColor Yellow

$profileDir = Split-Path -Path $PROFILE -Parent
if (-not (Test-Path $profileDir)) {
    New-Item -Path $profileDir -ItemType Directory -Force | Out-Null
}

$profileContent = @'
# PowerShell 7 Profile - Braka-Dev v1.0
# Plataforma: Windows Terminal + WSL2 Ubuntu-24.04 + fish 4.7.1
# Auto-managed by install-powershell-ovav-stack.ps1

# PSReadLine
Import-Module PSReadLine
Set-PSReadLineOption -PredictionSource History
Set-PSReadLineOption -EditMode Windows
Set-PSReadLineOption -BellStyle None
Set-PSReadLineOption -HistorySearchCursorMovesToEnd
Set-PSReadLineKeyHandler -Key UpArrow   -Function HistorySearchBackward
Set-PSReadLineKeyHandler -Key DownArrow -Function HistorySearchForward
Set-PSReadLineKeyHandler -Key Tab       -Function MenuComplete
Set-PSReadLineKeyHandler -Key Ctrl+d    -Function DeleteChar

# Terminal-Icons
Import-Module Terminal-Icons

# posh-git + oh-my-posh
Import-Module posh-git
Import-Module oh-my-posh
Set-PoshPrompt -Theme paradox

# Bridge aliases hacia fish/WSL (OVAV)
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

# Atajos locales
Set-Alias -Name ll -Value ls -Force -Option AllScope
Set-Alias -Name la -Value ls -Force -Option AllScope
function gs  { git status --short --branch @args }
function gd  { git diff @args }
function gds { git diff --staged @args }
function ga  { git add @args }
function gc  { git commit @args }
function gp  { git push @args }
function g   { git @args }

Write-Host "PS7 OK - Braka-Dev v1.0" -ForegroundColor Cyan
'@

Set-Content -Path $PROFILE -Value $profileContent -Encoding UTF8 -Force
Write-Host "  + $PROFILE written" -ForegroundColor Green

# ── 4. Reload profile ───────────────────────────────────────────────────────
Write-Host ""
Write-Host "[4/5] Reload profile..." -ForegroundColor Yellow
& $PROFILE

Write-Host ""
Write-Host "[5/5] DONE. Open a new tab in Windows Terminal, run: owc --help" -ForegroundColor Green
Write-Host "If any owc/owd alias errors with 'fish: command not found', open Ubuntu tab first." -ForegroundColor Gray