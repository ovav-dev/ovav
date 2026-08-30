[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Install', 'Start', 'Status', 'Stop', 'Uninstall', 'Recover', 'TestCase', 'TestMutex')]
    [string]$Mode,

    [string]$FixtureJson,
    [int]$TestHoldMilliseconds = 0
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$ManagerVersion = '1.1.0'
$StateSchema = 'ovav.windows-mcp-state.v2'
$EnvelopeSchema = 'ovav.windows-mcp-envelope.v1'
$TaskName = 'OVAV-MCP-2B7F6B1D'
$MutexName = if ($IsWindows) { 'Local\OVAV.MCP.Windows.2B7F6B1D' } else { 'OVAV.MCP.Windows.2B7F6B1D' }
$PlaywrightPort = 8931
$MemoryPort = 8932
$SessionTimeoutMilliseconds = 300000
$LocalAppDataRoot = if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { '' } else { [IO.Path]::GetFullPath($env:LOCALAPPDATA).TrimEnd('\') }
$ManagedRoot = if ($LocalAppDataRoot) { Join-Path $LocalAppDataRoot 'OVAV\mcp' } else { 'C:\OVAV-Test\mcp' }
$PackageRoot = Join-Path $ManagedRoot 'packages'
$DataRoot = Join-Path $ManagedRoot 'data'
$LogRoot = Join-Path $ManagedRoot 'logs'
$RunRoot = Join-Path $ManagedRoot 'run'
$ManagerRoot = Join-Path $ManagedRoot 'manager'
$StableManager = Join-Path $ManagerRoot "$ManagerVersion\ovav-mcp-windows.ps1"
$StableBundleRoot = Join-Path (Split-Path -Parent $StableManager) 'windows-bundle'
$StatePath = Join-Path $ManagedRoot 'state.json'
$RecoveryRoot = Join-Path $ManagedRoot 'recovery'
$MemoryPath = Join-Path $DataRoot 'memory.jsonl'
$ChromeUserDataRoot = Join-Path $DataRoot 'playwright-chrome'
$SourceBundleRoot = Join-Path (Split-Path -Parent $PSCommandPath) 'windows-bundle'
$script:MutationMutex = $null

function Get-Sha256Hex {
    param([Parameter(Mandatory = $true)][string]$Value)

    $bytes = [Text.UTF8Encoding]::new($false).GetBytes($Value)
    return [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($bytes)).ToLowerInvariant()
}

function Assert-IdentityValues {
    param(
        [Parameter(Mandatory = $true)]$Actual,
        [Parameter(Mandatory = $true)]$Expected
    )

    foreach ($property in @('Pid', 'ParentPid')) {
        if ([int]$Actual.$property -ne [int]$Expected.$property) {
            throw "Process $property identity mismatch"
        }
    }
    if ([Int64]$Actual.StartTimeTicks -ne [Int64]$Expected.StartTimeTicks) {
        throw 'Process start-time identity mismatch'
    }
    if (-not [string]::Equals([string]$Actual.Executable, [string]$Expected.Executable, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Process executable identity mismatch'
    }
    if (-not [string]::Equals([string]$Actual.CommandLine, [string]$Expected.CommandLine, [StringComparison]::Ordinal)) {
        throw 'Process command-line identity mismatch'
    }
}

function Assert-PathChainRecords {
    param([Parameter(Mandatory = $true)][array]$Chain)

    foreach ($record in $Chain) {
        if ([bool]$record.reparse) {
            throw "Refusing reparse-point path: $($record.path)"
        }
    }
}

function Assert-TaskOwnershipRecord {
    param(
        [Parameter(Mandatory = $true)]$Actual,
        [Parameter(Mandatory = $true)]$Expected
    )

    foreach ($property in @('execute', 'arguments', 'userId', 'managerPath', 'workingDirectory')) {
        $comparison = if ($property -eq 'arguments') { [StringComparison]::Ordinal } else { [StringComparison]::OrdinalIgnoreCase }
        if (-not [string]::Equals([string]$Actual.$property, [string]$Expected.$property, $comparison)) {
            throw "Scheduled task is not OVAV-owned: $property mismatch"
        }
    }
}

function Assert-StateChecksum {
    param(
        [Parameter(Mandatory = $true)][string]$StateJson,
        [Parameter(Mandatory = $true)][string]$Checksum
    )

    $actual = Get-Sha256Hex -Value $StateJson
    if (-not [string]::Equals($actual, $Checksum, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'State checksum mismatch'
    }
}

function Get-RecoveryDecision {
    param([Parameter(Mandatory = $true)][string]$Phase)

    if ($Phase -in @('intent', 'starting', 'stopping', 'recovery-required')) {
        return 'recover-recorded-only'
    }
    if ($Phase -eq 'running') {
        return 'validate-running'
    }
    throw "Unknown state phase: $Phase"
}

function Select-RecordedStopTargets {
    param(
        [Parameter(Mandatory = $true)][array]$Recorded,
        [Parameter(Mandatory = $true)][int[]]$LivePids
    )

    $ordered = @($Recorded)
    [array]::Reverse($ordered)
    return @($ordered | Where-Object { $LivePids -contains [int]$_.pid })
}

function Assert-ListenerOwned {
    param(
        [Parameter(Mandatory = $true)][int]$ListenerPid,
        [Parameter(Mandatory = $true)][int[]]$TrackedPids
    )

    if ($TrackedPids -notcontains $ListenerPid) {
        throw "Listener PID $ListenerPid is not in the tracked service tree"
    }
}

function Get-UninstallTargets {
    return @($PackageRoot, $RunRoot, $LogRoot, $ManagerRoot, $StatePath)
}

function Invoke-TestCase {
    param([Parameter(Mandatory = $true)][string]$Json)

    $fixture = $Json | ConvertFrom-Json
    $succeeded = $true
    try {
        switch ([string]$fixture.case) {
            'task-ownership' { Assert-TaskOwnershipRecord -Actual $fixture.actual -Expected $fixture.expected }
            'path-chain' { Assert-PathChainRecords -Chain @($fixture.chain) }
            'identity' { Assert-IdentityValues -Actual $fixture.actual -Expected $fixture.expected }
            'state-envelope' { Assert-StateChecksum -StateJson ([string]$fixture.stateJson) -Checksum ([string]$fixture.checksum) }
            'recovery-decision' {
                $decision = Get-RecoveryDecision -Phase ([string]$fixture.phase)
                if ($decision -ne [string]$fixture.expectedDecision) { throw "Decision mismatch: $decision" }
            }
            'recorded-stop-targets' {
                $targets = @(Select-RecordedStopTargets -Recorded @($fixture.recorded) -LivePids @($fixture.livePids))
                $actualPids = @($targets | ForEach-Object { [int]$_.pid })
                if (($actualPids -join ',') -ne (@($fixture.expectedPids) -join ',')) { throw 'Recorded stop target mismatch' }
            }
            'listener-owner' { Assert-ListenerOwned -ListenerPid ([int]$fixture.listenerPid) -TrackedPids @($fixture.trackedPids) }
            'uninstall-targets' {
                $targets = @(Get-UninstallTargets)
                foreach ($suffix in @($fixture.forbiddenSuffixes)) {
                    if ($targets | Where-Object { $_.EndsWith([string]$suffix, [StringComparison]::OrdinalIgnoreCase) }) {
                        throw "Uninstall target would delete preserved data: $suffix"
                    }
                }
            }
            default { throw "Unknown fixture case: $($fixture.case)" }
        }
    }
    catch {
        $succeeded = $false
    }
    if ($succeeded -ne [bool]$fixture.expectSuccess) {
        throw "Fixture $($fixture.name) expected success=$($fixture.expectSuccess), got $succeeded"
    }
    Write-Output "PASS: $($fixture.name)"
}

function Enter-MutationMutex {
    $createdNew = $false
    $script:MutationMutex = [Threading.Mutex]::new($false, $MutexName, [ref]$createdNew)
    $acquired = $false
    try {
        $acquired = $script:MutationMutex.WaitOne(0)
    }
    catch [Threading.AbandonedMutexException] {
        $acquired = $true
    }
    if (-not $acquired) {
        $script:MutationMutex.Dispose()
        $script:MutationMutex = $null
        throw 'Another OVAV MCP lifecycle mutation is already running'
    }
}

function Exit-MutationMutex {
    if ($null -ne $script:MutationMutex) {
        $script:MutationMutex.ReleaseMutex()
        $script:MutationMutex.Dispose()
        $script:MutationMutex = $null
    }
}

function Get-PathChain {
    param([Parameter(Mandatory = $true)][string]$Path)

    $full = [IO.Path]::GetFullPath($Path).TrimEnd('\')
    $parts = [Collections.Generic.List[string]]::new()
    $cursor = $full
    while (-not [string]::IsNullOrWhiteSpace($cursor)) {
        $parts.Add($cursor)
        $parent = [IO.Path]::GetDirectoryName($cursor)
        if ([string]::IsNullOrWhiteSpace($parent) -or [string]::Equals($parent.TrimEnd('\'), $cursor, [StringComparison]::OrdinalIgnoreCase)) { break }
        $cursor = $parent.TrimEnd('\')
    }
    $result = @($parts)
    [array]::Reverse($result)
    return $result
}

function Assert-NoReparsePath {
    param([Parameter(Mandatory = $true)][string]$Path)

    $records = [Collections.Generic.List[object]]::new()
    foreach ($candidate in @(Get-PathChain -Path $Path)) {
        if (Test-Path -LiteralPath $candidate) {
            $item = Get-Item -LiteralPath $candidate -Force
            $records.Add([pscustomobject]@{ path = $candidate; reparse = (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) })
        }
    }
    Assert-PathChainRecords -Chain @($records)
}

function Assert-ManagedPath {
    param([Parameter(Mandatory = $true)][string]$Path)

    if ([string]::IsNullOrWhiteSpace($LocalAppDataRoot)) { throw 'LOCALAPPDATA is required' }
    $root = [IO.Path]::GetFullPath($ManagedRoot).TrimEnd('\')
    $full = [IO.Path]::GetFullPath($Path).TrimEnd('\')
    if (-not [string]::Equals($full, $root, [StringComparison]::OrdinalIgnoreCase) -and
        -not $full.StartsWith($root + '\', [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing operation outside managed root: $full"
    }
    Assert-NoReparsePath -Path $LocalAppDataRoot
    Assert-NoReparsePath -Path $full
}

function New-PrivateDirectory {
    param([Parameter(Mandatory = $true)][string]$Path)

    Assert-ManagedPath -Path $Path
    $isNew = -not (Test-Path -LiteralPath $Path)
    if ($isNew) {
        [void](New-Item -ItemType Directory -Path $Path)
    }
    Assert-ManagedPath -Path $Path
    if ($isNew) {
        $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User
        $acl = [Security.AccessControl.DirectorySecurity]::new()
        $acl.SetOwner($sid)
        $acl.SetAccessRuleProtection($true, $false)
        $rights = [Security.AccessControl.FileSystemRights]::FullControl
        $inheritance = [Security.AccessControl.InheritanceFlags]'ContainerInherit, ObjectInherit'
        $rule = [Security.AccessControl.FileSystemAccessRule]::new($sid, $rights, $inheritance, [Security.AccessControl.PropagationFlags]::None, [Security.AccessControl.AccessControlType]::Allow)
        $acl.AddAccessRule($rule)
        Set-Acl -LiteralPath $Path -AclObject $acl
        Assert-ManagedPath -Path $Path
    }
}

function Initialize-ManagedDirectories {
    Assert-NoReparsePath -Path $LocalAppDataRoot
    $ovavRoot = Join-Path $LocalAppDataRoot 'OVAV'
    Assert-NoReparsePath -Path $ovavRoot
    if (-not (Test-Path -LiteralPath $ovavRoot)) {
        [void](New-Item -ItemType Directory -Path $ovavRoot)
    }
    foreach ($path in @($ManagedRoot, $PackageRoot, $DataRoot,
        $LogRoot, $RunRoot, $ManagerRoot, (Split-Path -Parent $StableManager),
        $StableBundleRoot, $RecoveryRoot, $ChromeUserDataRoot
    )) {
        New-PrivateDirectory -Path $path
    }
}

function Remove-SafeLeaf {
    param([Parameter(Mandatory = $true)][string]$Path)

    Assert-ManagedPath -Path $Path
    if (-not (Test-Path -LiteralPath $Path)) { return }
    $item = Get-Item -LiteralPath $Path -Force
    if (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw "Refusing deletion of reparse point: $Path"
    }
    Assert-ManagedPath -Path $Path
    Remove-Item -LiteralPath $Path -Force
}

function Remove-OwnedTreeSafely {
    param([Parameter(Mandatory = $true)][string]$Path)

    Assert-ManagedPath -Path $Path
    if (-not (Test-Path -LiteralPath $Path)) { return }
    if ([string]::Equals([IO.Path]::GetFullPath($Path).TrimEnd('\'), [IO.Path]::GetFullPath($DataRoot).TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Memory data root is never an uninstall target'
    }
    foreach ($child in @(Get-ChildItem -LiteralPath $Path -Force)) {
        Assert-ManagedPath -Path $child.FullName
        if (($child.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
            throw "Refusing deletion through reparse point: $($child.FullName)"
        }
        if ($child.PSIsContainer) { Remove-OwnedTreeSafely -Path $child.FullName } else { Remove-SafeLeaf -Path $child.FullName }
    }
    Remove-SafeLeaf -Path $Path
}

function Write-DurableBytes {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][byte[]]$Bytes
    )

    Assert-ManagedPath -Path $Path
    $stream = [IO.FileStream]::new($Path, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        $stream.Write($Bytes, 0, $Bytes.Length)
        $stream.Flush($true)
    }
    finally { $stream.Dispose() }
}

function Copy-DurableFile {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination
    )

    Assert-ManagedPath -Path $Destination
    $input = [IO.FileStream]::new($Source, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::Read)
    $output = [IO.FileStream]::new($Destination, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        $input.CopyTo($output)
        $output.Flush($true)
    }
    finally {
        $output.Dispose()
        $input.Dispose()
    }
}

function Write-AtomicState {
    param([Parameter(Mandatory = $true)]$State)

    Assert-ManagedPath -Path $StatePath
    $State.updatedAt = [DateTimeOffset]::UtcNow.ToString('o')
    $stateJson = $State | ConvertTo-Json -Depth 99 -Compress
    $checksum = Get-Sha256Hex -Value $stateJson
    # Store raw JSON bytes as base64 so re-serialization never diverges from checksum
    $stateBytes = [Text.UTF8Encoding]::new($false).GetBytes($stateJson)
    $envelope = [ordered]@{
        schema = $EnvelopeSchema
        checksumAlgorithm = 'sha256'
        checksum = $checksum
        stateBytes = [Convert]::ToBase64String($stateBytes)
    }
    $envelopeJson = $envelope | ConvertTo-Json -Depth 3 -Compress
    $bytes = [Text.UTF8Encoding]::new($false).GetBytes($envelopeJson)
    $temp = "$StatePath.$PID.$([Guid]::NewGuid().ToString('N')).tmp"
    Write-DurableBytes -Path $temp -Bytes $bytes
    Assert-ManagedPath -Path $StatePath
    [IO.File]::Move($temp, $StatePath, $true)
}

function Read-State {
    if (-not (Test-Path -LiteralPath $StatePath -PathType Leaf)) { return $null }
    Assert-ManagedPath -Path $StatePath
    $lastError = $null
    for ($attempt = 0; $attempt -lt 5; $attempt++) {
        try {
            $rawJson = Get-Content -LiteralPath $StatePath -Raw
            $envelope = $rawJson | ConvertFrom-Json
            if ([string]$envelope.schema -ne $EnvelopeSchema -or [string]$envelope.checksumAlgorithm -ne 'sha256') {
                throw 'Unsupported state envelope schema'
            }
            # Decode base64 to raw JSON bytes — avoids re-serialization divergence
            $stateBytes = [Convert]::FromBase64String([string]$envelope.stateBytes)
            $stateJson = [Text.UTF8Encoding]::new($false).GetString($stateBytes)
            Assert-StateChecksum -StateJson $stateJson -Checksum ([string]$envelope.checksum)
            $stateObj = $stateJson | ConvertFrom-Json
            [void](Get-RecoveryDecision -Phase ([string]$stateObj.phase))
            return $stateObj
        }
        catch [IO.IOException] {
            $lastError = $_
            Start-Sleep -Milliseconds 20
        }
    }
    if ($null -ne $lastError) { throw $lastError }
    throw 'State is corrupt or incomplete'
}

function Resolve-NodeToolchain {
    $wingetRoot = Join-Path $LocalAppDataRoot 'Microsoft\WinGet\Packages'
    Assert-NoReparsePath -Path $wingetRoot
    $packageDirs = @(Get-ChildItem -LiteralPath $wingetRoot -Directory -Filter 'OpenJS.NodeJS.22_*' -Force)
    if ($packageDirs.Count -ne 1) { throw "Expected exactly one OpenJS.NodeJS.22 winget package, found $($packageDirs.Count)" }
    $packageDir = $packageDirs[0].FullName
    Assert-NoReparsePath -Path $packageDir
    $nodeCandidates = @(Get-ChildItem -LiteralPath $packageDir -File -Filter 'node.exe' -Recurse -Force)
    $npmCandidates = @(Get-ChildItem -LiteralPath $packageDir -File -Filter 'npm-cli.js' -Recurse -Force |
        Where-Object { $_.FullName.EndsWith('\node_modules\npm\bin\npm-cli.js', [StringComparison]::OrdinalIgnoreCase) })
    if ($nodeCandidates.Count -ne 1 -or $npmCandidates.Count -ne 1) {
        throw "Ambiguous Node 22 toolchain: node=$($nodeCandidates.Count), npm=$($npmCandidates.Count)"
    }
    Assert-NoReparsePath -Path $nodeCandidates[0].FullName
    Assert-NoReparsePath -Path $npmCandidates[0].FullName
    $version = (& $nodeCandidates[0].FullName --version).Trim()
    if ($version -notmatch '^v22\.') { throw "Resolved winget Node is not major 22: $version" }
    return [pscustomobject]@{ Node = $nodeCandidates[0].FullName; Npm = $npmCandidates[0].FullName; Version = $version }
}

function Assert-SourceBundleReady {
    Assert-NoReparsePath -Path $PSCommandPath
    Assert-NoReparsePath -Path $SourceBundleRoot
    $packageJson = Join-Path $SourceBundleRoot 'package.json'
    $lockPath = Join-Path $SourceBundleRoot 'package-lock.json'
    if (-not (Test-Path -LiteralPath $packageJson -PathType Leaf)) { throw "Checked-in package manifest missing: $packageJson" }
    if (-not (Test-Path -LiteralPath $lockPath -PathType Leaf)) {
        throw "INSTALL BLOCKED: checked-in integrity-pinned package-lock.json is missing at $lockPath"
    }
    Assert-NoReparsePath -Path $packageJson
    Assert-NoReparsePath -Path $lockPath
    $manifest = Get-Content -LiteralPath $packageJson -Raw | ConvertFrom-Json
    $expected = [ordered]@{
        '@playwright/mcp' = '0.0.79'
        'supergateway' = '3.4.3'
        '@modelcontextprotocol/server-memory' = '2026.7.4'
    }
    foreach ($entry in $expected.GetEnumerator()) {
        if ([string]$manifest.dependencies.($entry.Key) -ne $entry.Value) { throw "Package manifest is not exact for $($entry.Key)" }
    }
    $lock = Get-Content -LiteralPath $lockPath -Raw | ConvertFrom-Json -AsHashtable
    if ([int]$lock.lockfileVersion -lt 2 -or $null -eq $lock.packages) { throw 'package-lock.json is not an npm integrity lock' }
    foreach ($entry in $expected.GetEnumerator()) {
        $locked = $lock.packages["node_modules/$($entry.Key)"]
        if ($null -eq $locked -or [string]$locked.version -ne $entry.Value) {
            throw "package-lock.json is not exact for $($entry.Key)"
        }
    }
    foreach ($property in $lock.packages.GetEnumerator()) {
        if ([string]::IsNullOrWhiteSpace([string]$property.Key)) { continue }
        $package = $property.Value
        if ([string]$package.resolved -like 'https://registry.npmjs.org/*' -and [string]$package.integrity -notmatch '^sha512-') {
            throw "Registry package lacks SHA-512 integrity: $($property.Key)"
        }
    }
    return [pscustomobject]@{ PackageJson = $packageJson; LockPath = $lockPath }
}

function Copy-StableBundle {
    param([Parameter(Mandatory = $true)]$Source)

    Assert-ManagedPath -Path $StableManager
    if (-not [string]::Equals([IO.Path]::GetFullPath($PSCommandPath), [IO.Path]::GetFullPath($StableManager), [StringComparison]::OrdinalIgnoreCase)) {
        Copy-Item -LiteralPath $PSCommandPath -Destination $StableManager -Force
    }
    $stablePackage = Join-Path $StableBundleRoot 'package.json'
    $stableLock = Join-Path $StableBundleRoot 'package-lock.json'
    if (-not [string]::Equals([IO.Path]::GetFullPath($Source.PackageJson), [IO.Path]::GetFullPath($stablePackage), [StringComparison]::OrdinalIgnoreCase)) {
        Copy-Item -LiteralPath $Source.PackageJson -Destination $stablePackage -Force
    }
    if (-not [string]::Equals([IO.Path]::GetFullPath($Source.LockPath), [IO.Path]::GetFullPath($stableLock), [StringComparison]::OrdinalIgnoreCase)) {
        Copy-Item -LiteralPath $Source.LockPath -Destination $stableLock -Force
    }
    foreach ($path in @($StableManager, (Join-Path $StableBundleRoot 'package.json'), (Join-Path $StableBundleRoot 'package-lock.json'))) {
        Assert-ManagedPath -Path $path
    }
}

function Install-Packages {
    param(
        [Parameter(Mandatory = $true)]$Toolchain,
        [Parameter(Mandatory = $true)]$Source
    )

    Copy-Item -LiteralPath $Source.PackageJson -Destination (Join-Path $PackageRoot 'package.json') -Force
    Copy-Item -LiteralPath $Source.LockPath -Destination (Join-Path $PackageRoot 'package-lock.json') -Force
    Push-Location $PackageRoot
    try {
        & $Toolchain.Node $Toolchain.Npm ci --ignore-scripts --no-audit --no-fund
    }
    finally {
        Pop-Location
    }
    if ($LASTEXITCODE -ne 0) { throw "npm ci failed with exit code $LASTEXITCODE" }
    $expected = [ordered]@{
        '@playwright/mcp' = '0.0.79'
        'supergateway' = '3.4.3'
        '@modelcontextprotocol/server-memory' = '2026.7.4'
    }
    foreach ($entry in $expected.GetEnumerator()) {
        $manifest = Join-Path $PackageRoot ("node_modules\" + $entry.Key.Replace('/', '\') + '\package.json')
        Assert-ManagedPath -Path $manifest
        if (-not (Test-Path -LiteralPath $manifest -PathType Leaf) -or
            [string](Get-Content -LiteralPath $manifest -Raw | ConvertFrom-Json).version -ne $entry.Value) {
            throw "Installed package version mismatch: $($entry.Key)"
        }
    }
}

function Quote-CmdValue {
    param([Parameter(Mandatory = $true)][string]$Value)
    if ($Value.Contains('"') -or $Value.Contains("`r") -or $Value.Contains("`n")) { throw 'Unsafe generated runner value' }
    return '"' + $Value + '"'
}

function Write-RunnerFiles {
    param([Parameter(Mandatory = $true)]$Toolchain)

    $playwrightCli = Join-Path $PackageRoot 'node_modules\@playwright\mcp\cli.js'
    $supergatewayCli = Join-Path $PackageRoot 'node_modules\supergateway\dist\index.js'
    $memoryCli = Join-Path $PackageRoot 'node_modules\@modelcontextprotocol\server-memory\dist\index.js'
    foreach ($path in @($playwrightCli, $supergatewayCli, $memoryCli)) {
        Assert-ManagedPath -Path $path
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Required managed entrypoint is missing: $path" }
    }
    $bootstrap = Join-Path $RunRoot 'localhost-supergateway.mjs'
    $bootstrapSource = @"
import net from 'node:net'
import { pathToFileURL } from 'node:url'
if (!process.env.OVAV_SUPERGATEWAY_ENTRY || !process.env.OVAV_MEMORY_COMMAND) throw new Error('OVAV memory gateway environment is incomplete')
const originalListen = net.Server.prototype.listen
net.Server.prototype.listen = function (...args) {
  if (typeof args[0] === 'object' && args[0] !== null) args[0] = { ...args[0], host: '127.0.0.1' }
  else if (typeof args[0] === 'number') args.splice(1, 0, '127.0.0.1')
  return originalListen.apply(this, args)
}
process.argv.push('--stdio', process.env.OVAV_MEMORY_COMMAND, '--outputTransport', 'streamableHttp', '--stateful', '--sessionTimeout', '$SessionTimeoutMilliseconds', '--port', '$MemoryPort', '--baseUrl', 'http://127.0.0.1:$MemoryPort', '--streamableHttpPath', '/mcp')
await import(pathToFileURL(process.env.OVAV_SUPERGATEWAY_ENTRY).href)
"@
    Set-Content -LiteralPath $bootstrap -Value $bootstrapSource -Encoding utf8
    $memoryChild = Join-Path $RunRoot 'memory-server.cmd'
    Set-Content -LiteralPath $memoryChild -Encoding ascii -Value ('@echo off' + "`r`n" + 'set "MEMORY_FILE_PATH=' + $MemoryPath + '"' + "`r`n" + (Quote-CmdValue $Toolchain.Node) + ' ' + (Quote-CmdValue $memoryCli))
    $playwrightRunner = Join-Path $RunRoot 'playwright.cmd'
    $playwrightLog = Join-Path $LogRoot 'playwright.log'
    # NOTE: --isolated mode does not support --user-data-dir or --shared-browser-context.
    # Chrome profile is managed internally by the MCP server in isolated mode.
    # NOTE: --allowed-hosts * is used because the playwright-core HTTP handler has a bug where
    # it compares the raw Host header (with port) against allowed hosts (without port).
    $playwrightArgs = " --host 127.0.0.1 --allowed-hosts * --port $PlaywrightPort --browser chrome --headless --isolated"
    Set-Content -LiteralPath $playwrightRunner -Encoding ascii -Value ('@echo off' + "`r`n" + (Quote-CmdValue $Toolchain.Node) + ' ' + (Quote-CmdValue $playwrightCli) + $playwrightArgs + ' 1>>' + (Quote-CmdValue $playwrightLog) + ' 2>>&1')
    $memoryRunner = Join-Path $RunRoot 'memory.cmd'
    $memoryLog = Join-Path $LogRoot 'memory.log'
    Set-Content -LiteralPath $memoryRunner -Encoding ascii -Value ('@echo off' + "`r`n" + 'set "OVAV_SUPERGATEWAY_ENTRY=' + $supergatewayCli + '"' + "`r`n" + 'set "OVAV_MEMORY_COMMAND=' + $memoryChild + '"' + "`r`n" + (Quote-CmdValue $Toolchain.Node) + ' ' + (Quote-CmdValue $bootstrap) + ' 1>>' + (Quote-CmdValue $memoryLog) + ' 2>>&1')
    return [pscustomobject]@{ playwright = $playwrightRunner; memory = $memoryRunner; playwrightCli = $playwrightCli; memoryCli = $memoryCli; bootstrap = $bootstrap; memoryChild = $memoryChild }
}

function Get-ProcessIdentity {
    param([Parameter(Mandatory = $true)][int]$ProcessId)

    $cim = Get-CimInstance Win32_Process -Filter "ProcessId = $ProcessId" -ErrorAction SilentlyContinue
    if ($null -eq $cim) { return $null }
    $created = if ($cim.CreationDate -is [datetime]) { [datetime]$cim.CreationDate } else { [Management.ManagementDateTimeConverter]::ToDateTime([string]$cim.CreationDate) }
    return [pscustomobject]@{
        Pid = [int]$cim.ProcessId
        ParentPid = [int]$cim.ParentProcessId
        StartTimeTicks = $created.ToUniversalTime().Ticks
        Executable = [string]$cim.ExecutablePath
        CommandLine = [string]$cim.CommandLine
    }
}

function Initialize-CommandLineParser {
    if ('OvavCommandLine' -as [type]) { return }
    Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
public static class OvavCommandLine {
  [DllImport("shell32.dll", SetLastError=true)] static extern IntPtr CommandLineToArgvW([MarshalAs(UnmanagedType.LPWStr)] string commandLine, out int argc);
  [DllImport("kernel32.dll")] static extern IntPtr LocalFree(IntPtr value);
  public static string[] Split(string commandLine) {
    int argc; IntPtr argv = CommandLineToArgvW(commandLine, out argc);
    if (argv == IntPtr.Zero) throw new System.ComponentModel.Win32Exception();
    try { var result = new List<string>(); for (int i=0; i<argc; i++) result.Add(Marshal.PtrToStringUni(Marshal.ReadIntPtr(argv, i * IntPtr.Size))); return result.ToArray(); }
    finally { LocalFree(argv); }
  }
}
'@
}

function Get-CommandTokens {
    param([Parameter(Mandatory = $true)][string]$CommandLine)
    Initialize-CommandLineParser
    return @([OvavCommandLine]::Split($CommandLine))
}

function Assert-TokenSequence {
    param(
        [Parameter(Mandatory = $true)][string[]]$Actual,
        [Parameter(Mandatory = $true)][string[]]$Expected
    )
    if ($Actual.Count -ne $Expected.Count) { throw "Command token count mismatch: $($Actual.Count) != $($Expected.Count)" }
    for ($i = 0; $i -lt $Actual.Count; $i++) {
        if (-not [string]::Equals($Actual[$i], $Expected[$i], [StringComparison]::Ordinal)) { throw "Command token mismatch at $i" }
    }
}

function Test-TokenSequence {
    param(
        [Parameter(Mandatory = $true)][string[]]$Actual,
        [Parameter(Mandatory = $true)][string[]]$Expected
    )
    try { Assert-TokenSequence -Actual $Actual -Expected $Expected; return $true } catch { return $false }
}

function Get-ApprovedChromePaths {
    $paths = [Collections.Generic.List[string]]::new()
    # Only system-scope Chrome installations are approved — no user-profile Chrome
    foreach ($root in @($env:ProgramFiles, ${env:ProgramFiles(x86)})) {
        if ([string]::IsNullOrWhiteSpace($root)) { continue }
        $candidate = Join-Path $root 'Google\Chrome\Application\chrome.exe'
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            Assert-NoReparsePath -Path $candidate
            $paths.Add([IO.Path]::GetFullPath($candidate))
        }
    }
    return @($paths)
}

function Test-ContainsExactUserDataDir {
    param([Parameter(Mandatory = $true)][string[]]$Tokens)
    for ($i = 0; $i -lt $Tokens.Count; $i++) {
        if ([string]::Equals($Tokens[$i], "--user-data-dir=$ChromeUserDataRoot", [StringComparison]::OrdinalIgnoreCase)) { return $true }
        if ([string]::Equals($Tokens[$i], '--user-data-dir', [StringComparison]::OrdinalIgnoreCase) -and $i + 1 -lt $Tokens.Count -and
            [string]::Equals($Tokens[$i + 1], $ChromeUserDataRoot, [StringComparison]::OrdinalIgnoreCase)) { return $true }
    }
    return $false
}

function Assert-ApprovedServiceProcess {
    param(
        [Parameter(Mandatory = $true)][string]$ServiceName,
        [Parameter(Mandatory = $true)]$Identity,
        [Parameter(Mandatory = $true)]$State,
        [Parameter(Mandatory = $true)]$Service
    )

    $tokens = @(Get-CommandTokens -CommandLine ([string]$Identity.CommandLine))
    if ($tokens.Count -eq 0) { throw 'Process has no command tokens' }
    if ([string]::Equals([string]$Identity.Executable, [string]$env:ComSpec, [StringComparison]::OrdinalIgnoreCase)) {
        $cmdTargets = @([string]$Service.runner)
        if ($ServiceName -eq 'memory' -and -not [string]::IsNullOrWhiteSpace([string]$Service.memoryChild)) { $cmdTargets += [string]$Service.memoryChild }
        foreach ($target in $cmdTargets) {
            if (Test-TokenSequence -Actual $tokens -Expected @([string]$env:ComSpec, '/d', '/s', '/c', 'call', $target)) { return }
            if (Test-TokenSequence -Actual $tokens -Expected @([string]$env:ComSpec, '/d', '/s', '/c', $target)) { return }
        }
        throw 'cmd.exe command line is not an exact managed runner'
    }
    if ([string]::Equals([string]$Identity.Executable, [string]$State.nodePath, [StringComparison]::OrdinalIgnoreCase)) {
        if ($ServiceName -eq 'playwright') {
            Assert-TokenSequence -Actual $tokens -Expected @([string]$State.nodePath, [string]$Service.playwrightCli, '--host', '127.0.0.1', '--allowed-hosts', '*', '--port', "$PlaywrightPort", '--browser', 'chrome', '--headless', '--isolated')
            return
        }
        if (Test-TokenSequence -Actual $tokens -Expected @([string]$State.nodePath, [string]$Service.bootstrap)) { return }
        if (Test-TokenSequence -Actual $tokens -Expected @([string]$State.nodePath, [string]$Service.memoryCli)) { return }
        throw 'Memory Node command line is not an exact managed entrypoint'
    }
    $chromePaths = @(Get-ApprovedChromePaths)
    if ($ServiceName -eq 'playwright' -and $chromePaths -contains [string]$Identity.Executable) {
        if (-not (Test-ContainsExactUserDataDir -Tokens $tokens)) { throw 'Chrome command lacks the exact OVAV-owned user-data-dir' }
        return
    }
    throw "Refusing unapproved $ServiceName process PID $($Identity.Pid): $($Identity.Executable)"
}

function Get-TrackedProcessTree {
    param(
        [Parameter(Mandatory = $true)][string]$ServiceName,
        [Parameter(Mandatory = $true)]$State,
        [Parameter(Mandatory = $true)]$Service
    )

    $root = Get-ProcessIdentity -ProcessId ([int]$Service.root.Pid)
    if ($null -eq $root) { throw "$ServiceName root exited" }
    Assert-IdentityValues -Actual $root -Expected $Service.root
    $all = @(Get-CimInstance Win32_Process)
    $result = [Collections.Generic.List[object]]::new()
    $result.Add($root)
    $queue = [Collections.Generic.Queue[int]]::new()
    $queue.Enqueue($root.Pid)
    while ($queue.Count -gt 0) {
        $parent = $queue.Dequeue()
        foreach ($candidate in @($all | Where-Object { [int]$_.ParentProcessId -eq $parent })) {
            $identity = Get-ProcessIdentity -ProcessId ([int]$candidate.ProcessId)
            if ($null -eq $identity) { continue }
            Assert-ApprovedServiceProcess -ServiceName $ServiceName -Identity $identity -State $State -Service $Service
            $result.Add($identity)
            $queue.Enqueue($identity.Pid)
        }
    }
    return @($result)
}

function Start-Runner {
    param([Parameter(Mandatory = $true)][string]$RunnerPath)
    $info = [Diagnostics.ProcessStartInfo]::new()
    $info.FileName = $env:ComSpec
    $info.ArgumentList.Add('/d'); $info.ArgumentList.Add('/s'); $info.ArgumentList.Add('/c'); $info.ArgumentList.Add('call'); $info.ArgumentList.Add($RunnerPath)
    $info.WorkingDirectory = $ManagedRoot
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $info
    if (-not $process.Start()) { throw "Could not start runner: $RunnerPath" }
    $identity = Get-ProcessIdentity -ProcessId $process.Id
    $process.Dispose()
    if ($null -eq $identity) { throw "Runner exited before identity capture: $RunnerPath" }
    return $identity
}

function Get-ListenerIdentity {
    param([Parameter(Mandatory = $true)][int]$Port)
    $listeners = @(Get-NetTCPConnection -State Listen -LocalAddress '127.0.0.1' -LocalPort $Port -ErrorAction SilentlyContinue)
    if ($listeners.Count -ne 1) { throw "Expected exactly one loopback listener for port $Port, found $($listeners.Count)" }
    $identity = Get-ProcessIdentity -ProcessId ([int]$listeners[0].OwningProcess)
    if ($null -eq $identity) { throw "Listener PID vanished for port $Port" }
    return $identity
}

function Test-PortOccupiedAnyAddress {
    param([Parameter(Mandatory = $true)][int]$Port)
    return @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue).Count -ne 0
}

function Convert-McpResponseContent {
    param([Parameter(Mandatory = $true)][string]$Content)
    $trimmed = $Content.Trim()
    if ($trimmed.StartsWith('{')) { return $trimmed | ConvertFrom-Json }
    foreach ($line in $Content -split '\r?\n') {
        if ($line.StartsWith('data:', [StringComparison]::OrdinalIgnoreCase)) {
            $data = $line.Substring(5).Trim()
            if ($data.StartsWith('{')) { return $data | ConvertFrom-Json }
        }
    }
    throw 'MCP response contained no JSON-RPC payload'
}

function Invoke-McpPost {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)]$Payload,
        [string]$SessionId
    )
    $headers = @{ Accept = 'application/json, text/event-stream' }
    if ($SessionId) { $headers['Mcp-Session-Id'] = $SessionId }
    return Invoke-WebRequest -UseBasicParsing -Method Post -Uri $Uri -Headers $headers -ContentType 'application/json' -Body ($Payload | ConvertTo-Json -Depth 10 -Compress) -TimeoutSec 4
}

function Test-McpFunctionalHealth {
    param([Parameter(Mandatory = $true)][int]$Port)
    try {
        $uri = "http://127.0.0.1:$Port/mcp"
        # Use curl.exe for reliable HTTP POST - better SSE handling than Invoke-WebRequest
        $body = (@{ jsonrpc = '2.0'; id = 1; method = 'initialize'; params = @{ protocolVersion = '2025-03-26'; capabilities = @{}; clientInfo = @{ name = 'ovav-manager'; version = $ManagerVersion } } } | ConvertTo-Json -Compress)
        $curlExe = 'curl.exe'
        $process = [Diagnostics.Process]::new()
        $process.StartInfo.FileName = $curlExe
        $process.StartInfo.Arguments = "-s --max-time 5 -X POST `"$uri`" -H `"Content-Type: application/json`" -H `"Accept: application/json, text/event-stream`" -d `"$body`""
        $process.StartInfo.RedirectStandardOutput = $true
        $process.StartInfo.RedirectStandardError = $true
        $process.StartInfo.UseShellExecute = $false
        $process.StartInfo.CreateNoWindow = $true
        $process.StartInfo.StandardOutputEncoding = [Text.UTF8Encoding]::new($false)
        $process.StartInfo.StandardErrorEncoding = [Text.UTF8Encoding]::new($false)
        $process.Start() | Out-Null
        $stdout = $process.StandardOutput.ReadToEnd()
        $process.WaitForExit(6000)
        if ($process.ExitCode -ne 0) { return $false }
        # Parse SSE response: extract JSON from "data: {...}" lines
        $jsonLine = ($stdout -split '\r?\n' | Where-Object { $_.StartsWith('data:') } | Select-Object -First 1)
        if (-not $jsonLine) { return $false }
        $jsonStr = $jsonLine.Substring(5).Trim()
        $payload = $jsonStr | ConvertFrom-Json
        if ($null -eq $payload.result) { return $false }
        return $true
    }
    catch { return $false }
}

function Wait-ServiceReady {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)]$State
    )
    $service = $State.services.$Name
    $port = if ($Name -eq 'playwright') { $PlaywrightPort } else { $MemoryPort }
    $deadline = [DateTime]::UtcNow.AddSeconds(60)
    while ([DateTime]::UtcNow -lt $deadline) {
        try {
            $processes = @(Get-TrackedProcessTree -ServiceName $Name -State $State -Service $service)
            $listener = Get-ListenerIdentity -Port $port
            Assert-ListenerOwned -ListenerPid $listener.Pid -TrackedPids @($processes | ForEach-Object Pid)
            Assert-ApprovedServiceProcess -ServiceName $Name -Identity $listener -State $State -Service $service
            $service.processes = $processes
            $service.listener = $listener
            Write-AtomicState -State $State
            if (Test-McpFunctionalHealth -Port $port) { return }
        }
        catch { }
        Start-Sleep -Milliseconds 250
    }
    throw "$Name did not obtain an owned listener and pass initialize/tools within 60 seconds"
}

function Get-ExpectedTaskRecord {
    $pwsh = Join-Path $PSHOME 'pwsh.exe'
    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    return [pscustomobject]@{
        execute = $pwsh
        arguments = "-NoLogo -NoProfile -NonInteractive -File `"$StableManager`" -Mode Start"
        userId = $sid
        managerPath = $StableManager
        workingDirectory = $ManagedRoot
    }
}

function Resolve-PrincipalSid {
    param([Parameter(Mandatory = $true)][string]$Identity)
    try { return [Security.Principal.SecurityIdentifier]::new($Identity).Value } catch { }
    return ([Security.Principal.NTAccount]::new($Identity).Translate([Security.Principal.SecurityIdentifier])).Value
}

function Assert-OwnedScheduledTask {
    param(
        [Parameter(Mandatory = $true)]$Task,
        [Parameter(Mandatory = $true)]$Expected
    )
    $actions = @($Task.Actions)
    if ($actions.Count -ne 1) { throw "Task $TaskName has unexpected action count" }
    $actual = [pscustomobject]@{
        execute = [string]$actions[0].Execute
        arguments = [string]$actions[0].Arguments
        userId = Resolve-PrincipalSid -Identity ([string]$Task.Principal.UserId)
        managerPath = $StableManager
        workingDirectory = [string]$actions[0].WorkingDirectory
    }
    Assert-TaskOwnershipRecord -Actual $actual -Expected $Expected
}

function Register-OwnedScheduledTask {
    $expected = Get-ExpectedTaskRecord
    if (-not (Test-Path -LiteralPath $expected.execute -PathType Leaf)) { throw 'PowerShell 7 executable was not found beside PSHOME' }
    $existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($null -ne $existing) { Assert-OwnedScheduledTask -Task $existing -Expected $expected }
    $action = New-ScheduledTaskAction -Execute $expected.execute -Argument $expected.arguments -WorkingDirectory $ManagedRoot
    $trigger = New-ScheduledTaskTrigger -AtLogOn -User $expected.userId
    $principal = New-ScheduledTaskPrincipal -UserId $expected.userId -LogonType Interactive -RunLevel Limited
    $settings = New-ScheduledTaskSettingsSet -MultipleInstances IgnoreNew -ExecutionTimeLimit ([TimeSpan]::Zero) -RestartCount 3 -RestartInterval ([TimeSpan]::FromMinutes(1))
    if ($null -eq $existing) {
        Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings | Out-Null
    }
    else {
        Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null
    }
}

function Install-Services {
    $source = Assert-SourceBundleReady
    Initialize-ManagedDirectories
    Copy-StableBundle -Source $source
    $toolchain = Resolve-NodeToolchain
    Install-Packages -Toolchain $toolchain -Source $source
    [void](Write-RunnerFiles -Toolchain $toolchain)
    Register-OwnedScheduledTask
    Write-Output "Installed OVAV MCP manager $ManagerVersion; memory retained at $MemoryPath"
}

function Start-Services {
    Initialize-ManagedDirectories
    $existing = Read-State
    if ($null -ne $existing) {
        if ([string]$existing.phase -ne 'running') { throw "State phase is $($existing.phase); run Recover before Start" }
        foreach ($name in @('playwright', 'memory')) {
            $service = $existing.services.$name
            $processes = @(Get-TrackedProcessTree -ServiceName $name -State $existing -Service $service)
            $listener = Get-ListenerIdentity -Port ([int]$service.port)
            Assert-IdentityValues -Actual $listener -Expected $service.listener
            Assert-ListenerOwned -ListenerPid $listener.Pid -TrackedPids @($processes | ForEach-Object Pid)
            if (-not (Test-McpFunctionalHealth -Port ([int]$service.port))) { throw "$name is owned but MCP-functional health failed" }
        }
        Write-Output 'OVAV MCP singleton services are already healthy and owned'
        return
    }
    $toolchain = Resolve-NodeToolchain
    $runners = Write-RunnerFiles -Toolchain $toolchain
    $state = [ordered]@{ schema = $StateSchema; managerVersion = $ManagerVersion; phase = 'intent'; nodePath = $toolchain.Node; nodeVersion = $toolchain.Version; updatedAt = ''; services = [ordered]@{}; errors = @() }
    Write-AtomicState -State $state
    try {
        foreach ($port in @($PlaywrightPort, $MemoryPort)) {
            if (Test-PortOccupiedAnyAddress -Port $port) {
                throw "Port $port is occupied before launch; refusing squatter"
            }
        }
        $state.phase = 'starting'; Write-AtomicState -State $state
        foreach ($name in @('playwright', 'memory')) {
            $runner = [string]$runners.$name
            $root = Start-Runner -RunnerPath $runner
            $service = [ordered]@{
                runner = $runner
                port = if ($name -eq 'playwright') { $PlaywrightPort } else { $MemoryPort }
                root = $root
                processes = @($root)
                listener = $null
                playwrightCli = if ($name -eq 'playwright') { [string]$runners.playwrightCli } else { '' }
                bootstrap = if ($name -eq 'memory') { [string]$runners.bootstrap } else { '' }
                memoryCli = if ($name -eq 'memory') { [string]$runners.memoryCli } else { '' }
                memoryChild = if ($name -eq 'memory') { [string]$runners.memoryChild } else { '' }
            }
            $state.services[$name] = $service
            Assert-ApprovedServiceProcess -ServiceName $name -Identity $root -State $state -Service $service
            Write-AtomicState -State $state
            Wait-ServiceReady -Name $name -State $state
        }
        $state.phase = 'running'; Write-AtomicState -State $state
    }
    catch {
        $state.phase = 'recovery-required'
        $state.errors = @([string]$_)
        Write-AtomicState -State $state
        throw 'Start failed; durable recovery state retained. Run Recover.'
    }
    Write-Output 'OVAV MCP singleton services started on 127.0.0.1:8931 and 127.0.0.1:8932'
}

function Stop-RecordedProcesses {
    param([Parameter(Mandatory = $true)]$State)
    $failures = [Collections.Generic.List[string]]::new()
    foreach ($name in @('memory', 'playwright')) {
        $service = $State.services.$name
        if ($null -eq $service) { continue }
        $livePids = @($service.processes | ForEach-Object {
            if ($null -ne (Get-ProcessIdentity -ProcessId ([int]$_.Pid))) { [int]$_.Pid }
        })
        $recorded = @(Select-RecordedStopTargets -Recorded @($service.processes) -LivePids $livePids)
        foreach ($expected in $recorded) {
            $actual = Get-ProcessIdentity -ProcessId ([int]$expected.Pid)
            if ($null -eq $actual) { continue }
            try {
                Assert-IdentityValues -Actual $actual -Expected $expected
                Assert-ApprovedServiceProcess -ServiceName $name -Identity $actual -State $State -Service $service
                Stop-Process -Id $actual.Pid -ErrorAction Stop
            }
            catch { $failures.Add("$name PID $($expected.Pid): $($_.Exception.Message)") }
        }
    }
    Start-Sleep -Milliseconds 300
    foreach ($name in @('memory', 'playwright')) {
        $service = $State.services.$name
        if ($null -eq $service) { continue }
        foreach ($expected in @($service.processes)) {
            $actual = Get-ProcessIdentity -ProcessId ([int]$expected.Pid)
            if ($null -eq $actual) { continue }
            try { Assert-IdentityValues -Actual $actual -Expected $expected; $failures.Add("$name PID $($expected.Pid) is still running") } catch { $failures.Add("$name PID $($expected.Pid) identity changed; recovery state retained") }
        }
    }
    return @($failures)
}

function Stop-Services {
    $state = Read-State
    if ($null -eq $state) {
        Write-Output 'OVAV MCP singleton services are already stopped'
        return
    }
    $state.phase = 'stopping'; Write-AtomicState -State $state
    $failures = @(Stop-RecordedProcesses -State $state)
    if ($failures.Count -ne 0) {
        $state.phase = 'recovery-required'; $state.errors = $failures; Write-AtomicState -State $state
        throw "Stop incomplete; recovery state retained: $($failures -join '; ')"
    }
    Remove-SafeLeaf -Path $StatePath
    Write-Output 'OVAV MCP singleton services stopped using durable recorded identities only'
}

function Recover-Services {
    if (-not (Test-Path -LiteralPath $StatePath -PathType Leaf)) { Write-Output 'No recovery state exists'; return }
    $state = $null
    try { $state = Read-State } catch {
        foreach ($port in @($PlaywrightPort, $MemoryPort)) {
            if (Test-PortOccupiedAnyAddress -Port $port) {
                throw "Corrupt state and occupied port ${port}: no process can be safely verified or killed"
            }
        }
        Initialize-ManagedDirectories
        $archive = Join-Path $RecoveryRoot ("corrupt-state-$([DateTime]::UtcNow.ToString('yyyyMMddHHmmssfff')).json")
        Assert-ManagedPath -Path $StatePath; Assert-ManagedPath -Path $archive
        [IO.File]::Move($StatePath, $archive)
        Write-Output "Archived corrupt state without killing processes: $archive"
        return
    }
    $state.phase = 'recovery-required'; Write-AtomicState -State $state
    $failures = @(Stop-RecordedProcesses -State $state)
    if ($failures.Count -ne 0) { $state.errors = $failures; Write-AtomicState -State $state; throw "Recovery incomplete: $($failures -join '; ')" }
    $archive = Join-Path $RecoveryRoot ("recovered-state-$([DateTime]::UtcNow.ToString('yyyyMMddHHmmssfff')).json")
    Assert-ManagedPath -Path $archive
    [IO.File]::Move($StatePath, $archive)
    Write-Output "Recovery stopped verified recorded identities and archived state: $archive"
}

function Get-ServiceStatus {
    $status = [ordered]@{ installed = (Test-Path -LiteralPath $StableManager -PathType Leaf); managerVersion = $ManagerVersion; state = 'stopped'; failClosed = $false; services = [ordered]@{} }
    $state = $null
    try { $state = Read-State } catch { $status.state = 'corrupt'; $status.failClosed = $true; $status.error = $_.Exception.Message }
    if ($null -ne $state) { $status.state = [string]$state.phase }
    foreach ($name in @('playwright', 'memory')) {
        $owned = $false; $healthy = $false; $listenerOwned = $false; $processId = $null
        if ($null -ne $state -and $null -ne $state.services.$name) {
            $service = $state.services.$name
            $actual = Get-ProcessIdentity -ProcessId ([int]$service.root.Pid)
            if ($null -ne $actual) {
                try {
                    Assert-IdentityValues -Actual $actual -Expected $service.root
                    Assert-ApprovedServiceProcess -ServiceName $name -Identity $actual -State $state -Service $service
                    $owned = $true; $processId = $actual.Pid
                    $listener = Get-ListenerIdentity -Port ([int]$service.port)
                    Assert-IdentityValues -Actual $listener -Expected $service.listener
                    $listenerOwned = $true
                    $healthy = Test-McpFunctionalHealth -Port ([int]$service.port)
                }
                catch { $owned = $false; $healthy = $false; $listenerOwned = $false }
            }
        }
        $status.services[$name] = [ordered]@{ owned = $owned; listenerOwned = $listenerOwned; mcpFunctional = $healthy; pid = $processId }
    }
    $status | ConvertTo-Json -Depth 8
}

function Uninstall-Services {
    if (Test-Path -LiteralPath $StatePath -PathType Leaf) { Stop-Services }
    foreach ($port in @($PlaywrightPort, $MemoryPort)) {
        if (Test-PortOccupiedAnyAddress -Port $port) {
            throw "Refusing uninstall while unverified listener occupies port $port"
        }
    }
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($null -ne $task) {
        Assert-OwnedScheduledTask -Task $task -Expected (Get-ExpectedTaskRecord)
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    }
    foreach ($path in @(Get-UninstallTargets)) {
        if ([string]::Equals($path, $StatePath, [StringComparison]::OrdinalIgnoreCase)) { Remove-SafeLeaf -Path $path } else { Remove-OwnedTreeSafely -Path $path }
    }
    Write-Output "Uninstalled OVAV MCP code; preserved all data under $DataRoot"
}

if ($PSVersionTable.PSVersion.Major -lt 7) { throw 'OVAV MCP service management requires PowerShell 7 or newer' }

try {
    if ($Mode -eq 'TestCase') { Invoke-TestCase -Json $FixtureJson; exit 0 }
    if ($Mode -eq 'TestMutex') {
        Enter-MutationMutex
        if ($TestHoldMilliseconds -gt 0) { Start-Sleep -Milliseconds $TestHoldMilliseconds }
        exit 0
    }
    if (-not $IsWindows) { throw 'OVAV MCP lifecycle modes require Windows PowerShell 7' }
    if ([string]::IsNullOrWhiteSpace($LocalAppDataRoot)) { throw 'LOCALAPPDATA is required' }
    if ($Mode -in @('Install', 'Start', 'Stop', 'Uninstall', 'Recover')) { Enter-MutationMutex }
    switch ($Mode) {
        'Install' { Install-Services }
        'Start' { Start-Services }
        'Status' { Get-ServiceStatus }
        'Stop' { Stop-Services }
        'Uninstall' { Uninstall-Services }
        'Recover' { Recover-Services }
    }
}
catch {
    Write-Error $_
    exit 1
}
finally {
    Exit-MutationMutex
}
