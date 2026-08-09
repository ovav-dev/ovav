# OVAV Harness Index — Guía rápida diaria

Índice práctico de harnesses, validators y herramientas para uso diario con OVAV.

## 🛡️ Health Checks (antes de cualquier trabajo)

```bash
python3 tools/governor/self_diagnosis.py        # Diagnóstico completo (35 checks)
python3 tools/validators/check_living_integrity.py --quick  # Integrity Mesh rápido
python3 tools/ovav_runtime.py context --next     # Contexto actual + próximo trabajo
```

## ✅ Validators (verificación de integridad)

| Comando | Qué verifica |
|---|---|
| `check_workspace_safety_gate.py` | Seguridad del workspace antes de writes |
| `check_git_push_gate.py` | Seguridad antes de push |
| `check_config_syntax.py` | Sintaxis YAML/JSON de 303 configs |
| `check_canonical_integrity.py` | Integridad canónica de archivos core |
| `check_ledger_write_path.py` | Escritores canónicos vs no-canónicos |
| `check_validator_coverage.py` | Cobertura de validators |
| `check_registry_drift.py` | Drift en registros |
| `check_contract_freshness.py` | Frescura de contratos |
| `check_handoff_sync.py` | Sincronización de estado (auto-alinea) |
| `check_todo_debt.py` | Deuda técnica (TODO/FIXME) |

```bash
python3 tools/validators/validate_all.py         # Todos los validators
```

## 🔧 Auto-Repair & Governance

| Sistema | Comando |
|---|---|
| **S13 ARC** | `python3 tools/agent_runtime/autonomous_repair_cortex.py health` |
| ARC Scan | `python3 tools/agent_runtime/autonomous_repair_cortex.py scan-and-repair` |
| **S14 IAP** | `python3 tools/governor/integration_acceptance_protocol.py status` |
| IAP Validate | `python3 tools/governor/integration_acceptance_protocol.py validate --manifest <file>` |
| **S15 SMC** | `python3 tools/governor/system_maturity_classifier.py analyze` |

## 🧠 Knowledge & Learning

| Comando | Qué hace |
|---|---|
| `snv_predictor.py predict` | Predicciones de fallos (auto-verifica) |
| `snv_predictor.py status` | Estado del predictor |
| `expand_canonical_base.py` | Expandir base canónica OutputRails |
| `generate_validator_tests.py` | Generar tests para validators |
| `standardize_governor_cli.py` | Estandarizar CLI de governor tools |

## 🔒 Git Safety

```bash
python3 tools/harnesses/workspace_safety_gate.py --mode mutate   # Pre-write
python3 tools/github/ovav_git_push_gate.py                       # Pre-push
python3 tools/validators/check_protected_branch.py --mode pre_write
```

## Flujo diario recomendado

1. `self_diagnosis.py` — verificar salud
2. `ovav_runtime.py context --next` — ver qué sigue
3. `validate_all.py` — validación completa
4. Trabajar → `workspace_safety_gate.py --mode mutate` → commit → `git_push_gate.py` → push
5. `system_maturity_classifier.py analyze` — verificar nivel de madurez
