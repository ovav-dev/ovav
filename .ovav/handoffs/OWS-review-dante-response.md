# OWS Cockpit Review — Dante (Digital Product Engineering)

**De:** Dante (Digital Product Engineering Lead)  
**Para:** Thavren (Platform Engineering Lead)  
**Fecha:** 2026-06-18 02:37 UTC-5  
**Referencia:** `docs/architecture/OWS_SPEC.md` §4, §13.2, §13.3, §15, §16  
**Decisión:** APROBADO con modificaciones. Las 6 capas de UX listadas abajo son obligatorias para Fase 5 (Cockpit).

---

## Resumen ejecutivo

El spec es sólido. Las 8 capas invisibles (state machine, policy engine, conflict prediction, AI resolution, event bus, SQLite audit, offline queue, verify pipeline) son exactamente lo que un producto premium 2026 necesita. Pero el Cockpit —la cara visible de todo esto— requiere 6 capas de UX que el spec actual no define. Esta review las especifica.

**Principio rector:** OWS debe sentirse como un copiloto que anticipa, no como una herramienta que obedece.

---

## P1: ¿Vista Worktrees como default al abrir Cockpit?

**Respuesta:** No. Dashboard como default. Worktrees como segundo hogar con memoria de estado.

**Razonamiento:**

| Aspecto | Dashboard | Worktrees |
|---------|-----------|------------|
| Propósito | Orientación, contexto global | Acción, trabajo activo |
| Usuario nuevo | Necesita ver el panorama | No sabe qué es un worktree todavía |
| Usuario recurrente | Quiere volver donde estaba | Quiere retomar su tarea activa |
| Carga cognitiva | Baja (vista general) | Media-alta (detalle operacional) |

**Recomendación UX (3 reglas):**

1. **Primera visita → Dashboard.** Es el "lobby" del producto. Muestra salud del sistema, worktrees activas (widget pequeño), y accesos rápidos a `owc`.
2. **Visitas subsiguientes → Última vista usada.** Persistir en SQLite (`cockpit_state.last_view`). Si el usuario cerró en Worktrees, ahí vuelve. Si cerró en Dashboard, ahí vuelve. Sin fricción.
3. **Atajo de teclado `W` desde cualquier vista → Worktrees.** El power user no navega menús. Una tecla y está trabajando.

```
Implementación Cockpit:
┌─────────────────────────────────────────┐
│ [D]ashboard  [W]orktrees  [P]olicies …  │  ← tabs persistentes con atajos
│                                         │
│   ↑ Tab activo recordado por sesión     │
└─────────────────────────────────────────┘
```

**Decisión:** Dashboard default, memoria de última vista, atajo `W` directo.

---

## P2: ¿Abstraer terminología git para no-dev?

**Respuesta:** Sí. Sistema de dos capas léxicas con toggle.

**Problema:** El spec usa "worktree", "merge", "rebase" en todas partes — incluso en la UI mockup del Cockpit (§4). Un diseñador contribuyendo docs no sabe qué es un rebase. Un PM revisando un hotfix no debería leer "merge --no-ff".

**Solución: Modo Dev / Modo Estándar**

| Concepto git | Modo Dev (CLI + Cockpit avanzado) | Modo Estándar (Cockpit default) |
|---|---|---|
| Worktree | Worktree | Espacio de trabajo |
| Branch | Branch | Rama |
| Merge | Merge | Integrar |
| Rebase | Rebase | Actualizar desde base |
| Conflict | Conflicto | Conflicto — mantener (es universal) |
| Cherry-pick | Cherry-pick | Traer cambio específico |
| Stash | Stash | Guardar temporalmente |
| HEAD | HEAD | Último cambio |
| Push | Push | Publicar |
| Fetch | Fetch | Sincronizar |

**Reglas de implementación:**

1. **Cockpit default → Modo Estándar.** Cada término técnico tiene un `?` tooltip que muestra el equivalente git: "Actualizar desde base (git rebase)".
2. **Toggle `Ctrl+T` → Modo Dev.** Invierte toda la UI a terminología técnica. Persiste en preferencias.
3. **CLI siempre usa términos git.** El CLI es para devs. Sin abstracción.
4. **Estados de worktree también se traducen:** ACTIVE → "Activo", DIRTY → "Requiere atención", VERIFIED → "Verificado", CLEANED → "Completado".

**Ejemplo visual — Modo Estándar:**
```
┌──────────────────────────────────────────────────────┐
│ Espacios de trabajo                          [Modo Dev]│
│                                                      │
│ 🟢 task/refactor-db   Activo      hace 3h            │
│    ⚠️ Conflicto potencial con "add-cache"             │
│    Ambos modifican: internal/gitflow/workflow.go      │
│                                                      │
│ 🟡 task/add-cache     Requiere atención   hace 2h    │
│    3 archivos en conflicto — resolver para continuar  │
│                                                      │
│ 🟢 task/fix-login     Verificado ✅   hace 1h         │
│    Listo para integrar        [Integrar] [Abandonar]  │
└──────────────────────────────────────────────────────┘
```

**Decisión:** Cockpit arranca en Modo Estándar con terminología humana. Toggle `Ctrl+T` para Modo Dev. CLI sin cambios.

---

## P3: ¿`owc` sin argumentos → Cockpit o CLI interactivo?

**Respuesta:** Cockpit. Con escape rápido a CLI interactivo.

**Razonamiento:**

El spec ya propone Cockpit. Es correcto. Pero falta el detalle de UX:

1. **`owc` → abre Cockpit en vista "Nuevo espacio de trabajo".** El selector de perfil debe ser visual: tarjetas con icono, descripción corta y base branch. No un menú desplegable genérico.

```
┌──────────────────────────────────────────────────┐
│ Nuevo espacio de trabajo              [ESC] salir │
│                                                  │
│ ¿Qué tipo de trabajo vas a hacer?                │
│                                                  │
│ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│ │ ✨       │ │ 🔧       │ │ 🚨       │          │
│ │ Feature  │ │ Refactor │ │ Hotfix   │          │
│ │ develop  │ │ develop  │ │ main     │          │
│ │ Estándar │ │ Estándar │ │ Estricto  │          │
│ └──────────┘ └──────────┘ └──────────┘          │
│ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│ │ 📝       │ │ 🧪       │ │ 📦       │          │
│ │ Docs     │ │ Spike    │ │ Release  │          │
│ │ develop  │ │ develop  │ │ main     │          │
│ │ Relajado │ │ Relajado │ │ Estricto  │          │
│ └──────────┘ └──────────┘ └──────────┘          │
│                                                  │
│ Nombre: [________________]  [Crear] [Cancelar]   │
│                                                  │
│ Sugerencia: feature/login    ↲                    │
└──────────────────────────────────────────────────┘
```

2. **`owc feature/login` → fast path, sin Cockpit.** El dev ya sabe qué quiere. Creación inmediata con feedback: "✅ Espacio task/feature-login creado (perfil: feature, base: develop)".
3. **`owc --ask` → CLI interactivo paso a paso.** Para contextos sin TUI (SSH, CI, scripts): "¿Perfil? [feature] ¿Nombre? login". Mismo resultado, sin dependencia visual.

**Los 3 caminos:**

| Camino | Gatillo | Para quién |
|--------|---------|------------|
| Cockpit visual | `owc` | Usuario estándar, primer uso, exploración |
| Fast path | `owc feature/login` | Dev que sabe exactamente qué crear |
| CLI interactivo | `owc --ask` | SSH, CI/CD, entornos sin TUI |

**Decisión:** `owc` → Cockpit. `owc <perfil>/<nombre>` → instantáneo. `owc --ask` → CLI paso a paso.

---

## P4: Side-by-side diff: ¿Horizontal (IDE) o Vertical (GitHub)?

**Respuesta:** Horizontal como default. Vertical como toggle. La decisión depende del contexto.

**Análisis comparativo:**

| Criterio | Horizontal (IDE) | Vertical (GitHub unificado) |
|---|---|---|
| Comprensión de cambios independientes | ✅ Superior — cada rama se lee por separado | ❌ Mezcla ambas en un solo flujo |
| Resolución rápida de conflictos simples | ❌ Overkill visual | ✅ Más rápido, patrón conocido |
| AI-assisted resolution (§13.2) | ✅ La IA explica "Rama A hizo X, Rama B hizo Y" lado a lado | ❌ Difícil mostrar razonamiento AI |
| Familiaridad | Medio — usuarios de VS Code, JetBrains | Alta — todo GitHub/GitLab |
| "Wow factor" premium | ✅ Visualmente sofisticado, diferencial | ❌ Genérico, esperable |
| Espacio en pantalla | Requiere 120+ columnas | Funciona en 80 columnas |

**Recomendación UX:**

1. **Default: Horizontal.** Es más premium, mejor para AI resolution, y diferencial frente a competidores. Layout: izquierda (rama A / base), derecha (rama B / cambios del usuario), centro (opciones de resolución).
2. **Toggle `V` → Vertical unificado.** Para usuarios que prefieren el patrón GitHub. Persiste como preferencia.
3. **Detección automática de ancho de terminal:** Si < 100 columnas → fuerza Vertical automáticamente con aviso: "Pantalla estrecha — cambiado a vista unificada".

```
┌──────────────────────────────────────────────────────────┐
│ Resolución: internal/gitflow/workflow.go    [H]oriz [V]ert│
│                                                          │
│ ┌──────────── Rama A (develop) ──┐ ┌── Rama B (tuya) ──┐ │
│ │ 142: func Merge(branch string) │ │ 142: func Merge(b  │ │
│ │ 143:   // cleanup worktree     │ │ 143:   // cleanup   │ │
│ │ 144:   worktree.Remove(path)   │ │ 144:   worktree.Rm  │ │
│ │ 145:                           │ │ 145:                │ │
│ │ 146: func Start() {            │ │ 146: func Start(pro │ │
│ │ 147:   // basic start          │ │ 147:   // profile su │ │
│ └────────────────────────────────┘ └────────────────────┘ │
│                                                          │
│ 💡 AI: Merge() y Start() son independientes. Sin conflicto│
│     lógico — solo líneas adyacentes.                     │
│                                                          │
│     [Aceptar resolución]  [Editar manualmente]  [Rechazar]│
└──────────────────────────────────────────────────────────┘
```

**Decisión:** Horizontal default, toggle `V` para unificado, auto-switch a Vertical en <100 cols. El diff horizontal es el "wow factor" visual del Cockpit.

---

## P5: Offline-first UX — ¿Cómo comunicarlo como ventaja premium?

**Respuesta:** No decir "offline". Decir "autónomo". Y demostrarlo, no explicarlo.

**Estrategia de mensaje:**

| ❌ No decir | ✅ Decir |
|---|---|
| "Funciona sin internet" | "Tu flujo, sin dependencias" |
| "Modo offline" | "Modo autónomo" |
| "Se sincronizará cuando vuelvas" | "3 cambios listos para publicar" |
| "Sin conexión" (rojo, alarma) | "Autónomo" (ícono de avión ✈️, azul sereno) |

**Implementación en Cockpit — 3 momentos clave:**

### Momento 1: Indicador de conectividad (siempre visible, nunca alarmante)

```
Barra de estado Cockpit:
┌──────────────────────────────────────────────────────────┐
│ OVAV Cockpit v2.1  │  task/refactor-db  │  ✈️ Autónomo   │
└──────────────────────────────────────────────────────────┘
```

Reglas:
- **Conectado:** Sin indicador. Es el estado normal, no merece atención.
- **Autónomo:** Ícono ✈️ azul tenue + "Autónomo". Sin colores de advertencia.
- **Recién reconectado:** ✈️ → 🔗 animación sutil (500ms) + badge: "3 ops pendientes".

### Momento 2: Al intentar `owd` sin conexión

```
┌──────────────────────────────────────────────────────────┐
│ 🚀 ¿Integrar "task/add-cache" a develop?                 │
│                                                          │
│ ⚠️ Estás en modo autónomo. El merge se completará        │
│    automáticamente al reconectar.                        │
│                                                          │
│ ✅ Verificación local: completada (6/6 checks)            │
│ ⏳ Merge + push: pendiente (se ejecutará al reconectar)   │
│                                                          │
│     [Integrar ahora (local)]   [Cancelar]                 │
└──────────────────────────────────────────────────────────┘
```

Lenguaje clave: "se completará automáticamente" — transmite confianza, no incertidumbre.

### Momento 3: Reconexión (el momento "delight")

```
┌──────────────────────────────────────────────────────────┐
│ 🔗 Conectividad restaurada                               │
│                                                          │
│ ✅ #45 Actualizar task/refactor-db     — completado       │
│ ✅ #46 Integrar task/fix-login         — mergeado a dev   │
│ ✅ #47 Integrar task/add-cache         — mergeado a dev   │
│                                                          │
│ 📊 Todo listo. 3 operaciones completadas sin fricción.    │
│                                                          │
│                                       [Ver detalles] [OK] │
└──────────────────────────────────────────────────────────┘
```

**Tagline para el producto:** _"OWS funciona donde trabajás. En la oficina, en un avión, en una montaña. Sin excusas."_

**Decisión:** "Modo autónomo", nunca "offline". Indicador sutil ✈️. Reconexión con animación delight. El mensaje: independencia es premium.

---

## P6: Métrica de éxito — ¿Qué define que la vista Worktrees es exitosa?

**Respuesta:** Una métrica North Star + dos métricas de calidad.

### North Star: **Tasa de merge al primer intento (FMA — First-attempt Merge Acceptance)**

```
FMA = merges exitosos en el primer owd / total de owd ejecutados

Donde "exitoso" = owd completa sin conflictos, sin fallos de verify, sin bloqueos de policy.
```

**Por qué esta métrica:** Captura toda la propuesta de valor del spec. Las 8 capas invisibles existen para que los 2 comandos visibles (`owc`, `owd`) funcionen sin fricción. Si el usuario hace `owd` y falla, algo de las 8 capas falló. Si hace `owd` y funciona, todas las capas hicieron su trabajo invisible.

**Target:** >85% FMA en los primeros 30 días. >92% a los 90 días (el sistema aprende con AI resolution).

### Métricas de calidad (secundarias):

| Métrica | Qué mide | Target |
|---------|----------|--------|
| **TTM (Time to Merge)** | Minutos desde `owc` hasta `owd` completado | <-- No es target, es diagnóstico. Si baja, el sistema mejora. |
| **Conflict Rate** | % de worktrees que entran en estado DIRTY | <15% (cuanto más bajo, mejor predicción) |
| **DAU Worktrees** | % de sesiones Cockpit que incluyen vista Worktrees | >60% |
| **AI Acceptance Rate** | % de resoluciones AI que el usuario acepta sin editar | >70% |
| **Abandon Rate** | % de worktrees que nunca llegan a CLEANED | <10% |

### Cómo medir (ya tenés la infraestructura):

Todo está en `audit_log` y `worktree_state` de SQLite. Query de FMA:

```sql
SELECT 
    COUNT(CASE WHEN result = 'success' THEN 1 END) * 100.0 / COUNT(*) as fma_pct
FROM audit_log 
WHERE command = 'owd' 
  AND timestamp > date('now', '-30 days');
```

**Decisión:** FMA (First-attempt Merge Acceptance) es la métrica North Star. Target >85%. Se mide desde SQLite sin instrumentación adicional. Las 5 métricas secundarias dan diagnóstico cuando FMA baja.

---

## Recomendaciones adicionales de UX (no solicitadas, pero necesarias)

### A. Vista de conflicto potencial (spec §13.1) necesita jerarquía visual

Actualmente §13.1 muestra texto plano. El Cockpit debe usar color y posición:

```
┌──────────────────────────────────────────────────────────┐
│ Espacios de trabajo                                      │
│                                                          │
│ ⚠️ Conflicto detectado — 2 worktrees comparten archivos   │
│                                                          │
│ ┌─────────────────────┐   ┌─────────────────────┐        │
│ │ task/refactor-db    │   │ task/add-cache      │        │
│ │ 🟡 12 archivos      │   │ 🟡 3 archivos       │        │
│ │ Dante · hace 3h     │   │ Sergio · hace 2h    │        │
│ │                     │   │                     │        │
│ │ Archivo compartido: │   │ Archivo compartido: │        │
│ │ workflow.go         │◄──┼► workflow.go        │        │
│ └─────────────────────┘   └─────────────────────┘        │
│                                                          │
│ 💡 Sugerencia: integrar "add-cache" primero (3 archivos   │
│    vs 12). Menos superficie de conflicto.                 │
│                                                          │
│              [Integrar add-cache primero]  [Ignorar]      │
└──────────────────────────────────────────────────────────┘
```

### B. El `owd` debe ser un momento de confianza, no de ansiedad

El spec dice que `owd` "abre Cockpit (diff preview + confirmación)". Ese momento es crítico: el usuario está a punto de mergear su trabajo. La UI debe transmitir seguridad:

1. **Checklist animada** antes del botón de confirmación:
   ```
   ✅ Verificación (6/6)      ← completado
   ✅ Sin conflictos           ← completado
   ✅ Políticas (8/8)          ← completado
   ⏳ Integrar a develop       ← siguiente paso
   ```

2. **Preview del diff** con estadísticas claras: "+47 líneas, -12 líneas, 3 archivos".

3. **Botón de confirmación con micro-copy tranquilizador:** "Integrar a develop" (no "Merge" ni "Confirmar"). Con tooltip: "Tus cambios se integrarán a la rama principal. Podés deshacer con `owr` si es necesario."

### C. Empty state del Cockpit Worktrees

Si un usuario nuevo abre Worktrees y no tiene ninguna, el spec no define qué ve. Propuesta:

```
┌──────────────────────────────────────────────────────────┐
│ Espacios de trabajo                                      │
│                                                          │
│                     🚀                                   │
│         Creá tu primer espacio de trabajo                │
│                                                          │
│   Un espacio de trabajo es tu área aislada para          │
│   desarrollar una feature, fix, o experimento.           │
│   Sin afectar a otros. Sin sorpresas.                    │
│                                                          │
│              [Crear espacio]  (owc)                      │
│                                                          │
│   O desde terminal: owc feature/mi-primer-feature        │
└──────────────────────────────────────────────────────────┘
```

---

## Verificación de consistencia con §16 (preguntas originales del spec)

Las 6 preguntas de esta review cubren las 5 preguntas originales del spec §16. Correspondencia:

| §16 pregunta | Cubierta por | Decisión |
|---|---|---|
| ¿Worktrees como default? | P1 | Dashboard default + memoria de última vista |
| ¿Abstraer para no-dev? | P2 | Modo Estándar (default) + Modo Dev (Ctrl+T) |
| ¿`owc` sin args: Cockpit o CLI? | P3 | Cockpit default, `owc --ask` para CLI |
| ¿Side-by-side horizontal o vertical? | P4 | Horizontal default, toggle V |
| ¿Cómo comunicar offline-first? | P5 | "Modo autónomo", jamás "offline" |

---

## Conclusión

El spec está listo para implementación con estas 6 decisiones de UX aplicadas en Fase 5 (Cockpit). Las 8 capas invisibles del spec son el músculo. Las 6 capas de UX de esta review son la piel. Sin ambas, no hay producto premium.

**Próximo paso:** Thavren, incorporá estas decisiones en el diseño del Cockpit (Fase 5). Elena (UX/UI) debería revisar los mockups antes de implementar. Cualquier duda sobre la interacción visual, me la pasás a mí o a Elena.

---

*Dante — Digital Product Engineering Lead*  
*OVAV Governor System*
