# ─────────────────────────────────────────────────────────────
#  OVAV PowerShell 7 Profile — PSReadLine Predictive IntelliSense
#  Rule #20: Predictive IntelliSense on (HistoryAndPlugin, InlineView)
#  Rule #21: theme follows Windows app theme (Dark/Light) automatically
#  Rule #29: prefers terminal defaults over hardcoded colors
# ─────────────────────────────────────────────────────────────

# ─── Module pre-load (idempotent) ──────────────────────────
$null = [Microsoft.PowerShell.PSConsoleReadLine]::ReadLine

# ─── PSReadLine configuration ───────────────────────────────
Set-PSReadLineOption -EditMode Windows
Set-PSReadLineOption -BellStyle None

# Predictive IntelliSense — HistoryAndPlugin (rule #20)
Set-PSReadLineOption -PredictionSource HistoryAndPlugin
Set-PSReadLineOption -PredictionViewStyle InlineView

# Keybinds — no conflicts with OpenCode, Readline defaults
Set-PSReadLineKeyHandler -Key UpArrow   -Function HistorySearchBackward
Set-PSReadLineKeyHandler -Key DownArrow -Function HistorySearchForward
Set-PSReadLineKeyHandler -Key Tab       -Function MenuComplete
Set-PSReadLineKeyHandler -Key 'Ctrl+d'  -Function DeleteCharOrExit

# ─── OVAV helper functions ─────────────────────────────────
function Get-OvavRoot { '/home/braka/Systems/ovav' }

function Enter-Ovav {
    Set-Location (Get-OvavRoot)
    Write-Host "OVAV workspace" -ForegroundColor Cyan
}

function Start-OvavWorkspace {
    # Launch OVAV Workspace action via WT command
    & wt.exe -w 0 new-tab -p "OVAV Ubuntu"
}

function Show-OvavStatus {
    & ovav status
}

Set-Alias -Name ovproj -Value Enter-Ovav -Force
Set-Alias -Name ovs    -Value Show-OvavStatus -Force
Set-Alias -Name ovv    -Value ovav -Value validate -Force

# ─── Theme sync — follows Windows app theme ────────────────
# Registry: HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize
# AppsUseLightTheme = 0 → Dark, 1 → Light
$ovavThemeReg = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize'
$ovavIsLight  = (Get-ItemProperty -Path $ovavThemeReg -Name AppsUseLightTheme -ErrorAction SilentlyContinue).AppsUseLightTheme -eq 1

if ($ovavIsLight) {
    $Host.UI.RawUI.BackgroundColor = '#F7F9FC'
    $Host.UI.RawUI.ForegroundColor = '#1B2430'
} else {
    $Host.UI.RawUI.BackgroundColor = '#0B1020'
    $Host.UI.RawUI.ForegroundColor = '#D8E1F0'
}

# ─── PSReadLine colors — adaptive to theme ─────────────────
if ($ovavIsLight) {
    Set-PSReadLineOption -Colors @{
        Command            = '#1B2430'
        Parameter          = '#2563EB'
        Operator           = '#475569'
        Variable           = '#7C3AED'
        String             = '#1F7A4C'
        Number             = '#8A5A00'
        Type               = '#2563EB'
        Comment            = '#94A3B8'
        Keyword            = '#7C3AED'
        Error              = '#C24156'
        Selection          = '#BFDBFE'
        InlinePrediction   = '#94A3B8'
        History            = '#1B2430'
    }
} else {
    Set-PSReadLineOption -Colors @{
        Command            = '#D8E1F0'
        Parameter          = '#6EA8FE'
        Operator           = '#8B95A8'
        Variable           = '#C099FF'
        String             = '#7EE787'
        Number             = '#F2CC60'
        Type               = '#6EA8FE'
        Comment            = '#4A5568'
        Keyword            = '#C099FF'
        Error              = '#FF6B81'
        Selection          = '#2A3A5C'
        InlinePrediction   = '#4A5568'
        History            = '#D8E1F0'
    }
}

# ─── Path augmentation (canonical Linux install) ────────────
if ($env:Path -notmatch [regex]::Escape('/usr/local/bin')) {
    $env:Path = '/usr/local/bin:' + $env:Path
}