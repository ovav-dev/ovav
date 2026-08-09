# Cloudflare Zero Trust — Tunnel + Access para OVAV cPanel

> **Para:** CEO  
> **Ejecutar en:** Cloudflare Dashboard (Zero Trust) + terminal Fly.io  
> **Tiempo estimado:** 25–35 minutos  
> **Prerrequisito:** Tener el dominio `ovav.dev` activo en Cloudflare y `flyctl` autenticado.

---

## 1. CLOUDFLARE TUNNEL SETUP

Creamos dos túneles — uno para producción y otro para staging. Cada túnel expone el cPanel corriendo en Fly.io (`localhost:5858`) a través de un subdominio imposible de adivinar.

### 1.1 Túnel de Producción (`ovav-cpanel-prod`)

#### Paso 1: Ir a la sección de Túneles

1. Abre https://one.dash.cloudflare.com/
2. En el menú lateral izquierdo, haz clic en **Networks**.
3. Haz clic en **Tunnels** (submenú que aparece debajo de Networks).
4. Haz clic en el botón azul **Create a tunnel**.

#### Paso 2: Crear el túnel

| Campo | Valor |
|-------|-------|
| Tunnel name | `ovav-cpanel-prod` |

5. Haz clic en **Save tunnel**.

#### Paso 3: Copiar el token del túnel

6. Después de guardar, Cloudflare te muestra la pantalla de instalación.
7. **IMPORTANTE:** En la sección «Install and run a connector», busca el token en el comando que aparece. El token es el string largo después de `--token`. Cópialo entero.
8. Guarda este token en un lugar seguro (gestor de contraseñas). Lo necesitarás en el Paso 5.

#### Paso 4: Generar subdominio un-guessable

9. Abre una terminal y ejecuta:

```bash
echo -n "TU_FRASE_SECRETA_AQUI" | sha256sum | cut -c1-8
```

> Usa una frase secreta que solo tú conozcas. Ejemplo de salida: `a7k3m2x9`

10. El resultado es tu subdominio. Ejemplo final: `a7k3m2x9.ovav.dev`

#### Paso 5: Configurar hostname público en el túnel

11. En la misma página del túnel recién creado, ve a la pestaña **Public Hostname**.
12. Haz clic en **Add a public hostname**.

| Campo | Valor |
|-------|-------|
| Subdomain | El hash del Paso 4 (ej: `a7k3m2x9`) |
| Domain | `ovav.dev` |
| Type | `HTTP` |
| URL | `localhost:5858` |

13. **NO marques** «Protect with Access» todavía (lo harás en la Sección 2).
14. Haz clic en **Save hostname**.

#### Paso 6: Configurar el token en Fly.io

15. En tu terminal local, ejecuta:

```bash
flyctl secrets set CF_TUNNEL_TOKEN=<token-del-paso-3> --app ovav-systems
```

16. Verifica que el secreto se guardó:

```bash
flyctl secrets list --app ovav-systems
```

> Debes ver `CF_TUNNEL_TOKEN` en la lista (el valor aparece encriptado).

#### Paso 7: Verificar que el túnel está HEALTHY

17. Vuelve a **Zero Trust → Networks → Tunnels**.
18. Busca `ovav-cpanel-prod` en la lista. El status debe mostrar **HEALTHY** (indicador verde).
19. Si muestra «Unhealthy» o «Inactive», ve a la Sección 5 (Troubleshooting).

---

### 1.2 Túnel de Staging (`ovav-cpanel-staging`)

Repite **exactamente los mismos pasos** de la Sección 1.1 con estos cambios:

| Campo | Producción | Staging |
|-------|-----------|---------|
| Tunnel name | `ovav-cpanel-prod` | `ovav-cpanel-staging` |
| Subdomain hash | El que generaste en Paso 4 | Genera uno **diferente** (con otra frase) |
| Ejemplo subdomain | `a7k3m2x9.ovav.dev` | `b9f4n1p6.ovav.dev` |
| App en Fly.io | `ovav-systems` | `ovav-systems-staging` |
| Comando secrets | `flyctl secrets set CF_TUNNEL_TOKEN=<...> --app ovav-systems` | `flyctl secrets set CF_TUNNEL_TOKEN=<...> --app ovav-systems-staging` |

> **⚠️ No uses el mismo subdominio para staging y producción.** Cada túnel necesita su propio hostname público. Si usas el mismo, Cloudflare rechazará el segundo.

---

## 2. CLOUDFLARE ACCESS SETUP

Ahora protegemos el subdominio de producción con Cloudflare Access. Solo tú (CEO) podrás ver el cPanel. Cualquier otra persona verá un 404 normal.

### 2.1 Agregar aplicación en Access

1. Ve a **Zero Trust → Access → Applications**.
2. Haz clic en **Add an application**.
3. Selecciona **Self-hosted**.

### 2.2 Configurar la aplicación

4. En la pantalla «Configure app», llena:

| Campo | Valor |
|-------|-------|
| Application name | `OVAV cPanel` |
| Session Duration | `24 hours` (o la duración que prefieras) |
| **Subdomain** | El hash de producción (ej: `a7k3m2x9`) |
| **Domain** | `ovav.dev` |

> 💡 **Tip:** Si quieres proteger también el path `/api/*` con reglas diferentes, puedes agregar múltiples aplicaciones. Para este setup, protegemos el subdominio completo.

5. Deja el campo **Path** vacío (protege todo el subdominio).

6. Haz clic en **Next**.

### 2.3 Configurar política de acceso

7. En la pantalla «Add policies», haz clic en **Add a policy**.

| Campo | Valor |
|-------|-------|
| Policy name | `CEO Only` |
| Action | `Allow` |

8. En **Configure rules**, agrega una regla:

| Selector | Operator | Value |
|----------|----------|-------|
| Emails | is | `tu-email@gmail.com` |

> Si prefieres usar GitHub como proveedor de identidad, usa el selector `GitHub Organization` o `GitHub User` en lugar de Emails.

9. Haz clic en **Save policy**.

### 2.4 Seleccionar proveedor de identidad

10. En la sección **Identity providers**, selecciona los proveedores que quieres habilitar:

| Proveedor | Cuándo usarlo |
|-----------|--------------|
| **Google** | Si tu email es Gmail/Google Workspace — recomendado, es lo más rápido |
| **GitHub** | Si prefieres autenticarte con tu cuenta de GitHub |

> Puedes seleccionar ambos. El usuario verá una pantalla para elegir con cuál autenticarse.

11. Activa el toggle **Instant Auth** si solo usas un proveedor (saltará la pantalla de selección de IdP).

12. Haz clic en **Next**.

### 2.5 Revisar y guardar

13. Revisa el resumen en la pantalla final.
14. Haz clic en **Add application**.

---

## 3. CUSTOM BLOCK PAGE (404 CAMOUFLAGE)

Por defecto, Cloudflare Access muestra una página que dice «Access Denied» a usuarios no autorizados. Esto revela que hay algo detrás de la URL. La camuflamos como un 404 normal.

### 3.1 Ir a Custom Pages

1. Ve a **Zero Trust → Settings → Custom Pages**.
2. Busca la sección **Access Custom Pages** y haz clic en **Manage**.

### 3.2 Crear página para «Non-identity failure»

3. Haz clic en **Add a page template**.
4. En el dropdown **Page type**, selecciona:

```
Non-identity → Non-identity failure
```

5. En **Name**, escribe: `404 Camouflage`

### 3.3 Pegar el HTML

6. Abre el archivo en tu workspace:

```
docs/infra/access_404_block.html
```

7. Copia **todo** el contenido del archivo (Ctrl+A, Ctrl+C).
8. Pégalo en el campo **HTML** de la página de Cloudflare.

> El HTML es un 404 genérico — fondo gris claro, «404» en grande, mensaje «The page you are looking for does not exist or has been moved.» No tiene logos, ni branding, ni referencias a OVAV.

9. Haz clic en **Save**.

### 3.4 Verificar que la página se aplica

10. Abre una ventana de incógnito/privada en tu navegador.
11. Visita `https://<tu-subdominio>.ovav.dev`
12. Debes ver la página 404 genérica, **no** un mensaje de «Access Denied» ni un login de Cloudflare.

---

## 4. VERIFICATION

Pruebas rápidas para confirmar que todo funciona.

### 4.1 Test desde incógnito (visitante no autorizado)

| Paso | Acción | Resultado esperado |
|------|--------|-------------------|
| 1 | Abre una ventana de **Incógnito** (Chrome) o **Private Window** (Firefox/Safari) | — |
| 2 | Visita `https://<tu-subdominio>.ovav.dev` | ✅ Página 404 gris genérica («The page you are looking for…») |
| 3 | Visita `https://<tu-subdominio>.ovav.dev/admin` | ✅ Misma página 404 |
| 4 | Visita `https://<tu-subdominio>.ovav.dev/api/v1/health` | ✅ Misma página 404 |

> Si ves un login de Google/GitHub en lugar del 404, el custom block page no se aplicó correctamente. Revisa Sección 3.

### 4.2 Test desde navegador autenticado (CEO)

| Paso | Acción | Resultado esperado |
|------|--------|-------------------|
| 1 | Abre tu navegador normal (donde ya tienes sesión de Google/GitHub) | — |
| 2 | Visita `https://<tu-subdominio>.ovav.dev` | ✅ Login de cPanel (o la app OVAV) |
| 3 | Inicia sesión en cPanel con OAuth | ✅ Dashboard del cPanel |
| 4 | Navega por el cPanel normalmente | ✅ Sin interrupciones ni re-autenticaciones |

### 4.3 Verificar salud del túnel

| Paso | Acción | Resultado esperado |
|------|--------|-------------------|
| 1 | Ve a **Zero Trust → Networks → Tunnels** | — |
| 2 | Busca `ovav-cpanel-prod` | ✅ Status: **HEALTHY** (indicador verde) |
| 3 | Busca `ovav-cpanel-staging` | ✅ Status: **HEALTHY** (indicador verde) |

### 4.4 Verificar desde Fly.io

```bash
# Verificar que el túnel está corriendo dentro del container
flyctl logs --app ovav-systems | grep -i tunnel
# Debe mostrar: "Registered tunnel connection" o similar

# Verificar que el puerto 5858 responde localmente
flyctl ssh console --app ovav-systems -C "curl -s -o /dev/null -w '%{http_code}' http://localhost:5858/health"
# Debe devolver: 200
```

---

## 5. TROUBLESHOOTING

### 5.1 El túnel muestra «INACTIVE» o «UNHEALTHY»

| Causa probable | Solución |
|----------------|----------|
| El secreto `CF_TUNNEL_TOKEN` no está configurado en Fly.io | Ejecuta `flyctl secrets set CF_TUNNEL_TOKEN=<token> --app ovav-systems` y redeploya |
| El container no tiene conectividad saliente al puerto 7844 | Verifica que Fly.io permita tráfico saliente a `region1.v2.argotunnel.com:7844` |
| El deploy más reciente no incluye `cloudflared` | Verifica que `Dockerfile.cpanel` instale y ejecute `cloudflared tunnel run --token $CF_TUNNEL_TOKEN` |
| El token expiró o fue revocado | Ve a **Zero Trust → Networks → Tunnels → ovav-cpanel-prod → Configure → Credentials**, rota el token y actualiza el secreto en Fly.io |

### 5.2 Access muestra «Access Denied» en vez del 404

| Causa probable | Solución |
|----------------|----------|
| No se configuró el Custom Page para «Non-identity failure» | Ve a **Zero Trust → Settings → Custom Pages → Access Custom Pages → Manage** y verifica que existe un template con tipo «Non-identity → Non-identity failure» |
| El HTML del template está vacío o mal formado | Abre `docs/infra/access_404_block.html` en el workspace, copia todo el contenido, y pégalo en el campo HTML del template |
| El tipo de página es incorrecto | Debe ser específicamente **«Non-identity failure»**, no «Block page» ni «Forbidden» |

### 5.3 El cPanel no carga después de autenticarse con Google/GitHub

| Causa probable | Solución |
|----------------|----------|
| El túnel no está apuntando al puerto correcto | En **Zero Trust → Networks → Tunnels → ovav-cpanel-prod → Public Hostname**, verifica que la URL es `localhost:5858` |
| El servicio en Fly.io no está escuchando en 0.0.0.0:5858 | Ejecuta `flyctl ssh console --app ovav-systems` y verifica con `ss -tlnp \| grep 5858` |
| OAuth redirect URI no coincide | Verifica que `OAUTH_REDIRECT_URI` en Fly.io y en Google Cloud Console / GitHub OAuth Apps use **exactamente** `https://<tu-subdominio>.ovav.dev` |

### 5.4 Error «ERR_CERT_COMMON_NAME_INVALID» en el navegador

| Causa probable | Solución |
|----------------|----------|
| El subdominio tiene múltiples niveles (ej: `admin.a7k3m2x9.ovav.dev`) | Cloudflare requiere un **Advanced Certificate** para subdominios multi-nivel. Usa un solo nivel (ej: `a7k3m2x9.ovav.dev`) o compra el Advanced Certificate en **SSL/TLS → Edge Certificates** |
| El registro DNS no está en modo Proxy | Ve a **DNS → Records**, busca el CNAME del subdominio, y asegúrate de que el toggle **Proxy status** está en naranja (Proxied) |

### 5.5 Error «Too Many Redirects»

| Causa probable | Solución |
|----------------|----------|
| El túnel está configurado con HTTPS pero el servicio solo escucha HTTP | En la configuración del túnel, cambia el Type a `HTTP` (no `HTTPS`). El túnel de Cloudflare ya maneja TLS en el borde |
| Cloudflare SSL/TLS está en modo «Flexible» | Ve a **SSL/TLS → Overview** y asegúrate de que el modo es **Full** o **Full (strict)** |

---

## Resumen post-configuración

| Componente | Estado esperado | Verificar en |
|-----------|----------------|--------------|
| `ovav-cpanel-prod` tunnel | 🟢 HEALTHY | Zero Trust → Networks → Tunnels |
| `ovav-cpanel-staging` tunnel | 🟢 HEALTHY | Zero Trust → Networks → Tunnels |
| Access app «OVAV cPanel» | 🟢 Active | Zero Trust → Access → Applications |
| Custom 404 block page | 🟢 Active | Incógnito → visitar subdominio |
| `CF_TUNNEL_TOKEN` en Fly.io | 🔒 Set (encriptado) | `flyctl secrets list --app ovav-systems` |
| cPanel accesible (CEO) | ✅ Login visible | Navegador autenticado → visitar subdominio |

---

## Referencias

| Recurso | Archivo / URL |
|---------|--------------|
| HTML del 404 camouflage | `docs/infra/access_404_block.html` |
| Cloudflare Zero Trust dashboard | https://one.dash.cloudflare.com/ |
| Docs oficiales de Tunnel | https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/ |
| Docs oficiales de Access | https://developers.cloudflare.com/cloudflare-one/applications/configure-apps/ |
| Fly.io secrets | https://fly.io/docs/flyctl/secrets/ |
