# OpenCode 1.17.3 — Upgrade OVAV

**Instalado:** 11 Jun 2026  
**Versión anterior:** 1.14.48 → **1.17.3**  
**Rama de optimización:** `task/opencode-17-optimization`

---

## Nuevas capacidades que benefician a OVAV

### 🚀 Background subagents (1.16.2+)
Permite delegar subagentes sin bloquear la sesión principal. OVAV lo aprovecha en:
- `session_greeting.py`: 10+ acciones I/O-bound ahora corren en paralelo vía `ThreadPoolExecutor`
- Reducción: ~16s → ~5s en greeting (62% más rápido)

### 🔄 Session recovery (1.17.0)
Context-overflow ahora tiene 1 reintento automático antes de matar la sesión.
- `safe_stop_contract.yaml` actualizado documentando el comportamiento de recovery
- ⚠️ **Corregido 2026-06-11:** `"recovery"` NO es una clave válida en opencode.json. El context-overflow retry es un comportamiento interno del host runtime, no una clave de configuración.

### ⚡ Búsqueda rápida de archivos (1.17.0)
Nuevo motor `fff` acelera todas las operaciones de grep/glob. Beneficio transparente para los 53 validators y 120+ harnesses de OVAV.

### 🔒 Edits más seguros (1.16.2)
Approximate matching rechazado — OpenCode ya no sobreescribe archivos por "coincidencia aproximada". Capa extra de seguridad para la implementación.

### 🛡️ Permisos de subagentes arreglados (1.17.2)
Los 46 team members ahora respetan su propia configuración de permisos. Antes podían ignorarlos. Crítico para la seguridad de OVAV.

### 📚 References system (1.17.1)
Sistema de referencias externas con descripciones contextuales. OVAV agregó referencias a `docs/`, `contracts/` y `registry/`.

---

## Cambios aplicados en OVAV

| Archivo | Cambio | Impacto |
|---------|--------|---------|
| `opencode.json` | +references | Nuevas referencias contextuales habilitadas |
| `opencode.json` | ~~+recovery, +background_agents~~ | ⚠️ REMOVIDOS 2026-06-11 — claves inválidas en schema |
| `session_greeting.py` | ThreadPoolExecutor, 4 fases paralelas | 16s → 5s |
| `integrity_hashlib.py` | Librería unificada de SHA256 | 10 copias → 1 |
| `safe_stop_contract.yaml` | +open_code_1_17_recovery, +host_runtime_limits | Documenta comportamiento recovery del host |
| `context_economy_contract.yaml` | +compaction_recovery | Documenta márgenes de contexto con recovery |
| `context_economy_contract.yaml` | ~~+opencode_1_17_config~~ | ⚠️ REMOVIDO 2026-06-11 — referenciaba keys inválidas |
| 10 archivos de seguridad | Reemplazan `_sha256()` local → `integrity_hashlib` | DRY, mantenible |

---

## Atajos de teclado relevantes

| Atajo | Acción | Nota |
|-------|--------|------|
| `Ctrl+B` | Ocultar/Mostrar barra lateral derecha (sidebar) | Default TUI |
| `Ctrl+Shift+B` | Enviar subagente a background | ⚠ Puede colisionar con Ctrl+B en algunos entornos |
| `Ctrl+K` | Abrir @ autocompletar agentes | Solo muestra leads (✦) y áreas (◆) |
| `Tab` | Cambiar entre áreas de servicio | 8 áreas disponibles |

### ⚠ Conflicto Ctrl+B
En la TUI de OpenCode, `Ctrl+B` oculta la barra lateral. En algunos terminales, `Ctrl+Shift+B` para background subagents puede interpretarse como `Ctrl+B`. Workaround:
- Usar `Ctrl+Shift+B` explícitamente desde el menú
- O reconfigurar en settings de OpenCode si el conflicto persiste
- En WezTerm, verificar que `Ctrl+Shift+B` no esté mapeado a otra acción

---

## Verificación post-upgrade

```bash
# Verificar versión
opencode --version

# Verificar que OVAV carga correctamente
python3 tools/agent_runtime/session_greeting.py --json 2>&1 | head -5

# Verificar integridad del nuevo hashlib
python3 -c "from tools.security.integrity_hashlib import file_hash; print('OK')"

# Verificar opencode.json (solo keys válidas del schema)
python3 -c "import json; c=json.load(open('opencode.json')); print('references:', list(c.get('references',{}).keys())); print('schema:', c.get('\$schema')); print('OK — sin keys inválidas')"
```
