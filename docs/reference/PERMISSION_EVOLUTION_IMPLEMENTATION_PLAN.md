# Plan de Implementación — Evolución de Permisos OVAV

> Rama: `task/permission-evolution-implementation`
> Worktree: `/home/braka/Systems/OVAV-perm-evo`
> Autoridad: `docs/research/PERMISSION_EVOLUTION_DECISIONS.md`
> Origen: Evaluación de 10 bloques, 163 reglas

## Fases

### F1 — Arquitectura + Observabilidad (Bloques 8, 10)

| # | Qué | Archivos afectados |
|---|---|---|
| 8.6 | AWS IAM: identity + resource policies + condiciones | `.ovav/policy/permission_authority.json` |
| 8.7 | OPA/Rego: policy-as-code, motor desacoplado | `tools/permissions/` |
| 8.11 | Google BeyondCorp: zero-trust paraguas | `.ovav/service_areas/shared/` |
| 8.16 | Policy simulation | `tools/permissions/simulate.py` |
| 8.17 | Formal verification | `tools/permissions/verify.py` |
| 10.1-5 | Superficies observadas (trace, drift, squads, context) | `tools/validators/` |

### F2 — Infraestructura (Bloque 3)

21 ALLOW, 3 DENY. System paths, directorio externo, config global, comportamiento live, plugins, claims.

### F3 — Roles (Bloques 2, 5, 6)

53 ALLOW, 4 DENY. Research, sandbox, budgets.

### F4 — Seguridad (Bloques 1, 4)

20 ALLOW, 11 DENY, 1 ASK. Bash, unsafe selectors.

### F5 — Avanzado (Bloques 7, 9)

22 ALLOW, 3 DENY. Estados nuevos, gate liberation.

---

**Inicio: F1 — Arquitectura + Observabilidad**
