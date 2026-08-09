## Summary

Describe the change and the user-facing impact.

## Validation

- [ ] `python3 -B bin/ovav release-check --version v1.0.0-rc3`
- [ ] `python3 -B bin/ovav publish-check`
- [ ] `python3 -B bin/ovav fresh-smoke`
- [ ] `python3 -B bin/ovav smoke`
- [ ] `python3 -B bin/ovav repo-check`
- [ ] `python3 -B tools/validators/validate_all.py`

## Safety

- [ ] No generated runtime output committed.
- [ ] No secrets or secret-like examples committed.
- [ ] No global install/plugin/live Engram/MCP/A2A claim introduced.
