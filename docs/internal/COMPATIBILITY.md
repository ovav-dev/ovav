# OVAV Compatibility Matrix

## Launch baseline

| Component | Launch status |
|---|---|
| WSL/Linux repo-local workflow | Supported |
| fish shell | Supported |
| NVIM editing workflow | Supported |
| OpenCode repo-local surfaces | Supported |
| Python validators | Supported |
| Source-local Git workflow | Supported |
| Global install | Governed; not default |
| Remote push | Requires configured remote and explicit approval |

## Expected repository surfaces

```txt
.opencode/agents/
.opencode/commands/
.opencode/skills/
.ovav/service_areas/
tools/validators/
tools/harnesses/
tools/memory/          [DEPRECATED — memory system removed 2026-06-11]
tools/snapshot/        [DEPRECATED — capsule system removed 2026-06-11]
```

## Minimum launch validation

```fish
python3 tools/validators/check_service_area_governance.py
python3 tools/validators/check_build17_canonical_review.py
python3 tools/validators/check_build18_launch_pack.py
python3 tools/validators/validate_all.py
```

## Not guaranteed by default

- arbitrary global configuration writes,
- unmanaged plugin install,
- external adapter execution without gating,
- production/global-ready claims before launch gates pass.
