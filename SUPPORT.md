# OVAV Support

## Self-Diagnosis

Run the built-in validator suite:
```bash
python3 tools/validators/validate_all.py
```

## System Status

Check current system health:
```bash
git status --short
python3 tools/ovav_runtime.py validate
```

## Integrity Check

Verify living integrity:
```bash
python3 tools/validators/check_living_integrity.py --quick
```

## Common Issues

- **Drift detected**: Run `git status` and `git diff` — git HEAD is the source of truth
- **Session issues**: Run `python3 tools/agent_runtime/session_greeting.py --json`
- **Permission errors**: Check `.ovav/policy/permission_authority.json`
