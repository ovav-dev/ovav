# Fly.io Certificate Cleanup — ACCIÓN REQUERIDA

**Fecha:** 2026-08-07  
**Estado:** 🔴 CRÍTICO - Requiere acción manual  
**Razón:** El token actual no tiene permisos para eliminar certificados

---

## Certificados Activos (DEBEN ELIMINARSE)

```
OVAV-SYSTEMS APP
═══════════════════════════════════════════════════════
api.ovav.dev       → DELETED (falta permisos)
cpanel.ovav.dev   → DELETED (falta permisos)
═══════════════════════════════════════════════════════
```

---

## Pasos para Eliminar desde Dashboard

1. **Ir a:** https://fly.io/dashboard
2. **Seleccionar org:** `alexander-salvador`
3. **Ir a app:** `ovav-systems`
4. **Navegar a:** Certificates
5. **Eliminar:**
   - `api.ovav.dev`
   - `cpanel.ovav.dev`

---

## Pasos para Crear Token con Permisos

1. **Ir a:** https://fly.io/user/tokens
2. **Crear token con scope:**
   ```
   apps:read
   apps:write
   certs:delete (para eliminar certificados)
   ```
3. **Guardar en vault:**
   ```bash
   echo "nuevo-token" > ~/.ovav/vault/tokens/FLY_API_TOKEN
   ```

---

## Verificación Post-Limpieza

```bash
# Debe mostrar solo ovav-systems (sin custom certs)
flyctl certs list -a ovav-systems

# Verificar que api.ovav.dev ya no responde
curl -I https://api.ovav.dev 2>/dev/null
# Esperado: SSL_ERROR o connection refused

# Verificar que cpanel.ovav.dev ya no responde
curl -I https://cpanel.ovav.dev 2>/dev/null
# Esperado: SSL_ERROR o connection refused
```

---

## Impacto de NO Eliminar

- `api.ovav.dev` expone información del backend
- `cpanel.ovav.dev` expone información del panel de control
- Certificados SSL válidos hacen estos dominios accesibles
- Potential vector de ataque por enumeración

---

**Última actualización:** 2026-08-07
