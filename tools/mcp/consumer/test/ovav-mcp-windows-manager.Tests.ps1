[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($PSVersionTable.PSVersion.Major -lt 7) {
    throw 'Tests require PowerShell 7 or newer'
}

$manager = Join-Path $PSScriptRoot '..\bin\ovav-mcp-windows.ps1'
$fixturesPath = Join-Path $PSScriptRoot 'fixtures\windows-manager-security-cases.json'
$pwsh = (Get-Process -Id $PID).Path

$tokens = $null
$parseErrors = $null
[void][Management.Automation.Language.Parser]::ParseFile($manager, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count -ne 0) {
    throw "Manager parser errors: $($parseErrors | ForEach-Object Message | Join-String -Separator '; ')"
}

$cases = @(Get-Content -LiteralPath $fixturesPath -Raw | ConvertFrom-Json)
foreach ($fixture in $cases) {
    & $pwsh -NoLogo -NoProfile -NonInteractive -File $manager -Mode TestCase -FixtureJson ($fixture | ConvertTo-Json -Depth 12 -Compress)
    if ($LASTEXITCODE -ne 0) {
        throw "Fixture failed: $($fixture.name)"
    }
}

$holder = Start-Process -FilePath $pwsh -ArgumentList @(
    '-NoLogo', '-NoProfile', '-NonInteractive', '-File', $manager,
    '-Mode', 'TestMutex', '-TestHoldMilliseconds', '3000'
) -PassThru
try {
    Start-Sleep -Milliseconds 700
    & $pwsh -NoLogo -NoProfile -NonInteractive -File $manager -Mode TestMutex -TestHoldMilliseconds 0 2>$null
    if ($LASTEXITCODE -eq 0) {
        throw 'Cross-process mutex accepted a concurrent owner'
    }
}
finally {
    $holder.WaitForExit(5000)
    $holder.Dispose()
}

Write-Output "PASS: $($cases.Count) security fixtures + parser + cross-process mutex"
