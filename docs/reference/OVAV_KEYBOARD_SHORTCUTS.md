# OVAV↔OpenCode — Atajos de Teclado

> **C7.4.2** — Documentación completa de atajos, conflictos y workarounds.

## Atajos OpenCode 1.17

| Atajo | Acción | Contexto |
|---|---|---|
| `Ctrl+B` | Abrir/cerrar sidebar | Global |
| `Ctrl+Shift+B` | Background subagent | Con sidebar cerrado |
| `Ctrl+K` | @ autocompletar (leads ✦ + áreas ◆) | Editor |
| `Ctrl+L` | Limpiar chat | Chat |
| `Ctrl+Enter` | Enviar mensaje (multilínea) | Editor |
| `Tab` | Cambiar área de servicio | Chat |
| `Ctrl+Shift+P` | Command palette | Global |
| `Ctrl+Shift+V` | Abrir vista previa Markdown | Editor |

## Conflicto Conocido: Ctrl+B / Ctrl+Shift+B

### El problema (C7.4.1)

`Ctrl+B` abre el sidebar de OpenCode. `Ctrl+Shift+B` lanza un background subagent. Como la sidebar captura `Ctrl+B` primero, presionar `Ctrl+Shift+B` puede interpretarse como `Ctrl+B` (abrir sidebar) + `Shift` (ignorado).

### Workaround

**Opción A (recomendada):** Cerrar la sidebar primero (`Ctrl+B` una vez), luego `Ctrl+Shift+B` funciona correctamente para lanzar el subagent.

**Opción B:** Usar la command palette (`Ctrl+Shift+P`) → escribir "background" → seleccionar "Run Background Task".

**Opción C:** Si usás `Ctrl+Shift+B` con frecuencia, mantené la sidebar cerrada por defecto y abrila solo cuando la necesites.

### Verificación

Para confirmar que `Ctrl+Shift+B` funciona: cerrá la sidebar, presioná `Ctrl+Shift+B`, debería aparecer el prompt de background subagent.

## Atajos OVAV

| Comando | Qué hace |
|---|---|
| `ovav status` | Estado completo del sistema |
| `ovav monitor` | Dashboard de costos en vivo |
| `ovav connect` | Centro de control de tokens (C7.3) |
| `ovav profile list` | Listar perfiles profesionales |
| `ovav profile apply <area>` | Activar perfil |
| `ovav update --check` | Verificar actualizaciones |
| `ovav validate` | Validación completa del sistema |

## WezTerm (C7.1.2)

| Atajo | Acción |
|---|---|
| `Alt+1` | Workspace HOME |
| `Alt+2` | Workspace SYSTEM |
| `Alt+3` | Workspace DEVBRK |
| `Alt+4` | Workspace OVAV |
| `Ctrl+T` | Nueva tab (dentro del workspace actual) |
| `Ctrl+W` | Cerrar tab |
| `Ctrl+Shift+T` | Reabrir última tab cerrada |
| `Ctrl+Shift+Enter` | Maximizar/restaurar panel |
| `Alt+←→↑↓` | Navegar entre panes |

---

*Última actualización: CAPA 7 (v2.6.0) — 2026-06-12*
