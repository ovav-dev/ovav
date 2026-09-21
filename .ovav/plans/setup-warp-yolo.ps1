# Warp 2026 — YOLO ALL Profile + Tab Configs Setup
# Run from PowerShell (Windows host, WSL access)
# CEO runs this ONCE → full YOLO mode + 8 tab configs

$ErrorActionPreference = 'Stop'

# Paths
$dst = "$env:APPDATA\warp\Warp\data\tab_configs"
$settingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml"
$wslSrc = "\\wsl$\Ubuntu\home\braka\Systems\ovav\.ovav\warp\tab-configs"

Write-Host "=== OVAV × Warp 2026 — Full YOLO Setup ===" -ForegroundColor Cyan
Write-Host ""

# ── Step 1: Create tab_configs dir if missing ────────────────────────
if (-not (Test-Path $dst)) {
    Write-Host "[1/4] Creating $dst ..." -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $dst -Force | Out-Null
} else {
    Write-Host "[1/4] tab_configs dir exists" -ForegroundColor Green
}

# ── Step 2: Import all 8 Tab Configs (4 old + 4 new environments) ────
Write-Host ""
Write-Host "[2/4] Importing Tab Configs from WSL ..." -ForegroundColor Yellow
if (-not (Test-Path $wslSrc)) {
    Write-Host "  ERROR: WSL source not found: $wslSrc" -ForegroundColor Red
    Write-Host "  Run: wsl --list --verbose" -ForegroundColor Yellow
    exit 1
}

$count = 0
Get-ChildItem -Path $wslSrc -Filter "*.toml" | ForEach-Object {
    $target = Join-Path $dst $_.Name
    Copy-Item $_.FullName $target -Force
    Write-Host "  + $($_.Name)" -ForegroundColor Green
    $count++
}
Write-Host "  Imported: $count Tab Configs" -ForegroundColor Cyan

# ── Step 3: Show CEO how to create YOLO ALL profile via UI ──────────
Write-Host ""
Write-Host "[3/4] YOLO ALL Profile setup (UI required)" -ForegroundColor Yellow
Write-Host "  Warp UI does NOT allow creating new profiles via TOML edit." -ForegroundColor Gray
Write-Host "  CEO must do this manually:" -ForegroundColor White
Write-Host ""
Write-Host "  1. Open Warp" -ForegroundColor White
Write-Host "  2. Settings (Ctrl+,) → Agents → Profiles" -ForegroundColor White
Write-Host "  3. Click '+ New profile'" -ForegroundColor White
Write-Host "  4. Name: 'YOLO ALL'" -ForegroundColor White
Write-Host "  5. Base model: MiniMax-M3" -ForegroundColor White
Write-Host "  6. Apply code diffs        -> Always allow" -ForegroundColor White
Write-Host "  7. Read files              -> Always allow" -ForegroundColor White
Write-Host "  8. Create plans            -> Always allow" -ForegroundColor White
Write-Host "  9. Execute commands        -> Always allow" -ForegroundColor White
Write-Host " 10. Interact running cmds   -> Always allow" -ForegroundColor White
Write-Host " 11. Ask clarifying question -> Never ask" -ForegroundColor White
Write-Host " 12. MCP permissions         -> allowlist (all)" -ForegroundColor White
Write-Host " 13. Save" -ForegroundColor White
Write-Host ""
Write-Host "  Then go to 'default' profile and set it as 'Make default' = 'YOLO ALL'" -ForegroundColor Gray
Write-Host ""

# ── Step 4: Secret redaction mode ─────────────────────────────────────
Write-Host "[4/4] Secret redaction mode" -ForegroundColor Yellow
Write-Host "  Settings → Features → Privacy → Secret redaction = 'asterisks'" -ForegroundColor White
Write-Host ""

Write-Host "=== Done! ===" -ForegroundColor Cyan
Write-Host "Restart Warp to apply Tab Configs." -ForegroundColor Yellow
Write-Host "Then 'Ctrl+T' → '+' menu → see 8 Tab Configs (4 old + 4 env_*)" -ForegroundColor Yellow