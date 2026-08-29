[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Start', 'Stop')]
    [string]$Mode,

    [int]$ChromePid,
    [Int64]$StartTimeTicks,
    [string]$Profile,
    [string]$Token,
    [string]$ChromePath
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$MarkerName = '.ovav-playwright-owner'
$ManagedRoot = Join-Path $env:LOCALAPPDATA 'Temp\ovav-playwright'

function Resolve-ChromePath {
    $candidates = @(
        $env:OVAV_CHROME_PATH,
        (Join-Path $env:PROGRAMFILES 'Google\Chrome\Application\chrome.exe'),
        (Join-Path ${env:PROGRAMFILES(X86)} 'Google\Chrome\Application\chrome.exe'),
        (Join-Path $env:LOCALAPPDATA 'Google\Chrome\Application\chrome.exe')
    )

    foreach ($candidate in $candidates) {
        if (-not [string]::IsNullOrWhiteSpace($candidate) -and (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            $resolved = (Resolve-Path -LiteralPath $candidate).Path
            if ([IO.Path]::GetFileName($resolved) -ieq 'chrome.exe') {
                return $resolved
            }
        }
    }

    throw 'Google Chrome executable was not found in approved locations'
}

function Assert-OwnedProfile {
    param(
        [Parameter(Mandatory = $true)][string]$OwnedProfile,
        [Parameter(Mandatory = $true)][string]$OwnerToken
    )

    $root = [IO.Path]::GetFullPath($ManagedRoot).TrimEnd('\') + '\'
    $fullProfile = [IO.Path]::GetFullPath($OwnedProfile).TrimEnd('\') + '\'
    if (-not $fullProfile.StartsWith($root, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Refusing profile operation outside the OVAV-managed Chrome root'
    }

    $profileItem = Get-Item -LiteralPath $OwnedProfile -Force
    if (($profileItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw 'Refusing profile operation through a reparse point'
    }

    $marker = Join-Path $OwnedProfile $MarkerName
    if (-not (Test-Path -LiteralPath $marker -PathType Leaf)) {
        throw 'Refusing profile operation without ownership marker'
    }
    $markerItem = Get-Item -LiteralPath $marker -Force
    if (($markerItem.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0) {
        throw 'Refusing ownership marker through a reparse point'
    }

    $actualToken = (Get-Content -LiteralPath $marker -Raw).Trim()
    if ($actualToken -cne $OwnerToken) {
        throw 'Refusing profile operation because ownership token does not match'
    }
}

function Remove-OwnedProfile {
    param(
        [Parameter(Mandatory = $true)][string]$OwnedProfile,
        [Parameter(Mandatory = $true)][string]$OwnerToken
    )

    Assert-OwnedProfile -OwnedProfile $OwnedProfile -OwnerToken $OwnerToken
    $lastError = $null
    for ($attempt = 1; $attempt -le 25; $attempt++) {
        try {
            Remove-Item -LiteralPath $OwnedProfile -Recurse -Force
            return
        }
        catch {
            $lastError = $_
            Start-Sleep -Milliseconds 200
        }
    }
    throw "Could not remove the owned Chrome profile after retries: $lastError"
}

function Close-OwnedBrowserViaCdp {
    param(
        [Parameter(Mandatory = $true)][string]$OwnedProfile,
        [Parameter(Mandatory = $true)][string]$OwnerToken
    )

    Assert-OwnedProfile -OwnedProfile $OwnedProfile -OwnerToken $OwnerToken
    $devToolsFile = Join-Path $OwnedProfile 'DevToolsActivePort'
    if (-not (Test-Path -LiteralPath $devToolsFile -PathType Leaf)) {
        return
    }

    $devToolsState = @(Get-Content -LiteralPath $devToolsFile)
    if ($devToolsState.Count -lt 2 -or $devToolsState[0] -notmatch '^\d+$' -or $devToolsState[1] -notmatch '^/devtools/browser/[A-Za-z0-9-]+$') {
        throw 'Refusing CDP cleanup because DevToolsActivePort is malformed'
    }

    $port = [int]$devToolsState[0]
    if ($port -le 0 -or $port -gt 65535) {
        throw 'Refusing CDP cleanup because the port is outside the valid range'
    }

    $socket = [System.Net.WebSockets.ClientWebSocket]::new()
    $timeout = [Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(3))
    try {
        $uri = [Uri]::new("ws://127.0.0.1:$port$($devToolsState[1])")
        [void]$socket.ConnectAsync($uri, $timeout.Token).GetAwaiter().GetResult()
        $payload = [Text.Encoding]::UTF8.GetBytes('{"id":1,"method":"Browser.close"}')
        $segment = [ArraySegment[byte]]::new($payload)
        [void]$socket.SendAsync($segment, [Net.WebSockets.WebSocketMessageType]::Text, $true, $timeout.Token).GetAwaiter().GetResult()
    }
    catch {
        # The browser may already be gone. Exact marker validation still makes
        # the subsequent profile cleanup safe; file locks remain the final gate.
        return
    }
    finally {
        $timeout.Dispose()
        $socket.Dispose()
    }
}

function Get-OwnedChromeProcess {
    param(
        [Parameter(Mandatory = $true)][int]$OwnedPid,
        [Parameter(Mandatory = $true)][Int64]$OwnedStartTimeTicks,
        [Parameter(Mandatory = $true)][string]$OwnedProfile,
        [Parameter(Mandatory = $true)][string]$OwnedChromePath
    )

    $process = Get-Process -Id $OwnedPid -ErrorAction SilentlyContinue
    if ($null -eq $process) {
        return $null
    }

    $process.Refresh()
    if ($process.StartTime.ToUniversalTime().Ticks -ne $OwnedStartTimeTicks) {
        throw 'Refusing to use a reused or mismatched PID'
    }
    if ($process.Path -ine $OwnedChromePath) {
        throw 'Refusing to use a process with a different executable path'
    }

    $escapedPid = [string]$OwnedPid
    $cimProcess = Get-CimInstance Win32_Process -Filter "ProcessId = $escapedPid"
    $expectedProfileArgument = "--user-data-dir=`"$OwnedProfile`""
    if ($null -eq $cimProcess -or [string]::IsNullOrWhiteSpace($cimProcess.CommandLine) -or -not $cimProcess.CommandLine.Contains($expectedProfileArgument)) {
        throw 'Refusing to use Chrome because its command line does not own the exact profile'
    }

    return $process
}

function Start-IsolatedChrome {
    $resolvedChrome = Resolve-ChromePath
    $ownerToken = [Guid]::NewGuid().ToString('D')
    $ownedProfile = Join-Path $ManagedRoot $ownerToken
    $process = $null
    $startedPid = 0
    $startedTicks = 0

    New-Item -ItemType Directory -Path $ownedProfile -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $ownedProfile $MarkerName) -Value $ownerToken -NoNewline

    try {
        $arguments = @(
            '--headless=new',
            '--disable-gpu',
            '--no-first-run',
            '--no-default-browser-check',
            '--remote-debugging-address=127.0.0.1',
            '--remote-debugging-port=0',
            "--user-data-dir=`"$ownedProfile`"",
            'about:blank'
        )

        $process = Start-Process -FilePath $resolvedChrome -ArgumentList $arguments -PassThru
        $startedPid = $process.Id
        $startedTicks = $process.StartTime.ToUniversalTime().Ticks
        Start-Sleep -Milliseconds 100
        $process.Refresh()
        if ($process.HasExited) {
            throw "Chrome exited during startup with code $($process.ExitCode)"
        }

        Write-Output "PID=$($process.Id)"
        Write-Output "START_TICKS=$($process.StartTime.ToUniversalTime().Ticks)"
        Write-Output "PROFILE=$ownedProfile"
        Write-Output "TOKEN=$ownerToken"
        Write-Output "CHROME_PATH=$resolvedChrome"
    }
    catch {
        if ($startedPid -gt 0 -and $startedTicks -gt 0) {
            $ownedProcess = Get-OwnedChromeProcess -OwnedPid $startedPid -OwnedStartTimeTicks $startedTicks -OwnedProfile $ownedProfile -OwnedChromePath $resolvedChrome
            if ($null -ne $ownedProcess) {
                Close-OwnedBrowserViaCdp -OwnedProfile $ownedProfile -OwnerToken $ownerToken
                Start-Sleep -Milliseconds 300
                $ownedProcess = Get-OwnedChromeProcess -OwnedPid $startedPid -OwnedStartTimeTicks $startedTicks -OwnedProfile $ownedProfile -OwnedChromePath $resolvedChrome
                if ($null -ne $ownedProcess) {
                    Stop-Process -Id $startedPid -ErrorAction SilentlyContinue
                }
            }
        }
        if (Test-Path -LiteralPath $ownedProfile) {
            Remove-OwnedProfile -OwnedProfile $ownedProfile -OwnerToken $ownerToken
        }
        throw
    }
}

function Stop-IsolatedChrome {
    if ([string]::IsNullOrWhiteSpace($Profile) -or [string]::IsNullOrWhiteSpace($Token)) {
        throw 'Chrome profile and ownership token are required'
    }

    if ($ChromePid -le 0 -or $StartTimeTicks -le 0 -or [string]::IsNullOrWhiteSpace($ChromePath)) {
        throw 'A positive Chrome PID, start time, and executable path are required'
    }

    Assert-OwnedProfile -OwnedProfile $Profile -OwnerToken $Token
    $process = Get-OwnedChromeProcess -OwnedPid $ChromePid -OwnedStartTimeTicks $StartTimeTicks -OwnedProfile $Profile -OwnedChromePath $ChromePath
    if ($null -eq $process) {
        Remove-OwnedProfile -OwnedProfile $Profile -OwnerToken $Token
        return
    }

    Close-OwnedBrowserViaCdp -OwnedProfile $Profile -OwnerToken $Token
    Start-Sleep -Milliseconds 300
    $process = Get-OwnedChromeProcess -OwnedPid $ChromePid -OwnedStartTimeTicks $StartTimeTicks -OwnedProfile $Profile -OwnedChromePath $ChromePath
    if ($null -ne $process) {
        Stop-Process -Id $ChromePid -ErrorAction Stop
    }
    Wait-Process -Id $ChromePid -Timeout 5 -ErrorAction SilentlyContinue
    Remove-OwnedProfile -OwnedProfile $Profile -OwnerToken $Token
}

if ($Mode -eq 'Start') {
    Start-IsolatedChrome
}
else {
    Stop-IsolatedChrome
}
