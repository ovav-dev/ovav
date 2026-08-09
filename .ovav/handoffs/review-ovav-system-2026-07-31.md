---
SERVICE_AREA: platform_engineering
VISIBLE_PROFILE: Thavren
LEAD: thavren
TASK_CLASS: system_review
RISK_LEVEL: low
DELEGATION_MODE: full_squad
TRACE_ID: review-ovav-system-2026-07-31
---

# Handoff: Revisión completa OVAV SYSTEM

## Contexto
Este es un handoff de emergencia para revisión completa del sistema OVAV.
Limpiar la acumulación de errores y asegurar que la memoria local persiste.

## tareas a realizar

### 1. Platform Engineering (Thavren)
- Verificar `go-runtime/cmd/memory-mcp/main.go` existe y compila
- Hacer `git rm go-runtime/cmd/cpanel/memory_mcp.go` (archivo contaminado creado por error)
- Verificar estado del vault: `.ovav/vault/`
- Rebuild `bin/ovav-mcp-memory` desde source limpio
- Corregir MEMORY.md líneas 70-76 (info errónea del relay cPanel)

### 2. Research Intelligence (Eidren)
- Hacer `go run ./cmd/ovav/ validate` y reportar estado
- Revisar `.ovav/plan/caps.yaml` para estado actual del proyecto
- Verificar cronología: `go run ./cmd/session_greeting --check-protected`

### 3. Memoria OVAV
- Verificar que `.ovav/runtime/agent_memory.yaml` tiene cards
- `go run ./cmd/session_greeting --json` verificar MemoryBlock
- Asegurar que el post-commit hook `.githooks/post-commit` existe y es ejecutable

## Notas críticas
- **NO crear nuevas rutas en cPanel local** — el cPanel real está en Cloudflare
- **El vault está vacío** — 4 providers sin credenciales (deepseek, openai, anthropic, qwen)
- **MEMORY.md tiene información contaminada** — sección de arquitectura necesita limpieza

## Respuesta esperada
Reporte breve por área: qué está bien, qué está mal, qué necesita修复.
