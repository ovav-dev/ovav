# Obsolete compatibility wrapper. The Go planner owns Windows Terminal merges.
[CmdletBinding()]
param(
    [string]$SettingsPath,
    [string]$FragmentPath,
    [switch]$Apply
)

$ErrorActionPreference = 'Stop'

if ($Apply) {
    throw 'This obsolete wrapper never writes settings. Review the dry-run plan and use a separately approved installer.'
}

if (-not $SettingsPath) {
    $packageSettings = Get-ChildItem "$env:LOCALAPPDATA\Packages\Microsoft.WindowsTerminal_*\LocalState\settings.json" -ErrorAction SilentlyContinue | Select-Object -First 1
    $directSettings = "$env:LOCALAPPDATA\Microsoft\Windows Terminal\settings.json"
    if ($packageSettings) {
        $SettingsPath = $packageSettings.FullName
    } elseif (Test-Path -LiteralPath $directSettings) {
        $SettingsPath = $directSettings
    } else {
        throw 'Windows Terminal settings.json not found. Open Windows Terminal once, then retry.'
    }
}

if (-not $FragmentPath) {
    $FragmentPath = Join-Path $PSScriptRoot '..\..\source\configs\windows-terminal\ovav.fragment.json'
}

foreach ($path in @($SettingsPath, $FragmentPath)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required JSON file not found: $path"
    }
    $json = Get-Content -LiteralPath $path -Raw
    if (-not (Test-Json -Json $json -ErrorAction Stop)) {
        throw "Invalid JSON: $path"
    }
}

Write-Warning 'merge-wt-settings.ps1 is obsolete and dry-run only. No backup or settings write will occur.'
Write-Host 'The plan includes the timestamped backup path an approved installer must create before applying.'

& ovav terminal windows plan --settings $SettingsPath --fragment $FragmentPath
if ($LASTEXITCODE -ne 0) {
    throw "OVAV Windows Terminal planner failed with exit code $LASTEXITCODE"
}
