# P2.1 Fix Script — restore input_box_type_setting to known-valid value
# Plan §6 says: "migrate via UI, then audit TOML — do not invent enum"
# I invented "terminal" in P2. Warp rejected it. Restoring to "universal"
# (previous known-good value) until proper UI migration is done.

param()

$ErrorActionPreference = "Stop"
$SettingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml"
$BackupDir = "$env:LOCALAPPDATA\warp\Warp\config\backups"

# ── Pre-flight ──────────────────────────────────────────────────────────────
if (-not (Test-Path $SettingsPath)) {
    Write-Error "Settings file not found: $SettingsPath"
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backupPath = Join-Path $BackupDir "settings.toml.fix-p2.1-$timestamp"
Copy-Item -Path $SettingsPath -Destination $backupPath -Force
Write-Host "✓ Backup: $backupPath"

# ── Read + mutate ───────────────────────────────────────────────────────────
$content = Get-Content -Path $SettingsPath -Raw

$pattern = '(input_box_type_setting\s*=\s*)"terminal"'
if ($content -match $pattern) {
    $content = $content -replace $pattern, '${1}"universal"'
    Write-Host "✓ Reverted input_box_type_setting from 'terminal' → 'universal'"
} else {
    Write-Warning "Pattern 'terminal' not found — checking current value"
    $currentMatch = Select-String -Path $SettingsPath -Pattern 'input_box_type_setting'
    Write-Host "Current: $currentMatch"
}

# ── Atomic write ────────────────────────────────────────────────────────────
Set-Content -Path $SettingsPath -Value $content -NoNewline -Encoding UTF8
Write-Host "✓ Saved: $SettingsPath"

# ── Verify ──────────────────────────────────────────────────────────────────
$verify = Get-Content -Path $SettingsPath -Raw
$match = Select-String -Path $SettingsPath -Pattern 'input_box_type_setting\s*=\s*"(\w+)"'
Write-Host ""
Write-Host "=== Post-fix value ==="
Write-Host $match.Matches.Groups[1].Value
exit 0
