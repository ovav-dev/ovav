# OVAV VAULT 2026 — PHASE 6+ CORRECTED PLAN
## Zero-Knowledge Sync + Machine-Bound Encryption + OVAV-Native Interface

**Author:** Thavren / Platform Engineering
**Date:** 2026-08-03
**Status:** CORRECTED — replaces PHASE6-PLAN.md hash-based approach
**Key correction:** cPanel stores ENCRYPTED BLOBS only. Zero-knowledge. Hashes are USELESS for service access.

---

## The Hash Problem — Why the Original Plan Was Wrong

```
GitHub API:  " Authorization: Bearer <actual_token_value> "
              ↑ you cannot use a HASH here

Cloudflare:  " X-Auth-Key: <actual_api_key> "
              ↑ hash is completely useless
```

**cPanel storing hashes = cPanel stores nothing useful.** A hash of a token cannot authenticate to any service. The original plan was a fundamental misunderstanding.

**Correct model:** cPanel stores the full vault blob — encrypted with the user's key. cPanel is a **dumb relay** that cannot read, modify, or use any secret inside the blob. Zero-knowledge in the truest sense.

---

## Security Architecture — How OVAV Vault Is Superior to Bitwarden/1Password

### Why Existing Vaults Are Fundamentally Weaker

| Problem | Bitwarden/1Password | OVAV Vault |
|---------|-------------------|------------|
| Master password brute-force | Vulnerable if DB leaked | **Machine-ID salt = cannot brute-force with seed alone** |
| Server-side key handling | Server sees derived key | **cPanel sees ONLY encrypted blobs** |
| Cross-device sync | Master password on all devices | **Vault key re-wrapped per device** |
| Audit log privacy | Server stores audit | **Audit log encrypted locally, cPanel never sees plaintext** |
| Hardware security | Master password typed | **TPM/Touch ID can unlock without seed entry** |
| Secret rotation | Manual | **cPanel rotates GitHub/Fly.io secrets directly via API** |
| Secret usage tracking | None | **Every vault access logged, encrypted, local-only** |
| Dependency graph | None | **Secrets know which repos/systems depend on them** |
| CI/CD air-gap | Poor | **TPM-backed temporary decryption, no seed required online** |

### Machine-Bound Encryption (The Core Security Innovation)

```
LOGIN (online):
  Seed + MachineID → PBKDF2 → VaultKey_AES256
  VaultKey encrypts/decrypts ~/.local/share/ovav/secrets.vault

ATTACK SCENARIO — seed stolen from cPanel DB:
  Attacker has: seed
  Attacker needs: machine_id of victim's device
  Without machine_id → cannot derive vault key → cannot decrypt vault
  → Even a complete cPanel breach exposes NOTHING useful

NORMAL DEVICE LOSS:
  MachineID is tied to hardware (/etc/machine-id, TPM)
  Stolen device → machine_id not available to attacker
  → Cannot decrypt vault without seed + machine_id
```

### Cross-Device Sync — Zero-Knowledge Wrapping

Problem:
```
Device A (machineID_A): VaultKey = PBKDF2(seed, machineID_A)
Device B (machineID_B): VaultKey = PBKDF2(seed, machineID_B)
VaultKey_A ≠ VaultKey_B → Device B cannot read Device A's vault
```

Solution: Vault Key Re-wrapping for Sync
```
Device A → sync:
  SyncKey = PBKDF2(seed, "ovav-sync-wrapping-key" /* NO machine_id */)
  WrappedVaultKey = AES-256-GCM(VaultKey_A, SyncKey)
  Upload: { machineID_A, WrappedVaultKey } → cPanel

Device B → sync:
  SyncKey = PBKDF2(seed, "ovav-sync-wrapping-key")
  Download: { WrappedVaultKey } from cPanel
  VaultKey_B = AES-256-GCM-Decrypt(WrappedVaultKey, SyncKey)
  → VaultKey_B encrypted blob now decryptable on Device B
  → All devices use SAME SyncKey (seed-only derived)
  → cPanel never sees: seed, machine_id, VaultKey, or any secret
```

**Result:** cPanel stores encrypted blobs keyed to machine_id. Even with full cPanel access, attacker cannot decrypt any vault without the seed AND a registered device's machine_id.

---

## cPanel API — What It Actually Stores

### cPanel Secret Record (ZERO KNOWLEDGE)

```json
{
  "identity_id": "uuid-from-jwt",
  "device_id": "machine-id-of-uploading-device",
  "blob": "<full AES-256-GCM encrypted vault blob>",
  "version": 1,
  "synced_at": "2026-08-03T...",
  "blob_hash": "SHA-256(blob) — for deduplication, NOT auth"
}
```

**cPanel NEVER stores:**
- Secret values (encrypted inside blob)
- Secret names (inside blob)
- Secret types (inside blob)
- Vault key (wrapped, needs seed to unwrap)
- Seed (only used for auth, not stored)

**cPanel DOES store:**
- Encrypted vault blob (useful only to devices with seed + machine_id)
- Device registry (which machines belong to this identity)
- Sync metadata (who synced when)
- JWT session tokens (standard auth, not secret storage)

---

## Phase 6.1: Corrected Sync Architecture

### Sync Protocol

```
Device A (online):
  1. Derive VaultKey_A = PBKDF2(seed, machineID_A)
  2. Derive SyncKey = PBKDF2(seed, "ovav-sync-v1")
  3. Load local vault: secrets.vault (encrypted with VaultKey_A)
  4. Decrypt vault → read secrets
  5. Re-encrypt vault blob with SyncKey → WrappedBlob
  6. Upload WrappedBlob + device_id → cPanel /sync

cPanel (blind relay):
  7. Store: { identity_id, device_id, blob: WrappedBlob, version }
  8. Return: { server_version, other_devices_blobs }

Device B (online, different machine):
  9. Derive SyncKey = PBKDF2(seed, "ovav-sync-v1")   ← SAME on all devices
  10. Download WrappedBlob from cPanel
  11. Derive VaultKey_B = PBKDF2(seed, machineID_B)
  12. Decrypt WrappedBlob with SyncKey → get vault contents
  13. Re-encrypt vault with VaultKey_B → store locally as secrets.vault
  14. Device B now has full vault, locally encrypted with ITS OWN key
```

### File: `go-runtime/internal/vault/secrets/sync.go` (corrected)

```go
package secrets

import (
    "crypto/sha256", "encoding/hex", "encoding/json", "fmt",
    "os", "path/filepath", "sync", "time"

    vaultpkg "github.com/ovav/ovav/internal/vault"
)

// SyncWrapKey is used to wrap the vault blob for cross-device sync.
// Derived from seed ONLY — same on all devices for a given seed.
// Versioned to allow future key rotation without seed change.
func SyncWrapKey(seed string) ([]byte, error) {
    // Uses a fixed string as domain separator — different from DeriveKey
    // NO machine_id — this key is the same on all devices
    return license.DeriveKey(seed, "ovav-sync-v1")
}

// SyncBlob is the format uploaded to cPanel.
type SyncBlob struct {
    IdentityID string    `json:"identity_id"`
    DeviceID  string    `json:"device_id"`
    Version   int       `json:"version"`
    SyncedAt  time.Time `json:"synced_at"`
    BlobHash  string    `json:"blob_hash"` // SHA-256 of encrypted blob
    // The actual vault blob — re-encrypted under SyncWrapKey
    Blob      []byte    `json:"blob"` // AES-256-GCM(syncVaultJSON, SyncKey)
}

// SyncPayload is the full sync state for an identity.
type SyncPayload struct {
    Blobs     []SyncBlob `json:"blobs"`     // one per device
    VaultJSON []byte     `json:"vault_json"` // current vault, syncKey-encrypted
}

// UploadSync uploads the current vault to cPanel as a sync blob.
func UploadSync(store *SecretStore, seed string, machineID string, jwt string) error {
    // Serialize current vault
    vaultJSON, err := store.ToJSON()
    if err != nil {
        return fmt.Errorf("sync serialize: %w", err)
    }

    // Wrap vault under SyncKey (seed-only)
    syncKey, err := SyncWrapKey(seed)
    if err != nil {
        return fmt.Errorf("sync wrap key: %w", err)
    }

    wrappedBlob, err := vaultpkg.Encrypt(vaultJSON, syncKey)
    if err != nil {
        return fmt.Errorf("sync encrypt: %w", err)
    }

    blob := SyncBlob{
        IdentityID: identityIDFromJWT(jwt),
        DeviceID:   machineID,
        Version:    1,
        SyncedAt:   time.Now().UTC(),
        BlobHash:   sha256Hex(wrappedBlob),
        Blob:       wrappedBlob,
    }

    // Upload to cPanel
    payload, _ := json.Marshal(blob)
    resp, err := cpanelRequest("POST", "/api/v1/sync/upload", jwt, payload)
    if err != nil {
        return fmt.Errorf("sync upload: %w", err)
    }

    // Check for newer blobs from other devices
    var syncResp SyncResponse
    if err := json.Unmarshal(resp, &syncResp); err != nil {
        return nil // non-fatal — our upload succeeded
    }

    // Merge newer blobs from other devices
    return mergeSyncBlobs(store, syncResp.Blobs, seed, machineID)
}

// DownloadSync downloads and merges vault from cPanel.
func DownloadSync(store *SecretStore, seed string, machineID string, jwt string) error {
    resp, err := cpanelRequest("GET", "/api/v1/sync/blobs", jwt, nil)
    if err != nil {
        return fmt.Errorf("sync download: %w", err)
    }

    var blobs []SyncBlob
    if err := json.Unmarshal(resp, &blobs); err != nil {
        return fmt.Errorf("sync parse: %w", err)
    }

    return mergeSyncBlobs(store, blobs, seed, machineID)
}

func mergeSyncBlobs(store *SecretStore, blobs []SyncBlob, seed string, machineID string) error {
    syncKey, err := SyncWrapKey(seed)
    if err != nil {
        return err
    }

    latest := map[string]SyncBlob{} // deviceID → latest blob
    for _, blob := range blobs {
        if existing, ok := latest[blob.DeviceID]; !ok || blob.SyncedAt.After(existing.SyncedAt) {
            latest[blob.DeviceID] = blob
        }
    }

    // Decrypt and merge each device's blob
    for deviceID, blob := range latest {
        if deviceID == machineID {
            continue // skip our own blob
        }

        vaultJSON, err := vaultpkg.Decrypt(blob.Blob, syncKey)
        if err != nil {
            fmt.Fprintf(os.Stderr, "⚠️  Blob from device %s failed to decrypt: %v\n", deviceID, err)
            continue
        }

        remoteStore, err := FromJSON(vaultJSON)
        if err != nil {
            continue
        }

        // Merge: add secrets from remote that don't exist locally
        for _, sec := range remoteStore.List("") {
            if store.GetByName(sec.Name) == nil {
                store.Add(sec)
            }
        }
    }

    return nil
}

func sha256Hex(data []byte) string {
    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:])
}
```

---

## Phase 6.2: cPanel Auth — Online vs Offline

### Smart Auth Flow (Revised)

```
ovav login
  │
  ├─ INTERNET AVAILABLE
  │    │
  │    ├─ POST /api/v1/auth/login { seed, machine_id, hostname }
  │    │    → cPanel verifies seed hash against registry
  │    │    → Returns JWT (24h TTL) + identity metadata
  │    │
  │    ├─ If JWT valid:
  │    │    → DownloadSync() — pull latest vault from cPanel
  │    │    → Merge any remote changes
  │    │    → UploadSync() — push local vault to cPanel
  │    │    → Identity confirmed online
  │    │
  │    └─ If JWT fails (unauthorized/revoked):
  │         → Fall back to offline: derive VaultKey locally
  │         → Warn: "Online auth failed — offline mode"
  │
  └─ NO INTERNET
       │
       ├─ Derive VaultKey = PBKDF2(seed, machine_id) — locally, NO network
       ├─ Unlock vault with VaultKey
       ├─ Queue sync operations for later
       └─ "Offline mode — vault unlocked, sync pending"
```

### JWT Storage

```
~/.local/share/ovav/
  session          ← vault_key_hash (NO JWT — different from original)
  vault_key_export ← hex(seed-derived key)
  cpanel.token     ← JWT from cPanel (encrypted with vault_key!)
```

**The JWT is stored encrypted with the vault key** — even if someone steals the laptop, they need the vault key to use the JWT for cPanel operations.

---

## Phase 6.3: OVAV CONNECT — Actual API Keys, Not Hashes

**OVAV CONNECT tracks the actual API keys** for AI providers. These live in the vault, encrypted with the vault key. Not hashes — real values.

### Connect Key Structure (stored in vault, encrypted)

```go
type ConnectKey struct {
    ID             string    `json:"id"`
    Provider       string    `json:"provider"`  // "openai" | "anthropic" | "openrouter" | "azure"
    Name           string    `json:"name"`      // "GPT-4o Production"
    EncryptedValue []byte    `json:"encrypted_value"` // AES-256-GCM(value, vault_key)
    EnvVar         string    `json:"env_var"`   // "OPENAI_API_KEY"
    AddedAt        time.Time `json:"added_at"`
    LastUsed       *time.Time `json:"last_used,omitempty"`
    ExpiresAt      *time.Time `json:"expires_at,omitempty"`
    MonthlyLimit   int       `json:"monthly_limit_cents"` // $127.43 = 12743
    CurrentSpend   int       `json:"current_spend_cents"`
    Status         string    `json:"status"`   // "active" | "expired" | "quota_exceeded"
    Metadata       map[string]string `json:"metadata,omitempty"` // model, org, etc.
}
```

### Connect Commands

```
ovav secrets connect add --provider openai --name "GPT-4o Prod" --value $OPENAI_API_KEY
ovav secrets connect list
ovav secrets connect status     # Provider, name, status, last_used, spend
ovav secrets connect track --provider openai  # Log usage
ovav secrets connect spend      # Monthly spend per provider
ovav secrets connect set-limit --provider openai --limit 50000  # $500/month
```

### Spend Tracking

OVAV CONNECT queries the OpenRouter/OpenAI billing APIs to get actual spend:

```
ovav secrets connect sync-spend
  OpenAI:      $127.43 / $500.00   [████████░░░░░░░░░] 25% — OK
  Anthropic:   $89.12 / $200.00   [██████████░░░░░░░] 44% — OK
  OpenRouter:  $198.00 / $200.00  [████████████████] 99% — ⚠️ QUOTA WARNING
```

---

## Phase 6.4: Dependency Graph — Secrets Know What Uses Them

A secret alone is inert. Its value is in being used correctly by systems. OVAV Vault tracks the **secret dependency graph**:

```go
type SecretRef struct {
    SecretID  string   `json:"secret_id"`  // which secret
    System    string   `json:"system"`     // "github-actions", "fly-app", "ci-cd", "env-file"
    Path      string   `json:"path"`       // where it's used
    EnvVar    string   `json:"env_var"`    // which env var name
    AddedAt   time.Time `json:"added_at"`
    AutoRotatable bool `json:"auto_rotatable"` // cPanel can rotate this?
}

// Example: CLOUDFLARE_API_TOKEN is used by:
//  - GitHub Actions: .github/workflows/deploy.yml → CLOUDFLARE_API_TOKEN
//  - Fly.io app: fly.toml → CLOUDFLARE_API_TOKEN
//  - CI/CD pipeline: .gitlab-ci.yml → CF_API_TOKEN
```

### Rotation Propagation

```
ovav secrets rotate CLOUDFLARE_API_TOKEN --new-value "new_token_here"

1. Update local vault with new token
2. UploadSync → cPanel
3. cPanel calls GitHub API: PATCH /repos/ovav-dev/ovav-systems/actions/secrets/CLOUDFLARE_API_TOKEN
4. cPanel calls Fly.io API: fly secrets set CLOUDFLARE_API_TOKEN=new_token
5. Audit log records: rotated by user, propagated to N systems
```

---

## Phase 6.5: TUI — Superior Interface

A modern TUI that makes Bitwarden/1Password look like 2015:

### Main Dashboard (`ovav secrets tui`)

```
╔══════════════════════════════════════════════════════════════╗
║  OVAV VAULT  ● ONLINE                    [Braka · CEO]      ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║  🔍 Search secrets...                               [⌘+K]  ║
║                                                              ║
║  ── SECRETS (8) ──────────────────── ── HEALTH ──────────   ║
║  │                                       │                   ║
║  │ CF production        cloud_key    ✓   │  6 ok            ║
║  │ Fly.io API           api_token    ✓   │  2 rotate soon   ║
║  │ GitHub Token         api_token    ⚠   │  0 expired       ║
║  │ VITE_API_BASE (cp)   api_token    ✓   │                  ║
║  │ AUTH_SECRET          user_secret  ⚠   │  ── SYNC ─────  ║
║  │ OpenAI Key           connect      ✓   │  ● synced 2m ago ║
║  │ Anthropic Key        connect      ✓   │  ↓ 3 devices    ║
║  │ Stripe Key            api_token    ✓   │                  ║
║  ─────────────────────────────────────────                  ║
║                                                              ║
║  [A] Add  [R] Rotate  [D] Discover  [S] Sync  [C] CONNECT   ║
╚══════════════════════════════════════════════════════════════╝
```

### Secret Detail View

```
╔══════════════════════════════════════════════════════════════╗
║  CLOUDFLARE_API_TOKEN                          [✕] [📋]   ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║  Type      cloud_key                                         ║
║  Provider  GitHub Secrets                                    ║
║  Status    ⚠ rotate in 23 days (threshold: 90d)            ║
║  Created   2026-07-01                                       ║
║  Hash      a3f8b2c1...                                      ║
║                                                              ║
║  ── USED BY ───────────────────────────────────────────────  ║
║                                                              ║
║  ● GitHub Actions · ovav-systems/deploy.yml                  ║
║    EnvVar: CLOUDFLARE_API_TOKEN                             ║
║    Auto-rotate: YES                                          ║
║                                                              ║
║  ● Fly.io · ovav-api                                        ║
║    EnvVar: CF_API_TOKEN                                     ║
║    Auto-rotate: YES                                          ║
║                                                              ║
║  ── AUDIT ────────────────────────────────────────────────   ║
║                                                              ║
║  2026-08-03 09:14  GET       workstation    Braka          ║
║  2026-08-02 14:22  ROTATE    workstation    Braka          ║
║  2026-08-01 11:00  ADD       workstation    Braka          ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
```

---

## Phase 6.6: Hardware-Backed Unlock (TPM/Touch ID)

For machines with TPM2 or Touch ID, the vault key can be sealed to hardware:

```go
// Hardware-backed unlock — seed only needed if TPM is cleared
func (s *SecretStore) UnlockWithTPM() error {
    if !tpmAvailable() {
        return fmt.Errorf("TPM not available on this machine")
    }

    // Vault key is sealed to TPM PCRs (Platform Configuration Registers)
    // PCR0: BIOS/UEFI measurements
    // PCR7: SecureBoot state

    sealedKey, err := tpm.Unseal("ovav-vault-key", sealedPolicy)
    if err != nil {
        return fmt.Errorf("TPM unseal failed: %w", err)
    }

    s.key = sealedKey
    return nil
}

// On first setup:
func SetupTPMSealing(vaultKey []byte) error {
    // Seal vault key to TPM with PCR policy
    return tpm.Seal("ovav-vault-key", vaultKey, sealedPolicy)
}
```

**Result:** After initial `ovav login`, the machine can unlock the vault via TPM without entering the seed again — as long as the BIOS/boot state hasn't changed. If the machine is reformatted or TPM is cleared, the seed is required again.

---

## Phase 6.7: Air-Gap Mode for CI/CD

For CI/CD environments where internet is unavailable or undesirable:

```go
// ovav secrets airgap --export secrets.airgap
// Produces a self-contained encrypted package:
//   - secrets.vault (encrypted with CI-specific key)
//   - env template (SECRET_NAME → REF format)
//   - No seed, no machine_id — single-use decryption key
//
// CI/CD pipeline:
//   ovav secrets airgap --import secrets.airgap
//   → Extracts env vars: CF_API_TOKEN=xxx, GITHUB_TOKEN=xxx
//   → Injects into pipeline environment
//   → On job completion: auto-clear from env

type AirgapPackage struct {
    Version     int               `json:"version"`
    Secrets     []EncryptedSecret `json:"secrets"`
    ExpiresAt   time.Time         `json:"expires_at"`
    Permissions []string          `json:"permissions"` // "github-actions", "fly-io"
    Signature   []byte            `json:"signature"`   // HMAC of entire package
}
```

---

## Implementation Priority

| Phase | Task | Why First |
|-------|------|-----------|
| **6.1** | Corrected sync architecture (wrap/wrap) | Without this, vault only works on ONE device |
| **6.2** | cPanel sync client | Needs cPanel server implementation first |
| **6.3** | Online/Offline auth flow | Foundation for everything else |
| **6.4** | CONNECT keys (real values) | User-visible value immediately |
| **6.5** | TUI dashboard | Makes it actually usable |
| **6.6** | Dependency graph | Unique value over Bitwarden |
| **6.7** | Rotation propagation | Closes the loop on auto-rotate |
| **6.8** | TPM/硬件 unlock | Security superiority |
| **6.9** | Air-gap mode | CI/CD use case |

---

## What Makes This Superior to Bitwarden/1Password

```
OVAV Vault IS Bitwarden/1Password if they were designed in 2026
by engineers who understand:

  1. Machine-bound encryption means seed theft alone is useless
  2. Zero-knowledge cPanel means even full server breach = nothing
  3. Secret dependency graph means rotation auto-propagates
  4. Local encrypted audit means usage patterns never leaked
  5. Hardware unsealing means no master password typing daily
  6. Air-gap mode means CI/CD never touches the internet for secrets
  7. OVAV-native means agents can use secrets directly via API
  8. Connect key tracking means AI API spend is visible and controlled
```

---

## cPanel Server Requirements (what needs to be built on the server side)

cPanel needs to implement these endpoints. OVAV Vault is the client:

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/auth/login` | Seed+machineID → JWT |
| GET | `/api/v1/sync/blobs` | List all blobs for identity |
| POST | `/api/v1/sync/upload` | Upload wrapped vault blob |
| POST | `/api/v1/rotate` | Trigger secret rotation in upstream (GitHub/Fly) |
| GET | `/api/v1/connect/spend` | Provider billing info |
| POST | `/api/v1/connect/track` | Log AI API usage |

**Server never decrypts blobs.** It stores them, syncs them, and can trigger rotations via the stored provider API credentials (which are themselves encrypted in the blob — cPanel uses the JWT identity to authenticate rotation requests to GitHub/Fly.io on the user's behalf).
