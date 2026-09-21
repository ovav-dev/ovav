# P5.2 revert — settings.toml restored

$ErrorActionPreference = "Stop"
$SettingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml"
$BackupPath = "$env:LOCALAPPDATA\warp\Warp\config\backups\settings.toml.pre-p5v2-20260819-000213"

# Verify backup exists and is the pre-p5v2 (default config)
$backupContent = Get-Content $BackupPath -Raw
if ($backupContent -match 'execution_profiles\.ovav') {
    Write-Error "Backup contains P5 profiles — wrong backup"
    exit 1
}

# Safety backup of current (broken) state
$brokenBackup = "$env:LOCALAPPDATA\warp\Warp\config\backups\settings.toml.broken-p5v2-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
Copy-Item -Path $SettingsPath -Destination $brokenBackup -Force
Write-Host "✓ Broken state saved: $brokenBackup"

# Restore
Copy-Item -Path $BackupPath -Destination $SettingsPath -Force
Write-Host "✓ Restored settings.toml from pre-p5v2 backup"

# Verify
$verify = Get-Content -Path $SettingsPath -Raw
$tests = @(
    @{ n = "execution_profiles.ovav blocks absent"; bad = $verify -match 'execution_profiles\.ovav' },
    @{ n = "default profile present";                 good = $verify -match '\[agents\.execution_profiles\.default\]' },
    @{ n = "base_model = auto-genius (default)";      good = $verify -match 'base_model = "auto-genius"' }
)

Write-Host ""
Write-Host "=== Verification ==="
foreach ($t in $tests) {
    $status = if ($t.bad) { "✗ STILL BROKEN" } elseif ($t.good) { "✓ restored" } else { "?" }
    $key = if ($t.bad) { "bad" } else { "good" }
    Write-Host "$status $($t.n) ($key)"
}

Write-Host ""
Write-Host "Restart Warp to clear the error banner."
