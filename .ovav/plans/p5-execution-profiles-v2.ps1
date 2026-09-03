# P5 v2 — Warp Execution Profiles + denylist rebuild
# Uses ONLY observed schema values from existing `default` profile.
# CRIT-009: no invented enum values. If Warp rejects, backup restores.

param(
    [string]$SettingsPath = "$env:LOCALAPPDATA\warp\Warp\config\settings.toml",
    [string]$BackupDir = "$env:LOCALAPPDATA\warp\Warp\config\backups"
)

$ErrorActionPreference = "Stop"

# ── Pre-flight ──────────────────────────────────────────────────────────────
if (-not (Test-Path $SettingsPath)) {
    Write-Error "Settings not found: $SettingsPath"
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backupPath = Join-Path $BackupDir "settings.toml.pre-p5v2-$timestamp"
Copy-Item -Path $SettingsPath -Destination $backupPath -Force
Write-Host "✓ Backup: $backupPath"

$content = Get-Content -Path $SettingsPath -Raw

# ── Mutation 1: REPLACE [agents.execution_profiles] block with 4 profiles ──
# Uses ONLY values observed in the default profile.
$profileBlock = @'

[agents.execution_profiles]

[agents.execution_profiles.ovav_build]
name = "OVAV Build"
apply_code_diffs = "always_allow"
ask_user_question = "always_ask"
autosync_plans_to_warp_drive = true
base_model = "MiniMax-M3"
command_allowlist = []
command_denylist = [
  'bash(\s.*)?',
  'fish(\s.*)?',
  'pwsh(\s.*)?',
  'sh(\s.*)?',
  'zsh(\s.*)?',
  'curl(\s.*)?',
  'eval(\s.*)?',
  'exec(\s.*)?',
  'source(\s.*)?',
  'wget(\s.*)?',
  'dig(\s.*)?',
  'nslookup(\s.*)?',
  'host(\s.*)?',
  'ssh(\s.*)?',
  'scp(\s.*)?',
  'rsync(\s.*)?',
  'telnet(\s.*)?',
  'rm(\s.*)?',
  'git\s+worktree(\s.*)?',
  'git\s+reset\s+--hard',
  'git\s+clean\s+-f',
  'git\s+push\s+--force',
  'git\s+branch\s+-D',
  'sudo(\s.*)?',
]
computer_use = "never"
context_window_limit = 1000000
directory_allowlist = []
execute_commands = "agent_decides"
mcp_allowlist = []
mcp_denylist = []
mcp_permissions = "agent_decides"
read_files = "always_allow"
run_agents = "always_ask"
web_search_enabled = true
write_to_pty = "always_ask"

[agents.execution_profiles.ovav_yolo]
name = "OVAV YOLO"
apply_code_diffs = "always_allow"
ask_user_question = "always_allow"
autosync_plans_to_warp_drive = true
base_model = "MiniMax-M3"
command_allowlist = []
command_denylist = [
  'curl(\s.*)?',
  'wget(\s.*)?',
  'ssh(\s.*)?',
  'sudo(\s.*)?',
  'git\s+worktree(\s.*)?',
  'rm\s+-rf\s+/',
  'rm\s+-rf\s+~',
  'rm\s+-rf\s+\*',
]
computer_use = "never"
context_window_limit = 1000000
directory_allowlist = []
execute_commands = "always_allow"
mcp_allowlist = []
mcp_denylist = []
mcp_permissions = "always_allow"
read_files = "always_allow"
run_agents = "always_allow"
web_search_enabled = true
write_to_pty = "always_allow"

[agents.execution_profiles.ovav_review]
name = "OVAV Review"
apply_code_diffs = "always_ask"
ask_user_question = "always_ask"
autosync_plans_to_warp_drive = true
base_model = "MiniMax-M3"
command_allowlist = []
command_denylist = [
  'rm\s+-rf\s+/',
  'rm\s+-rf\s+~',
  'curl(\s.*)?',
  'wget(\s.*)?',
  'ssh(\s.*)?',
  'sudo(\s.*)?',
  'git\s+worktree(\s.*)?',
  'git\s+push\s+--force',
  'git\s+reset\s+--hard',
]
computer_use = "never"
context_window_limit = 1000000
directory_allowlist = []
execute_commands = "always_ask"
mcp_allowlist = []
mcp_denylist = []
mcp_permissions = "agent_decides"
read_files = "always_allow"
run_agents = "always_ask"
web_search_enabled = true
write_to_pty = "always_ask"

[agents.execution_profiles.thavren_systems]
name = "Thavren Systems"
apply_code_diffs = "always_ask"
ask_user_question = "always_ask"
autosync_plans_to_warp_drive = true
base_model = "MiniMax-M3"
command_allowlist = []
command_denylist = [
  'rm\s+-rf\s+/',
  'rm\s+-rf\s+~',
  'curl(\s.*)?',
  'wget(\s.*)?',
  'ssh(\s.*)?',
  'sudo(\s.*)?',
  'git\s+worktree(\s.*)?',
  'git\s+push\s+--force',
  'git\s+reset\s+--hard',
  'git\s+clean\s+-f',
]
computer_use = "never"
context_window_limit = 1000000
directory_allowlist = []
execute_commands = "always_ask"
mcp_allowlist = []
mcp_denylist = []
mcp_permissions = "agent_decides"
read_files = "always_allow"
run_agents = "always_ask"
web_search_enabled = true
write_to_pty = "always_ask"
'@

# Match existing block (greedy until next [section])
$pattern = '(?ms)\[agents\.execution_profiles\][^\[]*?(?=\n\[agents\.[a-z_]+\]|\n\[agents\.execution_profiles\.[a-z._]+\]|\n\[|\z)'
if ($content -match $pattern) {
    $content = $content -replace $pattern, $profileBlock.TrimStart("`r`n")
    Write-Host "✓ Mutation 1: 4 profiles installed (CRIT-009-safe values)"
} else {
    Write-Error "Could not locate existing execution_profiles block"
}

# ── Atomic write ────────────────────────────────────────────────────────────
Set-Content -Path $SettingsPath -Value $content -NoNewline -Encoding UTF8
Write-Host "✓ Saved: $SettingsPath"

# ── Verification ──────────────────────────────────────────────────────────
$verify = Get-Content -Path $SettingsPath -Raw
$checks = @(
    @{ name = "ovav_build block present";     test = $verify -match '\[agents\.execution_profiles\.ovav_build\]' },
    @{ name = "ovav_yolo block present";      test = $verify -match '\[agents\.execution_profiles\.ovav_yolo\]' },
    @{ name = "ovav_review block present";    test = $verify -match '\[agents\.execution_profiles\.ovav_review\]' },
    @{ name = "thavren_systems block present"; test = $verify -match '\[agents\.execution_profiles\.thavren_systems\]' },
    @{ name = "default profile removed";       test = $verify -notmatch '\[agents\.execution_profiles\.default\]' },
    @{ name = "base_model set to MiniMax-M3"; test = $verify -match 'base_model = "MiniMax-M3"' },
    @{ name = "git worktree in denylist";      test = $verify -match 'git\\s\+worktree\(\\s\.\*\)\?' }
)

Write-Host ""
Write-Host "=== Verification ==="
$allPass = $true
foreach ($c in $checks) {
    $status = if ($c.test) { "✓" } else { "✗"; $allPass = $false }
    Write-Host "$status $($c.name)"
}

if ($allPass) {
    Write-Host ""
    Write-Host "✅ Profiles applied. CEO must restart Warp to load."
} else {
    Write-Host ""
    Write-Host "⚠️ Verification failed. Backup at: $backupPath"
}

Write-Host ""
Write-Host "=== Backup (keep for rollback) ==="
Write-Host $backupPath
exit 0
