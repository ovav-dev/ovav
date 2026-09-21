# Warp settings.toml — apply instructions (Windows 11)

## Where it goes

```
%LOCALAPPDATA%\warp\Warp\config\settings.toml
```

Expanded (depending on Windows install):

```
C:\Users\Alexa\AppData\Local\warp\Warp\config\settings.toml
```

## Step-by-step (Warp closed)

1. **Close Warp completely** (right-click system tray icon → Quit).
2. **Backup your current file**:
   - Press `Win+R`, paste: `%LOCALAPPDATA%\warp\Warp\config`
   - Copy `settings.toml` to a backup folder.
3. **Open `settings.toml` in your editor** (VSCode, Notepad++, Notepad).
4. **Ctrl+A → Delete** to clear it.
5. **Open** `ops/warp-config-fix/settings.toml` from your Linux side.
6. **Ctrl+A → Ctrl+C** to copy its entire content.
7. **Paste into the empty `settings.toml` on Windows**.
8. **Save as UTF-8, LF line endings** — Notepad defaults to CRLF; use
   Notepad++ or VSCode (`Ctrl+Shift+P → "Change End of Line Sequence → LF"`).
9. **Open Warp** — confirm no "Invalid value for ..." dialog.

## Quick sanity checks after applying

```powershell
# In Windows PowerShell (from the Linux side via wsl if you like)
Get-Content "$env:LOCALAPPDATA\warp\Warp\config\settings.toml" | Select-String "new_session_shell_override"
# Expected: new_session_shell_override = { custom = "wsl.exe -d Ubuntu-26.04" }

Get-Content "$env:LOCALAPPDATA\warp\Warp\config\settings.toml" | Select-String "ask_user_question"
# Expected: (no output — that field was removed)
```

## If Warp still complains

| Symptom | Likely cause | Fix |
|---|---|---|
| "Invalid value for 'base_model'" | UUID obsolete in your Warp version | UI: Settings → Agents → Profiles → Default → pick current model from picker → copy new UUID |
| "Invalid value for 'input_box_type_setting'" | (Already valid: `"universal"`) | n/a |
| New session opens Windows PowerShell instead of WSL Ubuntu | WSL distro not registered | `wsl --list --verbose` to confirm; `wsl --set-default Ubuntu-26.04` |
| Auto-approve keeps prompting | Denylist blocks the command | UI: Settings → Agents → Profiles → Default → Command denylist → empty (or remove specific entries). Read `/warp-config-fix/CHANGELOG.md` for why. |

## Optional: drop to PowerShell to apply in one shot

If you have SSH/WSL access from this Linux box to your Windows install, you can
push the file in one line. **Skip this if you do not have that bridge set up** —
the manual paste above is enough.

```bash
# Read the cleaned TOML
cat /home/braka/Systems/ovav/ops/warp-config-fix/settings.toml
# Then on Windows (PowerShell):
# 1. Backup
Copy-Item "$env:LOCALAPPDATA\warp\Warp\config\settings.toml" `
          "$env:LOCALAPPDATA\warp\Warp\config\settings.toml.bak_$(Get-Date -Format yyyyMMdd_HHmmss)"
# 2. Paste your edited content into the file, save as UTF-8 (no BOM).
```

## What this baseline ENABLES

- **YOLO mode** on the `Default` profile: 5/5 permissions = Always allow, Ask = Never ask.
- **WSL Ubuntu-26.04** as the default shell for new sessions.
- **Universal input** (AI-first terminal).
- **Secret redaction** with asterisks.
- **Settings Sync** enabled (so this baseline propagates to your other devices).

## What this baseline DISABLES

- **MCP auto-allow** (defaults to `decide`, must be enabled per profile).
- **Custom secret regex list** (was previously shipped in your TOML with a GitHub
  PAT regex included). Re-add only if a CLI tool needs regex-based detection.
- **Computer Use** defaults (UI toggle controls it).
