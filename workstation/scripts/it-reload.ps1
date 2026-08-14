# Intelligent Terminal (IT) v0.1.4 Reload

# This PowerShell script reloads IT settings without restarting IT.
# Per ADR-010.

# Method 1: Send WM_SETTINGCHANGE broadcast to all top-level windows
# This tells all apps that system settings (including IT's settings.json
# via the AppData path) have changed.

Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

public class Win32 {
    [DllImport("user32.dll", SetLastError = true)]
    public static extern IntPtr SendMessageTimeout(
        IntPtr hWnd,
        uint Msg,
        IntPtr wParam,
        IntPtr lParam,
        uint fuFlags,
        uint uTimeout,
        out IntPtr lpdwResult);

    public const uint HWND_BROADCAST = 0xffff;
    public const uint WM_SETTINGCHANGE = 0x001A;
    public const uint SMTO_ABORTIFHUNG = 0x0002;
}
"@

# Send the broadcast
$lpdwResult = [IntPtr]::Zero
$result = [Win32]::SendMessageTimeout(
    [Win32]::HWND_BROADCAST,
    [Win32]::WM_SETTINGCHANGE,
    [IntPtr]::Zero,
    [IntPtr]::Zero,
    [Win32]::SMTO_ABORTIFHUNG,
    5000,
    [ref]$lpdwResult
)

if ($result -eq [IntPtr]::Zero) {
    Write-Host "FAIL: WM_SETTINGCHANGE broadcast failed"
    exit 2
}

Write-Host "OK: WM_SETTINGCHANGE broadcast sent (result=$result)"
exit 0