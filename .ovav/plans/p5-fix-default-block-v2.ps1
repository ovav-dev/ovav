# Fix default block — replace base_model and other build-mode settings
$content = Get-Content -Path "$env:LOCALAPPDATA\warp\Warp\config\settings.toml" -Raw

$lines = $content -split "`n"
$out = @()
$inDefault = $false
$skipUntilNext = $false
$replacementWritten = $false

foreach ($line in $lines) {
    if ($line -match '^\[agents\.execution_profiles\.default\]') {
        $inDefault = $true
        $skipUntilNext = $true
        $out += '[agents.execution_profiles.default]'
        $out += 'name = "OVAV Build (default)"'
        $out += 'apply_code_diffs = "always_allow"'
        $out += 'ask_user_question = "always_ask"'
        $out += 'autosync_plans_to_warp_drive = true'
        $out += 'base_model = "MiniMax-M3"'
        $out += 'command_allowlist = []'
        $out += 'command_denylist = ['
        $out += "  'bash(\s.*)?',"
        $out += "  'fish(\s.*)?',"
        $out += "  'pwsh(\s.*)?',"
        $out += "  'sh(\s.*)?',"
        $out += "  'zsh(\s.*)?',"
        $out += "  'curl(\s.*)?',"
        $out += "  'wget(\s.*)?',"
        $out += "  'ssh(\s.*)?',"
        $out += "  'sudo(\s.*)?',"
        $out += "  'git\s+worktree(\s.*)?',"
        $out += ']'
        $out += 'computer_use = "never"'
        $out += 'context_window_limit = 1000000'
        $out += 'directory_allowlist = []'
        $out += 'execute_commands = "agent_decides"'
        $out += 'mcp_allowlist = []'
        $out += 'mcp_denylist = []'
        $out += 'mcp_permissions = "agent_decides"'
        $out += 'read_files = "always_allow"'
        $out += 'run_agents = "always_ask"'
        $out += 'web_search_enabled = true'
        $out += 'write_to_pty = "always_ask"'
        $replacementWritten = $true
    } elseif ($skipUntilNext -and $line -match '^\[') {
        $skipUntilNext = $false
        $out += $line
    } elseif (-not $skipUntilNext) {
        $out += $line
    }
}

if ($replacementWritten) {
    Set-Content -Path "$env:LOCALAPPDATA\warp\Warp\config\settings.toml" -Value ($out -join "`n") -NoNewline -Encoding UTF8
    Write-Host "✓ default block updated with OVAV Build defaults"
}

# Verify
$verify = Get-Content -Path "$env:LOCALAPPDATA\warp\Warp\config\settings.toml" -Raw
$matches = [regex]::Matches($verify, 'base_model = "([^"]+)"')
foreach ($m in $matches) {
    Write-Host "base_model found: $($m.Groups[1].Value)"
}
