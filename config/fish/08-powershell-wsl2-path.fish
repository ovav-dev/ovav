# OVAV — PowerShell PATH for WSL2 interop.
#
# OpenCode TUI (1.18+) reads clipboard images by shelling out to bare
# `powershell.exe` (verified in opencode.bin source — zw() function).
# WSL2 does NOT put /mnt/c/Windows/... in the default PATH inside tmux
# sessions, so the call fails silently and image paste never reaches the
# prompt. Same class of bug as commit 6ddad6d (win32yank.exe in WSL PATH).
#
# This file prepends PowerShell's directory to PATH on WSL2 hosts so the
# bare name resolves.
#
# Idempotent: only prepends when powershell.exe is actually present and
# when the directory is not already in PATH.
#
# Deploy target: ~/.config/fish/conf.d/08-powershell-wsl2-path.fish

if grep -qi 'microsoft\|wsl' /proc/version 2>/dev/null
    set -l ps_dir /mnt/c/Windows/System32/WindowsPowerShell/v1.0
    if test -x "$ps_dir/powershell.exe"
        if not string match -q "$ps_dir*" "$PATH"
            set -gx PATH "$ps_dir" $PATH
        end
    end
end
