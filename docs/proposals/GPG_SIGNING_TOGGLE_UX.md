# UX Design Spec: GPG Signing Toggle — OVAV Cockpit

**Autor:** Elena (Frontend Engineer)  
**Fecha:** 2026-06-19  
**Versión:** 1.0 — Draft para revisión  
**Target:** Cockpit TUI (Go + Bubble Tea)  
**Integración:** OWS (`owd` compliance), `git log --format=%G?`

---

## 0. Resumen Ejecutivo

Actualmente GPG en OVAV funciona solo como **verificación pasiva de compliance** al ejecutar `owd`: si el compliance level es `strict` o `maximum`, se cuentan commits firmados mediante `countSignedCommits()` y se bloquea el merge si ninguno está firmado. Pero **no hay superficie de control en el Cockpit** para activar/desactivar la firma, ver su estado, ni atribuir firmas a autores específicos. El usuario debe ejecutar `git config user.signingkey` manualmente — una fricción que rompe la experiencia integrada de OVAV.

Este diseño agrega **3 superficies de UX** al Cockpit:

| Superficie | Tipo | Propósito |
|---|---|---|
| **Status bar** (global) | Indicador persistente | Ver estado GPG en cualquier vista sin navegar |
| **Security panel** (nueva vista) | Configuración | Toggle on/off, selección de key, diagnóstico, actividad reciente |
| **owd pre-merge card** | Verificación | Atribución por commit, bloqueo visual cuando hay commits sin firma |

---

## 1. Status Bar — Indicador Global

### 1.1 Ubicación

Línea de ayuda inferior (`renderHelpBar`) en **todas las vistas** del Cockpit. Se añade un segmento compacto al extremo izquierdo, antes de los atajos de teclado.

```
🔏 gpg:thavren@ovav          ↑↓: Navigate  •  Enter: Detail  •  Esc: Back  •  ?: Help
```

### 1.2 Estados Visuales y Lógica

| Indicador | Color (lipgloss) | Condición |
|---|---|---|
| `🔏 gpg:thavren@ovav` | Green (`#22C55E`) | `user.signingkey` configurado + `commit.gpgsign=true` |
| `⚪ gpg:off` | Gray (`#9CA3AF`) | `commit.gpgsign=false` o no configurado (key sí existe) |
| `🔴 gpg:no-key` | Red (`#EF4444`) | `user.signingkey` vacío → no hay key configurada |
| `🟡 gpg:expires 3d` | Yellow (`#F59E0B`) | Key expira en <14 días (warning no bloqueante) |
| `🔴 gpg:expired` | Red (`#EF4444`) | Key expirada → commits NO se firman (aunque `commit.gpgsign=true`) |
| `⚪ gpg:?` | Muted (`#6B7280`) | No se pudo determinar (gpg no instalado, o error al leer config) |

### 1.3 Árbol de decisión

```
gpg --list-secret-keys → vacío?
  ├─ SÍ → 🔴 gpg:no-key
  └─ NO  → user.signingkey configurado?
              ├─ NO  → 🔴 gpg:no-key
              └─ SÍ  → commit.gpgsign?
                         ├─ false → ⚪ gpg:off
                         └─ true  → gpg --list-keys $SIGNINGKEY
                                      ├─ expirada → 🔴 gpg:expired
                                      ├─ expira <14d → 🟡 gpg:expires Nd
                                      └─ válida → 🔏 gpg:$email
```

### 1.4 Implementación

- Se añade `GPGInfo` al `Model` (struct en `model.go`)
- Se refresca en cada render (cache local de 5s para no llamar `gpg` en cada frame)
- `renderHelpBar` acepta un parámetro opcional `gpgStatus string` o se lee de `m.gpgInfo`

---

## 2. Security Panel — Vista Nueva

### 2.1 Entrada en el Menú Principal

Sexto ítem en el `Root Menu`, categoría `OVAV`:

```go
{id: "security", label: "Security & Signing", icon: "🔐", category: "OVAV", 
 desc: "GPG signing, compliance level, key management"},
```

**Navegación:** `Root → ↓ hasta Security → Enter`  
**Atajo directo:** `S` desde Root (key binding global)  
**Vuelta:** `Esc` → Root

### 2.2 Layout del Panel (mock textual)

```
┌──────────────────────────────────────────────────────────────────┐
│  OVAV Cockpit  —  Security & Signing                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─ GPG Commit Signing ──────────────────────────────────────┐  │
│  │                                                            │  │
│  │  Status:  🟢 ENABLED                                      │  │
│  │  Signing as:  thavren@ovav                                │  │
│  │  Key ID:  ABC12345DEF67890                                │  │
│  │  Key type:  RSA 4096 / Ed25519                            │  │
│  │  Created:   2026-01-15                                    │  │
│  │  Expires:   never  (or: 2027-03-15, 288 days remaining)   │  │
│  │                                                            │  │
│  │  ┌──────────────────────┐    ┌──────────────────────┐     │  │
│  │  │   🔓 Disable GPG     │    │   🔄 Change Key...    │     │  │
│  │  └──────────────────────┘    └──────────────────────┘     │  │
│  │                                                            │  │
│  │  git config: commit.gpgsign = true                         │  │
│  │              user.signingkey = ABC12345DEF67890            │  │
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─ Recent Signature Activity ───────────────────────────────┐  │
│  │  Branch: feature/sprint-1.4  (3 commits ahead of develop)  │  │
│  │                                                            │  │
│  │  ▸ d4e5f6a  feat(ows): S1-1 hooks        ✅ thavren@ovav │  │
│  │    aa0b90d  docs: auto-gen CHANGELOG      ✅ thavren@ovav │  │
│  │    843feff  chore(py): eliminar worktree  ⚪ UNSIGNED     │  │
│  │                                                            │  │
│  │  Summary: 2/3 signed  ⚠️ 1 unsigned                       │  │
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌─ Compliance Policy ───────────────────────────────────────┐  │
│  │                                                            │  │
│  │  Level:     strict                                         │  │
│  │  GPG:       ✅ Required (blocking merge if unsigned)       │  │
│  │  Secrets:   ✅ Required                                    │  │
│  │  Validate:  ✅ Required (77 validators)                    │  │
│  │  Reviewer:  ✅ Required (lead approval)                    │  │
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Tab: Switch panel  •  ↑↓: Navigate commits  •  Enter: Action   │
│  T: Toggle GPG  •  K: Change key  •  Esc: Back  •  ?: Help      │
└──────────────────────────────────────────────────────────────────┘
```

### 2.3 Interacciones por Panel

#### Panel 1: GPG Signing Card

| Tecla/Acción | Resultado |
|---|---|
| `Enter` en "Disable/Enable GPG" | Toggle `commit.gpgsign` → confirmación modal |
| `Enter` en "Change Key" | Abre selector de keys (panel 1.1) |
| `T` | Toggle directo sin mover foco |

#### Panel 2: Signature Activity

| Tecla/Acción | Resultado |
|---|---|
| `↑/↓` | Navegar lista de commits |
| `Enter` en commit | Ver detalle: hash completo, timestamp, key fingerprint |
| `S` en commit unsigned | Sugerir `git commit --amend -S` para firmar |

#### Panel 3: Compliance Policy

- Solo lectura. Muestra los requisitos del compliance level actual.
- El nivel se configura desde `owd` o desde `caps.yaml`. No se cambia desde aquí.

### 2.4 Flujo de Toggle (Disable/Enable)

```
Estado: ENABLED (🟢)
Usuario presiona Enter en "Disable GPG" o tecla T
  ↓
┌─ Popup Modal ──────────────────────────────────────────────┐
│                                                             │
│  ⚠️  Disable GPG commit signing?                           │
│                                                             │
│  Commits will NOT be cryptographically signed.              │
│  Merges at strict/maximum compliance WILL be blocked.       │
│                                                             │
│  This affects ALL future commits in this repository.        │
│                                                             │
│       ┌──────────────┐          ┌──────────────┐           │
│       │  ✅ Confirm  │          │  ❌ Cancel   │           │
│       └──────────────┘          └──────────────┘           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
  ↓ Confirm
git config commit.gpgsign false
  ↓
Status bar: 🟢 ENABLED → ⚪ DISABLED
Botón cambia: "Disable GPG" → "Enable GPG"
Toast temporal (3s): "⚪ GPG signing disabled — commits will not be signed"
```

### 2.5 Selector de Key (Change Key)

```
┌─ Select GPG Signing Key ────────────────────────────────────┐
│                                                              │
│  Available secret keys in keyring:                           │
│                                                              │
│  ▸ ABC12345  thavren@ovav              RSA 4096  (active)   │
│    DEF67890  thavren-backup@ovav       RSA 2048  expires 5d │
│    GHI11111  thavren@personal.com      Ed25519   (external) │
│                                                              │
│  ─────────────────────────────────────────────────────────── │
│  Active key will be set via:                                 │
│    git config --global user.signingkey <KEYID>               │
│                                                              │
│  Enter: Select  •  Esc: Cancel  •  I: Inspect key details    │
└──────────────────────────────────────────────────────────────┘
```

**Reglas de filtrado:**
- Solo muestra keys con `sec` (secret key available) en `gpg --list-secret-keys --with-colons`
- Keys con email externo (no `@ovav`) se marcan como `(external)` — son seleccionables pero con advertencia
- Keys expiradas se muestran en rojo, no seleccionables
- Keys que expiran en <14 días se marcan con `⚠️`

---

## 3. Modelo de Firma por Equipo

### 3.1 Decisión de Diseño

**Cada miembro del equipo firma con SU propia key GPG.** El lead (Thavren) NO firma por otros miembros.

### 3.2 Fundamento

1. **No repudiabilidad criptográfica:** La firma GPG vincula irreversiblemente un commit con su autor real. Si Thavren firmara commits de Sergio, se perdería la trazabilidad — en un audit, no se podría probar quién escribió realmente el código.
2. **Git lo soporta nativamente:** `git log --format=%G?` verifica la firma de cada commit individual. `%GS` muestra el signer, `%GK` muestra la key fingerprint.
3. **Modelo estándar de la industria:** Linux kernel, GitHub verified commits, y GitLab signed commits — todos usan firma individual, no firma delegada.

### 3.3 Flujo Completo del Equipo

```
┌─────────────────────────────────────────────────────────────────┐
│                       owc feature/login                          │
│                  Thavren crea el worktree                        │
└─────────────────────────────┬───────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
    ┌──────────┐        ┌──────────┐        ┌──────────┐
    │  Sergio  │        │  Elena   │        │  Dante   │
    │ backend  │        │ frontend │        │ product  │
    └────┬─────┘        └────┬─────┘        └────┬─────┘
         │                   │                   │
         │ git commit -S     │ git commit -S     │ git commit -S
         │ (auto via hook)   │ (auto via hook)   │ (auto via hook)
         ▼                   ▼                   ▼
    ┌──────────┐        ┌──────────┐        ┌──────────┐
    │ a1b2c3d  │        │ e4f5g6h  │        │ i7j8k9l  │
    │ ✅ sergio│        │ ✅ elena │        │ ✅ dante │
    │ @ovav    │        │ @ovav    │        │ @ovav    │
    └────┬─────┘        └────┬─────┘        └────┬─────┘
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   owv verify     │
                    │  go test + vet   │
                    │  secrets sweep   │
                    │  77 validators   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    owd done      │
                    │  (strict level)  │
                    │                  │
                    │  🔍 GPG Audit:   │
                    │  a1b2c3d ✅      │
                    │  e4f5g6h ✅      │
                    │  i7j8k9l ✅      │
                    │                  │
                    │  3/3 signed ✅   │
                    │  MERGE ALLOWED   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  🔏 SEAL GENERADO │
                    │  Sigs: 3         │
                    │  Hash: a1b2...   │
                    └─────────────────┘
```

### 3.4 Política de Firmantes Autorizados

Archivo canónico: `.ovav/policy/authorized_signers.yaml`

```yaml
# Firmantes autorizados para commits GPG en este repositorio OVAV
# Solo las keys listadas aquí son aceptadas en compliance strict/maximum

version: 1
updated_at: 2026-06-19
updated_by: thavren

signers:
  - email: thavren@ovav
    name: Thavren
    key_id: ABC12345DEF67890
    fingerprint: 1234 5678 90AB CDEF 1234 5678 90AB CDEF 1234 5678
    roles: [lead, reviewer, platform]
    status: active

  - email: sergio@ovav
    name: Sergio
    key_id: FED9876543210ABC
    fingerprint: FEDC BA98 7654 3210 FEDC BA98 7654 3210 FEDC BA98
    roles: [backend]
    status: active

  - email: elena@ovav
    name: Elena Frontend
    key_id: AAA1111BBBB2222
    fingerprint: AAAA 1111 BBBB 2222 AAAA 1111 BBBB 2222 AAAA 1111
    roles: [frontend]
    status: active

  - email: dante@ovav
    name: Dante
    key_id: CCC3333DDDD4444
    fingerprint: CCCC 3333 DDDD 4444 CCCC 3333 DDDD 4444 CCCC 3333
    roles: [lead, product]
    status: active
```

**Reglas de validación en `owd`:**
- Commit firmado con key en la whitelist → ✅
- Commit firmado con key válida pero NO en la whitelist → ⚠️ `UNKNOWN SIGNER` (advierte, no bloquea en standard, bloquea en strict/maximum)
- Commit sin firma → ⚪ `UNSIGNED` (bloquea en strict/maximum)
- Commit firmado con key expirada → 🔴 `EXPIRED KEY` (bloquea siempre)
- Commit firmado con key revocada → 🔴 `REVOKED KEY` (bloquea siempre)

---

## 4. Integración owd — Pre-merge Signature Display

### 4.1 Estado Actual

```text
🔍 GPG: checking commit signatures...
   ⚠️  No GPG-signed commits found
owd: GPG signing required — compliance strict mandates signed commits
```

Problema: el usuario no sabe **cuál** commit no está firmado, ni **quién** lo commiteó.

### 4.2 Propuesta: Tabla de Atribución

```
🔍 GPG Signature Audit — feature/sprint-1.4 (vs develop)

  ┌──────────┬──────────────────────────────┬──────────────────────┐
  │ Status   │ Commit                       │ Signer               │
  ├──────────┼──────────────────────────────┼──────────────────────┤
  │ ✅       │ d4e5f6a feat(ows): S1-1      │ thavren@ovav         │
  │ ✅       │ aa0b90d docs: changelog       │ thavren@ovav         │
  │ ⚪       │ 843feff chore(py): cleanup    │ (unsigned)           │
  └──────────┴──────────────────────────────┴──────────────────────┘

  ❌ owd BLOCKED: 1 unsigned commit found (843feff).
     Compliance level STRICT requires ALL commits signed.
     
     To fix: git checkout 843feff && git commit --amend --no-edit -S
```

### 4.3 Versión Cockpit (Health View Card)

Cuando el usuario está en el Cockpit y hay worktree activa, la vista Health muestra una card:

```
┌─ Pre-Merge GPG Status ───────────────────────────────────────┐
│                                                               │
│  🚦 STRICT compliance — all commits must be signed            │
│                                                               │
│  feature/sprint-1.4 → develop  (3 commits):                   │
│    ✅ d4e5f6a  thavren@ovav    hace 32 min                    │
│    ✅ aa0b90d  thavren@ovav    hace 1h 29min                  │
│    ⚠️ 843feff  UNSIGNED        hace 1h 29min                  │
│                                                               │
│  2/3 signed  ⚠️ MERGE WOULD BE BLOCKED                       │
│                                                               │
│  Actions:                                                     │
│    [S] Sign remaining commits (amend -S)                      │
│    [O] Override (waiver required — CEO only)                  │
│    [C] Cancel merge                                           │
│                                                               │
└───────────────────────────────────────────────────────────────┘
```

---

## 5. Estados de Error y Recuperación

| Estado | Detección | Mensaje en UI | Acción de Recuperación |
|---|---|---|---|
| **gpg no instalado** | `exec.LookPath("gpg")` → error | `🔴 GPG not installed` | Mostrar instrucciones: `brew install gnupg` / `apt install gnupg` |
| **Key no generada** | `gpg --list-secret-keys` vacío | `🔴 No GPG key found in keyring` | Botón: "Generate Key" → wizard interactivo (`gpg --gen-key`) |
| **Key no configurada en git** | `user.signingkey` vacío | `🔴 GPG key exists but not linked to git` | Botón: "Link Key" → selector de keys → `git config user.signingkey` |
| **commit.gpgsign = false** | `git config commit.gpgsign` → `false` | `⚪ GPG signing is OFF` | Botón toggle en Security panel |
| **Key expirada** | `gpg --list-keys --with-colons` → `e` flag | `🔴 Key EXPIRED on 2026-06-01 (18 days ago)` | Botón: "Extend Key" → `gpg --edit-key $KEYID expire` |
| **Key expira pronto** | <14 días | `🟡 Key expires in 5 days (2026-06-24)` | Warning amarillo no bloqueante. Botón: "Extend Now" |
| **Key revocada** | Revocation certificate presente | `🔴 Key has been REVOKED — cannot be used` | Sugerir generar nueva key. La key revocada no se puede recuperar. |
| **Email no coincide con OVAV** | Key email ≠ `*@ovav` | `🟡 Signing as external@email.com (not OVAV identity)` | Warning. La firma es criptográficamente válida pero no coincide con la identidad OVAV. En strict/maximum puede ser rechazada. |
| **Firma inválida (commit corrupto)** | `%G?` → `B` (bad) | `🔴 BAD signature on commit a1b2c3d` | El commit tiene una firma que no verifica contra el contenido. Posible manipulación. Bloquea siempre. |
| **No se pudo verificar (key no en keyring)** | `%G?` → `E` (error) o `N` (no key) | `⚠️ Cannot verify: key not in local keyring` | El commit está firmado con una key que no tienes importada. Sugerir `gpg --recv-keys`. |
| **Worktree sin commits propios** | `git log branch --not develop` → vacío | `⚪ No commits to sign yet` | No aplica. Hacer commit primero. |
| **Conflicto de keys entre worktrees** | Diferentes worktrees con diferente `user.signingkey` | `⚠️ Multiple signing keys detected across worktrees` | Warning informativo. No bloquea. |

---

## 6. Diagrama de Estados Completo

```
                         ┌─────────────────────────┐
                         │    INIT: Unknown state   │
                         └───────────┬─────────────┘
                                     │
                         ┌───────────▼──────────────┐
                         │  Check gpg binary exists  │
                         └───┬─────────────────┬────┘
                             │ NO              │ YES
                             ▼                 ▼
                     ┌──────────────┐  ┌──────────────────┐
                     │ 🔴 NO_GPG    │  │ Check secret keys │
                     │ Install gpg  │  └────────┬─────────┘
                     │ via package   │           │
                     │ manager       │  ┌────────┼──────────┐
                     └──────────────┘  │ NONE   │ KEYS     │
                                       ▼        ▼          │
                               ┌──────────┐ ┌──────────────────┐
                               │🔴 NO_KEY │ │ Check signingkey │
                               │ gen-key  │ │ in git config    │
                               └──────────┘ └────────┬─────────┘
                                                     │
                                        ┌────────────┼────────────┐
                                        │ NOT SET    │ SET        │
                                        ▼            ▼            │
                                ┌──────────┐  ┌──────────────────┐
                                │🔴 UNLINKED│ │ Check key expiry │
                                │ link key  │ └────────┬─────────┘
                                └──────────┘          │
                                            ┌─────────┼─────────┐
                                            │ EXPIRED │ VALID   │
                                            ▼         │         │
                                    ┌──────────┐     │         │
                                    │🔴 EXPIRED│     │         │
                                    │ extend   │     │         │
                                    │ or gen   │     │         │
                                    │ new key  │     │         │
                                    └──────────┘     │         │
                                            ┌────────┼────────┐│
                                            │ REVOKED│        ││
                                            ▼        │        ││
                                    ┌──────────────┐ │        ││
                                    │🔴 REVOKED    │ │        ││
                                    │ gen new key  │ │        ││
                                    └──────────────┘ │        ││
                                                     │        ││
                                         ┌───────────┼────────┼┘
                                         │           │        │
                                    ┌────▼────┐ ┌───▼────┐   │
                                    │commit.  │ │commit.  │   │
                                    │gpgsign  │ │gpgsign  │   │
                                    │= false  │ │= true   │   │
                                    └────┬────┘ └───┬─────┘   │
                                         │          │         │
                                         ▼          ▼         │
                                  ┌──────────┐ ┌──────────────┤
                                  │⚪ DISABLED│ │🟢 ENABLED    │
                                  │ toggle ON│ │🟡/🟢 expiry  │
                                  └──────────┘ │ verify       │
                                               │ commit.gpgsign│
                                               └──────────────┘
```

---

## 7. Especificación de Implementación (Referencia para Devs)

### 7.1 Nuevas Funciones en `internal/cli/`

| Función | Retorno | Fuente de datos |
|---|---|---|
| `GPGStatus()` | `GPGInfo` struct | `gpg --list-secret-keys --with-colons` + `git config` |
| `GPGListSignatures(branch, baseBranch)` | `[]CommitSig` | `git log --format=%H|%G?|%GS|%GK --not <base>` |
| `GPGToggle(enable bool)` | `error` | `git config commit.gpgsign true/false` |
| `GPGSetKey(keyID string)` | `error` | `git config user.signingkey KEYID` |
| `GPGAvailableKeys()` | `[]GPGKey` | `gpg --list-secret-keys --with-colons` |

### 7.2 Estructuras de Datos

```go
type GPGInfo struct {
    Installed    bool      // gpg binary found in PATH
    HasSecretKey bool      // at least one secret key in keyring
    KeyID        string    // configured signing key (from git config)
    Email        string    // key UID email
    KeyType      string    // RSA 4096, Ed25519, etc.
    Fingerprint  string    // full fingerprint
    CreatedAt    time.Time
    ExpiresAt    time.Time // zero if never expires
    IsExpired    bool
    ExpiresSoon  bool      // < 14 days
    DaysRemaining int      // days until expiry
    Enabled      bool      // commit.gpgsign == true
    Canonical    bool      // email matches OVAV domain (*@ovav)
}

type CommitSig struct {
    Hash      string
    Status    string // G=good, B=bad, U=unknown validity, N=no signature, E=error
    Signer    string // %GS
    KeyID     string // %GK
    Valid     bool   // Status == "G"
}
```

### 7.3 Archivos de Cockpit Afectados

| Archivo | Cambio |
|---|---|
| `model.go` | Añadir `gpgInfo GPGInfo` al struct Model. Refrescar en `Init()` y en cada `Update()` con cache de 5s. |
| `view.go` | Añadir `ViewSecurity` a la ruta del switch en `renderCurrentView()`. |
| `util.go` | Añadir constante `ViewSecurity = "security"`. Modificar `renderHelpBar()` para aceptar segmento GPG opcional. |
| `root.go` | Añadir `menuItem{id: "security", ...}` al slice `menuItems`. Añadir case `"security"` en `rootUpdate()`. Añadir shortcut global `S`. |
| `security.go` | **NUEVO** — Vista Security panel con tres sub-paneles: GPG Card, Activity Card, Policy Card. Manejar `securityUpdate()` y `renderSecurity()`. |
| `health.go` | Añadir card GPG compacta en `renderHealth()` (solo indicador + botón rápido). |
| `data/system.go` | Añadir campo `GPGEnabled bool` y `GPGSigner string` a `SystemInfo`. O bien crear `data/gpg.go` separado. |
| `data/gpg.go` | **NUEVO** — `GatherGPGInfo(root string) GPGInfo` usando `internal/cli`. |
| `styles/theme.go` | Añadir `GPGBorder` style (usando Cyan), `GPGBadgeEnabled`, `GPGBadgeDisabled`, `GPGBadgeError`. |

### 7.4 Nuevos Atajos de Teclado

| Tecla | Contexto | Acción |
|---|---|---|
| `S` | Root | Jump directo a Security view |
| `T` | Security | Toggle GPG on/off (sin confirmación si ya advirtió antes) |
| `K` | Security | Abrir selector de key |
| `Enter` | Security → Botón | Acción del botón enfocado |
| `Tab` | Security | Rotar foco entre paneles (GPG → Activity → Policy → GPG) |
| `g` + `s` | Cualquier vista | Quick popup overlay con estado GPG (3s) |

---

## 8. Prioridades y Estimaciones

| P | Componente | Estimación | Dependencias | Impacto |
|---|---|---|---|---|
| **P0** | Status bar GPG indicator | 2h | `internal/cli: GPGStatus()` | Visibilidad inmediata en TODO el Cockpit |
| **P0** | Security panel — toggle + key display | 4h | `internal/cli: GPGStatus(), GPGToggle()` | Feature core: prender/apagar GPG |
| **P1** | owd pre-merge per-commit display | 3h | `internal/cli: GPGListSignatures()` | Atribución + bloqueo visual |
| **P1** | Key selector (Change Key) | 2h | `internal/cli: GPGAvailableKeys(), GPGSetKey()` | UX completa de gestión de keys |
| **P2** | Expiry warnings + recovery flows | 2h | P0 + P1 | Robustez ante keys que expiran |
| **P2** | `authorized_signers.yaml` + whitelist validation | 3h | Spec de seguridad de Thavren | Política de equipo |
| **P2** | Quick popup overlay (`g s`) | 1h | P0 | Acceso rápido sin navegar |

---

## 9. Notas para Revisión de Leads

- **Dante (Lead Product):** ¿El Security panel va como ítem de Root o como sub-panel de Health? Recomiendo Root por visibilidad y porque crecerá (más settings de seguridad en el futuro).
- **Thavren (Platform Lead):** La whitelist `authorized_signers.yaml` necesita tu spec de seguridad. El formato YAML propuesto es un draft.
- **Cumplimiento de AGENTS.md:** Esta es una tarea de diseño UX. La implementación Go la hará Sergio (Backend) para `internal/cli/gpg.go`. Las vistas Bubble Tea las implementaré yo (Elena Frontend). Sin embargo, el Cockpit usa Go no JS — técnicamente es full-stack Go. Propongo que Sergio haga el CLI y yo las vistas, coordinando la interfaz `GPGInfo`.
- **Elena UI/UX Design Lead:** Este documento es la especificación funcional. Si se requiere un design system review (accesibilidad, jerarquía visual), queda pendiente de su revisión. Yo me enfoqué en la experiencia de uso (flujos, estados, interacciones) que es mi expertise.

---

*Documento generado como especificación de diseño UX para implementación. No contiene código ejecutable. Versión canónica en `docs/proposals/GPG_SIGNING_TOGGLE_UX.md`.*
