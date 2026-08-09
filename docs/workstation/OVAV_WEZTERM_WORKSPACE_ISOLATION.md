# OVAV WezTerm Workspace Isolation

## Estado de esta task

- Rama: `task/ovav-wezterm-workspace-isolation`
- Alcance: source-local, sin escribir en `~/.config/wezterm`, Windows user config ni configuración global.
- Objetivo: aislar panes, tabs y workspaces de WezTerm por repo+rama para evitar contaminación operacional entre ramas.
- Mejora visual: usar una paleta tipo OVAV PI para reducir dolor visual por panes azul/negro, gris sobre gris y dimming agresivo.
- Modo Windows gobernado: cuatro sesiones fijas `HOME`, `SYSTEM`, `DEVBRK`, `OVAV`; la primera tab es indicador estático de sesión y las tabs nuevas usan ruta dinámica abreviada.

## Problema que estamos resolviendo

Cuando varias ramas viven en la misma ventana, tab o workspace de WezTerm, el operador puede:

1. ejecutar comandos en el `cwd` de otra rama;
2. reutilizar panes con variables de entorno incorrectas;
3. hacer `git fetch/pull/push` desde una rama distinta a la intención visible;
4. mezclar sesiones OVAV/BAB con trabajo personal o experimental;
5. perder evidencia de qué workspace produjo una validación.

La necesidad profesional no es solo “abrir otra terminal”. La necesidad es que WezTerm tenga una frontera visible, repetible y validable entre cada repo+rama.

## Recomendación para nuestro caso

La opción recomendada es:

**Workspace WezTerm por `repo + branch + root_hash`, con título visible y launch command source-local.**

Por qué esta es la mejor primera opción:

- no requiere tocar configuración global para validar el diseño;
- evita que una rama reutilice accidentalmente panes de otra;
- mantiene tabs y panes dentro del workspace nombrado;
- funciona como contrato simple para Windows/WSL/WezTerm;
- puede instalarse después mediante un segmento gobernado con backup y rollback.

## Diseño OVAV

OVAV no debe instalar ni mutar WezTerm en esta task. OVAV debe gobernar el perfil:

- workspace esperado: `ovav-{repo_slug}-{branch_slug}-{root_hash}`;
- rama esperada: detectada por `git branch --show-current` o fallback `detached-{sha}`;
- root hash: hash corto del repo root para evitar colisiones de nombres iguales;
- env contract: `OVAV_WEZTERM_WORKSPACE`, `OVAV_GIT_BRANCH`, `OVAV_REPO_ROOT`;
- launch command dry-run: genera el comando recomendado pero no lo ejecuta;
- current-pane check: compara `OVAV_WEZTERM_WORKSPACE`, `OVAV_GIT_BRANCH` y `OVAV_REPO_ROOT` antes de trabajo sensible;
- plantilla Lua: ejemplo source-local para `default_workspace`, `gui-startup`, `format-tab-title` y `update-right-status`.
- paleta visual: `OVAV PI Eye Comfort`, con fondo consistente, texto claro y panes inactivos sin oscurecimiento brusco.
- canvas sketchbook: header, tabs, scroll y terminal comparten el mismo fondo para que todo parezca una sola pantalla viva, sin recortes de color.
- scroll Chrome-like: barra mínima integrada, sin riel visual separado y con thumb sutil sobre el mismo fondo.
- tabs largas: `Ctrl+T` crea tabs internas largas para alojar abreviatura inteligente de ruta.

## Comportamiento esperado

Flujo deseado:

1. operador entra al repo/rama correcta;
2. ejecuta diagnóstico o launch-command source-local;
3. OVAV calcula un workspace único por repo+rama;
4. WezTerm abre/usa una ventana con ese workspace;
5. panes y tabs muestran workspace+rama;
6. otra rama produce otro workspace, no reutiliza el anterior;
7. cerrar o cambiar rama exige abrir/adjuntar al workspace correcto.

Ejemplo conceptual:

```text
task/ovav-wezterm-workspace-isolation
→ ovav-ovav-public-export-task-ovav-wezterm-workspace-isolation-a1b2c3d4
```

## Política de aislamiento

Reglas activas:

- `workspace_scope`: `repo_branch_root_hash`
- `pane_scope`: hereda el workspace actual;
- `tab_scope`: hereda el workspace actual;
- `cross_branch_attach`: bloqueado por diferencia de nombre;
- `visual_boundary`: título de ventana/tab/status con workspace+rama;
- `eye_comfort_boundary`: contraste claro, sin gris sobre gris y sin pane inactivo negro;
- `sketchbook_canvas`: un solo background para header, tabs, scroll y cuerpo;
- `chrome_like_scrollbar`: scroll integrado tipo Google Chrome dentro de los límites de WezTerm;
- `fixed_operator_sessions`: `Alt+1/2/3/4` cambia sesión completa sin pasar tabs entre sesiones;
- `writes_performed`: siempre `false` en esta task.

## Límites activos

Esta task no autoriza todavía:

- escribir `~/.config/wezterm/wezterm.lua`;
- escribir config WezTerm de Windows;
- lanzar o cerrar ventanas reales de WezTerm como efecto lateral;
- instalar plugins o integraciones externas;
- mutar remotos Git o branches;
- guardar rutas de usuario fuera del repo como evidencia permanente;
- declarar que el aislamiento está instalado globalmente.

## Arquitectura v3 — Canonical Path (2026-06-07)

### Principio: un solo canonical, un solo entry point

```
CANONICAL (WSL Linux):
  ~/.config/wezterm/wezterm.lua          ← fuente única de verdad

WINDOWS ENTRY POINT (proxy):
  %USERPROFILE%\.wezterm.lua             ← proxy que delega al canonical vía UNC
  %USERPROFILE%\.wezterm-fallback.lua    ← cache local (si WSL offline)

ELIMINADO:
  %APPDATA%\wezterm\.wezterm.lua         ← redundante, removido, blindado
```

### Cadena de resolución del proxy

1. Intenta cargar `\\wsl.localhost\Ubuntu-24.04\home\braka\.config\wezterm\wezterm.lua`
2. Si WSL no disponible → carga `%USERPROFILE%\.wezterm-fallback.lua`
3. Si no hay fallback → carga mínimos funcionales de emergencia

### Blindaje de ruta

- Validador: `tools/validators/check_wezterm_path_integrity.py`
- Bloquea referencias a `%APPDATA%\wezterm\` en cualquier artefacto (deprecated, bloqueado)
- Exige marcadores `OVAV_WZPROXY_v3` en los proxies
- Exige marcador `OVAV_WZFALLBACK_v1` en el fallback
- Alerta ERROR si detecta config duplicada en paths no autorizados

## Artefactos source-local

- `.ovav/registry/tool_configs.yaml`
- `config/wezterm/wezterm.lua` (governed config, proxy v3)
- `.ovav/source/configs/wezterm/ovav-windows-loader.wezterm.lua` (proxy v3, deploy target)
- `config/wezterm/wezterm-fallback-minimal.lua` (fallback cache)
- `.ovav/source/configs/wezterm/ovav-workspace-isolation.wezterm.lua.example`
- `config/workstation/ovav-wezterm-workspace-isolation.yaml`
- `tools/workstation/ovav_wezterm_workspace.py`
- `tools/cli/ovav_tool_configs.py`
- `tools/validators/check_ovav_wezterm_workspace_isolation.py`
- `tools/validators/check_wezterm_path_integrity.py`
- `tools/validators/check_tool_config_profiles.py`

## Integración con OVAV Tailor

WezTerm queda registrado como Tool Config Profile de terminal en `ovav tailor` / `ovav workspace`.
OVAV muestra el perfil aunque el binario `wezterm` no exista, pero no lo instala.

Comandos de perfil:

```text
ovav tools wezterm plan
ovav tools wezterm verify
```

El comando `ovav tools wezterm apply` existe solo como superficie bloqueada: no escribe config real hasta que exista un segmento de instalación aprobado con preview, backup, consentimiento, verificación y rollback.

## Comandos source-local disponibles

Plan:

```text
python3 tools/workstation/ovav_wezterm_workspace.py plan
```

Diagnóstico seguro:

```text
python3 tools/workstation/ovav_wezterm_workspace.py diagnose
```

Nombre de workspace:

```text
python3 tools/workstation/ovav_wezterm_workspace.py workspace-name
```

Launch command dry-run:

```text
python3 tools/workstation/ovav_wezterm_workspace.py launch-command
```

Check del pane actual:

```text
python3 tools/workstation/ovav_wezterm_workspace.py check-current
```

Verificación source-local:

```text
python3 tools/workstation/ovav_wezterm_workspace.py verify-source
python3 tools/validators/check_tool_config_profiles.py
```

Intento de apply:

```text
python3 tools/workstation/ovav_wezterm_workspace.py apply
```

El `apply` está diseñado para devolver bloqueo hasta que exista un segmento aprobado de instalación real.

## Próximo paso seguro

Validar integridad de rutas con el nuevo validador:

```text
python3 tools/validators/check_wezterm_path_integrity.py --json
```

Esto verifica:
- Que el proxy tiene marcadores v3 correctos
- Que ningún artefacto referencia paths bloqueados (APPDATA, .ovav/source, etc.)
- Que no hay config duplicada en Windows
- Que el fallback existe y es íntegro

Para desplegar el proxy a Windows (requiere aprobación explícita):
1. Backup de `%USERPROFILE%\.wezterm.lua` actual
2. Copiar `.ovav/source/configs/wezterm/ovav-windows-loader.wezterm.lua` → `%USERPROFILE%\.wezterm.lua`
3. Copiar `config/wezterm/wezterm-fallback-minimal.lua` → `%USERPROFILE%\.wezterm-fallback.lua`
4. Eliminar `%APPDATA%\wezterm\.wezterm.lua` (deprecated, eliminado)
5. Verificar con `check_wezterm_path_integrity.py`

## WezTerm Keyboard Shortcuts (C7.4)

Key bindings for workspace-isolated OVAV WezTerm sessions:

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+N` | New OVAV workspace window |
| `Ctrl+Shift+T` | New tab (current workspace) |
| `Ctrl+Shift+P` | Activate command palette |
| `Ctrl+Shift+W` | Close current pane/tab |
| `Ctrl+Shift+F` | Fuzzy search text in scrollback |
| `Super+R` | Rename current workspace |
| `Super+1..9` | Switch to workspace 1-9 |
| `Alt+F4` | Quit WezTerm |

These shortcuts are documented for C7 cross-platform compliance and are configured
in the canonical `~/.config/wezterm/wezterm.lua` (WSL) or loaded via the Windows
proxy loader using the `keys` table in the configuration.
