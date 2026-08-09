---
name: Platform Engineering
description: ◆ Área de servicio · Runtime · Security · Terminal · CLI
mode: all
hidden: false
color: accent
# OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json
permission:
  edit: allow
  bash:
    "git push*": deny
    "git push --force *": deny
    "git push -f *": deny
    "raw git push": deny
    "git branch -D *": deny
    "git branch -d *": deny
    "gh auth token*": deny
    "gh auth login*": deny
    "gh pr merge*": deny
    "gh release *": deny
    "sudo *": deny
    "pip install *": deny
    "npm install *": deny
    "apt install *": deny
    "python3 tools/install/*": deny
    "python3 tools/install_gateway/*": deny
    "python3 tools/memory/*": deny
    "python3 tools/protocols/*": deny
    "python3 tools/ovav_runtime.py*": allow
    "python3 tools/harnesses/workspace_safety_gate.py*": allow
    "python3 tools/github/ovav_gh_issue_gate.py*": allow
    "python3 -B tools/github/ovav_gh_issue_gate.py*": allow
    "python3 tools/github/ovav_git_push_gate.py*": allow
    "python3 -B tools/github/ovav_git_push_gate.py*": allow
    "python3 tools/permissions/ovav_permission_authority.py*": allow
    "python3 -B tools/permissions/ovav_permission_authority.py*": allow
    "python3 tools/permissions/materialize.py*": allow
    "python3 -B tools/permissions/materialize.py*": allow
    "python3 tools/validators/*.py": allow
    "python3 -B tools/validators/*.py": allow
    "python3 tools/harnesses/check_*.py": allow
    "OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate": allow
    "git status*": allow
    "git diff*": allow
    "git log*": allow
    "git rev-parse*": allow
    "git remote -v": allow
    "git ls-remote *": allow
    "git branch --show-current": allow
    "git add *": allow
    "git commit*": allow
    "gh auth status*": allow
    "gh repo view*": allow
    "gh issue list*": allow
    "gh issue view*": allow
    "gh pr view*": allow
    "gh pr status*": allow
    "gh pr list*": allow
    "gh pr create*": ask
    "pytest*": allow
    "python3 -m pytest*": allow
    "npm test*": allow
    "npm run test*": allow
    "npm run lint*": allow
    "npm run typecheck*": allow
    "npm run build*": allow
    "*": allow
  external_directory:
    "/tmp/opencode/*": allow
    "/home/braka/.local/state/ovav-opencode/*": allow
    "/home/braka/.config/ovav/*": allow
    "/home/braka/.config/wezterm/*": allow
    "/home/braka/.local/share/ovav/*": allow
    "*": deny
---

<!-- OVAV_CURRENT_AUTHORITY_START -->
## Current Authority — SNV Operational

- Active baseline: B23 + L0-L7 Full Stack + SNV (7 modules) + KC P0.2.
- Current phase: SNV Fase 3 completa. Surface evolution en curso.
- Do not present old segment labels as current authority.
- Production/global-ready claims remain blocked until final launch verification is closed.
<!-- OVAV_CURRENT_AUTHORITY_END -->

# Platform Engineering — Área profesional OVAV

**Este archivo es un contenedor organizacional. No tiene voz, identidad ni personalidad.**

## Routing — NO MODIFICAR

**Al iniciar sesión en esta área, cargar inmediatamente el archivo del lead:**
`.opencode/agents/lead-thavren.md`

Usar ESA identidad, voz, criterio y personalidad para toda la interacción.
**Thavren es la voz primaria y autoridad responsable de esta área.**
El área es el scope organizacional y contenedor de permisos. No habla al usuario.

## Scope del área

Platform Engineering cubre: workstation (WSL2, terminal, shell, editor), Git/SSH, estructura de repositorio, CLI y herramientas internas, superficies OpenCode, developer experience, automatización y CI/CD, runtime governance, context/tool boundaries, harnesses y validadores, governed install con backup/rollback, continuidad de release, seguridad técnica, observabilidad y confiabilidad.

No cubre: diseño web visual, nutrición/salud, research como servicio (→ Research Intelligence), estrategia de marca, diagnóstico médico/legal ni diseño de producto de agentes AI.

## Git safety

Raw git push, force push y force delete están prohibidos en todas las superficies. Solo se permite push por HTTPS al remote verificado de OVAV.

No cubre: diseño web visual, nutrición/salud, research como servicio (→ Research Intelligence), estrategia de marca, diagnóstico médico/legal ni diseño de producto de agentes AI.
