# PIAGENT INPUT — Mockups Visuales

## MOCKUP v1.0.0
## Fecha: 2026-08-07
## Diseñador: Elena (UX Design) + Thavren (Platform Engineering)

---

# Estado Actual vs Futuro

## 🔴 ANTES (Estado Actual)

```
╔══════════════════════════════════════════════════════════════════════════════╗
║ OVAV v2.1.0 — PIAGENT HARNESS                              [████████░░] 80%║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  [Assistant]                                                                 ║
║  Sesión: ecommerce-platform                    Modelo: claude-sonnet-4         ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║                                                                              ║
║  > Implementa el carrito de compras                                           ║
║                                                                              ║
║                                                                              ║
║                                                                              ║
║                                                                              ║
║                                                                              ║
║                                                                              ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║  > _                                                                        ║
║  ▌                                                                        ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

**Problemas:**
- Input sin borde ni distinción
- Sin affordances visuales
- Sin contexto del proyecto
- Sin autocomplete
- Apariencia de bloc de notas

---

## 🟡 FASE 1: Quick Wins (1-2 días)

### Cambios Visibles:
1. Theme mejorado con bordes y colores
2. Autocomplete de `/` commands
3. Status bar informativa

```
╔══════════════════════════════════════════════════════════════════════════════╗
║ 🌐 OVAV v2.1.0 — PIAGENT                              [████████░░] 80%  🔴║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  [Assistant]                                                                 ║
║  📁 ecommerce-platform    🤖 claude-sonnet-4    💭 high    🔒 secured          ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║                                                                              ║
║  > Implementa el carrito de compras                                           ║
║                                                                              ║
║  ✓ Carrito básico implementado                                               ║
║  ✓ Hooks de estado agregados                                                 ║
║                                                                              ║
║                                                                              ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║  ┃ > Implementa el checkout_                                                  ║
║  ┃   ├─ /review ─── Code review completo                                     ║
║  ┃   ├─ /component ── Crear componente React                                  ║
║  ┃   ├─ /test ────── Generar tests                                           ║
║  ┃   └─ /deploy ──── Deploy a staging                                        ║
║  ▌                                                                        ║
╚══════════════════════════════════════════════════════════════════════════════╝
                          ↑ autocomplete aparece al escribir /
```

**Mejoras:**
- ✅ Borde del INPUT visible
- ✅ Autocomplete con `/` commands
- ✅ Status bar con contexto
- ✅ Iconos informativos

---

## 🟢 FASE 2: Medium Impact (3-5 días)

### Cambios Visibles:
1. Command Palette (`/cmd`)
2. Context Hints
3. Overlay de shortcuts

### 2.1 Command Palette

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║  ┌─ Command Palette ──────────────────────────────────────────────────┐    ║
║  │ 🔍 _                                                        Esc=close│    ║
║  ├───────────────────────────────────────────────────────────────────────┤    ║
║  │                                                                         │    ║
║  │  ↵  /review          Code review completo                              │    ║
║  │     /component       Crear componente React                              │    ║
║  │     /test            Generar tests unitarios                           │    ║
║  │     /deploy          Deploy a staging                                   │    ║
║  │     /compact         Compactar contexto                                 │    ║
║  │     /new             Nueva sesión                                       │    ║
║  │     /skill:pdf       Herramienta PDF                                   │    ║
║  │     /skill:search    Búsqueda web                                       │    ║
║  │                                                                         │    ║
║  ├───────────────────────────────────────────────────────────────────────┤    ║
║  │ ↑↓ navegar   ↵ seleccionar   Tab=complete   Esc=cerrar                 │    ║
║  └───────────────────────────────────────────────────────────────────────┘    ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║  ┃ > _                                                                        ║
║  ▌                                                                        ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

### 2.2 Context Hints (Sidebar)

```
╔════════════════════════════╗╔═══════════════════════════════════════════════════════════╗
║ 📋 CONTEXTO                 ║║                                                                 ║
╠════════════════════════════╣║                                                                 ║
║                            ║║  > Implementa el checkout_                                 ║
║ 🔄 Recientes               ║║    ├─ /review ─── Code review completo                      ║
║  • checkout.ts             ║║    ├─ /component ── Crear componente                        ║
║  • cart.ts                 ║║    └─ /test ────── Generar tests                            ║
║  • api.ts                  ║║                                                                 ║
║                            ║║                                                                 ║
║ 📝 Archivos                ║║                                                                 ║
║  M config.yaml             ║║                                                                 ║
║  M src/cart/              ║║                                                                 ║
║  ? src/checkout/          ║║                                                                 ║
║                            ║║                                                                 ║
║ ⚡ Quick Actions           ║║                                                                 ║
║  /git status              ║║                                                                 ║
║  /ovav daily              ║║                                                                 ║
║  /validate                ║║                                                                 ║
║                            ║║                                                                 ║
╠════════════════════════════╣║                                                                 ║
║ ⌨️ SHORTCUTS              ║║  ─────────────────────────────────────────────────────────   ║
║ Ctrl+P = modelo           ║║  ┃ > _                                                        ║
║ Ctrl+G = git              ║║  ▌                                                        ║
║ Ctrl+L = compact          ║║                                                                 ║
║ ? = ayuda                 ║║                                                                 ║
╚════════════════════════════╝╚═══════════════════════════════════════════════════════════╝
```

### 2.3 Shortcuts Overlay

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║                    ┌─ Atajos de Teclado ──────────────────────────────┐      ║
║                    │                                                          │      ║
║                    │  ⌨️ NAVEGACIÓN                   ⌨️ COMANDOS          │      ║
║                    │  ─────────────────────────────  ───────────────     │      ║
║                    │  Ctrl+P  ─── Cambiar modelo      /review  ── Code   │      ║
║                    │  Ctrl+G  ─── Git status         /component── Compon  │      ║
║                    │  Ctrl+L  ─── Compactar           /test  ──── Tests   │      ║
║                    │  Ctrl+H  ─── Help               /cmd  ──── Palette   │      ║
║                    │  ↑↓     ─── Historial           /skill ─── Skills   │      ║
║                    │                                                          │      ║
║                    │  🔧 EXTENSIONES OVAV                                │      ║
║                    │  ───────────────────────────────────────────────────  │      ║
║                    │  /ovav daily  ─── Estado del sistema                 │      ║
║                    │  /ovav next   ─── Siguiente tarea                    │      ║
║                    │  /validate   ─── Validar workspace                    │      ║
║                    │                                                          │      ║
║                    └─────────────────────── Esc=cerrar ──────────────────┘      ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║  ┃ > _                                                                        ║
║  ▌                                                                        ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

---

## 🔵 FASE 3: High Impact (1-2 semanas)

### Cambios Visibles:
1. INPUT Premium con frame profesional
2. Inline suggestions
3. Multi-line editing
4. Persistent context

### 3.1 INPUT Premium Frame

```
╔══════════════════════════════════════════════════════════════════════════════╗
║ 🌐 OVAV v2.1.0 — PIAGENT                              [████████░░] 80%  🔴║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  📁 ecommerce-platform  🤖 claude-sonnet-4  💭 high  🔒 OVAV 2.1  ⏱️ 14:32 ║
║                                                                              ║
║  ┌────────────────────────────────────────────────────────────────────────┐   ║
║  │  💬 Tu mensaje ────────────────────────────────────────────────────  │   ║
║  │                                                                        │   ║
║  │  > Implementa el checkout con Stripe                                     │   ║
║  │    ↳ Incluye validación de tarjetas y manejo de errores                │   ║
║  │    ↳ Usa los componentes del design system                              │   ║
║  │                                                                        │   ║
║  │  💡 Sugerencias:                                                       │   ║
║  │    • /review para código existente                                     │   ║
║  │    • /component para nuevos componentes                                 │   ║
║  │                                                                        │   ║
║  │  📎 Archivos: checkout.ts (modificado)  payment.ts (nuevo)            │   ║
║  │                                                                        │   ║
║  │  ┌──────────────────────────────────────────────────────────────────┐ │   ║
║  │  │ > _                                                               │ │   ║
║  │  └──────────────────────────────────────────────────────────────────┘ │   ║
║  │                                                                        │   ║
║  │  ↵ Enter=enviar   Tab=siguiente sugerencia   Ctrl+Space=completar    │   ║
║  └────────────────────────────────────────────────────────────────────────┘   ║
║                                                                              ║
║  ─────────────────────────────────────────────────────────────────────────   ║
║  │ 💬 checkout.ts │ stripe.ts │ payment.ts │ ✓ 3 archivos en contexto │   │   ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

### 3.2 Multi-line Editing

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                                                                              ║
║  ┌────────────────────────────────────────────────────────────────────────┐   ║
║  │  💬 Nueva tarea ────────────────────────────────────────────────────  │   ║
║  │                                                                        │   ║
║  │  ┌──────────────────────────────────────────────────────────────────┐ │   ║
║  │  │ Implementa el módulo de autenticación:                            │ │   ║
║  │  │                                                                · │ │   ║
║  │  │ • Login con email y password                                     │ │   ║
║  │  │ • Recuperación de contraseña                                     │ │   ║
║  │  │ • Tokens JWT con refresh token                                   │ │   ║
║  │  │ • Middleware de protección de rutas                              │ │   ║
║  │  │                                                                · │ │   ║
║  │  │                                                                · │ │   ║
║  │  └──────────────────────────────────────────────────────────────────┘ │   ║
║  │                                                                        │   ║
║  │  Líneas: 6  │  Caracteres: 245  │  ↵ Enviar  │  Ctrl+Enter=nueva línea│   ║
║  └────────────────────────────────────────────────────────────────────────┘   ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

### 3.3 Split View (Optional)

```
╔══════════════════════════════════════════════════════════════════════════════╗
║ 🌐 OVAV — Split View                                          [📐] [❌]    ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  ┌──────────────────────────────┬───────────────────────────────────────┐   ║
║  │  📝 INPUT PANEL             │  📊 CONTEXT PANEL                     │   ║
║  │                              │                                        │   ║
║  │  > Implementa checkout_       │  🔄 Archivos recientes                 │   ║
║  │    ├─ /review ────          │  • checkout.ts (editado)               │   ║
║  │    └─ /test ────────        │  • cart.ts (leído)                    │   ║
║  │                              │  • api.ts (referenciado)               │   ║
║  │                              │                                        │   ║
║  │  ┌────────────────────────┐ │  📋 Memorias OVAV                     │   ║
║  │  │ > _                   │ │  • Última sesión: ecommerce             │   ║
║  │  └────────────────────────┘ │  • Decisiones: 3 validadas            │   ║
║  │                              │  • Skills: pdf, search, code-review    │   ║
║  │                              │                                        │   ║
║  │                              │  ⚡ Quick Actions                      │   ║
║  │                              │  /ovav daily                          │   ║
║  │                              │  /git diff                            │   ║
║  │                              │  /validate                            │   ║
║  ├──────────────────────────────┴───────────────────────────────────────┤   ║
║  │  ↵ Enviar  │  Ctrl+S=split  │  Ctrl+X=close  │  ?=help                │   ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

---

# Resumen de Impacto Visual

## ✅ IMPLEMENTADO — Estado Real

| Fase | Feature | Status | Implementado |
|------|---------|--------|-------------|
| **Fase 1** | Status Bar Enhanced | ✅ | `ctx.ui.setStatus()` con proyecto, modelo, thinking |
| **Fase 1** | Autocomplete on `/` | ✅ | Widget `ovav-autocomplete` |
| **Fase 1** | Working Indicator | ✅ | Animación en agent processing |
| **Fase 2** | Command Palette | ✅ | `/cmd` widget con categorías |
| **Fase 2** | Shortcuts Overlay | ✅ | `?` muestra atajos |
| **Fase 2** | Context Detection | ✅ | Auto-detecta proyecto, archivos, git |
| **Fase 3** | Context Overlay | ✅ | Muestra al inicio, auto-hide 8s |
| **Fase 3** | Project Awareness | ✅ | Tools `ovav_context`, `ovav_commands` |

| Fase | Aspecto | Antes | Después |
|------|---------|-------|---------|
| **Fase 1** | INPUT | Primitivo | Borde visible |
| **Fase 1** | Autocomplete | ❌ | ✅ `/` commands |
| **Fase 1** | Status | Mínimo | Informativo |
| **Fase 2** | Commands | Solo texto | Command Palette |
| **Fase 2** | Contexto | Invisible | Sidebar hints |
| **Fase 2** | Shortcuts | Desconocidos | Overlay `?` |
| **Fase 3** | INPUT | 2 líneas | Premium frame |
| **Fase 3** | Sugerencias | ❌ | Inline hints |
| **Fase 3** | Multiline | ❌ | ✅ Soportado |
| **Fase 3** | Split | ❌ | ✅ Opcional |

---

# WCAG 2.1 AA Compliance

Todos los mockups cumplen:

| Criterio | Implementación |
|----------|---------------|
| 1.4.3 Contraste | Ratio mínimo 4.5:1 |
| 1.4.11 No color solo | Iconos + texto siempre |
| 2.1.1 Teclado | Todos accesibles por teclado |
| 2.4.7 Focus visible | Cursor y focus claros |
| 3.3.2 Labels | Etiquetas claras en inputs |

---

*Diseñado por Elena (UX Design Lead) + Thavren (Platform Engineering)*
*OVAV Governor System v2.1.0*
