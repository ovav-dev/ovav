# PowerShell Profile - Braka-Dev v1.0 (multi-version PS5 + PS7)

if ($PSVersionTable.PSVersion.Major -ge 7) {
    Import-Module PSReadLine
    Set-PSReadLineOption -EditMode Windows
    Set-PSReadLineOption -BellStyle None
    Set-PSReadLineOption -HistorySearchCursorMovesToEnd
    Set-PSReadLineKeyHandler -Key UpArrow   -Function HistorySearchBackward
    Set-PSReadLineKeyHandler -Key DownArrow -Function HistorySearchForward
    Set-PSReadLineKeyHandler -Key Tab       -Function MenuComplete
}

if (Get-Module -ListAvailable -Name Terminal-Icons) { Import-Module Terminal-Icons }
if (Get-Module -ListAvailable -Name posh-git) { Import-Module posh-git }

function owc  { wsl -d Ubuntu-24.04 -- fish -c "owc $args" }
function owd  { wsl -d Ubuntu-24.04 -- fish -c "owd $args" }
function owl  { wsl -d Ubuntu-24.04 -- fish -c "owl $args" }
function owv  { wsl -d Ubuntu-24.04 -- fish -c "owv $args" }
function ows  { wsl -d Ubuntu-24.04 -- fish -c "ows $args" }
function owr  { wsl -d Ubuntu-24.04 -- fish -c "owr $args" }
function owx  { wsl -d Ubuntu-24.04 -- fish -c "owx $args" }
function owa  { wsl -d Ubuntu-24.04 -- fish -c "owa $args" }
function obc  { wsl -d Ubuntu-24.04 -- fish -c "obc $args" }
function ovls { wsl -d Ubuntu-24.04 -- fish -c "ovls $args" }
function ovs  { wsl -d Ubuntu-24.04 -- fish -c "ovs $args" }

function gs  { if (Get-Command git -ErrorAction SilentlyContinue) { git status --short --branch @args } else { wsl -d Ubuntu-24.04 -- bash -c "git status --short --branch" } }
function gd  { if (Get-Command git -ErrorAction SilentlyContinue) { git diff @args } else { wsl -d Ubuntu-24.04 -- bash -c "git diff" } }
function ga  { if (Get-Command git -ErrorAction SilentlyContinue) { git add @args } else { wsl -d Ubuntu-24.04 -- bash -c "git add" } }
function gc  { if (Get-Command git -ErrorAction SilentlyContinue) { git commit @args } else { wsl -d Ubuntu-24.04 -- bash -c "git commit" } }
function gp  { if (Get-Command git -ErrorAction SilentlyContinue) { git push @args } else { wsl -d Ubuntu-24.04 -- bash -c "git push" } }
function g   { if (Get-Command git -ErrorAction SilentlyContinue) { git @args } else { wsl -d Ubuntu-24.04 -- bash -c "git @args" } }

Set-Alias -Name ll -Value ls -Force -Option AllScope -ErrorAction SilentlyContinue
Set-Alias -Name la -Value ls -Force -Option AllScope -ErrorAction SilentlyContinue

Write-Host "PS OK - Braka-Dev v1.0 (PS$($PSVersionTable.PSVersion.Major))" -ForegroundColor Cyan
