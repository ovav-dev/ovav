# L7 Feedback Loop — Specification

## Objetivo

Implementar un ciclo de aprendizaje operativo que conserva señales útiles entre
sesiones sin guardar raw chat, secretos ni memoria viva no gobernada.

## Superficies

- `tools/agent_runtime/belief_manager.py` — creencias revocables/emergentes.
- `tools/agent_runtime/feedback_loop.py` — captura de decisiones, compactación e
  mejora de paquetes de contexto.
- `tools/validators/check_L7_feedback_loop.py` — validador determinístico L7.
- `.ovav/artifacts/L7/L7_DECISION_PACKET.yaml` — evaluación previa obligatoria.

## Gobernanza

- F5.1 `new_states`: las creencias L7 usan solo estados `revocable` y
  `emergent`; estados desconocidos fallan cerrados.
- F5.2 `ledger_vivo`: las escrituras reales al ledger se bloquean si el gate no
  está liberado. En ese caso L7 hace safe-stop y no escribe.
- F0.2/F0.6: L7 sanea patrones de secretos antes de persistir y nunca guarda
  raw chat como autoridad.

## Done definition

- [x] `capture_decision()` registra decisiones sanitizadas cuando el gate lo
  permite.
- [x] `deprecate_belief()` marca creencias obsoletas como `deprecated`.
- [x] `compact_memory()` fusiona duplicados y evita acumulación infinita.
- [x] `improve_packet()` produce señales compactas para continuidad.
- [x] Creencias emergentes expiran/deprecan tras ventana semanal sin validación.
- [x] `ledger_vivo` se respeta: si el gate está cerrado, no hay escritura viva.
- [x] Los overrides de prueba solo funcionan sobre ledgers temporales.
- [x] El primitive público `save()` también está protegido por `ledger_vivo`.

## Evidencia esperada

- `python3 tools/harnesses/evaluation_pipeline_runner.py .ovav/artifacts/L7/L7_DECISION_PACKET.yaml --json`
- `python3 tools/validators/check_L7_feedback_loop.py`
- `python3 tools/validators/check_f5_advanced_hardening.py`
- `OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate`
