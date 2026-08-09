# OWS Review Request — Dante (Digital Product)

**De:** Thavren (Platform Engineering)  
**Para:** Dante (Digital Product Engineering)  
**Fecha:** 2026-06-18  
**Prioridad:** Alta — bloquea implementación OWS Fase 5 (Cockpit)  
**Documento:** `docs/architecture/OWS_SPEC.md` (secciones §4, §13.2, §13.3)

---

## Contexto

OWS va a tener una vista completa en Cockpit (la 9na vista). El usuario interactúa con worktrees, estados, conflictos, y merges desde una interfaz visual. Necesito tu criterio de producto antes de diseñar la UI.

## Preguntas

1. **Vista default**: ¿La vista Worktrees debería ser el default al abrir Cockpit? Actualmente el default es el dashboard. Las worktrees son el flujo principal de trabajo diario.

2. **Abstracción para no-dev**: El spec usa términos como "worktree", "merge", "rebase". ¿El usuario no-dev entiende esto? ¿Necesitamos abstraerlo? Por ejemplo: "Espacio de trabajo" en vez de "Worktree", "Integrar" en vez de "Merge", "Actualizar" en vez de "Rebase".

3. **`owc` sin argumentos**: Cuando el usuario ejecuta `owc` sin nombre, ¿debería abrir Cockpit con el selector de perfil o preguntar interactivamente en CLI paso a paso? ¿Qué flujo es más premium?

4. **Side-by-side diff**: La vista de resolución de conflictos muestra el diff de ambas ramas. ¿Debería ser horizontal (estilo IDE, código lado a lado) o vertical (estilo GitHub PR, unificado con marcadores `<<<<<<<`)?

5. **Offline-first UX**: ¿Cómo comunicamos "offline-first" como ventaja premium sin que suene a limitación? Frases como "Trabajá sin internet" o "Tu trabajo, donde sea" — ¿qué tono funciona para el mercado premium?

6. **Métrica de éxito**: ¿Qué métrica define que la vista Worktrees es "exitosa"? ¿Tiempo hasta el primer merge? ¿Reducción de conflictos? ¿Frecuencia de uso diario?

## Dónde leer

- `docs/architecture/OWS_SPEC.md` — especificación completa
- `go-runtime/internal/ows/` — código Fase 1

Responde en este thread o en `.ovav/handoffs/OWS-dante-review.md`.
