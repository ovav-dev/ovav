# OVAV Workstation Access Profile

## Propósito

El Workstation Access Profile es la forma de convertir una necesidad local del usuario —como acceso GitHub por SSH sin prompts repetidos— en una capacidad gobernada por OVAV.

No es un almacén de secretos. Es una capa de política, plantillas y validadores para que el usuario conserve control y seguridad.

## Caso inicial: GitHub SSH para OVAV/Thavren

El caso inicial usa:

- transporte SSH;
- llave dedicada `ovav_thavren_ed25519`;
- passphrase obligatoria;
- `ssh-agent` con lifetime diario de 24h;
- alias `github-ovav-thavren`;
- remoto Git separado del perfil global.

## Qué controla OVAV

OVAV puede controlar y validar:

1. que el remoto no use HTTPS accidentalmente;
2. que el host alias sea el esperado;
3. que la plantilla SSH use `IdentitiesOnly yes`;
4. que la plantilla no contenga material privado;
5. que el flujo fish/WSL no auto-desbloquee secretos en cada shell;
6. que exista una separación clara entre perfil OVAV/BAB y sistema global.

## Qué no controla OVAV

OVAV no debe:

- leer passphrases;
- leer llaves privadas;
- copiar contenido de `known_hosts` al repo;
- escribir en `~/.ssh` o `~/.config` sin aprobación explícita;
- modificar el remoto real sin gate;
- instalar 1Password, FIDO2, plugins o servicios externos como efecto lateral.

## Modo operativo recomendado

El usuario desbloquea una vez por sesión:

```text
ovav_ssh_unlock
```

Después, las operaciones Git usan el agente:

```text
git fetch
git pull
git push
```

Si se cierra la sesión real o vence el lifetime de 24h, el agente vuelve a pedir passphrase una sola vez y abre otro ciclo diario.

## Evolución del producto

Esta función puede convertirse en un comando futuro de OVAV:

```text
ovav workstation access diagnose
ovav workstation access plan --profile thavren-github
ovav workstation access apply --consent --accept-risk
ovav workstation access verify
ovav workstation access rollback
```

La primera versión debe permanecer source-local y diagnóstica. La instalación real debe quedar detrás de backup, consentimiento doble y rollback.

## Implementación source-local actual

La primera versión queda implementada como herramienta segura:

```text
python3 tools/workstation/ovav_workstation_access.py plan
python3 tools/workstation/ovav_workstation_access.py diagnose
python3 tools/workstation/ovav_workstation_access.py verify-source
python3 tools/workstation/ovav_workstation_access.py apply
```

El comando `apply` existe como superficie futura pero responde `blocked`; no escribe fuera del repo.
