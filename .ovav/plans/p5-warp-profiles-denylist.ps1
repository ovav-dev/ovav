# P5: Warp Execution Profiles + Denylist Rebuild
# Plan §12-13. Creates 4 profiles and rebuilds denylist.

param(
    [string]$SettingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml",
    [string]$BackupDir = "$env:LOCALAPPDATA\warp\Warp\config\backups"
)

$ErrorActionPreference = "Stop"

# ── Pre-flight ──────────────────────────────────────────────────────────────
if (-not (Test-Path $SettingsPath)) {
    Write-Error "Settings file not found: $SettingsPath"
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backupPath = Join-Path $BackupDir "settings.toml.pre-p5-$timestamp"
Copy-Item -Path $SettingsPath -Destination $backupPath -Force
Write-Host "✓ Backup: $backupPath"

# ── Read current content ───────────────────────────────────────────────────
$content = Get-Content -Path $SettingsPath -Raw

# ── Mutation 1: REPLACE existing [agents.execution_profiles.default] with 4 profiles ──
# Pattern: match the entire [agents.execution_profiles.default] block
$profileBlock = @'

[agents.execution_profiles.ovav_build]
name = "OVAV Build"
read_files = "always_allow"
apply_code_diffs = "always_allow"
execute_commands = "agent_decides"
ask_user_question = "always_ask"
run_agents = "always_ask"
computer_use = "never"
context_window_limit = 1000000
mcp_permissions = "agent_decides"
directory_allowlist = []
mcp_allowlist = []
mcp_denylist = []
command_allowlist = []
command_denylist = []
web_search_enabled = true
write_to_pty = "always_ask"

[agents.execution_profiles.ovav_yolo]
name = "OVAV YOLO"
read_files = "always_allow"
apply_code_diffs = "always_allow"
execute_commands = "always_allow"
ask_user_question = "never"
run_agents = "always_allow"
computer_use = "never"
context_window_limit = 1000000
mcp_permissions = "always_allow"
directory_allowlist = []
mcp_allowlist = []
mcp_denylist = []
command_allowlist = []
command_denylist = []
web_search_enabled = true
write_to_pty = "always_allow"

[agents.execution_profiles.ovav_review]
name = "OVAV Review"
read_files = "always_allow"
apply_code_diffs = "ask"
execute_commands = "ask"
ask_user_question = "always_ask"
run_agents = "ask"
computer_use = "never"
context_window_limit = 1000000
mcp_permissions = "agent_decides"
directory_allowlist = []
mcp_allowlist = []
mcp_denylist = []
command_allowlist = []
command_denylist = []
web_search_enabled = true
write_to_pty = "ask"

[agents.execution_profiles.thavren_systems]
name = "Thavren Systems"
read_files = "always_allow"
apply_code_diffs = "ask"
execute_commands = "agent_decides"
ask_user_question = "always_ask"
run_agents = "ask"
computer_use = "never"
context_window_limit = 1000000
mcp_permissions = "agent_decides"
directory_allowlist = []
mcp_allowlist = []
mcp_denylist = []
command_allowlist = []
command_denylist = []
web_search_enabled = true
write_to_pty = "ask"
'@

# Match existing [agents.execution_profiles] block (greedy until next [section])
$pattern = '(?ms)\[agents\.execution_profiles\][^\[]*?(?=\n\[|\z)'
if ($content -match $pattern) {
    $content = $content -replace $pattern, $profileBlock.TrimStart("`r`n")
    Write-Host "✓ Mutation 1: 4 execution profiles installed"
} else {
    Write-Warning "Mutation 1: profiles block not found"
}

# ── Mutation 2: rebuild command_denylist (granular patterns) ─────────────
# Remove broad shell blocks; add dangerous patterns + git worktree
$denylistBlock = @'

command_denylist = [
  'sudo(\s.*)?',
  'git\s+worktree(\s.*)?',
  'git\s+reset\s+--hard',
  'git\s+clean\s+-f',
  'git\s+push\s+--force',
  'git\s+branch\s+-D',
  'rm\s+-rf\s+/',
  'rm\s+-rf\s+~',
  'rm\s+-rf\s+\*',
  'curl(\s.*)?',
  'wget(\s.*)?',
  'ssh(\s.*)?',
  'scp(\s.*)?',
  'rsync(\s.*)?',
  'mkfs(\s.*)?',
  'dd(\s.*)?',
  'shutdown(\s.*)?',
  'reboot(\s.*)?',
  'wsl\s+--unregister',
  'wsl\s+--shutdown',
  'reg\s+delete',
  'powershell\s+-Command\s+.*-Force',
  'eval(\s.*)?',
  'exec(\s.*)?',
  'source(\s.*)?',
  'bash(\s.*)?',
  'fish(\s.*)?',
  'pwsh(\s.*)?',
  'sh(\s.*)?',
  'zsh(\s.*)?',
]
'@

# Match existing command_denylist = [...] block
$denylistPattern = '(?ms)command_denylist\s*=\s*\[[^\]]*\]'
if ($content -match $denylistPattern) {
    $content = $content -replace $denylistPattern, $denylistBlock.Trim().TrimStart("`r`n")
    Write-Host "✓ Mutation 2: denylist rebuilt (granular patterns)"
} else {
    Write-Warning "Mutation 2: denylist block not found"
}

# ── Mutation 3: disable denylist bypass flag ───────────────────────────────
# Note: setting these flags ensures Run Until Completion cannot bypass denylist
# Warp uses .autonomous_allowlist or auto-approve bypass settings; we add a
# key that prevents bypass. The exact key may need verification in UI.
$bypassPattern = 'allow_auto_approve_to_bypass_command_denylist\s*=\s*true'
if ($content -match $bypassPattern) {
    $content = $content -replace $bypassPattern, 'allow_auto_approve_to_bypass_command_denylist = false'
    Write-Host "✓ Mutation 3: denylist bypass disabled"
} else {
    Write-Host "  Mutation 3: bypass flag not found (may need UI verification)"
}

# ── Atomic write ────────────────────────────────────────────────────────────
Set-Content -Path $SettingsPath -Value $content -NoNewline -Encoding UTF8
Write-Host "✓ Saved: $SettingsPath"

# ── Verification ──────────────────────────────────────────────────────────
$verify = Get-Content -Path $SettingsPath -Raw
$checks = @(
    @{ name = "ovav_build profile";    test = $verify -match '\[agents\.execution_profiles\.ovav_build\]' },
    @{ name = "ovav_yolo profile";     test = $verify -match '\[agents\.execution_profiles\.ovav_yolo\]' },
    @{ name = "ovav_review profile";   test = $verify -match '\[agents\.execution_profiles\.ovav_review\]' },
    @{ name = "thavren_systems profile"; test = $verify -match '\[agents\.execution_profiles\.thavren_systems\]' },
    @{ name = "git worktree blocked";  test = $verify -match 'git\\s\+worktree\(\\s\.\*\)\?' },
    @{ name = "sudo blocked";          test = $verify -match "'sudo\(\\s\.\*\)\?'," }
)

Write-Host ""
Write-Host "=== Verification ==="
$allPass = $true
foreach ($c in $checks) {
    $status = if ($c.test) { "✓" } else { "✗"; $allPass = $false }
    Write-Host "$status $($c.name)"
}

if (-not $allPass) {
    Write-Host ""
    Write-Host "⚠️  Some checks failed. Verify manually."
}

Write-Host ""
Write-Host "=== Backup (keep for rollback) ==="
Write-Host $backupPath
exit 0
