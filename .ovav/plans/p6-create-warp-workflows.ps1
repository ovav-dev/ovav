#!/usr/bin/env powershell
# P6 — Warp Workflow Creator
# Reads .ovav/warp/workflows.json and creates each workflow via Warp Drive
# REQUIRES: Warp Desktop running, user logged in
# Usage: powershell -ExecutionPolicy Bypass -File .ovav/plans/p6-create-warp-workflows.ps1

param(
    [string]$ManifestPath = "$PSScriptRoot\..\warp\workflows.json"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $ManifestPath)) {
    Write-Error "Manifest not found: $ManifestPath"
}

$manifest = Get-Content -Path $ManifestPath -Raw | ConvertFrom-Json

Write-Host "=== OVAV × WARP 2026 — Workflow Creator ==="
Write-Host "Manifest: $ManifestPath"
Write-Host "Workflows: $($manifest.workflows.Count)"
Write-Host ""

# Warp Drive workflow definitions live in:
#   %LOCALAPPDATA%\warp\Warp\dev\workflows\*.json
# Each workflow file is a JSON document describing inputs, command, etc.
# Warp reads this directory on startup.

$warpDrivePath = Join-Path $env:LOCALAPPDATA "warp\Warp\dev\workflows"
if (-not (Test-Path $warpDrivePath)) {
    New-Item -ItemType Directory -Force -Path $warpDrivePath | Out-Null
    Write-Host "✓ Created Warp Drive path: $warpDrivePath"
}

$created = 0
$failed = 0

foreach ($wf in $manifest.workflows) {
    $outPath = Join-Path $warpDrivePath "$($wf.id).json"

    $workflowDef = @{
        id = $wf.id
        name = $wf.name
        description = $wf.description
        command = $wf.command
        inputs = $wf.inputs
        working_directory = $wf.working_directory
        shell = $wf.shell
        category = $wf.category
        autosync = $true
    }

    # Add optional fields if present
    if ($wf.long_running) { $workflowDef.long_running = $true }
    if ($wf.destructive) { $workflowDef.destructive = $true }
    if ($wf.warning) { $workflowDef.warning = $wf.warning }

    try {
        $workflowDef | ConvertTo-Json -Depth 10 | Set-Content -Path $outPath -Encoding UTF8
        Write-Host "✓ Created: $($wf.name)"
        $created++
    } catch {
        Write-Host "✗ Failed: $($wf.name) — $_"
        $failed++
    }
}

Write-Host ""
Write-Host "=== Summary ==="
Write-Host "Created: $created"
Write-Host "Failed:  $failed"
Write-Host ""
Write-Host "Restart Warp to load the workflows. Then run 'ovav-workflows' from ⊞+Shift+R or Ctrl+Shift+R."
