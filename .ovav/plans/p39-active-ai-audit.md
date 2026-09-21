# P39 — Active AI bajo demanda (audit)

## Policy

Per plan §39:
- Warp Agent: available, manual
- Active AI recommendations: OFF
- AI auto detection: OFF
- Natural Language detection: OFF
- Code suggestion noise: OFF
- OpenCode: manual
- Crush: manual
- Result: Terminal Mode = deterministic, Agent Mode = deliberado

## Verification (TOML state)

| Setting | Required | Actual | Status |
|---|---|---|---|
| `show_agent_notifications` | false | `true` | ⚠️ verify UI setting |
| `nld_in_terminal_enabled` | false | `false` | ✅ |
| `ai_auto_detection_enabled` | false | `false` | ✅ |
| `web_search_enabled` | (plan n/a) | `true` | OK |
| `data.is_any_ai_enabled` | true (user enabled) | `true` | ✅ |

## Notes

- `show_agent_notifications = true` is OK for Agent mode notifications;
- it's not the same as "AI recommendations" (which is Warp-specific).
- CEO confirms UI preferences: Notifications ON, Recommendations OFF.

## Acceptance criteria

- [x] Terminal Mode = deterministic
- [x] Agent Mode = manual (not auto)
- [x] NLD OFF
- [x] AI auto detect OFF
- [x] Notifications manual (CEO choice)

## Status

✅ P39 100% — minor clarification on notifications vs recommendations.
