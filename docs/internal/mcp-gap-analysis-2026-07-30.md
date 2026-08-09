# OVAV AGENTS — MCP Gap Analysis & Strategy
## Fecha: 2026-07-30

---

## Hallazgo: OVAV Skills vs MCP Protocol

**Contexto:** 7 de 10 herramientas CLI modernas soporta MCP (Model Context Protocol) como capa de plugins:
- Claude Code ✅
- OpenAI Codex ✅
- Cursor ✅
- GitHub Copilot ✅
- Devin ✅
- Codeium ✅
- Tabnine ✅

**OVAV Status:** Skills son propietarios, NO MCP-compatibles.

---

## Gap Analysis

| Aspect | MCP Standard | OVAV Skills |
|---|---|---|
| Plugin discovery | Automatic via MCP servers | Manual via skill_search |
| Tool invocation | Standardized JSON-RPC | Proprietary prompt injection |
| Auth context | MCP server handles | Skill injects via prompts |
| Lifecycle | Server lifecycle managed by host | Skill loaded per session |
| Registry | Public MCP registry (mcp.so) | OVAV private registry |
| Cross-platform | Works across all MCP hosts | OVAV-only |

---

## Strategic Options

### Option A: OVAV Skill Export as MCP Server
Exportar skills OVAV como MCP servers para que OVAV pueda consumir herramientas MCP de terceros.

**Pros:**
- Acceso a ecosistema MCP (100+ servers públicos)
- Interoperabilidad con Claude Code, Cursor, etc.
- No cambia arquitectura OVAV existente

**Cons:**
- Engineering effort significativo
- MCP server SDK requiere mantenimiento

**Action:** Evaluar `github.com/modelcontextprotocol/server` para Go.

### Option B: OVAV as MCP Server
Exponer skills OVAV como MCP server para que otras herramientas (Claude Code, Cursor) puedan usar OVAV como tool.

**Pros:**
- OVAV se convierte en herramienta reusable
- Compatible con todo el ecosistema

**Cons:**
- Requiere reescribir skills como MCP tools
- Perdería flexibilidad de prompts

**Action:** Prioridad baja — evaluar post-MVP.

### Option C: Keep Proprietary (Status Quo)
Mantener skills propietarios OVAV sin MCP.

**Pros:**
- Sin engineering effort adicional
- Skills pueden ser más complejos que MCP tools simples

**Cons:**
- No acceso a ecosistema MCP
- Vendor lock-in

**Recommendation:** Option A — exportar skills como MCP client en Go runtime.

---

## Implementation Plan (Option A)

1. **Fase 1:** Evaluar `github.com/modelcontextprotocol/spec` + Go SDK
2. **Fase 2:** Crear `internal/mcp/client.go` en go-runtime
3. **Fase 3:** Registrar servers MCP públicos (filesystem, git, web search)
4. **Fase 4:** Integrar con ovav-skill-resolver para auto-detectar MCP tools

---

## Prioridad

**MEDIUM** — No bloquea social-citas MVP. Agenda post-MVP.

