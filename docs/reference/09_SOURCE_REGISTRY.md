# Source Registry — Clasificación y Acceso a Fuentes

## Context Classes

```text
L0_public_external     — Fuentes públicas externas (web, papers, vendor docs)
L1_shared_governance   — Gobernanza compartida (contratos, registries, launch docs)
L2_platform_internal   — Interno de Platform Engineering (repo por tarea)
L2_research_internal   — Interno de Research Intelligence
L3_core_ovav_internal  — Core de OVAV (.opencode, snapshots, memory, harnesses, validators)
L4_sensitive_execution — Ejecución sensible (install tools, secrets, credentials)
```

---

## Mapping Completo

| Source | Path | Class | Platform | Research |
|---|---|---|---|---|
| Public external | `<external>`, `http://`, `https://` | L0 | limited | allow |
| Service area governance | `.ovav/service_areas/` | L1 | read | read |
| Launch docs | `docs/launch/` | L1 | read | read |
| Registry | `registry/` | L1 | read | read |
| OpenCode surface | `.opencode/` | L3 | allow_by_task | deny_by_default |
| OVAV context | `.ovav/context/` | L3 | allow_by_task | deny_by_default |
| OVAV snapshots | `.ovav/snapshots/` | L3 | allow_by_task | deny_by_default |
| Harnesses | `tools/harnesses/` | L3 | allow_by_task | deny_by_default |
| Validators | `tools/validators/` | L3 | allow_by_task | deny_by_default |
| Install tools | `tools/install/` | L4 | capability_grant_required | deny |
| Runtime evidence | `.ovav/runtime/evidence/` | L3 | allow_by_task | deny_by_default |
| Identity packets | `.ovav/runtime/` | L3 | allow_by_task | deny_by_default |
| Token vocabularies | `tools/agent_runtime/vocab/` | L2 | allow_by_task | deny_by_default |
| Agent runtime tools | `tools/agent_runtime/` | L2 | allow_by_task | deny_by_default |

---

## Reglas de Acceso

### Platform Engineering (Thavren)

```text
ALLOWED:
  · L0: limitado (lectura pública)
  · L1: lectura completa (gobernanza compartida)
  · L2: lectura/escritura interna por tarea
  · L3: lectura/escritura por tarea y scope

REQUIERE GRANT:
  · L4: capability_grant_required + backup + verify + rollback
  · global_config_write: requiere approval explícito
  · install_apply: requiere approval + backup + verify + rollback

DENIED:
  · unscoped_home_write
  · secret_exfiltration
  · uncontrolled_global_install
  · broad_git_staging (sin workspace safety gate)
```

### Research Intelligence (Eidren)

```text
ALLOWED:
  · L0: acceso completo (investigación pública)
  · L1: lectura (gobernanza compartida)

DENIED BY DEFAULT:
  · L2_platform_internal: no lee repo por asociación
  · L3_core_ovav_internal: no lee .opencode, snapshots, memory
  · L4_sensitive_execution: denegado siempre
  · repo_root: denegado por defecto
  · raw_snapshots: denegado

REQUIERE PERMISO EXPLÍCITO:
  · scoped_internal_review
  · platform_internal
  · specific_repo_file
  · sanitized_platform_handoff
```

---

## Reglas Globales

```text
1. unknown_path → deny_or_requires_permission (fail-closed)
2. no_research_repo_root_default → research no abre repo sin permiso
3. no_raw_snapshot_cross_area → snapshots no se comparten entre áreas
4. no_raw_chat_handoff → handoffs siempre sanitizados
5. deny-before-allow → negar primero, conceder solo si cumple criterios
6. semantic_similarity ≠ authorization → que una fuente "parezca" confiable no la autoriza
```
