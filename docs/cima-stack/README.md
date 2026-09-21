# CIMA Stack — Análisis completo

## Identidad

| Atributo | Valor |
|---|---|
| **Stack** | CIMA — Técnico Móvil |
| **Spreadsheet** | `1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y` (URL: https://docs.google.com/spreadsheets/d/1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y/edit) |
| **Apps Script ID** | `1fuA7kJHkWY6_4rM-IQHO9by6OLqVGZMDsSzra6n3D_6XRvMOWda4HQpg` |
| **GCP Project** | `gam-project-9wknn` (nº 899372363856) |
| **Owner script** | hello@ovav.dev (Alexander Salvador) |
| **Owner sheet** | igual |
| **Created** | 2026-09-18T06:11:35Z |
| **Last modified** | 2026-09-18T06:18:51Z |
| **Runtime** | V8 |
| **Timezone** | America/Bogota |

### Apps Script manifest (`appsscript.json`)

```json
{
  "timeZone": "America/Bogota",
  "dependencies": {},
  "exceptionLogging": "STACKDRIVER",
  "runtimeVersion": "V8",
  "webapp": {
    "executeAs": "USER_DEPLOYING",
    "access": "ANYONE_ANONYMOUS"
  }
}
```

⚠️ **Inconsistencias detectadas** vs CONFIG:
- `timezone`: dice **Bogota**, CONFIG dice **America/Lima**
- `webapp.access`: dice **ANYONE_ANONYMOUS**, CONFIG dice **ANY_LOGGED_IN_USER**
- `webapp.executeAs`: dice **USER_DEPLOYING** (sin anclar a un email específico)

---

## Arquitectura lógica

```
┌──────────────────────────────────────────────────────────────────┐
│ Apps Script (Proyecto sin título)                                │
│                                                                  │
│  onOpen() → menú CIMA con 7 items                               │
│                                                                  │
│  Sidebars:                                                       │
│    showCimaSidebar()    → HtmlService('Index')                   │
│    showAdminSidebar()   → HtmlService('Admin')                   │
│                                                                  │
│  WebApp:                                                         │
│    doGet()              → renderiza Index como webapp             │
│                                                                  │
│  RPC públicos (llamables vía Apps Script API):                    │
│    getInitialData()     → turns + operationTypes + actor         │
│    lookupAgent(dni)     → public agent profile                   │
│    appHealth()          → alias de diagnoseCima()                │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │ Triggers:                                                   │ │
│  │   onEdit(e)         INSTALL  — autocompleta PERSONAL        │ │
│  │   dailyClose()      every 15min (00:00–00:14)               │ │
│  │                       auto-reject PENDING si CONFIG=1        │ │
│  └─────────────────────────────────────────────────────────────┘ │
└────────────────────────┬─────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────────┐
│ Sheets (4 tabs)                                                  │
│   CIMA       → dashboard KPIs + pendientes                        │
│   PERSONAL   → maestro agentes (autocompleta grises por onEdit)  │
│   OPERACIONES → log único histórico (27 columnas)                │
│   CONFIG     → reglas + 10 turnos + políticas                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## Modelo de datos

### Columnas PERSONAL (16 cols, fila 4 = header)

```
DNI | Nombres completos | Teléfono | Correo personal | Correo corporativo |
Horario | Fecha ingreso | Modalidad | Estado administrativo | Supervisor |
TURNO_ID | Inicio | Fin | Segmento | Completitud | Estado CIMA
```

### Columnas OPERACIONES (26 cols, fila 4 = header)

```
REQUEST_ID | FECHA_OPERACION | TIPO | DNI | NOMBRE | HORARIO | TURNO_ID |
SEGMENTO | MODALIDAD | PRE_POST | HORAS | NUEVO_HORARIO | MOTIVO |
CONTRAPARTE | EMPRESA_EQUIPO | DESCANSO_ACTUAL | NUEVO_DESCANSO |
ADMIN_EVENTO | ESTADO | CREADO_EN | CREADO_POR | ACTUALIZADO_EN |
ACTUALIZADO_POR | OBSERVACION | GTR_EXPORT | DETALLE
```

### ESTADOS (`CIMA.STATUS`)

- `PENDING`
- `APPROVED`
- `REJECTED`
- `CANCELLED`
- `AUTO_REJECTED`

### TIPOS (`CIMA.TYPES`)

- `HORAS_EXTRA` (HE)
- `CAMBIO_HORARIO` (CH)
- `INTERCAMBIO_DESCANSO` (ID)
- `ADMINISTRATIVO` (AD)

---

## Funciones críticas (índice)

| Función | Archivo | Propósito |
|---|---|---|
| `installCima()` | Código.gs L56 | Bootstrap: guarda spreadsheet ID + instala triggers + refresh dashboard |
| `installTriggers_()` | L528 | Crea `dailyClose` cada 15 min |
| `dailyClose()` | L536 | Auto-rechaza PENDING a `AUTO_REJECTED` entre 00:00–00:14, 1×/día, si CONFIG=1 |
| `onEdit(e)` | L248 | Si PERSONAL editado → autocompleta TURNO_ID/Segmento/Completitud/Estado CIMA. Si CIMA col 9 row 11-20 → procesa APROBAR/RECHAZAR/CANCELAR |
| `refreshPersonalRow_(rowNumber)` | L202 | Recalcula campos grises de PERSONAL (turn_id, segment, completeness, status) |
| `createOperation(payload)` | L278 | Crea fila en OPERACIONES con `LockService` (10s timeout), valida, devuelve REQUEST_ID |
| `normalizeOperationPayload_(p)` | L310 | Normaliza fechas, DNIs, números |
| `validateOperation_(d)` | L329 | Aplica reglas de CONFIG: tipo permitido, DNI válido, fecha válida, OT ≤ Máx OT, OT enteros, etc. |
| `duplicateActiveOt_(dni, date)` | L350 | Si CONFIG "Una OT por persona/día"=1, rechaza duplicados activos |
| `changeOperationStatus(requestId, nextStatus)` | L360 | Valida transición PENDING→{APROVED,REJECTED,CANCELLED} o APPROVED→CANCELLED. Marca GTR_EXPORT=true si APROVED |
| `refreshDashboard()` | L467 | Recalcula KPIs en CIMA tab (B5-B7, D5-D7, F5-F7, H5-H7, B10:D13, A11:J20) |
| `diagnoseCima()` | L571 | Reporta sheets faltantes + triggers instalados |
| `exportApprovedGtr()` | L591 | Genera CSV en Drive folder `CIMA_GTR/` con operaciones APROVED + GTR_EXPORT=true, marca export=false después |
| `registerAdminEvent(payload)` | L435 | Solo admin (email matchea CONFIG["Supervisor email"]) registra ADMINISTRATIVO con STATUS=APPROVED |
| `lookupAgent(dni)` | L643 | WebApp RPC para ver perfil público del agente |
| `getInitialData()` | L635 | WebApp RPC: catálogos iniciales |

---

## Reglas de validación (extracto de `validateOperation_`)

```
Tipo permitido:   ∈ getOperationTypes_()  (lee CONFIG E12:G15)
DNI:              8 dígitos
Fecha:            parseable dd/mm/yyyy o yyyy-mm-dd
HORAS_EXTRA:      PRE|POST, horas enteras (si CONFIG.Horas enteras=1), 0 < h ≤ Máx OT diaria
CAMBIO_HORARIO:   new_schedule ∈ getTurnCatalog_()
INTERCAMBIO:      counterpart obligatorio, descansos válidos
ADMINISTRATIVO:   adminEvent obligatorio
```

---

## Hallazgos y discrepancias

### 1. Apps Script responde con HTML 404 al `projects.list`

El endpoint `GET /v1/projects` devuelve HTML genérico en vez de JSON, **incluso con scope `script.projects` válido y API habilitada**. El bridge usa `find --container` con fallback a `getContent` por scriptId — funciona cuando el scriptId se conoce.

**Workaround actual**: el CEO provee el scriptId manualmente desde la URL del editor.

### 2. HTML files no aparecen en `getContent`

El código referencia `Index.html` y `Admin.html` pero `getContent` solo devuelve 2 archivos (`appsscript` + `Código.gs`). Posibles causas:
- Los HTML están en un subdirectorio no enumerable
- Se agregan en runtime via `HtmlService.createTemplateFromFile()` leyendo de otra fuente
- El Apps Script está parcialmente deployado y esos archivos no existen aún

### 3. Timezone mismatch

- Manifest: `America/Bogota`
- CONFIG spreadsheet: `America/Lima`
- Sesión actual: `-0500` (Lima/Colombia)

### 4. Webapp access mismatch

- Manifest: `ANYONE_ANONYMOUS` (público sin login)
- CONFIG: `ANY_LOGGED_IN_USER` (requiere cuenta Google)

Si la webapp está deployada, cualquiera puede operarla. Si CONFIG se respeta, debería restringirse.

### 5. `webapp.executeAs = USER_DEPLOYING`

El script corre con permisos del deployer (no del usuario que opera la webapp). Las llamas a `Session.getActiveUser().getEmail()` devuelven el email del deployer, no del usuario real — esto rompe `registerAdminEvent()` y `isSheetAdmin_()`.

### 6. No hay versiones publicadas

`projects.versions` está disponible pero el CEO no ha hecho `snapshot --description "..."` nunca. Cero historial de versiones inmutable.

---

## Recomendaciones de seguridad

1. **Rotar el Client Secret de Google** — está expuesto en vault pero la OAuth screen solo tiene 1 test user. Ampliar a `Internal` o agregar `alexander.salvador.dev@gmail.com` como test user.
2. **Migrar vault a `Internal`** si es Workspace — evita pasar por consent screen.
3. **Documentar `webapp.executeAs = USER_DEPLOYING`** y entender implicancias.
4. **Auditar webapp access** vs CONFIG (`ANYONE_ANONYMOUS` vs `ANY_LOGGED_IN_USER`).
5. **Versionar Apps Script** antes de cualquier cambio OVAV: `scripts snapshot --description "baseline 2026-09-18"`.

---

## Cómo continuar

| Acción | Comando OVAV |
|---|---|
| Listar tabs | `ovav-sheets list` |
| Leer CIMA | `ovav-sheets table --tab CIMA` |
| Buscar agente | `ovav-sheets read --range "PERSONAL!A5:P5"` |
| Leer operaciones | `ovav-sheets table --tab OPERACIONES` |
| Actualizar CIMA | Disparar `refreshDashboard()` desde menú Apps Script |
| Exportar GTR | Disparar `exportApprovedGtr()` desde menú Apps Script |
| Pull Apps Script | `ovav-sheets scripts pull --id 1fuA7kJHkWY6_4rM-IQHO9by6OLqVGZMDsSzra6n3D_6XRvMOWda4HQpg --out ./appscript` |
| Snapshot Apps Script | `ovav-sheets scripts snapshot --id <SID> --description "v1.0 baseline"` |
| Ejecutar función | `ovav-sheets scripts run --id <SID> --function refreshDashboard` |
| Run webapp | abrir `https://script.google.com/macros/s/{DEPLOY_ID}/exec` |
