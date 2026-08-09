# OVAV × MiMoCode — Absorción de inteligencia + Browser MCP
# Plan de Arquitectura v1 — 2026-07-04
#
# Author: Thavren (Platform Engineering Lead)
# Status: Análisis completado. Listo para implementación.

## Parte 1: Absorción de MiMoCode Compose Pipeline

MiMoCode tiene 14 skills de desarrollo autónomo (compose pipeline). OVAV los absorbe
mapeándolos a sus service areas:

| Compose Skill | OVAV Service Area | ¿Quién lo ejecuta? |
|---|---|---|
| brainstorm | Digital Product (Dante) + UX Design (Elena) | Leads definen specs |
| plan | Platform Engineering (Thavren) | Thavren aprueba planes |
| tdd (test-first) | QA Engineer (Clara) | Clara ejecuta tests |
| execute | Implementadores (Andrés, Lucas) | Squad implementa |
| verify | QA + Code Review (Pablo) | Verificación dual |
| review | Code Review (Pablo) + Security (Diana) | Revisión pre-merge |
| report | Platform Engineering | Thavren recibe reporte |
| merge | Platform Engineering | Gobernado por gates |
| feedback | Todos los leads | Cross-area feedback |
| debug | Explorers (Helena, Irene) | Diagnóstico |
| parallel | Cualquier lead | Worktree isolation |
| ask | Cualquier agente | CEO decision gate |
| new-skill | Platform Engineering | Thavren crea skills |
| worktree | Platform Engineering | Git worktree isolation |

### Cómo se absorbe

Cada lead ya tiene su AGENTS.md con funciones. Los skills de compose se INYECTAN
como hooks en el system prompt de cada lead:

1. `experimental.chat.system.transform` hook → añade instrucciones compose al prompt
2. Cada lead recibe solo los skills de su área (no todos los 14)
3. La orquestación cross-area usa `ovav-agent-router` skill

## Parte 2: Browser MCP — Edición en vivo como Cursor

### Arquitectura

```
OVAV Agent → MCP Protocol (stdio JSON-RPC) → Browser MCP Server → Playwright → Browser
                                                           ↓
                                                    WebSocket stream
                                                    (edits en vivo)
```

### Stack

```yaml
mcp_server: ovav-browser
language: Python 3.12 (usa Playwright bindings)
protocol: MCP JSON-RPC 2.0 over stdio
runtime: Playwright (Chromium)
tools:
  - browser_navigate: Navegar a URL
  - browser_screenshot: Captura de pantalla
  - browser_click: Click en elemento (selector CSS)
  - browser_type: Escribir en input
  - browser_evaluate: Ejecutar JS en página
  - browser_get_html: Obtener HTML de elemento
  - browser_get_styles: Obtener estilos computados
  - browser_watch: WebSocket stream de cambios (hot reload)
```

### Implementación — 3 fases

| Fase | Qué | Tiempo | Archivos |
|---|---|---|---|
| F1 | MCP server base (navigate, screenshot, click, type) | 1-2h | tools/mcp/ovav_browser_server.py |
| F2 | Hot reload + CSS live edit | 2-3h | + websocket stream |
| F3 | Vite HMR integration — detecta cambios en build y recarga | 1h | + file watcher |

### Cómo el agente lo usa

```
Dante: "Arreglá el footer que está a la izquierda"
→ browser_navigate("http://localhost:5173")
→ browser_screenshot("#footer")
→ [ve que el footer está mal posicionado]
→ edit: Footer.css → justify-content: flex-end
→ browser_watch → [recarga automática]
→ browser_screenshot("#footer")
→ [confirma visualmente]
→ commit
```

## Parte 3: Adaptaciones a nivel de agentes y CLI

### 3A. Agentes — system prompt injection via hooks

OVAV ya tiene `ovav-security.js` usando `tool.execute.before` hooks.
Extendemos el patrón para:

```javascript
// .mimocode/hooks/ovav-intelligence.js
export default {
  // 1. Inyecta compose pipeline en leads según su área
  "experimental.chat.system.transform": async (input, output) => {
    const agent = output.system[0]; // nombre del agente
    const area = lookupArea(agent); // mapea agente → área
    const composeInstructions = getComposeFor(area); // skills relevantes
    output.system.push(composeInstructions);
  },

  // 2. Inyecta contexto cross-area cuando detecta intención
  "chat.params": async (input, output) => {
    if (input.messages[0]?.content?.includes("@dante")) {
      output.maxOutputTokens = 4096; // más tokens para multi-paso
    }
  },

  // 3. Bloquea herramientas no autorizadas por área
  "tool.execute.before": async (input, output) => {
    if (input.agent === "Dante" && input.tool === "webfetch") {
      output.cancel = false; // allow para leads
    }
  },
};
```

### 3B. CLI — nuevos comandos

```bash
ovav browser start     # Levanta MCP browser server + Playwright
ovav browser stop      # Detiene browser + cleanup
ovav compose plan <area>  # Inicia pipeline compose para un área
ovav compose status    # Estado del pipeline actual
ovav mcp list          # Lista MCP servers disponibles
ovav mcp start <name>  # Inicia un MCP server específico
ovav hook reload       # Recarga hooks sin reiniciar sesión
```

### 3C. Config — nuevo MCP en source config

```yaml
# .ovav/source/opencode/config.yaml → añadir:
mcp:
  ovav-budget:
    type: local
    command: [python3, tools/mcp/ovav_mcp_server.py, budget]
    enabled: true
  ovav-browser:                                    # NUEVO
    type: local
    command: [python3, tools/mcp/ovav_browser_server.py]
    enabled: true
    env:
      BROWSER_PORT: "9222"
      HEADLESS: "false"
```

## Resumen ejecutivo

| Entrega | Tipo | Impacto | Prioridad |
|---|---|---|---|
| Absorber compose skills → service areas | Hook injection | Leads + squads ganan pipeline autónomo | 🔴 ALTA |
| Browser MCP (F1: básico) | MCP Server | Agentes ven y editan browser en vivo | 🔴 ALTA |
| Browser MCP (F2: hot reload) | WebSocket | Ediciones CSS/HTML con feedback visual inmediato | 🟡 MEDIA |
| CLI: `ovav browser`, `ovav compose` | Go CLI | Operadores controlan browser desde terminal | 🟡 MEDIA |
| Hook: system prompt injection | JS Hook | Cada lead recibe solo sus skills relevantes | 🟢 BAJA |
