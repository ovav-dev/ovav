# SEG 2 — Cablear Launch: instalación real con progreso

## Objetivo

La opción "Instalar OVAV" del cockpit ejecuta el pipeline completo de instalación con barra de progreso real (no animación fake). El usuario ve cada paso, recibe feedback y termina con una pantalla de resultado.

## Alcance

- `action_payload()` en el engine ejecuta el comando `ovav install` con progreso en vivo.
- El progreso no es animación prefabricada: cada paso del pipeline (backup → consent → apply → verify) mueve la barra según output real del comando.
- Si el comando falla, se muestra el error con opción de reintentar.
- Pantalla de resultado muestra qué se instaló, verificación y siguiente paso.

## Pipeline de instalación

```
[1] Detectar entorno        ████████░░░░  40%
[2] Crear backup            ████████████  60%
[3] Solicitar consent       ████████████  70%
[4] Aplicar instalación     ████████████  85%
[5] Verificar               ████████████ 100%
```

## Archivos

- `tools/cli/ovav_first_run_cockpit.py` → `action_payload()`, `run_command()`, `progress()`, pantalla Launch
- `tools/cli/ovav_first_run_cockpit.py` → marcar como legacy (no se carga por defecto)

## Validación

- ENTER en "Instalar OVAV" → barra de progreso real avanza.
- Instalación completada → pantalla de resultado con verificación.
- Instalación fallida → error visible, opción de reintentar.
- `ovav install --json` por CLI sigue funcionando igual.

## Done when

- Launch ejecuta el pipeline real de install.
- Progreso refleja output real, no animación fake.
- Resultado muestra verificación post-install.
- Smoke test de install desde cockpit pasa.
