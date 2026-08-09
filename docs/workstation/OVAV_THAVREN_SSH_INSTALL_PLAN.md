# OVAV Thavren SSH Install Plan

## Estado

Este plan está implementado como **dry-run source-local**. No aplica cambios reales todavía.

La instalación real requiere aprobación explícita porque tocaría superficies sensibles:

- `~/.ssh`;
- `~/.config/fish`;
- remoto Git real;
- posible generación de llave SSH real.

## Resultado esperado después de instalación aprobada

El usuario debe poder trabajar un día completo con GitHub sin reingresar la passphrase en cada operación:

```text
ovav_ssh_unlock  -> pide passphrase una vez
git fetch/pull   -> reutiliza ssh-agent durante 24h
24h expira       -> pide passphrase una vez y reinicia el ciclo
```

## Orden de aplicación real

Cuando se apruebe la instalación real, el orden correcto será:

1. crear backup de fragmentos existentes;
2. confirmar o crear llave dedicada `ovav_thavren_ed25519`;
3. instalar fragmento SSH `github-ovav-thavren`;
4. instalar helper fish `ovav_ssh_unlock`;
5. cargar llave con lifetime `24h`;
6. probar `ssh -T github-ovav-thavren`;
7. migrar remoto Git a `git@github-ovav-thavren:ORG/REPO.git`;
8. validar `git fetch`;
9. documentar rollback.

## Qué queda bloqueado hasta aprobación

Este branch no ejecuta:

```text
ssh-keygen real
copy hacia ~/.ssh
copy hacia ~/.config/fish
```

## Comandos source-local disponibles

Plan:

```text
python3 tools/workstation/ovav_workstation_access.py plan
```

Diagnóstico seguro:

```text
python3 tools/workstation/ovav_workstation_access.py diagnose
```

Verificación source-local:

```text
python3 tools/workstation/ovav_workstation_access.py verify-source
```

Intento de apply:

```text
python3 tools/workstation/ovav_workstation_access.py apply
```

El `apply` está diseñado para devolver bloqueo hasta que exista un segmento aprobado de instalación real.
