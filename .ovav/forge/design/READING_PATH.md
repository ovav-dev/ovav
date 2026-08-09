# OVAV — Rutas de Lectura
# =========================
# Tres rutas distintas. Cada una con un propósito claro.

---

## RUTA A — ENTENDER OVAV (para alguien nuevo)
## ¿Qué es OVAV, cómo se usa, dónde está cada cosa?

  1. docs/OVAV_USER_GUIDE.md              ← ✅ COMPLETADO (Fase A5)
     → Guía práctica: comandos, flujo de trabajo, estructura.

  2. IMPLEMENTATION_PLAN.md               ← Qué sistemas existen y su estado
     → Tabla de ~40 sistemas con ✅/⏸️/🔲

  3. .ovav/connector_bus/connectors/      ← Dónde se conecta cada componente
     → 9 archivos YAML, uno por tipo. Leer el que necesites.

  4. tools/INDEX.md                       ← ✅ COMPLETADO (Fase A6)
     → Mapa de los 623 archivos en tools/

---

## RUTA B — IMPLEMENTAR (para Thavren o quien ejecute Fase E)
## ¿Qué hay que hacer, en qué orden, con qué criterio?

  1. IMPLEMENTATION_PLAN.md → Fase E
     → E0 a E5 en orden de dependencia. 20 pasos. Archivos exactos. Leyes impactadas.

  2. .ovav/connector_bus/connectors/
     → E0: Trigger Compiler, Surface Map, Projection Engine leen y escriben acá.

  3. .ovav/laws/ovav_laws.yaml
     → Verificar compliance antes de cerrar cada E.

---

## RUTA C — ABSORBER LA SESIÓN (para otro chat o agente)
## ¿Qué se discutió, diseñó y decidió en la sesión del 03-Jun?

  1. .ovav/forge/design/SESSION_ABSORPTION_COMPLETE_2026-06-03.md
     → TODO el criterio, diseño, gaps, feedback del usuario.
     → Leer esto equivale a haber estado en la sesión completa.
