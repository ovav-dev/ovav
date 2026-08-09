# L6 Security / Zero-Trust Runtime — Specification

## Objetivo

Risk scoring compuesto, quarantine dinámico y verificación de procedencia para
todo el runtime de OVAV. L6 corre SOBRE el hardening baseline F0-F4, no lo
reimplementa.

## Módulos

### 1. `tools/security/risk_scorer.py`

Evalúa el riesgo de cualquier operación combinando:

| Factor | Peso | Fuente |
|---|---|---|
| tool_class | 10-65 base | Clasificación de herramienta (read/write/mutate/external/exec) |
| context_tier | 0-90 | Sensibilidad del contexto accedido (T0→T5) |
| dependency_chain | 0-40 | Cantidad + verificación de dependencias |
| hardening_bonus | -50 a 0 | Capas F0-F4 activas (reduce el score) |
| operator_profile | -20 a 0 | Autoridad del operador (reduce el score) |

Output: `RiskAssessment { score, level, action, requires_quarantine }`

### 2. `tools/security/quarantine.py`

Flujo de aislamiento temporal:

1. `risk_scorer.score >70` → `quarantine.quarantine(artifact)`
2. `quarantine.verify(artifact)` → ejecuta F0 validadores + provenance
3. Verify PASA → `quarantine.release(artifact)`
4. Verify FALLA → `quarantine.deny(artifact)` → mueve a `.ovav/quarantine/denied/`

Directorio: `.ovav/quarantine/`
Registry: `.ovav/quarantine/registry.json`

### 3. `tools/security/provenance_checker.py`

Clasifica el origen de cualquier artefacto:

| Origen | Trust | Verificación |
|---|---|---|
| git_tracked | 95 | Hash contra git HEAD |
| sbom_registered | 90 | Hash contra SBOM estático |
| generated | 70 | Confianza condicional |
| external | 10 | Sandbox F3.2 obligatorio |
| unknown | 0 | Deny by default |

## Integración

- **F0.1 sbom.py**: baseline de hashes para verificación
- **F0.3 living_integrity.py**: pre-check de integridad antes de liberar quarantine
- **F3.2 sandbox_governance.py**: sandbox obligatorio para artefactos externos
- **F4.1 bash_commands.py**: reduce risk score en operaciones bash
- **F4.2 unsafe_selectors.py**: reduce risk score en selectores peligrosos
- **L5 context_firewall_v2.py**: risk score alimenta decisión de gate liberation

## Done Definition

- [x] `risk_scorer.py` — scoring compuesto con 5 factores
- [x] `quarantine.py` — flujo isolate → verify → release/deny
- [x] `provenance_checker.py` — 5 fuentes de procedencia con trust levels
- [x] Score >70 activa quarantine automático
- [x] Provenance no verificable → sandbox F3.2 → allowlist/denylist
- [x] Integración con L5 context firewall (risk score → gate decision)
- [x] Harness / validador L6 — check_L6_security_zero_trust.py 4/4 PASS

## Dependencias

- Layer 0, Layer 1, Layer 5, F0-F4 (completado ✅)
- Validación: `check_L6_security_zero_trust.py` — 4/4 PASS (2026-05-31)
