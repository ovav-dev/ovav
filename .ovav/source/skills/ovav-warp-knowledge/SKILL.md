---
name: ovav-warp-knowledge
description: Use when configuring Warp 2026 (settings.toml, profiles, sessions, custom inference endpoints, Tab Configs). Triggers: "warp", "Warp 2026", "settings.toml", "execution_profiles", "warp agent", "tab config", "profile permissions". Enforces CRIT-019 UI-first rule: read docs.warp.dev BEFORE any TOML edit. Never invent enum values.
license: proprietary
metadata:
  version: "1.0.0"
  domain: warp-2026
  triggers: [warp, settings.toml, execution_profiles, tab config, profile, warp agent, custom inference, CRIT-019]
---

# Warp 2026 Knowledge — CRIT-019 Enforcer

## When to load this skill

Load this skill **immediately** when the user mentions:
- "Warp", "Warp 2026", "warp config"
- `settings.toml`, `execution_profiles`, `tab_config`
- "Warp Agent", "Warp profile", "YOLO mode"
- "Custom inference endpoint", "MiniMax Warp"
- Any request to modify Warp behavior

## Core rule: UI-first, docs-first, NEVER invent

| Step | Action | Why |
|---|---|---|
| 1 | **Read docs.warp.dev** for the feature | Enum values are validated server-side; invented values get rejected |
| 2 | **Navigate Warp UI** to the setting | UI shows only valid enum options |
| 3 | **Make change via UI** | Warp writes TOML correctly |
| 4 | **Audit generated TOML** | Confirm value matches docs |
| 5 | **Commit TOML only after audit** | Never bypass UI for TOML edits |

## What NOT to do (CRIT-009 violations)

| ❌ Never invent | ✅ Real Warp 2026 enum |
|---|---|
| `input_box_type_setting = "terminal"` | `default_session_mode` (Terminal/Agent) |
| `execution_profiles.sudo` | Permissions: Apply diffs/Read files/Create plans/Execute/Interact/Ask questions |
| `mcp_permissions = "agent_decides_smart"` | `decide` / `allowlist` / `denylist` |
| `--alert security` on commits | Not in Warp schema |
| `command_allowlist = ["sudo"]` | regex patterns only |
| `base_model = "MiniMax-M2.7"` | Model picker shows only available models |

## Real permission levels (verified 2026-08-19)

| Permission | Levels (UI dropdown) |
|---|---|
| Apply code diffs | Agent decides / Always ask / Always allow |
| Read files | Agent decides / Always ask / Always allow |
| Create plans | Agent decides / Always ask / Always allow / Never |
| Execute commands | Agent decides / Always ask / Always allow |
| Interact with running commands | Agent decides / Always ask / Always allow |
| Ask clarifying questions | Never ask / Ask unless auto-approve / Always ask |

## Real paths (verified 2026-08-19)

| Feature | UI path |
|---|---|
| Default session mode | Settings → Features → General |
| Profiles | Settings → Agents → Profiles |
| Custom inference | Settings → Agents → Inference endpoint |
| Tab Configs | `warp://import_tab_config/<path>` URI scheme |
| Settings file location | `%LOCALAPPDATA%\warp\Warp\config\settings.toml` |
| Tab Configs storage | `%APPDATA%\warp\Warp\data\tab_configs\` |
| Backups | `%LOCALAPPDATA%\warp\Warp\config\backups\` |

## YOLO mode recipe

1. Edit `default` profile (or create new)
2. Base model → select from picker
3. All 5 action permissions → `Always allow`
4. Ask clarifying questions → `Never ask`
5. Save

## MiniMax integration

- Vault key name: `minimax_api_key`
- Endpoint: `https://api.minimax.io/v1`
- Model: `MiniMax-M3`
- Schema: `OpenAI Chat Completions`
- Warp stores API key in OS keychain (Windows Credential Manager), NEVER in TOML

## Pre-action checklist

Before ANY Warp configuration action:

- [ ] Did I read `docs.warp.dev/<feature>`?
- [ ] Is the path/setting I'm targeting actually in the UI?
- [ ] Is the enum value I'm using verified in docs?
- [ ] Have I shown the user the exact UI path with field values?
- [ ] Did I avoid inventing TOML keys not in real schema?

If any checkbox is NO → **stop, read docs, retry.**