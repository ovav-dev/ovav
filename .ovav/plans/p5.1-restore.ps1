# P5.1 — Restore settings.toml from pre-p5 backup
# Warp UI rejected 'execution_profiles' values; revert to last known-good.

param()

$ErrorActionPreference = "Stop"
$SettingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml"
$BackupPath = "$env:LOCALAPPDATA\warp\Warp\config\backups\settings.toml.pre-p5-20260818-224857"

if (-not (Test-Path $BackupPath)) {
    Write-Error "Backup not found: $BackupPath"
}

if (-not (Test-Path $SettingsPath)) {
    Write-Error "Settings not found: $SettingsPath"
}

# Verify backup is the pre-p5 version (no execution_profiles subsections)
$backupContent = Get-Content -Path $BackupPath -Raw
if ($backupContent -match '\[agents\.execution_profiles\.(ovav_build|ovav_yolo|ovav_review|thavren_systems)\]') {
    Write-Error "Backup file contains the rejected profiles — wrong backup"
}

# Create additional safety backup of current (broken) state
$brokenBackup = "$env:LOCALAPPDATA\warp\Warp\config\backups\settings.toml.broken-p5-attempt-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
Copy-Item -Path $SettingsPath -Destination $brokenBackup -Force
Write-Host "✓ Broken state saved: $brokenBackup"

# Atomic restore via Copy-Item (destination overwrite)
Copy-Item -Path $BackupPath -Destination $SettingsPath -Force
Write-Host "✓ Restored settings.toml from pre-p5 backup"

# Verify
$verify = Get-Content -Path $SettingsPath -Raw
$tests = @(
    @{ n = "execution_profiles.ovav_build present"; bad = $verify -match '\[agents\.execution_profiles\.ovav_build\]' },
    @{ n = "execution_profiles.ovav_yolo present";  bad = $verify -match '\[agents\.execution_profiles\.ovav_yolo\]' },
    @{ n = "execution_profiles.ovav_review present"; bad = $verify -match '\[agents\.execution_profiles\.ovav_review\]' },
    @{ n = "execution_profiles.thavren_systems present"; bad = $verify -match '\[agents\.execution_profiles\.thavren_systems\]' }
)

Write-Host ""
Write-Host "=== Verification: rejected profiles should NOT be present ==="
$allClean = $true
foreach ($t in $tests) {
    $status = if ($t.bad) { "✗ STILL PRESENT"; $allClean = $false } else { "✓ removed" }
    Write-Host "$status $($t.n)"
}

if ($allClean) {
    Write-Host ""
    Write-Host "✅ Warp should restart cleanly. P2/P2.1 settings preserved."
    Write-Host "✅ P5 profiles removed — must be created via Warp UI"
} else {
    Write-Error "Restore verification failed"
}

exit 0
