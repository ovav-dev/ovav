# P2 Mutation Script — Warp settings.toml
# Applies 4 changes per master plan:
# - Delete [session.new_session_shell_override] block (§2)
# - show_panel_in_restored_windows = true (§4)
# - show_warning_before_quitting = true (§6)
# - input_box_type_setting = "terminal" (§6)

param(
    [string]$SettingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml",
    [string]$BackupDir = "$env:LOCALAPPDATA\warp\Warp\config\backups"
)

$ErrorActionPreference = "Stop"

# ── Pre-flight: backup ──────────────────────────────────────────────────────
if (-not (Test-Path $SettingsPath)) {
    Write-Error "Settings file not found: $SettingsPath"
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backupPath = Join-Path $BackupDir "settings.toml.backup-$timestamp"
Copy-Item -Path $SettingsPath -Destination $backupPath -Force
Write-Host "✓ Backup: $backupPath"

# ── Read current settings ───────────────────────────────────────────────────
$content = Get-Content -Path $SettingsPath -Raw

# ── Mutation 1: delete [session.new_session_shell_override] block ──────────
$pattern1 = '(?ms)\n*\[session\.new_session_shell_override\]\s*\nwsl\s*=\s*"Ubuntu-26\.04"\s*\n*'
if ($content -match $pattern1) {
    $content = $content -replace $pattern1, "`n"
    Write-Host "✓ Mutation 1: removed [session.new_session_shell_override]"
} else {
    Write-Warning "Mutation 1: pattern not found (skipping)"
}

# ── Mutation 2: show_panel_in_restored_windows = true ──────────────────────
$pattern2 = '(\[appearance\.vertical_tabs\][^\[]*?show_panel_in_restored_windows\s*=\s*)false'
if ($content -match $pattern2) {
    $content = $content -replace $pattern2, '${1}true'
    Write-Host "✓ Mutation 2: show_panel_in_restored_windows = true"
} else {
    Write-Warning "Mutation 2: pattern not found (skipping)"
}

# ── Mutation 3: show_warning_before_quitting = true ────────────────────────
$pattern3 = '(\[general\][^\[]*?show_warning_before_quitting\s*=\s*)false'
if ($content -match $pattern3) {
    $content = $content -replace $pattern3, '${1}true'
    Write-Host "✓ Mutation 3: show_warning_before_quitting = true"
} else {
    Write-Warning "Mutation 3: pattern not found (skipping)"
}

# ── Mutation 4: input_box_type_setting = "terminal" ────────────────────────
$pattern4 = '(input_box_type_setting\s*=\s*)"universal"'
if ($content -match $pattern4) {
    $content = $content -replace $pattern4, '${1}"terminal"'
    Write-Host "✓ Mutation 4: input_box_type_setting = `"terminal`""
} else {
    Write-Warning "Mutation 4: pattern not found (skipping)"
}

# ── Atomic write ────────────────────────────────────────────────────────────
Set-Content -Path $SettingsPath -Value $content -NoNewline -Encoding UTF8
Write-Host "✓ Saved: $SettingsPath"

# ── Post-write verification ────────────────────────────────────────────────
$verify = Get-Content -Path $SettingsPath -Raw
$checks = @(
    @{ name = "new_session_shell_override removed";  test = $verify -notmatch '\[session\.new_session_shell_override\]' },
    @{ name = "show_panel_in_restored_windows = true"; test = $verify -match 'show_panel_in_restored_windows\s*=\s*true' },
    @{ name = "show_warning_before_quitting = true";  test = $verify -match 'show_warning_before_quitting\s*=\s*true' },
    @{ name = "input_box_type_setting = terminal";    test = $verify -match 'input_box_type_setting\s*=\s*"terminal"' }
)

Write-Host ""
Write-Host "=== Verification ==="
$allPass = $true
foreach ($c in $checks) {
    $status = if ($c.test) { "✓" } else { "✗" ; $allPass = $false }
    Write-Host "$status $($c.name)"
}

if (-not $allPass) {
    Write-Error "Verification failed — restore from backup if needed"
    exit 1
}

Write-Host ""
Write-Host "=== Backup path (keep for rollback) ==="
Write-Host $backupPath
exit 0
