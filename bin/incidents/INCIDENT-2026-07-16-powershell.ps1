# Pegar en PowerShell nativo Admin (o: powershell -ExecutionPolicy Bypass -File .\this.ps1)
Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -EA 0 | %{ $_.PSObject.Properties | ? { $_.Value -match 'ahk|autohot|window-move' } } | %{ Remove-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name $_.Name -Force -EA 0 }
Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Run' -EA 0 | %{ $_.PSObject.Properties | ? { $_.Value -match 'ahk|autohot|window-move' } } | %{ Remove-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Run' -Name $_.Name -Force -EA 0 }
Get-ScheduledTask | ? { $_.TaskName -match 'ahk|autohot|OVAV' } | Unregister-ScheduledTask -Confirm:$false -EA 0
Get-Process AutoHotkey* -EA 0 | Stop-Process -Force
Write-Host "OK Windows cleanup complete" -ForegroundColor Green
