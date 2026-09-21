@echo off
REM ============================================================
REM OVAV × Warp 2026 - YOLO Setup Launcher
REM Run this from PowerShell: .\setup-warp-yolo.bat
REM ============================================================

setlocal

REM Copy PS1 from WSL to %TEMP%
set "PS1_SRC=\\wsl$\Ubuntu\home\braka\Systems\ovav\.ovav\plans\setup-warp-yolo.ps1"
set "PS1_DST=%TEMP%\setup-warp-yolo.ps1"

echo.
echo === OVAV x Warp 2026 - YOLO Setup ===
echo.
echo [1/3] Copying script to %PS1_DST% ...
copy /Y "%PS1_SRC%" "%PS1_DST%" >nul
if errorlevel 1 (
    echo ERROR: Cannot copy from WSL. Check WSL is running.
    echo   Run: wsl --list --verbose
    exit /b 1
)

echo [2/3] Verifying PowerShell ...
where powershell >nul 2>&1
if errorlevel 1 (
    echo ERROR: PowerShell not in PATH
    exit /b 1
)

echo [3/3] Executing PowerShell script ...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%PS1_DST%"

endlocal
