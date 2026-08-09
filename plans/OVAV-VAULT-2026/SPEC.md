# OVAV VAULT 2.0 — System Specification
**Status:** PLANNING | **Version:** 2.0 | **Date:** 2026-08-03
**Lead:** Thavren (Platform Engineering) | **Target:** Bitwarden/1Password-class, engineering-focused

---

## 1. Concept & Vision

OVAV VAULT 2.0 is a **Zero-Knowledge Credential Intelligence Platform** — not just a password manager, but the **control plane for all machine identities** in an engineering organization.

**What it does:**
- Stores credentials: API keys, tokens, certificates, database passwords, cloud keys
- Discovers credentials: scans systems, env files, CI pipelines automatically
- Controls credentials: revokes, rotates, propagates changes across all systems
- Syncs across devices: encrypted zero-knowledge relay via cPanel
- Wipes credentials: `--revoke` calls provider APIs to truly invalidate, then deletes locally

**What makes it superior:**
| Feature | Bitwarden | 1Password | OVAV VAULT 2.0 |
|---------|-----------|-----------|----------------|
| Zero-knowledge server | ❌ Server has keys | ❌ Server has keys | ✅ cPanel never sees plaintext |
| Provider API revoke | ❌ | ❌ | ✅ Calls GitHub/Fly API on revoke |
| Dependency graph | ❌ | ❌ | ✅ Knows which systems use which secret |
| Auto-rotation | Limited | Limited | ✅ Smart rotation with propagation |
| Air-gap mode | ❌ | ❌ | ✅ Export/import encrypted package |
| Ephemeral in-memory | ❌ | ❌ | ✅ Secrets in RAM only, never on disk |
| TPM/hardware key | ❌ | ✅ | ✅ PCR-bound TPM unsealing |
| AI provider billing | ❌ | ❌ | ✅ OpenRouter/etc spend tracking |

---

## 2. Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                          OVAV SYSTEMS (complete)                              │
│                                                                              │
│  ┌─────────────────────────┐          ┌──────────────────────────────────┐  │
│  │  ovav.dev               │          │  d678beea.ovav.dev                │  │
│  │  Landing page (marketing)│          │  ┌────────────────────────────┐  │  │
│  │  Pricing, CTA, profiles │          │  │  cPanel ADMIN (internal)   │  │  │
│  └─────────────────────────┘          │  │  Validator config, gates,  │  │  │
│                                       │  │  session management         │  │  │
│                                       │  └────────────────────────────┘  │  │
│                                       │  ┌────────────────────────────┐  │  │
│                                       │  │  User Login (user portal)   │  │  │
│                                       │  │  Account + API keys +       │  │  │
│                                       │  │  Secrets management         │  │  │
│                                       │  └────────────────────────────┘  │  │
│                                       │  ┌────────────────────────────┐  │  │
│                                       │  │  VAULT SERVER (core)        │  │  │
│                                       │  │  All secrets stored here    │  │  │
│                                       │  │  Zero-knowledge relay       │  │  │
│                                       │  └────────────────────────────┘  │  │
│                                       └──────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                      VAULT CLIENT (local workstation)                     │
│                                                                          │
│  ~/.local/share/ovav/                                                    │
│    session              ← JWT de sesión (del vault server)               │
│    deps.graph           ← metadata de refs (NUNCA valores)               │
│    vault_key_export     ← KEK derivado del seed (solo para TPM unlock)   │
│                                                                          │
│  ❌ NO secrets.vault local — el vault es PURAMENTE REMOTO               │
│                                                                          │
│  CLI workflow:                                                           │
│    ovav vault login  → seed → vault server → JWT session (24h)           │
│    ovav vault get X  → fetch remote → decrypt con KEK → inject al env   │
│    Sin red           → TPM unlock del KEK local → secrets = 0           │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     │ HTTPS (JWT Bearer)
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                     VAULT SERVER  d678beea.ovav.dev                       │
│                                                                          │
│  Almacena TODOS los secretos (cifrados con la key del usuario)           │
│  Conoce: identity (SHA256(seed)), secrets cifrados, metadata, audit      │
│  NO conoce: seed plaintext, KEK, valores decrypted                       │
│                                                                          │
│  Endpoints:                                                              │
│    POST   /api/v1/vault/auth          login → JWT (24h TTL)              │
│    GET    /api/v1/vault/secrets       lista metadata (sin valores)       │
│    GET    /api/v1/vault/secrets/:name secret value (JWT requerido)       │
│    POST   /api/v1/vault/secrets       agregar secret (CLI encrypta 1°)   │
│    DELETE /api/v1/vault/secrets/:name borrar secret                      │
│    POST   /api/v1/vault/revoke        revoke + llamar provider API        │
│    POST   /api/v1/vault/rotate        rotate + propagar a providers       │
│    GET    /api/v1/vault/sync          estado de sync                      │
│    GET    /api/v1/vault/health       health check                        │
│                                                                          │
│  Providers (GitHub, Fly.io) llamados por el server en revoke/rotate      │
└──────────────────────────────────────────────────────────────────────────┘
```

### Zero-Knowledge Model (remoto puro)

```
Seed → PBKDF2(seed, machine_id) → KEK (key encryption key, solo local)
     → SHA256(seed) → identity (enviado al server para lookup)

Vault server almacena:
  { identity, encrypted_secrets[], metadata[], audit_log[] }
  → El server NO puede decryptar — necesita KEK (solo existe localmente)

Vault client (CLI):
  1. Login → seed → derive KEK → JWT del server
  2. Get SECRET → JWT → server retorna ciphertext
     → CLI decrypt con KEK → inject al env del proceso
  3. Sin conexión → TPM unlock del KEK
     → KEK disponible, pero secrets = 0 (offline = auth, no secretos)
```

### Flujo de sesión

```
ovav vault login
  → browser abre d678beea.ovav.dev/login
  → usuario ingresa credenciales
  → server valida → devuelve JWT (24h)
  → JWT en ~/.local/share/ovav/session
  → KEK derivado localmente (PBKDF2)

ovav vault secrets list
  → JWT → GET /api/v1/vault/secrets
  → server devuelve metadata (nombres, tipos, providers, refs)
  → CLI muestra tabla

ovav vault secrets get GITHUB_TOKEN
  → JWT → GET /api/v1/vault/secrets/GITHUB_TOKEN
  → server devuelve valor cifrado
  → CLI decrypt con KEK → inject al env del proceso actual

ovav vault secrets add NEW_SECRET --value "..."
  → CLI encrypta con KEK → POST ciphertext al server
  → server guarda sin poder ver el contenido

ovav vault secrets revoke GITHUB_TOKEN
  → JWT → POST /api/v1/vault/revoke
  → server llama GitHub API DELETE
  → server marca revoked + actualiza audit log

ovav vault secrets rotate GITHUB_TOKEN
  → JWT → POST /api/v1/vault/rotate
  → server genera nuevo valor → encrypta → PUT a GitHub
  → server actualiza vault con nuevo ciphertext

Sin conexión (offline)
  → TPM unlock del KEK local (si configurado)
  → KEK disponible en memoria
  → pero GET /api/v1/vault/secrets → empty (sin red)
  → ❌ No hay secretos disponibles offline
```

---

## 3. UX/UI Specification

### 3.1 CLI (primary interface)

```
ovav vault [command] [flags]

Commands:
  add         Add a secret (interactive or --value)
  list        List all secrets
  get         Get a secret (decrypted, shown once)
  remove      Remove a secret from vault (not from providers)
  revoke      Revoke from ALL providers + delete from vault + clean graph
  rotate      Rotate a secret (generate new + push to all providers)
  connect     Manage AI provider keys (OpenRouter, OpenAI, etc.)
  sync        Bidirectional sync via cPanel
  deps        Dependency graph: list / impact / orphans
  audit       Audit log viewer
  backup      Backup vault to encrypted file
  restore     Restore from backup
  airgap      Air-gap mode: export / import

Flags:
  --json      JSON output (for scripting)
  --env       Treat value as environment variable name to read from
  --provider  Provider type (github, fly, openrouter, etc.)
  --system    System type (github-actions, fly-io, local-env, etc.)
```

### 3.2 Interactive TUI (bubble tea)

```
╔══════════════════════════════════════════════════════════════╗
║  OVAV VAULT  ▸  secrets  ▸  ovav-vault-secrets  ▸  ● LIVE ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║   ┌─ SECRETS ─────────────────────────────────────────┐    ║
║   │  🔑 GITHUB_TOKEN              api_token   3d ago   │    ║
║   │  🔑 FLY_API_TOKEN            cloud_key   7d ago   │    ║
║   │  🔑 OPENAI_API_KEY           api_token   1h ago   │    ║
║   │  🔑 CF_API_TOKEN             cloud_key   expiring  │    ║
║   └────────────────────────────────────────────────────┘    ║
║                                                              ║
║   🔍 Search secrets...                          [+ Add]     ║
║                                                              ║
║   ── STATUS ─────────────────────────────────────────────── ║
║   ● Online   ↕ Synced 2m ago   ⚠ 1 expiring   🔒 8 vaulted ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
```

**Views:**
- `dashboard`: Secret list + health + sync status + spend bar
- `secret_detail`: Selected secret with USED BY graph, audit log, rotation status
- `add_secret`: Interactive form with type detection
- `revoke_confirm`: Confirmation with impact preview (what will be revoked)
- `spend_report`: OVAV CONNECT — per-provider spend visualization
- `sync_log`: Real-time sync status

### 3.3 Natural Language Queries (AI Query Engine)

```bash
$ ovav vault "what secrets expire next week"
$ ovav vault "show me all github tokens that haven't been rotated in 90 days"
$ ovav vault "which secrets are used in production"
$ ovav vault "who used CLOUDFLARE_API_KEY last month"
$ ovav vault "audit log for FLY_API_TOKEN"
```

### 3.4 Web Dashboard (future)

- Secret inventory with search/filter
- Dependency graph visualization
- Spend dashboard (CONNECT)
- Audit log with filters
- Team management (sharing)

---

## 4. Intelligence Layer

### 4.1 Automatic Discovery
- **GitHub Actions**: Scan all repos for `secrets: []` in actions
- **Fly.io**: Query `flyctl secrets list` for all apps
- **Filesystem**: Scan `.env`, `.env.local`, `*secrets*.yaml`, `credentials.json`
- **Environment**: Monitor newly exported env vars during session

### 4.2 Credential Health
- **Expiration tracking**: Warn 30d, 7d, 1d before expiry
- **Rotation reminders**: Auto-prompt when secret is stale (>90d)
- **Compromise detection**: Check HaveIBeenPwned API for exposed keys
- **Provider health**: Check if GitHub/Fly API is reachable before sync

### 4.3 Smart Rotation
- `rotate <name>`: Generate new secret value → push to all providers → update vault
- Rotation propagates through dependency graph automatically
- Fallback: if one provider fails, rollback and report

### 4.4 OVAV CONNECT (AI Provider Intelligence)
- **OpenRouter**: `GET /api/v1/credits` → real-time spend
- **OpenAI**: API key usage via platform.openai.com
- **Anthropic**: API key usage
- Spend alerts: 50%, 75%, 90%, 100% of limit
- Auto-detect new API keys in environment

---

## 5. Security Model

### 5.1 Key Hierarchy
```
seed (user-provided, never stored)
  ├── PBKDF2(seed, machine_id) → VaultKey (AES-256-GCM, per-device)
  │     └── encrypts/decrypts secrets.vault
  └── PBKDF2(seed, "ovav-sync-v1") → SyncKey (same on all devices)
        └── encrypts blob for cPanel relay (zero knowledge)
```

### 5.2 Ephemeral Secrets
- `vault get <name>` → decrypted value printed to stdout
- Value held in RAM only — not written to `~/.bash_history` or disk
- Memory zeroed after display (best effort in Go)
- TUI: value hidden by default, click "reveal" to show temporarily

### 5.3 TPM/ Secure Enclave (Phase 6.6)
- VaultKey can be sealed to TPM PCRs (PCR0 boot state, PCR7 secure state)
- On trusted boot: TPM unseals → no seed entry needed
- On untrusted boot: TPM unseal fails → seed entry required
- macOS: Keychain integration
- Windows: DPAPI integration

### 5.4 Air-Gap Mode (Phase 6.7)
```bash
# Export (on secure machine)
ovav vault airgap --export backup.airgap --seed <seed>

# Output: self-contained encrypted package
#   - vault blob (AES-256-GCM)
#   - env template (names only, no values)
#   - HMAC signature
#   - expiration (optional, e.g., 24h)
#   - revocation list (for revoked secrets)

# Import (on air-gapped machine)
ovav vault airgap --import backup.airgap
# Prompts for seed, extracts secrets, sets env vars
# On job end / machine sleep: auto-clear memory
```

---

## 6. Data Model

### 6.1 Secret
```go
type Secret struct {
    ID          string          // SHA256(name + type + source)
    Name        string          // e.g. "GITHUB_TOKEN"
    Type        SecretType      // api_token, oauth_creds, db_credential, cloud_key, encryption_key, user_secret, tunnel_token
    Value       string          `json:"-"` // NEVER in JSON (held in RAM only)
    Metadata    map[string]string // encrypted_b64, provider, last_rotated, expires_at, etc.
    Source      string          // "github", "fly", "filesystem", "manual"
    SourcePath  string          // "GitHub Actions: owner/repo", "Fly.io app: myapp"
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 6.2 DependencyGraph
```go
type SecretRef struct {
    ID            string    // SHA256(secretID+system+path)[:16]
    SecretID      string    // which secret
    System        System    // github-actions, fly-io, gitlab-ci, jenkins, local-env, ci-cd
    Path          string    // file or resource path
    EnvVar        string    // which env var name
    AddedAt       time.Time
    AutoRotatable bool      // cPanel can rotate this?
}
```

### 6.3 SyncBlob (for cPanel relay)
```go
type SyncBlob struct {
    DeviceID  string    // machine ID
    Version   int       // monotonically increasing
    SyncedAt  time.Time
    BlobHash  string    // SHA256 of encrypted blob (dedup)
    Blob      []byte    // AES-256-GCM(SyncKey, vaultJSON)
}
```

---

## 7. Implementation Phases

### Phase 0 — Foundation ✅ (DONE)
- SecretStore + AES-256-GCM encryption
- SecretType taxonomy (7 types)
- CRUD: add, list, get, remove

### Phase 1 — Discovery ✅ (DONE)
- GitHub provider (PAT scan)
- Fly.io provider (flyctl)
- Filesystem provider (.env scan)
- Auto-type detection

### Phase 2 — Health ✅ (DONE)
- Expiration tracking
- Credential health scoring
- Provider reachability checks

### Phase 3 — Audit ✅ (DONE)
- Append-only encrypted audit log
- LogEntry: timestamp, action, machine_id, source

### Phase 4 — Backup/Export ✅ (DONE)
- Encrypted backup to file
- Import from backup

### Phase 5 — OVAV CONNECT ✅ (DONE)
- AI provider key tracker
- Spend reporting (OpenRouter billing API)
- Auto-detect env keys

### Phase 6 — Sync + Control ⬜ (IN PROGRESS)
- 6.1 ✅ Zero-knowledge wrap/wrap sync
- 6.2 ✅ cPanel vault relay server (endpoints: /vault/auth, /vault/blobs, /vault/upload, /vault/blob/:id)
- 6.3 ✅ OVAV CONNECT (spend tracker)
- 6.4 ✅ Dependency graph
- 6.5 ⬜ TUI dashboard (bubble tea)
- 6.6 ⬜ TPM unlock (go-tpm)
- 6.7 ⬜ Air-gap mode
- **6.8 🔵 REVOKE engine** (GitHub API revoke + Fly.io API revoke)
- 6.9 ⬜ Rotate command (generate + push + update vault)

### Phase 7 — Intelligence ⬜ (NEW)
- 7.1 ⬜ Natural language queries (NLP → structured query)
- 7.2 ⬜ Auto-rotation engine (cron-like scheduler)
- 7.3 ⬜ HaveIBeenPwned integration (compromise detection)
- 7.4 ⬜ Anomaly detection (unusual access patterns)

### Phase 8 — Enterprise ⬜ (NEW)
- 8.1 ⬜ Team sharing (secret sharing with encrypted key envelopes)
- 8.2 ⬜ Web dashboard (HTTP server, browser UI)
- 8.3 ⬜ Vault Agent daemon (background service, API for agents)
- 8.4 ⬜ Hardware key support (YubiKey, NitroKey)

---

## 8. CLI Commands (Complete Reference)

| Command | Description |
|---------|-------------|
| `vault add` | Add a secret |
| `vault list` | List all secrets |
| `vault get <name>` | Show decrypted secret (stdout) |
| `vault remove <name>` | Remove from vault (not providers) |
| `vault revoke <name>` | Revoke from ALL providers + delete |
| `vault rotate <name>` | Generate new + push to all systems |
| `vault connect add` | Add AI provider key |
| `vault connect list` | List AI provider keys |
| `vault connect status` | Spend report |
| `vault connect detect` | Auto-detect env API keys |
| `vault sync` | Full bidirectional sync |
| `vault deps list` | Dependency graph |
| `vault deps impact <name>` | Rotation impact |
| `vault deps orphans` | Orphaned secrets |
| `vault audit` | Audit log |
| `vault backup` | Backup to file |
| `vault restore` | Restore from backup |
| `vault airgap --export` | Air-gap export |
| `vault airgap --import` | Air-gap import |
| `vault query "<nl>"` | Natural language query |

---

## 9. API Contracts (cPanel Server)

### POST /api/v1/vault/auth
```json
// Request
{ "seed": "...", "machine_id": "...", "hostname": "..." }

// Response 200
{ "jwt": "...", "exp": "2026-08-04T10:00:00Z", "identity_id": "uuid", "level": 1, "role": "vault-user" }
```

### GET /api/v1/vault/blobs
```
Authorization: Bearer <jwt>

// Response 200
{
  "identity_id": "uuid",
  "blobs": [
    { "device_id": "machine-1", "version": 3, "blob_hash": "sha256", "synced_at": "..." }
  ],
  "server_ts": "..."
}
```

### POST /api/v1/vault/upload
```
Authorization: Bearer <jwt>
Content-Type: application/octet-stream

<body>: raw encrypted vault blob bytes

// Response 200
{ "identity_id": "...", "device_id": "...", "blob_hash": "sha256", "synced_at": "..." }
```

### GET /api/v1/vault/blob/:deviceID
```
Authorization: Bearer <jwt>

// Response 200: raw vault blob bytes (Content-Type: application/octet-stream)
```

### DELETE /api/v1/vault/blob/:deviceID
```
Authorization: Bearer <jwt>

// Response 200
{ "deleted": "device_id" }
```

---

## 10. Comparison: OVAV VAULT vs Competition

| Capability | Bitwarden | 1Password | LastPass | OVAV VAULT 2.0 |
|------------|-----------|-----------|----------|----------------|
| Zero-knowledge server | ✅ | ✅ | ✅ | ✅ + SYNTHETIC ZERO |
| Secret types | Passwords, notes | Passwords, notes | Passwords | **API keys, tokens, certs, DB credentials, cloud keys** |
| CI/CD integration | Limited | SSH keys | Limited | **Native GitHub/Fly/GitLab** |
| Dependency graph | ❌ | ❌ | ❌ | **✅ Full graph** |
| Auto-revoke via API | ❌ | ❌ | ❌ | **✅** |
| Air-gap mode | ❌ | ❌ | ❌ | **✅ Encrypted package** |
| Hardware key (TPM) | ❌ | ✅ | ❌ | **✅ PCR-bound** |
| AI spend tracking | ❌ | ❌ | ❌ | **✅ OpenRouter/etc** |
| Ephemeral RAM only | ❌ | ❌ | ❌ | **✅** |
| Multi-device sync | ✅ | ✅ | ✅ | **✅ ZK relay** |
| Open source | Partial | ❌ | ❌ | **✅ Full** |
| Engineering focus | ❌ | Some | ❌ | **✅ Deep** |

---

## 11. Security Properties

1. **Server-blind**: cPanel only stores encrypted blobs + SHA256(seed) hashes
2. **Machine-bound**: VaultKey = PBKDF2(seed, machine_id) — stolen seed alone useless
3. **Ephemeral**: Secret values exist in RAM only during active session
4. **TPM-sealed**: VaultKey can require hardware-rooted unsealing
5. **Audit-complete**: Every get/add/remove/rotate logged, immutable
6. **Revocation-complete**: `--revoke` calls real provider APIs to invalidate
7. **Air-gap-ready**: Full vault can be exported as self-contained encrypted package
8. **Zero-trust cPanel**: cPanel cannot decrypt blobs, cannot forge identity

---

*OVAV VAULT 2.0 — engineered for the age of AI agent workflows*
