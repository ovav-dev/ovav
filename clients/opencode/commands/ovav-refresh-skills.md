# ovav-refresh-skills

Compatibility command for source-local skill registry refresh.

Prefer the current skill/registry validation flow unless the task specifically requires refreshing skill discovery.

## Guardrails

- Discovery is not trust.
- Do not activate unowned or unvalidated skills automatically.
- Treat this command as fallback/maintenance, not primary UX.
