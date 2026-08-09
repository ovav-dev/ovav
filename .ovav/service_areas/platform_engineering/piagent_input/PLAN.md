# PIAGENT INPUT — Plan de Mejoras UI/UX v2.0

## Estado: EN PROGRESO
## Ultima actualización: 2026-08-08
## Responsable: UI/UX Lead (Delegado)

---

## Resumen Ejecutivo

La extensión `ovav-premium-input` actualmente usa **widgets de string básico** que no aprovechan el sistema de componentes TUI de PI. Este plan documenta la migración a componentes TUI reales para una experiencia premium.

### Problema Actual
```typescript
// ❌ ACTUAL: Widget de string (no interactivo)
ctx.ui.setWidget("ovav-status", renderStatusWidget());
```

### Solución Target
```typescript
// ✅ TARGET: Componente TUI real (SelectList, Container, etc.)
await ctx.ui.custom((tui, theme, keybindings, done) => 
  new CommandPalette({ theme, onSelect: done })
);
```

---

## FASE 1: Migración a Componentes TUI (CRÍTICA)

### 1.1 Command Palette — Migración a SelectList

**Archivo:** `tools/extensions/ovav-premium-input/index.ts`

**Cambios requeridos:**
- Importar `SelectList`, `Container`, `DynamicBorder` de `@earendil-works/pi-tui`
- Usar `ctx.ui.custom()` con overlay para palette interactiva
- Implementar navegación ↑↓ y selección con Enter
- Fuzzy search con `enableSearch: true`

**Mockup objetivo:**
```
┌─ Command Palette ──────────────────────────────────────────────────┐
│ 🔍 _                                                        Esc=close│
├───────────────────────────────────────────────────────────────────────┤
│  ↵  /review          Code review completo                              │
│     /component       Crear componente React                              │
│     /test            Generar tests unitarios                           │
├───────────────────────────────────────────────────────────────────────┤
│ ↑↓ navegar   ↵ seleccionar   Tab=complete   Esc=cerrar                 │
└───────────────────────────────────────────────────────────────────────┘
```

**Implementación:**
```typescript
// Pattern 1: Selection Dialog (SelectList) de docs/tui.md
await ctx.ui.custom<string | null>((tui, theme, _kb, done) => {
  const container = new Container();
  
  // Top border
  container.addChild(new DynamicBorder((s: string) => theme.fg("accent", s)));
  
  // Title
  container.addChild(new Text(theme.fg("accent", theme.bold("Command Palette")), 1, 0));
  
  // SelectList con theme
  const selectList = new SelectList(items, Math.min(items.length, 10), {
    selectedPrefix: (t) => theme.fg("accent", t),
    selectedText: (t) => theme.fg("accent", t),
    description: (t) => theme.fg("muted", t),
    scrollInfo: (t) => theme.fg("dim", t),
    noMatch: (t) => theme.fg("warning", t),
  });
  selectList.onSelect = (item) => done(item.value);
  selectList.onCancel = () => done(null);
  container.addChild(selectList);
  
  // Help text
  container.addChild(new Text(theme.fg("dim", "↑↓ navigate • enter select • esc cancel"), 1, 0));
  
  return {
    render: (w) => container.render(w),
    invalidate: () => container.invalidate(),
    handleInput: (data) => { selectList.handleInput(data); tui.requestRender(); },
  };
}, { overlay: true });
```

### 1.2 Shortcuts Overlay — Migración a Container

**Mockup objetivo:**
```
┌─ Atajos de Teclado ──────────────────────────────┐
│  ⌨️ NAVEGACIÓN                   ⌨️ COMANDOS    │
│  Ctrl+P ─── Cambiar modelo        /review ──    │
│  Ctrl+G ─── Git status           /component──   │
└─────────────────────── Esc=cerrar ─────────────┘
```

**Implementación:**
```typescript
// Usar Container + Text + Spacer
const container = new Container();
container.addChild(new Text(theme.fg("accent", "⌨️ NAVEGACIÓN"), 1, 0));
container.addChild(new Text(theme.fg("muted", "Ctrl+P ─── Cambiar modelo"), 1, 0));
// ...
```

---

## FASE 2: Mejoras de Widgets (ALTA)

### 2.1 Status Bar — Theme-aware

**Mejora:** Usar `theme.fg()` para colores dinámicos según tema activo.

```typescript
function updateStatusBar(ctx: any) {
  const theme = ctx.ui.theme;
  const model = ctx.model?.id?.split("/").pop() || "none";
  const thinking = ctx.thinkingLevel || "off";
  
  // Usar theme para colores
  const status = [
    theme.fg("accent", "🌐"),
    " OVAV",
    theme.fg("muted", "│"),
    theme.fg("text", projectInfo.name),
    theme.fg("muted", "│"),
    theme.fg("success", `🤖 ${model}`),
    theme.fg("muted", "│"),
    theme.fg("warning", `💭 ${thinking}`),
  ].join(" ");
  
  ctx.ui.setStatus("ovav-main", status);
}
```

### 2.2 Working Indicator — Personalizado

```typescript
ctx.ui.setWorkingIndicator({
  frames: [
    theme.fg("accent", "◉"),
    theme.fg("muted", "◎"),
    theme.fg("accent", "◉"),
    theme.fg("muted", "◎"),
  ],
  intervalMs: 200,
});
```

---

## FASE 3: Custom Footer — Información Rica (MEDIA)

### 3.1 Footer con Git Info

```typescript
ctx.ui.setFooter((tui, theme, footerData) => {
  const branch = projectInfo.branch;
  const isWorktree = projectInfo.isWorktree;
  const worktreePath = projectInfo.path;
  const changes = projectInfo.hasChanges ? " ⚠️" : "";
  
  return {
    invalidate() {},
    render(width: number): string[] {
      const worktreeLabel = isWorktree 
        ? `${theme.fg("accent", "🌳")} ${theme.fg("text", "WORKTREE:")} ${theme.fg("muted", branch)}`
        : `${theme.fg("accent", "⌥")} ${theme.fg("text", branch)}`;
      
      const pathText = theme.fg("dim", worktreePath);
      const changesText = changes ? theme.fg("warning", changes) : "";
      
      return [`${worktreeLabel} | ${pathText}${changesText}`];
    },
    dispose: footerData.onBranchChange(() => tui.requestRender()),
  };
});
```

---

## FASE 4: Temas OVAV (MEDIA)

### 4.1 Tema Premium Actualizado

**Ubicación:** `~/.pi/agent/themes/ovav-premium-night.json`

**Tokens adicionales requeridos:**
- `thinkingMax` (opcional)
- `scrollbarThumb` (opcional)

### 4.2 Nuevo Tema: `ovav-elegant-night`

```json
{
  "$schema": "https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/src/modes/interactive/theme/theme-schema.json",
  "name": "ovav-elegant-night",
  "displayName": "OVAV Elegant Night",
  "description": "Premium dark theme with elegant color harmony for 12hr+ sessions",
  "type": "dark",
  "vars": {
    "primary": "#6c8ebf",
    "secondary": "#5a6a7a",
    "accent": "#d4a574",
    "success": "#8fbc8f",
    "warning": "#deb887",
    "error": "#cd9b9b",
    "muted": "#4a5568",
    "text": "#e8e8e8"
  },
  "colors": {
    "accent": "#d4a574",
    "border": "#4a5568",
    "borderAccent": "#6c8ebf",
    "borderMuted": "#3a4a58",
    "success": "#8fbc8f",
    "error": "#cd9b9b",
    "warning": "#deb887",
    "muted": "#5a6a7a",
    "dim": 240,
    "text": "#e8e8e8",
    "thinkingText": "#5a6a7a",
    "selectedBg": "#2d3748",
    "userMessageBg": "#1a202c",
    "userMessageText": "#e8e8e8",
    "customMessageBg": "#242c3c",
    "customMessageText": "#e8e8e8",
    "customMessageLabel": "#d4a574",
    "toolPendingBg": "#1a202c",
    "toolSuccessBg": "#1c2c1c",
    "toolErrorBg": "#2c1c1c",
    "toolTitle": "#6c8ebf",
    "toolOutput": "#c8c8c8",
    "mdHeading": "#d4a574",
    "mdLink": "#6c8ebf",
    "mdLinkUrl": "#5a6a7a",
    "mdCode": "#8fbc8f",
    "mdCodeBlock": "#121820",
    "mdCodeBlockBorder": "#3a4a58",
    "mdQuote": "#5a6a7a",
    "mdQuoteBorder": "#4a5568",
    "mdHr": "#3a4a58",
    "mdListBullet": "#6c8ebf",
    "toolDiffAdded": "#8fbc8f",
    "toolDiffRemoved": "#cd9b9b",
    "toolDiffContext": "#5a6a7a",
    "syntaxComment": "#5a6a7a",
    "syntaxKeyword": "#b4a7d6",
    "syntaxFunction": "#6c8ebf",
    "syntaxVariable": "#d4a574",
    "syntaxString": "#8fbc8f",
    "syntaxNumber": "#d4a574",
    "syntaxType": "#87ceeb",
    "syntaxOperator": "#6c8ebf",
    "syntaxPunctuation": "#c8c8c8",
    "thinkingOff": "#2d3748",
    "thinkingMinimal": "#3d4758",
    "thinkingLow": "#4a5568",
    "thinkingMedium": "#5a6a7a",
    "thinkingHigh": "#b4a7d6",
    "thinkingXhigh": "#cd9b9b",
    "thinkingMax": "#deb887",
    "bashMode": "#deb887"
  },
  "export": {
    "pageBg": "#121820",
    "cardBg": "#1a202c",
    "infoBg": "#2d3748",
    "codeBg": "#121820",
    "borderColor": "#4a5568",
    "textColor": "#e8e8e8",
    "linkColor": "#6c8ebf",
    "headingColor": "#d4a574",
    "mutedColor": "#5a6a7a"
  }
}
```

---

## TAREAS PRIORIZADAS

| # | Tarea | Prioridad | Estado |
|---|-------|-----------|--------|
| 1 | Importar componentes TUI | CRÍTICA | ⬜ |
| 2 | Migrar Command Palette a SelectList | CRÍTICA | ⬜ |
| 3 | Migrar Shortcuts a Container | ALTA | ⬜ |
| 4 | Status Bar theme-aware | ALTA | ⬜ |
| 5 | Working Indicator personalizado | MEDIA | ⬜ |
| 6 | Footer con git info | MEDIA | ⬜ |
| 7 | Crear tema ovav-elegant-night | BAJA | ⬜ |

---

## REFERENCIAS

- **TUI Docs:** `/home/braka/.nvm/versions/node/v22.23.1/lib/node_modules/@earendil-works/pi-coding-agent/docs/tui.md`
- **Theme Docs:** `/home/braka/.nvm/versions/node/v22.23.1/lib/node_modules/@earendil-works/pi-coding-agent/docs/themes.md`
- **Ejemplos:** `/home/braka/.nvm/versions/node/v22.23.1/lib/node_modules/@earendil-works/pi-coding-agent/examples/extensions/`

---

*Plan creado: 2026-08-08*
*Responsable: UI/UX Lead (Delegado)*
