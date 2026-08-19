# Warp 2026 UI Setup — Final Steps (PASO 4 + 5 + 6)

> **CEO manual steps** — these require Warp UI navigation.
> **Verified against docs.warp.dev** (2026-08-19).

---

## PASO 4 — Default Profile = YOLO

**Path UI:** Settings → Agents → Profiles → click `default`

| # | Field | Value |
|---|---|---|
| 1 | Base model | `MiniMax-M3` (Custom) |
| 2 | Apply code diffs | `Always allow` |
| 3 | Read files | `Always allow` |
| 4 | Create plans | `Always allow` |
| 5 | Execute commands | `Always allow` |
| 6 | Interact with running commands | `Always allow` |
| 7 | Ask clarifying questions | `Never ask` |
| 8 | Save | — |

**Trigger YOLO:** `Ctrl+Shift+I` toggles Run until completion.

---

## PASO 5 — Import 4 Tab Configs

**Path Windows:** `%APPDATA%\warp\Warp\data\tab_configs\`

**Automated (PowerShell):** Run from any shell:

```powershell
$dst = "$env:APPDATA\warp\Warp\data\tab_configs"
New-Item -ItemType Directory -Path $dst -Force
Copy-Item "\\wsl$\Ubuntu\home\braka\Systems\ovav\.ovav\warp\tab-configs\*.toml" -Destination $dst
```

**Manual (alternative):** Right-click each tab → Save as new config → name it.

**Verify in Warp UI:**
- Click `+` button → should show: `OVAV CORE`, `OVAV AGENT`, `OVAV REVIEW`, `OVAV SYSTEMS`

**Schema validated 2026-08-19:**
- ✅ All colors: blue/green/red/magenta (valid set)
- ✅ All types: terminal / agent (valid)
- ✅ All shells: fish (valid)
- ✅ Params: text / branch (valid)

---

## PASO 6 — P10 Secret Redaction Mode

**Path UI:** Settings → Features → Privacy → **Secret redaction mode**

| Option | Visible |
|---|---|
| `asterisks` (Recommended) | ●●●●●●●● |
| `block` | (blocks command entirely) |
| `warn` | warns + runs anyway |

Choose **`asterisks`**.

---

## Audit scripts (post-UI)

After all 3 PASOs, run from WSL:

```bash
cat /mnt/c/Users/Alexa/AppData/Local/warp/Warp/config/settings.toml | grep -A30 "\[profiles\]"
ls "/mnt/c/Users/Alexa/AppData/Roaming/warp/Warp/data/tab_configs/"
```

Expected:
- Profiles section: 1 profile (`default`) with all `Always allow` + `Never ask`
- tab_configs dir: 4 .toml files (ovav_core/agent/review/systems)

---

## After all UI steps done

| Final action | Result |
|---|---|
| Restart Warp | Loads new profile + tab configs |
| `Ctrl+T` | `+` menu shows 4 Tab Configs |
| Select MiniMax-M3 from model picker | Warp Agent uses MiniMax-M3 |
| Test YOLO: type prompt + Enter | Should execute without prompts |
| `Ctrl+Shift+I` | Toggle Run until completion |

---

*Generated 2026-08-19 — verified against docs.warp.dev. CRIT-019 enforced.*