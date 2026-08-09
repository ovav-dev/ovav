# OVAV Vault 2026 — Secrets Subsystem
# =============================================================================
# Plan: Autonomous execution at full power for 2026
# CEO: Braka | Lead: Thavren | Created: 2026-08-03
# =============================================================================

## Resumen Ejecutivo

Sistema nativo de gestión de credenciales para OVAV SYSTEMS. Compite con
Bitwarden/1Password en seguridad y UX, pero opera exclusivamente dentro del
ecosistema OVAV. Sin dependencias externas. AES-256-GCM nativo.

**Scope 2026:**
- Discovery automático + ingesta manual
- Segmentación inteligente por tipo de secreto
- Credential health activa
- Audit log + rotación programada
- Backups cifrados

---

## Arquitectura del Subsystem

```
go-runtime/internal/vault/secrets/     ← nuevo paquete
├── secrets.go                         ← tipos + interfaces
├── discovery.go                       ← scan automático filesystem
├── providers/
│   ├── github.go                      ← GitHub Secrets API
│   ├── fly.go                         ← Fly.io secrets API
│   ├── cloudflare.go                  ← CF API + Tunnel tokens
│   ├── firebase.go                    ← Firebase .env
│   ├── resend.go                      ← Resend API keys
│   └── oauth.go                       ← Google/GitHub OAuth
├── health.go                          ← credential health checks
├── audit.go                           ← access audit log
├── rotation.go                        ← rotation reminders
└── backup.go                          ← encrypted backup

cmd/ovav-vault-secrets/               ← binario standalone
cmd/ovav/ vault_* commands            ← integrado en ovav main
```

**Decisión de acoplamiento:** El subsystem usa las misma clave AES-256 del
vault existente (vault_key_export). Si se retira el subsystem, ovav central
sigue funcionando — los .enc de profiles/agents/skills no se tocan.

---

## Segmentación Inteligente — 7 Tipos de Secreto

| Tipo | Descripción | Ejemplos |
|------|-------------|---------|
| `api_token` | Tokens de API de terceros | CF_API_TOKEN, FLY_API_TOKEN, RESEND_API_KEY |
| `oauth_creds` | Client ID + Secret OAuth | OAUTH_GOOGLE_CLIENT_ID+SECRET |
| `db_credential` | Conexiones de base de datos | MySQL/PostgreSQL passwords |
| `cloud_key` | Llaves cloud | AWS access keys, GCP keys |
| `encryption_key` | Llaves de cifrado simétrico | HMAC_SECRET, JWT_SECRET |
| `user_secret` | Secrets de usuario final | Firebase API keys, DNI_API_KEY |
| `tunnel_token` | Tunnel credentials | TUNNEL_TOKEN (Cloudflare Tunnel) |

---

## CLI Interface

```bash
# Standalone
go run ./cmd/ovav-vault-secrets/ <cmd>

# Integrado en ovav
go run ./cmd/ovav/ vault secrets <cmd>

# Comandos
vault secrets discover        # Scan automático (GitHub, Fly, filesystem)
vault secrets add --type api_token --name "CF prod" --value $CF_API_TOKEN
vault secrets list --type api_token
vault secrets health          # Verificar todos los secretos vivos
vault secrets audit           # Ver log de accesos
vault secrets rotate <name>   # Generar nuevo, actualizar source
vault secrets backup           # Exportar .enc a storage
vault secrets restore         # Restaurar desde backup
```

---

## Fases de Implementación

```
Phase 1: Fundation  (semana 1-2)
Phase 2: Discovery  (semana 3-4)
Phase 3: Health     (semana 5-6)
Phase 4: Audit+Rot   (semana 7-8)
Phase 5: Backup+UX   (semana 9-10)
```

---

## Phase 1: Foundation ✅

**Goal:** Tipos + interfaz + almacenamiento base

### Tarea 1.1: Tipos y almacenamiento

**File:** `go-runtime/internal/vault/secrets/secrets.go`

```go
package secrets

type SecretType string

const (
    TypeAPIToken      SecretType = "api_token"
    TypeOAuthCreds    SecretType = "oauth_creds"
    TypeDBCredential  SecretType = "db_credential"
    TypeCloudKey      SecretType = "cloud_key"
    TypeEncryptionKey SecretType = "encryption_key"
    TypeUserSecret    SecretType = "user_secret"
    TypeTunnelToken   SecretType = "tunnel_token"
)

type Secret struct {
    ID        string      `json:"id"`         // uuid
    Name      string      `json:"name"`       // "CF production API token"
    Type      SecretType  `json:"type"`
    Provider  string      `json:"provider"`    // "cloudflare" | "fly.io" | "github" | etc
    Source    string      `json:"source"`      // "github_secrets" | "fly_secrets" | "manual"
    Value     []byte      `json:"-"`           // nunca serializar
    Hash      string      `json:"hash"`        // SHA256 para verificación
    CreatedAt time.Time   `json:"created_at"`
    ExpiresAt *time.Time  `json:"expires_at,omitempty"`
    LastUsed  *time.Time  `json:"last_used,omitempty"`
    Rotatable bool        `json:"rotatable"`   // puede regenerarse automáticamente
    Metadata  Metadata    `json:"metadata"`    // provider-specific
}

type Metadata map[string]string

type SecretStore struct {
   mu    sync.RWMutex
   secrets map[string]*Secret  // id -> secret
}
```

**Storage format:** `~/.local/share/ovav/secrets.vault`
- AES-256-GCM encrypted JSON blob
- Same key derivation as existing vault (PBKDF2-HMAC-SHA256)
- Version field for future key rotation

- [ ] **Step 1: Crear estructura de tipos en `secrets.go`**
- [ ] **Step 2: Implementar `SecretStore` con mutex + CRUD básico**
- [ ] **Step 3: Guardar/recuperar desde `~/.local/share/ovav/secrets.vault`**
- [ ] **Step 4: Integrar con vault key existente (no generar nueva clave)**
- [ ] **Step 5: Commit**

### Tarea 1.2: CLI básico

**File:** `go-runtime/cmd/ovav-vault-secrets/main.go`

- [x] **Step 1: Estructura básica con cobra/flag**
- [x] **Step 2: `vault secrets add --type --name --value`**
- [x] **Step 3: `vault secrets list --type --json`**
- [x] **Step 4: `vault secrets get --id --show` (con confirmación)**
- [x] **Step 5: `vault secrets remove --id`**
- [x] **Step 6: Commit**

---

## Phase 2: Discovery Automático ✅

**Goal:** Scanear credenciales desde GitHub, Fly.io y filesystem

### Tarea 2.1: GitHub Secrets Discovery

**File:** `go-runtime/internal/vault/secrets/providers/github.go`

```go
func DiscoverGitHubSecrets(ctx context.Context, owner, repo string) ([]*Secret, error)
```

- Lista secretos desde GitHub API (`GET /repos/{owner}/{repo}/actions/secrets`)
- No intenta leer valores (están encriptados en GitHub)
- Registra: nombre, tipo inferido, última actualización
- Fallback: leer desde `.env` en repos locales

- [x] **Step 1: GitHub API paginated list de secrets**
- [x] **Step 2: Inferir SecretType desde nombre del secreto**
- [x] **Step 3: Commit**

### Tarea 2.2: Fly.io Secrets Discovery

**File:** `go-runtime/internal/vault/secrets/providers/fly.go`

```go
func DiscoverFlySecrets(ctx context.Context, appName string) ([]*Secret, error)
```

- Usa `flyctl secrets list` o API directa
- Registra todos los secretos del app
- Detecta TUNNEL_TOKEN, FLY_API_TOKEN, OAUTH_*, RESEND_*

- [ ] **Step 1: Integración con fly CLI**
- [ ] **Step 2: Clasificador automático por nombre**
- [ ] **Step 3: Commit**

### Tarea 2.3: Filesystem Discovery

**File:** `go-runtime/internal/vault/secrets/discovery.go`

```go
type DiscoveryConfig struct {
    SearchPaths []string
    ExcludeDirs []string
}

func DiscoverSecrets(ctx context.Context, cfg DiscoveryConfig) ([]*Secret, error)
```

Patterns escaneados:
- `**/.env` → busca API_KEY, TOKEN, SECRET, PASSWORD
- `**/.env.production` → prioriza sobre .env
- `**/fly.toml` → registra app name
- `**/wrangler.toml` → registra CF workers config

- [ ] **Step 1: Glob .env files en ~/Systems/**
- [ ] **Step 2: Parser de clave=valor con detección de tipo**
- [ ] **Step 3: Integración con todos los providers**
- [ ] **Step 4: Commit**

---

## Phase 3: Credential Health ✅

**Goal:** Verificación activa + alertamiento

### Tarea 3.1: Health Checker

**File:** `go-runtime/internal/vault/secrets/health.go`

```go
type HealthStatus string

const (
    HealthOK         HealthStatus = "ok"
    HealthExpiring   HealthStatus = "expiring"
    HealthExpired    HealthStatus = "expired"
    HealthUnreachable HealthStatus = "unreachable"
    HealthRotated    HealthStatus = "rotated"
)

type HealthReport struct {
    SecretID  string      `json:"secret_id"`
    Name      string      `json:"name"`
    Type      SecretType  `json:"type"`
    Provider  string      `json:"provider"`
    Status    HealthStatus `json:"status"`
    Message   string      `json:"message,omitempty"`
    LastCheck time.Time   `json:"last_check"`
}
```

Verificaciones por tipo:
- `api_token`: HEAD request al endpoint del provider
- `oauth_creds`: OAuth introspection endpoint o test request
- `tunnel_token`: `cloudflared tunnel info` o DNS check
- `db_credential`: conexión de prueba (timeout 5s)

- [ ] **Step 1: Health checker framework**
- [ ] **Step 2: Implementación por tipo (CF, Fly, Firebase, Resend)**
- [ ] **Step 3: Commit**

### Tarea 3.2: Alertas + Notificaciones

- [ ] **Integración con `go run ./cmd/ovav/ alerts` (nuevo comando)**
- [ ] **Configuración: days_before_expiry para cada tipo**
- [ ] **Commit**

---

## Phase 4: Audit Log + Rotation ✅

### Tarea 4.1: Audit Log

**File:** `go-runtime/internal/vault/secrets/audit.go`

```go
type AuditEntry struct {
    ID        string    `json:"id"`
    SecretID  string    `json:"secret_id"`
    SecretName string   `json:"secret_name"`
    Action    string    `json:"action"` // "read" | "add" | "delete" | "rotate" | "export"
    Actor     string    `json:"actor"`  // "user" | "workflow" | "auto-rotate"
    Timestamp time.Time `json:"timestamp"`
    IP        string    `json:"ip,omitempty"`
}
```

- Append-only log en `~/.local/share/ovav/secrets_audit.log`
- Cifrado opcional (AES-256-GCM bundle, nuevo .enc)
- Query: `vault secrets audit --secret-id X --since 7d`

- [ ] **Step 1: Estructura AuditEntry + log writer**
- [ ] **Step 2: `vault secrets audit --since --secret-id --format json|table`**
- [ ] **Step 3: Commit**

### Tarea 4.2: Rotation Engine

```go
type RotationConfig struct {
    SecretID     string
    Provider     string
    GenerateNew  func() ([]byte, error) // callback por tipo
    UpdateSource func([]byte) error     // callback: escribir en GitHub/Fly/etc
}
```

- Rotation automática: FLY_API_TOKEN, RESEND_API_KEY (regenerables)
- Rotation manual: CF_API_TOKEN (requiere nuevo token del dashboard)
- GitHub Actions workflow trigger after rotation

- [ ] **Step 1: RotationConfig + callbacks por provider**
- [ ] **Step 2: `vault secrets rotate --id --auto --workflow`**
- [ ] **Step 3: Commit**

---

## Phase 5: Backup + UX ✅

### Tarea 5.1: Backup Cifrado

```bash
vault secrets backup --output ~/.local/share/ovav/secrets_backup_$(date +%Y%m%d).enc
vault secrets restore --input secrets_backup_20260803.enc
```

- Mismo formato .enc que vault de assets
- Exporta JSON cifrado con metadata: fecha, secretos count, hash del bundle
- Rotation: mantener últimos 5 backups

- [ ] **Step 1: Backup writer (encrypt + write)**
- [ ] **Step 2: Restore reader (read + decrypt)**
- [ ] **Step 3: Commit**

### Tarea 5.2: TUI Interactiva

**File:** `go-runtime/cmd/cockpit/` (extensión existente)

- Panel de secretos con tabs por tipo
- Status visual: verde=OK, amarillo=expirando, rojo=expired
- Click para ver detalles, rotar, o borrar

- [ ] **Pendiente: scope creep check — mantener CLI primero**

---

## Definition of Done

- [ ] `go run ./cmd/ovav-vault-secrets/ secrets add/list/get/remove` funciona
- [ ] Discovery encuentra secrets en GitHub, Fly.io, filesystem
- [ ] Health check verifica los 7 tipos de secreto
- [ ] Audit log registra todos los accesos
- [ ] Backup/restore funciona con .enc
- [ ] Cobertura tests ≥ 80% en `internal/vault/secrets/`
- [ ] `go run ./cmd/ovav/ validate` sigue pasando 21/21
- [ ] No se rompe el vault de assets existente (profiles/agents/skills)

---

## Referencias

- Vault actual: `go-runtime/internal/vault/encrypt.go` (AES-256-GCM)
- Cloudflare provider: `go-runtime/internal/infra/cloudflare.go`
- Fly.io CLI: `~/.fly/bin/fly`
- GitHub Secrets: API `repos/{owner}/{repo}/actions/secrets`
- credential_health.json: `.ovav/vault/credential_health.json` (verificado 2026-08-03)

---

*Thavren — Platform Engineering — 2026-08-03*
