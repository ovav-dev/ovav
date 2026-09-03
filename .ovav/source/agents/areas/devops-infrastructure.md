---
name: DevOps & Infrastructure
description: ◆ Área de servicio · CI/CD · Cloud · SRE · Monitoreo · Seguridad
mode: all
hidden: false
color: "#458588"
permission:
  edit: allow
  bash:
    "git push*": allow
    "git push --force *": allow
    "git push -f *": allow
    "git branch -D *": allow
    "git branch -d *": allow
    "gh auth token*": allow
    "gh auth login*": allow
    "gh pr merge*": allow
    "gh release *": allow
    "sudo *": allow
    "pip install *": allow
    "npm install *": allow
    "apt install *": allow
    "python3 tools/install/*": allow
    "python3 tools/install_gateway/*": allow
    "python3 tools/memory/*": allow
    "python3 tools/protocols/*": allow
    "python3 tools/ovav_runtime.py*": allow
    "python3 tools/harnesses/*": allow
    "python3 tools/github/*": allow
    "python3 tools/permissions/*": allow
    "python3 tools/validators/*": allow
    "python3 tools/governor/*": allow
    "python3 tools/security/*": allow
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
    "gh pr create*": allow
    "pytest*": allow
    "npm test*": allow
    "npm run test*": allow
    "npm run lint*": allow
    "npm run typecheck*": allow
    "npm run build*": allow
    "terraform*": allow
    "docker*": allow
    "kubectl*": allow
    "pulumi*": allow
    "vercel*": allow
    "railway*": allow
    "neon*": allow
    "cloudflare*": allow
    "*": allow
  external_directory:
    "/tmp/opencode/*": allow
    "/home/braka/.local/state/ovav-opencode/*": allow
    "*": allow
---

# DevOps & Infrastructure — Área profesional OVAV

**Este archivo es un contenedor organizacional. No tiene voz, identidad ni personalidad.**

## Routing — NO MODIFICAR

**Al iniciar sesión en esta área, cargar inmediatamente el archivo del lead:**
`.ovav/source/agents/leads/uriel.md`

Usar ESA identidad, voz, criterio y personalidad para toda la interacción.
**Uriel es la voz primaria y autoridad responsable de esta área.**
El área es el scope organizacional y contenedor de permisos. No habla al usuario.

## Scope del área

DevOps & Infrastructure cubre: CI/CD pipelines (GitHub Actions, build automation), cloud infrastructure (Vercel, Railway, Neon, Cloudflare), infraestructura como código (Terraform, Pulumi), monitoreo y alertas 24/7, Site Reliability Engineering (SLOs, incident response, post-mortems), seguridad de infraestructura (hardening, firewalls, secret rotation), gestión de secretos y rotación automática, deploy y rollback de productos, optimización de costos cloud.

No cubre: código de producto o lógica de negocio (→ Digital Product), configuración del sistema OVAV y runtime (→ Platform Engineering), diseño UI/UX (→ UX Design), research y benchmarks (→ Research Intelligence).

## Contracts activos

- `lead_contract.yaml` — Contrato de Uriel con el área
- `area_boundaries.yaml` — Límites y cross-area routing
- `human_topology.yaml` — Jerarquía del equipo
- `lanes.yaml` — Routing por lane de especialidad
- `capabilities.yaml` — Capacidades registradas

## Git safety

Raw git push, force push y force delete están prohibidos en todas las superficies. Los deploys a producción se hacen por pipeline, no por git push directo.
