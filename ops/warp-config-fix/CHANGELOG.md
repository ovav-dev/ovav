# Warp settings.toml — fix changelog (2026-08-19)

**Author**: thavren (Platform Engineering, OVAV)
**CRIT**: CRIT-019 — UI-first, docs-first, never invent enum values
**Source of truth**: https://docs.warp.dev/terminal/settings/all-settings/ + /agents/capabilities/agent-profiles-permissions/

## Root cause of the error loop

Warp's TOML schema validator is strict. Every field and value not present in the
public schema gets rejected with `Invalid value for '<key>'`. Several fields in
the previous TOML were **invented** keys/values (probably copied from older Warp
versions, blog posts, or hallucinated sources).

## Fixes applied

| # | Field | Was (broken) | Now (validated) | Why |
|---|---|---|---|---|
| 1 | `new_session_shell_override` | `{ w_s_l = "Ubuntu-26.04" }` | `{ custom = "wsl.exe -d Ubuntu-26.04" }` | Doc enum: `string \| { executable = "..." } \| { custom = "..." }`. `w_s_l` is not a schema key. |

## Fields REMOVED (unverified / invented)

| Field | Reason | Recover via |
|---|---|---|
| `ask_user_question = "ask_except_in_auto_approve"` | Wrong key name | Use `ask_questions = "never_ask"` (or `"ask_unless_auto_approve"` / `"always_ask"`) |
| `mcp_permissions = "always_allow"` | Invalid value | UI: Settings → Agents → Profiles → MCP permissions → `decide` / `allowlist` / `denylist` |
| `autosync_plans_to_warp_drive` | Not in public schema | UI: Settings → Agents → Warp Agent → Other |
| `cli_agent_model` | Not in public schema | UI base-model picker |
| `directory_allowlist` | Not in public schema | Removed (default Warp scope applies) |
| `web_search_enabled` | Not in profile TOML | UI: Settings → Agents → Web search |
| `write_to_pty` | Not in public schema | UI: Settings → Agents → Full Terminal Use |
| `computer_use` | Profile-level, UI-managed | UI: Settings → Agents → Profiles → Computer Use |
| `custom_secret_regex_list` block (with GitHub PAT regex) | Risk of leaking PAT; not needed for default redaction | Restore manually only if you need a custom pattern. NEVER paste tokens in `ops/warp-config-fix/`. |

## Fields KEPT (validated against docs)

| Section | Why kept |
|---|---|
| `[appearance.*]` blocks | All keys present in docs Appearance section |
| `[terminal]`, `[terminal.input]` | `osc52_clipboard_access`, `use_audible_bell`, `input_box_type_setting`, `honor_ps1` are all validated |
| `[privacy.*]` | `secret_redaction.enabled`, `hide_secrets_in_block_list`, `secret_display_mode_setting = "asterisks"` are validated |
| `[general]` | `default_session_mode`, `default_tab_config_path` are validated |
| `[agents.warp_agent.*]` | `is_any_ai_enabled`, `nld_in_terminal_enabled`, `ai_auto_detection_enabled`, `ai_command_denylist`, `show_agent_notifications`, `show_conversation_history`, `voice_input_toggle_key`, `voice_input_language`, `cloud_conversation_storage_enabled` all in docs |
| `[agents.execution_profiles.default]` 5 permissions | All 5 permission keys + `ask_questions` are validated; `command_allowlist` / `command_denylist` are validated as `array of strings` |
| `[code]`, `[code.editor]`, `[code.indexing]` | All keys present in docs |
| `[warp_drive]`, `[notifications.*]`, `[account]` | Validated |

## Recovery if you wanted one of the removed fields

1. **MCP**: `Settings → Agents → Profiles → [your profile] → MCP permissions`
2. **Web search**: `Settings → Agents → Warp Agent → Web search`
3. **Computer Use**: `Settings → Agents → Profiles → [your profile] → Computer use`
4. **Custom secret regex**: `Settings → Privacy → Custom secret regex patterns`

UI is the only validated path for these — the public docs do not expose their
TOML representation.

## What if Warp still rejects this TOML?

1. Open `Settings` UI. The exact field name in the error message maps 1:1 to
   the docs page. Read that page.
2. The model UUID `base_model` may be invalid for your Warp version → pick a
   current model from the UI base-model picker and replace the UUID.
3. Verify the file was saved as **UTF-8 without BOM** and uses LF line endings.
