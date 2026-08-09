# CLI Living Experience — Plan de Segmentación
## RC8.4 → RC9.0 · De cockpit muerto a experiencia viva

**Rama:** `task/cli-living-experience-rc9`
**Base:** `develop`
**Fase actual:** Final Launch Verification
**Objetivo:** Transformar el cockpit de un menú diagnóstico de solo lectura a una experiencia viva, interactiva y premium.

---

## Mapa de segmentos

```
RC8.3 (actual)         RC8.4           RC8.5          RC8.6           RC9.0
───────────────    ───────────    ───────────    ───────────    ───────────
Menú navegable     Launch real     Tailor vivo    Recovery+Upd   Pulido final
Solo lectura       Instala de      Toggle tools   Acciones       Smoke tests
Diagnóstico        verdad          Selección      reales          Cierre
                                  
   [SEG 1]          [SEG 2]        [SEG 3]      [SEG 4-5-6]     [SEG 7-8]
```

---

## Segmento 1 — Purga de opciones internas

**Objetivo:** Menú principal de 4 opciones limpias. Diagnóstico a `ovav doctor`.

**Menú resultante:** Instalar OVAV · Configurar · Actualizar · Recuperar

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`
**Issue:** `[SEG 1] Purga de opciones internas del menú principal`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_01_PURGE_INTERNAL_OPTIONS.md`

---

## Segmento 2 — Cablear Launch: instalación real

**Objetivo:** Instalar desde el cockpit con pipeline real: backup → consent → apply → verify.

**Pipeline:** Detectar → Backup → Consent → Apply → Verify → Resultado

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`
**Issue:** `[SEG 2] Cablear Launch: instalación real con progreso`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_02_WIRE_LAUNCH_REAL_INSTALL.md`

---

## Segmento 3 — Tailor interactivo con toggles

**Objetivo:** Configurar herramientas, roles y plan con Space=toggle, Enter=apply.

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`, `tools/cli/ovav_tailor_composer.py` (nuevo)
**Issue:** `[SEG 3] Tailor interactivo con toggles vivos`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_03_TAILOR_INTERACTIVE_TOGGLES.md`

---

## Segmento 4 — Recovery y Update reales

**Objetivo:** Update y Recovery ejecutan flujos reales con progreso, preview y confirmación.

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`
**Issue:** `[SEG 4] Recovery y Update con acciones reales`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_04_RECOVERY_UPDATE_REAL.md`

---

## Segmento 5 — Capa visual premium

**Objetivo:** Catppuccin Mocha completo, transiciones suaves, motion, status chips semánticos.

**Archivos:** `tools/cli/ovav_visual_theme.py` (nuevo), `tools/cli/ovav_first_run_cockpit.py`
**Issue:** `[SEG 5] Capa visual premium — Catppuccin Mocha + transiciones`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_05_PREMIUM_VISUAL_LAYER.md`

---

## Segmento 6 — Pantallas de resultado

**Objetivo:** Toda acción termina en resultado estructurado: qué pasó, verificación, próximo paso.

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`
**Issue:** `[SEG 6] Pantallas de resultado y verificación post-acción`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_06_RESULT_VERIFICATION_SCREENS.md`

---

## Segmento 7 — Guía contextual y first-run

**Objetivo:** First-run guiado, footers contextuales, usuario nunca perdido.

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`, `bin/ovav`
**Issue:** `[SEG 7] Guía contextual y detección first-run`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_07_CONTEXTUAL_GUIDANCE.md`

---

## Segmento 8 — Pulido, smoke y cierre RC9.0

**Objetivo:** Cero placeholders, cockpit legacy archivado, 6 smoke tests, VERSION → rc9.

**Archivos:** `tools/cli/ovav_first_run_cockpit.py`, `bin/ovav`, `VERSION`
**Issue:** `[SEG 8] Pulido final, smoke tests y cierre RC9.0`
**Task:** `.ovav/tasks/cli-living-experience/ISSUE_08_POLISH_SMOKE_CLOSURE.md`

---

## Flujo de trabajo

```
develop
  └── task/cli-living-experience-rc9
        │
        ├── SEG 1 → commit → push
        ├── SEG 2 → commit → push
        ├── SEG 3 → commit → push
        ├── SEG 4 → commit → push
        ├── SEG 5 → commit → push
        ├── SEG 6 → commit → push
        ├── SEG 7 → commit → push
        ├── SEG 8 → commit → push → merge a develop → tag RC9.0
        │
        └── Usuario: ovav update → recibe todas las mejoras
```

---

## Reglas OVAV

- Source-local only. No global writes hasta que el install governor lo permita.
- Cada segmento produce evidencia en `.ovav/artifacts/CLI_LIVING_EXPERIENCE/`.
- Validadores deben pasar antes de cada commit.
- Commits atómicos: un segmento por commit.
- El usuario nunca necesita un comando manual. Todo es experiencia.
