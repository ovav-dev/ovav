---
title: Vault Encryption
description: Encrypt and decrypt OVAV assets with AES-256-GCM — setup, scan, encrypt, decrypt, and backup strategy.
---

OVAV Vault provides **AES-256-GCM encryption** for sensitive source assets. All encryption is local-first — keys never leave your machine.

## Overview

The Vault encrypts three asset bundles:

| Bundle | Source | Output |
|---|---|---|
| `profiles.enc` | `.ovav/registry/service_profiles.yaml` | `.ovav/vault/profiles.enc` |
| `agents.enc` | `.opencode/agents/*.md` | `.ovav/vault/agents.enc` |
| `skills.enc` | `.opencode/skills/*/SKILL.md` | `.ovav/vault/skills.enc` |

Each bundle is a JSON map serialized and encrypted with AES-256-GCM.

## Quick Start

### 1. Generate a Master Key

```bash
ovav vault gen-key --out .ovav/vault/master.key
```

This creates a 256-bit (32-byte) key stored as a 64-character hex string. The file permissions are set to `0600` (owner read/write only).

:::caution[Important]
Copy this key and store it securely. It **cannot be recovered** if lost. All encrypted data becomes inaccessible without it.
:::

### 2. Scan for Assets

```bash
ovav vault scan
```

Output:

```
OVAV Vault — Asset Discovery

  📦 profiles (1 files)
     └─ service_profiles.yaml
  📦 agents (8 files)
     └─ thavren.md
     └─ eidren.md
     └─ ...
  📦 skills (15 files)
     └─ cloudflare/SKILL.md
     └─ ovav-artifact-flow/SKILL.md
     └─ ...

Total: 3 bundles, 24 files ready to encrypt.
```

### 3. Encrypt All Assets

```bash
ovav vault encrypt --key .ovav/vault/master.key
```

Output:

```
OVAV Vault — Encryption Complete

  ✅ .ovav/vault/profiles.enc (1247 bytes)
  ✅ .ovav/vault/agents.enc (8932 bytes)
  ✅ .ovav/vault/skills.enc (15483 bytes)

Encrypted 3 files → 25662 total bytes.
Key required for decryption. Store it securely.
```

### 4. Decrypt Assets

```bash
ovav vault decrypt --key .ovav/vault/master.key
```

Output:

```
OVAV Vault — Decryption Complete
All assets restored to their original locations.
```

## Key Management

### Using Environment Variable

Instead of `--key`, you can set the key via environment variable:

```bash
export OVAV_VAULT_KEY=$(cat .ovav/vault/master.key)
ovav vault encrypt
ovav vault decrypt
```

### Key Format

The key is a 64-character hex string representing 32 bytes (256 bits):

```
a1b2c3d4e5f6...  (64 hex characters)
```

If the key length is incorrect, you'll get an error:

```
invalid key length: 32 hex chars (expected 64 for AES-256)
```

## Encryption Details

### Algorithm

- **Cipher**: AES-256-GCM
- **Key derivation**: PBKDF2 (via `internal/license/`)
- **Nonce**: 12 bytes, randomly generated per encryption
- **Auth tag**: 16 bytes (GCM standard)
- **Output format**: `nonce(12) || tag(16) || ciphertext`

### Go Implementation

All encryption uses Go stdlib only — zero third-party dependencies:

```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
)

func Encrypt(plaintext, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

### Bundle Format

Each encrypted bundle is a JSON structure:

```json
{
  "kind": "profiles",
  "version": 1,
  "files": {
    "service_profiles.yaml": "content..."
  }
}
```

## Backup Strategy

### Automated Backup

OVAV includes a backup script (`tools/security/vault_backup.sh`) that:

1. Scans for assets
2. Encrypts with the master key
3. Creates a timestamped tarball
4. Verifies the backup
5. Enforces retention (keeps last 10 backups)

### Manual Backup

```bash
# Backup the vault directory
tar czf ovav-vault-backup-$(date +%Y%m%d).tar.gz .ovav/vault/

# Backup the key separately (critical!)
cp .ovav/vault/master.key /secure/location/
chmod 600 /secure/location/master.key
```

### Restore from Backup

```bash
# Extract backup
tar xzf ovav-vault-backup-YYYYMMDD.tar.gz

# Decrypt assets
ovav vault decrypt --key .ovav/vault/master.key
```

## Security Considerations

| Aspect | Detail |
|---|---|
| Key storage | File permissions `0600`, never committed to git |
| Encryption | AES-256-GCM (authenticated encryption) |
| Nonce | Random per-operation, never reused |
| Dependencies | Go stdlib only — no supply chain risk |
| Audit | All encrypt/decrypt operations logged |

## Next Steps

- [Security Reference](/reference/security) — Full security architecture
- [CLI Reference](/reference/cli) — All vault commands
- [Configuration](/reference/configuration) — Environment variables
