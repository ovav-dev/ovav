# Warp 2026 — Tab Configs Import Script
# CEO runs from PowerShell (Windows host)
# Copies 4 Tab Configs from WSL Ubuntu to Windows Warp config dir

$dst = "$env:APPDATA\warp\Warp\data\tab_configs"
$src = "\\wsl$\Ubuntu\home\braka\Systems\ovav\.ovav\warp\tab-configs"

Write-Host "=== OVAV Tab Configs Import ===" -ForegroundColor Cyan
Write-Host "Source: $src"
Write-Host "Dest:   $dst"
Write-Host ""

# Verify source
if (-not (Test-Path $src)) {
    Write-Host "ERROR: source dir not found: $src" -ForegroundColor Red
    exit 1
}

# Create dest if missing
if (-not (Test-Path $dst)) {
    Write-Host "Creating $dst ..."
    New-Item -ItemType Directory -Path $dst -Force | Out-Null
}

# Copy all .toml
$count = 0
Get-ChildItem -Path $src -Filter "*.toml" | ForEach-Object {
    $target = Join-Path $dst $_.Name
    Copy-Item $_.FullName $target -Force
    Write-Host "  + $($_.Name)" -ForegroundColor Green
    $count++
}

Write-Host ""
Write-Host "=== Done: $count files copied ===" -ForegroundColor Cyan
Write-Host "Next: open Warp, click '+' button — should see 4 OVAV configs" -ForegroundColor Yellow
Write-Host "Restart Warp to refresh."