# OVAV Workstation Profile — SSH Thavren Remote Access

## Estado de esta task

- Rama: `task/ovav-thavren-ssh-profile`
- Alcance: source-local, sin escribir en `~/.ssh`, `~/.config`, Windows user config ni configuración global.
- Objetivo: convertir el acceso remoto GitHub de OVAV/Thavren en un perfil seguro, repetible y validable.

## Problema que estamos resolviendo

El flujo actual usa SSH, pero pide autorización repetidamente al abrir sesiones nuevas, panes o terminales. Eso rompe el trabajo profesional porque obliga al usuario a reautorizar operaciones pequeñas y puede mezclar identidad personal/global con identidad OVAV/BAB.

La necesidad real no es solo “guardar la clave”. La necesidad es:

1. mantener SSH como transporte seguro;
2. aislar la identidad OVAV/Thavren de otras identidades del sistema;
3. evitar prompts repetidos durante una sesión válida;
4. volver a pedir autorización cuando la sesión expire o el entorno se cierre;
5. permitir que OVAV valide la postura sin custodiar secretos.

## Recomendación para nuestro caso

La opción recomendada para OVAV es:

**SSH Ed25519 dedicado + passphrase + ssh-agent con duración controlada + alias GitHub exclusivo.**

Por qué esta es la mejor primera opción:

- ofrece buen balance entre seguridad y velocidad diaria;
- funciona bien en WSL2, GitHub, WezTerm y fish;
- no requiere hardware adicional;
- permite que el usuario desbloquee una vez por sesión o por ventana de tiempo;
- separa el remoto OVAV de perfiles globales;
- puede evolucionar después a llave hardware FIDO2 sin rediseñar todo.

## Comparación de opciones 2026

| Opción | Recomendación | Uso ideal | Riesgo/control |
|---|---:|---|---|
| SSH Ed25519 dedicado + agent | Alta | OVAV/BAB diario | Seguro si la llave tiene passphrase y lifetime |
| SSH FIDO2/security key | Muy alta | Máxima seguridad | Requiere hardware/toque físico; menos fluido |
| 1Password SSH Agent | Alta | Si el usuario ya usa 1Password | Excelente UX; depende de software externo |
| HTTPS + Git Credential Manager | Media | Flujo simple global | Más fácil mezclar identidades/tokens |
| Llave sin passphrase | No recomendada | Nunca para OVAV | Si se copia la llave, el acceso queda expuesto |

## Diseño OVAV

OVAV no debe guardar ni leer la llave privada. OVAV debe gobernar el perfil:

- alias SSH esperado: `github-ovav-thavren`;
- remoto esperado: `git@github-ovav-thavren:ORG/REPO.git`;
- identidad dedicada: `~/.ssh/ovav_thavren_ed25519` o variante FIDO2;
- `IdentitiesOnly yes` para evitar que Git pruebe otras llaves;
- `AddKeysToAgent yes` para reducir prompts repetidos;
- lifetime del agente definido por política local: `24h` para OVAV/BAB diario;
- validadores que detecten HTTPS accidental, alias incorrecto o plantilla incompleta.

## Comportamiento esperado

Flujo deseado:

1. usuario abre WezTerm/WSL;
2. fish inicializa el entorno OVAV;
3. `ssh-agent` existe o se inicia;
4. si la llave no está cargada, se pide passphrase una vez;
5. durante las siguientes 24 horas, `git fetch/pull/push` no repite prompts;
6. al cerrar sesión real o expirar el lifetime de 24 horas, vuelve a pedir autorización una sola vez;
7. otros perfiles del usuario no heredan automáticamente esta identidad.

## Política diaria de expiración

Para el entorno OVAV/BAB, el default propuesto es `24h`:

```fish
set -gx OVAV_SSH_AGENT_LIFETIME 24h
```

Comportamiento esperado:

- primera operación/unlock del día: pide passphrase;
- operaciones posteriores del día: reutilizan el agente;
- al vencer 24h: pide passphrase una vez y abre otro ciclo de 24h;
- no requiere repetir la configuración larga;
- no guarda passphrases ni llaves privadas en OVAV.

## Función futura de OVAV

Este perfil puede evolucionar a una función del sistema:

**OVAV Workstation Access Profile**

Capacidades previstas:

- plan source-local del perfil SSH;
- diagnóstico sin secretos;
- verificación de remoto permitido;
- detección de mezcla HTTPS/SSH;
- verificación de agent activo sin imprimir llaves;
- instalación gobernada con backup y doble consentimiento;
- rollback del archivo de configuración aplicado;
- compatibilidad WSL2/Windows/WezTerm/fish.

## Límites activos

Esta task no autoriza todavía:

- escribir `~/.ssh/config`;
- generar o copiar llaves reales;
- cambiar el remoto real del repo;
- tocar configuración global de Windows o WezTerm;
- instalar agentes, plugins o integraciones externas;
- guardar passphrases, tokens o secretos en el repo.

## Próximo paso seguro

Primero se validan estos artefactos source-local:

- `config/ssh/ovav-thavren.ssh.config.example`
- `config/fish/ovav-thavren-ssh-agent.fish.example`
- `config/workstation/ovav-thavren-ssh-profile.yaml`
- `config/workstation/ovav-thavren-ssh-install-plan.yaml`
- `tools/validators/check_ovav_ssh_profile.py`
- `tools/workstation/ovav_workstation_access.py`

La función reutilizable queda descrita en:

- `docs/workstation/OVAV_WORKSTATION_ACCESS_PROFILE.md`

Luego, con aprobación explícita, se prepara un segmento de instalación gobernada para aplicar el perfil en el sistema real.
