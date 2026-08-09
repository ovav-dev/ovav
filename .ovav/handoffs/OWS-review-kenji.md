# OWS Review Request — Kenji (Adversarial Intelligence)

**De:** Thavren (Platform Engineering)  
**Para:** Kenji Tanaka (Adversarial Intelligence)  
**Fecha:** 2026-06-18  
**Prioridad:** Alta — bloquea implementación OWS Fase 2  
**Documento:** `docs/architecture/OWS_SPEC.md` (secciones §6, §9, §13.1, §13.2)

---

## Contexto

Estamos diseñando OVAV Worktree Orchestration System. 10 comandos, state machine con 10 estados, política engine, resolución de conflictos asistida por IA, modo offline.

Necesito tu criterio adversarial antes de implementar. Esto va a production para usuarios premium. Si hay un agujero, encontralo ahora.

## Preguntas

1. **`owx emergency` bypass**: El modo emergency permite bypass de políticas con waiver del CEO. ¿Qué vectores de ataque ves? ¿Un waiver robado o falsificado podría hacer push a main sin verificación? ¿Cómo debería validarse el waiver?

2. **`owlk` DoS entre agentes**: El lock de worktrees impide que otros agentes modifiquen. ¿Podría un agente malicioso o bug lockear worktrees de otros leads para bloquear desarrollo? ¿Qué mecanismo de unlock forzoso debería existir?

3. **Seguridad en `owv`**: El pipeline de verificación ejecuta 6 herramientas (gitleaks, semgrep, go test, go vet, gofmt, validate CLI). ¿Cubren todos los casos de seguridad relevantes para un sistema de gobernanza Git? ¿Qué falta?

4. **Resolución IA de conflictos**: La IA propone merge de código en conflicto. ¿Podría introducir código vulnerable? ¿Qué validación post-merge es necesaria? ¿Debería ser solo sugerencia (nunca automático)?

5. **Cola offline**: Las operaciones pendientes se encolan en SQLite y se ejecutan al reconectar. ¿Podría manipularse la cola para ejecutar operaciones no autorizadas? ¿Qué firma o validación necesita?

6. **Supply chain del SQLite**: `modernc.org/sqlite` es una dependencia externa. ¿Qué riesgo de supply chain introduce? ¿Hay alternativas más seguras?

## Dónde leer

- `docs/architecture/OWS_SPEC.md` — especificación completa (680 líneas)
- `go-runtime/internal/ows/` — código Fase 1

Responde en este thread o en `.ovav/handoffs/OWS-kenji-review.md`.
